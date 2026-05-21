package server

import "context"

// NotificationSender abstracts pushing messages to external platforms and lifecycle management.
type NotificationSender interface {
	// PushToSession sends a message of the given type to the session identified by sessionID.
	PushToSession(ctx context.Context, sessionID, msgType string, content map[string]interface{}) error
	// Shutdown gracefully shuts down the notification sender.
	Shutdown(ctx context.Context) error
}

// MessageType constants for notifications.
// These mirror the constants in the adapters package but are defined locally
// to avoid coupling the api module to adapters.
const (
	MsgTypeText        = "text"
	MsgTypeCard        = "card"
	MsgTypePost        = "post"
	MsgTypeInteractive = "interactive"
)
