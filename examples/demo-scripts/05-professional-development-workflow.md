# Professional Development Workflow Demo (7 minutes)

**Goal**: Demonstrate Guild's complete professional development workflow from commission to delivery, showing how it matches and exceeds traditional development team capabilities

## Pre-Demo Setup

```bash
cd /path/to/demo-workspace
guild init
cp ../../.guild/commissions/e-commerce-platform.md .guild/commissions/
cp ../../.guild/guild.yaml .guild/

# Ensure clean kanban state
rm -rf .guild/kanban/*
mkdir -p .guild/kanban/{todo,doing,review,blocked,done}
```

## Demo Script

### Act 1: Project Kickoff (0-90 seconds) 🚀

#### Opening: The Enterprise Challenge (0-30 seconds)

**Narrator**: "Enterprise software development requires more than just coding - it needs project management, code review, documentation, testing, and deployment coordination. Watch Guild deliver a complete professional workflow."

**Commands**:

```bash
guild chat --campaign e-commerce
```

**In Chat**:

```
We're starting a new e-commerce platform project. I need a complete project kickoff including requirements analysis, technical architecture, and development planning.
```

#### Requirements Analysis (30-60 seconds)

**Expected Response from @service-architect**:

```markdown
# 🏰 E-commerce Platform Project Kickoff

## Executive Summary
Based on our commission analysis, I'll lead the technical planning for our microservices e-commerce platform.

## Project Scope Analysis
- **Timeline**: 8-week development cycle
- **Team**: 6 specialized AI agents
- **Architecture**: Microservices with React frontend
- **Complexity**: Enterprise-grade with full observability

## Technical Requirements Summary
| Component | Technology | Responsibility |
|-----------|------------|----------------|
| Frontend | React 18 + TypeScript | @frontend-specialist |
| Backend | Go microservices | @backend-specialist |
| Infrastructure | Docker + Kubernetes | @devops-specialist |
| Quality | Automated testing | @qa-specialist |
| Documentation | API docs + guides | @documentation-specialist |

## Next Steps
1. **Architecture Design** (Week 1)
2. **Core Services Development** (Weeks 2-4)
3. **Integration & Testing** (Weeks 5-6)
4. **Deployment & Documentation** (Weeks 7-8)

**Recommendation**: Begin with API design and database schema to establish contracts for parallel development.
```

#### Sprint Planning (60-90 seconds)

**Commands**:

```bash
guild commission refine .guild/commissions/e-commerce-platform.md --create-sprint-plan
```

**Expected Output**:

```
🏰 Guild Sprint Planning Complete

📊 Project Breakdown:
├── 🎯 Epic 1: User Management System (16 story points)
├── 🎯 Epic 2: Product Catalog Service (20 story points)
├── 🎯 Epic 3: Shopping Cart & Checkout (18 story points)
├── 🎯 Epic 4: Order Management (14 story points)
├── 🎯 Epic 5: Payment Integration (12 story points)
└── 🎯 Epic 6: Admin Dashboard (10 story points)

📋 Sprint 1 Tasks Created:
├── 🏗️ Design authentication API (@service-architect) - 5 pts
├── 🎨 Create login components (@frontend-specialist) - 3 pts
├── ⚙️ Implement auth service (@backend-specialist) - 8 pts
├── 🧪 Auth testing strategy (@qa-specialist) - 3 pts
└── 📚 API documentation (@documentation-specialist) - 2 pts

📁 Kanban Board Initialized:
├── Todo: 5 tasks ready for development
├── Doing: 0 tasks in progress
├── Review: 0 tasks awaiting review
└── Done: 0 tasks completed

🚀 Sprint 1 ready to begin!
```

### Act 2: Development Execution (90-270 seconds) ⚔️

#### Task Assignment & Execution (90-150 seconds)

**Chat Commands**:

```
Let's start Sprint 1! @service-architect please begin with the authentication API design, then hand off to the implementation teams.
```

**@service-architect Response**:

```markdown
# 🏗️ Authentication API Design

## OpenAPI Specification
```yaml
openapi: 3.0.3
info:
  title: E-commerce Authentication API
  version: 1.0.0
  description: Secure authentication service for e-commerce platform

