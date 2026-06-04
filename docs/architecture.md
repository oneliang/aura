# Aura 架构文档

本文档描述 Aura 个人 AI 助手的系统架构和模块依赖关系。

**最后更新**: 2026-06-04

---

## 系统架构概览

```
┌─────────────────────────────────────────────────────────────────┐
│                        Application Layer                        │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │     CLI     │  │     API     │  │  Adapters   │             │
│  │  (Cobra +   │  │   (HTTP +   │  │ (Feishu,    │             │
│  │  Bubbletea) │  │  Web UI)    │  │  Email)     │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                          Core Layer                             │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │                      Engine (ReAct, Planning)               ││
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   ││
│  │  │   LLM    │  │  ReAct   │  │  Memory  │  │ Tools    │   ││
│  │  │  Client  │  │  Loop    │  │  System  │  │ Executor │   ││
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘   ││
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐                 ││
│  │  │ Planner  │  │Permis-   │  │ Sum-     │                 ││
│  │  │ (Task    │  │sions     │  │marizer   │                 ││
│  │  │ Planning)│  │ Manager) │  │          │                 ││
│  │  └──────────┘  └──────────┘  └──────────┘                 ││
│  └─────────────────────────────────────────────────────────────┘│
│  ┌─────────────────────────────────────────────────────────────┐│
│  │              Multi-Agent Orchestrator                       ││
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   ││
│  │  │   Doc    │  │   Task   │  │ Super    │  │  Sub     │   ││
│  │  │Coordinator│ │ Registry │  │ visor    │  │  Agents  │   ││
│  │  └──────────┘  └──────────┘  └──────────┘  └──────────┘   ││
│  │  ┌──────────┐                                              ││
│  │  │Workspace │                                              ││
│  │  │ Isolator │                                              ││
│  │  └──────────┘                                              ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       Service Layer                             │
│  ┌─────────────────────────┐  ┌─────────────────────────────┐  │
│  │     Session Manager     │  │    Command Executor         │  │
│  │  (JSONL Storage +       │  │    (Internal Commands)      │  │
│  │   Subscriptions)        │  │    + Skill-as-Command       │  │
│  └─────────────────────────┘  └─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                        Base Layer                               │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │  Shared  │  │  Tools   │  │Knowledge │  │ Personality  │   │
│  │ (Config, │  │   (File, │  │   (RAG,  │  │  (Profile,   │   │
│  │ Logger)  │  │  Shell,  │  │ Chroma)  │  │   Style)     │   │
│  │          │  │   SSH)   │  │          │  │              │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘   │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │  Skill   │  │ Storage  │  │  Agent   │  │   Habit      │   │
│  │ (Prompt  │  │(JSONL,   │  │ (Meta,   │  │ (Tracking,   │   │
│  │ Templates)│  │ Message) │  │ Loader)  │  │  Analysis)   │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘   │
│  ┌──────────┐                                                  │
│  │   MCP    │                                                  │
│  │(External │                                                  │
│  │ Servers) │                                                  │
│  └──────────┘                                                  │
└─────────────────────────────────────────────────────────────────┘
```

---

## 模块结构

Aura 使用 Go workspace (`go.work`) 管理多个模块：

```
aura/
├── go.work                    # Go workspace 定义
├── cmd/aura/                  # 应用入口
├── shared/                    # 共享工具（配置、日志、Hooks、Tasks、TypedManager）
├── storage/                   # 存储层（JSONL 消息存储、TaskStore）
├── tools/                     # 工具实现（含 tasktool、location）
├── knowledge/                 # RAG 知识库
├── personality/               # 用户档案与风格
├── habit/                     # 用户习惯学习（操作追踪、模式分析）
├── skill/                     # Skill 系统（Prompt 模板）
├── agent/                     # Agent 元数据和配置（含权限继承、Reviewer）
├── mcp/                       # MCP 服务器集成（stdio + HTTP/SSE）
├── session/                   # 会话管理
├── commands/                  # 内部命令系统（含 Agent 委托）
├── core/                      # Engine 和运行时（含 rollback、skilltool、prompt cache）
├── api/                       # HTTP API 服务器
├── adapters/                  # 外部平台适配器
├── cli/                       # CLI 和 TUI
└── docs/                      # 文档
```

### 模块详情

| 模块 | 导入路径 | 职责 |
|------|----------|------|
| `shared` | `github.com/oneliang/aura/shared` | 配置加载 (Viper)、日志 (Zerolog)、错误处理、国际化 (i18n)、常量定义、统一事件系统 (Event 接口、事件总线)、Hooks Framework (pkg/hooks)、Tasks 管理 (pkg/tasks)、TypedManager 泛型框架 (pkg/manager) |
| `storage` | `github.com/oneliang/aura/storage` | 存储层：JSONL 消息存储、统一消息数据结构、TaskStore 任务持久化 |
| `tools` | `github.com/oneliang/aura/tools` | 工具实现：文件、Shell、SSH、LSP、Web、计算器、文本处理等，新增 tasktool（任务追踪工具）、location（IP 地理位置） |
| `knowledge` | `github.com/oneliang/aura/knowledge` | RAG 系统：Chroma 向量库、embedding、检索、导入、DynamicRAG |
| `personality` | `github.com/oneliang/aura/personality` | 用户档案、响应风格、自适应学习、档案导入 |
| `habit` | `github.com/oneliang/aura/habit` | 用户习惯学习：操作追踪 (Action)、模式分析 (Analyzer)、习惯管理 (Manager)、偏好提取 (Preference)，完整模型 (Habit/Pattern/Frequency)，JSONL 存储（按用户隔离） |
| `skill` | `github.com/oneliang/aura/skill` | Skill 系统：Markdown Prompt 模板加载、构建、Skill Manager (CRUD)，权限等级 (read/write/execute/admin) |
| `agent` | `github.com/oneliang/aura/agent` | Agent 元数据 (AgentMeta)：权限继承 (PermissionMode)、Reviewer 集成 (UseReviewer)、配置加载器、Agent Manager (CRUD) |
| `mcp` | `github.com/oneliang/aura/mcp` | MCP 服务器集成：外部工具通过 Model Context Protocol 接入（stdio + HTTP/SSE 双传输）、并行启动、健康检查、工具发现 |
| `session` | `github.com/oneliang/aura/session` | 会话管理、多用户隔离、订阅路由、定时调度（使用 storage 模块） |
| `commands` | `github.com/oneliang/aura/commands` | 内部命令执行器（UI 无关，可复用），支持 skill-as-command、Agent 委托、意图识别、命令别名、依赖 agent 模块 |
| `core` | `github.com/oneliang/aura/core` | Engine、AgentRuntime、ReAct 引擎、LLM 客户端、SDK、Memory、Planner、Permissions、Orchestrator、Agent 委托、Intent 服务、rollback（Git 回滚）、skilltool（技能注入）、prompt cache（提示缓存），runtime 拆分为 boot/processing/eventing/confirmation/delegate/components |
| `api` | `github.com/oneliang/aura/api` | REST API 服务器、Webhooks、Web UI、会话 HTTP 端点、SSE (Server-Sent Events) 流式支持、多用户认证 |
| `adapters` | `github.com/oneliang/aura/adapters` | 飞书 (Feishu) 等外部平台集成适配器 |
| `cli` | `github.com/oneliang/aura/cli` | CLI (Cobra)、TUI (Bubbletea v2) 全屏界面，采用 MVU 模式，依赖 api 模块 |

---

## 依赖关系图

### 依赖层次（从底向上）

**分层原则：**

| 层次 | 职责 | 依赖规则 | 模块 |
|------|------|----------|------|
| **Layer 1 (基础层)** | 提供基础能力，可独立使用 | 无内部依赖 | `shared`, `storage`, `tools`, `knowledge`, `personality`, `habit`, `skill`, `agent`, `mcp` |
| **Layer 2 (服务层)** | 提供业务服务组合 | 仅依赖基础层 | `session`, `commands` |
| **Layer 3 (核心层)** | 整合下层模块，实现核心逻辑 | 依赖服务层和基础层 | `core` |
| **Layer 4 (应用层)** | 面向用户的入口 | 仅依赖核心层 | `cli`, `api`, `adapters` |

**分层约束：**
- **单向依赖**：上层可依赖下层，下层不可依赖上层
- **跨层限制**：Layer 4 不应直接依赖 Layer 1/2，应通过 Layer 3 访问
- **接口隔离**：层间通过接口通信，降低耦合度
- **agent 特例**：`agent` 位于 Layer 1（仅依赖 shared），负责 Agent 元数据和配置加载

```
┌─────────────────────────────────────────────────────────────┐
│ Layer 4: Application Layer (应用层)                          │
│ ┌─────────┐  ┌──────────┐  ┌──────────────┐                │
│ │   cli   │  │   api    │  │   adapters   │                │
│ │(全依赖) │  │(core,    │  │ (core,       │                │
│ │         │  │ session) │  │  session)    │                │
│ └─────────┘  └──────────┘  └──────────────┘                │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Core Layer (核心层)                                 │
│ ┌──────────────────────────────────────────────────────────┐│
│ │  core (agent, commands, session, shared, storage, tools,  ││
│ │        knowledge, personality, skill)                      ││
│ │  - Engine, AgentRuntime, ReAct, LLM, Memory, SDK         ││
│ │  - Planner, Permissions, Orchestrator                    ││
│ └──────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: Service Layer (服务层)                              │
│ ┌──────────────────┐  ┌──────────────────────────────────┐  │
│ │   session        │  │   commands                       │  │
│ │   (shared,       │  │   (knowledge, personality,       │  │
│ │    storage)      │  │    session, shared)              │  │
│ └──────────────────┘  └──────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            ▲
                            │
┌─────────────────────────────────────────────────────────────┐
│ Layer 1: Base Layer (基础层)                                 │
│ ┌──────────┐  ┌──────────┐  ┌────────────┐  ┌────────────┐ │
│ │  shared  │  │  tools   │  │ knowledge  │  │personality │ │
│ │ (无依赖) │  │(无依赖)  │  │ (无依赖)   │  │ (无依赖)   │ │
│ └──────────┘  └──────────┘  └────────────┘  └────────────┘ │
│ ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐    │
│ │  habit   │  │  skill   │  │ storage  │  │  agent   │    │
│ │(无依赖)  │  │(无依赖)  │  │(无依赖)  │  │(无依赖)  │    │
│ └──────────┘  └──────────┘  └──────────┘  └──────────┘    │
│ ┌──────────┐                                              │
│ │   mcp    │                                              │
│ │(shared,  │                                              │
│ │  tools)  │                                              │
│ └──────────┘                                              │
└─────────────────────────────────────────────────────────────┘
```

### 模块依赖矩阵

