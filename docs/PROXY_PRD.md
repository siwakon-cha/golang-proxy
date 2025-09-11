# Product Requirements Document (PRD)
## RPC Proxy Service with Health Check & Auto Failover

---

### Document Information
- **Version**: 1.0
- **Date**: July 24, 2025
- **Author**: Development Team
- **Status**: Draft
- **Stakeholders**: Backend Team, DevOps, Product Owner

---

## 1. Executive Summary

### 1.1 Background
ปัจจุบัน subgraph service ยิง request ไปยัง RPC endpoints โดยตรง ซึ่งเมื่อ RPC ล่มจะทำให้ subgraph service หยุดทำงาน ส่งผลกระทบต่อความเสถียรของระบบ

### 1.2 Problem Statement
- **Single Point of Failure**: เมื่อ RPC endpoint หลักล่ม service จะหยุดทำงาน
- **No Health Monitoring**: ไม่มีระบบตรวจสอบสถานะ RPC endpoints
- **Manual Recovery**: ต้องแก้ไข configuration และ restart service เมื่อต้องการเปลี่ยน RPC
- **Downtime Impact**: ส่งผลกระทบต่อ user experience และ business operations

### 1.3 Solution Overview
สร้าง RPC Proxy Service ที่ทำหน้าที่เป็น middleware ระหว่าง subgraph และ RPC endpoints พร้อมความสามารถ:
- Health check และ monitoring
- Automatic failover
- Load balancing
- Centralized RPC management

---

## 2. Business Objectives

### 2.1 Primary Goals
- **เพิ่มความเสถียร (Reliability)**: ลด downtime จาก RPC failures จาก ~5% เป็น <0.1%
- **ปรับปรุง Availability**: เพิ่ม system uptime จาก 95% เป็น 99.9%
- **ลดต้นทุนการดูแล**: ลดเวลาในการ manual intervention จาก 2 ชั่วโมง/สัปดาห์ เป็น 15 นาที/สัปดาห์

### 2.2 Secondary Goals
- **Performance Optimization**: ปรับปรุง response time ด้วย intelligent routing
- **Observability**: เพิ่ม visibility ในการใช้งาน RPC endpoints
- **Cost Optimization**: กระจาย load ระหว่าง RPC providers เพื่อประหยัดค่าใช้จ่าย

---

## 3. Success Metrics

### 3.1 Technical Metrics
| Metric | Current | Target | Measurement |
|--------|---------|--------|-------------|
| System Uptime | 95% | 99.9% | Monthly average |
| RPC Failure Recovery Time | 5-30 minutes | <30 seconds | Automated monitoring |
| Response Time P95 | 2000ms | 1500ms | Application logs |
| Memory Usage | 200MB | <100MB | Container metrics |

### 3.2 Business Metrics
- **Manual Interventions**: ลดจาก 8 ครั้ง/เดือน เป็น <2 ครั้ง/เดือน
- **Incident Resolution Time**: ลดจาก 30 นาที เป็น <5 นาที
- **Cost per Request**: ลด 20% จาก load balancing

---

## 4. User Stories & Requirements

### 4.1 Primary User Stories

#### US-001: Automatic RPC Failover
**As a** subgraph service  
**I want** automatic failover when RPC endpoints are unhealthy  
**So that** my service continues to work without interruption

**Acceptance Criteria:**
- ✅ Detect RPC endpoint failures within 30 seconds
- ✅ Automatically switch to healthy endpoint
- ✅ Resume using recovered endpoint when available
- ✅ Zero manual intervention required

#### US-002: Health Monitoring Dashboard
**As a** DevOps engineer  
**I want** to monitor RPC endpoint health status  
**So that** I can proactively manage infrastructure

**Acceptance Criteria:**
- ✅ Real-time health status for all RPC endpoints
- ✅ Response time metrics
- ✅ Block number synchronization status
- ✅ Historical performance data

#### US-003: Load Balancing
**As a** system administrator  
**I want** requests distributed across healthy RPC endpoints  
**So that** no single endpoint becomes overloaded

**Acceptance Criteria:**
- ✅ Round-robin distribution among healthy endpoints
- ✅ Respect rate limits of individual RPC providers
- ✅ Configurable weighting based on endpoint capacity

### 4.2 Secondary User Stories

