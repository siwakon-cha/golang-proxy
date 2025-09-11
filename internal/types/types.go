package types

import (
	"sync"
	"time"
)

// ChainIdentifier represents common chain identifiers
type ChainIdentifier struct {
	ChainID   int    `json:"chainId"`
	Name      string `json:"name"`
	RPCPath   string `json:"rpcPath"`
}

// Supported chains constants
var (
	ChainEthereum = ChainIdentifier{ChainID: 1, Name: "ethereum", RPCPath: "ethereum"}
	ChainSepolia  = ChainIdentifier{ChainID: 11155111, Name: "sepolia", RPCPath: "sepolia"}
	ChainMinato   = ChainIdentifier{ChainID: 1946, Name: "minato", RPCPath: "minato"}
	ChainSoneium  = ChainIdentifier{ChainID: 1868, Name: "soneium", RPCPath: "soneium"}

	SupportedChains = map[string]ChainIdentifier{
		"ethereum": ChainEthereum,
		"sepolia":  ChainSepolia,
		"minato":   ChainMinato,
		"soneium":  ChainSoneium,
	}

	ChainIDToName = map[int]string{
		1:        "ethereum",
		11155111: "sepolia",
		1946:     "minato",
		1868:     "soneium",
	}
)

// Chain represents supported blockchain networks
type Chain struct {
	ID                     int       `json:"id" db:"id"`
	ChainID                int       `json:"chainId" db:"chain_id"`
	Name                   string    `json:"name" db:"name"`
	DisplayName            string    `json:"displayName" db:"display_name"`
	RPCPath                string    `json:"rpcPath" db:"rpc_path"`
	IsTestnet              bool      `json:"isTestnet" db:"is_testnet"`
	IsEnabled              bool      `json:"isEnabled" db:"is_enabled"`
	NativeCurrencySymbol   string    `json:"nativeCurrencySymbol" db:"native_currency_symbol"`
	NativeCurrencyDecimals int       `json:"nativeCurrencyDecimals" db:"native_currency_decimals"`
	BlockExplorerURL       string    `json:"blockExplorerUrl" db:"block_explorer_url"`
	CreatedAt              time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt              time.Time `json:"updatedAt" db:"updated_at"`
}

// ChainConfig represents chain-specific configuration
type ChainConfig struct {
	ID          int       `json:"id" db:"id"`
	ChainID     int       `json:"chainId" db:"chain_id"`
	ConfigKey   string    `json:"configKey" db:"config_key"`
	ConfigValue string    `json:"configValue" db:"config_value"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type RPCEndpoint struct {
	ID           int       `json:"id" db:"id"`
	Name         string    `json:"name" db:"name"`
	URL          string    `json:"url" db:"url" yaml:"url"`
	Weight       int       `json:"weight" db:"weight" yaml:"weight"`
	Enabled      bool      `json:"enabled" db:"enabled"`
	ChainID      int       `json:"chainId" db:"chain_id"`
	ChainName    string    `json:"chainName" db:"-"` // Populated from join
	Healthy      bool      `json:"healthy"`
	LastCheck    time.Time `json:"lastCheck"`
	ResponseTime int64     `json:"responseTime"`
	BlockNumber  string    `json:"blockNumber"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at"`
	FailCount    int       `json:"-"`
	mu           sync.RWMutex
}

func (e *RPCEndpoint) SetHealthy(healthy bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.Healthy = healthy
	e.LastCheck = time.Now()
	if healthy {
		e.FailCount = 0
	} else {
		e.FailCount++
	}
}

func (e *RPCEndpoint) IsHealthy() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.Healthy
}

func (e *RPCEndpoint) GetFailCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.FailCount
}

func (e *RPCEndpoint) SetResponseTime(rt int64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.ResponseTime = rt
}

func (e *RPCEndpoint) SetBlockNumber(bn string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.BlockNumber = bn
}

func (e *RPCEndpoint) IncrementFailCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.FailCount++
	return e.FailCount
}

type JSONRPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}   `json:"id"`
}

type JSONRPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ChainHealthStatus represents health status for a specific chain
type ChainHealthStatus struct {
	Chain              *Chain         `json:"chain"`
	HealthyEndpoints   []*RPCEndpoint `json:"healthyEndpoints"`
	UnhealthyEndpoints []*RPCEndpoint `json:"unhealthyEndpoints"`
	TotalEndpoints     int            `json:"totalEndpoints"`
	HealthyCount       int            `json:"healthyCount"`
	CurrentRPC         string         `json:"currentRPC"`
}

// MultiChainHealthStatus represents overall proxy health status
type MultiChainHealthStatus struct {
	Proxy      string                        `json:"proxy"`
	TotalChains int                          `json:"totalChains"`
	HealthyChains int                        `json:"healthyChains"`
	Chains     map[string]*ChainHealthStatus `json:"chains"`
	Timestamp  time.Time                     `json:"timestamp"`
}

// Legacy HealthStatus for backward compatibility
type HealthStatus struct {
	Proxy        string        `json:"proxy"`
	CurrentRPC   string        `json:"currentRPC"`
	RPCEndpoints []*RPCEndpoint `json:"rpcEndpoints"`
	Chain        string        `json:"chain,omitempty"`
}