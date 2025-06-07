package eino

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/cloudwego/eino/components/model"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	jsonSchema "github.com/m4n5ter/another-me/pkg/schema"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// NoToolChatAdapter 是 eino 的不发送工具信息的适配器实现
// 适用于不支持工具调用的模型，如 deepseek-reasoner
type NoToolChatAdapter struct {
	// einoModel 持有原始的无工具模型
	einoModel model.BaseChatModel
}

// NewNoToolChatAdapter 创建一个新的不使用工具调用的eino适配器实例
func NewNoToolChatAdapter(ctx context.Context, initialModel model.BaseChatModel) (*NoToolChatAdapter, error) {
	adapter := &NoToolChatAdapter{
		einoModel: initialModel,
	}
	return adapter, nil
}

var _ llminterface.ChatAdapter = (*NoToolChatAdapter)(nil)

// GetFrameworkName 返回此适配器实例所适配的底层框架的名称
func (a *NoToolChatAdapter) GetFrameworkName() string {
	return "eino-no-tool"
}

// ProduceJSON 方法用于生成 JSON 格式的响应。
func (a *NoToolChatAdapter) ProduceJSON(ctx context.Context, input llminterface.ChatInput, jsonSchema Option[jsonSchema.Schema]) (string, error) {
	panic("ProduceJSON is not implemented for eino-no-tool adapter")
}

// Chat 方法用于向不支持工具调用的LLM发起一次对话交互
func (a *NoToolChatAdapter) Chat(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error) {
	// 将输入转换为eino消息格式，但不包含工具信息
	einoMsgs := ChatInputToEinoMsgs(input)

	// 使用模型进行流式对话
	streamReader, err := a.einoModel.Stream(ctx, einoMsgs)
	if err != nil {
		return nil, fmt.Errorf("eino model stream error: %w", err)
	}

	outputChan := make(chan llminterface.ChatOutputChunk)

	go func() {
		defer close(outputChan)

		finishReason := None[string]()
		for {
			msg, err := streamReader.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					return
				}
				outputChan <- llminterface.ChatOutputChunk{Error: err}
				return
			}

			if msg.ResponseMeta != nil && msg.ResponseMeta.FinishReason != "" {
				finishReason = Some(msg.ResponseMeta.FinishReason)
			}

			chunk := llminterface.ChatOutputChunk{}

			// 处理Eino消息
			parts, reasoningContent := ProcessEinoMessageToContentParts(msg)
			if reasoningContent != "" {
				chunk.Reasoning = Some(reasoningContent)
			}

			// 只有当有实际内容时才发送 chunk
			if len(parts) > 0 || chunk.Reasoning.IsSome() {
				chunk.ContentParts = parts
				chunk.FinishReason = finishReason
				outputChan <- chunk
			}
		}
	}()

	return outputChan, nil
}

// RegisterTools 实现接口，但不执行任何操作
// 因为这个适配器专门用于不支持工具调用的模型
func (a *NoToolChatAdapter) RegisterTools(ctx context.Context, registry *toolcore.Registry) error {
	slog.Info("NoToolChatAdapter: ignoring tool registration as this adapter is for models without tool support")
	return nil
}
