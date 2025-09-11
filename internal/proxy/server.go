package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"rpc-proxy/internal/config"
	"rpc-proxy/internal/health"
	"rpc-proxy/internal/types"
)

type Server struct {
	config         *config.Config
	multiChainHealthChecker *health.MultiChainChecker
	client         *http.Client
	mu             sync.RWMutex
	chainPathRegex *regexp.Regexp
}

func NewServer(cfg *config.Config, multiChainHealthChecker *health.MultiChainChecker) *Server {
	// Compile regex for chain path matching: /rpc/{chain}
	chainPathRegex := regexp.MustCompile(`^/rpc/([a-zA-Z0-9]+)/?$`)
	
	return &Server{
		config:         cfg,
		multiChainHealthChecker: multiChainHealthChecker,
		client: &http.Client{
			Timeout: cfg.Proxy.Timeout,
		},
		chainPathRegex: chainPathRegex,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	
	// Multi-chain health endpoint
	mux.HandleFunc("/health", s.handleMultiChainHealth)
	
	// Chain-specific health endpoints
	mux.HandleFunc("/health/", s.handleChainHealth)
	
	// Multi-chain RPC endpoints
	mux.HandleFunc("/rpc/", s.handleMultiChainRPC)
	
	// Legacy single-chain RPC endpoint (defaults to ethereum)
	mux.HandleFunc("/rpc", s.handleLegacyRPC)
	mux.HandleFunc("/", s.handleLegacyRPC)

	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")
		w.Header().Set("Access-Control-Expose-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// handleMultiChainHealth returns overall health status for all chains
func (s *Server) handleMultiChainHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	multiChainStatus := s.multiChainHealthChecker.GetMultiChainStatus()

	// Mark as unhealthy if no chains have healthy endpoints
	if multiChainStatus.HealthyChains == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(multiChainStatus)
}

// handleChainHealth returns health status for a specific chain
func (s *Server) handleChainHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract chain name from path: /health/{chainName}
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathParts) != 2 || pathParts[0] != "health" {
		http.Error(w, "Invalid path format. Use /health/{chainName}", http.StatusBadRequest)
		return
	}

	chainName := pathParts[1]
	chainStatus := s.multiChainHealthChecker.GetChainStatus(chainName)
	if chainStatus == nil {
		http.Error(w, fmt.Sprintf("Chain %s not found", chainName), http.StatusNotFound)
		return
	}

	// Legacy format for backward compatibility
	legacyStatus := types.HealthStatus{
		Proxy:        "healthy",
		CurrentRPC:   chainStatus.CurrentRPC,
		RPCEndpoints: append(chainStatus.HealthyEndpoints, chainStatus.UnhealthyEndpoints...),
		Chain:        chainName,
	}

	if chainStatus.HealthyCount == 0 {
		legacyStatus.Proxy = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(legacyStatus)
}

// handleMultiChainRPC handles requests to specific chains via /rpc/{chainName}
func (s *Server) handleMultiChainRPC(w http.ResponseWriter, r *http.Request) {
	// Extract chain name from URL path
	matches := s.chainPathRegex.FindStringSubmatch(r.URL.Path)
	if len(matches) != 2 {
		log.Printf("Invalid multi-chain RPC path: %s", r.URL.Path)
		s.writeErrorResponse(w, -32600, "Invalid request path. Use /rpc/{chainName}", nil)
		return
	}

	chainName := matches[1]
	s.handleRPCForChain(w, r, chainName)
}

// handleLegacyRPC handles legacy requests to /rpc (defaults to ethereum mainnet)
func (s *Server) handleLegacyRPC(w http.ResponseWriter, r *http.Request) {
	s.handleRPCForChain(w, r, "ethereum") // Default to Ethereum mainnet
}

// handleRPCForChain processes RPC requests for a specific chain
func (s *Server) handleRPCForChain(w http.ResponseWriter, r *http.Request, chainName string) {
	// Log incoming request details for debugging
	log.Printf("Incoming request: Method=%s, ContentType=%s, ContentLength=%d, URL=%s, Chain=%s", 
		r.Method, r.Header.Get("Content-Type"), r.ContentLength, r.URL.Path, chainName)

	if r.Method != "POST" && r.Method != "GET" {
		log.Printf("Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Accept any Content-Type for POST requests, don't validate
	if r.Method == "POST" {
		contentType := r.Header.Get("Content-Type")
		log.Printf("POST request with Content-Type: %s", contentType)
		
		// Log request body for debugging
		if r.ContentLength > 0 && r.ContentLength < 1000 {
			bodyBytes, _ := io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			previewLen := len(bodyBytes)
			if previewLen > 200 {
				previewLen = 200
			}
			log.Printf("Request body preview: %s", string(bodyBytes[:previewLen]))
		}
	}

	start := time.Now()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		s.writeErrorResponse(w, -32700, "Parse error", nil)
		return
	}
	defer r.Body.Close()

	healthyEndpoints := s.multiChainHealthChecker.GetHealthyEndpoints(chainName)
	if len(healthyEndpoints) == 0 {
		log.Printf("No healthy RPC endpoints available for chain: %s", chainName)
		s.writeErrorResponse(w, -32000, fmt.Sprintf("No healthy RPC endpoints available for chain: %s", chainName), nil)
		return
	}

	// Sort endpoints by weight (highest first) for failover
	sortedEndpoints := s.getSortedEndpointsByWeight(healthyEndpoints)
	var lastErr error
	
	// Try each endpoint by weight priority
	for i, endpoint := range sortedEndpoints {
		resp, err := s.forwardRequest(r.Context(), endpoint, body, r.Header)
		if err != nil {
			log.Printf("Request to %s failed (attempt %d/%d): %v", endpoint.URL, i+1, len(sortedEndpoints), err)
			lastErr = err
			continue
		}

		s.copyResponse(w, resp)
		resp.Body.Close()
		
		duration := time.Since(start)
		log.Printf("Request forwarded to %s (chain: %s, weight: %d) completed in %v", endpoint.URL, chainName, endpoint.Weight, duration)
		return
	}

	log.Printf("All retry attempts failed, last error: %v", lastErr)
	s.writeErrorResponse(w, -32000, "All RPC endpoints failed", lastErr.Error())
}

func (s *Server) selectHealthyEndpointForChain(chainName string) *types.RPCEndpoint {
	healthyEndpoints := s.multiChainHealthChecker.GetHealthyEndpoints(chainName)
	if len(healthyEndpoints) == 0 {
		return nil
	}

	return s.selectEndpointByWeight(healthyEndpoints)
}

func (s *Server) selectEndpointByWeight(endpoints []*types.RPCEndpoint) *types.RPCEndpoint {
	sortedEndpoints := s.getSortedEndpointsByWeight(endpoints)
	if len(sortedEndpoints) == 0 {
		return nil
	}
	return sortedEndpoints[0]
}

func (s *Server) getSortedEndpointsByWeight(endpoints []*types.RPCEndpoint) []*types.RPCEndpoint {
	if len(endpoints) == 0 {
		return nil
	}

	// Sort endpoints by weight (highest first)
	sortedEndpoints := make([]*types.RPCEndpoint, len(endpoints))
	copy(sortedEndpoints, endpoints)
	
	// Simple bubble sort by weight (descending)
	for i := 0; i < len(sortedEndpoints)-1; i++ {
		for j := 0; j < len(sortedEndpoints)-i-1; j++ {
			if sortedEndpoints[j].Weight < sortedEndpoints[j+1].Weight {
				sortedEndpoints[j], sortedEndpoints[j+1] = sortedEndpoints[j+1], sortedEndpoints[j]
			}
		}
	}

	return sortedEndpoints
}

func (s *Server) forwardRequest(ctx context.Context, endpoint *types.RPCEndpoint, body []byte, headers http.Header) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Copy headers first, then ensure Content-Type is set correctly
	for key, values := range headers {
		if key == "Host" || key == "Content-Length" {
			continue
		}
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	// Always ensure Content-Type is application/json for RPC requests
	req.Header.Set("Content-Type", "application/json")
	
	log.Printf("Forwarding request to %s with Content-Type: %s", endpoint.URL, req.Header.Get("Content-Type"))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	log.Printf("Response from %s: Status=%d, Content-Type=%s", endpoint.URL, resp.StatusCode, resp.Header.Get("Content-Type"))
	return resp, nil
}

func (s *Server) copyResponse(w http.ResponseWriter, resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (s *Server) writeErrorResponse(w http.ResponseWriter, code int, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := types.JSONRPCResponse{
		Jsonrpc: "2.0",
		Error: &types.JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: nil,
	}

	json.NewEncoder(w).Encode(response)
}