| 模块 | 依赖的内部模块 | 层次 |
|------|----------------|------|
| `shared` | - | Layer 1 |
| `storage` | `shared` | Layer 1 |
| `tools` | `shared` | Layer 1 |
| `knowledge` | `shared` | Layer 1 |
| `personality` | - | Layer 1 |
| `skill` | `shared` | Layer 1 |
| `agent` | `shared` | Layer 1 |
| `habit` | `shared`, `storage` | Layer 1 |
| `mcp` | `shared`, `tools` | Layer 1 |
| `session` | `shared`, `storage` | Layer 2 |
| `commands` | `agent`, `knowledge`, `personality`, `session`, `shared`, `skill`, `storage` | Layer 2 |
| `core` | `agent`, `commands`, `mcp`, `session`, `shared`, `skill`, `storage`, `tools`, `knowledge`, `personality` | Layer 3 |
| `api` | `commands`, `core`, `session`, `shared`, `skill` | Layer 4 |
| `adapters` | `commands`, `core`, `session`, `shared`, `skill` | Layer 4 |
| `cli` | `api`, `commands`, `core`, `knowledge`, `mcp`, `personality`, `session`, `shared`, `skill`, `tools` | Layer 4 |
| `cmd/aura` | `cli` | Entry |

**core 模块子包依赖：**

> **设计说明**：core 直接依赖 knowledge/personality/skill 以支持 RAG 检索、人格管理和技能系统集成。commands 是独立模块，不绑定 agent 功能，因此 api/adapters 可直接访问 commands 而无需通过 core 代理。

- `core/pkg/engine` - Engine 核心实现 (ReAct 循环、显式规划、工具执行、并行工具执行) → `core/pkg/llm`, `core/pkg/memory`, `core/pkg/planner`, `tools`, `commands`
- `core/pkg/runtime` - 统一运行时（拆分为 boot.go/processing.go/eventing.go/confirmation.go/delegate.go） → `core/pkg/engine`, `core/pkg/factory`, `core/pkg/llm`, `core/pkg/memory`, `core/pkg/permissions`, `core/pkg/prompt`, `skill`, `storage`, `tools`, `agent`, `habit`
- `core/pkg/sdk` - SDK 导出层 → `core/pkg/factory`, `core/pkg/llm`, `core/pkg/permissions`, `core/pkg/prompt`, `core/pkg/runtime`, `core/pkg/orchestrator`, `core/pkg/memory`, `core/pkg/intent`, `session`, `shared`, `commands`, `tools`
- `core/pkg/planner` - 任务规划器 → `core/pkg/llm`
- `core/pkg/memory` - 对话记忆系统 → `core/pkg/llm`
- `core/pkg/permissions` - 权限管理器 → 无内部依赖
- `core/pkg/orchestrator` - 多 Agent 编排器 → `core/pkg/workspace`, `shared`, `session`
- `core/pkg/workspace` - 工作空间隔离器 → 无内部依赖
- `core/pkg/factory` - 工厂函数 → `core/pkg/engine`, `core/pkg/llm`, `core/pkg/permissions`, `core/pkg/prompt`, `tools`, `commands`, `shared`, `session`
- `core/pkg/intent` - 意图识别服务 → `commands`, `shared`
- `core/pkg/llm` - LLM 客户端 (Ollama, OpenAI, Anthropic)
- `core/pkg/prompt` - Prompt 构建器 → `shared`
- `core/pkg/rollback` - Git 回滚管理器（Plan 模式快照与回滚） → 无内部依赖
- `core/pkg/skilltool` - 技能匹配和注入（渐进式披露，LLM 语义匹配 + 关键词回退） → `skill`

---

## 核心接口定义

### Engine 接口 (`core/pkg/engine/engine.go`)

```go
// Engine 是核心执行引擎，管理 ReAct 循环、规划、工具执行
type Engine struct {
    client     llm.Client
    memory     memory.Memory
    regTools   map[string]tools.Tool
    config     EngineConfig
    planner    *planner.Planner
    logger     *logger.Logger
    hookEngine *hooks.Engine

    // 规划模式状态
    currentPlan       *planner.Plan
    currentStepIndex  int
    currentRequestID  string
    planModeState     PlanModeState
    toolAllowlist     []string
    currentPhase      Phase

    // 输入队列管理
    inputMu      sync.Mutex
    inputQueue   chan InputRequest
    ctx          context.Context
    cancel       context.CancelFunc
    processingMu sync.Mutex

    // 任务管理
    taskList  *tasks.TaskList
    taskStore *taskstore.TaskStore

    // 回滚管理
    rollbackMgr       *rollback.Manager
    rollbackSnapshotID string

    // Agent 委托
    agentDelegateFn func(ctx context.Context, agentName string, task string) (string, error)

    // 规划模式回调
    planReviewFn      PlanReviewHandler
    rollbackConfirmFn RollbackConfirmHandler
    askUserQuestionFn AskUserQuestionHandler
}

// EngineConfig 配置
type EngineConfig struct {
    SystemPrompt        string
    Tools               []tools.Tool
    Commands            commands.Command
    ConfirmationHandler ToolConfirmationHandler
    PlanningMode        PlanningMode  // implicit/explicit/auto
    PlannerClient       llm.Client
    PlanConfig          PlanConfig    // 规划配置（verify commands、reviewer）

    // 上下文优化
    EnableSummarization bool
    Summarizer          *memory.Summarizer
    EnableDynamicRAG    bool
    DynamicRAG          *retrieval.DynamicRAG

    // 循环控制
    MaxSteps        int  // ReAct 循环最大迭代次数（0 = 无限制）
    MaxParallelTools int // 并行工具执行最大并发数（0 = 默认 5，1 = 串行）

    // Thinking 模式
    Thinking *llm.ThinkingConfig

    // 自验证
    EnableReflection bool
    ReflectionConfig ReflectionConfig

    // 提示缓存
    EnablePromptCache bool
    PromptCacheConfig *llm.PromptCacheConfig

    // 技能注入
    SkillInjector *skilltool.SkillInjector
}

// PlanConfig 规划模式配置
type PlanConfig struct {
    VerifyCommands     []string // 验证阶段执行的命令
    UseReviewerAgent   bool     // 使用 reviewer agent 验证
    ParallelExplore    bool     // 并行探索
    MaxParallelExplore int      // 最大并行探索数
}

// 规划模式
type PlanningMode string
const (
    ModeImplicit PlanningMode = "implicit"  // LLM 在 ReAct 中隐式规划
    ModeExplicit PlanningMode = "explicit"  // 先创建显式计划，再逐步执行
    ModeAuto     PlanningMode = "auto"      // 根据任务复杂度自动选择
)

// PlanModeState 规划模式状态（显式规划阶段）
type PlanModeState string
const (
    PlanModeStateNone    PlanModeState = "none"     // 未在规划模式
    PlanModeStateExplore PlanModeState = "explore"  // 探索阶段（只读）
    PlanModeStateDesign  PlanModeState = "design"   // 设计阶段（创建计划）
    PlanModeStateReview  PlanModeState = "review"   // 审查阶段（等待用户批准）
    PlanModeStateExecute PlanModeState = "execute"  // 执行阶段（执行步骤）
    PlanModeStateVerify  PlanModeState = "verify"   // 验证阶段（运行验证命令）
)

// Phase 执行阶段
type Phase int
const (
    PhaseNormal     Phase = 0  // 正常执行
    PhaseExploration Phase = 1 // 探索阶段（工具受限）
)

// 回调处理器类型
type ToolConfirmationHandler func(ctx context.Context, toolName string, params map[string]any) (bool, error)
type PlanReviewHandler func(ctx context.Context, goal string, steps []string) (bool, error)
type RollbackConfirmHandler func(ctx context.Context, snapshotID string, files []string, reason string) (bool, error)
type AskUserQuestionHandler func(ctx context.Context, question string, options []events.QuestionOption, questionType string) (*events.QuestionResponse, error)

// 核心方法
func (e *Engine) Run(ctx context.Context, input string) (<-chan events.Event, error)
func (e *Engine) Shutdown()
func (e *Engine) AddTool(tool tools.Tool)
func (e *Engine) GetTools() []tools.Tool
func (e *Engine) SetToolAllowlist(names []string)
func (e *Engine) GetPhase() Phase
func (e *Engine) GetPlanModeState() PlanModeState
func (e *Engine) LoadTasks() error
func (e *Engine) GetTaskList() *tasks.TaskList
```

### AgentMeta (`agent/pkg/agent/agent.go`)

```go
// AgentMeta 是 Agent 的 YAML frontmatter 元数据，嵌入 config.AgentConfig
type AgentMeta struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`  // LLM 触发匹配

    // Agent 特定字段
    LLMModel     string   `yaml:"llm_model,omitempty"`
    DisableTools []string `yaml:"disable_tools,omitempty"`

    // 权限继承（子 Agent 执行）
    PermissionMode  string `yaml:"permission_mode,omitempty"`  // inherit/inherit_downgrade/independent
    PermissionLevel string `yaml:"permission_level,omitempty"` // allow/ask/deny（用于 downgrade）

    // Reviewer 集成（执行后验证）
    UseReviewer   bool   `yaml:"use_reviewer,omitempty"`     // 启用后执行验证
    ReviewerAgent string `yaml:"reviewer_agent,omitempty"`   // Reviewer agent 名称（默认 "code-reviewer"）

    // 嵌入 AgentConfig 实现继承
    config.AgentConfig `yaml:",inline"`
}

// AgentConfig 配置继承字段
type AgentConfig struct {
    PlanningMode  string  // 规划模式：implicit/explicit/auto
    Temperature   float64 // LLM 温度
    SummaryTemp   float64 // 摘要温度
}

// Agent 从 AGENT.md 文件加载的完整定义
type Agent struct {
    Name        string
    Description string
    FilePath    string
    Content     string
    Body        string  // YAML frontmatter 之后的内容
    Meta        AgentMeta
}

// 核心方法
func (m *AgentMeta) Validate() error
func (m *AgentMeta) GetLLMOverride() *config.LLMConfig

// Item 接口方法（支持 TypedManager）
func (a *Agent) GetName() string
func (a *Agent) GetDescription() string
func (a *Agent) GetFilePath() string
func (a *Agent) GetContent() string
func (a *Agent) GetBody() string
```

### AgentLoader (`agent/pkg/loader/loader.go`)

```go
// Loader 从配置目录加载所有 Agent
type Loader struct {
    config      *config.AgentsConfig
    agents      map[string]*agentpkg.Agent
    agentByName map[string]*agentpkg.Agent
}

