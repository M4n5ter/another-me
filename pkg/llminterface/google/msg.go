package google

import (
	"log/slog"

	json "github.com/json-iterator/go"
	"google.golang.org/genai"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

// ChatInputToGenaiMsgs 将 ChatInput 转换为 genai.Content
func ChatInputToGenaiMsgs(input llminterface.ChatInput) []*genai.Content {
	msgs := make([]*genai.Content, 0, len(input.Messages))
	for _, msg := range input.Messages {
		if len(msg.Content) == 0 {
			continue
		}

		switch msg.Role {
		case llminterface.RoleUser:
			if content := InputMessageToGenaiContent(msg); content != nil {
				msgs = append(msgs, content)
			}
		case llminterface.RoleAssistant:
			if content := InputMessageToGenaiContent(msg); content != nil {
				msgs = append(msgs, content)
			}
		case llminterface.RoleToolResult:
			role := RoleToGenaiRole(msg.Role)
			genaiParts := make([]*genai.Part, 0, len(msg.Content))

			for _, part := range msg.Content {
				genaiParts = append(genaiParts, &genai.Part{
					FunctionResponse: &genai.FunctionResponse{
						ID:   msg.ToolCallID.Unwrap(),
						Name: msg.ToolName.Unwrap(),
						Response: map[string]any{
							"output": part.Text,
						},
					},
				})
			}

			msgs = append(msgs, genai.NewContentFromParts(genaiParts, role))
		} // 不处理 System，genai sdk 在别处接收，而不是 genai.Content 中
	}
	return msgs
}

// InputMessageToGenaiContent 将 InputMessage 转换为 genai.Content
func InputMessageToGenaiContent(input llminterface.InputMessage) *genai.Content {
	genaiParts := make([]*genai.Part, 0, len(input.Content))
	for _, part := range input.Content {
		switch part.Type {
		case llminterface.PartTypeText:
			genaiPart := &genai.Part{
				Text: part.Text,
			}
			genaiParts = append(genaiParts, genaiPart)
		case llminterface.PartTypeImageURL:
			imgData, err := part.ImageURL.UnwrapAsPtr().ToImageBytes()
			if err != nil {
				slog.Error("Failed to convert image url to image bytes", "error", err)
				continue
			}
			genaiPart := &genai.Part{
				InlineData: &genai.Blob{
					Data:     imgData,
					MIMEType: part.ImageURL.UnwrapAsPtr().MimeType(),
				},
			}
			genaiParts = append(genaiParts, genaiPart)
		case llminterface.PartTypeToolCallRequest:
			for _, call := range part.ToolCallValues.UnwrapAsPtr().Calls {
				// call.Arguments 是 JSON 字符串，需要解析为 map[string]any
				args := make(map[string]any)
				err := json.Unmarshal([]byte(call.Arguments), &args)
				if err != nil {
					slog.Error("Failed to unmarshal tool call arguments", "error", err)
					continue
				}

				genaiPart := &genai.Part{
					FunctionCall: &genai.FunctionCall{
						ID:   call.ID,
						Name: call.Name,
						Args: args,
					},
				}
				genaiParts = append(genaiParts, genaiPart)
			}

		}
	}

	// 如果没有有效的 parts，返回 nil 以避免创建空的 Content
	if len(genaiParts) == 0 {
		slog.Warn("No valid parts found for message, skipping", "role", input.Role)
		return nil
	}

	return genai.NewContentFromParts(genaiParts, RoleToGenaiRole(input.Role))
}

// ExtractSystemPromptAsGenaiContent 提取系统提示词并转换为 genai.Content
func ExtractSystemPromptAsGenaiContent(input llminterface.ChatInput) *genai.Content {
	for _, msg := range input.Messages {
		if msg.Role == llminterface.RoleSystem {
			return &genai.Content{
				Parts: []*genai.Part{
					{
						Text: msg.Content[0].Text,
					},
				},
			}
		}
	}
	return nil
}

// RoleToGenaiRole 将 MessageRole 转换为 genai.Role
func RoleToGenaiRole(role llminterface.MessageRole) genai.Role {
	switch role {
	case llminterface.RoleUser, llminterface.RoleToolResult:
		return genai.RoleUser
	case llminterface.RoleAssistant:
		return genai.RoleModel
	default:
		return genai.RoleUser
	}
}
