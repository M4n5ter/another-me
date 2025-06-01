package core

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	json "github.com/json-iterator/go"

	feedbackanalyzer "github.com/m4n5ter/another-me/internal/core/tools/feedback_analyzer"

	"github.com/m4n5ter/another-me/internal/core/types"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	"github.com/m4n5ter/another-me/pkg/reactagent"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

const agentPromptTemplateDefault = `你是一位专注于Agent执行结果分析的智能反馈专家。

**任务**: 请基于以下提供的Agent执行数据和分析要求，进行深入分析。

**执行数据概览**:
- 总结果数: {{.TotalResults}}
- 成功数: {{.SuccessCount}}
- 失败数: {{.FailureCount}}
- 成功率: {{.SuccessRate}}%

**详细执行结果**:
{{.DetailedResults}}

**核心分析要求**:
{{.AnalysisRequirements}}

**历史分析参考**:
{{if .HistoricalContext}}
  {{range $key, $value := .HistoricalContext}}
- {{$key}}: {{$value}}
  {{end}}
{{else}}
- 无可用的历史分析数据。
{{end}}

**指示**: 请运用 "AgentFeedbackAnalysis" 工具来构建你的分析报告。确保全面覆盖工具所要求的所有参数，包括：关键发现、可行动的洞察、是否需要用户输入、分析的置信水平、推荐行动、风险评估详情、下一步骤建议、检测到的模式以及潜在的改进机会。

**请遵循以下分析原则**:
1. 深度挖掘执行模式与趋势。
2. 精准识别潜在问题与改进契机。
3. 提供具体且可操作的建议。
4. 全面评估相关风险并提出有效的缓解策略。
5. (若可能)基于历史数据预测未来趋势。
6. 清晰阐述分析的推理过程及置信度评估。`

const qualityAssessmentPromptTemplate = `你是一位经验丰富的AI应用执行质量评估专家。

**任务**: 请根据下方提供的Agent执行数据，进行全面细致的质量评估。

**执行数据概览**:
- 总结果数: {{.TotalResults}}
- 成功数: {{.SuccessCount}}
- 失败数: {{.FailureCount}}
- 成功率: {{.SuccessRate}}%

**详细执行结果**:
{{.DetailedResults}}

**核心评估要求**:
{{.AnalysisRequirements}}

**历史评估参考**:
{{if .HistoricalContext}}
  {{range $key, $value := .HistoricalContext}}
- {{$key}}: {{$value}}
  {{end}}
{{else}}
- 无可用的历史评估数据。
{{end}}

**指示**: 请使用 "AgentQualityAssessment" 工具提交你的评估报告。你需要准确填充工具所需的全部参数，包括：综合质量评分，各维度（如效率、准确性、稳定性、资源利用率）的评分，主要的优点和不足，详细的文字评估报告，以及具体的改进建议。

**评估重点维度**:
1.  **效率**: 任务执行速度、耗时、有无不必要的延迟。
2.  **准确性/正确性**: 任务成功率、错误发生情况、输出结果是否符合预期。
3.  **稳定性**: 执行过程中是否出现异常、崩溃、重试等情况。
4.  **资源利用率**: (若有数据) CPU、内存等资源使用是否合理。`

const riskAssessmentPromptTemplate = `你是一位资深的AI系统风险管理与预测专家。

**任务**: 请基于当前执行结果、计划中的后续行动、当前系统状态以及历史分析数据，进行全面的潜在风险评估。

**当前执行结果摘要**:
{{.CurrentResultsSummary}}

**计划行动摘要**:
{{.ProposedActionsSummary}}

**当前系统状态摘要**:
{{.SystemStateSummary}}

**核心风险评估要求**:
{{.AnalysisRequirements}}

**历史风险数据参考**:
{{if .HistoricalContext}}
  {{range $key, $value := .HistoricalContext}}
- {{$key}}: {{$value}}
  {{end}}
{{else}}
- 无可用的历史风险数据。
{{end}}

**指示**: 请使用 "AgentRiskAssessment" 工具提交你的风险评估报告。务必完整填写工具所需参数，包括：风险级别（高/中/低）、主要的风险因素、具体的缓解措施、对整体风险情况的详细描述、风险发生时的预测影响，以及对风险发生可能性的评估。`

const insightsGenerationPromptTemplate = `你是一名顶级的AI系统分析与策略顾问。

**任务**: 请基于提供的初步分析报告、历史执行数据摘要以及已识别的关键执行模式，进行深度分析，并生成一份富有洞察力的报告。

**基础分析摘要**:
{{.BaseAnalysisSummary}}

**历史执行数据摘要**:
{{.HistoricalDataSummary}}

**已识别模式摘要**:
{{.DetectedPatternsSummary}}

**核心洞察要求**:
{{.AnalysisRequirements}}

**历史洞察参考**:
{{if .HistoricalContext}}
  {{range $key, $value := .HistoricalContext}}
- {{$key}}: {{$value}}
  {{end}}
{{else}}
- 无可用的历史洞察数据。
{{end}}

**指示**: 请运用 "AgentInsights" 工具来构建并提交你的洞察报告。确保全面、准确地填充该工具的所有参数，包括：核心发现总结，可执行的改进建议（每条均需包含具体建议、理由、优先级评估及潜在影响描述），潜在的改进机会（每项均需包含机会描述、潜在益处及预估实现难度），预测的下一步行动，你对整体洞察的置信度评分，以及对整个洞察报告的简明扼要的总结。`

// SmartFeedbackAnalyzer 智能反馈分析器实现 - 基于Agent的智能分析
type SmartFeedbackAnalyzer struct {
	agent      *reactagent.ReActAgent
	llmAdapter llminterface.ChatAdapter
	logger     *slog.Logger
	config     FeedbackAnalyzerConfig

	// 分析历史缓存
	analysisHistory []FeedbackAnalysisRecord
	patternCache    map[string]PatternAnalysisResult
}

// FeedbackAnalyzerConfig 反馈分析器配置
type FeedbackAnalyzerConfig struct {
	// Agent配置
	AgentPromptTemplate string        `json:"agent_prompt_template"` // Agent提示词模板
	AnalysisTimeout     time.Duration `json:"analysis_timeout"`      // 分析超时时间
	MaxRetries          int           `json:"max_retries"`           // 最大重试次数

	// 分析配置
	AnalysisDepth            string  `json:"analysis_depth"`             // 分析深度: basic, detailed, comprehensive
	EnablePatternDetection   bool    `json:"enable_pattern_detection"`   // 启用模式检测
	EnablePredictiveAnalysis bool    `json:"enable_predictive_analysis"` // 启用预测分析
	MinConfidenceThreshold   float64 `json:"min_confidence_threshold"`   // 最小置信度阈值

	// 模式检测配置
	PatternWindowSize          int     `json:"pattern_window_size"`          // 模式检测窗口大小
	PatternMinOccurrences      int     `json:"pattern_min_occurrences"`      // 模式最小出现次数
	PatternSimilarityThreshold float64 `json:"pattern_similarity_threshold"` // 模式相似度阈值

	// 风险评估配置
	RiskFactorWeights map[string]float64 `json:"risk_factor_weights"` // 风险因子权重
	DefaultRiskLevel  string             `json:"default_risk_level"`  // 默认风险级别

	// 历史记录配置
	HistoryRetention      int  `json:"history_retention"`       // 历史记录保留数量
	EnableHistoryAnalysis bool `json:"enable_history_analysis"` // 启用历史分析

	// 缓存配置
	EnablePatternCache bool          `json:"enable_pattern_cache"` // 启用模式缓存
	CacheTTL           time.Duration `json:"cache_ttl"`            // 缓存TTL
}

// 分析相关的数据结构

// FeedbackAnalysisRecord 反馈分析记录
type FeedbackAnalysisRecord struct {
	Timestamp       time.Time                 `json:"timestamp"`
	Results         []types.ExecutionResult   `json:"results"`
	Analysis        types.AgentOutputAnalysis `json:"analysis"`
	AnalysisContext map[string]any            `json:"analysis_context"`
}

// PatternAnalysisResult 模式分析结果
type PatternAnalysisResult struct {
	Pattern     string         `json:"pattern"`
	Confidence  float64        `json:"confidence"`
	Occurrences int            `json:"occurrences"`
	Context     map[string]any `json:"context"`
	CachedAt    time.Time      `json:"cached_at"`
}

// AgentAnalysisPromptData Agent分析提示数据
type AgentAnalysisPromptData struct {
	TotalResults         int            `json:"total_results"`
	SuccessCount         int            `json:"success_count"`
	FailureCount         int            `json:"failure_count"`
	SuccessRate          float64        `json:"success_rate"`
	DetailedResults      string         `json:"detailed_results"`
	AnalysisRequirements string         `json:"analysis_requirements"`
	HistoricalContext    map[string]any `json:"historical_context"`
}

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

