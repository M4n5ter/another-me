package core

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/internal/core/types"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// SmartContinuousDecisionEngine 智能持续决策引擎实现 - 基于Agent的智能持续决策
type SmartContinuousDecisionEngine struct {
	agent            Agent // 底层智能Agent
	feedbackAnalyzer FeedbackAnalyzer
	logger           *slog.Logger
	config           ContinuousDecisionConfig
	decisionHistory  []types.ContinuousDecisionResult
}

// ContinuousDecisionConfig 持续决策引擎配置
type ContinuousDecisionConfig struct {
	// Agent配置
	AgentPromptTemplate string        `json:"agent_prompt_template"` // Agent提示词模板
	AnalysisTimeout     time.Duration `json:"analysis_timeout"`      // 分析超时时间
	DecisionTimeout     time.Duration `json:"decision_timeout"`      // 决策超时时间

	// 决策阈值配置
	ContinueConfidenceThreshold float64 `json:"continue_confidence_threshold"` // 继续执行的置信度阈值
	StopConfidenceThreshold     float64 `json:"stop_confidence_threshold"`     // 停止执行的置信度阈值
	MaxIterations               int     `json:"max_iterations"`                // 最大迭代次数

	// 分析权重配置
	OutputAnalysisWeight    float64 `json:"output_analysis_weight"`    // 输出分析权重
	SystemMetricsWeight     float64 `json:"system_metrics_weight"`     // 系统指标权重
	HistoricalPatternWeight float64 `json:"historical_pattern_weight"` // 历史模式权重

	// 风险评估配置
	RiskToleranceLevel string  `json:"risk_tolerance_level"` // 风险容忍级别: low, medium, high
	MaxRiskScore       float64 `json:"max_risk_score"`       // 最大风险分数

	// 性能配置
	HistoryRetention int `json:"history_retention"` // 历史记录保留数量
}

// DefaultContinuousDecisionConfig 返回默认配置
func DefaultContinuousDecisionConfig() ContinuousDecisionConfig {
	return ContinuousDecisionConfig{
		AgentPromptTemplate: `你是一个智能持续决策分析助手，专门负责分析Agent执行结果并决定是否继续执行。

请基于以下信息进行持续决策分析：

**执行状态**:
- 计划ID: {{.PlanID}}
- 当前步骤: {{.CurrentStep}}/{{.TotalSteps}}
- 迭代次数: {{.IterationCount}}
- 执行状态: {{.ExecutionStatus}}
- 完成任务数: {{.CompletedTasks}}
- 失败任务数: {{.FailedTasks}}

**最新执行结果**: {{.LatestResults}}

**Agent输出分析**: {{.OutputAnalysis}}

**系统状态**: {{.SystemContext}}

**历史模式**: {{.HistoricalPattern}}

请按照以下JSON格式输出决策结果：
{
  "should_continue": true/false,
  "next_execution_plan": {...},
  "should_enter_wait_mode": true/false,
  "monitoring_tasks": [...],
  "reasoning_steps": ["推理步骤1", "推理步骤2"],
  "confidence": 0.0-1.0,
  "estimated_duration_minutes": 15,
  "priority": 1-10,
  "requires_user_approval": true/false
}

**决策原则**:
1. 分析执行结果的成功率和质量
2. 评估继续执行的价值和风险
3. 考虑系统性能和资源使用情况
4. 基于历史模式预测后续发展
5. 在不确定时倾向于谨慎决策
6. 给出详细的推理过程和置信度评估`,
		AnalysisTimeout:             30 * time.Second,
		DecisionTimeout:             15 * time.Second,
		ContinueConfidenceThreshold: 0.7,
		StopConfidenceThreshold:     0.3,
		MaxIterations:               10,
		OutputAnalysisWeight:        0.4,
		SystemMetricsWeight:         0.3,
		HistoricalPatternWeight:     0.3,
		RiskToleranceLevel:          "medium",
		MaxRiskScore:                0.7,
		HistoryRetention:            100,
	}
}