#### US-004: Configuration Management
**As a** DevOps engineer  
**I want** to add/remove RPC endpoints without service restart  
**So that** I can manage infrastructure dynamically

#### US-005: Request Logging & Analytics
**As a** developer  
**I want** detailed request/response logging  
**So that** I can debug issues and optimize performance

---

## 5. Functional Requirements

### 5.1 Core Features

#### 5.1.1 Health Check System
- **REQ-001**: ทำ health check ทุก 30 วินาที (configurable)
- **REQ-002**: ใช้ `eth_blockNumber` method สำหรับ health check
- **REQ-003**: Track response time, block synchronization status
- **REQ-004**: Mark endpoint as unhealthy after 3 consecutive failures
- **REQ-005**: Mark endpoint as healthy after 1 successful check

#### 5.1.2 Proxy & Routing
- **REQ-006**: Forward all JSON-RPC requests to healthy endpoints
- **REQ-007**: Preserve original request/response format
- **REQ-008**: Handle HTTP methods: GET, POST, OPTIONS
- **REQ-009**: Support CORS for browser clients
- **REQ-010**: Implement request timeout (configurable, default 10s)

#### 5.1.3 Failover Logic
- **REQ-011**: Automatic failover within 30 seconds of detection
- **REQ-012**: Round-robin selection among healthy endpoints
- **REQ-013**: Retry failed requests on next available endpoint
- **REQ-014**: Circuit breaker pattern to prevent cascade failures

#### 5.1.4 Monitoring & Observability
- **REQ-015**: Expose `/health` endpoint with detailed status
- **REQ-016**: Log all requests with timestamp, endpoint, response time
- **REQ-017**: Export metrics in Prometheus format
- **REQ-018**: Alert on endpoint failures via webhook/email

### 5.2 API Specifications

#### 5.2.1 Health Check Endpoint
```
GET /health
Response: 200 OK
{
  "proxy": "healthy",
  "currentRPC": "https://rpc1.example.com",
  "rpcEndpoints": [
    {
      "url": "https://rpc1.example.com",
      "healthy": true,
      "lastCheck": "2025-07-24T10:30:00Z",
      "responseTime": 150,
      "blockNumber": "0x12a4b2c"
    }
  ]
}
```

#### 5.2.2 Proxy Endpoint
```
POST /rpc
Content-Type: application/json

Request: JSON-RPC 2.0 format
Response: Forward from upstream RPC
```

---

## 6. Non-Functional Requirements

### 6.1 Performance Requirements
- **NFR-001**: Support 1000 concurrent requests
- **NFR-002**: Response time P95 < 1500ms
- **NFR-003**: Memory usage < 100MB under normal load
- **NFR-004**: CPU usage < 50% under normal load

### 6.2 Scalability Requirements
- **NFR-005**: Support up to 10 RPC endpoints
- **NFR-006**: Handle 10,000 requests per minute
- **NFR-007**: Horizontal scaling capability

### 6.3 Reliability Requirements
- **NFR-008**: 99.9% uptime availability
- **NFR-009**: Graceful shutdown with connection draining
- **NFR-010**: Automatic recovery from failures
- **NFR-011**: Data consistency during failover

### 6.4 Security Requirements
- **NFR-012**: No exposure of RPC credentials in logs
- **NFR-013**: Rate limiting per client IP
- **NFR-014**: Input validation for all requests
- **NFR-015**: HTTPS support for production deployment

---

## 7. Technical Architecture

### 7.1 Technology Stack
- **Language**: Go 1.21+
- **Framework**: net/http (standard library)
- **Containerization**: Docker
- **Orchestration**: Docker Compose
- **Monitoring**: Prometheus + Grafana (future)

### 7.2 System Architecture
```
[Subgraph Service] → [RPC Proxy] → [RPC Endpoint 1]
                                 → [RPC Endpoint 2]
                                 → [RPC Endpoint 3]
```

### 7.3 Data Flow
1. Subgraph service sends request to `/rpc`
2. Proxy selects healthy RPC endpoint
3. Forward request to selected endpoint
4. Return response to client
5. Log transaction and update metrics

### 7.4 Configuration Management
```yaml
rpc_endpoints:
  - url: "https://rpc1.example.com"
    weight: 1
  - url: "https://rpc2.example.com" 
    weight: 2

health_check:
  interval: 30s
  timeout: 5s
  retries: 3

proxy:
  timeout: 10s
  max_connections: 1000
```

