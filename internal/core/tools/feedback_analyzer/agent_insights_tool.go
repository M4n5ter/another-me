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

// AgentInsightsResult mirrors the structure in feedback_analyzer.go
// AgentInsightsResult LLM返回的洞察结果
type AgentInsightsResult struct {
	KeyTakeaways              []string `json:"key_takeaways"`
	ActionableRecommendations []struct {
		Recommendation  string `json:"recommendation"`
		Rationale       string `json:"rationale"`
		Priority        string `json:"priority"`
		PotentialImpact string `json:"potential_impact"`
	} `json:"actionable_recommendations"`
	ImprovementOpportunities []struct {
		Opportunity      string `json:"opportunity"`
		PotentialBenefit string `json:"potential_benefit"`
		Difficulty       string `json:"difficulty"`
	} `json:"improvement_opportunities"`
	PredictedNextSteps []string `json:"predicted_next_steps"`
	ConfidenceLevel    float64  `json:"confidence_level"`
	Summary            string   `json:"summary"`
}

type AgentInsightsTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

func NewAgentInsightsTool(i18nMgr *i18n.Manager) *AgentInsightsTool {
	return &AgentInsightsTool{
		logger:  slog.Default().WithGroup("agent_insights_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*AgentInsightsTool)(nil)

func (t *AgentInsightsTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		// Placeholder i18n keys
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.agent_insights.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.agent_insights.name", nil)
	}

	actionableRecSubParamsDef := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "recommendation", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_insights.actionable_recommendations.recommendation", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "rationale", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_insights.actionable_recommendations.rationale", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "priority", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_insights.actionable_recommendations.priority", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "potential_impact", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_insights.actionable_recommendations.potential_impact", option.None[toolcore.ParameterDefinition]()),
	}

	improvementOppSubParamsDef := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "opportunity", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_insights.improvement_opportunities.opportunity", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "potential_benefit", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_insights.improvement_opportunities.potential_benefit", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "difficulty", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_insights.improvement_opportunities.difficulty", option.None[toolcore.ParameterDefinition]()),
	}

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "key_takeaways", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_insights.arg.key_takeaways", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "actionable_recommendations", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_insights.arg.actionable_recommendations", option.Some(toolcore.ParameterDefinition{
			Type:       toolcore.ParamTypeObject,
			Properties: option.Some(actionableRecSubParamsDef),
		})),
		common.CreateParamDef(ctx, t.i18nMgr, "improvement_opportunities", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_insights.arg.improvement_opportunities", option.Some(toolcore.ParameterDefinition{
			Type:       toolcore.ParamTypeObject,
			Properties: option.Some(improvementOppSubParamsDef),
		})),
		common.CreateParamDef(ctx, t.i18nMgr, "predicted_next_steps", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_insights.arg.predicted_next_steps", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "confidence_level", toolcore.ParamTypeNumber, true, option.None[[]any](), "tool.agent_insights.arg.confidence_level", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "summary", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_insights.arg.summary", option.None[toolcore.ParameterDefinition]()),
	}

	return toolcore.ToolSchema{
		Name:             "agent_insights_generator",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: []toolcore.ParameterDefinition{},
	}, nil
}

func (t *AgentInsightsTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var result AgentInsightsResult
	if err := json.Unmarshal([]byte(inputJSON), &result); err != nil {
		t.logger.Error("解析 AgentInsightsResult 参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}
	return inputJSON, nil
}
