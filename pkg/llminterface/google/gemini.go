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
	model  string
	tools  Option[[]*genai.Tool]
}

// NewGeminiAdapter 创建 google genai 的适配器
func NewGeminiAdapter(ctx context.Context, client *genai.Client, model string, registry *toolcore.Registry) (*GeminiAdapter, error) {
	adapter := &GeminiAdapter{
		client: client,
		model:  model,
		tools:  None[[]*genai.Tool](),
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

		for response, err := range g.client.Models.GenerateContentStream(ctx, g.model, genaiMsgs, &genai.GenerateContentConfig{
			SystemInstruction: ExtractSystemPromptAsGenaiContent(input),
			Tools:             *g.tools.UnwrapAsPtr(),
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

	if g.tools.IsNone() || len(*g.tools.UnwrapAsPtr()) == 0 {
		g.tools = Some(make([]*genai.Tool, 0, len(tools)))
	}

	for _, tool := range tools {
		toolSchema, err := tool.Schema(ctx)
		if err != nil {
			return fmt.Errorf("failed to get tool schema: %w", err)
		}
		googleTool := ToolCoreSchemaToGoogleFunctionDeclaration(&toolSchema)
		g.tools = Some(append(*g.tools.UnwrapAsPtr(), &genai.Tool{
			FunctionDeclarations: []*genai.FunctionDeclaration{googleTool},
		}))
	}
	return nil
}
