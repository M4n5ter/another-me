package reactagent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/m4n5ter/another-me/pkg/llminterface"

	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// Agent 表示一个 ReAct 智能体。
type Agent struct {
	llmAdapter    llminterface.ChatAdapter
	toolRegistry  *toolcore.Registry
	logger        *slog.Logger
	maxIterations int
	systemPrompt  Option[llminterface.InputMessage]
}

// AgentBuilder 是用于构建 Agent 的构建器。
type AgentBuilder struct {
	llmAdapter    llminterface.ChatAdapter
	toolRegistry  *toolcore.Registry
	logger        *slog.Logger
	maxIterations int
	systemPrompt  Option[string]
}

// NewAgentBuilder 创建一个新的 AgentBuilder 实例。
func NewAgentBuilder() *AgentBuilder {
	return &AgentBuilder{
		maxIterations: 10, // 默认最大迭代次数
	}
}

// WithLLMAdapter 设置 LLM 适配器。
func (b *AgentBuilder) WithLLMAdapter(adapter llminterface.ChatAdapter) *AgentBuilder {
	b.llmAdapter = adapter
	return b
}

// WithToolRegistry 设置工具注册表。
func (b *AgentBuilder) WithToolRegistry(registry *toolcore.Registry) *AgentBuilder {
	b.toolRegistry = registry
	return b
}

// WithLogger 设置日志记录器。
func (b *AgentBuilder) WithLogger(logger *slog.Logger) *AgentBuilder {
	b.logger = logger
	return b
}

// WithMaxIterations 设置最大迭代次数。
func (b *AgentBuilder) WithMaxIterations(maxIter int) *AgentBuilder {
	if maxIter > 0 {
		b.maxIterations = maxIter
	}
	return b
}

// WithSystemPrompt 设置系统提示。
func (b *AgentBuilder) WithSystemPrompt(prompt string) *AgentBuilder {
	b.systemPrompt = Some(prompt)
	return b
}

// Build 构建并返回一个 Agent 实例。
func (b *AgentBuilder) Build() (*Agent, error) {
	if b.llmAdapter == nil {
		return nil, fmt.Errorf("LLMAdapter 不能为空")
	}
	if b.toolRegistry == nil {
		return nil, fmt.Errorf("ToolRegistry 不能为空")
	}

	logger := b.logger
	if logger == nil {
		logger = slog.Default().WithGroup("react_agent")
	}

	// 系统提示
	var sysPromptOpt Option[llminterface.InputMessage]
	if b.systemPrompt.IsSome() {
		sysPromptOpt = Some(llminterface.InputMessage{
			Role: llminterface.RoleSystem,
			Content: []llminterface.ContentPart{
				{
					Type: llminterface.PartTypeText,
					Text: b.systemPrompt.Unwrap(),
				},
			},
		})
	}

	return &Agent{
		llmAdapter:    b.llmAdapter,
		toolRegistry:  b.toolRegistry,
		logger:        logger,
		maxIterations: b.maxIterations,
		systemPrompt:  sysPromptOpt,
	}, nil
}

