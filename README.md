# Aura 使用说明

Aura 是一个个人 AI 助手，支持多 Agent 协作、任务规划、知识库管理等功能。

**核心特性**：
- **单次查询/交互式 CLI/TUI 全屏/API 服务器** - 多种使用模式
- **Agent 委托系统** - LLM 可动态生成和委托子 Agent 完成任务
- **Multi-Agent Orchestrator** - 支持多 Agent 协作完成复杂任务
- **任务规划** - 显式/隐式/自动三种规划模式
- **知识库 (RAG)** - 本地向量数据库，支持语义搜索
- **Skill 系统** - 通过 Markdown 定义自定义技能
- **会话管理** - 多会话、角色系统、订阅路由
- **个人档案** - 自适应学习用户偏好
- **习惯学习** - 自动识别用户操作习惯（工具偏好、命令序列、输出风格）
- **MCP 集成** - 支持接入外部 Model Context Protocol 服务器（stdio + HTTP/SSE 传输）

## 快速开始

### 1. 构建

```bash
cd /path/to/aura
make build
```

### 2. 配置

```bash
# 复制配置模板
cp configs/config.yaml ~/.aura/config.yaml

# 编辑配置，设置 LLM 提供商和 API Key
vim ~/.aura/config.yaml
```

**前置要求：**
- Go 1.26.1+
- Ollama 服务器运行中，并安装模型：`ollama pull qwen3:8b`
- Embedding 模型（用于知识库）：`ollama pull nomic-embed-text`

### 3. 运行

```bash
./bin/aura --help
```

---

## 使用模式

### 单次查询模式

直接传入消息，获取一次响应后退出：

```bash
./bin/aura "你好，介绍一下你自己"
./bin/aura "今天是几月几日？"
./bin/aura "计算 123 * 456"
```

### 交互式 CLI 模式

```bash
./bin/aura
```

进入交互模式后可以使用以下命令：
- `/exit`、`/quit` 或 `/q` - 退出
- `/clear` - 清屏
- `/help` - 显示帮助
- `/sessions` - 列出所有会话
- `/session create` - 创建新会话
- `/skills` - 列出已加载技能
- `/tools` - 列出可用工具
- `/config` - 显示配置
- `/knowledge` - 搜索知识库
- `/profile` - 显示用户档案
- `/memory` - 显示内存使用
- `/model` - 显示模型信息
- `/compact` - 压缩对话
- `/role` - 显示/设置会话角色
- `/mcp` - 列出 MCP 服务器
- `/version` - 显示版本

### TUI 全屏模式

```bash
./bin/aura --tui
```

TUI 模式提供全屏交互界面，具有更好的视觉体验和会话管理界面。

**键盘快捷键**：
| 快捷键 | 功能 | 快捷键 | 功能 |
|--------|------|--------|------|
| `Ctrl+L` | 清屏 | `Ctrl+T` | 列出工具 |
| `Ctrl+H` | 帮助 | `Ctrl+P` | 用户档案 |
| `Ctrl+S` | 切换会话 | `Ctrl+M` | MCP 服务器 |
| `Ctrl+Q` | 退出 | `Ctrl+K` | 技能列表 |
| `Ctrl+D` | 压缩内存 | `Ctrl+E` | 执行状态 |

**命令补全**：输入 `/` 后按 `Tab` 可自动补全命令列表，`↑/↓` 导航，`Tab` 选中补全。

**界面特性**：全屏 Alt-Screen 模式，支持鼠标滚动查看历史消息，工具执行结果以结构化 IN/OUT 块展示。

### 禁用工具模式

```bash
./bin/aura --no-tools "纯聊天，不使用任何工具"
```

### Web UI 模式

```bash
./bin/aura serve --web
```

Server 模式提供 Web 聊天界面，支持 Markdown 渲染、实时消息流和流式渲染（闪烁光标显示响应）。

### API 服务器模式

启动 HTTP API 服务器，支持 24/7 会话管理和外部触发：

```bash
./bin/aura serve
./bin/aura serve --port 8080
./bin/aura serve --llm-url http://localhost:11434 --llm-model qwen3:8b
```

**注意：** `serve` 命令会优先使用 `~/.aura/config.yaml` 中的配置（如 LLM URL、模型等），命令行参数会覆盖配置值。

---

## Agent 系统

Aura 支持两种 Agent 模式：预定义 Agent 和动态生成的 SubAgent。

### Agent 委托系统

Agent 委托允许 LLM 根据任务需求动态生成或选择子 Agent 来执行特定任务。

**预定义 Agent** 存储在 `~/.aura/agents/{name}/AGENT.md`：

```bash
# 示例：创建一个代码审查 Agent
mkdir -p ~/.aura/agents/code-reviewer
cat > ~/.aura/agents/code-reviewer/AGENT.md << 'EOF'
---
name: code-reviewer
description: Review code for quality, security issues, and best practices
llm_model: qwen3:8b
disable_tools: bash,file_write
temperature: 0.3
---

## Role
You are an expert code reviewer specializing in Go projects.

## Guidelines
1. Check for code quality and best practices
2. Identify potential security vulnerabilities
3. Suggest improvements for maintainability
EOF
```

**配置继承**：Agent 可以继承全局配置并覆盖特定字段：
- `planning_mode` - 规划模式（implicit/explicit/auto）
- `temperature` - LLM 温度
- `summary_temp` - 摘要温度

**注意**：子 Agent 默认继承父级的 LLM 配置（`llm_model` 字段仅用于独立执行场景）。如需使用不同模型，应修改父级配置。

### Multi-Agent Orchestrator

Orchestrator 支持多 Agent 协作完成复杂任务，提供：
- **任务分解** - 将复杂任务分解为多个子任务
- **协作文档** - Agent 间通过文档进行异步通信
- **健康监督** - 检测停滞 Agent 和陈旧文档
- **工作队列** - 任务调度和负载均衡

**专用工具**：
| 工具 | 功能 |
|------|------|
| `spawn_agent` | 生成子 Agent |
| `create_collaboration_doc` | 创建协作文档 |
| `process_collaboration_doc` | 处理协作文档 |
| `query_work_queue` | 查询工作队列 |

---

## 可用工具

### 文件系统工具

| 工具 | 功能 | 示例 |
|------|------|------|
| `file_read` | 读取文件 | "读取 /etc/hosts" |
| `file_write` | 写入文件（需确认） | "把 hello 写入 /tmp/test.txt" |
| `file_search` | 文件内容搜索 | "在 main.go 中搜索 func" |
| `file_list` | 列出目录 | "列出当前目录有哪些文件" |

### 搜索工具（Claude Code 风格）

| 工具 | 功能 | 示例 |
|------|------|------|
| `glob` | 文件模式匹配（如 `**/*.go`） | "查找所有 *.go 文件" |
| `grep` | 正则表达式内容搜索 | "搜索所有包含 'TODO' 的行" |

**后端说明：**
- `glob`: 优先使用 `find` 命令，Fallback 到 Go `filepath.Glob`
- `grep`: 优先使用 `rg` (ripgrep)，Fallback 到 `grep` 命令，最终 Fallback 到 Go 原生实现

**性能提示：** 安装 ripgrep 后 `grep` 工具速度可提升 5-10 倍：
```bash
# macOS
brew install ripgrep

# Linux
apt-get install ripgrep

# 验证安装
rg --version
```

### 系统工具