---

## 8. Integration Requirements

### 8.1 Docker Compose Integration
- **INT-001**: เพิ่ม `rpc-proxy` service ใน docker-compose.yml
- **INT-002**: Graph-node service depends on rpc-proxy
- **INT-003**: Update ethereum environment variable ใน graph-node

### 8.2 Subgraph Integration
- **INT-004**: เปลี่ยน RPC URL จาก direct endpoint เป็น proxy URL
- **INT-005**: ไม่ต้องแก้ไข subgraph code
- **INT-006**: Backward compatibility กับ existing queries

---

## 9. Deployment Plan

### 9.1 Phase 1: Development & Testing (Week 1-2)
- [ ] Implement core proxy functionality
- [ ] Add health check system
- [ ] Create Docker configuration
- [ ] Unit testing และ integration testing

### 9.2 Phase 2: Staging Deployment (Week 3)
- [ ] Deploy to staging environment
- [ ] Performance testing
- [ ] Load testing with simulated traffic
- [ ] Failover testing

### 9.3 Phase 3: Production Deployment (Week 4)
- [ ] Blue-green deployment strategy
- [ ] Monitor system metrics
- [ ] Gradual rollout to 100% traffic
- [ ] Documentation และ runbook

### 9.4 Rollback Plan
- **Rollback Trigger**: Response time > 3000ms หรือ error rate > 1%
- **Rollback Process**:
    1. Revert docker-compose.yml
    2. Restart graph-node with direct RPC endpoints
    3. Monitor recovery metrics

---

## 10. Risk Assessment

### 10.1 Technical Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Proxy becomes single point of failure | High | Medium | Implement proxy clustering |
| Performance degradation | Medium | Low | Load testing และ optimization |
| Configuration errors | Medium | Medium | Validation และ staged rollout |
| Memory leaks | High | Low | Comprehensive testing |

### 10.2 Business Risks

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Extended downtime during deployment | High | Low | Blue-green deployment |
| Increased infrastructure costs | Low | Medium | Cost monitoring |
| Team knowledge gap | Medium | Medium | Documentation และ training |

---

## 11. Success Criteria & Definition of Done

### 11.1 Technical DoD
- [ ] All functional requirements implemented และ tested
- [ ] Performance benchmarks met
- [ ] Security scan passed
- [ ] Code review completed
- [ ] Documentation updated

### 11.2 Business DoD
- [ ] System uptime > 99.9% for 30 days
- [ ] Manual interventions < 2 per month
- [ ] Response time improvement achieved
- [ ] Stakeholder sign-off received

### 11.3 Go-Live Criteria
- [ ] All tests passed (unit, integration, load)
- [ ] Monitoring และ alerting configured
- [ ] Runbook และ troubleshooting guide ready
- [ ] Team trained on new system
- [ ] Rollback plan validated

---

## 12. Future Enhancements

### 12.1 Phase 2 Features (Q4 2025)
- **Advanced Load Balancing**: Weighted round-robin based on response time
- **Caching Layer**: Cache frequent requests to reduce RPC load
- **Rate Limiting**: Per-client rate limiting
- **Metrics Dashboard**: Grafana dashboard for monitoring

### 12.2 Phase 3 Features (Q1 2026)
- **Multi-Chain Support**: Support multiple blockchain networks
- **Request Routing**: Route requests based on method type
- **Cost Optimization**: Choose endpoints based on pricing
- **Machine Learning**: Predictive endpoint selection

---

## 13. Appendices

### 13.1 Glossary
- **RPC**: Remote Procedure Call - blockchain node communication protocol
- **Subgraph**: The Graph protocol indexing service
- **Failover**: Automatic switching to backup system
- **Circuit Breaker**: Pattern to prevent cascade failures

### 13.2 References
- [The Graph Documentation](https://thegraph.com/docs/)
- [JSON-RPC Specification](https://www.jsonrpc.org/specification)
- [Go HTTP Reverse Proxy](https://pkg.go.dev/net/http/httputil#ReverseProxy)
- [Docker Best Practices](https://docs.docker.com/develop/best-practices/)

---

**Document Status**: Draft  
**Next Review Date**: July 31, 2025  
**Approval Required**: Product Owner, Tech Lead, DevOps Lead