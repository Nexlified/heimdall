package handlers

import (
	"io"
	"net/http"

	"github.com/your-org/heimdall/internal/core"
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
	if err!= nil {
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
	if err!= nil {
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
	if err!= nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// We proxy the request/response directly to/from the policy engine.
	// This keeps Heimdall stateless and fast.
	resp, err := h.app.PDP.Check(r.Context(), body)
	if err!= nil {
		http.Error(w, "Policy check failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	
	// TODO: Analyze the 'resp' (which is a Cerbos response)
	// and return a simple 200 OK or 403 Forbidden.
	// For now, just proxy the response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

// HandlePlanResources is the core AuthZ PEP endpoint for list filtering.
func (h *HTTPHandlers) HandlePlanResources(w http.ResponseWriter, r *http.Request) {
	if h.app.PDP == nil {
		http.Error(w, "Policy Engine not configured", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err!= nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	resp, err := h.app.PDP.PlanResources(r.Context(), body)
	if err!= nil {
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
	if data!= nil {
		// Quick and dirty JSON for example purposes
		b, _ := io.ReadAll(nil) // Placeholder for actual JSON marshalling
		w.Write(b)
	}
}
