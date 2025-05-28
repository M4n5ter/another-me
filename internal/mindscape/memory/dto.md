# Memory DTO

Option 均可使用 `github.com/m4n5ter/another-me/pkg/option` 中的 `Option` 类型，它们是等价的。

```go
package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/m4n5ter/mindscape/memory/core"
	. "github.com/m4n5ter/mindscape/pkg/option"
)

// StoreMemoryRequest 存储记忆的请求
type StoreMemoryRequest struct {
	Type             core.MemoryType               `json:"type" binding:"required"`        // 记忆类型
	Source           Option[string]                `json:"source"`                         // 来源
	ContentRaw       string                        `json:"content_raw" binding:"required"` // 原始记忆内容
	ContentProcessed Option[core.ProcessedContent] `json:"content_processed"`              // 处理后的记忆内容
	ImportanceScore  Option[float64]               `json:"importance_score"`               // 重要性评分
	ConfidenceScore  Option[float64]               `json:"confidence_score"`               // 置信度评分
	ContextTags      Option[[]string]              `json:"context_tags"`                   // 上下文标签
	UserAssociation  Option[core.UserAssociation]  `json:"user_association"`               // 用户关联信息
	Metadata         Option[map[string]any]        `json:"metadata"`                       // 其他任意元数据
}

// MemoryFragmentResponse 记忆片段响应
type MemoryFragmentResponse struct {
	ID                    uuid.UUID             `json:"id"`                             // UUID，唯一标识符
	TimestampCreated      time.Time             `json:"timestamp_created"`              // 创建时间戳
	TimestampLastAccessed time.Time             `json:"timestamp_last_accessed"`        // 最后访问时间戳
	Type                  core.MemoryType       `json:"type"`                           // 记忆类型
	Source                string                `json:"source,omitempty"`               // 来源
	ContentRaw            string                `json:"content_raw"`                    // 原始记忆内容
	ContentProcessed      core.ProcessedContent `json:"content_processed"`              // 处理后的记忆内容
	ImportanceScore       float64               `json:"importance_score,omitempty"`     // 重要性评分
	ConfidenceScore       float64               `json:"confidence_score,omitempty"`     // 置信度评分
	AccessFrequency       int                   `json:"access_frequency,omitempty"`     // 访问频率
	ContextTags           []string              `json:"context_tags,omitempty"`         // 上下文标签
	RelatedFragmentIDs    []string              `json:"related_fragment_ids,omitempty"` // 相关记忆片段的ID
	UserAssociation       *core.UserAssociation `json:"user_association,omitempty"`     // 用户关联信息
	Metadata              map[string]any        `json:"metadata,omitempty"`             // 其他任意元数据
}

// RecallMemoriesRequest 回忆记忆的请求
type RecallMemoriesRequest struct {
	QueryText string  `json:"query_text" binding:"required"` // 查询文本
	UserID    string  `json:"user_id,omitempty"`             // 用户ID（可选）
	TopK      int     `json:"top_k,omitempty"`               // 返回数量限制，默认为10
	MinScore  float64 `json:"min_score,omitempty"`           // 最小相似度分数，默认为0.0
}

// RecallMemoriesResponse 回忆记忆的响应
type RecallMemoriesResponse struct {
	Fragments []*MemoryFragmentResponse `json:"fragments"` // 检索到的记忆片段列表
	Total     int                       `json:"total"`     // 总数
}

// UpdateUserProfileRequest 更新用户画像的请求
type UpdateUserProfileRequest struct {
	Username         Option[string]               `json:"username"`          // 用户名
	Preferences      Option[map[string]string]    `json:"preferences"`       // 偏好设置
	Habits           Option[[]string]             `json:"habits"`            // 用户习惯
	KnowledgeAreas   Option[[]string]             `json:"knowledge_areas"`   // 知识领域
	CustomEmbeddings Option[map[string][]float32] `json:"custom_embeddings"` // 用户特定的概念嵌入
}

// UserProfileResponse 用户画像响应
type UserProfileResponse struct {
	ID               string               `json:"id"`                          // 用户ID
	Username         string               `json:"username,omitempty"`          // 用户名
	Preferences      map[string]string    `json:"preferences,omitempty"`       // 偏好设置
	Habits           []string             `json:"habits,omitempty"`            // 用户习惯
	KnowledgeAreas   []string             `json:"knowledge_areas,omitempty"`   // 知识领域
	LastInteraction  time.Time            `json:"last_interaction"`            // 最后交互时间
	CustomEmbeddings map[string][]float32 `json:"custom_embeddings,omitempty"` // 用户特定的概念嵌入
}

// MemoryFragmentToResponse 将MemoryFragment转换为响应DTO
func MemoryFragmentToResponse(fragment *core.MemoryFragment) *MemoryFragmentResponse {
	if fragment == nil {
		return nil
	}
	return &MemoryFragmentResponse{
		ID:                    fragment.ID,
		TimestampCreated:      fragment.TimestampCreated,
		TimestampLastAccessed: fragment.TimestampLastAccessed,
		Type:                  fragment.Type,
		Source:                fragment.Source,
		ContentRaw:            fragment.ContentRaw,
		ContentProcessed:      fragment.ContentProcessed,
		ImportanceScore:       fragment.ImportanceScore,
		ConfidenceScore:       fragment.ConfidenceScore,
		AccessFrequency:       fragment.AccessFrequency,
		ContextTags:           fragment.ContextTags,
		RelatedFragmentIDs:    fragment.RelatedFragmentIDs,
		UserAssociation:       fragment.UserAssociation,
		Metadata:              fragment.Metadata,
	}
}

// MemoryFragmentsToResponse 将MemoryFragment列表转换为响应DTO列表
func MemoryFragmentsToResponse(fragments []*core.MemoryFragment) []*MemoryFragmentResponse {
	if fragments == nil {
		return nil
	}
	responses := make([]*MemoryFragmentResponse, len(fragments))
	for i, fragment := range fragments {
		responses[i] = MemoryFragmentToResponse(fragment)
	}
	return responses
}

// UserProfileToResponse 将UserProfile转换为响应DTO
func UserProfileToResponse(profile *core.UserProfile) *UserProfileResponse {
	if profile == nil {
		return nil
	}
	return &UserProfileResponse{
		ID:               profile.ID,
		Username:         profile.Username,
		Preferences:      profile.Preferences,
		Habits:           profile.Habits,
		KnowledgeAreas:   profile.KnowledgeAreas,
		LastInteraction:  profile.LastInteraction,
		CustomEmbeddings: profile.CustomEmbeddings,
	}
}

// GenericErrorResponse 通用错误响应
type GenericErrorResponse struct {
	Error string `json:"error"`
}

// ValidationErrorDetail 验证错误详情
type ValidationErrorDetail struct {
	Field   string `json:"field"`   // 发生错误的字段
	Message string `json:"message"` // 错误信息
}

// ValidationErrorResponse 验证错误响应
type ValidationErrorResponse struct {
	Errors []ValidationErrorDetail `json:"errors"`
}

// GetMemoryFragmentResponse 获取单个记忆片段的响应
type GetMemoryFragmentResponse struct {
	Fragment *MemoryFragmentResponse `json:"fragment"` // 记忆片段
}

// ListMemoryFragmentsRequest 列出记忆片段的请求查询参数
type ListMemoryFragmentsRequest struct {
	Page   int    `form:"page,default=1" binding:"min=1"`           // 页码，默认为1
	Limit  int    `form:"limit,default=10" binding:"min=1,max=100"` // 每页数量，默认为10，最大100
	UserID string `form:"user_id,omitempty"`                        // 用户ID（可选）
	Type   string `form:"type,omitempty"`                           // 记忆类型（可选）
}

// ListMemoryFragmentsResponse 列出记忆片段的响应
type ListMemoryFragmentsResponse struct {
	Fragments []*MemoryFragmentResponse `json:"fragments"` // 记忆片段列表
	Total     int                       `json:"total"`     // 总数
	Page      int                       `json:"page"`      // 当前页码
	Limit     int                       `json:"limit"`     // 每页数量
	HasMore   bool                      `json:"has_more"`  // 是否有更多数据
}

```

