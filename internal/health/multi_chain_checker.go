package health

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"rpc-proxy/internal/types"
)

// ChainConfig represents configuration for a single chain
type ChainConfig struct {
	Chain     *types.Chain
	Endpoints []*types.RPCEndpoint
}

// MultiChainChecker manages health checks for multiple blockchain networks
type MultiChainChecker struct {
	chains        map[string]*ChainConfig
	healthConfig  HealthCheckConfig
	client        *http.Client
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	mu            sync.RWMutex
	isRunning     bool
}

// NewMultiChainChecker creates a new multi-chain health checker
func NewMultiChainChecker(chains map[string]*ChainConfig, healthConfig HealthCheckConfig) *MultiChainChecker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MultiChainChecker{
		chains:       chains,
		healthConfig: healthConfig,
		client: &http.Client{
			Timeout: healthConfig.Timeout,
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start begins health checking for all chains
func (mc *MultiChainChecker) Start() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if mc.isRunning {
		return
	}
	
	mc.isRunning = true
	log.Printf("Starting multi-chain health checker for %d chains", len(mc.chains))
	
	// Start health checker for each chain
	for chainName, chainConfig := range mc.chains {
		mc.wg.Add(1)
		go mc.runChainHealthChecker(chainName, chainConfig)
	}
}

// Stop stops all health checking
func (mc *MultiChainChecker) Stop() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	if !mc.isRunning {
		return
	}
	
	mc.isRunning = false
	mc.cancel()
	mc.wg.Wait()
	log.Printf("Multi-chain health checker stopped")
}

// GetHealthyEndpoints returns healthy endpoints for a specific chain
func (mc *MultiChainChecker) GetHealthyEndpoints(chainName string) []*types.RPCEndpoint {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	chainConfig, exists := mc.chains[chainName]
	if !exists {
		return nil
	}
	
	var healthy []*types.RPCEndpoint
	for _, endpoint := range chainConfig.Endpoints {
		if endpoint.IsHealthy() {
			healthy = append(healthy, endpoint)
		}
	}
	
	return healthy
}

// GetAllEndpoints returns all endpoints for a specific chain
func (mc *MultiChainChecker) GetAllEndpoints(chainName string) []*types.RPCEndpoint {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	chainConfig, exists := mc.chains[chainName]
	if !exists {
		return nil
	}
	
	return chainConfig.Endpoints
}

// GetMultiChainStatus returns comprehensive health status for all chains
func (mc *MultiChainChecker) GetMultiChainStatus() *types.MultiChainHealthStatus {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	status := &types.MultiChainHealthStatus{
		Chains:        make(map[string]*types.ChainHealthStatus),
		TotalChains:   len(mc.chains),
		HealthyChains: 0,
		Timestamp:     time.Now(),
	}
	
	for chainName, chainConfig := range mc.chains {
		chainStatus := mc.getChainHealthStatus(chainName, chainConfig)
		status.Chains[chainName] = chainStatus
		
		if chainStatus.HealthyCount > 0 {
			status.HealthyChains++
		}
	}
	
	if status.HealthyChains > 0 {
		status.Proxy = "healthy"
	} else {
		status.Proxy = "unhealthy"
	}
	
	return status
}

// GetChainStatus returns health status for a specific chain
func (mc *MultiChainChecker) GetChainStatus(chainName string) *types.ChainHealthStatus {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	
	chainConfig, exists := mc.chains[chainName]
	if !exists {
		return nil
	}
	
	return mc.getChainHealthStatus(chainName, chainConfig)
}

// runChainHealthChecker runs health checking loop for a specific chain
func (mc *MultiChainChecker) runChainHealthChecker(chainName string, chainConfig *ChainConfig) {
	defer mc.wg.Done()
	
	log.Printf("Started health checker for chain: %s", chainName)
	ticker := time.NewTicker(mc.healthConfig.Interval)
	defer ticker.Stop()
	
	// Initial health check
	mc.checkChainHealth(chainName, chainConfig)
	
	for {
		select {
		case <-mc.ctx.Done():
			log.Printf("Health checker for chain %s stopped", chainName)
			return
		case <-ticker.C:
			mc.checkChainHealth(chainName, chainConfig)
		}
	}
}

// checkChainHealth performs health check for all endpoints in a chain
func (mc *MultiChainChecker) checkChainHealth(chainName string, chainConfig *ChainConfig) {
	log.Printf("Checking health for chain: %s (%d endpoints)", chainName, len(chainConfig.Endpoints))
	
	var wg sync.WaitGroup
	for _, endpoint := range chainConfig.Endpoints {
		if !endpoint.Enabled {
			continue
		}
		
		wg.Add(1)
		go func(ep *types.RPCEndpoint) {
			defer wg.Done()
			mc.checkEndpointHealth(chainName, ep)
		}(endpoint)
	}
	wg.Wait()
	
	// Log chain health summary
	healthy := mc.GetHealthyEndpoints(chainName)
	log.Printf("Chain %s health check completed: %d/%d endpoints healthy", 
		chainName, len(healthy), len(chainConfig.Endpoints))
}

