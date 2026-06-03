// Package app provides application-level orchestration for Aura.
package app

import (
	"context"
	"fmt"

	"github.com/oneliang/aura/cli/pkg/tui"
	"github.com/oneliang/aura/core/pkg/sdk"
	"github.com/oneliang/aura/shared/pkg/events"
)

// Orchestrator 整合Agent和UI的事件流
// Agent和UI都是黑盒，只知道发送和接收事件
// Orchestrator负责转发事件流
type Orchestrator struct {
	agent *sdk.Runtime  // Agent黑盒
	ui    *tui.Model    // UI黑盒

	// 运行状态
	running bool
}

// NewOrchestrator 创建整合器
func NewOrchestrator(agent *sdk.Runtime, ui *tui.Model) *Orchestrator {
	return &Orchestrator{
		agent: agent,
		ui:    ui,
	}
}

// Run 运行整合
// 启动Agent和UI的事件流转发
func (o *Orchestrator) Run(ctx context.Context) error {
	// 1. 启动Agent
	if err := o.agent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	o.running = true

	// 2. 获取Agent事件流（OUT）
	agentEvents := o.agent.Events()

	// 3. 转发Agent.out → UI.in
	go func() {
		for event := range agentEvents {
			o.ui.ReceiveEvent(event)
		}
	}()

	// 4. 转发UI.out → Agent.in
	go func() {
		uiEvents := o.ui.Events()
		for event := range uiEvents {
			if err := o.agent.SendEvent(ctx, event); err != nil {
				// 发送失败，记录错误
				// TODO: 使用logger记录
				fmt.Printf("Orchestrator: failed to send event to agent: %v\n", err)
			}
		}
	}()

	return nil
}

// Stop 停止整合
func (o *Orchestrator) Stop(ctx context.Context) error {
	o.running = false

	// 停止Agent
	if err := o.agent.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop agent: %w", err)
	}

	return nil
}

// IsRunning 检查是否正在运行
func (o *Orchestrator) IsRunning() bool {
	return o.running
}

// SendUserInput 发送用户输入到Agent
// 通过事件流发送EventTypeUserInput
func (o *Orchestrator) SendUserInput(ctx context.Context, input string, requestID string) error {
	if !o.running {
		return fmt.Errorf("orchestrator not running")
	}

	event := events.NewEvent(events.EventTypeUserInput, input, requestID)
	return o.agent.SendEvent(ctx, event)
}

// SendInteractionResponse 发送交互响应到Agent
// 通过事件流发送EventTypeInteractionResponse
func (o *Orchestrator) SendInteractionResponse(ctx context.Context, requestID string, approved bool, interactionType events.InteractionType) error {
	if !o.running {
		return fmt.Errorf("orchestrator not running")
	}

	extra := map[string]any{
		"approved": approved,
		"type":     interactionType,
	}

	event := events.NewEventWithExtra(events.EventTypeInteractionResponse, "", extra, requestID)
	return o.agent.SendEvent(ctx, event)
}