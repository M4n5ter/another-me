package reactagent

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// TextFormatParser 定义了如何从模型返回的文本中提取工具调用的接口
type TextFormatParser interface {
	// ParseToolCalls 从文本中解析出工具调用请求
	// 返回解析出的工具调用列表和解析后的剩余文本
	ParseToolCalls(text string) ([]llminterface.ToolCall, string)
	// ExceptFormat 返回解析器期望的格式
	ExceptFormat() string
}

// TextBasedAgent 表示一个基于文本格式的 ReAct 智能体，不依赖 function call 能力
type TextBasedAgent struct {
	llmAdapter       llminterface.ChatAdapter
	taskEvaluator    Option[llminterface.ChatAdapter]
	toolRegistry     *toolcore.Registry
	logger           *slog.Logger
	maxIterations    int
	systemPrompt     Option[llminterface.InputMessage]
	textFormatParser TextFormatParser
	checkpoint       Option[*llminterface.ChatInput] // 可选的检查点，用于继续执行
}

// TextBasedAgentBuilder 是用于构建 TextBasedAgent 的构建器
type TextBasedAgentBuilder struct {
	llmAdapter       llminterface.ChatAdapter
	taskEvaluator    Option[llminterface.ChatAdapter]
	toolRegistry     *toolcore.Registry
	logger           *slog.Logger
	maxIterations    int
	systemPrompt     Option[string]
	textFormatParser TextFormatParser
	checkpoint       Option[*llminterface.ChatInput]
}

// NewTextBasedAgentBuilder 创建一个新的 TextBasedAgentBuilder 实例
func NewTextBasedAgentBuilder() *TextBasedAgentBuilder {
	return &TextBasedAgentBuilder{
		maxIterations: 10, // 默认最大迭代次数
	}
}

// WithLLMAdapter 设置 LLM 适配器
func (b *TextBasedAgentBuilder) WithLLMAdapter(adapter llminterface.ChatAdapter) *TextBasedAgentBuilder {
	b.llmAdapter = adapter
	return b
}

// WithTaskEvaluator 设置任务评估器
func (b *TextBasedAgentBuilder) WithTaskEvaluator(evaluator llminterface.ChatAdapter) *TextBasedAgentBuilder {
	b.taskEvaluator = Some(evaluator)
	return b
}

// WithToolRegistry 设置工具注册表
func (b *TextBasedAgentBuilder) WithToolRegistry(registry *toolcore.Registry) *TextBasedAgentBuilder {
	b.toolRegistry = registry
	return b
}

// WithLogger 设置日志记录器
func (b *TextBasedAgentBuilder) WithLogger(logger *slog.Logger) *TextBasedAgentBuilder {
	b.logger = logger
	return b
}

// WithMaxIterations 设置最大迭代次数
func (b *TextBasedAgentBuilder) WithMaxIterations(maxIter int) *TextBasedAgentBuilder {
	if maxIter > 0 {
		b.maxIterations = maxIter
	}
	return b
}

// WithSystemPrompt 设置系统提示
func (b *TextBasedAgentBuilder) WithSystemPrompt(prompt string) *TextBasedAgentBuilder {
	b.systemPrompt = Some(prompt)
	return b
}

// WithTextFormatParser 设置文本格式解析器
func (b *TextBasedAgentBuilder) WithTextFormatParser(parser TextFormatParser) *TextBasedAgentBuilder {
	b.textFormatParser = parser
	return b
}

// WithCheckpoint 设置检查点
func (b *TextBasedAgentBuilder) WithCheckpoint(checkpoint *llminterface.ChatInput) *TextBasedAgentBuilder {
	b.checkpoint = Some(checkpoint)
	return b
}

