package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/deepseek"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface/eino"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/fetchtool"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// 使用全局i18n服务
	i18nService := i18n.GlobalManager

	// 创建上下文
	ctx := context.Background()

	// 创建工具注册表并注册工具
	registry := toolcore.NewRegistry()

	// 注册fetch工具
	fetchTool := fetchtool.NewFetchTool(i18nService)
	err := registry.Register(ctx, fetchTool)
	if err != nil {
		logger.Error("Failed to register fetch tool", "error", err)
		os.Exit(1)
	}
	logger.Info("已注册工具", "toolName", "fetch")

	// 从环境变量获取API密钥
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		logger.Error("API Key is not set")
		os.Exit(1)
	}

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:      apiKey,
		Model:       "deepseek-reasoner", // 不支持 function call
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		logger.Error("Failed to create deepseek model", "error", err)
		os.Exit(1)
	}

	llmAdapter, err := eino.NewNoToolChatAdapter(ctx, chatModel)
	if err != nil {
		logger.Error("Failed to create no-tool LLM adapter", "error", err)
		os.Exit(1)
	}

	systemPrompt := `你是一个强大的助手，需要使用工具来帮助用户完成任务。

对于所有任务，你必须使用以下Markdown代码块格式来调用工具：

首先，分析问题：
我需要解决的问题是...

然后，必须使用Markdown代码块来调用工具，格式如下：
` + "```" + `工具名称
{"参数1": "值1", "参数2": "值2"}
` + "```" + `

可用的工具有：
1. fetch - 从网络获取内容
   参数: 
   - url: 要获取的URL (必需参数，字符串格式)
   - start_index: 返回内容的起始索引 (可选参数，数字格式，默认0)
   - max_length: 最大返回字符数 (可选参数，数字格式，默认10000)
   - raw: 是否返回原始内容 (可选参数，布尔值格式，默认false)
   - ignore_robots_txt: 是否忽略 robots.txt 限制 (可选参数，布尔值格式，默认false)
   - user_agent: 自定义 User-Agent 字符串 (可选参数，字符串格式)
   - proxy_url: 代理URL (可选参数，字符串格式，例如 socks5://127.0.0.1:55535)
   - is_manual_request: 是否由用户操作手动触发 (可选参数，布尔值格式，默认false)

例如，要获取某网站的内容，应该这样调用：

我需要获取网站内容，应该使用fetch工具。

` + "```" + `fetch
{"url": "https://example.com", "ignore_robots_txt": true}
` + "```" + `

你必须严格按照这个Markdown代码块格式输出工具调用，否则系统将无法执行并处理你的请求。必须记住所有的工具调用都不是来自用户，而是由你通过工具调用操控系统来执行的。
`
	parser := &reactagent.MarkdownFormatParser{}
	agentBuilder := reactagent.NewTextBasedAgentBuilder()
	agent, err := agentBuilder.
		WithLLMAdapter(llmAdapter).
		WithToolRegistry(registry).
		WithLogger(logger).
		WithSystemPrompt(systemPrompt).
		WithTextFormatParser(parser).
		WithMaxIterations(10).
		Build()
	if err != nil {
		logger.Error("Failed to create agent", "error", err)
		os.Exit(1)
	}

	userInput := `获取百度 (https://www.baidu.com) 的首页内容并总结。必须无视 robots.txt 的限制。`
	conversationID := fmt.Sprintf("text-agent-%d", time.Now().Unix())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(5 * time.Second)
		cancel()
	}()

	fmt.Println("开始执行任务:", userInput)
	outputChan, err := agent.Run(ctx, userInput, conversationID)
	if err != nil {
		logger.Error("Failed to run agent", "error", err)
		os.Exit(1)
	}

	for chunk := range outputChan {
		switch chunk.Type {
		case reactagent.AgentChunkTypeText:
			// 流式打印文本
			fmt.Print(chunk.TextDelta)

		case reactagent.AgentChunkTypeReasoning:
			// 打印推理内容
			fmt.Print(chunk.ReasoningContent)

		case reactagent.AgentChunkTypeToolStart:
			logger.Info("执行工具开始", "toolName", chunk.ToolName, "arguments", chunk.ToolArguments)
			fmt.Printf("\n[执行工具: %s %s]\n", chunk.ToolName, chunk.ToolArguments)

		case reactagent.AgentChunkTypeToolEnd:
			if chunk.Error != "" {
				logger.Error("工具执行失败", "toolName", chunk.ToolName, "error", chunk.Error)
				fmt.Printf("\n[工具执行失败: %s - %s]\n", chunk.ToolName, chunk.Error)
			} else {
				logger.Info("工具执行完成", "toolName", chunk.ToolName, "resultLength", len(chunk.ToolResult))
				fmt.Printf("\n[工具执行完成: %s]\n", chunk.ToolName)
				// 打印工具调用结果前1000个字符以便调试
				resultPreview := chunk.ToolResult
				if len(resultPreview) > 1000 {
					resultPreview = resultPreview[:1000] + "... (内容截断)"
				}
				fmt.Printf("\n[工具结果预览]\n%s\n", resultPreview)
			}

		case reactagent.AgentChunkTypeFinish:
			logger.Info("Agent完成", "finalResponse", chunk.FinalResponse)

		case reactagent.AgentChunkTypeError:
			logger.Error("Agent执行错误", "error", chunk.Error)

		case reactagent.AgentChunkTypeMaxIter:
			logger.Warn("达到最大迭代次数", "error", chunk.Error)
		}

		if chunk.IsLast {
			fmt.Println(chunk.AccumulatedThoughts)
			logger.Info("Agent执行结束")
			break
		}
	}
}
