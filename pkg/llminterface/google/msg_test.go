package google

import (
	"testing"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
)

func TestInputMessageToGenaiContent_EmptyPartsHandling(t *testing.T) {
	// 测试当所有工具调用都失败时，应该返回 nil 而不是空的 Content
	msg := llminterface.InputMessage{
		Role: llminterface.RoleAssistant,
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeToolCallRequest,
				ToolCallValues: Some(llminterface.ToolCallContent{
					Calls: []llminterface.ToolCall{
						{
							ID:        "test-id",
							Name:      "test-tool",
							Arguments: `invalid-json`, // 这会导致解析失败
						},
					},
				}),
			},
		},
	}

	result := InputMessageToGenaiContent(msg)
	if result != nil {
		t.Errorf("Expected nil result when all tool calls fail to parse, got %v", result)
	}
}

func TestInputMessageToGenaiContent_ValidToolCall(t *testing.T) {
	// 测试有效的工具调用应该正常处理
	msg := llminterface.InputMessage{
		Role: llminterface.RoleAssistant,
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeToolCallRequest,
				ToolCallValues: Some(llminterface.ToolCallContent{
					Calls: []llminterface.ToolCall{
						{
							ID:        "test-id",
							Name:      "test-tool",
							Arguments: `{"param": "value"}`, // 有效的 JSON
						},
					},
				}),
			},
		},
	}

	result := InputMessageToGenaiContent(msg)
	if result == nil {
		t.Error("Expected non-nil result for valid tool call")
	}

	if result != nil && len(result.Parts) != 1 {
		t.Errorf("Expected 1 part, got %d", len(result.Parts))
	}

	if result.Parts[0].FunctionCall == nil {
		t.Error("Expected FunctionCall to be set")
	}

	if result.Parts[0].FunctionCall.Name != "test-tool" {
		t.Errorf("Expected tool name 'test-tool', got %s", result.Parts[0].FunctionCall.Name)
	}
}

func TestInputMessageToGenaiContent_MixedContent(t *testing.T) {
	// 测试混合内容（文本 + 工具调用）
	msg := llminterface.InputMessage{
		Role: llminterface.RoleAssistant,
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeText,
				Text: "I'll help you with that.",
			},
			{
				Type: llminterface.PartTypeToolCallRequest,
				ToolCallValues: Some(llminterface.ToolCallContent{
					Calls: []llminterface.ToolCall{
						{
							ID:        "test-id",
							Name:      "test-tool",
							Arguments: `{"param": "value"}`,
						},
					},
				}),
			},
		},
	}

	result := InputMessageToGenaiContent(msg)
	if result == nil {
		t.Error("Expected non-nil result for mixed content")
	}

	if result != nil && len(result.Parts) != 2 {
		t.Errorf("Expected 2 parts, got %d", len(result.Parts))
	}
}

func TestChatInputToGenaiMsgs_SkipsNilContent(t *testing.T) {
	// 测试 ChatInputToGenaiMsgs 正确跳过 nil content
	input := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: "Hello",
					},
				},
			},
			{
				Role: llminterface.RoleAssistant,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeToolCallRequest,
						ToolCallValues: Some(llminterface.ToolCallContent{
							Calls: []llminterface.ToolCall{
								{
									ID:        "test-id",
									Name:      "test-tool",
									Arguments: `invalid-json`, // 这会导致该消息被跳过
								},
							},
						}),
					},
				},
			},
		},
	}

	result := ChatInputToGenaiMsgs(input)

	// 应该只有一个消息（用户消息），助手消息因为无效工具调用被跳过
	if len(result) != 1 {
		t.Errorf("Expected 1 message, got %d", len(result))
	}

	if result[0].Role != "user" {
		t.Errorf("Expected user role, got %s", result[0].Role)
	}
}
