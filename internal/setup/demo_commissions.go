// Copyright (C) 2025 SWS Industries LLC (DBA Blockhead Consulting)
// SPDX-License-Identifier: LicenseRef-ANGRY-GOAT-0.2

package setup

import (
	"context"
	"fmt"
	"strings"

	"github.com/lancekrogers/guild/pkg/gerror"
)

// DemoCommissionType represents the type of demo commission
type DemoCommissionType string

const (
	DemoTypeAPIService    DemoCommissionType = "api"
	DemoTypeWebApp        DemoCommissionType = "webapp"
	DemoTypeCLITool       DemoCommissionType = "cli"
	DemoTypeDataAnalysis  DemoCommissionType = "data"
	DemoTypeMicroservices DemoCommissionType = "microservices"
	DemoTypeAI            DemoCommissionType = "ai"
	DemoTypeDefault       DemoCommissionType = "default"
)

// DemoCommission contains a demo commission template
type DemoCommission struct {
	Type        DemoCommissionType
	Title       string
	Description string
	Content     string
	Tags        []string
}

// DemoCommissionGenerator handles creation of demo commissions
type DemoCommissionGenerator struct {
	templates map[DemoCommissionType]DemoCommission
}

// NewDemoCommissionGenerator creates a new demo commission generator
func NewDemoCommissionGenerator() *DemoCommissionGenerator {
	return &DemoCommissionGenerator{
		templates: initializeDemoTemplates(),
	}
}

// GenerateCommission generates a demo commission based on the type
func (g *DemoCommissionGenerator) GenerateCommission(ctx context.Context, demoType DemoCommissionType) (string, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return "", gerror.Wrap(err, gerror.ErrCodeCancelled, "commission generation cancelled").
			WithComponent("DemoCommissions").WithOperation("GenerateCommission")
	}

	template, exists := g.templates[demoType]
	if !exists {
		// Fall back to default if type not found
		template = g.templates[DemoTypeDefault]
	}

	return template.Content, nil
}

// GetAvailableTypes returns all available demo commission types
func (g *DemoCommissionGenerator) GetAvailableTypes() []DemoCommissionType {
	types := make([]DemoCommissionType, 0, len(g.templates))
	for t := range g.templates {
		if t != DemoTypeDefault { // Exclude default from the list
			types = append(types, t)
		}
	}
	return types
}

// GetDemoDescription returns a description for a demo type
func (g *DemoCommissionGenerator) GetDemoDescription(demoType DemoCommissionType) string {
	if template, exists := g.templates[demoType]; exists {
		return template.Description
	}
	return "Unknown demo type"
}

// InferDemoType attempts to infer the best demo type based on project context
func (g *DemoCommissionGenerator) InferDemoType(ctx context.Context, projectPath string) (DemoCommissionType, error) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return DemoTypeDefault, gerror.Wrap(err, gerror.ErrCodeCancelled, "demo type inference cancelled").
			WithComponent("DemoCommissions").WithOperation("InferDemoType")
	}

	// This is a simple implementation - in a real scenario, we would:
	// 1. Check for presence of package.json -> webapp
	// 2. Check for go.mod with main package -> api or cli
	// 3. Check for requirements.txt or notebooks -> data
	// 4. Check for ML frameworks -> ai
	// For now, return default
	return DemoTypeDefault, nil
}

