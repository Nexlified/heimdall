package handlers

import (
	"encoding/json"
	"io"
	"net/http"

	effectv1 "github.com/cerbos/cerbos/api/genpb/cerbos/effect/v1"
	responsev1 "github.com/cerbos/cerbos/api/genpb/cerbos/response/v1"
	"github.com/nexlified/heimdall/internal/core"
)

type HTTPHandlers struct {
	app *core.Application
}

func NewHTTPHandlers(app *core.Application) *HTTPHandlers {
	return &HTTPHandlers{app: app}
}

// HandleLogin initiates the OIDC login flow.
func (h *HTTPHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if h.app.IDP == nil {
		http.Error(w, "Identity Provider not configured", http.StatusInternalServerError)
		return
	}
	h.app.IDP.InitiateLogin(w, r)
}

// HandleAuthCallback handles the OIDC redirect.
func (h *HTTPHandlers) HandleAuthCallback(w http.ResponseWriter, r *http.Request) {
	if h.app.IDP == nil {
		http.Error(w, "Identity Provider not configured", http.StatusInternalServerError)
		return
	}

	tokenResponse, err := h.app.IDP.HandleAuthCallback(r)
	if err != nil {
		http.Error(w, "Failed to handle callback: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// For now, just write the token response as JSON.
	// In a real app, you'd set a cookie or return it to a mobile client.
	writeJSON(w, http.StatusOK, tokenResponse)
}

// HandleRefreshToken provides a new access token.
func (h *HTTPHandlers) HandleRefreshToken(w http.ResponseWriter, r *http.Request) {
	// TODO: Extract refresh token from request
	refreshToken := ""

	tokenResponse, err := h.app.IDP.RefreshToken(refreshToken)
	if err != nil {
		http.Error(w, "Failed to refresh token: "+err.Error(), http.StatusUnauthorized)
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse)
}

// HandleCheck is the core AuthZ PEP endpoint for the API Gateway.
func (h *HTTPHandlers) HandleCheck(w http.ResponseWriter, r *http.Request) {
	if h.app.PDP == nil {
		http.Error(w, "Policy Engine not configured", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// We proxy the request/response directly to/from the policy engine.
	// This keeps Heimdall stateless and fast.
	resp, err := h.app.PDP.Check(r.Context(), body)
	if err != nil {
		http.Error(w, "Policy check failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Analyze the Cerbos response to determine if the request should be allowed or denied.
	// Return 200 OK if all resources are EFFECT_ALLOW, or 403 Forbidden if any are EFFECT_DENY.
	statusCode, err := analyzeCheckResponse(resp)
	if err != nil {
		http.Error(w, "Failed to analyze policy response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(resp)
}

// HandlePlanResources is the core AuthZ PEP endpoint for list filtering.
func (h *HTTPHandlers) HandlePlanResources(w http.ResponseWriter, r *http.Request) {
	if h.app.PDP == nil {
		http.Error(w, "Policy Engine not configured", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.app.PDP.PlanResources(r.Context(), body)
	if err != nil {
		http.Error(w, "Policy plan failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// writeJSON is a simple helper
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	// In a real app, you'd use a more robust JSON encoder
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		// Quick and dirty JSON for example purposes
		b, _ := io.ReadAll(nil) // Placeholder for actual JSON marshalling
		w.Write(b)
	}
}

// analyzeCheckResponse parses a Cerbos CheckResourcesResponse and determines
// the appropriate HTTP status code based on the authorization decision.
// Returns 200 OK if all resources/actions are EFFECT_ALLOW, or 403 Forbidden if any are EFFECT_DENY.
func analyzeCheckResponse(respBytes []byte) (int, error) {
	var checkResp responsev1.CheckResourcesResponse
	if err := json.Unmarshal(respBytes, &checkResp); err != nil {
		return 0, err
	}

	// Iterate through all results and check their action effects
	for _, result := range checkResp.Results {
		if result == nil {
			continue
		}

		// Check each action in the result
		for _, effect := range result.Actions {
			// If any action has EFFECT_DENY, return 403 Forbidden
			if effect == effectv1.Effect_EFFECT_DENY {
				return http.StatusForbidden, nil
			}
		}
	}

	// All actions are allowed (or there are no results)
	return http.StatusOK, nil
}
