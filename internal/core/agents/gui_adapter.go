package agents

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/internal/core"
	"github.com/m4n5ter/another-me/internal/core/types"
	guiagent "github.com/m4n5ter/another-me/internal/gui_agent"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/m4n5ter/another-me/pkg/tools/gui"
)

// GUIAgentAdapter 将现有的GUIAgent适配到标准Agent接口
type GUIAgentAdapter struct {
	guiAgent *guiagent.GUIAgent
	guiTool  *gui.Tool
	logger   *slog.Logger
}

// NewGUIAgentAdapter 创建一个新的GUIAgentAdapter实例
func NewGUIAgentAdapter(ctx context.Context, llmAdapter llminterface.ChatAdapter) (*GUIAgentAdapter, error) {
	// 创建底层的GUIAgent
	guiAgent, err := guiagent.NewGUIAgent(ctx, llmAdapter)
	if err != nil {
		return nil, fmt.Errorf("创建GUIAgent失败: %w", err)
	}

	// 创建GUI工具用于截图等操作
	guiTool := gui.NewGUITool(nil) // 传入nil，使用默认的i18n管理器

	logger := slog.Default().WithGroup("gui_agent_adapter")

	return &GUIAgentAdapter{
		guiAgent: guiAgent,
		guiTool:  guiTool,
		logger:   logger,
	}, nil
}

var _ core.Agent = (*GUIAgentAdapter)(nil)

// Execute 实现Agent接口的Execute方法
func (g *GUIAgentAdapter) Execute(ctx context.Context, task types.Task, initialContext map[string]any) (types.ExecutionResult, error) {
	startTime := time.Now()
	g.logger.Info("开始执行GUI任务", "task_id", task.ID, "description", task.Description)

	// 从任务参数中提取指令和截图URL
	instruction, ok := task.Parameters["instruction"].(string)
	if !ok || instruction == "" {
		return types.ExecutionResult{
			TaskID:    task.ID,
			Status:    types.ExecutionStatusFailure,
			Error:     "任务参数中缺少instruction字段",
			StartTime: startTime,
			EndTime:   time.Now(),
		}, fmt.Errorf("任务参数中缺少instruction字段")
	}

	// 获取截图URL，如果没有提供则自动截图
	var imageURL string
	if url, exists := task.Parameters["image_url"].(string); exists && url != "" {
		imageURL = url
	} else {
		// 自动截图
		screenshot, err := g.guiTool.Screenshot()
		if err != nil {
			return types.ExecutionResult{
				TaskID:    task.ID,
				Status:    types.ExecutionStatusFailure,
				Error:     fmt.Sprintf("自动截图失败: %v", err),
				StartTime: startTime,
				EndTime:   time.Now(),
			}, fmt.Errorf("自动截图失败: %w", err)
		}

		// 解析截图结果获取base64 URL
		var screenshotResult map[string]any
		if err := json.Unmarshal([]byte(screenshot), &screenshotResult); err != nil {
			return types.ExecutionResult{
				TaskID:    task.ID,
				Status:    types.ExecutionStatusFailure,
				Error:     fmt.Sprintf("解析截图结果失败: %v", err),
				StartTime: startTime,
				EndTime:   time.Now(),
			}, fmt.Errorf("解析截图结果失败: %w", err)
		}

		imageURL = screenshotResult["result"].(string)
		g.logger.Debug("自动截图完成", "image_size", fmt.Sprintf("%.0fx%.0f",
			screenshotResult["width"], screenshotResult["height"]))
	}

	// 调用底层GUIAgent执行任务
	result, err := g.guiAgent.Execute(ctx, instruction, imageURL)
	endTime := time.Now()

	if err != nil {
		g.logger.Error("GUI任务执行失败", "task_id", task.ID, "error", err)
		return types.ExecutionResult{
			TaskID:    task.ID,
			Status:    types.ExecutionStatusFailure,
			Error:     err.Error(),
			StartTime: startTime,
			EndTime:   endTime,
			Metadata: map[string]any{
				"instruction": instruction,
				"duration":    endTime.Sub(startTime).String(),
			},
		}, err
	}

	// 转换执行结果
	observations := []string{}
	if result.Thought != "" {
		observations = append(observations, fmt.Sprintf("思考过程: %s", result.Thought))
	}
	if result.Action != "" {
		observations = append(observations, fmt.Sprintf("执行动作: %s", result.Action))
	}

	g.logger.Info("GUI任务执行成功", "task_id", task.ID, "action", result.Action)

	return types.ExecutionResult{
		TaskID:       task.ID,
		Status:       types.ExecutionStatusSuccess,
		Output:       result,
		Observations: observations,
		StartTime:    startTime,
		EndTime:      endTime,
		Metadata: map[string]any{
			"instruction":      instruction,
			"action":           result.Action,
			"thought":          result.Thought,
			"execution_output": result.ExecutionOutput,
			"duration":         endTime.Sub(startTime).String(),
		},
	}, nil
}

// Name 返回Agent的名称
func (g *GUIAgentAdapter) Name() string {
	return "GUIAgent"
}

// Type 返回Agent的类型
func (g *GUIAgentAdapter) Type() types.AgentType {
	return types.AgentTypeGUI
}

// IsAvailable 检查Agent是否可用
func (g *GUIAgentAdapter) IsAvailable(ctx context.Context) bool {
	// 检查是否能够正常截图，作为可用性检查
	_, err := g.guiTool.Screenshot()
	return err == nil
}

// GetCapabilities 获取Agent的能力描述
func (g *GUIAgentAdapter) GetCapabilities() []string {
	return []string{
		"GUI操作自动化",
		"鼠标点击、拖拽、移动",
		"键盘输入和快捷键",
		"屏幕截图和坐标转换",
		"滚动和窗口操作",
		"基于视觉的界面交互",
	}
}
