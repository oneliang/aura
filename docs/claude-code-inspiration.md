# Claude Code System Prompts 启发分析

> 来源：`/Users/oneliang/AgentProjects/claude-code-system-prompts`（Piebald AI 提取的 Claude Code v2.1.117 系统提示词）
> 日期：2026-04-23
> 状态：待逐项讨论

---

## 1. Adversarial Verification（对抗性验证）

Claude Code 有一个 **Verification Specialist** agent，专门用来"挑刺"——不是简单地看代码能不能跑，而是主动用边界值、并发、幂等性、孤儿操作等角度去攻击实现。

**Aura 可借鉴**：在 ReAct 循环或 explicit planning 完成后，加一个 self-verification 步骤，让 LLM 以批评者视角审查自己的方案。这对 agent delegation 的子 agent 结果验证尤其有价值。

**关键文件**：
- `modules/core/pkg/engine/react_loop.go`
- `modules/core/pkg/runtime/delegate.go`

---

## 2. Hook 系统（生命周期钩子）

Claude Code 有 9+ 种 hook 事件（PreToolUse、PostToolUse、PermissionRequest、Stop、PreCompact 等），支持 command/prompt/agent 三种钩子类型，结构化 JSON 输入输出。

**Aura 可借鉴**：在 engine 的工具执行前后加入 hook 点，允许用户配置"工具执行前自动校验参数"、"工具执行后自动格式化输出"等行为。这比硬编码在 engine 里灵活得多。

**关键文件**：
- `modules/shared/pkg/hooks/types.go`
- `modules/shared/pkg/hooks/engine.go`
- `modules/shared/pkg/hooks/exec.go`
- `modules/core/pkg/runtime/components.go` — HookSystem组件定义（新增）

---

## 3. Memory 系统的分层设计

Claude Code 的 memory 是多层的：session memory（summary.md）→ dream consolidation（夜间合并）→ pruning → staleness verification → team vs personal 分离。

**Aura 现状**：有 conversation memory（SQLite），但缺少**跨会话的工作记忆**。Aura 已经在 CLI 里有类似的 dream memory 概念（`.claude/projects/` 下的 memory 目录），但可以借鉴其 staleness verification 机制——记忆过期后自动对照当前代码状态校验。

**关键文件**：
- `modules/core/pkg/memory/base.go`
- `modules/core/pkg/memory/session_memory.go`
- `modules/core/pkg/memory/summarizer.go`
- `modules/core/pkg/memory/trim.go` — 独立提取的token修剪逻辑（新增）
- `modules/core/pkg/runtime/components.go` — SessionContext组件定义

---

## 4. Read-Only Agent 约束模式

Claude Code 的 Explore/Plan agent 有显式的 "CRITICAL: READ-ONLY MODE" 约束块，禁止文件创建、修改、删除和状态变更。

**Aura 可借鉴**：在 `modules/core/pkg/runtime/delegate.go` 中，给子 agent 增加更严格的执行约束——比如 intent recognition 阶段只读不改，delegation 阶段才允许写操作。当前 Aura 的子 agent 权限继承是统一的，缺乏阶段性的权限降级。

**关键文件**：
- `modules/core/pkg/engine/planning.go` — exploration phase实现
- `modules/core/pkg/engine/const.go` — explorationTools列表
- `modules/core/pkg/runtime/delegate.go` — sub-agent delegation
- `modules/core/pkg/runtime/components.go` — SharedResources定义（新增）
- `modules/agent/pkg/agent/agent.go` — `DisableTools` 配置

---

## 5. Structured Output Enforcement（结构化输出强制）

Claude Code 要求特定场景输出特定格式：JSON 用于摘要、VERDICT: PASS/FAIL/PARTIAL 用于验证、Markdown 表格用于批量进度。

**Aura 可借鉴**：当前 Aura 的 tool execution 结果是非结构化的文本。可以在关键工具（如 task tracking、session management）中引入结构化输出约束，让引擎更容易解析和路由。

**影响范围**：整个 ReAct 底座架构设计——从 Action 解析、Observation 聚合、到最终 Response 生成，全部受此影响。

**关键文件**：
- `modules/core/pkg/engine/action_parser.go` — Action/Thinking 解析
- `modules/core/pkg/engine/tool_executor.go` — 工具执行与 Observation
- `modules/core/pkg/engine/react_loop.go` — ReAct 循环主逻辑
- `modules/core/pkg/memory/` — 记忆存储格式
- `modules/tools/pkg/tool.go` — Tool 接口定义

---

## 6. Plan Mode 的 5 阶段工作流

Claude Code 的 plan mode 是 5 阶段：explore（并行 agent）→ design（plan agent）→ review → phase four → ExitPlanMode。

**Aura 可借鉴**：Aura 的 engine 有 explicit planning mode，但更像是一个 LLM 调用循环，没有**强制性的只读探索阶段**。可以在 explicit planning 之前加入一个"先读后写"的探索约束——LLM 必须先搜索现有实现再设计方案，避免重复造轮子。

