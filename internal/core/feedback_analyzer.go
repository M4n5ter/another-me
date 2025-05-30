package core

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/m4n5ter/another-me/internal/core/types"
)

// SmartFeedbackAnalyzer 智能反馈分析器实现 - 基于Agent的智能分析
type SmartFeedbackAnalyzer struct {
	agent  Agent // 底层智能Agent
	logger *slog.Logger
	config FeedbackAnalyzerConfig

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

// DefaultFeedbackAnalyzerConfig 返回默认配置
func DefaultFeedbackAnalyzerConfig() FeedbackAnalyzerConfig {
	return FeedbackAnalyzerConfig{
		AgentPromptTemplate: `你是一个智能反馈分析专家，负责深度分析Agent执行结果并提供专业洞察。

请分析以下执行结果：

**执行结果摘要**:
- 总结果数: {{.TotalResults}}
- 成功数: {{.SuccessCount}}
- 失败数: {{.FailureCount}}
- 成功率: {{.SuccessRate}}%

**详细结果**:
{{.DetailedResults}}

**分析要求**:
{{.AnalysisRequirements}}

请按以下JSON格式返回分析结果：
{
  "key_findings": ["发现1", "发现2"],
  "actionable_insights": ["洞察1", "洞察2"],
  "requires_user_input": true/false,
  "confidence_level": 0.0-1.0,
  "recommended_actions": ["行动1", "行动2"],
  "risk_assessment": {
    "level": "low/medium/high",
    "factors": ["因子1", "因子2"],
    "mitigation": ["缓解措施1", "缓解措施2"],
    "description": "风险描述"
  },
  "next_step_suggestions": ["建议1", "建议2"],
  "patterns_detected": ["模式1", "模式2"],
  "improvement_opportunities": ["改进点1", "改进点2"]
}

**分析原则**:
1. 深度分析执行模式和趋势
2. 识别潜在问题和改进机会
3. 提供具体可执行的建议
4. 评估风险并给出缓解方案
5. 基于历史数据预测未来趋势
6. 给出详细的推理过程和置信度评估`,
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
	agent Agent,
	config FeedbackAnalyzerConfig,
	logger *slog.Logger,
) *SmartFeedbackAnalyzer {
	if logger == nil {
		logger = slog.Default().WithGroup("smart_feedback_analyzer")
	}

	return &SmartFeedbackAnalyzer{
		agent:           agent,
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
	// if len(results) == 0 {
	// 	return types.AgentOutputAnalysis{
	// 		KeyFindings:         []string{"没有执行结果"},
	// 		ActionableInsights:  []string{},
	// 		RequiresUserInput:   false,
	// 		ConfidenceLevel:     0.0,
	// 		RecommendedActions:  []string{},
	// 		RiskAssessment:      types.RiskAssessment{Level: "low"},
	// 		NextStepSuggestions: []string{},
	// 	}, nil
	// }

	// fa.logger.Info("开始AI驱动的执行结果分析", "result_count", len(results))

	// // 设置分析超时
	// analysisCtx, cancel := context.WithTimeout(ctx, fa.config.AnalysisTimeout)
	// defer cancel()

	// // 1. 构建分析提示数据
	// promptData := fa.buildAnalysisPromptData(results)

	// // 2. 调用Agent进行智能分析
	// agentResult, err := fa.invokeAgentForAnalysis(analysisCtx, promptData)
	// if err != nil {
	// 	fa.logger.Error("AI分析失败，使用后备分析", "error", err)
	// 	return fa.createFallbackAnalysis(results), nil
	// }

	// // 3. 转换Agent结果为标准分析结果
	// analysis := fa.convertAgentResultToAnalysis(agentResult)

	// // 4. 更新分析历史
	// fa.updateAnalysisHistory(results, analysis)

	// fa.logger.Info("AI执行结果分析完成",
	// 	"confidence", analysis.ConfidenceLevel,
	// 	"findings_count", len(analysis.KeyFindings),
	// 	"insights_count", len(analysis.ActionableInsights))

	return types.AgentOutputAnalysis{}, nil
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

// AssessRisk 评估风险 - 实现FeedbackAnalyzer接口
func (fa *SmartFeedbackAnalyzer) AssessRisk(
	ctx context.Context,
	proposedActions []types.Task,
	systemState types.SystemState,
) (types.RiskAssessment, error) {
	fa.logger.Debug("评估风险",
		"action_count", len(proposedActions),
		"error_count", systemState.ErrorCount)

	if !fa.config.EnableHistoryAnalysis {
		return types.RiskAssessment{
			Level:       "unknown",
			Factors:     []string{"风险评估未启用"},
			Mitigation:  []string{},
			Description: "风险评估功能未启用",
		}, nil
	}

	// 1. 收集风险因子
	riskFactors := fa.identifyRiskFactors(proposedActions, systemState)

	// 2. 计算风险分数
	riskScore := fa.calculateRiskScore(riskFactors, systemState)

	// 3. 确定风险级别
	riskLevel := fa.determineRiskLevel(riskScore)

	// 4. 生成缓解措施
	mitigation := fa.generateMitigationStrategies(riskFactors, riskLevel)

	// 5. 生成风险描述
	description := fa.generateRiskDescription(riskLevel, riskScore, riskFactors)

	assessment := types.RiskAssessment{
		Level:       riskLevel,
		Factors:     riskFactors,
		Mitigation:  mitigation,
		Description: description,
	}

	fa.logger.Info("风险评估完成",
		"risk_level", riskLevel,
		"risk_score", riskScore,
		"factor_count", len(riskFactors))

	return assessment, nil
}

// GenerateInsights 生成洞察 - 实现FeedbackAnalyzer接口
func (fa *SmartFeedbackAnalyzer) GenerateInsights(
	ctx context.Context,
	analysis types.AgentOutputAnalysis,
) ([]string, error) {
	fa.logger.Debug("生成洞察", "confidence", analysis.ConfidenceLevel)

	insights := []string{}

	// 1. 基于置信度的洞察
	if analysis.ConfidenceLevel < fa.config.MinConfidenceThreshold {
		insights = append(insights,
			fmt.Sprintf("执行置信度较低(%.2f)，建议增加验证步骤", analysis.ConfidenceLevel))
	}

	// 2. 基于风险评估的洞察
	switch analysis.RiskAssessment.Level {
	case "high":
		insights = append(insights, "检测到高风险，建议谨慎执行或寻求人工确认")
	case "medium":
		insights = append(insights, "存在中等风险，建议增加监控和错误处理")
	}

	// 3. 基于关键发现的洞察
	if len(analysis.KeyFindings) > 3 {
		insights = append(insights, "执行过程中发现多个问题，建议优化任务流程")
	}

	// 4. 基于历史模式的洞察
	if len(fa.analysisHistory) > 10 {
		historicalInsights := fa.generateHistoricalInsights()
		insights = append(insights, historicalInsights...)
	}

	// 5. 基于推荐行动的洞察
	if len(analysis.RecommendedActions) == 0 {
		insights = append(insights, "未生成推荐行动，可能需要用户指导")
	}

	fa.logger.Info("洞察生成完成", "insight_count", len(insights))
	return insights, nil
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

// 辅助方法实现

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

func (fa *SmartFeedbackAnalyzer) detectTaskTypePattern(
	history []types.ExecutionResult,
) string {
	// 这里需要更多的任务类型信息，暂时简化实现
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

func (fa *SmartFeedbackAnalyzer) determineRiskLevel(score float64) string {
	if score >= 0.8 {
		return "high"
	} else if score >= 0.5 {
		return "medium"
	}
	return "low"
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
