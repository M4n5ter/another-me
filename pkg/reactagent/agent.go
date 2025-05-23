package reactagent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// ToolCallingAgent 表示一个 ReAct 智能体，它依赖底模的Tool Calling/Function Calling能力。
type ToolCallingAgent struct {
	llmAdapter    llminterface.ChatAdapter
	taskEvaluator Option[llminterface.ChatAdapter]
	toolRegistry  *toolcore.Registry
	logger        *slog.Logger
	maxIterations int
	systemPrompt  Option[llminterface.InputMessage]
	checkpoint    Option[*llminterface.ChatInput] // 可选的检查点，用于继续执行
}

// ToolCallingAgentBuilder 是用于构建 ToolCallingAgent 的构建器。
type ToolCallingAgentBuilder struct {
	llmAdapter    llminterface.ChatAdapter
	taskEvaluator Option[llminterface.ChatAdapter]
	toolRegistry  *toolcore.Registry
	logger        *slog.Logger
	maxIterations int
	systemPrompt  Option[string]
	checkpoint    Option[*llminterface.ChatInput]
}

// NewToolCallingAgentBuilder 创建一个新的 ToolCallingAgentBuilder 实例。
func NewToolCallingAgentBuilder() *ToolCallingAgentBuilder {
	return &ToolCallingAgentBuilder{
		maxIterations: 10, // 默认最大迭代次数
	}
}

// WithLLMAdapter 设置 LLM 适配器。
func (b *ToolCallingAgentBuilder) WithLLMAdapter(adapter llminterface.ChatAdapter) *ToolCallingAgentBuilder {
	b.llmAdapter = adapter
	return b
}

// WithTaskEvaluator 设置任务评估器。
func (b *ToolCallingAgentBuilder) WithTaskEvaluator(evaluator llminterface.ChatAdapter) *ToolCallingAgentBuilder {
	b.taskEvaluator = Some(evaluator)
	return b
}

// WithToolRegistry 设置工具注册表。
func (b *ToolCallingAgentBuilder) WithToolRegistry(registry *toolcore.Registry) *ToolCallingAgentBuilder {
	b.toolRegistry = registry
	return b
}

// WithLogger 设置日志记录器。
func (b *ToolCallingAgentBuilder) WithLogger(logger *slog.Logger) *ToolCallingAgentBuilder {
	b.logger = logger
	return b
}

// WithMaxIterations 设置最大迭代次数。
func (b *ToolCallingAgentBuilder) WithMaxIterations(maxIter int) *ToolCallingAgentBuilder {
	if maxIter > 0 {
		b.maxIterations = maxIter
	}
	return b
}

// WithSystemPrompt 设置系统提示。
func (b *ToolCallingAgentBuilder) WithSystemPrompt(prompt string) *ToolCallingAgentBuilder {
	b.systemPrompt = Some(prompt)
	return b
}

// WithCheckpoint 设置检查点。
func (b *ToolCallingAgentBuilder) WithCheckpoint(checkpoint *llminterface.ChatInput) *ToolCallingAgentBuilder {
	b.checkpoint = Some(checkpoint)
	return b
}

// Build 构建并返回一个 ToolCallingAgent 实例。
func (b *ToolCallingAgentBuilder) Build() (*ToolCallingAgent, error) {
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

	return &ToolCallingAgent{
		llmAdapter:    b.llmAdapter,
		taskEvaluator: b.taskEvaluator,
		toolRegistry:  b.toolRegistry,
		logger:        logger,
		maxIterations: b.maxIterations,
		systemPrompt:  sysPromptOpt,
		checkpoint:    b.checkpoint,
	}, nil
}