**关键文件**：
- `modules/core/pkg/engine/planning.go` — 4阶段planning实现
- `modules/core/pkg/planner/planner.go` — LLM-driven任务分解
- `modules/core/pkg/runtime/components.go` — SessionContext组件定义

---

## 7. Cache-Aware Design（Prompt Cache 感知）

Claude Code 的 agent prompt 中有明确的 cache invalidation 指导：system prompt 变更、模型切换、工具变更都会使 cache 失效。

**Aura 可借鉴**：在 `modules/core/pkg/llm/` 中，可以利用这个知识优化 prompt 构建顺序——将不变的 system prompt 放前面、可变的用户消息放后面，最大化 Ollama/OpenAI 的 cache 命中率。

**关键文件**：
- `modules/core/pkg/llm/` — LLM 客户端
- `modules/core/pkg/factory/engine_factory.go` — Engine 构建

---

## 8. Skill 系统设计（已变更）

**Claude Code**: SkillMatcher (LLM语义+keyword fallback) → SkillInjector 两阶段匹配

**Aura 原设计**（已废弃）: SkillMatcher + SkillInjector

**Aura 当前设计**: `skill_activate` 工具 - LLM主动激活，无自动匹配

**变更说明**：
- `matcher.go` 已删除（commit d69a64b）
- 改用 `skill_activate` 工具让LLM主动请求skill指令
- SkillInjector 保留，用于去重和注入skill body到系统消息
- 去重机制: `ShouldInject()`, `MarkInjected()`

**设计差异分析**：
- Claude Code: 自动匹配 → 自动注入
- Aura: LLM调用工具 → 工具返回指令 → Engine注入

两者都满足渐进式披露需求，但Aura给LLM更多主动权。

**关键文件**：
- `modules/core/pkg/skilltool/injector.go` — 注入skill body
- `modules/core/pkg/skilltool/skill_activate.go` — skill激活工具
- `modules/core/pkg/runtime/components.go` — SkillSystem组件定义

---

## 优先级排序（2026-05更新）

| 优先级 | 改进点 | 收益 | 复杂度 | Aura现状 |
|--------|--------|------|--------|----------|
| P0 | 子 agent 阶段性权限约束 | 安全+效率 | 低 | ✅已实现（readonly mode + phase-based downgrade） |
| P0 | Explicit planning 前强制只读探索 | 避免重复实现 | - | ✅已实现 |
| P1 | 工具执行 Hook 系统 | 可扩展性 | - | ✅已实现（17种钩子） |
| P1 | 结构化输出约束 | 引擎解析可靠性 | 低 | ✅已实现（OutputSchemaProvider） |
| P2 | Memory staleness verification | 记忆准确性 | 中 | ⚠️部分（时间过期存在，语义验证可选增强） |
| P2 | LLM prompt cache 优化 | 性能+成本 | 低 | ✅已实现（分层缓存） |
| P3 | Self-verification step | 方案质量 | 中 | ✅已实现（verification-specialist agent） |

---

## 讨论记录

---

## Aura 架构变更记录（2026-04）

### 重大变更

| 变更 | 说明 | 影响 |
|------|------|------|
| Skill Matcher删除 | matcher.go删除，改为skill_activate工具模式 | 第8点设计已变更 |
| Runtime组件化 | 新增components.go，分离SharedResources/SkillSystem/AgentSystem/MCPSystem/HookSystem/SessionContext | 子agent资源共享更清晰 |
| Manager泛型化 | 新增typed_manager.go，使用Go泛型实现TypedManager | Skill/Agent CRUD类型安全 |
| i18n国际化 | 新增shared/pkg/i18n，支持en/zh-CN | 用户消息多语言 |
| LSP多语言 | 新增detector.go，支持Go/Rust/TypeScript/Python/C/C++ | code_navigate不再限于gopls |
| Memory trim提取 | trim.go独立文件，新增TrimMessagesByTokens函数 | 测试覆盖更完善 |

### 关键新增文件

- `modules/core/pkg/runtime/components.go` — 组件定义（314行）
- `modules/shared/pkg/manager/typed_manager.go` — 泛型Manager
- `modules/shared/pkg/i18n/locales/en.yaml` — 英文消息
- `modules/shared/pkg/i18n/locales/zh-CN.yaml` — 中文消息
- `modules/tools/pkg/lsp/internal/detector.go` — LSP检测器
- `modules/tools/pkg/lsp/internal/language.go` — 语言映射
- `modules/core/pkg/memory/trim.go` — Token修剪逻辑

### 已删除文件

- `modules/core/pkg/skilltool/matcher.go` — Skill匹配器（已废弃）
- `modules/core/pkg/skilltool/matcher_test.go` — 匹配器测试
- `modules/core/pkg/skilltool/matcher_integration_test.go` — 集成测试
