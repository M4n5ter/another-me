package llminterface

import (
	"testing"

	json "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// TestContentPartTypes 测试 ContentPartType 常量定义
func TestContentPartTypes(t *testing.T) {
	assert.Equal(t, ContentPartType("text"), PartTypeText, "PartTypeText 应为 'text'")
	assert.Equal(t, ContentPartType("image_url"), PartTypeImageURL, "PartTypeImageURL 应为 'image_url'")
}

// TestImageURLDetails 测试 ImageURLDetail 常量定义
func TestImageURLDetails(t *testing.T) {
	assert.Equal(t, ImageURLDetail("low"), ImageDetailLow, "ImageDetailLow 应为 'low'")
	assert.Equal(t, ImageURLDetail("high"), ImageDetailHigh, "ImageDetailHigh 应为 'high'")
	assert.Equal(t, ImageURLDetail("auto"), ImageDetailAuto, "ImageDetailAuto 应为 'auto'")
}

// TestMessageRoles 测试 MessageRole 常量定义
func TestMessageRoles(t *testing.T) {
	assert.Equal(t, MessageRole("user"), RoleUser, "RoleUser 应为 'user'")
	assert.Equal(t, MessageRole("assistant"), RoleAssistant, "RoleAssistant 应为 'assistant'")
	assert.Equal(t, MessageRole("system"), RoleSystem, "RoleSystem 应为 'system'")
}

// TestContentPartJSON 测试 ContentPart 的 JSON 编码和解码
func TestContentPartJSON(t *testing.T) {
	// 测试文本类型的 ContentPart
	textPart := ContentPart{
		Type: PartTypeText,
		Text: "Hello, world!",
	}

	textJSON, err := json.MarshalToString(textPart)
	require.NoError(t, err, "Marshal 文本类型的 ContentPart 不应返回错误")

	expectedTextJSON := `{"type":"text","text":"Hello, world!"}`
	assert.JSONEq(t, expectedTextJSON, textJSON, "文本类型的 ContentPart 应正确编码为 JSON")

	var decodedTextPart ContentPart
	err = json.UnmarshalFromString(textJSON, &decodedTextPart)
	require.NoError(t, err, "Unmarshal 文本类型的 JSON 不应返回错误")
	assert.Equal(t, textPart, decodedTextPart, "解码后的 ContentPart 应与原始对象相同")

	// 测试图像 URL 类型的 ContentPart
	imagePart := ContentPart{
		Type: PartTypeImageURL,
		ImageURL: Some(ImageURLContent{
			URL:    "https://example.com/image.jpg",
			Detail: Some(ImageDetailHigh),
		}),
	}

	imageJSON, err := json.MarshalToString(imagePart)
	require.NoError(t, err, "Marshal 图像 URL 类型的 ContentPart 不应返回错误")

	expectedImageJSON := `{"type":"image_url","image_url":{"url":"https://example.com/image.jpg","detail":"high"}}`
	assert.JSONEq(t, expectedImageJSON, imageJSON, "图像 URL 类型的 ContentPart 应正确编码为 JSON")

	var decodedImagePart ContentPart
	err = json.UnmarshalFromString(imageJSON, &decodedImagePart)
	require.NoError(t, err, "Unmarshal 图像 URL 类型的 JSON 不应返回错误")
	assert.Equal(t, imagePart.Type, decodedImagePart.Type, "解码后的 Type 应匹配")
	assert.Equal(t, imagePart.ImageURL.Unwrap().URL, decodedImagePart.ImageURL.Unwrap().URL, "解码后的 URL 应匹配")
	assert.Equal(t, imagePart.ImageURL.Unwrap().Detail, decodedImagePart.ImageURL.Unwrap().Detail, "解码后的 Detail 应匹配")
}

// TestInputMessageJSON 测试 InputMessage 的 JSON 编码和解码
func TestInputMessageJSON(t *testing.T) {
	// 创建一个包含文本和图像的复合消息
	message := InputMessage{
		Role: RoleUser,
		Content: []ContentPart{
			{
				Type: PartTypeText,
				Text: "有关此图像的信息：",
			},
			{
				Type: PartTypeImageURL,
				ImageURL: Some(ImageURLContent{
					URL:    "https://example.com/image.jpg",
					Detail: Some(ImageDetailHigh),
				}),
			},
		},
	}

	messageJSON, err := json.MarshalToString(message)
	require.NoError(t, err, "Marshal InputMessage 不应返回错误")

	var decodedMessage InputMessage
	err = json.UnmarshalFromString(messageJSON, &decodedMessage)
	require.NoError(t, err, "Unmarshal InputMessage JSON 不应返回错误")

	assert.Equal(t, message.Role, decodedMessage.Role, "解码后的 Role 应匹配")
	assert.Equal(t, len(message.Content), len(decodedMessage.Content), "解码后的 Content 长度应匹配")
	assert.Equal(t, message.Content[0].Type, decodedMessage.Content[0].Type, "第一个 Content 的 Type 应匹配")
	assert.Equal(t, message.Content[0].Text, decodedMessage.Content[0].Text, "第一个 Content 的 Text 应匹配")
	assert.Equal(t, message.Content[1].Type, decodedMessage.Content[1].Type, "第二个 Content 的 Type 应匹配")
	assert.Equal(t, message.Content[1].ImageURL.Unwrap().URL, decodedMessage.Content[1].ImageURL.Unwrap().URL, "第二个 Content 的 URL 应匹配")
}

// TestChatInputJSON 测试 ChatInput 的 JSON 编码和解码
func TestChatInputJSON(t *testing.T) {
	// 创建一个包含系统消息和用户消息的聊天输入
	chatInput := ChatInput{
		Messages: []InputMessage{
			{
				Role: RoleSystem,
				Content: []ContentPart{
					{
						Type: PartTypeText,
						Text: "你是一个有用的助手",
					},
				},
			},
			{
				Role: RoleUser,
				Content: []ContentPart{
					{
						Type: PartTypeText,
						Text: "请帮我解释这个图片",
					},
					{
						Type: PartTypeImageURL,
						ImageURL: Some(ImageURLContent{
							URL:    "https://example.com/image.jpg",
							Detail: Some(ImageDetailHigh),
						}),
					},
				},
			},
		},
		ConversationID: "test-conversation",
	}

	chatInputJSON, err := json.MarshalToString(chatInput)
	require.NoError(t, err, "Marshal ChatInput 不应返回错误")

	// ConversationID 应被省略，因为其 JSON 标签为 "-"
	assert.NotContains(t, chatInputJSON, "conversation_id", "ConversationID 不应出现在 JSON 中")
	assert.NotContains(t, chatInputJSON, "test-conversation", "ConversationID 的值不应出现在 JSON 中")

	var decodedChatInput ChatInput
	err = json.UnmarshalFromString(chatInputJSON, &decodedChatInput)
	require.NoError(t, err, "Unmarshal ChatInput JSON 不应返回错误")

	assert.Equal(t, len(chatInput.Messages), len(decodedChatInput.Messages), "解码后的 Messages 长度应匹配")
	assert.Equal(t, "", decodedChatInput.ConversationID, "解码后的 ConversationID 应为空，因为它不包含在 JSON 中")
}

// TestChatOutputChunkJSON 测试 ChatOutputChunk 的 JSON 编码和解码
func TestChatOutputChunkJSON(t *testing.T) {
	// 测试正常的回复块
	normalChunk := ChatOutputChunk{
		ContentParts: []ContentPart{
			{
				Type: PartTypeText,
				Text: "Hello, ",
			},
		},
	}

	normalJSON, err := json.MarshalToString(normalChunk)
	require.NoError(t, err, "Marshal 正常的 ChatOutputChunk 不应返回错误")

	expectedNormalJSON := `{"content_parts":[{"type":"text","text":"Hello, "}]}`
	assert.JSONEq(t, expectedNormalJSON, normalJSON, "正常的 ChatOutputChunk 应正确编码为 JSON")

	// 测试带有结束原因的最终块
	finishReason := "stop"
	finalChunk := ChatOutputChunk{
		ContentParts: []ContentPart{
			{
				Type: PartTypeText,
				Text: "world!",
			},
		},
		FinishReason: Some(finishReason),
	}

	finalJSON, err := json.MarshalToString(finalChunk)
	require.NoError(t, err, "Marshal 带有结束原因的 ChatOutputChunk 不应返回错误")

	expectedFinalJSON := `{"content_parts":[{"type":"text","text":"world!"}],"finish_reason":"stop"}`
	assert.JSONEq(t, expectedFinalJSON, finalJSON, "带有结束原因的 ChatOutputChunk 应正确编码为 JSON")

	// 测试带有错误的块
	errorChunk := ChatOutputChunk{
		ContentParts: []ContentPart{
			{
				Type: PartTypeText,
				Text: "",
			},
		},
		Error: assert.AnError,
	}

	errorJSON, err := json.MarshalToString(errorChunk)
	require.NoError(t, err, "Marshal 带有错误的 ChatOutputChunk 不应返回错误")

	// Error 应被省略，因为其 JSON 标签为 "-"
	assert.NotContains(t, errorJSON, "error", "Error 不应出现在 JSON 中")

	var decodedErrorChunk ChatOutputChunk
	err = json.UnmarshalFromString(errorJSON, &decodedErrorChunk)
	require.NoError(t, err, "Unmarshal 带有错误的 ChatOutputChunk JSON 不应返回错误")
	assert.Nil(t, decodedErrorChunk.Error, "解码后的 Error 应为 nil")
}

// TestLLMResponse 测试 LLMResponse 的基本功能
func TestLLMResponse(t *testing.T) {
	// 测试一个包含文本和工具调用的响应
	response := LLMResponse{
		Role: RoleAssistant,
		Content: []ContentPart{
			{
				Type: PartTypeText,
				Text: "我需要查询天气情况",
			},
			{
				Type: PartTypeToolCallRequest,
				ToolCallValues: Some(ToolCallContent{
					Calls: []ToolCall{
						{
							ID:        "call_123",
							Name:      "get_weather",
							Arguments: `{"location": "北京", "unit": "celsius"}`,
						},
					},
				}),
			},
		},
		FullText:     "我需要查询天气情况",
		FinishReason: Some("tool_calls"),
	}

	// 测试 HasToolCalls 方法
	assert.True(t, response.HasToolCalls(), "包含工具调用的响应 HasToolCalls 应返回 true")

	// 测试 GetToolCalls 方法
	toolCalls := response.GetToolCalls()
	assert.Len(t, toolCalls, 1, "应该提取出一个工具调用")
	assert.Equal(t, "call_123", toolCalls[0].ID, "提取的工具调用 ID 应匹配")
	assert.Equal(t, "get_weather", toolCalls[0].Name, "提取的工具调用名称应匹配")

	// 测试 ToInputMessage 方法
	inputMsg := response.ToInputMessage()
	assert.Equal(t, RoleAssistant, inputMsg.Role, "转换后的消息角色应为 assistant")
	assert.Len(t, inputMsg.Content, 2, "转换后的消息应包含所有内容部分")
	assert.Equal(t, response.Content, inputMsg.Content, "转换后的消息内容应与原始响应匹配")

	// 测试 ToUserMessage 方法
	userMsg := response.ToUserMessage()
	assert.Equal(t, RoleUser, userMsg.Role, "ToUserMessage 返回的消息角色应为 user")
	assert.Len(t, userMsg.Content, 1, "ToUserMessage 返回的消息应只包含文本内容")
	assert.Equal(t, PartTypeText, userMsg.Content[0].Type, "ToUserMessage 返回的内容类型应为文本")
	assert.Equal(t, response.FullText, userMsg.Content[0].Text, "ToUserMessage 返回的文本应匹配 FullText")
}

// TestLLMResponseWithoutRole 测试没有显式设置 Role 时的行为
func TestLLMResponseWithoutRole(t *testing.T) {
	response := LLMResponse{
		Content: []ContentPart{
			{
				Type: PartTypeText,
				Text: "测试响应",
			},
		},
		FullText: "测试响应",
	}

	// 测试未设置 Role 时 ToInputMessage 方法使用默认角色
	inputMsg := response.ToInputMessage()
	assert.Equal(t, RoleAssistant, inputMsg.Role, "未设置 Role 时，ToInputMessage 应使用 RoleAssistant 作为默认角色")
}

// TestLLMResponseWithoutToolCalls 测试没有工具调用的情况
func TestLLMResponseWithoutToolCalls(t *testing.T) {
	response := LLMResponse{
		Content: []ContentPart{
			{
				Type: PartTypeText,
				Text: "只有文本内容",
			},
		},
		FullText: "只有文本内容",
	}

	// 测试没有工具调用时 HasToolCalls 方法
	assert.False(t, response.HasToolCalls(), "没有工具调用的响应 HasToolCalls 应返回 false")

	// 测试没有工具调用时 GetToolCalls 方法
	toolCalls := response.GetToolCalls()
	assert.Empty(t, toolCalls, "没有工具调用的响应 GetToolCalls 应返回空切片")
}
