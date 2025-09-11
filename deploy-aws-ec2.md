# Deploy RPC Proxy to AWS EC2

## 1. EC2 Instance Setup

### Required Instance Specifications
- **Type**: t3.small or t3.medium (for production)
- **OS**: Ubuntu 22.04 LTS or Amazon Linux 2023
- **Storage**: 20-30 GB SSD
- **Region**: Same as your subgraphs (reduce latency)

### Security Group Rules
```
Inbound Rules:
- SSH (22): Your IP
- Custom TCP (8888): 0.0.0.0/0 or specific IPs/subnets
- Custom TCP (8888): Your VPC CIDR (for internal access)

Outbound Rules:
- All traffic: 0.0.0.0/0 (for RPC endpoints)
```

## 2. Installation Script

### Option A: Using Docker (Recommended)
```bash
#!/bin/bash
# save as: deploy-proxy.sh

# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker ubuntu

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Clone repository
git clone https://github.com/your-repo/golang-proxy.git
cd golang-proxy

# Create .env file
cat > .env << 'EOF'
# Server Configuration
SERVER_PORT=8888

# Database Configuration (optional)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-secure-password
DB_NAME=rpc_proxy
DB_SSLMODE=disable

# Ethereum endpoints
ETHEREUM_RPC_ENDPOINTS=https://eth.llamarpc.com,https://ethereum.publicnode.com

# Sepolia endpoints
SEPOLIA_RPC_ENDPOINTS=https://sepolia.drpc.org,https://ethereum-sepolia-rpc.publicnode.com

# Soneium endpoints
SONEIUM_RPC_ENDPOINTS=https://soneium.drpc.org,https://rpc.soneium.org
EOF

# Run with Docker Compose
docker-compose up -d
```

### Option B: Using Binary
```bash
#!/bin/bash
# Install Go
wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Build and run
git clone https://github.com/your-repo/golang-proxy.git
cd golang-proxy
go build -o rpc-proxy
./rpc-proxy &
```

## 3. Systemd Service (For Production)

Create service file:
```bash
sudo nano /etc/systemd/system/rpc-proxy.service
```

```ini
[Unit]
Description=RPC Proxy Service
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/ubuntu/golang-proxy
Environment="PATH=/usr/local/go/bin:/usr/bin:/bin"
ExecStart=/home/ubuntu/golang-proxy/rpc-proxy
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl enable rpc-proxy
sudo systemctl start rpc-proxy
sudo systemctl status rpc-proxy
```

## 4. Connect Your Subgraphs

### From Local Machine
```yaml
# docker-compose.yml in your subgraph
environment:
  ethereum: "soneium:http://YOUR-EC2-PUBLIC-IP:8888/rpc/soneium"
```

### From AWS (Same VPC)
```yaml
environment:
  ethereum: "soneium:http://PRIVATE-IP:8888/rpc/soneium"
```

### Using Domain Name
```yaml
# After setting up Route53 or your DNS
environment:
  ethereum: "soneium:https://rpc-proxy.yourdomain.com/rpc/soneium"
```

## 5. Nginx Reverse Proxy with SSL (Optional)

```bash
# Install Nginx and Certbot
sudo apt install nginx certbot python3-certbot-nginx -y

# Configure Nginx
sudo nano /etc/nginx/sites-available/rpc-proxy
```

```nginx
server {
    listen 80;
    server_name rpc-proxy.yourdomain.com;

    location / {
        proxy_pass http://localhost:8888;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeouts for RPC
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }
}
```

```bash
# Enable site
sudo ln -s /etc/nginx/sites-available/rpc-proxy /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx

# Get SSL certificate
sudo certbot --nginx -d rpc-proxy.yourdomain.com
```

## 6. Monitoring and Maintenance

### Health Check
```bash
# From EC2
curl http://localhost:8888/health

# From outside
curl http://YOUR-EC2-IP:8888/health
```

### View Logs
```bash
# Docker
docker-compose logs -f rpc-proxy

# Systemd
sudo journalctl -u rpc-proxy -f
```

### Update Proxy
```bash
cd ~/golang-proxy
git pull
docker-compose down
docker-compose build
docker-compose up -d
```

## 7. Cost Optimization

### Auto-scaling Group (Advanced)
```yaml
# terraform/cloudformation template
Resources:
  ProxyAutoScalingGroup:
    Type: AWS::AutoScaling::AutoScalingGroup
    Properties:
      MinSize: 1
      MaxSize: 3
      DesiredCapacity: 1
      TargetGroupARNs:
        - !Ref ProxyTargetGroup
      HealthCheckType: ELB
      HealthCheckGracePeriod: 300
```

### Use Application Load Balancer
- Distribute traffic across multiple proxy instances
- Automatic health checks
- SSL termination

## 8. Security Best Practices

1. **Restrict Access**
   ```bash
   # IP whitelist in Security Group
   # Or use VPN/Private connectivity
   ```

2. **Environment Variables**
   ```bash
   # Use AWS Secrets Manager
   aws secretsmanager get-secret-value --secret-id rpc-proxy-env
   ```

3. **CloudWatch Monitoring**
   ```bash
   # Install CloudWatch agent
   wget https://s3.amazonaws.com/amazoncloudwatch-agent/ubuntu/amd64/latest/amazon-cloudwatch-agent.deb
   sudo dpkg -i amazon-cloudwatch-agent.deb
   ```

4. **Backup Configuration**
   ```bash
   # Backup to S3
   aws s3 sync /home/ubuntu/golang-proxy s3://your-backup-bucket/rpc-proxy/
   ```

## Quick Start Commands

```bash
# 1. Launch EC2 instance (t3.small, Ubuntu 22.04)
# 2. SSH into instance
ssh -i your-key.pem ubuntu@ec2-ip

# 3. Run installation
curl -O https://raw.githubusercontent.com/your-repo/golang-proxy/main/deploy-proxy.sh
chmod +x deploy-proxy.sh
./deploy-proxy.sh

# 4. Test
curl http://localhost:8888/health

# 5. Update your subgraphs to use EC2 IP
echo "ethereum: soneium:http://EC2-IP:8888/rpc/soneium"
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Connection refused | Check Security Group rules |
| Timeout | Increase proxy timeout settings |
| High latency | Use same region/AZ as subgraphs |
| SSL issues | Check Nginx and Certbot logs |
| Out of memory | Upgrade instance type |