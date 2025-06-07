package anthropic

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// AnthropicAdapter 实现 ChatAdapter 接口
type AnthropicAdapter struct {
	client   *anthropic.Client
	config   *AnthropicAdapterConfig
	logger   *slog.Logger
	registry Option[*toolcore.Registry]
}

// AnthropicAdapterConfig 配置 AnthropicAdapter 的参数
type AnthropicAdapterConfig struct {
	Model       anthropic.Model                    // Claude 模型名称
	MaxTokens   int64                              // 最大生成令牌数
	Temperature Option[float64]                    // 温度参数，控制输出随机性
	TopP        Option[float64]                    // 核采样参数
	Registry    *toolcore.Registry                 // 工具注册表
	Tools       Option[[]anthropic.ToolUnionParam] // 工具列表
}

// NewAnthropicAdapter 创建一个新的 Anthropic Chat Adapter
func NewAnthropicAdapter(apiKey string, baseURL Option[string], config *AnthropicAdapterConfig) *AnthropicAdapter {
	if config == nil {
		config = &AnthropicAdapterConfig{
			Model:       anthropic.ModelClaudeSonnet4_20250514,
			MaxTokens:   4096,
			Temperature: None[float64](),
			TopP:        None[float64](),
		}
	}

	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}

	if baseURL.IsSome() {
		opts = append(opts, option.WithBaseURL(baseURL.Unwrap()))
	}

	client := anthropic.NewClient(opts...)

	return &AnthropicAdapter{
		client:   &client,
		config:   config,
		logger:   slog.Default().WithGroup("anthropic_adapter"),
		registry: None[*toolcore.Registry](),
	}
}

// 接口实现检查
var _ llminterface.ChatAdapter = (*AnthropicAdapter)(nil)

// Chat 实现 ChatAdapter 接口的 Chat 方法
func (a *AnthropicAdapter) Chat(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error) {
	if a.config.Registry != nil {
		err := a.RegisterTools(ctx, a.config.Registry)
		if err != nil {
			return nil, err
		}
	}

	anthropicMsgs, err := a.convertChatInputToAnthropicMsgs(input)
	if err != nil {
		return nil, fmt.Errorf("转换消息失败: %w", err)
	}

	params := anthropic.MessageNewParams{
		MaxTokens: a.config.MaxTokens,
		Messages:  anthropicMsgs,
		Model:     a.config.Model,
	}

	// 添加可选参数
	if a.config.Temperature.IsSome() {
		params.Temperature = anthropic.Float(a.config.Temperature.Unwrap())
	}
	if a.config.TopP.IsSome() {
		params.TopP = anthropic.Float(a.config.TopP.Unwrap())
	}
	if a.config.Tools.IsSome() {
		params.Tools = a.config.Tools.Unwrap()
	}

	// 创建流式响应
	stream := a.client.Messages.NewStreaming(ctx, params)

	// 创建响应通道
	outputChan := make(chan llminterface.ChatOutputChunk)

	// 启动协程处理流式响应
	go func() {
		defer close(outputChan)
		defer func() {
			if err := stream.Close(); err != nil {
				a.logger.Error("关闭流式响应失败", "error", err)
			}
		}()

		var currentToolCallID string
		var currentToolName string
		var currentToolArgs strings.Builder

		for stream.Next() {
			event := stream.Current()

			switch eventVariant := event.AsAny().(type) {
			case anthropic.ContentBlockStartEvent:
				// 处理内容块开始事件
				if contentVariant, ok := eventVariant.ContentBlock.AsAny().(anthropic.ToolUseBlock); ok {
					currentToolCallID = contentVariant.ID
					currentToolName = contentVariant.Name
					currentToolArgs.Reset()
				}

			case anthropic.ContentBlockDeltaEvent:
				// 处理内容增量事件
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.TextDelta:
					// 文本增量
					chunk := llminterface.ChatOutputChunk{
						ContentParts: []llminterface.ContentPart{
							{
								Type: llminterface.PartTypeText,
								Text: deltaVariant.Text,
							},
						},
					}
					outputChan <- chunk

				case anthropic.InputJSONDelta:
					// 工具调用参数增量
					currentToolArgs.WriteString(deltaVariant.PartialJSON)
				}

			case anthropic.ContentBlockStopEvent:
				// 内容块结束事件 - 如果是工具调用，发送完整的工具调用请求
				if currentToolCallID != "" && currentToolName != "" {
					chunk := llminterface.ChatOutputChunk{
						ContentParts: []llminterface.ContentPart{
							{
								Type: llminterface.PartTypeToolCallRequest,
								ToolCallValues: Some(llminterface.ToolCallContent{
									Calls: []llminterface.ToolCall{
										{
											ID:        currentToolCallID,
											Name:      currentToolName,
											Arguments: currentToolArgs.String(),
										},
									},
								}),
							},
						},
					}
					outputChan <- chunk

					// 重置状态
					currentToolCallID = ""
					currentToolName = ""
					currentToolArgs.Reset()
				}

			case anthropic.MessageStopEvent:
				// 消息结束事件
				chunk := llminterface.ChatOutputChunk{
					FinishReason: Some("stop"),
				}
				outputChan <- chunk
				return
			}
		}

		// 检查流错误
		if stream.Err() != nil {
			chunk := llminterface.ChatOutputChunk{
				Error: stream.Err(),
			}
			outputChan <- chunk
		}
	}()

	return outputChan, nil
}

