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

// AgentConfig 用于配置 ReAct 智能体。
type AgentConfig struct {
	LLMAdapter    llminterface.ChatAdapter
	ToolRegistry  *toolcore.Registry
	Logger        *slog.Logger
	MaxIterations int
	SystemPrompt  string // 可选的系统提示
}

// NewAgent 创建一个新的 ReAct 智能体实例。
func NewAgent(config AgentConfig) (*Agent, error) {
	if config.LLMAdapter == nil {
		return nil, fmt.Errorf("LLMAdapter 不能为空")
	}
	if config.ToolRegistry == nil {
		return nil, fmt.Errorf("ToolRegistry 不能为空")
	}

	logger := config.Logger
	if logger == nil {
		logger = slog.Default().WithGroup("react_agent")
	}

	maxIter := config.MaxIterations
	if maxIter <= 0 {
		maxIter = 10 // 默认最大迭代次数
	}

	var sysPromptOpt Option[llminterface.InputMessage]
	if config.SystemPrompt != "" {
		sysPromptOpt = Some(llminterface.InputMessage{
			Role: llminterface.RoleSystem,
			Content: []llminterface.ContentPart{
				{
					Type: llminterface.PartTypeText,
					Text: config.SystemPrompt,
				},
			},
		})
	}

	return &Agent{
		llmAdapter:    config.LLMAdapter,
		toolRegistry:  config.ToolRegistry,
		logger:        logger,
		maxIterations: maxIter,
		systemPrompt:  sysPromptOpt,
	}, nil
}

