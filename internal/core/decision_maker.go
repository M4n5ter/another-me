package core

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// SmartDecisionMaker 智能决策引擎实现
type SmartDecisionMaker struct {
	mindscapeService MindscapeService
	logger           *slog.Logger
	config           DecisionMakerConfig
}

// DecisionMakerConfig 决策引擎配置
type DecisionMakerConfig struct {
	// 记忆检索配置
	MemoryQueryLimit         int     `json:"memory_query_limit"`         // 记忆查询数量限制
	MemoryRelevanceThreshold float64 `json:"memory_relevance_threshold"` // 记忆相关性阈值

	// Agent选择配置
	DefaultAgent       AgentType `json:"default_agent"`        // 默认Agent类型
	GUIKeywordWeight   float64   `json:"gui_keyword_weight"`   // GUI关键词权重
	ReActKeywordWeight float64   `json:"react_keyword_weight"` // ReAct关键词权重

	// 监控策略配置
	MonitoringKeywords   []string      `json:"monitoring_keywords"`    // 监控关键词
	DefaultMonitoringTTL time.Duration `json:"default_monitoring_ttl"` // 默认监控时长
	MaxMonitoringTasks   int           `json:"max_monitoring_tasks"`   // 最大监控任务数

	// 决策权重配置
	HistoryWeight float64 `json:"history_weight"` // 历史经验权重
	KeywordWeight float64 `json:"keyword_weight"` // 关键词权重
	ContextWeight float64 `json:"context_weight"` // 上下文权重
}

// DefaultDecisionMakerConfig 返回默认决策引擎配置
func DefaultDecisionMakerConfig() DecisionMakerConfig {
	return DecisionMakerConfig{
		MemoryQueryLimit:         20,
		MemoryRelevanceThreshold: 0.6,
		DefaultAgent:             AgentTypeReAct,
		GUIKeywordWeight:         1.0,
		ReActKeywordWeight:       1.0,
		MonitoringKeywords:       []string{"监控", "定时", "通知", "观察", "跟踪", "检查", "提醒"},
		DefaultMonitoringTTL:     24 * time.Hour,
		MaxMonitoringTasks:       10,
		HistoryWeight:            0.3,
		KeywordWeight:            0.4,
		ContextWeight:            0.3,
	}
}

// NewSmartDecisionMaker 创建新的智能决策引擎
func NewSmartDecisionMaker(mindscapeService MindscapeService, config DecisionMakerConfig, logger *slog.Logger) *SmartDecisionMaker {
	if logger == nil {
		logger = slog.Default().WithGroup("decision_maker")
	}

	return &SmartDecisionMaker{
		mindscapeService: mindscapeService,
		logger:           logger,
		config:           config,
	}
}

// AnalyzeUserInput 分析用户输入，识别意图和任务类型
func (dm *SmartDecisionMaker) AnalyzeUserInput(ctx context.Context, decisionCtx DecisionContext) (DecisionResult, error) {
	userInput := extractUserInputFromContext(decisionCtx)
	dm.logger.Info("开始分析用户输入", "user_input", userInput)

	// 1. 提取相关记忆
	memories, err := dm.retrieveRelevantMemories(ctx, decisionCtx)
	if err != nil {
		dm.logger.Warn("检索相关记忆失败", "error", err)
	}

	// 2. 分析用户意图
	intent := dm.analyzeIntent(userInput, memories)

	// 3. 选择Agent类型
	agentType := dm.selectAgentType(userInput, intent, memories)

	// 4. 评估任务优先级
	priority := dm.evaluatePriority(decisionCtx, intent)

	// 5. 确定是否需要监控
	monitoringRequired := dm.shouldSetupMonitoring(userInput, intent)

	// 6. 生成任务定义
	task := dm.generateTask(decisionCtx, intent, agentType, priority)

	// 7. 生成监控条件（如果需要）
	var monitoringTasks []MonitoringTask
	if monitoringRequired {
		monitoring := dm.defineMonitoringConditions(decisionCtx, intent, task)
		monitoringTasks = []MonitoringTask{monitoring}
	}

	result := DecisionResult{
		ShouldExecuteTask:   true,
		Task:                Some(task),
		MonitoringTasks:     monitoringTasks,
		ShouldEnterWaitMode: false,
		ReasoningSteps:      []string{dm.generateReasoning(intent, agentType, monitoringRequired)},
		Confidence:          dm.calculateConfidence(intent, memories),
		ExpectedDuration:    Some(5 * time.Minute), // 默认预期5分钟
	}

	dm.logger.Info("用户输入分析完成",
		"agent_type", agentType,
		"priority", priority,
		"monitoring_required", monitoringRequired,
		"confidence", result.Confidence)

	return result, nil
}

