// Package importer provides document importers for the knowledge base.
package importer

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/oneliang/aura/knowledge/pkg/storage"
)

// Importer imports documents into a storage collection.
type Importer struct {
	collection storage.Collection
	chunkSize  int
	overlap    int
}

// Option configures an Importer.
type Option func(*Importer)

// WithChunkSize sets the chunk size in characters.
func WithChunkSize(size int) Option {
	return func(i *Importer) { i.chunkSize = size }
}

// WithOverlap sets the overlap between chunks in characters.
func WithOverlap(n int) Option {
	return func(i *Importer) { i.overlap = n }
}

// New creates a new Importer.
func New(collection storage.Collection, opts ...Option) *Importer {
	imp := &Importer{
		collection: collection,
		chunkSize:  1000,
		overlap:    200,
	}
	for _, o := range opts {
		o(imp)
	}
	return imp
}

// ImportFile imports a single file. Supports .md, .txt, .go, and other text files.
func (im *Importer) ImportFile(ctx context.Context, path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	source := filepath.Base(path)

	chunks := im.chunk(string(content))
	docs := make([]storage.Document, 0, len(chunks))
	for i, chunk := range chunks {
		docs = append(docs, storage.Document{
			ID:      fmt.Sprintf("%s#%d", path, i),
			Content: chunk,
			Metadata: map[string]any{
				"source": source,
				"path":   path,
				"type":   ext,
				"chunk":  i,
			},
		})
	}

	if err := im.collection.Add(ctx, docs); err != nil {
		return 0, fmt.Errorf("add documents: %w", err)
	}
	return len(docs), nil
}

// ImportDir recursively imports all text files in a directory.
func (im *Importer) ImportDir(ctx context.Context, dir string) (int, error) {
	total := 0
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !isTextFile(path) {
			return nil
		}
		n, err := im.ImportFile(ctx, path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skip %s: %v\n", path, err)
			return nil
		}
		total += n
		return nil
	})
	return total, err
}

// ImportText imports raw text with a given source label.
func (im *Importer) ImportText(ctx context.Context, text, source string) (int, error) {
	chunks := im.chunk(text)
	docs := make([]storage.Document, 0, len(chunks))
	for i, chunk := range chunks {
		docs = append(docs, storage.Document{
			ID:      fmt.Sprintf("%s#%d", source, i),
			Content: chunk,
			Metadata: map[string]any{
				"source": source,
				"chunk":  i,
			},
		})
	}
	if err := im.collection.Add(ctx, docs); err != nil {
		return 0, err
	}
	return len(docs), nil
}

// chunk splits text into overlapping chunks, splitting on paragraph or sentence
// boundaries where possible.
func (im *Importer) chunk(text string) []string {
	if len(text) <= im.chunkSize {
		return []string{strings.TrimSpace(text)}
	}

	var chunks []string
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Split(bufio.ScanLines)

	var buf strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if buf.Len()+len(line)+1 > im.chunkSize && buf.Len() > 0 {
			chunks = append(chunks, strings.TrimSpace(buf.String()))
			// Keep overlap from end of current buffer
			content := buf.String()
			overlapStart := len(content) - im.overlap
			if overlapStart < 0 {
				overlapStart = 0
			}
			buf.Reset()
			buf.WriteString(content[overlapStart:])
		}
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	if buf.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(buf.String()))
	}
	return chunks
}

var textExtensions = map[string]bool{
	".md": true, ".txt": true, ".go": true, ".py": true,
	".js": true, ".ts": true, ".java": true, ".c": true,
	".cpp": true, ".h": true, ".rs": true, ".yaml": true,
	".yml": true, ".toml": true, ".json": true, ".sh": true,
	".html": true, ".css": true, ".sql": true, ".rb": true,
}

func isTextFile(path string) bool {
	return textExtensions[strings.ToLower(filepath.Ext(path))]
}
