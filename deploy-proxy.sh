#!/bin/bash

# RPC Proxy AWS EC2 Deployment Script
# Usage: ./deploy-proxy.sh

set -e

echo "🚀 Starting RPC Proxy Deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if running on EC2
if [ ! -f /sys/hypervisor/uuid ] || [ $(head -c 3 /sys/hypervisor/uuid) != "ec2" ]; then
    echo -e "${YELLOW}Warning: This script is optimized for AWS EC2 instances${NC}"
fi

# Update system
echo -e "${GREEN}1. Updating system packages...${NC}"
sudo apt update && sudo apt upgrade -y

# Install Docker if not exists
if ! command -v docker &> /dev/null; then
    echo -e "${GREEN}2. Installing Docker...${NC}"
    curl -fsSL https://get.docker.com -o get-docker.sh
    sudo sh get-docker.sh
    sudo usermod -aG docker $USER
    rm get-docker.sh
else
    echo -e "${YELLOW}Docker already installed${NC}"
fi

# Install Docker Compose if not exists
if ! command -v docker-compose &> /dev/null; then
    echo -e "${GREEN}3. Installing Docker Compose...${NC}"
    sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
else
    echo -e "${YELLOW}Docker Compose already installed${NC}"
fi

# Install Git if not exists
if ! command -v git &> /dev/null; then
    echo -e "${GREEN}4. Installing Git...${NC}"
    sudo apt install git -y
fi

# Clone or update repository
REPO_DIR="$HOME/golang-proxy"
if [ -d "$REPO_DIR" ]; then
    echo -e "${GREEN}5. Updating existing repository...${NC}"
    cd $REPO_DIR
    git pull
else
    echo -e "${GREEN}5. Cloning repository...${NC}"
    # Replace with your actual repository URL
    git clone https://github.com/your-username/golang-proxy.git $REPO_DIR
    cd $REPO_DIR
fi

# Create .env file if not exists
if [ ! -f .env ]; then
    echo -e "${GREEN}6. Creating .env configuration...${NC}"
    
    # Get EC2 metadata
    EC2_PUBLIC_IP=$(curl -s http://169.254.169.254/latest/meta-data/public-ipv4 2>/dev/null || echo "")
    EC2_PRIVATE_IP=$(curl -s http://169.254.169.254/latest/meta-data/local-ipv4 2>/dev/null || echo "")
    
    cat > .env << EOF
# Server Configuration
SERVER_PORT=8888

# Database Configuration
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=$(openssl rand -base64 32)
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
APP_ENV=production
LOG_LEVEL=info

# Ethereum Mainnet
ETHEREUM_RPC_ENDPOINTS=https://eth.llamarpc.com,https://ethereum.publicnode.com,https://rpc.ankr.com/eth

# Sepolia Testnet
SEPOLIA_RPC_ENDPOINTS=https://sepolia.drpc.org,https://ethereum-sepolia-rpc.publicnode.com,https://rpc.ankr.com/eth_sepolia

# Soneium Mainnet
SONEIUM_RPC_ENDPOINTS=https://soneium.drpc.org,https://rpc.soneium.org,https://soneium-mainnet.public.blastapi.io

# EC2 Instance Info (auto-detected)
EC2_PUBLIC_IP=$EC2_PUBLIC_IP
EC2_PRIVATE_IP=$EC2_PRIVATE_IP
EOF
    
    echo -e "${GREEN}Configuration created with:${NC}"
    echo -e "  Public IP: ${YELLOW}$EC2_PUBLIC_IP${NC}"
    echo -e "  Private IP: ${YELLOW}$EC2_PRIVATE_IP${NC}"
else
    echo -e "${YELLOW}6. .env file already exists, skipping...${NC}"
fi

# Build and start services
echo -e "${GREEN}7. Building and starting services...${NC}"
sudo docker-compose down 2>/dev/null || true
sudo docker-compose build
sudo docker-compose up -d

# Wait for services to be ready
echo -e "${GREEN}8. Waiting for services to be ready...${NC}"
sleep 10

# Health check
echo -e "${GREEN}9. Running health check...${NC}"
if curl -f http://localhost:8888/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Proxy is running successfully!${NC}"
    
    # Display connection info
    echo -e "\n${GREEN}=== Connection Information ===${NC}"
    echo -e "Local access: ${YELLOW}http://localhost:8888${NC}"
    
    if [ ! -z "$EC2_PUBLIC_IP" ]; then
        echo -e "Public access: ${YELLOW}http://$EC2_PUBLIC_IP:8888${NC}"
    fi
    
    if [ ! -z "$EC2_PRIVATE_IP" ]; then
        echo -e "VPC access: ${YELLOW}http://$EC2_PRIVATE_IP:8888${NC}"
    fi
    
    echo -e "\n${GREEN}=== Available Endpoints ===${NC}"
    echo -e "Health Check: ${YELLOW}/health${NC}"
    echo -e "Ethereum RPC: ${YELLOW}/rpc/ethereum${NC}"
    echo -e "Sepolia RPC: ${YELLOW}/rpc/sepolia${NC}"
    echo -e "Soneium RPC: ${YELLOW}/rpc/soneium${NC}"
    
    echo -e "\n${GREEN}=== Update Your Subgraphs ===${NC}"
    echo -e "In your subgraph's docker-compose.yml, update the ethereum field:"
    echo -e "${YELLOW}ethereum: \"soneium:http://$EC2_PUBLIC_IP:8888/rpc/soneium\"${NC}"
    
else
    echo -e "${RED}❌ Health check failed!${NC}"
    echo -e "Check logs with: ${YELLOW}docker-compose logs rpc-proxy${NC}"
    exit 1
fi

# Setup systemd service for auto-restart
echo -e "\n${GREEN}10. Setting up auto-restart service...${NC}"
sudo tee /etc/systemd/system/rpc-proxy.service > /dev/null << EOF
[Unit]
Description=RPC Proxy Docker Compose
After=docker.service
Requires=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=$REPO_DIR
ExecStart=/usr/local/bin/docker-compose up -d
ExecStop=/usr/local/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable rpc-proxy.service

echo -e "${GREEN}✅ Deployment complete!${NC}"
echo -e "\n${YELLOW}Important Security Note:${NC}"
echo -e "Remember to configure your AWS Security Group to allow inbound traffic on port 8888"
echo -e "from your subgraph servers or specific IP ranges."

# Display useful commands
echo -e "\n${GREEN}=== Useful Commands ===${NC}"
echo -e "View logs: ${YELLOW}docker-compose logs -f rpc-proxy${NC}"
echo -e "Restart proxy: ${YELLOW}docker-compose restart rpc-proxy${NC}"
echo -e "Stop proxy: ${YELLOW}docker-compose down${NC}"
echo -e "Update proxy: ${YELLOW}git pull && docker-compose up -d --build${NC}"
echo -e "Check status: ${YELLOW}curl http://localhost:8888/health | jq .${NC}"