// SelectAgent 根据任务上下文选择最适合的Agent
func (dm *SmartDecisionMaker) SelectAgent(ctx context.Context, task Task, availableAgents []AgentType) (AgentType, error) {
	dm.logger.Debug("选择Agent", "task_type", task.Type, "available_agents", availableAgents)

	if len(availableAgents) == 0 {
		return AgentTypeUnknown, fmt.Errorf("没有可用的Agent")
	}

	// 从任务描述中分析意图
	intent := dm.analyzeIntent(task.Description, []MemoryItem{})

	// 计算每个可用Agent的适合度分数
	scores := make(map[AgentType]float64)
	for _, agentType := range availableAgents {
		scores[agentType] = dm.calculateAgentScore(task, intent, agentType)
	}

	// 选择分数最高的Agent
	bestAgent := AgentTypeUnknown
	bestScore := 0.0
	for agentType, score := range scores {
		if score > bestScore {
			bestScore = score
			bestAgent = agentType
		}
	}

	if bestAgent == AgentTypeUnknown {
		bestAgent = availableAgents[0] // 默认选择第一个可用的Agent
	}

	dm.logger.Info("Agent选择完成", "selected_agent", bestAgent, "score", bestScore)
	return bestAgent, nil
}

// DefineMonitoringConditions 定义监控条件
func (dm *SmartDecisionMaker) DefineMonitoringConditions(ctx context.Context, userInput string, context map[string]any) ([]MonitorCondition, error) {
	dm.logger.Debug("定义监控条件", "user_input", userInput)

	conditions := []MonitorCondition{}

	// 分析用户输入中的监控需求
	if dm.containsMonitoringKeywords(userInput) {
		// 时间条件监控
		if timeCondition := dm.extractTimeCondition(userInput); timeCondition.IsSome() {
			conditions = append(conditions, timeCondition.Unwrap())
		}

		// 事件条件监控
		if eventConditions := dm.extractEventConditions(userInput, context); len(eventConditions) > 0 {
			conditions = append(conditions, eventConditions...)
		}

		// 状态条件监控
		if stateConditions := dm.extractStateConditions(userInput, context); len(stateConditions) > 0 {
			conditions = append(conditions, stateConditions...)
		}
	}

	dm.logger.Info("监控条件定义完成", "condition_count", len(conditions))
	return conditions, nil
}

// MakeDecisionBasedOnMemory 基于历史记忆做决策
func (dm *SmartDecisionMaker) MakeDecisionBasedOnMemory(ctx context.Context, memories []MemoryItem, currentContext DecisionContext) (DecisionResult, error) {
	dm.logger.Debug("基于记忆做决策", "memory_count", len(memories))

	// 分析记忆中的模式
	patterns := dm.analyzeMemoryPatterns(memories)

	// 结合当前上下文和历史模式做决策
	enhancedContext := dm.enhanceContextWithMemory(currentContext, patterns)

	// 使用增强的上下文进行标准决策流程
	return dm.AnalyzeUserInput(ctx, enhancedContext)
}