paths:
  /api/v1/auth/login:
    post:
      summary: User login
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [email, password]
              properties:
                email:
                  type: string
                  format: email
                  example: "user@example.com"
                password:
                  type: string
                  minLength: 8
                  example: "SecurePass123!"
      responses:
        '200':
          description: Login successful
          content:
            application/json:
              schema:
                type: object
                properties:
                  access_token:
                    type: string
                    example: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
                  refresh_token:
                    type: string
                    example: "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4="
                  expires_in:
                    type: integer
                    example: 900
                  user:
                    $ref: '#/components/schemas/User'
        '401':
          description: Invalid credentials
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/Error'

components:
  schemas:
    User:
      type: object
      properties:
        id:
          type: string
          format: uuid
        email:
          type: string
          format: email
        first_name:
          type: string
        last_name:
          type: string
        role:
          type: string
          enum: [customer, admin, vendor]
```

## Database Schema

```sql
-- Users table with proper constraints
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100) NOT NULL,
    role VARCHAR(20) DEFAULT 'customer' CHECK (role IN ('customer', 'admin', 'vendor')),
    email_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);

-- Refresh tokens for secure session management
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token VARCHAR(255) UNIQUE NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_id) -- One refresh token per user
);

-- Indexes for performance
CREATE INDEX idx_users_email ON users(email) WHERE deleted_at IS NULL;
CREATE INDEX idx_refresh_tokens_token ON refresh_tokens(token);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

**Handoff Notes:**

- **@frontend-specialist**: Please implement login form with validation
- **@backend-specialist**: Implement this exact API specification
- **@qa-specialist**: Focus on security testing and edge cases
- **@documentation-specialist**: Create user-facing auth documentation

Moving task to DOING status...

```

#### Parallel Development (150-210 seconds)

**Commands**:
```bash
# Show task status update
guild kanban status

# Trigger parallel development
```

**Chat Commands**:

```
Perfect! @frontend-specialist and @backend-specialist please start your implementations in parallel. @qa-specialist prepare test cases.
```

**Expected Multi-Agent Response Showing Parallel Work**:

**@frontend-specialist**:

```typescript
# 🎨 Login Component Implementation

Moving authentication task to DOING...

## React Login Form with Validation
```tsx
// components/auth/LoginForm.tsx
import React, { useState } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useAuth } from '../../hooks/useAuth';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Alert } from '../ui/Alert';

const loginSchema = z.object({
  email: z.string().email('Invalid email address'),
  password: z.string().min(8, 'Password must be at least 8 characters'),
});

type LoginFormData = z.infer<typeof loginSchema>;

export const LoginForm: React.FC = () => {
  const { login, isLoading } = useAuth();
  const [apiError, setApiError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isValid },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
    mode: 'onChange',
  });

  const onSubmit = async (data: LoginFormData) => {
    setApiError(null);

    try {
      await login(data.email, data.password);
      // Navigation handled by auth context
    } catch (error) {
      setApiError(error instanceof Error ? error.message : 'Login failed');
    }
  };

  return (
    <div className="max-w-md mx-auto p-6 bg-white rounded-lg shadow-md">
      <h2 className="text-2xl font-bold text-center mb-6 text-gray-800">
        Sign In
      </h2>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div>
          <Input
            {...register('email')}
            type="email"
            placeholder="Email address"
            error={errors.email?.message}
            autoComplete="email"
          />
        </div>

        <div>
          <Input
            {...register('password')}
            type="password"
            placeholder="Password"
            error={errors.password?.message}
            autoComplete="current-password"
          />
        </div>

        {apiError && (
          <Alert variant="error">
            {apiError}
          </Alert>
        )}

        <Button
          type="submit"
          disabled={!isValid || isLoading}
          loading={isLoading}
          className="w-full"
        >
          {isLoading ? 'Signing in...' : 'Sign In'}
        </Button>
      </form>

      <div className="mt-4 text-center">
        <a href="/forgot-password" className="text-blue-600 hover:underline text-sm">
          Forgot your password?
        </a>
      </div>
    </div>
  );
};
```

**Implementation Status**: ✅ Login form complete
**Next**: Implementing authentication context and API integration
**Dependencies**: Waiting for @backend-specialist API endpoints

```

**@backend-specialist** (parallel response):
```go
# ⚙️ Authentication Service Implementation

Moving task to DOING status...

## Go Authentication Service
```go
// internal/auth/handler.go
package auth

import (
    "context"
    "net/http"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/go-playground/validator/v10"
)

type Handler struct {
    authService AuthService
    validator   *validator.Validate
    logger      Logger
}

type LoginRequest struct {
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8"`
}

type LoginResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int64  `json:"expires_in"`
    User         User   `json:"user"`
}

