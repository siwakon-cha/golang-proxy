# RPC Proxy Service

A high-availability RPC proxy service with automatic failover, health monitoring, and load balancing for blockchain RPC endpoints. Features PostgreSQL database integration with GORM, environment-based configuration with Viper, and comprehensive admin API.

## 🚀 Features

- **Health Monitoring**: Continuous health checks using `eth_blockNumber` method
- **Automatic Failover**: Seamless switching to healthy endpoints within 30 seconds
- **Load Balancing**: Round-robin distribution among healthy endpoints
- **Circuit Breaker**: Prevents cascade failures with intelligent retry logic
- **Database Integration**: PostgreSQL with GORM for dynamic endpoint management
- **Admin API**: Full CRUD operations for managing RPC endpoints and settings
- **Environment Configuration**: Uses godotenv + Viper for flexible configuration
- **CORS Support**: Ready for browser-based applications
- **Docker Support**: Complete containerization with PostgreSQL

## 📦 Quick Start

### Using Docker Compose (Recommended)

1. **Clone and setup:**
```bash
git clone <repository>
cd golang-proxy
cp .env.example .env
# Edit .env with your settings
```

2. **Start all services:**
```bash
docker-compose up -d
```

3. **Check health:**
```bash
curl http://localhost:8080/health
```

### Manual Setup

1. **Setup PostgreSQL database:**
```bash
createdb rpc_proxy
```

2. **Configure environment:**
```bash
cp .env.example .env
# Edit .env with your database credentials
```

3. **Install and run:**
```bash
go mod tidy
go run main.go
```

## ⚙️ Configuration

Configuration is managed through environment variables using Viper. Create a `.env` file:

```bash
# Server Configuration
SERVER_PORT=8080

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=rpc_proxy
DB_SSLMODE=disable

# Health Check Configuration
HEALTH_CHECK_INTERVAL=30s
HEALTH_CHECK_TIMEOUT=5s
HEALTH_CHECK_RETRIES=3

# Proxy Configuration
PROXY_TIMEOUT=10s
PROXY_MAX_CONNECTIONS=1000

# Application Configuration
APP_ENV=development
LOG_LEVEL=info

# Fallback RPC endpoints (comma-separated)
FALLBACK_RPC_ENDPOINTS=https://eth.llamarpc.com,https://ethereum.publicnode.com
```

## 🔧 Admin API

### RPC Endpoints Management
```bash
# List all endpoints
GET /admin/endpoints

# Get specific endpoint
GET /admin/endpoints/:id

# Create new endpoint
POST /admin/endpoints
{
  "name": "Infura",
  "url": "https://mainnet.infura.io/v3/YOUR_KEY",
  "weight": 2,
  "enabled": true
}

# Update endpoint
PUT /admin/endpoints/:id
{
  "enabled": false
}

# Delete endpoint
DELETE /admin/endpoints/:id
```

### Settings Management
```bash
# List all settings
GET /admin/settings

# Get specific setting
GET /admin/settings/:key

# Update setting
PUT /admin/settings/health_check_interval
{
  "value": "15s",
  "description": "Interval between health checks"
}

# Delete setting
DELETE /admin/settings/:key
```

### Health Check History
```bash
# Get health check history for endpoint
GET /admin/health-checks/:endpoint_id?limit=50
```

## 🌐 Proxy Usage

### JSON-RPC Requests
```bash
# Make RPC calls through the proxy
curl -X POST http://localhost:8080/rpc \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

### Integration with The Graph
```yaml
# docker-compose.yml
graph-node:
  environment:
    - ethereum=mainnet:http://rpc-proxy:8080/rpc
```

## 🗄️ Database Schema

The service uses GORM with PostgreSQL:

- **rpc_endpoints**: Store RPC endpoint configurations
- **health_checks**: Track health check history and metrics  
- **settings**: Store configuration settings

Auto-migration runs on startup, creating tables and seeding default data.

## 📊 Monitoring

### Health Endpoint
```bash
GET /health
```

Returns comprehensive status including:
- Proxy service status
- Current active RPC endpoint
- Health status of all configured endpoints
- Response times and block numbers
- Last check timestamps

### Example Response
```json
{
  "proxy": "healthy",
  "currentRPC": "https://eth.llamarpc.com",
  "rpcEndpoints": [
    {
      "id": 1,
      "name": "LlamaRPC",
      "url": "https://eth.llamarpc.com",
      "healthy": true,
      "lastCheck": "2025-07-25T10:30:00Z",
      "responseTime": 150,
      "blockNumber": "0x12a4b2c"
    }
  ]
}
```

## 🏗️ Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Admin API     │    │   Health Checker │    │   Proxy Server  │
│  (CRUD Endpoints│    │   (Background)   │    │  (Handle RPCs)  │
└─────────┬───────┘    └─────────┬────────┘    └─────────┬───────┘
          │                      │                       │
          └──────────────────────┼───────────────────────┘
                                 │
                    ┌────────────▼────────────┐
                    │      PostgreSQL         │
                    │    (GORM + Auto-Migrate)│
                    │                         │
                    │  - rpc_endpoints        │
                    │  - health_checks        │
                    │  - settings             │
                    └─────────────────────────┘
```

## 🔍 Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP server port |
| `DB_HOST` | localhost | PostgreSQL host |
| `DB_PORT` | 5432 | PostgreSQL port |
| `DB_USER` | postgres | Database user |
| `DB_PASSWORD` | - | Database password |
| `DB_NAME` | rpc_proxy | Database name |
| `DB_SSLMODE` | disable | SSL mode for database |
| `HEALTH_CHECK_INTERVAL` | 30s | Interval between health checks |
| `HEALTH_CHECK_TIMEOUT` | 5s | Health check timeout |
| `HEALTH_CHECK_RETRIES` | 3 | Retries before marking unhealthy |
| `PROXY_TIMEOUT` | 10s | Proxy request timeout |
| `PROXY_MAX_CONNECTIONS` | 1000 | Maximum concurrent connections |
| `APP_ENV` | development | Application environment |
| `LOG_LEVEL` | info | Logging level |

## 🚀 Performance

- Supports 1000+ concurrent requests
- Sub-30 second failover time
- Response time P95 < 1500ms
- Memory usage < 100MB under normal load
- Automatic connection pooling with GORM

## 📄 License

MIT License