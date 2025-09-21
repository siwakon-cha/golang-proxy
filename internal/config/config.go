package config

import (
	"fmt"
	"log"
	"strings"
	"time"

	"rpc-proxy/internal/database"
	"rpc-proxy/internal/health"
	"rpc-proxy/internal/repository/gorm"
	"rpc-proxy/internal/types"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server      ServerConfig
	Database    DatabaseConfig
	HealthCheck health.HealthCheckConfig
	Proxy       ProxyConfig
	App         AppConfig

	// Multi-chain runtime fields loaded from database
	Chains         []*types.Chain
	ChainEndpoints map[string][]*types.RPCEndpoint // chainName -> endpoints
	ChainConfigs   map[string]map[string]string    // chainName -> configKey -> configValue

	// Legacy single-chain support (deprecated)
	RPCEndpoints []*types.RPCEndpoint
}

type ServerConfig struct {
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type ProxyConfig struct {
	Timeout        time.Duration
	MaxConnections int
}

type AppConfig struct {
	Environment          string
	LogLevel             string
	FallbackRPCEndpoints []string
}

func Load() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables: %v", err)
	}

	// Configure viper
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set default values
	setDefaults()

	config := &Config{
		Server: ServerConfig{
			Port: viper.GetInt("server.port"),
		},
		Database: DatabaseConfig{
			Host:     viper.GetString("db.host"),
			Port:     viper.GetInt("db.port"),
			User:     viper.GetString("db.user"),
			Password: viper.GetString("db.password"),
			DBName:   viper.GetString("db.name"),
			SSLMode:  viper.GetString("db.sslmode"),
		},
		HealthCheck: health.HealthCheckConfig{
			Interval: viper.GetDuration("health_check.interval"),
			Timeout:  viper.GetDuration("health_check.timeout"),
			Retries:  viper.GetInt("health_check.retries"),
		},
		Proxy: ProxyConfig{
			Timeout:        viper.GetDuration("proxy.timeout"),
			MaxConnections: viper.GetInt("proxy.max_connections"),
		},
		App: AppConfig{
			Environment:          viper.GetString("app.env"),
			LogLevel:             viper.GetString("log.level"),
			FallbackRPCEndpoints: viper.GetStringSlice("fallback.rpc_endpoints"),
		},
	}

	// Load multi-chain configuration from database if available
	if config.Database.Host != "" {
		if err := loadMultiChainConfigFromDB(config); err != nil {
			log.Printf("Warning: Failed to load multi-chain config from database: %v", err)
			// Use fallback configuration
			config = createFallbackMultiChainConfig(config)
		}

		// Load and override settings from database
		if err := loadSettingsFromDB(config); err != nil {
			log.Printf("Warning: Failed to load settings from database: %v", err)
		}
	} else {
		// Use fallback configuration if no database configured
		config = createFallbackMultiChainConfig(config)
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func setDefaults() {
	// Server defaults
	viper.SetDefault("server.port", 8888)

	// Database defaults - set empty to disable DB by default
	viper.SetDefault("db.host", "")
	viper.SetDefault("db.port", 5432)
	viper.SetDefault("db.user", "postgres")
	viper.SetDefault("db.password", "")
	viper.SetDefault("db.name", "rpc_proxy")
	viper.SetDefault("db.sslmode", "disable")

	// Health check defaults
	viper.SetDefault("health_check.interval", "30s")
	viper.SetDefault("health_check.timeout", "5s")
	viper.SetDefault("health_check.retries", 3)

	// Proxy defaults
	viper.SetDefault("proxy.timeout", "10s")
	viper.SetDefault("proxy.max_connections", 1000)

	// App defaults
	viper.SetDefault("app.env", "development")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("fallback.rpc_endpoints", []string{
		"https://eth.llamarpc.com",
		"https://ethereum.publicnode.com",
		"https://cloudflare-eth.com",
	})
}

func loadRPCEndpointsFromDB(config *Config) error {
	dbConfig := database.Config{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		User:     config.Database.User,
		Password: config.Database.Password,
		DBName:   config.Database.DBName,
		SSLMode:  config.Database.SSLMode,
	}

	db, err := database.NewGormConnection(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Run auto-migrations
	if err := db.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to run auto-migrations: %w", err)
	}

	// Seed default data
	if err := db.SeedData(); err != nil {
		return fmt.Errorf("failed to seed default data: %w", err)
	}

	repo := gorm.NewRPCEndpointRepository(db)
	endpoints, err := repo.GetEnabled()
	if err != nil {
		return fmt.Errorf("failed to get enabled endpoints: %w", err)
	}

	config.RPCEndpoints = endpoints
	return nil
}

func loadSettingsFromDB(config *Config) error {
	dbConfig := database.Config{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		User:     config.Database.User,
		Password: config.Database.Password,
		DBName:   config.Database.DBName,
		SSLMode:  config.Database.SSLMode,
	}

	db, err := database.NewGormConnection(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	settingsRepo := gorm.NewSettingsRepository(db)
	settings, err := settingsRepo.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	// Override config with database settings
	if val, exists := settings["health_check_interval"]; exists {
		if duration, err := time.ParseDuration(val); err == nil {
			config.HealthCheck.Interval = duration
		}
	}
	if val, exists := settings["health_check_timeout"]; exists {
		if duration, err := time.ParseDuration(val); err == nil {
			config.HealthCheck.Timeout = duration
		}
	}
	if val, exists := settings["health_check_retries"]; exists {
		if retries := viper.Get(val); retries != nil {
			if r, ok := retries.(int); ok {
				config.HealthCheck.Retries = r
			}
		}
	}
	if val, exists := settings["proxy_timeout"]; exists {
		if duration, err := time.ParseDuration(val); err == nil {
			config.Proxy.Timeout = duration
		}
	}
	if val, exists := settings["max_connections"]; exists {
		if maxConn := viper.Get(val); maxConn != nil {
			if mc, ok := maxConn.(int); ok {
				config.Proxy.MaxConnections = mc
			}
		}
	}
	if val, exists := settings["server_port"]; exists {
		if port := viper.Get(val); port != nil {
			if p, ok := port.(int); ok {
				config.Server.Port = p
			}
		}
	}

	return nil
}

// loadMultiChainConfigFromDB loads chains, endpoints, and chain-specific configs from database
func loadMultiChainConfigFromDB(config *Config) error {
	dbConfig := database.Config{
		Host:     config.Database.Host,
		Port:     config.Database.Port,
		User:     config.Database.User,
		Password: config.Database.Password,
		DBName:   config.Database.DBName,
		SSLMode:  config.Database.SSLMode,
	}

	db, err := database.NewGormConnection(dbConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Run auto-migrations
	if err := db.AutoMigrate(); err != nil {
		return fmt.Errorf("failed to run auto-migrations: %w", err)
	}

	// Seed default data
	if err := db.SeedData(); err != nil {
		return fmt.Errorf("failed to seed default data: %w", err)
	}

	// Initialize maps
	config.ChainEndpoints = make(map[string][]*types.RPCEndpoint)
	config.ChainConfigs = make(map[string]map[string]string)

	// Load chains
	chainRepo := gorm.NewChainRepository(db)
	chains, err := chainRepo.GetAll()
	if err != nil {
		return fmt.Errorf("failed to load chains: %w", err)
	}
	config.Chains = chains

	// Load endpoints for each chain
	endpointRepo := gorm.NewRPCEndpointRepository(db)
	chainConfigRepo := gorm.NewChainConfigRepository(db)

	for _, chain := range chains {
		// Load endpoints for this chain
		endpoints, err := endpointRepo.GetAllByChain(chain.Name)
		if err != nil {
			log.Printf("Warning: Failed to load endpoints for chain %s: %v", chain.Name, err)
			config.ChainEndpoints[chain.Name] = []*types.RPCEndpoint{}
		} else {
			config.ChainEndpoints[chain.Name] = endpoints
		}

		// Load chain-specific config
		chainConfigs, err := chainConfigRepo.GetByChainName(chain.Name)
		if err != nil {
			log.Printf("Warning: Failed to load config for chain %s: %v", chain.Name, err)
			config.ChainConfigs[chain.Name] = make(map[string]string)
		} else {
			config.ChainConfigs[chain.Name] = chainConfigs
		}
	}

	// Legacy fallback for backward compatibility
	legacyRepo := gorm.NewRPCEndpointRepository(db)
	legacyEndpoints, err := legacyRepo.GetEnabled()
	if err != nil {
		log.Printf("Warning: Failed to load legacy endpoints: %v", err)
	} else {
		config.RPCEndpoints = legacyEndpoints
	}

	log.Printf("Multi-chain configuration loaded: %d chains, %d total endpoints",
		len(config.Chains), len(config.RPCEndpoints))

	return nil
}

// createFallbackMultiChainConfig creates fallback configuration when database is unavailable
func createFallbackMultiChainConfig(config *Config) *Config {
	// Create fallback chains
	config.Chains = []*types.Chain{
		{
			ID:                     1,
			ChainID:                1,
			Name:                   "ethereum",
			DisplayName:            "Ethereum Mainnet",
			RPCPath:                "ethereum",
			IsTestnet:              false,
			IsEnabled:              true,
			NativeCurrencySymbol:   "ETH",
			NativeCurrencyDecimals: 18,
			BlockExplorerURL:       "https://etherscan.io",
		},
		{
			ID:                     2,
			ChainID:                11155111,
			Name:                   "sepolia",
			DisplayName:            "Sepolia Testnet",
			RPCPath:                "sepolia",
			IsTestnet:              true,
			IsEnabled:              true,
			NativeCurrencySymbol:   "ETH",
			NativeCurrencyDecimals: 18,
			BlockExplorerURL:       "https://sepolia.etherscan.io",
		},
		{
			ID:                     3,
			ChainID:                1868,
			Name:                   "soneium",
			DisplayName:            "Soneium Mainnet",
			RPCPath:                "soneium",
			IsTestnet:              false,
			IsEnabled:              true,
			NativeCurrencySymbol:   "ETH",
			NativeCurrencyDecimals: 18,
			BlockExplorerURL:       "https://explorer.soneium.org",
		},
		{
			ID:                     4,
			ChainID:                1946,
			Name:                   "soneium-testnet",
			DisplayName:            "Soneium Testnet",
			RPCPath:                "soneium-testnet",
			IsTestnet:              true,
			IsEnabled:              true,
			NativeCurrencySymbol:   "ETH",
			NativeCurrencyDecimals: 18,
			BlockExplorerURL:       "https://explorer-testnet.soneium.org",
		},
	}

	// Create fallback endpoints
	config.ChainEndpoints = map[string][]*types.RPCEndpoint{
		"ethereum": {
			{ID: 1, Name: "Ethereum-LlamaRPC", URL: "https://eth.llamarpc.com", Weight: 3, Enabled: true, ChainID: 1},
			{ID: 2, Name: "Ethereum-PublicNode", URL: "https://ethereum.publicnode.com", Weight: 2, Enabled: true, ChainID: 1},
			{ID: 3, Name: "Ethereum-Cloudflare", URL: "https://cloudflare-eth.com", Weight: 2, Enabled: true, ChainID: 1},
		},
		"sepolia": {
			{ID: 4, Name: "Sepolia-1RPC", URL: "https://1rpc.io/sepolia", Weight: 3, Enabled: true, ChainID: 2},
			{ID: 5, Name: "Sepolia-PublicNode", URL: "https://ethereum-sepolia-rpc.publicnode.com", Weight: 2, Enabled: true, ChainID: 2},
			{ID: 6, Name: "Sepolia-DRPC", URL: "https://sepolia.drpc.org", Weight: 2, Enabled: true, ChainID: 2},
		},
		"soneium": {
			{ID: 7, Name: "Soneium-DRPC", URL: "https://soneium.drpc.org", Weight: 3, Enabled: true, ChainID: 3},
			{ID: 8, Name: "Soneium-Official", URL: "https://rpc.soneium.org", Weight: 2, Enabled: true, ChainID: 3},
		},
		"soneium-testnet": {
			{ID: 9, Name: "Soneium-Testnet-Official", URL: "https://rpc.minato.soneium.org", Weight: 3, Enabled: true, ChainID: 4},
			{ID: 10, Name: "Soneium-Testnet-DRPC", URL: "https://soneium-minato.drpc.org", Weight: 2, Enabled: true, ChainID: 4},
		},
	}

	// Create fallback chain configs
	config.ChainConfigs = map[string]map[string]string{
		"ethereum": {
			"max_block_lag":            "5",
			"gas_price_gwei_threshold": "100",
		},
		"sepolia": {
			"max_block_lag":            "10",
			"gas_price_gwei_threshold": "20",
		},
		"soneium": {
			"max_block_lag":            "5",
			"gas_price_gwei_threshold": "50",
		},
		"soneium-testnet": {
			"max_block_lag":            "10",
			"gas_price_gwei_threshold": "20",
		},
	}

	// Legacy fallback endpoints
	config.RPCEndpoints = createFallbackEndpoints(config.App.FallbackRPCEndpoints)

	return config
}

func createFallbackEndpoints(urls []string) []*types.RPCEndpoint {
	endpoints := make([]*types.RPCEndpoint, len(urls))
	for i, url := range urls {
		endpoints[i] = &types.RPCEndpoint{
			ID:      i + 1,
			Name:    fmt.Sprintf("Fallback-%d", i+1),
			URL:     url,
			Weight:  1,
			Enabled: true,
			ChainID: 1, // Default to Ethereum mainnet
		}
	}
	return endpoints
}

// CreateMultiChainHealthChecker creates a multi-chain health checker from config
func (c *Config) CreateMultiChainHealthChecker() *health.MultiChainChecker {
	chainsConfig := make(map[string]*health.ChainConfig)

	for _, chain := range c.Chains {
		if !chain.IsEnabled {
			continue
		}

		endpoints, exists := c.ChainEndpoints[chain.Name]
		if !exists || len(endpoints) == 0 {
			log.Printf("Warning: No endpoints configured for chain %s, skipping", chain.Name)
			continue
		}

		chainsConfig[chain.Name] = &health.ChainConfig{
			Chain:     chain,
			Endpoints: endpoints,
		}
	}

	return health.NewMultiChainChecker(chainsConfig, c.HealthCheck)
}

// GetChainByName returns chain configuration by name
func (c *Config) GetChainByName(chainName string) *types.Chain {
	for _, chain := range c.Chains {
		if chain.Name == chainName {
			return chain
		}
	}
	return nil
}

func validateConfig(config *Config) error {
	// Validate multi-chain configuration
	if len(config.Chains) == 0 && len(config.RPCEndpoints) == 0 {
		return fmt.Errorf("at least one chain or legacy RPC endpoint must be configured")
	}

	// Validate that each enabled chain has at least one endpoint
	for _, chain := range config.Chains {
		if chain.IsEnabled {
			if endpoints, exists := config.ChainEndpoints[chain.Name]; !exists || len(endpoints) == 0 {
				return fmt.Errorf("enabled chain %s must have at least one RPC endpoint", chain.Name)
			}
		}
	}

	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535")
	}

	if config.HealthCheck.Interval <= 0 {
		return fmt.Errorf("health check interval must be positive")
	}

	if config.HealthCheck.Timeout <= 0 {
		return fmt.Errorf("health check timeout must be positive")
	}

	if config.HealthCheck.Retries <= 0 {
		return fmt.Errorf("health check retries must be positive")
	}

	if config.Proxy.Timeout <= 0 {
		return fmt.Errorf("proxy timeout must be positive")
	}

	if config.Proxy.MaxConnections <= 0 {
		return fmt.Errorf("max connections must be positive")
	}

	return nil
}
