package memory

import "github.com/surrealdb/surrealdb.go/pkg/models"

// 记忆实体，代表记忆中涉及到的具体事物
type MemoryEntity struct {
	ID              models.RecordID
	Name            string    // 实体的常用名称
	Type            string    // 实体的类型
	Description     string    // 实体的简要描述
	EntityEmbedding []float32 // 实体名称和描述的向量嵌入
}

// 记忆实体类型
const (
	MemoryEntityTypePerson       = "PERSON"       // 人
	MemoryEntityTypeTopic        = "TOPIC"        // 话题
	MemoryEntityTypeWebsite      = "WEBSITE"      // 网站
	MemoryEntityTypeOrganization = "ORGANIZATION" // 组织
	MemoryEntityTypeProduct      = "PRODUCT"      // 产品
	MemoryEntityTypeLocation     = "LOCATION"     // 地点
)
