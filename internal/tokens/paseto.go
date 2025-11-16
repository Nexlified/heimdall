package tokens

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/nexlified/heimdall/internal/core"
)

// TokenService is responsible for minting and validating Heimdall's internal PASETO tokens.
// It uses PASETO v4.local (symmetric encrypted) tokens for secure, stateless authentication.
type TokenService struct {
	key paseto.V4SymmetricKey
}

// NewTokenService creates a new TokenService with the provided symmetric key.
// The key must be exactly 32 bytes (256 bits) for PASETO v4.local encryption.
// Returns an error if the key length is invalid.
func NewTokenService(key string) (*TokenService, error) {
	keyBytes := []byte(key)

	// PASETO v4.local requires a 32-byte (256-bit) symmetric key
	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("invalid key length: expected 32 bytes, got %d bytes", len(keyBytes))
	}

	// Create PASETO v4 symmetric key from the provided bytes
	symmetricKey, err := paseto.V4SymmetricKeyFromBytes(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create symmetric key: %w", err)
	}

	return &TokenService{
		key: symmetricKey,
	}, nil
}

// GenerateToken creates a new PASETO v4.local token with the given principal ID and attributes.
// The token will expire after the specified duration.
// Returns a TokenResponse containing the access token and expiration time.
func (ts *TokenService) GenerateToken(principalID string, attributes map[string]interface{}, duration time.Duration) (*core.TokenResponse, error) {
	if principalID == "" {
		return nil, errors.New("principalID cannot be empty")
	}

	if duration <= 0 {
		return nil, errors.New("duration must be positive")
	}

	// Create a new PASETO v4 token
	token := paseto.NewToken()

	// Set the subject (principal ID)
	token.SetSubject(principalID)

	// Set issued at time
	token.SetIssuedAt(time.Now())

	// Set expiration time
	expiresAt := time.Now().Add(duration)
	token.SetExpiration(expiresAt)

	// Set a unique token ID (jti) for token revocation/tracking if needed
	jti := make([]byte, 16)
	if _, err := rand.Read(jti); err != nil {
		return nil, fmt.Errorf("failed to generate token ID: %w", err)
	}
	token.SetJti(fmt.Sprintf("%x", jti))

	// Set custom attributes as claims
	if attributes != nil {
		for key, value := range attributes {
			if err := token.Set(key, value); err != nil {
				return nil, fmt.Errorf("failed to set claim %q: %w", key, err)
			}
		}
	}

	// Encrypt the token with the symmetric key
	encrypted := token.V4Encrypt(ts.key, nil)

	// Calculate the expiration time in seconds (for the expires_in field)
	expiresIn := int64(duration.Seconds())

	return &core.TokenResponse{
		AccessToken: encrypted,
		ExpiresIn:   expiresIn,
	}, nil
}

// ValidateToken parses and validates a PASETO v4.local token.
// It checks the token signature, expiration time, and other standard claims.
// Returns the parsed token if valid, or an error if validation fails.
func (ts *TokenService) ValidateToken(tokenString string) (*paseto.Token, error) {
	if tokenString == "" {
		return nil, errors.New("token cannot be empty")
	}

	// Create a parser with validation rules
	parser := paseto.NewParser()

	// Add validation rule for expiration time
	parser.AddRule(paseto.NotExpired())

	// Parse and decrypt the token
	token, err := parser.ParseV4Local(ts.key, tokenString, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token, nil
}
