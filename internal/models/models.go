package models

import (
	"sync"
	"time"

	"gorm.io/gorm"
)

// Chain represents a blockchain network
type Chain struct {
	ID                   uint      `json:"id" gorm:"primaryKey"`
	ChainID              int       `json:"chainId" gorm:"uniqueIndex;not null"`
	Name                 string    `json:"name" gorm:"uniqueIndex;size:50;not null"`
	DisplayName          string    `json:"displayName" gorm:"size:100;not null"`
	RPCPath              string    `json:"rpcPath" gorm:"uniqueIndex;size:50;not null"`
	IsTestnet            bool      `json:"isTestnet" gorm:"default:false"`
	IsEnabled            bool      `json:"isEnabled" gorm:"default:true;index"`
	NativeCurrencySymbol string    `json:"nativeCurrencySymbol" gorm:"size:10;default:'ETH'"`
	BlockExplorerURL     string    `json:"blockExplorerUrl" gorm:"size:500"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`

	// Relationships
	RPCEndpoints []RPCEndpoint `json:"rpcEndpoints,omitempty" gorm:"foreignKey:ChainID;constraint:OnDelete:CASCADE"`
	ChainConfigs []ChainConfig `json:"chainConfigs,omitempty" gorm:"foreignKey:ChainID;constraint:OnDelete:CASCADE"`
}

// ChainConfig represents chain-specific configuration
type ChainConfig struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ChainID     uint      `json:"chainId" gorm:"not null;index"`
	ConfigKey   string    `json:"configKey" gorm:"size:100;not null"`
	ConfigValue string    `json:"configValue" gorm:"not null;type:text"`
	Description string    `json:"description" gorm:"type:text"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`

	// Relationships
	Chain Chain `json:"chain,omitempty" gorm:"foreignKey:ChainID"`
}

// RPCEndpoint represents an RPC endpoint in the database
type RPCEndpoint struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Name      string    `json:"name" gorm:"size:100;not null"`
	URL       string    `json:"url" gorm:"size:500;not null"`
	Weight    int       `json:"weight" gorm:"default:1;check:weight > 0"`
	Enabled   bool      `json:"enabled" gorm:"default:true;index"`
	ChainID   uint      `json:"chainId" gorm:"not null;index"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Runtime fields (not stored in database)
	Healthy      bool         `json:"healthy" gorm:"-"`
	LastCheck    time.Time    `json:"lastCheck" gorm:"-"`
	ResponseTime int64        `json:"responseTime" gorm:"-"`
	BlockNumber  string       `json:"blockNumber" gorm:"-"`
	FailCount    int          `json:"-" gorm:"-"`
	mu           sync.RWMutex `json:"-" gorm:"-"`

	// Relationships
	Chain        Chain         `json:"chain,omitempty" gorm:"foreignKey:ChainID"`
	HealthChecks []HealthCheck `json:"healthChecks,omitempty" gorm:"foreignKey:EndpointID;constraint:OnDelete:CASCADE"`
}

// Thread-safe methods for runtime fields
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

// HealthCheck represents a health check record
type HealthCheck struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	EndpointID     uint      `json:"endpointId" gorm:"not null;index"`
	Healthy        bool      `json:"healthy" gorm:"not null"`
	ResponseTimeMs int64     `json:"responseTimeMs"`
	BlockNumber    string    `json:"blockNumber" gorm:"size:20"`
	ErrorMessage   string    `json:"errorMessage" gorm:"type:text"`
	CheckedAt      time.Time `json:"checkedAt" gorm:"index;default:CURRENT_TIMESTAMP"`

	// Relationships
	Endpoint RPCEndpoint `json:"endpoint,omitempty" gorm:"foreignKey:EndpointID"`
}

// Setting represents a configuration setting
type Setting struct {
	Key         string    `json:"key" gorm:"primaryKey;size:100"`
	Value       string    `json:"value" gorm:"not null;type:text"`
	Description string    `json:"description" gorm:"type:text"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// GORM hooks for Chain
func (c *Chain) BeforeCreate(tx *gorm.DB) error {
	c.CreatedAt = time.Now()
	c.UpdatedAt = time.Now()
	return nil
}

func (c *Chain) BeforeUpdate(tx *gorm.DB) error {
	c.UpdatedAt = time.Now()
	return nil
}

// GORM hooks for ChainConfig
func (cc *ChainConfig) BeforeCreate(tx *gorm.DB) error {
	cc.CreatedAt = time.Now()
	cc.UpdatedAt = time.Now()
	return nil
}

func (cc *ChainConfig) BeforeUpdate(tx *gorm.DB) error {
	cc.UpdatedAt = time.Now()
	return nil
}

// GORM hooks for RPCEndpoint
func (e *RPCEndpoint) BeforeCreate(tx *gorm.DB) error {
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	return nil
}

func (e *RPCEndpoint) BeforeUpdate(tx *gorm.DB) error {
	e.UpdatedAt = time.Now()
	return nil
}

// GORM hooks for HealthCheck
func (h *HealthCheck) BeforeCreate(tx *gorm.DB) error {
	if h.CheckedAt.IsZero() {
		h.CheckedAt = time.Now()
	}
	return nil
}

// GORM hooks for Setting
func (s *Setting) BeforeCreate(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return nil
}

func (s *Setting) BeforeUpdate(tx *gorm.DB) error {
	s.UpdatedAt = time.Now()
	return nil
}

// Migration function to run auto-migration
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&Chain{},
		&ChainConfig{},
		&RPCEndpoint{},
		&HealthCheck{},
		&Setting{},
	)
}

// Seed function to insert default data
func SeedDefaultData(db *gorm.DB) error {
	// Insert default settings
	defaultSettings := []Setting{
		{Key: "health_check_interval", Value: "30s", Description: "Interval between health checks"},
		{Key: "health_check_timeout", Value: "5s", Description: "Timeout for each health check"},
		{Key: "health_check_retries", Value: "3", Description: "Number of retries before marking endpoint unhealthy"},
		{Key: "proxy_timeout", Value: "10s", Description: "Timeout for proxy requests"},
		{Key: "max_connections", Value: "1000", Description: "Maximum concurrent connections"},
		{Key: "server_port", Value: "8080", Description: "Server port number"},
	}

	for _, setting := range defaultSettings {
		var existingSetting Setting
		if err := db.Where("key = ?", setting.Key).First(&existingSetting).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				if err := db.Create(&setting).Error; err != nil {
					return err
				}
			} else {
				return err
			}
		}
	}

	return nil
}