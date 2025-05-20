package guiagent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/gui"
)

// GUIAgent 是一个用于 GUI 操作的 Agent
type GUIAgent struct {
	llm    llminterface.ChatAdapter
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
		llm:    llm,
		react:  guiReactAgent,
		logger: logger,
	}, nil
}

// Execute 执行 GUI 操作
//
// 输入应该是一条 GUI 指令，比如 "移动鼠标到(100, 100)"，一般是较小的指令
func (a *GUIAgent) Execute(ctx context.Context, instruction, imageURL string) (string, error) {
	llmResponse, err := llminterface.ChatAndGetFullResponse(ctx, a.llm, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{Type: llminterface.PartTypeText, Text: instruction},
					{Type: llminterface.PartTypeImageURL, ImageURL: Some(llminterface.ImageURLContent{
						URL: imageURL,
					})},
				},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("GUIAgent: failed to execute: %w", err)
	}

	parsedJSONResult, err := ParseActionOutput(llmResponse.FullText)
	if err != nil {
		return "", fmt.Errorf("GUIAgent: failed to execute: %w", err)
	}

	return parsedJSONResult, nil
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
		// WithToolRegistry(registry).
		WithLogger(logger).
		WithMaxIterations(15). // TODO: 需要一个合适的最大迭代次数
		WithSystemPrompt(i18n.GlobalManager.T(ctx, "assistant.gui.prompt", nil)).
		Build()
	if err != nil {
		return nil, fmt.Errorf("guiLLM: failed to create react agent: %w", err)
	}

	return react, nil
}