// NewSmartContinuousDecisionEngine 创建新的智能持续决策引擎
func NewSmartContinuousDecisionEngine(
	agent Agent,
	feedbackAnalyzer FeedbackAnalyzer,
	config ContinuousDecisionConfig,
	logger *slog.Logger,
) *SmartContinuousDecisionEngine {
	if logger == nil {
		logger = slog.Default().WithGroup("smart_continuous_decision_engine")
	}

	return &SmartContinuousDecisionEngine{
		agent:            agent,
		feedbackAnalyzer: feedbackAnalyzer,
		logger:           logger,
		config:           config,
		decisionHistory:  make([]types.ContinuousDecisionResult, 0, config.HistoryRetention),
	}
}

var _ ContinuousDecisionEngine = (*SmartContinuousDecisionEngine)(nil)

// ContinuousDecisionPromptData 持续决策提示词数据
type ContinuousDecisionPromptData struct {
	PlanID            string                    `json:"plan_id"`
	CurrentStep       int                       `json:"current_step"`
	TotalSteps        int                       `json:"total_steps"`
	IterationCount    int                       `json:"iteration_count"`
	ExecutionStatus   string                    `json:"execution_status"`
	CompletedTasks    int                       `json:"completed_tasks"`
	FailedTasks       int                       `json:"failed_tasks"`
	LatestResults     []types.ExecutionResult   `json:"latest_results"`
	OutputAnalysis    types.AgentOutputAnalysis `json:"output_analysis"`
	SystemContext     map[string]any            `json:"system_context"`
	HistoricalPattern map[string]any            `json:"historical_pattern"`
}

// AgentContinuousDecisionResult Agent返回的持续决策结果
type AgentContinuousDecisionResult struct {
	ShouldContinue           bool                   `json:"should_continue"`
	NextExecutionPlan        *types.ExecutionPlan   `json:"next_execution_plan,omitempty"`
	ShouldEnterWaitMode      bool                   `json:"should_enter_wait_mode"`
	MonitoringTasks          []types.MonitoringTask `json:"monitoring_tasks"`
	ReasoningSteps           []string               `json:"reasoning_steps"`
	Confidence               float64                `json:"confidence"`
	EstimatedDurationMinutes int                    `json:"estimated_duration_minutes"`
	Priority                 int                    `json:"priority"`
	RequiresUserApproval     bool                   `json:"requires_user_approval"`
}

// MakeContinuousDecision 进行持续决策 - 实现ContinuousDecisionEngine接口
func (cde *SmartContinuousDecisionEngine) MakeContinuousDecision(
	ctx context.Context,
	decisionContext types.ContinuousDecisionContext,
) (types.ContinuousDecisionResult, error) {
	cde.logger.Info("开始AI驱动的持续决策分析",
		"plan_id", decisionContext.ExecutionState.PlanID,
		"iteration_count", decisionContext.ExecutionState.IterationCount)

	// 设置分析超时
	analysisCtx, cancel := context.WithTimeout(ctx, cde.config.AnalysisTimeout)
	defer cancel()

	// 1. 分析Agent输出
	outputAnalysis, err := cde.analyzeCurrentOutput(analysisCtx, decisionContext)
	if err != nil {
		cde.logger.Error("分析Agent输出失败", "error", err)
		return cde.createEmergencyStopDecision("输出分析失败", err), nil
	}

	// 2. 构建决策提示数据
	promptData := cde.buildDecisionPromptData(decisionContext, outputAnalysis)

	// 3. 调用Agent进行智能持续决策
	agentResult, err := cde.invokeAgentForContinuousDecision(analysisCtx, promptData)
	if err != nil {
		cde.logger.Error("AI持续决策失败", "error", err)
		return cde.createEmergencyStopDecision("AI决策失败", err), nil
	}

	// 4. 转换Agent结果为标准决策结果
	result := cde.convertAgentResultToContinuousDecisionResult(agentResult)

	// 5. 更新决策历史
	if err := cde.UpdateDecisionHistory(ctx, result); err != nil {
		cde.logger.Warn("更新决策历史失败", "error", err)
	}

	cde.logger.Info("AI持续决策完成",
		"should_continue", result.ShouldContinue,
		"confidence", result.Confidence,
		"priority", result.Priority)

	return result, nil
}

