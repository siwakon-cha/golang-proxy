package health

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"rpc-proxy/internal/types"
)

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Interval time.Duration
	Timeout  time.Duration
	Retries  int
}

type Checker struct {
	endpoints   []*types.RPCEndpoint
	config      HealthCheckConfig
	client      *http.Client
	stopChan    chan bool
	running     bool
	mu          sync.RWMutex
}

func NewChecker(endpoints []*types.RPCEndpoint, config HealthCheckConfig) *Checker {
	return &Checker{
		endpoints: endpoints,
		config:    config,
		client: &http.Client{
			Timeout: config.Timeout,
		},
		stopChan: make(chan bool),
	}
}

func (hc *Checker) Start() {
	hc.mu.Lock()
	if hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = true
	hc.mu.Unlock()

	go hc.healthCheckLoop()
}

func (hc *Checker) Stop() {
	hc.mu.Lock()
	if !hc.running {
		hc.mu.Unlock()
		return
	}
	hc.running = false
	hc.mu.Unlock()

	close(hc.stopChan)
}

func (hc *Checker) healthCheckLoop() {
	ticker := time.NewTicker(hc.config.Interval)
	defer ticker.Stop()

	hc.performHealthCheck()

	for {
		select {
		case <-ticker.C:
			hc.performHealthCheck()
		case <-hc.stopChan:
			return
		}
	}
}

func (hc *Checker) performHealthCheck() {
	var wg sync.WaitGroup
	
	for _, endpoint := range hc.endpoints {
		wg.Add(1)
		go func(ep *types.RPCEndpoint) {
			defer wg.Done()
			hc.checkEndpoint(ep)
		}(endpoint)
	}
	
	wg.Wait()
}

func (hc *Checker) checkEndpoint(endpoint *types.RPCEndpoint) {
	start := time.Now()
	
	reqBody := types.JSONRPCRequest{
		Jsonrpc: "2.0",
		Method:  "eth_blockNumber",
		Params:  []interface{}{},
		ID:      1,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Printf("Failed to marshal health check request for %s: %v", endpoint.URL, err)
		hc.markUnhealthy(endpoint, 0, "")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Failed to create health check request for %s: %v", endpoint.URL, err)
		hc.markUnhealthy(endpoint, 0, "")
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := hc.client.Do(req)
	if err != nil {
		log.Printf("Health check failed for %s: %v", endpoint.URL, err)
		hc.markUnhealthy(endpoint, 0, "")
		return
	}
	defer resp.Body.Close()

	responseTime := time.Since(start).Milliseconds()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Health check returned non-200 status for %s: %d", endpoint.URL, resp.StatusCode)
		hc.markUnhealthy(endpoint, responseTime, "")
		return
	}

	var rpcResp types.JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		log.Printf("Failed to decode health check response for %s: %v", endpoint.URL, err)
		hc.markUnhealthy(endpoint, responseTime, "")
		return
	}

	if rpcResp.Error != nil {
		log.Printf("RPC error in health check for %s: %s", endpoint.URL, rpcResp.Error.Message)
		hc.markUnhealthy(endpoint, responseTime, "")
		return
	}

	blockNumber, ok := rpcResp.Result.(string)
	if !ok {
		log.Printf("Invalid block number format in health check for %s", endpoint.URL)
		hc.markUnhealthy(endpoint, responseTime, "")
		return
	}

	hc.markHealthy(endpoint, responseTime, blockNumber)
}

func (hc *Checker) markHealthy(endpoint *types.RPCEndpoint, responseTime int64, blockNumber string) {
	wasUnhealthy := !endpoint.IsHealthy()
	endpoint.SetHealthy(true)
	endpoint.SetResponseTime(responseTime)
	endpoint.SetBlockNumber(blockNumber)

	if wasUnhealthy {
		log.Printf("Endpoint %s is now healthy (response time: %dms, block: %s)", 
			endpoint.URL, responseTime, blockNumber)
	}
}

func (hc *Checker) markUnhealthy(endpoint *types.RPCEndpoint, responseTime int64, blockNumber string) {
	wasHealthy := endpoint.IsHealthy()
	endpoint.SetResponseTime(responseTime)
	endpoint.SetBlockNumber(blockNumber)
	
	// Increment fail count
	failCount := endpoint.IncrementFailCount()

	if failCount >= hc.config.Retries {
		endpoint.SetHealthy(false)
		if wasHealthy {
			log.Printf("Endpoint %s marked as unhealthy after %d consecutive failures", 
				endpoint.URL, hc.config.Retries)
		}
	}
}

func (hc *Checker) GetHealthyEndpoints() []*types.RPCEndpoint {
	var healthy []*types.RPCEndpoint
	for _, endpoint := range hc.endpoints {
		if endpoint.IsHealthy() {
			healthy = append(healthy, endpoint)
		}
	}
	return healthy
}

func (hc *Checker) GetAllEndpoints() []*types.RPCEndpoint {
	return hc.endpoints
}