// 核心方法
func (l *Loader) Load() ([]*agentpkg.Agent, error)
func (l *Loader) GetAgent(name string) (*agentpkg.Agent, error)
func (l *Loader) GetAgents() []*agentpkg.Agent
func (l *Loader) GetItems() []sharedmanager.Item  // 支持共享管理器框架
```

### AgentRuntime (`core/pkg/runtime/runtime.go`)

> **文件拆分**：runtime 拆分为 boot.go（初始化）、processing.go（处理）、eventing.go（事件）、confirmation.go（确认）、delegate.go（委托）、components.go（组件）。

```go
// AgentRuntime 是统一运行时，管理 LLM、工具、记忆、会话、Agent 委托
type AgentRuntime struct {
    config        *RuntimeConfig
    llmClient     llm.Client
    httpClient    *http.Client
    webHttpClient *http.Client
    agent         *enginepkg.Engine
    permMgr       *permissions.Manager
    promptBuilder *prompt.PromptBuilder
    memory        *memory.SessionMemory
    commandProvider commands.Command
    intentService *intent.Service

    // 技能系统
    skillLoader  *loader.Loader
    skillInjector *skilltool.SkillInjector

    // Agent 系统
    agentLoader *agentloader.Loader
    agentDelegateFn func(ctx context.Context, agentName string, task string) (string, error)

    // MCP 系统
    mcpManager *mcpmanager.Manager

    // Hooks 系统
    hookEngine *hooks.Engine

    // 会话管理
    sessionID    string
    userID       string
    sessionStore *jsonl.MessageStore
    dataDir      string

    // Habit 追踪
    habitManager *manager.Manager

    // 组件架构（新增）
    shared   *SharedResources  // 共享资源（子 Agent 可继承）
    skills   *SkillSystem       // 技能系统（nil 当 disabled）
    agents   *AgentSystem       // Agent 系统（nil 当 disabled）
    mcp      *MCPSystem         // MCP 系统（nil 当未注入）
    hooks    *HookSystem        // Hooks 系统（nil 当无 hooks.yaml）
    session  *SessionContext    // 会话上下文（子 Agent 独立）

    // 提示缓存
    cacheManager *prompt.PromptCacheManager

    // 事件流架构
    eventOutCh         chan Event
    interactionPending map[string]chan events.InteractionResponse
    inputQueue         chan inputRequest

    // 子 Agent 快速路径
    skipInitialize      bool
    preBuiltTools       []tools.Tool
    parentToolAllowlist []string
}

// 核心方法
func NewRuntime(cfg *RuntimeConfig, opts ...RuntimeOption) (*AgentRuntime, error)
func NewSubAgentRuntime(parent *AgentRuntime, subCfg *RuntimeConfig, disabledTools []string, logger *logger.Logger) (*AgentRuntime, error)
func (r *AgentRuntime) Initialize(ctx context.Context) error
func (r *AgentRuntime) Start(ctx context.Context) error
func (r *AgentRuntime) Stop(ctx context.Context)
func (r *AgentRuntime) SendEvent(ctx context.Context, event Event) error
func (r *AgentRuntime) Events() <-chan Event
func (r *AgentRuntime) Process(ctx context.Context, input string) (<-chan Event, error)
func (r *AgentRuntime) Shutdown()

// 多用户隔离和技能查询
func (r *AgentRuntime) GetUserID() string
func (r *AgentRuntime) IsSkillsEnabled() bool
func (r *AgentRuntime) GetSkillDirectories() []string
```

### Runtime Components (`core/pkg/runtime/components.go`)

```go
// SharedResources - 与子 Agent 共享的只读资源（线程安全）
type SharedResources struct {
    llmClient       llm.Client
    httpClient      *http.Client
    webHttpClient   *http.Client
    permMgr         *permissions.Manager
    commandProvider commands.Command
}

func (s *SharedResources) GetPermMgr() *permissions.Manager
func (s *SharedResources) ClonePermMgrWithDowngrade(level permissions.PermissionControlLevel) *permissions.Manager

// SkillSystem - 技能加载和渐进式披露（nil 当 Skills.Enabled=false）
type SkillSystem struct {
    loader      *loader.Loader
    injector    *skilltool.SkillInjector
    intentSvc   *intent.Service
    directories []string
}

func (s *SkillSystem) IsEnabled() bool
func (s *SkillSystem) GetDirectories() []string
func (s *SkillSystem) GetLoader() *loader.Loader
func (s *SkillSystem) GetInjector() *skilltool.SkillInjector

// AgentSystem - Agent 加载和委托（nil 当 Agents.Enabled=false）
type AgentSystem struct {
    loader     *agentloader.Loader
    delegateFn func(ctx context.Context, agentName string, task string) (string, error)
    mu         sync.RWMutex
}

func (a *AgentSystem) FindAgent(agentName string) (*agentpkg.Agent, error)
func (a *AgentSystem) GetLoader() *agentloader.Loader
func (a *AgentSystem) SetDelegateFn(fn func(ctx context.Context, agentName string, task string) (string, error))
func (a *AgentSystem) GetDelegateFn() func(ctx context.Context, agentName string, task string) (string, error)

// MCPSystem - MCP 服务器生命周期（nil 当未注入）
type MCPSystem struct {
    manager *mcpmanager.Manager
    started bool  // 防止子 Agent 停止父级的 MCP 服务器
}

func (m *MCPSystem) StartAll(ctx context.Context) ([]tools.Tool, error)
func (m *MCPSystem) StopAll(ctx context.Context) error
func (m *MCPSystem) GetManager() *mcpmanager.Manager

// HookSystem - Hooks 引擎（nil 当 hooks.yaml 不存在）
type HookSystem struct {
    engine *hooks.Engine
}

func (h *HookSystem) Fire(ctx context.Context, event hooks.HookEventType, data map[string]any)
func (h *HookSystem) FireBlocking(ctx context.Context, event hooks.HookEventType, data map[string]any) (*hooks.HookResult, error)
func (h *HookSystem) GetEngine() *hooks.Engine

// SessionContext - 会话特定状态（子 Agent 独立）
type SessionContext struct {
    sessionID     string
    userID        string
    sessionStore  *jsonl.MessageStore
    dataDir       string
    initialized   bool
    logger        *logger.Logger
    promptBuilder *prompt.PromptBuilder
    habitManager  *manager.Manager
    taskList      *tasks.TaskList
}

func (s *SessionContext) GetUserID() string
func (s *SessionContext) GetSessionID() string
func (s *SessionContext) GetDataDir() string
func (s *SessionContext) GetLogger() *logger.Logger
func (s *SessionContext) GetPromptBuilder() *prompt.PromptBuilder
func (s *SessionContext) GetHabitManager() *manager.Manager
func (s *SessionContext) GetTaskList() *tasks.TaskList
```

### Agent 委托流程

```
1. 用户输入 → runtime.Process()
2. Engine.Run() 开始 ReAct 循环
3. LLM 决定委托给 Agent → 生成 command_delegate_to_agent
4. CommandProvider 执行委托命令
5. agentHandler.ExecuteCommand() 调用 delegateFn()
6. delegateFn() 创建 SubAgent:
   - 从 agentLoader 查找目标 Agent
   - 构建 SubAgent system prompt（agent body + task）
   - 创建轻量级 SubAgent Runtime（共享父级 LLM 客户端、工具、MCP 管理器等资源）
   - 应用 config 继承（planning_mode/temperature/summary_temp）
   - 注册工具（排除 disable_tools）
   - 执行任务并收集事件
7. 返回结果给主 Engine
```

**注意**：子 Agent 不再创建临时工作目录，而是直接共享父运行时的昂贵资源（LLM 客户端、HTTP 客户端、工具实例、MCP 管理器、技能/Agent 加载器）。子 Agent 默认继承父级的 LLM 配置，`llm_model` 字段仅用于独立执行场景。

### Orchestrator 接口 (`core/pkg/orchestrator/orchestrator.go`)

```go
// Orchestrator 是多 Agent 编排器，管理子 Agent、协作文档和监督
type Orchestrator struct {
    config      *config.OrchestratorConfig
    workspace   *workspace.Isolator    // 工作空间隔离器
    docStore    *DocStore              // 文档存储
    registry    *TaskRegistry          // 任务注册表
    coordinator *DocCoordinator        // 文档协调器
    supervisor  *Supervisor            // 健康监督器
    subAgents   map[string]*SubAgent   // 子 Agent 映射
}

// 核心方法
func (o *Orchestrator) Start(ctx context.Context) error
func (o *Orchestrator) Stop()
func (o *Orchestrator) SpawnAgent(ctx context.Context, agentID string, llmOverride *config.LLMConfig) (*SubAgent, error)
func (o *Orchestrator) StopAgent(agentID string) error
func (o *Orchestrator) CreateDoc(doc *CollaboDoc) (string, error)
func (o *Orchestrator) GetPendingDocs(agentID string) ([]*CollaboDoc, error)
func (o *Orchestrator) UpdateDocStatus(id, agentID string, status DocStatus, note string) error
```

### Planner 接口 (`core/pkg/planner/planner.go`)

```go
// Plan 表示任务计划
type Plan struct {
    Goal     string `json:"goal"`
    Steps    []Step `json:"steps"`
    Current  int    `json:"current"`
    Complete bool   `json:"complete"`
}

// Step 表示计划步骤
type Step struct {
    ID          string `json:"id"`
    Description string `json:"description"`
    Status      string `json:"status"` // pending, running, completed, failed
    Result      string `json:"result,omitempty"`
}

// Planner 创建和管理计划
type Planner struct {
    client llm.Client
}

func (p *Planner) CreatePlan(ctx context.Context, goal string) (*Plan, error)
func (p *Plan) GetCurrentStep() *Step
func (p *Plan) Advance()
func (p *Plan) GetProgress() (int, int)
```

### Skill 接口 (`skill/pkg/skill/skill.go`)

```go
type Skill struct {
    Name        string  // 技能标识符
    Description string  // 触发描述，LLM 根据此判断是否需要参考
    FilePath    string  // SKILL.md 文件路径
    Content     string  // 完整 SKILL.md 内容（元数据 + 正文）
    Body        string  // 技能正文（排除 YAML frontmatter）
}
```

### 渐进式披露 — SkillMatcher & SkillInjector (`core/pkg/skilltool/`)

Aura 不再将所有技能内容包含在每次系统提示中，而是使用渐进式披露机制：

```go
// Option configures a SkillMatcher
type Option func(*SkillMatcher)

// WithLLMClient sets the LLM client for NLP-based semantic skill matching.
func WithLLMClient(client llm.Client, model string) Option

// SkillMatcher 分析用户输入并匹配相关技能
type SkillMatcher struct {
    loader    *skillloader.Loader
    llmClient llm.Client
    model     string
}

func NewSkillMatcher(loader *skillloader.Loader, opts ...Option) *SkillMatcher
func (m *SkillMatcher) MatchSkills(userInput string, intentResult *intent.IntentResult) []skill.Skill
// 匹配策略（配置 LLM 客户端时）：
// 1. NLP 语义匹配（优先）：调用 LLM 判断哪些技能与用户输入相关，支持传入意图识别结果
// 2. 关键词回退：LLM 调用失败或未配置时，回退到名称/描述关键词匹配
// 无 LLM 客户端时直接使用关键词匹配。

// SkillInjector 将匹配的技能内容动态注入系统提示
type SkillInjector struct{}

func NewSkillInjector() *SkillInjector
func (i *SkillInjector) InjectSkills(systemPrompt string, matchedSkills []skill.Skill) string
```

### Memory 接口 (`core/pkg/memory/memory.go`)

```go
// ConversationMemory 是对话记忆存储
type ConversationMemory struct {
    messages    []llm.Message
    totalTokens int            // 缓存的令牌计数
    maxLen      int            // 最大消息数（后备）
    maxTokens   int            // 最大令牌数
    tokenizer   TokenEstimator // 令牌估算器

    // 摘要支持
    summarizer      *Summarizer
    summaryText     string  // 当前对话摘要
    lastSummaryAt   int     // 上次摘要时的消息数
    archiveOriginal bool    // 是否归档原始消息
}

