package openai

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	json "github.com/json-iterator/go"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// OpenAIChatAdapter 是 OpenAI API 的适配器
type OpenAIChatAdapter struct {
	client *openai.Client
	config *OpenAIAdapterConfig
	logger *slog.Logger
}

// OpenAIAdapterConfig 是 OpenAI 适配器的配置
type OpenAIAdapterConfig struct {
	Model           string
	MaxTokens       Option[int64]
	Temperature     Option[float64]
	TopP            Option[float64]
	Registry        *toolcore.Registry
	Tools           Option[[]openai.ChatCompletionToolParam]
	ReasoningConfig Option[OpenAIReasoningConfig]
}

// OpenAIReasoningConfig 是 OpenAI 推理模式的配置
type OpenAIReasoningConfig struct {
	// 推理努力程度，可以是 "low", "medium", "high"
	// 仅适用于 o1 系列模型
	Effort Option[string]
}

// ToOpenAIReasoningEffort 将 OpenAIReasoningConfig 转换为 shared.ReasoningEffort
func (c *OpenAIReasoningConfig) ToOpenAIReasoningEffort() *shared.ReasoningEffort {
	if c.Effort.IsNone() {
		return nil
	}

	var effort shared.ReasoningEffort
	switch c.Effort.Unwrap() {
	case "low":
		effort = shared.ReasoningEffortLow
	case "medium":
		effort = shared.ReasoningEffortMedium
	case "high":
		effort = shared.ReasoningEffortHigh
	default:
		effort = shared.ReasoningEffortMedium // 默认使用 medium
	}

	return &effort
}

// NewOpenAIChatAdapter 创建一个新的 OpenAI 聊天适配器
func NewOpenAIChatAdapter(apiKey string, baseURL Option[string], config *OpenAIAdapterConfig) *OpenAIChatAdapter {
	if config == nil {
		config = &OpenAIAdapterConfig{
			Model: "gpt-4o",
		}
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}

	if baseURL.IsSome() {
		opts = append(opts, option.WithBaseURL(baseURL.Unwrap()))
	}

	client := openai.NewClient(opts...)

	return &OpenAIChatAdapter{
		client: &client,
		config: config,
		logger: slog.Default().WithGroup("openai_adapter"),
	}
}

var _ llminterface.ChatAdapter = (*OpenAIChatAdapter)(nil)

// Chat 实现 llminterface.ChatAdapter 接口的 Chat 方法
func (o *OpenAIChatAdapter) Chat(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error) {
	if o.config.Registry != nil {
		err := o.RegisterTools(ctx, o.config.Registry)
		if err != nil {
			return nil, err
		}
	}

	openaiMsgs, err := o.convertChatInputToOpenAIMsgs(input)
	if err != nil {
		o.logger.Error("failed to convert chat input to openai msgs", "error", err)
		return nil, fmt.Errorf("failed to convert chat input to openai msgs: %w", err)
	}

	outputChan := make(chan llminterface.ChatOutputChunk, 10)

	go func() {
		defer close(outputChan)

		// 构建请求参数
		params := openai.ChatCompletionNewParams{
			Model:    o.config.Model,
			Messages: openaiMsgs,
		}

		// 设置可选参数
		if o.config.MaxTokens.IsSome() {
			params.MaxTokens = openai.Int(o.config.MaxTokens.Unwrap())
		}
		if o.config.Temperature.IsSome() {
			params.Temperature = openai.Float(o.config.Temperature.Unwrap())
		}
		if o.config.TopP.IsSome() {
			params.TopP = openai.Float(o.config.TopP.Unwrap())
		}
		if o.config.Tools.IsSome() {
			params.Tools = o.config.Tools.Unwrap()
		}
		// 设置推理参数
		if o.config.ReasoningConfig.IsSome() {
			config := o.config.ReasoningConfig.Unwrap()
			if reasoningEffort := config.ToOpenAIReasoningEffort(); reasoningEffort != nil {
				params.ReasoningEffort = *reasoningEffort
			}
		}

		// 创建流式响应
		stream := o.client.Chat.Completions.NewStreaming(ctx, params)

		for stream.Next() {
			chunk := stream.Current()
			outputChunk := o.convertOpenAIChunkToOutputChunk(chunk)
			outputChan <- outputChunk
		}

		if err := stream.Err(); err != nil {
			o.logger.Error("openai streaming response error", "error", err)
			outputChan <- llminterface.ChatOutputChunk{
				Error: fmt.Errorf("openai streaming response error: %w", err),
			}
		}
	}()

	return outputChan, nil
}

