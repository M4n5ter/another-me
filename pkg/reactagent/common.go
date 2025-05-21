package reactagent

import (
	"fmt"
	"log/slog"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// UpdateAccumulatedThoughts 更新累积的思考内容，按照迭代格式化
func UpdateAccumulatedThoughts(llmThinks, currentIterationThinks string, iterationNum int) string {
	if iterationNum == 0 {
		// 第一次迭代，不需要添加前导换行符
		return fmt.Sprintf("Iteration %d:\n%s", iterationNum+1, currentIterationThinks)
	} else {
		// 后续迭代，添加换行符分隔
		return fmt.Sprintf("%s\n\nIteration %d:\n%s", llmThinks, iterationNum+1, currentIterationThinks)
	}
}

// CreateFinishChunk 创建一个标记正常结束的输出块
func CreateFinishChunk(finalResponse, accumulatedThoughts string, messages []llminterface.InputMessage, conversationID string) AgentOutputChunk {
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

// CreateErrorChunk 创建一个错误输出块
func CreateErrorChunk(errorMsg string, messages []llminterface.InputMessage, conversationID, accumulatedThoughts string) AgentOutputChunk {
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

// CreateMaxIterChunk 创建一个表示达到最大迭代次数的输出块
func CreateMaxIterChunk(finalResponse, accumulatedThoughts string, maxIterations int, messages []llminterface.InputMessage, conversationID string) AgentOutputChunk {
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

// CreateThoughtChunk 创建当前迭代思考内容的输出块
func CreateThoughtChunk(thoughtContent string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:                      AgentChunkTypeThought,
		CurrentIterThoughtContent: thoughtContent,
	}
}

// CreateReasoningChunk 创建推理内容的输出块
func CreateReasoningChunk(reasoningContent string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:             AgentChunkTypeReasoning,
		ReasoningContent: reasoningContent,
	}
}

// CreateTextChunk 创建文本输出块
func CreateTextChunk(textDelta string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:      AgentChunkTypeText,
		TextDelta: textDelta,
	}
}

// CreateToolStartChunk 创建工具开始执行的信号块
func CreateToolStartChunk(toolID, toolName, toolArguments string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:          AgentChunkTypeToolStart,
		ToolCallID:    toolID,
		ToolName:      toolName,
		ToolArguments: toolArguments,
	}
}

// CreateToolEndChunk 创建工具执行结束的信号块
func CreateToolEndChunk(toolID, toolName, toolArguments, toolResult, errorMsg string) AgentOutputChunk {
	return AgentOutputChunk{
		Type:          AgentChunkTypeToolEnd,
		ToolCallID:    toolID,
		ToolName:      toolName,
		ToolArguments: toolArguments,
		ToolResult:    toolResult,
		Error:         errorMsg,
	}
}

// HandleLLMChatInitError 处理LLM聊天初始错误
func HandleLLMChatInitError(logger *slog.Logger, err error, conversationID string, messages []llminterface.InputMessage) AgentOutputChunk {
	logger.Error("LLM Chat returned an initial error", "error", err, "conversationID", conversationID)
	return CreateErrorChunk(fmt.Sprintf("LLM Chat 初始错误: %v", err), messages, conversationID, "")
}

// HandleLLMStreamError 处理LLM流处理错误
func HandleLLMStreamError(logger *slog.Logger, err error, conversationID string, messages []llminterface.InputMessage) AgentOutputChunk {
	logger.Error("Error in LLM stream chunk", "error", err, "conversationID", conversationID)
	return CreateErrorChunk(fmt.Sprintf("LLM 流处理错误: %v", err), messages, conversationID, "")
}

// HandleMaxIterations 处理达到最大迭代次数的情况
func HandleMaxIterations(logger *slog.Logger, maxIterations int, conversationID string,
	lastResponseText, llmThinks string, messages []llminterface.InputMessage,
) AgentOutputChunk {
	logger.Warn("Max iterations reached", "maxIterations", maxIterations, "conversationID", conversationID)
	return CreateMaxIterChunk(lastResponseText, llmThinks, maxIterations, messages, conversationID)
}

// HandleNoToolCallsFinish 处理没有工具调用时的结束情况
func HandleNoToolCallsFinish(logger *slog.Logger, conversationID, lastResponseText, llmThinks string,
	messages []llminterface.InputMessage, reason string,
) AgentOutputChunk {
	logger.Info("No tool calls, returning final response", "conversationID", conversationID, "reason", reason)
	return CreateFinishChunk(lastResponseText, llmThinks, messages, conversationID)
}

// CreateUserMessage 创建用户消息
func CreateUserMessage(text string) llminterface.InputMessage {
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

// CreateAssistantMessage 创建助手消息
func CreateAssistantMessage(text string) llminterface.InputMessage {
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

// CreateToolResultMessage 创建工具结果消息（工具调用版本）
func CreateToolResultMessage(toolID, toolName, result string) llminterface.InputMessage {
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

// CreateToolErrorMessage 创建工具错误消息（工具调用版本）
func CreateToolErrorMessage(toolID, toolName, errorMsg string) llminterface.InputMessage {
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

// CreateTextBasedToolResultMessage 创建文本格式的工具结果消息（非工具调用版本）
func CreateTextBasedToolResultMessage(toolName, result string) llminterface.InputMessage {
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

// CreateTextBasedToolErrorMessage 创建文本格式的工具错误消息（非工具调用版本）
func CreateTextBasedToolErrorMessage(toolName, errorMsg string) llminterface.InputMessage {
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

// PrepareInitialMessages 准备初始消息
func PrepareInitialMessages(systemPrompt Option[llminterface.InputMessage], userInput string, logger *slog.Logger) []llminterface.InputMessage {
	messages := make([]llminterface.InputMessage, 0)

	// 添加系统提示（如果存在）
	if systemPrompt.IsSome() {
		messages = append(messages, systemPrompt.Unwrap())
		logger.Debug("System prompt added to messages")
	}

	// 添加用户初始输入
	userMessage := CreateUserMessage(userInput)
	messages = append(messages, userMessage)
	logger.Debug("User input added to messages", "userInput", userInput)

	return messages
}

// PrepareCheckpointMessages 准备从检查点恢复的消息
func PrepareCheckpointMessages(checkpoint *llminterface.ChatInput, userInput string, logger *slog.Logger) []llminterface.InputMessage {
	// 使用检查点中的消息历史
	messages := make([]llminterface.InputMessage, len(checkpoint.Messages))
	copy(messages, checkpoint.Messages)

	// 添加新的用户输入
	userMessage := CreateUserMessage(userInput)
	messages = append(messages, userMessage)
	logger.Debug("Checkpoint loaded and new user input added",
		"messagesCount", len(messages),
		"userInput", userInput)

	return messages
}
