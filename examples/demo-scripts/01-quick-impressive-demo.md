# Quick Impressive Demo (2 minutes)

**Goal**: Immediate "wow" factor showing Guild's visual superiority over competitors like Aider and Claude Code

## Pre-Demo Setup (30 seconds before recording)

```bash
# Ensure clean environment
cd /path/to/demo-workspace
guild init
cp ../../.guild/commissions/e-commerce-platform.md .guild/commissions/
cp ../../.guild/guild.yaml .guild/
```

## Demo Script

### Opening (0-15 seconds) - "The Commission" 🏰

**Narrator**: "Watch how Guild transforms a complex e-commerce project into coordinated AI development"

**Commands**:

```bash
# Show the rich commission document
cat .guild/commissions/e-commerce-platform.md | head -30
```

**Visual Expectations**:

- ✨ Rich markdown rendering with professional formatting
- 🎨 Headers, emphasis, and structured content display
- 📊 Technical specifications in readable format
- 🏗️ Mermaid architecture diagrams (if rendered)

**Key Visual Differentiator**: Unlike plain-text competitors, Guild displays rich, formatted content

### Action (15-60 seconds) - "Assemble the Guild" ⚔️

**Narrator**: "Six specialized AI agents, each with unique expertise"

**Commands**:

```bash
guild chat --campaign e-commerce
```

**In chat interface**:

```
@service-architect Design the core API architecture for our e-commerce platform with microservices
```

**Expected Rich Response**:

```markdown
# 🏰 E-commerce API Architecture

## Core Microservices Design

### Authentication Service
```go
type AuthService struct {
    db     *sql.DB
    jwt    *JWTManager
    redis  *redis.Client
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
    user, err := s.validateCredentials(ctx, req.Email, req.Password)
    if err != nil {
        return nil, fmt.Errorf("authentication failed: %w", err)
    }

    token, err := s.jwt.GenerateToken(user.ID, user.Roles)
    if err != nil {
        return nil, fmt.Errorf("token generation failed: %w", err)
    }

    return &LoginResponse{
        Token: token,
        User:  user,
    }, nil
}
```

### Database Schema

```sql
-- User Management
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Product Catalog
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    price DECIMAL(10,2) NOT NULL,
    inventory INTEGER NOT NULL DEFAULT 0
);
```

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/auth/login` | User authentication |
| GET | `/api/v1/products` | List products |
| POST | `/api/v1/cart/items` | Add to cart |
| POST | `/api/v1/orders` | Create order |

```

**Visual Expectations**:
- 🤖 Agent status indicator shows: "thinking" → "working" → "complete"
- 🎨 Rich markdown response with headers, code blocks, tables
- 🌈 Syntax highlighting for Go, SQL, and other languages
- 📱 Professional formatting that looks like a real development tool

### Coordination (60-90 seconds) - "Multi-Agent Collaboration" 🤝

**Commands in chat**:
```

@frontend-specialist @backend-specialist Coordinate on the user authentication system implementation

```

**Expected Parallel Responses**:

**@frontend-specialist**:
```typescript
// React Authentication Hook
export const useAuth = () => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const login = async (email: string, password: string) => {
    setLoading(true);
    try {
      const response = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });

      const data = await response.json();
      setUser(data.user);
      localStorage.setItem('auth_token', data.token);
    } catch (error) {
      throw new Error('Login failed');
    } finally {
      setLoading(false);
    }
  };

  return { user, login, loading };
};
```

**@backend-specialist**:

```go
// JWT Middleware for Go
func (s *Server) authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        if token == "" {
            c.JSON(401, gin.H{"error": "Authorization required"})
            c.Abort()
            return
        }

        claims, err := s.jwt.ValidateToken(token)
        if err != nil {
            c.JSON(401, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        c.Set("user_id", claims.UserID)
        c.Set("user_roles", claims.Roles)
        c.Next()
    }
}
```

**Visual Expectations**:

- 👥 Multiple agent status indicators active simultaneously
- 🔄 Visual coordination between frontend and backend specializations
- 💻 Rich code examples in both TypeScript and Go
- 🎯 Clear demonstration of specialized expertise

### Finale (90-120 seconds) - "Real Development Pipeline" 🚀

**Commands**:

```bash
# Show commission refinement and task creation
guild commission refine .guild/commissions/e-commerce-platform.md --create-tasks
```

**Expected Output**:

```
📋 Commission Analysis Complete
🎯 6 Specialized Agents Identified
📦 Microservices Architecture Planned

✅ Tasks Created:
├── 🏗️  API Design & Database Schema (@service-architect)
├── 🎨 React Frontend Components (@frontend-specialist)
├── ⚙️  Go Backend Services (@backend-specialist)
├── 🐳 Docker & Kubernetes Setup (@devops-specialist)
├── 🧪 Testing Strategy & Automation (@qa-specialist)
└── 📚 API Documentation (@documentation-specialist)

🏰 Guild ready for coordinated development!
```

**Visual Expectations**:

- 📊 Commission processing pipeline visualization
- ✅ Task creation with clear agent assignments
- 🎯 Professional kanban-style task organization
- 🏰 Medieval-themed success messaging

**Closing Narration**: "Guild: Where AI agents become productive development teams. Notice the rich content, specialized expertise, and seamless coordination - capabilities that set Guild apart from plain-text alternatives."

## Recording Notes

### Technical Setup

- **Terminal size**: 120x40 characters for optimal readability
- **Theme**: Monokai with medieval purple accents (#6B46C1)
- **Font**: Monospace font, size 14-16 for screen recording
- **Typing speed**: 80-100 WPM with natural pauses

### Timing Breakdown

- **0-15s**: Commission display (focus on rich formatting)
- **15-60s**: Single agent response (highlight syntax highlighting)
- **60-90s**: Multi-agent coordination (show parallel processing)
- **90-120s**: Task creation (demonstrate complete pipeline)

### Key Visual Moments

1. **Rich Markdown**: Headers, emphasis, code blocks rendering beautifully
2. **Syntax Highlighting**: Go, TypeScript, SQL with proper colors
3. **Agent Status**: Visual indicators showing AI thinking/working
4. **Multi-Agent**: Parallel responses from specialized agents
5. **Professional UI**: Clean, development-tool appearance

### Success Criteria

- ✅ Viewers impressed within first 15 seconds
- ✅ Clear visual superiority over Aider/Claude Code
- ✅ Rich content renders without glitches
- ✅ Agent coordination visibly demonstrated
- ✅ Professional development tool appearance
- ✅ Medieval theming adds memorable character

### Error Recovery

- If agent response is slow: "Notice how Guild's specialized agents take time to provide thoughtful, detailed responses"
- If formatting issues: Pre-test all content rendering before recording
- If commands fail: Have backup commands ready and tested

This demo is designed to immediately showcase Guild's visual and coordination advantages while maintaining the professional quality expected in modern development tools.
