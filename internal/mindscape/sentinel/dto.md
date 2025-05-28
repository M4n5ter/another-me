# Sentinel DTO

Option 均可使用 `github.com/m4n5ter/another-me/pkg/option` 中的 `Option` 类型，它们是等价的。

```go
package dto

import (
	"time"

	"github.com/google/uuid"

	. "github.com/m4n5ter/mindscape/pkg/option"
	"github.com/m4n5ter/mindscape/sentinel/core/tasktypes"
)

// CreateTaskRequest 定义了创建新任务请求的结构。
type CreateTaskRequest struct {
	Type                tasktypes.TaskType             `json:"type" binding:"required"`                       // 例如："crypto", "twitter"
	Parameters          tasktypes.TaskParameters       `json:"parameters" binding:"required"`                 // 例如：{"condition": "price_gt", "target_value": 10000}
	NotificationMethods []tasktypes.NotificationMethod `json:"notification_methods" binding:"required,min=1"` // 例如：["webhook", "mq"]
	WebhookURL          Option[string]                 `json:"webhook_url,omitempty"`                         // 如果 notification_methods 中包含 "webhook"，则此字段为必需
	MQTopic             Option[string]                 `json:"mq_topic,omitempty"`                            // 可选，如果 notification_methods 中包含 "mq"，可以自动生成或指定
	MaxRetries          Option[int]                    `json:"max_retries,omitempty"`                         // 可选，默认为系统设定值 (例如：3)
}

// TaskResponse 定义了 API 返回任务时的结构。
type TaskResponse struct {
	ID                  uuid.UUID                      `json:"id"`
	Type                tasktypes.TaskType             `json:"type"`
	Status              tasktypes.TaskStatus           `json:"status"`
	Parameters          tasktypes.TaskParameters       `json:"parameters"`
	NotificationMethods []tasktypes.NotificationMethod `json:"notification_methods"`
	WebhookURL          Option[string]                 `json:"webhook_url,omitempty"`
	MQTopic             Option[string]                 `json:"mq_topic,omitempty"`
	RetryCount          int                            `json:"retry_count"`
	MaxRetries          int                            `json:"max_retries"`
	LastAttemptAt       Option[time.Time]              `json:"last_attempt_at,omitempty"`
	TriggeredAt         Option[time.Time]              `json:"triggered_at,omitempty"`
	CreatedAt           time.Time                      `json:"created_at"`
	UpdatedAt           time.Time                      `json:"updated_at"`
}

// TaskToResponse 将 tasktypes.Task 模型转换为 TaskResponse DTO。
func TaskToResponse(t *tasktypes.Task) *TaskResponse {
	if t == nil {
		return nil
	}
	return &TaskResponse{
		ID:                  t.ID,
		Type:                t.Type,
		Status:              t.Status,
		Parameters:          t.Parameters,
		NotificationMethods: t.NotificationMethods,
		WebhookURL:          t.WebhookURL,
		MQTopic:             t.MQTopic,
		RetryCount:          t.RetryCount,
		MaxRetries:          t.MaxRetries,
		LastAttemptAt:       t.LastAttemptAt,
		TriggeredAt:         t.TriggeredAt,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
	}
}

// TasksToResponse 将 tasktypes.Task 模型切片转换为 TaskResponse DTO 切片。
func TasksToResponse(tasks []*tasktypes.Task) []*TaskResponse {
	responses := make([]*TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		responses = append(responses, TaskToResponse(t))
	}
	return responses
}

// GenericErrorResponse 是用于错误响应的通用结构。
type GenericErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrorDetail 提供有关验证错误的更具体信息。
type ValidationErrorDetail struct {
	Field   string `json:"field"`   // 发生错误的字段
	Message string `json:"message"` // 错误信息
}

// ValidationErrorResponse 用于验证错误，通常来自请求绑定。
type ValidationErrorResponse struct {
	Errors []ValidationErrorDetail `json:"errors"`
}

```

# tasktypes