// Build 构建并返回一个 TextBasedAgent 实例
func (b *TextBasedAgentBuilder) Build() (*TextBasedAgent, error) {
	if b.llmAdapter == nil {
		return nil, fmt.Errorf("LLMAdapter 不能为空")
	}
	if b.toolRegistry == nil {
		return nil, fmt.Errorf("ToolRegistry 不能为空")
	}
	if b.textFormatParser == nil {
		return nil, fmt.Errorf("TextFormatParser 不能为空")
	}

	logger := b.logger
	if logger == nil {
		logger = slog.Default().WithGroup("text_based_react_agent")
	}

	// 文本格式解析器提示词
	textFormatParserPrompt := fmt.Sprintf("Expected format:\n%s", b.textFormatParser.ExceptFormat())

	// 工具注册表提示词
	toolRegistryPrompt, err := toolRegistryToPrompt(b.toolRegistry)
	if err != nil {
		return nil, fmt.Errorf("failed to create tool registry prompt: %w", err)
	}

	// 系统提示
	var sysPromptOpt Option[llminterface.InputMessage]
	if b.systemPrompt.IsSome() {
		sysPromptOpt = Some(llminterface.InputMessage{
			Role: llminterface.RoleSystem,
			Content: []llminterface.ContentPart{
				{
					Type: llminterface.PartTypeText,
					Text: fmt.Sprintf("%s\n\n%s\n\n%s", b.systemPrompt.Unwrap(), textFormatParserPrompt, toolRegistryPrompt),
				},
			},
		})
	}

	return &TextBasedAgent{
		llmAdapter:       b.llmAdapter,
		taskEvaluator:    b.taskEvaluator,
		toolRegistry:     b.toolRegistry,
		logger:           logger,
		maxIterations:    b.maxIterations,
		systemPrompt:     sysPromptOpt,
		textFormatParser: b.textFormatParser,
		checkpoint:       b.checkpoint,
	}, nil
}

var _ ReAct = (*TextBasedAgent)(nil)