// ProduceJSON 实现 llminterface.ChatAdapter 接口的 ProduceJSON 方法
func (o *OpenAIChatAdapter) ProduceJSON(ctx context.Context, input llminterface.ChatInput, jsonSchema Option[schema.Schema]) (string, error) {
	openaiMsgs, err := o.convertChatInputToOpenAIMsgs(input)
	if err != nil {
		o.logger.Error("failed to convert chat input to openai msgs", "error", err)
		return "", fmt.Errorf("failed to convert chat input to openai msgs: %w", err)
	}

	// 构建请求参数
	params := openai.ChatCompletionNewParams{
		Model:    o.config.Model,
		Messages: openaiMsgs,
	}

	if jsonSchema.IsSome() {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: openai.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        jsonSchema.Unwrap().Title,
					Description: openai.String(jsonSchema.Unwrap().Description),
					Schema:      jsonSchema.Unwrap(),
					Strict:      openai.Bool(true),
				},
			},
		}
	} else {
		params.ResponseFormat = openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONObject: &openai.ResponseFormatJSONObjectParam{},
		}
	}

	// 设置其他可选参数
	if o.config.MaxTokens.IsSome() {
		params.MaxTokens = openai.Int(o.config.MaxTokens.Unwrap())
	}
	if o.config.Temperature.IsSome() {
		params.Temperature = openai.Float(o.config.Temperature.Unwrap())
	}
	// 设置推理参数
	if o.config.ReasoningConfig.IsSome() {
		config := o.config.ReasoningConfig.Unwrap()
		if reasoningEffort := config.ToOpenAIReasoningEffort(); reasoningEffort != nil {
			params.ReasoningEffort = *reasoningEffort
		}
	}

	// 发起请求
	response, err := o.client.Chat.Completions.New(ctx, params)
	if err != nil {
		o.logger.Error("failed to request openai", "error", err)
		return "", fmt.Errorf("failed to request openai: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no choices in openai response")
	}

	return response.Choices[0].Message.Content, nil
}

// RegisterTools 实现 llminterface.ChatAdapter 接口的 RegisterTools 方法
func (o *OpenAIChatAdapter) RegisterTools(ctx context.Context, registry *toolcore.Registry) error {
	tools := registry.GetAll()
	if len(tools) == 0 {
		return nil
	}

	openaiTools := make([]openai.ChatCompletionToolParam, 0, len(tools))
	for _, tool := range tools {
		toolSchema, err := tool.Schema(ctx)
		if err != nil {
			o.logger.Error("failed to get tool schema", "tool", toolSchema.Name, "error", err)
			return fmt.Errorf("failed to get tool schema: %w", err)
		}

		openaiTool := o.convertToolSchemaToOpenAITool(&toolSchema)
		openaiTools = append(openaiTools, openaiTool)
	}

	o.config.Tools = Some(openaiTools)
	return nil
}

// GetFrameworkName 实现 llminterface.ChatAdapter 接口的 GetFrameworkName 方法
func (o *OpenAIChatAdapter) GetFrameworkName() string {
	return "openai-go"
}

// convertChatInputToOpenAIMsgs 将内部消息格式转换为 OpenAI 消息格式
func (o *OpenAIChatAdapter) convertChatInputToOpenAIMsgs(input llminterface.ChatInput) ([]openai.ChatCompletionMessageParamUnion, error) {
	messages := make([]openai.ChatCompletionMessageParamUnion, 0, len(input.Messages))

	for _, msg := range input.Messages {
		content := o.convertContentPartsToString(msg.Content)

		switch msg.Role {
		case llminterface.RoleSystem:
			messages = append(messages, openai.ChatCompletionMessageParamUnion{
				OfSystem: &openai.ChatCompletionSystemMessageParam{
					Content: openai.ChatCompletionSystemMessageParamContentUnion{
						OfString: openai.String(content),
					},
				},
			})
		case llminterface.RoleUser:
			messages = append(messages, openai.ChatCompletionMessageParamUnion{
				OfUser: &openai.ChatCompletionUserMessageParam{
					Content: openai.ChatCompletionUserMessageParamContentUnion{
						OfString: openai.String(content),
					},
				},
			})
		case llminterface.RoleAssistant:
			assistantMsg := &openai.ChatCompletionAssistantMessageParam{}

			// 设置文本内容（如果有）
			if content != "" {
				assistantMsg.Content = openai.ChatCompletionAssistantMessageParamContentUnion{
					OfString: openai.String(content),
				}
			}

			// 检查是否包含工具调用
			toolCalls := o.extractToolCallsFromContentParts(msg.Content)
			if len(toolCalls) > 0 {
				assistantMsg.ToolCalls = toolCalls
			}

			messages = append(messages, openai.ChatCompletionMessageParamUnion{
				OfAssistant: assistantMsg,
			})
		case llminterface.RoleToolResult:
			// 工具结果消息必须包含 tool_call_id
			if msg.ToolCallID.IsNone() {
				return nil, fmt.Errorf("tool result message missing tool_call_id")
			}

			messages = append(messages, openai.ChatCompletionMessageParamUnion{
				OfTool: &openai.ChatCompletionToolMessageParam{
					Content: openai.ChatCompletionToolMessageParamContentUnion{
						OfString: openai.String(content),
					},
					ToolCallID: msg.ToolCallID.Unwrap(),
				},
			})
		default:
			return nil, fmt.Errorf("unsupported message role: %s", msg.Role)
		}
	}

	return messages, nil
}