// TokenEstimator 令牌估算接口
type TokenEstimator interface {
    EstimateMessages(messages []llm.Message) int
}
```

### Message 结构 (`shared/pkg/memory/memory.go`)

Message 采用单一 ContentBlocks 字段设计，支持结构化内容：

```go
// Message represents a chat message in the conversation.
type Message struct {
    Role          string         `json:"role"`            // system, user, assistant
    ContentBlocks []ContentBlock `json:"content_blocks"`  // 结构化内容块
    Type          MessageType    `json:"type,omitempty"`  // 消息类型（可选）
    Parts         []MessagePart  `json:"parts,omitempty"` // 多模态消息（可选）
}
```

**ContentBlock 多态接口**（需要自定义 JSON 序列化）：
- `TextBlock` — 文本内容（`{type: "text", text: "..."}`)
- `ThinkingBlock` — 思考内容（`{type: "thinking", thinking: "..."}`)
- `ToolUseBlock` — 工具调用（`{type: "tool_use", id: "...", name: "...", input: {...}}`)
- `ToolResultBlock` — 工具结果（`{type: "tool_result", tool_use_id: "...", content: "..."}`)

**文本提取模式**：
```go
// 从 ContentBlocks 提取文本内容
func extractText(blocks []ContentBlock) string {
    for _, block := range blocks {
        if tb, ok := block.(TextBlock); ok {
            return tb.Text
        }
    }
    return ""
}
```

**消息构造模式**：
```go
msg := llm.Message{
    Role: "user",
    ContentBlocks: []sharedmemory.ContentBlock{
        sharedmemory.TextBlock{Type: sharedmemory.BlockTypeText, Text: content},
    },
}
```

### SessionMemory (`core/pkg/memory/session_memory.go`)

```go
// SessionMemory 管理会话级别的对话记忆，与持久化存储同步
type SessionMemory struct {
    *BaseMemory  // 嵌入基础记忆
    sessionID    string
    userID       string
    source       MessageSource
    store        *jsonl.MessageStore  // JSONL 持久化存储

    // 新增字段
    hookEngine          *hooks.Engine          // Hooks 引擎
    selectiveCompressor *SelectiveCompressor   // 选择性压缩器
    persistWg           sync.WaitGroup         // 异步持久化追踪
}

// 核心方法
func (m *SessionMemory) Add(role, content string)
func (m *SessionMemory) AddWithType(role, content, msgType string)
func (m *SessionMemory) GetMessages() []llm.Message
func (m *SessionMemory) Clear()
func (m *SessionMemory) GetStats() MemoryStats
func (m *SessionMemory) MaybeSummarize(ctx context.Context) error  // 摘要
func (m *SessionMemory) MaybeCompact(ctx context.Context) (*CompactMetadata, error)  // 选择性压缩
func (m *SessionMemory) Shutdown()  // 等待持久化完成
```

### Event 接口 (`shared/pkg/events/events.go`)

```go
// EventType 表示事件类型
type EventType string

// Engine 执行事件
const (
    EventTypeThinkingStart       EventType = "thinking_start"
    EventTypeThinkingEnd         EventType = "thinking_end"
    EventTypeAction              EventType = "action"
    EventTypeResult              EventType = "result"
    EventTypeResponse            EventType = "response"
    EventTypeResponseChunk       EventType = "response_chunk"
    EventTypeError               EventType = "error"
    EventTypeStep                EventType = "step"
    EventTypeToolStart           EventType = "tool_start"
    EventTypeToolEnd             EventType = "tool_end"
)

// 规划事件
const (
    EventTypePlanCreated         EventType = "plan_created"
    EventTypePlanStep            EventType = "plan_step"
    EventTypePlanComplete        EventType = "plan_complete"
)

// 运行时生命周期事件
const (
    EventTypeDone                EventType = "done"
    EventTypeConfirmationRequest EventType = "confirmation_request"
)

// 任务事件
const (
    EventTypeTaskCreate            EventType = "task_create"
    EventTypeTaskUpdate            EventType = "task_update"
    EventTypeTaskList              EventType = "task_list"
)

// 内存事件
const (
    EventTypeMemoryClearRequest  EventType = "memory_clear_request"
    EventTypeMemoryCleared       EventType = "memory_cleared"
    EventTypeMemoryStatsRequest  EventType = "memory_stats_request"
    EventTypeMemoryStats         EventType = "memory_stats"
    EventTypeMemoryCompacted     EventType = "memory_compacted"
)

// 会话事件
const (
    EventTypeSessionCreated      EventType = "session_created"
    EventTypeSessionSwitched     EventType = "session_switched"
    EventTypeSessionDeleted      EventType = "session_deleted"
)

// 命令事件（事件驱动通信）
const (
    EventTypeCommandRequest      EventType = "command_request"
    EventTypeCommandResponse     EventType = "command_response"
    EventTypeCommandMatched      EventType = "command_matched"
    EventTypeCommandResult       EventType = "command_result"
)

// Event 是所有事件的接口
type Event interface {
    Type() EventType
    Content() string
    Extra() map[string]any
    Timestamp() time.Time
    RequestID() string  // 用于事件分组和全链路追踪
}
```

### Permissions Manager (`core/pkg/permissions/manager.go`)

```go
// Manager 管理工具权限检查
type Manager struct {
    config       *PermissionConfig
    commandCheck *CommandChecker  // Shell 命令检查器
    sshCheck     *SSHChecker      // SSH 检查器
    sessions     map[string]*SessionPermissions
    trustedDirs  []string  // 可信目录列表
}

// 核心方法
func (m *Manager) CheckPermission(ctx context.Context, toolName string, params map[string]any) (bool, bool, string)
func (m *Manager) CheckCommand(command string) (bool, string)
func (m *Manager) CheckSSHHost(host string) (bool, string)
func (m *Manager) IsTrustedPath(path string) bool
func (m *Manager) GrantSessionPermission(sessionID, toolName string)
func (m *Manager) CloneWithDowngrade(level PermissionControlLevel) (*Manager, error)  // 子 Agent 权限降级
```

### Hooks Framework (`shared/pkg/hooks/`)

Hooks Framework 提供服务器级别的子进程钩子，在关键执行点触发：

```go
// HookEventType - 执行生命周期事件类型
type HookEventType string

const (
    EventSessionStart     HookEventType = "SessionStart"      // 会话开始
    EventUserPromptSubmit HookEventType = "UserPromptSubmit"  // 用户提交
    EventPreToolUse       HookEventType = "PreToolUse"        // 工具执行前（可阻塞）
    EventPostToolUse      HookEventType = "PostToolUse"       // 工具执行后
    EventStop             HookEventType = "Stop"              // 请求结束
    EventPreCompact       HookEventType = "PreCompact"        // 压缩前
    EventPostCompact      HookEventType = "PostCompact"       // 压缩后
    EventEnterPlanMode    HookEventType = "EnterPlanMode"     // 进入规划模式
    EventExitPlanMode     HookEventType = "ExitPlanMode"      // 退出规划模式
)

// HookResult - 钩子执行结果
type HookResult struct {
    ExitCode int           // 退出码（2 = 阻塞主流程）
    Output   HookOutput    // 结构化输出
    Error    string        // 错误信息
}

// HookOutput - 可影响主流程的输出
type HookOutput struct {
    SystemMessage      string  // 注入到系统提示
    Continue           *bool   // false = 阻止执行
    PermissionDecision string  // 覆盖权限决策（allow/deny）
    ShouldRetry        bool    // 用于 PreResponse 钩子
    ReflectionFeedback string  // 质量反馈
}

// Engine - Hooks 引擎
type Engine struct {
    config     *config.HooksConfig
    execPool   *pool.ExecPool  // 并发执行池
    logger     *logger.Logger
}

func NewEngine(cfg *config.HooksConfig, logger *logger.Logger) *Engine
func (e *Engine) Fire(ctx context.Context, event HookEventType, data map[string]any)  // 非阻塞
func (e *Engine) FireBlocking(ctx context.Context, event HookEventType, data map[string]any) (*HookResult, error)  // 阻塞等待
func (e *Engine) FireWithToolName(ctx context.Context, event HookEventType, toolName string, data map[string]any)
func (e *Engine) Shutdown()

// 配置文件位置：~/.aura/hooks.yaml
```

**阻塞行为**：退出码 `2` 表示阻塞，阻止主流程继续执行。用于 `PreToolUse` 钩子拒绝工具执行。

### AgentManager 接口 (`agent/pkg/manager/manager.go`)

Agent 实现 `sharedmanager.Item` 接口方法，委托给通用 TypedManager：

```go
// AgentManager 委托给 TypedManager[Agent]
type AgentManager struct {
    mgr *sharedmanager.TypedManager[*agent.Agent]
}

// CRUD 操作（全部委托）
func (m *AgentManager) Create(ctx context.Context, req *CreateAgentRequest) (*agent.Agent, error)
func (m *AgentManager) Update(ctx context.Context, name string, req *UpdateAgentRequest) (*agent.Agent, error)
func (m *AgentManager) Delete(ctx context.Context, name string) error
func (m *AgentManager) Get(name string) *agent.Agent
func (m *AgentManager) List() []*agent.Agent
func (m *AgentManager) Reload(ctx context.Context) error
```

### TypedManager 泛型框架 (`shared/pkg/manager/typed_manager.go`)

TypedManager 提供类型安全的泛型 CRUD 管理，AgentManager 和 SkillManager 共享此框架：

```go
// Item 接口 - 管理项必须实现
type Item interface {
    GetName() string
    GetDescription() string
    GetFilePath() string
    GetContent() string
    GetBody() string
}

// TypedManager[T] - 类型安全的泛型管理器
type TypedManager[T Item] struct {
    mgr    *Manager
    config TypedConfig[T]
}

// TypedConfig[T] - 类型特定配置回调
type TypedConfig[T Item] struct {
    Validate       func(req CreateRequest) error
    BuildContent   func(item T, body string) string
    ConstructItem  func(name, desc, path, content, body string) T
    MergeUpdate    func(existing T, req UpdateRequest) T
    FindByName     func(items []T, name string) T
}

// 核心方法
func (m *TypedManager[T]) Create(ctx context.Context, req CreateRequest) (T, error)
func (m *TypedManager[T]) Update(ctx context.Context, name string, req UpdateRequest) (T, error)
func (m *TypedManager[T]) Delete(ctx context.Context, name string) error
func (m *TypedManager[T]) Get(name string) T
func (m *TypedManager[T]) List() []T
func (m *TypedManager[T]) Reload(ctx context.Context) error
func (m *TypedManager[T]) FindByName(name string) T
```

**设计优势**：消除类型断言，简化 SkillManager/AgentManager 代码，约 200 行重复 CRUD 代码合并为单一泛型框架。

**Agent Item 接口方法** (`agent/pkg/agent/agent.go`):
```go
func (a *Agent) GetName() string
func (a *Agent) GetDescription() string
func (a *Agent) GetFilePath() string
func (a *Agent) GetContent() string
func (a *Agent) GetBody() string
```

### SkillManager 接口 (`skill/pkg/manager/manager.go`)

Skill 同样实现 `sharedmanager.Item` 接口，委托给 TypedManager：

```go
// SkillManager 委托给 TypedManager[Skill]
type SkillManager struct {
    mgr *sharedmanager.TypedManager[*skill.Skill]
}

