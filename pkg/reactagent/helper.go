package reactagent

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// updateAccumulatedThoughts 更新累积的思考内容，按照迭代格式化
func updateAccumulatedThoughts(llmThinks, currentIterationThinks string, iterationNum int) string {
	if iterationNum == 0 {
		// 第一次迭代，不需要添加前导换行符
		return fmt.Sprintf("Iteration %d:\n%s", iterationNum+1, currentIterationThinks)
	} else {
		// 后续迭代，添加换行符分隔
		return fmt.Sprintf("%s\n\nIteration %d:\n%s", llmThinks, iterationNum+1, currentIterationThinks)
	}
}

// createFinishChunk 创建一个标记正常结束的输出块
func createFinishChunk(finalResponse, accumulatedThoughts string, messages []llminterface.InputMessage, conversationID string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:                AgentChunkTypeFinish,
		FinalResponse:       finalResponse,
		AccumulatedThoughts: accumulatedThoughts,
		MessageHistory: &llminterface.ChatInput{
			Messages:       messages,
			ConversationID: conversationID,
		},
		IsLast: true,
	}
}

// createErrorChunk 创建一个错误输出块
func createErrorChunk(errorMsg string, messages []llminterface.InputMessage, conversationID, accumulatedThoughts string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:                AgentChunkTypeError,
		Error:               errorMsg,
		AccumulatedThoughts: accumulatedThoughts,
		MessageHistory: &llminterface.ChatInput{
			Messages:       messages,
			ConversationID: conversationID,
		},
		IsLast: true,
	}
}

// createMaxIterChunk 创建一个表示达到最大迭代次数的输出块
func createMaxIterChunk(finalResponse, accumulatedThoughts string, maxIterations int, messages []llminterface.InputMessage, conversationID string) AgentOutputChunk {
	var errMsg string
	if finalResponse != "" {
		errMsg = fmt.Sprintf("达到最大迭代次数 (%d)，返回最后观察到的LLM输出", maxIterations)
	} else {
		errMsg = fmt.Sprintf("达到最大迭代次数 (%d) 且没有LLM的最终响应", maxIterations)
	}

	return AgentOutputChunk{
		Type:                AgentChunkTypeMaxIter,
		FinalResponse:       finalResponse,
		AccumulatedThoughts: accumulatedThoughts,
		Error:               errMsg,
		MessageHistory: &llminterface.ChatInput{
			Messages:       messages,
			ConversationID: conversationID,
		},
		IsLast: true,
	}
}

// createThoughtChunk 创建当前迭代思考内容的输出块
func createThoughtChunk(thoughtContent string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:                      AgentChunkTypeThought,
		CurrentIterThoughtContent: thoughtContent,
	}
}

// createReasoningChunk 创建推理内容的输出块
func createReasoningChunk(reasoningContent string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:             AgentChunkTypeReasoning,
		ReasoningContent: reasoningContent,
	}
}

// createTextChunk 创建文本输出块
func createTextChunk(textDelta string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:      AgentChunkTypeText,
		TextDelta: textDelta,
	}
}

// createToolStartChunk 创建工具开始执行的信号块
func createToolStartChunk(toolID, toolName, toolArguments string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:          AgentChunkTypeToolStart,
		ToolCallID:    toolID,
		ToolName:      toolName,
		ToolArguments: toolArguments,
	}
}

// createToolEndChunk 创建工具执行结束的信号块
func createToolEndChunk(toolID, toolName, toolArguments, toolResult, errorMsg string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:          AgentChunkTypeToolEnd,
		ToolCallID:    toolID,
		ToolName:      toolName,
		ToolArguments: toolArguments,
		ToolResult:    toolResult,
		Error:         errorMsg,
	}
}

// handleLLMChatInitError 处理LLM聊天初始错误
func handleLLMChatInitError(logger *slog.Logger, err error, conversationID string, messages []llminterface.InputMessage) AgentOutputChunk {
	logger.Error("LLM Chat returned an initial error", "error", err, "conversationID", conversationID)
	return createErrorChunk(fmt.Sprintf("LLM Chat 初始错误: %v", err), messages, conversationID, "")
}

