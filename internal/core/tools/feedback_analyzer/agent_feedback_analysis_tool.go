package feedbackanalyzer

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/internal/core/types"
	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// AgentFeedbackAnalysisResult mirrors the structure in feedback_analyzer.go
// AgentFeedbackAnalysisResult Agent返回的反馈分析结果
type AgentFeedbackAnalysisResult struct {
	KeyFindings              []string             `json:"key_findings"`
	ActionableInsights       []string             `json:"actionable_insights"`
	RequiresUserInput        bool                 `json:"requires_user_input"`
	ConfidenceLevel          float64              `json:"confidence_level"`
	RecommendedActions       []string             `json:"recommended_actions"`
	RiskAssessment           types.RiskAssessment `json:"risk_assessment"`
	NextStepSuggestions      []string             `json:"next_step_suggestions"`
	PatternsDetected         []string             `json:"patterns_detected"`
	ImprovementOpportunities []string             `json:"improvement_opportunities"`
}

type AgentFeedbackAnalysisTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

func NewAgentFeedbackAnalysisTool(i18nMgr *i18n.Manager) *AgentFeedbackAnalysisTool {
	return &AgentFeedbackAnalysisTool{
		logger:  slog.Default().WithGroup("agent_feedback_analysis_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*AgentFeedbackAnalysisTool)(nil)

func (t *AgentFeedbackAnalysisTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.agent_feedback_analysis.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.agent_feedback_analysis.name", nil)
	}

	riskAssessmentDescKey := "tool.common.risk_assessment.description"
	riskAssessmentLevelDescKey := "tool.common.risk_assessment.level.description"
	riskAssessmentFactorsDescKey := "tool.common.risk_assessment.factors.description"
	riskAssessmentMitigationDescKey := "tool.common.risk_assessment.mitigation.description"
	riskAssessmentDescDescKey := "tool.common.risk_assessment.description.description"

	riskAssessmentSubParams := []any{
		common.CreateParamDef(ctx, t.i18nMgr, "level", toolcore.ParamTypeString, true, option.None[[]any](), riskAssessmentLevelDescKey, option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "factors", toolcore.ParamTypeArray, true, option.None[[]any](), riskAssessmentFactorsDescKey, option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "mitigation", toolcore.ParamTypeArray, true, option.None[[]any](), riskAssessmentMitigationDescKey, option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "description", toolcore.ParamTypeString, true, option.None[[]any](), riskAssessmentDescDescKey, option.None[toolcore.ParameterDefinition]()),
	}

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "key_findings", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_feedback_analysis.arg.key_findings", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "actionable_insights", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_feedback_analysis.arg.actionable_insights", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "requires_user_input", toolcore.ParamTypeBoolean, true, option.None[[]any](), "tool.agent_feedback_analysis.arg.requires_user_input", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "confidence_level", toolcore.ParamTypeNumber, true, option.None[[]any](), "tool.agent_feedback_analysis.arg.confidence_level", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "recommended_actions", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_feedback_analysis.arg.recommended_actions", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "risk_assessment", toolcore.ParamTypeObject, true, option.Some[[]any](riskAssessmentSubParams), riskAssessmentDescKey, option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "next_step_suggestions", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_feedback_analysis.arg.next_step_suggestions", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "patterns_detected", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_feedback_analysis.arg.patterns_detected", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "improvement_opportunities", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_feedback_analysis.arg.improvement_opportunities", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
	}

	return toolcore.ToolSchema{
		Name:             "agent_feedback_analyzer",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: []toolcore.ParameterDefinition{},
	}, nil
}

func (t *AgentFeedbackAnalysisTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var result AgentFeedbackAnalysisResult
	if err := json.Unmarshal([]byte(inputJSON), &result); err != nil {
		t.logger.Error("解析 AgentFeedbackAnalysisResult 参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}
	return inputJSON, nil
}
