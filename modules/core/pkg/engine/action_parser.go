package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oneliang/aura/core/pkg/llm"
	"github.com/oneliang/aura/shared/pkg/events"
	"github.com/oneliang/aura/shared/pkg/i18n"
	"github.com/oneliang/aura/shared/pkg/memory"
	tools "github.com/oneliang/aura/tools/pkg"
)

// toolCallsToActions converts LLM tool calls to ToolActions.
// This is the primary path when the provider supports structured tool use.
func (e *Engine) toolCallsToActions(toolCalls []llm.ToolCall) []*ToolAction {
	var actions []*ToolAction
	for _, tc := range toolCalls {
		toolName := tc.Name
		if toolName == "" {
			continue
		}

		// Route command_* through internal_command or verify tool exists
		action := e.verifyAndRouteTool(toolName, tc.Parameters)
		if action != nil {
			actions = append(actions, action)
		}
	}
	return actions
}

// parseAction parses a tool action from LLM response.
// Returns the parsed action and true if an action was found, false otherwise.
// Supports two formats:
//  1. JSON format: Action: {"tool": "tool_name", "parameters": {"param1": "value1"}}
//  2. Multiline format: Action: tool_name\nParameters: {"param1": "value1"}
//  3. Bare JSON: JSON objects in response without Action: prefix (e.g. backtick-wrapped)
func (e *Engine) parseAction(response string) (*ToolAction, bool) {
	// Try JSON format with Action: prefix
	if action, ok := e.parseActionJSON(response); ok {
		return action, true
	}

	// Try multiline format
	if action, ok := e.parseActionMultiline(response); ok {
		return action, true
	}

	// Fallback: try bare JSON without Action: prefix (handles backtick-wrapped JSON)
	return e.parseActionBareJSON(response)
}

// findJSONBounds finds the start and end indices of a balanced JSON object
// in s, starting the search at start. Returns (start, end) where end is the
// index after the closing brace, or (-1, -1) if no balanced object found.
func findJSONBounds(s string, start int) (int, int) {
	if start < 0 || start >= len(s) {
		return -1, -1
	}
	braceCount := 0
	for i := start; i < len(s); i++ {
		if s[i] == '{' {
			braceCount++
		} else if s[i] == '}' {
			braceCount--
			if braceCount == 0 {
				return start, i + 1
			}
		}
	}
	return -1, -1
}

// routeCommandTool routes command_* tools through internal_command if available.
// Returns the routed ToolAction if routing applies, nil otherwise.
func (e *Engine) routeCommandTool(toolName string, params map[string]any) *ToolAction {
	if !strings.HasPrefix(toolName, "command_") {
		return nil
	}
	if _, exists := e.regTools["internal_command"]; !exists {
		return nil
	}
	return &ToolAction{
		Tool: "internal_command",
		Parameters: map[string]any{
			"command": toolName,
			"params":  params,
		},
	}
}

// verifyAndRouteTool verifies tool exists and routes command_* through internal_command.
// Returns the final ToolAction (possibly routed) or nil if tool doesn't exist.
func (e *Engine) verifyAndRouteTool(toolName string, params map[string]any) *ToolAction {
	// Try direct tool lookup
	if _, exists := e.regTools[toolName]; exists {
		return &ToolAction{Tool: toolName, Parameters: params}
	}
	// Try routing through internal_command
	return e.routeCommandTool(toolName, params)
}

// extractActions extracts tool actions from response, preferring tool calls.
// This unifies the pattern: toolCallsToActions first, then parseActions fallback.
func (e *Engine) extractActions(response string, toolCalls []llm.ToolCall) []*ToolAction {
	if len(toolCalls) > 0 {
		return e.toolCallsToActions(toolCalls)
	}
	return e.parseActions(response)
}