| 工具 | 功能 | 示例 |
|------|------|------|
| `bash` | 执行 shell 命令（需确认） | "运行 ls -la" |
| `datetime` | 获取日期时间 | "现在几点了" |
| `calculator` | 计算器 | "计算 2 + 3 * 4" |
| `text` | 文本处理 | "将这段文字转为大写" |
| `internalcmd` | 内部命令执行（自然语言触发） | "退出程序" / "显示帮助" |

**internalcmd 工具说明：**
支持通过自然语言执行内部命令，如 `command_exit`（退出）、`command_help`（帮助）、`command_clear`（清屏）等。LLM 可自动识别用户意图并调用相应命令。

### 网络工具

| 工具 | 功能 | 示例 |
|------|------|------|
| `web_fetch` | 获取网页内容 | "获取 https://example.com 的内容" |
| `web_search` | 网络搜索 | "搜索 Go 语言最新特性" |
| `location` | IP 地理位置（支持缓存） | "我现在在哪里" |

### SSH 远程执行

| 工具 | 功能 | 示例 |
|------|------|------|
| `ssh_exec` | 在远程服务器执行命令（需确认） | "在生产服务器上检查磁盘空间" |

**前置配置：** 在 `~/.aura/config.yaml` 中配置服务器：

```yaml
ssh:
  servers:
    # 密钥认证（推荐）
    - name: "production"
      host: "192.168.1.100"
      port: 22
      user: "ubuntu"
      key_path: "~/.ssh/id_rsa"

    # 密码认证
    # - name: "production"
    #   host: "prod.example.com"
    #   port: 22
    #   user: "deploy"
    #   password: "your-password-here"
```

使用时直接使用服务器名称：
```bash
./bin/aura "在生产服务器上检查磁盘空间"
```

### LSP 代码导航

Aura 集成了 gopls（Go 语言服务器），提供专业级的代码理解能力：

| 操作 | 说明 | 必需参数 |
|------|------|---------|
| `definition` | 查找符号定义 | file, line, column |
| `references` | 查找所有引用 | file, line, column |
| `symbols` | 列出文件符号 | file |
| `format` | 格式化代码 | file |
| `diagnostics` | 检查错误/警告 | file |
| `rename` | 重命名符号 | file, line, column, newName |

**前置要求：**
```bash
# 安装 gopls
go install golang.org/x/tools/gopls@latest
```

### 知识库工具

| 工具 | 功能 | 示例 |
|------|------|------|
| `knowledge_search` | 搜索知识库 | "搜索 Go 语言相关知识" |
| `knowledge_import` | 导入文档到知识库（需确认） | "导入 /tmp/doc.md 到知识库" |

**前置要求：** 需要安装 embedding 模型
```bash
# Ollama 服务器执行
ollama pull nomic-embed-text
```

### 任务追踪工具

| 工具 | 功能 | 示例 |
|------|------|------|
| `task` | 任务追踪（create/update/list） | "创建一个任务：实现用户登录功能" |

内置任务追踪工具，支持任务的创建、更新和列表查询。任务数据按会话持久化到 `~/.aura/sessions/` 目录，TUI 模式提供任务列表组件可视化显示。

**支持的操作：**
- `create` - 创建新任务（指定名称和可选描述）
- `update` - 更新任务状态（如标记为 in_progress、completed、cancelled）
- `list` - 列出所有任务

### MCP 外部工具

| 工具 | 功能 | 示例 |
|------|------|------|
| `mcp__{server}__{tool}` | 外部 MCP 服务器工具 | 取决于配置的服务器 |

MCP (Model Context Protocol) 允许接入外部工具服务器，支持两种传输方式：

**stdio 传输（默认）：**
```bash
# 添加文件系统 MCP 服务器
./bin/aura mcp add filesystem -- npx -y @modelcontextprotocol/server-filesystem ~/.aura/workspace

# 添加 fetch MCP 服务器
./bin/aura mcp add fetch -- uvx mcp-server-fetch
```

**HTTP/SSE 传输：**
直接连接已有 MCP 服务器的 HTTP 端点，无需启动本地进程。配置 `~/.aura/mcp.json`：
```json
{
  "mcpServers": {
    "my-http-server": {
      "type": "http",
      "url": "https://example.com/sse",
      "headers": {
        "Authorization": "Bearer xxx"
      }
    }
  }
}
```

**管理命令：**
```bash
./bin/aura mcp list               # 列出已配置的 MCP 服务器
./bin/aura mcp status [name]      # 查看服务器状态
./bin/aura mcp remove <name>      # 移除服务器
```

**交互模式 `/mcp` 命令：** CLI 和 TUI 模式下输入 `/mcp` 即可查看服务器实时状态（名称、运行状态、工具数、错误信息）。

---

## 管理命令

```bash
# 查看配置
./bin/aura config

# 列出可用工具
./bin/aura tools

# 查看版本号
./bin/aura version

# 个人档案管理
./bin/aura profile show          # 查看当前档案
./bin/aura profile init          # 创建默认档案
./bin/aura profile import <path> # 从文件导入档案信息

# 知识库管理
./bin/aura knowledge stats       # 显示知识库统计
./bin/aura knowledge search <query>  # 搜索知识库
./bin/aura knowledge import <path>   # 导入文件到知识库

# 会话管理
./bin/aura session list           # 列出所有会话
./bin/aura session create [name]  # 创建新会话
./bin/aura session create "助手" --role=helper  # 使用角色创建会话
./bin/aura session show <id>      # 查看会话详情
./bin/aura session delete <id>    # 删除会话
./bin/aura session subscribe <id> --trigger="keyword" --source="feishu"  # 添加订阅
./bin/aura session update <id> --role=helper     # 更新会话角色
./bin/aura session update <id> --prompt="..."    # 更新自定义提示词

# 习惯管理（多用户模式）
./bin/aura habit list             # 查看当前用户习惯列表
./bin/aura habit show <id>        # 查看习惯详情
./bin/aura habit delete <id>      # 删除习惯
./bin/aura habit refresh          # 重新分析操作记录并刷新习惯
./bin/aura preference list        # 查看用户偏好列表

# MCP 服务器管理
./bin/aura mcp list               # 列出已配置的 MCP 服务器
./bin/aura mcp add <name> -- <command> [args...]  # 添加 MCP 服务器
./bin/aura mcp remove <name>      # 移除 MCP 服务器
./bin/aura mcp status [name]      # 查看 MCP 服务器状态
```

---

## API 服务器

启动 API 服务器后，可以通过 HTTP API 进行会话管理和消息发送。

### 启动服务器

```bash
./bin/aura serve --port 8080
```

**配置优先级：** 命令行参数 > 配置文件 (`~/.aura/config.yaml`) > 默认值

- `--port`：端口（默认 8080）
- `--llm-url`：LLM 地址（默认从 config 读取）
- `--llm-model`：LLM 模型（默认从 config 读取）
- `--data-dir`：数据目录（默认 `~/.aura/sessions`）

### API 端点

| 方法 | 端点 | 说明 |
|------|------|------|
| `GET` | `/api/v1/health` | 健康检查 |
| `GET` | `/api/v1/sessions` | 列出所有会话 |
| `POST` | `/api/v1/sessions` | 创建新会话 |
| `GET` | `/api/v1/sessions/{id}` | 获取会话详情 |
| `GET` | `/api/v1/sessions/{id}/messages` | 获取会话消息 |
| `POST` | `/api/v1/sessions/{id}/message` | 发送消息到会话（异步处理） |
| `DELETE` | `/api/v1/sessions/{id}` | 删除会话 |

