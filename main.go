package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"log/slog"
	"os"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/fetchtool"
	"github.com/m4n5ter/another-me/pkg/tools/gui"
)

//go:embed pkg/i18n/locales
var embeddedLocalesFS embed.FS

func main() {
	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
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

	// 输出注册的工具
	ctx := context.Background()
	tools := registry.GetAll()
	logger.Info("Registered tools", "count", len(tools))

	for _, tool := range tools {
		schema, err := tool.Schema(ctx)
		if err != nil {
			logger.Error("Failed to get tool schema", "error", err)
			continue
		}
		logger.Info("Tool registered", "name", schema.Name)
	}

	// 注意：这里可以添加更多的初始化代码，如HTTP服务器等
	logger.Info("Another-Me initialized successfully")
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
