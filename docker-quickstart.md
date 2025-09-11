# RPC Proxy Docker Quick Start

## 🚀 Fastest Way to Run (1 Command)

### Option 1: Using Pre-built Image (Recommended)
```bash
docker run -d \
  --name rpc-proxy \
  -p 8888:8888 \
  -e ETHEREUM_RPC_ENDPOINTS="https://eth.llamarpc.com,https://ethereum.publicnode.com" \
  -e SEPOLIA_RPC_ENDPOINTS="https://sepolia.drpc.org,https://ethereum-sepolia-rpc.publicnode.com" \
  -e SONEIUM_RPC_ENDPOINTS="https://soneium.drpc.org,https://rpc.soneium.org" \
  --restart unless-stopped \
  your-dockerhub-username/rpc-proxy:latest
```

### Option 2: Using Docker Compose
```bash
# Download docker-compose file
curl -O https://raw.githubusercontent.com/your-repo/golang-proxy/main/docker-compose.simple.yml

# Run
docker-compose -f docker-compose.simple.yml up -d

# Check health
curl http://localhost:8888/health
```

## 📦 Build and Push to Docker Hub

### 1. Build Image
```bash
cd /Users/siwakon.cha/workspace/moonshot/golang-proxy

# Build for multiple architectures (AMD64 + ARM64)
docker buildx create --use
docker buildx build --platform linux/amd64,linux/arm64 \
  -t your-dockerhub-username/rpc-proxy:latest \
  -t your-dockerhub-username/rpc-proxy:v1.0.0 \
  --push .

# Or simple build for current architecture
docker build -t your-dockerhub-username/rpc-proxy:latest .
```

### 2. Test Locally
```bash
docker run --rm -p 8888:8888 your-dockerhub-username/rpc-proxy:latest
```

### 3. Push to Docker Hub
```bash
# Login to Docker Hub
docker login

# Push image
docker push your-dockerhub-username/rpc-proxy:latest
docker push your-dockerhub-username/rpc-proxy:v1.0.0
```

## 🌍 Deploy Anywhere

### AWS EC2
```bash
# SSH to EC2
ssh ubuntu@your-ec2-ip

# Run proxy
docker run -d \
  --name rpc-proxy \
  -p 8888:8888 \
  --restart always \
  your-dockerhub-username/rpc-proxy:latest
```

### Google Cloud Run
```bash
gcloud run deploy rpc-proxy \
  --image your-dockerhub-username/rpc-proxy:latest \
  --port 8888 \
  --allow-unauthenticated \
  --region us-central1
```

### DigitalOcean
```bash
doctl apps create --spec - <<EOF
name: rpc-proxy
services:
- name: proxy
  image:
    registry_type: DOCKER_HUB
    registry: your-dockerhub-username
    repository: rpc-proxy
    tag: latest
  http_port: 8888
EOF
```

### Kubernetes
```yaml
# rpc-proxy-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rpc-proxy
spec:
  replicas: 2
  selector:
    matchLabels:
      app: rpc-proxy
  template:
    metadata:
      labels:
        app: rpc-proxy
    spec:
      containers:
      - name: rpc-proxy
        image: your-dockerhub-username/rpc-proxy:latest
        ports:
        - containerPort: 8888
        env:
        - name: SONEIUM_RPC_ENDPOINTS
          value: "https://soneium.drpc.org,https://rpc.soneium.org"
---
apiVersion: v1
kind: Service
metadata:
  name: rpc-proxy
spec:
  selector:
    app: rpc-proxy
  ports:
  - port: 8888
    targetPort: 8888
  type: LoadBalancer
```

```bash
kubectl apply -f rpc-proxy-deployment.yaml
```

## 🔧 Configuration

### Environment Variables
```yaml
# Essential
SERVER_PORT: 8888

# Chain RPC Endpoints (comma-separated)
ETHEREUM_RPC_ENDPOINTS: "https://eth1.com,https://eth2.com"
SEPOLIA_RPC_ENDPOINTS: "https://sep1.com,https://sep2.com"
SONEIUM_RPC_ENDPOINTS: "https://son1.com,https://son2.com"

# Add more chains
POLYGON_RPC_ENDPOINTS: "https://polygon-rpc.com"
ARBITRUM_RPC_ENDPOINTS: "https://arb1.arbitrum.io/rpc"
OPTIMISM_RPC_ENDPOINTS: "https://mainnet.optimism.io"
```

