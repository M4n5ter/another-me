package main

import (
	"context"
	"log/slog"
	"os"

	arc "github.com/cloudwego/eino-ext/components/model/ark"
	json "github.com/json-iterator/go"

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

	// 检查环境变量
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		logger.Error("环境变量ARK_API_KEY未设置，请设置后再运行")
		os.Exit(1)
	}

	chatModel, err := arc.NewChatModel(context.Background(), &arc.ChatModelConfig{
		APIKey:      apiKey,
		Model:       "doubao-1.5-ui-tars-250328",
		MaxTokens:   maxTokens.UnwrapAsPtr(),
		Temperature: temperature.UnwrapAsPtr(),
		TopP:        topP.UnwrapAsPtr(),
	})
	if err != nil {
		logger.Error("创建eino模型失败", "error", err)
		os.Exit(1)
	}

	// 创建 eino 适配器
	chatAdapter, err := eino.NewChatAdapter(context.Background(), chatModel, nil)
	if err != nil {
		logger.Error("创建eino适配器失败", "error", err)
		os.Exit(1)
	}

	// 创建GUI Agent
	guiAgent, err := guiagent.NewGUIAgent(context.Background(), chatAdapter)
	if err != nil {
		logger.Error("创建GUI Agent失败", "error", err)
		os.Exit(1)
	}

	// 获取屏幕截图
	guiTool := gui.NewGUITool(i18n.GlobalManager)
	screenshot, err := guiTool.Screenshot()
	if err != nil {
		logger.Error("截图失败", "error", err)
		os.Exit(1)
	}

	var screenshotResultMap map[string]any
	err = json.Unmarshal([]byte(screenshot), &screenshotResultMap)
	if err != nil {
		logger.Error("解析截图失败", "error", err)
		os.Exit(1)
	}

	base64PNGURL := screenshotResultMap["result"].(string)
	imgWidth := screenshotResultMap["width"].(float64)
	imgHeight := screenshotResultMap["height"].(float64)

	logger.Info("已获取屏幕截图", "width", imgWidth, "height", imgHeight)

	// 用户指令示例
	// instruction := "点击左上角的图标"
	instruction := "输入内容：echo 'hello world'"
	logger.Info("执行指令", "instruction", instruction)

	// 执行GUI操作
	result, err := guiAgent.Execute(context.Background(), instruction, base64PNGURL)
	if err != nil {
		logger.Error("执行GUI Agent失败", "error", err)
		os.Exit(1)
	}

	if result.Thought != "" {
		logger.Info("思考过程", "thought", result.Thought)
	}

	if result.Action != "" {
		logger.Info("执行动作", "action", result.Action)
	}

	if result.ExecutionOutput != "" {
		logger.Info("执行结果", "execution_result", result.ExecutionOutput)
	}

	logger.Info("任务执行完成")
}