func NewHandler(authService AuthService, logger Logger) *Handler {
    return &Handler{
        authService: authService,
        validator:   validator.New(),
        logger:      logger,
    }
}

func (h *Handler) Login(c *gin.Context) {
    var req LoginRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": map[string]interface{}{
                "code":    "INVALID_REQUEST",
                "message": "Invalid request format",
                "details": err.Error(),
            },
        })
        return
    }

    if err := h.validator.Struct(req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{
            "error": map[string]interface{}{
                "code":    "VALIDATION_ERROR",
                "message": "Validation failed",
                "details": formatValidationErrors(err),
            },
        })
        return
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
    defer cancel()

    response, err := h.authService.Login(ctx, req.Email, req.Password)
    if err != nil {
        h.logger.Errorf("Login failed for email %s: %v", req.Email, err)

        switch err {
        case ErrInvalidCredentials:
            c.JSON(http.StatusUnauthorized, gin.H{
                "error": map[string]interface{}{
                    "code":    "INVALID_CREDENTIALS",
                    "message": "Invalid email or password",
                },
            })
        case ErrAccountLocked:
            c.JSON(http.StatusLocked, gin.H{
                "error": map[string]interface{}{
                    "code":    "ACCOUNT_LOCKED",
                    "message": "Account temporarily locked due to too many failed attempts",
                },
            })
        default:
            c.JSON(http.StatusInternalServerError, gin.H{
                "error": map[string]interface{}{
                    "code":    "INTERNAL_ERROR",
                    "message": "An unexpected error occurred",
                },
            })
        }
        return
    }

    h.logger.Infof("Successful login for user %s", response.User.ID)
    c.JSON(http.StatusOK, response)
}

func (h *Handler) Logout(c *gin.Context) {
    userID := c.GetString("user_id")
    if userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
        return
    }

    ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
    defer cancel()

    if err := h.authService.Logout(ctx, userID); err != nil {
        h.logger.Errorf("Logout failed for user %s: %v", userID, err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Logout failed"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
}

// Route registration
func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
    auth := router.Group("/auth")
    {
        auth.POST("/login", rateLimitMiddleware(5, time.Minute), h.Login)
        auth.POST("/logout", authMiddleware(), h.Logout)
        auth.POST("/refresh", h.RefreshToken)
        auth.GET("/profile", authMiddleware(), h.GetProfile)
    }
}
```

**Implementation Status**: ✅ API handlers complete
**Next**: Database integration and JWT token management
**Integration**: Ready for @frontend-specialist API calls

```

#### Code Review Process (210-270 seconds)

**Commands**:
```bash
# Simulate task completion and review process
guild kanban move auth-api-design doing review
guild kanban move auth-frontend-impl doing review
guild kanban move auth-backend-impl doing review
```

**Chat Commands**:

```
@qa-specialist please review the authentication implementations and provide feedback
```

**@qa-specialist Review Response**:

```markdown
# 🧪 Authentication Implementation Review

## Code Review Summary
I've reviewed the authentication implementations from @frontend-specialist and @backend-specialist.

### ✅ Strengths Identified
1. **Security**: Proper password validation and JWT implementation
2. **Error Handling**: Comprehensive error cases covered
3. **Validation**: Both client and server-side validation implemented
4. **User Experience**: Loading states and clear error messages

### ⚠️ Issues Found & Recommendations

#### Frontend Issues
```typescript
// ISSUE: Missing CSRF protection
// CURRENT:
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ email, password }),
});

// RECOMMENDED:
const response = await fetch('/api/v1/auth/login', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
    'X-CSRF-Token': getCsrfToken(), // Add CSRF protection
  },
  credentials: 'include', // Include cookies for CSRF
  body: JSON.stringify({ email, password }),
});
```

#### Backend Issues

```go
// ISSUE: Missing rate limiting per user
// CURRENT: Rate limiting only by IP
auth.POST("/login", rateLimitMiddleware(5, time.Minute), h.Login)

// RECOMMENDED: Add per-user rate limiting
auth.POST("/login",
    rateLimitMiddleware(5, time.Minute),
    userRateLimitMiddleware(3, time.Minute), // 3 attempts per user per minute
    h.Login)

