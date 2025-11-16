# Heimdall

> A decoupled, high-performance authentication and authorization orchestrator

Heimdall is a central, pluggable microservice written in Go that functions as both an **Identity Orchestrator** and a **Policy Decision Point (PDP)**. It orchestrates best-in-class, cloud-native authentication and authorization services while maintaining complete decoupling from your business logic.

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

## Table of Contents

- [Purpose](#purpose)
- [Architecture Philosophy](#architecture-philosophy)
- [Tech Stack](#tech-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Quick Start with Docker Compose](#quick-start-with-docker-compose)
  - [Configuration](#configuration)
- [Usage Guide](#usage-guide)
  - [Authentication Flow](#authentication-flow)
  - [Authorization Checks](#authorization-checks)
  - [Resource Planning](#resource-planning)
- [Developer Documentation](#developer-documentation)
  - [Project Structure](#project-structure)
  - [Core Interfaces](#core-interfaces)
  - [Building from Source](#building-from-source)
  - [Running Tests](#running-tests)
  - [Adding New Plugins](#adding-new-plugins)
- [Integration Steps](#integration-steps)
  - [Integrating with Your API Gateway](#integrating-with-your-api-gateway)
  - [Event-Driven Attribute Synchronization](#event-driven-attribute-synchronization)
  - [Example Integration Patterns](#example-integration-patterns)
- [API Reference](#api-reference)
- [Contributing](#contributing)
- [License](#license)

---

## Purpose

Heimdall solves the problem of **tightly-coupled authentication and authorization** in modern microservices architectures. Instead of building yet another custom auth service, Heimdall orchestrates industry-leading open-source tools into a cohesive, event-driven system.

### Key Problems Solved

1. **Vendor Lock-In**: Heimdall's pluggable architecture means you're never locked into a specific auth provider
2. **Business Logic Coupling**: Heimdall knows nothing about your business domain—it only understands principals, resources, and attributes
3. **N+1 Query Problem**: Built-in support for query planning enables efficient list filtering without individual checks
4. **Synchronous External Calls**: Event-driven architecture prevents blocking calls to external services during authorization
5. **Token Security**: Uses PASETO (not JWT) for cryptographically secure tokens by default

### What Heimdall Does

- **Orchestrates Authentication**: Wraps identity providers (like Ory Kratos/Hydra) for SSO, OIDC, and OAuth2 flows
- **Centralizes Authorization**: Integrates with policy engines (like Cerbos) for fine-grained, attribute-based access control
- **Mints Secure Tokens**: Issues PASETO v4 tokens with principal identity and attributes
- **Synchronizes State**: Consumes events from business services (via NATS JetStream) to keep authorization attributes up-to-date
- **Enables Gateway Integration**: Provides simple HTTP APIs that your API Gateway can call to make auth decisions

---

## Architecture Philosophy

Heimdall follows a core principle: **"Orchestrate, Don't Create"**

### Design Principles

1. **Pluggable by Design**: All code is written against interfaces defined in `/internal/core/interfaces.go`. Implementations live in `/internal/plugins/`.
   
2. **Decoupled from Business Logic**: Heimdall operates on generic concepts (principals, resources, attributes)—not your domain models.

3. **Event-Driven**: External state synchronization happens asynchronously via message queues, never blocking auth decisions.

4. **Secure by Default**: PASETO tokens, not JWT. Attribute-based access control, not role-based.

5. **Cloud-Native**: Designed for Kubernetes, Docker Compose, and container orchestration from day one.

---

## Tech Stack

Heimdall uses the following battle-tested technologies:

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **HTTP Framework** | [go-chi/chi v5](https://github.com/go-chi/chi) | Lightweight, idiomatic HTTP routing |
| **Authentication** | [Ory Kratos](https://www.ory.sh/kratos/) & [Hydra](https://www.ory.sh/hydra/) | Identity management and OAuth2/OIDC |
| **Authorization** | [Cerbos](https://cerbos.dev/) | Policy decision point with attribute-based access control |
| **Tokens** | [PASETO v4](https://paseto.io/) | Platform-agnostic security tokens |
| **Event Bus** | [NATS JetStream](https://nats.io/) | Reliable, distributed messaging |
| **Language** | [Go 1.24+](https://go.dev/) | Performance, concurrency, and simplicity |

---

## Getting Started

### Prerequisites

- **Docker** & **Docker Compose** (for local development)
- **Go 1.24+** (if building from source)
- **Make** (optional, for build automation)

### Quick Start with Docker Compose

The fastest way to run Heimdall with all dependencies:

```bash
# Clone the repository
git clone https://github.com/Nexlified/heimdall.git
cd heimdall

# Start all services (Heimdall, Kratos, Hydra, Cerbos, NATS)
docker-compose up -d

# Verify Heimdall is running
curl http://localhost:8080/health
```

This starts:
- **Heimdall** on `http://localhost:8080`
- **Ory Kratos** (admin: 4434, public: 4433)
- **Ory Hydra** (admin: 4445, public: 4444)
- **Cerbos** (gRPC: 3592, HTTP: 3593)
- **NATS JetStream** (client: 4222, monitoring: 8222)

### Configuration

Heimdall is configured via environment variables:

```bash
# Identity Provider (Ory Kratos/Hydra)
KRATOS_ADMIN_URL=http://kratos:4434
HYDRA_ADMIN_URL=http://hydra:4445
HYDRA_PUBLIC_URL=http://localhost:4444

# Policy Engine (Cerbos)
CERBOS_GRPC_URL=cerbos:3592

# Token Service (PASETO)
PASETO_SYMMETRIC_KEY=your-32-byte-secret-key-here!!!

# Event Consumer (NATS)
NATS_URL=nats://nats:4222

# Server
PORT=8080
```

**Important**: The `PASETO_SYMMETRIC_KEY` must be exactly 32 bytes for AES-256.

---

## Usage Guide

### Authentication Flow

#### 1. Initiate Login

Your frontend redirects users to Heimdall's login endpoint:

```bash
# User navigates to:
GET http://localhost:8080/auth/login?redirect_uri=https://myapp.com/dashboard
```

This triggers the OAuth2/OIDC flow with Kratos and Hydra.

#### 2. Handle Callback

After successful authentication, Kratos/Hydra redirects to:

```bash
GET http://localhost:8080/auth/callback?code=<authorization_code>
```

Heimdall exchanges the code for tokens and returns a PASETO token:

```json
{
  "access_token": "v4.local.encoded-paseto-token...",
  "expires_in": 3600
}
```

#### 3. Use the Token

Include the token in subsequent requests:

```bash
curl -H "Authorization: Bearer v4.local...." \
  http://localhost:8080/check
```

### Authorization Checks

Check if a principal can perform actions on resources:

```bash
POST http://localhost:8080/check
Content-Type: application/json

{
  "principal": {
    "id": "user123",
    "roles": ["user", "editor"],
    "attr": {
      "department": "engineering",
      "plan": "pro"
    }
  },
  "resources": [
    {
      "kind": "document",
      "id": "doc456",
      "attr": {
        "owner": "user123",
        "status": "draft"
      },
      "actions": ["view", "edit", "delete"]
    }
  ]
}
```

**Response** (200 OK = allowed, 403 Forbidden = denied):

```json
{
  "resourceId": "doc456",
  "actions": {
    "view": "EFFECT_ALLOW",
    "edit": "EFFECT_ALLOW",
    "delete": "EFFECT_DENY"
  }
}
```

### Resource Planning

Solve the N+1 query problem by getting a query plan for filtering lists:

```bash
POST http://localhost:8080/plan/resources
Content-Type: application/json

{
  "principal": {
    "id": "user123",
    "roles": ["user"],
    "attr": {
      "department": "engineering"
    }
  },
  "resource": {
    "kind": "document",
    "attr": {}
  },
  "actions": ["view"]
}
```

**Response**:

```json
{
  "filter": {
    "kind": "KIND_CONDITIONAL",
    "condition": {
      "expression": "request.resource.attr.owner == request.principal.id || request.resource.attr.public == true"
    }
  }
}
```

Use this filter in your database query to fetch only authorized resources.

---

## Developer Documentation

### Project Structure

```
heimdall/
├── cmd/
│   └── heimdall/
│       └── main.go              # Application entry point
├── internal/
│   ├── core/
│   │   └── interfaces.go        # Core interfaces (NEVER modify implementations here)
│   ├── handlers/
│   │   ├── http.go              # HTTP request handlers
│   │   └── http_test.go         # Handler tests
│   ├── plugins/                 # Plugin implementations
│   │   ├── authn/
│   │   │   └── kratos/          # Ory Kratos/Hydra identity provider
│   │   ├── authz/
│   │   │   └── cerbos/          # Cerbos policy engine
│   │   └── events/
│   │       └── nats/            # NATS JetStream event consumer
│   └── tokens/
│       └── paseto.go            # PASETO token service
├── infra/                       # Infrastructure configs
│   ├── kratos/
│   │   └── kratos.yml
│   └── cerbos/
│       └── policies/
├── docker-compose.yml           # Local development stack
├── Dockerfile                   # Multi-stage production build
└── go.mod                       # Go dependencies
```

### Core Interfaces

All features are built against these interfaces (see `/internal/core/interfaces.go`):

#### IdentityProvider

```go
type IdentityProvider interface {
    InitiateLogin(w http.ResponseWriter, r *http.Request)
    HandleAuthCallback(r *http.Request) (*TokenResponse, error)
    RefreshToken(refreshToken string) (*TokenResponse, error)
}
```

#### PolicyEngine

```go
type PolicyEngine interface {
    Check(ctx context.Context, checkRequest []byte) ([]byte, error)
    PlanResources(ctx context.Context, planRequest []byte) ([]byte, error)
    UpdateAttributes(ctx context.Context, principalID string, attributes map[string]any) error
}
```

#### EventConsumer

```go
type EventConsumer interface {
    Consume(pdp PolicyEngine) error
}
```

### Building from Source

```bash
# Install dependencies
go mod download

# Build the binary
go build -o heimdall ./cmd/heimdall

# Run locally (requires env vars)
./heimdall
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/plugins/authz/cerbos/...

# Run with coverage
go test -cover ./...
```

### Adding New Plugins

Heimdall's architecture makes it easy to add new implementations:

#### Example: Adding a new Identity Provider

1. Create a new package under `/internal/plugins/authn/<provider>/`
2. Implement the `core.IdentityProvider` interface
3. Write comprehensive tests
4. Update `cmd/heimdall/main.go` to wire it up

```go
// internal/plugins/authn/auth0/auth0.go
package auth0

import "github.com/nexlified/heimdall/internal/core"

type Client struct {
    // Your implementation
}

func (c *Client) InitiateLogin(w http.ResponseWriter, r *http.Request) {
    // Implement Auth0 login flow
}

func (c *Client) HandleAuthCallback(r *http.Request) (*core.TokenResponse, error) {
    // Implement Auth0 callback handling
}

func (c *Client) RefreshToken(refreshToken string) (*core.TokenResponse, error) {
    // Implement token refresh
}
```

---

## Integration Steps

### Integrating with Your API Gateway

Heimdall is designed to work as a sidecar or upstream service for your API Gateway (Kong, Traefik, Nginx, etc.).

#### Pattern 1: Forward Auth (Traefik, Nginx)

Configure your gateway to call Heimdall before routing requests:

```yaml
# Traefik example
http:
  middlewares:
    heimdall-auth:
      forwardAuth:
        address: "http://heimdall:8080/check"
        authResponseHeaders:
          - "X-User-ID"
          - "X-User-Roles"
```

#### Pattern 2: External Auth (Kong, Envoy)

```yaml
# Kong example
plugins:
  - name: external-auth
    config:
      uri: "http://heimdall:8080/check"
      method: "POST"
```

#### Pattern 3: Direct Integration

Your application can call Heimdall's API directly:

```go
// In your Go application
func (s *Server) ListDocuments(w http.ResponseWriter, r *http.Request) {
    // 1. Get user from token (set by gateway or middleware)
    userID := r.Header.Get("X-User-ID")
    
    // 2. Get query plan from Heimdall
    plan, err := s.heimdallClient.PlanResources(ctx, PlanRequest{
        Principal: Principal{ID: userID, Roles: []string{"user"}},
        Resource:  Resource{Kind: "document"},
        Actions:   []string{"view"},
    })
    
    // 3. Apply plan to database query
    docs, err := s.db.ListDocuments(ctx, plan.Filter)
    
    // 4. Return results
    json.NewEncoder(w).Encode(docs)
}
```

### Event-Driven Attribute Synchronization

Heimdall stays up-to-date with your business state via NATS JetStream events.

#### Publishing Events from Your Services

**Billing Service** (when subscription changes):

```go
// Publish subscription.updated event
natsConn.Publish("subscription.updated", []byte(`{
  "user_id": "user123",
  "plan": "enterprise",
  "attributes": {
    "max_users": 500,
    "features": ["sso", "audit", "priority_support"]
  }
}`))
```

**Business Service** (when usage metrics change):

```go
// Publish usage.updated event
natsConn.Publish("usage.updated", []byte(`{
  "user_id": "user123",
  "attributes": {
    "current_users": 287,
    "storage_used_gb": 1250,
    "api_calls_today": 45000
  }
}`))
```

Heimdall's EventConsumer automatically:
1. Listens for these events
2. Validates and parses them
3. Calls `PolicyEngine.UpdateAttributes()` to acknowledge receipt
4. Ensures attributes are available for subsequent authorization checks

### Example Integration Patterns

#### Microservices Architecture

```
┌─────────────┐      ┌──────────────┐      ┌─────────────┐
│   Frontend  │─────▶│ API Gateway  │─────▶│ Heimdall    │
│  (React)    │      │  (Traefik)   │      │  (AuthN/Z)  │
└─────────────┘      └──────────────┘      └─────────────┘
                             │                     │
                             │                     │
                             ▼                     ▼
                     ┌──────────────┐      ┌─────────────┐
                     │  Business    │─────▶│   NATS      │
                     │  Services    │      │ JetStream   │
                     └──────────────┘      └─────────────┘
                                                  │
                                                  │
                                           ┌──────▼──────┐
                                           │  Cerbos     │
                                           │  (Policies) │
                                           └─────────────┘
```

#### Single-Page Application

```javascript
// Login flow
const login = async () => {
  window.location.href = 'http://heimdall.example.com/auth/login?redirect_uri=' 
    + encodeURIComponent(window.location.origin + '/dashboard');
};

// Handle callback
const handleCallback = async () => {
  const params = new URLSearchParams(window.location.search);
  const code = params.get('code');
  
  const response = await fetch('http://heimdall.example.com/auth/callback', {
    method: 'GET',
    headers: { 'Authorization': `Bearer ${code}` }
  });
  
  const { access_token } = await response.json();
  localStorage.setItem('token', access_token);
};

// Make authorized requests
const fetchDocuments = async () => {
  const token = localStorage.getItem('token');
  
  const response = await fetch('http://api.example.com/documents', {
    headers: { 'Authorization': `Bearer ${token}` }
  });
  
  return response.json();
};
```

---

## API Reference

### Authentication Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/auth/login` | GET | Initiates OAuth2/OIDC login flow |
| `/auth/callback` | GET | Handles OAuth2 callback, returns PASETO token |
| `/auth/refresh` | POST | Exchanges refresh token for new access token |

### Authorization Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/check` | POST | Performs access control check |
| `/plan/resources` | POST | Returns query plan for list filtering |

### Request/Response Examples

See [Usage Guide](#usage-guide) for detailed examples.

---

## Contributing

We welcome contributions! Please follow these guidelines:

1. **Read the Architecture**: Review `/internal/core/interfaces.go` and the [Developer Documentation](#developer-documentation)
2. **Write Tests**: All new code must have comprehensive unit tests
3. **Follow Conventions**: Use `gofmt`, `go vet`, and write idiomatic Go
4. **Open Issues First**: Discuss significant changes before implementing
5. **Document Your Code**: Add comments for complex logic

See [Implemented-By-Copilot.md](./Implemented-By-Copilot.md) for examples of completed implementations.

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) file for details.

---

## Support

- **Issues**: [GitHub Issues](https://github.com/Nexlified/heimdall/issues)
- **Discussions**: [GitHub Discussions](https://github.com/Nexlified/heimdall/discussions)
- **Documentation**: [Project Wiki](https://github.com/Nexlified/heimdall/wiki)

---

**Built with ❤️ by the Nexlified team**
