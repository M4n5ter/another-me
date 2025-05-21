package eino

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/cloudwego/eino/schema"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// ProcessEinoMessageToContentParts 将eino消息转换为内容部分数组，并返回推理内容
func ProcessEinoMessageToContentParts(msg *schema.Message) ([]llminterface.ContentPart, string) {
	var parts []llminterface.ContentPart

	reasoningContent := ""
	if msg == nil {
		return parts, reasoningContent
	}

	for key, value := range msg.Extra {
		if key == "" || value == nil {
			continue
		}

		if strings.Contains(key, "reason") || strings.Contains(key, "think") {
			switch v := value.(type) {
			case string:
				reasoningContent = v
			default:
				reasoningContent = fmt.Sprintf("%v", v)
			}
		}
	}

	// 处理普通内容 (Content 和 MultiContent)
	if msg.Content != "" {
		parts = append(parts, llminterface.ContentPart{
			Type: llminterface.PartTypeText,
			Text: msg.Content,
		})
	} else if msg.MultiContent != nil {
		for _, mcPart := range msg.MultiContent {
			switch mcPart.Type {
			case schema.ChatMessagePartTypeText:
				parts = append(parts, llminterface.ContentPart{
					Type: llminterface.PartTypeText,
					Text: mcPart.Text,
				})
			case schema.ChatMessagePartTypeImageURL:
				if mcPart.ImageURL != nil {
					imageDetail := llminterface.ImageDetailAuto // Default
					if mcPart.ImageURL.Detail != "" {
						switch mcPart.ImageURL.Detail {
						case schema.ImageURLDetailLow:
							imageDetail = llminterface.ImageDetailLow
						case schema.ImageURLDetailHigh:
							imageDetail = llminterface.ImageDetailHigh
						case schema.ImageURLDetailAuto:
							imageDetail = llminterface.ImageDetailAuto
						default:
							slog.Warn("Unknown eino ImageURLDetail, defaulting to auto", "detail", mcPart.ImageURL.Detail)
						}
					}
					parts = append(parts, llminterface.ContentPart{
						Type: llminterface.PartTypeImageURL,
						ImageURL: Some(llminterface.ImageURLContent{
							URL:    mcPart.ImageURL.URL,
							Detail: Some(imageDetail),
						}),
					})
				}
			default:
				slog.Warn("Unsupported eino.ChatMessagePartType in MultiContent.", "type", mcPart.Type)
			}
		}
	}

	return parts, reasoningContent
}