// CRUD 操作（全部委托）
func (m *SkillManager) Create(ctx context.Context, req *CreateSkillRequest) (*skill.Skill, error)
func (m *SkillManager) Update(ctx context.Context, name string, req *UpdateSkillRequest) (*skill.Skill, error)
func (m *SkillManager) Delete(ctx context.Context, name string) error
func (m *SkillManager) Get(name string) *skill.Skill
func (m *SkillManager) List() []*skill.Skill
func (m *SkillManager) Reload(ctx context.Context) error
```

**Skill Item 接口方法** (`skill/pkg/skill/skill.go`):
```go
func (s *Skill) GetName() string
func (s *Skill) GetDescription() string
func (s *Skill) GetFilePath() string
func (s *Skill) GetContent() string
func (s *Skill) GetBody() string
```

### MCP Manager (`mcp/pkg/manager/manager.go`)

```go
// Manager 管理 MCP 服务器生命周期（启动、停止、工具发现）
type Manager struct {
    loader  *loader.Loader
    clients map[string]*client.Client      // server name → client
    tools   map[string]*tool.MCPTool       // fullName → MCPTool
}

// 核心方法
func (m *Manager) StartAll(ctx context.Context) ([]tools.Tool, error)
func (m *Manager) StartServer(ctx context.Context, name string, cfg config.ServerConfig) ([]tools.Tool, error)
func (m *Manager) StopServer(ctx context.Context, name string) error
func (m *Manager) StopAll(ctx context.Context) error
func (m *Manager) GetTools() []tools.Tool
func (m *Manager) AddServer(ctx context.Context, name string, cfg config.ServerConfig) ([]tools.Tool, error)
func (m *Manager) RemoveServer(ctx context.Context, name string) error
func (m *Manager) ListServers() []config.ServerInfo
func (m *Manager) ServerInfoForName(name string) config.ServerInfo
```

### MCPTool (`mcp/pkg/tool/tool.go`)

```go
// MCPTool 将外部 MCP 工具包装为 Aura tools.Tool 接口
type MCPTool struct {
    serverName  string    // MCP 服务器名称
    toolName    string    // 原始工具名称
    fullName    string    // mcp__{server}__{tool}
    description string
    inputSchema map[string]any
    callFn      func(ctx context.Context, args map[string]any) (string, error)
}

func NewMCPTool(serverName, toolName, description string, inputSchema map[string]any,
    callFn func(ctx context.Context, args map[string]any) (string, error)) *MCPTool

func (t *MCPTool) Name() string         // 返回 mcp__{server}__{tool}
func (t *MCPTool) Description() string
func (t *MCPTool) Execute(ctx context.Context, params map[string]any) (string, error)
func (t *MCPTool) ServerName() string   // 返回服务器名称
func (t *MCPTool) ToolName() string     // 返回原始工具名称
```

### MCP Client (`mcp/pkg/client/client.go`)

```go
// Client 封装 MCP 客户端连接（支持 stdio 和 HTTP/SSE 两种传输）
type Client struct {
    // stdio 字段
    command string
    args    []string
    env     []string
    // HTTP 字段
    url     string
    headers map[string]string
    // 共享
    mcp     *client.Client
}

func NewStdioClient(command string, args []string, env map[string]string) *Client
func NewHTTPClient(url string, headers map[string]string) *Client
func (c *Client) IsHTTP() bool
func (c *Client) Initialize(ctx context.Context) error
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error)
func (c *Client) CallTool(ctx context.Context, name string, args map[string]any) (string, error)
func (c *Client) CallToolWithTimeout(ctx context.Context, name string, args map[string]any, timeout time.Duration) (string, error)
func (c *Client) Close() error
func (c *Client) HealthCheck(ctx context.Context) bool
```

### MCP 配置 (`mcp/pkg/config/config.go`)

```go
// Config 表示 MCP 配置（兼容 Claude Code mcp.json 格式）
type Config struct {
    MCPServers map[string]ServerConfig `json:"mcpServers"`
}

// ServerConfig 表示单个 MCP 服务器配置（支持 stdio 和 HTTP/SSE 两种传输）
type ServerConfig struct {
    // stdio 字段
    Command  string            `json:"command"`              // 可执行文件（stdio 模式）
    Args     []string          `json:"args"`                 // 命令行参数
    Env      map[string]string `json:"env,omitempty"`        // 环境变量
    // HTTP 字段
    Type     string            `json:"type,omitempty"`       // "stdio"（默认）或 "http"
    URL      string            `json:"url,omitempty"`        // HTTP/SSE 端点 URL
    Headers  map[string]string `json:"headers,omitempty"`    // HTTP 请求头
    // 通用字段
    Disabled bool              `json:"disabled,omitempty"`   // 跳过此服务器
    Timeout  time.Duration     `json:"timeout,omitempty"`    // 工具调用超时
}

// ServerInfo 表示 MCP 服务器运行时状态
type ServerInfo struct {
    Name      string    `json:"name"`
    Command   string    `json:"command"`
    Args      []string  `json:"args"`
    Status    string    `json:"status"`  // running/stopped/error/crashed
    ToolCount int       `json:"tool_count"`
    Error     string    `json:"error,omitempty"`
    LastSeen  time.Time `json:"last_seen"`
}
```

**MCP 配置示例**（`~/.aura/mcp.json`，兼容 Claude Code `mcp.json` 格式）：

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "~/.aura/workspace"]
    },
    "remote-server": {
      "type": "http",
      "url": "https://example.com/sse",
      "headers": {
        "Authorization": "Bearer xxx"
      }
    }
  }
}
```

### MCP 内部命令处理器（`commands/pkg/mcp_handler.go`）

```go
// MCPInfo 表示 MCP 服务器运行时状态（用于展示）
type MCPInfo struct {
    Name      string
    Command   string
    Args      string
    Status    string  // running/stopped
    ToolCount int
    Error     string
    LastSeen  time.Time
}

// ListServersFunc 回调类型：获取服务器列表
type ListServersFunc func() []MCPInfo

// MCPHandler 处理 MCP 相关命令（/mcp list）
type MCPHandler struct {
    listServersFn ListServersFunc
}

func (h *MCPHandler) ExecuteCommand(ctx context.Context, cmd string, params map[string]any) (string, error)
```

### IntentRecognizer 接口 (`commands/pkg/intent/recognizer.go`)

```go
// Recognizer 从自然语言输入识别命令意图
type Recognizer struct {
    aliasManager   *alias.Manager
    commands       []commands.CommandInfo
    keywordLoader  *KeywordLoader
}

// Result 包含识别结果
type Result struct {
    Matched    bool           // 是否匹配到命令
    Command    string         // 命令名称
    Params     map[string]any // 提取的参数
    Confidence float64        // 置信度：high/medium/low
    Source     string         // 匹配来源：alias/command/keyword
}

func (r *Recognizer) Recognize(ctx context.Context, input string) (*Result, error)
```

### AliasManager 接口 (`commands/pkg/alias/alias.go`)

```go
// Manager 管理命令别名和同义词
type Manager struct {
    aliases map[string]string // alias -> command
}

// 内置别名示例：
// "创建会话" -> session_create
// "列出技能" -> skill_list
// "显示配置" -> config_show
// "清空记忆" -> clear

func (m *Manager) Resolve(aliasStr string) (string, bool)
func (m *Manager) ResolveWithPrefix(input string) (string, bool)
func (m *Manager) Register(aliasStr, command string) error
func (m *Manager) Unregister(aliasStr string) error
func (m *Manager) List() map[string]string
```

### Event Bus 接口 (`shared/pkg/events/bus.go`)

```go
// Bus 提供事件发布/订阅功能，支持异步事件和同步请求/响应模式
type Bus struct {
    subscribers map[EventType][]chan Event
    cmdHandlers map[string]CommandHandler
    droppedCount map[EventType]int  // 各类型丢弃事件计数
}

// CommandHandler 处理命令请求并返回响应
type CommandHandler func(ctx context.Context, req *CommandRequest) CommandResponse

func (b *Bus) Subscribe(typ EventType) <-chan Event
func (b *Bus) Unsubscribe(typ EventType, ch <-chan Event)
func (b *Bus) Publish(event Event)
func (b *Bus) RegisterCommandHandler(commandType string, handler CommandHandler)
func (b *Bus) ExecuteCommand(ctx context.Context, req *CommandRequest) CommandResponse
func (b *Bus) Stats() map[EventType]int  // 返回各类型丢弃事件统计
```

### Logger Registry & 审计日志 (`shared/pkg/logger/`)

集中式日志注册表和 JSONL 审计日志系统：

```go
// Registry 线程安全的日志注册表（单例模式）
type Registry struct {
    loggers map[string]*Logger
    mu      sync.RWMutex
}

func (r *Registry) Register(name string, log *Logger) error
func (r *Registry) Get(name string) (*Logger, error)
func (r *Registry) MustGet(name string) *Logger
func (r *Registry) CloseAll()

// LLMAuditLogger 记录 LLM 请求/响应到 JSONL 文件
type LLMAuditLogger struct {
    filePath string
    mu       sync.Mutex
}

func (l *LLMAuditLogger) LogRequest(requestID, sessionID, provider, model string,
    messages []llm.Message, response *Response, duration time.Duration) error
// 每条记录包含: request_id, session_id, provider, model, messages, response, duration_ms

// DelegationAuditLogger 记录 Agent 委托事件
type DelegationAuditLogger struct {
    filePath string
    mu       sync.Mutex
}

// DelegationFileLogger 记录委托子 Agent 的内部日志
type DelegationFileLogger struct {
    filePath string
    mu       sync.Mutex
}
```

### IntentService 接口 (`core/pkg/intent/service.go`)

```go
// Service 提供 Core 层意图识别服务，包装 Layer 2 的 Recognizer
type Service struct {
    recognizer     *intentpkg.Recognizer
    commandProvider commands.Command
    enabled        bool
}

func NewService(cmdProvider commands.Command, threshold float64) *Service
func (s *Service) Recognize(ctx context.Context, input string) (*IntentResult, error)
func (s *Service) ExecuteCommand(ctx context.Context, result *IntentResult) (string, error)
func (s *Service) SetEnabled(enabled bool)
func (s *Service) IsEnabled() bool
```

### Agent Delegation (`core/pkg/runtime/delegate.go`)

```go
// Agent 委托功能（单一轻量级路径）
func (r *AgentRuntime) findAgent(agentName string) (*agentpkg.Agent, error)
func buildSubAgentConfig(parent *AgentRuntime, foundAgent *agentpkg.Agent, task string) *RuntimeConfig
func executeSubAgent(ctx context.Context, subAgentRuntime *AgentRuntime, task string, subAgentID string) (string, error)
func (r *AgentRuntime) createAgentDelegateFn(ctx context.Context) func(ctx context.Context, agentName string, task string) (string, error)

// 子 Agent 运行时（共享父级资源）
func NewSubAgentRuntime(parent *AgentRuntime, subCfg *RuntimeConfig, disabledTools []string, delegationLogger *logger.DelegationFileLogger) (*AgentRuntime, error)
```

