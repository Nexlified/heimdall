package tokens

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewTokenService tests the constructor with various key lengths
func TestNewTokenService(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		expectError bool
	}{
		{
			name:        "valid 32-byte key",
			key:         "12345678901234567890123456789012",
			expectError: false,
		},
		{
			name:        "key too short",
			key:         "short",
			expectError: true,
		},
		{
			name:        "key too long",
			key:         "123456789012345678901234567890123",
			expectError: true,
		},
		{
			name:        "empty key",
			key:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := NewTokenService(tt.key)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, ts)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, ts)
			}
		})
	}
}

// TestGenerateToken tests successful token generation
func TestGenerateToken(t *testing.T) {
	// Create a token service with a valid 32-byte key
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)
	require.NotNil(t, ts)

	// Define test parameters
	principalID := "user-123"
	attributes := map[string]interface{}{
		"email": "user@example.com",
		"role":  "admin",
		"plan":  "premium",
	}
	duration := 1 * time.Hour

	// Generate a token
	tokenResponse, err := ts.GenerateToken(principalID, attributes, duration)
	require.NoError(t, err)
	require.NotNil(t, tokenResponse)

	// Verify the response structure
	assert.NotEmpty(t, tokenResponse.AccessToken)
	assert.Equal(t, int64(3600), tokenResponse.ExpiresIn)
	assert.Empty(t, tokenResponse.RefreshToken) // We're not generating refresh tokens yet

	// Verify the token starts with the correct PASETO v4.local prefix
	assert.Contains(t, tokenResponse.AccessToken, "v4.local.")
}

// TestGenerateToken_EmptyPrincipalID tests that generation fails with empty principal ID
func TestGenerateToken_EmptyPrincipalID(t *testing.T) {
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	tokenResponse, err := ts.GenerateToken("", nil, 1*time.Hour)
	assert.Error(t, err)
	assert.Nil(t, tokenResponse)
	assert.Contains(t, err.Error(), "principalID cannot be empty")
}

// TestGenerateToken_InvalidDuration tests that generation fails with invalid duration
func TestGenerateToken_InvalidDuration(t *testing.T) {
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	tokenResponse, err := ts.GenerateToken("user-123", nil, 0)
	assert.Error(t, err)
	assert.Nil(t, tokenResponse)
	assert.Contains(t, err.Error(), "duration must be positive")
}

// TestValidateToken tests successful token validation
func TestValidateToken(t *testing.T) {
	// Create a token service with a valid 32-byte key
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	// Generate a token
	principalID := "user-456"
	attributes := map[string]interface{}{
		"email": "test@example.com",
		"role":  "user",
	}
	duration := 1 * time.Hour

	tokenResponse, err := ts.GenerateToken(principalID, attributes, duration)
	require.NoError(t, err)
	require.NotNil(t, tokenResponse)

	// Validate the token
	parsedToken, err := ts.ValidateToken(tokenResponse.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, parsedToken)

	// Verify the token claims
	subject, err := parsedToken.GetSubject()
	require.NoError(t, err)
	assert.Equal(t, principalID, subject)

	// Verify custom attributes
	var email string
	err = parsedToken.Get("email", &email)
	require.NoError(t, err)
	assert.Equal(t, "test@example.com", email)

	var role string
	err = parsedToken.Get("role", &role)
	require.NoError(t, err)
	assert.Equal(t, "user", role)
}

// TestValidateToken_Expired tests that validation fails on an expired token
func TestValidateToken_Expired(t *testing.T) {
	// Create a token service with a valid 32-byte key
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	// Generate a token that expires in 1 second
	principalID := "user-789"
	duration := 1 * time.Second

	tokenResponse, err := ts.GenerateToken(principalID, nil, duration)
	require.NoError(t, err)
	require.NotNil(t, tokenResponse)

	// Wait for the token to expire
	time.Sleep(2 * time.Second)

	// Try to validate the expired token
	parsedToken, err := ts.ValidateToken(tokenResponse.AccessToken)
	assert.Error(t, err)
	assert.Nil(t, parsedToken)
	assert.Contains(t, err.Error(), "failed to parse token")
}

