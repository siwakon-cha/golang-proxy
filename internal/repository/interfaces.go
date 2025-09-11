package repository

import "rpc-proxy/internal/types"

type RPCEndpointRepository interface {
	GetAll() ([]*types.RPCEndpoint, error)
	GetEnabled() ([]*types.RPCEndpoint, error)
	GetEnabledByChain(chainName string) ([]*types.RPCEndpoint, error)
	GetAllByChain(chainName string) ([]*types.RPCEndpoint, error)
	GetByID(id int) (*types.RPCEndpoint, error)
	GetByName(name string) (*types.RPCEndpoint, error)
	Create(endpoint *CreateRPCEndpointRequest) (*types.RPCEndpoint, error)
	Update(id int, endpoint *UpdateRPCEndpointRequest) (*types.RPCEndpoint, error)
	Delete(id int) error
	SetEnabled(id int, enabled bool) error
	UpdateHealthStatus(id int, healthy bool, responseTime int64, blockNumber string, errorMsg string) error
}

type ChainRepository interface {
	GetAll() ([]*types.Chain, error)
	GetByName(name string) (*types.Chain, error)
	GetByChainID(chainID int) (*types.Chain, error)
	GetByRPCPath(rpcPath string) (*types.Chain, error)
	Create(chain *types.Chain) error
	Update(chain *types.Chain) error
	Delete(id int) error
}

type ChainConfigRepository interface {
	GetByChainID(chainID int) (map[string]string, error)
	GetByChainName(chainName string) (map[string]string, error)
	GetConfig(chainID int, configKey string) (string, error)
	GetAll() ([]*types.ChainConfig, error)
	SetConfig(chainID int, configKey, configValue, description string) error
	DeleteConfig(chainID int, configKey string) error
	DeleteAllByChainID(chainID int) error
}

type SettingsRepository interface {
	Get(key string) (string, error)
	Set(key, value, description string) error
	GetAll() (map[string]string, error)
	Delete(key string) error
}

type HealthCheckRepository interface {
	Create(healthCheck *CreateHealthCheckRequest) error
	GetByEndpointID(endpointID int, limit int) ([]*HealthCheck, error)
	GetLatestByEndpointID(endpointID int) (*HealthCheck, error)
	DeleteOldRecords(days int) error
}

// Request/Response types
type CreateRPCEndpointRequest struct {
	Name    string `json:"name" validate:"required,min=1,max=100"`
	URL     string `json:"url" validate:"required,url,max=500"`
	Weight  int    `json:"weight" validate:"min=1,max=100"`
	Enabled bool   `json:"enabled"`
}

type UpdateRPCEndpointRequest struct {
	Name    *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	URL     *string `json:"url,omitempty" validate:"omitempty,url,max=500"`
	Weight  *int    `json:"weight,omitempty" validate:"omitempty,min=1,max=100"`
	Enabled *bool   `json:"enabled,omitempty"`
}

type CreateHealthCheckRequest struct {
	EndpointID     int    `json:"endpointId"`
	Healthy        bool   `json:"healthy"`
	ResponseTimeMs int64  `json:"responseTimeMs"`
	BlockNumber    string `json:"blockNumber"`
	ErrorMessage   string `json:"errorMessage"`
}

type HealthCheck struct {
	ID             int    `json:"id" db:"id"`
	EndpointID     int    `json:"endpointId" db:"endpoint_id"`
	Healthy        bool   `json:"healthy" db:"healthy"`
	ResponseTimeMs int64  `json:"responseTimeMs" db:"response_time_ms"`
	BlockNumber    string `json:"blockNumber" db:"block_number"`
	ErrorMessage   string `json:"errorMessage" db:"error_message"`
	CheckedAt      string `json:"checkedAt" db:"checked_at"`
}

type Setting struct {
	Key         string `json:"key" db:"key"`
	Value       string `json:"value" db:"value"`
	Description string `json:"description" db:"description"`
	UpdatedAt   string `json:"updatedAt" db:"updated_at"`
}