// initializeDemoTemplates creates all demo commission templates
func initializeDemoTemplates() map[DemoCommissionType]DemoCommission {
	templates := make(map[DemoCommissionType]DemoCommission)

	// API Service Demo
	templates[DemoTypeAPIService] = DemoCommission{
		Type:        DemoTypeAPIService,
		Title:       "RESTful Task Management API",
		Description: "Build a production-ready REST API with authentication, testing, and documentation",
		Tags:        []string{"api", "backend", "rest", "demo"},
		Content: `# 🚀 RESTful Task Management API

> **Project Objective**: Create a production-ready REST API that demonstrates modern backend development practices, showcasing Guild's ability to coordinate multiple agents for API design, implementation, testing, and documentation.

## 🎯 Project Overview

Build a comprehensive task management API that serves as the backend for productivity applications. This project will demonstrate clean architecture, proper error handling, authentication, and extensive test coverage - perfect for showcasing multi-agent collaboration.

## 🛠 Technical Requirements

### Core Technologies
- **Language**: Go (latest stable)
- **Framework**: Standard library net/http with chi router
- **Database**: PostgreSQL with migrations
- **Authentication**: JWT with refresh tokens
- **Documentation**: OpenAPI 3.0 specification
- **Testing**: Table-driven tests with mocks

### API Endpoints

#### Authentication
- POST /auth/register - User registration
- POST /auth/login - User login with JWT
- POST /auth/refresh - Refresh access token
- POST /auth/logout - Invalidate refresh token

#### Task Management
- GET /api/v1/tasks - List tasks (with pagination & filtering)
- POST /api/v1/tasks - Create new task
- GET /api/v1/tasks/{id} - Get task details
- PUT /api/v1/tasks/{id} - Update task
- DELETE /api/v1/tasks/{id} - Delete task
- PATCH /api/v1/tasks/{id}/status - Update task status

#### Project Organization
- GET /api/v1/projects - List user projects
- POST /api/v1/projects - Create project
- GET /api/v1/projects/{id}/tasks - Get project tasks
- POST /api/v1/projects/{id}/collaborators - Add collaborator

## 📋 Implementation Tasks

### Phase 1: Foundation (Manager & Architect Agents)
- [ ] Design database schema with proper relationships
- [ ] Set up project structure following clean architecture
- [ ] Configure PostgreSQL with connection pooling
- [ ] Implement database migrations system
- [ ] Create base error handling middleware

### Phase 2: Core Features (Developer Agents)
- [ ] Implement user registration and authentication
- [ ] Build JWT middleware with token validation
- [ ] Create CRUD operations for tasks
- [ ] Add project management endpoints
- [ ] Implement proper validation for all inputs

### Phase 3: Advanced Features (Specialist Agents)
- [ ] Add pagination, filtering, and sorting
- [ ] Implement rate limiting middleware
- [ ] Create background job for notifications
- [ ] Add Redis caching for frequently accessed data
- [ ] Build activity logging system

### Phase 4: Quality & Documentation (Tester & Documenter Agents)
- [ ] Write comprehensive unit tests (>80% coverage)
- [ ] Create integration tests for all endpoints
- [ ] Generate OpenAPI documentation
- [ ] Add request/response examples
- [ ] Create API client SDK

## 🎨 Code Quality Requirements

- **Error Handling**: Consistent error responses with proper HTTP codes
- **Validation**: Input validation with detailed error messages
- **Security**: SQL injection prevention, XSS protection
- **Performance**: Query optimization, connection pooling
- **Monitoring**: Structured logging with correlation IDs

## 🧪 Testing Strategy

1. **Unit Tests**: All business logic with mocked dependencies
2. **Integration Tests**: Database operations with test containers
3. **API Tests**: Full request/response cycle testing
4. **Load Tests**: Performance benchmarks for concurrent requests
5. **Security Tests**: Authentication and authorization edge cases

## 📊 Success Metrics

- ✅ All endpoints return < 100ms response time
- ✅ 100% of endpoints documented with examples
- ✅ Test coverage > 80% for critical paths
- ✅ Zero security vulnerabilities in dependencies
- ✅ Handles 1000+ concurrent connections

## 🚀 Demonstration Value

This project showcases Guild's ability to:
- **Coordinate** multiple specialized agents effectively
- **Design** robust API architecture from specifications
- **Implement** production-ready code with best practices
- **Test** comprehensively across different levels
- **Document** APIs for easy integration

> 💡 **Why This Demo?** APIs are the backbone of modern applications. This demo shows how Guild can handle the full lifecycle of API development, from design to deployment-ready code.`,
	}

	// Web Application Demo
	templates[DemoTypeWebApp] = DemoCommission{
		Type:        DemoTypeWebApp,
		Title:       "Modern Dashboard Web Application",
		Description: "Create a responsive web dashboard with real-time features and beautiful UI",
		Tags:        []string{"webapp", "frontend", "fullstack", "demo"},
		Content: `# 🎨 Modern Analytics Dashboard

> **Project Objective**: Build a beautiful, responsive analytics dashboard that demonstrates Guild's capability to coordinate frontend, backend, and design agents for creating modern web applications.

## 🎯 Project Overview

Create a real-time analytics dashboard that visualizes business metrics, user activity, and system performance. This project showcases modern web development with React, TypeScript, and a Node.js backend, perfect for demonstrating multi-agent collaboration across the full stack.

## 🛠 Technical Stack

### Frontend
- **Framework**: React 18 with TypeScript
- **Styling**: Tailwind CSS + Shadcn/ui components
- **State Management**: Zustand for global state
- **Data Fetching**: TanStack Query (React Query)
- **Charts**: Recharts for data visualization
- **Build Tool**: Vite for fast development

### Backend
- **Runtime**: Node.js with Express
- **Language**: TypeScript
- **Database**: PostgreSQL + Redis
- **Real-time**: WebSockets with Socket.io
- **Authentication**: JWT with secure cookies
- **API**: RESTful + GraphQL endpoints

## 🎨 Feature Requirements

### Dashboard Features
1. **Real-time Metrics**
   - Live user count and activity
   - Revenue and conversion tracking
   - System performance indicators
   - Customizable widget layout

2. **Data Visualization**
   - Line charts for trends
   - Bar charts for comparisons
   - Pie charts for distributions
   - Heat maps for user activity

3. **User Management**
   - User list with search/filter
   - Role-based permissions
   - Activity timeline
   - Bulk operations

4. **Responsive Design**
   - Mobile-first approach
   - Tablet optimization
   - Desktop power features
   - Dark/light theme toggle

## 📋 Implementation Phases

### Phase 1: Foundation (Architect & Designer Agents)
- [ ] Design component architecture and data flow
- [ ] Create responsive layout system
- [ ] Set up TypeScript configurations
- [ ] Design consistent color scheme and typography
- [ ] Plan real-time data architecture

### Phase 2: Backend Development (Backend Agents)
- [ ] Build authentication system with JWT
- [ ] Create RESTful API endpoints
- [ ] Implement WebSocket connections
- [ ] Set up database models and migrations
- [ ] Add data aggregation services

### Phase 3: Frontend Implementation (Frontend Agents)
- [ ] Build reusable component library
- [ ] Implement dashboard layout with widgets
- [ ] Create interactive charts and graphs
- [ ] Add real-time data updates
- [ ] Implement theme switching

### Phase 4: Integration & Polish (Full-stack Agents)
- [ ] Connect frontend to backend APIs
- [ ] Implement error boundaries and fallbacks
- [ ] Add loading states and skeletons
- [ ] Optimize bundle size and performance
- [ ] Create smooth animations and transitions

### Phase 5: Testing & Documentation (QA Agents)
- [ ] Write component unit tests
- [ ] Create E2E tests with Cypress
- [ ] Test responsive design across devices
- [ ] Document component props and usage
- [ ] Create user guide with screenshots

## 🎯 Success Criteria

### Performance
- ⚡ First Contentful Paint < 1.5s
- ⚡ Time to Interactive < 3s
- ⚡ Lighthouse score > 90
- ⚡ Smooth 60fps animations

### User Experience
- 🎨 Consistent design language
- 📱 Fully responsive on all devices
- ♿ WCAG 2.1 AA accessibility
- 🌐 Internationalization ready

### Code Quality
- 📏 TypeScript strict mode enabled
- 🧪 >80% test coverage
- 📚 Comprehensive documentation
- 🔧 ESLint/Prettier configured

## 🚀 Demonstration Impact

This project showcases Guild's ability to:
- **Coordinate** frontend and backend specialists
- **Design** beautiful, functional interfaces
- **Implement** modern web technologies
- **Optimize** for performance and UX
- **Test** across different browsers and devices

> 💡 **Why This Demo?** Dashboards are complex applications that require coordination between design, frontend, and backend. This demonstrates Guild's multi-agent orchestration capabilities perfectly.`,
	}

	// CLI Tool Demo
	templates[DemoTypeCLITool] = DemoCommission{
		Type:        DemoTypeCLITool,
		Title:       "Developer Productivity CLI Tool",
		Description: "Build a powerful CLI tool that enhances developer workflow and productivity",
		Tags:        []string{"cli", "devtools", "automation", "demo"},
		Content: `# 🛠️ Developer Productivity CLI Tool

> **Project Objective**: Create a comprehensive CLI tool that automates common development tasks, demonstrating Guild's ability to build sophisticated command-line applications with multiple specialized agents.

## 🎯 Project Overview

Build a powerful CLI tool called "DevFlow" that streamlines developer workflows by automating project setup, code generation, testing, and deployment tasks. This showcases how Guild agents can collaborate to create developer-friendly tools with excellent UX.

## 🛠 Technical Specifications

### Core Technologies
- **Language**: Go (for performance and single binary)
- **CLI Framework**: Cobra for commands, Viper for config
- **UI Components**: Bubble Tea for interactive TUI
- **Testing**: Testify for assertions
- **Distribution**: Goreleaser for multi-platform builds

### Key Features

1. **Project Scaffolding**
   - Initialize projects from templates
   - Support multiple languages/frameworks
   - Interactive configuration wizard
   - Git repository initialization

2. **Code Generation**
   - Generate boilerplate code
   - Create API clients from OpenAPI
   - Database models from schema
   - Test scaffolding

3. **Development Tools**
   - Run multiple services with live reload
   - Environment variable management
   - Database migrations
   - Container orchestration

4. **Workflow Automation**
   - Custom task definitions
   - Pipeline execution
   - Conditional workflows
   - Parallel task running

## 📋 Implementation Roadmap

### Phase 1: Core Architecture (Architect Agents)
- [ ] Design plugin architecture for extensibility
- [ ] Create command structure and routing
- [ ] Implement configuration management
- [ ] Set up testing framework
- [ ] Design error handling strategy

### Phase 2: Base Commands (Developer Agents)
- [ ] Implement 'init' command for project setup
- [ ] Create 'generate' command framework
- [ ] Build 'run' command for service management
- [ ] Add 'config' command for settings
- [ ] Implement 'plugin' management system

### Phase 3: Interactive Features (UX Agents)
- [ ] Create beautiful TUI components
- [ ] Add interactive project wizard
- [ ] Implement progress indicators
- [ ] Build selection menus
- [ ] Add syntax highlighting for output

### Phase 4: Advanced Features (Specialist Agents)
- [ ] Add template engine integration
- [ ] Implement hot-reload functionality
- [ ] Create workflow DSL parser
- [ ] Add cloud service integrations
- [ ] Build update notification system

### Phase 5: Polish & Distribution (DevOps Agents)
- [ ] Create comprehensive help system
- [ ] Add shell completion scripts
- [ ] Build installation scripts
- [ ] Package for multiple platforms
- [ ] Create homebrew formula

## 🎨 User Experience Goals

### Command Design
` + "```bash" + `
# Intuitive command structure
devflow init --template react-typescript
devflow generate component --name UserProfile
devflow run --services api,frontend --watch
devflow deploy staging --dry-run
` + "```" + `

### Interactive Mode
- 🎯 Guided wizards for complex operations
- 🎨 Colorful, informative output
- ⚡ Real-time feedback and progress
- 🔍 Helpful error messages with solutions

## 🧪 Testing Strategy

1. **Unit Tests**: Core logic and utilities
2. **Integration Tests**: Command execution flows
3. **E2E Tests**: Full workflow scenarios
4. **Performance Tests**: Large project handling
5. **Compatibility Tests**: Cross-platform verification

## 📊 Success Metrics

- ⚡ Commands execute in < 100ms
- 🎯 Zero crashes in normal usage
- 📚 100% of commands documented
- 🧪 >90% test coverage
- 🌍 Works on Mac, Linux, Windows

## 🚀 Demonstration Value

This project showcases Guild's ability to:
- **Design** developer-friendly interfaces
- **Implement** complex CLI architectures
- **Create** powerful automation tools
- **Test** thoroughly across platforms
- **Document** for developer adoption

> 💡 **Why This Demo?** CLI tools are essential for developer productivity. This demonstrates Guild's ability to create sophisticated tools that developers love to use.`,
	}

	// Data Analysis Demo
	templates[DemoTypeDataAnalysis] = DemoCommission{
		Type:        DemoTypeDataAnalysis,
		Title:       "Data Pipeline & Analytics Platform",
		Description: "Build a data processing pipeline with analytics and visualization",
		Tags:        []string{"data", "analytics", "etl", "visualization", "demo"},
		Content: `# 📊 Data Pipeline & Analytics Platform

> **Project Objective**: Create a comprehensive data analytics platform that demonstrates Guild's ability to coordinate data engineers, analysts, and ML specialists for building modern data solutions.

## 🎯 Project Overview

Build an end-to-end data platform that ingests, processes, analyzes, and visualizes e-commerce data. This project showcases ETL pipelines, real-time analytics, machine learning, and interactive dashboards - perfect for demonstrating multi-agent expertise in data engineering.

## 🛠 Technical Architecture

### Data Stack
- **Orchestration**: Apache Airflow for workflow management
- **Processing**: Apache Spark for large-scale processing
- **Streaming**: Kafka for real-time data ingestion
- **Storage**: PostgreSQL for structured, S3 for raw data
- **Analytics**: Pandas, NumPy for data analysis
- **ML Framework**: Scikit-learn for predictive models
- **Visualization**: Plotly Dash for interactive dashboards

### Infrastructure
- **Containerization**: Docker for all services
- **Language**: Python for data processing
- **API**: FastAPI for serving insights
- **Monitoring**: Prometheus + Grafana

## 📈 Analytics Features

### Data Sources
1. **E-commerce Events**
   - User clickstream data
   - Purchase transactions
   - Product interactions
   - Search queries

2. **Business Metrics**
   - Revenue analytics
   - Customer segmentation
   - Product performance
   - Conversion funnels

3. **Predictive Analytics**
   - Sales forecasting
   - Churn prediction
   - Recommendation engine
   - Anomaly detection

## 📋 Implementation Phases

### Phase 1: Data Infrastructure (Data Engineers)
- [ ] Design data warehouse schema
- [ ] Set up Kafka for event streaming
- [ ] Configure Airflow DAGs
- [ ] Implement data quality checks
- [ ] Create data lake structure

### Phase 2: ETL Pipelines (Pipeline Engineers)
- [ ] Build ingestion pipelines
- [ ] Implement data transformations
- [ ] Create aggregation jobs
- [ ] Set up incremental updates
- [ ] Add error handling and retries

### Phase 3: Analytics Layer (Data Analysts)
- [ ] Create business metric calculations
- [ ] Build customer segmentation
- [ ] Implement cohort analysis
- [ ] Design A/B test framework
- [ ] Create reporting tables

### Phase 4: Machine Learning (ML Engineers)
- [ ] Build recommendation system
- [ ] Implement churn prediction model
- [ ] Create demand forecasting
- [ ] Add anomaly detection
- [ ] Set up model monitoring

### Phase 5: Visualization (Frontend Engineers)
- [ ] Create interactive dashboards
- [ ] Build real-time metrics display
- [ ] Add drill-down capabilities
- [ ] Implement export features
- [ ] Create automated reports

## 🎯 Data Processing Goals

### Performance Targets
- 📊 Process 1M events per minute
- ⚡ Dashboard refresh < 5 seconds
- 🎯 ML predictions < 100ms
- 💾 Data freshness < 5 minutes

### Data Quality
- ✅ 99.9% data completeness
- 🔍 Automated anomaly detection
- 📏 Standardized metrics
- 🛡️ PII data protection

## 🧪 Validation Strategy

1. **Data Validation**: Schema enforcement, null checks
2. **Pipeline Testing**: Unit tests for transformations
3. **Model Validation**: Cross-validation, A/B tests
4. **Performance Testing**: Load testing pipelines
5. **Dashboard Testing**: User acceptance testing

## 📊 Deliverables

### Dashboards
- 📈 Executive KPI Dashboard
- 👥 Customer Analytics View
- 📦 Product Performance Metrics
- 💰 Revenue Analytics
- 🔮 Predictive Insights

### APIs
- 🚀 Real-time metrics endpoint
- 📊 Historical data API
- 🤖 ML prediction service
- 📑 Report generation API

## 🚀 Demonstration Impact

This project showcases Guild's ability to:
- **Orchestrate** complex data workflows
- **Process** large-scale data efficiently
- **Analyze** business metrics comprehensively
- **Predict** future trends with ML
- **Visualize** insights beautifully

> 💡 **Why This Demo?** Data is the foundation of modern business decisions. This demonstrates Guild's capability to handle complex data engineering tasks with multiple specialized agents.`,
	}

	// Microservices Demo
	templates[DemoTypeMicroservices] = DemoCommission{
		Type:        DemoTypeMicroservices,
		Title:       "Cloud-Native E-commerce Platform",
		Description: "Build a production-ready microservices architecture for modern e-commerce",
		Tags:        []string{"microservices", "cloud-native", "distributed", "demo"},
		Content: `# 🛒 Modern Microservices E-commerce Platform

> **Project Objective**: Build a production-ready e-commerce platform using microservices architecture that demonstrates modern cloud-native patterns, API design, and distributed systems principles.

## 🎯 Project Overview

This comprehensive demo project showcases real-world software architecture while remaining accessible to developers across different experience levels. The platform will demonstrate the complexity and coordination required for modern distributed systems through a familiar e-commerce domain.

## 🛠 Technical Stack

### Backend Services
- **Language**: Go for high-performance services
- **Framework**: Gin for HTTP, gRPC for inter-service
- **Database**: PostgreSQL (primary), Redis (cache)
- **Message Queue**: NATS for event streaming
- **API Gateway**: Kong for routing and auth

### Infrastructure
- **Containers**: Docker with multi-stage builds
- **Orchestration**: Kubernetes for production
- **Service Mesh**: Istio for observability
- **Monitoring**: Prometheus + Grafana
- **Tracing**: Jaeger for distributed tracing

## 🏗 Service Architecture

### Core Services

1. **🔐 Authentication Service**
   - JWT token management
   - OAuth2 integration
   - Role-based access control
   - Session management

2. **📦 Product Catalog Service**
   - Product CRUD operations
   - Inventory management
   - Search with Elasticsearch
   - Category hierarchies

3. **🛒 Shopping Cart Service**
   - Session-based carts
   - Real-time inventory checks
   - Price calculations
   - Persistent storage

4. **💳 Payment Service**
   - Multiple payment gateways
   - Transaction processing
   - Refund handling
   - PCI compliance

5. **📋 Order Management Service**
   - Order lifecycle management
   - Status tracking
   - Invoice generation
   - Shipping integration

6. **📧 Notification Service**
   - Email notifications
   - SMS alerts
   - Push notifications
   - Template management

## 📋 Implementation Strategy

### Phase 1: Foundation (Week 1)
- [ ] Set up development environment
- [ ] Create service templates
- [ ] Implement authentication service
- [ ] Set up API gateway
- [ ] Configure service discovery

### Phase 2: Core Services (Week 2)
- [ ] Build product catalog service
- [ ] Implement shopping cart
- [ ] Create order management
- [ ] Set up inter-service communication
- [ ] Add event streaming

### Phase 3: Advanced Features (Week 3)
- [ ] Integrate payment processing
- [ ] Add notification system
- [ ] Implement search functionality
- [ ] Create recommendation engine
- [ ] Add inventory tracking

### Phase 4: Production Ready (Week 4)
- [ ] Implement circuit breakers
- [ ] Add distributed tracing
- [ ] Set up monitoring dashboards
- [ ] Create health checks
- [ ] Load testing and optimization

## 🎯 Architectural Patterns

### Design Patterns
- 🏗️ **CQRS**: Separate read/write models
- 📡 **Event Sourcing**: Audit trail for orders
- 🔄 **Saga Pattern**: Distributed transactions
- 💔 **Circuit Breaker**: Fault tolerance
- 🎭 **API Gateway**: Single entry point

### Communication Patterns
- 🔄 **Synchronous**: REST for client APIs
- 📨 **Asynchronous**: Events for updates
- 🚀 **gRPC**: High-performance internal
- 📡 **WebSockets**: Real-time updates

## 🧪 Testing Strategy

### Testing Levels
1. **Unit Tests**: Service logic (>80% coverage)
2. **Integration Tests**: Database and API tests
3. **Contract Tests**: Service interactions
4. **E2E Tests**: User journey validation
5. **Chaos Tests**: Resilience testing

### Performance Goals
- ⚡ API response time < 200ms (p95)
- 🚀 Handle 10K requests/second
- 📊 99.9% uptime SLA
- 🔄 Zero-downtime deployments

## 📊 Observability

### Monitoring Stack
- 📈 **Metrics**: Service performance, business KPIs
- 📝 **Logging**: Centralized with correlation IDs
- 🔍 **Tracing**: Request flow visualization
- 🚨 **Alerting**: Proactive issue detection

## 🚀 Demonstration Value

This project showcases Guild's ability to:
- **Architect** distributed systems properly
- **Implement** microservices best practices
- **Coordinate** multiple service teams
- **Handle** complex integrations
- **Ensure** production readiness

> 💡 **Why This Demo?** Microservices architecture is the standard for modern applications. This demonstrates Guild's capability to handle complex, distributed systems with multiple specialized agents working in harmony.`,
	}

	// AI/ML Demo
	templates[DemoTypeAI] = DemoCommission{
		Type:        DemoTypeAI,
		Title:       "Intelligent Content Recommendation System",
		Description: "Build an AI-powered recommendation engine with real-time learning",
		Tags:        []string{"ai", "ml", "recommendation", "nlp", "demo"},
		Content: `# 🤖 Intelligent Content Recommendation System

> **Project Objective**: Create a sophisticated AI-powered recommendation system that demonstrates Guild's ability to coordinate ML engineers, data scientists, and backend developers for building production-ready AI applications.

## 🎯 Project Overview

Build a real-time content recommendation engine that learns from user behavior, understands content semantics, and delivers personalized experiences. This project showcases modern ML practices, from data pipeline to model serving, perfect for demonstrating multi-agent AI expertise.

## 🛠 Technical Architecture

### ML Stack
- **Framework**: PyTorch for deep learning models
- **NLP**: Transformers for content understanding
- **Serving**: TorchServe for model deployment
- **Feature Store**: Feast for feature management
- **Experiment Tracking**: MLflow
- **Vector Database**: Pinecone for embeddings

### Backend Infrastructure
- **API**: FastAPI for high-performance serving
- **Queue**: Celery for async processing
- **Cache**: Redis for feature caching
- **Database**: PostgreSQL for user data
- **Monitoring**: Weights & Biases

## 🧠 AI Components

### Recommendation Models

1. **Content-Based Filtering**
   - Text embeddings with BERT
   - Image features with ResNet
   - Multi-modal fusion
   - Semantic similarity search

2. **Collaborative Filtering**
   - Matrix factorization
   - Deep learning approach
   - User behavior patterns
   - Real-time updates

3. **Hybrid Approach**
   - Ensemble methods
   - Contextual bandits
   - Reinforcement learning
   - A/B testing framework

### NLP Pipeline
- 📝 Content analysis and tagging
- 🔍 Named entity recognition
- 💭 Sentiment analysis
- 🌐 Multi-language support

## 📋 Implementation Roadmap

### Phase 1: Data Foundation (Data Engineers)
- [ ] Design feature engineering pipeline
- [ ] Set up data collection infrastructure
- [ ] Implement user behavior tracking
- [ ] Create training data pipelines
- [ ] Build feature store

### Phase 2: Model Development (ML Engineers)
- [ ] Implement content embedding models
- [ ] Build collaborative filtering system
- [ ] Create hybrid recommendation model
- [ ] Set up experiment tracking
- [ ] Optimize model performance

### Phase 3: Serving Infrastructure (Backend Engineers)
- [ ] Build prediction API service
- [ ] Implement model versioning
- [ ] Create feature serving layer
- [ ] Add caching strategies
- [ ] Set up A/B testing

### Phase 4: Real-time Learning (ML Ops Engineers)
- [ ] Implement online learning
- [ ] Build feedback loops
- [ ] Create model monitoring
- [ ] Set up automated retraining
- [ ] Add drift detection

### Phase 5: Production Features (Full-stack Engineers)
- [ ] Create explanation UI
- [ ] Build admin dashboard
- [ ] Add performance metrics
- [ ] Implement fallback strategies
- [ ] Create recommendation API

## 🎯 Performance Requirements

### Model Metrics
- 🎯 Precision@10 > 0.8
- 📈 Recall@50 > 0.6
- ⚡ Inference time < 50ms
- 🔄 Model update < 1 hour

### System Performance
- 🚀 10K predictions/second
- 💾 Feature cache hit > 95%
- 🔄 Real-time model updates
- 📊 99.9% API availability

## 🧪 Evaluation Strategy

### Offline Evaluation
1. **Accuracy Metrics**: Precision, recall, F1
2. **Ranking Metrics**: NDCG, MAP
3. **Diversity Metrics**: Coverage, novelty
4. **Bias Testing**: Fairness evaluation

### Online Evaluation
1. **A/B Testing**: Conversion rates
2. **User Engagement**: CTR, dwell time
3. **Business Metrics**: Revenue impact
4. **User Satisfaction**: Feedback scores

## 📊 Deliverables

### Models
- 🧠 Content understanding model
- 🤝 Collaborative filtering model
- 🎯 Hybrid recommendation engine
- 📈 Ranking optimization model

### APIs
- 🚀 Real-time prediction endpoint
- 📊 Batch recommendation API
- 🔍 Similar content search
- 📈 Analytics dashboard API

### Tools
- 🛠️ Model management UI
- 📊 Performance dashboard
- 🧪 A/B testing framework
- 📈 Business metrics tracker

## 🚀 Demonstration Impact

This project showcases Guild's ability to:
- **Design** complex ML systems
- **Implement** state-of-the-art models
- **Deploy** production-ready AI
- **Monitor** model performance
- **Iterate** based on user feedback

> 💡 **Why This Demo?** AI/ML systems require deep expertise across multiple domains. This demonstrates Guild's ability to coordinate specialists to build sophisticated, production-ready AI applications.`,
	}

	// Default fallback demo
	templates[DemoTypeDefault] = DemoCommission{
		Type:        DemoTypeDefault,
		Title:       "Simple API Development Task",
		Description: "A straightforward REST API project to demonstrate Guild's capabilities",
		Tags:        []string{"api", "starter", "demo"},
		Content: `# Simple API Development Task

## Objective
Create a basic REST API with essential endpoints to demonstrate Guild's code generation and testing capabilities.

## Requirements

### Core API Features
- Create a simple Go HTTP server
- Implement basic CRUD operations for a "tasks" resource
- Add proper error handling and HTTP status codes
- Include basic logging

### Technical Specifications
- Use Go's standard library (net/http)
- Implement JSON request/response handling
- Add input validation
- Follow REST conventions

### Endpoints Required
1. GET /tasks - List all tasks
2. POST /tasks - Create a new task  
3. GET /tasks/{id} - Get specific task
4. PUT /tasks/{id} - Update task
5. DELETE /tasks/{id} - Delete task

### Testing Requirements
- Write unit tests for each endpoint
- Include integration tests
- Test error scenarios
- Achieve >80% test coverage

## Success Criteria
- All endpoints respond correctly
- Tests pass and have good coverage
- Code follows Go best practices
- API is well-documented

## Notes
This is a demo commission designed to showcase Guild's multi-agent development workflow. The Manager will break this down into smaller tasks and assign them to appropriate specialized agents.`,
	}

	return templates
}

