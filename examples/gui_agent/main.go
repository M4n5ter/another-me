package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"

	arc "github.com/cloudwego/eino-ext/components/model/ark"

	guiagent "github.com/m4n5ter/another-me/internal/gui_agent"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface/eino"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/tools/gui"
)

func main() {
	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
	// 设置 eino 模型

	maxTokens := Some(4096)
	temperature := Some(float32(0.0))
	topP := Some(float32(0.7))

	chatModel, err := arc.NewChatModel(context.Background(), &arc.ChatModelConfig{
		APIKey:      os.Getenv("ARK_API_KEY"),
		Model:       "doubao-1.5-ui-tars-250328",
		MaxTokens:   maxTokens.UnwrapAsPtr(),
		Temperature: temperature.UnwrapAsPtr(),
		TopP:        topP.UnwrapAsPtr(),
	})
	if err != nil {
		logger.Error("Failed to create eino model", "error", err)
		os.Exit(1)
	}

	// 创建 eino 适配器
	chatAdapter, err := eino.NewChatAdapter(context.Background(), chatModel, nil, "zh")
	if err != nil {
		logger.Error("Failed to create eino adapter", "error", err)
		os.Exit(1)
	}

	guiAgent, err := guiagent.NewGUIAgent(context.Background(), chatAdapter)
	if err != nil {
		logger.Error("Failed to create gui agent", "error", err)
		os.Exit(1)
	}

	guiTool := gui.NewGUITool(i18n.GlobalManager)
	screenshot, err := guiTool.Screenshot()
	if err != nil {
		logger.Error("Failed to screenshot", "error", err)
		os.Exit(1)
	}

	var screenshotResultMap map[string]any
	err = json.Unmarshal([]byte(screenshot), &screenshotResultMap)
	if err != nil {
		logger.Error("Failed to unmarshal screenshot", "error", err)
		os.Exit(1)
	}

	base64PNGURL := screenshotResultMap["result"].(string)

	result, err := guiAgent.Execute(context.Background(), "点击左上角的图标", base64PNGURL)
	if err != nil {
		logger.Error("Failed to execute gui agent", "error", err)
		os.Exit(1)
	}
	logger.Info("Result", "result", result)
}
