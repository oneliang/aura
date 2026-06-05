package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/oneliang/aura/core/pkg/permissions"
	"github.com/oneliang/aura/shared/pkg/constants"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/hooks"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/logger"
	"github.com/oneliang/aura/shared/pkg/utils"
	tools "github.com/oneliang/aura/tools/pkg"
)

// validateParameters validates action parameters against the tool's input schema.
// Returns an error string if validation fails, empty string if valid or no schema.
func (e *Engine) validateParameters(action *ToolAction) string {
	tool, exists := e.regTools[action.Tool]
	if !exists {
		return constants.MsgToolNotFoundShort
	}
	isp, ok := tool.(InputSchemaProvider)
	if !ok {
		return ""
	}
	schema := isp.InputSchema()
	if schema == nil {
		return ""
	}
	if required, ok := schema["required"].([]string); ok {
		for _, key := range required {
			if _, exists := action.Parameters[key]; !exists {
				return fmt.Sprintf(constants.MsgMissingRequiredParam, key)
			}
		}
	}
	if props, ok := schema["properties"].(map[string]any); ok {
		for key, val := range action.Parameters {
			if prop, ok := props[key].(map[string]any); ok {
				if expectedType, ok := prop["type"].(string); ok {
					if !typeMatches(val, expectedType) {
						return fmt.Sprintf(constants.MsgParamTypeMismatch, key, expectedType, val)
					}
				}
			}
		}
	}
	return ""
}

// typeMatches checks if a Go value matches a JSON schema type string.
func typeMatches(val any, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := val.(string)
		return ok
	case "number", "integer":
		// JSON unmarshaling produces float64
		if _, ok := val.(float64); ok {
			return true
		}
		// Go tools may return int/int64 directly
		switch val.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		}
		return false
	case "boolean":
		_, ok := val.(bool)
		return ok
	case "array":
		// Check if it's any kind of slice/array
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
			return true
		}
		return false
	case "object":
		// Check if it's any kind of map
		rv := reflect.ValueOf(val)
		if rv.Kind() == reflect.Map {
			return true
		}
		return false
	default:
		return true
	}
}

// validateOutputSchema validates a tool's ToolResult.Data against its declared
// OutputSchema. Returns an error string if validation fails, empty string if valid
// or no schema declared.
func (e *Engine) validateOutputSchema(toolName string, result *tools.ToolResult) string {
	tool, exists := e.regTools[toolName]
	if !exists {
		return ""
	}
	osp, ok := tool.(tools.OutputSchemaProvider)
	if !ok {
		return ""
	}
	schema := osp.OutputSchema()
	if schema == nil {
		return ""
	}
	if len(result.Data) == 0 {
		return ""
	}
	return validateSchema(schema, result.Data, toolName)
}

// validateSchema recursively validates data against a JSON schema.
// Returns empty string if valid, error description if invalid.
func validateSchema(schema, data map[string]any, toolName string) string {
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return ""
	}

	requiredSet := make(map[string]bool)
	if req, ok := schema["required"].([]string); ok {
		for _, r := range req {
			requiredSet[r] = true
		}
	}
	// Also handle []any (from some schema providers)
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				requiredSet[s] = true
			}
		}
	}

	// Check required fields
	for key := range requiredSet {
		if _, exists := data[key]; !exists {
			return fmt.Sprintf(i18n.T("tool.output_missing_field"), toolName, key)
		}
	}

	// Validate types of present fields
	for key, val := range data {
		propSchema, ok := props[key].(map[string]any)
		if !ok {
			continue
		}
		// If the prop schema is nested (action-based like task tool), validate recursively
		propType, hasType := propSchema["type"].(string)
		if hasType && propType == "object" {
			if childData, ok := val.(map[string]any); ok {
				if childSchema, ok := propSchema["properties"].(map[string]any); ok {
					childSchemaMap := map[string]any{
						"properties": childSchema,
						"required":   propSchema["required"],
					}
					if err := validateSchema(childSchemaMap, childData, toolName); err != "" {
						return err
					}
				}
			}
		} else if hasType {
			if !typeMatches(val, propType) {
				return fmt.Sprintf(i18n.T("tool.output_type_mismatch"), toolName, key, propType, val)
			}
		}
	}
	return ""
}

