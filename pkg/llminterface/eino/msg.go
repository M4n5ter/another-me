package eino

import (
	"log/slog"
	"strings"

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

		if m.Role == llminterface.RoleToolResult {
			if m.ToolCallID.IsSome() {
				einoMsg.ToolCallID = m.ToolCallID.Unwrap()
			}
			if m.ToolName.IsSome() {
				einoMsg.Name = m.ToolName.Unwrap() // eino uses 'Name' for tool name in tool messages
			}

			// 工具结果通常是文本，取第一个文本类型的 ContentPart
			foundContent := false
			for _, cp := range m.Content {
				if cp.Type == llminterface.PartTypeText {
					// 对于工具结果，我们假设它是一个完整的文本输出，而不是流片段
					// 直接将其赋值给 einoMsg.Content
					einoMsg.Content = cp.Text
					foundContent = true
					break
				}
			}
			if !foundContent {
				slog.Warn("ToolResult message does not contain a primary text content part or is empty.", "messageRole", m.Role)
				// einoMsg.Content 将保持其零值 (空字符串)，这对于某些模型可能表示空结果
			}
		} else {
			// 处理非 ToolResult 类型的消息 (User, Assistant, System)
			var einoMsgContent string
			var hasContent bool

			switch len(m.Content) {
			case 1:
				cp := m.Content[0]
				parsedParts, isToolCallRequest := ContentPartToEinoChatMessagePart(cp)
				if isToolCallRequest {
					if cp.ToolCallValues.IsSome() {
						einoMsg.ToolCalls = convertToEinoToolCalls(cp.ToolCallValues.Unwrap().Calls)
					}
				} else if len(parsedParts) == 1 && parsedParts[0].Type == schema.ChatMessagePartTypeText {
					einoMsgContent = parsedParts[0].Text
					hasContent = true
				} else if len(parsedParts) > 0 {
					// DeepSeek 不支持 MultiContent。如果单个部分不是文本，这里会有问题。
					// 暂时假设单个非文本部分不会触发 MultiContent 错误，但最佳做法是避免。
					// 或者，如果 eino client 库能将单个非文本部分放入合适的字段（如果 Message 支持的话）。
					// 为了安全，如果单个部分不是文本，我们可能需要警告或只处理文本。
					// 实际上，如果 deepseek 不支持 MultiContent，那么它可能也不支持 Image 等。
					slog.Warn("Single non-text content part found for DeepSeek; it might be ignored or cause issues as MultiContent is disabled.", "partType", parsedParts[0].Type)
					// Fallback: try to get text if any, otherwise content remains empty.
					for _, pp := range parsedParts {
						if pp.Type == schema.ChatMessagePartTypeText {
							einoMsgContent = pp.Text // 只取第一个文本部分（如果有多个 text part 合并的情况，下面 default 会处理）
							hasContent = true
							break
						}
					}
				}
			case 0:
				// 消息内容为空
			default:
				// 多个内容部分，合并文本，收集工具调用
				textContent := strings.Builder{}
				firstText := true
				for _, cp := range m.Content {
					parsedParts, isToolCallRequest := ContentPartToEinoChatMessagePart(cp)
					if isToolCallRequest {
						if cp.ToolCallValues.IsSome() {
							einoMsg.ToolCalls = append(einoMsg.ToolCalls, convertToEinoToolCalls(cp.ToolCallValues.Unwrap().Calls)...)
						}
					} else {
						for _, pp := range parsedParts {
							if pp.Type == schema.ChatMessagePartTypeText {
								if !firstText {
									textContent.WriteString("\n") // 用换行符合并多个文本部分
								}
								textContent.WriteString(pp.Text)
								firstText = false
								hasContent = true
							} else {
								// 忽略非文本、非工具调用的部分，因为 DeepSeek 不支持 MultiContent
								slog.Warn("Ignoring non-text, non-tool-call part in multi-part message for DeepSeek due to no MultiContent support.", "partType", pp.Type)
							}
						}
					}
				}
				if textContent.Len() > 0 {
					einoMsgContent = textContent.String()
				}
			}

			if hasContent {
				einoMsg.Content = einoMsgContent
			}
			einoMsg.MultiContent = nil // 确保 MultiContent 始终为 nil
		}
		msgs = append(msgs, einoMsg)
	}
	return msgs
}

