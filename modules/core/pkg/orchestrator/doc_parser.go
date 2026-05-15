package orchestrator

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const frontMatterDelimiter = "---"

// ParseDoc parses a raw Markdown file with YAML front-matter into a CollaboDoc.
// Expected format:
// ---
// id: doc-123
// type: task_assignment
// ...
// ---
// Document body markdown content...
func ParseDoc(raw []byte) (*CollaboDoc, error) {
	content := string(raw)

	// Find front-matter delimiters
	parts := strings.SplitN(content, frontMatterDelimiter, 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid document format: missing front-matter delimiters")
	}

	// parts[0] is empty (before first ---)
	// parts[1] is YAML content
	// parts[2] is the rest (should start with \n then body)
	yamlContent := strings.TrimSpace(parts[1])
	body := strings.TrimPrefix(parts[2], "\n")
	body = strings.TrimSpace(body)

	// Parse YAML front-matter
	var meta struct {
		ID           string         `yaml:"id"`
		Type         DocType        `yaml:"type"`
		From         string         `yaml:"from"`
		To           string         `yaml:"to,omitempty"`
		Priority     Priority       `yaml:"priority"`
		Status       DocStatus      `yaml:"status"`
		Title        string         `yaml:"title"`
		Dependencies []string       `yaml:"dependencies,omitempty"`
		CreatedAt    time.Time      `yaml:"created_at"`
		UpdatedAt    time.Time      `yaml:"updated_at"`
		History      []HistoryEntry `yaml:"history,omitempty"`
	}

	if err := yaml.Unmarshal([]byte(yamlContent), &meta); err != nil {
		return nil, fmt.Errorf("failed to parse YAML front-matter: %w", err)
	}

	// Validate required fields
	if meta.ID == "" {
		return nil, fmt.Errorf("missing required field: id")
	}
	if meta.Type == "" {
		return nil, fmt.Errorf("missing required field: type")
	}
	if meta.From == "" {
		return nil, fmt.Errorf("missing required field: from")
	}
	if meta.Status == "" {
		return nil, fmt.Errorf("missing required field: status")
	}
	if meta.Title == "" {
		return nil, fmt.Errorf("missing required field: title")
	}

	// Set default priority
	if meta.Priority == "" {
		meta.Priority = PriorityNormal
	}

	return &CollaboDoc{
		ID:           meta.ID,
		Type:         meta.Type,
		From:         meta.From,
		To:           meta.To,
		Priority:     meta.Priority,
		Status:       meta.Status,
		Title:        meta.Title,
		Dependencies: meta.Dependencies,
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    meta.UpdatedAt,
		Body:         body,
		History:      meta.History,
	}, nil
}

// RenderDoc serializes a CollaboDoc back to []byte for writing.
func RenderDoc(doc *CollaboDoc) ([]byte, error) {
	var buf bytes.Buffer

	// Write front-matter
	buf.WriteString(frontMatterDelimiter)
	buf.WriteString("\n")

	// Create metadata struct for YAML serialization
	meta := struct {
		ID           string         `yaml:"id"`
		Type         DocType        `yaml:"type"`
		From         string         `yaml:"from"`
		To           string         `yaml:"to,omitempty"`
		Priority     Priority       `yaml:"priority"`
		Status       DocStatus      `yaml:"status"`
		Title        string         `yaml:"title"`
		Dependencies []string       `yaml:"dependencies,omitempty"`
		CreatedAt    time.Time      `yaml:"created_at"`
		UpdatedAt    time.Time      `yaml:"updated_at"`
		History      []HistoryEntry `yaml:"history,omitempty"`
	}{
		ID:           doc.ID,
		Type:         doc.Type,
		From:         doc.From,
		To:           doc.To,
		Priority:     doc.Priority,
		Status:       doc.Status,
		Title:        doc.Title,
		Dependencies: doc.Dependencies,
		CreatedAt:    doc.CreatedAt,
		UpdatedAt:    doc.UpdatedAt,
		History:      doc.History,
	}

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(meta); err != nil {
		return nil, fmt.Errorf("failed to encode YAML front-matter: %w", err)
	}
	encoder.Close()

	buf.WriteString(frontMatterDelimiter)
	buf.WriteString("\n\n")
	buf.WriteString(doc.Body)

	return buf.Bytes(), nil
}