// --- Structs for Quality Assessment ---
// QualityAssessmentPromptData LLM质量评估提示数据
type QualityAssessmentPromptData struct {
	TotalResults         int            `json:"total_results"`
	SuccessCount         int            `json:"success_count"`
	FailureCount         int            `json:"failure_count"`
	SuccessRate          float64        `json:"success_rate"`
	DetailedResults      string         `json:"detailed_results"`
	AnalysisRequirements string         `json:"analysis_requirements"`
	HistoricalContext    map[string]any `json:"historical_context,omitempty"`
}

// AgentQualityAssessmentResult LLM返回的质量评估结果 (matches JSON in qualityAssessmentPromptTemplate)
type AgentQualityAssessmentResult struct {
	OverallQualityScore     float64            `json:"overall_quality_score"`
	DimensionScores         map[string]float64 `json:"dimension_scores"`
	Strengths               []string           `json:"strengths"`
	Weaknesses              []string           `json:"weaknesses"`
	DetailedReport          string             `json:"detailed_report"`
	RecommendedImprovements []string           `json:"recommended_improvements"`
}

// --- Structs for Insights Generation ---
// InsightsGenerationPromptData LLM洞察生成的提示数据
type InsightsGenerationPromptData struct {
	BaseAnalysisSummary     string         `json:"base_analysis_summary"`     // 基础分析的摘要
	HistoricalDataSummary   string         `json:"historical_data_summary"`   // 历史执行数据的摘要
	DetectedPatternsSummary string         `json:"detected_patterns_summary"` // 已识别模式的摘要
	AnalysisRequirements    string         `json:"analysis_requirements"`     // 分析的具体要求
	HistoricalContext       map[string]any `json:"historical_context,omitempty"`
}

// AgentInsightsResult LLM返回的洞察结果 (matches JSON in insightsGenerationPromptTemplate)
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

// DefaultFeedbackAnalyzerConfig 返回默认配置
func DefaultFeedbackAnalyzerConfig() FeedbackAnalyzerConfig {
	return FeedbackAnalyzerConfig{
		AgentPromptTemplate:        agentPromptTemplateDefault,
		AnalysisTimeout:            30 * time.Second,
		MaxRetries:                 3,
		AnalysisDepth:              "detailed",
		EnablePatternDetection:     true,
		EnablePredictiveAnalysis:   true,
		MinConfidenceThreshold:     0.6,
		PatternWindowSize:          10,
		PatternMinOccurrences:      3,
		PatternSimilarityThreshold: 0.8,
		RiskFactorWeights: map[string]float64{
			"failure_rate":            0.4,
			"error_frequency":         0.3,
			"performance_degradation": 0.2,
			"resource_consumption":    0.1,
		},
		DefaultRiskLevel:      "medium",
		HistoryRetention:      50,
		EnableHistoryAnalysis: true,
		EnablePatternCache:    true,
		CacheTTL:              1 * time.Hour,
	}
}

// NewSmartFeedbackAnalyzer 创建新的智能反馈分析器
func NewSmartFeedbackAnalyzer(
	agent *reactagent.ReActAgent,
	llmAdapter llminterface.ChatAdapter,
	config FeedbackAnalyzerConfig,
	logger *slog.Logger,
) *SmartFeedbackAnalyzer {
	if logger == nil {
		logger = slog.Default().WithGroup("smart_feedback_analyzer")
	}

	registry := toolcore.NewRegistry()
	err := feedbackanalyzer.RegistFeedbackAnalyzerTools(context.Background(), registry)
	if err != nil {
		logger.Error("注册反馈分析工具失败，无法继续初始化", "error", err)
		return nil
	}

	err = llmAdapter.RegisterTools(context.Background(), registry)
	if err != nil {
		logger.Error("注册决策工具失败，无法继续初始化", "error", err)
		return nil
	}

	return &SmartFeedbackAnalyzer{
		agent:           agent,
		llmAdapter:      llmAdapter,
		logger:          logger,
		config:          config,
		analysisHistory: make([]FeedbackAnalysisRecord, 0, config.HistoryRetention),
		patternCache:    make(map[string]PatternAnalysisResult),
	}
}

var _ FeedbackAnalyzer = (*SmartFeedbackAnalyzer)(nil)

// AnalyzeExecutionResults 分析执行结果 - 实现FeedbackAnalyzer接口
func (fa *SmartFeedbackAnalyzer) AnalyzeExecutionResults(
	ctx context.Context,
	results []types.ExecutionResult,
) (types.AgentOutputAnalysis, error) {
	if len(results) == 0 {
		return types.AgentOutputAnalysis{
			KeyFindings:         []string{"没有执行结果"},
			ActionableInsights:  []string{},
			RequiresUserInput:   false,
			ConfidenceLevel:     0.0,
			RecommendedActions:  []string{},
			RiskAssessment:      types.RiskAssessment{Level: "low"},
			NextStepSuggestions: []string{},
		}, nil
	}

	fa.logger.Info("开始AI驱动的执行结果分析", "result_count", len(results))

	analysisCtx, cancel := context.WithTimeout(ctx, fa.config.AnalysisTimeout)
	defer cancel()

	// 1. 构建分析提示数据
	promptData, err := fa.buildAnalysisPromptData(results)
	if err != nil {
		fa.logger.Error("构建Agent分析提示数据失败", "error", err)
		return fa.createFallbackAnalysis(results), nil
	}

	promptMessage, err := fa.buildLLMPromptMessage(promptData, agentPromptTemplateDefault, "AgentAnalysis")
	if err != nil {
		fa.logger.Error("构建Agent分析提示消息失败", "error", err)
		return fa.createFallbackAnalysis(results), nil
	}

	agentResult, err := fa.invokeLLMForAnalysis(analysisCtx, promptMessage)
	if err != nil {
		fa.logger.Error("AI分析失败，使用后备分析", "error", err)
		return fa.createFallbackAnalysis(results), nil
	}

	analysis, err := fa.convertAgentResultToAnalysis(agentResult)
	if err != nil {
		fa.logger.Error("转换Agent结果为标准分析结果失败", "error", err)
		return fa.createFallbackAnalysis(results), nil
	}

	fa.updateAnalysisHistory(results, analysis)

	fa.logger.Info("AI执行结果分析完成",
		"confidence", analysis.ConfidenceLevel,
		"findings_count", len(analysis.KeyFindings),
		"insights_count", len(analysis.ActionableInsights))

	return analysis, nil
}

// DetectPatterns 检测执行模式 - 实现FeedbackAnalyzer接口
func (fa *SmartFeedbackAnalyzer) DetectPatterns(
	ctx context.Context,
	history []types.ExecutionResult,
) ([]string, error) {
	fa.logger.Debug("检测执行模式", "history_count", len(history))

	if len(history) < 3 {
		return []string{"历史数据不足，无法检测模式"}, nil
	}

	patterns := []string{}

	// 1. 成功/失败模式
	successPattern := fa.detectSuccessFailurePattern(history)
	if successPattern != "" {
		patterns = append(patterns, successPattern)
	}

	// 2. 时间模式
	timePattern := fa.detectTimePattern(history)
	if timePattern != "" {
		patterns = append(patterns, timePattern)
	}

	// 3. 任务类型模式
	taskTypePattern := fa.detectTaskTypePattern(history)
	if taskTypePattern != "" {
		patterns = append(patterns, taskTypePattern)
	}

	// 4. 错误模式
	errorPattern := fa.detectErrorPattern(history)
	if errorPattern != "" {
		patterns = append(patterns, errorPattern)
	}

	// 5. 性能模式
	performancePattern := fa.detectPerformancePattern(history)
	if performancePattern != "" {
		patterns = append(patterns, performancePattern)
	}

	if len(patterns) == 0 {
		patterns = append(patterns, "未检测到明显的执行模式")
	}

	fa.logger.Info("模式检测完成", "pattern_count", len(patterns))
	return patterns, nil
}

// PredictNextSteps 预测下一步操作 - 实现FeedbackAnalyzer接口
func (fa *SmartFeedbackAnalyzer) PredictNextSteps(
	ctx context.Context,
	currentResults []types.ExecutionResult,
	systemState types.SystemState,
) ([]string, error) {
	fa.logger.Debug("预测下一步操作",
		"current_results", len(currentResults),
		"system_active", systemState.IsActive)

	if !fa.config.EnablePredictiveAnalysis {
		return []string{"预测分析未启用"}, nil
	}

	predictions := []string{}

	// 1. 基于当前结果预测
	if len(currentResults) > 0 {
		resultBasedPrediction := fa.predictFromResults(currentResults)
		predictions = append(predictions, resultBasedPrediction...)
	}

	// 2. 基于系统状态预测
	stateBasedPrediction := fa.predictFromSystemState(systemState)
	predictions = append(predictions, stateBasedPrediction...)

	// 3. 基于历史模式预测
	if len(fa.analysisHistory) > 0 {
		historyBasedPrediction := fa.predictFromHistory()
		predictions = append(predictions, historyBasedPrediction...)
	}

	// 4. 去重和排序
	predictions = fa.deduplicateAndSort(predictions)

	fa.logger.Info("下一步预测完成", "prediction_count", len(predictions))
	return predictions, nil
}

