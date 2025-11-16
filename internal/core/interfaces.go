package core

import (
	"context"
	"net/http"
)

// Application holds the core dependencies for the HTTP handlers.
// It is built from interfaces, not concrete types.
type Application struct {
	IDP IdentityProvider
	PDP PolicyEngine
}

// TokenResponse is the standard token format returned by Heimdall.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in"`
}

// IdentityProvider defines the contract for an Authentication (AuthN) backend.
// It abstracts the "who" of the system (e.g., Ory Kratos).
type IdentityProvider interface {
	// InitiateLogin begins the OIDC/SSO login flow, redirecting the user.
	InitiateLogin(w http.ResponseWriter, r *http.Request)
	
	// HandleAuthCallback processes the OIDC callback, exchanges the code for
	// internal tokens, and mints a new Heimdall PASETO token.
	HandleAuthCallback(r *http.Request) (*TokenResponse, error)
	
	// RefreshToken exchanges a Heimdall refresh token for a new access token.
	RefreshToken(refreshToken string) (*TokenResponse, error)
}

// PolicyEngine defines the contract for an Authorization (AuthZ) backend.
// It abstracts the "what" of the system (e.g., Cerbos).
type PolicyEngine interface {
	// Check performs a simple "allow/deny" check for a request.
	// The request payload should be the raw bytes of the Cerbos CheckResource API.
	Check(ctx context.Context, checkRequestbyte) (byte, error)
	
	// PlanResources returns a query plan for list filtering (The "N+1" solution).
	// The request payload is the raw Cerbos PlanResources API.
	PlanResources(ctx context.Context, planRequestbyte) (byte, error)
	
	// UpdateAttributes updates the attributes for a principal (e.g., from an event).
	// This is called by the EventConsumer.
	UpdateAttributes(ctx context.Context, principalID string, attributes map[string]any) error
}

// EventConsumer defines the contract for an event-driven data synchronizer.
// It abstracts the message bus (e.g., NATS, Kafka).
type EventConsumer interface {
	// Consume starts a blocking listener that receives events (e.g., from NATS)
	// and uses the PolicyEngine to update attributes.
	Consume(pdp PolicyEngine) error
}