// TestValidateToken_WrongKey tests that validation fails with a different key
func TestValidateToken_WrongKey(t *testing.T) {
	// Create a token service with one key
	key1 := "12345678901234567890123456789012"
	ts1, err := NewTokenService(key1)
	require.NoError(t, err)

	// Generate a token
	principalID := "user-999"
	duration := 1 * time.Hour

	tokenResponse, err := ts1.GenerateToken(principalID, nil, duration)
	require.NoError(t, err)
	require.NotNil(t, tokenResponse)

	// Create a different token service with a different key
	key2 := "abcdefghijklmnopqrstuvwxyz123456"
	ts2, err := NewTokenService(key2)
	require.NoError(t, err)

	// Try to validate the token with the wrong key
	parsedToken, err := ts2.ValidateToken(tokenResponse.AccessToken)
	assert.Error(t, err)
	assert.Nil(t, parsedToken)
	assert.Contains(t, err.Error(), "failed to parse token")
}

// TestValidateToken_EmptyToken tests that validation fails with an empty token
func TestValidateToken_EmptyToken(t *testing.T) {
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	parsedToken, err := ts.ValidateToken("")
	assert.Error(t, err)
	assert.Nil(t, parsedToken)
	assert.Contains(t, err.Error(), "token cannot be empty")
}

// TestValidateToken_InvalidToken tests that validation fails with a malformed token
func TestValidateToken_InvalidToken(t *testing.T) {
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	parsedToken, err := ts.ValidateToken("not-a-valid-paseto-token")
	assert.Error(t, err)
	assert.Nil(t, parsedToken)
	assert.Contains(t, err.Error(), "failed to parse token")
}

// TestGenerateToken_WithNilAttributes tests token generation with nil attributes
func TestGenerateToken_WithNilAttributes(t *testing.T) {
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	principalID := "user-nil"
	duration := 1 * time.Hour

	tokenResponse, err := ts.GenerateToken(principalID, nil, duration)
	require.NoError(t, err)
	require.NotNil(t, tokenResponse)

	// Validate the token
	parsedToken, err := ts.ValidateToken(tokenResponse.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, parsedToken)

	subject, err := parsedToken.GetSubject()
	require.NoError(t, err)
	assert.Equal(t, principalID, subject)
}

// TestGenerateToken_WithEmptyAttributes tests token generation with empty attributes map
func TestGenerateToken_WithEmptyAttributes(t *testing.T) {
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	principalID := "user-empty"
	attributes := map[string]interface{}{}
	duration := 1 * time.Hour

	tokenResponse, err := ts.GenerateToken(principalID, attributes, duration)
	require.NoError(t, err)
	require.NotNil(t, tokenResponse)

	// Validate the token
	parsedToken, err := ts.ValidateToken(tokenResponse.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, parsedToken)

	subject, err := parsedToken.GetSubject()
	require.NoError(t, err)
	assert.Equal(t, principalID, subject)
}

// TestTokenService_RoundTrip tests a complete round trip of token generation and validation
func TestTokenService_RoundTrip(t *testing.T) {
	key := "12345678901234567890123456789012"
	ts, err := NewTokenService(key)
	require.NoError(t, err)

	principalID := "user-roundtrip"
	attributes := map[string]interface{}{
		"email":        "roundtrip@example.com",
		"subscription": "enterprise",
		"permissions":  []string{"read", "write", "admin"},
	}
	duration := 24 * time.Hour

	// Generate token
	tokenResponse, err := ts.GenerateToken(principalID, attributes, duration)
	require.NoError(t, err)
	require.NotNil(t, tokenResponse)

	// Validate token
	parsedToken, err := ts.ValidateToken(tokenResponse.AccessToken)
	require.NoError(t, err)
	require.NotNil(t, parsedToken)

	// Verify all claims
	subject, err := parsedToken.GetSubject()
	require.NoError(t, err)
	assert.Equal(t, principalID, subject)

	var email string
	err = parsedToken.Get("email", &email)
	require.NoError(t, err)
	assert.Equal(t, "roundtrip@example.com", email)

	var subscription string
	err = parsedToken.Get("subscription", &subscription)
	require.NoError(t, err)
	assert.Equal(t, "enterprise", subscription)

	var permissions []interface{}
	err = parsedToken.Get("permissions", &permissions)
	require.NoError(t, err)
	assert.Len(t, permissions, 3)
}