// AssessRisk 评估风险 - 实现FeedbackAnalyzer接口 (LLM Implementation)
func (fa *SmartFeedbackAnalyzer) AssessRisk(
	ctx context.Context,
	currentResults []types.ExecutionResult,
	proposedActions []types.Task,
	systemState types.SystemState,
	history []FeedbackAnalysisRecord,
) (types.RiskAssessment, error) {
	fa.logger.Debug("开始LLM驱动的风险评估", "current_results", len(currentResults), "actions", len(proposedActions))

	// 1. 构建分析提示数据
	promptData, err := fa.buildRiskAssessmentPromptData(currentResults, proposedActions, systemState, history)
	if err != nil {
		fa.logger.Error("构建LLM风险评估提示数据失败", "error", err)
		// Fallback to a simple rule-based assessment or error
		return types.RiskAssessment{Level: "unknown", Description: fmt.Sprintf("构建风险评估提示失败: %v", err)}, err
	}

	// 2. 构建LLM提示消息
	promptMessage, err := fa.buildLLMPromptMessage(promptData, riskAssessmentPromptTemplate, "RiskAssessment")
	if err != nil {
		fa.logger.Error("构建LLM风险评估提示消息失败", "error", err)
		return types.RiskAssessment{Level: "unknown", Description: fmt.Sprintf("构建风险评估提示消息失败: %v", err)}, err
	}

	analysisCtx, cancel := context.WithTimeout(ctx, fa.config.AnalysisTimeout) // Reuse existing timeout
	defer cancel()

	// 3. 调用LLM进行智能分析
	llmResultJSON, err := fa.invokeLLMForAnalysis(analysisCtx, promptMessage)
	if err != nil {
		fa.logger.Error("LLM风险评估失败", "error", err)
		wrappedError := fmt.Errorf("LLM风险评估API调用失败: %w", err)
		return types.RiskAssessment{Level: "unknown", Description: fmt.Sprintf("LLM风险评估API调用失败: %v", err)}, wrappedError // Return the wrapped error
	}

	// 4. 转换LLM结果为标准分析结果
	riskAssessment, err := fa.convertLLMResultToRiskAssessment(llmResultJSON)
	if err != nil {
		fa.logger.Error("转换LLM风险评估结果失败", "error", err, "llm_json", llmResultJSON)
		return types.RiskAssessment{Level: "unknown", Description: fmt.Sprintf("转换LLM风险评估结果失败: %v", err)}, err
	}

	fa.logger.Info("LLM风险评估完成", "risk_level", riskAssessment.Level)
	return riskAssessment, nil
}

// GenerateInsights 生成洞察 - 实现FeedbackAnalyzer接口 (LLM Implementation)
func (fa *SmartFeedbackAnalyzer) GenerateInsights(
	ctx context.Context,
	baseAnalysis types.AgentOutputAnalysis, // 新增参数
	history []types.ExecutionResult, // 新增参数
	detectedPatterns []string, // 新增参数
) (types.GeneratedInsights, error) { // 返回类型修改
	fa.logger.Debug("开始LLM驱动的洞察生成", "base_analysis_findings", len(baseAnalysis.KeyFindings), "history_count", len(history), "patterns_count", len(detectedPatterns))

	// 1. 构建分析提示数据
	promptData, err := fa.buildInsightsGenerationPromptData(baseAnalysis, history, detectedPatterns)
	if err != nil {
		fa.logger.Error("构建LLM洞察生成提示数据失败", "error", err)
		// Fallback: return a basic insight or error
		wrappedError := fmt.Errorf("构建洞察生成提示数据失败: %w", err)
		return types.GeneratedInsights{Summary: fmt.Sprintf("构建洞察生成提示失败: %v", err)}, wrappedError // Return the wrapped error
	}

	// 2. 构建LLM提示消息
	promptMessage, err := fa.buildLLMPromptMessage(promptData, insightsGenerationPromptTemplate, "InsightsGeneration")
	if err != nil {
		fa.logger.Error("构建LLM洞察生成提示消息失败", "error", err)
		wrappedError := fmt.Errorf("构建洞察生成提示消息失败: %w", err)
		return types.GeneratedInsights{Summary: fmt.Sprintf("构建洞察生成提示消息失败: %v", err)}, wrappedError // Return the wrapped error
	}

	analysisCtx, cancel := context.WithTimeout(ctx, fa.config.AnalysisTimeout) // Reuse existing timeout
	defer cancel()

	// 3. 调用LLM进行智能分析
	llmResultJSON, err := fa.invokeLLMForAnalysis(analysisCtx, promptMessage)
	if err != nil {
		fa.logger.Error("LLM洞察生成失败", "error", err)
		wrappedError := fmt.Errorf("LLM洞察生成API调用失败: %w", err)
		return types.GeneratedInsights{Summary: fmt.Sprintf("LLM洞察生成API调用失败: %v", err)}, wrappedError // Return the wrapped error
	}

	// 4. 转换LLM结果为标准分析结果
	insights, err := fa.convertLLMResultToGeneratedInsights(llmResultJSON)
	if err != nil {
		fa.logger.Error("转换LLM洞察生成结果失败", "error", err, "llm_json", llmResultJSON)
		wrappedError := fmt.Errorf("转换LLM洞察生成结果失败: %w", err)
		return types.GeneratedInsights{Summary: fmt.Sprintf("转换LLM洞察生成结果失败: %v", err)}, wrappedError // Return the wrapped error
	}

	fa.logger.Info("LLM洞察生成完成", "key_takeaways", len(insights.KeyTakeaways), "confidence", insights.ConfidenceLevel)
	return insights, nil
}

// AssessExecutionQuality 评估执行质量 - 实现FeedbackAnalyzer接口 (新方法)
func (fa *SmartFeedbackAnalyzer) AssessExecutionQuality(
	ctx context.Context,
	results []types.ExecutionResult,
) (types.QualityAssessment, error) {
	fa.logger.Debug("开始LLM驱动的执行质量评估", "result_count", len(results))

	if len(results) == 0 {
		fa.logger.Info("没有执行结果可供质量评估")
		return types.QualityAssessment{}, nil
	}

	// 1. 构建分析提示数据
	// For detailed results, we can reuse parts of buildAnalysisPromptData logic if appropriate or simplify
	// For now, let's create a simplified representation for detailed results.
	promptData, err := fa.buildQualityAssessmentPromptData(results)
	if err != nil {
		fa.logger.Error("构建LLM质量评估提示数据失败", "error", err)
		// Consider a fallback or simple rule-based assessment here if desired
		return types.QualityAssessment{}, fmt.Errorf("构建质量评估提示数据失败: %w", err)
	}

	// 2. 构建LLM提示消息
	promptMessage, err := fa.buildLLMPromptMessage(promptData, qualityAssessmentPromptTemplate, "QualityAssessment")
	if err != nil {
		fa.logger.Error("构建LLM质量评估提示消息失败", "error", err)
		return types.QualityAssessment{}, fmt.Errorf("构建质量评估提示消息失败: %w", err)
	}

	// 设置分析超时 (can use existing config or a new one for quality assessment)
	analysisCtx, cancel := context.WithTimeout(ctx, fa.config.AnalysisTimeout) // Using existing AnalysisTimeout
	defer cancel()

	// 3. 调用LLM进行智能分析
	llmResultJSON, err := fa.invokeLLMForAnalysis(analysisCtx, promptMessage)
	if err != nil {
		fa.logger.Error("LLM质量评估失败", "error", err)
		// Fallback: return a basic assessment or error
		return types.QualityAssessment{}, fmt.Errorf("LLM质量评估API调用失败: %w", err) // Propagate error to caller to decide
	}

	// 4. 转换LLM结果为标准分析结果
	qualityAssessment, err := fa.convertLLMResultToQualityAssessment(llmResultJSON)
	if err != nil {
		fa.logger.Error("转换LLM质量评估结果失败", "error", err, "llm_json", llmResultJSON)
		return types.QualityAssessment{}, fmt.Errorf("转换LLM质量评估结果失败: %w", err)
	}

	fa.logger.Info("LLM执行质量评估完成", "overall_score", qualityAssessment.OverallQualityScore)
	return qualityAssessment, nil
}

