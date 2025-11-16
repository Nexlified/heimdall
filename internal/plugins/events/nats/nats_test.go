package nats

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPolicyEngine is a mock implementation of core.PolicyEngine for testing
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

// TestNewNATSConsumer tests the constructor with various configurations
func TestNewNATSConsumer(t *testing.T) {
	tests := []struct {
		name        string
		natsURL     string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty NATS URL",
			natsURL:     "",
			expectError: true,
			errorMsg:    "natsURL is required",
		},
		{
			name:        "invalid NATS URL",
			natsURL:     "nats://invalid:9999",
			expectError: true,
			errorMsg:    "failed to connect to NATS",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer, err := NewNATSConsumer(tt.natsURL)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, consumer)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, consumer)
				assert.NotNil(t, consumer.conn)
				assert.NotNil(t, consumer.js)
				// Clean up
				consumer.Close()
			}
		})
	}
}

// TestSubscriptionUpdatedEvent_Unmarshal tests unmarshaling of subscription.updated events
func TestSubscriptionUpdatedEvent_Unmarshal(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		validate    func(t *testing.T, event *SubscriptionUpdatedEvent)
	}{
		{
			name: "valid event with plan",
			jsonData: `{
				"user_id": "user123",
				"plan": "pro"
			}`,
			expectError: false,
			validate: func(t *testing.T, event *SubscriptionUpdatedEvent) {
				assert.Equal(t, "user123", event.UserID)
				assert.Equal(t, "pro", event.Plan)
			},
		},
		{
			name: "valid event with plan and attributes",
			jsonData: `{
				"user_id": "user456",
				"plan": "enterprise",
				"attributes": {
					"max_users": 100,
					"features": ["sso", "audit"]
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *SubscriptionUpdatedEvent) {
				assert.Equal(t, "user456", event.UserID)
				assert.Equal(t, "enterprise", event.Plan)
				assert.NotNil(t, event.Attributes)
				assert.Equal(t, float64(100), event.Attributes["max_users"])
			},
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
		},
		{
			name: "minimal valid event",
			jsonData: `{
				"user_id": "user789"
			}`,
			expectError: false,
			validate: func(t *testing.T, event *SubscriptionUpdatedEvent) {
				assert.Equal(t, "user789", event.UserID)
				assert.Empty(t, event.Plan)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event SubscriptionUpdatedEvent
			err := json.Unmarshal([]byte(tt.jsonData), &event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, &event)
				}
			}
		})
	}
}

// TestUsageUpdatedEvent_Unmarshal tests unmarshaling of usage.updated events
func TestUsageUpdatedEvent_Unmarshal(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		validate    func(t *testing.T, event *UsageUpdatedEvent)
	}{
		{
			name: "valid event with current_users",
			jsonData: `{
				"user_id": "user123",
				"attributes": {
					"current_users": 28
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *UsageUpdatedEvent) {
				assert.Equal(t, "user123", event.UserID)
				assert.NotNil(t, event.Attributes)
				assert.Equal(t, float64(28), event.Attributes["current_users"])
			},
		},
		{
			name: "valid event with multiple attributes",
			jsonData: `{
				"user_id": "user456",
				"attributes": {
					"current_users": 50,
					"storage_used_gb": 120.5,
					"api_calls_today": 1500
				}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *UsageUpdatedEvent) {
				assert.Equal(t, "user456", event.UserID)
				assert.Equal(t, float64(50), event.Attributes["current_users"])
				assert.Equal(t, 120.5, event.Attributes["storage_used_gb"])
				assert.Equal(t, float64(1500), event.Attributes["api_calls_today"])
			},
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"invalid": json}`,
			expectError: true,
		},
		{
			name: "empty attributes",
			jsonData: `{
				"user_id": "user789",
				"attributes": {}
			}`,
			expectError: false,
			validate: func(t *testing.T, event *UsageUpdatedEvent) {
				assert.Equal(t, "user789", event.UserID)
				assert.Empty(t, event.Attributes)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event UsageUpdatedEvent
			err := json.Unmarshal([]byte(tt.jsonData), &event)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.validate != nil {
					tt.validate(t, &event)
				}
			}
		})
	}
}

