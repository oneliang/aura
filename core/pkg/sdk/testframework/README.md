# 统一自动化测试框架

## 概述

本测试框架实现了**一套测试覆盖 CLI、TUI、Server 三种模式**的自动化测试方案。

## 架构设计

```
核心思想：以 sdk.Event 为唯一接口，通过录制 - 回放机制验证三种消费者

              同一份核心逻辑 (runtime/engine)
                        │
                        ▼
              sdk.Event 事件流 (channel)
        ┌───────────────┼───────────────┐
        │               │               │
   消费者 A         消费者 B        消费者 C
   CLI: fmt       TUI: Adapter    Server: SSE
```

## 模块说明

### 1. testrecorder (`modules/core/pkg/sdk/testrecorder/`)

**用途**：录制和回放事件流

**核心类型**：
- `EventRecorder` - 录制器，从 channel 录制事件
- `TestEvent` - 简化的事件结构用于测试
- `RecordedEvent` - 可序列化的事件记录

**关键方法**：
```go
// 从 channel 录制事件
recorder := testrecorder.NewEventRecorder("name", "desc", "input")
recorder.RecordFromChannel(eventCh)

// 保存到文件
recorder.SaveToFile("testdata/events.json")

// 从文件加载
recorder, _ := testrecorder.LoadFromFile("testdata/events.json")

// 转换为测试用事件
testEvents := recorder.ToTestEvents()

// 回放为 SDK channel
sdkCh := recorder.ToSDKChannel()
```

**工具函数**：
- `AssertEventOrder(events, expectedTypes)` - 验证事件顺序
- `FilterByType(events, typ)` - 按类型过滤事件
- `CountByType(events, typ)` - 统计事件数量

### 2. testconsumers (`modules/core/pkg/sdk/testconsumers/`)

**用途**：模拟三种消费者行为

**核心类型**：
- `CLIConsumer` - 模拟 CLI 控制台输出
- `TUIConsumer` - 模拟 TUI 事件处理
- `TUIAdapter` - TUI 事件转换器
- `ServerConsumer` - 模拟 Server SSE 输出

**关键方法**：
```go
// 单独使用
cli := testconsumers.NewCLIConsumer()
output := cli.ProcessAll(events)

tui := testconsumers.NewTUIConsumer()
tui.ProcessAll(events)
events := tui.GetEvents()

server := testconsumers.NewServerConsumer()
sse := server.ProcessAll(events)

// 一次性测试所有消费者
result, _ := testconsumers.TestAllConsumers(events)
```

### 3. 事件流测试 (`modules/core/pkg/sdk/event_flow_test.go`)

**用途**：统一验证三种模式下事件处理的正确性

**测试场景**：
- `TestEventFlow_SimpleChat` - 简单对话
- `TestEventFlow_ToolExecution` - 工具执行
- `TestEventFlow_ErrorHandling` - 错误处理
- `TestEventFlow_CommandMatched` - 命令匹配
- `TestEventFlow_MultipleSteps` - 多步骤 ReAct
- `TestEventFlow_EmptyStream` - 空事件流
- `TestEventFlow_PartialStream` - 部分事件流
- `TestEventFlow_ExtraData` - 额外数据

### 4. TUI Adapter 测试 (`modules/cli/pkg/tui/integration_test.go`)

**用途**：验证真实 TUI Adapter 的事件转换逻辑

**测试覆盖**：
- 所有事件类型的转换
- 额外数据保留
- 未知事件类型处理
- RequestID 传递

## 使用示例

### 示例 1: 测试新事件类型的处理

```go
func TestEventFlow_NewEventType(t *testing.T) {
    // 1. 定义包含新事件的测试数据
    events := []testrecorder.TestEvent{
        {Type: events.EventTypeThinkingStart},
        {Type: events.EventTypeNewFeature},  // 新事件
        {Type: events.EventTypeResponse, Content: "Result"},
    }
    
    // 2. 运行统一测试
    result, err := testconsumers.TestAllConsumers(events)
    if err != nil {
        t.Fatal(err)
    }
    
    // 3. 验证各消费者正确处理
    assert.Contains(t, result.CLI, "Result")
    assert.True(t, result.TUI.HasEventType(events.EventTypeNewFeature))
}
```

