package nats

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/nexlified/heimdall/internal/core"
)

// Consumer is the concrete implementation of core.EventConsumer
// using NATS JetStream for event-driven data synchronization.
type Consumer struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// NewNATSConsumer initializes a new NATS consumer with the provided NATS URL.
// It connects to NATS and creates a JetStream context for subscribing to events.
func NewNATSConsumer(natsURL string) (*Consumer, error) {
	if natsURL == "" {
		return nil, errors.New("natsURL is required")
	}

	// Connect to NATS
	conn, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Get JetStream context
	js, err := conn.JetStream()
	if err != nil {
		// Close connection on error
		conn.Close()
		return nil, fmt.Errorf("failed to get JetStream context: %w", err)
	}

	return &Consumer{
		conn: conn,
		js:   js,
	}, nil
}

// Close closes the NATS connection
func (c *Consumer) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// SubscriptionUpdatedEvent represents the event structure for subscription updates
// from the Billing App
type SubscriptionUpdatedEvent struct {
	UserID     string                 `json:"user_id"`
	Plan       string                 `json:"plan"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// UsageUpdatedEvent represents the event structure for usage updates
// from the Business App
type UsageUpdatedEvent struct {
	UserID     string                 `json:"user_id"`
	Attributes map[string]interface{} `json:"attributes"`
}

// Consume starts a blocking listener that receives events from NATS JetStream
// and uses the PolicyEngine to update principal attributes.
// It subscribes to subscription.updated and usage.updated subjects.
func (c *Consumer) Consume(pdp core.PolicyEngine) error {
	if pdp == nil {
		return errors.New("PolicyEngine is required")
	}

	// Subscribe to subscription.updated events
	subUpdateSub, err := c.js.Subscribe("subscription.updated", func(msg *nats.Msg) {
		// Parse the event message
		var event SubscriptionUpdatedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("ERROR: Failed to parse subscription.updated event: %v", err)
			// Acknowledge the message even on parse error to prevent redelivery
			msg.Ack()
			return
		}

		// Build the attributes map
		attributes := make(map[string]any)
		if event.Plan != "" {
			attributes["plan"] = event.Plan
		}
		// Merge any additional attributes from the event
		for k, v := range event.Attributes {
			attributes[k] = v
		}

		// Update the PolicyEngine with the new attributes
		ctx := context.Background()
		if err := pdp.UpdateAttributes(ctx, event.UserID, attributes); err != nil {
			log.Printf("ERROR: Failed to update attributes for user %s: %v", event.UserID, err)
			// Don't acknowledge on error to allow retry
			return
		}

		log.Printf("INFO: Updated attributes for user %s with plan %s", event.UserID, event.Plan)
		// Acknowledge the message after successful processing
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to subscription.updated: %w", err)
	}
	defer subUpdateSub.Unsubscribe()

	// Subscribe to usage.updated events
	usageUpdateSub, err := c.js.Subscribe("usage.updated", func(msg *nats.Msg) {
		// Parse the event message
		var event UsageUpdatedEvent
		if err := json.Unmarshal(msg.Data, &event); err != nil {
			log.Printf("ERROR: Failed to parse usage.updated event: %v", err)
			// Acknowledge the message even on parse error to prevent redelivery
			msg.Ack()
			return
		}

		// Update the PolicyEngine with the new attributes
		ctx := context.Background()
		if err := pdp.UpdateAttributes(ctx, event.UserID, event.Attributes); err != nil {
			log.Printf("ERROR: Failed to update attributes for user %s: %v", event.UserID, err)
			// Don't acknowledge on error to allow retry
			return
		}

		log.Printf("INFO: Updated usage attributes for user %s", event.UserID)
		// Acknowledge the message after successful processing
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to usage.updated: %w", err)
	}
	defer usageUpdateSub.Unsubscribe()

	// Block forever, keeping the subscriptions active
	// In a production environment, this would be controlled by a context or signal
	log.Println("INFO: EventConsumer is listening for events...")
	select {}
}
