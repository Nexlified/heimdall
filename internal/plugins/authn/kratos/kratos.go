package kratos

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nexlified/heimdall/internal/core"
	"github.com/nexlified/heimdall/internal/tokens"
	hydra "github.com/ory/hydra-client-go/v2"
	kratos "github.com/ory/kratos-client-go"
)

// Client is the concrete implementation of core.IdentityProvider
// using Ory Kratos for identity management and Ory Hydra for OAuth2/OIDC flows.
type Client struct {
	kratosClient   *kratos.APIClient
	hydraClient    *hydra.APIClient
	tokenService   *tokens.TokenService
	hydraPublicURL string
}

// Config holds the configuration for creating a new Kratos/Hydra client.
type Config struct {
	KratosAdminURL string
	HydraAdminURL  string
	HydraPublicURL string
	TokenService   *tokens.TokenService
}

// NewKratosClient initializes a new Kratos/Hydra client with the provided configuration.
// It creates API clients for both Kratos (identity management) and Hydra (OAuth2/OIDC).
func NewKratosClient(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, errors.New("config cannot be nil")
	}

	if cfg.KratosAdminURL == "" {
		return nil, errors.New("KratosAdminURL is required")
	}

	if cfg.HydraAdminURL == "" {
		return nil, errors.New("HydraAdminURL is required")
	}

	if cfg.HydraPublicURL == "" {
		return nil, errors.New("HydraPublicURL is required")
	}

	if cfg.TokenService == nil {
		return nil, errors.New("TokenService is required")
	}

	// Initialize Kratos API client
	kratosConfig := kratos.NewConfiguration()
	kratosConfig.Servers = kratos.ServerConfigurations{
		{
			URL: cfg.KratosAdminURL,
		},
	}
	kratosClient := kratos.NewAPIClient(kratosConfig)

	// Initialize Hydra API client
	hydraConfig := hydra.NewConfiguration()
	hydraConfig.Servers = hydra.ServerConfigurations{
		{
			URL: cfg.HydraAdminURL,
		},
	}
	hydraClient := hydra.NewAPIClient(hydraConfig)

	return &Client{
		kratosClient:   kratosClient,
		hydraClient:    hydraClient,
		tokenService:   cfg.TokenService,
		hydraPublicURL: cfg.HydraPublicURL,
	}, nil
}

// InitiateLogin begins the OIDC login flow by creating a new OAuth2 authorization request
// via Hydra and redirecting the user to the login URL.
func (c *Client) InitiateLogin(w http.ResponseWriter, r *http.Request) {
	// Create a new OAuth2 login request via Hydra admin API
	ctx := r.Context()

	// In a production setup, you would create an OAuth2 authorization code flow request
	// For now, we'll redirect to Hydra's public OAuth2 authorization endpoint
	// The actual flow would involve:
	// 1. Creating a login challenge
	// 2. Redirecting to Hydra's login endpoint
	// 3. Hydra redirecting to Kratos for authentication
	// 4. User authenticating with Kratos
	// 5. Kratos redirecting back to Hydra
	// 6. Hydra redirecting to our callback with an authorization code

	// Construct the OAuth2 authorization URL
	// This is a simplified version - in production, you'd need to:
	// - Generate a state parameter for CSRF protection
	// - Define the scope
	// - Set the redirect_uri to our callback endpoint
	loginURL := fmt.Sprintf("%s/oauth2/auth?client_id=heimdall&response_type=code&scope=openid&redirect_uri=%s",
		c.hydraPublicURL,
		"http://localhost:8080/auth/callback") // This should come from config

	// Log the initiation (in production, use proper logging)
	_ = ctx // Use context if needed for logging/tracing

	// Redirect the user to the OAuth2 authorization endpoint
	http.Redirect(w, r, loginURL, http.StatusFound)
}