// AnalyzeAgentOutput 分析Agent输出 - 实现ContinuousDecisionEngine接口
func (cde *SmartContinuousDecisionEngine) AnalyzeAgentOutput(
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

	cde.logger.Debug("AI分析Agent输出", "result_count", len(results))

	// 使用FeedbackAnalyzer进行详细分析，如果可用
	if cde.feedbackAnalyzer != nil {
		return cde.feedbackAnalyzer.AnalyzeExecutionResults(ctx, results)
	}

	// 回退到AI Agent分析
	return cde.performAIOutputAnalysis(ctx, results)
}

// EvaluateContinuationStrategy 评估持续策略 - 实现ContinuousDecisionEngine接口
func (cde *SmartContinuousDecisionEngine) EvaluateContinuationStrategy(
	ctx context.Context,
	executionState types.ExecutionState,
	outputAnalysis types.AgentOutputAnalysis,
) (types.ContinuousDecisionResult, error) {
	cde.logger.Debug("AI评估持续策略",
		"current_step", executionState.CurrentStepIndex,
		"confidence_level", outputAnalysis.ConfidenceLevel)

	// 构建策略评估提示
	promptData := ContinuousDecisionPromptData{
		PlanID:          executionState.PlanID,
		CurrentStep:     executionState.CurrentStepIndex,
		TotalSteps:      executionState.TotalTaskCount, // 使用总任务数
		IterationCount:  executionState.IterationCount,
		ExecutionStatus: string(executionState.Status),
		CompletedTasks:  executionState.CompletedTaskCount,
		FailedTasks:     executionState.FailedTaskCount,
		OutputAnalysis:  outputAnalysis,
		SystemContext: map[string]any{
			"evaluation_type": "continuation_strategy",
		},
		HistoricalPattern: cde.getHistoricalPattern(),
	}

	// 调用Agent评估策略
	agentResult, err := cde.invokeAgentForContinuousDecision(ctx, promptData)
	if err != nil {
		return cde.createEmergencyStopDecision("策略评估失败", err), nil
	}

	return cde.convertAgentResultToContinuousDecisionResult(agentResult), nil
}

// GenerateNextActions 生成下一步行动 - 实现ContinuousDecisionEngine接口
func (cde *SmartContinuousDecisionEngine) GenerateNextActions(
	ctx context.Context,
	analysis types.AgentOutputAnalysis,
	systemContext map[string]any,
) ([]types.Task, error) {
	cde.logger.Debug("AI生成下一步行动", "insights_count", len(analysis.ActionableInsights))

	// 构建行动生成提示
	prompt := cde.buildNextActionsPrompt(analysis, systemContext)

	// 创建行动生成任务
	actionTask := types.Task{
		ID:          fmt.Sprintf("next_actions_%d", time.Now().UnixNano()),
		Type:        "action_generation",
		Description: prompt,
		Priority:    7,
		AgentType:   types.AgentTypeReAct,
		Parameters: map[string]any{
			"analysis":       analysis,
			"system_context": systemContext,
		},
		Context: map[string]any{
			"generation_type": "next_actions",
		},
		CreatedAt: time.Now(),
	}

	// 调用Agent生成行动
	result, err := cde.agent.Execute(ctx, actionTask, map[string]any{
		"analysis_type": "action_generation",
		"expects_json":  true,
	})
	if err != nil {
		cde.logger.Error("AI生成下一步行动失败", "error", err)
		return cde.createFallbackActions(analysis), nil
	}

	// 解析生成的行动
	nextTasks := cde.parseNextActionsResult(result.Output, analysis, systemContext)

	cde.logger.Info("AI下一步行动生成完成", "task_count", len(nextTasks))
	return nextTasks, nil
}

