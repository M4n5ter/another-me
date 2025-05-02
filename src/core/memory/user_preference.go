package memory

import (
	"time"

	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// 用户偏好/习惯/兴趣/偏好等
type UserPreference struct {
	ID                      models.RecordID
	Type                    string    // 偏好类型
	Description             string    // 偏好的文字描述(如 "对 AI 领域的创业公司新闻感兴趣", "倾向于早上浏览技术博客", "沟通风格简洁直接")
	Confidence              float64   // Agent 对该偏好判断的置信度0-1)
	LastUpdated             time.Time // 最后更新时间
	SourceDescription       string    // 推断出此偏好的简要原因或来源
	UserPreferenceEmbedding []float64 // 用户偏好的向量表示
}

// 用户偏好类型
const (
	UserPreferenceTypeInterest           = "INTEREST"            // 兴趣
	UserPreferenceTypeHabit              = "HABIT"               // 习惯
	UserPreferenceTypeCommunicationStyle = "COMMUNICATION_STYLE" // 沟通风格
	UserPreferenceTypePersonalityTrait   = "PERSONALITY_TRAIT"   // 性格特质
	UserPreferenceTypeValue              = "VALUE"               // 价值观
)