// executeTool executes a tool action and returns the full ToolResult.
func (e *Engine) executeTool(ctx context.Context, action *ToolAction) (*tools.ToolResult, error) {
	// Step 1: Check tool exists
	tool, checkErr := e.checkToolExists(action.Tool)
	if checkErr != nil {
		return checkErr, nil
	}

	// Step 2: Check tool permission
	if !e.isToolAllowed(action.Tool) {
		return e.toolNotAllowedResult(action.Tool), nil
	}

	// Step 3: Validate parameters
	if validationErr := e.validateParameters(action); validationErr != "" {
		return e.invalidParamsResult(validationErr), nil
	}

	// Step 4: Handle confirmation if required
	approved, confErr := e.handleToolConfirmation(ctx, tool, action)
	if confErr != nil {
		return nil, confErr
	}
	if !approved {
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: constants.MsgDeniedByUser}, nil
	}

	// Step 5: Fire PreToolUse hook (blocking)
	if blocked := e.firePreToolUseHook(ctx, action); blocked != nil {
		return blocked, nil
	}

	// Step 6: Execute tool and validate output
	return e.executeToolCore(ctx, tool, action)
}

// checkToolExists verifies the tool is registered.
// Returns the tool or a ToolResult error.
func (e *Engine) checkToolExists(toolName string) (tools.Tool, *tools.ToolResult) {
	tool, exists := e.regTools[toolName]
	if !exists {
		return nil, &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf(constants.MsgToolNotFound, toolName)}
	}
	return tool, nil
}

// toolNotAllowedResult creates error result for disallowed tool.
func (e *Engine) toolNotAllowedResult(toolName string) *tools.ToolResult {
	e.logger.Debug("executeTool: tool not allowed in current phase", "module", "engine", "tool", toolName)
	return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf(i18n.T("tool.not_allowed_phase"), toolName)}
}

// invalidParamsResult creates error result for invalid parameters.
func (e *Engine) invalidParamsResult(validationErr string) *tools.ToolResult {
	e.logger.Warn("executeTool: parameter validation failed", "error", validationErr)
	return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf(constants.MsgInvalidParams, validationErr)}
}

// handleToolConfirmation checks if tool requires confirmation and handles it.
// Returns (approved, error) - approved is false if denied, error if handler failed.
func (e *Engine) handleToolConfirmation(ctx context.Context, tool tools.Tool, action *ToolAction) (bool, error) {
	e.logger.Debug("executeTool: checking confirmation requirement", "module", "engine", "tool", action.Tool)

	// Check PermissionTool interface (preferred)
	if permTool, ok := tool.(PermissionTool); ok {
		permLevel := permTool.PermissionLevel()
		if parsePermissionLevel(permLevel).RequiresConfirmation() {
			return e.requestConfirmation(ctx, action.Tool, action.Parameters)
		}
		e.logger.Debug("executeTool: tool does NOT require confirmation", "module", "engine", "tool", action.Tool, "perm_level", permLevel)
		return true, nil
	}

	// Check SensitiveTool interface (fallback)
	if sensitive, ok := tool.(SensitiveTool); ok && sensitive.RequiresConfirmation() {
		return e.requestConfirmation(ctx, action.Tool, action.Parameters)
	}

	e.logger.Debug("executeTool: tool is NOT sensitive", "module", "engine", "tool", action.Tool)
	return true, nil
}

// requestConfirmation requests user confirmation for tool execution.
// Returns (approved, error) - approved is false if denied.
func (e *Engine) requestConfirmation(ctx context.Context, toolName string, params map[string]any) (bool, error) {
	if e.config.ConfirmationHandler == nil {
		e.logger.Debug("executeTool: no confirmation handler", "module", "engine", "tool", toolName)
		return true, nil
	}

	e.logger.Debug("executeTool: calling confirmation handler", "module", "engine", "tool", toolName)
	approved, err := e.config.ConfirmationHandler(ctx, toolName, params)
	if err != nil {
		return false, err
	}

	e.logger.Debug("executeTool: confirmation result", "module", "engine", "tool", toolName, "approved", approved)
	return approved, nil
}

// firePreToolUseHook fires the PreToolUse hook and returns blocking result if applicable.
func (e *Engine) firePreToolUseHook(ctx context.Context, action *ToolAction) *tools.ToolResult {
	hookResult, hookErr := e.hookEngine.FireBlockingWithToolName(ctx, hooks.EventPreToolUse, action.Tool, map[string]any{
		"tool_name":  action.Tool,
		"tool_input": action.Parameters,
	})
	if hookErr != nil {
		e.logger.Warn("executeTool: PreToolUse hook error", "error", hookErr.Error(), "tool", action.Tool)
		return nil
	}
	if hookResult == nil {
		return nil
	}
	if e.hookEngine.ShouldBlock(hookResult) {
		reason := constants.MsgHookBlockedReason
		if hookResult.Parsed != nil && hookResult.Parsed.StopReason != "" {
			reason = hookResult.Parsed.StopReason
		}
		return &tools.ToolResult{Status: tools.ToolStatusError, Error: fmt.Sprintf(constants.MsgHookBlocked, reason)}
	}
	return nil
}