### Webhook 端点

| 方法 | 端点 | 说明 |
|------|------|------|
| `POST` | `/api/v1/webhooks/feishu` | 飞书 webhook |
| `POST` | `/api/v1/cron` | 定时任务触发 |
| `POST` | `/api/v1/webhooks/{source}` | 通用 webhook |

### 创建会话示例

```bash
# 创建会话
curl -X POST http://localhost:8080/api/v1/sessions \
  -H "Content-Type: application/json" \
  -d '{"name": "工作助手", "system_prompt": "你是一个专业的工作助手"}'

# 发送消息
curl -X POST http://localhost:8080/api/v1/sessions/{session_id}/message \
  -H "Content-Type: application/json" \
  -d '{"content": "你好，请帮我分析这个项目"}'

# 获取消息历史
curl http://localhost:8080/api/v1/sessions/{session_id}/messages
```

---

## 会话管理系统

Aura 支持多会话管理，每个会话有独立的对话历史和订阅规则。

### 角色系统

会话可以使用预定义的角色来设置系统提示词。角色文件存储在 `~/.aura/roles/{role}.md`。

```bash
# 使用角色创建会话
./bin/aura session create "编程助手" --role=coder

# 更新会话角色
./bin/aura session update <session_id> --role=writer

# 使用自定义提示词（不能用 --role）
./bin/aura session update <session_id> --prompt="你是一个专业的翻译助手"
```

**注意：** `--role` 和 `--prompt` 是互斥的，不能同时使用。

### 会话订阅

订阅用于将外部事件路由到指定会话：

```bash
# 为会话添加飞书订阅（当消息包含"工作"时路由到此会话）
./bin/aura session subscribe <session_id> --trigger="工作" --source="feishu"

# 为会话添加定时任务订阅
./bin/aura session subscribe <session_id> --trigger="每日报告" --source="cron"
```

### 订阅源类型

| 源 | 说明 |
|----|------|
| `feishu` | 飞书消息 |
| `email` | 邮件 |
| `cron` | 定时任务 |
| `api` | API 调用 |
| `cli` | 命令行 |
| `*` | 所有源 |

---

## 配置说明

### 全局配置 `~/.aura/config.yaml`

```yaml
# LLM 提供商设置
llm:
  provider: ollama           # ollama / openai / anthropic
  base_url: http://localhost:11434
  model: qwen3:8b
  api_key: ${API_KEY}        # 使用环境变量
  embedding_model: nomic-embed-text

  # LLM 请求重试配置
  retry:
    max_retries: 3           # 最大重试次数（0 = 禁用）
    initial_delay: 1s        # 初始退避延迟
    max_delay: 30s           # 最大退避延迟

# 记忆系统设置
# Message 架构：单一 ContentBlocks 字段，支持多态内容块（TextBlock、ThinkingBlock、ToolUseBlock、ToolResultBlock）
memory:
  type: sqlite
  storage_dir: ~/.aura/memory
  max_tokens: 8000           # 最大令牌数（触发修剪）
  summary_threshold: 0.7     # 令牌比率触发摘要（0.0-1.0）
  context_threshold: 0.8     # 上下文窗口令牌比率（0.0-1.0）

# 工具启用列表
tools:
  enabled:
    - file_read
    - file_write
    - file_search
    - file_list
    - bash
    - datetime
    - calculator
    - web_fetch
    - knowledge_search
    - knowledge_import
    - code_navigate

# 日志设置
log:
  level: info                # debug / info / warn / error
  format: text               # text / json
  output: stdout             # stdout / 文件路径

# SSH 服务器配置
# 支持两种认证方式：密钥（推荐）和密码
ssh:
  servers:
    # 示例 1: 密钥认证（推荐）
    - name: "my-server"
      host: "192.168.1.100"
      port: 22
      user: "ubuntu"
      key_path: "~/.ssh/id_rsa"

    # 示例 2: 密码认证
    # - name: "production"
    #   host: "prod.example.com"
    #   port: 22
    #   user: "deploy"
    #   password: "your-password-here"

# 权限配置
permissions:
  default_level: ask         # ask / allow / deny

  tools:
    # 只读工具（无需确认）
    file_read: allow
    file_list: allow
    file_search: allow
    datetime: allow
    calculator: allow
    text: allow
    web_fetch: allow
    web_search: allow
    knowledge_search: allow
    code_navigate: allow

    # 写入工具（需要确认）
    file_write: ask
    knowledge_import: ask

    # 执行工具（需要确认 + 命令限制）
    bash: ask
    ssh_exec: ask

  # Shell 命令限制
  shell_restrictions:
    denied_commands:
      - "rm -rf /"
      - "rm -rf /*"
      - "mkfs *"
      - "dd if=*"
      - "curl * | sh"
      - "curl * | bash"
      - "wget * | sh"
      - "wget * | bash"

  # SSH 限制
  ssh:
    allowed_hosts: []
    denied_hosts: []
    allowed_commands: []
    denied_commands: []

# Skill 系统配置
skills:
  enabled: true
  directories:
    - ~/.aura/skills

# Agent 配置
agents:
  enabled: true
  directories:
    - ~/.aura/agents

# Agent 运行时配置
agent:
  # 规划模式：implicit（隐式）/ explicit（显式）/ auto（自动）
  planning_mode: implicit
  # LLM 温度（0.0-1.0，越高越随机）
  temperature: 0.7
  # 摘要温度（用于对话摘要生成）
  summary_temp: 0.3

# LLM 请求重试配置
llm:
  retry:
    max_retries: 3         # 最大重试次数（0 = 禁用）
    initial_delay: 1s      # 初始退避延迟
    max_delay: 30s         # 最大退避延迟

  # Anthropic Thinking 模式配置
  thinking:
    enabled: false         # 启用 extended thinking
    budget_tokens: 10000   # thinking 令牌预算（必须 < max_tokens）
    reasoning_effort: medium  # 推理强度 (low/medium/high)

# Plan 配置（显式规划模式）
plan:
  enable_review: true      # 启用计划审查阶段
  verify_commands:         # 验证命令列表
    - "go build ./..."
    - "go test ./..."
  use_reviewer_agent: false  # 使用 Agent 进行审查
  parallel_explore: true   # 并行探索
  max_parallel_explore: 3  # 最大并行探索数

# 工具超时配置
tools:
  default_timeout: 60      # 默认超时（秒）
  shell_timeout: 120       # Shell 命令超时
  ssh_timeout: 60          # SSH 命令超时
  web_timeout: 30          # Web 请求超时

# 国际化配置
i18n:
  locale: zh-CN            # 当前语言 (en/zh-CN)

# Orchestrator 配置
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

# Hooks 配置（外部脚本集成）
hooks:
  enabled: true              # 启用 hooks 框架
  # 配置文件：~/.aura/hooks.yaml
  # 事件类型：SessionStart, UserPromptSubmit, PreToolUse, PostToolUse, Stop

# Location 配置
location:
  auto_detect: true          # IP 自动检测
  fixed_city: ""             # 固定城市（覆盖自动检测）
  fixed_country: ""          # 固定国家

# Adapters 配置（外部平台集成）
adapters:
  enabled: true              # 启用适配器
  data_dir: ~/.aura/adapters # 适配器数据目录
  feishu:
    app_id: ""               # 飞书 App ID
    app_secret: ""           # 飞书 App Secret

# Users 配置（多用户支持）
users:
  default: "default"         # 默认用户 ID
  definitions:               # 用户定义列表
    - id: "default"
      name: "Default User"
      knowledge_shared: true # 共享知识库
    - id: "user2"
      name: "Second User"
      knowledge_shared: false

# Orchestrator 详细配置
orchestrator:
  enabled: true
  max_sub_agents: 5
  workspace_dir: ~/.aura/workspace
  supervision_interval: 30s  # 监督检查间隔
  stale_doc_threshold: 1h    # 陈旧文档阈值
  auto_cleanup: true         # 自动清理
  sub_agent_llm:             # 子 Agent LLM 配置（可选覆盖）
    model: qwen3:8b

# 调试配置
debug:
  show_tokens: true          # 在 TUI 显示令牌使用
  log_tokens: true           # 记录令牌变化
  log_llm_interactions: true # 记录 LLM 请求/响应到文件
```