// buildQualityAssessmentPromptData 构建质量评估的提示数据
func (fa *SmartFeedbackAnalyzer) buildQualityAssessmentPromptData(results []types.ExecutionResult) (QualityAssessmentPromptData, error) {
	totalResults := len(results)
	successCount := 0
	failureCount := 0
	detailedResultsBuilder := strings.Builder{}

	// Simplified detailed results for quality assessment prompt (can be expanded)
	for i, res := range results {
		if res.Status == types.ExecutionStatusSuccess {
			successCount++
		} else if res.Status == types.ExecutionStatusFailure {
			failureCount++
		}
		detailedResultsBuilder.WriteString(fmt.Sprintf("  结果 %d: 状态=%s, 耗时=%s", i+1, res.Status, res.EndTime.Sub(res.StartTime)))
		if res.Error != "" {
			detailedResultsBuilder.WriteString(fmt.Sprintf(", 错误=%s", res.Error))
		}
		detailedResultsBuilder.WriteString("\n")
	}

	successRate := 0.0
	if totalResults > 0 {
		successRate = float64(successCount) / float64(totalResults) * 100
	}

	// TODO: Populate AnalysisRequirements and HistoricalContext meaningfully for quality assessment
	analysisRequirements := "请对执行效率、准确性、稳定性进行综合评估。"

	data := QualityAssessmentPromptData{
		TotalResults:         totalResults,
		SuccessCount:         successCount,
		FailureCount:         failureCount,
		SuccessRate:          successRate,
		DetailedResults:      detailedResultsBuilder.String(),
		AnalysisRequirements: analysisRequirements,
		// HistoricalContext can be added if relevant past quality scores are stored
	}
	return data, nil
}

// convertLLMResultToQualityAssessment 将LLM的JSON输出转换为 QualityAssessment 类型
func (fa *SmartFeedbackAnalyzer) convertLLMResultToQualityAssessment(llmResultJSON string) (types.QualityAssessment, error) {
	cleanedJSON := fa.cleanLLMJSONOutput(llmResultJSON) // Using a common cleaning function

	var agentAssessment AgentQualityAssessmentResult
	err := json.Unmarshal([]byte(cleanedJSON), &agentAssessment)
	if err != nil {
		return types.QualityAssessment{}, fmt.Errorf("解析LLM质量评估JSON失败: %w, json_content: %s", err, cleanedJSON)
	}

	// Map from AgentQualityAssessmentResult to types.QualityAssessment
	// In this case, fields are identical, so direct mapping works.
	// If structures were different, manual mapping would be needed.
	return types.QualityAssessment{
		OverallQualityScore:     agentAssessment.OverallQualityScore,
		DimensionScores:         agentAssessment.DimensionScores,
		Strengths:               agentAssessment.Strengths,
		Weaknesses:              agentAssessment.Weaknesses,
		DetailedReport:          agentAssessment.DetailedReport,
		RecommendedImprovements: agentAssessment.RecommendedImprovements,
	}, nil
}

func (fa *SmartFeedbackAnalyzer) getSystemPrompt() string {
	return `你是一位顶尖的AI反馈分析专家。你的任务是深入理解和分析用户提供的Agent执行数据，并生成有价值的反馈报告。

为了帮助你完成任务，你拥有以下强大的分析工具：
- **AgentFeedbackAnalysis**: 用于对Agent的执行结果进行全面和详细的反馈分析。
- **AgentQualityAssessment**: 用于评估Agent执行的整体质量，包括效率、准确性和稳定性等维度。
- **AgentRiskAssessment**: 用于识别和评估与Agent执行相关的潜在风险。
- **AgentInsights**: 用于从分析数据中提炼核心发现、可行动建议和改进机会。

请仔细理解用户的具体分析要求和提供的数据，然后选择最合适的工具来构建你的回应。确保为所选工具的所有参数提供准确和完整的信息。`
}

// buildLLMPromptMessage 构建LLM提示消息 (generic helper)
func (fa *SmartFeedbackAnalyzer) buildLLMPromptMessage(promptData any, templateContent, templateName string) (llminterface.InputMessage, error) {
	tmpl, err := template.New(templateName).Parse(templateContent)
	if err != nil {
		return llminterface.InputMessage{}, fmt.Errorf("解析LLM提示模板 '%s' 失败: %w", templateName, err)
	}

	var promptBuffer bytes.Buffer
	if err := tmpl.Execute(&promptBuffer, promptData); err != nil {
		return llminterface.InputMessage{}, fmt.Errorf("执行LLM提示模板 '%s' 失败: %w", templateName, err)
	}

	return llminterface.InputMessage{
		Role: llminterface.RoleSystem, // Or RoleUser, depending on how the LLM expects it for the specific task
		Content: []llminterface.ContentPart{
			{
				Type: llminterface.PartTypeText,
				Text: promptBuffer.String(), // This is the text content
			},
		},
	}, nil
}

// cleanLLMJSONOutput helper to remove markdown fences and trim space
func (fa *SmartFeedbackAnalyzer) cleanLLMJSONOutput(jsonString string) string {
	cleaned := jsonString
	if strings.HasPrefix(cleaned, "```json\n") {
		cleaned = strings.TrimPrefix(cleaned, "```json\n")
	}
	if strings.HasPrefix(cleaned, "```json") {
		cleaned = strings.TrimPrefix(cleaned, "```json")
	}
	if strings.HasSuffix(cleaned, "\n```") {
		cleaned = strings.TrimSuffix(cleaned, "\n```")
	}
	if strings.HasSuffix(cleaned, "```") {
		cleaned = strings.TrimSuffix(cleaned, "```")
	}
	return strings.TrimSpace(cleaned)
}

// invokeLLMForAnalysis 调用LLM进行分析
func (fa *SmartFeedbackAnalyzer) invokeLLMForAnalysis(ctx context.Context, msg llminterface.InputMessage) (string, error) {
	if fa.llmAdapter == nil {
		return "", fmt.Errorf("llmAdapter 未初始化")
	}

	response, err := llminterface.ChatAndGetFullResponse(ctx, fa.llmAdapter, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			llminterface.SystemInputMessage(fa.getSystemPrompt()),
			msg,
		},
		ConversationID: uuid.NewString(),
	})
	if err != nil {
		return "", fmt.Errorf("决策失败: %w", err)
	}

	if response.HasToolCalls() && len(response.GetToolCalls()) > 0 {
		return response.GetToolCalls()[0].Arguments, nil
	}

	fa.logger.Warn("LLM 响应不包含预期的工具调用")
	return "", fmt.Errorf("LLM 响应不包含工具调用")
}

// convertAgentResultToAnalysis 将Agent结果转换为标准分析结果
func (fa *SmartFeedbackAnalyzer) convertAgentResultToAnalysis(agentResultJSON string) (types.AgentOutputAnalysis, error) {
	cleanedJSON := fa.cleanLLMJSONOutput(agentResultJSON) // Use the helper

	var agentFeedback AgentFeedbackAnalysisResult
	err := json.Unmarshal([]byte(cleanedJSON), &agentFeedback)
	if err != nil {
		return types.AgentOutputAnalysis{}, fmt.Errorf("解析Agent反馈JSON失败: %w, json_content: %s", err, cleanedJSON)
	}

	analysis := types.AgentOutputAnalysis{
		KeyFindings:         agentFeedback.KeyFindings,
		ActionableInsights:  agentFeedback.ActionableInsights,
		RequiresUserInput:   agentFeedback.RequiresUserInput,
		ConfidenceLevel:     agentFeedback.ConfidenceLevel,
		RecommendedActions:  agentFeedback.RecommendedActions,
		RiskAssessment:      agentFeedback.RiskAssessment,
		NextStepSuggestions: agentFeedback.NextStepSuggestions,
	}
	if len(agentFeedback.PatternsDetected) > 0 {
		fa.logger.Debug("LLM检测到的模式", "patterns", agentFeedback.PatternsDetected)
	}
	if len(agentFeedback.ImprovementOpportunities) > 0 {
		fa.logger.Debug("LLM建议的改进机会", "opportunities", agentFeedback.ImprovementOpportunities)
	}

	return analysis, nil
}

// updateAnalysisHistory 更新分析历史
func (fa *SmartFeedbackAnalyzer) updateAnalysisHistory(results []types.ExecutionResult, analysis types.AgentOutputAnalysis) {
	record := FeedbackAnalysisRecord{
		Timestamp: time.Now(),
		Results:   results, // Storing full results might be memory intensive; consider summaries.
		Analysis:  analysis,
		AnalysisContext: map[string]any{
			"config_snapshot": fa.config, // Storing the whole config might be large. Consider storing key params or version.
			"result_count":    len(results),
		},
	}

	fa.analysisHistory = append(fa.analysisHistory, record)

	if len(fa.analysisHistory) > fa.config.HistoryRetention && fa.config.HistoryRetention > 0 {
		excess := len(fa.analysisHistory) - fa.config.HistoryRetention
		fa.analysisHistory = fa.analysisHistory[excess:]
	}
}

