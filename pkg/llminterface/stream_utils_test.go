package llminterface

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// TestMergeChatOutputChunks 测试合并聊天输出块的函数
func TestMergeChatOutputChunks(t *testing.T) {
	// 测试正常情况：从 channel 接收多个块并合并
	t.Run("normal case with multiple chunks", func(t *testing.T) {
		// 创建测试用的 chunks
		chunks := []ChatOutputChunk{
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "你"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "好"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "，"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "世界"},
				},
				FinishReason: Some("stop"),
			},
		}

		// 创建 channel 并发送 chunks
		chunkChan := make(chan ChatOutputChunk, len(chunks))
		for _, chunk := range chunks {
			chunkChan <- chunk
		}
		close(chunkChan)

		// 调用被测函数
		merged, err := MergeChatOutputChunks(context.Background(), chunkChan)

		// 验证结果
		require.NoError(t, err, "MergeChatOutputChunks 应成功合并 chunks")
		assert.Len(t, merged.ContentParts, 4, "合并后应有 4 个 ContentPart")
		assert.Equal(t, "你", merged.ContentParts[0].Text, "第一个部分应为 '你'")
		assert.Equal(t, "好", merged.ContentParts[1].Text, "第二个部分应为 '好'")
		assert.Equal(t, "，", merged.ContentParts[2].Text, "第三个部分应为 '，'")
		assert.Equal(t, "世界", merged.ContentParts[3].Text, "第四个部分应为 '世界'")
		assert.True(t, merged.FinishReason.IsSome(), "应有 FinishReason")
		assert.Equal(t, "stop", merged.FinishReason.Unwrap(), "FinishReason 应为 'stop'")
		assert.Nil(t, merged.Error, "不应有错误")
	})

	// 测试空通道：应返回空的合并结果
	t.Run("empty channel", func(t *testing.T) {
		chunkChan := make(chan ChatOutputChunk)
		close(chunkChan)

		merged, err := MergeChatOutputChunks(context.Background(), chunkChan)

		require.NoError(t, err, "从空 channel 合并不应返回错误")
		assert.Empty(t, merged.ContentParts, "从空 channel 合并应得到空的 ContentParts")
		assert.True(t, merged.FinishReason.IsNone(), "从空 channel 合并不应有 FinishReason")
		assert.Nil(t, merged.Error, "从空 channel 合并不应有错误")
	})

	// 测试带错误的情况：最后一个 chunk 带有 EOF 错误
	t.Run("last chunk with EOF error", func(t *testing.T) {
		chunks := []ChatOutputChunk{
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "部分"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "消息"},
				},
				Error: io.EOF,
			},
		}

		chunkChan := make(chan ChatOutputChunk, len(chunks))
		for _, chunk := range chunks {
			chunkChan <- chunk
		}
		close(chunkChan)

		merged, err := MergeChatOutputChunks(context.Background(), chunkChan)

		require.NoError(t, err, "带 EOF 错误的合并应成功完成")
		assert.Len(t, merged.ContentParts, 2, "合并后应有 2 个 ContentPart")
		assert.Equal(t, io.EOF, merged.Error, "合并结果应包含 EOF 错误")
	})

	// 测试带错误的情况：中间 chunk 带有非 EOF 错误
	t.Run("chunk with non-EOF error", func(t *testing.T) {
		testErr := errors.New("测试错误")
		chunks := []ChatOutputChunk{
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "开始"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "错误前"},
				},
				Error: testErr,
			},
			// 这个 chunk 不应该被处理，因为前一个有非 EOF 错误
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "不应处理"},
				},
			},
		}

		chunkChan := make(chan ChatOutputChunk, len(chunks))
		for _, chunk := range chunks {
			chunkChan <- chunk
		}
		close(chunkChan)

		merged, err := MergeChatOutputChunks(context.Background(), chunkChan)

		assert.Error(t, err, "带非 EOF 错误的合并应返回错误")
		assert.Equal(t, testErr, err, "返回的错误应匹配原始错误")
		assert.Len(t, merged.ContentParts, 2, "合并应包含错误之前的内容")
		assert.Equal(t, "开始", merged.ContentParts[0].Text, "第一部分应正确")
		assert.Equal(t, "错误前", merged.ContentParts[1].Text, "第二部分应正确")
		assert.Equal(t, testErr, merged.Error, "合并结果的 Error 应匹配原始错误")
	})

	// 测试上下文取消
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// 创建一个带缓冲的 channel
		chunkChan := make(chan ChatOutputChunk, 1)

		// 先放入一个 chunk
		chunkChan <- ChatOutputChunk{
			ContentParts: []ContentPart{
				{Type: PartTypeText, Text: "部分数据"},
			},
		}

		// 启动后台协程继续发送数据，但会被阻塞（因为我们延迟读取）
		go func() {
			time.Sleep(100 * time.Millisecond) // 确保上下文已经超时
			chunkChan <- ChatOutputChunk{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "这不应该被包含"},
				},
			}
			close(chunkChan)
		}()

		merged, err := MergeChatOutputChunks(ctx, chunkChan)

		assert.Error(t, err, "上下文取消应返回错误")
		assert.True(t, errors.Is(err, ErrStreamCanceled), "错误应包含 ErrStreamCanceled")
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "错误应包含原始上下文错误 DeadlineExceeded")
		assert.Len(t, merged.ContentParts, 1, "合并结果应包含取消前收到的内容")
		assert.Equal(t, "部分数据", merged.ContentParts[0].Text, "应包含取消前收到的文本")
		assert.True(t, errors.Is(merged.Error, context.DeadlineExceeded), "合并结果的 Error 应为 context.DeadlineExceeded")
	})
}

