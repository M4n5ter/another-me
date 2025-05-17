package llminterface

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	textJSON, err := json.Marshal(textPart)
	require.NoError(t, err, "Marshal 文本类型的 ContentPart 不应返回错误")

	expectedTextJSON := `{"type":"text","text":"Hello, world!"}`
	assert.JSONEq(t, expectedTextJSON, string(textJSON), "文本类型的 ContentPart 应正确编码为 JSON")

	var decodedTextPart ContentPart
	err = json.Unmarshal(textJSON, &decodedTextPart)
	require.NoError(t, err, "Unmarshal 文本类型的 JSON 不应返回错误")
	assert.Equal(t, textPart, decodedTextPart, "解码后的 ContentPart 应与原始对象相同")

	// 测试图像 URL 类型的 ContentPart
	imagePart := ContentPart{
		Type: PartTypeImageURL,
		ImageURL: &ImageURLContent{
			URL:    "https://example.com/image.jpg",
			Detail: ImageDetailHigh,
		},
	}

	imageJSON, err := json.Marshal(imagePart)
	require.NoError(t, err, "Marshal 图像 URL 类型的 ContentPart 不应返回错误")

	expectedImageJSON := `{"type":"image_url","image_url":{"url":"https://example.com/image.jpg","detail":"high"}}`
	assert.JSONEq(t, expectedImageJSON, string(imageJSON), "图像 URL 类型的 ContentPart 应正确编码为 JSON")

	var decodedImagePart ContentPart
	err = json.Unmarshal(imageJSON, &decodedImagePart)
	require.NoError(t, err, "Unmarshal 图像 URL 类型的 JSON 不应返回错误")
	assert.Equal(t, imagePart.Type, decodedImagePart.Type, "解码后的 Type 应匹配")
	assert.Equal(t, imagePart.ImageURL.URL, decodedImagePart.ImageURL.URL, "解码后的 URL 应匹配")
	assert.Equal(t, imagePart.ImageURL.Detail, decodedImagePart.ImageURL.Detail, "解码后的 Detail 应匹配")
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
				ImageURL: &ImageURLContent{
					URL:    "https://example.com/image.jpg",
					Detail: ImageDetailHigh,
				},
			},
		},
	}

	messageJSON, err := json.Marshal(message)
	require.NoError(t, err, "Marshal InputMessage 不应返回错误")

	var decodedMessage InputMessage
	err = json.Unmarshal(messageJSON, &decodedMessage)
	require.NoError(t, err, "Unmarshal InputMessage JSON 不应返回错误")

	assert.Equal(t, message.Role, decodedMessage.Role, "解码后的 Role 应匹配")
	assert.Equal(t, len(message.Content), len(decodedMessage.Content), "解码后的 Content 长度应匹配")
	assert.Equal(t, message.Content[0].Type, decodedMessage.Content[0].Type, "第一个 Content 的 Type 应匹配")
	assert.Equal(t, message.Content[0].Text, decodedMessage.Content[0].Text, "第一个 Content 的 Text 应匹配")
	assert.Equal(t, message.Content[1].Type, decodedMessage.Content[1].Type, "第二个 Content 的 Type 应匹配")
	assert.Equal(t, message.Content[1].ImageURL.URL, decodedMessage.Content[1].ImageURL.URL, "第二个 Content 的 URL 应匹配")
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
						ImageURL: &ImageURLContent{
							URL:    "https://example.com/image.jpg",
							Detail: ImageDetailHigh,
						},
					},
				},
			},
		},
		ConversationID: "test-conversation",
	}

	chatInputJSON, err := json.Marshal(chatInput)
	require.NoError(t, err, "Marshal ChatInput 不应返回错误")

	// ConversationID 应被省略，因为其 JSON 标签为 "-"
	assert.NotContains(t, string(chatInputJSON), "conversation_id", "ConversationID 不应出现在 JSON 中")
	assert.NotContains(t, string(chatInputJSON), "test-conversation", "ConversationID 的值不应出现在 JSON 中")

	var decodedChatInput ChatInput
	err = json.Unmarshal(chatInputJSON, &decodedChatInput)
	require.NoError(t, err, "Unmarshal ChatInput JSON 不应返回错误")

	assert.Equal(t, len(chatInput.Messages), len(decodedChatInput.Messages), "解码后的 Messages 长度应匹配")
	assert.Equal(t, "", decodedChatInput.ConversationID, "解码后的 ConversationID 应为空，因为它不包含在 JSON 中")
}

// TestChatOutputChunkJSON 测试 ChatOutputChunk 的 JSON 编码和解码
func TestChatOutputChunkJSON(t *testing.T) {
	// 测试正常的回复块
	normalChunk := ChatOutputChunk{
		TextDelta: "Hello, ",
	}

	normalJSON, err := json.Marshal(normalChunk)
	require.NoError(t, err, "Marshal 正常的 ChatOutputChunk 不应返回错误")

	expectedNormalJSON := `{"text_delta":"Hello, "}`
	assert.JSONEq(t, expectedNormalJSON, string(normalJSON), "正常的 ChatOutputChunk 应正确编码为 JSON")

	// 测试带有结束原因的最终块
	finishReason := "stop"
	finalChunk := ChatOutputChunk{
		TextDelta:    "world!",
		FinishReason: &finishReason,
	}

	finalJSON, err := json.Marshal(finalChunk)
	require.NoError(t, err, "Marshal 带有结束原因的 ChatOutputChunk 不应返回错误")

	expectedFinalJSON := `{"text_delta":"world!","finish_reason":"stop"}`
	assert.JSONEq(t, expectedFinalJSON, string(finalJSON), "带有结束原因的 ChatOutputChunk 应正确编码为 JSON")

	// 测试带有错误的块
	errorChunk := ChatOutputChunk{
		TextDelta: "",
		Error:     assert.AnError,
	}

	errorJSON, err := json.Marshal(errorChunk)
	require.NoError(t, err, "Marshal 带有错误的 ChatOutputChunk 不应返回错误")

	// Error 应被省略，因为其 JSON 标签为 "-"
	assert.NotContains(t, string(errorJSON), "error", "Error 不应出现在 JSON 中")

	var decodedErrorChunk ChatOutputChunk
	err = json.Unmarshal(errorJSON, &decodedErrorChunk)
	require.NoError(t, err, "Unmarshal 带有错误的 ChatOutputChunk JSON 不应返回错误")
	assert.Nil(t, decodedErrorChunk.Error, "解码后的 Error 应为 nil")
}