// parseActions parses ALL tool actions from an LLM response.
// Returns a slice of parsed actions. Empty slice if no actions found.
// Supports parallel tool execution: LLM outputs multiple "Action:" lines consecutively.
func (e *Engine) parseActions(response string) []*ToolAction {
	var actions []*ToolAction

	// Try to find all "Action:" lines with JSON objects
	remaining := response
	for {
		idx := strings.Index(strings.ToLower(remaining), "action:")
		if idx == -1 {
			break
		}

		// Find the start of the JSON object after "Action:"
		jsonStart := strings.Index(remaining[idx:], "{")
		if jsonStart == -1 {
			// No JSON after this Action: — try multiline format.
			// Note: only extracts toolName; Parameters: on subsequent lines are NOT parsed.
			line := strings.TrimSpace(remaining[idx+actionPrefixLen:])
			end := strings.Index(line, "\n")
			if end == -1 {
				end = len(line)
			}
			toolName := strings.TrimSpace(line[:end])
			if toolName == "" {
				break
			}
			if _, exists := e.regTools[toolName]; exists {
				actions = append(actions, &ToolAction{Tool: toolName})
			}
			remaining = remaining[idx+len("action:"):]
			continue
		}
		jsonStart += idx

		// Find the matching closing brace
		jsonStartIdx, jsonEndIdx := findJSONBounds(remaining, jsonStart)
		if jsonEndIdx == -1 {
			break
		}

		jsonStr := remaining[jsonStartIdx:jsonEndIdx]
		if action, ok := e.tryParseToolActionJSON(jsonStr); ok {
			actions = append(actions, action)
		}
		remaining = remaining[jsonEndIdx:]
	}

	// Fallback: if no "Action:" found, try bare JSON (single action)
	if len(actions) == 0 {
		if action, ok := e.parseActionBareJSON(response); ok {
			actions = append(actions, action)
		}
	}

	return actions
}

// parseActionJSON parses the JSON format: Action: {"tool": "tool_name", "parameters": {...}}
// Also supports alias format: Action: {"command": "command_name", "params": {...}}
func (e *Engine) parseActionJSON(response string) (*ToolAction, bool) {
	// Try to find Action: pattern with proper JSON matching
	actionIdx := strings.Index(strings.ToLower(response), "action:")
	if actionIdx == -1 {
		return nil, false
	}

	// Find the start of the JSON object
	jsonStart := strings.Index(response[actionIdx:], "{")
	if jsonStart == -1 {
		return nil, false
	}
	jsonStart += actionIdx

	// Find the matching closing brace
	jsonStartIdx, jsonEndIdx := findJSONBounds(response, jsonStart)
	if jsonEndIdx == -1 {
		return nil, false
	}

	jsonStr := response[jsonStartIdx:jsonEndIdx]

	// Try standard format first (tool/parameters)
	var action ToolAction
	if err := json.Unmarshal([]byte(jsonStr), &action); err == nil {
		if action.Tool != "" {
			// Verify tool exists or route command_*
			result := e.verifyAndRouteTool(action.Tool, action.Parameters)
			if result != nil {
				return result, true
			}
		}
	}

	// Try alias format (command/params)
	var aliasAction struct {
		Command string         `json:"command"`
		Params  map[string]any `json:"params"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &aliasAction); err == nil {
		if aliasAction.Command != "" {
			// Route command_* or verify tool exists
			result := e.verifyAndRouteTool(aliasAction.Command, aliasAction.Params)
			if result != nil {
				return result, true
			}
		}
	}

	return nil, false
}

// parseActionMultiline parses the multiline format: Action: tool_name\nParameters: {...}
func (e *Engine) parseActionMultiline(response string) (*ToolAction, bool) {
	lines := strings.Split(response, "\n")
	var toolName string
	var params map[string]any

	for _, line := range lines {
		line = strings.TrimSpace(line)
		lowerLine := strings.ToLower(line)

		if strings.HasPrefix(lowerLine, "action:") {
			// Extract tool name after "action:"
			toolName = strings.TrimSpace(line[actionPrefixLen:])
		} else if strings.HasPrefix(lowerLine, "parameters:") {
			// Extract JSON after "parameters:"
			jsonStr := strings.TrimSpace(line[parametersPrefixLen:])
			if jsonStr == "" {
				continue
			}
			if err := json.Unmarshal([]byte(jsonStr), &params); err != nil {
				// Failed to parse parameters
				return nil, false
			}
		}
	}

	if toolName == "" {
		return nil, false
	}

	// Verify tool exists or route command_*
	result := e.verifyAndRouteTool(toolName, params)
	if result == nil {
		return nil, false
	}
	return result, true
}

// parseActionBareJSON attempts to parse bare JSON from the response without requiring
// an "Action:" prefix. This handles cases where the LLM wraps JSON in markdown
// code blocks (backticks) without the Action: prefix.
func (e *Engine) parseActionBareJSON(response string) (*ToolAction, bool) {
	// Scan for JSON objects by finding { and matching }
	for start := 0; start < len(response); {
		braceIdx := strings.Index(response[start:], "{")
		if braceIdx == -1 {
			break
		}
		braceIdx += start

		// Find matching closing brace
		s, end := findJSONBounds(response, braceIdx)
		if end == -1 {
			start = braceIdx + 1
			continue
		}

		jsonStr := response[s:end]
		if action, ok := e.tryParseToolActionJSON(jsonStr); ok {
			return action, true
		}
		start = braceIdx + 1
	}

	return nil, false
}

// tryParseToolActionJSON tries to parse a JSON string as a tool action.
func (e *Engine) tryParseToolActionJSON(jsonStr string) (*ToolAction, bool) {
	// Try standard format first (tool/parameters)
	var action ToolAction
	if err := json.Unmarshal([]byte(jsonStr), &action); err == nil {
		if action.Tool != "" {
			// Verify tool exists or route command_*
			result := e.verifyAndRouteTool(action.Tool, action.Parameters)
			if result != nil {
				return result, true
			}
		}
	}

	// Try alias format (command/params)
	var aliasAction struct {
		Command string         `json:"command"`
		Params  map[string]any `json:"params"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &aliasAction); err == nil {
		if aliasAction.Command != "" {
			// Route command_* or verify tool exists
			result := e.verifyAndRouteTool(aliasAction.Command, aliasAction.Params)
			if result != nil {
				return result, true
			}
		}
	}

	return nil, false
}

