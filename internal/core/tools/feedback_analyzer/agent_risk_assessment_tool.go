package feedbackanalyzer

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// AgentRiskAssessmentResult mirrors the structure in feedback_analyzer.go
// AgentRiskAssessmentResult LLM返回的风险评估结果
type AgentRiskAssessmentResult struct {
	Level                   string   `json:"level"`
	Factors                 []string `json:"factors"`
	Mitigation              []string `json:"mitigation"`
	Description             string   `json:"description"`
	PredictedImpactIfOccurs string   `json:"predicted_impact_if_occurs,omitempty"`
	ProbabilityAssessment   string   `json:"probability_assessment,omitempty"`
}

type AgentRiskAssessmentTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

func NewAgentRiskAssessmentTool(i18nMgr *i18n.Manager) *AgentRiskAssessmentTool {
	return &AgentRiskAssessmentTool{
		logger:  slog.Default().WithGroup("agent_risk_assessment_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*AgentRiskAssessmentTool)(nil)

func (t *AgentRiskAssessmentTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		// Placeholder i18n keys
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.agent_risk_assessment.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.agent_risk_assessment.name", nil)
	}

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "level", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_risk_assessment.arg.level", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "factors", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_risk_assessment.arg.factors", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "mitigation", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_risk_assessment.arg.mitigation", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "description", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_risk_assessment.arg.description", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "predicted_impact_if_occurs", toolcore.ParamTypeString, false, option.None[[]any](), "tool.agent_risk_assessment.arg.predicted_impact_if_occurs", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "probability_assessment", toolcore.ParamTypeString, false, option.None[[]any](), "tool.agent_risk_assessment.arg.probability_assessment", option.None[toolcore.ParameterDefinition]()),
	}

	return toolcore.ToolSchema{
		Name:             "agent_risk_assessor",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: []toolcore.ParameterDefinition{},
	}, nil
}

func (t *AgentRiskAssessmentTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var result AgentRiskAssessmentResult
	if err := json.Unmarshal([]byte(inputJSON), &result); err != nil {
		t.logger.Error("解析 AgentRiskAssessmentResult 参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}
	return inputJSON, nil
}