// ISSUE: Password should have complexity requirements
func validatePassword(password string) error {
    if len(password) < 12 { // Increase from 8 to 12
        return errors.New("password must be at least 12 characters")
    }

    // Add complexity requirements
    hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
    hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
    hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
    hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

    if !hasUpper || !hasLower || !hasNumber || !hasSpecial {
        return errors.New("password must contain uppercase, lowercase, number, and special character")
    }

    return nil
}
```

### 🧪 Test Coverage Analysis

```go
// Required test cases to add:
func TestAuthenticationSecurity(t *testing.T) {
    t.Run("SQL injection prevention", testSQLInjectionPrevention)
    t.Run("Timing attack prevention", testTimingAttackPrevention)
    t.Run("Rate limiting enforcement", testRateLimiting)
    t.Run("Token expiry handling", testTokenExpiry)
    t.Run("CSRF protection", testCSRFProtection)
}
```

### 📋 Review Checklist

- ✅ API specification compliance
- ✅ Error handling comprehensive
- ✅ Input validation present
- ⚠️ Security hardening needed (CSRF, rate limiting)
- ⚠️ Password complexity insufficient
- ✅ Code structure and readability good
- ⚠️ Test coverage needs improvement

**Recommendation**: Address security issues before moving to DONE. Implementations are solid but need hardening for production.

**Status**: Moving back to DOING for security improvements.

```

### Act 3: Quality Assurance & Deployment (270-420 seconds) 🛡️

#### Security Improvements (270-330 seconds)

**Chat Commands**:
```

Thanks @qa-specialist! @frontend-specialist and @backend-specialist please address the security concerns before we proceed.

```

**Visual Expectation**: Quick updates from both specialists addressing security issues

#### Documentation & Deployment Prep (330-390 seconds)

**Chat Commands**:
```

@documentation-specialist create the final API documentation and @devops-specialist prepare deployment configuration

```

**Expected Documentation Response**:
```markdown
# 📚 Authentication API Documentation

## Overview
Complete authentication system with JWT tokens and refresh token rotation.

## Quick Start
```bash
# Login
curl -X POST https://api.ecommerce.com/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "SecurePass123!"}'

# Response
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "refresh_token": "dGhpcyBpcyBhIHJlZnJlc2ggdG9rZW4=",
  "expires_in": 900,
  "user": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe",
    "role": "customer"
  }
}
```

## Integration Examples

[Comprehensive examples for different frameworks]

```

#### Final Integration Test (390-420 seconds)

**Commands**:
```bash
guild commission status e-commerce-platform
```

**Expected Output**:

```
🏰 E-commerce Platform Commission Status

📊 Sprint 1 Completion:
├── ✅ Authentication API Design (5 pts) - @service-architect
├── ✅ Login Components (3 pts) - @frontend-specialist
├── ✅ Auth Service Implementation (8 pts) - @backend-specialist
├── ✅ Security Testing (3 pts) - @qa-specialist
└── ✅ API Documentation (2 pts) - @documentation-specialist

🎯 Sprint 1 Results:
├── Story Points Completed: 21/21 (100%)
├── Code Review Issues: 0 critical, 2 minor resolved
├── Test Coverage: 95% (target: 90%+)
├── Documentation: Complete with examples
└── Security Score: A+ (all vulnerabilities addressed)

📈 Project Health:
├── Velocity: On track for 8-week timeline
├── Quality: All standards met
├── Coordination: Excellent inter-agent collaboration
└── Deliverables: Production-ready authentication system

🚀 Ready for Sprint 2: Product Catalog Development
```

## Closing: Professional Excellence (390-420 seconds) 🏆

**Narrator**: "This is professional software development - complete with project planning, parallel development, code review, security hardening, and comprehensive documentation. Guild delivers the full enterprise development experience with AI agents that work like a real development team."

## Recording Notes

### Professional Elements Demonstrated

1. **Project Management**: Sprint planning, task tracking, status reporting
2. **Code Review**: Formal review process with specific feedback
3. **Quality Assurance**: Security testing, standards compliance
4. **Documentation**: Complete API docs and integration guides
5. **Team Coordination**: Handoffs, parallel work, issue resolution

### Key Visual Moments

1. **Sprint Planning**: Professional project breakdown
2. **Parallel Development**: Multiple agents working simultaneously
3. **Code Review Process**: Formal feedback and improvements
4. **Quality Gates**: Security and testing standards
5. **Project Status**: Professional reporting and metrics

### Success Criteria

- ✅ Complete professional development workflow
- ✅ Real project management processes
- ✅ Quality assurance and security practices
- ✅ Comprehensive documentation
- ✅ Team coordination and collaboration
- ✅ Enterprise-grade deliverables

This demo positions Guild as a complete professional development solution, not just a coding assistant.