// createFallbackAnalysis 创建后备分析 (当AI分析失败时)
func (fa *SmartFeedbackAnalyzer) createFallbackAnalysis(results []types.ExecutionResult) types.AgentOutputAnalysis {
	fa.logger.Info("执行后备分析")
	basicStats := fa.performBasicAnalysis(results)

	failureRate, _ := basicStats["failure_rate"].(float64)
	errorCount := 0
	if errors, ok := basicStats["errors"].([]string); ok {
		errorCount = len(errors)
	}

	riskLevel := "low"
	if failureRate > 0.5 || errorCount > 3 {
		riskLevel = "high"
	} else if failureRate > 0.2 || errorCount > 0 {
		riskLevel = "medium"
	}

	keyFindings := []string{
		fmt.Sprintf("总共执行任务数: %v", basicStats["total_count"]),
		fmt.Sprintf("成功率: %.2f%%", basicStats["success_rate"].(float64)*100),
	}
	if errorCount > 0 {
		keyFindings = append(keyFindings, fmt.Sprintf("错误数: %d", errorCount))
	}

	return types.AgentOutputAnalysis{
		KeyFindings:        keyFindings,
		ActionableInsights: []string{"建议检查执行日志以获取详细信息。"},
		RequiresUserInput:  riskLevel == "high" || failureRate > 0.6,
		ConfidenceLevel:    0.2, // Low confidence for fallback
		RecommendedActions: []string{"如果问题持续存在，请查看错误并考虑调整配置。"},
		RiskAssessment: types.RiskAssessment{
			Level:       riskLevel,
			Factors:     []string{fmt.Sprintf("失败率: %.2f", failureRate), fmt.Sprintf("错误数: %d", errorCount)},
			Mitigation:  []string{"审查失败的任务并解决根本原因。"},
			Description: "基于基本统计数据的后备风险评估。",
		},
		NextStepSuggestions: []string{"重试失败的任务或调整策略。"},
	}
}

// 私有辅助方法

func (fa *SmartFeedbackAnalyzer) performBasicAnalysis(
	results []types.ExecutionResult,
) map[string]any {
	total := len(results)
	successCount := 0
	failureCount := 0
	inProgressCount := 0
	cancelledCount := 0

	totalDuration := time.Duration(0)
	errors := []string{}

	for _, result := range results {
		switch result.Status {
		case types.ExecutionStatusSuccess:
			successCount++
		case types.ExecutionStatusFailure:
			failureCount++
			if result.Error != "" {
				errors = append(errors, result.Error)
			}
		case types.ExecutionStatusInProgress:
			inProgressCount++
		case types.ExecutionStatusCancelled:
			cancelledCount++
		}

		duration := result.EndTime.Sub(result.StartTime)
		totalDuration += duration
	}

	var avgDuration time.Duration
	if total > 0 {
		avgDuration = totalDuration / time.Duration(total)
	}

	return map[string]any{
		"total_count":       total,
		"success_count":     successCount,
		"failure_count":     failureCount,
		"in_progress_count": inProgressCount,
		"cancelled_count":   cancelledCount,
		"success_rate":      float64(successCount) / float64(total),
		"failure_rate":      float64(failureCount) / float64(total),
		"average_duration":  avgDuration,
		"total_duration":    totalDuration,
		"errors":            errors,
	}
}

func (fa *SmartFeedbackAnalyzer) detectSuccessFailurePattern(
	history []types.ExecutionResult,
) string {
	if len(history) < 5 {
		return ""
	}

	// 检查最近的成功/失败模式
	recent := history[len(history)-5:]
	successCount := 0
	for _, result := range recent {
		if result.Status == types.ExecutionStatusSuccess {
			successCount++
		}
	}

	if successCount == 5 {
		return "连续成功模式：最近5次执行全部成功"
	} else if successCount == 0 {
		return "连续失败模式：最近5次执行全部失败"
	} else if successCount >= 4 {
		return "高成功率模式：最近执行成功率很高"
	} else if successCount <= 1 {
		return "高失败率模式：最近执行成功率很低"
	}

	return ""
}

func (fa *SmartFeedbackAnalyzer) detectTimePattern(
	history []types.ExecutionResult,
) string {
	if len(history) < 3 {
		return ""
	}

	durations := []time.Duration{}
	for _, result := range history {
		duration := result.EndTime.Sub(result.StartTime)
		durations = append(durations, duration)
	}

	// 检查执行时间趋势
	if len(durations) >= 3 {
		last3 := durations[len(durations)-3:]
		if last3[0] < last3[1] && last3[1] < last3[2] {
			return "执行时间递增模式：任务执行时间逐渐增长"
		} else if last3[0] > last3[1] && last3[1] > last3[2] {
			return "执行时间递减模式：任务执行时间逐渐缩短"
		}
	}

	return ""
}

func (fa *SmartFeedbackAnalyzer) detectErrorPattern(
	history []types.ExecutionResult,
) string {
	errorCount := 0
	for _, result := range history {
		if result.Status == types.ExecutionStatusFailure && result.Error != "" {
			errorCount++
		}
	}

	if errorCount > len(history)/2 {
		return "高错误率模式：大部分执行都出现错误"
	}

	return ""
}

func (fa *SmartFeedbackAnalyzer) detectPerformancePattern(
	history []types.ExecutionResult,
) string {
	if len(history) < 5 {
		return ""
	}

	// 计算平均执行时间
	totalDuration := time.Duration(0)
	for _, result := range history {
		totalDuration += result.EndTime.Sub(result.StartTime)
	}
	avgDuration := totalDuration / time.Duration(len(history))

	if avgDuration > 10*time.Minute {
		return "性能问题模式：平均执行时间过长"
	}

	return ""
}

func (fa *SmartFeedbackAnalyzer) predictFromResults(
	results []types.ExecutionResult,
) []string {
	predictions := []string{}

	successCount := 0
	for _, result := range results {
		if result.Status == types.ExecutionStatusSuccess {
			successCount++
		}
	}

	successRate := float64(successCount) / float64(len(results))

	if successRate > 0.8 {
		predictions = append(predictions, "基于当前成功率，建议继续当前策略")
	} else if successRate < 0.5 {
		predictions = append(predictions, "基于当前失败率，建议调整执行策略")
	}

	return predictions
}

func (fa *SmartFeedbackAnalyzer) predictFromSystemState(
	systemState types.SystemState,
) []string {
	predictions := []string{}

	if systemState.ErrorCount > 3 {
		predictions = append(predictions, "系统错误较多，建议进行健康检查")
	}

	if systemState.IsWaitingMode {
		predictions = append(predictions, "系统处于等待模式，可能需要外部触发")
	}

	return predictions
}

func (fa *SmartFeedbackAnalyzer) predictFromHistory() []string {
	predictions := []string{}

	if len(fa.analysisHistory) > 10 {
		recentAnalyses := fa.analysisHistory[len(fa.analysisHistory)-10:]
		avgConfidence := 0.0
		for _, analysis := range recentAnalyses {
			avgConfidence += analysis.Analysis.ConfidenceLevel
		}
		avgConfidence /= float64(len(recentAnalyses))

		if avgConfidence > 0.8 {
			predictions = append(predictions, "基于历史分析，系统运行稳定")
		} else if avgConfidence < 0.5 {
			predictions = append(predictions, "基于历史分析，系统需要优化")
		}
	}

	return predictions
}

func (fa *SmartFeedbackAnalyzer) deduplicateAndSort(
	predictions []string,
) []string {
	seen := make(map[string]bool)
	unique := []string{}

	for _, pred := range predictions {
		if !seen[pred] {
			seen[pred] = true
			unique = append(unique, pred)
		}
	}

	sort.Strings(unique)
	return unique
}

func (fa *SmartFeedbackAnalyzer) determineRiskLevel(score float64) string {
	if score >= 0.8 {
		return "high"
	} else if score >= 0.5 {
		return "medium"
	}
	return "low"
}

func (fa *SmartFeedbackAnalyzer) isCriticalError(errorMsg string) bool {
	criticalKeywords := []string{
		"panic", "fatal", "critical", "severe",
		"崩溃", "严重", "致命", "关键",
	}

	errorLower := strings.ToLower(errorMsg)
	for _, keyword := range criticalKeywords {
		if strings.Contains(errorLower, keyword) {
			return true
		}
	}

	return false
}

func (fa *SmartFeedbackAnalyzer) extractKeyTerms(
	observations []string,
	outputs []string,
) []string {
	// 简化的关键词提取
	termCount := make(map[string]int)

	allText := append(observations, outputs...)
	for _, text := range allText {
		words := strings.Fields(strings.ToLower(text))
		for _, word := range words {
			if len(word) > 3 { // 只考虑长度大于3的词
				termCount[word]++
			}
		}
	}

	// 找出频率最高的词
	var terms []string
	for term, count := range termCount {
		if count > 1 {
			terms = append(terms, term)
		}
	}

	if len(terms) > 10 {
		terms = terms[:10] // 限制返回数量
	}

	return terms
}

