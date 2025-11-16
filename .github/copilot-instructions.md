# Heimdall Project: AI Contributor Guide

Welcome to the Heimdall project! You are helping build a decoupled, high-performance, open-source authentication (AuthN) and authorization (AuthZ) service.

Your primary goal is to **write code that fulfills the GitHub Issues** while adhering to the project's architecture.

## 1. Core Philosophy: "Orchestrate, Don't Create"

Heimdall's job is to be the "glue" that orchestrates best-in-class, cloud-native tools.

- **We are Pluggable:** All code MUST be written against the interfaces defined in `/internal/core/interfaces.go`. 1 Never write code in a handler that directly imports a specific vendor SDK (like Kratos or Cerbos). Handlers call the `core.Application` interfaces. The _implementations_ of those interfaces live in `/internal/plugins/`.
    
- **We are Decoupled:** Heimdall knows _nothing_ about business logic (like "projects" or "documents"). It only knows about "principals," "resources," and "attributes." 3
    
- **We are Event-Driven:** Heimdall _must not_ make synchronous, blocking calls to other services (like the Billing App) during an auth check. All external state (like a user's subscription plan) is synchronized _asynchronously_ via the `EventConsumer` (NATS). 5
    
- **We are Secure by Default:** We use PASETO, not JWT, for all tokens we mint. 8
    

## 2. The Canon Tech Stack

When implementing a feature, use these specific technologies.

|**Component**|**Technology**|**Go Module**|
|---|---|---|
|**HTTP Routing**|**go-chi (v5)**|`github.com/go-chi/chi/v5`|
|**AuthN Backend**|**Ory Kratos/Hydra**|`github.com/ory/kratos-client-go`, `github.com/ory/hydra-client-go`|
|**AuthZ Backend**|**Cerbos**|`github.com/cerbos/cerbos-sdk-go`|
|**Access Tokens**|**PASETO (v4)**|`aidanwoods.dev/go-paseto`|
|**Event Bus**|**NATS (JetStream)**|`github.com/nats-io/nats.go`|

## 3. How to Contribute (Your Task Workflow)

1. **Assign an Issue:** Pick an unassigned GitHub Issue.
    
2. **Create a Branch:** Create a new branch named `issue/123-short-description`.
    
3. **Write the Code:**
    
    - Implement the feature as described in the issue.
        
    - **Always write code against the interfaces in `/internal/core/`.**
        
    - Use standard Go concurrency patterns (goroutines, channels, `sync.WaitGroup`) where appropriate. 12
        
    - **CRITICAL:** Every function that handles a request or could be long-running MUST accept `context.Context` as its first argument for cancellation and timeouts. 14
        
4. **Write the Tests:**
    
    - All new code _must_ be accompanied by unit tests.
        
    - Use the standard `testing` package and `testify/assert` for assertions.
        
    - When testing a plugin, use mocks (e.g., `testify/mock`) for the external SDKs.
        
    - When testing an HTTP handler, use `net/http/httptest` to mock the request and response.
        
5. **Format and Lint:** Run `gofmt -w.` and `go vet./...` before committing.
    
6. **Open a Pull Request:** Reference the issue number in your PR (e.g., "Closes #123").
    

## 4. How to Run the Project for Testing

1. Ensure you have Docker and Docker Compose installed.
    
2. Ensure the configuration files for Kratos, Hydra, and Cerbos exist in `./infra/`.
    
3. Run `docker-compose up -d --build`.
    
4. This will start Heimdall and all its dependencies. Heimdall will be available at `http://localhost:8080`.
