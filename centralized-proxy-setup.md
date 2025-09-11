# Centralized RPC Proxy Setup for Multiple Subgraphs

## Architecture Overview
```
┌─────────────────────────────────────────────┐
│           RPC Proxy (Port 8888)             │
│  ┌─────────┬──────────┬─────────────────┐  │
│  │Ethereum │ Sepolia  │     Soneium     │  │
│  │  /rpc/  │  /rpc/   │      /rpc/      │  │
│  │ethereum │ sepolia  │     soneium     │  │
│  └─────────┴──────────┴─────────────────┘  │
└──────────────────┬──────────────────────────┘
                   │
    ┌──────────────┼──────────────┐
    │              │              │
┌───▼────┐   ┌────▼────┐   ┌─────▼─────┐
│Subgraph│   │Subgraph │   │ Subgraph  │
│   #1   │   │   #2    │   │    #3     │
└────────┘   └─────────┘   └───────────┘
```

## Setup Options

### Option 1: Single Proxy, Multiple Graph Nodes (Recommended)
**Best for:** Running multiple subgraphs on different chains

#### 1. Start the Centralized RPC Proxy
```bash
cd /Users/siwakon.cha/workspace/moonshot/golang-proxy

# Add Soneium to config (choose one method):

# Method A: Using database
docker-compose up -d postgres
psql -h localhost -U postgres -d rpc_proxy < add-soneium.sql

# Method B: Using environment variables
cat .env.soneium >> .env

# Start the proxy
./rpc-proxy
# or with Docker
docker-compose up -d rpc-proxy
```

#### 2. Update Each Subgraph's docker-compose.yml

**For Soneium subgraph:**
```yaml
# /Users/siwakon.cha/workspace/moonshot/subgraph/velo/yay-stone-gauge/docker-compose.yml
environment:
  ethereum: "soneium:http://host.docker.internal:8888/rpc/soneium"
```

**For Astar subgraph (add Astar to proxy first):**
```yaml
# /Users/siwakon.cha/workspace/moonshot/subgraph/stakingtoken/docker-compose.yml
environment:
  ethereum: "astar:http://host.docker.internal:8888/rpc/astar"
```

#### 3. Run Each Subgraph
```bash
# Terminal 1: Soneium subgraph
cd /Users/siwakon.cha/workspace/moonshot/subgraph/velo/yay-stone-gauge
docker-compose up -d
npm run create-local
npm run deploy-local

# Terminal 2: Other subgraph
cd /Users/siwakon.cha/workspace/moonshot/subgraph/stakingtoken
docker-compose up -d
npm run create-local
npm run deploy-local
```

### Option 2: Shared Infrastructure
**Best for:** Resource optimization

Create a single docker-compose with all services:
```yaml
# /Users/siwakon.cha/workspace/moonshot/shared-infra/docker-compose.yml
version: '3.8'

services:
  # Single RPC Proxy
  rpc-proxy:
    build: ../golang-proxy
    ports:
      - "8888:8888"
    environment:
      - SERVER_PORT=8888
    
  # Shared IPFS
  ipfs:
    image: ipfs/go-ipfs:latest
    ports:
      - "5001:5001"
      
  # Graph Node 1 - Soneium
  graph-node-soneium:
    image: graphprotocol/graph-node
    environment:
      ethereum: "soneium:http://rpc-proxy:8888/rpc/soneium"
    ports:
      - "8000:8000"  # GraphQL
      - "8020:8020"  # Admin
      
  # Graph Node 2 - Ethereum
  graph-node-ethereum:
    image: graphprotocol/graph-node
    environment:
      ethereum: "mainnet:http://rpc-proxy:8888/rpc/ethereum"
    ports:
      - "9000:8000"  # GraphQL
      - "9020:8020"  # Admin
```

### Option 3: Port Forwarding
**Best for:** Development/testing

```bash
# Start proxy on host machine
cd /Users/siwakon.cha/workspace/moonshot/golang-proxy
./rpc-proxy

# Each subgraph uses host.docker.internal:8888
```

## Adding New Chains to Proxy

### 1. Create SQL file for new chain
```sql
-- add-[chain-name].sql
INSERT INTO chains (...) VALUES (...);
INSERT INTO rpc_endpoints (...) VALUES (...);
```

### 2. Or add to environment variables
```bash
# .env
[CHAIN_NAME]_RPC_ENDPOINTS=https://rpc1.com,https://rpc2.com
```

### 3. Restart proxy
```bash
docker-compose restart rpc-proxy
# or
./rpc-proxy
```

## Benefits of Centralized Proxy

1. **Single Point of RPC Management**: All RPC endpoints in one place
2. **Automatic Failover**: Proxy handles RPC failures automatically
3. **Load Balancing**: Distributes requests across multiple RPCs
4. **Health Monitoring**: `/health` endpoint shows all chains status
5. **Easy Scaling**: Add new chains without modifying subgraphs
6. **Resource Efficiency**: One proxy serves all subgraphs

## Monitoring

Check proxy health:
```bash
# All chains
curl http://localhost:8888/health | jq .

# Specific chain
curl http://localhost:8888/health/soneium | jq .

# Test RPC call
curl -X POST http://localhost:8888/rpc/soneium \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

## Troubleshooting

1. **Connection refused**: Check proxy is running on port 8888
2. **Chain not found**: Verify chain is added to proxy config
3. **RPC errors**: Check `/health` endpoint for RPC status
4. **Docker networking**: Use `host.docker.internal` or container names