// executeToolCore executes the tool and validates output schema.
// Each tool gets an independent timeout context to avoid timeout accumulation
// from previous confirmations or tool executions.
func (e *Engine) executeToolCore(ctx context.Context, tool tools.Tool, action *ToolAction) (*tools.ToolResult, error) {
	// Plan mode write guard - restrict file modifications during plan mode
	if e.planModeState != PlanModeStateNone && e.planModeState != PlanModeStateExecute && e.planModeState != PlanModeStateVerify {
		toolName := tool.Name()

		// Check file write/edit operations
		if toolName == "file_write" || toolName == "file_edit" {
			targetPath, _ := action.Parameters["path"].(string)
			if targetPath != e.planFilePath {
				return &tools.ToolResult{
					Status: tools.ToolStatusError,
					Error:  fmt.Sprintf("Plan mode restricts writes to plan file only. Current plan file: %s", e.planFilePath),
				}, nil
			}
		}

		// Check bash commands that modify files
		if toolName == "bash" {
			cmd, _ := action.Parameters["command"].(string)
			if isModifyingCommand(cmd) {
				return &tools.ToolResult{
					Status: tools.ToolStatusError,
					Error:  "Plan mode restricts file modifications. Wait for execution phase.",
				}, nil
			}
		}
	}

	// Create independent timeout context for this tool execution
	// Use e.ctx (engine background context) as base to avoid inheriting accumulated timeouts
	execCtx, execCancel := e.createToolExecutionContext(ctx, tool)
	defer execCancel()

	// Check if cancelled before starting
	select {
	case <-execCtx.Done():
		return nil, execCtx.Err()
	default:
	}

	// Run in goroutine with context monitoring
	type result struct {
		value *tools.ToolResult
		err   error
	}
	done := make(chan result, 1)
	go func() {
		res, err := tool.Execute(execCtx, action.Parameters)
		done <- result{res, err}
	}()

	select {
	case <-execCtx.Done():
		// Drain channel to prevent goroutine leak
		go func() {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
			}
		}()
		return nil, execCtx.Err()
	case r := <-done:
		if r.err != nil {
			return nil, r.err
		}
		if r.value == nil {
			return nil, fmt.Errorf(constants.MsgNilToolResult, action.Tool)
		}
		// Validate output against declared output schema
		e.validateAndAugmentOutput(action.Tool, r.value)
		return r.value, nil
	}
}

// createToolExecutionContext creates an independent timeout context for tool execution.
// Base context is e.ctx (engine background) to avoid inheriting accumulated timeouts,
// but we also monitor the request context (ctx) for user cancellation.
func (e *Engine) createToolExecutionContext(ctx context.Context, tool tools.Tool) (context.Context, context.CancelFunc) {
	// Get timeout from tool if it implements TimeoutProvider
	timeout := constants.DefaultToolTimeout
	if tp, ok := tool.(tools.TimeoutProvider); ok {
		timeout = tp.Timeout()
	}

	// Create context with timeout from engine background context
	// This gives each tool a fresh timeout, independent of previous operations
	execCtx, execCancel := context.WithTimeout(e.ctx, timeout)

	// Also monitor request context for user cancellation
	// If request context is cancelled, we should cancel tool execution too
	go func() {
		select {
		case <-ctx.Done():
			execCancel()
		case <-execCtx.Done():
			// Tool timeout or execution completed, no need to monitor request ctx anymore
		}
	}()

	return execCtx, execCancel
}

// validateAndAugmentOutput validates output schema and augments content with warnings.
func (e *Engine) validateAndAugmentOutput(toolName string, result *tools.ToolResult) {
	if schemaErr := e.validateOutputSchema(toolName, result); schemaErr != "" {
		e.logger.Warn("executeTool: output schema validation failed", "tool", toolName, "error", schemaErr)
		result.Content += fmt.Sprintf("\n[Output schema validation warning: %s]", schemaErr)
	}
}

// parsePermissionLevel parses a permission level string.
func parsePermissionLevel(level string) permissions.PermissionLevel {
	switch level {
	case "read", "readonly":
		return permissions.PermissionReadOnly
	case "write":
		return permissions.PermissionWrite
	case "execute", "exec":
		return permissions.PermissionExecute
	case "admin":
		return permissions.PermissionAdmin
	default:
		// Default to execute (requires confirmation) for safety
		return permissions.PermissionExecute
	}
}

