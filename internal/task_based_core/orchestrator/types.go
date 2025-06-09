package orchestrator

import (
	"time"

	"google.golang.org/genai"

	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	"github.com/m4n5ter/another-me/pkg/schema"
)

// TaskPlan 任务计划 - 用于内部规划阶段的简化任务定义
type TaskPlan struct {
	ID                   string         `json:"id"`
	Name                 string         `json:"name"`
	Type                 string         `json:"type"`
	Priority             state.Priority `json:"priority"`
	Description          string         `json:"description"`
	RequiredCapabilities []string       `json:"required_capabilities"`
	EstimatedDuration    time.Duration  `json:"estimated_duration"`
	Metadata             map[string]any `json:"metadata,omitempty"`

	// 依赖关系和并行执行支持
	DependsOn    []string `json:"depends_on,omitempty"` // 依赖的任务ID列表
	CanParallel  bool     `json:"can_parallel"`         // 是否可以并行执行
	SubTaskIndex int      `json:"sub_task_index"`       // 在原始子任务列表中的索引
}

// WorkerSelectionRequest Worker选择请求
type WorkerSelectionRequest struct {
	TaskInfo         TaskAnalysisRequest  `json:"task_info"`
	AnalysisResult   TaskAnalysisResponse `json:"analysis_result"`
	AvailableWorkers []AvailableWorker    `json:"available_workers"`
}

// TaskAnalysisRequest 任务分析请求
type TaskAnalysisRequest struct {
	TaskID      string         `json:"task_id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Priority    string         `json:"priority"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// TaskAnalysisResponse 任务分析响应
type TaskAnalysisResponse struct {
	// RequiresDecomposition 是否需要分解
	RequiresDecomposition bool `json:"requires_decomposition"`

	// ReasoningProcess 分析推理过程
	ReasoningProcess string `json:"reasoning_process"`

	// RecommendedAction 推荐的行动
	RecommendedAction string `json:"recommended_action"`

	// SubTasks 子任务列表（如果需要分解）
	SubTasks []SubTaskDefinition `json:"sub_tasks,omitempty"`

	// EstimatedComplexity 估计的复杂度 (1-10)
	EstimatedComplexity int `json:"estimated_complexity"`

	// EstimatedDuration 估计的执行时间（分钟）
	EstimatedDuration int `json:"estimated_duration_minutes"`

	// RequiredWorkerType 推荐的Worker类型
	RequiredWorkerType string `json:"required_worker_type"`

	// Prerequisites 前置条件
	Prerequisites []string `json:"prerequisites,omitempty"`

	// RiskAssessment 风险评估
	RiskAssessment RiskAssessment `json:"risk_assessment"`
}

// SubTaskDefinition 子任务定义
type SubTaskDefinition struct {
	// Name 子任务名称
	Name string `json:"name"`

	// Description 子任务描述
	Description string `json:"description"`

	// WorkerType 所需的Worker类型
	WorkerType string `json:"worker_type"`

	// Priority 优先级
	Priority string `json:"priority"`

	// DependsOn 依赖的其他子任务索引
	DependsOn []int `json:"depends_on,omitempty"`

	// EstimatedDuration 估计执行时间（分钟）
	EstimatedDuration int `json:"estimated_duration_minutes"`

	// Metadata 子任务元数据
	Metadata map[string]any `json:"metadata,omitempty"`
}

// RiskAssessment 风险评估
type RiskAssessment struct {
	// Level 风险级别 (low, medium, high, critical)
	Level string `json:"level"`

	// Factors 风险因子
	Factors []string `json:"factors,omitempty"`

	// Mitigation 缓解措施
	Mitigation []string `json:"mitigation,omitempty"`
}

// AvailableWorker 可用的Worker信息
type AvailableWorker struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	State        string   `json:"state"`
	Capabilities []string `json:"capabilities"`
}

