package notification

import (
	"context"

	"project-management-tools/internal/domain/shared"
)

// Event is the message emitted to connected clients after a write operation.
type Event struct {
	Event   string `json:"event"`
	Payload any    `json:"payload"`
}

// Notifier is the driven port for real-time event delivery.
// Defined here, in the consumer (application layer).
// Implementations must be safe for concurrent use.
type Notifier interface {
	// Notify emits an event to all clients connected as ownerID.
	// It is fire-and-forget: errors are handled internally by the implementation.
	Notify(ctx context.Context, ownerID shared.ID, event Event)
}

// NoopNotifier is a Notifier that discards all events.
// Use it when real-time delivery is not required (e.g. CLI tools, tests).
type NoopNotifier struct{}

func (NoopNotifier) Notify(_ context.Context, _ shared.ID, _ Event) {}