---

## 个人档案

个人档案存储于 `~/.aura/profile.yaml`，Aura 会根据档案调整回答风格。

### 管理命令

```bash
# 查看当前档案
./bin/aura profile show

# 创建默认档案模板
./bin/aura profile init

# 从文件导入档案信息（如从简历中提取技能、经历等）
./bin/aura profile import /path/to/resume.md
```

### 配置项

```yaml
basic:
  name: 你的名字
  occupation: 你的职业
  location: 你的位置
  role: 你的角色
  background: 个人背景描述

style:
  tone: casual        # formal（正式）, casual（随意）, technical（技术）
  vocabulary: simple  # simple（简单）, technical（专业）
  humor: 0.3          # 0-1，越高越幽默
  verbosity: concise  # concise（简洁）, detailed（详细）

skills:
  - name: Go
    level: expert
    category: Programming
  - name: Python
    level: intermediate
    category: Programming

experiences:
  - title: 工作经历
    description: 详细描述

preferences:
  - name: 偏好项
    value: 偏好值
```

### 自适应学习

Aura 会在对话中自动学习你的偏好：
- 说 "haha"、"funny"、"lol" → 增加幽默度
- 说 "more detail"、"elaborate" → 增加详细度
- 说 "brief"、"concise"、"too long" → 减少详细度

风格变化会自动保存到档案中。

---

## 知识库

本地向量数据库，支持语义搜索。

### 使用方式

```bash
# 通过自然语言导入
./bin/aura "把 /tmp/doc.md 导入知识库"

# 通过自然语言搜索
./bin/aura "搜索知识库中关于并发的内容"

# 使用命令行管理
./bin/aura knowledge stats          # 显示知识库统计
./bin/aura knowledge search <query> # 搜索知识库
./bin/aura knowledge import <path>  # 导入文件或目录
```

### 支持的格式

- Markdown (`.md`)
- 文本文件 (`.txt`)
- 目录（批量导入）

### 存储位置

知识库存储于 `~/.aura/knowledge/`，使用 Chroma 向量数据库。

---

## Skill 系统

Skill 系统允许通过 Markdown 文件定义自定义技能和行为模式（纯 Prompt 模板，无外部代码执行）。

### 创建 Skill

```bash
# 创建 Skill 目录
mkdir -p ~/.aura/skills/my-skill

# 创建 SKILL.md 文件
cat > ~/.aura/skills/my-skill/SKILL.md << 'EOF'
---
name: my-skill
description: 当用户需要 XXX 时使用此技能
---

## 工作流程

1. 第一步...
2. 第二步...
EOF
```

### 技能触发机制

Aura 使用 **渐进式披露（Progressive Disclosure）** 机制：
- **SkillMatcher**（`core/pkg/skilltool/matcher.go`）— 分析用户输入，匹配相关技能
  - **LLM 语义匹配**（优先）：使用 LLM 进行语义理解，传入用户输入和可选的意图识别结果
  - **关键词回退**：当 LLM 调用失败或未配置 LLM 客户端时，回退到名称/描述关键词匹配
  - 支持别名匹配（如 postgres→postgresql, xlsx→excel）
- **SkillInjector**（`core/pkg/skilltool/injector.go`）— 动态将匹配的技能内容注入系统提示（不是所有技能都包含在每次提示中）
- LLM 根据用户请求和 skill description 判断是否需要参考技能
- 当 LLM 判断需要某技能时，会参考其完整 body 内容

**详细文档：** [Skill System](skill/README.md)

---

## 常见问题

### Ollama 连接失败

```bash
# 检查服务是否运行
curl http://localhost:11434/api/tags

# 重启 Ollama
ollama serve
```

### 知识库导入失败

```bash
# 确认 embedding 模型已安装
curl http://localhost:11434/api/tags | grep nomic

# 安装模型
ollama pull nomic-embed-text
```

### 查看日志

```bash
# 设置 debug 级别日志
export AURA_LOG_LEVEL=debug
./bin/aura "hello"
```

### LSP 工具不可用

```bash
# 确认 gopls 已安装
gopls version

# 安装 gopls
go install golang.org/x/tools/gopls@latest
```

---

## 项目结构

Aura 使用 Go workspace (`go.work`) 管理多个模块，采用分层架构设计。

### 模块依赖层次

```
Layer 4 (应用层):  cli, api, adapters
                        ▲
Layer 3 (核心层):       core (含 Engine、Orchestrator、Planner、Memory 系统)
                        ▲
Layer 2 (服务层):       session, commands
                        ▲
Layer 1 (基础层):  agent, habit, mcp, storage, tools, knowledge, personality, skill, shared
```

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

### 依赖关系图

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
┌─────────────────────────────────────────────────────────────┐
│ Layer 3: Core Layer (核心层)                                 │
│ ┌──────────────────────────────────────────────────────────┐│
│ │  core (agent, commands, session, shared, storage, tools) ││
│ │  - Engine, AgentRuntime, ReAct, LLM, Memory, SDK         ││
│ │  - Planner, Permissions, Orchestrator                    ││
│ └──────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────┘
                            ▲
┌─────────────────────────────────────────────────────────────┐
│ Layer 2: Service Layer (服务层)                              │
│ ┌──────────────────┐  ┌──────────────────────────────────┐  │
│ │   session        │  │   commands                       │  │
│ │   (shared,       │  │   (knowledge, personality,       │  │
│ │    storage)      │  │    session, shared)              │  │
│ └──────────────────┘  └──────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            ▲
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

| 模块 | 依赖的内部模块 |
|------|----------------|
| `shared` | - |
| `storage` | - |
| `tools` | - |
| `knowledge` | - |
| `personality` | - |
| `skill` | - |
| `agent` | `shared` |
| `habit` | `shared`, `storage` |
| `mcp` | `shared`, `tools` |
| `session` | `shared`, `storage` |
| `commands` | `knowledge`, `personality`, `session`, `shared` |
| `core` | `agent`, `commands`, `session`, `shared`, `storage`, `tools`, `knowledge`, `personality`, `skill` |
| `api` | `core`, `session` |
| `adapters` | `core`, `session`, `shared` |
| `cli` | `agent`, `api`, `commands`, `core`, `knowledge`, `personality`, `session`, `tools`, `shared`, `habit` |