// UpdateDecisionHistory 更新决策历史 - 实现ContinuousDecisionEngine接口
func (cde *SmartContinuousDecisionEngine) UpdateDecisionHistory(
	ctx context.Context,
	decisionResult types.ContinuousDecisionResult,
) error {
	// 添加到内存历史
	cde.decisionHistory = append(cde.decisionHistory, decisionResult)

	// 保持历史记录数量限制
	if len(cde.decisionHistory) > cde.config.HistoryRetention {
		cde.decisionHistory = cde.decisionHistory[1:]
	}

	cde.logger.Debug("决策历史已更新",
		"history_count", len(cde.decisionHistory),
		"should_continue", decisionResult.ShouldContinue)

	return nil
}

// GetDecisionInsights 获取决策洞察 - 实现ContinuousDecisionEngine接口
func (cde *SmartContinuousDecisionEngine) GetDecisionInsights(
	ctx context.Context,
	timeRange time.Duration,
) (map[string]any, error) {
	cde.logger.Debug("AI获取决策洞察", "time_range", timeRange)

	// 使用AI分析历史决策模式
	prompt := cde.buildInsightsPrompt(timeRange)

	// 创建洞察分析任务
	insightTask := types.Task{
		ID:          fmt.Sprintf("decision_insights_%d", time.Now().UnixNano()),
		Type:        "insight_analysis",
		Description: prompt,
		Priority:    5,
		AgentType:   types.AgentTypeReAct,
		Parameters: map[string]any{
			"time_range":       timeRange,
			"decision_history": cde.decisionHistory,
		},
		Context: map[string]any{
			"analysis_type": "decision_insights",
		},
		CreatedAt: time.Now(),
	}

	// 调用Agent进行洞察分析
	result, err := cde.agent.Execute(ctx, insightTask, map[string]any{
		"analysis_type": "insights",
		"expects_json":  true,
	})
	if err != nil {
		cde.logger.Error("AI洞察分析失败", "error", err)
		return cde.createFallbackInsights(timeRange), nil
	}

	// 解析洞察结果
	insights := cde.parseInsightsResult(result.Output, timeRange)

	cde.logger.Info("AI决策洞察获取完成", "insights_keys", len(insights))
	return insights, nil
}

// ConfigureStrategy 配置决策策略 - 实现ContinuousDecisionEngine接口
func (cde *SmartContinuousDecisionEngine) ConfigureStrategy(
	ctx context.Context,
	strategy types.ContinuationStrategy,
) error {
	cde.logger.Info("配置AI决策策略",
		"max_iterations", strategy.MaxIterations,
		"idle_threshold", strategy.IdleThreshold)

	// 更新配置
	cde.config.MaxIterations = strategy.MaxIterations

	// 根据策略调整阈值
	if len(strategy.StopConditions) > 0 {
		cde.config.StopConfidenceThreshold = 0.4 // 更敏感的停止条件
	}

	if len(strategy.ContinueConditions) > 0 {
		cde.config.ContinueConfidenceThreshold = 0.6 // 更宽松的继续条件
	}

	cde.logger.Info("AI决策策略配置完成")
	return nil
}

// 私有方法实现

func (cde *SmartContinuousDecisionEngine) analyzeCurrentOutput(
	ctx context.Context,
	decisionContext types.ContinuousDecisionContext,
) (types.AgentOutputAnalysis, error) {
	// 提取最新的执行结果
	var latestResults []types.ExecutionResult

	if len(decisionContext.StepResults) > 0 {
		latestStep := decisionContext.StepResults[len(decisionContext.StepResults)-1]
		latestResults = latestStep.TaskResults
	}

	return cde.AnalyzeAgentOutput(ctx, latestResults)
}

