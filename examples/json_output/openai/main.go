package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/m4n5ter/another-me/pkg/llminterface/openai"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
)

func main() {
	// 设置日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	apiKey := os.Getenv("OPENAI_API_KEY")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	model := os.Getenv("OPENAI_MODEL")

	// 创建 openai 适配器
	chatAdapter := openai.NewOpenAIChatAdapter(apiKey, Some(baseURL), &openai.OpenAIAdapterConfig{
		Model: model,
	})

	output, err := chatAdapter.ProduceJSON(context.Background(), llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.UserInputMessageText("你好，请告诉我你的名字和年龄，并返回一个JSON对象。格式为：{name: string, age: number}"),
		},
	}, None[schema.Schema]()) // 不指定JSONSchema，则需要在提示词中明确返回的JSON格式
	if err != nil {
		slog.Error("ProduceJSON failed", "error", err)
		return
	}

	fmt.Println(output)
}
