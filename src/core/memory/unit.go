package memory

import (
	"time"

	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

// MemoryUnit 记忆单元
type MemoryUnit struct {
	ID                models.RecordID
	Timestamp         time.Time     // 记忆发生或记录的时间
	Type              string        // 记忆的类型
	Source            string        // 记忆的来源
	ContentText       string        // 记忆的文本内容: 比如网页摘要、聊天记录、用户指令、Agent 思考过程、动作描述等
	ContentStructured surrealdb.Obj // 结构化内容: 比如点击坐标、API 调用参数、表单数据等
	ContentEmbedding  []float32     // ContentText 或者其它关键信息的向量嵌入，用于语义检索
	Importance        float64       // 记忆的重要性评分(0-1)，可基于来源，用户反馈，访问频率、情感强度等动态调整
	Metadata          surrealdb.Obj // 元数据，比如 URL，应用名，对话伙伴 ID，文件路径等各种灵活的元数据
}

// 记忆类型
const (
	MemoryUnitTypeObservation        = "OBSERVATION"         // 观察
	MemoryUnitTypeAction             = "ACTION"              // 行动
	MemoryUnitTypeCommunication      = "COMMUNICATION"       // 沟通
	MemoryUnitTypeUserFeedback       = "USER_FEEDBACK"       // 用户反馈
	MemoryUnitTypeInferredPreference = "INFERRED_PREFERENCE" // 推断偏好
	MemoryUnitTypeInferredKnowledge  = "INFERRED_KNOWLEDGE"  // 推断知识
	MemoryUnitTypeAgentThought       = "AGENT_THOUGHT"       // 思考
	MemoryUnitTypeGoal               = "GOAL"                // 目标
	MemoryUnitTypeSummary            = "SUMMARY"             // 总结
)

// 记忆来源
const (
	MemoryUnitSourceBrowserPlugin = "BROWSER_PLUGIN" // 浏览器插件
	MemoryUnitSourceChatAPI       = "CHAT_API"       // 聊天API
	MemoryUnitSourceUserInput     = "USER_INPUT"     // 用户输入
	MemoryUnitSourceAgentReflect  = "AGENT_REFLECT"  // Agent 反思
	MemoryUnitSourceScheduler     = "SCHEDULER"      // 调度器
)