func (cde *SmartContinuousDecisionEngine) buildDecisionPromptData(
	decisionContext types.ContinuousDecisionContext,
	outputAnalysis types.AgentOutputAnalysis,
) ContinuousDecisionPromptData {
	var latestResults []types.ExecutionResult
	if len(decisionContext.StepResults) > 0 {
		latestStep := decisionContext.StepResults[len(decisionContext.StepResults)-1]
		latestResults = latestStep.TaskResults
	}

	// 构建系统上下文
	systemContext := map[string]any{
		"system_metrics":  decisionContext.SystemMetrics,
		"execution_state": decisionContext.ExecutionState,
		"timestamp":       decisionContext.Timestamp,
	}

	return ContinuousDecisionPromptData{
		PlanID:            decisionContext.ExecutionState.PlanID,
		CurrentStep:       decisionContext.ExecutionState.CurrentStepIndex,
		TotalSteps:        decisionContext.ExecutionState.TotalTaskCount, // 使用总任务数作为近似
		IterationCount:    decisionContext.ExecutionState.IterationCount,
		ExecutionStatus:   string(decisionContext.ExecutionState.Status),
		CompletedTasks:    decisionContext.ExecutionState.CompletedTaskCount,
		FailedTasks:       decisionContext.ExecutionState.FailedTaskCount,
		LatestResults:     latestResults,
		OutputAnalysis:    outputAnalysis,
		SystemContext:     systemContext,
		HistoricalPattern: cde.getHistoricalPattern(),
	}
}

func (cde *SmartContinuousDecisionEngine) invokeAgentForContinuousDecision(
	ctx context.Context,
	promptData ContinuousDecisionPromptData,
) (*AgentContinuousDecisionResult, error) {
	// 构建决策提示
	prompt := cde.buildContinuousDecisionPrompt(promptData)

	// 创建持续决策任务
	decisionTask := types.Task{
		ID:          fmt.Sprintf("continuous_decision_%d", time.Now().UnixNano()),
		Type:        "continuous_decision",
		Description: prompt,
		Priority:    8, // 高优先级
		AgentType:   types.AgentTypeReAct,
		Parameters: map[string]any{
			"decision_context": promptData,
		},
		Context: map[string]any{
			"analysis_type": "continuous_decision",
		},
		CreatedAt: time.Now(),
	}

	// 调用Agent
	result, err := cde.agent.Execute(ctx, decisionTask, map[string]any{
		"analysis_type": "continuous_decision",
		"expects_json":  true,
	})
	if err != nil {
		return nil, fmt.Errorf("Agent执行失败: %w", err)
	}

	// 解析Agent输出的JSON结果
	var agentResult AgentContinuousDecisionResult
	if err := json.Unmarshal([]byte(result.Output.(string)), &agentResult); err != nil {
		// 如果JSON解析失败，尝试从文本中提取关键信息
		cde.logger.Warn("JSON解析失败，尝试文本解析", "error", err, "output", result.Output)
		return cde.parseDecisionFromText(result.Output.(string)), nil
	}

	return &agentResult, nil
}

func (cde *SmartContinuousDecisionEngine) buildContinuousDecisionPrompt(data ContinuousDecisionPromptData) string {
	template := cde.config.AgentPromptTemplate

	// 模板替换
	template = strings.ReplaceAll(template, "{{.PlanID}}", data.PlanID)
	template = strings.ReplaceAll(template, "{{.CurrentStep}}", fmt.Sprintf("%d", data.CurrentStep))
	template = strings.ReplaceAll(template, "{{.TotalSteps}}", fmt.Sprintf("%d", data.TotalSteps))
	template = strings.ReplaceAll(template, "{{.IterationCount}}", fmt.Sprintf("%d", data.IterationCount))
	template = strings.ReplaceAll(template, "{{.ExecutionStatus}}", data.ExecutionStatus)
	template = strings.ReplaceAll(template, "{{.CompletedTasks}}", fmt.Sprintf("%d", data.CompletedTasks))
	template = strings.ReplaceAll(template, "{{.FailedTasks}}", fmt.Sprintf("%d", data.FailedTasks))
	template = strings.ReplaceAll(template, "{{.LatestResults}}", cde.formatExecutionResults(data.LatestResults))
	template = strings.ReplaceAll(template, "{{.OutputAnalysis}}", cde.formatOutputAnalysis(data.OutputAnalysis))
	template = strings.ReplaceAll(template, "{{.SystemContext}}", cde.formatContext(data.SystemContext))
	template = strings.ReplaceAll(template, "{{.HistoricalPattern}}", cde.formatHistoricalPattern(data.HistoricalPattern))

	return template
}

