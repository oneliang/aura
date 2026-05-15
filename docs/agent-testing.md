# Agent 手工测试指南

## 前置检查

```bash
# 1. 确认 Agent 已加载
ls ~/.aura/agents/
# 应看到：api-designer, bug-fix-engineer, code-reviewer, documentation-writer, refactoring-engineer, test-writer

# 2. 确认配置中 agents 启用
cat ~/.aura/config.yaml | grep -A 3 "agents:"
# 应看到: enabled: true
```

## 测试方法

### 方式一：交互模式（推荐）

```bash
./bin/aura
```

输入以下请求测试各个 Agent：

| Agent | 输入请求 |
|-------|----------|
| **code-reviewer** | `帮我审查 modules/core/pkg/engine/engine.go 的代码质量` |
| **test-writer** | `给 modules/core/pkg/runtime/runtime.go 的 createMemory 方法写单元测试` |
| **refactoring-engineer** | `重构 modules/core/pkg/runtime/delegate.go，消除重复逻辑` |
| **bug-fix-engineer** | `帮我分析一下为什么 agent delegation 没有触发，可能的原因是什么` |
| **api-designer** | `设计一个管理用户会话的 RESTful API，包括创建、查询、删除` |
| **documentation-writer** | `给 modules/core/pkg/runtime 包写一份 README 文档` |

### 方式二：TUI 全屏模式（视觉效果更好）

```bash
./bin/aura --tui
```

同样的请求，在 TUI 中能看到状态栏动画、事件流等更直观的效果。

## 验证委托成功

看到以下任一标志即表示委托已触发：

1. **LLM 输出委托指令**：`Command: delegate_to_agent` 或 `Action: {"command": "command_delegate_to_agent", ...}`
2. **子 Agent 开始执行**：出现新的 Thinking 阶段（状态栏有动画）
3. **返回结果格式**：`【SubAgent xxx completed the task】` 开头

## 验证调试日志

委托执行后，查看委托日志：

```bash
# 查看最近的委托记录
cat ~/.aura/logs/agent_delegation.log | python3 -m json.tool

# 只看特定 agent 的委托
cat ~/.aura/logs/agent_delegation.log | grep "code-reviewer" | python3 -m json.tool

# 查看委托耗时统计
cat ~/.aura/logs/agent_delegation.log | jq '{event, agent_name, duration_ms, path}'
```

日志格式（JSONL，每行一个 JSON 对象）：
- `start` — 委托开始，含 agent_name, task_preview
- `step` — 委托步骤，含 step 名称和耗时
- `complete` — 委托完成，含总耗时和路径（fast/full）
- `error` — 委托错误，含错误信息和步骤

## 调试日志位置

- 委托日志：`~/.aura/logs/agent_delegation.log`
- LLM 日志：`~/.aura/logs/llm_requests.log`