// Run 方法执行 ReAct 代理的核心逻辑。
// 它接收初始用户输入，并返回最终的助手响应或错误。
func (a *Agent) Run(ctx context.Context, userInput, conversationID string) (string, error) {
	a.logger.Info("ReAct Agent Run started", "conversationID", conversationID, "userInput", userInput)

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

	for i := range a.maxIterations {
		a.logger.Info("Iteration start", "iteration", i+1, "conversationID", conversationID)

		chatInput := llminterface.ChatInput{
			Messages:       messages,
			ConversationID: conversationID,
		}

		outputChan, err := a.llmAdapter.Chat(ctx, chatInput)
		if err != nil {
			a.logger.Error("LLM Chat returned an initial error", "error", err, "conversationID", conversationID)
			return "", fmt.Errorf("LLM Chat 初始错误: %w", err)
		}

		// 聚合LLM响应
		// TODO: 在 stream_utils.go 中实现 AggregateChunks(ctx, outputChan) (fullResponseContent []llminterface.ContentPart, toolCalls []llminterface.ToolCall, finalFinishReason string, err error)
		// 目前我们先用 AggregateTextFromChunks，并且只处理第一个文本部分和第一个工具调用（如果有的话）。
		// 这是一个简化，理想情况下应该能处理多个并发工具调用或混合内容。

		var llmErr error
		var finishReason Option[string]

		// 收集所有块
		rawChunks := make([]llminterface.ChatOutputChunk, 0)
		for chunk := range outputChan {
			rawChunks = append(rawChunks, chunk)
			if chunk.Error != nil && !errors.Is(chunk.Error, context.Canceled) {
				llmErr = chunk.Error
				a.logger.Error("Error in LLM stream chunk", "error", llmErr, "conversationID", conversationID)
				// 不立即中断，聚合所有已收到的内容
			}
			if chunk.FinishReason.IsSome() {
				finishReason = chunk.FinishReason
			}
		}

		if llmErr != nil && len(rawChunks) == 0 { // 如果第一个块就有错，且没有内容
			return "", fmt.Errorf("LLM 流处理错误: %w", llmErr)
		}

		// 从收集的块中提取内容和工具调用
		llmThinks := ""

		// 创建一个 map 来聚合相同 ID 的工具调用
		toolCallsMap := make(map[string]llminterface.ToolCall)
		// 记录工具调用的顺序
		toolCallOrder := make([]string, 0)
		// 记录当前正在组装参数的工具调用ID（如果在处理参数流）
		var currentToolCallID string

		for i, chunk := range rawChunks {
			a.logger.Debug("Processing chunk", "index", i, "contentPartsCount", len(chunk.ContentParts), "finishReason", chunk.FinishReason, "conversationID", conversationID)

			for _, part := range chunk.ContentParts {
				a.processContentPart(part, &llmThinks, toolCallsMap, &toolCallOrder, &currentToolCallID, conversationID)
			}
		}

		// 按照原始顺序重建工具调用列表
		toolCallsToExecute := make([]llminterface.ToolCall, 0, len(toolCallOrder))
		for _, id := range toolCallOrder {
			toolCallsToExecute = append(toolCallsToExecute, toolCallsMap[id])
		}

		a.logger.Debug("LLM raw response aggregated", "llm_thinks_length", len(llmThinks), "tool_calls_count", len(toolCallsToExecute), "finish_reason", finishReason, "conversationID", conversationID)

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
		// assistantResponseContentParts 已经聚合了LLM返回的所有内容部分
		if len(assistantResponseContentParts) > 0 {
			assistantMessage := llminterface.InputMessage{
				Role:    llminterface.RoleAssistant,
				Content: assistantResponseContentParts, // 直接使用聚合的、未经修改的LLM输出部分
			}
			messages = append(messages, assistantMessage)
			a.logger.Debug("Assistant's full response added to messages", "conversationID", conversationID, "partsCount", len(assistantResponseContentParts))
		}

		lastResponseText = llmThinks // 保存最后一次LLM的文本输出，以备没有工具调用时作为最终结果

		// TODOL 结束迭代逻辑还需要改进
		// 如果没有工具调用，并且LLM给出了结束原因，或者LLM有文本输出但没有结束原因（可能是隐式结束）
		if len(toolCallsToExecute) == 0 {
			switch {
			case finishReason.IsSome() && finishReason.Unwrap() == "stop":
				a.logger.Info("LLM indicated stop, no tool calls. Returning last response.", "conversationID", conversationID, "finishReason", finishReason.Unwrap())
				return llmThinks, nil
			case finishReason.IsSome() && finishReason.Unwrap() != "tool_calls":
				a.logger.Info("LLM provided a non-tool_calls finish reason, no tool calls. Returning last response.", "conversationID", conversationID, "finishReason", finishReason.Unwrap())
				return llmThinks, nil
			case llmThinks != "" && !finishReason.IsSome():
				a.logger.Info("LLM provided text output without a finish reason, no tool calls. Assuming this is the final response.", "conversationID", conversationID)
				return llmThinks, nil
			case llmErr != nil:
				a.logger.Error("LLM stream resulted in an error, and no valid tool calls were made or executed.", "error", llmErr, "conversationID", conversationID)
				llmThinks, err := llminterface.AggregateTextFromChunks(ctx, outputChan)
				if err != nil {
					return "", fmt.Errorf("LLM stream resulted in an error: %w", err)
				}
				return llmThinks, nil
			default:
				a.logger.Warn("LLM produced no text, no tool calls, and no finish reason. Potential loop or LLM issue.", "conversationID", conversationID)
				// 可以在这里决定是返回错误还是空响应，或者允许循环继续（可能达到最大迭代）
				// 为避免无限循环，如果迭代多次仍无进展，外层循环的 maxIterations 会捕获
			}
		}

		if len(toolCallsToExecute) > 0 {
			a.logger.Info("LLM requested tool calls", "count", len(toolCallsToExecute), "conversationID", conversationID)
			for _, toolCall := range toolCallsToExecute {
				if toolCall.Name == "" || toolCall.ID == "" {
					a.logger.Warn("LLM requested a tool call with empty name or ID, skipping.", "toolName", toolCall.Name, "toolID", toolCall.ID, "conversationID", conversationID)
					continue
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
			}
		} else if llmErr != nil { // 如果流中有错误，但没有工具调用，则返回错误和已聚合的文本
			// 在此之前，助手消息（可能为空或仅包含无效工具调用）已经根据上面的逻辑添加了
			// 如果 llmErr 是在聚合所有块之后发生的，并且没有有效的工具调用被执行，
			// 那么 lastResponseText （即 llmThinks）和 llmErr 是最合适的返回值。
			a.logger.Error("LLM stream resulted in an error, and no valid tool calls were made or executed.", "error", llmErr, "conversationID", conversationID)
			return llmThinks, llmErr
		}

		// 如果上一个LLM响应中既没有工具调用，也没有文本思考，并且没有错误，那么这可能是一个问题。
		// 但是，外层循环的迭代限制会处理这种情况。
		if len(toolCallsToExecute) == 0 && llmThinks == "" && llmErr == nil && !finishReason.IsSome() {
			a.logger.Warn("LLM did not request tool calls, produce text, or signal finish. Continuing iteration.", "iteration", i+1, "conversationID", conversationID)
		}

	}

	a.logger.Warn("Max iterations reached", "maxIterations", a.maxIterations, "conversationID", conversationID)
	// 如果达到最大迭代次数，返回最后一次LLM的文本输出（如果有）以及一个错误
	if lastResponseText != "" {
		return lastResponseText, fmt.Errorf("达到最大迭代次数 (%d)，返回最后观察到的LLM输出", a.maxIterations)
	}
	return "", fmt.Errorf("达到最大迭代次数 (%d) 且没有LLM的最终响应", a.maxIterations)
}

// processContentPart 处理单个内容部分，提取文本或处理工具调用
func (a *Agent) processContentPart(part llminterface.ContentPart, llmThinks *string,
	toolCallsMap map[string]llminterface.ToolCall, toolCallOrder *[]string,
	currentToolCallID *string, conversationID string,
) {
	if part.Type == llminterface.PartTypeText {
		*llmThinks += part.Text
		a.logger.Debug("Added text from chunk", "textLength", len(part.Text), "conversationID", conversationID)
	} else if part.Type == llminterface.PartTypeToolCallRequest && part.ToolCallValues.IsSome() {
		toolCalls := part.ToolCallValues.Unwrap().Calls
		a.logger.Debug("Processing tool calls", "count", len(toolCalls), "conversationID", conversationID)

		for _, call := range toolCalls {
			a.processToolCall(call, toolCallsMap, toolCallOrder, currentToolCallID, conversationID)
		}
	}
}

// processToolCall 处理单个工具调用
func (a *Agent) processToolCall(call llminterface.ToolCall,
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
		a.processToolCallArguments(call, toolCallsMap, toolCallOrder, *currentToolCallID, conversationID)
	}
}

// processToolCallArguments 处理工具调用参数
func (a *Agent) processToolCallArguments(call llminterface.ToolCall,
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