// 其他私有辅助方法实现

func (cde *SmartContinuousDecisionEngine) createEmergencyStopDecision(
	reason string,
	err error,
) types.ContinuousDecisionResult {
	return types.ContinuousDecisionResult{
		ShouldContinue:       false,
		NextExecutionPlan:    None[types.ExecutionPlan](),
		ShouldEnterWaitMode:  true,
		MonitoringTasks:      []types.MonitoringTask{},
		ReasoningSteps:       []string{fmt.Sprintf("紧急停止: %s - %v", reason, err)},
		Confidence:           1.0,
		EstimatedDuration:    None[time.Duration](),
		Priority:             10,
		RequiresUserApproval: false,
	}
}

func (cde *SmartContinuousDecisionEngine) convertAgentResultToContinuousDecisionResult(
	agentResult *AgentContinuousDecisionResult,
) types.ContinuousDecisionResult {
	result := types.ContinuousDecisionResult{
		ShouldContinue:       agentResult.ShouldContinue,
		ShouldEnterWaitMode:  agentResult.ShouldEnterWaitMode,
		MonitoringTasks:      agentResult.MonitoringTasks,
		ReasoningSteps:       agentResult.ReasoningSteps,
		Confidence:           agentResult.Confidence,
		Priority:             agentResult.Priority,
		RequiresUserApproval: agentResult.RequiresUserApproval,
	}

	// 设置预期执行时间
	if agentResult.EstimatedDurationMinutes > 0 {
		result.EstimatedDuration = Some(time.Duration(agentResult.EstimatedDurationMinutes) * time.Minute)
	} else {
		result.EstimatedDuration = None[time.Duration]()
	}

	// 设置执行计划
	if agentResult.NextExecutionPlan != nil {
		result.NextExecutionPlan = Some(*agentResult.NextExecutionPlan)
	} else {
		result.NextExecutionPlan = None[types.ExecutionPlan]()
	}

	return result
}

func (cde *SmartContinuousDecisionEngine) performAIOutputAnalysis(
	ctx context.Context,
	results []types.ExecutionResult,
) (types.AgentOutputAnalysis, error) {
	// 构建输出分析提示
	prompt := fmt.Sprintf(`请分析以下执行结果并提供详细的输出分析：

执行结果数量: %d

执行详情:
%s

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
  "next_step_suggestions": ["建议1", "建议2"]
}`, len(results), cde.formatExecutionResults(results))

	// 创建分析任务
	analysisTask := types.Task{
		ID:          fmt.Sprintf("output_analysis_%d", time.Now().UnixNano()),
		Type:        "output_analysis",
		Description: prompt,
		Priority:    6,
		AgentType:   types.AgentTypeReAct,
		Parameters: map[string]any{
			"execution_results": results,
		},
		Context: map[string]any{
			"analysis_type": "output_analysis",
		},
		CreatedAt: time.Now(),
	}

	// 调用Agent进行分析
	result, err := cde.agent.Execute(ctx, analysisTask, map[string]any{
		"analysis_type": "output_analysis",
		"expects_json":  true,
	})
	if err != nil {
		return cde.createFallbackOutputAnalysis(results), nil
	}

	// 解析分析结果
	return cde.parseOutputAnalysisResult(result.Output, results)
}

func (cde *SmartContinuousDecisionEngine) getHistoricalPattern() map[string]any {
	if len(cde.decisionHistory) == 0 {
		return map[string]any{"pattern": "no_history"}
	}

	recentDecisions := cde.decisionHistory
	if len(recentDecisions) > 10 {
		recentDecisions = recentDecisions[len(recentDecisions)-10:]
	}

	continueCount := 0
	totalConfidence := 0.0

	for _, decision := range recentDecisions {
		if decision.ShouldContinue {
			continueCount++
		}
		totalConfidence += decision.Confidence
	}

	continueRate := float64(continueCount) / float64(len(recentDecisions))
	avgConfidence := totalConfidence / float64(len(recentDecisions))

	return map[string]any{
		"continue_rate":      continueRate,
		"average_confidence": avgConfidence,
		"decision_count":     len(recentDecisions),
		"recent_trend":       cde.analyzeTrend(recentDecisions),
	}
}

