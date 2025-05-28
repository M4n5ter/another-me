package core

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	memoryDTO "github.com/m4n5ter/mindscape/memory/api/dto"
	memoryCore "github.com/m4n5ter/mindscape/memory/core"
	sentinelDTO "github.com/m4n5ter/mindscape/sentinel/api/dto"
	"github.com/m4n5ter/mindscape/sentinel/core/tasktypes"

	"github.com/google/uuid"

	"github.com/m4n5ter/another-me/internal/mindscape"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// MindscapeConnector 实现MindscapeService接口，封装Mindscape客户端
type MindscapeConnector struct {
	client          *mindscape.Client
	logger          *slog.Logger
	wakeupListener  Option[WakeupListener]
	pendingQueue    []QueuedOperation // 本地持久化队列，用于Mindscape不可用时
	queueMutex      sync.RWMutex
	isHealthy       bool
	lastHealthCheck time.Time
	config          MindscapeConnectorConfig
}

// MindscapeConnectorConfig Mindscape连接器配置
type MindscapeConnectorConfig struct {
	Host                string        `json:"host"`
	Port                int           `json:"port"`
	TLS                 bool          `json:"tls"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
	QueueMaxSize        int           `json:"queue_max_size"`
	WebhookListenPort   int           `json:"webhook_listen_port"`
	WebhookListenPath   string        `json:"webhook_listen_path"`
}

// QueuedOperation 排队的操作
type QueuedOperation struct {
	Type      string    `json:"type"`      // "store_memory", "delegate_task", "clear_tasks"
	Data      any       `json:"data"`      // 操作数据
	Timestamp time.Time `json:"timestamp"` // 操作时间
	Retries   int       `json:"retries"`   // 重试次数
	ID        string    `json:"id"`        // 操作ID
}

// DefaultMindscapeConnectorConfig 返回默认配置
func DefaultMindscapeConnectorConfig() MindscapeConnectorConfig {
	return MindscapeConnectorConfig{
		Host:                "localhost",
		Port:                8080,
		TLS:                 false,
		HealthCheckInterval: 30 * time.Second,
		RetryAttempts:       3,
		RetryDelay:          5 * time.Second,
		QueueMaxSize:        1000,
		WebhookListenPort:   8081,
		WebhookListenPath:   "/wakeup",
	}
}

// NewMindscapeConnector 创建新的Mindscape连接器
func NewMindscapeConnector(config MindscapeConnectorConfig, logger *slog.Logger) *MindscapeConnector {
	if logger == nil {
		logger = slog.Default().WithGroup("mindscape_connector")
	}

	mindscapeConfig := mindscape.Config{
		Host: config.Host,
		Port: config.Port,
		TLS:  config.TLS,
	}

	client := mindscape.NewClient(mindscapeConfig)

	connector := &MindscapeConnector{
		client:         client,
		logger:         logger,
		wakeupListener: None[WakeupListener](),
		pendingQueue:   make([]QueuedOperation, 0),
		isHealthy:      false,
		config:         config,
	}

	// 启动后台健康检查
	go connector.startHealthChecker()

	return connector
}

// StoreMemory 存储记忆到Mindscape
func (mc *MindscapeConnector) StoreMemory(ctx context.Context, memoryData MemoryItem) error {
	mc.logger.Debug("存储记忆", "memory_id", memoryData.ID, "type", memoryData.Type)

	// 转换记忆类型
	var memoryType memoryCore.MemoryType
	switch memoryData.Type {
	case MemoryTypeObservation:
		memoryType = memoryCore.EpisodicInteraction
	case MemoryTypeUserPref:
		memoryType = memoryCore.UserPreference
	case MemoryTypeUserProfile:
		memoryType = memoryCore.UserTrait
	default:
		memoryType = memoryCore.SemanticFact
	}

	// 转换为Mindscape API格式
	storeReq := memoryDTO.StoreMemoryRequest{
		Type:             memoryType,
		Source:           Some("another-me"),
		ContentRaw:       fmt.Sprintf("%v", memoryData.Content),
		ContentProcessed: None[memoryCore.ProcessedContent](),
		ImportanceScore:  Some(memoryData.Importance),
		ConfidenceScore:  Some(0.9), // 默认置信度
		ContextTags:      Some(memoryData.Keywords),
		UserAssociation:  mc.createUserAssociation(memoryData.UserID),
		Metadata:         Some(memoryData.Metadata),
	}

	// 检查连接健康状态
	if !mc.isHealthy {
		mc.logger.Warn("Mindscape不可用，将操作加入队列", "memory_id", memoryData.ID)
		return mc.queueOperation("store_memory", storeReq)
	}

	// 尝试存储
	_, err := mc.client.Memory.StoreMemoryFragment(ctx, storeReq)
	if err != nil {
		mc.logger.Error("存储记忆失败", "error", err, "memory_id", memoryData.ID)
		// 如果是网络错误，加入队列重试
		if mc.isNetworkError(err) {
			return mc.queueOperation("store_memory", storeReq)
		}
		return fmt.Errorf("存储记忆失败: %w", err)
	}

	mc.logger.Info("记忆存储成功", "memory_id", memoryData.ID)
	return nil
}

// RetrieveMemories 从Mindscape检索相关记忆
func (mc *MindscapeConnector) RetrieveMemories(ctx context.Context, queryContext map[string]any) ([]MemoryItem, error) {
	mc.logger.Debug("检索记忆", "query_context", queryContext)

	// 检查连接健康状态
	if !mc.isHealthy {
		mc.logger.Warn("Mindscape不可用，返回空结果")
		return []MemoryItem{}, nil
	}

	// 构建检索请求
	recallReq := memoryDTO.RecallMemoriesRequest{
		QueryText: extractStringFromContext(queryContext, "query", ""),
		UserID:    extractStringFromContext(queryContext, "user_id", ""),
		TopK:      extractIntFromContext(queryContext, "limit", 10),
		MinScore:  float64(extractIntFromContext(queryContext, "min_score", 0)) / 100.0,
	}

	// 执行检索
	resp, err := mc.client.Memory.RecallMemory(ctx, recallReq)
	if err != nil {
		mc.logger.Error("检索记忆失败", "error", err)
		return nil, fmt.Errorf("检索记忆失败: %w", err)
	}

	// 转换结果
	memories := make([]MemoryItem, 0, len(resp.Fragments))
	for _, fragment := range resp.Fragments {
		memory := MemoryItem{
			ID:         fragment.ID.String(),
			Timestamp:  fragment.TimestampCreated,
			Type:       mc.convertMemoryType(fragment.Type),
			Content:    fragment.ContentRaw,
			Keywords:   fragment.ContextTags,
			Importance: fragment.ImportanceScore,
			RelatedIDs: fragment.RelatedFragmentIDs,
			UserID:     mc.extractUserIDFromAssociation(fragment.UserAssociation),
			Metadata:   fragment.Metadata,
		}
		memories = append(memories, memory)
	}

	mc.logger.Info("记忆检索成功", "count", len(memories))
	return memories, nil
}

// DelegateMonitoringTask 委托监控任务给Mindscape
func (mc *MindscapeConnector) DelegateMonitoringTask(ctx context.Context, taskDetails MonitoringTask) (string, error) {
	mc.logger.Debug("委托监控任务", "description", taskDetails.Description)

	// 转换为Mindscape API格式
	createReq := mc.convertToSentinelCreateRequest(taskDetails)

	// 检查连接健康状态
	if !mc.isHealthy {
		mc.logger.Warn("Mindscape不可用，将监控任务加入队列")
		return "", mc.queueOperation("delegate_task", createReq)
	}

	// 创建监控任务
	resp, err := mc.client.Sentinel.CreateTask(ctx, createReq)
	if err != nil {
		mc.logger.Error("创建监控任务失败", "error", err)
		// 如果是网络错误，加入队列重试
		if mc.isNetworkError(err) {
			return "", mc.queueOperation("delegate_task", createReq)
		}
		return "", fmt.Errorf("创建监控任务失败: %w", err)
	}

	taskID := resp.ID.String()
	mc.logger.Info("监控任务创建成功", "task_id", taskID)
	return taskID, nil
}

// ClearOrUpdateMonitoringTasks 清除或更新监控任务
func (mc *MindscapeConnector) ClearOrUpdateMonitoringTasks(ctx context.Context, taskUpdate TaskUpdate) error {
	mc.logger.Debug("更新监控任务", "tasks_to_update", len(taskUpdate.TasksToUpdate), "tasks_to_delete", len(taskUpdate.TaskIDsToDelete))

	// 检查连接健康状态
	if !mc.isHealthy {
		mc.logger.Warn("Mindscape不可用，将任务更新加入队列")
		return mc.queueOperation("clear_tasks", taskUpdate)
	}

	// 删除任务
	for _, taskID := range taskUpdate.TaskIDsToDelete {
		taskUUID, err := uuid.Parse(taskID)
		if err != nil {
			mc.logger.Error("任务ID格式错误", "task_id", taskID, "error", err)
			continue
		}

		err = mc.client.Sentinel.DeleteTask(ctx, taskUUID)
		if err != nil {
			mc.logger.Error("删除监控任务失败", "task_id", taskID, "error", err)
			return fmt.Errorf("删除监控任务失败: %w", err)
		}
		mc.logger.Info("监控任务删除成功", "task_id", taskID)
	}

	// 更新任务（通过删除+创建实现）
	for _, task := range taskUpdate.TasksToUpdate {
		if task.ID.IsSome() {
			// 先删除旧任务
			taskUUID, err := uuid.Parse(task.ID.Unwrap())
			if err != nil {
				mc.logger.Error("任务ID格式错误", "task_id", task.ID.Unwrap(), "error", err)
				continue
			}

			err = mc.client.Sentinel.DeleteTask(ctx, taskUUID)
			if err != nil {
				mc.logger.Error("删除旧监控任务失败", "task_id", task.ID.Unwrap(), "error", err)
			}
		}

		// 创建新任务
		if task.IsEnabled {
			_, err := mc.DelegateMonitoringTask(ctx, task)
			if err != nil {
				defaultTaskID := "new"
				if task.ID.IsSome() {
					defaultTaskID = task.ID.Unwrap()
				}
				mc.logger.Error("更新监控任务失败", "task_id", defaultTaskID, "error", err)
				return err
			}
		}
	}

	mc.logger.Info("监控任务更新完成")
	return nil
}

// SetupWakeUpListener 设置唤醒监听器
func (mc *MindscapeConnector) SetupWakeUpListener(handler func(wakeupData WakeupEvent) error) error {
	mc.logger.Info("设置唤醒监听器", "port", mc.config.WebhookListenPort)

	// 创建HTTP监听器作为唤醒监听器
	listener := mc.createWebhookWakeupListener(mc.config.WebhookListenPort, mc.config.WebhookListenPath, mc.logger)
	listener.SetHandler(handler)

	// 启动监听器
	ctx := context.Background()
	err := listener.Start(ctx)
	if err != nil {
		return fmt.Errorf("启动唤醒监听器失败: %w", err)
	}

	mc.wakeupListener = Some(listener)
	mc.logger.Info("唤醒监听器启动成功", "address", listener.GetListenAddress())
	return nil
}

// IsHealthy 检查Mindscape连接是否健康
func (mc *MindscapeConnector) IsHealthy(ctx context.Context) bool {
	// 如果最近检查过且结果是健康的，直接返回
	if time.Since(mc.lastHealthCheck) < mc.config.HealthCheckInterval && mc.isHealthy {
		return mc.isHealthy
	}

	// 执行健康检查
	err := mc.client.Sentinel.HealthCheck(ctx)
	mc.lastHealthCheck = time.Now()
	mc.isHealthy = err == nil

	if !mc.isHealthy {
		mc.logger.Warn("Mindscape健康检查失败", "error", err)
	} else {
		mc.logger.Debug("Mindscape健康检查通过")
	}

	return mc.isHealthy
}

// GetUserProfile 获取用户画像
func (mc *MindscapeConnector) GetUserProfile(ctx context.Context, userID string) (*MemoryItem, error) {
	mc.logger.Debug("获取用户画像", "user_id", userID)

	if !mc.isHealthy {
		return nil, fmt.Errorf("Mindscape服务不可用")
	}

	resp, err := mc.client.Memory.GetUserProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("获取用户画像失败: %w", err)
	}

	profile := &MemoryItem{
		ID:         resp.ID,
		Timestamp:  resp.LastInteraction,
		Type:       MemoryTypeUserProfile,
		Content:    resp.Preferences,
		Keywords:   []string{"用户画像", "profile"},
		Importance: 1.0,
		RelatedIDs: []string{},
		UserID:     resp.ID,
		Metadata: map[string]any{
			"username":         resp.Username,
			"habits":           resp.Habits,
			"knowledge_areas":  resp.KnowledgeAreas,
			"last_interaction": resp.LastInteraction,
		},
	}

	return profile, nil
}

// UpdateUserProfile 更新用户画像
func (mc *MindscapeConnector) UpdateUserProfile(ctx context.Context, userID string, profileData map[string]any) error {
	mc.logger.Debug("更新用户画像", "user_id", userID)

	if !mc.isHealthy {
		return mc.queueOperation("update_profile", map[string]any{
			"user_id": userID,
			"profile": profileData,
		})
	}

	updateReq := memoryDTO.UpdateUserProfileRequest{
		Username:         extractStringOption(profileData, "username"),
		Preferences:      extractStringMapOption(profileData, "preferences"),
		Habits:           extractStringSliceOption(profileData, "habits"),
		KnowledgeAreas:   extractStringSliceOption(profileData, "knowledge_areas"),
		CustomEmbeddings: None[map[string][]float32](),
	}

	_, err := mc.client.Memory.UpdateUserProfile(ctx, userID, updateReq)
	if err != nil {
		return fmt.Errorf("更新用户画像失败: %w", err)
	}

	mc.logger.Info("用户画像更新成功", "user_id", userID)
	return nil
}

// 私有方法

// startHealthChecker 启动后台健康检查
func (mc *MindscapeConnector) startHealthChecker() {
	ticker := time.NewTicker(mc.config.HealthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		wasHealthy := mc.isHealthy
		mc.IsHealthy(ctx)
		cancel()

		// 如果从不健康变为健康，处理排队的操作
		if !wasHealthy && mc.isHealthy {
			mc.processQueuedOperations()
		}
	}
}

// processQueuedOperations 处理排队的操作
func (mc *MindscapeConnector) processQueuedOperations() {
	mc.queueMutex.Lock()
	defer mc.queueMutex.Unlock()

	if len(mc.pendingQueue) == 0 {
		return
	}

	mc.logger.Info("开始处理排队的操作", "count", len(mc.pendingQueue))

	processedCount := 0
	failedOperations := make([]QueuedOperation, 0)

	for _, op := range mc.pendingQueue {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err := mc.processQueuedOperation(ctx, op)
		cancel()

		if err != nil {
			op.Retries++
			if op.Retries < mc.config.RetryAttempts {
				failedOperations = append(failedOperations, op)
			} else {
				mc.logger.Error("操作重试次数超限，丢弃", "op_id", op.ID, "type", op.Type, "retries", op.Retries)
			}
		} else {
			processedCount++
		}
	}

	mc.pendingQueue = failedOperations
	mc.logger.Info("排队操作处理完成", "processed", processedCount, "failed", len(failedOperations))
}

// processQueuedOperation 处理单个排队的操作
func (mc *MindscapeConnector) processQueuedOperation(ctx context.Context, op QueuedOperation) error {
	switch op.Type {
	case "store_memory":
		if req, ok := op.Data.(memoryDTO.StoreMemoryRequest); ok {
			_, err := mc.client.Memory.StoreMemoryFragment(ctx, req)
			return err
		}
	case "delegate_task":
		if req, ok := op.Data.(sentinelDTO.CreateTaskRequest); ok {
			_, err := mc.client.Sentinel.CreateTask(ctx, req)
			return err
		}
	case "clear_tasks":
		if req, ok := op.Data.(TaskUpdate); ok {
			return mc.ClearOrUpdateMonitoringTasks(ctx, req)
		}
	case "update_profile":
		if data, ok := op.Data.(map[string]any); ok {
			userID := data["user_id"].(string)
			profile := data["profile"].(map[string]any)
			return mc.UpdateUserProfile(ctx, userID, profile)
		}
	}
	return fmt.Errorf("未知的操作类型: %s", op.Type)
}

// queueOperation 将操作加入队列
func (mc *MindscapeConnector) queueOperation(opType string, data any) error {
	mc.queueMutex.Lock()
	defer mc.queueMutex.Unlock()

	if len(mc.pendingQueue) >= mc.config.QueueMaxSize {
		mc.logger.Error("队列已满，无法添加新操作", "type", opType)
		return fmt.Errorf("操作队列已满")
	}

	op := QueuedOperation{
		Type:      opType,
		Data:      data,
		Timestamp: time.Now(),
		Retries:   0,
		ID:        generateOperationID(),
	}

	mc.pendingQueue = append(mc.pendingQueue, op)
	mc.logger.Info("操作已加入队列", "type", opType, "queue_size", len(mc.pendingQueue))
	return nil
}

// isNetworkError 检查是否为网络错误
func (mc *MindscapeConnector) isNetworkError(err error) bool {
	// 简单的网络错误检测逻辑
	return err != nil
}

// 工具函数

func extractStringFromContext(ctx map[string]any, key, defaultValue string) string {
	if val, ok := ctx[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

func extractIntFromContext(ctx map[string]any, key string, defaultValue int) int {
	if val, ok := ctx[key]; ok {
		if i, ok := val.(int); ok {
			return i
		}
	}
	return defaultValue
}

func extractStringSliceFromContext(ctx map[string]any, key string) Option[[]string] {
	if val, ok := ctx[key]; ok {
		if slice, ok := val.([]string); ok {
			return Some(slice)
		}
		if interfaces, ok := val.([]any); ok {
			strs := make([]string, 0, len(interfaces))
			for _, item := range interfaces {
				if str, ok := item.(string); ok {
					strs = append(strs, str)
				}
			}
			return Some(strs)
		}
	}
	return None[[]string]()
}

func (mc *MindscapeConnector) convertToSentinelCreateRequest(task MonitoringTask) sentinelDTO.CreateTaskRequest {
	// 将MonitoringTask转换为Mindscape Sentinel API格式
	// 这里需要根据实际的Mindscape API来实现转换逻辑
	parameters := map[string]any{
		"description": task.Description,
		"conditions":  task.Conditions,
		"target_data": task.TargetData,
		"enabled":     task.IsEnabled,
	}

	if task.WebhookURL.IsSome() {
		parameters["webhook_url"] = task.WebhookURL.Unwrap()
	}

	if task.MaxRetries.IsSome() {
		parameters["max_retries"] = task.MaxRetries.Unwrap()
	}

	// 转换任务类型
	taskType := mc.convertToSentinelTaskType(task.MindscapeTaskType)

	// 设置通知方法
	notificationMethods := []tasktypes.NotificationMethod{}
	for _, method := range task.NotificationMethods {
		if method == "webhook" {
			notificationMethods = append(notificationMethods, tasktypes.NotifyWebhook)
		} else if method == "mq" {
			notificationMethods = append(notificationMethods, tasktypes.NotifyMQ)
		}
	}

	return sentinelDTO.CreateTaskRequest{
		Type:                taskType,
		Parameters:          parameters,
		NotificationMethods: notificationMethods,
		WebhookURL:          task.WebhookURL,
		MQTopic:             task.MQTopic,
		MaxRetries:          task.MaxRetries,
	}
}

func (mc *MindscapeConnector) convertToSentinelTaskType(taskTypeStr string) tasktypes.TaskType {
	switch taskTypeStr {
	case "crypto":
		return tasktypes.TaskTypeCrypto
	case "twitter":
		return tasktypes.TaskTypeTwitter
	case "web":
		return tasktypes.TaskTypeWeb
	case "rss":
		return tasktypes.TaskTypeRSS
	default:
		return tasktypes.TaskTypeWeb // 默认使用web类型
	}
}

func parseUUID(str string) (uuid.UUID, error) {
	return uuid.Parse(str)
}

func generateOperationID() string {
	// 生成操作ID，实际应该使用UUID库
	return fmt.Sprintf("op-%d", time.Now().UnixNano())
}

func (mc *MindscapeConnector) convertMemoryType(memoryType memoryCore.MemoryType) string {
	switch memoryType {
	case memoryCore.EpisodicInteraction:
		return MemoryTypeObservation
	case memoryCore.UserPreference:
		return MemoryTypeUserPref
	case memoryCore.UserTrait:
		return MemoryTypeUserProfile
	default:
		return MemoryTypeTaskSummary // 使用已存在的常量替代SemanticFact
	}
}

func (mc *MindscapeConnector) extractUserIDFromAssociation(association *memoryCore.UserAssociation) string {
	if association == nil {
		return ""
	}
	return association.UserID
}

// createUserAssociation 创建用户关联信息
func (mc *MindscapeConnector) createUserAssociation(userID string) Option[memoryCore.UserAssociation] {
	if userID == "" {
		return None[memoryCore.UserAssociation]()
	}

	association := memoryCore.UserAssociation{
		UserID:               userID,
		RelevanceToUserScore: 1.0, // 默认相关性评分
	}

	return Some(association)
}

// createWebhookWakeupListener 创建Webhook唤醒监听器的简单实现
func (mc *MindscapeConnector) createWebhookWakeupListener(port int, path string, logger *slog.Logger) WakeupListener {
	// 简单实现，实际需要一个完整的HTTP服务器监听器
	return &webhookWakeupListener{
		port:    port,
		path:    path,
		logger:  logger,
		handler: nil,
		running: false,
	}
}

// 工具函数
func extractStringOption(data map[string]any, key string) Option[string] {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return Some(str)
		}
	}
	return None[string]()
}

func extractStringMapOption(data map[string]any, key string) Option[map[string]string] {
	if val, ok := data[key]; ok {
		if m, ok := val.(map[string]string); ok {
			return Some(m)
		}
		if m, ok := val.(map[string]any); ok {
			result := make(map[string]string)
			for k, v := range m {
				if s, ok := v.(string); ok {
					result[k] = s
				}
			}
			return Some(result)
		}
	}
	return None[map[string]string]()
}

func extractStringSliceOption(data map[string]any, key string) Option[[]string] {
	if val, ok := data[key]; ok {
		if slice, ok := val.([]string); ok {
			return Some(slice)
		}
		if slice, ok := val.([]any); ok {
			result := make([]string, 0, len(slice))
			for _, item := range slice {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return Some(result)
		}
	}
	return None[[]string]()
}