// Run 方法执行 ReAct 代理的核心逻辑。
// 它接收初始用户输入，并返回一个用于流式输出的只读通道。
// 流式输出是通过 AgentOutputChunk 结构体实现的，它可包含文本块、工具执行信号或错误。
// 客户端应从返回的通道中读取，直到通道关闭或收到 IsLast=true 的数据块。
func (a *Agent) Run(ctx context.Context, userInput, conversationID string) (<-chan AgentOutputChunk, error) {
	a.logger.Info("ReAct Agent Run started", "conversationID", conversationID, "userInput", userInput)

	// 创建输出通道，用于流式传输 AgentOutputChunk
	outputChan := make(chan AgentOutputChunk)

	// 启动一个 goroutine 执行 ReAct 循环，并通过 outputChan 流式传输结果
	go func() {
		defer close(outputChan) // 确保在函数退出时关闭通道

		messages := make([]llminterface.InputMessage, 0)

		// 添加系统提示（如果存在）
		if a.systemPrompt.IsSome() {
			messages = append(messages, a.systemPrompt.Unwrap())
			a.logger.Debug("System prompt added to messages")
		}

		// 添加用户初始输入
		userMessage := llminterface.InputMessage{
			Role: llminterface.RoleUser,
			Content: []llminterface.ContentPart{
				{
					Type: llminterface.PartTypeText,
					Text: userInput,
				},
			},
		}
		messages = append(messages, userMessage)
		a.logger.Debug("User input added to messages", "userInput", userInput)

		var lastResponseText string

		// ReAct 循环
		for i := range a.maxIterations {
			a.logger.Info("Iteration start", "iteration", i+1, "conversationID", conversationID)

			chatInput := llminterface.ChatInput{
				Messages:       messages,
				ConversationID: conversationID,
			}

			// 调用 LLM
			llmOutputChan, err := a.llmAdapter.Chat(ctx, chatInput)
			if err != nil {
				a.logger.Error("LLM Chat returned an initial error", "error", err, "conversationID", conversationID)
				outputChan <- AgentOutputChunk{
					Type:   AgentChunkTypeError,
					Error:  fmt.Sprintf("LLM Chat 初始错误: %v", err),
					IsLast: true,
				}
				return
			}

			// 用于聚合 LLM 响应的变量
			llmThinks := ""
			// 创建一个 map 来聚合相同 ID 的工具调用
			toolCallsMap := make(map[string]llminterface.ToolCall)
			// 记录工具调用的顺序
			toolCallOrder := make([]string, 0)
			// 记录当前正在组装参数的工具调用ID（如果在处理参数流）
			var currentToolCallID string
			var finishReason Option[string]

			// 处理 LLM 输出流
			for chunk := range llmOutputChan {
				// 错误处理
				if chunk.Error != nil && !errors.Is(chunk.Error, context.Canceled) {
					a.logger.Error("Error in LLM stream chunk", "error", chunk.Error, "conversationID", conversationID)
					outputChan <- AgentOutputChunk{
						Type:   AgentChunkTypeError,
						Error:  fmt.Sprintf("LLM 流处理错误: %v", chunk.Error),
						IsLast: true,
					}
					return
				}

				if chunk.FinishReason.IsSome() {
					finishReason = chunk.FinishReason
				}

				// 处理内容部分
				for _, part := range chunk.ContentParts {
					// 处理文本部分 - 流式传输到 outputChan 并累积内部状态
					if part.Type == llminterface.PartTypeText {
						// 流式输出文本
						outputChan <- AgentOutputChunk{
							Type:      AgentChunkTypeText,
							TextDelta: part.Text,
						}

						// 累积文本，用于内部使用
						llmThinks += part.Text
					} else if part.Type == llminterface.PartTypeToolCallRequest && part.ToolCallValues.IsSome() {
						// 处理工具调用请求（不直接流式输出工具调用内容）
						toolCalls := part.ToolCallValues.Unwrap().Calls
						a.logger.Debug("Processing tool calls", "count", len(toolCalls), "conversationID", conversationID)

						for _, call := range toolCalls {
							a.processToolCallForStreaming(call, toolCallsMap, &toolCallOrder, &currentToolCallID, conversationID)
						}
					}
				}
			}

			// 按照原始顺序重建工具调用列表
			toolCallsToExecute := make([]llminterface.ToolCall, 0, len(toolCallOrder))
			for _, id := range toolCallOrder {
				toolCallsToExecute = append(toolCallsToExecute, toolCallsMap[id])
			}

			// 在流结束后，发送完整的思考内容（可选）
			if llmThinks != "" {
				outputChan <- AgentOutputChunk{
					Type:           AgentChunkTypeThought,
					ThoughtContent: llmThinks,
				}
			}

			// 重建 assistant 的响应内容，确保工具调用信息是完整的
			assistantResponseContentParts := make([]llminterface.ContentPart, 0)

			// 添加文本部分（如果有）
			if llmThinks != "" {
				assistantResponseContentParts = append(assistantResponseContentParts, llminterface.ContentPart{
					Type: llminterface.PartTypeText,
					Text: llmThinks,
				})
			}

			// 添加工具调用部分（如果有）
			if len(toolCallsToExecute) > 0 {
				assistantResponseContentParts = append(assistantResponseContentParts, llminterface.ContentPart{
					Type: llminterface.PartTypeToolCallRequest,
					ToolCallValues: Some(llminterface.ToolCallContent{
						Calls: toolCallsToExecute,
					}),
				})
			}

			// 将LLM的完整回复（包括思考和工具调用请求）添加到消息历史
			if len(assistantResponseContentParts) > 0 {
				assistantMessage := llminterface.InputMessage{
					Role:    llminterface.RoleAssistant,
					Content: assistantResponseContentParts,
				}
				messages = append(messages, assistantMessage)
				a.logger.Debug("Assistant's full response added to messages", "conversationID", conversationID, "partsCount", len(assistantResponseContentParts))
			}

			lastResponseText = llmThinks // 保存最后一次LLM的文本输出，以备没有工具调用时作为最终结果

			// 如果没有工具调用，并且LLM给出了结束原因，或者LLM有文本输出但没有结束原因（可能是隐式结束）
			if len(toolCallsToExecute) == 0 {
				switch {
				case finishReason.IsSome() && finishReason.Unwrap() == "stop":
					a.logger.Info("LLM indicated stop, no tool calls. Returning last response.", "conversationID", conversationID, "finishReason", finishReason.Unwrap())
					outputChan <- AgentOutputChunk{
						Type:          AgentChunkTypeFinish,
						FinalResponse: llmThinks,
						IsLast:        true,
					}
					return
				case finishReason.IsSome() && finishReason.Unwrap() != "tool_calls":
					a.logger.Info("LLM provided a non-tool_calls finish reason, no tool calls. Returning last response.", "conversationID", conversationID, "finishReason", finishReason.Unwrap())
					outputChan <- AgentOutputChunk{
						Type:          AgentChunkTypeFinish,
						FinalResponse: llmThinks,
						IsLast:        true,
					}
					return
				case llmThinks != "" && finishReason.IsNone():
					a.logger.Info("LLM provided text output without a finish reason, no tool calls. Assuming this is the final response.", "conversationID", conversationID)
					outputChan <- AgentOutputChunk{
						Type:          AgentChunkTypeFinish,
						FinalResponse: llmThinks,
						IsLast:        true,
					}
					return
				default:
					a.logger.Warn("LLM produced no text, no tool calls, and no finish reason. Potential loop or LLM issue.", "conversationID", conversationID)
					// 继续循环
				}
			}

			// 执行工具调用
			if len(toolCallsToExecute) > 0 {
				a.logger.Info("LLM requested tool calls", "count", len(toolCallsToExecute), "conversationID", conversationID)
				for _, toolCall := range toolCallsToExecute {
					if toolCall.Name == "" || toolCall.ID == "" {
						a.logger.Warn("LLM requested a tool call with empty name or ID, skipping.", "toolName", toolCall.Name, "toolID", toolCall.ID, "conversationID", conversationID)
						continue
					}

					// 发送工具开始执行的信号
					outputChan <- AgentOutputChunk{
						Type:     AgentChunkTypeToolStart,
						ToolName: toolCall.Name,
					}

					a.logger.Info("Executing tool", "toolName", toolCall.Name, "toolID", toolCall.ID, "argsLength", len(toolCall.Arguments), "conversationID", conversationID)

					tool, exists := a.toolRegistry.Get(toolCall.Name)
					if !exists {
						a.logger.Error("Tool not found in registry", "toolName", toolCall.Name, "conversationID", conversationID)
						toolResultMessage := llminterface.InputMessage{
							Role:       llminterface.RoleToolResult,
							ToolCallID: Some(toolCall.ID),
							ToolName:   Some(toolCall.Name),
							Content: []llminterface.ContentPart{
								{
									Type: llminterface.PartTypeText,
									Text: fmt.Sprintf("错误：工具 '%s' 未找到。", toolCall.Name),
								},
							},
						}
						messages = append(messages, toolResultMessage)

						// 发送工具结束执行的信号（带错误）
						outputChan <- AgentOutputChunk{
							Type:     AgentChunkTypeToolEnd,
							ToolName: toolCall.Name,
							Error:    fmt.Sprintf("错误：工具 '%s' 未找到。", toolCall.Name),
						}

						continue // 继续处理下一个工具调用（如果有）
					}

					// 执行工具调用
					toolOutputJSON, toolErr := tool.Call(ctx, toolCall.Arguments)
					if toolErr != nil {
						a.logger.Error("Tool execution failed", "toolName", toolCall.Name, "toolID", toolCall.ID, "error", toolErr, "conversationID", conversationID)
						toolResultMessage := llminterface.InputMessage{
							Role:       llminterface.RoleToolResult,
							ToolCallID: Some(toolCall.ID),
							ToolName:   Some(toolCall.Name),
							Content: []llminterface.ContentPart{
								{
									Type: llminterface.PartTypeText,
									Text: fmt.Sprintf("错误：工具 '%s' 执行失败: %s", toolCall.Name, toolErr.Error()),
								},
							},
						}
						messages = append(messages, toolResultMessage)

						// 发送工具结束执行的信号（带错误）
						outputChan <- AgentOutputChunk{
							Type:     AgentChunkTypeToolEnd,
							ToolName: toolCall.Name,
							Error:    fmt.Sprintf("错误：工具 '%s' 执行失败: %s", toolCall.Name, toolErr.Error()),
						}

						continue
					}

					// 添加工具调用结果
					toolResultMessage := llminterface.InputMessage{
						Role:       llminterface.RoleToolResult,
						ToolCallID: Some(toolCall.ID),
						ToolName:   Some(toolCall.Name),
						Content: []llminterface.ContentPart{
							{
								Type: llminterface.PartTypeText,
								Text: toolOutputJSON,
							},
						},
					}
					messages = append(messages, toolResultMessage)
					a.logger.Debug("Tool result added to messages", "toolName", toolCall.Name, "toolID", toolCall.ID, "conversationID", conversationID)

					// 发送工具结束执行的信号（成功）
					outputChan <- AgentOutputChunk{
						Type:     AgentChunkTypeToolEnd,
						ToolName: toolCall.Name,
					}
				}
			}
		}

		// 如果达到最大迭代次数
		a.logger.Warn("Max iterations reached", "maxIterations", a.maxIterations, "conversationID", conversationID)
		if lastResponseText != "" {
			outputChan <- AgentOutputChunk{
				Type:          AgentChunkTypeMaxIter,
				FinalResponse: lastResponseText,
				Error:         fmt.Sprintf("达到最大迭代次数 (%d)，返回最后观察到的LLM输出", a.maxIterations),
				IsLast:        true,
			}
		} else {
			outputChan <- AgentOutputChunk{
				Type:   AgentChunkTypeMaxIter,
				Error:  fmt.Sprintf("达到最大迭代次数 (%d) 且没有LLM的最终响应", a.maxIterations),
				IsLast: true,
			}
		}
	}()

	return outputChan, nil
}

