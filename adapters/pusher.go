// Package adapters provides a unified interface for external platform adapters.
package adapters

import (
	"context"
)

// MessagePusher is an optional interface that adapters can implement
// to support proactive message pushing (not just reply).
// This is useful for sending notifications, alerts, and scheduled messages.
type MessagePusher interface {
	// PushMessage sends a proactive message to a target.
	// targetType: "open_id", "user_id", "chat_id", "union_id"
	// targetID: the identifier for the target
	// msgType: "text", "post", "card"
	// content: message content (format depends on msgType)
	PushMessage(ctx context.Context, targetType, targetID, msgType string, content map[string]interface{}) error

	// BroadcastMessage sends the same message to multiple targets.
	// Returns a map of targetID -> error (nil means success)
	BroadcastMessage(ctx context.Context, targets []MessageTarget, msgType string, content map[string]interface{}) map[string]error
}

// MessageTarget represents a message recipient.
type MessageTarget struct {
	// TargetType is the type of identifier: "open_id", "user_id", "chat_id", "union_id"
	TargetType string

	// TargetID is the actual identifier
	TargetID string

	// Name is an optional friendly name (for logging)
	Name string
}

// MessageType constants.
const (
	MsgTypeText      = "text"
	MsgTypePost      = "post"
	MsgTypeCard      = "card"
	MsgTypeImage     = "image"
	MsgTypeFile      = "file"
	MsgTypeAudio     = "audio"
	MsgTypeSticker   = "sticker"
	MsgTypeShareChat = "share_chat"
)

// CanPushMessage checks if an adapter supports proactive message pushing.
func CanPushMessage(adapter Adapter) (MessagePusher, bool) {
	pusher, ok := adapter.(MessagePusher)
	return pusher, ok
}
