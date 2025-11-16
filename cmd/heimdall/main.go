package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nexlified/heimdall/internal/core"
	"github.com/nexlified/heimdall/internal/handlers"
	"github.com/nexlified/heimdall/internal/plugins/authn/kratos"
	"github.com/nexlified/heimdall/internal/plugins/authz/cerbos"
	"github.com/nexlified/heimdall/internal/plugins/events/nats"
	"github.com/nexlified/heimdall/internal/tokens"
)

func main() {
	// --- Configuration ---
	// Load configuration from environment variables
	kratosAdminURL := os.Getenv("KRATOS_ADMIN_URL")
	if kratosAdminURL == "" {
		kratosAdminURL = "http://localhost:4434" // Default for local development
	}

	hydraAdminURL := os.Getenv("HYDRA_ADMIN_URL")
	if hydraAdminURL == "" {
		hydraAdminURL = "http://localhost:4445" // Default for local development
	}

	hydraPublicURL := os.Getenv("HYDRA_PUBLIC_URL")
	if hydraPublicURL == "" {
		hydraPublicURL = "http://localhost:4444" // Default for local development
	}

	cerbosGRPCURL := os.Getenv("CERBOS_GRPC_URL")
	if cerbosGRPCURL == "" {
		cerbosGRPCURL = "localhost:3593" // Default for local development
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222" // Default for local development
	}

	pasetoKey := os.Getenv("PASETO_SYMMETRIC_KEY")
	if pasetoKey == "" {
		log.Fatal("PASETO_SYMMETRIC_KEY environment variable is required (must be exactly 32 bytes)")
	}

	// --- Initialize Token Service ---
	// TokenService is used by Kratos client to mint PASETO tokens
	tokenService, err := tokens.NewTokenService(pasetoKey)
	if err != nil {
		log.Fatalf("Failed to initialize token service: %v", err)
	}

	// --- Pluggable Dependencies ---
	// Initialize the concrete implementations of our interfaces.

	// Initialize Identity Provider (Kratos/Hydra)
	idp, err := kratos.NewKratosClient(&kratos.Config{
		KratosAdminURL: kratosAdminURL,
		HydraAdminURL:  hydraAdminURL,
		HydraPublicURL: hydraPublicURL,
		TokenService:   tokenService,
	})
	if err != nil {
		log.Fatalf("Failed to initialize Kratos client: %v", err)
	}

	// Initialize Policy Decision Point (Cerbos)
	pdp, err := cerbos.NewCerbosClient(cerbosGRPCURL)
	if err != nil {
		log.Fatalf("Failed to initialize Cerbos client: %v", err)
	}

	// Initialize NATS Event Consumer
	consumer, err := nats.NewNATSConsumer(natsURL)
	if err != nil {
		log.Printf("Warning: Failed to initialize NATS consumer: %v. Event processing will be disabled.", err)
		consumer = nil
	}

	// --- Core Application ---
	// The core app struct holds its dependencies (the interfaces)
	app := &core.Application{
		IDP: idp,
		PDP: pdp,
	}

	// --- Event Consumer ---
	// Start the event consumer in a separate goroutine
	// It needs a reference to the PolicyEngine to update attributes
	if consumer != nil {
		go func() {
			log.Println("Starting event consumer...")
			if err := consumer.Consume(pdp); err != nil {
				log.Fatalf("Event consumer failed: %v", err)
			}
		}()
	} else {
		log.Println("Warning: EventConsumer is not configured. No events will be processed.")
	}

	// --- HTTP Server & Routes ---
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Initialize handlers, passing the application core
	h := handlers.NewHTTPHandlers(app)

	// Health check endpoint
	r.Get("/health", h.HandleHealth)

	// Public routes for authentication
	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", h.HandleLogin)
		r.Get("/callback", h.HandleAuthCallback)
		r.Post("/refresh", h.HandleRefreshToken)
	})

	// --- API Gateway / PEP Routes ---
	// These endpoints are called by the API Gateway to make auth decisions
	r.Post("/check", h.HandleCheck)
	r.Post("/plan/resources", h.HandlePlanResources)

	// TODO: Add a /consent route for Ory Hydra

	log.Println("Starting Heimdall server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