// ProduceJSON 实现 ChatAdapter 接口的 ProduceJSON 方法
//
// TODO: 系统提示词需要优化
func (a *AnthropicAdapter) ProduceJSON(ctx context.Context, input llminterface.ChatInput, jsonSchema Option[schema.Schema]) (string, error) {
	// Anthropic 不直接支持 JSON 模式，需要在系统提示中指定 JSON 格式要求
	var systemPrompt string
	if jsonSchema.IsSome() {
		schemaBytes, _ := json.Marshal(jsonSchema.Unwrap())
		systemPrompt = fmt.Sprintf("请以严格的 JSON 格式回应，遵循以下 schema：%s", string(schemaBytes))
	} else {
		systemPrompt = "请以有效的 JSON 格式回应。"
	}

	// 将系统提示添加到消息开头
	modifiedInput := input
	if len(modifiedInput.Messages) > 0 && modifiedInput.Messages[0].Role == llminterface.RoleSystem {
		// 如果已有系统消息，追加到现有内容
		modifiedInput.Messages[0].Content = append(modifiedInput.Messages[0].Content,
			llminterface.ContentPart{
				Type: llminterface.PartTypeText,
				Text: "\n\n" + systemPrompt,
			})
	} else {
		// 创建新的系统消息
		systemMsg := llminterface.InputMessage{
			Role: llminterface.RoleSystem,
			Content: []llminterface.ContentPart{
				{
					Type: llminterface.PartTypeText,
					Text: systemPrompt,
				},
			},
		}
		modifiedInput.Messages = append([]llminterface.InputMessage{systemMsg}, modifiedInput.Messages...)
	}

	// 使用修改后的输入调用 Chat 方法
	chatChan, err := a.Chat(ctx, modifiedInput)
	if err != nil {
		return "", fmt.Errorf("生成 JSON 响应失败: %w", err)
	}

	// 聚合流式响应
	var result strings.Builder
	for chunk := range chatChan {
		if chunk.Error != nil {
			return "", fmt.Errorf("生成 JSON 响应失败: %w", chunk.Error)
		}
		for _, part := range chunk.ContentParts {
			if part.Type == llminterface.PartTypeText {
				result.WriteString(part.Text)
			}
		}
	}

	return result.String(), nil
}

// RegisterTools 实现 ChatAdapter 接口的 RegisterTools 方法
func (a *AnthropicAdapter) RegisterTools(ctx context.Context, registry *toolcore.Registry) error {
	a.registry = Some(registry)

	if registry == nil {
		a.config.Tools = None[[]anthropic.ToolUnionParam]()
		return nil
	}

	tools := make([]anthropic.ToolUnionParam, 0)

	for _, tool := range registry.GetAll() {
		schema, err := tool.Schema(ctx)
		if err != nil {
			a.logger.Error("获取工具 schema 失败", "tool", schema.Name, "error", err)
			continue
		}

		// 获取工具描述（使用默认语言）
		description := schema.Descriptions["zh"] // 优先使用中文
		if description == "" {
			description = schema.Descriptions["en"] // 回退到英文
		}
		if description == "" && len(schema.Descriptions) > 0 {
			// 使用任何可用的描述
			for _, desc := range schema.Descriptions {
				description = desc
				break
			}
		}

		// 构建输入参数的 properties
		properties := make(map[string]any)
		required := make([]string, 0)

		for _, param := range schema.InputParameters {
			paramDescription := param.Description["zh"]
			if paramDescription == "" {
				paramDescription = param.Description["en"]
			}
			if paramDescription == "" && len(param.Description) > 0 {
				for _, desc := range param.Description {
					paramDescription = desc
					break
				}
			}

			var paramType string
			switch param.Type {
			case toolcore.ParamTypeString:
				paramType = "string"
			case toolcore.ParamTypeInteger:
				paramType = "integer"
			case toolcore.ParamTypeNumber:
				paramType = "number"
			case toolcore.ParamTypeBoolean:
				paramType = "boolean"
			case toolcore.ParamTypeArray:
				paramType = "array"
			case toolcore.ParamTypeObject:
				paramType = "object"
			default:
				paramType = "string"
			}

			paramSchema := map[string]any{
				"type": paramType,
			}
			if paramDescription != "" {
				paramSchema["description"] = paramDescription
			}
			if param.EnumValues.IsSome() && len(param.EnumValues.Unwrap()) > 0 {
				paramSchema["enum"] = param.EnumValues.Unwrap()
			}

			properties[param.Name] = paramSchema

			if param.Required {
				required = append(required, param.Name)
			}
		}

		inputSchema := anthropic.ToolParam{
			Name:        schema.Name,
			Description: anthropic.String(description),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: properties,
			},
		}

		if len(required) > 0 {
			inputSchema.InputSchema.Required = required
		}

		tools = append(tools, anthropic.ToolUnionParam{
			OfTool: &inputSchema,
		})
	}

	if len(tools) > 0 {
		a.config.Tools = Some(tools)
	} else {
		a.config.Tools = None[[]anthropic.ToolUnionParam]()
	}

	a.logger.Info("已注册工具", "count", len(tools))
	return nil
}