// Run 方法执行 ReAct 代理的核心逻辑。
// 它接收初始用户输入，并返回一个用于流式输出的只读通道。
// 流式输出是通过 AgentOutputChunk 结构体实现的，它可包含文本块、工具执行信号或错误。
// 客户端应从返回的通道中读取，直到通道关闭或收到 IsLast=true 的数据块。
func (a *ToolCallingAgent) Run(ctx context.Context, userInput, conversationID string) (<-chan AgentOutputChunk, error) {
	a.logger.Info("ReAct Agent Run started", "conversationID", conversationID, "userInput", userInput)

	// 如果存在检查点，优先使用检查点继续执行
	if a.checkpoint.IsSome() {
		checkpoint := a.checkpoint.Unwrap()
		// 使用传入的会话ID覆盖检查点中的会话ID
		checkpoint.ConversationID = conversationID
		return a.ContinueFromCheckpoint(ctx, userInput, checkpoint)
	}

	// 创建输出通道，用于流式传输 AgentOutputChunk
	outputChan := make(chan AgentOutputChunk, 100)

	// 启动一个 goroutine 执行 ReAct 循环，并通过 outputChan 流式传输结果
	go func() {
		defer close(outputChan) // 确保在函数退出时关闭通道

		messages := prepareInitialMessages(a.systemPrompt, userInput, a.logger)

		// 执行ReAct循环核心逻辑
		a.handleReactLoop(ctx, messages, conversationID, userInput, outputChan)
	}()

	return outputChan, nil
}

// ContinueFromCheckpoint 方法从某个检查点继续执行
// 它接收新的用户输入和先前保存的对话历史记录，并返回一个用于流式输出的只读通道。
func (a *ToolCallingAgent) ContinueFromCheckpoint(ctx context.Context, userInput string, checkpoint *llminterface.ChatInput) (<-chan AgentOutputChunk, error) {
	if checkpoint == nil {
		return nil, fmt.Errorf("检查点不能为空")
	}

	// 使用传递的会话ID
	conversationID := checkpoint.ConversationID
	a.logger.Info("ReAct Agent ContinueFromCheckpoint started", "conversationID", conversationID, "userInput", userInput)

	// 创建输出通道，用于流式传输 AgentOutputChunk
	outputChan := make(chan AgentOutputChunk, 100)

	// 启动一个 goroutine 执行 ReAct 循环，并通过 outputChan 流式传输结果
	go func() {
		defer close(outputChan) // 确保在函数退出时关闭通道

		messages := prepareCheckpointMessages(checkpoint, userInput, a.logger)

		// For checkpoints, the initialUserInput is the userInput that continues the conversation.
		a.handleReactLoop(ctx, messages, conversationID, userInput, outputChan)
	}()

	return outputChan, nil
}