// handleLLMStreamError 处理LLM流处理错误
func handleLLMStreamError(logger *slog.Logger, err error, conversationID string, messages []llminterface.InputMessage) AgentOutputChunk {
	logger.Error("Error in LLM stream chunk", "error", err, "conversationID", conversationID)
	return createErrorChunk(fmt.Sprintf("LLM 流处理错误: %v", err), messages, conversationID, "")
}

// handleMaxIterations 处理达到最大迭代次数的情况
func handleMaxIterations(logger *slog.Logger, maxIterations int, conversationID string,
	lastResponseText, llmThinks string, messages []llminterface.InputMessage,
) AgentOutputChunk {
	logger.Warn("Max iterations reached", "maxIterations", maxIterations, "conversationID", conversationID)
	return createMaxIterChunk(lastResponseText, llmThinks, maxIterations, messages, conversationID)
}

// createUserMessage 创建用户消息
func createUserMessage(text string) llminterface.InputMessage {
	return llminterface.InputMessage{
		Role: llminterface.RoleUser,
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeText,
				Text: text,
			},
		},
	}
}

// createAssistantMessage 创建助手消息
func createAssistantMessage(text string) llminterface.InputMessage {
	return llminterface.InputMessage{
		Role: llminterface.RoleAssistant,
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeText,
				Text: text,
			},
		},
	}
}

// createToolResultMessage 创建工具结果消息（工具调用版本）
func createToolResultMessage(toolID, toolName, result string) llminterface.InputMessage {
	return llminterface.InputMessage{
		Role:       llminterface.RoleToolResult,
		ToolCallID: Some(toolID),
		ToolName:   Some(toolName),
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeText,
				Text: result,
			},
		},
	}
}

// createToolErrorMessage 创建工具错误消息（工具调用版本）
func createToolErrorMessage(toolID, toolName, errorMsg string) llminterface.InputMessage {
	return llminterface.InputMessage{
		Role:       llminterface.RoleToolResult,
		ToolCallID: Some(toolID),
		ToolName:   Some(toolName),
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeText,
				Text: errorMsg,
			},
		},
	}
}

// createTextBasedToolResultMessage 创建文本格式的工具结果消息（非工具调用版本）
func createTextBasedToolResultMessage(toolName, result string) llminterface.InputMessage {
	return llminterface.InputMessage{
		Role: llminterface.RoleUser,
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeText,
				Text: fmt.Sprintf("工具 '%s' 执行结果:\n%s", toolName, result),
			},
		},
	}
}

// createTextBasedToolErrorMessage 创建文本格式的工具错误消息（非工具调用版本）
func createTextBasedToolErrorMessage(toolName, errorMsg string) llminterface.InputMessage {
	return llminterface.InputMessage{
		Role: llminterface.RoleUser,
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeText,
				Text: fmt.Sprintf("错误：工具 '%s' %s", toolName, errorMsg),
			},
		},
	}
}

// prepareInitialMessages 准备初始消息
func prepareInitialMessages(systemPrompt Option[llminterface.InputMessage], userInput string, logger *slog.Logger) []llminterface.InputMessage {
	messages := make([]llminterface.InputMessage, 0)

	// 添加系统提示（如果存在）
	if systemPrompt.IsSome() {
		messages = append(messages, systemPrompt.Unwrap())
		logger.Debug("System prompt added to messages")
	}

	// 添加用户初始输入
	userMessage := createUserMessage(userInput)
	messages = append(messages, userMessage)
	logger.Debug("User input added to messages", "userInput", userInput)

	return messages
}

// prepareCheckpointMessages 准备从检查点恢复的消息
func prepareCheckpointMessages(checkpoint *llminterface.ChatInput, userInput string, logger *slog.Logger) []llminterface.InputMessage {
	// 使用检查点中的消息历史
	messages := make([]llminterface.InputMessage, len(checkpoint.Messages))
	copy(messages, checkpoint.Messages)

	// 添加新的用户输入
	userMessage := createUserMessage(userInput)
	messages = append(messages, userMessage)
	logger.Debug("Checkpoint loaded and new user input added",
		"messagesCount", len(messages),
		"userInput", userInput)

	return messages
}