// generateExecutionID generates a unique ID for tool execution tracking.
// Uses short UUID (8 characters) for global uniqueness guarantee.
func generateExecutionID() string {
	return uuid.New().String()[:8]
}

// emitToolStartEvent emits a tool start event with execution ID for precise tracking.
func emitToolStartEvent(log *logger.Logger, eventsCh chan<- events.Event, toolName string, params map[string]any, requestID, executionID string) {
	paramsJSON, _ := json.Marshal(params)
	// 发送事件（先发送，后记录日志以验证实际发送顺序）
	eventsCh <- events.NewEventWithExtra(
		events.EventTypeToolStart,
		toolName,
		map[string]any{
			"params":       string(paramsJSON),
			"execution_id": executionID,
		},
		requestID,
	)
	log.Debug("emitToolStartEvent: 发送完成", "execution_id", executionID, "tool", toolName, "params", string(paramsJSON))
}

// emitToolEndEvent emits a tool end event with execution ID for precise matching.
func emitToolEndEvent(log *logger.Logger, eventsCh chan<- events.Event, toolName, result string, duration time.Duration, requestID, executionID string) {
	// 发送事件（先发送，后记录日志以验证实际发送顺序）
	eventsCh <- events.NewEventWithExtra(
		events.EventTypeToolEnd,
		utils.Truncate(result, toolEndEventTruncateLen),
		map[string]any{
			"tool":         toolName,
			"result":       utils.Truncate(result, toolEndResultPreviewLen),
			"duration":     duration,
			"execution_id": executionID,
		},
		requestID,
	)
	log.Debug("emitToolEndEvent: 发送完成", "execution_id", executionID, "tool", toolName)
}

// executeToolWithEvents executes a single tool action, emitting start/end events
// with execution ID and returning the result. On failure, emits an error event.
func (e *Engine) executeToolWithEvents(ctx context.Context, action *ToolAction, eventsCh chan<- events.Event, requestID string) (*tools.ToolResult, error) {
	executionID := generateExecutionID()
	emitToolStartEvent(e.logger, eventsCh, action.Tool, action.Parameters, requestID, executionID)
	startTime := time.Now()
	result, err := e.executeTool(ctx, action)
	duration := time.Since(startTime)
	if err != nil {
		eventsCh <- events.NewEvent(events.EventTypeError, fmt.Sprintf("Error: %v", err), requestID)
		emitToolEndEvent(e.logger, eventsCh, action.Tool, err.Error(), duration, requestID, executionID)
		return nil, err
	}
	content := ""
	if result != nil {
		if result.Status == tools.ToolStatusError {
			content = result.Error
		} else {
			content = result.Content
		}
	}
	emitToolEndEvent(e.logger, eventsCh, action.Tool, content, duration, requestID, executionID)
	return result, nil
}

// aggregateToolObservations aggregates tool results into observation string.
// Returns the aggregated observation and the first error encountered (if any).
func aggregateToolObservations(results []toolResult, actions []*ToolAction) (string, error) {
	if len(results) == 0 {
		return i18n.T("tool.observation.no_output"), nil
	}

	if len(results) == 1 {
		tr := results[0]
		if tr.Err != nil {
			return "", tr.Err
		}
		if tr.Result != nil {
			return tr.Result.Content, nil
		}
		return i18n.T("tool.observation.no_output"), nil
	}

	// Multi-tool: aggregate all observations
	var obsSb strings.Builder
	var firstErr error
	for i, action := range actions {
		tr := results[i]
		if tr.Err != nil {
			obsSb.WriteString(fmt.Sprintf(i18n.T("tool.observation.error_format"), action.Tool, tr.Err))
			if firstErr == nil {
				firstErr = tr.Err
			}
		} else if tr.Result != nil {
			obsSb.WriteString(fmt.Sprintf(i18n.T("tool.observation.output_format"), action.Tool, tr.Result.Content))
		} else {
			obsSb.WriteString(fmt.Sprintf(i18n.T("tool.observation.output_empty"), action.Tool))
		}
	}
	return obsSb.String(), firstErr
}

// isModifyingCommand checks if a bash command modifies files.
// Returns true if the command contains patterns that would modify the filesystem.
func isModifyingCommand(cmd string) bool {
	modifyingPatterns := []string{
		"rm ", "rm -", "rmdir ",
		"mv ", "cp ", "mkdir ",
		"touch ", "chmod ", "chown ",
		">", ">>",
		"sed -i", "sed -i ",
		"git rm", "git mv",
		"git checkout", "git reset", "git stash",
	}
	for _, pattern := range modifyingPatterns {
		if strings.Contains(cmd, pattern) {
			return true
		}
	}
	return false
}