// HandleAuthCallback processes the OIDC callback after user authentication.
// It exchanges the authorization code for tokens, fetches the user's identity from Kratos,
// and mints a new Heimdall PASETO token.
func (c *Client) HandleAuthCallback(r *http.Request) (*core.TokenResponse, error) {
	ctx := r.Context()

	// Extract the authorization code from the query parameters
	code := r.URL.Query().Get("code")
	if code == "" {
		return nil, errors.New("missing authorization code")
	}

	// Exchange the authorization code for tokens using Hydra
	tokenResponse, err := c.exchangeCodeForTokens(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for tokens: %w", err)
	}

	// Extract the subject (user ID) from the ID token
	subject, err := c.extractSubjectFromIDToken(tokenResponse.IDToken)
	if err != nil {
		return nil, fmt.Errorf("failed to extract subject from ID token: %w", err)
	}

	// Fetch the user's identity (traits) from Kratos
	identity, err := c.getIdentityFromKratos(ctx, subject)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch identity from Kratos: %w", err)
	}

	// Convert Kratos traits to a map for our token
	attributes := make(map[string]interface{})
	if identity.Traits != nil {
		// Kratos traits is an interface{}, typically a map
		if traitsMap, ok := identity.Traits.(map[string]interface{}); ok {
			attributes = traitsMap
		}
	}

	// Mint a new Heimdall PASETO token
	// Use a 1-hour expiration for access tokens
	heimdallToken, err := c.tokenService.GenerateToken(subject, attributes, 1*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("failed to generate Heimdall token: %w", err)
	}

	return heimdallToken, nil
}

// RefreshToken exchanges a Heimdall refresh token for a new access token.
// This is a stub implementation - full refresh token support would require
// storing refresh tokens and implementing proper rotation.
func (c *Client) RefreshToken(refreshToken string) (*core.TokenResponse, error) {
	if refreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	// TODO: Implement refresh token logic
	// This would involve:
	// 1. Validating the refresh token
	// 2. Looking up the associated user
	// 3. Checking if the refresh token is still valid
	// 4. Generating a new access token
	// 5. Optionally rotating the refresh token

	return nil, errors.New("refresh token not implemented yet")
}

// exchangeCodeForTokens exchanges an authorization code for OAuth2 tokens via Hydra.
// This is a helper method that makes the token exchange request to Hydra's public API.
func (c *Client) exchangeCodeForTokens(ctx context.Context, code string) (*OAuth2TokenResponse, error) {
	// In a real implementation, we would use Hydra's OAuth2 token endpoint
	// to exchange the code for tokens. For now, this is a simplified version.

	// The actual implementation would look like:
	// req := c.hydraClient.OAuth2API.Oauth2Token(ctx)
	// req = req.GrantType("authorization_code")
	// req = req.Code(code)
	// req = req.RedirectUri("http://localhost:8080/auth/callback")
	// req = req.ClientId("heimdall")
	// tokenResp, _, err := req.Execute()

	// For now, we'll return a mock response to illustrate the structure
	// In production, this would be replaced with the actual Hydra API call
	return nil, errors.New("exchangeCodeForTokens requires actual Hydra integration")
}

// extractSubjectFromIDToken extracts the subject (user ID) from an OIDC ID token.
// This parses the JWT token (without verification, as it comes from our trusted Hydra instance)
// and extracts the 'sub' claim.
func (c *Client) extractSubjectFromIDToken(idToken string) (string, error) {
	if idToken == "" {
		return "", errors.New("ID token is empty")
	}

	// Parse JWT token without verification (since it comes from our trusted Hydra instance)
	// JWT format: header.payload.signature
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return "", errors.New("invalid JWT token format")
	}

	// Decode the payload (second part)
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("failed to decode JWT payload: %w", err)
	}

	// Parse the payload JSON
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", fmt.Errorf("failed to unmarshal JWT claims: %w", err)
	}

	// Extract the subject claim
	sub, ok := claims["sub"].(string)
	if !ok || sub == "" {
		return "", errors.New("subject claim not found in ID token")
	}

	return sub, nil
}

// getIdentityFromKratos fetches a user's identity and traits from Kratos.
func (c *Client) getIdentityFromKratos(ctx context.Context, userID string) (*kratos.Identity, error) {
	if userID == "" {
		return nil, errors.New("userID is required")
	}

	// Call Kratos Admin API to get the identity
	identity, resp, err := c.kratosClient.IdentityAPI.GetIdentity(ctx, userID).Execute()
	if err != nil {
		return nil, fmt.Errorf("Kratos API error: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from Kratos: %d", resp.StatusCode)
	}

	return identity, nil
}

// OAuth2TokenResponse represents the response from an OAuth2 token exchange.
// This is a helper struct for the token exchange flow.
type OAuth2TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token"`
	Scope        string `json:"scope,omitempty"`
}
