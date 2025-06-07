package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"google.golang.org/genai"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/m4n5ter/another-me/pkg/llminterface/google"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
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
	chatAdapter, err := google.NewGeminiAdapter(ctx, client, nil, &google.GeminiAdapterConfig{
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

	output, err := chatAdapter.ProduceJSON(ctx, llminterface.ChatInput{
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
