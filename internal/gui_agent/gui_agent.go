package guiagent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// GUIAgent 是一个用于 GUI 操作的 Agent
type GUIAgent struct {
	llm    llminterface.ChatAdapter
	logger *slog.Logger
}

// NewGUIAgent 创建一个新的GUIAgent实例
func NewGUIAgent(ctx context.Context, llm llminterface.ChatAdapter) (*GUIAgent, error) {
	logger := slog.Default().WithGroup("gui_agent")

	return &GUIAgent{
		llm:    llm,
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