// AgentOutputChunk 代表 ReAct Agent 流式输出的一个数据块。
// 注意：工具调用的具体结果不会通过此通道流式传输，它们会被合并到内部消息历史中，
// LLM 对工具结果的后续思考（文本）才会通过此通道流式输出。
type AgentOutputChunk struct {
	Type AgentOutputChunkType `json:"type"` // 块类型

	// 当 Type 为 AgentChunkTypeText 时，此字段包含流式文本片段。
	// 对于其他类型，此字段通常为空。
	TextDelta string `json:"text_delta,omitempty"`

	// 当 Type 为 AgentChunkTypeToolStart 或 AgentChunkTypeToolEnd 时，包含工具名称。
	ToolName string `json:"tool_name,omitempty"`

	// 当 Type 为 AgentChunkTypeThought 时，此字段包含当前迭代LLM的完整思考文本。
	ThoughtContent string `json:"thought_content,omitempty"`

	// 当 Type 为 AgentChunkTypeError 时，此字段包含错误信息。
	// 对于 AgentChunkTypeMaxIter，也可能用此字段传递相关消息。
	Error string `json:"error,omitempty"`

	// 当 Type 为 AgentChunkTypeFinish 时，此字段可能包含最终的响应文本。
	FinalResponse string `json:"final_response,omitempty"`

	// 指示这是否是整个 ReAct 序列的最后一个块。
	// 对于 AgentChunkTypeFinish, AgentChunkTypeError, AgentChunkTypeMaxIter 应该为 true。
	IsLast bool `json:"is_last"`
}

