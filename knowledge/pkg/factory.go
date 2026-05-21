// Package knowledge provides factory functions for creating knowledge collections.
package knowledge

import (
	"context"
	"fmt"

	"github.com/oneliang/aura/knowledge/pkg/embedding"
	"github.com/oneliang/aura/knowledge/pkg/storage"
	"github.com/oneliang/aura/shared/pkg/config"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/user"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
)

// CollectionFactory defines the interface for creating knowledge collections.
// This abstraction allows callers to provide custom collection creation logic.
type CollectionFactory interface {
	NewCollection(ctx context.Context, userID string) (*storage.ChromemCollection, error)
}

// DefaultCollectionFactory is the default implementation of CollectionFactory.
// It creates collections using the standard Aura knowledge base configuration.
type DefaultCollectionFactory struct {
	config      *config.Config
	userManager *config.UsersConfig // User configuration for resolving user paths
}

// NewDefaultCollectionFactory creates a new default collection factory.
func NewDefaultCollectionFactory(cfg *config.Config) *DefaultCollectionFactory {
	var userManager *config.UsersConfig
	if cfg != nil {
		userManager = &cfg.Users
	}
	return &DefaultCollectionFactory{
		config:      cfg,
		userManager: userManager,
	}
}

// getUserKnowledgeDir returns the knowledge directory for a user.
func (f *DefaultCollectionFactory) getUserKnowledgeDir(userID string, isShared bool) (string, error) {
	if userID == "" || userID == "default" {
		// Legacy mode: use global knowledge directory
		return ffp.MustAuraHomePath(constants.DirKnowledge), nil
	}

	// Find user config
	var userCfg *config.UserConfig
	for i := range f.userManager.Definitions {
		if f.userManager.Definitions[i].ID == userID {
			userCfg = &f.userManager.Definitions[i]
			break
		}
	}

	if userCfg == nil {
		return "", fmt.Errorf("user not found: %s", userID)
	}

	// If user has configured knowledge directories, use them
	if len(userCfg.KnowledgeDirs) > 0 {
		if isShared && len(userCfg.KnowledgeDirs) > 1 {
			// Use shared directory (second one)
			return userCfg.KnowledgeDirs[1], nil
		}
		// Use first directory (private or only)
		return userCfg.KnowledgeDirs[0], nil
	}

	// Fallback: construct default path
	kbType := user.KnowledgePrivate
	if isShared {
		kbType = user.KnowledgeShared
	}
	return ffp.MustAuraHomePath(constants.DirUsers, userID, "knowledge", kbType), nil
}

// NewCollection creates a new knowledge collection for the specified user.
func (f *DefaultCollectionFactory) NewCollection(ctx context.Context, userID string) (*storage.ChromemCollection, error) {
	// Determine data directory (default to private knowledge)
	dataDir, err := f.getUserKnowledgeDir(userID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get user knowledge dir: %w", err)
	}

	// Determine collection name using shared utility
	collectionName := user.GetUserCollectionName(userID)

	// Use configured embedding model or default
	embeddingModel := "nomic-embed-text"
	if f.config != nil && f.config.LLM.EmbeddingModel != "" {
		embeddingModel = f.config.LLM.EmbeddingModel
	}

	baseURL := ""
	if f.config != nil {
		baseURL = f.config.LLM.BaseURL
	}

	embFn := embedding.OllamaEmbeddingFunc(baseURL, embeddingModel)

	return storage.NewChromemCollection(ctx, storage.ChromemOptions{
		DataDir:       dataDir,
		Name:          collectionName,
		EmbeddingFunc: embFn,
	})
}
