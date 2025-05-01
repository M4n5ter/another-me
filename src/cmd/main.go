package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/schema"
	"github.com/m4n5ter/another-me/src/conf"
	"github.com/m4n5ter/another-me/src/core"
	"github.com/m4n5ter/another-me/src/core/db"
	"github.com/m4n5ter/another-me/src/locale"
	"github.com/m4n5ter/another-me/src/tool"
	"github.com/spf13/viper"
)

func main() {
	locale.SetLocaleFromStr(conf.GetLocale())

	surrealDBConfig := conf.GetSurrealDBConfig()
	db.SetDBs(surrealDBConfig)

	ctx := context.Background()

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey: viper.GetString("deepseek.api_key"),
		Model:  viper.GetString("deepseek.model"),
		// 设置返回格式为 JSON
		// ResponseFormatType: deepseek.ResponseFormatTypeJSONObject,
	})
	if err != nil {
		slog.Error("failed to create chat model", "error", err)
		os.Exit(1)
	}

	// but, err := browseruse.NewBrowserUseTool(ctx, &browseruse.Config{
	// 	Headless: true,
	// })
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// thinkingTool, err := sequentialthinking.NewTool()
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// am, err := core.ReAct(chatModel, core.WithTools([]tool.BaseTool{thinkingTool}), core.WithMaxLoop(10), core.WithSystemPrompt("你是一个 golang 开发专家."))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	am, err := core.NewAnotherMe(chatModel, nil, core.WithTool(tool.NewTaskEvaluatorTool()))
	if err != nil {
		slog.Error("failed to create another me", "error", err)
		os.Exit(1)
	}

	var msgReader *schema.StreamReader[*schema.Message]
	msgReader, err = am.Stream(ctx, []*schema.Message{
		schema.UserMessage("用Go实现一个Lisp解释器."),
	})
	if err != nil {
		slog.Error("failed to stream", "error", err)
		os.Exit(1)
	}

	slog.Info("start to stream")
	for {
		// msg type is *schema.Message
		msg, err := msgReader.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				// finish
				break
			}
			// error
			slog.Error("failed to recv", "error", err)
			os.Exit(1)
		}

		fmt.Print(msg.Content)
	}

	slog.Info("print history")
	fmt.Printf("%s", am.GetHistory())
}
