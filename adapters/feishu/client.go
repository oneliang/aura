package feishu

import (
	"context"
	"encoding/json"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/oneliang/aura/shared/pkg/logger"
)

// APIError represents a Feishu API error with code and message.
type APIError struct {
	Code    int
	Message string
	Op      string // Operation that failed
}

func (e *APIError) Error() string {
	return fmt.Sprintf("feishu API error: %s failed: code=%d, msg=%s", e.Op, e.Code, e.Message)
}

// Is implements errors.Is interface for comparison.
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// newAPIError creates a new APIError from response.
func newAPIError(op string, code int, msg string) *APIError {
	return &APIError{
		Op:      op,
		Code:    code,
		Message: msg,
	}
}

// Client is the Feishu Open API client using official SDK.
type Client struct {
	appID     string
	appSecret string
	apiClient *lark.Client
	logger    *logger.Logger
}

// NewClient creates a new Feishu API client.
func NewClient(appID, appSecret string, log *logger.Logger) *Client {
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		apiClient: lark.NewClient(appID, appSecret),
		logger:    log,
	}
}

// SendTextMessage sends a text message to a Feishu user or chat.
func (c *Client) SendTextMessage(ctx context.Context, receiveID, receiveIDType, text string) error {
	// Marshal the entire content object to ensure valid JSON
	contentJSON, err := json.Marshal(map[string]string{"text": text})
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}
	content := string(contentJSON)

	c.logger.Debug("Sending message", "module", "feishu", "to", receiveID, "type", receiveIDType, "content", content)

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType(larkim.MsgTypeText).
			Content(content).
			Build()).
		Build()

	resp, err := c.apiClient.Im.Message.Create(ctx, req)
	if err != nil {
		c.logger.Error("Failed to send message", "module", "feishu", "error", err.Error())
		return fmt.Errorf("failed to send message: %w", err)
	}

	if !resp.Success() {
		c.logger.Error("Send message failed", "module", "feishu", "code", resp.Code, "msg", resp.Msg)
		return newAPIError("SendTextMessage", resp.Code, resp.Msg)
	}

	c.logger.Info("Message sent", "module", "feishu", "to", receiveID)
	return nil
}

// SendMessage sends a message to a Feishu user or chat.
func (c *Client) SendMessage(ctx context.Context, receiveID, receiveIDType string, content map[string]string) error {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("failed to marshal content: %w", err)
	}

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType(larkim.MsgTypeText).
			Content(string(contentJSON)).
			Build()).
		Build()

	resp, err := c.apiClient.Im.Message.Create(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	if !resp.Success() {
		return newAPIError("SendMessage", resp.Code, resp.Msg)
	}

	return nil
}

// SendReplyMessage sends a reply message in a chat (as a new message in the same chat).
// Note: The SDK doesn't support setting reply_id directly, so this sends a new message to the chat.
func (c *Client) SendReplyMessage(ctx context.Context, chatID, parentMessageID, msgType string, content map[string]string) error {
	// For simplicity, just send to the same chat without threading
	// Advanced threading would require using the API with reply_id parameter
	return c.SendTextMessage(ctx, chatID, larkim.ReceiveIdTypeChatId, content["text"])
}

// SendPostMessage sends a post (rich text) message to a Feishu user or chat.
// Post messages support formatted text with tags like bold, italic, link, etc.
func (c *Client) SendPostMessage(ctx context.Context, receiveID, receiveIDType string, postContent map[string]interface{}) error {
	contentJSON, err := json.Marshal(postContent)
	if err != nil {
		return fmt.Errorf("failed to marshal post content: %w", err)
	}

	c.logger.Debug("Sending post message", "module", "feishu", "to", receiveID, "type", receiveIDType, "content", postContent)

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType(larkim.MsgTypePost).
			Content(string(contentJSON)).
			Build()).
		Build()

	resp, err := c.apiClient.Im.Message.Create(ctx, req)
	if err != nil {
		c.logger.Error("Failed to send post message", "module", "feishu", "error", err.Error())
		return fmt.Errorf("failed to send post message: %w", err)
	}

	if !resp.Success() {
		c.logger.Error("Send post message failed", "module", "feishu", "code", resp.Code, "msg", resp.Msg)
		return newAPIError("SendPostMessage", resp.Code, resp.Msg)
	}

	c.logger.Info("Post message sent", "module", "feishu", "to", receiveID)
	return nil
}

// SendCardMessage sends an interactive card message to a Feishu user or chat.
// Card messages support rich layouts, colors, buttons, and interactive elements.
func (c *Client) SendCardMessage(ctx context.Context, receiveID, receiveIDType string, cardContent map[string]interface{}) error {
	contentJSON, err := json.Marshal(cardContent)
	if err != nil {
		return fmt.Errorf("failed to marshal card content: %w", err)
	}

	c.logger.Debug("Sending card message", "module", "feishu", "to", receiveID, "type", receiveIDType, "content", cardContent)

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType(receiveIDType).
		Body(larkim.NewCreateMessageReqBodyBuilder().
			ReceiveId(receiveID).
			MsgType(larkim.MsgTypeInteractive).
			Content(string(contentJSON)).
			Build()).
		Build()

	resp, err := c.apiClient.Im.Message.Create(ctx, req)
	if err != nil {
		c.logger.Error("Failed to send card message", "module", "feishu", "error", err.Error())
		return fmt.Errorf("failed to send card message: %w", err)
	}

	if !resp.Success() {
		c.logger.Error("Send card message failed", "module", "feishu", "code", resp.Code, "msg", resp.Msg)
		return newAPIError("SendCardMessage", resp.Code, resp.Msg)
	}

	c.logger.Info("Card message sent", "module", "feishu", "to", receiveID)
	return nil
}

