package mocks

import (
	"github.com/opik/miau/internal/ports"
	"github.com/stretchr/testify/mock"
)

// EventBus is a mock implementation of ports.EventBus
type EventBus struct {
	mock.Mock
}

// Publish publishes an event to all subscribers
func (m *EventBus) Publish(event ports.Event) {
	m.Called(event)
}

// Subscribe subscribes to events of a specific type
func (m *EventBus) Subscribe(eventType ports.EventType, handler ports.EventHandler) func() {
	var args = m.Called(eventType, handler)
	if args.Get(0) == nil {
		return func() {}
	}
	return args.Get(0).(func())
}

// SubscribeAll subscribes to all events
func (m *EventBus) SubscribeAll(handler ports.EventHandler) func() {
	var args = m.Called(handler)
	if args.Get(0) == nil {
		return func() {}
	}
	return args.Get(0).(func())
}

// Ensure EventBus implements ports.EventBus
var _ ports.EventBus = (*EventBus)(nil)