// AgentOutputChunkType 定义了 Agent 输出块的类型。
type AgentOutputChunkType string

const (
	// AgentChunkTypeText 纯文本输出
	AgentChunkTypeText AgentOutputChunkType = "text"
	// AgentChunkTypeToolStart 工具开始执行的信号
	AgentChunkTypeToolStart AgentOutputChunkType = "tool_start"
	// AgentChunkTypeToolEnd 工具结束执行的信号 (可选, 结果不从此输出)
	AgentChunkTypeToolEnd AgentOutputChunkType = "tool_end"
	// AgentChunkTypeError 错误输出
	AgentChunkTypeError AgentOutputChunkType = "error"
	// AgentChunkTypeThought LLM的完整思考步骤文本 (在决定工具调用前或最终回复前)
	AgentChunkTypeThought AgentOutputChunkType = "thought"
	// AgentChunkTypeFinish ReAct 序列正常结束
	AgentChunkTypeFinish AgentOutputChunkType = "finish"
	// AgentChunkTypeMaxIter 达到最大迭代次数
	AgentChunkTypeMaxIter AgentOutputChunkType = "max_iterations"
)

// processToolCallForStreaming 处理单个工具调用，用于流式输出场景
func (a *Agent) processToolCallForStreaming(call llminterface.ToolCall,
	toolCallsMap map[string]llminterface.ToolCall, toolCallOrder *[]string,
	currentToolCallID *string, conversationID string,
) {
	a.logger.Debug("Tool call details", "id", call.ID, "name", call.Name, "argsLength", len(call.Arguments), "conversationID", conversationID)

	// 如果有ID，这是一个新工具调用或工具调用的开始部分
	if call.ID != "" {
		*currentToolCallID = call.ID // 更新当前处理的ID

		if existingCall, exists := toolCallsMap[call.ID]; exists {
			// 已存在相同ID的工具调用，更新信息
			if call.Name != "" && existingCall.Name == "" {
				existingCall.Name = call.Name
			}

			// 追加或替换参数
			if call.Arguments != "" {
				if existingCall.Arguments == "" || existingCall.Arguments == "{}" {
					existingCall.Arguments = call.Arguments
				} else {
					// 追加参数，需要小心JSON结构
					existingCall.Arguments += call.Arguments
				}
			}

			toolCallsMap[call.ID] = existingCall
			a.logger.Debug("Updated existing tool call", "id", call.ID, "name", existingCall.Name,
				"argsLength", len(existingCall.Arguments), "conversationID", conversationID)
		} else {
			// 新工具调用，添加到映射
			toolCallsMap[call.ID] = call
			*toolCallOrder = append(*toolCallOrder, call.ID)
			a.logger.Debug("Added new tool call", "id", call.ID, "name", call.Name, "conversationID", conversationID)
		}
	} else if call.Arguments != "" {
		// 这是一个只包含参数的chunk
		a.processToolCallArgumentsForStreaming(call, toolCallsMap, toolCallOrder, *currentToolCallID, conversationID)
	}
}

