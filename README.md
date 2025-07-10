# SaaS Sync Platform

> **Modern cloud-based synchronization platform for Sage 200c integrations**

A scalable SaaS solution that replaces traditional desktop applications and Windows services with a centralized web platform for managing data synchronization between Sage 200c and external services like Bitrix24, Tickelia, and more.

## 🎯 Purpose

### **The Problem We're Solving**
Our clients using Sage 200c previously required:
- Individual desktop application installations
- Windows service deployments on each machine  
- Manual configuration file management
- Limited scalability and monitoring capabilities
- Complex maintenance and updates

### **Our SaaS Solution**
We're building a centralized platform where:
- Clients are registered through a web interface
- Synchronization happens in the cloud
- Real-time monitoring and management
- Automatic updates and maintenance
- Scalable architecture supporting unlimited clients

## 🏗️ Architecture Overview

```
                    ┌─────────────────┐
                    │   React Web     │
                    │   Dashboard     │
                    │   (Cloud)       │
                    └─────────┬───────┘
                              │
                    ┌─────────▼───────┐
                    │   Go Backend    │
                    │   API Server    │
                    │   (Cloud)       │
                    └─────────┬───────┘
                              │
                    ┌─────────▼───────┐
                    │   PostgreSQL    │
                    │   Database      │
                    │   (Cloud)       │
                    └─────────────────┘
                              │
                              │ HTTPS/WSS
                              │
              ┌───────────────▼───────────────┐
              │                               │
              ▼                               ▼
    ┌─────────────────┐             ┌─────────────────┐
    │   Sync Agent    │             │   Sync Agent    │
    │   (Client A)    │             │   (Client B)    │
    │                 │             │                 │
    │ ┌─────────────┐ │             │ ┌─────────────┐ │
    │ │ Sage 200c   │ │             │ │ Sage 200c   │ │
    │ │ SQL Server  │ │             │ │ SQL Server  │ │
    │ └─────────────┘ │             │ └─────────────┘ │
    └─────────────────┘             └─────────────────┘
              │                               │
              ▼                               ▼
    ┌─────────────────┐             ┌─────────────────┐
    │   Bitrix24      │             │   Tickelia      │
    │   Integration   │             │   Integration   │
    └─────────────────┘             └─────────────────┘
```

## 🛠️ Technology Stack

- **Backend**: Go (Golang) with Gin framework
- **Frontend**: React.js with modern hooks
- **Database**: PostgreSQL
- **Containerization**: Docker & Docker Compose
- **CI/CD**: GitHub Actions
- **Deployment**: Ubuntu Server
- **Authentication**: JWT tokens
- **External Integrations**: REST APIs (Bitrix24, Tickelia)

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- Node.js 18+
- Docker & Docker Compose
- PostgreSQL (or use Docker)
- Git

### Development Setup

1. **Clone the repository**
   ```bash
   git clone https://github.com/your-org/saas-sync-platform.git
   cd saas-sync-platform
   ```

2. **Set up environment variables**
   ```bash
   cp .env.example .env
   # Edit .env with your local configuration
   ```

3. **Start the development environment**
   ```bash
   # Start PostgreSQL and other services
   docker-compose up -d postgres redis
   
   # Install Go dependencies
   go mod download
   
   # Install Node.js dependencies
   cd web && npm install && cd ..
   
   # Run database migrations
   go run cmd/migrator/main.go
   
   # Start the backend API
   go run cmd/api/main.go
   
   # In another terminal, start the frontend
   cd web && npm start
   ```

4. **Access the application**
   - Frontend: http://localhost:3000
   - API: http://localhost:8080
   - API Documentation: http://localhost:8080/docs

## 🌿 Branching Strategy

We follow a **Git Flow** approach with these protected branches:

### **Branch Structure**
- `main` - **Production-ready code** (Protected)
- `test` - **Testing/staging environment** (Protected)  
- `develop` - **Integration branch** for features
- `feature/*` - **Individual features** (e.g., `feature/bitrix24-integration`)
- `hotfix/*` - **Emergency production fixes**

