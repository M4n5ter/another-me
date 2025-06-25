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

// MonitoringDefinition 监控定义结果
type MonitoringDefinition struct {
	MonitoringTasks []types.MonitoringTask `json:"monitoring_tasks" jsonschema:"title=monitoring_tasks,description=Defined monitoring tasks"`
	Reasoning       string                 `json:"reasoning" jsonschema:"title=reasoning,description=Reasoning for monitoring definitions"`
	Confidence      float64                `json:"confidence" jsonschema:"title=confidence,description=Confidence level,minimum=0,maximum=1"`
}

// DefineMonitoringTool 监控定义工具
type DefineMonitoringTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

// NewDefineMonitoringTool 创建监控定义工具实例
func NewDefineMonitoringTool(i18nMgr *i18n.Manager) *DefineMonitoringTool {
	return &DefineMonitoringTool{
		logger:  slog.Default().WithGroup("define_monitoring_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*DefineMonitoringTool)(nil)

// Schema 返回工具的完整模式
func (t *DefineMonitoringTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.define_monitoring.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.define_monitoring.name", nil)
	}

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "monitoring_tasks", toolcore.ParamTypeArray, true, nil, "tool.define_monitoring.arg.monitoring_tasks", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "reasoning", toolcore.ParamTypeString, true, nil, "tool.define_monitoring.arg.reasoning", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "confidence", toolcore.ParamTypeNumber, true, nil, "tool.define_monitoring.arg.confidence", nil),
	}

	return toolcore.ToolSchema{
		Name:             "define_monitoring",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: []toolcore.ParameterDefinition{},
	}, nil
}

// Call 工具调用方法
func (t *DefineMonitoringTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var definition MonitoringDefinition
	if err := json.Unmarshal([]byte(inputJSON), &definition); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	return inputJSON, nil
}