// HandleWakeupEvent 处理唤醒事件
func (dm *SmartDecisionMaker) HandleWakeupEvent(ctx context.Context, wakeupEvent WakeupEvent) (DecisionResult, error) {
	dm.logger.Info("处理唤醒事件", "task_id", wakeupEvent.MonitoringTaskID, "reason", wakeupEvent.Reason)

	// 构建基于唤醒事件的决策上下文
	decisionCtx := DecisionContext{
		WakeupEvent: Some(wakeupEvent),
		SystemState: map[string]any{
			"mode":          "monitoring",
			"last_activity": wakeupEvent.TriggerTime,
			"memory_usage":  "normal",
		},
		RetrievedMemories: []MemoryItem{},
		Timestamp:         wakeupEvent.TriggerTime,
	}

	// 检索相关记忆
	memories, err := dm.mindscapeService.RetrieveMemories(ctx, map[string]any{
		"query":   wakeupEvent.Reason,
		"task_id": wakeupEvent.MonitoringTaskID,
		"limit":   dm.config.MemoryQueryLimit,
	})
	if err != nil {
		dm.logger.Warn("检索唤醒相关记忆失败", "error", err)
		memories = []MemoryItem{}
	}

	decisionCtx.RetrievedMemories = memories

	// 基于唤醒事件和相关记忆做决策
	result, err := dm.AnalyzeUserInput(ctx, decisionCtx)
	if err != nil {
		return DecisionResult{}, fmt.Errorf("处理唤醒事件决策失败: %w", err)
	}

	dm.logger.Info("唤醒事件处理完成", "should_execute", result.ShouldExecuteTask)
	return result, nil
}

// 私有方法实现

// retrieveRelevantMemories 检索相关记忆
func (dm *SmartDecisionMaker) retrieveRelevantMemories(ctx context.Context, decisionCtx DecisionContext) ([]MemoryItem, error) {
	userInput := extractUserInputFromContext(decisionCtx)
	queryContext := map[string]any{
		"query":     userInput,
		"limit":     dm.config.MemoryQueryLimit,
		"min_score": int(dm.config.MemoryRelevanceThreshold * 100),
	}

	// 检查是否有用户ID信息
	if userID, exists := decisionCtx.SystemState["user_id"]; exists {
		queryContext["user_id"] = userID
	}

	result, err := dm.mindscapeService.RetrieveMemories(ctx, queryContext)
	if err != nil {
		return nil, fmt.Errorf("检索相关记忆失败: %w", err)
	}

	return result, nil
}

// analyzeIntent 分析用户意图
func (dm *SmartDecisionMaker) analyzeIntent(userInput string, memories []MemoryItem) map[string]any {
	intent := map[string]any{
		"gui_score":        dm.calculateGUIScore(userInput),
		"react_score":      dm.calculateReActScore(userInput),
		"monitoring_score": dm.calculateMonitoringScore(userInput),
		"priority_score":   dm.calculatePriorityScore(userInput),
		"keywords":         dm.extractKeywords(userInput),
		"action_type":      dm.determineActionType(userInput),
	}

	// 利用历史记忆增强意图分析
	if len(memories) > 0 {
		intent["historical_patterns"] = dm.extractHistoricalPatterns(memories)
	}

	return intent
}

// selectAgentType 选择Agent类型
func (dm *SmartDecisionMaker) selectAgentType(userInput string, intent map[string]any, memories []MemoryItem) AgentType {
	guiScore := intent["gui_score"].(float64)
	reactScore := intent["react_score"].(float64)

	// 基于历史偏好调整分数
	if len(memories) > 0 {
		historyAdjustment := dm.getHistoricalAgentPreference(memories)
		guiScore += historyAdjustment["gui"] * dm.config.HistoryWeight
		reactScore += historyAdjustment["react"] * dm.config.HistoryWeight
	}

	if guiScore > reactScore {
		return AgentTypeGUI
	} else if reactScore > guiScore {
		return AgentTypeReAct
	}

	return dm.config.DefaultAgent
}

// calculateGUIScore 计算GUI相关分数
func (dm *SmartDecisionMaker) calculateGUIScore(input string) float64 {
	guiKeywords := []string{
		"点击", "click", "拖拽", "drag", "输入", "input", "type",
		"截图", "screenshot", "滚动", "scroll", "窗口", "window",
		"按钮", "button", "菜单", "menu", "界面", "interface",
		"鼠标", "mouse", "键盘", "keyboard", "选择", "select",
	}

	return dm.calculateKeywordScore(input, guiKeywords) * dm.config.GUIKeywordWeight
}

// calculateReActScore 计算ReAct相关分数
func (dm *SmartDecisionMaker) calculateReActScore(input string) float64 {
	reactKeywords := []string{
		"搜索", "search", "查询", "query", "分析", "analyze",
		"计算", "calculate", "对话", "chat", "工具", "tool",
		"思考", "think", "推理", "reason", "解决", "solve",
		"处理", "process", "获取", "get", "调用", "call",
	}

	return dm.calculateKeywordScore(input, reactKeywords) * dm.config.ReActKeywordWeight
}

