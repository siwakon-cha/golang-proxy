# RPC Proxy System Documentation

## ภาพรวมของระบบ

RPC Proxy เป็นระบบ load balancer และ health checker สำหรับ Ethereum RPC endpoints โดยมีเป้าหมายหรือเป็นตัวกลางในการกระจายการเรียกใช้งาน RPC requests ไปยัง endpoints ต่างๆ และตรวจสอบสถานะของ endpoints แบบต่อเนื่อง

## สถาปัตยกรรมของระบบ

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│     Client      │───▶│   Proxy Server  │───▶│  RPC Endpoints  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │ Health Checker  │
                       └─────────────────┘
                              │
                              ▼
                       ┌─────────────────┐
                       │   PostgreSQL    │
                       └─────────────────┘
```

## การทำงานของระบบ

### 1. จุดเริ่มต้น - main.go:18

ระบบเริ่มต้นการทำงานจากไฟล์ `main.go` โดยมีขั้นตอนดังนี้:

1. **โหลดการตั้งค่า** (`main.go:19`)
   - เรียก `config.Load()` เพื่อโหลดการตั้งค่าจาก environment variables, database และ fallback values

2. **สร้าง Health Checker** (`main.go:24`)
   - สร้าง health checker instance พร้อม endpoints และการตั้งค่า

3. **สร้าง Proxy Server** (`main.go:25`)
   - สร้าง proxy server instance

4. **เริ่มต้น Services** (`main.go:27-28`)
   - เริ่มต้น health checker ทำงานแบบ background
   - ตั้งค่า graceful shutdown

### 2. การจัดการการตั้งค่า - internal/config/config.go

**จุดเริ่มต้น**: `config.Load()` (`config.go:58`)

```go
func Load() (*Config, error) {
    // 1. โหลด .env file
    // 2. ตั้งค่า default values
    // 3. โหลด endpoints จาก database
    // 4. โหลด settings จาก database
    // 5. validate การตั้งค่า
}
```

**การทำงาน**:
- โหลดการตั้งค่าจาก environment variables และ .env file
- ถ้ามี database จะโหลด RPC endpoints และ settings จาก database (`config.go:100`)
- ถ้าไม่มี database จะใช้ fallback endpoints (`config.go:113`)
- Override การตั้งค่าด้วยค่าจาก database settings (`config.go:207`)

### 3. Health Checker - internal/health/checker.go

**จุดเริ่มต้น**: `health.NewChecker()` และ `checker.Start()` (`health/checker.go:25,36`)

```go
func (hc *Checker) Start() {
    // เริ่มต้น background goroutine
    go hc.healthCheckLoop()
}
```

**การทำงาน**:
1. **Health Check Loop** (`health/checker.go:60`)
   - ทำงานทุกๆ interval ที่กำหนด (default 30 วินาที)
   - เรียก `performHealthCheck()` เพื่อตรวจสอบ endpoints ทั้งหมด

2. **การตรวจสอบ Endpoint** (`health/checker.go:90`)
   - ส่ง `eth_blockNumber` RPC request ไปยัง endpoint
   - วัด response time
   - ตรวจสอบ response และ block number
   - อัพเดทสถานะ healthy/unhealthy

3. **Failure Handling** (`health/checker.go:170`)
   - นับจำนวนครั้งที่ fail
   - มาร์ค endpoint เป็น unhealthy หลังจาก fail เกินจำนวนครั้งที่กำหนด

### 4. Proxy Server - internal/proxy/server.go

**จุดเริ่มต้น**: `proxy.NewServer()` และ HTTP server (`main.go:25,30`)

**Routes**:
- `/health` - ดูสถานะ health ของ proxy และ endpoints
- `/rpc` และ `/` - รับและ forward RPC requests

**การทำงาน**:

1. **RPC Request Handling** (`proxy/server.go:91`)
   ```go
   func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
       // 1. เลือก healthy endpoint
       // 2. อ่าน request body
       // 3. forward request พร้อม retry logic
       // 4. ส่ง response กลับ
   }
   ```

2. **Endpoint Selection** (`proxy/server.go:144`)
   - ใช้ round-robin algorithm
   - เลือกจาก healthy endpoints เท่านั้น

3. **Request Forwarding** (`proxy/server.go:158`)
   - สร้าง HTTP request พร้อม context
   - คัดลอก headers (ยกเว้น Host และ Content-Length)
   - ส่ง request และรับ response

4. **Retry Logic** (`proxy/server.go:117`)
   - พยายาม 3 ครั้งถ้า request ล้มเหลว
   - เลือก endpoint ใหม่สำหรับแต่ละครั้งที่ retry

### 5. Database Layer

**Models** - `internal/models/models.go`:
- `RPCEndpoint` - เก็บข้อมูล RPC endpoints
- `HealthCheck` - เก็บประวัติการตรวจสอบ health
- `Setting` - เก็บการตั้งค่าระบบ

**Repository Pattern** - `internal/repository/`:
- Interface definitions ใน `interfaces.go`
- GORM implementations ใน `gorm/` directory

## Flow การทำงานหลัก

### 1. เมื่อ Client ส่ง RPC Request

```
Client Request → Proxy Server → Select Healthy Endpoint → Forward Request → Return Response
```

**รายละเอียด**:
1. Client ส่ง POST request ไปที่ `/` หรือ `/rpc`
2. Proxy server เรียก `selectHealthyEndpoint()` เพื่อเลือก endpoint ที่ healthy
3. Forward request ไปยัง endpoint ที่เลือก
4. ถ้า request ล้มเหลว จะ retry กับ endpoint อื่น (สูงสุด 3 ครั้ง)
5. ส่ง response กลับไปให้ client

### 2. Health Check Process

```
Timer Trigger → Check All Endpoints → Update Status → Store Results
```

**รายละเอียด**:
1. Health checker ทำงานทุกๆ 30 วินาที (configurable)
2. ส่ง `eth_blockNumber` request ไปยัง endpoints ทั้งหมดพร้อมกัน (parallel)
3. วัด response time และตรวจสอบ response
4. อัพเดทสถานะ healthy/unhealthy ของแต่ละ endpoint
5. บันทึกผลการตรวจสอบลงฐานข้อมูล (ถ้ามี)

### 3. Configuration Loading

```
Environment Variables → Database Settings → Fallback Values → Final Config
```

**รายละเอียด**:
1. โหลดค่าจาก environment variables และ .env file
2. ถ้ามี database configuration จะ:
   - โหลด RPC endpoints จาก database
   - โหลด settings จาก database เพื่อ override default values
3. ถ้าไม่มี database หรือโหลดไม่สำเร็จ จะใช้ fallback endpoints
4. Validate configuration ก่อนใช้งาน

## ไฟล์สำคัญและหน้าที่

| ไฟล์ | หน้าที่ |
|------|--------|
| `main.go` | จุดเริ่มต้นของระบบ, orchestration |
| `internal/config/config.go` | จัดการการตั้งค่าระบบ |
| `internal/proxy/server.go` | HTTP server และ RPC proxy logic |
| `internal/health/checker.go` | Health checking system |
| `internal/types/types.go` | Type definitions และ data structures |
| `internal/models/models.go` | Database models และ GORM hooks |
| `internal/repository/` | Data access layer |

## การปรับแต่งและการตั้งค่า

### Environment Variables

- `SERVER_PORT` - พอร์ตของ proxy server (default: 8888)
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` - การตั้งค่า database
- `HEALTH_CHECK_INTERVAL` - ช่วงเวลาการตรวจสอบ health (default: 30s)
- `PROXY_TIMEOUT` - timeout สำหรับ proxy requests (default: 10s)
- `FALLBACK_RPC_ENDPOINTS` - backup endpoints เมื่อไม่มี database

### Database Settings

ระบบสามารถปรับแต่งการตั้งค่าผ่าน database ได้ โดยเก็บไว้ในตาราง `settings` ซึ่งจะ override environment variables

สำหรับการใช้งานเบื้องต้น ระบบสามารถทำงานได้โดยไม่ต้องมี database โดยจะใช้ fallback endpoints ที่กำหนดไว้