### 示例 2: 验证事件顺序

```go
func TestEventOrder(t *testing.T) {
    events := []testrecorder.TestEvent{
        {Type: events.EventTypeThinkingStart},
        {Type: events.EventTypeToolStart},
        {Type: events.EventTypeToolEnd},
        {Type: events.EventTypeResponse},
        {Type: events.EventTypeDone},
    }
    
    expected := []events.EventType{
        events.EventTypeThinkingStart,
        events.EventTypeToolStart,
        events.EventTypeToolEnd,
        events.EventTypeResponse,
        events.EventTypeDone,
    }
    
    err := testrecorder.AssertEventOrder(events, expected)
    if err != nil {
        t.Errorf("Event order validation failed: %v", err)
    }
}
```

### 示例 3: 录制真实事件用于回归测试

```go
// 手动运行录制脚本
func RecordRealEvents(t *testing.T) {
    // 创建真实 runtime
    rt, _ := sdk.NewRuntime(cfg)
    
    // 创建录制器
    recorder := testrecorder.NewEventRecorder(
        "real_scenario",
        "Real user interaction",
        "请帮我分析这个文件",
    )
    
    // 录制
    events, _ := rt.Process(ctx, "请帮我分析这个文件")
    recorder.RecordFromChannel(events)
    
    // 保存
    recorder.SaveToFile("testdata/real_scenario.json")
}
```

## 运行测试

```bash
# 运行所有新测试
go test ./modules/core/pkg/sdk/... ./modules/cli/pkg/tui/... -count=1

# 运行特定测试
go test -run TestEventFlow ./modules/core/pkg/sdk/
go test -run TestAdapter ./modules/cli/pkg/tui/

# 生成覆盖率报告
go test -coverprofile=cover.out ./modules/core/pkg/sdk/...
go tool cover -html=cover.out
```

## 测试覆盖分析

| 测试层 | 覆盖率 | 说明 |
|--------|--------|------|
| 核心层单元测试 | 核心逻辑 | `runtime_test.go`, `engine_test.go` |
| 录制器测试 | 事件录制/回放 | `recorder_test.go` |
| 消费者模拟器测试 | 三种消费者 | `consumers_test.go` |
| 事件流集成测试 | 端到端流程 | `event_flow_test.go` |
| TUI Adapter 测试 | 真实转换逻辑 | `integration_test.go` |

## 优势

1. **一套测试覆盖三层**：测一次 = 测 CLI + TUI + Server
2. **快速回归**：无需启动完整应用，纯单元测试速度
3. **无需 LLM**：使用录制的事件，不依赖外部服务
4. **定位问题快**：哪个消费者失败一目了然
5. **CI 友好**：完全自动化

## 局限

以下场景仍需人工验证：
- TUI 视觉渲染效果
- 键盘导航体验
- 不同终端兼容性
- 中文显示和长文本换行

## 扩展

### 添加新测试场景

1. 在 `event_flow_test.go` 添加新测试函数
2. 定义事件序列
3. 使用 `TestAllConsumers` 验证
4. 添加针对性的断言

### 添加新的消费者类型

1. 在 `testconsumers/` 创建新的 Consumer 类型
2. 实现 `Process(event) string` 方法
3. 在 `TestAllConsumers` 中集成

## 文件结构

```
modules/core/pkg/sdk/
├── testrecorder/
│   ├── recorder.go          # 录制器实现
│   └── recorder_test.go     # 录制器测试
├── testconsumers/
│   ├── consumers.go         # 消费者模拟器
│   └── consumers_test.go    # 消费者测试
├── event_flow_test.go       # 事件流集成测试
└── sdk.go                   # SDK 定义

modules/cli/pkg/tui/
├── integration.go           # Adapter 实现
└── integration_test.go      # Adapter 测试
```

## 最佳实践

1. **优先使用录制的事件**：避免依赖真实 LLM
2. **测试边界情况**：空流、部分流、错误事件
3. **验证事件顺序**：使用 `AssertEventOrder`
4. **保留 RequestID**：验证事件分组追踪
5. **定期更新录制数据**：反映真实使用场景
