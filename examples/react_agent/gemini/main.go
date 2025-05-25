package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"google.golang.org/genai"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface/google"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/fetchtool"
)

const reactSystemPrompt = `你是一个精通各种技术的 AI 助手。你的目标是通过逐步思考来回答用户的问题。`

func main() {
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

	// 创建 google genai 客户端
	apiKey := os.Getenv("GEMINI_API_KEY")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			BaseURL: "https://gateway.ai.cloudflare.com/v1/ef2319bf182b2b327281a937e203cf85/another-me/google-ai-studio",
		},
	})
	if err != nil {
		slog.Error("NewClient of gemini failed", "error", err)
		return
	}

	// 设置 google genai 适配器
	chatAdapter, err := google.NewGeminiAdapter(ctx, client, registry, &google.GeminiAdapterConfig{
		Model:       "gemini-2.5-flash-preview-05-20",
		Temperature: Some(float32(0.1)),
		ThinkingConfig: Some(google.GeminiThinkingConfig{
			IncludeThoughts: true,
			ThinkingBudget:  Some(int32(1000)),
		}),
	})
	if err != nil {
		logger.Error("Failed to create google genai adapter", "error", err)
		return
	}

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

	userInput := "通过网络抓取去调查如何实现方便的为每个用户提供一个隔离的虚拟安卓 GUI 环境，并给出一份最终报告。"
	conversationID := "test-react-hackernews-001"

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
	// 注册 Fetch 工具
	fetchTool := fetchtool.NewFetchTool(i18nMgr)
	err := registry.Register(ctx, fetchTool)
	if err != nil {
		slog.Error("Failed to register fetch tool", "error", err)
		os.Exit(1)
	}

	// 注册 MCP 工具
	// tools, err := toolcore.STDIOMCPTools("uvx", nil, "mcp-server-fetch")
	// if err != nil {
	// 	log.Fatalf("Failed to register mcp tool: %v", err)
	// }
	// for _, tool := range tools {
	// 	err := registry.Register(ctx, tool)
	// 	if err != nil {
	// 		log.Fatalf("Failed to register mcp tool: %v", err)
	// 	}
	// }
}