### **Branch Protection Rules**
- ✅ `main` and `test` branches are **protected**
- ✅ Require **pull request reviews** before merging
- ✅ Require **status checks** to pass (tests, linting)
- ✅ Require **up-to-date branches** before merging
- ❌ **Direct pushes to main/test are forbidden**

### **Deployment Flow**
```
feature/branch → develop → test → main
     ↓              ↓        ↓      ↓
   Local        Integration Test  Production
   Testing       Server    Server   Server
```

## 🤝 Contributing

We welcome contributions from all team members! Please follow these guidelines:

### **Development Workflow**

1. **Create an Issue First**
   - Use our [issue templates](.github/ISSUE_TEMPLATE/)
   - Describe the feature/bug clearly
   - Add appropriate labels
   - Assign to yourself if you plan to work on it

2. **Create a Feature Branch**
   ```bash
   git checkout develop
   git pull origin develop
   git checkout -b feature/your-feature-name
   ```

3. **Make Your Changes**
   - Write clean, well-documented code
   - Follow our [coding standards](#coding-standards)
   - Add tests for new functionality
   - Update documentation if needed

4. **Commit Your Changes**
   ```bash
   # Use conventional commit messages
   git commit -m "feat: add Bitrix24 contact synchronization
   
   - Implement API client for Bitrix24
   - Add contact mapping logic
   - Include error handling and retries
   
   Closes #123"
   ```

5. **Push and Create Pull Request**
   ```bash
   git push origin feature/your-feature-name
   ```
   - Use our [PR template](.github/PULL_REQUEST_TEMPLATE.md)
   - Link related issues
   - Request review from team members

### **Pull Request Guidelines**

#### **Requirements for PR Approval**
- ✅ All tests must pass
- ✅ Code coverage should not decrease
- ✅ At least **2 reviewers** must approve
- ✅ No merge conflicts with target branch
- ✅ Documentation updated (if applicable)

#### **PR Review Process**
1. **Automatic Checks**: GitHub Actions will run tests, linting, and security scans
2. **Code Review**: Team members review for quality, security, and best practices  
3. **Testing**: Reviewer tests the feature locally or in test environment
4. **Approval**: Once approved, a maintainer will merge the PR

#### **Who Can Merge PRs**
Only these team members can merge to protected branches:
- **Tech Lead/Senior Developer**
- **Project Maintainer**
- **Designated Code Reviewers**

### **Issue Management**

#### **Creating Issues**
Use these labels to categorize issues:
- `enhancement` - New features
- `bug` - Bug reports  
- `documentation` - Documentation improvements
- `integration` - External service integrations
- `performance` - Performance improvements
- `security` - Security-related issues

#### **Issue Templates**
We provide templates for:
- 🐛 **Bug Report** - Report bugs with steps to reproduce
- ✨ **Feature Request** - Propose new features  
- 📚 **Documentation** - Documentation improvements
- 🔗 **Integration** - New service integrations

## 📋 Coding Standards

### **Go Backend Standards**
- Follow **Go best practices** and `gofmt` styling
- Use **dependency injection** for services
- Write **comprehensive tests** (aim for >80% coverage)
- Handle **errors explicitly** - no silent failures
- Use **structured logging** with appropriate levels
- Document **public functions** with Go comments

### **React Frontend Standards**  
- Use **functional components** with hooks
- Follow **ESLint** and **Prettier** configurations
- Write **component tests** with React Testing Library
- Use **TypeScript** for type safety
- Implement **responsive design** principles
- Follow **accessibility** guidelines (WCAG 2.1)

### **Database Standards**
- Use **migrations** for all schema changes
- Write **reversible migrations** when possible
- Include **proper indexing** for performance
- Use **transactions** for data consistency
- Document **complex queries** with comments

## 🧪 Testing Strategy

### **Test Types**
- **Unit Tests**: Individual functions and components
- **Integration Tests**: API endpoints and database interactions  
- **End-to-End Tests**: Complete user workflows
- **Performance Tests**: Load testing for sync operations

### **Running Tests**
```bash
# Backend tests
go test ./...

# Frontend tests  
cd web && npm test

# Integration tests
docker-compose -f docker-compose.test.yml up --abort-on-container-exit

# Performance tests
go test -bench=. ./internal/sync/...
```

### **Test Coverage Requirements**
- **Backend**: Minimum 80% coverage
- **Frontend**: Minimum 70% coverage
- **Critical paths**: 95% coverage (authentication, sync logic)

## 🚀 Deployment

### **Environment Setup**

#### **Test Server** (test.yourdomain.com)
- Deploys automatically from `test` branch
- Used for staging and QA testing
- Mirrors production configuration

#### **Production Server** (app.yourdomain.com)  
- Deploys from `main` branch after manual approval
- Zero-downtime deployments with Docker
- Automated backups and monitoring

### **Deployment Process**
1. Merge to `test` → Auto-deploy to test server
2. QA testing on test environment
3. Create PR from `test` → `main` 
4. Code review and approval
5. Manual deployment trigger to production

## 📊 Monitoring & Logging

### **Application Monitoring**
- **Health checks**: `/health` endpoint
- **Metrics**: Prometheus + Grafana
- **Logs**: Structured JSON logs with log levels
- **Alerts**: Slack notifications for critical issues

### **Sync Monitoring**
- Real-time sync status per client
- Error tracking and retry mechanisms  
- Performance metrics (sync duration, throughput)
- Client-specific dashboards

## 🔒 Security Guidelines

### **API Security**
- **JWT authentication** for all endpoints
- **Rate limiting** to prevent abuse
- **Input validation** and sanitization
- **HTTPS only** in production

### **Database Security**
- **Encrypted connections** (SSL/TLS)
- **Environment variables** for secrets
- **Principle of least privilege** for database users
- **Regular security updates**

### **Code Security**
- **Dependency scanning** with GitHub Security Advisories
- **Static analysis** with CodeQL
- **Secret scanning** to prevent credential commits
- **Regular security reviews**

## 📞 Support & Communication

### **Documentation**
- **API Documentation**: Automatically generated from code
- **Architecture Decisions**: Documented in `/docs/adr/`
- **Runbooks**: Operational procedures in `/docs/ops/`

### **Getting Help**
- **Technical Questions**: Ask in Teams or create a discussion
- **Bug Reports**: Create an issue with the bug template
- **Feature Ideas**: Create an issue with the feature template

## 📈 Project Roadmap

### **Phase 1: Foundation**
- [ ] Project setup and CI/CD
- [ ] Authentication system
- [ ] Basic client management
- [ ] Sage 200c integration module

### **Phase 2: Core Integrations**
- [ ] Bitrix24 integration
- [ ] Tickelia integration
- [ ] Sync engine and scheduling
- [ ] Error handling and retries

### **Phase 3: Advanced Features**
- [ ] Real-time monitoring dashboard
- [ ] Advanced sync configurations
- [ ] Multi-company support
- [ ] Performance optimizations

### **Phase 4: Production**
- [ ] Load testing and optimization
- [ ] Security audit
- [ ] Production deployment
- [ ] Client migration tools

## 📄 License

This project is proprietary software owned by **Business Tic Consultoria**.
All rights reserved. Unauthorized copying or distribution is prohibited.

## 👥 Team

- **Project Lead**: Arnau Forcada & Jordi Ardura
- **Backend Developer**: Jordi Ardura
- **Frontend Developer**: Rafa Bermúdez
- **DevOps Engineer**: Jordi Ardura

---

**Questions?** Reach out in our Teams server or create a GitHub discussion!

🚀 **Happy coding!**