// extractThinking extracts thinking content from LLM response.
func (e *Engine) extractThinking(response string) string {
	actionIdx := strings.Index(strings.ToLower(response), "action:")
	if actionIdx > 0 {
		thinking := strings.TrimSpace(response[:actionIdx])
		// Strip "Thought:" or "thought:" prefix
		lowered := strings.ToLower(thinking)
		for _, prefix := range []string{"thought:", "thought: "} {
			if strings.HasPrefix(lowered, prefix) {
				thinking = strings.TrimSpace(thinking[len(prefix):])
				break
			}
		}
		return thinking
	}
	return ""
}

// getToolDescriptions returns formatted tool descriptions including input schema
// and output schema when available, so text-based providers can see full
// parameter and result constraints.
// Respects toolAllowlist — only allowed tools are described.
func (e *Engine) getToolDescriptions() string {
	available := 0
	for name := range e.regTools {
		if e.isToolAllowed(name) {
			available++
		}
	}
	if available == 0 {
		return "No tools available."
	}

	var sb strings.Builder
	for name, tool := range e.regTools {
		if !e.isToolAllowed(name) {
			continue
		}
		sb.WriteString(fmt.Sprintf("- %s: %s", name, tool.Description()))
		// Input schema (parameters)
		if isp, ok := tool.(InputSchemaProvider); ok {
			if schema := isp.InputSchema(); schema != nil {
				if params := formatSchemaParams(schema); params != "" {
					sb.WriteString("\n  Input: " + params)
				}
			}
		}
		// Output schema (result structure)
		if osp, ok := tool.(tools.OutputSchemaProvider); ok {
			if schema := osp.OutputSchema(); schema != nil {
				if props := formatSchemaParams(schema); props != "" {
					sb.WriteString("\n  Output: " + props)
				}
			}
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// formatSchemaParams extracts a concise parameter list from a JSON schema.
// Returns "param (type, required), ..." or empty string if no properties.
func formatSchemaParams(schema map[string]any) string {
	props, _ := schema["properties"].(map[string]any)
	if len(props) == 0 {
		return ""
	}
	requiredSet := make(map[string]bool)
	if req, ok := schema["required"].([]any); ok {
		for _, r := range req {
			if s, ok := r.(string); ok {
				requiredSet[s] = true
			}
		}
	}

	var params []string
	for name, prop := range props {
		p, ok := prop.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := p["type"].(string)
		if typ == "" {
			typ = "any"
		}
		param := fmt.Sprintf("%s (%s", name, typ)
		if requiredSet[name] {
			param += ", required"
		}
		if desc, ok := p["description"].(string); ok && desc != "" {
			// Truncate long descriptions
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}
			param += ", " + desc
		}
		param += ")"
		params = append(params, param)
	}
	return strings.Join(params, "; ")
}

// getReActSystemPrompt builds the ReAct system prompt.
func (e *Engine) getReActSystemPrompt() string {
	toolDescriptions := e.getToolDescriptions()

	basePrompt := e.config.SystemPrompt
	if basePrompt == "" {
		basePrompt = DefaultSystemPrompt
	}

	// Try to get localized ReAct format from i18n
	reactFormat := i18n.T("agent.react_format")
	// If i18n not initialized or key not found, use English fallback
	if reactFormat == "agent.react_format" || reactFormat == "" {
		prompt := fmt.Sprintf(`%s

You have access to the following tools:
%s

To use a tool, use the following format:

Action: {"tool": "tool_name", "parameters": {"param1": "value1"}}
Observation: [result of the tool]
... (this Thought/Action/Observation can repeat N times)

When you have a final answer, respond directly without using Action format.

Important rules:
1. Only use tools when necessary
2. Actions MUST use the exact JSON format shown above: Action: {valid JSON}
3. NEVER use XML-style tags, function wrappers, or any other format (e.g., NEVER output <tool>, <function>, {{, or similar)
4. After receiving an observation, continue thinking or provide final answer
5. Be concise in your responses
6. When multiple independent tools can be called at once, output multiple Action: lines consecutively. They will execute in parallel.
7. Tool observations may contain "Structured Data" sections with JSON — use this structured data to make more informed decisions about next steps.`, basePrompt, toolDescriptions)

		return prompt
	}

	// Use localized format - also append important rules
	return fmt.Sprintf(`%s

You have access to the following tools:
%s

%s

Important rules:
1. Only use tools when necessary
2. Actions MUST use the exact JSON format shown above: Action: {valid JSON}
3. NEVER use XML-style tags, function wrappers, or any other format (e.g., NEVER output <tool>, <function>, {{, or similar)
4. After receiving an observation, continue thinking or provide final answer
5. Be concise in your responses
6. When multiple independent tools can be called at once, output multiple Action: lines consecutively. They will execute in parallel.
7. Tool observations may contain "Structured Data" sections with JSON — use this structured data to make more informed decisions about next steps.`,
		basePrompt, toolDescriptions, reactFormat)
}

// augmentSystemPromptWithRAG attempts to augment systemPrompt using DynamicRAG.
// Returns the augmented prompt on success, or the original systemPrompt if RAG
// is disabled, no recent conversation context exists, or augmentation fails.
func (e *Engine) augmentSystemPromptWithRAG(ctx context.Context, systemPrompt string) string {
	if !e.config.EnableDynamicRAG || e.config.DynamicRAG == nil {
		return systemPrompt
	}
	rm, ok := e.memory.(memory.RecentMemory)
	if !ok {
		return systemPrompt
	}
	recent := rm.Last(5)
	var querySb strings.Builder
	for _, msg := range recent {
		// Extract text from ContentBlocks
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(memory.TextBlock); ok {
				querySb.WriteString(tb.Text)
				querySb.WriteString(" ")
				break
			}
		}
	}
	query := querySb.String()
	if query == "" {
		return systemPrompt
	}
	augmented, err := e.config.DynamicRAG.AugmentQuery(ctx, query, systemPrompt)
	if err != nil || augmented == "" {
		return systemPrompt
	}
	return augmented
}

// buildReActMessages builds messages for ReAct loop.
// When prompt caching is enabled, system prompt is sent via PromptCacheConfig,
// not in messages array. Dynamic content (RAG, Summary) goes into messages.
func (e *Engine) buildReActMessages(ctx context.Context) []llm.Message {
	messages := make([]llm.Message, 0)

	// When caching is enabled, system prompt is handled via PromptCacheConfig
	// We only add dynamic content to messages
	if e.config.EnablePromptCache && e.config.PromptCacheConfig != nil {
		// Add RAG knowledge as separate system message (not cached)
		if rag := e.getRAGKnowledge(ctx); rag != "" {
			messages = append(messages, llm.Message{
				Role:          memory.RoleSystem,
				ContentBlocks: []memory.ContentBlock{
					memory.TextBlock{Type: memory.BlockTypeText, Text: rag},
				},
			})
		}

		// Add summary as separate system message (not cached)
		if e.config.EnableSummarization {
			if summary := e.getSummary(); summary != "" {
				messages = append(messages, llm.Message{
					Role:          memory.RoleSystem,
					ContentBlocks: []memory.ContentBlock{
						memory.TextBlock{Type: memory.BlockTypeText, Text: "Previous conversation summary:\n" + summary},
					},
				})
			}
		}

		// Add activated skill bodies as dynamic system messages (not cached)
		// Skill bodies are retrieved from SkillInjector, not from memory
		if e.config.SkillInjector != nil {
			for _, body := range e.config.SkillInjector.GetInjectedBodies() {
				messages = append(messages, llm.Message{
					Role:          memory.RoleSystem,
					ContentBlocks: []memory.ContentBlock{
						memory.TextBlock{Type: memory.BlockTypeText, Text: body},
					},
				})
			}
		}

		// Add conversation history (without summary, since we added it separately)
		messages = append(messages, e.memory.Get()...)
	} else {
		// Legacy behavior: inline system prompt into messages
		systemPrompt := e.getReActSystemPrompt()
		systemPrompt = e.augmentSystemPromptWithRAG(ctx, systemPrompt)

		messages = append(messages, llm.Message{
			Role:          memory.RoleSystem,
			ContentBlocks: []memory.ContentBlock{
				memory.TextBlock{Type: memory.BlockTypeText, Text: systemPrompt},
			},
		})

		// Add activated skill bodies (same logic as cache-aware path)
		if e.config.SkillInjector != nil {
			for _, body := range e.config.SkillInjector.GetInjectedBodies() {
				messages = append(messages, llm.Message{
					Role:          memory.RoleSystem,
					ContentBlocks: []memory.ContentBlock{
						memory.TextBlock{Type: memory.BlockTypeText, Text: body},
					},
				})
			}
		}

		if e.config.EnableSummarization {
			if sm, ok := e.memory.(memory.SummarizingMemory); ok {
				messages = append(messages, sm.GetMessagesWithSummary()...)
			} else {
				messages = append(messages, e.memory.Get()...)
			}
		} else {
			messages = append(messages, e.memory.Get()...)
		}
	}

	return messages
}

// getRAGKnowledge retrieves RAG knowledge without modifying system prompt.
func (e *Engine) getRAGKnowledge(ctx context.Context) string {
	if !e.config.EnableDynamicRAG || e.config.DynamicRAG == nil {
		return ""
	}
	rm, ok := e.memory.(memory.RecentMemory)
	if !ok {
		return ""
	}
	recent := rm.Last(5)
	var querySb strings.Builder
	for _, msg := range recent {
		// Extract text from ContentBlocks
		for _, block := range msg.GetContentBlocks() {
			if tb, ok := block.(memory.TextBlock); ok {
				querySb.WriteString(tb.Text)
				querySb.WriteString(" ")
				break
			}
		}
	}
	query := querySb.String()
	if query == "" {
		return ""
	}
	augmented, err := e.config.DynamicRAG.AugmentQuery(ctx, query, "")
	if err != nil || augmented == "" {
		return ""
	}
	return augmented
}

// getSummary returns the conversation summary if available.
func (e *Engine) getSummary() string {
	if sm, ok := e.memory.(memory.SummarizingMemory); ok {
		return sm.GetSummary()
	}
	return ""
}

// buildRequest builds an LLM request with caching configuration.
func (e *Engine) buildRequest(messages []llm.Message) *llm.Request {
	req := &llm.Request{
		Messages: messages,
		Tools:    e.buildToolSchemas(),
		Thinking: e.config.Thinking,
	}

	// Add caching configuration if enabled
	if e.config.EnablePromptCache && e.config.PromptCacheConfig != nil {
		req.PromptCache = e.config.PromptCacheConfig
	}

	return req
}

// getCompleteResponse gets a complete response from LLM.
// Returns the response text, any structured tool calls, and an error.
// Emits thinking content event if eventsCh is provided.
func (e *Engine) getCompleteResponse(ctx context.Context, messages []llm.Message, eventsCh chan<- events.Event, requestID string) (string, []llm.ToolCall, error) {
	req := e.buildRequest(messages)
	resp, err := e.client.Complete(ctx, req)
	if err != nil {
		return "", nil, err
	}
	// Strip LLM thinking content from non-streaming response
	var content string
	for _, block := range resp.Message.GetContentBlocks() {
		if tb, ok := block.(memory.TextBlock); ok {
			content = tb.Text
			break
		}
	}
	content, thinking := (&thinkingFilter{}).stripThinking(content)
	if thinking != "" && eventsCh != nil {
		eventsCh <- events.NewEvent(events.EventTypeThinkingContent, thinking, requestID)
	}
	return content, resp.ToolCalls, nil
}