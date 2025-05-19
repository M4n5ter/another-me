package llminterface

import (
	"context"
	"fmt"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// BaseChatAdapter 提供了ChatAdapter接口的基础实现
// 它可以被直接使用或被其他特定实现包含来复用通用功能
type BaseChatAdapter struct {
	// ChatFunc 是一个函数，它实现了实际的聊天处理逻辑
	// 如果提供了此函数，BaseChatAdapter的Chat方法会直接调用它
	ChatFunc func(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error)

	// FrameworkName 是此适配器实例所适配的底层框架的名称
	FrameworkName string
}

// GetFrameworkName 返回此适配器实例所适配的底层框架的名称
func (b *BaseChatAdapter) GetFrameworkName() string {
	if b.FrameworkName != "" {
		return b.FrameworkName
	}
	return "base-adapter"
}

// Chat 实现ChatAdapter接口的Chat方法
// 如果提供了ChatFunc，则调用它；否则返回错误
func (b *BaseChatAdapter) Chat(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error) {
	if b.ChatFunc != nil {
		return b.ChatFunc(ctx, input)
	}
	return nil, fmt.Errorf("没有提供ChatFunc实现")
}

// NewBaseChatAdapter 创建一个新的BaseChatAdapter实例
func NewBaseChatAdapter(frameworkName string, chatFunc func(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error)) *BaseChatAdapter {
	return &BaseChatAdapter{
		FrameworkName: frameworkName,
		ChatFunc:      chatFunc,
	}
}

// ChatAndGetFullResponse 是一个辅助函数，用于执行Chat操作并返回完整的LLMResponse
// 它处理所有流式响应的聚合，并提供一个更简单的接口来获取LLM响应
func ChatAndGetFullResponse(ctx context.Context, adapter ChatAdapter, input ChatInput) (LLMResponse, error) {
	chunkChan, err := adapter.Chat(ctx, input)
	if err != nil {
		return LLMResponse{Error: err}, fmt.Errorf("chat方法返回错误: %w", err)
	}

	// 合并所有聊天输出块
	mergedChunk, err := MergeChatOutputChunks(ctx, chunkChan)
	if err != nil {
		return LLMResponse{
			Content:      mergedChunk.ContentParts,
			Error:        err,
			FinishReason: mergedChunk.FinishReason,
		}, fmt.Errorf("合并聊天输出块时返回错误: %w", err)
	}

	// 提取所有文本内容
	fullText, err := AggregateTextFromChunks(ctx, replayChatOutputChunks(ctx, mergedChunk))

	return LLMResponse{
		Content:      mergedChunk.ContentParts,
		FullText:     fullText,
		Error:        mergedChunk.Error,
		FinishReason: mergedChunk.FinishReason,
	}, err
}

// replayChatOutputChunks 创建一个通道，在该通道上重放一个ChatOutputChunk
// 这是为了复用AggregateTextFromChunks函数而不必重新实现文本聚合逻辑
func replayChatOutputChunks(ctx context.Context, chunk ChatOutputChunk) <-chan ChatOutputChunk {
	ch := make(chan ChatOutputChunk, 1)
	go func() {
		select {
		case <-ctx.Done():
			close(ch)
			return
		case ch <- chunk:
			close(ch)
			return
		}
	}()
	return ch
}

// CreateSimpleChatAdapter 创建一个简单的ChatAdapter，它使用提供的生成函数来处理聊天请求
// 这是创建自定义ChatAdapter的最简单方式，无需实现完整接口
func CreateSimpleChatAdapter(
	frameworkName string,
	generatorFunc func(ctx context.Context, messages []InputMessage) (string, error),
) ChatAdapter {
	return NewBaseChatAdapter(frameworkName, func(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error) {
		ch := make(chan ChatOutputChunk, 1)

		go func() {
			defer close(ch)

			// 调用生成函数获取完整响应
			fullText, err := generatorFunc(ctx, input.Messages)
			if err != nil {
				ch <- ChatOutputChunk{
					Error: err,
				}
				return
			}

			// 构建并发送完整响应
			ch <- ChatOutputChunk{
				ContentParts: []ContentPart{
					{
						Type: PartTypeText,
						Text: fullText,
					},
				},
				FinishReason: Some("stop"),
			}
		}()

		return ch, nil
	})
}
