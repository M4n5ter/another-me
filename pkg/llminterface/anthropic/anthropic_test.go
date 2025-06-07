package anthropic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
)

func TestAnthropicAdapter_Basic(t *testing.T) {
	// 创建基本的适配器配置
	config := &AnthropicAdapterConfig{
		MaxTokens: 1000,
	}

	// 创建适配器（使用虚拟的API密钥进行测试）
	adapter := NewAnthropicAdapter("test-api-key", None[string](), config)

	// 验证适配器创建成功
	assert.NotNil(t, adapter)
	assert.Equal(t, "anthropic-sdk-go", adapter.GetFrameworkName())
}

func TestAnthropicAdapter_ProduceJSON_WithoutSchema(t *testing.T) {
	config := &AnthropicAdapterConfig{
		MaxTokens: 1000,
	}

	adapter := NewAnthropicAdapter("test-api-key", None[string](), config)

	// 测试输入
	input := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: "生成一个简单的JSON对象",
					},
				},
			},
		},
	}

	// 测试ProduceJSON方法（不提供schema）
	// 注意：这里会失败，因为我们没有真实的API密钥，但至少可以验证方法签名正确
	_, err := adapter.ProduceJSON(context.Background(), input, None[schema.Schema]())

	// 我们期望会有错误（因为使用了虚拟API密钥），但不应该是编译错误
	assert.Error(t, err)
}

func TestAnthropicAdapter_RegisterTools(t *testing.T) {
	config := &AnthropicAdapterConfig{
		MaxTokens: 1000,
	}

	adapter := NewAnthropicAdapter("test-api-key", None[string](), config)

	// 测试注册空的工具注册表
	err := adapter.RegisterTools(context.Background(), nil)
	assert.NoError(t, err)
}

func TestAnthropicAdapter_ConvertMessages(t *testing.T) {
	config := &AnthropicAdapterConfig{
		MaxTokens: 1000,
	}

	adapter := NewAnthropicAdapter("test-api-key", None[string](), config)

	// 测试消息转换
	input := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: "测试消息",
					},
				},
			},
		},
	}

	messages, err := adapter.convertChatInputToAnthropicMsgs(input)
	assert.NoError(t, err)
	assert.Len(t, messages, 1)
}