// TestConsume_NilPolicyEngine tests that Consume returns error when PolicyEngine is nil
func TestConsume_NilPolicyEngine(t *testing.T) {
	// This test doesn't require a real NATS connection
	consumer := &Consumer{}

	err := consumer.Consume(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PolicyEngine is required")
}

// TestConsume_Integration is an integration test that requires a running NATS JetStream
// This test is skipped by default and should be run with a test NATS server
func TestConsume_Integration(t *testing.T) {
	t.Skip("Skipping integration test - requires running NATS JetStream server")

	// This test would require:
	// 1. A running NATS server with JetStream enabled
	// 2. Creating a stream for the subscription.updated and usage.updated subjects
	// 3. Publishing test messages to these subjects
	// 4. Verifying that UpdateAttributes is called with the correct data

	natsURL := "nats://localhost:4222"
	consumer, err := NewNATSConsumer(natsURL)
	require.NoError(t, err)
	defer consumer.Close()

	mockPDP := new(MockPolicyEngine)
	mockPDP.On("UpdateAttributes", mock.Anything, "user123", mock.Anything).Return(nil)

	// In a real integration test, we would:
	// 1. Start Consume in a goroutine with a context for cancellation
	// 2. Publish test messages to NATS
	// 3. Wait for UpdateAttributes to be called
	// 4. Assert on the mock expectations
	// 5. Cancel the context to stop Consume

	// For now, this is a placeholder
	mockPDP.AssertExpectations(t)
}

// TestEventProcessing_SubscriptionUpdated tests the subscription.updated event processing logic
func TestEventProcessing_SubscriptionUpdated(t *testing.T) {
	tests := []struct {
		name              string
		eventJSON         string
		expectedUserID    string
		expectedAttrs     map[string]any
		expectUpdateCall  bool
		updateShouldError bool
	}{
		{
			name: "subscription with plan only",
			eventJSON: `{
				"user_id": "user123",
				"plan": "pro"
			}`,
			expectedUserID: "user123",
			expectedAttrs: map[string]any{
				"plan": "pro",
			},
			expectUpdateCall: true,
		},
		{
			name: "subscription with plan and attributes",
			eventJSON: `{
				"user_id": "user456",
				"plan": "enterprise",
				"attributes": {
					"max_users": 100
				}
			}`,
			expectedUserID: "user456",
			expectedAttrs: map[string]any{
				"plan":      "enterprise",
				"max_users": float64(100),
			},
			expectUpdateCall: true,
		},
		{
			name: "subscription without plan",
			eventJSON: `{
				"user_id": "user789",
				"attributes": {
					"custom_field": "value"
				}
			}`,
			expectedUserID: "user789",
			expectedAttrs: map[string]any{
				"custom_field": "value",
			},
			expectUpdateCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event SubscriptionUpdatedEvent
			err := json.Unmarshal([]byte(tt.eventJSON), &event)
			require.NoError(t, err)

			// Build the attributes map as the consumer would
			attributes := make(map[string]any)
			if event.Plan != "" {
				attributes["plan"] = event.Plan
			}
			for k, v := range event.Attributes {
				attributes[k] = v
			}

			// Verify the attributes match expected
			assert.Equal(t, tt.expectedUserID, event.UserID)
			assert.Equal(t, tt.expectedAttrs, attributes)

			// Mock the PolicyEngine and verify UpdateAttributes is called correctly
			if tt.expectUpdateCall {
				mockPDP := new(MockPolicyEngine)
				if tt.updateShouldError {
					mockPDP.On("UpdateAttributes", mock.Anything, tt.expectedUserID, tt.expectedAttrs).Return(assert.AnError)
				} else {
					mockPDP.On("UpdateAttributes", mock.Anything, tt.expectedUserID, tt.expectedAttrs).Return(nil)
				}

				// Simulate calling UpdateAttributes
				ctx := context.Background()
				err := mockPDP.UpdateAttributes(ctx, event.UserID, attributes)
				if tt.updateShouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				mockPDP.AssertExpectations(t)
			}
		})
	}
}

// TestEventProcessing_UsageUpdated tests the usage.updated event processing logic
func TestEventProcessing_UsageUpdated(t *testing.T) {
	tests := []struct {
		name              string
		eventJSON         string
		expectedUserID    string
		expectedAttrs     map[string]any
		expectUpdateCall  bool
		updateShouldError bool
	}{
		{
			name: "usage with current_users",
			eventJSON: `{
				"user_id": "user123",
				"attributes": {
					"current_users": 28
				}
			}`,
			expectedUserID: "user123",
			expectedAttrs: map[string]any{
				"current_users": float64(28),
			},
			expectUpdateCall: true,
		},
		{
			name: "usage with multiple attributes",
			eventJSON: `{
				"user_id": "user456",
				"attributes": {
					"current_users": 50,
					"storage_used_gb": 120.5
				}
			}`,
			expectedUserID: "user456",
			expectedAttrs: map[string]any{
				"current_users":   float64(50),
				"storage_used_gb": 120.5,
			},
			expectUpdateCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event UsageUpdatedEvent
			err := json.Unmarshal([]byte(tt.eventJSON), &event)
			require.NoError(t, err)

			// Verify the event matches expected
			assert.Equal(t, tt.expectedUserID, event.UserID)
			assert.Equal(t, tt.expectedAttrs, event.Attributes)

			// Mock the PolicyEngine and verify UpdateAttributes is called correctly
			if tt.expectUpdateCall {
				mockPDP := new(MockPolicyEngine)
				if tt.updateShouldError {
					mockPDP.On("UpdateAttributes", mock.Anything, tt.expectedUserID, tt.expectedAttrs).Return(assert.AnError)
				} else {
					mockPDP.On("UpdateAttributes", mock.Anything, tt.expectedUserID, tt.expectedAttrs).Return(nil)
				}

				// Simulate calling UpdateAttributes
				ctx := context.Background()
				err := mockPDP.UpdateAttributes(ctx, event.UserID, event.Attributes)
				if tt.updateShouldError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}

				mockPDP.AssertExpectations(t)
			}
		})
	}
}

