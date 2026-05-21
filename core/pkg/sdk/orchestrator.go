// Package sdk provides the unified agent runtime and factories.
// This file provides Orchestrator — multi-agent coordination via SDK.
package sdk

import (
	orchestratorpkg "github.com/oneliang/aura/core/pkg/orchestrator"
	"github.com/oneliang/aura/shared/pkg/config"
	toolspkg "github.com/oneliang/aura/tools/pkg"
)

// Orchestrator types.
type (
	Orchestrator       = orchestratorpkg.Orchestrator
	OrchestratorConfig = orchestratorpkg.OrchestratorConfig
	SubAgent           = orchestratorpkg.SubAgent
	CollaboDoc         = orchestratorpkg.CollaboDoc
	DocStatus          = orchestratorpkg.DocStatus
	DocType            = orchestratorpkg.DocType
	Priority           = orchestratorpkg.Priority
	HistoryEntry       = orchestratorpkg.HistoryEntry
)

// DocStatus constants.
const (
	DocStatusPending    = orchestratorpkg.DocStatusPending
	DocStatusInProgress = orchestratorpkg.DocStatusInProgress
	DocStatusCompleted  = orchestratorpkg.DocStatusCompleted
	DocStatusRejected   = orchestratorpkg.DocStatusRejected
	DocStatusBlocked    = orchestratorpkg.DocStatusBlocked
)

// DocType constants.
const (
	DocTypeCollabRequest = orchestratorpkg.DocTypeCollabRequest
	DocTypeTaskAssign    = orchestratorpkg.DocTypeTaskAssign
	DocTypeHandoff       = orchestratorpkg.DocTypeHandoff
	DocTypeReviewRequest = orchestratorpkg.DocTypeReviewRequest
	DocTypeInfoShare     = orchestratorpkg.DocTypeInfoShare
)

// Priority constants.
const (
	PriorityUrgent = orchestratorpkg.PriorityUrgent
	PriorityNormal = orchestratorpkg.PriorityNormal
	PriorityLow    = orchestratorpkg.PriorityLow
)

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator(cfg *config.Config) (*Orchestrator, error) {
	return orchestratorpkg.New(cfg)
}

// Orchestrator tools (registerable via Runtime.AddTool).
func NewSpawnAgentTool(o *Orchestrator) toolspkg.Tool {
	return orchestratorpkg.NewSpawnAgentTool(o)
}

func NewCreateDocTool(o *Orchestrator) toolspkg.Tool {
	return orchestratorpkg.NewCreateDocTool(o)
}

func NewProcessDocTool(o *Orchestrator, agentID string) toolspkg.Tool {
	return orchestratorpkg.NewProcessDocTool(o, agentID)
}

func NewQueryQueueTool(o *Orchestrator) toolspkg.Tool {
	return orchestratorpkg.NewQueryQueueTool(o)
}
