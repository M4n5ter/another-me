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

// AgentQualityAssessmentResult mirrors the structure in feedback_analyzer.go
// AgentQualityAssessmentResult LLM返回的质量评估结果
type AgentQualityAssessmentResult struct {
	OverallQualityScore     float64            `json:"overall_quality_score"`
	DimensionScores         map[string]float64 `json:"dimension_scores"`
	Strengths               []string           `json:"strengths"`
	Weaknesses              []string           `json:"weaknesses"`
	DetailedReport          string             `json:"detailed_report"`
	RecommendedImprovements []string           `json:"recommended_improvements"`
}

type AgentQualityAssessmentTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

func NewAgentQualityAssessmentTool(i18nMgr *i18n.Manager) *AgentQualityAssessmentTool {
	return &AgentQualityAssessmentTool{
		logger:  slog.Default().WithGroup("agent_quality_assessment_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*AgentQualityAssessmentTool)(nil)

func (t *AgentQualityAssessmentTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		// Placeholder i18n keys
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.agent_quality_assessment.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.agent_quality_assessment.name", nil)
	}

	// For map[string]float64 (DimensionScores), we define it as an object with additionalProperties of type number.
	// However, toolcore.ParameterDefinition doesn't directly support "additionalProperties".
	// A common way to represent this for LLMs is often as an array of key-value pairs or a more structured object if keys are known.
	// Given the current toolcore capabilities, we might represent it as a generic object or expect a specific structure if possible.
	// For now, let's treat it as a general object, which means the LLM needs to ensure the map structure.
	// Alternatively, if specific dimension score keys are known (e.g., "efficiency", "accuracy"), they can be defined explicitly.
	// Assuming generic object for now.

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "overall_quality_score", toolcore.ParamTypeNumber, true, option.None[[]any](), "tool.agent_quality_assessment.arg.overall_quality_score", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "dimension_scores", toolcore.ParamTypeObject, true, option.None[[]any](), "tool.agent_quality_assessment.arg.dimension_scores", option.None[toolcore.ParameterDefinition]()), // Representing map as generic object
		common.CreateParamDef(ctx, t.i18nMgr, "strengths", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_quality_assessment.arg.strengths", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "weaknesses", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_quality_assessment.arg.weaknesses", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
		common.CreateParamDef(ctx, t.i18nMgr, "detailed_report", toolcore.ParamTypeString, true, option.None[[]any](), "tool.agent_quality_assessment.arg.detailed_report", option.None[toolcore.ParameterDefinition]()),
		common.CreateParamDef(ctx, t.i18nMgr, "recommended_improvements", toolcore.ParamTypeArray, true, option.None[[]any](), "tool.agent_quality_assessment.arg.recommended_improvements", option.Some(toolcore.ParameterDefinition{Type: toolcore.ParamTypeString})),
	}

	return toolcore.ToolSchema{
		Name:             "agent_quality_assessor",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: []toolcore.ParameterDefinition{},
	}, nil
}

func (t *AgentQualityAssessmentTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var result AgentQualityAssessmentResult
	if err := json.Unmarshal([]byte(inputJSON), &result); err != nil {
		t.logger.Error("解析 AgentQualityAssessmentResult 参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}
	return inputJSON, nil
}