### Docker Compose with Custom Config
```yaml
# docker-compose.custom.yml
version: '3.8'

services:
  rpc-proxy:
    image: your-dockerhub-username/rpc-proxy:latest
    ports:
      - "8888:8888"
    env_file:
      - .env.production  # Your custom env file
    volumes:
      - ./config:/config  # Optional: mount config files
    restart: always
    deploy:
      resources:
        limits:
          memory: 512M
          cpus: '0.5'
```

## 🔗 Connect Your Subgraphs

### Local Development
```yaml
# subgraph docker-compose.yml
environment:
  ethereum: "soneium:http://host.docker.internal:8888/rpc/soneium"
```

### Production (Same Network)
```yaml
environment:
  ethereum: "soneium:http://rpc-proxy:8888/rpc/soneium"
```

### Remote Proxy
```yaml
environment:
  ethereum: "soneium:https://your-proxy-domain.com/rpc/soneium"
```

## 📊 Monitoring

### Health Check
```bash
# Basic health
curl http://localhost:8888/health

# Pretty JSON
curl -s http://localhost:8888/health | jq .

# Specific chain
curl http://localhost:8888/health/soneium
```

### Docker Logs
```bash
# View logs
docker logs rpc-proxy

# Follow logs
docker logs -f rpc-proxy

# Last 100 lines
docker logs --tail 100 rpc-proxy
```

### Container Stats
```bash
# Resource usage
docker stats rpc-proxy

# Inspect container
docker inspect rpc-proxy
```

## 🚨 Troubleshooting

| Issue | Solution |
|-------|----------|
| Port already in use | `docker ps` and stop conflicting container |
| Can't connect from subgraph | Use `host.docker.internal` or container name |
| High memory usage | Add memory limits in docker-compose |
| Slow responses | Check RPC endpoints health at `/health` |
| Container keeps restarting | Check logs: `docker logs rpc-proxy` |

## 🎯 Quick Commands

```bash
# Start
docker-compose -f docker-compose.simple.yml up -d

# Stop
docker-compose -f docker-compose.simple.yml down

# Restart
docker-compose -f docker-compose.simple.yml restart

# Update image
docker-compose -f docker-compose.simple.yml pull
docker-compose -f docker-compose.simple.yml up -d

# View logs
docker-compose -f docker-compose.simple.yml logs -f

# Scale (run multiple instances)
docker-compose -f docker-compose.simple.yml up -d --scale rpc-proxy=3
```

## 🔒 Production Tips

1. **Use specific tags** instead of `latest`
2. **Set resource limits** to prevent OOM
3. **Enable health checks** for auto-recovery
4. **Use secrets** for sensitive data
5. **Setup monitoring** with Prometheus/Grafana
6. **Use reverse proxy** (Nginx/Traefik) for SSL

## 📝 Example: Complete Production Setup

```bash
#!/bin/bash
# deploy-production.sh

# Pull latest image
docker pull your-dockerhub-username/rpc-proxy:v1.0.0

# Stop old container
docker stop rpc-proxy && docker rm rpc-proxy

# Run with production settings
docker run -d \
  --name rpc-proxy \
  -p 127.0.0.1:8888:8888 \
  --restart always \
  --memory="512m" \
  --cpus="0.5" \
  --log-driver json-file \
  --log-opt max-size=10m \
  --log-opt max-file=3 \
  -e LOG_LEVEL=warn \
  -e PROXY_TIMEOUT=30s \
  -e ETHEREUM_RPC_ENDPOINTS="$PROD_ETH_RPCS" \
  -e SONEIUM_RPC_ENDPOINTS="$PROD_SONEIUM_RPCS" \
  your-dockerhub-username/rpc-proxy:v1.0.0

# Setup Nginx reverse proxy with SSL
# (Nginx config handles SSL termination)
```