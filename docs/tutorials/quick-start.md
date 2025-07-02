# Guild Interactive Tutorial

Welcome to the Guild interactive tutorial! This hands-on guide will teach you Guild's core features by building a real project together.

## Tutorial Structure

Each lesson includes:
- 🎯 **Objective**: What you'll learn
- 📚 **Concepts**: Key ideas explained
- 💻 **Practice**: Hands-on exercises
- ✅ **Validation**: Check your understanding

## Lesson 1: Your First Commission

### 🎯 Objective
Create your first commission and watch artisans collaborate to build a simple API.

### 📚 Concepts

**Commissions** are how you give projects to your guild. Think of them as detailed work orders that Elena (your project manager) uses to coordinate the team.

### 💻 Practice

1. **Start Guild**:
   ```bash
   guild chat
   ```

2. **Create a Commission**:
   Type the following message:
   ```
   I want to build a simple REST API that returns random quotes. It should have one endpoint: GET /quote
   ```

3. **Answer Elena's Questions**:
   Elena will ask about your preferences. Try these responses:
   - Framework: "Express.js"
   - Language: "JavaScript"
   - Any tests: "Yes, basic tests"

4. **Observe the Planning**:
   Watch as Elena creates a plan. You should see something like:
   ```
   Elena: Here's the commission plan:
   
   1. Set up Express project (Marcus)
   2. Create quote database (Marcus)
   3. Implement GET /quote endpoint (Marcus)
   4. Write endpoint tests (Vera)
   ```

5. **Approve the Plan**:
   Type: `yes`

### ✅ Validation

Open a new terminal and run:
```bash
guild kanban
```

You should see tasks moving from TODO → IN PROGRESS → DONE.

**Success Criteria**:
- [ ] Commission created successfully
- [ ] Tasks visible on kanban board
- [ ] At least one task completed

---

## Lesson 2: Artisan Collaboration

### 🎯 Objective
Learn how artisans work together and communicate during development.

### 📚 Concepts

Guild's artisans collaborate like a real team:
- **Elena** coordinates and assigns work
- **Marcus** implements features
- **Vera** ensures quality

### 💻 Practice

1. **Watch Real-time Progress**:
   In your chat, you'll see updates like:
   ```
   Marcus: I'm setting up the Express project structure...
   Marcus: Created app.js with basic Express configuration
   
   Vera: I'll prepare the test framework while Marcus works on the implementation
   ```

2. **Ask for Status**:
   Type: `@elena what's the current status?`
   
   Elena will respond with:
   ```
   Elena: Here's our progress:
   ✅ Project setup complete
   🔄 Quote endpoint in progress (Marcus)
   🔄 Test framework setup (Vera)
   ⏳ Endpoint tests pending
   ```

3. **Request Changes**:
   Type: `@marcus can you add a /quotes endpoint that returns all quotes?`
   
   Watch as Marcus acknowledges and implements the change.

### ✅ Validation

Check the generated code:
```bash
cd quotes-api
cat app.js
```

You should see both endpoints implemented.

---

## Lesson 3: Handling Blockers

### 🎯 Objective
Learn how to resolve blocked tasks when artisans need human input.

### 📚 Concepts

Artisans may encounter **blockers** - situations requiring human decisions or clarification. Guild's review system makes resolution easy.

### 💻 Practice

1. **Create a Blocker Scenario**:
   Type: `@marcus add user authentication to the API`
   
   Marcus might respond:
   ```
   Marcus: I need clarification on the authentication approach. Should I use:
   1. JWT tokens
   2. API keys
   3. Basic auth
   
   I'm marking this task as blocked pending your decision.
   ```

2. **Check Blocked Tasks**:
   ```bash
   ls .guild/kanban/review/
   ```
   
   You'll see a file like `20240115_142345_AUTH-001.md`

3. **Resolve the Blocker**:
   ```bash
   $EDITOR .guild/kanban/review/20240115_142345_AUTH-001.md
   ```
   
   Add your resolution:
   ```yaml
   resolution:
     action: provide_info
     details: Use JWT tokens with 24h expiration
     provided_info:
       approach: "JWT"
       expiration: "24h"
       refresh_tokens: false
   ```