func (fa *SmartFeedbackAnalyzer) analyzeSentiment(
	observations []string,
	outputs []string,
) string {
	// 简化的情感分析
	positiveWords := []string{"成功", "完成", "正确", "好", "excellent", "success", "complete"}
	negativeWords := []string{"失败", "错误", "问题", "异常", "error", "fail", "problem", "issue"}

	allText := strings.Join(append(observations, outputs...), " ")
	allTextLower := strings.ToLower(allText)

	positiveCount := 0
	negativeCount := 0

	for _, word := range positiveWords {
		positiveCount += strings.Count(allTextLower, word)
	}

	for _, word := range negativeWords {
		negativeCount += strings.Count(allTextLower, word)
	}

	if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	}

	return "neutral"
}

func (fa *SmartFeedbackAnalyzer) analyzeDurationTrend(
	durations []time.Duration,
) string {
	if len(durations) < 3 {
		return "insufficient_data"
	}

	// 简化的趋势分析
	increasing := 0
	decreasing := 0

	for i := 1; i < len(durations); i++ {
		if durations[i] > durations[i-1] {
			increasing++
		} else if durations[i] < durations[i-1] {
			decreasing++
		}
	}

	if increasing > decreasing {
		return "increasing"
	} else if decreasing > increasing {
		return "decreasing"
	}

	return "stable"
}

func (fa *SmartFeedbackAnalyzer) buildAnalysisPromptData(results []types.ExecutionResult) (AgentAnalysisPromptData, error) {
	totalResults := len(results)
	successCount := 0
	failureCount := 0
	detailedResultsBuilder := strings.Builder{}

	for i, res := range results {
		if res.Status == types.ExecutionStatusSuccess {
			successCount++
		} else if res.Status == types.ExecutionStatusFailure {
			failureCount++
		}
		detailedResultsBuilder.WriteString(fmt.Sprintf("  结果 %d:%s", i+1, "\n"))
		detailedResultsBuilder.WriteString(fmt.Sprintf("    任务ID: %s%s", res.TaskID, "\n"))
		detailedResultsBuilder.WriteString(fmt.Sprintf("    状态: %s%s", res.Status, "\n"))
		if res.Error != "" {
			detailedResultsBuilder.WriteString(fmt.Sprintf("    错误: %s%s", res.Error, "\n"))
		}
		detailedResultsBuilder.WriteString(fmt.Sprintf("    开始时间: %s%s", res.StartTime.Format(time.RFC3339), "\n"))
		detailedResultsBuilder.WriteString(fmt.Sprintf("    结束时间: %s%s", res.EndTime.Format(time.RFC3339), "\n"))
		outputBytes, err := json.Marshal(res.Output)
		if err == nil {
			detailedResultsBuilder.WriteString(fmt.Sprintf("    输出: %s%s", string(outputBytes), "\n"))
		} else {
			detailedResultsBuilder.WriteString(fmt.Sprintf("    输出: (无法序列化: %v)%s", err, "\n"))
		}
		if len(res.Observations) > 0 {
			detailedResultsBuilder.WriteString(fmt.Sprintf("    观察: %s%s", strings.Join(res.Observations, "; "), "\n"))
		}
	}

	successRate := 0.0
	if totalResults > 0 {
		successRate = float64(successCount) / float64(totalResults) * 100
	}

	analysisRequirements := "请进行详细分析并提供可操作的见解。" // Default or from config
	historicalContext := make(map[string]any)
	if fa.config.EnableHistoryAnalysis && len(fa.analysisHistory) > 0 {
		historicalContext["previous_analysis_count"] = len(fa.analysisHistory)
		if len(fa.analysisHistory) > 0 {
			lastRecord := fa.analysisHistory[len(fa.analysisHistory)-1]
			historicalContext["last_analysis_confidence"] = lastRecord.Analysis.ConfidenceLevel
			historicalContext["last_analysis_risk"] = lastRecord.Analysis.RiskAssessment.Level
		}
	}

	data := AgentAnalysisPromptData{
		TotalResults:         totalResults,
		SuccessCount:         successCount,
		FailureCount:         failureCount,
		SuccessRate:          successRate,
		DetailedResults:      detailedResultsBuilder.String(),
		AnalysisRequirements: analysisRequirements,
		HistoricalContext:    historicalContext,
	}
	return data, nil
}

// --- Structs for Risk Assessment (New) ---
type RiskAssessmentPromptData struct {
	CurrentResultsSummary  string         `json:"current_results_summary"`
	ProposedActionsSummary string         `json:"proposed_actions_summary"`
	SystemStateSummary     string         `json:"system_state_summary"`
	HistoricalContext      map[string]any `json:"historical_context,omitempty"`
	AnalysisRequirements   string         `json:"analysis_requirements"`
}

// AgentRiskAssessmentResult LLM返回的风险评估结果 (matches JSON in riskAssessmentPromptTemplate)
type AgentRiskAssessmentResult struct {
	Level                   string   `json:"level"`
	Factors                 []string `json:"factors"`
	Mitigation              []string `json:"mitigation"`
	Description             string   `json:"description"`
	PredictedImpactIfOccurs string   `json:"predicted_impact_if_occurs,omitempty"` // Optional based on template
	ProbabilityAssessment   string   `json:"probability_assessment,omitempty"`     // Optional based on template
}

// buildRiskAssessmentPromptData 构建风险评估的提示数据
func (fa *SmartFeedbackAnalyzer) buildRiskAssessmentPromptData(
	currentResults []types.ExecutionResult,
	proposedActions []types.Task,
	systemState types.SystemState,
	history []FeedbackAnalysisRecord,
) (RiskAssessmentPromptData, error) {
	// Summarize currentResults
	crSummaryBuilder := strings.Builder{}
	crSummaryBuilder.WriteString(fmt.Sprintf("最近执行结果 (%d条):\n", len(currentResults)))
	for i, r := range currentResults {
		if i >= 3 { // Limit summary to a few recent results
			crSummaryBuilder.WriteString(fmt.Sprintf("...还有%d条更早的结果\n", len(currentResults)-i))
			break
		}
		crSummaryBuilder.WriteString(fmt.Sprintf("  - 状态: %s, 错误: %s\n", r.Status, r.Error))
	}

	// Summarize proposedActions
	paSummaryBuilder := strings.Builder{}
	paSummaryBuilder.WriteString(fmt.Sprintf("计划行动 (%d条):\n", len(proposedActions)))
	for i, a := range proposedActions {
		if i >= 3 { // Limit summary
			paSummaryBuilder.WriteString(fmt.Sprintf("...还有%d个行动\n", len(proposedActions)-i))
			break
		}
		paSummaryBuilder.WriteString(fmt.Sprintf("  - 任务ID: %s, 类型: %s, 优先级: %d\n", a.ID, a.Type, a.Priority))
	}

	// Summarize systemState
	ssSummary := fmt.Sprintf("当前系统状态: 活跃=%t, 错误数=%d, 等待模式=%t\n", systemState.IsActive, systemState.ErrorCount, systemState.IsWaitingMode)

	// Summarize history
	historicalCtx := make(map[string]any)
	if fa.config.EnableHistoryAnalysis && len(history) > 0 {
		historicalCtx["total_historical_analyses"] = len(history)
		// Add more specific historical data points if needed, e.g., past high-risk events
		if len(history) > 0 {
			lastRecord := history[len(history)-1]
			historicalCtx["last_analysis_risk_level"] = lastRecord.Analysis.RiskAssessment.Level
			historicalCtx["last_analysis_factors_count"] = len(lastRecord.Analysis.RiskAssessment.Factors)
		}
	}
	analysisReq := "请全面评估风险因素、可能性和潜在影响，并提供缓解措施。"

	data := RiskAssessmentPromptData{
		CurrentResultsSummary:  crSummaryBuilder.String(),
		ProposedActionsSummary: paSummaryBuilder.String(),
		SystemStateSummary:     ssSummary,
		HistoricalContext:      historicalCtx,
		AnalysisRequirements:   analysisReq,
	}
	return data, nil
}

// convertLLMResultToRiskAssessment 将LLM的JSON输出转换为 types.RiskAssessment 类型
func (fa *SmartFeedbackAnalyzer) convertLLMResultToRiskAssessment(llmResultJSON string) (types.RiskAssessment, error) {
	cleanedJSON := fa.cleanLLMJSONOutput(llmResultJSON)

	var agentAssessment AgentRiskAssessmentResult
	err := json.Unmarshal([]byte(cleanedJSON), &agentAssessment)
	if err != nil {
		return types.RiskAssessment{}, fmt.Errorf("解析LLM风险评估JSON失败: %w, json_content: %s", err, cleanedJSON)
	}

	// Map from AgentRiskAssessmentResult to types.RiskAssessment
	// Note: types.RiskAssessment might not have PredictedImpactIfOccurs and ProbabilityAssessment.
	// The prompt asks for them, so they'll be in AgentRiskAssessmentResult.
	// We only map what's available in types.RiskAssessment.
	// If types.RiskAssessment is to be extended, this mapping should be updated.
	riskAssessment := types.RiskAssessment{
		Level:       agentAssessment.Level,
		Factors:     agentAssessment.Factors,
		Mitigation:  agentAssessment.Mitigation,
		Description: agentAssessment.Description,
	}

	// Log extra fields if needed
	if agentAssessment.PredictedImpactIfOccurs != "" {
		fa.logger.Debug("LLM风险评估 - 预测影响", "impact", agentAssessment.PredictedImpactIfOccurs)
	}
	if agentAssessment.ProbabilityAssessment != "" {
		fa.logger.Debug("LLM风险评估 - 概率评估", "probability", agentAssessment.ProbabilityAssessment)
	}

	return riskAssessment, nil
}