```go
package tasktypes

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	. "github.com/m4n5ter/mindscape/pkg/option"
)

// TaskType 表示任务的类型。
type TaskType string

const (
	// TaskTypeCrypto 表示加密货币监控任务。
	TaskTypeCrypto TaskType = "crypto"
	// TaskTypeEquity 表示股票监控任务。
	TaskTypeEquity TaskType = "equity"
	// TaskTypeMetal 表示金属监控任务。
	TaskTypeMetal TaskType = "metal"
	// TaskTypeRates 表示利率监控任务。
	TaskTypeRates TaskType = "rates"
	// TaskTypeCryptoRedemptionRate 表示加密货币赎回率监控任务。
	TaskTypeCryptoRedemptionRate TaskType = "crypto_redemption_rate"
	// TaskTypeFX 表示外汇监控任务。
	TaskTypeFX TaskType = "fx"
	// TaskTypeTwitter 表示 Twitter 监控任务。
	TaskTypeTwitter TaskType = "twitter"
	// TaskTypeWeb 表示通用网页内容监控任务。
	TaskTypeWeb TaskType = "web"
	// TaskTypeRSS 表示 RSS 监控任务。
	TaskTypeRSS TaskType = "rss"
)

// TaskStatus 表示任务的状态。
type TaskStatus string

const (
	// StatusPending 表示任务已创建但尚未被主动监控。
	StatusPending TaskStatus = "pending"
	// StatusActive 表示任务正在被主动监控。
	StatusActive TaskStatus = "active"
	// StatusTriggered 表示任务的条件已满足，并已尝试发送通知。
	StatusTriggered TaskStatus = "triggered"
	// StatusFailed 表示任务在处理过程中遇到错误或通知尝试次数已用尽。
	StatusFailed TaskStatus = "failed"
	// StatusPaused 表示任务监控被用户/系统暂时暂停。
	StatusPaused TaskStatus = "paused"
	// StatusStopped 表示任务被用户/系统停止。
	StatusStopped TaskStatus = "stopped"
)

// NotificationMethod 表示通知方法。
type NotificationMethod string

const (
	// NotifyWebhook 表示通过 Webhook URL 进行通知。
	NotifyWebhook NotificationMethod = "webhook"
	// NotifyMQ 表示通过消息队列进行通知。
	NotifyMQ NotificationMethod = "mq"
)

// TaskParameters 定义任务的具体参数，这些参数可能因 TaskType 而异。
// 使用 map[string]any 以获得灵活性，可以在 PostgreSQL 中序列化为 JSONB。
type TaskParameters map[string]any

/*
TaskTypeCrypto 的 TaskParameters 示例:
{
  "source": "binance", // 或 "coinGecko" 等交易所或数据源
  "symbol": "BTCUSDT",   // 例如, 比特币/泰达币
  "condition": "price_gt", // price_gt (大于), price_lt (小于), price_eq (等于), price_change_percent_gt (百分比变化大于)
  "target_value": 70000.0, // 目标值
  "change_interval": "1h" // 可选: 用于 price_change_percent_gt, 例如 "1h", "24h"
}

TaskTypeTwitter 的 TaskParameters 示例:
{
  "user_id": "123456789", // Twitter 用户 ID
  "keywords": ["go", "golang", "#opensource"], // 可选: 在新推文中查找的关键词
  "match_type": "any" // 关键词匹配类型: "any" (任意一个) 或 "all" (所有)
}
*/

// Task 代表 Sentinel 系统中的一个监控任务。
// 这是任务的主要数据结构。
type Task struct {
	ID                  uuid.UUID            `json:"id" db:"id"` // 主键
	Type                TaskType             `json:"type" db:"type"`
	Status              TaskStatus           `json:"status" db:"status"`
	Parameters          TaskParameters       `json:"parameters" db:"parameters"`                     // 在数据库中存储为 JSONB
	NotificationMethods []NotificationMethod `json:"notification_methods" db:"notification_methods"` // 在数据库中存储为 TEXT ARRAY 或 JSONB
	WebhookURL          Option[string]       `json:"webhook_url,omitempty" db:"webhook_url"`         // 可空
	// MQTopic 是如果使用 NotifyMQ 时通知消息将发送到的主题。
	// 这个值可以派生出来或显式设置。
	MQTopic Option[string] `json:"mq_topic,omitempty" db:"mq_topic"` // 可空

	RetryCount    int               `json:"retry_count" db:"retry_count"`                   // 当前重试次数
	MaxRetries    int               `json:"max_retries" db:"max_retries"`                   // 最大通知重试次数
	LastAttemptAt Option[time.Time] `json:"last_attempt_at,omitempty" db:"last_attempt_at"` // 上次尝试时间，可空
	TriggeredAt   Option[time.Time] `json:"triggered_at,omitempty" db:"triggered_at"`       // 条件首次满足时间，可空

	CreatedAt time.Time `json:"created_at" db:"created_at"` // 创建时间
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"` // 更新时间

	// 如果将来任务需要与特定用户/租户关联，可以添加 UserID
	// UserID string `json:"user_id" db:"user_id"`
}

// NewTask 使用默认值创建一个新任务。
func NewTask(taskType TaskType, params TaskParameters, methods []NotificationMethod, webhookURL, mqTopic Option[string], maxRetries int) (*Task, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("创建任务ID失败: %w", err)
	}

	now := time.Now().UTC()
	if maxRetries <= 0 {
		maxRetries = 3
	}

	return &Task{
		ID:                  id,
		Type:                taskType,
		Status:              StatusPending,
		Parameters:          params,
		NotificationMethods: methods,
		WebhookURL:          webhookURL,
		MQTopic:             mqTopic,
		MaxRetries:          maxRetries,
		RetryCount:          0,
		CreatedAt:           now,
		UpdatedAt:           now,
	}, nil
}

// AnySliceToStringSlice 将 any 类型的切片转换为 string 类型的切片。
func AnySliceToStringSlice(input []any) []string {
	result := make([]string, len(input))
	for i, v := range input {
		switch val := v.(type) {
		case string:
			result[i] = val
		case fmt.Stringer:
			result[i] = val.String()
		default:
			result[i] = fmt.Sprintf("%v", v)
		}
	}
	return result
}

```