4. **Save and Watch**:
   Save the file. In chat, you'll see:
   ```
   System: Task AUTH-001 unblocked
   Marcus: Thanks! I'll implement JWT authentication with 24h tokens.
   ```

### ✅ Validation

The task should move from BLOCKED to IN PROGRESS on the kanban board.

---

## Lesson 4: Knowledge Management

### 🎯 Objective
Teach Guild about your coding standards and patterns.

### 📚 Concepts

Guild's **corpus** is a knowledge base that artisans learn from. Add your team's best practices, patterns, and decisions.

### 💻 Practice

1. **Add a Coding Standard**:
   ```
   /corpus add pattern "Always use async/await instead of callbacks in our Express routes"
   ```

2. **Add an Example**:
   Create a file `good-route.js`:
   ```javascript
   // Good pattern
   app.get('/users', async (req, res) => {
     try {
       const users = await db.getUsers();
       res.json(users);
     } catch (error) {
       res.status(500).json({ error: error.message });
     }
   });
   ```
   
   Then: `/corpus add example ./good-route.js`

3. **Test Knowledge Retrieval**:
   Type: `@marcus implement a GET /users endpoint`
   
   Marcus should follow your pattern:
   ```
   Marcus: I'll implement the /users endpoint following our async/await pattern...
   ```

### ✅ Validation

Check that Marcus used async/await in the implementation.

---

## Lesson 5: Advanced Workflows

### 🎯 Objective
Use Guild's advanced features for complex projects.

### 📚 Concepts

- **Parallel Execution**: Multiple artisans working simultaneously
- **Session Management**: Save and resume work
- **Cost Optimization**: Monitor and control API usage

### 💻 Practice

1. **Start a Complex Project**:
   ```
   I need a full-stack todo application with:
   - React frontend
   - Express backend  
   - PostgreSQL database
   - User authentication
   - Real-time updates via WebSocket
   ```

2. **Watch Parallel Execution**:
   Elena will coordinate parallel work:
   ```
   Elena: I've identified tasks that can run in parallel:
   
   Parallel Group 1:
   - Marcus: Set up backend structure
   - Marcus: Set up frontend structure
   - Marcus: Design database schema
   
   This will save approximately 2 hours.
   ```

3. **Save Your Session**:
   ```
   /session save todo-app-project
   ```

4. **Monitor Costs**:
   Open a new terminal:
   ```bash
   guild cost
   ```

### ✅ Validation

You should see:
- Multiple artisans working simultaneously
- Cost tracking in real-time
- Session saved successfully

---

## Lesson 6: Production Deployment

### 🎯 Objective
Prepare your Guild-built application for production.

### 📚 Concepts

Guild can help with:
- Deployment configuration
- Performance optimization
- Security hardening
- Documentation generation

### 💻 Practice

1. **Request Deployment Prep**:
   ```
   @elena prepare this application for production deployment on AWS
   ```

2. **Review Generated Assets**:
   Elena coordinates the team to create:
   - Dockerfile
   - docker-compose.yml
   - .env.example
   - Deployment documentation
   - GitHub Actions workflow

3. **Security Review**:
   ```
   @vera perform a security audit
   ```
   
   Vera will check for:
   - Exposed secrets
   - Security headers
   - Input validation
   - SQL injection risks

### ✅ Validation

Run the security checks:
```bash
npm audit
npm test -- --coverage
```

---

## Congratulations! 🎉

You've completed the Guild interactive tutorial! You've learned:

✅ Creating commissions
✅ Artisan collaboration
✅ Handling blockers
✅ Knowledge management
✅ Advanced workflows
✅ Production preparation

## Next Steps

1. **Explore Advanced Features**:
   - Custom artisans: `docs/advanced/custom-artisans.md`
   - Tool development: `docs/advanced/custom-tools.md`
   - Integration: `docs/advanced/integrations.md`

2. **Join the Community**:
   - Discord: https://discord.gg/guild-framework
   - GitHub: https://github.com/guild-framework/guild

3. **Share Your Experience**:
   - Tweet with #GuildFramework
   - Star us on GitHub
   - Share your projects

Happy building with Guild! 🏗️