**core 模块子包依赖：**
- `core/pkg/engine` - Engine 核心实现 (ReAct 循环、显式规划、工具执行、并行工具执行) → `llm`, `memory`, `planner`, `tools`, `commands`
- `core/pkg/runtime` - 统一运行时（拆分为 boot.go/processing.go/eventing.go/confirmation.go/delegate.go） → `engine`, `factory`, `llm`, `memory`, `permissions`, `prompt`, `skill`, `storage`, `tools`, `agent`, `habit`
- `core/pkg/sdk` - SDK 导出层 → `factory`, `llm`, `permissions`, `prompt`, `runtime`, `session`, `shared`, `orchestrator`, `memory`, `intent`, `commands`, `tools`
- `core/pkg/planner` - 任务规划器 → `llm`
- `core/pkg/memory` - 对话记忆系统 → `llm`
- `core/pkg/permissions` - 权限管理器 → 无内部依赖
- `core/pkg/orchestrator` - 多 Agent 编排器 → `workspace`, `shared`, `session`
- `core/pkg/workspace` - 工作空间隔离器 → 无内部依赖
- `core/pkg/factory` - 工厂函数 → `engine`, `llm`, `permissions`, `prompt`, `tools`, `commands`, `shared`, `session`
- `core/pkg/intent` - 意图识别服务 → `commands`, `shared`
- `core/pkg/llm` - LLM 客户端 (Ollama, OpenAI, Anthropic)
- `core/pkg/prompt` - Prompt 构建器 → `shared`
- `core/pkg/skilltool` - 技能匹配和注入（渐进式披露，LLM 语义匹配 + 关键词回退） → `skill`
- `core/pkg/prompt` - Prompt 构建器 → `shared`
- `core/pkg/skilltool` - 技能匹配和注入（渐进式披露） → `skill`

### 目录结构

```
aura/
├── cmd/aura/             # 应用入口 (main.go)
├── shared/               # 基础层：配置 (Viper)、日志 (Zerolog)、错误处理、国际化 (i18n)
│   ├── pkg/config/         # 配置加载 (Viper)
│   ├── pkg/logger/         # 日志系统 (Zerolog)
│   ├── pkg/i18n/           # 国际化支持
│   ├── pkg/events/         # 统一事件系统（Event 接口、事件总线）
│   ├── pkg/hooks/          # Hooks 框架（事件驱动子进程集成）
│   ├── pkg/tasks/          # 任务追踪管理
│   ├── pkg/manager/        # TypedManager 通用 CRUD 框架
│   └── pkg/memory/         # 共享内存接口
├── storage/              # 基础层：消息数据结构 (Message)、JSONL 存储实现
│   └── pkg/taskstore/      # 任务持久化存储
├── tools/                # 基础层：工具实现 (文件、Shell、SSH、LSP、Web、计算器等)
│   └── pkg/tasktool/       # 任务追踪工具
│   └── pkg/location/       # IP 地理位置（支持缓存）
├── knowledge/            # 基础层：RAG 系统 (Chroma 向量库、embedding、检索、DynamicRAG)
├── personality/          # 基础层：用户档案、响应风格、自适应学习
├── habit/                # 基础层：用户习惯学习（操作追踪、模式分析、习惯管理）
│   ├── pkg/model/          # 数据模型（Habit, Action, Preference）
│   ├── pkg/storage/        # JSONL 存储（按用户目录隔离）
│   ├── pkg/tracker/        # 操作追踪器
│   ├── pkg/analyzer/       # 习惯分析器（工具频率、输出风格、工作流模式）
│   └── pkg/manager/        # 统一管理器
├── skill/                # 基础层：Skill 系统（Markdown Prompt 模板加载和构建）
│   ├── pkg/skill/          # Skill 数据结构
│   ├── pkg/loader/         # 从目录加载 SKILL.md 文件
│   └── pkg/manager/        # Skill CRUD 操作管理
├── agent/                # 基础层：Agent 元数据、配置加载器、提示构建器
│   ├── pkg/agent/          # AgentMeta 定义（YAML frontmatter，含权限继承）
│   ├── pkg/loader/         # 从目录加载 AGENT.md 文件
│   ├── pkg/builder/        # 构建系统提示和委托提示
│   └── pkg/manager/        # Agent CRUD 操作管理
├── mcp/                  # 基础层：MCP 服务器集成（外部工具扩展）
│   ├── pkg/config/         # MCP 服务器配置
│   ├── pkg/constants/      # 常量定义
│   ├── pkg/loader/         # 配置文件加载
│   ├── pkg/client/         # MCP 客户端（stdio + HTTP/SSE）
│   ├── pkg/tool/           # MCPTool 适配器（实现 tools.Tool）
│   └── pkg/manager/        # MCP 服务器生命周期管理（并行启动）
├── session/              # 服务层：会话管理、订阅路由、定时调度
├── commands/             # 服务层：内部命令执行器 (UI 无关，可复用)
│   ├── pkg/executor.go     # 命令执行器
│   ├── pkg/intent/         # 自然语言意图识别
│   ├── pkg/alias/          # 命令别名系统
│   ├── pkg/agent_handler.go # Agent 命令处理
│   └── pkg/skill_handler.go # Skill 命令处理
├── core/                 # 核心层：Engine、ReAct 引擎、LLM 客户端、SDK、Memory、Planner、Permissions、Orchestrator
│   ├── pkg/engine/         # Engine 核心 (ReAct, planning, tool execution)
│   │   ├── engine.go       # Engine 结构体、配置（含 hookEngine、taskList、rollbackMgr、PlanModeState）
│   │   ├── react_loop.go   # ReAct 循环
│   │   ├── planning.go     # 显式规划
│   │   ├── tool_executor.go # 工具执行（含 Phase 枚举）
│   │   └── action_parser.go # Action 解析（支持多行格式）
│   ├── pkg/runtime/        # 统一运行时（拆分为多文件）
│   │   ├── runtime.go      # AgentRuntime 结构体
│   │   ├── boot.go         # 9 阶段初始化
│   │   ├── processing.go   # 消息处理流程
│   │   ├── eventing.go     # 事件处理
│   │   ├── confirmation.go # 确认处理
│   │   ├── delegate.go     # Agent 委托功能（含权限继承）
│   │   ├── components.go   # Runtime 组件定义（SharedResources、SkillSystem 等）
│   │   └── events.go       # 事件定义
│   ├── pkg/intent/         # Core 层意图识别服务
│   ├── pkg/sdk/            # SDK 导出层
│   ├── pkg/planner/        # 任务规划器
│   ├── pkg/memory/         # 对话记忆系统（令牌感知、摘要生成、MaybeCompact）
│   ├── pkg/permissions/    # 权限管理器（含 CloneWithDowngrade）
│   ├── pkg/orchestrator/   # 多 Agent 编排器
│   ├── pkg/workspace/      # 工作空间隔离器
│   ├── pkg/factory/        # 工厂函数
│   ├── pkg/llm/            # LLM 客户端 (Ollama、OpenAI、Anthropic，含 ThinkingConfig、PromptCacheConfig)
│   ├── pkg/prompt/         # Prompt 构建器（含 PromptCacheManager 5 层缓存）
│   ├── pkg/rollback/       # Git stash 回滚管理器
│   └── pkg/skilltool/      # 技能匹配和注入（渐进式披露，LLM 语义匹配 + 关键词回退）
├── api/                  # 应用层：REST API 服务器、Webhooks、Web UI
│   ├── pkg/server/         # HTTP 服务器
│   │   ├── server.go       # 主服务器逻辑
│   │   └── sse.go          # SSE (Server-Sent Events) 支持
│   └── pkg/handlers/       # HTTP 处理器
├── adapters/             # 应用层：外部平台适配器 (Feishu 飞书等)
└── cli/                  # 应用层：CLI (Cobra) 和 TUI (Bubbletea)
    ├── pkg/cli/            # CLI 命令定义（Cobra）
    │   ├── root.go         # 根命令
    │   └── run_agent.go    # Agent 运行逻辑分解
    ├── pkg/common/         # CLI 通用依赖（接口定义、默认实现）
    └── pkg/tui/            # TUI 全屏界面（Bubbletea v2，MVU 模式）
        ├── model.go        # 数据模型
        ├── state.go        # 状态管理
        ├── update.go       # 更新逻辑
        ├── view.go         # 视图渲染（Viewport + 固定底部栏）
        ├── status.go       # 状态栏组件（提取为 Widget）
        ├── commands.go     # TUI 命令处理
        ├── events.go       # 事件处理
        ├── thinking.go     # 思考状态组件（Braille 波浪动画）
        ├── processing.go   # 处理状态组件（工具执行阶段旋转动画，支持多工具追踪）
        ├── tasks.go        # 任务列表组件
        ├── keymap.go       # 全局键盘快捷键注册（Ctrl+{l,h,s,q,t,p,m,k,d,e}）
        ├── command_popup.go # 命令补全弹窗（/ 前缀触发，Tab 自动补全）
        ├── overlay.go      # 弹窗居中渲染（带边框）
        ├── renderer.go     # 工具块渲染样式（IN/OUT 结构化块）
        └── input.go        # 输入管理器（动态高度、Shift+Enter 换行）
├── configs/              # 配置模板
├── docs/                 # 文档
└── go.work               # Go workspace 定义
```