// Run 方法执行基于文本的 ReAct 代理的核心逻辑
// 它接收初始用户输入，并返回一个用于流式输出的只读通道
func (a *TextBasedAgent) Run(ctx context.Context, userInput, conversationID string) (<-chan AgentOutputChunk, error) {
	a.logger.Info("Text-based ReAct Agent Run started", "conversationID", conversationID, "userInput", userInput)

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
func (a *TextBasedAgent) ContinueFromCheckpoint(ctx context.Context, userInput string, checkpoint *llminterface.ChatInput) (<-chan AgentOutputChunk, error) {
	if checkpoint == nil {
		return nil, fmt.Errorf("检查点不能为空")
	}

	// 使用传递的会话ID
	conversationID := checkpoint.ConversationID
	a.logger.Info("Text-based ReAct Agent ContinueFromCheckpoint started", "conversationID", conversationID, "userInput", userInput)

	// 创建输出通道，用于流式传输 AgentOutputChunk
	outputChan := make(chan AgentOutputChunk, 100)

	// 启动一个 goroutine 执行 ReAct 循环，并通过 outputChan 流式传输结果
	go func() {
		defer close(outputChan) // 确保在函数退出时关闭通道

		messages := prepareCheckpointMessages(checkpoint, userInput, a.logger)

		// 执行ReAct循环核心逻辑
		a.handleReactLoop(ctx, messages, conversationID, userInput, outputChan)
	}()

	return outputChan, nil
}

// handleReactLoop 处理共享的React循环逻辑
func (a *TextBasedAgent) handleReactLoop(
	ctx context.Context,
	messages []llminterface.InputMessage,
	conversationID string,
	initialUserInput string,
	outputChan chan<- AgentOutputChunk,
) {
	var lastResponseText string
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

		// 处理 LLM 输出流
		for chunk := range llmOutputChan {
			// 错误处理
			if chunk.Error != nil && !errors.Is(chunk.Error, context.Canceled) {
				outputChan <- handleLLMStreamError(a.logger, chunk.Error, conversationID, messages)
				return
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
				}
			}

			switch {
			case ctx.Err() != nil:
				// 将LLM的完整回复添加到消息历史
				assistantMessage := createAssistantMessage(currentIterationThinks)
				messages = append(messages, assistantMessage)
				a.logger.Info("Context canceled, exiting loop, sending last chunk", "conversationID", conversationID, "why", ctx.Err().Error())
				// 上下文取消不认为是错误块，而是正常结束
				outputChan <- createFinishChunk(currentIterationThinks, llmThinks+currentIterationThinks, messages, conversationID)
				return
			default:
				continue
			}
		}

		// 将LLM的完整回复添加到消息历史
		assistantMessage := createAssistantMessage(currentIterationThinks)
		messages = append(messages, assistantMessage)
		a.logger.Debug("Assistant's full response added to messages", "conversationID", conversationID)

		// 发送当前迭代的LLM思考内容
		if currentIterationThinks != "" {
			outputChan <- createThoughtChunk(currentIterationThinks)
		}

		lastResponseText = currentIterationThinks // 保存最后一次LLM的文本输出，以备没有工具调用时作为最终结果

		// 更新累积的思考内容
		llmThinks = updateAccumulatedThoughts(llmThinks, currentIterationThinks, i)

		// 解析LLM响应中的工具调用
		toolCalls, remainingText := a.textFormatParser.ParseToolCalls(currentIterationThinks)

		// 如果没有工具调用，尝试使用任务评估器
		if len(toolCalls) == 0 {
			evaluated, needContinue := taskEvaluate(ctx, a.logger, a.taskEvaluator, initialUserInput, llmThinks, remainingText, &messages, outputChan, conversationID)
			if evaluated {
				if needContinue {
					continue
				}

				outputChan <- createFinishChunk(lastResponseText, llmThinks, messages, conversationID)
				return
			}

			// 评估器不存在或者没能正确评估
			if !evaluated {
				a.logger.Info("No tool calls parsed, returning final response", "conversationID", conversationID)
				outputChan <- createFinishChunk(remainingText, llmThinks, messages, conversationID)
				return
			}
		}

		// 执行工具调用
		if len(toolCalls) > 0 {
			var allToolExecutionOutputs []string // 存储每个工具的执行输出或错误信息

			for _, toolCall := range toolCalls {
				// 发送工具开始执行的信号
				outputChan <- createToolStartChunk(toolCall.ID, toolCall.Name, toolCall.Arguments)

				a.logger.Info("Executing tool", "toolName", toolCall.Name, "toolID", toolCall.ID, "argsLength", len(toolCall.Arguments), "conversationID", conversationID)

				tool, exists := a.toolRegistry.Get(toolCall.Name)
				var currentToolOutputForLLM string
				var toolResultForChunk string // For createToolEndChunk
				var errorMsgForChunk string   // For createToolEndChunk

				if !exists {
					a.logger.Error("Tool not found in registry", "toolName", toolCall.Name, "conversationID", conversationID)
					errMsg := fmt.Sprintf("未找到工具: %s", toolCall.Name)
					// 格式化字符串，加到LLM的输出中
					currentToolOutputForLLM = fmt.Sprintf("错误：工具 '%s' %s", toolCall.Name, errMsg)
					errorMsgForChunk = fmt.Sprintf("错误：工具 '%s' 未找到。", toolCall.Name)
				} else {
					// 执行工具调用
					toolOutputJSON, toolErr := tool.Call(ctx, toolCall.Arguments)
					if toolErr != nil {
						a.logger.Error("Tool execution failed", "toolName", toolCall.Name, "toolID", toolCall.ID, "error", toolErr, "conversationID", conversationID)
						errMsg := fmt.Sprintf("执行失败: %s", toolErr.Error())
						currentToolOutputForLLM = fmt.Sprintf("错误：工具 '%s' %s", toolCall.Name, errMsg)
						errorMsgForChunk = fmt.Sprintf("错误：工具 '%s' 执行失败: %s", toolCall.Name, toolErr.Error())
					} else {
						currentToolOutputForLLM = fmt.Sprintf("工具 '%s' 执行结果:\n%s", toolCall.Name, toolOutputJSON)
						toolResultForChunk = toolOutputJSON
					}
				}
				allToolExecutionOutputs = append(allToolExecutionOutputs, currentToolOutputForLLM)
				// 发送工具结束执行的信号（成功或失败）
				outputChan <- createToolEndChunk(toolCall.ID, toolCall.Name, toolCall.Arguments, toolResultForChunk, errorMsgForChunk)
			}

			// 所有工具执行完成后，将它们的组合结果添加到消息历史中
			if len(allToolExecutionOutputs) > 0 {
				aggregatedResultsText := strings.Join(allToolExecutionOutputs, "\n\n")
				// 创建一个包含所有工具执行结果的单个用户消息（避免相同角色的消息连续出现，一些模型不支持，比如deepseek-reasoner）
				// 这个消息代表了执行后的观察结果
				aggregatedObservationMessage := llminterface.InputMessage{
					Role: llminterface.RoleUser, // 这是从用户角度来看的"观察"部分，用于LLM
					Content: []llminterface.ContentPart{
						{
							Type: llminterface.PartTypeText,
							Text: aggregatedResultsText,
						},
					},
				}
				messages = append(messages, aggregatedObservationMessage)
				a.logger.Debug("Aggregated tool results added to messages as a single user message", "count", len(allToolExecutionOutputs), "conversationID", conversationID)
			}
		}
	}

	// 如果达到最大迭代次数
	outputChan <- handleMaxIterations(a.logger, a.maxIterations, conversationID, lastResponseText, llmThinks, messages)
}
