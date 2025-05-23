package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/cloudwego/eino-ext/components/model/deepseek"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface/eino"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/browsertool"
	"github.com/m4n5ter/another-me/pkg/tools/fetchtool"
)

func main() {
	// 输出标题信息
	fmt.Println("Browser Agent 基础示例")
	fmt.Println("=================")
	fmt.Println()

	// 运行基础示例
	runBasicExample()
}

// runBasicExample 运行基础浏览器示例
func runBasicExample() {
	ctx := context.Background()

	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 创建工具注册表
	registry := toolcore.NewRegistry()

	// 注册工具
	registerTools(registry, i18n.GlobalManager)

	// 设置 eino 模型
	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:      os.Getenv("DEEPSEEK_API_KEY"),
		Model:       "deepseek-chat",
		MaxTokens:   4096,
		Temperature: 0.1,
	})
	if err != nil {
		logger.Error("创建eino模型失败", "error", err)
		os.Exit(1)
	}

	// 创建 eino 适配器
	chatAdapter, err := eino.NewChatAdapter(ctx, chatModel, registry)
	if err != nil {
		logger.Error("创建eino适配器失败", "error", err)
		os.Exit(1)
	}

	// 创建Browser Agent
	browserAgent, err := reactagent.NewToolCallingAgentBuilder().
		WithLLMAdapter(chatAdapter).
		WithToolRegistry(registry).
		WithLogger(logger.WithGroup("browser_agent_main")).
		WithMaxIterations(100).
		WithSystemPrompt(i18n.GlobalManager.T(ctx, "browser.prompt.default", nil)).
		Build()
	if err != nil {
		logger.Error("创建Browser agent失败", "error", err)
		os.Exit(1)
	}

	userInput := "访问 Hacker News 首页，逐个浏览首页的帖子，然后立即开始进入每个帖子查看，每次都需要进行工具调用，除非任务完成，每完成一个帖子，总结一份报告。中途不要询问我，直到任务完成。"
	conversationID := "browser-agent-demo-001"

	fmt.Println("执行浏览器任务...")
	fmt.Printf("任务: %s\n\n", userInput)

	outputChan, err := browserAgent.Run(ctx, userInput, conversationID)
	if err != nil {
		logger.Error("运行Browser agent失败", "error", err)
		os.Exit(1)
	}

	taskCompleted := handleAgentOutput(outputChan)

	// 任务完成后等待用户按Enter键关闭浏览器
	if taskCompleted {
		fmt.Println("\n任务已完成，按Enter键关闭浏览器...")
		_, err = fmt.Scanln() // 等待用户按Enter
		if err != nil {
			logger.Error("等待用户按Enter失败", "error", err)
		}
	}

	// 关闭浏览器
	closeBrowserInRegistry(registry)
}

// handleAgentOutput 处理Agent的输出流，返回任务是否成功完成
func handleAgentOutput(outputChan <-chan reactagent.AgentOutputChunk) bool {
	taskCompleted := false

	for chunk := range outputChan {
		switch chunk.Type {
		case reactagent.AgentChunkTypeText:
			// 立即显示增量文本
			fmt.Print(chunk.TextDelta)

		case reactagent.AgentChunkTypeReasoning:
			// 打印推理内容
			fmt.Print(chunk.CurrentIterThoughtContent)

		case reactagent.AgentChunkTypeToolStart:
			// 显示工具正在执行的指示
			fmt.Printf("\n[执行工具: %s %s]\n", chunk.ToolName, chunk.ToolArguments)

		case reactagent.AgentChunkTypeToolEnd:
			// 显示工具执行完成的指示
			if chunk.Error != "" {
				fmt.Printf("\n[工具执行失败: %s - %s]\n", chunk.ToolName, chunk.Error)
			} else {
				fmt.Printf("\n[工具执行完成: %s]\n", chunk.ToolName)
			}

		case reactagent.AgentChunkTypeError:
			// 显示错误
			fmt.Printf("\n错误: %s\n", chunk.Error)

		case reactagent.AgentChunkTypeFinish, reactagent.AgentChunkTypeMaxIter:
			// 显示结束信息
			if chunk.Error != "" {
				fmt.Printf("\n%s: %s\n", chunk.Type, chunk.Error)
			}
		}
		// 检查是否是最后一个块
		if chunk.IsLast {
			fmt.Println(chunk.AccumulatedThoughts)
			fmt.Println("\n[对话结束]")
			taskCompleted = true
		}
	}

	return taskCompleted
}

// closeBrowserInRegistry 尝试关闭注册表中的浏览器
func closeBrowserInRegistry(registry *toolcore.Registry) {
	// 获取所有注册的工具
	tools := registry.GetAll()
	for _, tool := range tools {
		if browserTool, ok := tool.(*browsertool.BrowserTool); ok {
			if err := browserTool.Close(); err != nil {
				fmt.Printf("关闭浏览器失败: %v\n", err)
			} else {
				fmt.Println("浏览器已关闭")
			}
			break
		}
	}
}

// registerTools 注册所需的工具
func registerTools(registry *toolcore.Registry, i18nMgr *i18n.Manager) {
	ctx := context.Background()

	// 注册浏览器工具
	browserConfig := browsertool.NewBrowserConfig()
	browserConfig.Headless = false // 有头，方便看
	browserConfig.WindowWidth = 1280
	browserConfig.WindowHeight = 800
	browserConfig.Timeout = 120
	browserConfig.URLTimeout = 45
	browserConfig.DataPath = "./data/browser_agent"
	browserConfig.DefaultLanguage = "zh-CN"
	browserTool := browsertool.NewBrowserToolWithConfig(i18nMgr, browserConfig)
	err := registry.Register(ctx, browserTool)
	if err != nil {
		slog.Error("注册浏览器工具失败", "error", err)
		os.Exit(1)
	}

	fetchTool := fetchtool.NewFetchTool(i18nMgr)
	err = registry.Register(ctx, fetchTool)
	if err != nil {
		slog.Error("注册 Fetch 工具失败", "error", err)
		os.Exit(1)
	}
}
