package commands

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/oneliang/aura/api/pkg/handlers"
	"github.com/oneliang/aura/api/pkg/server"
	cmds "github.com/oneliang/aura/commands/pkg"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/logger"
	ffp "github.com/oneliang/aura/shared/pkg/utils/filepath"
	skillloader "github.com/oneliang/aura/skill/pkg/loader"
	skillmanager "github.com/oneliang/aura/skill/pkg/manager"
	"github.com/spf13/cobra"
)

var (
	servePort     string
	serveLLMURL   string
	serveLLMModel string
	globalLogger  *logger.Logger
)

// ServeCmd is the root command for API server.
var ServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the API server",
	Long:  `Start the Aura API server for 24/7 session management and external triggers.`,
	Run:   runServe,
}

func init() {
	ServeCmd.Flags().StringVar(&servePort, "port", "", "Port to listen on (default: 8080 or config)")
	ServeCmd.Flags().StringVar(&serveLLMURL, "llm-url", "", "LLM base URL (default: from config)")
	ServeCmd.Flags().StringVar(&serveLLMModel, "llm-model", "", "LLM model to use (default: from config)")
}

func runServe(cmd *cobra.Command, args []string) {
	cmdCtx := GetCommandContext()
	if cmdCtx == nil {
		cmdCtx = DefaultCommandContext()
		SetCommandContext(cmdCtx)
	}

	// Load config to get LLM settings
	cfg, err := cmdCtx.ConfigLoader.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger with config
	globalLogger = logger.NewNamed(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
		Output: cfg.Log.Output,
		Path:   cfg.Log.Path,
		Module: "serve",
	})
	defer logger.CloseAll()

	globalLogger.Info().Msg("Starting Aura API Server...")

	// Use config values as defaults, CLI flags override
	port := servePort
	if port == "" {
		// Try config file first, then fall back to default
		if cfg.API.Port != "" {
			port = cfg.API.Port
		} else {
			port = strconv.Itoa(constants.DefaultAPIPort)
		}
	}

	llmURL := serveLLMURL
	if llmURL == "" {
		llmURL = cfg.LLM.BaseURL
	}

	llmModel := serveLLMModel
	if llmModel == "" {
		llmModel = cfg.LLM.Model
	}

	// Use default sessions directory
	sessionsDir := ffp.MustAuraHomePath(constants.DirSessions)

	globalLogger.Info().
		Str("module", "serve").
		Str("port", port).
		Str("sessions_dir", sessionsDir).
		Str("llm_url", llmURL).
		Str("llm_model", llmModel).
		Msg("Server configuration")

	srv, err := server.NewServer(server.ServerConfig{
		Port:        port,
		SessionsDir: sessionsDir,
		LLMBaseURL:  llmURL,
		LLMModel:    llmModel,
		Config:      cfg,
		UserID:      cmdCtx.UserID,
	})
	if err != nil {
		globalLogger.Error().Err(err).Str("module", "serve").Msg("Failed to create server")
		os.Exit(1)
	}

	// Create and set skills handler via SkillManager
	if cfg.Skills.Enabled && len(cfg.Skills.Directories) > 0 {
		skillLoader := skillloader.NewLoader(cfg.Skills.Directories)
		skillMgr := skillmanager.NewSkillManager(skillLoader, cfg.Skills.Directories)
		if _, err := skillLoader.Load(); err != nil {
			logger.RegistryDefault().Warn().Err(err).Msg("Failed to load skills for API server")
		} else {
			cmdProvider := cmds.NewCommandProvider(cmds.CommandProviderDeps{
				Config:       cfg,
				ConfigPath:   "",
				UserID:       cmdCtx.UserID,
				SkillLoader:  skillLoader,
				SkillManager: skillMgr,
			})
			srv.SetSkillsHandler(handlers.NewSkillsHandler(handlers.NewSkillsServiceWrapper(cmdProvider)))
		}
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	go func() {
		globalLogger.Info().Str("module", "serve").Str("port", port).Msg("Server listening")
		globalLogger.Info().Str("module", "serve").Msg("Press Ctrl+C to stop")
		globalLogger.Info().Str("module", "serve").Msg("API Endpoints:")
		globalLogger.Debug().Str("module", "serve").Msgf("  GET    http://localhost:%s/api/v1/health", port)
		globalLogger.Debug().Str("module", "serve").Msgf("  GET    http://localhost:%s/api/v1/sessions", port)
		globalLogger.Debug().Str("module", "serve").Msgf("  POST   http://localhost:%s/api/v1/sessions", port)
		globalLogger.Debug().Str("module", "serve").Msgf("  GET    http://localhost:%s/api/v1/sessions/{id}", port)
		globalLogger.Debug().Str("module", "serve").Msgf("  GET    http://localhost:%s/api/v1/sessions/{id}/messages", port)
		globalLogger.Debug().Str("module", "serve").Msgf("  POST   http://localhost:%s/api/v1/sessions/{id}/message", port)
		globalLogger.Debug().Str("module", "serve").Msgf("  DELETE http://localhost:%s/api/v1/sessions/{id}", port)

		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			globalLogger.Error().Err(err).Str("module", "serve").Msg("Server error")
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	globalLogger.Info().Str("module", "serve").Msg("Shutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		globalLogger.Error().Err(err).Str("module", "serve").Msg("Shutdown error")
		os.Exit(1)
	}

	globalLogger.Info().Str("module", "serve").Msg("Server stopped.")
}
