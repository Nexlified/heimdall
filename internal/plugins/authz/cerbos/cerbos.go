package cerbos

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/cerbos/cerbos-sdk-go/cerbos"
)

// Client is the concrete implementation of core.PolicyEngine
// using Cerbos for authorization decisions.
type Client struct {
	cerbosClient *cerbos.GRPCClient
}

// NewCerbosClient initializes a new Cerbos client with the provided gRPC URL.
// It creates a gRPC client for communicating with the Cerbos Policy Decision Point (PDP).
func NewCerbosClient(grpcURL string) (*Client, error) {
	if grpcURL == "" {
		return nil, errors.New("grpcURL is required")
	}

	// Initialize Cerbos gRPC client
	cerbosClient, err := cerbos.New(grpcURL, cerbos.WithPlaintext())
	if err != nil {
		return nil, fmt.Errorf("failed to create Cerbos client: %w", err)
	}

	return &Client{
		cerbosClient: cerbosClient,
	}, nil
}

// CheckRequest represents the JSON structure for a Check request.
// This matches the Cerbos CheckResources API format.
type CheckRequest struct {
	Principal *PrincipalData   `json:"principal"`
	Resources []*ResourceEntry `json:"resources"`
}

// PrincipalData represents the principal (user) making the request.
type PrincipalData struct {
	ID         string                 `json:"id"`
	Roles      []string               `json:"roles"`
	Attributes map[string]interface{} `json:"attr,omitempty"`
	Scope      string                 `json:"scope,omitempty"`
}

// ResourceEntry represents a resource to be checked with its actions.
type ResourceEntry struct {
	Kind       string                 `json:"kind"`
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attr,omitempty"`
	Actions    []string               `json:"actions"`
	Scope      string                 `json:"scope,omitempty"`
}

// PlanResourcesRequest represents the JSON structure for a PlanResources request.
// This matches the Cerbos PlanResources API format for solving the N+1 query problem.
type PlanResourcesRequest struct {
	Principal *PrincipalData `json:"principal"`
	Resource  *ResourceData  `json:"resource"`
	Actions   []string       `json:"actions"`
}

// ResourceData represents a resource template (without specific ID) for planning.
// Used in PlanResources to describe a set of resources rather than a concrete instance.
type ResourceData struct {
	Kind       string                 `json:"kind"`
	Attributes map[string]interface{} `json:"attr,omitempty"`
	Scope      string                 `json:"scope,omitempty"`
}

// Check performs a simple "allow/deny" check for a request.
// The request payload should be the raw JSON bytes of the Cerbos CheckResource API.
// It unmarshals the JSON, calls the Cerbos CheckResources method, and returns the marshaled response.
func (c *Client) Check(ctx context.Context, checkRequest []byte) ([]byte, error) {
	if len(checkRequest) == 0 {
		return nil, errors.New("checkRequest cannot be empty")
	}

	// Unmarshal the JSON request into our CheckRequest struct
	var req CheckRequest
	if err := json.Unmarshal(checkRequest, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal check request: %w", err)
	}

	// Validate the request structure
	if req.Principal == nil {
		return nil, errors.New("principal is required")
	}
	if len(req.Resources) == 0 {
		return nil, errors.New("at least one resource is required")
	}

	// Build the Cerbos Principal
	principal := cerbos.NewPrincipal(req.Principal.ID, req.Principal.Roles...)
	if req.Principal.Attributes != nil {
		principal = principal.WithAttributes(req.Principal.Attributes)
	}
	if req.Principal.Scope != "" {
		principal = principal.WithScope(req.Principal.Scope)
	}

	// Build the Cerbos ResourceBatch
	resourceBatch := cerbos.NewResourceBatch()
	for _, res := range req.Resources {
		resource := cerbos.NewResource(res.Kind, res.ID)
		if res.Attributes != nil {
			resource = resource.WithAttributes(res.Attributes)
		}
		if res.Scope != "" {
			resource = resource.WithScope(res.Scope)
		}
		resourceBatch.Add(resource, res.Actions...)
	}

	// Call the Cerbos CheckResources method
	resp, err := c.cerbosClient.CheckResources(ctx, principal, resourceBatch)
	if err != nil {
		return nil, fmt.Errorf("failed to check resources with Cerbos: %w", err)
	}

	// Marshal the response back to JSON
	// The CheckResourcesResponse has a MarshalJSON method
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal check response: %w", err)
	}

	return respJSON, nil
}

// PlanResources returns a query plan for list filtering (The "N+1" solution).
// The request payload is the raw Cerbos PlanResources API in JSON format.
// It unmarshals the request, calls Cerbos to generate a query plan, and returns the marshaled response.
// This enables efficient filtering of resource lists without making individual authorization checks.
func (c *Client) PlanResources(ctx context.Context, planRequest []byte) ([]byte, error) {
	if len(planRequest) == 0 {
		return nil, errors.New("planRequest cannot be empty")
	}

	// Unmarshal the JSON request into our PlanResourcesRequest struct
	var req PlanResourcesRequest
	if err := json.Unmarshal(planRequest, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan request: %w", err)
	}

	// Validate the request structure
	if req.Principal == nil {
		return nil, errors.New("principal is required")
	}
	if req.Resource == nil {
		return nil, errors.New("resource is required")
	}
	if len(req.Actions) == 0 {
		return nil, errors.New("at least one action is required")
	}

	// Build the Cerbos Principal
	principal := cerbos.NewPrincipal(req.Principal.ID, req.Principal.Roles...)
	if req.Principal.Attributes != nil {
		principal = principal.WithAttributes(req.Principal.Attributes)
	}
	if req.Principal.Scope != "" {
		principal = principal.WithScope(req.Principal.Scope)
	}

	// Build the Cerbos Resource for planning
	// Note: PlanResources uses a resource template without a specific ID
	resource := cerbos.NewResource(req.Resource.Kind, "")
	if req.Resource.Attributes != nil {
		resource = resource.WithAttributes(req.Resource.Attributes)
	}
	if req.Resource.Scope != "" {
		resource = resource.WithScope(req.Resource.Scope)
	}

	// Call the Cerbos PlanResources method
	resp, err := c.cerbosClient.PlanResources(ctx, principal, resource, req.Actions...)
	if err != nil {
		return nil, fmt.Errorf("failed to plan resources with Cerbos: %w", err)
	}

	// Marshal the response back to JSON
	respJSON, err := json.Marshal(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal plan response: %w", err)
	}

	return respJSON, nil
}

// UpdateAttributes updates the attributes for a principal (e.g., from an event).
// This is called by the EventConsumer.
// This is a stub implementation for now.
func (c *Client) UpdateAttributes(ctx context.Context, principalID string, attributes map[string]any) error {
	// TODO: Implement UpdateAttributes
	// This would typically update a cache or data store that Cerbos consumes
	return errors.New("UpdateAttributes not implemented yet")
}
