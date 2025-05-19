package llminterface

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/m4n5ter/another-me/pkg/option"
)

func TestBaseChatAdapter_GetFrameworkName(t *testing.T) {
	// 测试指定了框架名称的情况
	adapter := BaseChatAdapter{FrameworkName: "test-framework"}
	assert.Equal(t, "test-framework", adapter.GetFrameworkName(), "应返回指定的框架名称")

	// 测试未指定框架名称的默认情况
	adapter = BaseChatAdapter{}
	assert.Equal(t, "base-adapter", adapter.GetFrameworkName(), "未指定时应返回默认框架名称")
}

func TestBaseChatAdapter_Chat(t *testing.T) {
	ctx := context.Background()
	input := ChatInput{}

	// 测试未提供ChatFunc时
	adapter := BaseChatAdapter{}
	_, err := adapter.Chat(ctx, input)
	assert.Error(t, err, "未提供ChatFunc时应返回错误")
	assert.Contains(t, err.Error(), "没有提供ChatFunc实现", "错误消息应指出未提供ChatFunc")

	// 测试提供了ChatFunc时
	mockChatFunc := func(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error) {
		ch := make(chan ChatOutputChunk, 1)
		ch <- ChatOutputChunk{
			ContentParts: []ContentPart{
				{Type: PartTypeText, Text: "测试响应"},
			},
		}
		close(ch)
		return ch, nil
	}

	adapter = BaseChatAdapter{ChatFunc: mockChatFunc}
	result, err := adapter.Chat(ctx, input)
	assert.NoError(t, err, "提供ChatFunc时不应返回错误")

	chunk := <-result
	assert.Equal(t, "测试响应", chunk.ContentParts[0].Text, "应返回ChatFunc产生的响应")
}

func TestNewBaseChatAdapter(t *testing.T) {
	mockChatFunc := func(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error) {
		return nil, nil
	}

	adapter := NewBaseChatAdapter("test-framework", mockChatFunc)

	assert.Equal(t, "test-framework", adapter.FrameworkName, "应正确设置框架名称")
	assert.NotNil(t, adapter.ChatFunc, "应正确设置ChatFunc")
}

func TestChatAndGetFullResponse(t *testing.T) {
	ctx := context.Background()
	input := ChatInput{}
	testErr := errors.New("测试错误")

	t.Run("adapter error", func(t *testing.T) {
		// 测试适配器返回错误的情况
		adapter := NewBaseChatAdapter("test", func(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error) {
			return nil, testErr
		})

		response, err := ChatAndGetFullResponse(ctx, adapter, input)
		assert.Error(t, err, "适配器返回错误时应传递该错误")
		assert.Equal(t, testErr, err, "应返回原始错误")
		assert.Equal(t, testErr, response.Error, "响应中也应包含错误")
	})

	t.Run("successful response", func(t *testing.T) {
		// 测试成功响应
		adapter := NewBaseChatAdapter("test", func(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error) {
			ch := make(chan ChatOutputChunk, 3)
			ch <- ChatOutputChunk{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "你好"},
				},
			}
			ch <- ChatOutputChunk{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "，世界"},
				},
			}
			ch <- ChatOutputChunk{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "！"},
				},
				FinishReason: Some("stop"),
			}
			close(ch)
			return ch, nil
		})

		response, err := ChatAndGetFullResponse(ctx, adapter, input)
		assert.NoError(t, err, "成功响应不应返回错误")
		assert.Equal(t, "你好，世界！", response.FullText, "应正确合并所有文本内容")
		assert.Len(t, response.Content, 3, "应包含所有内容部分")
		assert.True(t, response.FinishReason.IsSome(), "应包含完成原因")
		assert.Equal(t, "stop", response.FinishReason.Unwrap(), "应正确设置完成原因")
	})

	t.Run("context cancellation", func(t *testing.T) {
		// 测试上下文取消
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		adapter := NewBaseChatAdapter("test", func(ctx context.Context, input ChatInput) (<-chan ChatOutputChunk, error) {
			ch := make(chan ChatOutputChunk)
			go func() {
				time.Sleep(50 * time.Millisecond) // 等待时间长于上下文超时时间
				ch <- ChatOutputChunk{
					ContentParts: []ContentPart{
						{Type: PartTypeText, Text: "此内容不应被处理"},
					},
				}
				close(ch)
			}()
			return ch, nil
		})

		response, err := ChatAndGetFullResponse(ctx, adapter, input)
		assert.Error(t, err, "上下文取消应返回错误")
		assert.True(t, errors.Is(err, ErrStreamCanceled), "错误应包含ErrStreamCanceled")
		assert.Empty(t, response.FullText, "上下文取消时不应有文本内容")
	})
}

func TestCreateSimpleChatAdapter(t *testing.T) {
	ctx := context.Background()
	testErr := errors.New("生成错误")

	t.Run("successful generation", func(t *testing.T) {
		// 测试成功生成
		adapter := CreateSimpleChatAdapter("test-simple", func(ctx context.Context, messages []InputMessage) (string, error) {
			return "简单的响应", nil
		})

		assert.Equal(t, "test-simple", adapter.GetFrameworkName(), "应正确设置框架名称")

		result, err := adapter.Chat(ctx, ChatInput{})
		require.NoError(t, err, "不应返回错误")

		chunk := <-result
		assert.Equal(t, "简单的响应", chunk.ContentParts[0].Text, "应返回生成函数产生的文本")
		assert.True(t, chunk.FinishReason.IsSome(), "应设置完成原因")
		assert.Equal(t, "stop", chunk.FinishReason.Unwrap(), "完成原因应为stop")

		_, ok := <-result
		assert.False(t, ok, "通道应已关闭")
	})

	t.Run("generation error", func(t *testing.T) {
		// 测试生成错误
		adapter := CreateSimpleChatAdapter("test-simple", func(ctx context.Context, messages []InputMessage) (string, error) {
			return "", testErr
		})

		result, err := adapter.Chat(ctx, ChatInput{})
		require.NoError(t, err, "Chat方法本身不应返回错误")

		chunk := <-result
		assert.Equal(t, testErr, chunk.Error, "应通过块的Error字段返回错误")
		assert.Empty(t, chunk.ContentParts, "错误时不应有内容")

		_, ok := <-result
		assert.False(t, ok, "通道应已关闭")
	})

	t.Run("context cancellation", func(t *testing.T) {
		// 测试上下文取消
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		adapter := CreateSimpleChatAdapter("test-simple", func(ctx context.Context, messages []InputMessage) (string, error) {
			time.Sleep(50 * time.Millisecond) // 等待时间长于上下文超时时间
			return "此内容不应被返回", nil
		})

		result, err := adapter.Chat(ctx, ChatInput{})
		require.NoError(t, err, "Chat方法本身不应返回错误")

		// 通道可能还没关闭，但至少不应该有内容
		time.Sleep(20 * time.Millisecond)
		select {
		case chunk, ok := <-result:
			if ok {
				assert.Error(t, chunk.Error, "如果收到块，应包含错误")
			}
		default:
			// 这是预期的空通道行为，什么都不做
		}
	})
}