// convertContentPartsToString 将内容部分转换为字符串
func (o *OpenAIChatAdapter) convertContentPartsToString(parts []llminterface.ContentPart) string {
	var result string
	for _, part := range parts {
		if part.Type == llminterface.PartTypeText {
			result += part.Text
		}
	}
	return result
}

// extractToolCallsFromContentParts 从内容部分中提取工具调用
func (o *OpenAIChatAdapter) extractToolCallsFromContentParts(parts []llminterface.ContentPart) []openai.ChatCompletionMessageToolCallParam {
	var toolCalls []openai.ChatCompletionMessageToolCallParam

	for _, part := range parts {
		if part.Type == llminterface.PartTypeToolCallRequest && part.ToolCallValues.IsSome() {
			toolCallContent := part.ToolCallValues.Unwrap()
			for _, call := range toolCallContent.Calls {
				toolCalls = append(toolCalls, openai.ChatCompletionMessageToolCallParam{
					ID: call.ID,
					Function: openai.ChatCompletionMessageToolCallFunctionParam{
						Name:      call.Name,
						Arguments: call.Arguments,
					},
				})
			}
		}
	}

	return toolCalls
}

// convertOpenAIChunkToOutputChunk 将 OpenAI 块转换为输出块
func (o *OpenAIChatAdapter) convertOpenAIChunkToOutputChunk(chunk openai.ChatCompletionChunk) llminterface.ChatOutputChunk {
	if len(chunk.Choices) == 0 {
		return llminterface.ChatOutputChunk{}
	}

	choice := chunk.Choices[0]

	// 创建内容部分
	var contentParts []llminterface.ContentPart
	outputChunk := llminterface.ChatOutputChunk{}

	// 处理推理内容（如果存在）
	// 注意：真正的OpenAI 模型不会直接暴露推理内容
	// 但是对于DeepSeek等兼容模型，可能会通过reasoning_content字段返回
	if choice.Delta.Content != "" {
		contentParts = append(contentParts, llminterface.ContentPart{
			Type: llminterface.PartTypeText,
			Text: choice.Delta.Content,
		})
	}

	//nolint:nestif // 这里需要嵌套if，没办法
	if extraChoice := chunk.JSON.Choices; extraChoice.Valid() {
		extraChoiceStr := extraChoice.Raw()
		var extraChoiceMap []map[string]any
		err := json.UnmarshalFromString(extraChoiceStr, &extraChoiceMap)
		if err != nil {
			o.logger.Error("parse extra choice failed", "error", err)
			return outputChunk
		}

		if len(extraChoiceMap) > 0 {
			if delta, ok := extraChoiceMap[0]["delta"]; ok {
				deltaMap, ok := delta.(map[string]any)
				if ok {
					if reasoningContent, ok := deltaMap["reasoning_content"]; ok {
						if reasoningContentStr, ok := reasoningContent.(string); ok && reasoningContentStr != "" {
							outputChunk.Reasoning = Some(reasoningContentStr)
						}
					} else {
						// 看看有没有跟思考相关的字段
						for key, value := range deltaMap {
							if strings.Contains(key, "reason") || strings.Contains(key, "think") {
								if reasoningContentStr, ok := value.(string); ok && reasoningContentStr != "" {
									outputChunk.Reasoning = Some(reasoningContentStr)
								}
							}
						}
					}
				}
			}
		}
	}

	// 处理工具调用
	if len(choice.Delta.ToolCalls) > 0 {
		toolCalls := make([]llminterface.ToolCall, 0, len(choice.Delta.ToolCalls))
		for _, tc := range choice.Delta.ToolCalls {
			toolCalls = append(toolCalls, llminterface.ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}

		contentParts = append(contentParts, llminterface.ContentPart{
			Type: llminterface.PartTypeToolCallRequest,
			ToolCallValues: Some(llminterface.ToolCallContent{
				Calls: toolCalls,
			}),
		})
	}

	outputChunk.ContentParts = contentParts

	// 设置结束原因
	if choice.FinishReason != "" {
		outputChunk.FinishReason = Some(choice.FinishReason)
	}

	return outputChunk
}

// convertToolSchemaToOpenAITool 将 toolcore.ToolSchema 转换为 OpenAI 工具参数
func (o *OpenAIChatAdapter) convertToolSchemaToOpenAITool(toolSchema *toolcore.ToolSchema) openai.ChatCompletionToolParam {
	// 使用 toolcore 的转换功能
	rawSchema := toolcore.ConvertParamsToRawSchema(toolSchema.InputParameters, toolSchema.Name, "")

	// 获取工具描述
	var description string
	if len(toolSchema.Descriptions) > 0 {
		// 使用第一个可用的描述
		for _, desc := range toolSchema.Descriptions {
			description = desc
			break
		}
	}

	return openai.ChatCompletionToolParam{
		Function: shared.FunctionDefinitionParam{
			Name:        toolSchema.Name,
			Description: openai.String(description),
			Parameters:  rawSchema,
		},
	}
}
