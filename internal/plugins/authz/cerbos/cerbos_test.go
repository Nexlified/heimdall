package cerbos

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewCerbosClient tests the constructor with various configurations
func TestNewCerbosClient(t *testing.T) {
	tests := []struct {
		name        string
		grpcURL     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid gRPC URL",
			grpcURL:     "localhost:3593",
			expectError: false,
		},
		{
			name:        "empty gRPC URL",
			grpcURL:     "",
			expectError: true,
			errorMsg:    "grpcURL is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewCerbosClient(tt.grpcURL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, client)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.NotNil(t, client.cerbosClient)
			}
		})
	}
}

// TestCheckRequest_Unmarshal tests unmarshaling of various JSON structures
func TestCheckRequest_Unmarshal(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		validate    func(t *testing.T, req *CheckRequest)
	}{
		{
			name: "valid request with single resource",
			jsonData: `{
				"principal": {
					"id": "user123",
					"roles": ["user", "viewer"],
					"attr": {
						"department": "engineering"
					}
				},
				"resources": [
					{
						"kind": "document",
						"id": "doc123",
						"actions": ["view", "edit"],
						"attr": {
							"owner": "user123"
						}
					}
				]
			}`,
			expectError: false,
			validate: func(t *testing.T, req *CheckRequest) {
				assert.Equal(t, "user123", req.Principal.ID)
				assert.Equal(t, []string{"user", "viewer"}, req.Principal.Roles)
				assert.Equal(t, "engineering", req.Principal.Attributes["department"])
				assert.Len(t, req.Resources, 1)
				assert.Equal(t, "document", req.Resources[0].Kind)
				assert.Equal(t, "doc123", req.Resources[0].ID)
				assert.Equal(t, []string{"view", "edit"}, req.Resources[0].Actions)
			},
		},
		{
			name: "valid request with multiple resources",
			jsonData: `{
				"principal": {
					"id": "user456",
					"roles": ["admin"]
				},
				"resources": [
					{
						"kind": "project",
						"id": "proj1",
						"actions": ["delete"]
					},
					{
						"kind": "project",
						"id": "proj2",
						"actions": ["view"]
					}
				]
			}`,
			expectError: false,
			validate: func(t *testing.T, req *CheckRequest) {
				assert.Equal(t, "user456", req.Principal.ID)
				assert.Len(t, req.Resources, 2)
			},
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
		},
		{
			name:        "empty JSON",
			jsonData:    `{}`,
			expectError: false,
			validate: func(t *testing.T, req *CheckRequest) {
				assert.Nil(t, req.Principal)
				assert.Nil(t, req.Resources)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req CheckRequest
			err := json.Unmarshal([]byte(tt.jsonData), &req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, &req)
				}
			}
		})
	}
}

