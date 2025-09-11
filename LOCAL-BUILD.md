# วิธีรัน Proxy แบบ Local Build (ไม่ต้อง Push Docker Hub)

## 🚀 Quick Start - รัน 2 คำสั่ง

```bash
cd /Users/siwakon.cha/workspace/moonshot/golang-proxy

# Build และรันในคำสั่งเดียว
docker-compose -f docker-compose.local.yml up -d --build
```

## ✅ ตรวจสอบว่าทำงาน

```bash
# ดู logs
docker-compose -f docker-compose.local.yml logs -f

# เช็ค health
curl http://localhost:8888/health

# ทดสอบ Soneium RPC
curl -X POST http://localhost:8888/rpc/soneium \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
```

## 🔧 แก้ไข Configuration

แก้ไขใน `docker-compose.local.yml`:
```yaml
environment:
  # เพิ่ม/แก้ไข RPC endpoints
  - SONEIUM_RPC_ENDPOINTS=https://your-rpc1.com,https://your-rpc2.com
  
  # เพิ่ม Chain ใหม่
  - POLYGON_RPC_ENDPOINTS=https://polygon-rpc.com
  - ARBITRUM_RPC_ENDPOINTS=https://arb1.arbitrum.io/rpc
```

หลังแก้ไข rebuild:
```bash
docker-compose -f docker-compose.local.yml up -d --build
```

## 🔗 เชื่อมต่อ Subgraph

แก้ไขใน subgraph's `docker-compose.yml`:
```yaml
environment:
  # ถ้ารันบนเครื่องเดียวกัน
  ethereum: "soneium:http://host.docker.internal:8888/rpc/soneium"
  
  # ถ้ารันบน EC2/Server อื่น
  ethereum: "soneium:http://YOUR-SERVER-IP:8888/rpc/soneium"
```

## 📝 คำสั่งที่ใช้บ่อย

```bash
# Start พร้อม build
docker-compose -f docker-compose.local.yml up -d --build

# Start (ไม่ build ใหม่)
docker-compose -f docker-compose.local.yml up -d

# Stop
docker-compose -f docker-compose.local.yml down

# Restart
docker-compose -f docker-compose.local.yml restart

# ดู logs
docker-compose -f docker-compose.local.yml logs -f

# ดู container status
docker-compose -f docker-compose.local.yml ps

# Rebuild อย่างเดียว (ไม่ start)
docker-compose -f docker-compose.local.yml build

# Remove ทุกอย่าง รวม volumes
docker-compose -f docker-compose.local.yml down -v
```

## 🚀 Deploy บน EC2/Server

```bash
# 1. Copy files ไปยัง server
scp -r . ubuntu@your-server:/home/ubuntu/golang-proxy

# 2. SSH เข้า server
ssh ubuntu@your-server

# 3. Build และรัน
cd golang-proxy
docker-compose -f docker-compose.local.yml up -d --build

# 4. เปิด Security Group port 8888
```

## 🛠️ Troubleshooting

| ปัญหา | วิธีแก้ |
|-------|---------|
| Build ช้า | ใช้ `.dockerignore` exclude ไฟล์ที่ไม่จำเป็น |
| Port 8888 ถูกใช้แล้ว | เปลี่ยน port ใน docker-compose.local.yml |
| Connection refused | ตรวจสอบ firewall/security group |
| Out of memory | เพิ่ม memory limit ใน docker-compose |

## 💡 Tips

1. **ไม่ต้อง push Docker Hub** - build local ใช้เอง
2. **แก้ code แล้ว rebuild** - ใช้ `--build` flag
3. **ใช้ .env file** - สร้าง `.env` แทนการใส่ใน docker-compose
4. **Multi-stage build** - Dockerfile ใช้ multi-stage ทำให้ image เล็ก

## 📊 Monitor Performance

```bash
# ดู resource usage
docker stats rpc-proxy

# ดู container details
docker inspect golang-proxy_rpc-proxy_1

# เข้าไป debug ใน container
docker exec -it golang-proxy_rpc-proxy_1 sh
```

## 🔄 Update Code

```bash
# 1. แก้ไข code
vim main.go

# 2. Rebuild และ restart
docker-compose -f docker-compose.local.yml up -d --build

# Image ใหม่จะถูก build และ container จะ restart อัตโนมัติ
```