# core types

```go
package core

import (
	"time"

	"github.com/google/uuid"
)

// MemoryType 定义了记忆的类型
type MemoryType string

// String 方法使得 MemoryType 可以被 fmt 包和日志库正确打印
func (mt MemoryType) String() string {
	return string(mt)
}

const (
	// EpisodicInteraction 一段对话交互
	EpisodicInteraction MemoryType = "EPISODIC_INTERACTION"
	// SemanticFact 一个事实或知识点
	SemanticFact MemoryType = "SEMANTIC_FACT"
	// UserPreference 用户的偏好
	UserPreference MemoryType = "USER_PREFERENCE"
	// UserTrait 用户的某个特征或习惯
	UserTrait MemoryType = "USER_TRAIT"
	// AgentSkill 智能体学会的某个技能或程序
	AgentSkill MemoryType = "AGENT_SKILL"
	// EmotionalMarker 与某记忆相关的情感标记
	EmotionalMarker MemoryType = "EMOTIONAL_MARKER"
)

// ProcessedContent 存储处理后的记忆内容
type ProcessedContent struct {
	Summary   string   `json:"summary,omitempty"`   // 摘要
	Entities  []Entity `json:"entities,omitempty"`  // 提取的实体
	Intent    string   `json:"intent,omitempty"`    // 意图
	Sentiment string   `json:"sentiment,omitempty"` // 情感，可以是简单字符串或更复杂的对象
}

// Entity 代表从内容中提取的实体
type Entity struct {
	Text string `json:"text"`            // 实体文本
	Type string `json:"type"`            // 实体类型
	KgID string `json:"kg_id,omitempty"` // 知识图谱ID
}

// UserAssociation 存储与用户相关的记忆信息
type UserAssociation struct {
	UserID               string  `json:"user_id"`                           // 用户ID
	RelevanceToUserScore float64 `json:"relevance_to_user_score,omitempty"` // 对该用户的相关性评分
}

// MemoryFragment 是记忆的基本单元
type MemoryFragment struct {
	ID                    uuid.UUID        `json:"id"`                             // UUID，唯一标识符
	TimestampCreated      time.Time        `json:"timestamp_created"`              // 创建时间戳
	TimestampLastAccessed time.Time        `json:"timestamp_last_accessed"`        // 最后访问时间戳
	Type                  MemoryType       `json:"type"`                           // 记忆类型
	Source                string           `json:"source,omitempty"`               // 来源，例如："user_utterance", "agent_inference"
	ContentRaw            string           `json:"content_raw"`                    // 原始记忆内容
	ContentProcessed      ProcessedContent `json:"content_processed"`              // 处理后的记忆内容
	EmbeddingVector       []float32        `json:"embedding_vector,omitempty"`     // 向量嵌入
	ImportanceScore       float64          `json:"importance_score,omitempty"`     // 重要性评分 (0.0-1.0)
	ConfidenceScore       float64          `json:"confidence_score,omitempty"`     // 置信度评分 (0.0-1.0)
	AccessFrequency       int              `json:"access_frequency,omitempty"`     // 访问频率
	ContextTags           []string         `json:"context_tags,omitempty"`         // 上下文标签
	RelatedFragmentIDs    []string         `json:"related_fragment_ids,omitempty"` // 相关记忆片段的ID
	UserAssociation       *UserAssociation `json:"user_association,omitempty"`     // 用户关联信息
	Metadata              map[string]any   `json:"metadata,omitempty"`             // 其他任意元数据
}

// NewMemoryFragment 是 MemoryFragment 的构造函数
func NewMemoryFragment(id uuid.UUID, contentRaw string, memType MemoryType) *MemoryFragment {
	return &MemoryFragment{
		ID:                    id,
		TimestampCreated:      time.Now(),
		TimestampLastAccessed: time.Now(),
		Type:                  memType,
		ContentRaw:            contentRaw,
		AccessFrequency:       1,
		ImportanceScore:       0.5, // 默认重要性
		ConfidenceScore:       1.0, // 默认置信度
	}
}

// UserProfile 代表用户的画像信息
type UserProfile struct {
	ID               string               `json:"id"`                          // 用户ID
	Username         string               `json:"username,omitempty"`          // 用户名
	Preferences      map[string]string    `json:"preferences,omitempty"`       // 偏好设置，例如：{"theme": "dark", "language": "en"}
	Habits           []string             `json:"habits,omitempty"`            // 用户习惯，例如：["经常询问天气"]
	KnowledgeAreas   []string             `json:"knowledge_areas,omitempty"`   // 知识领域，例如：["golang", "kubernetes"]
	LastInteraction  time.Time            `json:"last_interaction"`            // 最后交互时间
	CustomEmbeddings map[string][]float32 `json:"custom_embeddings,omitempty"` // 用户特定的概念嵌入
	// Add other relevant fields for user modeling // 添加其他相关的用户建模字段
}

```