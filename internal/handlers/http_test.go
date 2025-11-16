package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	effectv1 "github.com/cerbos/cerbos/api/genpb/cerbos/effect/v1"
	responsev1 "github.com/cerbos/cerbos/api/genpb/cerbos/response/v1"
	"github.com/nexlified/heimdall/internal/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPolicyEngine is a mock implementation of core.PolicyEngine
type MockPolicyEngine struct {
	mock.Mock
}

func (m *MockPolicyEngine) Check(ctx context.Context, checkRequest []byte) ([]byte, error) {
	args := m.Called(ctx, checkRequest)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockPolicyEngine) PlanResources(ctx context.Context, planRequest []byte) ([]byte, error) {
	args := m.Called(ctx, planRequest)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockPolicyEngine) UpdateAttributes(ctx context.Context, principalID string, attributes map[string]any) error {
	args := m.Called(ctx, principalID, attributes)
	return args.Error(0)
}

// TestHandleCheck_NoPolicyEngine tests the handler when no policy engine is configured
func TestHandleCheck_NoPolicyEngine(t *testing.T) {
	app := &core.Application{
		PDP: nil, // No policy engine configured
	}
	handlers := NewHTTPHandlers(app)

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()

	handlers.HandleCheck(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Policy Engine not configured")
}

// TestHandleCheck_AllowResponse tests that EFFECT_ALLOW returns 200 OK
func TestHandleCheck_AllowResponse(t *testing.T) {
	mockPDP := new(MockPolicyEngine)

	// Create a Cerbos response with EFFECT_ALLOW
	cerbosResp := &responsev1.CheckResourcesResponse{
		RequestId: "test-request-id",
		Results: []*responsev1.CheckResourcesResponse_ResultEntry{
			{
				Resource: &responsev1.CheckResourcesResponse_ResultEntry_Resource{
					Id:   "doc123",
					Kind: "document",
				},
				Actions: map[string]effectv1.Effect{
					"view": effectv1.Effect_EFFECT_ALLOW,
					"edit": effectv1.Effect_EFFECT_ALLOW,
				},
			},
		},
	}

	// Marshal the response to JSON
	respJSON, err := json.Marshal(cerbosResp)
	require.NoError(t, err)

	// Set up the mock to return this response
	mockPDP.On("Check", mock.Anything, mock.Anything).Return(respJSON, nil)

	app := &core.Application{
		PDP: mockPDP,
	}
	handlers := NewHTTPHandlers(app)

	requestBody := `{
		"principal": {
			"id": "user123",
			"roles": ["user"]
		},
		"resources": [
			{
				"kind": "document",
				"id": "doc123",
				"actions": ["view", "edit"]
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader([]byte(requestBody)))
	w := httptest.NewRecorder()

	handlers.HandleCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Verify the response body is valid JSON
	var respMap map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &respMap)
	assert.NoError(t, err)

	mockPDP.AssertExpectations(t)
}

// TestHandleCheck_DenyResponse tests that EFFECT_DENY returns 403 Forbidden
func TestHandleCheck_DenyResponse(t *testing.T) {
	mockPDP := new(MockPolicyEngine)

	// Create a Cerbos response with EFFECT_DENY
	cerbosResp := &responsev1.CheckResourcesResponse{
		RequestId: "test-request-id",
		Results: []*responsev1.CheckResourcesResponse_ResultEntry{
			{
				Resource: &responsev1.CheckResourcesResponse_ResultEntry_Resource{
					Id:   "doc456",
					Kind: "document",
				},
				Actions: map[string]effectv1.Effect{
					"view":   effectv1.Effect_EFFECT_ALLOW,
					"delete": effectv1.Effect_EFFECT_DENY, // This should cause 403
				},
			},
		},
	}

	// Marshal the response to JSON
	respJSON, err := json.Marshal(cerbosResp)
	require.NoError(t, err)

	// Set up the mock to return this response
	mockPDP.On("Check", mock.Anything, mock.Anything).Return(respJSON, nil)

	app := &core.Application{
		PDP: mockPDP,
	}
	handlers := NewHTTPHandlers(app)

	requestBody := `{
		"principal": {
			"id": "user123",
			"roles": ["user"]
		},
		"resources": [
			{
				"kind": "document",
				"id": "doc456",
				"actions": ["view", "delete"]
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader([]byte(requestBody)))
	w := httptest.NewRecorder()

	handlers.HandleCheck(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	mockPDP.AssertExpectations(t)
}

// TestHandleCheck_MixedResponse tests that any EFFECT_DENY causes 403 Forbidden
func TestHandleCheck_MixedResponse(t *testing.T) {
	mockPDP := new(MockPolicyEngine)

	// Create a Cerbos response with multiple resources, one with DENY
	cerbosResp := &responsev1.CheckResourcesResponse{
		RequestId: "test-request-id",
		Results: []*responsev1.CheckResourcesResponse_ResultEntry{
			{
				Resource: &responsev1.CheckResourcesResponse_ResultEntry_Resource{
					Id:   "doc1",
					Kind: "document",
				},
				Actions: map[string]effectv1.Effect{
					"view": effectv1.Effect_EFFECT_ALLOW,
				},
			},
			{
				Resource: &responsev1.CheckResourcesResponse_ResultEntry_Resource{
					Id:   "doc2",
					Kind: "document",
				},
				Actions: map[string]effectv1.Effect{
					"view": effectv1.Effect_EFFECT_DENY, // This should cause 403
				},
			},
		},
	}

	// Marshal the response to JSON
	respJSON, err := json.Marshal(cerbosResp)
	require.NoError(t, err)

	// Set up the mock to return this response
	mockPDP.On("Check", mock.Anything, mock.Anything).Return(respJSON, nil)

	app := &core.Application{
		PDP: mockPDP,
	}
	handlers := NewHTTPHandlers(app)

	requestBody := `{}`

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader([]byte(requestBody)))
	w := httptest.NewRecorder()

	handlers.HandleCheck(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)

	mockPDP.AssertExpectations(t)
}

// TestHandleCheck_EmptyResults tests that empty results return 200 OK
func TestHandleCheck_EmptyResults(t *testing.T) {
	mockPDP := new(MockPolicyEngine)

	// Create a Cerbos response with no results
	cerbosResp := &responsev1.CheckResourcesResponse{
		RequestId: "test-request-id",
		Results:   []*responsev1.CheckResourcesResponse_ResultEntry{},
	}

	// Marshal the response to JSON
	respJSON, err := json.Marshal(cerbosResp)
	require.NoError(t, err)

	// Set up the mock to return this response
	mockPDP.On("Check", mock.Anything, mock.Anything).Return(respJSON, nil)

	app := &core.Application{
		PDP: mockPDP,
	}
	handlers := NewHTTPHandlers(app)

	requestBody := `{}`

	req := httptest.NewRequest(http.MethodPost, "/check", bytes.NewReader([]byte(requestBody)))
	w := httptest.NewRecorder()

	handlers.HandleCheck(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	mockPDP.AssertExpectations(t)
}

// TestAnalyzeCheckResponse tests the analyzeCheckResponse function directly
func TestAnalyzeCheckResponse(t *testing.T) {
	tests := []struct {
		name           string
		response       *responsev1.CheckResourcesResponse
		expectedStatus int
	}{
		{
			name: "all allow",
			response: &responsev1.CheckResourcesResponse{
				Results: []*responsev1.CheckResourcesResponse_ResultEntry{
					{
						Actions: map[string]effectv1.Effect{
							"view": effectv1.Effect_EFFECT_ALLOW,
							"edit": effectv1.Effect_EFFECT_ALLOW,
						},
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "single deny",
			response: &responsev1.CheckResourcesResponse{
				Results: []*responsev1.CheckResourcesResponse_ResultEntry{
					{
						Actions: map[string]effectv1.Effect{
							"delete": effectv1.Effect_EFFECT_DENY,
						},
					},
				},
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "mixed effects - should deny",
			response: &responsev1.CheckResourcesResponse{
				Results: []*responsev1.CheckResourcesResponse_ResultEntry{
					{
						Actions: map[string]effectv1.Effect{
							"view": effectv1.Effect_EFFECT_ALLOW,
						},
					},
					{
						Actions: map[string]effectv1.Effect{
							"delete": effectv1.Effect_EFFECT_DENY,
						},
					},
				},
			},
			expectedStatus: http.StatusForbidden,
		},
		{
			name: "empty results",
			response: &responsev1.CheckResourcesResponse{
				Results: []*responsev1.CheckResourcesResponse_ResultEntry{},
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			respJSON, err := json.Marshal(tt.response)
			require.NoError(t, err)

			status, err := analyzeCheckResponse(respJSON)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedStatus, status)
		})
	}
}

// TestHandleHealth tests the health check endpoint
func TestHandleHealth(t *testing.T) {
	app := &core.Application{
		PDP: nil, // Health endpoint doesn't depend on PDP
		IDP: nil, // Health endpoint doesn't depend on IDP
	}
	handlers := NewHTTPHandlers(app)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handlers.HandleHealth(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Verify the response body contains expected JSON
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}
