package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/nexlified/heimdall/internal/core"
	"github.com/nexlified/heimdall/internal/handlers"
	// TODO: Import real plugin implementations once created
	// "github.com/nexlified/heimdall/internal/plugins/authn/kratos"
	// "github.com/nexlified/heimdall/internal/plugins/authz/cerbos"
	// "github.com/nexlified/heimdall/internal/plugins/events/nats"
)

func main() {
	// --- Configuration ---
	// In a real app, load this from env vars or a config file
	_ = os.Getenv("NATS_URL") // natsURL - will be used when EventConsumer is implemented
	//... other config (Kratos URL, Cerbos URL, PASETO Key)

	// --- Pluggable Dependencies ---
	// Initialize the concrete implementations of our interfaces.
	// For now, they are nil. Issues will be created to build these.
	var idp core.IdentityProvider
	var pdp core.PolicyEngine
	var consumer core.EventConsumer

	// Example of real initialization (once plugins are built)
	// idp = kratos.NewKratosClient(os.Getenv("KRATOS_ADMIN_URL"), os.Getenv("HYDRA_ADMIN_URL"))
	// pdp = cerbos.NewCerbosClient(os.Getenv("CERBOS_GRPC_URL"))
	// consumer = nats.NewNATSConsumer(natsURL)

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
		log.Println(" EventConsumer is nil. No events will be processed.")
	}

	// --- HTTP Server & Routes ---
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Initialize handlers, passing the application core
	h := handlers.NewHTTPHandlers(app)

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