func (cde *SmartContinuousDecisionEngine) buildNextActionsPrompt(
	analysis types.AgentOutputAnalysis,
	systemContext map[string]any,
) string {
	return fmt.Sprintf(`基于以下分析结果，生成下一步行动计划：

**输出分析**:
关键发现: %v
可执行洞察: %v
推荐行动: %v
风险级别: %s

**系统上下文**: %s

请生成JSON格式的任务列表：
{
  "tasks": [
    {
      "id": "任务ID",
      "type": "任务类型",
      "description": "任务描述",
      "priority": 1-10,
      "agent_type": "gui/react",
      "parameters": {},
      "context": {}
    }
  ]
}`,
		analysis.KeyFindings,
		analysis.ActionableInsights,
		analysis.RecommendedActions,
		analysis.RiskAssessment.Level,
		cde.formatContext(systemContext))
}

func (cde *SmartContinuousDecisionEngine) parseNextActionsResult(
	output any,
	analysis types.AgentOutputAnalysis,
	systemContext map[string]any,
) []types.Task {
	// 尝试解析JSON结果
	var taskList struct {
		Tasks []types.Task `json:"tasks"`
	}

	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", output)), &taskList); err == nil {
		return taskList.Tasks
	}

	// 解析失败，返回后备任务
	return cde.createFallbackActions(analysis)
}

func (cde *SmartContinuousDecisionEngine) createFallbackActions(analysis types.AgentOutputAnalysis) []types.Task {
	var tasks []types.Task

	// 基于分析结果创建后备任务
	if len(analysis.ActionableInsights) > 0 {
		for i, insight := range analysis.ActionableInsights {
			task := types.Task{
				ID:          fmt.Sprintf("fallback_insight_%d", i),
				Type:        "analysis_task",
				Description: insight,
				Priority:    5,
				AgentType:   types.AgentTypeReAct,
				Parameters: map[string]any{
					"source": "fallback_insight",
				},
				Context: map[string]any{
					"generated_by": "fallback",
				},
				CreatedAt: time.Now(),
			}
			tasks = append(tasks, task)
		}
	}

	return tasks
}

func (cde *SmartContinuousDecisionEngine) buildInsightsPrompt(timeRange time.Duration) string {
	return fmt.Sprintf(`请分析过去%v时间内的决策历史，生成洞察报告：

决策历史: %s

请按以下JSON格式返回洞察：
{
  "total_decisions": 数量,
  "continue_rate": 0.0-1.0,
  "average_confidence": 0.0-1.0,
  "insights": ["洞察1", "洞察2"],
  "recommendations": ["建议1", "建议2"],
  "trends": "趋势分析"
}`, timeRange, cde.formatDecisionHistory())
}

func (cde *SmartContinuousDecisionEngine) parseInsightsResult(output any, timeRange time.Duration) map[string]any {
	var insights map[string]any
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", output)), &insights); err == nil {
		return insights
	}

	// 解析失败，返回后备洞察
	return cde.createFallbackInsights(timeRange)
}

func (cde *SmartContinuousDecisionEngine) createFallbackInsights(timeRange time.Duration) map[string]any {
	return map[string]any{
		"total_decisions":    len(cde.decisionHistory),
		"continue_rate":      0.5,
		"average_confidence": 0.5,
		"insights":           []string{"数据不足，无法生成有效洞察"},
		"time_range":         timeRange.String(),
	}
}

