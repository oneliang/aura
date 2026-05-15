# 交互式输入功能设计

>日期：2026-05-15
> 状态：设计完成，待实现

## 背景

Aura 当前运行时，用户无法在处理过程中输入新消息。用户希望实现类似 Claude Code 的体验：处理过程中可以继续输入，消息进入队列，在合适的时机被 LLM 处理。

## 核心设计决策

| 决策项 | 选择 | 说明 |
|--------|------|------|
| 并发模式 | 队列模式 | 用户输入不中断当前任务 |
| 消息标记 | 特殊标记 | 标记为"补充信息"，LLM 自行整合 |
| 输入限制 | 无限制 | 用户可输入任何内容 |
| 检查时机 | 每次 Think前 | Engine 在调用 LLM 前检查队列 |
| 多条处理 | 全部带上 | 队列消息合并发送，LLM 整合 |
| 取消功能 | Ctrl+C 全局中断 | 中断会话，队列清空 |
| 界面呈现 | 输入框上方缓存区 | 显示待处理消息 |
| 交互方式 | 输入框清空 | 可继续输入下一条 |

## 方案选择

**方案 B：消息队列解耦方案**

引入独立的消息队列组件，解耦 TUI 和 Engine：
- MessageQueue 作为 SessionContext 的组件
- TUI 通过 push/pop 与队列交互
- Engine 通过接口获取队列消息

## 架构设计

```
┌─────────────────────────────────────────────────────┐
│                     TUI Layer                        │
│  ┌─────────────┐    ┌─────────────────────────┐    │
│  │ InputManager│───▶│   PendingMessagesView   │    │
│  │ (不禁用)    │    │   (缓存区渲染)          │    │
│  └─────────────┘    └─────────────────────────┘    │
│         │                      │                    │
│         │ push(userMsg)        │                    │
│         ▼                      │                    │
│  ┌─────────────────────────────────────────┐       │
│  │              MessageQueue                │       │
│  │  - pendingMessages []PendingMessage     │       │
│  │  - push(msg)                            │       │
│  │  - popAll() []PendingMessage            │       │
│  │  - clear()                              │       │
│  └─────────────────────────────────────────┘       │
│                       │                              │
│                       │ popAll()                     │
│                       ▼                              │
├─────────────────────────────────────────────────────┤
│                   Engine Layer                       │
│  ┌─────────────────────────────────────────┐       │
│  │  runReActLoop()                          │       │
│  │    Think前:                               │       │
│  │      pending = queue.popAll()            │       │
│  │      if len(pending) > 0:                │       │
│  │        标记为"补充信息"加入 messages     │       │
│  │      调用 LLM                             │       │
│  └─────────────────────────────────────────┘       │
└─────────────────────────────────────────────────────┘
```

## MessageQueue 组件设计

**文件**：`modules/core/pkg/runtime/message_queue.go`

```go
// PendingMessage 表示待处理的用户消息
type PendingMessage struct {
    Content   string    // 消息内容
    Timestamp time.Time // 入队时间
}

const MaxPendingMessages = 10

// MessageQueue 管理待处理消息队列
type MessageQueue struct {
    mu      sync.Mutex
    pending []PendingMessage
}

func (q *MessageQueue) Push(content string) error
func (q *MessageQueue) PopAll() []PendingMessage
func (q *MessageQueue) Clear()
func (q *MessageQueue) Peek() []PendingMessage  // 供 TUI 渲染
func (q *MessageQueue) Len() int
```

## Engine Think 前检查

**文件**：`modules/core/pkg/engine/react_loop.go`

在每轮 Think（调用 LLM）前检查队列：

```go
func (e *Engine) runReActLoop(ctx context.Context, ...) {
    for {
        // Think 前检查队列
        pending := e.sessionCtx.MessageQueue.PopAll()
        if len(pending) > 0 {
            for _, msg := range pending {
                supplementMsg := Message{
                    Role:    "user",
                    Content: fmt.Sprintf("[补充信息] %s", msg.Content),
                }
                e.messages = append(e.messages, supplementMsg)
            }
        }
        
        // 原有 Think 流程
        response := e.streamAndBufferResponse(ctx, ...)
        // ...
    }
}
```

