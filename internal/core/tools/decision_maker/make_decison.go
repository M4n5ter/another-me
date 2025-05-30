package decisionmaker

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/internal/core/types"
	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

type Decision struct {
	ShouldExecuteTask       bool                     `json:"should_execute_task" jsonschema:"title=should_execute_task,description=Whether to execute the task,example=true,example=false,default=false"`
	SelectedAgentType       string                   `json:"selected_agent_type"`
	TaskType                string                   `json:"task_type"`
	TaskDescription         string                   `json:"task_description"`
	TaskPriority            int                      `json:"task_priority"`
	TaskParameters          map[string]any           `json:"task_parameters"`
	ShouldSetupMonitoring   bool                     `json:"should_setup_monitoring"`
	MonitoringConditions    []types.MonitorCondition `json:"monitoring_conditions"`
	ShouldEnterWaitMode     bool                     `json:"should_enter_wait_mode"`
	ReasoningSteps          []string                 `json:"reasoning_steps"`
	Confidence              float64                  `json:"confidence"`
	ExpectedDurationMinutes int                      `json:"expected_duration_minutes"`
}

type MakeDecisionTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

func NewMakeDecisionTool(i18nMgr *i18n.Manager) *MakeDecisionTool {
	return &MakeDecisionTool{
		logger:  slog.Default().WithGroup("decision_maker_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*MakeDecisionTool)(nil)

func (t *MakeDecisionTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.decision_maker.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.decision_maker.name", nil)
	}

	// 定义输入参数
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "should_execute_task", toolcore.ParamTypeBoolean, true, nil, "tool.decision_maker.arg.should_execute_task", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "selected_agent_type", toolcore.ParamTypeString, true, nil, "tool.decision_maker.arg.selected_agent_type", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "task_type", toolcore.ParamTypeString, true, nil, "tool.decision_maker.arg.task_type", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "task_description", toolcore.ParamTypeString, true, nil, "tool.decision_maker.arg.task_description", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "task_priority", toolcore.ParamTypeInteger, true, nil, "tool.decision_maker.arg.task_priority", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "task_parameters", toolcore.ParamTypeObject, true, nil, "tool.decision_maker.arg.task_parameters", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "should_setup_monitoring", toolcore.ParamTypeBoolean, true, nil, "tool.decision_maker.arg.should_setup_monitoring", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "monitoring_conditions", toolcore.ParamTypeArray, true, nil, "tool.decision_maker.arg.monitoring_conditions", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "should_enter_wait_mode", toolcore.ParamTypeBoolean, true, nil, "tool.decision_maker.arg.should_enter_wait_mode", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "reasoning_steps", toolcore.ParamTypeArray, true, nil, "tool.decision_maker.arg.reasoning_steps", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "confidence", toolcore.ParamTypeNumber, true, nil, "tool.decision_maker.arg.confidence", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "expected_duration_minutes", toolcore.ParamTypeInteger, true, nil, "tool.decision_maker.arg.expected_duration_minutes", nil),
	}

	// 返回工具的完整模式
	return toolcore.ToolSchema{
		Name:             "decision_maker",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: []toolcore.ParameterDefinition{},
	}, nil
}

func (t *MakeDecisionTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var decision Decision
	if err := json.Unmarshal([]byte(inputJSON), &decision); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	return inputJSON, nil
}
