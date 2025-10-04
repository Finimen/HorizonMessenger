<div align="center">

# Horizon Messenger

> **Secure, Real-Time Messaging Platform with Proprietary Technology**

![Status](https://img.shields.io/badge/Status-Active-success)
![License](https://img.shields.io/badge/License-Proprietary-red)
![Version](https://img.shields.io/badge/Version-0.14B-blue)
![Platform](https://img.shields.io/badge/Platform-Web_Mobile_Desktop-informational)

</div>

## 🚀 Overview

**Horizon Messenger** is a cutting-edge real-time messaging platform designed for secure, focused communication. Built with modern technologies and a privacy-first approach, it offers a clean alternative to overloaded messaging apps.

---

### Lead Architect
**Finimen Sniper** - 📧 finimensniper@gmail.com

---

<div align="center">

*Copyright (c) 2025 Finimen Sniper / FSC. All rights reserved.*

</div>

---

## 🛡️ Core Security Architecture

### Military-Grade Encryption
- **End-to-End** Encryption for all messages
- **JWT** with **SHA-256** token revocation system
- **Bcrypt** password hashing with configurable cost
- Automatic token invalidation via **Redis** blacklisting
- **Rate limiting** with sliding window algorithm
- **CSP & Security** Headers protection

### Zero-Trust Security Model
```go
// Multi-layered security middleware chain
gin.Use(services.SecurityMiddleware())
gin.Use(services.RequestIDMiddleware())
gin.Use(RateLimitMiddleware(limiter))
gin.Use(AuthMiddleware())
```

---

## 🏗️ System Architecture
Microservices Pattern

## 🛠 Technology Stack

### Backend Infrastructure
- **Go 1.21+** - High-performance concurrent runtime
- **Gin Framework** - Enterprise HTTP web framework
- **PostgreSQL** - ACID-compliant relational database
- **Redis** - In-memory data structure store
- **WebSocket** - Full-duplex real-time communication
- **Jaeger** - Distributed tracing system
- **Prometheus** - Systems monitoring and alerting toolkit

### Production Monitoring
**OpenTelemetry** - Vendor-agnostic observability framework
**Structured Logging** - JSON logging with slog
**Health Checks** - K8s-ready liveness/readiness probes
**Metrics Export** - Prometheus metrics endpoint

### Enterprise Authentication
```go
type AuthService struct {
    userRepo     ports.IUserRepository
    hasher       ports.IHasher  
    tokenRepo    ports.TokenRepository
    emailService ports.IEmailService
    // ... JWT, validation, revocation
}
```

### Advanced Chat Management
- Multi-user group chats with participant management
- Message pagination with configurable limits
- Chat member permissions and moderation
- Message encryption at rest and in transit
- Real-time notifications for chat events

---

## 🔧 Core Features
### Real-Time Communication
- WebSocket Cluster with hub-based message broadcasting
- Horizontal scaling ready architecture
- Automatic reconnection with session recovery
- Typing indicators and presence tracking
- Message persistence with configurable retention

<div align="center">

# 📡 API Specification
## Authentication Endpoints

| Method | Endpoint                      | Description                           | Security |
|--------|-------------------------------|---------------------------------------|----------|
| POST   | `/api/auth/register`          | User registration with email verification | Public   |
| POST   | `/api/auth/login`             | JWT token issuance                    | Public   |
| POST   | `/api/auth/logout`            | Token revocation                      | Bearer   |
| GET    | `/api/auth/verify-email`      | Email confirmation                    | Public   |
| GET    | `/api/auth/verification-status` | Check verification status             | Bearer   |

## Chat Management

| Method | Endpoint                       | Description                     | Security |
|--------|--------------------------------|---------------------------------|----------|
| POST   | `/api/chats`                   | Create new chat/group           | Bearer   |
| GET    | `/api/chats`                   | List user's chats               | Bearer   |
| GET    | `/api/chats/{id}/messages`     | Paginated message history       | Bearer   |
| DELETE | `/api/chats/{id}`              | Delete chat (members only)      | Bearer   |

## Real-Time Endpoints

| Method | Endpoint   | Description               | Protocol |
|--------|------------|---------------------------|----------|
| GET    | `/api/ws`  | WebSocket connection upgrade | WS       |

</div>

---

## 🚀 Deployment & Operations
### Health Monitoring
```yaml
# Kubernetes readiness/liveness probes
/live  - Application liveness
/ready - Service readiness  
/health - Dependency health (DB, Redis)
```
### Configuration Management
```go
type Config struct {
    Environment EnvironmentConfig
    Server      ServerConfig
    Database    DatabaseConfig
    Redis       RedisConfig
    JWT         JWTConfig
    RateLimit   RateLimitConfig
    Email       EmailConfig
    Tracing     TracingConfig
}
```
### Observability Stack
- Distributed Tracing with Jaeger integration
- Structured Logging with correlation IDs
- Custom Metrics for business logic monitoring
- Performance Tracing for bottleneck identification

---

## 🔐 Security Implementation
### Token Management
```go
// Automatic token revocation
func (s *AuthService) RevokeToken(ctx context.Context, tokenString string, expiration time.Duration) error {
    hash := sha256.Sum256([]byte(tokenString))
    tokenHash := hex.EncodeToString(hash[:])
    return s.tokenRepo.Revoke(ctx, tokenHash, expiration)
}
```
### Rate Limiting
- Sliding window algorithm implementation
- IP-based and user-based limiting
- Configurable thresholds and windows
- Redis-backed for distributed consistency

---

## 📊 Performance Characteristics
### Benchmarks
- Response Time: < 50ms for API endpoints
- WebSocket Latency: < 10ms message delivery
- Concurrent Connections: 10k+ per instance
- Message Throughput: 50k+ messages/second

### Scalability Features
- Stateless authentication for horizontal scaling
- Connection pooling with configurable limits
- Database connection management with health checks
- Memory-efficient WebSocket hub implementation

---

## 🛠️ Development Setup
## Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Redis 6+
- SMTP server (for email verification)

# Quick Start
```bash
# Clone repository
git clone https://github.com/Finimen/SafeMassager

# Configure environment
cp config/app.example.yaml config/app.yaml

# Run migrations and start
go run main.go

# Or run all proejct
go run .
```

---

## Screens
<img width="1919" height="1079" alt="Screenshot 2025-10-02 094553" src="https://github.com/user-attachments/assets/432156f0-4060-47dd-9b5b-a94c783eb680" />
<img width="1919" height="1079" alt="Screenshot 2025-10-02 094548" src="https://github.com/user-attachments/assets/32c18694-798d-459b-b0a4-d105f50acdf9" />
<img width="1919" height="1075" alt="Screenshot 2025-10-02 094517" src="https://github.com/user-attachments/assets/f0bee90d-620e-4fd6-9413-87c3edfca195" />
<img width="1918" height="1079" alt="Screenshot 2025-10-02 094523" src="https://github.com/user-attachments/assets/6fff0181-9ba0-47f3-947c-690b20083573" />
<img width="1919" height="1079" alt="Screenshot 2025-10-02 094440" src="https://github.com/user-attachments/assets/8a22581d-ad1f-40e3-8851-421030e1843d" />
<img width="1916" height="1079" alt="Screenshot 2025-10-02 094451" src="https://github.com/user-attachments/assets/acb0c461-20ec-4927-a741-5bdd12511e37" />
<img width="1919" height="1079" alt="Screenshot 2025-10-02 094540" src="https://github.com/user-attachments/assets/c37eba7a-0fc0-414f-992c-370791cbad99" />

---

# 🔮 Roadmap
## Q1 2026
### Message encryption key rotation
### File attachment support
### Mobile application clients

## Q2 2026
### Voice/video call integration
### Message reactions and replies
### Advanced search functionality

## Q3 2026
### Federation protocol support
### Plugin system for extensions
### Enterprise SSO integration

---

# CONFIDENTIALITY NOTICE

This project contains proprietary intellectual property.
All source code, documentation, and related materials are
confidential information protected by copyright law.

Any unauthorized copying, distribution, or usage is
prohibited and may result in legal consequences.

© 2025 Finimen Sniper / FSC. All rights reserved.
