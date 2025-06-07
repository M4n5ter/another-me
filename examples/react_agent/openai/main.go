package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface/openai"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/fetchtool"
)

const reactSystemPrompt = `我是资深的加密货币投资者，擅长从大量角度分析加密货币的走势，擅长利用工具获取信息和攥写金融报告。`

func main() {
	i18n.GlobalManager.SetDefaultLanguage("zh")

	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 创建工具注册表
	registry := toolcore.NewRegistry()

	// 注册工具
	registerTools(registry, i18n.GlobalManager)

	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	model := os.Getenv("OPENAI_MODEL")

	// 创建 openai 适配器
	chatAdapter := openai.NewOpenAIChatAdapter(apiKey, Some(baseURL), &openai.OpenAIAdapterConfig{
		Model:    model,
		Registry: registry,
	})

	// --- ReAct Agent 测试 ---
	reactAgent, err := reactagent.NewToolCallingAgentBuilder().
		WithLLMAdapter(chatAdapter).
		WithTaskEvaluator(chatAdapter).
		WithToolRegistry(registry).
		WithLogger(logger.WithGroup("react_agent_main")).
		WithMaxIterations(50).
		WithSystemPrompt(reactSystemPrompt).
		Build()
	if err != nil {
		logger.Error("Failed to create ReAct agent", "error", err)
		return
	}

	userInput := "利用可用工具，从尽可能多的角度分析一下接下来短期、中期、长期Solana的走势，并给出一份详尽最终报告，需要给出精确的价格点位和时间点来支撑你的观点。"
	conversationID := "test-react-openai-001"

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(300 * time.Second)
		cancel()
	}()

	// 获取流式输出通道
	outputChan, err := reactAgent.Run(ctx, userInput, conversationID)
	if err != nil {
		// 处理初始错误
		logger.Error("Failed to run ReAct agent", "error", err)
		return
	}

	// 从通道中读取并处理流式数据
	for chunk := range outputChan {
		switch chunk.Type {
		case reactagent.AgentChunkTypeText:
			// 立即显示增量文本
			fmt.Print(chunk.TextDelta)

		case reactagent.AgentChunkTypeReasoning:
			// 打印推理内容
			fmt.Print(chunk.ReasoningContent)

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
		}
	}
}

// registerTools 注册所有可用的工具
func registerTools(registry *toolcore.Registry, i18nMgr *i18n.Manager) {
	ctx := context.Background()
	ctx = i18n.ContextWithLanguage(ctx, i18nMgr.GetDefaultLanguage())
	// 注册 Fetch 工具
	fetchTool := fetchtool.NewFetchTool(i18nMgr)
	err := registry.Register(ctx, fetchTool)
	if err != nil {
		slog.Error("Failed to register fetch tool", "error", err)
		os.Exit(1)
	}

	// 注册 MCP 工具
	tools, err := toolcore.STDIOMCPTools("bunx", nil, "-y", "@mcpfun/mcp-server-ccxt")
	if err != nil {
		slog.Error("Failed to register mcp tool", "error", err)
		return
	}
	for _, tool := range tools {
		err := registry.Register(ctx, tool)
		if err != nil {
			slog.Error("Failed to register mcp tool", "error", err)
		}
	}
}
