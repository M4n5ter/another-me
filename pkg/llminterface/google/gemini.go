package google

import (
	"context"
	"fmt"

	"google.golang.org/genai"

	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// GeminiAdapter 是 google genai 的适配器
type GeminiAdapter struct {
	client *genai.Client
	config *GeminiAdapterConfig
}

// GeminiAdapterConfig 是 google genai 的适配器配置
type GeminiAdapterConfig struct {
	Model           string
	Tools           Option[[]*genai.Tool]
	Temperature     Option[float32]
	TopP            Option[float32]
	TopK            Option[float32]
	MaxOutputTokens Option[int32]
	StopSequences   Option[[]string]
	ThinkingConfig  Option[GeminiThinkingConfig]
}

// GeminiThinkingConfig 是 google genai 的思考配置
type GeminiThinkingConfig struct {
	// 是否包含思考
	IncludeThoughts bool
	// 思考预算，单位是 token
	ThinkingBudget Option[int32]
}

// ToGenaiThinkingConfig 将 GeminiThinkingConfig 转换为 genai.ThinkingConfig
func (c *GeminiThinkingConfig) ToGenaiThinkingConfig() *genai.ThinkingConfig {
	return &genai.ThinkingConfig{
		IncludeThoughts: c.IncludeThoughts,
		ThinkingBudget:  c.ThinkingBudget.UnwrapAsPtr(),
	}
}

// NewGeminiAdapter 创建 google genai 的适配器
func NewGeminiAdapter(ctx context.Context, client *genai.Client, registry *toolcore.Registry, config *GeminiAdapterConfig) (*GeminiAdapter, error) {
	adapter := &GeminiAdapter{
		client: client,
		config: config,
	}

	err := adapter.RegisterTools(ctx, registry)
	return adapter, err
}

var _ llminterface.ChatAdapter = (*GeminiAdapter)(nil)

// Chat implements llminterface.ChatAdapter.
func (g *GeminiAdapter) Chat(ctx context.Context, input llminterface.ChatInput) (<-chan llminterface.ChatOutputChunk, error) {
	genaiMsgs := ChatInputToGenaiMsgs(input)

	outputChan := make(chan llminterface.ChatOutputChunk, 10)

	go func() {
		defer close(outputChan)

		for response, err := range g.client.Models.GenerateContentStream(ctx, g.config.Model, genaiMsgs, &genai.GenerateContentConfig{
			SystemInstruction: ExtractSystemPromptAsGenaiContent(input),
			Tools:             *g.config.Tools.UnwrapAsPtr(),
			Temperature:       g.config.Temperature.UnwrapAsPtr(),
			TopP:              g.config.TopP.UnwrapAsPtr(),
			TopK:              g.config.TopK.UnwrapAsPtr(),
			MaxOutputTokens:   g.config.MaxOutputTokens.Unwrap(),
			StopSequences:     g.config.StopSequences.Unwrap(),
			ThinkingConfig:    g.config.ThinkingConfig.UnwrapAsPtr().ToGenaiThinkingConfig(),
		}) {
			if err != nil {
				outputChan <- llminterface.ChatOutputChunk{
					Error: err,
				}
				return
			}

			chunk := ProcessGoogleResponseToChatOutputChunk(response)
			outputChan <- chunk
		}
	}()

	return outputChan, nil
}

// GetFrameworkName implements llminterface.ChatAdapter.
func (g *GeminiAdapter) GetFrameworkName() string {
	return "google-genai"
}

// RegisterTools implements llminterface.ChatAdapter.
func (g *GeminiAdapter) RegisterTools(ctx context.Context, registry *toolcore.Registry) error {
	tools := registry.GetAll()
	if len(tools) == 0 {
		return nil
	}

	if g.config.Tools.IsNone() || len(*g.config.Tools.UnwrapAsPtr()) == 0 {
		g.config.Tools = Some(make([]*genai.Tool, 0, len(tools)))
	}

	functions := make([]*genai.FunctionDeclaration, 0, len(tools))
	for _, tool := range tools {
		toolSchema, err := tool.Schema(ctx)
		if err != nil {
			return fmt.Errorf("failed to get tool schema: %w", err)
		}
		functions = append(functions, ToolCoreSchemaToGoogleFunctionDeclaration(&toolSchema))
	}
	g.config.Tools = Some(append(*g.config.Tools.UnwrapAsPtr(), &genai.Tool{
		FunctionDeclarations: functions,
	}))

	return nil
}
