package guiagent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/gui"
)

// GUIAgent 是一个用于 GUI 操作的 Agent
type GUIAgent struct {
	react  *reactagent.Agent
	logger *slog.Logger
}

// NewGUIAgent 创建一个新的GUIAgent实例
func NewGUIAgent(ctx context.Context, llm llminterface.ChatAdapter) (*GUIAgent, error) {
	logger := slog.Default().WithGroup("gui_agent")

	guiReactAgent, err := guiReactAgent(ctx, llm, logger)
	if err != nil {
		return nil, fmt.Errorf("NewGUIAgent: failed to create guiLLM: %w", err)
	}

	return &GUIAgent{
		react:  guiReactAgent,
		logger: logger,
	}, nil
}

// Execute 执行 GUI 操作
//
// 输入应该是一条 GUI 指令，比如 "移动鼠标到(100, 100)"
func (a *GUIAgent) Execute(ctx context.Context, instruction string) (llminterface.LLMResponse, error) {
	response := llminterface.LLMResponse{}

	outputChan, err := a.react.Run(ctx, instruction, uuid.New().String())
	if err != nil {
		return response, fmt.Errorf("GUIAgent: failed to execute: %w", err)
	}

	// TODO
	for chunk := range outputChan {
		switch chunk.Type {
		case reactagent.AgentChunkTypeText:
			// 立即显示增量文本
			fmt.Print(chunk.TextDelta)
		case reactagent.AgentChunkTypeToolStart:
			// 显示工具正在执行的指示
			fmt.Printf("\n[执行工具: %s]\n", chunk.ToolName)
		case reactagent.AgentChunkTypeToolEnd:
			// 显示工具执行完成的指示
			if chunk.Error != "" {
				fmt.Printf("\n[工具执行失败: %s - %s]\n", chunk.ToolName, chunk.Error)
			} else {
				fmt.Printf("\n[工具执行完成: %s]\n", chunk.ToolName)
			}
		case reactagent.AgentChunkTypeError:
			// 显示错误
			fmt.Printf("\n错误: %s\n", chunk.Error)
		case reactagent.AgentChunkTypeFinish, reactagent.AgentChunkTypeMaxIter:
			// 显示结束信息
			if chunk.Error != "" {
				fmt.Printf("\n%s: %s\n", chunk.Type, chunk.Error)
			}
			// 检查是否是最后一个块
			if chunk.IsLast {
				fmt.Println("\n[对话结束]")
			}
		}
	}
	return response, nil
}

func guiReactAgent(ctx context.Context, llm llminterface.ChatAdapter, logger *slog.Logger) (*reactagent.Agent, error) {
	registry := toolcore.NewRegistry()
	for _, tool := range gui.NewGUITools(i18n.GlobalManager) {
		err := registry.Register(ctx, tool)
		if err != nil {
			return nil, fmt.Errorf("guiLLM: failed to register tool: %w", err)
		}
	}

	react, err := reactagent.NewAgentBuilder().
		WithLLMAdapter(llm).
		WithToolRegistry(registry).
		WithLogger(logger).
		WithMaxIterations(15). // TODO: 需要一个合适的最大迭代次数
		WithSystemPrompt(i18n.GlobalManager.T(ctx, "assistant.gui.prompt", nil)).
		Build()
	if err != nil {
		return nil, fmt.Errorf("guiLLM: failed to create react agent: %w", err)
	}

	return react, nil
}