### Rollback Manager (`core/pkg/rollback/manager.go`)

Git 基础的回滚管理器，用于 Plan 模式执行失败时恢复：

```go
// Manager 管理 Git 快照和回滚
type Manager struct {
    workDir   string
    logger    *logger.Logger
    snapshots map[string]*Snapshot
}

// Snapshot - Git stash 快照
type Snapshot struct {
    ID        string      // 快照 ID
    StashRef  string      // Git stash 引用
    Message   string      // 快照消息
    CreatedAt time.Time   // 创建时间
    Files     []string    // 快照文件列表
}

// RollbackResult - 回滚结果
type RollbackResult struct {
    Success bool     // 是否成功
    Message string   // 结果消息
    Files   []string // 恢复的文件列表
}

// 核心方法
func NewManager(workDir string, logger *logger.Logger) *Manager
func (m *Manager) CreateSnapshot(ctx context.Context, id string, files []string) (*Snapshot, error)
func (m *Manager) Rollback(ctx context.Context, snapshotID string) (*RollbackResult, error)
func (m *Manager) Cleanup(ctx context.Context, snapshotID string) error  // 清理而不恢复
```

**使用场景**：Plan 模式在执行前创建快照，失败时调用 `Rollback`，成功时调用 `Cleanup`。

### Prompt Cache Manager (`core/pkg/prompt/cache_manager.go`)

5 层缓存架构管理器，支持 Anthropic 和 OpenAI 提示缓存：

```go
// PromptCacheManager 管理缓存层
type PromptCacheManager struct {
    staticSystem     string  // Layer 0: 静态系统提示
    toolsBlock       string  // Layer 1: 工具定义
    skillsBlock      string  // Layer 2: 技能元数据
    agentsBlock      string  // Layer 3: Agent 元数据
    projectAuraBlock string  // Layer 4: 项目 AURA.md
}

// 缓存层常量
const (
    CacheLayerStaticSystem = 0
    CacheLayerTools        = 1
    CacheLayerSkills       = 2
    CacheLayerAgents       = 3
    CacheLayerProjectAura  = 4
)

// 核心方法
func NewPromptCacheManager() *PromptCacheManager
func (m *PromptCacheManager) SetStaticSystem(prompt string)
func (m *PromptCacheManager) SetToolsBlock(block string)
func (m *PromptCacheManager) SetSkillsBlock(block string)
func (m *PromptCacheManager) SetAgentsBlock(block string)
func (m *PromptCacheManager) SetProjectAuraBlock(block string)
func (m *PromptCacheManager) BuildSystemBlocks() []llm.SystemBlock  // Anthropic 风格
func (m *PromptCacheManager) InvalidateTools()
func (m *PromptCacheManager) InvalidateSkills()
func (m *PromptCacheManager) InvalidateAgents()
```

**缓存优势**：减少 LLM API 成本，静态层缓存命中时仅支付增量令牌费用。

### Tasks Package (`shared/pkg/tasks/task.go`)

会话级任务追踪，支持任务创建、更新和持久化：

```go
// TaskStatus - 任务状态
type TaskStatus string
const (
    TaskStatusPending    TaskStatus = "pending"
    TaskStatusInProgress TaskStatus = "in_progress"
    TaskStatusCompleted  TaskStatus = "completed"
)

// Task - 任务项
type Task struct {
    ID         int         // 任务 ID
    Content     string      // 任务内容（命令式）
    ActiveForm  string      // 进行形式（用于 UI）
    Status      TaskStatus  // 状态
    Notes       string      // 备注
    PlanStepID  string      // 关联 Plan 步骤 ID
    PlanGoal    string      // 关联 Plan 目标
}

// TaskList - 会话级任务列表
type TaskList struct {
    tasks  []Task
    nextID int
}

// 核心方法
func NewTaskList() *TaskList
func (tl *TaskList) Create(content string) Task
func (tl *TaskList) Update(id int, status TaskStatus, notes string) (Task, error)
func (tl *TaskList) CreateFromPlanStep(stepID, goal, content string) Task
func (tl *TaskList) List() []Task
func (tl *TaskList) Reset()
func (tl *TaskList) Restore(saved []Task)
```

**UI 事件**：任务变化时发送 `EventTypeTaskCreate`/`EventTypeTaskUpdate`/`EventTypeTaskList` 事件。

### LLM Client 接口 (`core/pkg/llm/client.go`)

```go
// Client LLM 客户端接口
type Client interface {
    Complete(ctx context.Context, req *Request) (*Response, error)
    Stream(ctx context.Context, req *Request) (<-chan Chunk, error)
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Request LLM 请求
type Request struct {
    Messages       []Message
    Model          string
    MaxTokens      int
    Temperature    float64
    Stream         bool
    Tools          []ToolSchema
    ToolChoice     string
    ResponseFormat *string

    // 新增：Thinking 模式
    Thinking       *ThinkingConfig

    // 新增：提示缓存
    PromptCache    *PromptCacheConfig
}

// ThinkingConfig - 原生 LLM 思考/推理模式
type ThinkingConfig struct {
    Enabled         bool
    ReasoningEffort string  // low/medium/high (OpenAI)
    BudgetTokens    int     // 最大思考令牌数 (Anthropic)
}

// PromptCacheConfig - 提示缓存配置
type PromptCacheConfig struct {
    Enabled      bool
    SystemBlocks []SystemBlock  // Anthropic 风格缓存块
    CacheType    string         // OpenAI 请求级缓存
}

// SystemBlock - Anthropic 风格系统块
type SystemBlock struct {
    Type         string
    Text         string
    CacheControl *CacheControl  // {type: "ephemeral"}
}

// Response LLM 响应
type Response struct {
    Message      Message
    Model        string
    Usage        Usage
    FinishReason string
    ToolCalls    []ToolCall
}

// Usage 令牌使用统计
type Usage struct {
    PromptTokens            int
    CompletionTokens        int
    TotalTokens             int
    // 新增：缓存令牌统计
    CacheCreationInputTokens int  // Anthropic 缓存创建
    CacheReadInputTokens     int  // Anthropic 缓存读取
}

// Chunk 流式响应块
type Chunk struct {
    Content          string
    ReasoningContent string  // 思考内容（流式）
    FinishReason     string
    Done             bool
    ToolCallDelta    *ToolCall
}

// 提供者实现：
// - Ollama Client (`core/pkg/llm/ollama/client.go`) - 本地推理，支持 tools
// - OpenAI Client (`core/pkg/llm/openai/client.go`) - GPT 系列，支持 reasoning_effort
// - Anthropic Client (`core/pkg/llm/anthropic/client.go`) - Claude 系列，支持 budget_tokens
```

### LLM RetryClient (`core/pkg/llm/retry_client.go`)

RetryClient 是 LLM Client 的包装器，提供指数退避 + jitter 重试机制：

```go
// RetryConfig 定义重试配置
type RetryConfig struct {
    MaxRetries  int           // 最大重试次数（默认 3，0 = 禁用）
    InitialDelay time.Duration // 初始退避延迟（默认 1s）
    MaxDelay     time.Duration // 最大退避延迟（默认 30s）
}

// RetryClient 包装 LLM Client 并添加重试逻辑
type RetryClient struct {
    client Client
    config RetryConfig
}

// 可重试的错误类型：
// - HTTP 429 (Rate Limited)
// - HTTP 5xx (Server Error)
// - Network errors (connection refused/reset, EOF)
// - 支持 Retry-After 响应头解析

func (rc *RetryClient) Complete(ctx context.Context, req *Request) (*Response, error)
func (rc *RetryClient) Stream(ctx context.Context, req *Request) (<-chan Chunk, error)
func (rc *RetryClient) Embed(ctx context.Context, texts []string) ([][]float32, error)
// 退避公式: delay = min(InitialDelay * 2^attempt + jitter, MaxDelay)
// jitter = random(0, 100ms)
```

### HabitManager 接口 (`habit/pkg/manager/manager.go`)

```go
// Manager 统一管理习惯记录、分析和查询
type Manager struct {
    tracker  *tracker.Tracker
    analyzer *analyzer.Analyzer
    store    *storage.Storage
}

// 核心方法
func (m *Manager) RecordAction(ctx context.Context, userID string, action *Action) error
func (m *Manager) GetHabits(ctx context.Context, userID string) ([]*Habit, error)
func (m *Manager) RefreshHabits(ctx context.Context, userID string) ([]*Habit, error)
func (m *Manager) GetPreferences(ctx context.Context, userID string) ([]*Preference, error)
func (m *Manager) DeleteHabit(ctx context.Context, userID, habitID string) error
func (m *Manager) Cleanup(ctx context.Context, userID string, maxAge time.Duration) error

// Habit 习惯模型
type Habit struct {
    ID         string    // 唯一标识
    UserID     string    // 用户标识（隔离关键）
    SessionID  string    // 会话标识
    Name       string    // 习惯名称
    Category   string    // tool_usage/command/style/preference/workflow
    Pattern    Pattern   // 触发模式
    Frequency  Frequency // 频率统计
    Confidence float64   // 置信度 0-1
    LastSeen   time.Time
}

// Pattern 触发模式
type Pattern struct {
    Context    string     // 触发上下文
    Keywords   []string   // 关键词触发
    ToolUsage  []string   // 工具使用模式
    CommandSeq []string   // 命令序列
}

// Frequency 频率统计
type Frequency struct {
    TotalCount int     // 总次数
    DailyAvg   float64 // 日均次数
    WeeklyAvg  float64 // 周均次数
    Trend      string  // increasing/stable/decreasing
}

// Action 操作记录
type Action struct {
    ID          string
    UserID      string
    SessionID   string
    Timestamp   time.Time
    Input       string     // 用户输入
    ToolsUsed   []string   // 使用的工具
    OutputStyle string     // 输出风格
    Duration    time.Duration
    Feedback    string     // 用户反馈
}

// Preference 用户偏好
type Preference struct {
    ID        string
    UserID    string
    Category  string    // tool/style/format
    Name      string    // 偏好名称
    Value     string    // 偏好值
    Source    string    // explicit/implicit
    UpdatedAt time.Time
}
```

### 任务追踪工具 (`tools/pkg/tasktool/`)

内置任务追踪工具，支持任务的创建、更新和列表查询：

```go
// Task 表示一个任务
type Task struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    Status      string    `json:"status"` // pending/in_progress/completed/cancelled
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// TaskList 管理任务集合
type TaskList struct {
    tasks map[string]*Task
    mu    sync.RWMutex
}

func (l *TaskList) Create(name, description string) *Task
func (l *TaskList) Update(id, status string) (*Task, error)
func (l *TaskList) List() []*Task
func (l *TaskList) Restore(tasks []*Task)

// TaskStore 负责任务持久化
type TaskStore struct {
    dir string
}

func (s *TaskStore) Load() ([]*Task, error)
func (s *TaskStore) Save(tasks []*Task) error

// TaskTool 实现 tools.Tool 接口
type TaskTool struct {
    taskList  *TaskList
    saveFn    func()
    eventCh   chan<- events.Event
    requestID string
}

func (t *TaskTool) Name() string  // "task"
func (t *TaskTool) Execute(ctx context.Context, params map[string]any) (string, error)
// Actions: create, update, list
```

