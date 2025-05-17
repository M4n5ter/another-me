package mock

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewMockChatAdapter 测试创建新的 MockChatAdapter 实例
func TestNewMockChatAdapter(t *testing.T) {
	mock := NewMockChatAdapter()
	assert.NotNil(t, mock, "NewMockChatAdapter 返回的实例不应为 nil")
	assert.Empty(t, mock.RecordedChatInputs, "新创建的 mock 不应有记录的输入")
	assert.Equal(t, 0, mock.GetChatCallCount(), "新创建的 mock 的调用计数应为 0")
	assert.Empty(t, mock.FrameworkNameResult, "新创建的 mock 的框架名称应为空")
}

// TestMockChatAdapter_GetFrameworkName 测试 GetFrameworkName 方法
func TestMockChatAdapter_GetFrameworkName(t *testing.T) {
	mock := NewMockChatAdapter()
	
	// 初始应为空
	assert.Empty(t, mock.GetFrameworkName(), "初始框架名称应为空")
	
	// 设置后应返回设置的值
	mock.SetFrameworkName("test-framework")
	assert.Equal(t, "test-framework", mock.GetFrameworkName(), "应返回设置的框架名称")
}

// TestMockChatAdapter_Chat_WithPredefinedResponses 测试带有预定义响应的 Chat 方法
func TestMockChatAdapter_Chat_WithPredefinedResponses(t *testing.T) {
	mock := NewMockChatAdapter()
	ctx := context.Background()
	
	// 定义测试输入
	input := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: "你好",
					},
				},
			},
		},
	}
	
	// 添加预定义响应
	successChunks := []llminterface.ChatOutputChunk{
		{TextDelta: "你"},
		{TextDelta: "好"},
		{TextDelta: "！"},
	}
	mock.AddPredefinedChatResponse(successChunks, nil)
	
	// 调用 Chat 方法并检查结果
	resultChan, err := mock.Chat(ctx, input)
	require.NoError(t, err, "Chat 方法不应返回初始错误")
	require.NotNil(t, resultChan, "Chat 方法应返回有效的 channel")
	
	// 从 channel 接收所有块并验证
	var receivedChunks []llminterface.ChatOutputChunk
	for chunk := range resultChan {
		receivedChunks = append(receivedChunks, chunk)
	}
	
	assert.Equal(t, len(successChunks), len(receivedChunks), "应接收到预期数量的块")
	for i, expectedChunk := range successChunks {
		assert.Equal(t, expectedChunk.TextDelta, receivedChunks[i].TextDelta, "接收到的第 %d 个块的 TextDelta 应匹配", i)
	}
	
	// 验证调用计数和记录的输入
	assert.Equal(t, 1, mock.GetChatCallCount(), "调用计数应为 1")
	assert.Len(t, mock.RecordedChatInputs, 1, "应记录 1 个输入")
	assert.Equal(t, input, mock.RecordedChatInputs[0], "记录的输入应与传入的输入匹配")
	
	// 测试清除预定义响应
	mock.ClearPredefinedChatResponses()
	assert.Equal(t, 0, mock.GetChatCallCount(), "清除后调用计数应重置为 0")
	
	// 测试没有预定义响应时的错误情况
	_, err = mock.Chat(ctx, input)
	assert.Error(t, err, "没有预定义响应时 Chat 应返回错误")
	assert.Contains(t, err.Error(), "no predefined response", "错误消息应指出没有预定义响应")
}

// TestMockChatAdapter_Chat_WithInitialError 测试带有初始错误的预定义响应
func TestMockChatAdapter_Chat_WithInitialError(t *testing.T) {
	mock := NewMockChatAdapter()
	ctx := context.Background()
	
	// 添加带初始错误的预定义响应
	expectedError := errors.New("初始错误")
	mock.AddPredefinedChatResponse(nil, expectedError)
	
	// 调用 Chat 方法并检查结果
	resultChan, err := mock.Chat(ctx, llminterface.ChatInput{})
	assert.Error(t, err, "Chat 方法应返回初始错误")
	assert.Equal(t, expectedError, err, "返回的错误应匹配预设的错误")
	assert.Nil(t, resultChan, "发生初始错误时，返回的 channel 应为 nil")
}