## TUI 层改动

### InputManager

处理时不禁用输入，改为状态标记：

```go
func (im *InputManager) SetProcessing(processing bool) {
    im.processing = processing  // 仅改变样式，不禁用
}
```

### Update 逻辑

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if msg.Type == tea.KeyEnter && !m.input.Disabled() {
        input := m.input.Value()
        if input != "" {
            if m.isProcessing {
                // 处理中：push 到队列
                m.runtime.SessionCtx.MessageQueue.Push(input)
                m.pendingMessages = append(m.pendingMessages, ...)
                m.input.Clear()
            } else {
                // 空闲：直接发送
                return m, m.sendMessage(input)
            }
        }
    }
}
```

### View 渲染

输入框上方显示缓存区：

```go
func (m Model) View() string {
    pendingView := ""
    if len(m.pendingMessages) > 0 {
        pendingView = renderPendingMessages(m.pendingMessages)
    }
    return pendingView + "\n" + m.input.View() + "\n" + m.messagesView
}
```

### Done 后合并发送

```go
case EventTypeDone:
    m.isProcessing = false
    pending := m.pendingMessages
    if len(pending) > 0 {
        combined := combineMessages(pending)
        m.pendingMessages = nil
        m.runtime.SessionCtx.MessageQueue.Clear()
        return m, m.sendMessage(combined)
    }
```

## CLI 模式改动

**文件**：`modules/cli/pkg/cli/root.go`

引入非阻塞输入：

```go
inputCh := make(chan string, 10)

go func() {
    reader := bufio.NewReader(os.Stdin)
    for {
        line, _ := reader.ReadString('\n')
        inputCh <- strings.TrimSpace(line)
    }
}()

for {
    select {
    case input := <-inputCh:
        if !isProcessing {
            events := runtime.Process(ctx, input)
            isProcessing = true
        } else {
            runtime.SessionCtx.MessageQueue.Push(input)
        }
    default:
        // 检查处理状态
    }
}
```

## 错误处理

| 场景 | 处理方式 |
|------|----------|
| 队列溢出 | 上限 10 条，超出时拒绝并提示 |
| Ctrl+C 中断 | 清空队列，中断处理 |
| LLM 调用失败 | 队列消息保留，下次 Think 重新带上 |

## 文件改动清单

| 文件 | 改动类型 |
|------|----------|
| `modules/core/pkg/runtime/message_queue.go` | 新增 |
| `modules/core/pkg/runtime/message_queue_test.go` | 新增 |
| `modules/core/pkg/runtime/components.go` | 修改 |
| `modules/core/pkg/engine/react_loop.go` | 修改 |
| `modules/cli/pkg/tui/model.go` | 修改 |
| `modules/cli/pkg/tui/input.go` | 修改 |
| `modules/cli/pkg/tui/update.go` | 修改 |
| `modules/cli/pkg/tui/view.go` | 修改 |
| `modules/cli/pkg/tui/events.go` | 修改 |
| `modules/cli/pkg/cli/root.go` | 修改 |

## 实现顺序

1. MessageQueue 组件 + 单元测试
2. SessionContext 集成
3. Engine Think 前检查
4. TUI 输入不禁用 + 队列 push
5. TUI 缓存区渲染
6. TUI Done 后合并发送
7. CLI 模式非阻塞输入
8. 集成测试 + 手动验证

## 验证方式

1. 单元测试：MessageQueue 基础功能
2. 集成测试：Engine Think 前检查队列
3. 手动测试：
   - 启动 TUI，发送消息后立即输入补充信息
   - 观察 LLM 响应是否包含补充信息处理
   - 多条队列消息合并发送验证
   - Ctrl+C 中断验证队列清空