// handleReactLoop 处理共享的React循环逻辑
func (a *ToolCallingAgent) handleReactLoop(
	ctx context.Context,
	messages []llminterface.InputMessage,
	conversationID string,
	initialUserInput string,
	outputChan chan<- AgentOutputChunk,
) {
	// 保存最后一次LLM的文本输出，以备没有工具调用时作为最终结果
	lastResponseText := ""

	// 用于累积所有迭代的LLM响应
	// 格式:
	// Iteration 1:
	// ....
	// Iteration 2:
	// ....
	// Iteration 3:
	// ....
	// Iteration N:
	// ....
	llmThinks := ""

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
			outputChan <- handleLLMChatInitError(a.logger, err, conversationID, messages)
			return
		}

		// 用于聚合当前迭代LLM响应的变量
		currentIterationThinks := ""
		// 创建一个 map 来聚合相同 ID 的工具调用
		toolCallsMap := make(map[string]llminterface.ToolCall)
		// 记录工具调用的顺序
		toolCallOrder := make([]string, 0)
		// 记录当前正在组装参数的工具调用ID（如果在处理参数流）
		currentToolCallID := ""
		finishReason := None[string]()

		// 处理 LLM 输出流
		for chunk := range llmOutputChan {
			// 错误处理
			if chunk.Error != nil && !errors.Is(chunk.Error, context.Canceled) {
				outputChan <- handleLLMStreamError(a.logger, chunk.Error, conversationID, messages)
				return
			}

			if chunk.FinishReason.IsSome() {
				finishReason = chunk.FinishReason
			}

			if chunk.Reasoning.IsSome() {
				outputChan <- createReasoningChunk(chunk.Reasoning.Unwrap())
			}

			// 处理内容部分
			for _, part := range chunk.ContentParts {
				// 处理文本部分 - 流式传输到 outputChan 并累积内部状态
				if part.Type == llminterface.PartTypeText {
					// 流式输出文本
					outputChan <- createTextChunk(part.Text)

					// 累积文本，用于内部使用
					currentIterationThinks += part.Text
				} else if part.Type == llminterface.PartTypeToolCallRequest && part.ToolCallValues.IsSome() {
					// 处理工具调用请求（不直接流式输出工具调用内容）
					toolCalls := part.ToolCallValues.Unwrap().Calls
					a.logger.Debug("Processing tool calls", "count", len(toolCalls), "conversationID", conversationID)

					for _, call := range toolCalls {
						a.processToolCallForStreaming(call, toolCallsMap, &toolCallOrder, &currentToolCallID, conversationID)
					}
				}
			}

			switch {
			case ctx.Err() != nil:
				// 按照原始顺序重建工具调用列表
				toolCallsToExecute := make([]llminterface.ToolCall, 0, len(toolCallOrder))
				for _, id := range toolCallOrder {
					toolCallsToExecute = append(toolCallsToExecute, toolCallsMap[id])
				}
				messages = append(messages, composeMessage(messages, currentIterationThinks, toolCallsToExecute)...)
				a.logger.Info("Context canceled, exiting loop,", "messages", messages)
				return
			default:
				continue
			}
		}

		// 按照原始顺序重建工具调用列表
		toolCallsToExecute := make([]llminterface.ToolCall, 0, len(toolCallOrder))
		for _, id := range toolCallOrder {
			toolCallsToExecute = append(toolCallsToExecute, toolCallsMap[id])
		}

		// 在流结束后，发送当前迭代的LLM思考内容
		if currentIterationThinks != "" {
			outputChan <- createThoughtChunk(currentIterationThinks)
		}

		messages = append(messages, composeMessage(messages, currentIterationThinks, toolCallsToExecute)...)

		// 根据当前迭代的LLM输出和工具调用情况，确定本次迭代的思考内容
		thoughtForThisIteration := ""
		switch {
		case currentIterationThinks != "":
			thoughtForThisIteration = currentIterationThinks
			lastResponseText = currentIterationThinks // 保存最后一次LLM的文本输出，以备没有工具调用时作为最终结果
		case len(toolCallsToExecute) > 0:
			var sb strings.Builder
			for _, tc := range toolCallsToExecute {
				sb.WriteString(fmt.Sprintf("Tool Call: %s, Args: %s\n", tc.Name, tc.Arguments))
			}
			thoughtForThisIteration = sb.String()
		default:
			thoughtForThisIteration = "[No LLM text output or tool calls in this iteration]"
		}

		llmThinks = updateAccumulatedThoughts(llmThinks, thoughtForThisIteration, i)

		if len(toolCallsToExecute) == 0 {
			evaluated, needContinue := taskEvaluate(ctx, a.logger, a.taskEvaluator, initialUserInput, llmThinks, lastResponseText, &messages, outputChan, conversationID)
			if evaluated {
				if needContinue {
					continue
				}

				outputChan <- createFinishChunk(lastResponseText, llmThinks, messages, conversationID)
				return
			}

			// 评估器不存在或者没能正确评估
			if !evaluated {
				shouldReturn := false
				switch {
				case finishReason.IsSome() && finishReason.Unwrap() == "stop":
					a.logger.Info("LLM indicated stop (no evaluator), no tool calls. Returning.", "conversationID", conversationID)
					shouldReturn = true
				case finishReason.IsSome() && finishReason.Unwrap() != "tool_calls":
					a.logger.Info("LLM non-tool_calls finish reason (no evaluator), no tool calls. Returning.", "conversationID", conversationID)
					shouldReturn = true
				case currentIterationThinks != "" && finishReason.IsNone():
					a.logger.Info("LLM text output, no finish reason (no evaluator), no tool calls. Assuming final. Returning.", "conversationID", conversationID)
					shouldReturn = true
				default:
					a.logger.Warn("LLM no text, no tool calls, no finish reason (no evaluator). Continuing.", "conversationID", conversationID)
				}
				if shouldReturn {
					outputChan <- createFinishChunk(lastResponseText, llmThinks, messages, conversationID)
					return
				}
				// 继续处理下一个迭代
				continue
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
				outputChan <- createToolStartChunk(toolCall.ID, toolCall.Name, toolCall.Arguments)

				a.logger.Info("Executing tool", "toolName", toolCall.Name, "toolID", toolCall.ID, "argsLength", len(toolCall.Arguments), "conversationID", conversationID)

				tool, exists := a.toolRegistry.Get(toolCall.Name)
				if !exists {
					a.logger.Error("Tool not found in registry", "toolName", toolCall.Name, "conversationID", conversationID)
					errorMsg := fmt.Sprintf("错误：工具 '%s' 未找到。", toolCall.Name)
					toolResultMessage := createToolErrorMessage(toolCall.ID, toolCall.Name, errorMsg)
					messages = append(messages, toolResultMessage)

					// 发送工具结束执行的信号（带错误）
					outputChan <- createToolEndChunk(toolCall.ID, toolCall.Name, toolCall.Arguments, "", errorMsg)

					continue // 继续处理下一个工具调用（如果有）
				}

				// 执行工具调用
				toolOutputJSON, toolErr := tool.Call(ctx, toolCall.Arguments)
				if toolErr != nil {
					a.logger.Error("Tool execution failed", "toolName", toolCall.Name, "toolID", toolCall.ID, "error", toolErr, "conversationID", conversationID)
					errorMsg := fmt.Sprintf("错误：工具 '%s' 执行失败: %s", toolCall.Name, toolErr.Error())
					toolResultMessage := createToolErrorMessage(toolCall.ID, toolCall.Name, errorMsg)
					messages = append(messages, toolResultMessage)

					// 发送工具结束执行的信号（带错误）
					outputChan <- createToolEndChunk(toolCall.ID, toolCall.Name, toolCall.Arguments, "", errorMsg)

					continue
				}

				// 添加工具调用结果
				toolResultMessage := createToolResultMessage(toolCall.ID, toolCall.Name, toolOutputJSON)
				messages = append(messages, toolResultMessage)
				a.logger.Debug("Tool result added to messages", "toolName", toolCall.Name, "toolID", toolCall.ID, "conversationID", conversationID)

				// 发送工具结束执行的信号（成功）
				outputChan <- createToolEndChunk(toolCall.ID, toolCall.Name, toolCall.Arguments, toolOutputJSON, "")
			}
		}
	}

	// 如果达到最大迭代次数
	outputChan <- handleMaxIterations(a.logger, a.maxIterations, conversationID, lastResponseText, llmThinks, messages)
}

