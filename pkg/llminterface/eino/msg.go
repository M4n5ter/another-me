package eino

import (
	"log/slog"

	"github.com/cloudwego/eino/schema"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

// ChatInputToEinoMsgs 将 llminterface.ChatInput 转换为 eino 的 []*schema.Message
func ChatInputToEinoMsgs(input llminterface.ChatInput) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(input.Messages))
	for _, m := range input.Messages {
		einoMsg := &schema.Message{
			Role: RoleToEinoRole(m.Role),
		}

		switch len(m.Content) {
		case 1:
			if m.Content[0].Type == llminterface.PartTypeText {
				einoMsg.Content = m.Content[0].Text
			}
		case 0:
			// 如果消息既没有Content也没有MultiContent，根据eino的行为，Content默认为空字符串
			// 这里保持einoMsg.Content和einoMsg.MultiContent都为它们的零值
		default:
			multiContent := make([]schema.ChatMessagePart, 0)
			for _, cp := range m.Content {
				multiContent = append(multiContent, ContentPartToEinoChatMessagePart(cp)...)
			}
			einoMsg.MultiContent = multiContent
		}

		msgs = append(msgs, einoMsg)
	}
	return msgs
}

// RoleToEinoRole 将 llminterface.MessageRole 转换为 eino 的 schema.RoleType
func RoleToEinoRole(role llminterface.MessageRole) schema.RoleType {
	switch role {
	case llminterface.RoleUser:
		return schema.User
	case llminterface.RoleAssistant:
		return schema.Assistant
	case llminterface.RoleSystem:
		return schema.System
	case llminterface.RoleToolResult:
		return schema.Tool
	default:
		return schema.User
	}
}

// ContentPartToEinoChatMessagePart 将 llminterface.ContentPart 转换为 eino 的 []schema.ChatMessagePart
func ContentPartToEinoChatMessagePart(contentPart llminterface.ContentPart) []schema.ChatMessagePart {
	switch contentPart.Type {
	case llminterface.PartTypeText:
		return []schema.ChatMessagePart{
			{
				Type: schema.ChatMessagePartTypeText,
				Text: contentPart.Text,
			},
		}
	case llminterface.PartTypeImageURL:
		if contentPart.ImageURL.IsNone() {
			return []schema.ChatMessagePart{}
		}

		imageURLContent := contentPart.ImageURL.Unwrap()
		parts := make([]schema.ChatMessagePart, 0)
		if contentPart.Text != "" {
			parts = append(parts, schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeText,
				Text: contentPart.Text,
			})
		}

		detail := schema.ImageURLDetailAuto // 默认值
		if imageURLContent.Detail.IsSome() {
			detail = schema.ImageURLDetail(imageURLContent.Detail.Unwrap())
		}

		parts = append(parts, schema.ChatMessagePart{
			Type: schema.ChatMessagePartTypeImageURL,
			ImageURL: &schema.ChatMessageImageURL{
				URL:    imageURLContent.URL,
				Detail: detail,
			},
		})

		return parts
	default:
		slog.Error("unknown content part type", "type", contentPart.Type)
		return []schema.ChatMessagePart{}
	}
}