**TUI 任务组件** (`cli/pkg/tui/task_widget.go`):
```go
// TaskWidget 在 TUI 中显示任务列表
type TaskWidget struct {
    tasks []*Task
}

func (w *TaskWidget) Render() string
```

### Location 工具 (`tools/pkg/location/location.go`)

IP 地理位置检测，支持配置覆盖和缓存：

```go
// LocationTool 提供 IP 地理位置
type LocationTool struct {
    client *http.Client
    cfg    LocationConfig
    cache  *LocationData
    expiry time.Time
}

type LocationConfig struct {
    FixedCity    string  // 固定城市（覆盖自动检测）
    FixedCountry string  // 固定国家
    AutoDetect   bool    // IP 自动检测
}

type LocationData struct {
    City        string  `json:"city"`
    Region      string  `json:"region"`
    Country     string  `json:"country"`
    CountryCode string  `json:"country_code"`
    Lat         float64 `json:"lat"`
    Lon         float64 `json:"lon"`
    Source      string  `json:"source"`
}

// APIs: ipinfo.io (primary), ip-api.com (fallback)
// Cache TTL: 24 hours
```

### EngineFactory (`core/pkg/factory/engine_factory.go`)

EngineFactory 支持 sessionID 传播到 Engine 以进行任务持久化：

```go
type EngineFactory struct {
    llmClient      llm.Client
    config         *config.AgentConfig
    permMgr        *permissions.Manager
    systemPrompt   string
    commands       commands.Command
    confirmHandler engine.ToolConfirmationHandler
    logger         *logger.Logger
    dataDir        string    // 会话数据目录
    sessionID      string    // 会话 ID（用于任务持久化）
}

func NewEngineFactory(llmClient llm.Client, cfg *config.AgentConfig, permMgr *permissions.Manager, opts ...EngineFactoryOption) *EngineFactory
func (f *EngineFactory) Create(mem memory.Memory) (*engine.Engine, error)
func (f *EngineFactory) CreateWithSession(sessionID string, mem memory.Memory) (*engine.Engine, error)

// 配置选项
func WithSystemPrompt(prompt string) EngineFactoryOption
func WithCommands(cmdProvider commands.Command) EngineFactoryOption
func WithConfirmationHandler(handler engine.ToolConfirmationHandler) EngineFactoryOption
func WithLogger(log *logger.Logger) EngineFactoryOption
func WithDataDir(dataDir string) EngineFactoryOption
func WithSessionID(sessionID string) EngineFactoryOption
```

### SDK SessionManager (`core/pkg/sdk/session.go`)

SessionManager 提供用户级会话管理，绑定 userID，调用方无需每次传递：

```go
type SessionManager struct {
    wrapper *sessionMgr.SessionServiceWrapper
    userID  string
}

func NewSessionManager(dataDir string, userID string, cfg *config.Config) (*SessionManager, error)
func (m *SessionManager) ListSessions() ([]SessionInfo, error)
func (m *SessionManager) CreateSession(name, role string) (*SessionInfo, error)
func (m *SessionManager) GetSession(id string) (*SessionInfo, error)
func (m *SessionManager) GetMostRecentSession() (*SessionInfo, error)
func (m *SessionManager) DeleteSession(id string) error
func (m *SessionManager) UpdateSessionRole(id, role string) error
func (m *SessionManager) GetSubscriptions(sessionID string) ([]SubscriptionInfo, error)
func (m *SessionManager) AddSubscription(sessionID, trigger, source string) error
func (m *SessionManager) RemoveSubscription(sessionID, subscriptionID string) error
func (m *SessionManager) ToggleSubscriptionStatus(sessionID, subscriptionID string) error
func (m *SessionManager) GetOrCreateSession(namingFormat string) (string, error)
```

### SDK Skill 加载 (`core/pkg/sdk/skills.go`)

```go
type SkillInfo struct {
    Name                 string
    Description          string
    Body                 string
    FilePath             string
    PermissionLevel      string
    RequiresConfirmation bool
}

func LoadSkills(directories []string) ([]SkillInfo, error)
```

### SDK Orchestrator (`core/pkg/sdk/orchestrator.go`)

```go
type (
    Orchestrator       = orchestratorpkg.Orchestrator
    OrchestratorConfig = orchestratorpkg.OrchestratorConfig
    SubAgent           = orchestratorpkg.SubAgent
    CollaboDoc         = orchestratorpkg.CollaboDoc
    DocStatus          = orchestratorpkg.DocStatus
    Priority           = orchestratorpkg.Priority
)

const (
    DocStatusPending    = orchestratorpkg.DocStatusPending
    DocStatusInProgress = orchestratorpkg.DocStatusInProgress
    DocStatusCompleted  = orchestratorpkg.DocStatusCompleted
    DocStatusRejected   = orchestratorpkg.DocStatusRejected
    DocStatusBlocked    = orchestratorpkg.DocStatusBlocked
)

func NewOrchestrator(cfg *config.Config) (*Orchestrator, error)
func NewSpawnAgentTool(o *Orchestrator) tools.Tool
func NewCreateDocTool(o *Orchestrator) tools.Tool
func NewProcessDocTool(o *Orchestrator, agentID string) tools.Tool
func NewQueryQueueTool(o *Orchestrator) tools.Tool
```

### SDK MCP 管理 (`core/pkg/sdk/mcp.go`)

MCP 管理函数已提升至 SDK 导出层：

```go
func AddMCPServer(ctx context.Context, name, command string, args []string) ([]tools.Tool, error)
func RemoveMCPServer(ctx context.Context, name string) error
func ListMCPServers() []ServerInfo
func GetMCPServerStatus(ctx context.Context, name string) *ServerInfo
func LoadMCPConfig() (*MCPConfig, error)
```

### SDK 运行时类型 (`core/pkg/sdk/sdk.go`)

```go
type RuntimeMode = runtime.RuntimeMode
const (
    RuntimeModeCLI = runtime.RuntimeModeCLI
    RuntimeModeTUI = runtime.RuntimeModeTUI
    RuntimeModeAPI = runtime.RuntimeModeAPI
)

type MessageSource = memory.MessageSource
const (
    SourceCLI = memory.SourceCLI
    SourceTUI = memory.SourceTUI
    SourceAPI = memory.SourceAPI
)

type (
    Command           = commands.Command
    Summarizer        = memory.Summarizer
    IntentService     = intent.Service
    PermissionManager = permissions.Manager
)
```

---

## 数据流

### ReAct 循环数据流

```
用户输入
    │
    ▼
┌─────────────────┐
│ runtime.Process()│ 处理输入
└─────────────────┘
    │
    ▼
┌─────────────────────────────────────────┐
│ Engine.ReAct()  ReAct 推理循环            │
│ 1. 思考 (Thinking)                       │
│ 2. 选择工具 (Action)                      │
│   - 单 Action → 串行执行                   │
│   - 多 Action → 并行执行（信号量限并发）     │
│ 3. 执行工具 (Observation)                 │
│   - 聚合所有观察结果为一条记忆条目            │
│ 4. 无 Action → 输出最终响应               │
│ 5. 检查 MaxSteps → 超过则截断              │
└─────────────────────────────────────────┘
    │
    ▼
┌─────────────────┐
│ Event Stream    │ 返回事件流
└─────────────────┘
    │
    ├──────┬──────┬──────┬───────┬────────┐
    ▼      ▼      ▼      ▼       ▼        ▼
Thinking Action Result Response Task    Done
    │      │      │      │       │        │
    ▼      ▼      ▼      ▼       ▼        ▼
 UI 显示 UI 显示 UI 显示 UI 显示 UI 显示  UI 显示
```

### Adapter 消息处理流程

```
外部平台消息 (Feishu/Email)
    │
    ▼
┌─────────────────────────┐
│ AdapterResourceManager  │
│ GetOrCreateSession()    │ 根据用户标识获取/创建会话
└─────────────────────────┘
    │
    ▼
┌─────────────────────────┐
│ AdapterResourceManager  │
│ GetRuntime()            │ 获取/创建运行时
└─────────────────────────┘
    │
    ▼
┌─────────────────────────┐
│ Runtime.Process()       │ 处理消息
│ - memory.Add(User)      │ 添加用户消息
│ - Engine.ReAct()        │ 执行 ReAct 循环
│ - SessionMemory 持久化  │ 由 Engine 协调持久化
└─────────────────────────┘
    │
    ▼
┌─────────────────────────┐
│ Event Stream            │ 返回事件流给外部平台
└─────────────────────────┘
```

**Adapter 事件处理说明：**
- `SessionMemory` 负责所有消息持久化，确保与存储同步
- Tool 事件（`EventTypeToolStart`/`EventTypeToolEnd`）仅用于调试，不持久化
- Response 事件由 `Engine.runReActLoop` 通过 `memory.AddWithType` 持久化

### Runtime 初始化流程（9 阶段）

```
Runtime.Initialize() 9 阶段初始化：

Phase 1: initClients()
  ├─ HTTP Client（标准 + Web 专用）
  ├─ LLM Client（通过 Factory）
  └─ Permission Manager

Phase 2: initSkillSystem()
  ├─ Skill Loader（扫描目录）
  ├─ Skill Injector（渐进式披露）
  └─ Intent Service（意图识别）

Phase 3: initAgentSystem()
  └─ Agent Loader（扫描 AGENT.md）

Phase 4: initMemoryAndPrompt()
  ├─ SessionMemory（JSONL 持久化）
  └─ PromptBuilder（角色 + 档案）

Phase 5: initPromptCachePreEngine()
  └─ PromptCacheManager（创建缓存层）

Phase 6: initEngine()
  ├─ Engine 创建（含所有配置）
  └─ 注册 Options（Thinking、Cache、Hooks）

Phase 7: initTools()
  ├─ 内置工具注册
  ├─ MCP 工具注册
  └─ 任务工具注册

Phase 8: initPromptCachePostTools()
  └─ 设置 Tools 缓存块

Phase 9: initPostSetup()
  ├─ Hooks Engine
  ├─ Agent 委托函数
  └─ Habit Manager

子 Agent 快速路径：
  ├─ Skip phases 1-8
  ├─ 共享父级 SharedResources（read-only）
  ├─ 共享 SkillSystem/AgentSystem/MCPSystem/HookSystem
  └─ 仅创建新 SessionContext（isolated）
```

### Agent 委托流程（更新）