// processToolCallForStreaming 处理单个工具调用，用于流式输出场景
func (a *ToolCallingAgent) processToolCallForStreaming(call llminterface.ToolCall,
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
					// 合并参数，安全处理JSON结构
					mergedArgs, err := mergeJSONArgs(existingCall.Arguments, call.Arguments)
					if err != nil {
						a.logger.Debug("Failed to merge tool call arguments properly", "id", call.ID,
							"error", err, "conversationID", conversationID)
						// 回退到简单拼接，但记录警告
						existingCall.Arguments += call.Arguments
					} else {
						existingCall.Arguments = mergedArgs
					}
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
func (a *ToolCallingAgent) processToolCallArgumentsForStreaming(call llminterface.ToolCall,
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
				// 合并参数，安全处理JSON结构
				mergedArgs, err := mergeJSONArgs(existingCall.Arguments, call.Arguments)
				if err != nil {
					a.logger.Debug("Failed to merge tool call arguments properly", "id", currentToolCallID,
						"error", err, "conversationID", conversationID)
					// 回退到简单拼接，但记录警告
					existingCall.Arguments += call.Arguments
				} else {
					existingCall.Arguments = mergedArgs
				}
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
			// 合并参数，安全处理JSON结构
			mergedArgs, err := mergeJSONArgs(existingCall.Arguments, call.Arguments)
			if err != nil {
				a.logger.Debug("Failed to merge tool call arguments properly", "id", lastID,
					"error", err, "conversationID", conversationID)
				// 回退到简单拼接，但记录警告
				existingCall.Arguments += call.Arguments
			} else {
				existingCall.Arguments = mergedArgs
			}
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

// mergeJSONArgs 尝试合并两个JSON参数片段，以安全的方式处理JSON结构
// 常见情况包括:
// 1. 两个完整的JSON对象合并
// 2. 一个JSON前缀和一个JSON后缀合并
// 3. 处理流式传输中的JSON片段
func mergeJSONArgs(existing, new string) (string, error) {
	// 如果现有参数为空，直接返回新参数
	if existing == "" || existing == "{}" {
		return new, nil
	}

	// 如果新参数为空，直接返回现有参数
	if new == "" {
		return existing, nil
	}

	// 首先尝试作为完整的JSON对象处理
	var existingMap, newMap map[string]any

	// 检查现有参数是否为有效JSON
	if err := json.Unmarshal([]byte(existing), &existingMap); err == nil {
		// 现有参数是有效的JSON对象

		// 检查新参数是否为有效JSON
		if err := json.Unmarshal([]byte(new), &newMap); err == nil {
			// 两者都是有效的JSON对象，合并它们
			maps.Copy(existingMap, newMap)

			// 将合并后的映射转换回JSON字符串
			merged, err := json.Marshal(existingMap)
			if err != nil {
				return "", fmt.Errorf("合并JSON后无法序列化: %w", err)
			}

			return string(merged), nil
		}
	}

	// 如果不能作为完整的JSON对象处理，尝试作为流式JSON片段处理

	// 检查是否是直接拼接可以形成有效JSON的情况
	combined := existing + new
	var combinedMap map[string]any
	if err := json.Unmarshal([]byte(combined), &combinedMap); err == nil {
		return combined, nil
	}

	// 特殊处理常见的流式传输模式，例如拼接不完整的JSON
	// 检查现有字符串是否以花括号开始但没有结束
	if strings.HasPrefix(existing, "{") && !strings.HasSuffix(strings.TrimSpace(existing), "}") {
		// 现有字符串可能是不完整的JSON对象前缀
		// 检查新字符串是否能够完成这个JSON对象
		if !strings.HasPrefix(new, "{") && (strings.Contains(new, "}") || strings.HasSuffix(new, "}")) {
			// 新字符串可能是JSON对象的后缀部分
			combined := existing + new
			var testMap map[string]any
			if err := json.Unmarshal([]byte(combined), &testMap); err == nil {
				return combined, nil
			}
		}
	}

	// 如果上述尝试都失败，回退到简单拼接，并返回错误以便调用者记录
	return existing + new, fmt.Errorf("无法安全合并JSON参数，回退到简单拼接")
}

func composeMessage(messages []llminterface.InputMessage, currentIterationThinks string, toolCallsToExecute []llminterface.ToolCall) []llminterface.InputMessage {
	// 重建 assistant 的响应内容，确保工具调用信息是完整的
	assistantResponseContentParts := make([]llminterface.ContentPart, 0)

	// 添加文本部分（如果有）
	if currentIterationThinks != "" {
		assistantResponseContentParts = append(assistantResponseContentParts, llminterface.ContentPart{
			Type: llminterface.PartTypeText,
			Text: currentIterationThinks,
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
	}
	return messages
}