// checkEndpointHealth performs health check for a single endpoint
func (mc *MultiChainChecker) checkEndpointHealth(chainName string, endpoint *types.RPCEndpoint) {
	start := time.Now()
	
	// Create health check request (get latest block)
	requestBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
		"id":      1,
	}
	
	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("Failed to marshal request for %s: %v", endpoint.URL, err)
		endpoint.SetHealthy(false)
		return
	}
	
	// Create HTTP request with timeout
	ctx, cancel := context.WithTimeout(mc.ctx, mc.healthConfig.Timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewReader(jsonBody))
	if err != nil {
		log.Printf("Failed to create request for %s: %v", endpoint.URL, err)
		endpoint.SetHealthy(false)
		return
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	// Perform request with retries
	var lastErr error
	for attempt := 0; attempt < mc.healthConfig.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-time.After(time.Second):
			case <-ctx.Done():
				endpoint.SetHealthy(false)
				return
			}
		}
		
		resp, err := mc.client.Do(req)
		if err != nil {
			lastErr = err
			log.Printf("Health check attempt %d/%d failed for %s: %v", 
				attempt+1, mc.healthConfig.Retries, endpoint.URL, err)
			continue
		}
		
		// Process response
		if mc.processHealthCheckResponse(endpoint, resp, start) {
			return
		}
		
		lastErr = fmt.Errorf("invalid response from %s", endpoint.URL)
	}
	
	// All retries failed
	endpoint.SetHealthy(false)
	responseTime := time.Since(start).Milliseconds()
	endpoint.SetResponseTime(responseTime)
	
	log.Printf("Health check failed for %s after %d attempts: %v", 
		endpoint.URL, mc.healthConfig.Retries, lastErr)
}

// processHealthCheckResponse processes the health check response
func (mc *MultiChainChecker) processHealthCheckResponse(endpoint *types.RPCEndpoint, resp *http.Response, start time.Time) bool {
	defer resp.Body.Close()
	
	responseTime := time.Since(start).Milliseconds()
	endpoint.SetResponseTime(responseTime)
	
	if resp.StatusCode != http.StatusOK {
		log.Printf("Health check failed for %s: HTTP %d", endpoint.URL, resp.StatusCode)
		endpoint.SetHealthy(false)
		return true
	}
	
	// Parse JSON response
	var jsonResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&jsonResp); err != nil {
		log.Printf("Failed to decode response from %s: %v", endpoint.URL, err)
		endpoint.SetHealthy(false)
		return true
	}
	
	// Check for JSON-RPC error
	if errorObj, exists := jsonResp["error"]; exists && errorObj != nil {
		log.Printf("JSON-RPC error from %s: %v", endpoint.URL, errorObj)
		endpoint.SetHealthy(false)
		return true
	}
	
	// Extract block number
	if result, exists := jsonResp["result"]; exists && result != nil {
		if blockHex, ok := result.(string); ok && strings.HasPrefix(blockHex, "0x") {
			if blockNum, err := strconv.ParseInt(blockHex[2:], 16, 64); err == nil {
				endpoint.SetBlockNumber(fmt.Sprintf("%d", blockNum))
				endpoint.SetHealthy(true)
				log.Printf("Health check passed for %s: block %d, response time %dms", 
					endpoint.URL, blockNum, responseTime)
				return true
			}
		}
	}
	
	log.Printf("Invalid block number response from %s", endpoint.URL)
	endpoint.SetHealthy(false)
	return true
}

// getChainHealthStatus creates health status for a chain (must be called with lock held)
func (mc *MultiChainChecker) getChainHealthStatus(chainName string, chainConfig *ChainConfig) *types.ChainHealthStatus {
	var healthyEndpoints []*types.RPCEndpoint
	var unhealthyEndpoints []*types.RPCEndpoint
	var currentRPC string
	
	for _, endpoint := range chainConfig.Endpoints {
		if endpoint.IsHealthy() {
			healthyEndpoints = append(healthyEndpoints, endpoint)
			if currentRPC == "" && endpoint.Enabled {
				currentRPC = endpoint.URL
			}
		} else {
			unhealthyEndpoints = append(unhealthyEndpoints, endpoint)
		}
	}
	
	return &types.ChainHealthStatus{
		Chain:              chainConfig.Chain,
		HealthyEndpoints:   healthyEndpoints,
		UnhealthyEndpoints: unhealthyEndpoints,
		TotalEndpoints:     len(chainConfig.Endpoints),
		HealthyCount:       len(healthyEndpoints),
		CurrentRPC:         currentRPC,
	}
}

// AddChain adds a new chain to be monitored (thread-safe)
func (mc *MultiChainChecker) AddChain(chainName string, chainConfig *ChainConfig) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	mc.chains[chainName] = chainConfig
	
	if mc.isRunning {
		mc.wg.Add(1)
		go mc.runChainHealthChecker(chainName, chainConfig)
	}
	
	log.Printf("Added chain %s to health checker", chainName)
}

// RemoveChain removes a chain from monitoring (thread-safe)
func (mc *MultiChainChecker) RemoveChain(chainName string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	
	delete(mc.chains, chainName)
	log.Printf("Removed chain %s from health checker", chainName)
}

