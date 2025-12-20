# Spec 0109: Deployment Guides

## Overview

Create comprehensive deployment documentation for Mizu applications, covering all major deployment scenarios from Docker containers to Kubernetes orchestration to serverless platforms.

## Reorganized Guides Structure

### Getting Started (introduction + setup)
- `guides/welcome` (icon: house) - Introduction and quick example
- `overview/why` (icon: heart) - Design philosophy (MOVED UP)
- `guides/installation` (icon: download) - Setup instructions
- `guides/quick-start` (icon: rocket) - First app in 5 minutes

### Core Concepts (reordered for natural learning)
- `guides/concepts/overview` (icon: book-open) - Concepts introduction
- `guides/concepts/app` (icon: server) - App lifecycle and configuration
- `guides/concepts/routing` (icon: route) - URL patterns and groups
- `guides/concepts/handler` (icon: code) - Handler signature and patterns
- `guides/concepts/context` (icon: box) - Ctx wrapper and access
- `guides/concepts/request` (icon: arrow-down-to-line) - Reading request data
- `guides/concepts/response` (icon: arrow-up-from-line) - Writing responses
- `guides/concepts/middleware` (icon: layer-group) - Middleware patterns
- `guides/concepts/error` (icon: bug) - Error handling
- `guides/concepts/logging` (icon: scroll) - Structured logging
- `guides/concepts/static` (icon: folder-open) - Static file serving

### Building Your App (tutorials)
- `guides/first-api` (icon: code) - Build a REST API
- `guides/first-website` (icon: globe) - Build a website
- `guides/first-fullstack` (icon: layer-group) - Full-stack app
- `guides/project-structure` (icon: sitemap) - Code organization
- `guides/testing` (icon: vial) - Testing patterns

### Deployment (NEW comprehensive section)
- `guides/deployment/overview` (icon: cloud) - Deployment strategies
- `guides/deployment/docker` (icon: docker) - Docker containerization
- `guides/deployment/kubernetes` (icon: dharmachakra) - Kubernetes orchestration
- `guides/deployment/cloud` (icon: cloud-arrow-up) - Cloud providers (AWS, GCP, DO)
- `guides/deployment/serverless` (icon: bolt) - Serverless (Lambda, Cloud Run)
- `guides/deployment/traditional` (icon: server) - Traditional servers (systemd, nginx)
- `guides/deployment/cicd` (icon: arrows-rotate) - CI/CD pipelines

### Learn More
- `overview/features` (icon: sparkles) - Feature summary
- `overview/use-cases` (icon: lightbulb) - Use cases
- `guides/architecture` (icon: diagram-project) - Architecture overview
- `overview/roadmap` (icon: road) - Future plans
- `guides/faq` (icon: circle-question) - FAQ
- `guides/community` (icon: users) - Community

## Deployment Pages Content

### 1. deployment/overview.mdx
**Purpose**: High-level overview of deployment options and decision guide

**Content**:
- Build for production (cross-compilation, ldflags)
- Deployment decision tree (when to use what)
- Environment configuration
- Health checks and readiness
- Graceful shutdown
- Monitoring and observability
- Security considerations

### 2. deployment/docker.mdx
**Purpose**: Complete Docker deployment guide

**Content**:
- Multi-stage Dockerfile (build + runtime)
- Distroless vs Alpine images
- Building for different architectures
- Docker Compose for local development
- Docker Compose for production (with databases)
- Environment variables and secrets
- Volume mounts for data persistence
- Container health checks
- Docker networking
- Image optimization (size, layers)
- Private registries

### 3. deployment/kubernetes.mdx
**Purpose**: Kubernetes deployment and orchestration

**Content**:
- Deployment manifest
- Service exposure (ClusterIP, LoadBalancer, Ingress)
- ConfigMaps and Secrets
- Health probes (liveness, readiness, startup)
- Resource limits and requests
- Horizontal Pod Autoscaler
- Rolling updates and rollbacks
- Helm chart basics
- Ingress with TLS
- Persistent volumes (if needed)
- Namespace organization
- Common kubectl commands

### 4. deployment/cloud.mdx
**Purpose**: Cloud provider specific guides

**Content**:
- **AWS**:
  - EC2 (direct deployment)
  - Elastic Beanstalk
  - ECS (Fargate)
  - App Runner
  - Load Balancer + Auto Scaling
- **Google Cloud**:
  - Compute Engine
  - Cloud Run
  - GKE basics
- **DigitalOcean**:
  - Droplets
  - App Platform
  - Kubernetes (DOKS)
- **Azure**:
  - Azure Container Instances
  - Azure App Service
- **Fly.io**:
  - fly.toml configuration
  - Scaling and regions
- **Railway/Render**:
  - One-click deployment

### 5. deployment/serverless.mdx
**Purpose**: Serverless and function-as-a-service deployment

**Content**:
- AWS Lambda with API Gateway
- AWS Lambda with Function URLs
- Google Cloud Functions
- Google Cloud Run (serverless containers)
- Azure Functions
- Vercel (Go runtime)
- Cold starts and optimization
- Limitations and workarounds
- When serverless makes sense

### 6. deployment/traditional.mdx
**Purpose**: Traditional server deployment

**Content**:
- Building the binary
- Uploading to server (scp, rsync)
- systemd service configuration
- Running as non-root user
- Reverse proxy with Caddy
- Reverse proxy with Nginx
- Let's Encrypt SSL certificates
- Log rotation
- Process monitoring (monit, supervisor)
- Firewall configuration (ufw, iptables)
- Zero-downtime deployments

### 7. deployment/cicd.mdx
**Purpose**: Continuous integration and deployment

**Content**:
- GitHub Actions workflow
- GitLab CI/CD pipeline
- Testing in CI
- Building Docker images in CI
- Pushing to container registries
- Deploying to servers via SSH
- Deploying to Kubernetes
- Deploying to cloud platforms
- Secrets management in CI
- Release automation
- Semantic versioning
- Changelog generation

## Implementation Plan

1. Create `docs/guides/deployment/` directory
2. Write all 7 deployment pages with comprehensive examples
3. Update `docs.json` with new structure
4. Add icons to all new pages
5. Update cross-references in existing pages

## Success Criteria

- Each deployment guide is self-contained
- Real, working code examples
- Copy-paste ready configurations
- Security best practices included
- Troubleshooting sections where appropriate
