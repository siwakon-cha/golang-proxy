package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"rpc-proxy/internal/config"
	"rpc-proxy/internal/health"
	"rpc-proxy/internal/types"
)

// MultiChainAdminHandler handles multi-chain administration endpoints
type MultiChainAdminHandler struct {
	config                  *config.Config
	multiChainHealthChecker *health.MultiChainChecker
}

// NewMultiChainAdminHandler creates a new multi-chain admin handler
func NewMultiChainAdminHandler(cfg *config.Config, healthChecker *health.MultiChainChecker) *MultiChainAdminHandler {
	return &MultiChainAdminHandler{
		config:                  cfg,
		multiChainHealthChecker: healthChecker,
	}
}

// RegisterRoutes registers all multi-chain admin routes
func (h *MultiChainAdminHandler) RegisterRoutes(mux *http.ServeMux) {
	// Chain management endpoints
	mux.HandleFunc("/admin/chains", h.handleChains)
	mux.HandleFunc("/admin/chains/", h.handleChain)
	
	// Chain endpoint management
	mux.HandleFunc("/admin/chains/{chainName}/endpoints", h.handleChainEndpoints)
	mux.HandleFunc("/admin/chains/{chainName}/endpoints/", h.handleChainEndpoint)
	
	// Chain configuration management
	mux.HandleFunc("/admin/chains/{chainName}/config", h.handleChainConfig)
	
	// Health check management
	mux.HandleFunc("/admin/health", h.handleHealthOverview)
	mux.HandleFunc("/admin/health/", h.handleChainHealthDetails)
	
	// Statistics and monitoring
	mux.HandleFunc("/admin/stats", h.handleStats)
	mux.HandleFunc("/admin/status", h.handleStatus)
}