// WorkerPerformanceMetrics Worker性能指标
type WorkerPerformanceMetrics struct {
	TasksCompleted  int     `json:"tasks_completed"`
	TasksFailed     int     `json:"tasks_failed"`
	SuccessRate     float64 `json:"success_rate"`
	AvgDurationMins float64 `json:"avg_duration_minutes"`
	LastErrorReason string  `json:"last_error_reason,omitempty"`
}

// WorkerSelectionResponse Worker选择响应
type WorkerSelectionResponse struct {
	// SelectedWorkerID 选择的Worker ID
	SelectedWorkerID string `json:"selected_worker_id"`

	// SelectionReason 选择原因
	SelectionReason string `json:"selection_reason"`

	// Confidence 置信度 (0-1)
	Confidence float64 `json:"confidence"`

	// AlternativeWorkers 备选Worker
	AlternativeWorkers []string `json:"alternative_workers,omitempty"`

	// ExpectedPerformance 期望性能
	ExpectedPerformance ExpectedPerformance `json:"expected_performance"`
}

// ExpectedPerformance 期望性能
type ExpectedPerformance struct {
	// EstimatedSuccessRate 估计成功率
	EstimatedSuccessRate float64 `json:"estimated_success_rate"`

	// EstimatedDuration 估计执行时间（分钟）
	EstimatedDuration int `json:"estimated_duration_minutes"`

	// QualityScore 质量评分 (1-10)
	QualityScore float64 `json:"quality_score"`
}

// TaskRequestEnrichmentRequest 任务请求丰富化请求
type TaskRequestEnrichmentRequest struct {
	UserInput       string         `json:"user_input"`
	Context         map[string]any `json:"context,omitempty"`
	PreviousTasks   []string       `json:"previous_tasks,omitempty"`
	AvailableAssets []string       `json:"available_assets,omitempty"`
}

// TaskRequestEnrichmentResponse 任务请求丰富化响应
type TaskRequestEnrichmentResponse struct {
	EnrichedDescription string         `json:"enriched_description"`
	IdentifiedGoals     []string       `json:"identified_goals"`
	MissingInformation  []string       `json:"missing_information,omitempty"`
	Assumptions         []string       `json:"assumptions,omitempty"`
	Scope               string         `json:"scope"`
	Constraints         []string       `json:"constraints,omitempty"`
	SuccessCriteria     []string       `json:"success_criteria"`
	Metadata            map[string]any `json:"metadata,omitempty"`
}

// WorkerTaskMappingRequest Worker任务映射请求
type WorkerTaskMappingRequest struct {
	EnrichmentResult TaskRequestEnrichmentResponse `json:"enrichment_result"`
	AnalysisResult   TaskAnalysisResponse          `json:"analysis_result"`
	AvailableWorkers []AvailableWorker             `json:"available_workers"`
}

// WorkerTaskMappingResponse Worker任务映射响应
type WorkerTaskMappingResponse struct {
	TaskAssignments     []TaskAssignment   `json:"task_assignments"`
	RequiredNewWorkers  []NewWorkerRequest `json:"required_new_workers,omitempty"`
	UnassignedTasks     []UnassignedTask   `json:"unassigned_tasks,omitempty"`
	MappingReasoning    string             `json:"mapping_reasoning"`
	OverallStrategy     string             `json:"overall_strategy"`
	EstimatedCompletion int                `json:"estimated_completion_minutes"`
}

// TaskAssignment 任务分配
type TaskAssignment struct {
	TaskID           string   `json:"task_id"`
	TaskName         string   `json:"task_name"`
	AssignedWorkerID string   `json:"assigned_worker_id"`
	WorkerType       string   `json:"worker_type"`
	Priority         string   `json:"priority"`
	Dependencies     []string `json:"dependencies,omitempty"`
	EstimatedTime    int      `json:"estimated_time_minutes"`
	AssignmentReason string   `json:"assignment_reason"`
}