// buildInsightsGenerationPromptData 构建洞察生成的提示数据
func (fa *SmartFeedbackAnalyzer) buildInsightsGenerationPromptData(
	baseAnalysis types.AgentOutputAnalysis,
	history []types.ExecutionResult,
	detectedPatterns []string,
) (InsightsGenerationPromptData, error) {
	// Summarize baseAnalysis
	baseAnalysisSummary := fmt.Sprintf("基础分析摘要: 关键发现数=%d, 风险级别=%s, 置信度=%.2f",
		len(baseAnalysis.KeyFindings),
		baseAnalysis.RiskAssessment.Level,
		baseAnalysis.ConfidenceLevel)

	// Summarize history - keeping it concise for the prompt
	historicalDataSummaryBuilder := strings.Builder{}
	historicalDataSummaryBuilder.WriteString(fmt.Sprintf("历史执行结果摘要 (%d条):\n", len(history)))
	successCount := 0
	failureCount := 0
	limit := 5 // Limit how many historical results to detail to keep prompt shorter
	for i, res := range history {
		if res.Status == types.ExecutionStatusSuccess {
			successCount++
		} else {
			failureCount++
		}
		if i < limit {
			historicalDataSummaryBuilder.WriteString(fmt.Sprintf("  - 结果 %d: 状态=%s, 耗时=%s\n",
				i+1, res.Status, res.EndTime.Sub(res.StartTime)))
		}
	}
	if len(history) > limit {
		historicalDataSummaryBuilder.WriteString(fmt.Sprintf("...等 %d 条更早的记录...\n", len(history)-limit))
	}
	historicalDataSummaryBuilder.WriteString(fmt.Sprintf("历史总览: %d 成功, %d 失败\n", successCount, failureCount))

	// Summarize detectedPatterns
	detectedPatternsSummary := fmt.Sprintf("已识别的模式: %s", strings.Join(detectedPatterns, ", "))
	if len(detectedPatterns) == 0 {
		detectedPatternsSummary = "已识别的模式: 无"
	}

	analysisRequirements := "请基于提供的信息进行深度分析，生成包含核心发现、可执行建议、改进机会、预测下一步和置信度的洞察报告。"

	// Populate historical context if needed, similar to other prompt builders
	historicalCtx := make(map[string]any)
	if fa.config.EnableHistoryAnalysis && len(fa.analysisHistory) > 0 {
		// Example: add overall system stability trend if available from history
		historicalCtx["num_past_analyses"] = len(fa.analysisHistory)
	}

	data := InsightsGenerationPromptData{
		BaseAnalysisSummary:     baseAnalysisSummary,
		HistoricalDataSummary:   historicalDataSummaryBuilder.String(),
		DetectedPatternsSummary: detectedPatternsSummary,
		AnalysisRequirements:    analysisRequirements,
		HistoricalContext:       historicalCtx,
	}
	return data, nil
}

// convertLLMResultToGeneratedInsights 将LLM的JSON输出转换为 types.GeneratedInsights 类型
func (fa *SmartFeedbackAnalyzer) convertLLMResultToGeneratedInsights(llmResultJSON string) (types.GeneratedInsights, error) {
	cleanedJSON := fa.cleanLLMJSONOutput(llmResultJSON)

	var agentInsights AgentInsightsResult
	err := json.Unmarshal([]byte(cleanedJSON), &agentInsights)
	if err != nil {
		return types.GeneratedInsights{}, fmt.Errorf("解析LLM洞察生成JSON失败: %w, json_content: %s", err, cleanedJSON)
	}

	// Map from AgentInsightsResult to types.GeneratedInsights
	// The structures are designed to be similar, but explicit mapping is safer.
	generatedInsights := types.GeneratedInsights{
		KeyTakeaways:       agentInsights.KeyTakeaways,
		PredictedNextSteps: agentInsights.PredictedNextSteps,
		ConfidenceLevel:    agentInsights.ConfidenceLevel,
		Summary:            agentInsights.Summary,
		ActionableRecommendations: make([]struct {
			Recommendation  string `json:"recommendation"`
			Rationale       string `json:"rationale"`
			Priority        string `json:"priority"`
			PotentialImpact string `json:"potential_impact"`
		}, len(agentInsights.ActionableRecommendations)),
		ImprovementOpportunities: make([]struct {
			Opportunity      string `json:"opportunity"`
			PotentialBenefit string `json:"potential_benefit"`
			Difficulty       string `json:"difficulty"`
		}, len(agentInsights.ImprovementOpportunities)),
	}

	for i, ar := range agentInsights.ActionableRecommendations {
		generatedInsights.ActionableRecommendations[i] = struct {
			Recommendation  string `json:"recommendation"`
			Rationale       string `json:"rationale"`
			Priority        string `json:"priority"`
			PotentialImpact string `json:"potential_impact"`
		}{
			Recommendation:  ar.Recommendation,
			Rationale:       ar.Rationale,
			Priority:        ar.Priority,
			PotentialImpact: ar.PotentialImpact,
		}
	}

	for i, io := range agentInsights.ImprovementOpportunities {
		generatedInsights.ImprovementOpportunities[i] = struct {
			Opportunity      string `json:"opportunity"`
			PotentialBenefit string `json:"potential_benefit"`
			Difficulty       string `json:"difficulty"`
		}{
			Opportunity:      io.Opportunity,
			PotentialBenefit: io.PotentialBenefit,
			Difficulty:       io.Difficulty,
		}
	}

	return generatedInsights, nil
}

// 未使用的函数
func (fa *SmartFeedbackAnalyzer) performContentAnalysis(
	ctx context.Context,
	results []types.ExecutionResult,
) map[string]any {
	// 分析执行结果的内容
	observations := []string{}
	outputs := []string{}

	for _, result := range results {
		observations = append(observations, result.Observations...)
		if output, ok := result.Output.(string); ok {
			outputs = append(outputs, output)
		}
	}

	// 简单的内容分析
	keyTerms := fa.extractKeyTerms(observations, outputs)
	sentiment := fa.analyzeSentiment(observations, outputs)

	return map[string]any{
		"observation_count": len(observations),
		"output_count":      len(outputs),
		"key_terms":         keyTerms,
		"sentiment":         sentiment,
	}
}

func (fa *SmartFeedbackAnalyzer) performPatternAnalysis(
	results []types.ExecutionResult,
) map[string]any {
	// 基本模式分析
	patterns := map[string]any{}

	// 状态变化模式
	statusSequence := []types.ExecutionStatus{}
	for _, result := range results {
		statusSequence = append(statusSequence, result.Status)
	}
	patterns["status_sequence"] = statusSequence

	// 时间模式
	durations := []time.Duration{}
	for _, result := range results {
		duration := result.EndTime.Sub(result.StartTime)
		durations = append(durations, duration)
	}
	patterns["duration_trend"] = fa.analyzeDurationTrend(durations)

	return patterns
}

func (fa *SmartFeedbackAnalyzer) performRiskAssessment(
	results []types.ExecutionResult,
	basicAnalysis map[string]any,
) types.RiskAssessment {
	riskFactors := []string{}
	riskScore := 0.0

	// 基于失败率评估风险
	if failureRate, ok := basicAnalysis["failure_rate"].(float64); ok {
		if failureRate > 0.3 {
			riskFactors = append(riskFactors, fmt.Sprintf("高失败率: %.2f", failureRate))
			riskScore += failureRate * 0.4
		}
	}

	// 基于错误类型评估风险
	if errors, ok := basicAnalysis["errors"].([]string); ok {
		if len(errors) > 0 {
			criticalErrors := 0
			for _, err := range errors {
				if fa.isCriticalError(err) {
					criticalErrors++
				}
			}
			if criticalErrors > 0 {
				riskFactors = append(riskFactors, fmt.Sprintf("发现%d个严重错误", criticalErrors))
				riskScore += float64(criticalErrors) * 0.2
			}
		}
	}

	level := fa.determineRiskLevel(riskScore)

	return types.RiskAssessment{
		Level:       level,
		Factors:     riskFactors,
		Mitigation:  fa.generateMitigationStrategies(riskFactors, level),
		Description: fmt.Sprintf("风险级别: %s, 评分: %.2f", level, riskScore),
	}
}