### 模块导入路径

| 模块 | 导入路径 | 依赖 |
|------|----------|------|
| `shared` | `github.com/oneliang/aura/shared` | - |
| `storage` | `github.com/oneliang/aura/storage` | - |
| `tools` | `github.com/oneliang/aura/tools` | - |
| `knowledge` | `github.com/oneliang/aura/knowledge` | - |
| `personality` | `github.com/oneliang/aura/personality` | - |
| `skill` | `github.com/oneliang/aura/skill` | - |
| `agent` | `github.com/oneliang/aura/agent` | `shared` |
| `habit` | `github.com/oneliang/aura/habit` | `shared`, `storage` |
| `mcp` | `github.com/oneliang/aura/mcp` | `shared`, `tools` |
| `session` | `github.com/oneliang/aura/session` | `shared`, `storage` |
| `commands` | `github.com/oneliang/aura/commands` | `agent`, `knowledge`, `personality`, `session`, `shared`, `skill`, `storage` |
| `core` | `github.com/oneliang/aura/core` | `agent`, `commands`, `mcp`, `session`, `shared`, `skill`, `storage`, `tools`, `knowledge`, `personality` |
| `api` | `github.com/oneliang/aura/api` | `commands`, `core`, `session`, `shared`, `skill` |
| `adapters` | `github.com/oneliang/aura/adapters` | `commands`, `core`, `session`, `shared`, `skill` |
| `cli` | `github.com/oneliang/aura/cli` | `api`, `commands`, `core`, `knowledge`, `mcp`, `personality`, `session`, `shared`, `skill`, `tools` |

**core 模块子包：**

| 子包 | 导入路径 | 职责 |
|------|----------|------|
| `core/pkg/engine` | `github.com/oneliang/aura/core/pkg/engine` | Engine 核心 (ReAct 循环、显式规划、工具执行、简单聊天) |
| `core/pkg/runtime` | `github.com/oneliang/aura/core/pkg/runtime` | 统一运行时 (AgentRuntime、事件流、Agent 委托) |
| `core/pkg/sdk` | `github.com/oneliang/aura/core/pkg/sdk` | SDK 导出层 (Runtime 工厂、事件类型、选项、SessionManager、Skill 加载、Orchestrator 工具、类型别名) |
| `core/pkg/planner` | `github.com/oneliang/aura/core/pkg/planner` | 任务规划器 (LLM 驱动的计划创建和步骤管理) |
| `core/pkg/memory` | `github.com/oneliang/aura/core/pkg/memory` | 对话记忆系统（令牌感知、摘要生成、Summarizer） |
| `core/pkg/permissions` | `github.com/oneliang/aura/core/pkg/permissions` | 权限管理器 (工具权限、命令检查、SSH 检查、可信目录) |
| `core/pkg/orchestrator` | `github.com/oneliang/aura/core/pkg/orchestrator` | 多 Agent 编排器 (SubAgent 管理、协作文档、健康监督) |
| `core/pkg/workspace` | `github.com/oneliang/aura/core/pkg/workspace` | 工作空间隔离器 (为子 Agent 提供独立环境) |
| `core/pkg/factory` | `github.com/oneliang/aura/core/pkg/factory` | 工厂函数 (EngineFactory、ToolRegistry) |
| `core/pkg/llm` | `github.com/oneliang/aura/core/pkg/llm` | LLM 客户端 (Ollama、OpenAI、Anthropic，含 ThinkingConfig、PromptCacheConfig) |
| `core/pkg/prompt` | `github.com/oneliang/aura/core/pkg/prompt` | Prompt 构建器 (系统提示、角色加载、PromptCacheManager 5 层缓存) |
| `core/pkg/intent` | `github.com/oneliang/aura/core/pkg/intent` | 意图识别服务 (自然语言命令识别) |
| `core/pkg/rollback` | `github.com/oneliang/aura/core/pkg/rollback` | Git stash 回滚管理器 (Plan Mode 快照和恢复) |
| `core/pkg/skilltool` | `github.com/oneliang/aura/core/pkg/skilltool` | 技能匹配和注入（渐进式披露，LLM 语义匹配 + 关键词回退） |

### 核心接口

**Engine** (`core/pkg/engine/engine.go`):
```go
type Engine struct {
    client          llm.Client
    memory          memory.Memory
    regTools        map[string]tools.Tool
    config          EngineConfig
    planner         *planner.Planner
    currentPlan     *planner.Plan
}

// EngineConfig 配置
type EngineConfig struct {
    SystemPrompt        string
    Tools               []tools.Tool
    Commands            commands.Command
    ConfirmationHandler ToolConfirmationHandler
    PlanningMode        PlanningMode  // implicit/explicit/auto
    PlannerClient       llm.Client
    EnableSummarization bool
    Summarizer          *memory.Summarizer
    EnableDynamicRAG    bool
    DynamicRAG          *retrieval.DynamicRAG
    MaxSteps            int           // ReAct 循环最大迭代次数（0 = 无限制）
    MaxParallelTools    int           // 并行工具执行最大并发数（0 = 默认 5，1 = 串行）
}

// Planning modes
type PlanningMode string
const (
    ModeImplicit PlanningMode = "implicit"  // LLM 在 ReAct 中隐式规划
    ModeExplicit PlanningMode = "explicit"  // 先创建显式计划，再逐步执行
    ModeAuto     PlanningMode = "auto"      // 根据任务复杂度自动选择
)
```

**Message** (`shared/pkg/memory/memory.go`):

Message 采用单一 ContentBlocks 字段架构，支持结构化内容：

