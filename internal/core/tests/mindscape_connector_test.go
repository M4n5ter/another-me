package tests

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/m4n5ter/another-me/internal/core"
	. "github.com/m4n5ter/another-me/pkg/option"
)

func TestDefaultMindscapeConnectorConfig(t *testing.T) {
	config := core.DefaultMindscapeConnectorConfig()

	assert.Equal(t, "localhost", config.Host)
	assert.Equal(t, 8080, config.Port)
	assert.False(t, config.TLS)
	assert.Equal(t, 30*time.Second, config.HealthCheckInterval)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, 5*time.Second, config.RetryDelay)
	assert.Equal(t, 1000, config.QueueMaxSize)
	assert.Equal(t, 8081, config.WebhookListenPort)
	assert.Equal(t, "/wakeup", config.WebhookListenPath)
}

func TestMindscapeConnectorConfig(t *testing.T) {
	config := core.MindscapeConnectorConfig{
		Host:                "example.com",
		Port:                9090,
		TLS:                 true,
		HealthCheckInterval: 60 * time.Second,
		RetryAttempts:       5,
		RetryDelay:          10 * time.Second,
		QueueMaxSize:        500,
		WebhookListenPort:   8082,
		WebhookListenPath:   "/webhook",
	}

	assert.Equal(t, "example.com", config.Host)
	assert.Equal(t, 9090, config.Port)
	assert.True(t, config.TLS)
	assert.Equal(t, 60*time.Second, config.HealthCheckInterval)
	assert.Equal(t, 5, config.RetryAttempts)
	assert.Equal(t, 10*time.Second, config.RetryDelay)
	assert.Equal(t, 500, config.QueueMaxSize)
	assert.Equal(t, 8082, config.WebhookListenPort)
	assert.Equal(t, "/webhook", config.WebhookListenPath)
}

func TestQueuedOperation(t *testing.T) {
	now := time.Now()
	op := core.QueuedOperation{
		Type:      "store_memory",
		Data:      map[string]any{"test": "data"},
		Timestamp: now,
		Retries:   0,
		ID:        "test-op-001",
	}

	assert.Equal(t, "store_memory", op.Type)
	assert.Equal(t, map[string]any{"test": "data"}, op.Data)
	assert.Equal(t, now, op.Timestamp)
	assert.Equal(t, 0, op.Retries)
	assert.Equal(t, "test-op-001", op.ID)
}

func TestMindscapeConnectorMemoryItem(t *testing.T) {
	now := time.Now()
	memoryItem := core.MemoryItem{
		ID:         "memory-test-001",
		Timestamp:  now,
		Type:       core.MemoryTypeObservation,
		Content:    "测试记忆内容",
		Keywords:   []string{"测试", "记忆"},
		Importance: 0.8,
		RelatedIDs: []string{"memory-002"},
		UserID:     "user-001",
		Metadata:   map[string]any{"source": "test"},
	}
	
	assert.Equal(t, "memory-test-001", memoryItem.ID)
	assert.Equal(t, now, memoryItem.Timestamp)
	assert.Equal(t, core.MemoryTypeObservation, memoryItem.Type)
	assert.Equal(t, "测试记忆内容", memoryItem.Content)
	assert.Contains(t, memoryItem.Keywords, "测试")
	assert.Equal(t, 0.8, memoryItem.Importance)
	assert.Equal(t, "user-001", memoryItem.UserID)
}

func TestMonitoringTaskWithWebhook(t *testing.T) {
	now := time.Now()
	task := core.MonitoringTask{
		ID:                  Some("task-001"),
		Description:         "测试监控任务",
		MindscapeTaskType:   "web",
		Conditions:          []core.MonitorCondition{},
		TargetData:          []string{"url", "status"},
		NotificationMethods: []string{"webhook"},
		WebhookURL:          Some("http://localhost:8081/wakeup"),
		MQTopic:             None[string](),
		MaxRetries:          Some(3),
		IsEnabled:           true,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	
	assert.True(t, task.ID.IsSome())
	assert.Equal(t, "task-001", task.ID.Unwrap())
	assert.Equal(t, "测试监控任务", task.Description)
	assert.Equal(t, "web", task.MindscapeTaskType)
	assert.Contains(t, task.NotificationMethods, "webhook")
	assert.True(t, task.WebhookURL.IsSome())
	assert.Equal(t, "http://localhost:8081/wakeup", task.WebhookURL.Unwrap())
	assert.True(t, task.IsEnabled)
}

func TestMindscapeConnectorWakeupEvent(t *testing.T) {
	triggerTime := time.Now()
	event := core.WakeupEvent{
		MonitoringTaskID: "task-001",
		TriggerTime:      triggerTime,
		ObservedData: map[string]any{
			"url":    "https://example.com",
			"status": 200,
		},
		Reason: "网站状态检查",
		Metadata: map[string]any{
			"response_time": 150,
		},
	}
	
	assert.Equal(t, "task-001", event.MonitoringTaskID)
	assert.Equal(t, triggerTime, event.TriggerTime)
	assert.Equal(t, "https://example.com", event.ObservedData["url"])
	assert.Equal(t, 200, event.ObservedData["status"])
	assert.Equal(t, "网站状态检查", event.Reason)
	assert.Equal(t, 150, event.Metadata["response_time"])
}

// 这里暂时不测试NewMindscapeConnector，因为它需要实际的Mindscape服务
// 在集成测试中会进行更完整的测试