func (fa *SmartFeedbackAnalyzer) generateInsights(
	basicAnalysis map[string]any,
	contentAnalysis map[string]any,
	patternAnalysis map[string]any,
) []string {
	insights := []string{}

	// 基于成功率的洞察
	if successRate, ok := basicAnalysis["success_rate"].(float64); ok {
		if successRate > 0.8 {
			insights = append(insights, fmt.Sprintf("执行成功率较高(%.2f)，系统运行良好", successRate))
		} else if successRate < 0.5 {
			insights = append(insights, fmt.Sprintf("执行成功率偏低(%.2f)，需要优化", successRate))
		}
	}

	// 基于执行时间的洞察
	if avgDuration, ok := basicAnalysis["average_duration"].(time.Duration); ok {
		if avgDuration > 5*time.Minute {
			insights = append(insights, fmt.Sprintf("平均执行时间较长(%v)，可能需要优化", avgDuration))
		}
	}

	// 基于内容分析的洞察
	if sentiment, ok := contentAnalysis["sentiment"].(string); ok {
		if sentiment == "negative" {
			insights = append(insights, "检测到负面情绪，可能存在用户体验问题")
		}
	}

	return insights
}

func (fa *SmartFeedbackAnalyzer) generateRecommendations(
	basicAnalysis map[string]any,
	contentAnalysis map[string]any,
	riskAssessment types.RiskAssessment,
) []string {
	recommendations := []string{}

	// 基于失败率的建议
	if failureRate, ok := basicAnalysis["failure_rate"].(float64); ok {
		if failureRate > 0.2 {
			recommendations = append(recommendations, "增加错误处理和重试机制")
		}
	}

	// 基于风险评估的建议
	switch riskAssessment.Level {
	case "high":
		recommendations = append(recommendations, "暂停自动执行，需要人工干预")
	case "medium":
		recommendations = append(recommendations, "增加监控和日志记录")
	case "low":
		recommendations = append(recommendations, "继续当前执行策略")
	}

	// 基于错误的建议
	if errors, ok := basicAnalysis["errors"].([]string); ok {
		if len(errors) > 0 {
			recommendations = append(recommendations, "分析和修复检测到的错误")
		}
	}

	return recommendations
}

func (fa *SmartFeedbackAnalyzer) generateNextStepSuggestions(
	basicAnalysis map[string]any,
	insights []string,
	recommendations []string,
) []string {
	suggestions := []string{}

	// 基于成功率建议下一步
	if successRate, ok := basicAnalysis["success_rate"].(float64); ok {
		if successRate > 0.8 {
			suggestions = append(suggestions, "可以考虑增加任务复杂度或并行执行")
		} else if successRate < 0.5 {
			suggestions = append(suggestions, "建议简化任务或增加验证步骤")
		}
	}

	// 基于洞察建议下一步
	if len(insights) > 2 {
		suggestions = append(suggestions, "基于分析洞察调整执行策略")
	}

	// 基于推荐行动建议下一步
	if len(recommendations) > 0 {
		suggestions = append(suggestions, "实施推荐的改进措施")
	}

	return suggestions
}

func (fa *SmartFeedbackAnalyzer) calculateOverallConfidence(
	basicAnalysis map[string]any,
	contentAnalysis map[string]any,
	riskAssessment types.RiskAssessment,
) float64 {
	confidence := 0.5 // 基础置信度

	// 基于成功率调整置信度
	if successRate, ok := basicAnalysis["success_rate"].(float64); ok {
		confidence += successRate * 0.3
	}

	// 基于风险级别调整置信度
	switch riskAssessment.Level {
	case "low":
		confidence += 0.2
	case "medium":
		confidence += 0.0
	case "high":
		confidence -= 0.3
	}

	// 基于错误数量调整置信度
	if errors, ok := basicAnalysis["errors"].([]string); ok {
		errorPenalty := float64(len(errors)) * 0.05
		confidence -= errorPenalty
	}

	// 确保置信度在 [0, 1] 范围内
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

func (fa *SmartFeedbackAnalyzer) extractKeyFindings(
	basicAnalysis map[string]any,
	contentAnalysis map[string]any,
) []string {
	findings := []string{}

	// 基础统计发现
	if total, ok := basicAnalysis["total_count"].(int); ok {
		findings = append(findings, fmt.Sprintf("共执行%d个任务", total))
	}

	if successRate, ok := basicAnalysis["success_rate"].(float64); ok {
		findings = append(findings, fmt.Sprintf("成功率: %.2f", successRate))
	}

	if avgDuration, ok := basicAnalysis["average_duration"].(time.Duration); ok {
		findings = append(findings, fmt.Sprintf("平均执行时间: %v", avgDuration))
	}

	// 错误发现
	if errors, ok := basicAnalysis["errors"].([]string); ok && len(errors) > 0 {
		findings = append(findings, fmt.Sprintf("发现%d个错误", len(errors)))
	}

	return findings
}

func (fa *SmartFeedbackAnalyzer) determineUserInputRequired(
	basicAnalysis map[string]any,
	riskAssessment types.RiskAssessment,
) bool {
	// 高风险情况需要用户输入
	if riskAssessment.Level == "high" {
		return true
	}

	// 低成功率需要用户确认
	if successRate, ok := basicAnalysis["success_rate"].(float64); ok {
		if successRate < 0.5 {
			return true
		}
	}

	// 大量错误需要用户干预
	if errors, ok := basicAnalysis["errors"].([]string); ok {
		if len(errors) > 3 {
			return true
		}
	}

	return false
}

func (fa *SmartFeedbackAnalyzer) recordAnalysis(
	analysis types.AgentOutputAnalysis,
	resultCount int,
) {
	record := FeedbackAnalysisRecord{
		Timestamp: time.Now(),
		Results:   []types.ExecutionResult{},
		Analysis:  analysis,
		AnalysisContext: map[string]any{
			"config": fa.config,
		},
	}

	fa.analysisHistory = append(fa.analysisHistory, record)

	// 保持历史记录限制
	if len(fa.analysisHistory) > fa.config.HistoryRetention {
		fa.analysisHistory = fa.analysisHistory[1:]
	}
}

func (fa *SmartFeedbackAnalyzer) detectTaskTypePattern(
	history []types.ExecutionResult,
) string {
	// 这里需要更多的任务类型信息，暂时简化实现
	return ""
}

func (fa *SmartFeedbackAnalyzer) identifyRiskFactors(
	actions []types.Task,
	systemState types.SystemState,
) []string {
	factors := []string{}

	// 基于系统状态识别风险
	if systemState.ErrorCount > 5 {
		factors = append(factors, "系统错误计数过高")
	}

	// 基于任务数量识别风险
	if len(actions) > 10 {
		factors = append(factors, "待执行任务数量过多")
	}

	// 基于任务优先级识别风险
	highPriorityCount := 0
	for _, action := range actions {
		if action.Priority > 8 {
			highPriorityCount++
		}
	}
	if highPriorityCount > 3 {
		factors = append(factors, "高优先级任务过多")
	}

	return factors
}

func (fa *SmartFeedbackAnalyzer) calculateRiskScore(
	factors []string,
	systemState types.SystemState,
) float64 {
	score := 0.0

	// 基于风险因子数量
	score += float64(len(factors)) * 0.2

	// 基于系统错误数量
	score += float64(systemState.ErrorCount) * 0.1

	// 基于系统状态
	if !systemState.IsActive {
		score += 0.3
	}

	return score
}

func (fa *SmartFeedbackAnalyzer) generateMitigationStrategies(
	factors []string,
	level string,
) []string {
	strategies := []string{}

	switch level {
	case "high":
		strategies = append(strategies, "立即暂停自动执行")
		strategies = append(strategies, "进行系统健康检查")
		strategies = append(strategies, "通知管理员介入")
	case "medium":
		strategies = append(strategies, "增加监控频率")
		strategies = append(strategies, "启用详细日志记录")
		strategies = append(strategies, "准备回滚计划")
	case "low":
		strategies = append(strategies, "继续正常监控")
	}

	return strategies
}

func (fa *SmartFeedbackAnalyzer) generateRiskDescription(
	level string,
	score float64,
	factors []string,
) string {
	description := fmt.Sprintf("风险级别: %s, 评分: %.2f", level, score)

	if len(factors) > 0 {
		description += fmt.Sprintf(", 主要风险因子: %s", strings.Join(factors, ", "))
	}

	return description
}

func (fa *SmartFeedbackAnalyzer) generateHistoricalInsights() []string {
	insights := []string{}

	// 基于历史分析生成洞察
	if len(fa.analysisHistory) > 20 {
		recent := fa.analysisHistory[len(fa.analysisHistory)-20:]

		avgConfidence := 0.0
		highRiskCount := 0

		for _, record := range recent {
			avgConfidence += record.Analysis.ConfidenceLevel
			if record.Analysis.RiskAssessment.Level == "high" {
				highRiskCount++
			}
		}

		avgConfidence /= float64(len(recent))

		if avgConfidence > 0.8 {
			insights = append(insights, "历史分析显示系统稳定性较高")
		}

		if highRiskCount > 5 {
			insights = append(insights, "近期高风险事件较多，需要关注")
		}
	}

	return insights
}
