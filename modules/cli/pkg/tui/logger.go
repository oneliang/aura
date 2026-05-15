package tui

import (
	"github.com/oneliang/aura/shared/pkg/logdir"
	"github.com/oneliang/aura/shared/pkg/logger"
)

// log is the shared logger for the TUI package.
var log *logger.Logger

func init() {
	logPath, err := logdir.GetLogFile("tui.log")
	if err != nil {
		// Fallback to stdout if log dir unavailable
		log = logger.NewNamed(logger.Config{Level: "debug", Format: "text", Output: "stdout", Module: "tui"})
		return
	}
	log = logger.NewNamed(logger.Config{
		Level:  "debug",
		Format: "text",
		Output: "file",
		Path:   logPath,
		Module: "tui",
	})
}

// GetLogger returns the shared TUI logger for injection.
func GetLogger() *logger.Logger {
	return log
}
