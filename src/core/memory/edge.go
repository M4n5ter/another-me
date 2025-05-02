package memory

import "time"

// 记忆边，代表记忆之间的核心关系

// MemoryUnit -> MemoryUnit
// 连接按时间顺序发生的记忆单元。
// 用途: 重建行为序列，理解因果关系
type MemoryEdgeSequenced struct {
	TimeDiff time.Duration // 时间间隔
}

// MemoryUnit -> MemoryEntity
// 记忆单元提到了某个实体。
// 用途: 基于实体/主题检索相关记忆
type MemoryEdgeMentions struct {
	Relevance float64 // 实体在记忆中的相关度
}

// MemoryUnit -> MemoryUnit
// 两个记忆单元在语义上相似（通过向量搜索发现，或 LLM 判断）
// 用途: 关联相似情境或内容的记忆。
type MemoryEdgeRelatedSemantically struct {
	SimilarityScore float64 // 两个记忆单元的语义相似度
}

// MemoryUnit -> UserPreference
// 某个记忆单元是推断出某个用户偏好的证据
// 用途: 追溯偏好来源，根据新证据更新置信度
type MemoryEdgeSupports struct {
	Weight float64 // 该证据的权重
}

// MemoryUnit -> UserPreference
// 某个记忆单元与某个用户偏好相矛盾
// 用途: 修正或降低偏好置信度
type MemoryEdgeContradicts struct {
	Weight float64
}

// MemoryUnit -> MemoryUnit where target.type == 'SUMMARY'
// 一组原始记忆被总结成一个更高层次的摘要记忆
// 用途: 用途: 实现记忆的分层和压缩
type MemoryEdgeAbstractedInto struct{}

// UserPreference -> UserPreference
// 一个更具体或高阶的偏好是从另一个偏好推导出来的
// 用途: 构建偏好之间的层级关系
type MemoryEdgeDerivedFrom struct{}
