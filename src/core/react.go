package core

import (
	"context"
	"log/slog"
	"math"
	"os"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// NewReAct 创建一个默认的 ReAct，默认无限循环，
func NewReAct(chatModel model.ToolCallingChatModel, opts ...ReActOption) (*react.Agent, error) {
	config := NewReActConfig()
	return NewReActWithConfig(chatModel, config, opts...)
}

// NewReActWithConfig 从配置文件创建一个 ReAct，使用工具的最大次数为 MaxLoop - 1
func NewReActWithConfig(chatModel model.ToolCallingChatModel, config *ReActConfig, opts ...ReActOption) (*react.Agent, error) {
	logger := slog.Default().WithGroup("ReAct")

	if config == nil {
		config = NewReActConfig()
		logger.Warn("config is nil, using default config")
	}

	for _, opt := range opts {
		opt(config)
	}

	// 防止 maxLoop 过大，导致 maxStep 溢出
	if config.MaxLoop <= 0 || config.MaxLoop > math.MaxInt/2 {
		config.MaxLoop = math.MaxInt / 2
	}

	agnet, err := react.NewAgent(context.Background(), &react.AgentConfig{
		ToolCallingModel: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: config.Tools,
		},
		MaxStep: config.MaxLoop * 2, // ReAct 有 2 个节点，12 步表示 6 次循环
		MessageModifier: func(ctx context.Context, input []*schema.Message) []*schema.Message {
			res := make([]*schema.Message, 0, len(input)+1)

			// 如果系统提示不为空，并且没有设置记忆，则添加系统提示
			if config.SystemPrompt != "" && config.Memory == nil {
				res = append(res, schema.SystemMessage(config.SystemPrompt))
			}

			// 如果设置了记忆，则添加记忆
			if config.Memory != nil {
				res = append(res, config.Memory...)
			}

			res = append(res, input...)

			// 如果设置了消息查看器，则调用消息查看器
			if config.MessageViewer != nil {
				config.MessageViewer(res)
			}
			return res
		},
	})
	if err != nil {
		logger.Error("failed to create ReAct", "error", err)
		return nil, err
	}

	return agnet, nil
}

type ReActOption func(*ReActConfig)

type ReActConfig struct {
	SystemPrompt        string
	MaxLoop             int
	Tools               []tool.BaseTool
	ToolReturnDirectSet map[string]struct{}
	Memory              []*schema.Message
	MessageViewer       func(input []*schema.Message)
}

// NewReActConfig 创建一个 ReActConfig
// 默认最大循环次数为 6，工具列表为空
// 注意：传递给 ReAct 时，如果没有设置 Tools 会报错
func NewReActConfig() *ReActConfig {
	return &ReActConfig{
		MaxLoop: 6,
		Tools:   []tool.BaseTool{},
	}
}

// WithSystemPrompt 设置系统提示
func WithSystemPrompt(systemPrompt string) ReActOption {
	return func(config *ReActConfig) {
		config.SystemPrompt = systemPrompt
	}
}

// WithMaxLoop 设置最大循环次数
func WithMaxLoop(maxLoop int) ReActOption {
	return func(config *ReActConfig) {
		config.MaxLoop = maxLoop
	}
}

// WithTools 设置工具列表
func WithTools(tools []tool.BaseTool) ReActOption {
	return func(config *ReActConfig) {
		config.Tools = tools
	}
}

// WithTool 设置工具
func WithTool(tool tool.BaseTool) ReActOption {
	return func(config *ReActConfig) {
		config.Tools = append(config.Tools, tool)
	}
}

func WithReturnDirectlyTool(tool tool.BaseTool) ReActOption {
	toolInfo, err := tool.Info(context.Background())
	if err != nil {
		slog.Error("get tool info failed", "error", err)
		os.Exit(1)
	}

	return func(config *ReActConfig) {
		config.ToolReturnDirectSet[toolInfo.Name] = struct{}{}
	}
}

// WithMemory 设置记忆
func WithMemory(memory []*schema.Message) ReActOption {
	return func(config *ReActConfig) {
		config.Memory = memory
	}
}

// WithMessageViewer 设置消息查看器
func WithMessageViewer(messageViewer func(input []*schema.Message)) ReActOption {
	return func(config *ReActConfig) {
		config.MessageViewer = messageViewer
	}
}