// AddMessageReaction adds an emoji reaction to a message.
// The reactionType should be a Feishu emoji type like "THINKING", "STANDARD_EMOJI_1", etc.
// See https://open.feishu.cn/document/server-docs/im-v1/message-reaction/emojis-introduce
func (c *Client) AddMessageReaction(ctx context.Context, messageID string, reactionType string) error {
	c.logger.Debug("Adding message reaction", "module", "feishu", "message_id", messageID, "reaction", reactionType)

	req := larkim.NewCreateMessageReactionReqBuilder().
		MessageId(messageID).
		Body(larkim.NewCreateMessageReactionReqBodyBuilder().
			ReactionType(&larkim.Emoji{EmojiType: &reactionType}).
			Build()).
		Build()

	resp, err := c.apiClient.Im.MessageReaction.Create(ctx, req)
	if err != nil {
		c.logger.Debug("Failed to add message reaction", "module", "feishu", "error", err.Error())
		return fmt.Errorf("failed to add message reaction: %w", err)
	}

	if !resp.Success() {
		c.logger.Debug("Add message reaction failed", "module", "feishu", "code", resp.Code, "msg", resp.Msg)
		return newAPIError("AddMessageReaction", resp.Code, resp.Msg)
	}

	return nil
}

// RemoveMessageReaction removes an emoji reaction from a message.
// Note: This requires the reaction_id which is returned when adding the reaction.
// If you don't have the reaction_id, you can use RemoveMessageReactionByType which lists all reactions first.
func (c *Client) RemoveMessageReaction(ctx context.Context, messageID, reactionID string) error {
	c.logger.Debug("Removing message reaction", "module", "feishu", "message_id", messageID, "reaction_id", reactionID)

	req := larkim.NewDeleteMessageReactionReqBuilder().
		MessageId(messageID).
		ReactionId(reactionID).
		Build()

	resp, err := c.apiClient.Im.MessageReaction.Delete(ctx, req)
	if err != nil {
		c.logger.Debug("Failed to remove message reaction", "module", "feishu", "error", err.Error())
		return fmt.Errorf("failed to remove message reaction: %w", err)
	}

	if !resp.Success() {
		c.logger.Debug("Remove message reaction failed", "module", "feishu", "code", resp.Code, "msg", resp.Msg)
		return newAPIError("RemoveMessageReaction", resp.Code, resp.Msg)
	}

	return nil
}

// RemoveMessageReactionByType removes an emoji reaction from a message by emoji type.
// This method lists all reactions and finds the one matching the given type.
func (c *Client) RemoveMessageReactionByType(ctx context.Context, messageID, reactionType string) error {
	c.logger.Debug("Removing message reaction by type", "module", "feishu", "message_id", messageID, "reaction", reactionType)

	// List all reactions for the message
	listReq := larkim.NewListMessageReactionReqBuilder().
		MessageId(messageID).
		Build()

	listResp, err := c.apiClient.Im.MessageReaction.List(ctx, listReq)
	if err != nil || !listResp.Success() {
		c.logger.Debug("Failed to list message reactions", "module", "feishu", "error", err.Error())
		return fmt.Errorf("failed to list message reactions: %w", err)
	}

	// Find the reaction with matching type
	if listResp.Data != nil && listResp.Data.Items != nil {
		for _, reaction := range listResp.Data.Items {
			if reaction != nil && reaction.ReactionType != nil && reaction.ReactionType.EmojiType != nil {
				if *reaction.ReactionType.EmojiType == reactionType && reaction.ReactionId != nil {
					return c.RemoveMessageReaction(ctx, messageID, *reaction.ReactionId)
				}
			}
		}
	}

	return nil
}

// DeleteMessage deletes a message by message ID.
// This is used to remove temporary "processing" indicator messages.
func (c *Client) DeleteMessage(ctx context.Context, messageID string) error {
	c.logger.Debug("Deleting message", "module", "feishu", "message_id", messageID)

	req := larkim.NewDeleteMessageReqBuilder().
		MessageId(messageID).
		Build()

	resp, err := c.apiClient.Im.Message.Delete(ctx, req)
	if err != nil {
		c.logger.Debug("Failed to delete message", "module", "feishu", "error", err.Error())
		return fmt.Errorf("failed to delete message: %w", err)
	}

	if !resp.Success() {
		c.logger.Debug("Delete message failed", "module", "feishu", "code", resp.Code, "msg", resp.Msg)
		return newAPIError("DeleteMessage", resp.Code, resp.Msg)
	}

	return nil
}
