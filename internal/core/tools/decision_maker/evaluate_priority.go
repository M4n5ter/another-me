package decisionmaker

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// TaskPriorityItem 任务优先级项
type TaskPriorityItem struct {
	TaskID   string `json:"task_id" jsonschema:"title=task_id,description=Task identifier"`
	Priority int    `json:"priority" jsonschema:"title=priority,description=Task priority,minimum=1,maximum=10"`
	Reason   string `json:"reason" jsonschema:"title=reason,description=Reason for this priority"`
}

// TaskPriorityEvaluation 任务优先级评估结果
type TaskPriorityEvaluation struct {
	PrioritizedTasks []TaskPriorityItem `json:"prioritized_tasks" jsonschema:"title=prioritized_tasks,description=Tasks sorted by priority"`
	Reasoning        string             `json:"reasoning" jsonschema:"title=reasoning,description=Overall reasoning for priority assignments"`
	Confidence       float64            `json:"confidence" jsonschema:"title=confidence,description=Confidence level,minimum=0,maximum=1"`
}

// EvaluateTaskPriorityTool 任务优先级评估工具
type EvaluateTaskPriorityTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

// NewEvaluateTaskPriorityTool 创建任务优先级评估工具实例
func NewEvaluateTaskPriorityTool(i18nMgr *i18n.Manager) *EvaluateTaskPriorityTool {
	return &EvaluateTaskPriorityTool{
		logger:  slog.Default().WithGroup("evaluate_priority_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*EvaluateTaskPriorityTool)(nil)

// Schema 返回工具的完整模式
func (t *EvaluateTaskPriorityTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.evaluate_priority.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.evaluate_priority.name", nil)
	}

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "prioritized_tasks", toolcore.ParamTypeArray, true, nil, "tool.evaluate_priority.arg.prioritized_tasks", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "reasoning", toolcore.ParamTypeString, true, nil, "tool.evaluate_priority.arg.reasoning", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "confidence", toolcore.ParamTypeNumber, true, nil, "tool.evaluate_priority.arg.confidence", nil),
	}

	return toolcore.ToolSchema{
		Name:             "evaluate_priority",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: []toolcore.ParameterDefinition{},
	}, nil
}

// Call 工具调用方法
func (t *EvaluateTaskPriorityTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var evaluation TaskPriorityEvaluation
	if err := json.Unmarshal([]byte(inputJSON), &evaluation); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	return inputJSON, nil
}
