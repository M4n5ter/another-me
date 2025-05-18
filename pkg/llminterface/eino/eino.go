package eino

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// ChatAdapter 是 eino 的 ChatAdapter 实现。
type ChatAdapter struct {
	// einoModel 持有经过工具绑定的 ToolCallingChatModel 实例
	einoModel model.ToolCallingChatModel
}

// NewChatAdapter 创建一个新的 eino ChatAdapter 实例。
// 它会从 toolRegistry 中获取所有工具，将它们转换为 eino 格式，并绑定到 einoModel。
func NewChatAdapter(ctx context.Context, initialModel model.ToolCallingChatModel, toolRegistry *toolcore.Registry, defaultLang string) (*ChatAdapter, error) {
	adapter := &ChatAdapter{
		einoModel: initialModel,
	}

	if toolRegistry != nil {
		tcTools := toolRegistry.GetAll()
		if len(tcTools) > 0 {
			einoToolInfos := make([]*schema.ToolInfo, 0, len(tcTools))
			for _, tcTool := range tcTools {
				tcSchema, err := tcTool.Schema(ctx)
				if err != nil {
					slog.Error("Failed to get schema for toolcore tool during eino adapter init", "toolName", "unknown_yet", "error", err)
					// 根据策略，可以选择跳过此工具或返回错误
					// 这里选择跳过并记录日志
					continue
				}

				einoToolInfo, err := ToolCoreSchemaToEinoToolInfo(&tcSchema, defaultLang)
				if err != nil {
					slog.Error("Failed to convert toolcore schema to eino tool info", "toolName", tcSchema.Name, "error", err)
					continue // 跳过转换失败的工具
				}

				einoToolInfos = append(einoToolInfos, einoToolInfo)
			}

			if len(einoToolInfos) > 0 {
				modelWithTools, err := initialModel.WithTools(einoToolInfos)
				if err != nil {
					return nil, fmt.Errorf("eino adapter: failed to bind tools to model: %w", err)
				}
				adapter.einoModel = modelWithTools // 使用绑定了工具的模型

				slog.Info("eino adapter: tools bound to model successfully", "num_tools", len(einoToolInfos))
			} else {
				slog.Info("eino adapter: no valid tools found or converted to bind.")
			}
		} else {
			slog.Info("eino adapter: tool registry is empty, no tools to bind.")
		}
	} else {
		slog.Info("eino adapter: tool registry is nil, no tools to bind.")
	}

	return adapter, nil
}

var _ llminterface.ChatAdapter = (*ChatAdapter)(nil)

// GetFrameworkName 返回此适配器实例所适配的底层框架的名称。
func (a *ChatAdapter) GetFrameworkName() string {
	return "eino"
}

// Chat 方法用于向 LLM 发起一次对话交互。
func (a *ChatAdapter) Chat(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error) {
	einoMsgs := ChatInputToEinoMsgs(input)

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
			var parts []llminterface.ContentPart

			// 1. 处理普通内容 (Content 和 MultiContent)
			if msg.Content != "" {
				parts = append(parts, llminterface.ContentPart{
					Type: llminterface.PartTypeText,
					Text: msg.Content,
				})
			} else if msg.MultiContent != nil {
				for _, mcPart := range msg.MultiContent {
					switch mcPart.Type {
					case schema.ChatMessagePartTypeText:
						parts = append(parts, llminterface.ContentPart{
							Type: llminterface.PartTypeText,
							Text: mcPart.Text,
						})
					case schema.ChatMessagePartTypeImageURL:
						if mcPart.ImageURL != nil {
							imageDetail := llminterface.ImageDetailAuto // Default
							if mcPart.ImageURL.Detail != "" {
								switch mcPart.ImageURL.Detail {
								case schema.ImageURLDetailLow:
									imageDetail = llminterface.ImageDetailLow
								case schema.ImageURLDetailHigh:
									imageDetail = llminterface.ImageDetailHigh
								case schema.ImageURLDetailAuto:
									imageDetail = llminterface.ImageDetailAuto
								default:
									slog.Warn("Unknown eino ImageURLDetail, defaulting to auto", "detail", mcPart.ImageURL.Detail)
								}
							}
							parts = append(parts, llminterface.ContentPart{
								Type: llminterface.PartTypeImageURL,
								ImageURL: Some(llminterface.ImageURLContent{
									URL:    mcPart.ImageURL.URL,
									Detail: Some(imageDetail),
								}),
							})
						}
					default:
						slog.Warn("Unsupported eino.ChatMessagePartType in MultiContent.", "type", mcPart.Type)
					}
				}
			}

			// 2. 处理工具调用请求 (ToolCalls)
			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]llminterface.ToolCall, len(msg.ToolCalls))
				for i, tc := range msg.ToolCalls {
					toolCalls[i] = llminterface.ToolCall{
						ID:        tc.ID,
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					}
				}
				parts = append(parts, llminterface.ContentPart{
					Type: llminterface.PartTypeToolCallRequest,
					ToolCallValues: Some(llminterface.ToolCallContent{
						Calls: toolCalls,
					}),
				})
			}

			// 只有当有实际内容或工具调用时才发送 chunk
			if len(parts) > 0 {
				chunk.ContentParts = parts
				chunk.FinishReason = finishReason
				outputChan <- chunk
			}
		}
	}()

	return outputChan, nil
}