// NewWorkerRequest 新Worker请求
type NewWorkerRequest struct {
	WorkerID             string         `json:"worker_id"`
	WorkerType           string         `json:"worker_type"`
	WorkerName           string         `json:"worker_name"`
	RequiredCapabilities []string       `json:"required_capabilities"`
	SystemPrompt         string         `json:"system_prompt"`
	TasksToHandle        []string       `json:"tasks_to_handle"`
	SpecialInstructions  string         `json:"special_instructions,omitempty"`
	EstimatedLifetime    int            `json:"estimated_lifetime_minutes"`
	Metadata             map[string]any `json:"metadata,omitempty"`
}

// UnassignedTask 未分配任务
type UnassignedTask struct {
	TaskID      string   `json:"task_id"`
	TaskName    string   `json:"task_name"`
	Reason      string   `json:"reason"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// 创建任务分析的JSON Schema
func CreateTaskAnalysisSchema() *schema.Schema {
	min1 := 1.0
	max10 := 10.0

	return &schema.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*schema.Schema{
			"requires_decomposition": {
				Type:        genai.TypeBoolean,
				Description: "任务是否需要分解为子任务",
			},
			"reasoning_process": {
				Type:        genai.TypeString,
				Description: "分析任务的推理过程",
			},
			"recommended_action": {
				Type:        genai.TypeString,
				Description: "推荐的行动方案",
			},
			"sub_tasks": {
				Type:        genai.TypeArray,
				Description: "子任务列表（如果需要分解）",
				Items: &schema.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*schema.Schema{
						"name": {
							Type:        genai.TypeString,
							Description: "子任务名称",
						},
						"description": {
							Type:        genai.TypeString,
							Description: "子任务详细描述",
						},
						"worker_type": {
							Type:        genai.TypeString,
							Description: "所需的Worker类型",
							Enum:        []string{"file_system", "web_ui", "data_analysis", "temporary"},
						},
						"priority": {
							Type:        genai.TypeString,
							Description: "任务优先级",
							Enum:        []string{"low", "normal", "high", "critical"},
						},
						"depends_on": {
							Type:        genai.TypeArray,
							Description: "依赖的其他子任务索引",
							Items:       &schema.Schema{Type: genai.TypeInteger},
						},
						"estimated_duration_minutes": {
							Type:        genai.TypeInteger,
							Description: "估计执行时间（分钟）",
							Minimum:     &min1,
						},
						"metadata": {
							Type:        genai.TypeObject,
							Description: "子任务元数据",
						},
					},
					Required: []string{"name", "description", "worker_type", "priority", "estimated_duration_minutes"},
				},
			},
			"estimated_complexity": {
				Type:        genai.TypeInteger,
				Description: "估计的复杂度等级 (1-10)",
				Minimum:     &min1,
				Maximum:     &max10,
			},
			"estimated_duration_minutes": {
				Type:        genai.TypeInteger,
				Description: "估计的总执行时间（分钟）",
				Minimum:     &min1,
			},
			"required_worker_type": {
				Type:        genai.TypeString,
				Description: "推荐的Worker类型",
				Enum:        []string{"file_system", "web_ui", "data_analysis", "temporary"},
			},
			"prerequisites": {
				Type:        genai.TypeArray,
				Description: "执行前置条件",
				Items:       &schema.Schema{Type: genai.TypeString},
			},
			"risk_assessment": {
				Type:        genai.TypeObject,
				Description: "风险评估",
				Properties: map[string]*schema.Schema{
					"level": {
						Type:        genai.TypeString,
						Description: "风险级别",
						Enum:        []string{"low", "medium", "high", "critical"},
					},
					"factors": {
						Type:        genai.TypeArray,
						Description: "风险因子",
						Items:       &schema.Schema{Type: genai.TypeString},
					},
					"mitigation": {
						Type:        genai.TypeArray,
						Description: "缓解措施",
						Items:       &schema.Schema{Type: genai.TypeString},
					},
				},
				Required: []string{"level"},
			},
		},
		Required: []string{
			"requires_decomposition",
			"reasoning_process",
			"recommended_action",
			"estimated_complexity",
			"estimated_duration_minutes",
			"required_worker_type",
			"risk_assessment",
		},
	}
}

// CreateWorkerSelectionSchema 创建Worker选择的JSON Schema
func CreateWorkerSelectionSchema() *schema.Schema {
	min0 := 0.0
	min1 := 1.0
	max1 := 1.0
	max10 := 10.0

	return &schema.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*schema.Schema{
			"selected_worker_id": {
				Type:        genai.TypeString,
				Description: "选择的Worker ID",
			},
			"selection_reason": {
				Type:        genai.TypeString,
				Description: "选择该Worker的详细原因",
			},
			"confidence": {
				Type:        genai.TypeNumber,
				Description: "选择的置信度 (0-1)",
				Minimum:     &min0,
				Maximum:     &max1,
			},
			"alternative_workers": {
				Type:        genai.TypeArray,
				Description: "备选Worker ID列表",
				Items:       &schema.Schema{Type: genai.TypeString},
			},
			"expected_performance": {
				Type:        genai.TypeObject,
				Description: "期望性能指标",
				Properties: map[string]*schema.Schema{
					"estimated_success_rate": {
						Type:        genai.TypeNumber,
						Description: "估计成功率 (0-1)",
						Minimum:     &min0,
						Maximum:     &max1,
					},
					"estimated_duration_minutes": {
						Type:        genai.TypeInteger,
						Description: "估计执行时间（分钟）",
						Minimum:     &min1,
					},
					"quality_score": {
						Type:        genai.TypeNumber,
						Description: "质量评分 (1-10)",
						Minimum:     &min1,
						Maximum:     &max10,
					},
				},
				Required: []string{"estimated_success_rate", "estimated_duration_minutes", "quality_score"},
			},
		},
		Required: []string{"selected_worker_id", "selection_reason", "confidence", "expected_performance"},
	}
}

// CreateTaskRequestEnrichmentSchema 创建任务请求丰富化的JSON Schema
func CreateTaskRequestEnrichmentSchema() *schema.Schema {
	return &schema.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*schema.Schema{
			"enriched_description": {
				Type:        genai.TypeString,
				Description: "丰富和详细化后的任务描述",
			},
			"identified_goals": {
				Type:        genai.TypeArray,
				Description: "识别出的具体目标列表",
				Items:       &schema.Schema{Type: genai.TypeString},
			},
			"missing_information": {
				Type:        genai.TypeArray,
				Description: "缺失的重要信息列表",
				Items:       &schema.Schema{Type: genai.TypeString},
			},
			"assumptions": {
				Type:        genai.TypeArray,
				Description: "基于上下文做出的合理假设",
				Items:       &schema.Schema{Type: genai.TypeString},
			},
			"scope": {
				Type:        genai.TypeString,
				Description: "任务范围的明确定义",
			},
			"constraints": {
				Type:        genai.TypeArray,
				Description: "识别出的约束条件",
				Items:       &schema.Schema{Type: genai.TypeString},
			},
			"success_criteria": {
				Type:        genai.TypeArray,
				Description: "成功标准列表",
				Items:       &schema.Schema{Type: genai.TypeString},
			},
			"metadata": {
				Type:        genai.TypeObject,
				Description: "其他相关元数据",
			},
		},
		Required: []string{
			"enriched_description",
			"identified_goals",
			"scope",
			"success_criteria",
		},
	}
}

// CreateWorkerTaskMappingSchema 创建Worker任务映射的JSON Schema
func CreateWorkerTaskMappingSchema() *schema.Schema {
	min1 := 1.0

	return &schema.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*schema.Schema{
			"task_assignments": {
				Type:        genai.TypeArray,
				Description: "任务分配列表",
				Items: &schema.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*schema.Schema{
						"task_id": {
							Type:        genai.TypeString,
							Description: "任务ID",
						},
						"task_name": {
							Type:        genai.TypeString,
							Description: "任务名称",
						},
						"assigned_worker_id": {
							Type:        genai.TypeString,
							Description: "分配的Worker ID。如果分配的是临时Worker，则需要使用临时Worker的ID",
						},
						"worker_type": {
							Type:        genai.TypeString,
							Description: "Worker类型",
						},
						"priority": {
							Type:        genai.TypeString,
							Description: "任务优先级",
							Enum:        []string{"low", "normal", "high", "critical"},
						},
						"dependencies": {
							Type:        genai.TypeArray,
							Description: "依赖的任务ID列表",
							Items:       &schema.Schema{Type: genai.TypeString},
						},
						"estimated_time_minutes": {
							Type:        genai.TypeInteger,
							Description: "估计执行时间（分钟）",
							Minimum:     &min1,
						},
						"assignment_reason": {
							Type:        genai.TypeString,
							Description: "分配原因",
						},
					},
					Required: []string{"task_id", "task_name", "assigned_worker_id", "worker_type", "priority", "estimated_time_minutes", "assignment_reason"},
				},
			},
			"required_new_workers": {
				Type:        genai.TypeArray,
				Description: "需要创建的新Worker列表",
				Items: &schema.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*schema.Schema{
						"worker_id": {
							Type:        genai.TypeString,
							Description: "Worker ID, 需要全局唯一",
							Example:     "temp_worker_1_{timestamp}",
						},
						"worker_type": {
							Type:        genai.TypeString,
							Description: "Worker类型",
						},
						"worker_name": {
							Type:        genai.TypeString,
							Description: "Worker名称",
						},
						"required_capabilities": {
							Type:        genai.TypeArray,
							Description: "所需能力列表",
							Items:       &schema.Schema{Type: genai.TypeString},
						},
						"system_prompt": {
							Type:        genai.TypeString,
							Description: "为此Worker精心设计的系统提示词",
						},
						"tasks_to_handle": {
							Type:        genai.TypeArray,
							Description: "此Worker要处理的任务ID列表",
							Items:       &schema.Schema{Type: genai.TypeString},
						},
						"special_instructions": {
							Type:        genai.TypeString,
							Description: "特殊指令",
						},
						"estimated_lifetime_minutes": {
							Type:        genai.TypeInteger,
							Description: "估计生命周期（分钟）",
							Minimum:     &min1,
						},
						"metadata": {
							Type:        genai.TypeObject,
							Description: "Worker元数据",
						},
					},
					Required: []string{"worker_type", "worker_name", "required_capabilities", "system_prompt", "tasks_to_handle", "estimated_lifetime_minutes"},
				},
			},
			"unassigned_tasks": {
				Type:        genai.TypeArray,
				Description: "无法分配的任务列表",
				Items: &schema.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*schema.Schema{
						"task_id": {
							Type:        genai.TypeString,
							Description: "任务ID",
						},
						"task_name": {
							Type:        genai.TypeString,
							Description: "任务名称",
						},
						"reason": {
							Type:        genai.TypeString,
							Description: "无法分配的原因",
						},
						"suggestions": {
							Type:        genai.TypeArray,
							Description: "解决建议",
							Items:       &schema.Schema{Type: genai.TypeString},
						},
					},
					Required: []string{"task_id", "task_name", "reason"},
				},
			},
			"mapping_reasoning": {
				Type:        genai.TypeString,
				Description: "任务映射的推理过程",
			},
			"overall_strategy": {
				Type:        genai.TypeString,
				Description: "整体执行策略",
			},
			"estimated_completion_minutes": {
				Type:        genai.TypeInteger,
				Description: "估计总完成时间（分钟）",
				Minimum:     &min1,
			},
		},
		Required: []string{
			"task_assignments",
			"mapping_reasoning",
			"overall_strategy",
			"estimated_completion_minutes",
		},
	}
}
