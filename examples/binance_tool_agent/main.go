package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/cloudwego/eino-ext/components/model/deepseek"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface/eino"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	binancetool "github.com/m4n5ter/another-me/pkg/tools/cryptoexchange/binance"
)

const reactSystemPrompt = `You are a meticulous and precise AI assistant. Your goal is to answer the user's request by thinking step-by-step. `

func main() {
	// 创建上下文
	ctx := context.Background()

	binanceClient := binancetool.NewBinance("", "", Some("socks5://127.0.0.1:55535"))
	listTickerPricesTool := binancetool.NewListTickerPricesTool(i18n.GlobalManager, binanceClient)

	symbolPrices, err := listTickerPricesTool.Call(ctx, `{"symbols": ["BTCUSDT", "ETHUSDT", "SOLUSDT", "XRPUSDT", "DOGEUSDT", "ADAUSDT", "DOTUSDT", "LINKUSDT", "BCHUSDT", "LTCUSDT", "XLMUSDT", "XMRUSDT"]}`)
	if err != nil {
		slog.Error("failed to list ticker prices", "error", err)
		return
	}

	fmt.Println(symbolPrices)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	registry := toolcore.NewRegistry()

	err = registry.Register(ctx, listTickerPricesTool)
	if err != nil {
		logger.Error("Failed to register list ticker prices tool", "error", err)
		return
	}

	// 从环境变量获取API密钥
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		logger.Error("API Key is not set")
		return
	}

	chatModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		APIKey:      apiKey,
		Model:       "deepseek-chat",
		MaxTokens:   4096,
		Temperature: 0.7,
	})
	if err != nil {
		logger.Error("Failed to create deepseek model", "error", err)
		return
	}

	llmAdapter, err := eino.NewChatAdapter(ctx, chatModel, registry, "zh")
	if err != nil {
		logger.Error("Failed to create no-tool LLM adapter", "error", err)
		return
	}

	reactAgent, err := reactagent.NewToolCallingAgentBuilder().
		WithLLMAdapter(llmAdapter).
		WithTaskEvaluator(llmAdapter).
		WithToolRegistry(registry).
		WithLogger(logger.WithGroup("react_agent_main")).
		WithMaxIterations(7).
		WithSystemPrompt(reactSystemPrompt).
		Build()
	if err != nil {
		logger.Error("Failed to create ReAct agent", "error", err)
		return
	}

	outputChan, err := reactAgent.Run(ctx, "查看一下比特币的价格，如果比特币的价格大于十万美元，那么再看一下以太坊的价格，如果以太坊的价格大于2000美元，那么再看一下Solana的价格。如果比特币的价格低于十万美元，那么查看一下DOGE的价格", "test-binance-tool-001")
	if err != nil {
		logger.Error("Failed to run ReAct agent", "error", err)
		return
	}

	// 从通道中读取并处理流式数据
	for chunk := range outputChan {
		switch chunk.Type {
		case reactagent.AgentChunkTypeText:
			// 立即显示增量文本
			fmt.Print(chunk.TextDelta)

		case reactagent.AgentChunkTypeReasoning:
			// 打印推理内容
			fmt.Print(chunk.ReasoningContent)

		case reactagent.AgentChunkTypeToolStart:
			// 显示工具正在执行的指示
			fmt.Printf("\n[执行工具: %s %s]\n", chunk.ToolName, chunk.ToolArguments)

		case reactagent.AgentChunkTypeToolEnd:
			// 显示工具执行完成的指示
			if chunk.Error != "" {
				fmt.Printf("\n[工具执行失败: %s - %s]\n", chunk.ToolName, chunk.Error)
			} else {
				fmt.Printf("\n[工具执行完成: %s]\n", chunk.ToolName)
			}

		case reactagent.AgentChunkTypeError:
			// 显示错误
			fmt.Printf("\n错误: %s\n", chunk.Error)

		case reactagent.AgentChunkTypeFinish, reactagent.AgentChunkTypeMaxIter:
			// 显示结束信息
			if chunk.Error != "" {
				fmt.Printf("\n%s: %s\n", chunk.Type, chunk.Error)
			}
		}
		// 检查是否是最后一个块
		if chunk.IsLast {
			fmt.Println(chunk.AccumulatedThoughts)
			fmt.Println("\n[对话结束]")
		}
	}
}