```go
type Message struct {
    Role          string         `json:"role"`            // system, user, assistant
    ContentBlocks []ContentBlock `json:"content_blocks"`  // 结构化内容块（多态）
    Type          MessageType    `json:"type,omitempty"`  // 消息类型（可选）
}

// ContentBlock 多态类型（需自定义 JSON 序列化）
type TextBlock struct { Type string; Text string }
type ThinkingBlock struct { Type string; Thinking string }
type ToolUseBlock struct { Type string; ID string; Name string; Input map[string]any }
type ToolResultBlock struct { Type string; ToolUseID string; Content string }
```

**AgentMeta** (`agent/pkg/agent/agent.go`):
```go
type AgentMeta struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    LLMModel    string `yaml:"llm_model,omitempty"`
    DisableTools []string `yaml:"disable_tools,omitempty"`
    config.AgentConfig `yaml:",inline"`  // 继承：PlanningMode, Temperature, SummaryTemp
}

// AgentConfig (shared/pkg/config/config.go)
type AgentConfig struct {
    PlanningMode string  `mapstructure:"planning_mode" yaml:"planning_mode"`
    Temperature  float64 `mapstructure:"temperature" yaml:"temperature"`
    SummaryTemp  float64 `mapstructure:"summary_temp" yaml:"summary_temp"`
}
```

**AgentRuntime** (`core/pkg/runtime/runtime.go`):
```go
type AgentRuntime struct {
    // 统一管理 LLM、工具、记忆、会话、Agent 委托
}

func (rt *AgentRuntime) Initialize(ctx context.Context) error
func (rt *AgentRuntime) Process(ctx context.Context, input string) (<-chan Event, error)
func (rt *AgentRuntime) Shutdown()
func (rt *AgentRuntime) AddTool(tool tools.Tool) error
func (rt *AgentRuntime) SetEventHandler(handler EventHandler)
func (rt *AgentRuntime) SetConfirmationHandler(handler ConfirmationHandler)
func (rt *AgentRuntime) SetAgentDelegateFn(fn func(ctx context.Context, agentName string, task string) (string, error))
func (rt *AgentRuntime) GetAgentDelegateFn() func(ctx context.Context, agentName string, task string) (string, error)
func (rt *AgentRuntime) GetToolNames() []string
func (rt *AgentRuntime) GetAgent() *enginepkg.Engine
func (rt *AgentRuntime) GetMemory() *memory.SessionMemory
func (rt *AgentRuntime) GetSummarizer() *memory.Summarizer

// 多用户隔离
func (rt *AgentRuntime) GetUserID() string
func (rt *AgentRuntime) IsSkillsEnabled() bool
func (rt *AgentRuntime) GetSkillDirectories() []string

// 子 Agent 运行时（轻量级，共享父运行时资源）
func NewSubAgentRuntime(parent *AgentRuntime, subCfg *RuntimeConfig, disabledTools []string, delegationLogger *logger.DelegationFileLogger) (*AgentRuntime, error)
```

**Orchestrator** (`core/pkg/orchestrator/orchestrator.go`):
```go
type Orchestrator struct {
    config      *config.OrchestratorConfig
    workspace   *workspace.Isolator
    docStore    *DocStore
    coordinator *DocCoordinator
    supervisor  *Supervisor
    registry    *TaskRegistry
    subAgents   map[string]*SubAgent
}
```

**AgentManager** (`agent/pkg/manager/manager.go`):

Agent 实现 `sharedmanager.Item` 接口方法（GetName/GetDescription/GetFilePath/GetContent/GetBody），委托给通用管理器：
```go
// AgentManager 委托给 sharedmanager.Manager
type AgentManager struct {
    manager *sharedmanager.Manager
}

// CRUD 操作（全部委托）
func (m *AgentManager) Create(ctx context.Context, req *CreateAgentRequest) (*agent.Agent, error)
func (m *AgentManager) Update(ctx context.Context, name string, req *UpdateAgentRequest) (*agent.Agent, error)
func (m *AgentManager) Delete(ctx context.Context, name string) error
func (m *AgentManager) Get(name string) *agent.Agent
func (m *AgentManager) List() []agent.Agent
func (m *AgentManager) Reload(ctx context.Context) error
```

Loader 提供 `GetItems() []sharedmanager.Item` 支持列表操作。

**SkillManager** (`skill/pkg/manager/manager.go`):

Skill 同样实现 `sharedmanager.Item` 接口，委托给通用管理器：
```go
// SkillManager 委托给 sharedmanager.Manager
type SkillManager struct {
    manager *sharedmanager.Manager
}

// CRUD 操作（全部委托）
func (m *SkillManager) Create(ctx context.Context, req *CreateSkillRequest) (*skill.Skill, error)
func (m *SkillManager) Update(ctx context.Context, name string, req *UpdateSkillRequest) (*skill.Skill, error)
func (m *SkillManager) Delete(ctx context.Context, name string) error
func (m *SkillManager) Get(name string) *skill.Skill
func (m *SkillManager) List() []skill.Skill
func (m *SkillManager) Reload(ctx context.Context) error
```

Loader 提供 `GetItems() []sharedmanager.Item` 支持列表操作。

**IntentRecognizer** (`commands/pkg/intent/recognizer.go`):
```go
type Recognizer struct {
    aliasManager   *alias.Manager
    commands       []commands.CommandInfo
    keywordLoader  *KeywordLoader
}

type Result struct {
    Matched    bool           // 是否匹配到命令
    Command    string         // 命令名称
    Params     map[string]any // 提取的参数
    Confidence float64        // 置信度
    Source     string         // 匹配来源：alias/command/keyword
}

func (r *Recognizer) Recognize(ctx context.Context, input string) (*Result, error)
```

**AliasManager** (`commands/pkg/alias/alias.go`):
```go
type Manager struct {
    aliases map[string]string // alias -> command
}

// 内置别名示例：
// "创建会话" -> session_create
// "列出技能" -> skill_list
// "显示配置" -> config_show

func (m *Manager) Resolve(aliasStr string) (string, bool)
func (m *Manager) Register(aliasStr, command string) error
```

**Event Bus** (`shared/pkg/events/bus.go`):
```go
// Bus 提供事件发布/订阅功能
type Bus struct {
    subscribers map[EventType][]chan Event
    cmdHandlers map[string]CommandHandler
}

func (b *Bus) Subscribe(typ EventType) <-chan Event
func (b *Bus) Publish(event Event)
func (b *Bus) RegisterCommandHandler(commandType string, handler CommandHandler)
func (b *Bus) ExecuteCommand(ctx context.Context, req *CommandRequest) CommandResponse
```

**IntentService** (`core/pkg/intent/service.go`):
```go
// Service 提供 Core 层意图识别服务
type Service struct {
    recognizer     *intentpkg.Recognizer
    commandProvider commands.Command
    enabled        bool
}

func (s *Service) Recognize(ctx context.Context, input string) (*IntentResult, error)
func (s *Service) ExecuteCommand(ctx context.Context, result *IntentResult) (string, error)
func (s *Service) SetEnabled(enabled bool)
func (s *Service) IsEnabled() bool
```

**IntentResult** (`core/pkg/intent/service.go`):
```go
// IntentResult 表示意图识别结果
type IntentResult struct {
    Matched    bool           // 是否匹配到命令
    Command    string         // 命令名称
    Params     map[string]any // 提取的参数
    Confidence float64        // 置信度：0.7-1.0
    Source     string         // 匹配来源：alias/command/keyword
}
```

