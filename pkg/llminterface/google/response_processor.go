package google

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"
	"google.golang.org/genai"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// ProcessGoogleResponseToChatOutputChunk 处理 google genai 的响应并转换为 ChatOutputChunk
func ProcessGoogleResponseToChatOutputChunk(response *genai.GenerateContentResponse) llminterface.ChatOutputChunk {
	if len(response.Candidates) == 0 {
		return llminterface.ChatOutputChunk{
			Error: fmt.Errorf("no candidates in response"),
		}
	}

	answer := response.Candidates[0]

	return GenaiCandidateToChatOutputChunk(answer)
}

// GenaiCandidateToChatOutputChunk 处理 google genai 的响应并转换为 ChatOutputChunk
func GenaiCandidateToChatOutputChunk(candidate *genai.Candidate) llminterface.ChatOutputChunk {
	chunk := llminterface.ChatOutputChunk{}

	if len(candidate.Content.Parts) == 0 {
		return chunk
	}

	if candidate.FinishReason != genai.FinishReasonUnspecified {
		fr := string(candidate.FinishReason)
		if fr != "" {
			chunk.FinishReason = Some(fr)
		}
	}

	chunk.ContentParts = make([]llminterface.ContentPart, 0, len(candidate.Content.Parts))
	for _, part := range candidate.Content.Parts {

		if part.Thought && part.Text != "" {
			chunk.Reasoning = Some(part.Text)
		}

		if part.FunctionCall != nil {
			// part.FunctionCall.Args 已经是 map[string]any，直接序列化为 JSON 字符串
			arguments, err := json.MarshalToString(part.FunctionCall.Args)
			if err != nil {
				slog.Error("Failed to marshal function call arguments", "error", err)
				continue
			}

			chunk.ContentParts = append(chunk.ContentParts, llminterface.ContentPart{
				Type: llminterface.PartTypeToolCallRequest,
				ToolCallValues: Some(llminterface.ToolCallContent{
					Calls: []llminterface.ToolCall{
						{
							ID:        uuid.NewString(), // gemini 的 function call 不给 id
							Name:      part.FunctionCall.Name,
							Arguments: arguments,
						},
					},
				}),
			})
		}

		if !part.Thought && part.Text != "" {
			chunk.ContentParts = append(chunk.ContentParts, llminterface.ContentPart{
				Type: llminterface.PartTypeText,
				Text: part.Text,
			})
		}
	}

	chunkJSON, err := json.MarshalToString(chunk)
	if err != nil {
		slog.Error("Failed to marshal chunk", "error", err)
	}
	slog.Debug("Chunk", "chunk", chunkJSON)

	return chunk
}
