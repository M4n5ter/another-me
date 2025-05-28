package agents

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/m4n5ter/another-me/internal/core"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// ReActAgentAdapter 将现有的ReActAgent适配到标准Agent接口
type ReActAgentAdapter struct {
	reactAgent   *reactagent.ToolCallingAgent
	toolRegistry *toolcore.Registry
	logger       *slog.Logger
}

// NewReActAgentAdapter 创建一个新的ReActAgentAdapter实例
func NewReActAgentAdapter(
	ctx context.Context,
	llmAdapter llminterface.ChatAdapter,
	toolRegistry *toolcore.Registry,
	systemPrompt string,
	maxIterations int,
) (*ReActAgentAdapter, error) {
	if toolRegistry == nil {
		return nil, fmt.Errorf("toolRegistry不能为空")
	}

	logger := slog.Default().WithGroup("react_agent_adapter")

	// 使用构建器模式创建ReActAgent
	reactAgent, err := reactagent.NewToolCallingAgentBuilder().
		WithLLMAdapter(llmAdapter).
		WithTaskEvaluator(llmAdapter).
		WithToolRegistry(toolRegistry).
		WithLogger(logger).
		WithMaxIterations(maxIterations).
		WithSystemPrompt(systemPrompt).
		Build()

	if err != nil {
		return nil, fmt.Errorf("创建ReActAgent失败: %w", err)
	}

	return &ReActAgentAdapter{
		reactAgent:   reactAgent,
		toolRegistry: toolRegistry,
		logger:       logger,
	}, nil
}

// Execute 实现Agent接口的Execute方法
func (r *ReActAgentAdapter) Execute(ctx context.Context, task core.Task, initialContext map[string]any) (core.ExecutionResult, error) {
	startTime := time.Now()
	r.logger.Info("开始执行ReAct任务", "task_id", task.ID, "description", task.Description)

	// 从任务参数中提取用户输入
	userInput, ok := task.Parameters["user_input"].(string)
	if !ok || userInput == "" {
		// 如果没有明确的user_input，使用任务描述作为输入
		userInput = task.Description
	}

	// 生成会话ID
	conversationID := fmt.Sprintf("task_%s_%d", task.ID, startTime.Unix())

	// 执行ReAct Agent
	outputChan, err := r.reactAgent.Run(ctx, userInput, conversationID)
	if err != nil {
		return core.ExecutionResult{
			TaskID:    task.ID,
			Status:    core.ExecutionStatusFailure,
			Error:     err.Error(),
			StartTime: startTime,
			EndTime:   time.Now(),
		}, fmt.Errorf("启动ReAct Agent失败: %w", err)
	}

	// 收集流式输出
	var finalOutput strings.Builder
	var observations []string
	var hasError bool
	var errorMsg string

	for chunk := range outputChan {
		switch chunk.Type {
		case reactagent.AgentChunkTypeText:
			finalOutput.WriteString(chunk.TextDelta)
			if chunk.TextDelta != "" {
				observations = append(observations, fmt.Sprintf("文本输出: %s", chunk.TextDelta))
			}

		case reactagent.AgentChunkTypeReasoning:
			if chunk.ReasoningContent != "" {
				observations = append(observations, fmt.Sprintf("推理过程: %s", chunk.ReasoningContent))
			}

		case reactagent.AgentChunkTypeThought:
			if chunk.CurrentIterThoughtContent != "" {
				observations = append(observations, fmt.Sprintf("思考过程: %s", chunk.CurrentIterThoughtContent))
			}

		case reactagent.AgentChunkTypeToolStart:
			if chunk.ToolName != "" {
				observations = append(observations, fmt.Sprintf("开始调用工具: %s", chunk.ToolName))
			}

		case reactagent.AgentChunkTypeToolEnd:
			if chunk.ToolName != "" {
				observations = append(observations, fmt.Sprintf("工具调用完成: %s", chunk.ToolName))
				if chunk.ToolResult != "" {
					observations = append(observations, fmt.Sprintf("工具结果: %s", chunk.ToolResult))
				}
			}

		case reactagent.AgentChunkTypeError:
			hasError = true
			errorMsg = chunk.Error
			r.logger.Error("ReAct执行出错", "task_id", task.ID, "error", chunk.Error)

		case reactagent.AgentChunkTypeFinish:
			if chunk.FinalResponse != "" {
				finalOutput.WriteString(chunk.FinalResponse)
			}
			r.logger.Info("ReAct任务完成", "task_id", task.ID)

		case reactagent.AgentChunkTypeMaxIter:
			r.logger.Warn("ReAct达到最大迭代次数", "task_id", task.ID)
			if chunk.Error != "" {
				observations = append(observations, fmt.Sprintf("达到最大迭代次数: %s", chunk.Error))
			}
		}

		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			return core.ExecutionResult{
				TaskID:       task.ID,
				Status:       core.ExecutionStatusCancelled,
				Error:        "任务被取消",
				Observations: observations,
				StartTime:    startTime,
				EndTime:      time.Now(),
			}, ctx.Err()
		default:
		}
	}

	endTime := time.Now()

	// 确定执行状态
	status := core.ExecutionStatusSuccess
	if hasError {
		status = core.ExecutionStatusFailure
	}

	result := core.ExecutionResult{
		TaskID:       task.ID,
		Status:       status,
		Output:       finalOutput.String(),
		Observations: observations,
		Error:        errorMsg,
		StartTime:    startTime,
		EndTime:      endTime,
		Metadata: map[string]any{
			"user_input":      userInput,
			"conversation_id": conversationID,
			"duration":        endTime.Sub(startTime).String(),
			"tool_count":      len(r.toolRegistry.GetAll()),
		},
	}

	if hasError {
		r.logger.Error("ReAct任务执行失败", "task_id", task.ID, "error", errorMsg)
		return result, fmt.Errorf("ReAct执行失败: %s", errorMsg)
	}

	r.logger.Info("ReAct任务执行成功", "task_id", task.ID, "output_length", len(finalOutput.String()))
	return result, nil
}

// Name 返回Agent的名称
func (r *ReActAgentAdapter) Name() string {
	return "ReActAgent"
}

// Type 返回Agent的类型
func (r *ReActAgentAdapter) Type() core.AgentType {
	return core.AgentTypeReAct
}

// IsAvailable 检查Agent是否可用
func (r *ReActAgentAdapter) IsAvailable(ctx context.Context) bool {
	// 检查工具注册表是否有可用工具
	tools := r.toolRegistry.GetAll()
	return len(tools) > 0
}

// GetCapabilities 获取Agent的能力描述
func (r *ReActAgentAdapter) GetCapabilities() []string {
	capabilities := []string{
		"基于ReAct范式的推理和行动",
		"多轮对话和任务规划",
		"工具调用和结果处理",
		"复杂任务分解和执行",
	}

	// 添加可用工具的能力描述
	tools := r.toolRegistry.GetAll()
	if len(tools) > 0 {
		capabilities = append(capabilities, fmt.Sprintf("支持%d个工具", len(tools)))
		for _, tool := range tools {
			schema, err := tool.Schema(context.Background())
			if err == nil {
				capabilities = append(capabilities, fmt.Sprintf("工具: %s", schema.Name))
			}
		}
	}

	return capabilities
} 