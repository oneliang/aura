// Package memory provides memory interfaces and types for the Aura system.
package memory

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/oneliang/aura/shared/pkg/i18n"
)

// ContentBlockType defines the type of a content block.
type ContentBlockType string

const (
	BlockTypeText       ContentBlockType = "text"
	BlockTypeThinking   ContentBlockType = "thinking"
	BlockTypeToolUse    ContentBlockType = "tool_use"
	BlockTypeToolResult ContentBlockType = "tool_result"
)

// ContentBlock is the interface for message content blocks.
type ContentBlock interface {
	BlockType() ContentBlockType
}

// TextBlock represents a text content block.
type TextBlock struct {
	Type ContentBlockType `json:"type"`
	Text string           `json:"text"`
}

func (b TextBlock) BlockType() ContentBlockType { return b.Type }

// ThinkingBlock represents LLM thinking/reasoning content.
type ThinkingBlock struct {
	Type      ContentBlockType `json:"type"`
	Thinking  string           `json:"thinking"`
	Signature string           `json:"signature,omitempty"`
}

func (b ThinkingBlock) BlockType() ContentBlockType { return b.Type }

// ToolUseBlock represents a tool invocation from the LLM.
type ToolUseBlock struct {
	Type  ContentBlockType `json:"type"`
	ID    string           `json:"id"`
	Name  string           `json:"name"`
	Input json.RawMessage  `json:"input"`
}

func (b ToolUseBlock) BlockType() ContentBlockType { return b.Type }

// ToolResultBlock represents the result of a tool execution.
type ToolResultBlock struct {
	Type       ContentBlockType `json:"type"`
	ToolUseID  string           `json:"tool_use_id"`
	Content    []ContentBlock   `json:"content"`
	IsError    bool             `json:"is_error,omitempty"`
}

func (b ToolResultBlock) BlockType() ContentBlockType { return b.Type }

// RawContentBlock is an intermediate structure for JSON unmarshaling.
// It handles polymorphic deserialization of ContentBlock interface.
type RawContentBlock struct {
	Type       ContentBlockType  `json:"type"`
	Text       *string           `json:"text,omitempty"`
	Thinking   *string           `json:"thinking,omitempty"`
	Signature  *string           `json:"signature,omitempty"`
	ID         *string           `json:"id,omitempty"`
	Name       *string           `json:"name,omitempty"`
	Input      json.RawMessage   `json:"input,omitempty"`
	ToolUseID  *string           `json:"tool_use_id,omitempty"`
	Content    []RawContentBlock `json:"content,omitempty"`
	IsError    bool              `json:"is_error,omitempty"`
}

// ToContentBlock converts RawContentBlock to the appropriate ContentBlock type.
// Returns an error if required fields are missing for the block type.
func (r RawContentBlock) ToContentBlock() (ContentBlock, error) {
	switch r.Type {
	case BlockTypeText:
		if r.Text == nil {
			return nil, errors.New(i18n.T("error.content_block.text_missing"))
		}
		return TextBlock{Type: r.Type, Text: *r.Text}, nil
	case BlockTypeThinking:
		if r.Thinking == nil {
			return nil, errors.New(i18n.T("error.content_block.thinking_missing"))
		}
		sig := ""
		if r.Signature != nil {
			sig = *r.Signature
		}
		return ThinkingBlock{Type: r.Type, Thinking: *r.Thinking, Signature: sig}, nil
	case BlockTypeToolUse:
		if r.ID == nil {
			return nil, errors.New(i18n.T("error.content_block.tool_use_id_missing"))
		}
		if r.Name == nil {
			return nil, errors.New(i18n.T("error.content_block.tool_use_name_missing"))
		}
		return ToolUseBlock{Type: r.Type, ID: *r.ID, Name: *r.Name, Input: r.Input}, nil
	case BlockTypeToolResult:
		if r.ToolUseID == nil {
			return nil, errors.New(i18n.T("error.content_block.tool_result_id_missing"))
		}
		content := make([]ContentBlock, 0)
		for _, c := range r.Content {
			cb, err := c.ToContentBlock()
			if err != nil {
				return nil, errors.New(fmt.Sprintf(i18n.T("error.content_block.tool_result_content"), err))
			}
			content = append(content, cb)
		}
		return ToolResultBlock{Type: r.Type, ToolUseID: *r.ToolUseID, Content: content, IsError: r.IsError}, nil
	default:
		return nil, errors.New(fmt.Sprintf(i18n.T("error.content_block.unknown_type"), r.Type))
	}
}

// ContentBlocksToRaw converts ContentBlock slice to RawContentBlock slice for marshaling.
// Panics if an unknown ContentBlock type is encountered.
func ContentBlocksToRaw(blocks []ContentBlock) []RawContentBlock {
	raw := make([]RawContentBlock, len(blocks))
	for i, b := range blocks {
		switch block := b.(type) {
		case TextBlock:
			raw[i] = RawContentBlock{Type: block.Type, Text: &block.Text}
		case ThinkingBlock:
			raw[i] = RawContentBlock{Type: block.Type, Thinking: &block.Thinking, Signature: &block.Signature}
		case ToolUseBlock:
			raw[i] = RawContentBlock{Type: block.Type, ID: &block.ID, Name: &block.Name, Input: block.Input}
		case ToolResultBlock:
			content := ContentBlocksToRaw(block.Content)
			raw[i] = RawContentBlock{Type: block.Type, ToolUseID: &block.ToolUseID, Content: content, IsError: block.IsError}
		default:
			panic(fmt.Sprintf("ContentBlocksToRaw: unknown content block type: %T", b))
		}
	}
	return raw
}

// MarshalContentBlocks marshals ContentBlock slice to JSON.
func MarshalContentBlocks(blocks []ContentBlock) ([]byte, error) {
	return json.Marshal(ContentBlocksToRaw(blocks))
}

// UnmarshalContentBlocks unmarshals JSON to ContentBlock slice.
// Returns an error if any content block has missing required fields.
func UnmarshalContentBlocks(data []byte) ([]ContentBlock, error) {
	var raw []RawContentBlock
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	blocks := make([]ContentBlock, 0, len(raw))
	for i, r := range raw {
		cb, err := r.ToContentBlock()
		if err != nil {
			return nil, errors.New(fmt.Sprintf(i18n.T("error.content_block.index_error"), i, err))
		}
		blocks = append(blocks, cb)
	}
	return blocks, nil
}