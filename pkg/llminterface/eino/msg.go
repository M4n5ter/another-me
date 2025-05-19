package eino

import (
	"github.com/cloudwego/eino/schema"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

// ChatInputToEinoMsgs 将 llminterface.ChatInput 转换为 eino 的 []*schema.Message
func ChatInputToEinoMsgs(input llminterface.ChatInput) []*schema.Message {
	msgs := make([]*schema.Message, 0, len(input.Messages))
	for _, m := range input.Messages {
		if len(m.Content) == 0 {
			continue
		}

		einoMsg := &schema.Message{
			Role: RoleToEinoRole(m.Role),
		}

		switch m.Role {
		case llminterface.RoleSystem:
			// System prompt
			if len(m.Content) > 0 && m.Content[0].Type == llminterface.PartTypeText {
				einoMsg.Content = m.Content[0].Text
			}
		case llminterface.RoleUser:
			// 用户消息：如果是单个文本部分，则直接放入 Content；否则（单个图片、或多个混合部分）构建 MultiContent
			if len(m.Content) == 1 && m.Content[0].Type == llminterface.PartTypeText {
				einoMsg.Content = m.Content[0].Text
			} else if len(m.Content) > 0 {
				einoMsg.MultiContent = buildEinoMultiContent(m.Content)
			}
		case llminterface.RoleAssistant:
			// 助手消息：可能包含文本、图片和工具调用请求
			textAndImageParts := make([]llminterface.ContentPart, 0)
			var allToolCalls []llminterface.ToolCall

			for _, cp := range m.Content {
				switch cp.Type {
				case llminterface.PartTypeText, llminterface.PartTypeImageURL:
					textAndImageParts = append(textAndImageParts, cp)
				case llminterface.PartTypeToolCallRequest:
					if cp.ToolCallValues.IsSome() {
						if allToolCalls == nil { // 延迟初始化
							allToolCalls = make([]llminterface.ToolCall, 0)
						}
						allToolCalls = append(allToolCalls, cp.ToolCallValues.Unwrap().Calls...)
					}
				}
			}

			// 处理文本和图片部分
			if len(textAndImageParts) == 1 && textAndImageParts[0].Type == llminterface.PartTypeText {
				einoMsg.Content = textAndImageParts[0].Text
			} else if len(textAndImageParts) > 0 {
				einoMsg.MultiContent = buildEinoMultiContent(textAndImageParts)
			}

			// 处理工具调用请求部分
			if len(allToolCalls) > 0 {
				einoMsg.ToolCalls = convertToEinoToolCalls(allToolCalls)
			}
		case llminterface.RoleToolResult:
			einoMsg.ToolCallID = m.ToolCallID.Unwrap()
			einoMsg.Name = m.ToolName.Unwrap()

			// 内容部分：如果是单个文本，则放入 Content；否则（单个图片或多个部分）构建 MultiContent
			if len(m.Content) == 1 && m.Content[0].Type == llminterface.PartTypeText {
				einoMsg.Content = m.Content[0].Text
			} else if len(m.Content) > 0 { // 包括单个图片或多个部分的情况
				einoMsg.MultiContent = buildEinoMultiContent(m.Content)
			}
		default:
			// 对于未明确处理的角色，如果存在内容，尝试作为 MultiContent 处理
			if len(m.Content) > 0 {
				einoMsg.MultiContent = buildEinoMultiContent(m.Content)
			}
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

// buildEinoMultiContent 将 llminterface.ContentPart 转换为 eino 的 []schema.ChatMessagePart，主要处理文本和图片类型。
func buildEinoMultiContent(contentParts []llminterface.ContentPart) []schema.ChatMessagePart {
	multiContent := make([]schema.ChatMessagePart, 0, len(contentParts))
	for _, cp := range contentParts {
		switch cp.Type {
		case llminterface.PartTypeText:
			multiContent = append(multiContent, schema.ChatMessagePart{Type: schema.ChatMessagePartTypeText, Text: cp.Text})
		case llminterface.PartTypeImageURL:
			if cp.ImageURL.IsSome() {
				imgURL := cp.ImageURL.Unwrap()
				multiContent = append(multiContent, schema.ChatMessagePart{
					Type: schema.ChatMessagePartTypeImageURL,
					ImageURL: &schema.ChatMessageImageURL{
						URL:    imgURL.URL,
						Detail: schema.ImageURLDetail(imgURL.Detail.TakeOr(llminterface.ImageDetailAuto)),
					},
				})
			}
		}
	}
	return multiContent
}

func convertToEinoToolCalls(calls []llminterface.ToolCall) []schema.ToolCall {
	einoToolCalls := make([]schema.ToolCall, 0, len(calls))
	for _, c := range calls {
		einoToolCalls = append(einoToolCalls, schema.ToolCall{
			ID:   c.ID,
			Type: "function",
			Function: schema.FunctionCall{
				Name:      c.Name,
				Arguments: c.Arguments,
			},
		})
	}

	return einoToolCalls
}