// processToolCallArgumentsForStreaming 处理工具调用参数，用于流式输出场景
func (a *Agent) processToolCallArgumentsForStreaming(call llminterface.ToolCall,
	toolCallsMap map[string]llminterface.ToolCall, toolCallOrder *[]string,
	currentToolCallID, conversationID string,
) {
	switch {
	case currentToolCallID != "":
		// 使用当前正在处理的工具调用ID
		if existingCall, exists := toolCallsMap[currentToolCallID]; exists {
			// 追加参数
			if existingCall.Arguments == "" || existingCall.Arguments == "{}" {
				existingCall.Arguments = call.Arguments
			} else {
				existingCall.Arguments += call.Arguments
			}
			toolCallsMap[currentToolCallID] = existingCall
			a.logger.Debug("Appended arguments to current tool call", "id", currentToolCallID,
				"newArgsLength", len(call.Arguments), "totalArgsLength", len(existingCall.Arguments),
				"conversationID", conversationID)
		} else {
			// 这种情况不应该发生：有当前ID但在map中找不到
			a.logger.Warn("Found arguments for non-existent tool call ID", "currentID", currentToolCallID,
				"arguments", call.Arguments, "conversationID", conversationID)

			// 创建一个新条目（应急处理）
			newCall := llminterface.ToolCall{
				ID:        currentToolCallID,
				Arguments: call.Arguments,
			}
			toolCallsMap[currentToolCallID] = newCall
			*toolCallOrder = append(*toolCallOrder, currentToolCallID)
		}
	case len(*toolCallOrder) > 0:
		// 如果没有当前ID但有之前的工具调用，使用最后一个
		lastID := (*toolCallOrder)[len(*toolCallOrder)-1]
		existingCall := toolCallsMap[lastID]

		// 追加参数
		if existingCall.Arguments == "" || existingCall.Arguments == "{}" {
			existingCall.Arguments = call.Arguments
		} else {
			existingCall.Arguments += call.Arguments
		}
		toolCallsMap[lastID] = existingCall
		a.logger.Debug("Appended arguments to last tool call", "id", lastID,
			"newArgsLength", len(call.Arguments), "totalArgsLength", len(existingCall.Arguments),
			"conversationID", conversationID)
	default:
		// 没有工具调用上下文，但收到了参数 - 这是异常情况
		a.logger.Warn("Received tool call arguments without context",
			"arguments", call.Arguments, "conversationID", conversationID)
	}
}
