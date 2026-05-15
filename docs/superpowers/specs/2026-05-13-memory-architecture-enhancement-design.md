---
name: Aura Memory Architecture Enhancement
description: 重构 Aura 记忆架构，采用 Content Blocks 模式对齐 Claude Code 实现
type: project
---

# Aura Memory Architecture Enhancement Spec

## Problem Statement

Aura 当前记忆架构仅持久化用户消息和 LLM 最终回复，导致：
- Tool 调用结果丢失，无法跨轮次引用
- Thinking/推理内容不存储，无法追溯决策过程
- 多步骤任务执行路径丢失
- LLM 在复杂任务中显得"笨"

## Solution

采用 Content Blocks 数组模式重构消息存储，参考 Claude Code 实现：
- 持久化 Thinking 内容
- 持久化 Tool 调用及结果
- 完整消息链追溯
- 智能选择性压缩

---

## Data Model

### ContentBlock Types

```
ContentBlock (interface)
├── TextBlock       {type: "text", text: string}
├── ThinkingBlock   {type: "thinking", thinking: string, signature?: string}
├── ToolUseBlock    {type: "tool_use", id: string, name: string, input: json}
└── ToolResultBlock {type: "tool_result", tool_use_id: string, content: ContentBlock[], is_error?: bool}
```

### Message Structure

```
Message {
  role: string (user/assistant/system/tool)
  content: ContentBlock[]

  // Required metadata
  uuid: string
  parent_uuid?: string
  timestamp: int64
  session_id: string
  type: string
  subtype?: string
  is_sidechain: bool
  cwd: string
  git_branch: string

  // Model info (assistant)
  model?: string
  usage?: {input_tokens, output_tokens}

  // Permission (user)
  permission_mode?: string
}
```

---

## Storage Format

JSONL per session, messages stored with full content blocks:

```jsonl
{"type":"user","uuid":"...","message":{"role":"user","content":[{"type":"text","text":"..."}]},...}
{"type":"assistant","uuid":"...","message":{"role":"assistant","content":[{"type":"thinking","thinking":"..."},{"type":"tool_use",...},{"type":"text","text":"..."}],"model":"...","usage":{...}},...}
{"type":"user","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"...","content":[{"type":"text","text":"..."}]}]},...}
{"type":"system","subtype":"compact_boundary","compact_metadata":{"trigger":"auto","pre_tokens":175225}}
```

---

## LLM Conversion

Different APIs require different message formats:

- **Anthropic**: Direct content blocks array
- **OpenAI**: Split content and tool_calls for assistant messages

---

## Compression Strategy

Selective compression preserves critical information:

| Content Type | Recent N | Older |
|-------------|----------|-------|
| Tool result | Full (N=10) | Summary ("Read success, X lines") |
| Thinking | Full (N=5) | Key conclusion summary |
| Tool use | Full | Keep invocation (what was called) |
| User text | Full | Keep key instructions |
| Assistant text | Full | Keep final replies |

Compact boundary marker records compression event.

---

## Implementation Phases

1. **Phase 1**: ContentBlock types + Message structure refactor
2. **Phase 2**: JSONL storage adaptation + Tool persistence
3. **Phase 3**: Thinking persistence + Compression mechanism
4. **Phase 4**: Migration + Backward compatibility

---

## Files to Modify

- `modules/shared/pkg/memory/memory.go` - Message structure
- `modules/shared/pkg/memory/content_block.go` - NEW: ContentBlock types
- `modules/core/pkg/memory/base.go` - Adapt to new structure
- `modules/core/pkg/memory/session_memory.go` - Persist all message types
- `modules/core/pkg/memory/compact.go` - NEW: Selective compression
- `modules/storage/pkg/message/message.go` - Storage message format
- `modules/storage/pkg/jsonl/store.go` - JSONL adapter
- `modules/core/pkg/engine/react_loop.go` - Record full blocks
- `modules/core/pkg/llm/converters.go` - NEW: API format converters

---

## Verification

1. Basic message storage (user → assistant with thinking/text)
2. Tool invocation storage (tool_use → tool_result)
3. Cross-turn reference ("what was that result?")
4. Session resume (restart → load → continue)
5. Compression trigger (exceed limit → compress → compact_boundary)

---

## Why This Matters

Claude Code's intelligence comes from complete memory including thinking, tools, decisions. This upgrade brings Aura to the same capability level.