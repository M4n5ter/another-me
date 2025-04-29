package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino/schema"
	"github.com/m4n5ter/another-me/src/core"
	"github.com/m4n5ter/another-me/src/locale"
	"github.com/m4n5ter/another-me/src/tool"
)

func main() {
	locale.SetLocale(locale.LocaleZH)

	ctx := context.Background()

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey: "sk-xxx",
		Model:  "deepseek-chat",
		// 设置返回格式为 JSON
		// ResponseFormatType: deepseek.ResponseFormatTypeJSONObject,
	})
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	var msgReader *schema.StreamReader[*schema.Message]
	msgReader, err = am.Stream(ctx, []*schema.Message{
		schema.UserMessage("用Go实现一个Lisp解释器."),
	})
	if err != nil {
		log.Fatal(err)
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
			return
		}

		fmt.Print(msg.Content)
	}

	slog.Info("print history")
	fmt.Printf("%s", am.GetHistory())
}
