package reactagent

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// TextFormatParser 定义了如何从模型返回的文本中提取工具调用的接口
type TextFormatParser interface {
	// ParseToolCalls 从文本中解析出工具调用请求
	// 返回解析出的工具调用列表和解析后的剩余文本
	ParseToolCalls(text string) ([]llminterface.ToolCall, string)
}

// TextBasedAgent 表示一个基于文本格式的 ReAct 智能体，不依赖 function call 能力
type TextBasedAgent struct {
	llmAdapter       llminterface.ChatAdapter
	toolRegistry     *toolcore.Registry
	logger           *slog.Logger
	maxIterations    int
	systemPrompt     Option[llminterface.InputMessage]
	textFormatParser TextFormatParser
}

// TextBasedAgentBuilder 是用于构建 TextBasedAgent 的构建器
type TextBasedAgentBuilder struct {
	llmAdapter       llminterface.ChatAdapter
	toolRegistry     *toolcore.Registry
	logger           *slog.Logger
	maxIterations    int
	systemPrompt     Option[string]
	textFormatParser TextFormatParser
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

	return &TextBasedAgent{
		llmAdapter:       b.llmAdapter,
		toolRegistry:     b.toolRegistry,
		logger:           logger,
		maxIterations:    b.maxIterations,
		systemPrompt:     sysPromptOpt,
		textFormatParser: b.textFormatParser,
	}, nil
}

// Run 方法执行基于文本的 ReAct 代理的核心逻辑
// 它接收初始用户输入，并返回一个用于流式输出的只读通道
func (a *TextBasedAgent) Run(ctx context.Context, userInput, conversationID string) (<-chan AgentOutputChunk, error) {
	a.logger.Info("Text-based ReAct Agent Run started", "conversationID", conversationID, "userInput", userInput)

	// 创建输出通道，用于流式传输 AgentOutputChunk
	outputChan := make(chan AgentOutputChunk, 100)

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

			// 处理 LLM 输出流
			for chunk := range llmOutputChan {
				// 错误处理
				if chunk.Error != nil {
					a.logger.Error("Error in LLM stream chunk", "error", chunk.Error, "conversationID", conversationID)
					outputChan <- AgentOutputChunk{
						Type:   AgentChunkTypeError,
						Error:  fmt.Sprintf("LLM 流处理错误: %v", chunk.Error),
						IsLast: true,
					}
					return
				}

				if chunk.Reasoning.IsSome() {
					outputChan <- AgentOutputChunk{
						Type:           AgentChunkTypeReasoning,
						ThoughtContent: chunk.Reasoning.Unwrap(),
					}
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
					}
				}
			}

			// 发送完整的思考内容
			if llmThinks != "" {
				outputChan <- AgentOutputChunk{
					Type:           AgentChunkTypeThought,
					ThoughtContent: llmThinks,
				}
			}

			lastResponseText = llmThinks // 保存最后一次LLM的文本输出，以备没有工具调用时作为最终结果

			// 解析LLM响应中的工具调用
			toolCalls, remainingText := a.textFormatParser.ParseToolCalls(llmThinks)

			// 如果没有工具调用，结束循环并返回最终响应
			if len(toolCalls) == 0 {
				a.logger.Info("No tool calls parsed, returning final response", "conversationID", conversationID)
				outputChan <- AgentOutputChunk{
					Type:          AgentChunkTypeFinish,
					FinalResponse: remainingText,
					IsLast:        true,
				}
				return
			}

			// 将LLM的完整回复添加到消息历史
			assistantMessage := llminterface.InputMessage{
				Role: llminterface.RoleAssistant,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: llmThinks,
					},
				},
			}
			messages = append(messages, assistantMessage)
			a.logger.Debug("Assistant's full response added to messages", "conversationID", conversationID)

			// 执行工具调用
			for _, toolCall := range toolCalls {
				// 发送工具开始执行的信号
				outputChan <- AgentOutputChunk{
					Type:          AgentChunkTypeToolStart,
					ToolCallID:    toolCall.ID,
					ToolName:      toolCall.Name,
					ToolArguments: toolCall.Arguments,
				}

				a.logger.Info("Executing tool", "toolName", toolCall.Name, "toolID", toolCall.ID, "argsLength", len(toolCall.Arguments), "conversationID", conversationID)

				tool, exists := a.toolRegistry.Get(toolCall.Name)
				if !exists {
					a.logger.Error("Tool not found in registry", "toolName", toolCall.Name, "conversationID", conversationID)
					toolResultMessage := llminterface.InputMessage{
						Role: llminterface.RoleUser,
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
						Type:          AgentChunkTypeToolEnd,
						ToolCallID:    toolCall.ID,
						ToolName:      toolCall.Name,
						ToolArguments: toolCall.Arguments,
						Error:         fmt.Sprintf("错误：工具 '%s' 未找到。", toolCall.Name),
					}

					continue
				}

				// 执行工具调用
				toolOutputJSON, toolErr := tool.Call(ctx, toolCall.Arguments)
				if toolErr != nil {
					a.logger.Error("Tool execution failed", "toolName", toolCall.Name, "toolID", toolCall.ID, "error", toolErr, "conversationID", conversationID)
					toolResultMessage := llminterface.InputMessage{
						Role: llminterface.RoleUser,
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
						Type:          AgentChunkTypeToolEnd,
						ToolCallID:    toolCall.ID,
						ToolName:      toolCall.Name,
						ToolArguments: toolCall.Arguments,
						Error:         fmt.Sprintf("错误：工具 '%s' 执行失败: %s", toolCall.Name, toolErr.Error()),
					}

					continue
				}

				// 添加工具调用结果
				toolResultMessage := llminterface.InputMessage{
					Role: llminterface.RoleUser,
					Content: []llminterface.ContentPart{
						{
							Type: llminterface.PartTypeText,
							Text: fmt.Sprintf("工具 '%s' 执行结果:\n%s", toolCall.Name, toolOutputJSON),
						},
					},
				}
				messages = append(messages, toolResultMessage)
				a.logger.Debug("Tool result added to messages", "toolName", toolCall.Name, "toolID", toolCall.ID, "conversationID", conversationID)

				// 发送工具结束执行的信号（成功）
				outputChan <- AgentOutputChunk{
					Type:          AgentChunkTypeToolEnd,
					ToolCallID:    toolCall.ID,
					ToolName:      toolCall.Name,
					ToolArguments: toolCall.Arguments,
					ToolResult:    toolOutputJSON,
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
