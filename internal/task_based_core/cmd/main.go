package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/m4n5ter/another-me/internal/task_based_core/system"
	"github.com/m4n5ter/another-me/pkg/i18n"
)

func main() {
	ctx := context.Background()
	i18n.GlobalManager.SetDefaultLanguage("zh")
	ctx = i18n.ContextWithLanguage(ctx, i18n.GlobalManager.GetDefaultLanguage())

	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// 创建并启动多智能体系统
	system, err := system.NewMultiAgentSystem(ctx, logger)
	if err != nil {
		logger.Error("创建多智能体系统失败", "error", err)
		return
	}

	// 启动系统
	if err := system.Start(ctx); err != nil {
		logger.Error("启动多智能体系统失败", "error", err)
		return
	}

	// 等待信号
	waitForShutdown(system, logger)
}

// waitForShutdown 等待关闭信号
func waitForShutdown(system *system.MultiAgentSystem, logger *slog.Logger) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigChan
	logger.Info("接收到关闭信号", "signal", sig.String())

	if err := system.Stop(); err != nil {
		logger.Error("系统关闭失败", "error", err)
	}
}