// TestCheck_Validation tests the validation logic in the Check method
func TestCheck_Validation(t *testing.T) {
	// Create a client (will fail to connect but we just need it for validation tests)
	client, err := NewCerbosClient("localhost:9999")
	require.NoError(t, err)

	tests := []struct {
		name        string
		requestJSON string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty request",
			requestJSON: "",
			expectError: true,
			errorMsg:    "checkRequest cannot be empty",
		},
		{
			name:        "invalid JSON",
			requestJSON: `{"invalid": json}`,
			expectError: true,
			errorMsg:    "failed to unmarshal check request",
		},
		{
			name: "missing principal",
			requestJSON: `{
				"resources": [
					{
						"kind": "document",
						"id": "doc123",
						"actions": ["view"]
					}
				]
			}`,
			expectError: true,
			errorMsg:    "principal is required",
		},
		{
			name: "missing resources",
			requestJSON: `{
				"principal": {
					"id": "user123",
					"roles": ["user"]
				},
				"resources": []
			}`,
			expectError: true,
			errorMsg:    "at least one resource is required",
		},
		{
			name: "no resources field",
			requestJSON: `{
				"principal": {
					"id": "user123",
					"roles": ["user"]
				}
			}`,
			expectError: true,
			errorMsg:    "at least one resource is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := client.Check(ctx, []byte(tt.requestJSON))

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Note: Will likely fail with connection error in unit test environment
				// but we're testing validation, not actual Cerbos connection
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheck_Integration tests the Check method with a valid request structure
// Note: This test validates the request/response flow but will fail without a running Cerbos instance
func TestCheck_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires running Cerbos instance")

	client, err := NewCerbosClient("localhost:3593")
	require.NoError(t, err)

	requestJSON := `{
		"principal": {
			"id": "user123",
			"roles": ["user"],
			"attr": {
				"department": "engineering"
			}
		},
		"resources": [
			{
				"kind": "document",
				"id": "doc123",
				"actions": ["view", "edit"],
				"attr": {
					"owner": "user123"
				}
			}
		]
	}`

	ctx := context.Background()
	resp, err := client.Check(ctx, []byte(requestJSON))

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Validate that response is valid JSON
	var responseMap map[string]interface{}
	err = json.Unmarshal(resp, &responseMap)
	assert.NoError(t, err)
}

// TestPlanResources_Validation tests the validation logic in the PlanResources method
func TestPlanResources_Validation(t *testing.T) {
	// Create a client (will fail to connect but we just need it for validation tests)
	client, err := NewCerbosClient("localhost:9999")
	require.NoError(t, err)

	tests := []struct {
		name        string
		requestJSON string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty request",
			requestJSON: "",
			expectError: true,
			errorMsg:    "planRequest cannot be empty",
		},
		{
			name:        "invalid JSON",
			requestJSON: `{"invalid": json}`,
			expectError: true,
			errorMsg:    "failed to unmarshal plan request",
		},
		{
			name: "missing principal",
			requestJSON: `{
				"resource": {
					"kind": "document"
				},
				"actions": ["view"]
			}`,
			expectError: true,
			errorMsg:    "principal is required",
		},
		{
			name: "missing resource",
			requestJSON: `{
				"principal": {
					"id": "user123",
					"roles": ["user"]
				},
				"actions": ["view"]
			}`,
			expectError: true,
			errorMsg:    "resource is required",
		},
		{
			name: "missing actions",
			requestJSON: `{
				"principal": {
					"id": "user123",
					"roles": ["user"]
				},
				"resource": {
					"kind": "document"
				},
				"actions": []
			}`,
			expectError: true,
			errorMsg:    "at least one action is required",
		},
		{
			name: "no actions field",
			requestJSON: `{
				"principal": {
					"id": "user123",
					"roles": ["user"]
				},
				"resource": {
					"kind": "document"
				}
			}`,
			expectError: true,
			errorMsg:    "at least one action is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			resp, err := client.PlanResources(ctx, []byte(tt.requestJSON))

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, resp)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				// Note: Will likely fail with connection error in unit test environment
				// but we're testing validation, not actual Cerbos connection
				assert.NoError(t, err)
			}
		})
	}
}

// TestPlanResourcesRequest_Unmarshal tests unmarshaling of various JSON structures
func TestPlanResourcesRequest_Unmarshal(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		validate    func(t *testing.T, req *PlanResourcesRequest)
	}{
		{
			name: "valid request with attributes",
			jsonData: `{
				"principal": {
					"id": "user123",
					"roles": ["user", "viewer"],
					"attr": {
						"department": "engineering"
					}
				},
				"resource": {
					"kind": "document",
					"attr": {
						"public": false
					}
				},
				"actions": ["view", "edit"]
			}`,
			expectError: false,
			validate: func(t *testing.T, req *PlanResourcesRequest) {
				assert.Equal(t, "user123", req.Principal.ID)
				assert.Equal(t, []string{"user", "viewer"}, req.Principal.Roles)
				assert.Equal(t, "engineering", req.Principal.Attributes["department"])
				assert.Equal(t, "document", req.Resource.Kind)
				assert.Equal(t, false, req.Resource.Attributes["public"])
				assert.Equal(t, []string{"view", "edit"}, req.Actions)
			},
		},
		{
			name: "valid request with scope",
			jsonData: `{
				"principal": {
					"id": "user456",
					"roles": ["admin"],
					"scope": "tenant:acme"
				},
				"resource": {
					"kind": "project",
					"scope": "tenant:acme"
				},
				"actions": ["delete"]
			}`,
			expectError: false,
			validate: func(t *testing.T, req *PlanResourcesRequest) {
				assert.Equal(t, "user456", req.Principal.ID)
				assert.Equal(t, "tenant:acme", req.Principal.Scope)
				assert.Equal(t, "project", req.Resource.Kind)
				assert.Equal(t, "tenant:acme", req.Resource.Scope)
				assert.Equal(t, []string{"delete"}, req.Actions)
			},
		},
		{
			name: "minimal valid request",
			jsonData: `{
				"principal": {
					"id": "user789",
					"roles": ["user"]
				},
				"resource": {
					"kind": "file"
				},
				"actions": ["read"]
			}`,
			expectError: false,
			validate: func(t *testing.T, req *PlanResourcesRequest) {
				assert.Equal(t, "user789", req.Principal.ID)
				assert.Equal(t, "file", req.Resource.Kind)
				assert.Equal(t, []string{"read"}, req.Actions)
			},
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
		},
		{
			name:        "empty JSON",
			jsonData:    `{}`,
			expectError: false,
			validate: func(t *testing.T, req *PlanResourcesRequest) {
				assert.Nil(t, req.Principal)
				assert.Nil(t, req.Resource)
				assert.Nil(t, req.Actions)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req PlanResourcesRequest
			err := json.Unmarshal([]byte(tt.jsonData), &req)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, &req)
				}
			}
		})
	}
}

