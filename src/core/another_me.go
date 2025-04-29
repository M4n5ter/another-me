package core

import (
	"context"
	"log/slog"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/m4n5ter/another-me/src/locale"
)

// AnotherMe 另一个我
type AnotherMe struct {
	react   *react.Agent
	history *Conversation

	logger *slog.Logger
}

// NewAnotherMe 创建另一个我，如果传入了 memory，则使用传入的 memory 作为记忆。
func NewAnotherMe(model model.ToolCallingChatModel, memory []*schema.Message, opts ...ReActOption) (*AnotherMe, error) {
	am := &AnotherMe{
		history: NewConversation(),
		logger:  slog.Default().WithGroup("another_me"),
	}

	opts = append(opts, WithSystemPrompt(locale.AnotherMeSystemPrompt()), WithMemory(memory), WithMessageViewer(func(input []*schema.Message) {
		// 无法得到最终的输出，因为最终的输出不会再走 MessageViewer，而是直接返回
		// 最终的输出在 Stream 和 Generate 中处理
		am.history.Set(input)
	}))
	react, err := NewReAct(model, opts...)
	if err != nil {
		return nil, err
	}

	am.react = react

	return am, nil
}

func (a *AnotherMe) GetHistory() *Conversation {
	return a.history
}

// Stream 流式生成
func (a *AnotherMe) Stream(ctx context.Context, messages []*schema.Message, opts ...agent.AgentOption) (*schema.StreamReader[*schema.Message], error) {
	stream, err := a.react.Stream(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	streamCloned := stream.Copy(1)[0]
	go func() {
		defer streamCloned.Close()
		msg, err := schema.ConcatMessageStream(streamCloned)
		if err != nil {
			a.logger.Error("Failed to read stream.")
		}
		a.history.Add(msg)
	}()

	return stream, nil
}

// Generate 非流式生成
func (a *AnotherMe) Generate(ctx context.Context, messages []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
	msg, err := a.react.Generate(ctx, messages, opts...)
	if err != nil {
		return nil, err
	}

	a.history.Add(msg)
	return msg, nil
}