// taskEvaluate 执行任务评估
func taskEvaluate(ctx context.Context, logger *slog.Logger, taskEvaluator Option[llminterface.ChatAdapter], initialUserInput, llmThinks, lastResponseText string, messages *[]llminterface.InputMessage, outputChan chan<- AgentOutputChunk, conversationID string) (evaluated, needContinue bool) {
	evaluated = false
	needContinue = false

	if taskEvaluator.IsSome() {
		evaluator := taskEvaluator.Unwrap()
		logger.Info("Task evaluator is present. Evaluating current progress.", "conversationID", conversationID)

		if evaluatorPromptMsg, err := prepareEvaluatorPrompt(initialUserInput, llmThinks); err == nil {
			if evaluatorResponse, err := llminterface.ChatAndGetFullResponse(ctx, evaluator, llminterface.ChatInput{
				Messages: []llminterface.InputMessage{evaluatorPromptMsg},
			}); err == nil {
				if isCompleted, feedback, err := parseEvaluatorResponse(evaluatorResponse.FullText); err == nil {
					logger.Info("Task evaluator response", "isCompleted", isCompleted, "feedback", feedback, "conversationID", conversationID)

					if isCompleted {
						outputChan <- createFinishChunk(lastResponseText, llmThinks, *messages, conversationID)
						evaluated = true
						return
					}

					*messages = append(*messages, llminterface.InputMessage{
						Role: llminterface.RoleUser,
						Content: []llminterface.ContentPart{
							{Type: llminterface.PartTypeText, Text: fmt.Sprintf("Task Manager Says:\n\n%s", feedback)},
						},
					})

					evaluated = true
					needContinue = true
				} else {
					logger.Error("Failed to parse evaluator response", "error", err, "conversationID", conversationID)
				}
			} else {
				logger.Error("Failed to get evaluator response", "error", err, "conversationID", conversationID)
			}
		} else {
			logger.Error("Failed to prepare evaluator prompt", "error", err, "conversationID", conversationID)
		}
	}

	return
}

// prepareEvaluatorPrompt 准备评估器提示
func prepareEvaluatorPrompt(initialUserInput, llmThinks string) (llminterface.InputMessage, error) {
	promptTemplate := i18n.GlobalManager.T(context.Background(), "evaluator.prompt", nil)

	prompt := strings.ReplaceAll(promptTemplate, "{userInput}", initialUserInput)
	prompt = strings.ReplaceAll(prompt, "{llmThinks}", llmThinks)

	return llminterface.InputMessage{
		Role: llminterface.RoleSystem,
		Content: []llminterface.ContentPart{
			{Type: llminterface.PartTypeText, Text: prompt},
		},
	}, nil
}

// parseEvaluatorResponse 解析评估器响应
func parseEvaluatorResponse(responseText string) (isCompleted bool, feedback string, err error) {
	statusRegex := regexp.MustCompile(`<completion_status>(COMPLETED|NOT_COMPLETED)</completion_status>`)
	feedbackRegex := regexp.MustCompile(`<feedback_if_not_completed>(.*?)</feedback_if_not_completed>`)

	statusMatch := statusRegex.FindStringSubmatch(responseText)
	if len(statusMatch) < 2 {
		return false, "", fmt.Errorf("could not find or parse <completion_status> tag in response: %s", responseText)
	}

	if statusMatch[1] == "COMPLETED" {
		isCompleted = true
	} else {
		isCompleted = false
	}

	if !isCompleted {
		feedbackMatch := feedbackRegex.FindStringSubmatch(responseText)
		// feedbackMatch can be nil if the tag is not present, which is fine.
		if len(feedbackMatch) > 1 {
			feedback = strings.TrimSpace(feedbackMatch[1])
		}
	}
	return isCompleted, feedback, nil
}