// calculateMonitoringScore 计算监控相关分数
func (dm *SmartDecisionMaker) calculateMonitoringScore(input string) float64 {
	return dm.calculateKeywordScore(input, dm.config.MonitoringKeywords)
}

// calculateKeywordScore 计算关键词分数
func (dm *SmartDecisionMaker) calculateKeywordScore(input string, keywords []string) float64 {
	input = strings.ToLower(input)
	score := 0.0

	for _, keyword := range keywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			score += 1.0
		}
	}

	return score / float64(len(keywords))
}

// evaluatePriority 评估任务优先级
func (dm *SmartDecisionMaker) evaluatePriority(ctx DecisionContext, intent map[string]any) int {
	priority := 50 // 默认优先级

	// 基于关键词调整优先级
	priorityScore := intent["priority_score"].(float64)
	priority += int(priorityScore * 30)

	// 基于系统状态调整优先级
	if mode, exists := ctx.SystemState["mode"]; exists {
		if mode == "monitoring" {
			priority += 20 // 监控触发的任务优先级较高
		}
	}

	// 基于触发事件调整优先级
	if ctx.WakeupEvent.IsSome() {
		priority += 25 // 事件触发的任务优先级高
	}

	// 限制优先级范围
	if priority > 100 {
		priority = 100
	} else if priority < 1 {
		priority = 1
	}

	return priority
}

// shouldSetupMonitoring 判断是否需要设置监控
func (dm *SmartDecisionMaker) shouldSetupMonitoring(userInput string, intent map[string]any) bool {
	// 直接检查是否包含监控关键词
	if dm.containsMonitoringKeywords(userInput) {
		return true
	}

	// 额外检查监控分数
	monitoringScore := intent["monitoring_score"].(float64)
	return monitoringScore > 0.2 // 降低监控分数阈值
}

// generateTask 生成任务定义
func (dm *SmartDecisionMaker) generateTask(ctx DecisionContext, intent map[string]any, agentType AgentType, priority int) Task {
	taskID := fmt.Sprintf("task_%d", time.Now().UnixNano())
	userInput := extractUserInputFromContext(ctx)

	// 确定任务类型
	var taskType string
	if agentType == AgentTypeGUI {
		taskType = dm.determineGUITaskType(userInput)
	} else {
		taskType = dm.determineReActTaskType(userInput)
	}

	// 构建任务参数
	parameters := map[string]any{
		"user_input": userInput,
	}

	if agentType == AgentTypeGUI {
		parameters["instruction"] = userInput
		// 如果有截图URL，添加到参数中
		if imageURL, exists := ctx.SystemState["screenshot_url"]; exists {
			parameters["image_url"] = imageURL
		}
	}

	return Task{
		ID:          taskID,
		Type:        taskType,
		Description: userInput,
		AgentType:   agentType,
		Priority:    priority,
		Parameters:  parameters,
		CreatedAt:   time.Now(),
		Context:     ctx.SystemState,
	}
}

// defineMonitoringConditions 定义监控条件（内部方法）
func (dm *SmartDecisionMaker) defineMonitoringConditions(ctx DecisionContext, intent map[string]any, task Task) MonitoringTask {
	userInput := extractUserInputFromContext(ctx)
	conditions, _ := dm.DefineMonitoringConditions(context.Background(), userInput, ctx.SystemState)

	return MonitoringTask{
		ID:                  None[string](),
		Description:         fmt.Sprintf("监控任务: %s", userInput),
		MindscapeTaskType:   "web", // 默认web监控
		Conditions:          conditions,
		TargetData:          []string{"status", "content", "timestamp"},
		NotificationMethods: []string{"webhook"},
		WebhookURL:          Some("http://localhost:8081/wakeup"), // 默认webhook URL
		MQTopic:             None[string](),
		MaxRetries:          Some(3),
		IsEnabled:           true,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}
}