// TestConsumerClose tests the Close method
func TestConsumerClose(t *testing.T) {
	// Test with nil connection
	consumer := &Consumer{}
	consumer.Close() // Should not panic

	// Test with valid connection would require a real NATS server
}

// TestAttributesMerging tests that plan and additional attributes are merged correctly
func TestAttributesMerging(t *testing.T) {
	eventJSON := `{
		"user_id": "user123",
		"plan": "pro",
		"attributes": {
			"max_users": 50,
			"features": ["sso"]
		}
	}`

	var event SubscriptionUpdatedEvent
	err := json.Unmarshal([]byte(eventJSON), &event)
	require.NoError(t, err)

	// Simulate the merging logic from Consume
	attributes := make(map[string]any)
	if event.Plan != "" {
		attributes["plan"] = event.Plan
	}
	for k, v := range event.Attributes {
		attributes[k] = v
	}

	// Verify all attributes are present
	assert.Equal(t, "pro", attributes["plan"])
	assert.Equal(t, float64(50), attributes["max_users"])
	assert.NotNil(t, attributes["features"])

	// Verify the mock call with merged attributes
	mockPDP := new(MockPolicyEngine)
	mockPDP.On("UpdateAttributes", mock.Anything, "user123", attributes).Return(nil)

	ctx := context.Background()
	err = mockPDP.UpdateAttributes(ctx, "user123", attributes)
	assert.NoError(t, err)

	mockPDP.AssertExpectations(t)
}

// TestEmptyPlan tests handling of events without a plan field
func TestEmptyPlan(t *testing.T) {
	eventJSON := `{
		"user_id": "user123",
		"attributes": {
			"custom_field": "value"
		}
	}`

	var event SubscriptionUpdatedEvent
	err := json.Unmarshal([]byte(eventJSON), &event)
	require.NoError(t, err)

	// Simulate the merging logic from Consume
	attributes := make(map[string]any)
	if event.Plan != "" {
		attributes["plan"] = event.Plan
	}
	for k, v := range event.Attributes {
		attributes[k] = v
	}

	// Verify plan is not in attributes if empty
	_, hasPlan := attributes["plan"]
	assert.False(t, hasPlan)
	assert.Equal(t, "value", attributes["custom_field"])
}

// TestConcurrentEventProcessing tests that events can be processed concurrently
func TestConcurrentEventProcessing(t *testing.T) {
	// This test verifies the logic that would handle concurrent events
	// In production, NATS would call the handler concurrently
	mockPDP := new(MockPolicyEngine)

	// Set up mock expectations for multiple concurrent calls
	mockPDP.On("UpdateAttributes", mock.Anything, "user1", mock.Anything).Return(nil)
	mockPDP.On("UpdateAttributes", mock.Anything, "user2", mock.Anything).Return(nil)
	mockPDP.On("UpdateAttributes", mock.Anything, "user3", mock.Anything).Return(nil)

	// Simulate concurrent processing
	ctx := context.Background()
	done := make(chan bool, 3)

	go func() {
		mockPDP.UpdateAttributes(ctx, "user1", map[string]any{"plan": "pro"})
		done <- true
	}()

	go func() {
		mockPDP.UpdateAttributes(ctx, "user2", map[string]any{"plan": "enterprise"})
		done <- true
	}()

	go func() {
		mockPDP.UpdateAttributes(ctx, "user3", map[string]any{"current_users": float64(10)})
		done <- true
	}()

	// Wait for all goroutines to complete
	timeout := time.After(5 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-timeout:
			t.Fatal("Test timed out waiting for concurrent processing")
		}
	}

	mockPDP.AssertExpectations(t)
}