// TestAggregateTextFromChunks 测试聚合文本的函数
func TestAggregateTextFromChunks(t *testing.T) {
	// 测试正常情况：从 channel 接收多个块并合并文本
	t.Run("normal case with multiple text chunks", func(t *testing.T) {
		// 创建测试用的 chunks
		chunks := []ChatOutputChunk{
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "你好"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "，"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "世界"},
				},
				FinishReason: Some("stop"),
			},
		}

		// 创建 channel 并发送 chunks
		chunkChan := make(chan ChatOutputChunk, len(chunks))
		for _, chunk := range chunks {
			chunkChan <- chunk
		}
		close(chunkChan)

		// 调用被测函数
		text, err := AggregateTextFromChunks(context.Background(), chunkChan)

		// 验证结果
		require.NoError(t, err, "AggregateTextFromChunks 应成功聚合文本")
		assert.Equal(t, "你好，世界", text, "应正确连接所有文本片段")
	})

	// 测试混合内容：有些 ContentPart 不是文本
	t.Run("mixed content types", func(t *testing.T) {
		chunks := []ChatOutputChunk{
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "文本开始 "},
					{Type: PartTypeImageURL}, // 不是文本类型
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "中间文本 "},
					{Type: PartTypeToolCallRequest}, // 不是文本类型
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "文本结束"},
				},
			},
		}

		chunkChan := make(chan ChatOutputChunk, len(chunks))
		for _, chunk := range chunks {
			chunkChan <- chunk
		}
		close(chunkChan)

		text, err := AggregateTextFromChunks(context.Background(), chunkChan)

		require.NoError(t, err, "混合内容类型的聚合应成功")
		assert.Equal(t, "文本开始 中间文本 文本结束", text, "应只聚合文本类型的内容")
	})

	// 测试错误处理
	t.Run("error handling", func(t *testing.T) {
		testErr := errors.New("测试错误")
		chunks := []ChatOutputChunk{
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "错误前的文本 "},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "带有错误 "},
				},
				Error: testErr,
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "这不应该被包含"},
				},
			},
		}

		chunkChan := make(chan ChatOutputChunk, len(chunks))
		for _, chunk := range chunks {
			chunkChan <- chunk
		}
		close(chunkChan)

		text, err := AggregateTextFromChunks(context.Background(), chunkChan)

		assert.Error(t, err, "带错误的聚合应返回错误")
		assert.Equal(t, testErr, err, "返回的错误应匹配原始错误")
		assert.Equal(t, "错误前的文本 带有错误 ", text, "应返回错误前的所有文本")
	})

	// 测试 EOF 错误处理，应该视为正常结束
	t.Run("EOF error handling", func(t *testing.T) {
		chunks := []ChatOutputChunk{
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "正常文本 "},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "EOF前 "},
				},
				Error: io.EOF,
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "EOF后，应该被包含"},
				},
			},
		}

		chunkChan := make(chan ChatOutputChunk, len(chunks))
		for _, chunk := range chunks {
			chunkChan <- chunk
		}
		close(chunkChan)

		text, err := AggregateTextFromChunks(context.Background(), chunkChan)

		require.NoError(t, err, "EOF 错误不应导致函数返回错误")
		assert.Equal(t, "正常文本 EOF前 EOF后，应该被包含", text, "应返回包括 EOF 后的所有文本")
	})

	// 测试上下文取消
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// 创建一个带缓冲的 channel
		chunkChan := make(chan ChatOutputChunk, 1)

		// 先放入一个 chunk
		chunkChan <- ChatOutputChunk{
			ContentParts: []ContentPart{
				{Type: PartTypeText, Text: "取消前的文本"},
			},
		}

		// 启动后台协程继续发送数据，但会被阻塞（因为我们延迟读取）
		go func() {
			time.Sleep(100 * time.Millisecond) // 确保上下文已经超时
			chunkChan <- ChatOutputChunk{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "这不应该被包含"},
				},
			}
			close(chunkChan)
		}()

		text, err := AggregateTextFromChunks(ctx, chunkChan)

		assert.Error(t, err, "上下文取消应返回错误")
		assert.True(t, errors.Is(err, ErrStreamCanceled), "错误应包含 ErrStreamCanceled")
		assert.True(t, errors.Is(err, context.DeadlineExceeded), "错误应包含原始上下文错误 DeadlineExceeded")
		assert.Equal(t, "取消前的文本", text, "应返回取消前的所有文本")
	})

	// 测试带初始容量参数
	t.Run("with initial capacity", func(t *testing.T) {
		chunks := []ChatOutputChunk{
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "测试"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "初始"},
				},
			},
			{
				ContentParts: []ContentPart{
					{Type: PartTypeText, Text: "容量"},
				},
			},
		}

		chunkChan := make(chan ChatOutputChunk, len(chunks))
		for _, chunk := range chunks {
			chunkChan <- chunk
		}
		close(chunkChan)

		// 使用自定义初始容量
		text, err := AggregateTextFromChunks(context.Background(), chunkChan, 100)

		require.NoError(t, err, "带初始容量参数的聚合应成功")
		assert.Equal(t, "测试初始容量", text, "应正确聚合所有文本")
	})
}
