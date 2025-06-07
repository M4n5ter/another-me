package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/m4n5ter/another-me/internal/task_based_core/communication"
)

func main() {
	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	fmt.Println("🚀 通信模块演示程序")
	fmt.Println("==========================================")

	// 运行复杂工作流演示
	communication.ExampleComplexWorkflow()

	fmt.Println("==========================================")
	fmt.Println("✅ 演示完成")
}