// TestPlanResources_Integration tests the PlanResources method with a valid request structure
// Note: This test validates the request/response flow but will fail without a running Cerbos instance
func TestPlanResources_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires running Cerbos instance")

	client, err := NewCerbosClient("localhost:3593")
	require.NoError(t, err)

	requestJSON := `{
		"principal": {
			"id": "user123",
			"roles": ["user"],
			"attr": {
				"department": "engineering"
			}
		},
		"resource": {
			"kind": "document",
			"attr": {
				"public": false
			}
		},
		"actions": ["view", "edit"]
	}`

	ctx := context.Background()
	resp, err := client.PlanResources(ctx, []byte(requestJSON))

	assert.NoError(t, err)
	assert.NotNil(t, resp)

	// Validate that response is valid JSON
	var responseMap map[string]interface{}
	err = json.Unmarshal(resp, &responseMap)
	assert.NoError(t, err)
}

// TestPlanResources_OldStub is removed - replaced with comprehensive tests above
// The PlanResources method is now fully implemented

// TestUpdateAttributes tests the UpdateAttributes method validation and behavior
func TestUpdateAttributes(t *testing.T) {
	client, err := NewCerbosClient("localhost:9999")
	require.NoError(t, err)

	tests := []struct {
		name        string
		principalID string
		attributes  map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid update with attributes",
			principalID: "user123",
			attributes: map[string]any{
				"department": "engineering",
				"plan":       "pro",
			},
			expectError: false,
		},
		{
			name:        "valid update with empty attributes map",
			principalID: "user456",
			attributes:  map[string]any{},
			expectError: false,
		},
		{
			name:        "empty principal ID",
			principalID: "",
			attributes: map[string]any{
				"department": "engineering",
			},
			expectError: true,
			errorMsg:    "principalID is required",
		},
		{
			name:        "nil attributes",
			principalID: "user789",
			attributes:  nil,
			expectError: true,
			errorMsg:    "attributes cannot be nil",
		},
		{
			name:        "update with complex attributes",
			principalID: "user999",
			attributes: map[string]any{
				"department":    "engineering",
				"current_users": 28,
				"max_users":     50,
				"plan":          "enterprise",
				"features":      []string{"sso", "audit"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			err := client.UpdateAttributes(ctx, tt.principalID, tt.attributes)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestUpdateAttributes_ContextCancellation tests that UpdateAttributes respects context cancellation
func TestUpdateAttributes_ContextCancellation(t *testing.T) {
	client, err := NewCerbosClient("localhost:9999")
	require.NoError(t, err)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attributes := map[string]any{
		"department": "engineering",
	}

	// Even with a cancelled context, UpdateAttributes should complete quickly
	// since it's a no-op. This test verifies it doesn't hang.
	err = client.UpdateAttributes(ctx, "user123", attributes)
	assert.NoError(t, err)
}

// TestCheckRequest_WithScope tests requests with scope
func TestCheckRequest_WithScope(t *testing.T) {
	requestJSON := `{
		"principal": {
			"id": "user123",
			"roles": ["user"],
			"scope": "tenant:acme"
		},
		"resources": [
			{
				"kind": "document",
				"id": "doc123",
				"actions": ["view"],
				"scope": "tenant:acme"
			}
		]
	}`

	var req CheckRequest
	err := json.Unmarshal([]byte(requestJSON), &req)
	require.NoError(t, err)

	assert.Equal(t, "tenant:acme", req.Principal.Scope)
	assert.Equal(t, "tenant:acme", req.Resources[0].Scope)
}