// handleChains handles requests to /admin/chains
func (h *MultiChainAdminHandler) handleChains(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		h.listChains(w, r)
	case "POST":
		h.createChain(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleChain handles requests to /admin/chains/{chainName}
func (h *MultiChainAdminHandler) handleChain(w http.ResponseWriter, r *http.Request) {
	chainName := h.extractChainNameFromPath(r.URL.Path, "/admin/chains/")
	if chainName == "" {
		http.Error(w, "Invalid chain name", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.getChain(w, r, chainName)
	case "PUT":
		h.updateChain(w, r, chainName)
	case "DELETE":
		h.deleteChain(w, r, chainName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleChainEndpoints handles requests to /admin/chains/{chainName}/endpoints
func (h *MultiChainAdminHandler) handleChainEndpoints(w http.ResponseWriter, r *http.Request) {
	chainName := h.extractChainNameFromPath(r.URL.Path, "/admin/chains/")
	chainName = strings.Split(chainName, "/")[0] // Remove /endpoints part
	
	if chainName == "" {
		http.Error(w, "Invalid chain name", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.listChainEndpoints(w, r, chainName)
	case "POST":
		h.createChainEndpoint(w, r, chainName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleChainEndpoint handles requests to /admin/chains/{chainName}/endpoints/{endpointId}
func (h *MultiChainAdminHandler) handleChainEndpoint(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 5 {
		http.Error(w, "Invalid endpoint path", http.StatusBadRequest)
		return
	}
	
	chainName := parts[2]
	endpointIDStr := parts[4]
	endpointID, err := strconv.Atoi(endpointIDStr)
	if err != nil {
		http.Error(w, "Invalid endpoint ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.getChainEndpoint(w, r, chainName, endpointID)
	case "PUT":
		h.updateChainEndpoint(w, r, chainName, endpointID)
	case "DELETE":
		h.deleteChainEndpoint(w, r, chainName, endpointID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleChainConfig handles chain-specific configuration
func (h *MultiChainAdminHandler) handleChainConfig(w http.ResponseWriter, r *http.Request) {
	chainName := h.extractChainNameFromPath(r.URL.Path, "/admin/chains/")
	chainName = strings.Split(chainName, "/")[0] // Remove /config part
	
	if chainName == "" {
		http.Error(w, "Invalid chain name", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		h.getChainConfig(w, r, chainName)
	case "PUT":
		h.updateChainConfig(w, r, chainName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleHealthOverview provides overall health status across all chains
func (h *MultiChainAdminHandler) handleHealthOverview(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chainStatuses := h.multiChainHealthChecker.GetAllChainStatuses()
	stats := h.multiChainHealthChecker.GetHealthCheckStats()

	response := map[string]interface{}{
		"chains": chainStatuses,
		"stats":  stats,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleChainHealthDetails provides detailed health information for a specific chain
func (h *MultiChainAdminHandler) handleChainHealthDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chainName := h.extractChainNameFromPath(r.URL.Path, "/admin/health/")
	if chainName == "" {
		http.Error(w, "Invalid chain name", http.StatusBadRequest)
		return
	}

	status := h.multiChainHealthChecker.GetChainStatus(chainName)
	if status == nil {
		http.Error(w, fmt.Sprintf("Chain %s not found", chainName), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleStats provides comprehensive statistics
func (h *MultiChainAdminHandler) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := map[string]interface{}{
		"health_check": h.multiChainHealthChecker.GetHealthCheckStats(),
		"supported_chains": h.multiChainHealthChecker.GetSupportedChains(),
		"server_info": map[string]interface{}{
			"version": "1.0.0",
			"mode":    "multi-chain",
			"uptime":  "calculated_uptime_placeholder",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleStatus provides real-time status information
func (h *MultiChainAdminHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	chainStatuses := h.multiChainHealthChecker.GetAllChainStatuses()
	totalHealthyChains := 0
	totalChains := len(chainStatuses)
	
	for _, status := range chainStatuses {
		if status.HealthyCount > 0 {
			totalHealthyChains++
		}
	}

	overallStatus := "healthy"
	if totalHealthyChains == 0 {
		overallStatus = "unhealthy"
	} else if totalHealthyChains < totalChains {
		overallStatus = "degraded"
	}

	response := map[string]interface{}{
		"status":              overallStatus,
		"total_chains":        totalChains,
		"healthy_chains":      totalHealthyChains,
		"degraded_chains":     totalChains - totalHealthyChains,
		"chains":              chainStatuses,
	}

	// Set appropriate HTTP status code
	if overallStatus == "unhealthy" {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else if overallStatus == "degraded" {
		w.WriteHeader(http.StatusPartialContent)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Implementation methods

func (h *MultiChainAdminHandler) listChains(w http.ResponseWriter, r *http.Request) {
	chains := h.config.Chains
	
	response := map[string]interface{}{
		"chains": chains,
		"total":  len(chains),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MultiChainAdminHandler) createChain(w http.ResponseWriter, r *http.Request) {
	// Placeholder implementation
	http.Error(w, "Chain creation not implemented yet", http.StatusNotImplemented)
}

func (h *MultiChainAdminHandler) getChain(w http.ResponseWriter, r *http.Request, chainName string) {
	chain := h.config.GetChainByName(chainName)
	if chain == nil {
		http.Error(w, fmt.Sprintf("Chain %s not found", chainName), http.StatusNotFound)
		return
	}

	endpoints := h.config.ChainEndpoints[chainName]
	configs := h.config.ChainConfigs[chainName]

	response := map[string]interface{}{
		"chain":     chain,
		"endpoints": endpoints,
		"configs":   configs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MultiChainAdminHandler) updateChain(w http.ResponseWriter, r *http.Request, chainName string) {
	// Placeholder implementation
	http.Error(w, "Chain update not implemented yet", http.StatusNotImplemented)
}

func (h *MultiChainAdminHandler) deleteChain(w http.ResponseWriter, r *http.Request, chainName string) {
	// Placeholder implementation
	http.Error(w, "Chain deletion not implemented yet", http.StatusNotImplemented)
}

func (h *MultiChainAdminHandler) listChainEndpoints(w http.ResponseWriter, r *http.Request, chainName string) {
	if !h.multiChainHealthChecker.IsChainSupported(chainName) {
		http.Error(w, fmt.Sprintf("Chain %s not found", chainName), http.StatusNotFound)
		return
	}

	endpoints := h.multiChainHealthChecker.GetAllEndpointsForChain(chainName)
	healthyEndpoints := h.multiChainHealthChecker.GetHealthyEndpointsForChain(chainName)

	response := map[string]interface{}{
		"chain_name":        chainName,
		"total_endpoints":   len(endpoints),
		"healthy_endpoints": len(healthyEndpoints),
		"endpoints":         endpoints,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MultiChainAdminHandler) createChainEndpoint(w http.ResponseWriter, r *http.Request, chainName string) {
	// Placeholder implementation
	http.Error(w, "Endpoint creation not implemented yet", http.StatusNotImplemented)
}

func (h *MultiChainAdminHandler) getChainEndpoint(w http.ResponseWriter, r *http.Request, chainName string, endpointID int) {
	// Placeholder implementation
	http.Error(w, "Individual endpoint retrieval not implemented yet", http.StatusNotImplemented)
}

func (h *MultiChainAdminHandler) updateChainEndpoint(w http.ResponseWriter, r *http.Request, chainName string, endpointID int) {
	// Placeholder implementation
	http.Error(w, "Endpoint update not implemented yet", http.StatusNotImplemented)
}

func (h *MultiChainAdminHandler) deleteChainEndpoint(w http.ResponseWriter, r *http.Request, chainName string, endpointID int) {
	// Placeholder implementation
	http.Error(w, "Endpoint deletion not implemented yet", http.StatusNotImplemented)
}

func (h *MultiChainAdminHandler) getChainConfig(w http.ResponseWriter, r *http.Request, chainName string) {
	if !h.multiChainHealthChecker.IsChainSupported(chainName) {
		http.Error(w, fmt.Sprintf("Chain %s not found", chainName), http.StatusNotFound)
		return
	}

	configs := h.config.ChainConfigs[chainName]
	if configs == nil {
		configs = make(map[string]string)
	}

	response := map[string]interface{}{
		"chain_name": chainName,
		"configs":    configs,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *MultiChainAdminHandler) updateChainConfig(w http.ResponseWriter, r *http.Request, chainName string) {
	// Placeholder implementation
	http.Error(w, "Chain config update not implemented yet", http.StatusNotImplemented)
}

// Helper methods

func (h *MultiChainAdminHandler) extractChainNameFromPath(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	
	remainder := strings.TrimPrefix(path, prefix)
	parts := strings.Split(remainder, "/")
	if len(parts) == 0 {
		return ""
	}
	
	return parts[0]
}

func (h *MultiChainAdminHandler) writeJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *MultiChainAdminHandler) writeErrorResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	response := map[string]interface{}{
		"error":   true,
		"message": message,
		"code":    code,
	}
	
	json.NewEncoder(w).Encode(response)
}