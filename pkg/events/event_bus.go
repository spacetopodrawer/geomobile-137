// Package events provides event publishing and subscription
package events

import (
	"context"
	"fmt"
	"log"
	"sync"
)

// EventBus implements publish-subscribe pattern for real-time updates
type EventBus struct {
	subscribers map[string][]chan interface{}
	mu          sync.RWMutex
	logger      *log.Logger
}

// NewEventBus creates a new event bus
func NewEventBus(logger *log.Logger) *EventBus {
	if logger == nil {
		logger = log.New(log.Writer(), "[EventBus] ", log.LstdFlags)
	}
	return &EventBus{
		subscribers: make(map[string][]chan interface{}),
		logger:      logger,
	}
}

// Publish sends an event to all subscribers of a topic
func (eb *EventBus) Publish(ctx context.Context, topic string, payload interface{}) error {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	subscribers, exists := eb.subscribers[topic]
	if !exists || len(subscribers) == 0 {
		eb.logger.Printf("Published to topic '%s' (no subscribers)\n", topic)
		return nil
	}

	eb.logger.Printf("Publishing to topic '%s' (%d subscribers)\n", topic, len(subscribers))

	// Send to all subscribers (non-blocking)
	for _, ch := range subscribers {
		select {
		case ch <- payload:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Channel buffer full, skip to avoid blocking
			eb.logger.Printf("Warning: subscriber channel full for topic '%s'\n", topic)
		}
	}

	return nil
}

// Subscribe creates a subscription to a topic
// Returns a channel that receives events
func (eb *EventBus) Subscribe(ctx context.Context, topic string) (<-chan interface{}, error) {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan interface{}, 10) // Buffered channel to avoid blocking

	eb.subscribers[topic] = append(eb.subscribers[topic], ch)
	eb.logger.Printf("Subscriber added to topic '%s' (%d total)\n", topic, len(eb.subscribers[topic]))

	return ch, nil
}

// Unsubscribe removes a subscription (for WebSocket cleanup)
func (eb *EventBus) Unsubscribe(ctx context.Context, topic string, ch <-chan interface{}) error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	subscribers, exists := eb.subscribers[topic]
	if !exists {
		return fmt.Errorf("topic not found: %s", topic)
	}

	// Find and remove the channel
	// Note: In production, use a proper subscription ID instead
	for i, sub := range subscribers {
		if sub == ch {
			eb.subscribers[topic] = append(subscribers[:i], subscribers[i+1:]...)
			eb.logger.Printf("Subscriber removed from topic '%s' (%d remaining)\n", topic, len(eb.subscribers[topic]))
			return nil
		}
	}

	return fmt.Errorf("subscription not found for topic: %s", topic)
}

// GetSubscriberCount returns the number of subscribers for a topic
func (eb *EventBus) GetSubscriberCount(topic string) int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	return len(eb.subscribers[topic])
}

// Close closes the event bus and all subscriptions
func (eb *EventBus) Close() error {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	for topic, subscribers := range eb.subscribers {
		for _, ch := range subscribers {
			close(ch)
		}
		eb.logger.Printf("Closed %d subscribers for topic '%s'\n", len(subscribers), topic)
	}

	eb.subscribers = make(map[string][]chan interface{})
	return nil
}

// Event types for cadastral operations
const (
	EventCadastreEntitiesCreated = "cadastre:entities:created"
	EventCadastreEntitiesUpdated = "cadastre:entities:updated"
	EventCadastreEntitiesDeleted = "cadastre:entities:deleted"
	EventTilesRegenerated        = "cadastre:tiles:regenerated"
)

// CadastreEvent is the payload for cadastral entity events
type CadastreEvent struct {
	Type      string      `json:"type"`      // "created", "updated", "deleted"
	Code      string      `json:"code"`      // Cadastre code
	EntityID  string      `json:"entity_id"`
	Region    string      `json:"region"`
	AdminUnit string      `json:"admin_unit"`
	Timestamp string      `json:"timestamp"`
	Metadata  interface{} `json:"metadata,omitempty"`
}

// TileRegeneratedEvent is the payload for tile regeneration events
type TileRegeneratedEvent struct {
	BBox struct {
		MinX float64 `json:"min_x"`
		MinY float64 `json:"min_y"`
		MaxX float64 `json:"max_x"`
		MaxY float64 `json:"max_y"`
	} `json:"bbox"`
	AffectedZoomLevels []int  `json:"affected_zoom_levels"`
	TilesRegenerated   int    `json:"tiles_regenerated"`
	Timestamp          string `json:"timestamp"`
}