func (cde *SmartContinuousDecisionEngine) parseDecisionFromText(output string) *AgentContinuousDecisionResult {
	// 文本解析后备方案
	result := &AgentContinuousDecisionResult{
		ShouldContinue:           false,
		ShouldEnterWaitMode:      true,
		MonitoringTasks:          []types.MonitoringTask{},
		ReasoningSteps:           []string{"基于文本解析的默认决策"},
		Confidence:               0.5,
		EstimatedDurationMinutes: 15,
		Priority:                 5,
		RequiresUserApproval:     true,
	}

	// 简单的关键词分析
	outputLower := strings.ToLower(output)
	if strings.Contains(outputLower, "继续") || strings.Contains(outputLower, "continue") {
		result.ShouldContinue = true
		result.ShouldEnterWaitMode = false
	}

	return result
}

// 格式化辅助方法

func (cde *SmartContinuousDecisionEngine) formatExecutionResults(results []types.ExecutionResult) string {
	if len(results) == 0 {
		return "无执行结果"
	}

	var formatted strings.Builder
	for i, result := range results {
		formatted.WriteString(fmt.Sprintf("%d. 状态: %s, 输出: %v, 错误: %s\n",
			i+1, result.Status, result.Output, result.Error))
	}
	return formatted.String()
}

func (cde *SmartContinuousDecisionEngine) formatOutputAnalysis(analysis types.AgentOutputAnalysis) string {
	analysisJson, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return fmt.Sprintf("分析格式化错误: %v", err)
	}
	return string(analysisJson)
}

func (cde *SmartContinuousDecisionEngine) formatContext(context map[string]any) string {
	if len(context) == 0 {
		return "无上下文信息"
	}

	contextJson, err := json.MarshalIndent(context, "", "  ")
	if err != nil {
		return fmt.Sprintf("上下文格式化错误: %v", err)
	}
	return string(contextJson)
}

func (cde *SmartContinuousDecisionEngine) formatHistoricalPattern(pattern map[string]any) string {
	if len(pattern) == 0 {
		return "无历史模式"
	}

	patternJson, err := json.MarshalIndent(pattern, "", "  ")
	if err != nil {
		return fmt.Sprintf("历史模式格式化错误: %v", err)
	}
	return string(patternJson)
}

func (cde *SmartContinuousDecisionEngine) formatDecisionHistory() string {
	if len(cde.decisionHistory) == 0 {
		return "无决策历史"
	}

	historyJson, err := json.MarshalIndent(cde.decisionHistory, "", "  ")
	if err != nil {
		return fmt.Sprintf("决策历史格式化错误: %v", err)
	}
	return string(historyJson)
}

func (cde *SmartContinuousDecisionEngine) createFallbackOutputAnalysis(results []types.ExecutionResult) types.AgentOutputAnalysis {
	successCount := 0
	for _, result := range results {
		if result.Status == types.ExecutionStatusSuccess {
			successCount++
		}
	}

	successRate := float64(successCount) / float64(len(results))

	return types.AgentOutputAnalysis{
		KeyFindings:         []string{fmt.Sprintf("成功率: %.2f", successRate)},
		ActionableInsights:  []string{},
		RequiresUserInput:   successRate < 0.5,
		ConfidenceLevel:     successRate,
		RecommendedActions:  []string{},
		RiskAssessment:      types.RiskAssessment{Level: "low"},
		NextStepSuggestions: []string{},
	}
}

func (cde *SmartContinuousDecisionEngine) parseOutputAnalysisResult(output any, results []types.ExecutionResult) (types.AgentOutputAnalysis, error) {
	var analysis types.AgentOutputAnalysis
	if err := json.Unmarshal([]byte(fmt.Sprintf("%v", output)), &analysis); err != nil {
		return cde.createFallbackOutputAnalysis(results), nil
	}
	return analysis, nil
}

func (cde *SmartContinuousDecisionEngine) analyzeTrend(decisions []types.ContinuousDecisionResult) string {
	if len(decisions) < 3 {
		return "insufficient_data"
	}

	// 分析最近决策的趋势
	recentContinue := 0
	for _, decision := range decisions[len(decisions)-3:] {
		if decision.ShouldContinue {
			recentContinue++
		}
	}

	if recentContinue >= 2 {
		return "tend_to_continue"
	} else if recentContinue <= 1 {
		return "tend_to_stop"
	}

	return "mixed"
}
