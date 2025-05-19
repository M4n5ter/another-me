package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"log/slog"
	"os"

	"github.com/cloudwego/eino-ext/components/model/deepseek"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface/eino"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/fetchtool"
	"github.com/m4n5ter/another-me/pkg/tools/gui"
)

//go:embed pkg/i18n/locales
var embeddedLocalesFS embed.FS

const reactSystemPrompt = `You are a meticulous and precise AI assistant. Your goal is to answer the user's request by thinking step-by-step. `

func main() {
	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// 获取正确的 locales 目录 FS
	localesDirFS, err := fs.Sub(embeddedLocalesFS, "pkg/i18n/locales")
	if err != nil {
		logger.Error("Failed to create sub FS for locales", "error", err)
		os.Exit(1)
	}

	// 初始化国际化管理器
	i18nMgr, err := i18n.NewManager(localesDirFS, "en")
	if err != nil {
		logger.Error("Failed to initialize i18n manager", "error", err)
		os.Exit(1)
	}

	// 创建工具注册表
	registry := toolcore.NewRegistry()

	// 注册工具
	registerTools(registry, i18nMgr)

	// 设置 eino 模型
	chatModel, err := deepseek.NewChatModel(context.Background(), &deepseek.ChatModelConfig{
		APIKey:      os.Getenv("DEEPSEEK_API_KEY"),
		Model:       "deepseek-chat",
		MaxTokens:   4096,
		Temperature: 0.1,
	})
	if err != nil {
		logger.Error("Failed to create eino model", "error", err)
		os.Exit(1)
	}

	// 创建 eino 适配器
	chatAdapter, err := eino.NewChatAdapter(context.Background(), chatModel, registry, "zh")
	if err != nil {
		logger.Error("Failed to create eino adapter", "error", err)
		os.Exit(1)
	}

	// --- ReAct Agent 测试 ---
	reactAgent, err := reactagent.NewAgentBuilder().
		WithLLMAdapter(chatAdapter).
		WithToolRegistry(registry).
		WithLogger(logger.WithGroup("react_agent_main")).
		WithMaxIterations(7).
		WithSystemPrompt(reactSystemPrompt).
		Build()
	if err != nil {
		logger.Error("Failed to create ReAct agent", "error", err)
		os.Exit(1)
	}

	userInput := "获取 Hacker News (news.ycombinator.com) 首页的最新5条消息的标题和链接，并根据它们各自的链接中的内容，生成一个关于这些消息的总结。"
	conversationID := "test-react-hackernews-001"
	logger.Info("Running ReAct Agent with Hacker News query and new system prompt", "input", userInput)

	logger.Debug("--- CHECKPOINT: About to call reactAgent.Run ---")

	finalResponse, err := reactAgent.Run(context.Background(), userInput, conversationID)
	if err != nil {
		logger.Error("ReAct agent Run failed", "error", err)
		if finalResponse != "" {
			fmt.Println("\n--- Agent's Partial Response on Error ---")
			fmt.Println(finalResponse)
			fmt.Println("--- End of Partial Response ---")
		}
		os.Exit(1)
	}

	fmt.Println("\n--- Agent's Final Response ---")
	fmt.Println(finalResponse)
	fmt.Println("--- End of Agent's Final Response ---")
}

// registerTools 注册所有可用的工具
func registerTools(registry *toolcore.Registry, i18nMgr *i18n.Manager) {
	ctx := context.Background()
	// 注册 Fetch 工具
	fetchTool := fetchtool.NewFetchTool(i18nMgr)
	err := registry.Register(ctx, fetchTool)
	if err != nil {
		log.Fatalf("Failed to register fetch tool: %v", err)
	}

	// 注册 GUI 工具
	guiTools := gui.NewGUITools(i18nMgr)
	for _, tool := range guiTools {
		err := registry.Register(ctx, tool)
		if err != nil {
			log.Fatalf("Failed to register gui tool: %v", err)
		}
	}

	// 注意：这里可以注册更多工具
}
