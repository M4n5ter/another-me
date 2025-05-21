package tools

import (
	"context"
	"log/slog"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/m4n5ter/another-me/pkg/tools/admintool"
	"github.com/m4n5ter/another-me/pkg/tools/browsertool"
	"github.com/m4n5ter/another-me/pkg/tools/fetchtool"
	"github.com/m4n5ter/another-me/pkg/tools/gui"
)

// RegisterAllTools 将所有工具注册到提供的注册表中
func RegisterAllTools(ctx context.Context, registry *toolcore.Registry, i18nMgr *i18n.Manager) {
	// 注册所有工具组
	registerTools(ctx, registry, admintool.NewAdminTools(i18nMgr))
	registerTools(ctx, registry, fetchtool.NewFetchTools(i18nMgr))
	registerTools(ctx, registry, gui.NewGUITools(i18nMgr))
	registerTools(ctx, registry, browsertool.NewBrowserTools(i18nMgr))
}

// registerTools 将工具列表注册到注册表
func registerTools(ctx context.Context, registry *toolcore.Registry, tools []toolcore.Tool) {
	logger := slog.Default().WithGroup("tools_register")

	for _, tool := range tools {
		err := registry.Register(ctx, tool)
		if err != nil {
			logger.Error("注册工具失败", "tool", tool, "error", err)
		} else {
			logger.Info("成功注册工具", "tool", tool)
		}
	}
}