// TestMockChatAdapter_Chat_WithCustomFunction 测试使用自定义函数的 Chat 方法
func TestMockChatAdapter_Chat_WithCustomFunction(t *testing.T) {
	mock := NewMockChatAdapter()
	ctx := context.Background()
	
	// 设置自定义 ChatFunc
	customCalled := false
	customOutput := make(chan llminterface.ChatOutputChunk, 1)
	customOutput <- llminterface.ChatOutputChunk{TextDelta: "自定义响应"}
	close(customOutput)
	
	mock.ChatFunc = func(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error) {
		customCalled = true
		return customOutput, nil
	}
	
	// 调用 Chat 方法并检查结果
	resultChan, err := mock.Chat(ctx, llminterface.ChatInput{})
	require.NoError(t, err, "自定义 ChatFunc 不应返回错误")
	assert.True(t, customCalled, "自定义 ChatFunc 应被调用")
	
	// 不应直接比较 channel 的值，而是检查能否从 channel 中收到预期的数据
	chunk, ok := <-resultChan
	assert.True(t, ok, "应能从返回的 channel 中接收数据")
	assert.Equal(t, "自定义响应", chunk.TextDelta, "接收到的数据应匹配")
	
	// 确认 channel 已关闭
	_, ok = <-resultChan
	assert.False(t, ok, "channel 应已关闭")
}

// TestMockChatAdapter_Chat_WithContextCancellation 测试上下文取消的场景
func TestMockChatAdapter_Chat_WithContextCancellation(t *testing.T) {
	mock := NewMockChatAdapter()
	
	// 创建可取消的上下文
	ctx, cancel := context.WithCancel(context.Background())
	
	// 添加一个将会发送多个块的预定义响应
	longResponse := make([]llminterface.ChatOutputChunk, 5)
	for i := range longResponse {
		longResponse[i] = llminterface.ChatOutputChunk{TextDelta: "块 " + string(rune(i+'0'))}
	}
	mock.AddPredefinedChatResponse(longResponse, nil)
	
	// 调用 Chat 方法
	resultChan, err := mock.Chat(ctx, llminterface.ChatInput{})
	require.NoError(t, err, "Chat 方法不应返回初始错误")
	
	// 读取第一个块
	firstChunk := <-resultChan
	assert.Equal(t, "块 0", firstChunk.TextDelta, "第一个块的内容应匹配")
	
	// 取消上下文
	cancel()
	
	// 等待一小段时间，让 goroutine 有机会响应取消
	time.Sleep(50 * time.Millisecond)
	
	// 尝试读取剩余的块 - 应该不会阻塞，而是返回错误或关闭通道
	for chunk := range resultChan {
		if chunk.Error != nil {
			assert.ErrorIs(t, chunk.Error, context.Canceled, "channel 上的错误应是 context.Canceled")
			break
		}
	}
}

// TestMockChatAdapter_InputRetrieval 测试获取记录的输入
func TestMockChatAdapter_InputRetrieval(t *testing.T) {
	mock := NewMockChatAdapter()
	ctx := context.Background()
	
	// 调用 Chat 方法几次
	mock.AddPredefinedChatResponse([]llminterface.ChatOutputChunk{{TextDelta: "响应1"}}, nil)
	mock.AddPredefinedChatResponse([]llminterface.ChatOutputChunk{{TextDelta: "响应2"}}, nil)
	
	input1 := llminterface.ChatInput{Messages: []llminterface.InputMessage{{Role: llminterface.RoleUser, Content: []llminterface.ContentPart{{Type: llminterface.PartTypeText, Text: "输入1"}}}}}
	input2 := llminterface.ChatInput{Messages: []llminterface.InputMessage{{Role: llminterface.RoleUser, Content: []llminterface.ContentPart{{Type: llminterface.PartTypeText, Text: "输入2"}}}}}
	
	_, _ = mock.Chat(ctx, input1)
	_, _ = mock.Chat(ctx, input2)
	
	// 测试 GetLastRecordedChatInput
	lastInput, exists := mock.GetLastRecordedChatInput()
	assert.True(t, exists, "应能获取最后记录的输入")
	assert.Equal(t, input2, lastInput, "最后记录的输入应是第二个输入")
	
	// 测试 GetRecordedChatInputAt
	firstInput, exists := mock.GetRecordedChatInputAt(0)
	assert.True(t, exists, "应能获取第一个记录的输入")
	assert.Equal(t, input1, firstInput, "第一个记录的输入应是第一个输入")
	
	secondInput, exists := mock.GetRecordedChatInputAt(1)
	assert.True(t, exists, "应能获取第二个记录的输入")
	assert.Equal(t, input2, secondInput, "第二个记录的输入应是第二个输入")
	
	// 测试索引越界
	_, exists = mock.GetRecordedChatInputAt(2)
	assert.False(t, exists, "获取不存在的输入应返回 exists=false")
	
	// 测试 ClearRecordedChatInputs
	mock.ClearRecordedChatInputs()
	assert.Empty(t, mock.RecordedChatInputs, "清除后不应有记录的输入")
	
	_, exists = mock.GetLastRecordedChatInput()
	assert.False(t, exists, "清除后应无法获取最后记录的输入")
} 