// GetRecommendedDemo analyzes the project and recommends a demo type
func (g *DemoCommissionGenerator) GetRecommendedDemo(ctx context.Context, projectInfo map[string]interface{}) (DemoCommissionType, string) {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return DemoTypeDefault, "Using default demo due to cancellation"
	}

	// Simple heuristics for demo recommendation
	if projectName, ok := projectInfo["project_name"].(string); ok {
		projectNameLower := strings.ToLower(projectName)

		// Check project name for hints
		switch {
		case strings.Contains(projectNameLower, "api") || strings.Contains(projectNameLower, "service"):
			return DemoTypeAPIService, "Project name suggests an API service"
		case strings.Contains(projectNameLower, "web") || strings.Contains(projectNameLower, "app"):
			return DemoTypeWebApp, "Project name suggests a web application"
		case strings.Contains(projectNameLower, "cli") || strings.Contains(projectNameLower, "tool"):
			return DemoTypeCLITool, "Project name suggests a CLI tool"
		case strings.Contains(projectNameLower, "data") || strings.Contains(projectNameLower, "analytics"):
			return DemoTypeDataAnalysis, "Project name suggests data analysis"
		case strings.Contains(projectNameLower, "ai") || strings.Contains(projectNameLower, "ml"):
			return DemoTypeAI, "Project name suggests AI/ML project"
		}
	}

	// Check for technology hints
	if tech, ok := projectInfo["detected_tech"].([]string); ok {
		for _, t := range tech {
			switch strings.ToLower(t) {
			case "react", "vue", "angular":
				return DemoTypeWebApp, fmt.Sprintf("Detected %s framework", t)
			case "tensorflow", "pytorch", "scikit-learn":
				return DemoTypeAI, fmt.Sprintf("Detected %s ML framework", t)
			case "pandas", "numpy", "jupyter":
				return DemoTypeDataAnalysis, fmt.Sprintf("Detected %s data tool", t)
			}
		}
	}

	return DemoTypeDefault, "No specific project type detected, using default"
}