// calculateConfidence 计算决策置信度
func (dm *SmartDecisionMaker) calculateConfidence(intent map[string]any, memories []MemoryItem) float64 {
	confidence := 0.5 // 基础置信度

	// 基于关键词匹配度调整置信度
	guiScore := intent["gui_score"].(float64)
	reactScore := intent["react_score"].(float64)

	maxScore := max(guiScore, reactScore)
	confidence += maxScore * 0.3

	// 基于历史记忆调整置信度
	if len(memories) > 0 {
		memoryBonus := min(float64(len(memories))/10.0, 0.2)
		confidence += memoryBonus
	}

	// 限制置信度范围
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// generateReasoning 生成决策推理过程
func (dm *SmartDecisionMaker) generateReasoning(intent map[string]any, agentType AgentType, monitoringRequired bool) string {
	var reasoning strings.Builder

	reasoning.WriteString(fmt.Sprintf("选择%s Agent，因为", agentType))

	if agentType == AgentTypeGUI {
		reasoning.WriteString("检测到GUI操作相关关键词")
	} else {
		reasoning.WriteString("检测到工具调用或推理分析需求")
	}

	if monitoringRequired {
		reasoning.WriteString("；检测到监控需求，将设置相应的监控任务")
	}

	return reasoning.String()
}

// 工具方法

func (dm *SmartDecisionMaker) containsMonitoringKeywords(input string) bool {
	input = strings.ToLower(input)
	for _, keyword := range dm.config.MonitoringKeywords {
		if strings.Contains(input, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func (dm *SmartDecisionMaker) extractTimeCondition(input string) Option[MonitorCondition] {
	// 简单的时间条件提取逻辑
	timeRegex := regexp.MustCompile(`(\d+)(分钟|小时|天|秒)`)
	matches := timeRegex.FindStringSubmatch(input)

	if len(matches) >= 3 {
		return Some(MonitorCondition{
			Type:     "time_interval",
			Property: "interval",
			Operator: "every",
			Value:    matches[0],
		})
	}

	return None[MonitorCondition]()
}

func (dm *SmartDecisionMaker) extractEventConditions(input string, context map[string]any) []MonitorCondition {
	conditions := []MonitorCondition{}

	// 基于输入提取事件条件的简单逻辑
	if strings.Contains(strings.ToLower(input), "变化") || strings.Contains(strings.ToLower(input), "更新") {
		conditions = append(conditions, MonitorCondition{
			Type:     "change_detection",
			Property: "content",
			Operator: "changed",
			Value:    "",
		})
	}

	return conditions
}

func (dm *SmartDecisionMaker) extractStateConditions(input string, context map[string]any) []MonitorCondition {
	conditions := []MonitorCondition{}

	// 基于输入提取状态条件的简单逻辑
	if strings.Contains(strings.ToLower(input), "状态") {
		conditions = append(conditions, MonitorCondition{
			Type:     "status_check",
			Property: "status",
			Operator: "equals",
			Value:    "active",
		})
	}

	return conditions
}

func (dm *SmartDecisionMaker) calculatePriorityScore(input string) float64 {
	urgentKeywords := []string{"紧急", "urgent", "立即", "immediately", "马上", "asap"}
	return dm.calculateKeywordScore(input, urgentKeywords)
}

func (dm *SmartDecisionMaker) extractKeywords(input string) []string {
	// 简单的关键词提取
	words := strings.Fields(strings.ToLower(input))
	return words
}

func (dm *SmartDecisionMaker) determineActionType(input string) string {
	switch {
	case strings.Contains(strings.ToLower(input), "点击"):
		return "click"
	case strings.Contains(strings.ToLower(input), "输入"):
		return "input"
	case strings.Contains(strings.ToLower(input), "搜索"):
		return "search"
	}
	return "general"
}

func (dm *SmartDecisionMaker) extractHistoricalPatterns(memories []MemoryItem) map[string]any {
	patterns := map[string]any{
		"preferred_agent": "unknown",
		"common_tasks":    []string{},
		"success_rate":    0.0,
	}

	// 分析历史记忆中的模式
	agentCounts := map[string]int{}
	successCount := 0

	for _, memory := range memories {
		if memory.Type == MemoryTypeTaskSummary {
			if metadata, ok := memory.Metadata["agent_type"].(string); ok {
				agentCounts[metadata]++
			}
			if success, ok := memory.Metadata["success"].(bool); ok && success {
				successCount++
			}
		}
	}

	// 确定偏好的Agent
	maxCount := 0
	for agent, count := range agentCounts {
		if count > maxCount {
			maxCount = count
			patterns["preferred_agent"] = agent
		}
	}

	// 计算成功率
	if len(memories) > 0 {
		patterns["success_rate"] = float64(successCount) / float64(len(memories))
	}

	return patterns
}

func (dm *SmartDecisionMaker) getHistoricalAgentPreference(memories []MemoryItem) map[string]float64 {
	preference := map[string]float64{
		"gui":   0.0,
		"react": 0.0,
	}

	for _, memory := range memories {
		if agentType, ok := memory.Metadata["agent_type"].(string); ok {
			switch agentType {
			case "gui":
				preference["gui"] += 0.1
			case "react":
				preference["react"] += 0.1
			}
		}
	}

	return preference
}

func (dm *SmartDecisionMaker) calculateAgentScore(task Task, intent map[string]any, agentType AgentType) float64 {
	score := 0.0

	switch agentType {
	case AgentTypeGUI:
		score = intent["gui_score"].(float64)
	case AgentTypeReAct:
		score = intent["react_score"].(float64)
	}

	// 基于任务类型调整分数
	if agentType == AgentTypeGUI && strings.Contains(task.Type, "gui") {
		score += 0.5
	} else if agentType == AgentTypeReAct && strings.Contains(task.Type, "react") {
		score += 0.5
	}

	return score
}

func (dm *SmartDecisionMaker) analyzeMemoryPatterns(memories []MemoryItem) map[string]any {
	patterns := map[string]any{
		"task_frequency":   map[string]int{},
		"success_patterns": []string{},
		"failure_patterns": []string{},
		"time_patterns":    map[string]int{},
	}

	taskFreq := map[string]int{}
	successPatterns := []string{}
	failurePatterns := []string{}

	for _, memory := range memories {
		// 分析任务频率
		if taskType, ok := memory.Metadata["task_type"].(string); ok {
			taskFreq[taskType]++
		}

		// 分析成功/失败模式
		if success, ok := memory.Metadata["success"].(bool); ok {
			if success {
				successPatterns = append(successPatterns, memory.Content.(string))
			} else {
				failurePatterns = append(failurePatterns, memory.Content.(string))
			}
		}
	}

	patterns["task_frequency"] = taskFreq
	patterns["success_patterns"] = successPatterns
	patterns["failure_patterns"] = failurePatterns

	return patterns
}

func (dm *SmartDecisionMaker) enhanceContextWithMemory(ctx DecisionContext, patterns map[string]any) DecisionContext {
	// 使用记忆模式增强决策上下文
	enhancedCtx := ctx

	// 添加历史模式到系统状态中
	if enhancedCtx.SystemState == nil {
		enhancedCtx.SystemState = map[string]any{}
	}

	enhancedCtx.SystemState["memory_patterns"] = patterns

	return enhancedCtx
}

func (dm *SmartDecisionMaker) determineGUITaskType(input string) string {
	input = strings.ToLower(input)

	switch {
	case strings.Contains(input, "点击"):
		return "gui_click"
	case strings.Contains(input, "输入"):
		return "gui_input"
	case strings.Contains(input, "拖拽"):
		return "gui_drag"
	case strings.Contains(input, "截图"):
		return "gui_screenshot"
	}

	return "gui_general"
}

func (dm *SmartDecisionMaker) determineReActTaskType(input string) string {
	input = strings.ToLower(input)

	switch {
	case strings.Contains(input, "搜索"):
		return "react_search"
	case strings.Contains(input, "分析"):
		return "react_analysis"
	case strings.Contains(input, "计算"):
		return "react_calculate"
	case strings.Contains(input, "工具"):
		return "react_tool_call"
	}

	return "react_general"
}

// extractUserInputFromContext 从决策上下文中提取用户输入
func extractUserInputFromContext(ctx DecisionContext) string {
	// 从唤醒事件中提取
	if ctx.WakeupEvent.IsSome() {
		return fmt.Sprintf("监控任务触发: %s", ctx.WakeupEvent.Unwrap().Reason)
	}

	// 从系统状态中提取
	if userInput, exists := ctx.SystemState["user_input"]; exists {
		if str, ok := userInput.(string); ok {
			return str
		}
	}

	// 从元数据中提取
	if userInput, exists := ctx.SystemState["instruction"]; exists {
		if str, ok := userInput.(string); ok {
			return str
		}
	}

	return "系统自动决策" // 默认值
}