```
1. 用户输入 → Engine.ReAct 循环
2. LLM 决定委托 → 生成 command_delegate_to_agent
3. delegateFn() 执行：
   ├─ 从 AgentSystem.loader 查找目标 Agent
   ├─ 读取 AgentMeta 权限配置：
   │   ├─ PermissionMode: inherit/inherit_downgrade/independent
   │   ├─ PermissionLevel: allow/ask/deny（用于 downgrade）
   │   └─ ClonePermMgrWithDowngrade() 如需降级
   ├─ 构建子 Agent system prompt（agent body + task）
   ├─ NewSubAgentRuntime():
   │   ├─ 共享 SharedResources（LLM、HTTP、PermMgr）
   │   ├─ 共享 SkillSystem/AgentSystem/MCPSystem/HookSystem
   │   ├─ 创建独立 SessionContext
   │   └─ 排除 disable_tools
   ├─ 执行任务并收集事件
   └─ 如 UseReviewer=true:
   │   └─ 委托 ReviewerAgent 验证结果
4. 返回结果给主 Engine
```

---

## 配置与存储

### 配置文件位置

| 文件 | 路径 | 用途 |
|------|------|------|
| `config.yaml` | `~/.aura/config.yaml` | 主配置（LLM、工具、权限、Thinking、Prompt Cache） |
| `profile.yaml` | `~/.aura/profile.yaml` | 用户档案和风格偏好 |
| `roles/*.md` | `~/.aura/roles/{role}.md` | 会话角色定义 |
| `skills/*/SKILL.md` | `~/.aura/skills/` | Skill 定义（Prompt 模板） |
| `agents/*/AGENT.md` | `~/.aura/agents/` | Agent 定义（系统提示模板、权限继承） |
| `mcp.json` | `~/.aura/mcp.json` | MCP 服务器配置（stdio + HTTP/SSE） |
| `hooks.yaml` | `~/.aura/hooks.yaml` | 服务器级子进程钩子配置 |

### 数据存储位置

| 数据 | 路径 | 格式 |
|------|------|------|
| 会话数据 | `~/.aura/sessions/` | JSONL |
| 对话记忆 | `~/.aura/memory/` | SQLite |
| 知识库 | `~/.aura/knowledge/` | Chroma DB |
| 用户习惯 | `~/.aura/users/{user_id}/habits/` | JSONL |
| Skills | `~/.aura/skills/` | Markdown (SKILL.md) |
| Agents | `~/.aura/agents/` | Markdown (AGENT.md) |

### 配置结构

```yaml
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: qwen3:8b
  api_key: ${API_KEY}
  embedding_model: nomic-embed-text

memory:
  type: sqlite
  storage_dir: ~/.aura/memory
  max_context: 50
  max_tokens: 8000           # 最大令牌数（触发修剪）
  summary_threshold: 0.7     # 令牌比率触发摘要（0.0-1.0）
  context_threshold: 0.8     # 上下文窗口令牌比率（0.0-1.0）

tools:
  enabled:
    - file_read
    - file_write
    - bash
    - datetime
    - calculator

log:
  level: info
  format: text
  output: stdout

permissions:
  default_level: ask
  tools:
    file_read: allow
    file_write: ask
    bash: ask

agents:
  enabled: true
  directories:
    - ~/.aura/agents

agent:
  planning_mode: implicit      # implicit/explicit/auto
  temperature: 0.7
  summary_temp: 0.3

  # Plan 模式配置 - 新增
  plan:
    enable_review: true        # 启用计划审查
    verify_commands:           # 验证阶段命令
      - "make test"
    use_reviewer_agent: true   # 使用 reviewer agent
    reviewer_agent: "code-reviewer"
    parallel_explore: true     # 并行探索
    max_parallel_explore: 3    # 最大并行探索数

llm:
  retry:
    max_retries: 3             # 最大重试次数（0 = 禁用）
    initial_delay: 1s          # 初始退避延迟
    max_delay: 30s             # 最大退避延迟

  # Thinking 配置 - 新增
  thinking:
    enabled: true
    reasoning_effort: medium   # OpenAI: low/medium/high
    budget_tokens: 10000       # Anthropic: 最大思考令牌

  # Prompt Cache 配置 - 新增
  prompt_cache:
    enabled: true              # 启用提示缓存

# Hooks 配置 - 新增
hooks:
  enabled: true
  config_path: ~/.aura/hooks.yaml

# MCP 服务器配置（存储在 ~/.aura/mcp.json）
# 使用 aura mcp add/list/remove/status 命令管理

orchestrator:
  enabled: true
  max_sub_agents: 5

# 意图识别配置
intent:
  enabled: true              # 启用自然语言命令识别
  confidence_threshold: 0.7  # 置信度阈值（0.0-1.0）

# 习惯追踪配置
habit:
  enabled: true              # 启用习惯追踪
  min_occurrences: 3         # 模式出现次数阈值
  conf_threshold: 0.3        # 置信度阈值（0.0-1.0）
  max_action_age_days: 30    # 操作最大天数
  analysis_limit: 500        # 每次分析最大操作数

# 调试配置
debug:
  show_tokens: true          # 在 TUI 显示令牌使用
  log_tokens: true           # 记录令牌变化
  log_llm_interactions: true # 记录 LLM 请求/响应到文件
```

---

## 设计模式

### 工厂模式
- `LLMFactory` - 根据配置创建 LLM 客户端
- `EngineFactory` - 创建 Engine 实例
- `ToolRegistry` - 工具注册和查找

### 事件流模式
所有模式（CLI/TUI/API）使用统一的事件流接口处理输出

### 依赖注入
通过 `RuntimeOption` 模式注入配置（事件处理器、确认处理器、会话存储等）

### 接口驱动
所有核心组件由接口定义，支持多种实现

### 渐进式披露（Progressive Disclosure）
技能系统使用 `SkillMatcher`（LLM 语义匹配 + 关键词回退）分析用户输入匹配相关技能，通过 `SkillInjector` 动态注入到系统提示，而不是每次提示都包含所有技能内容。

### 并行工具执行
ReAct 循环支持单步内并行执行多个独立工具：
- `parseActions()` — 从 LLM 响应中提取所有 `Action:` 行
- `executeToolsParallel()` — 使用信号量限制并发数（默认 5），并发执行工具
- `executeToolsSerial()` — 单工具时自动回退串行执行
- 所有观察结果聚合成一条记忆条目
- 配置：`EngineConfig.MaxParallelTools`（0 = 默认 5，1 = 串行）

### 共享管理器框架
AgentManager 和 SkillManager 已统一委托给 `shared/pkg/manager` 中的通用 Item Manager：
- `Item` 接口（GetName/GetDescription/GetFilePath/GetContent/GetBody）由 Agent 和 Skill 实现
- `ManagerConfig` 通过回调注入类型特定逻辑（Validate/BuildContent/ConstructItem/MergeUpdate/FindByName）
- Loader 提供 `GetItems() []sharedmanager.Item` 支持列表操作
- 消除了 Agent 和 Skill 管理器之间 ~200 行重复 CRUD 代码

### Hooks Framework
服务器级子进程钩子，在关键执行点触发：
- **配置文件**：`~/.aura/hooks.yaml`
- **事件类型**：
  - `SessionStart` - 会话开始
  - `UserPromptSubmit` - 用户提交
  - `PreToolUse` - 工具执行前（可阻塞）
  - `PostToolUse` - 工具执行后
  - `Stop` - 请求结束
  - `PreCompact/PostCompact` - 压缩前后
  - `EnterPlanMode/ExitPlanMode` - 规划模式进出
- **HookOutput 影响**：
  - `SystemMessage` — 注入系统提示
  - `Continue` — false = 阻止执行
  - `PermissionDecision` — 覆盖权限（allow/deny）
  - `ShouldRetry` — PreResponse 重试
  - `ReflectionFeedback` — 质量反馈
- **阻塞行为**：Exit code 2 阻止主流程继续

见 [hooks/types.go](shared/pkg/hooks/types.go)

### Runtime Components 模式
六大组件封装运行时资源，支持轻量级子 Agent 继承：
- `SharedResources` — LLM/HTTP client, permMgr (read-only, 线程安全)
- `SkillSystem` — loader/injector/intentSvc (nil when disabled)
- `AgentSystem` — loader/delegateFn (nil when disabled)
- `MCPSystem` — manager/started flag (nil when disabled, 子 Agent 无法停止父级服务器)
- `HookSystem` — engine (nil when no hooks.yaml)
- `SessionContext` — sessionID/userID/store/taskList (isolated per sub-agent)

**子 Agent 继承规则**：
- SharedResources 共享（read-only）
- SkillSystem/AgentSystem/MCPSystem/HookSystem 共享（指针传递）
- SessionContext 独立创建（资源隔离）

见 [components.go](core/pkg/runtime/components.go)

### Prompt Caching Pattern
5 层缓存架构，支持 Anthropic 和 OpenAI 提示缓存：
- **Layer 0**: StaticSystem — 静态系统提示（最高缓存率）
- **Layer 1**: Tools — 工具定义
- **Layer 2**: Skills — 技能元数据
- **Layer 3**: Agents — Agent 元数据
- **Layer 4**: ProjectAura — 项目 AURA.md
- **Provider 实现**：
  - Anthropic: `SystemBlocks` + `cache_control: {type: "ephemeral"}`
  - OpenAI: `cache_type: "ephemeral"` 在请求级
- **缓存指标**：`Usage.CacheCreationInputTokens` / `CacheReadInputTokens`

见 [prompt/cache_manager.go](core/pkg/prompt/cache_manager.go)

### Plan Mode Pattern
显式规划模式支持完整的计划生命周期：
- **PlanModeState 状态流转**：none → explore → design → review → execute → verify
- **阶段限制**：
  - Explore: 只读工具（ToolAllowlist 限制）
  - Execute: 全工具可用
- **ToolAllowlist**：阶段性工具限制（`explorationTools` 列表）
- **Rollback 集成**：
  - SnapshotCreated — 执行前创建 git stash
  - RollbackOffer — 失败时提供回滚选项
  - RollbackComplete — 恢复或清理
- **Reviewer Agent**：`UseReviewer=true` 时委托验证

配置：`plan.enable_review`, `plan.verify_commands`, `plan.parallel_explore`

### TypedManager 泛型管理
类型安全的泛型 CRUD 管理器：
- `TypedManager[T Item]` — 包装 `Manager` 提供类型安全 API
- `TypedConfig[T]` — 类型特定回调（Validate/BuildContent/ConstructItem/MergeUpdate）
- 操作：Create/Update/Delete/Get/List/Reload/FindByName
- 消除类型断言，SkillManager/AgentManager 共享框架

见 [typed_manager.go](shared/pkg/manager/typed_manager.go)

---

## 工具分类

### 安全工具（无需确认）
- `file_read`, `file_list`, `file_search` - 文件读取
- `datetime`, `calculator`, `text` - 工具类
- `web_fetch`, `web_search` - 网络类
- `knowledge_search`, `code_navigate` - 查询类
- `task` - 任务追踪

### 敏感工具（需要确认）
- `file_write`, `knowledge_import` - 写入类
- `bash`, `ssh_exec` - 执行类

---

## 测试

### 运行测试

```bash
# 运行所有测试
make test

# 运行特定包测试
go test -v ./core/pkg/...

# 运行覆盖率
make test-coverage
```

### 测试框架

| 模块 | 测试框架 |
|------|----------|
| 所有模块 | `testing` + `testify` |

---

## 参考资料

- [Skill 系统](../modules/skill/README.md)
- [Agent 测试](agent-testing.md)