// GetFrameworkName 实现 ChatAdapter 接口的 GetFrameworkName 方法
func (a *AnthropicAdapter) GetFrameworkName() string {
	return "anthropic-sdk-go"
}

// convertChatInputToAnthropicMsgs 将 ChatInput 转换为 Anthropic SDK 的消息格式
func (a *AnthropicAdapter) convertChatInputToAnthropicMsgs(input llminterface.ChatInput) ([]anthropic.MessageParam, error) {
	messages := make([]anthropic.MessageParam, 0, len(input.Messages))

	for _, msg := range input.Messages {
		var role anthropic.MessageParamRole
		switch msg.Role {
		case llminterface.RoleUser:
			role = anthropic.MessageParamRoleUser
		case llminterface.RoleAssistant:
			role = anthropic.MessageParamRoleAssistant
		case llminterface.RoleToolResult:
			// 工具结果消息需要特殊处理
			if msg.ToolCallID.IsNone() {
				return nil, fmt.Errorf("工具结果消息缺少 tool_call_id")
			}

			// 提取文本内容作为工具结果
			var resultContent string
			for _, part := range msg.Content {
				if part.Type == llminterface.PartTypeText {
					resultContent += part.Text
				}
			}

			toolResultMsg := anthropic.MessageParam{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{
						OfToolResult: &anthropic.ToolResultBlockParam{
							ToolUseID: msg.ToolCallID.Unwrap(),
							Content: []anthropic.ToolResultBlockParamContentUnion{
								{
									OfText: &anthropic.TextBlockParam{
										Text: resultContent,
									},
								},
							},
						},
					},
				},
			}
			messages = append(messages, toolResultMsg)
			continue
		default:
			return nil, fmt.Errorf("不支持的消息角色: %s", msg.Role)
		}

		// 转换内容部分
		content := make([]anthropic.ContentBlockParamUnion, 0, len(msg.Content))

		for _, part := range msg.Content {
			switch part.Type {
			case llminterface.PartTypeText:
				content = append(content, anthropic.ContentBlockParamUnion{
					OfText: &anthropic.TextBlockParam{
						Text: part.Text,
					},
				})

			case llminterface.PartTypeImageURL:
				if part.ImageURL.IsNone() {
					continue
				}
				imageURL := part.ImageURL.Unwrap()

				// 解析 Data URI
				if strings.HasPrefix(imageURL.URL, "data:") {
					// 提取 MIME 类型和 base64 数据
					parts := strings.Split(imageURL.URL, ",")
					if len(parts) == 2 {
						mimeType := imageURL.MimeType()
						base64Data := parts[1]

						content = append(content, anthropic.ContentBlockParamUnion{
							OfImage: &anthropic.ImageBlockParam{
								Source: anthropic.ImageBlockParamSourceUnion{
									OfBase64: &anthropic.Base64ImageSourceParam{
										MediaType: anthropic.Base64ImageSourceMediaType(mimeType),
										Data:      base64Data,
									},
								},
							},
						})
					}
				}

			case llminterface.PartTypeToolCallRequest:
				if part.ToolCallValues.IsNone() {
					continue
				}

				// 处理工具调用请求
				for _, toolCall := range part.ToolCallValues.Unwrap().Calls {
					// 解析参数 JSON
					var inputArgs any
					if toolCall.Arguments != "" {
						err := json.Unmarshal([]byte(toolCall.Arguments), &inputArgs)
						if err != nil {
							a.logger.Warn("解析工具调用参数失败", "error", err, "args", toolCall.Arguments)
							inputArgs = toolCall.Arguments
						}
					}

					content = append(content, anthropic.ContentBlockParamUnion{
						OfToolUse: &anthropic.ToolUseBlockParam{
							ID:    toolCall.ID,
							Name:  toolCall.Name,
							Input: inputArgs,
						},
					})
				}
			}
		}

		if len(content) > 0 {
			messages = append(messages, anthropic.MessageParam{
				Role:    role,
				Content: content,
			})
		}
	}

	return messages, nil
}