func convertToEinoToolCalls(calls []llminterface.ToolCall) []schema.ToolCall {
	// 初始化为容量为len(calls)的空切片，只添加有效的工具调用
	validEinoSDKToolCalls := make([]schema.ToolCall, 0, len(calls))
	var lastValidID string                // 保存最后一个有效的工具调用ID
	var lastValidNameWithEmptyArgs string // 保存最后一个有ID但参数为空的工具名

	// 第一次遍历: 处理有ID的工具调用
	for _, c := range calls {
		slog.Debug("convertToEinoToolCalls", "call", c)

		// 处理有ID的工具调用
		if c.ID != "" && c.Name != "" {
			validEinoSDKToolCalls = append(validEinoSDKToolCalls, schema.ToolCall{
				ID:   c.ID,
				Type: "function",
				Function: schema.FunctionCall{
					Name:      c.Name,
					Arguments: c.Arguments,
				},
			})

			lastValidID = c.ID

			// 记录参数为空的工具调用
			if c.Arguments == "{}" || c.Arguments == "" {
				lastValidNameWithEmptyArgs = c.Name
			}
		} else if c.Name != "" && lastValidID != "" {
			// 这是一个没有ID但有名称的工具调用，使用上一个有效ID
			// 通常这种情况是由于流式传输中参数是分开发送的

			// 只有当名称相同时才合并（或者未保存过最后有效名称时）
			if lastValidNameWithEmptyArgs == c.Name {
				// 找到最后一个带有此ID的工具调用
				for i := len(validEinoSDKToolCalls) - 1; i >= 0; i-- {
					if validEinoSDKToolCalls[i].ID == lastValidID {
						// 如果前一个工具调用的参数为空，但当前工具调用有参数，则更新
						if (validEinoSDKToolCalls[i].Function.Arguments == "{}" ||
							validEinoSDKToolCalls[i].Function.Arguments == "") &&
							c.Arguments != "" && c.Arguments != "{}" {

							validEinoSDKToolCalls[i].Function.Arguments = c.Arguments
							slog.Info("更新了工具调用参数", "toolID", lastValidID, "toolName", c.Name, "newArgs", c.Arguments)
						}
						break
					}
				}
			} else {
				// 名称不同，作为新的工具调用添加（使用保存的最后一个ID）
				validEinoSDKToolCalls = append(validEinoSDKToolCalls, schema.ToolCall{
					ID:   lastValidID,
					Type: "function",
					Function: schema.FunctionCall{
						Name:      c.Name,
						Arguments: c.Arguments,
					},
				})
				slog.Debug("为没有ID的工具调用使用上一个ID", "toolID", lastValidID, "toolName", c.Name)
			}
		}
	}

	return validEinoSDKToolCalls
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
// 它现在返回一个额外的布尔值，指示是否处理了 ToolCallRequest
func ContentPartToEinoChatMessagePart(contentPart llminterface.ContentPart) ([]schema.ChatMessagePart, bool) {
	switch contentPart.Type {
	case llminterface.PartTypeText:
		return []schema.ChatMessagePart{
			{
				Type: schema.ChatMessagePartTypeText,
				Text: contentPart.Text,
			},
		}, false
	case llminterface.PartTypeImageURL:
		if contentPart.ImageURL.IsNone() {
			return []schema.ChatMessagePart{}, false
		}

		imageURLContent := contentPart.ImageURL.Unwrap()
		parts := make([]schema.ChatMessagePart, 0)
		// According to OpenAI spec, image_url type part can also have a text field for describing the image.
		// However, eino's schema.ChatMessagePartTypeImageURL only has ImageURL *schema.ChatMessageImageURL.
		// If contentPart.Text is present for an image, it's better to send it as a separate text part.
		// For now, we are only processing the ImageURL itself for eino.
		// If contentPart.Text needs to be sent alongside an image for eino, the input llminterface.ContentPart
		// list should already have a separate text part.

		detail := schema.ImageURLDetailAuto // 默认值
		if imageURLContent.Detail.IsSome() {
			// Convert llminterface.ImageURLDetail to schema.ImageURLDetail
			switch imageURLContent.Detail.Unwrap() {
			case llminterface.ImageDetailLow:
				detail = schema.ImageURLDetailLow
			case llminterface.ImageDetailHigh:
				detail = schema.ImageURLDetailHigh
			case llminterface.ImageDetailAuto:
				detail = schema.ImageURLDetailAuto
			default:
				slog.Warn("Unknown llminterface.ImageURLDetail, defaulting to auto for eino.", "detail", imageURLContent.Detail.Unwrap())
			}
		}

		parts = append(parts, schema.ChatMessagePart{
			Type: schema.ChatMessagePartTypeImageURL,
			ImageURL: &schema.ChatMessageImageURL{
				URL:    imageURLContent.URL,
				Detail: detail,
			},
		})

		return parts, false
	case llminterface.PartTypeToolCallRequest:
		// 现在，我们将这个类型标记为特殊处理，但不在这里返回错误或空。
		// 调用方 (ChatInputToEinoMsgs) 将使用这个信息来填充 einoMsg.ToolCalls。
		// 这个函数本身不直接转换它为 ChatMessagePart，因为它的目标是 einoMsg.ToolCalls。
		return nil, true // 返回 nil ChatMessagePart，但标记为 isToolCallRequest = true
	default:
		slog.Error("unknown content part type in ContentPartToEinoChatMessagePart", "type", contentPart.Type)
		return []schema.ChatMessagePart{}, false
	}
}
