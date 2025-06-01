package types

// QualityAssessment 执行质量评估结果
type QualityAssessment struct {
	OverallQualityScore     float64            `json:"overall_quality_score"`    // 综合质量评分 (0.0-1.0)
	DimensionScores         map[string]float64 `json:"dimension_scores"`         // 各维度评分 (如: "efficiency", "accuracy")
	Strengths               []string           `json:"strengths"`                // 主要优点
	Weaknesses              []string           `json:"weaknesses"`               // 主要不足
	DetailedReport          string             `json:"detailed_report"`          // 详细评估文本
	RecommendedImprovements []string           `json:"recommended_improvements"` // 推荐改进点
}

// GeneratedInsights LLM生成的深度洞察结果
type GeneratedInsights struct {
	KeyTakeaways              []string `json:"key_takeaways"` // 核心发现
	ActionableRecommendations []struct {
		Recommendation  string `json:"recommendation"`   // 建议内容
		Rationale       string `json:"rationale"`        // 理由
		Priority        string `json:"priority"`         // 优先级 (high/medium/low)
		PotentialImpact string `json:"potential_impact"` // 潜在影响
	} `json:"actionable_recommendations"`
	ImprovementOpportunities []struct {
		Opportunity      string `json:"opportunity"`       // 机会描述
		PotentialBenefit string `json:"potential_benefit"` // 潜在益处
		Difficulty       string `json:"difficulty"`        // 实现难度 (high/medium/low)
	} `json:"improvement_opportunities"`
	PredictedNextSteps []string `json:"predicted_next_steps"` // 预测的下一步行动
	ConfidenceLevel    float64  `json:"confidence_level"`     // 洞察的置信度 (0.0-1.0)
	Summary            string   `json:"summary"`              // 洞察总结
}

// FeedbackAnalysisRecord 简单定义，因为在 `core` 包中已存在，这里是为了让 `interfaces.go` 中的引用能够解析。
// 在实际的接口使用中，`core.FeedbackAnalysisRecord` 将被使用。
// 或者，可以将 FeedbackAnalysisRecord 的完整定义移至此 types 包中，以避免包之间的依赖问题。
// For now, this is a placeholder if core.FeedbackAnalysisRecord is not directly accessible or causes import cycles.
// However, based on the files, `interfaces.go` is in package `core` and `feedback_analyzer.go` (which defines FeedbackAnalysisRecord) is also in `core`.
// So `interfaces.go` can directly refer to `FeedbackAnalysisRecord` without needing it in `types` package.
// The TODO markdown suggests `history []FeedbackAnalysisRecord` for AssessRisk in the interface.
// Let's assume `core.FeedbackAnalysisRecord` is the one to be used in the interface.

// Ensure other necessary types like ExecutionResult, Task, SystemState, AgentOutputAnalysis, RiskAssessment are also in this package or properly imported where needed.
// For now, assuming they are already defined in this 'types' package based on existing code structure.
