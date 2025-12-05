package ports

import "time"

// Event represents a system event that can be published and subscribed to.
// This enables decoupling between components.
type Event interface {
	Type() EventType
	Timestamp() time.Time
}

// EventType identifies the type of event
type EventType string

const (
	// Connection events
	EventTypeConnected    EventType = "connected"
	EventTypeDisconnected EventType = "disconnected"
	EventTypeConnectError EventType = "connect_error"

	// Sync events
	EventTypeSyncStarted   EventType = "sync_started"
	EventTypeSyncCompleted EventType = "sync_completed"
	EventTypeSyncError     EventType = "sync_error"

	// Email events
	EventTypeNewEmail     EventType = "new_email"
	EventTypeEmailRead    EventType = "email_read"
	EventTypeEmailArchive EventType = "email_archive"
	EventTypeEmailDelete  EventType = "email_delete"

	// Send events
	EventTypeSendStarted   EventType = "send_started"
	EventTypeSendCompleted EventType = "send_completed"
	EventTypeSendError     EventType = "send_error"
	EventTypeBounce        EventType = "bounce"

	// Draft events
	EventTypeDraftCreated   EventType = "draft_created"
	EventTypeDraftScheduled EventType = "draft_scheduled"
	EventTypeDraftCancelled EventType = "draft_cancelled"

	// Batch events
	EventTypeBatchCreated   EventType = "batch_created"
	EventTypeBatchConfirmed EventType = "batch_confirmed"
	EventTypeBatchExecuted  EventType = "batch_executed"
	EventTypeBatchCancelled EventType = "batch_cancelled"

	// Index events
	EventTypeIndexStarted   EventType = "index_started"
	EventTypeIndexProgress  EventType = "index_progress"
	EventTypeIndexCompleted EventType = "index_completed"

	// Thread events
	EventTypeThreadMarkedRead   EventType = "thread_marked_read"
	EventTypeThreadMarkedUnread EventType = "thread_marked_unread"

	// Contact events
	EventTypeContactSyncStarted   EventType = "contact_sync_started"
	EventTypeContactSyncCompleted EventType = "contact_sync_completed"
	EventTypeContactSyncFailed    EventType = "contact_sync_failed"
)

// BaseEvent provides common event fields
type BaseEvent struct {
	EventType EventType
	Time      time.Time
}

func (e BaseEvent) Type() EventType    { return e.EventType }
func (e BaseEvent) Timestamp() time.Time { return e.Time }

// NewBaseEvent creates a new base event
func NewBaseEvent(t EventType) BaseEvent {
	return BaseEvent{EventType: t, Time: time.Now()}
}

// ConnectedEvent is emitted when connection is established
type ConnectedEvent struct {
	BaseEvent
}

// DisconnectedEvent is emitted when connection is lost
type DisconnectedEvent struct {
	BaseEvent
	Reason string
}

// ConnectErrorEvent is emitted when connection fails
type ConnectErrorEvent struct {
	BaseEvent
	Error error
}

// SyncStartedEvent is emitted when sync begins
type SyncStartedEvent struct {
	BaseEvent
	Folder string
}

// SyncCompletedEvent is emitted when sync completes
type SyncCompletedEvent struct {
	BaseEvent
	Folder string
	Result *SyncResult
}

// SyncErrorEvent is emitted when sync fails
type SyncErrorEvent struct {
	BaseEvent
	Folder string
	Error  error
}

// NewEmailEvent is emitted when a new email arrives
type NewEmailEvent struct {
	BaseEvent
	Email EmailMetadata
}

// EmailReadEvent is emitted when an email is marked as read
type EmailReadEvent struct {
	BaseEvent
	EmailID int64
	Read    bool
}

// SendCompletedEvent is emitted when an email is sent
type SendCompletedEvent struct {
	BaseEvent
	Result *SendResult
}

// BounceEvent is emitted when a bounce is detected
type BounceEvent struct {
	BaseEvent
	Bounce BounceInfo
}

// BatchCreatedEvent is emitted when a batch operation is created
type BatchCreatedEvent struct {
	BaseEvent
	Operation *BatchOperation
}

// IndexProgressEvent is emitted during indexing
type IndexProgressEvent struct {
	BaseEvent
	Current int
	Total   int
}

// ThreadMarkedReadEvent is emitted when a thread is marked as read
type ThreadMarkedReadEvent struct {
	BaseEvent
	ThreadID string
	Count    int // number of messages marked as read
}

// ThreadMarkedUnreadEvent is emitted when a thread is marked as unread
type ThreadMarkedUnreadEvent struct {
	BaseEvent
	ThreadID string
}

// ContactSyncStartedEvent is emitted when contact sync begins
type ContactSyncStartedEvent struct {
	BaseEvent
	AccountID int64
	FullSync  bool
}

// ContactSyncCompletedEvent is emitted when contact sync completes
type ContactSyncCompletedEvent struct {
	BaseEvent
	AccountID   int64
	TotalSynced int
	FullSync    bool
}

// ContactSyncFailedEvent is emitted when contact sync fails
type ContactSyncFailedEvent struct {
	BaseEvent
	AccountID int64
	Error     string
}

// EventHandler is a function that handles events
type EventHandler func(Event)

// EventBus allows publishing and subscribing to events
type EventBus interface {
	// Publish publishes an event to all subscribers
	Publish(event Event)

	// Subscribe subscribes to events of a specific type
	Subscribe(eventType EventType, handler EventHandler) (unsubscribe func())

	// SubscribeAll subscribes to all events
	SubscribeAll(handler EventHandler) (unsubscribe func())
}