**ThinkingWidget** (`cli/pkg/tui/thinking.go`):
```go
// ThinkingWidget 管理思考状态指示器，独立控制显示状态
type ThinkingWidget struct {
    state     ThinkingState  // Idle/Active/Done
    startTime time.Time
    endTime   time.Time
    rendered  string
    cleared   bool
    frame     int
}

func (w *ThinkingWidget) Start()                              // 开始思考
func (w *ThinkingWidget) StartAndRender() (string, tea.Cmd)   // 开始并返回 "Thinking..." + tick Cmd
func (w *ThinkingWidget) Update(msg tea.Msg) (string, tea.Cmd) // 处理动画 tick
func (w *ThinkingWidget) Stop() string                        // 完成思考，返回时长
func (w *ThinkingWidget) Complete() string                    // 完成并返回 "Thought for Xs"
func (w *ThinkingWidget) Clear()                              // 静默完成
func (w *ThinkingWidget) IsActive() bool                      // 是否正在思考
func (w *ThinkingWidget) Duration() time.Duration             // 获取思考时长
func (w *ThinkingWidget) Rendered() string                    // 获取当前渲染字符串
func (w *ThinkingWidget) Reset()                              // 重置到空闲状态
```

**SDK SessionManager** (`core/pkg/sdk/session.go`):
```go
// SessionManager 提供用户级会话管理，绑定 userID，调用方无需每次传递
type SessionManager struct {
    wrapper *sessionMgr.SessionServiceWrapper
    userID  string
}

func NewSessionManager(dataDir string, userID string, cfg *config.Config) (*SessionManager, error)
func (m *SessionManager) ListSessions() ([]SessionInfo, error)
func (m *SessionManager) CreateSession(name, role string) (*SessionInfo, error)
func (m *SessionManager) GetSession(id string) (*SessionInfo, error)
func (m *SessionManager) DeleteSession(id string) error
func (m *SessionManager) GetSubscriptions(sessionID string) ([]SubscriptionInfo, error)
func (m *SessionManager) AddSubscription(sessionID, trigger, source string) error
func (m *SessionManager) GetOrCreateSession(namingFormat string) (string, error)
```

**SDK Skill 加载** (`core/pkg/sdk/skills.go`):
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

**SDK Orchestrator** (`core/pkg/sdk/orchestrator.go`):
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
)

func NewOrchestrator(cfg *config.Config) (*Orchestrator, error)
func NewSpawnAgentTool(o *Orchestrator) tools.Tool
func NewCreateDocTool(o *Orchestrator) tools.Tool
func NewProcessDocTool(o *Orchestrator, agentID string) tools.Tool
func NewQueryQueueTool(o *Orchestrator) tools.Tool
```

**SDK MCP 管理** (`core/pkg/sdk/mcp.go`):

MCP 管理函数已提升至 SDK 导出层，外部可直接调用：

```go
func AddMCPServer(ctx context.Context, name, command string, args []string) ([]tools.Tool, error)
func RemoveMCPServer(ctx context.Context, name string) error
func ListMCPServers() []ServerInfo
func GetMCPServerStatus(ctx context.Context, name string) *ServerInfo
func LoadMCPConfig() (*MCPConfig, error)
```

**SDK 运行时类型** (`core/pkg/sdk/sdk.go`):
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

**Agent Delegation** (`core/pkg/runtime/delegate.go`):
```go
// Agent 委托功能
func (r *AgentRuntime) findAgent(agentName string) (*agentpkg.Agent, error)
func createTempWorkspace() (string, error)
func buildSubAgentConfig(parent *AgentRuntime, foundAgent *agentpkg.Agent, task string) *RuntimeConfig
func executeSubAgent(ctx context.Context, subAgentRuntime *AgentRuntime, task string) (string, error)
func (r *AgentRuntime) createAgentDelegateFn(ctx context.Context) func(ctx context.Context, agentName string, task string) (string, error)
```

---

## 设计模式

### 工厂模式
- `LLMFactory` - 根据配置创建 LLM 客户端
- `EngineFactory` - 创建 Engine 实例
- `ToolRegistry` - 工具注册和查找

### 事件流模式
所有模式（CLI/TUI/API）使用统一的事件流接口处理输出

**事件类型** (`shared/pkg/events/events.go`):
```go
// Engine 执行事件
EventTypeThinkingStart, EventTypeThinkingEnd       // 思考开始/结束（配对事件）
EventTypeAction, EventTypeResult                   // 动作决策/执行结果
EventTypeResponse, EventTypeResponseChunk          // 完整响应/流式响应块
EventTypeError                                     // 错误事件
EventTypeStep                                      // 步骤进度
EventTypeToolStart, EventTypeToolEnd               // 工具开始/结束

// 规划事件
EventTypePlanCreated, EventTypePlanStep, EventTypePlanComplete

// 运行时生命周期事件
EventTypeDone, EventTypeConfirmationRequest

// 内存事件
EventTypeMemoryClearRequest, EventTypeMemoryCleared
EventTypeMemoryStatsRequest, EventTypeMemoryStats
EventTypeMemoryCompacted

// 会话事件
EventTypeSessionCreated, EventTypeSessionSwitched, EventTypeSessionDeleted

// 任务事件
EventTypeTaskCreate, EventTypeTaskUpdate, EventTypeTaskList  // 任务创建/更新/列表

// 命令事件（事件驱动通信）
EventTypeCommandRequest, EventTypeCommandResponse
EventTypeCommandMatched, EventTypeCommandResult    // 意图识别匹配/执行结果

// 已废弃（不再使用，保留向后兼容）
// EventTypeReflection, EventTypeSelfCorrection, EventTypeLoopDetected, EventTypeMaxStepsExceeded
```

**Event Bus 统计**：
```go
func (b *Bus) Stats() map[EventType]int  // 返回各类型丢弃事件计数
```

### 依赖注入
通过 `RuntimeOption` 模式注入配置（事件处理器、确认处理器、会话存储等）

### 接口驱动
所有核心组件由接口定义，支持多种实现

### 渐进式披露（Progressive Disclosure）
技能系统不将所有技能内容都包含在每次系统提示中，而是通过 `SkillMatcher` 分析用户输入匹配相关技能（优先使用 LLM 语义匹配，关键词回退），再通过 `SkillInjector` 动态注入到系统提示，减少提示大小并提高相关性。

### 并行工具执行
ReAct 循环支持单步内并行执行多个工具。LLM 可在单次响应中输出多行 `Action:` 表示独立工具调用：
- `parseActions()` 提取所有 Action 行
- `executeToolsParallel()` 使用信号量限制并发数（默认 5，通过 `MaxParallelTools` 配置）
- 所有观察结果聚合成一条记忆条目
- 单个工具时自动回退到串行执行

### 共享管理器框架
AgentManager 和 SkillManager 已统一委托给 `shared/pkg/manager` 中的通用 Item Manager：
- `Item` 接口（GetName/GetDescription/GetFilePath/GetContent/GetBody）由 Agent 和 Skill 实现
- `ManagerConfig` 通过回调注入类型特定逻辑（Validate/BuildContent/ConstructItem/MergeUpdate/FindByName）
- Loader 提供 `GetItems() []sharedmanager.Item` 支持列表操作
- 消除了 Agent 和 Skill 管理器之间 ~200 行重复 CRUD 代码

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

## 更多信息

- [架构和依赖](docs/architecture.md)
- [Agent 测试](docs/agent-testing.md)
- [Skill 系统](skill/README.md)
