// Package services implements business logic for the miau application.
package services

import (
	"sync"

	"github.com/opik/miau/internal/ports"
)

// eventBus implements ports.EventBus
type eventBus struct {
	mu          sync.RWMutex
	handlers    map[ports.EventType][]ports.EventHandler
	allHandlers []ports.EventHandler
	nextID      int
	unsubMap    map[int]func() // maps subscription ID to cleanup function
}

// NewEventBus creates a new EventBus
func NewEventBus() ports.EventBus {
	return &eventBus{
		handlers:  make(map[ports.EventType][]ports.EventHandler),
		unsubMap:  make(map[int]func()),
	}
}

// Publish publishes an event to all subscribers
func (eb *eventBus) Publish(event ports.Event) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	// Notify type-specific handlers
	if handlers, ok := eb.handlers[event.Type()]; ok {
		for _, handler := range handlers {
			go handler(event)
		}
	}

	// Notify all-events handlers
	for _, handler := range eb.allHandlers {
		go handler(event)
	}
}

// Subscribe subscribes to events of a specific type
func (eb *eventBus) Subscribe(eventType ports.EventType, handler ports.EventHandler) func() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.handlers[eventType] = append(eb.handlers[eventType], handler)

	// Create unsubscribe function
	var id = eb.nextID
	eb.nextID++

	var unsubscribe = func() {
		eb.mu.Lock()
		defer eb.mu.Unlock()

		var handlers = eb.handlers[eventType]
		for i, h := range handlers {
			// Compare function pointers
			if &h == &handler {
				eb.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
				break
			}
		}
		delete(eb.unsubMap, id)
	}

	eb.unsubMap[id] = unsubscribe
	return unsubscribe
}

// SubscribeAll subscribes to all events
func (eb *eventBus) SubscribeAll(handler ports.EventHandler) func() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	eb.allHandlers = append(eb.allHandlers, handler)

	var id = eb.nextID
	eb.nextID++

	var unsubscribe = func() {
		eb.mu.Lock()
		defer eb.mu.Unlock()

		for i, h := range eb.allHandlers {
			if &h == &handler {
				eb.allHandlers = append(eb.allHandlers[:i], eb.allHandlers[i+1:]...)
				break
			}
		}
		delete(eb.unsubMap, id)
	}

	eb.unsubMap[id] = unsubscribe
	return unsubscribe
}
