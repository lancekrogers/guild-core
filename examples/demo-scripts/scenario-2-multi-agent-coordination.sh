#!/bin/bash
# Scenario 2: Multi-Agent Coordination
# Duration: 3-4 minutes
# Purpose: Show multiple agents working together on a complex task

echo "🏰 Guild Framework Demo - Scenario 2: Multi-Agent Coordination"
echo "============================================================="
echo ""
echo "🚀 Starting e-commerce campaign with 6 specialized agents..."
echo ""

# Terminal 1 command (shown but not executed in demo script)
echo "Terminal 1 - Campaign Monitor:"
echo "$ guild campaign watch e-commerce"
echo ""
echo "📊 Agent Status Dashboard:"
cat << 'EOF'
┌─ Active Agents ─────────────────────────┐
│ 🟢 service-architect    (idle)          │
│ 🟢 catalog-craftsman    (idle)          │
│ 🟢 payment-sentinel     (idle)          │
│ 🟢 order-orchestrator   (idle)          │
│ 🟢 gateway-guardian     (idle)          │
│ 🟢 deployment-marshal   (idle)          │
└─────────────────────────────────────────┘
EOF

echo ""
read -p "Open a second terminal for chat? Press Enter to continue..."
echo ""

# Terminal 2 command
echo "Terminal 2 - Interactive Chat:"
echo "$ guild chat --campaign e-commerce"
echo ""
echo ">>> Welcome to Guild Chat - E-Commerce Campaign"
echo ">>> 6 specialized agents ready to assist"
echo ""

# Simulate multi-agent task
echo "You: Create the user authentication API with JWT tokens and OAuth2 support"
echo ""
sleep 1

# Show agent coordination
cat << 'EOF'
🤔 service-architect is thinking...
🤔 gateway-guardian is thinking...

ServiceArchitect: I'll design the authentication API with JWT and OAuth2 support. Let me create a comprehensive solution.

⚙️ service-architect is working...
   Designing API endpoints...
   [████████░░░░░░░░] 50%

GatewayGuardian: I'll configure the API gateway for the authentication endpoints with proper rate limiting.

⚙️ gateway-guardian is working...
   Configuring gateway rules...
   [██████░░░░░░░░░░] 35%

ServiceArchitect: Here's the authentication service design:

```go
// auth-service/api/handlers/auth.go
package handlers

import (
    "time"
    "github.com/gin-gonic/gin"
    "github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
    userService  UserService
    tokenService TokenService
    oauthService OAuthService
}

// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
    var req RegisterRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request"})
        return
    }
    
    // Validate and create user
    user, err := h.userService.CreateUser(req)
    if err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Generate tokens
    tokens, err := h.tokenService.GenerateTokenPair(user.ID)
    if err != nil {
        c.JSON(500, gin.H{"error": "Token generation failed"})
        return
    }
    
    c.JSON(201, gin.H{
        "user": user,
        "tokens": tokens,
    })
}
```

GatewayGuardian: Gateway configuration for the auth endpoints:

```yaml
# api-gateway/config/auth-routes.yaml
http:
  routers:
    auth-router:
      rule: "PathPrefix(`/api/v1/auth`)"
      service: auth-service
      middlewares:
        - auth-rate-limit
        - secure-headers
        
  middlewares:
    auth-rate-limit:
      rateLimit:
        average: 10
        burst: 20
        period: 1m
        
    secure-headers:
      headers:
        customRequestHeaders:
          X-Request-ID: "{{.RequestID}}"
        customResponseHeaders:
          X-Content-Type-Options: "nosniff"
          X-Frame-Options: "DENY"
```

✅ service-architect completed API design
✅ gateway-guardian completed gateway configuration

DeploymentMarshal: I'll create the Docker configuration for the auth service:

⚙️ deployment-marshal is working...
   Creating Dockerfile...
   [████████████████] 100%

```dockerfile
# auth-service/Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/auth-service .
EXPOSE 8080
CMD ["./auth-service"]
```

✅ deployment-marshal completed Docker configuration
EOF

echo ""
echo "📊 Task Summary:"
echo "- ServiceArchitect: Created auth API with 5 endpoints"
echo "- GatewayGuardian: Configured rate limiting and security headers"
echo "- DeploymentMarshal: Created optimized Docker build"
echo ""
echo "🎯 This demonstrates Guild's multi-agent coordination:"
echo "- Agents activate based on expertise"
echo "- Work happens in parallel when possible"
echo "- Each agent contributes their specialized knowledge"
echo "- Results are integrated into a cohesive solution"