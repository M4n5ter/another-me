package orchestrator

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/internal/task_based_core/state"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
)

// SimpleOrchestrator 简化的编排器，专注展示ProduceJSON的使用
type SimpleOrchestrator struct {
	logger       *slog.Logger
	llm          llminterface.ChatAdapter
	stateManager state.StateManagerInterface
}

// NewSimpleOrchestrator 创建简化的编排器
func NewSimpleOrchestrator(
	llm llminterface.ChatAdapter,
	stateManager state.StateManagerInterface,
) *SimpleOrchestrator {
	return &SimpleOrchestrator{
		logger:       slog.Default().WithGroup("simple_orchestrator"),
		llm:          llm,
		stateManager: stateManager,
	}
}

// AnalyzeTaskWithLLM 使用LLM分析任务，展示ProduceJSON的正确用法
func (o *SimpleOrchestrator) AnalyzeTaskWithLLM(ctx context.Context, taskInfo *state.TaskInfo) (*TaskAnalysisResponse, error) {
	// 准备任务分析请求
	request := TaskAnalysisRequest{
		TaskID:      taskInfo.ID,
		Name:        taskInfo.Name,
		Description: taskInfo.Description,
		Priority:    taskInfo.Priority.String(),
		Metadata:    taskInfo.Metadata,
	}

	// 构建系统提示
	systemPrompt := `你是一个智能任务分析专家。请仔细分析用户提供的任务，并提供详细的分析结果。

任务分析指南：
1. 判断任务是否需要分解为多个子任务
2. 评估任务的复杂度（1-10级）
3. 估算执行时间
4. 推荐合适的Worker类型
5. 识别潜在风险并提供缓解措施

可用的Worker类型：
- file_system: 文件系统操作，如读写文件、目录管理等
- web_ui: Web界面操作，如浏览器自动化、网页交互等  
- data_analysis: 数据分析，如Excel处理、数据统计等
- temporary: 临时性工作，如文本处理、简单计算等

请基于任务内容提供结构化的分析结果。`

	// 构建用户提示
	requestJSON, err := json.MarshalToString(request)
	if err != nil {
		return nil, fmt.Errorf("序列化任务请求失败: %w", err)
	}

	userPrompt := fmt.Sprintf("请分析以下任务：\n%s", requestJSON)

	// 构建ChatInput
	chatInput := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleSystem,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: systemPrompt,
					},
				},
			},
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: userPrompt,
					},
				},
			},
		},
		ConversationID: fmt.Sprintf("task_analysis_%s", taskInfo.ID),
	}

	// 创建Schema
	schema := CreateTaskAnalysisSchema()

	// 使用ProduceJSON获取结构化响应
	result, err := o.llm.ProduceJSON(ctx, chatInput, Some(*schema))
	if err != nil {
		return nil, fmt.Errorf("LLM任务分析失败: %w", err)
	}

	// 解析响应
	var analysisResponse TaskAnalysisResponse
	if err := json.UnmarshalFromString(result, &analysisResponse); err != nil {
		return nil, fmt.Errorf("解析任务分析响应失败: %w", err)
	}

	o.logger.Info("任务分析完成",
		"task_id", taskInfo.ID,
		"requires_decomposition", analysisResponse.RequiresDecomposition,
		"complexity", analysisResponse.EstimatedComplexity,
		"estimated_duration", analysisResponse.EstimatedDuration,
		"required_worker_type", analysisResponse.RequiredWorkerType,
	)

	return &analysisResponse, nil
}

// SelectWorkerWithLLM 使用LLM选择Worker，展示复杂结构化数据的处理
func (o *SimpleOrchestrator) SelectWorkerWithLLM(ctx context.Context, taskInfo *state.TaskInfo, analysis *TaskAnalysisResponse, availableWorkers []AvailableWorker) (*WorkerSelectionResponse, error) {
	// 准备选择请求
	request := WorkerSelectionRequest{
		TaskInfo: TaskAnalysisRequest{
			TaskID:      taskInfo.ID,
			Name:        taskInfo.Name,
			Description: taskInfo.Description,
			Priority:    taskInfo.Priority.String(),
			Metadata:    taskInfo.Metadata,
		},
		AnalysisResult:   *analysis,
		AvailableWorkers: availableWorkers,
	}

	systemPrompt := `你是一个智能Worker选择专家。请根据任务需求和可用Worker的状态，选择最合适的Worker来执行任务。

选择标准：
1. Worker类型是否匹配任务需求
2. Worker当前负载情况
3. Worker历史性能表现
4. 任务优先级和复杂度

请提供详细的选择理由和置信度评估。`

	requestJSON, err := json.MarshalToString(request)
	if err != nil {
		return nil, fmt.Errorf("序列化Worker选择请求失败: %w", err)
	}

	userPrompt := fmt.Sprintf("请为以下任务选择最合适的Worker：\n%s", requestJSON)

	// 构建ChatInput
	chatInput := llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleSystem,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: systemPrompt,
					},
				},
			},
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{
						Type: llminterface.PartTypeText,
						Text: userPrompt,
					},
				},
			},
		},
		ConversationID: fmt.Sprintf("worker_selection_%s", taskInfo.ID),
	}

	// 创建Schema
	schema := CreateWorkerSelectionSchema()

	// 使用ProduceJSON获取结构化响应
	result, err := o.llm.ProduceJSON(ctx, chatInput, Some(*schema))
	if err != nil {
		return nil, fmt.Errorf("LLM Worker选择失败: %w", err)
	}

	// 解析响应
	var selectionResponse WorkerSelectionResponse
	if err := json.UnmarshalFromString(result, &selectionResponse); err != nil {
		return nil, fmt.Errorf("解析Worker选择响应失败: %w", err)
	}

	o.logger.Info("Worker选择完成",
		"task_id", taskInfo.ID,
		"selected_worker_id", selectionResponse.SelectedWorkerID,
		"confidence", selectionResponse.Confidence,
		"reason", selectionResponse.SelectionReason,
	)

	return &selectionResponse, nil
}

// GenerateTaskExecutionPlan 生成任务执行计划的示例方法
func (o *SimpleOrchestrator) GenerateTaskExecutionPlan(ctx context.Context, taskInfo *state.TaskInfo, analysis *TaskAnalysisResponse) (*TaskExecutionPlan, error) {
	// 这里可以进一步扩展，使用更复杂的Schema来生成执行计划
	// 展示了如何构建嵌套的结构化数据

	plan := &TaskExecutionPlan{
		TaskID:                 taskInfo.ID,
		ExecutionStrategy:      "sequential", // 可以根据任务复杂度决定
		Steps:                  make([]ExecutionStep, 0),
		TotalEstimatedDuration: analysis.EstimatedDuration,
		ResourceRequirements: ResourceRequirements{
			MinWorkers:           1,
			MaxWorkers:           3,
			PreferredWorkerTypes: []string{analysis.RequiredWorkerType},
		},
		MonitoringPoints: []MonitoringPoint{
			{
				StepID:    "step_1",
				CheckType: "progress",
				Condition: "progress > 50%",
				Action:    "log_milestone",
			},
		},
	}

	// 如果需要分解，为每个子任务创建执行步骤
	if analysis.RequiresDecomposition {
		for i, subTask := range analysis.SubTasks {
			step := ExecutionStep{
				StepID:            fmt.Sprintf("step_%d", i+1),
				Name:              subTask.Name,
				Description:       subTask.Description,
				WorkerID:          "", // 将在运行时分配
				EstimatedDuration: subTask.EstimatedDuration,
				Parallel:          len(subTask.DependsOn) == 0, // 无依赖的可以并行
				RetryPolicy: RetryPolicy{
					MaxRetries:        2,
					RetryDelay:        30,
					BackoffMultiplier: 1.5,
					RetryableErrors:   []string{"timeout", "temporary_failure"},
				},
			}

			// 设置依赖关系
			if len(subTask.DependsOn) > 0 {
				step.DependsOn = make([]string, len(subTask.DependsOn))
				for j, dep := range subTask.DependsOn {
					step.DependsOn[j] = fmt.Sprintf("step_%d", dep+1)
				}
			}

			plan.Steps = append(plan.Steps, step)
		}
	} else {
		// 简单任务，只需要一个步骤
		plan.Steps = append(plan.Steps, ExecutionStep{
			StepID:            "step_1",
			Name:              taskInfo.Name,
			Description:       taskInfo.Description,
			WorkerID:          "",
			EstimatedDuration: analysis.EstimatedDuration,
			Parallel:          false,
			RetryPolicy: RetryPolicy{
				MaxRetries:        1,
				RetryDelay:        15,
				BackoffMultiplier: 1.0,
				RetryableErrors:   []string{"timeout"},
			},
		})
	}

	o.logger.Info("生成任务执行计划",
		"task_id", taskInfo.ID,
		"strategy", plan.ExecutionStrategy,
		"steps_count", len(plan.Steps),
		"total_duration", plan.TotalEstimatedDuration,
	)

	return plan, nil
}

// DemoUsage 展示如何使用结构化的LLM交互
func (o *SimpleOrchestrator) DemoUsage(ctx context.Context) error {
	// 创建一个示例任务
	sampleTask := &state.TaskInfo{
		ID:          "demo_task_001",
		Name:        "处理用户数据报表",
		Description: "从Excel文件中读取用户数据，进行统计分析，生成可视化图表并发送给管理层",
		Priority:    state.PriorityHigh,
		Metadata: map[string]any{
			"source_file": "/data/users.xlsx",
			"output_dir":  "/reports/",
			"recipients":  []string{"manager@company.com", "admin@company.com"},
		},
	}

	// 步骤1: 分析任务
	o.logger.Info("开始演示：分析任务")
	analysis, err := o.AnalyzeTaskWithLLM(ctx, sampleTask)
	if err != nil {
		return fmt.Errorf("任务分析失败: %w", err)
	}

	// 步骤2: 模拟一些可用的Workers
	availableWorkers := []AvailableWorker{
		{
			ID:           "worker_fs_001",
			Type:         "file_system",
			State:        "ready",
			Capabilities: []string{"read_file", "write_file", "manage_directory"},
			CurrentLoad:  0.2,
			Performance: WorkerPerformanceMetrics{
				TasksCompleted:  15,
				TasksFailed:     1,
				SuccessRate:     0.93,
				AvgDurationMins: 3.5,
			},
		},
		{
			ID:           "worker_data_001",
			Type:         "data_analysis",
			State:        "ready",
			Capabilities: []string{"excel_processing", "data_statistics", "chart_generation"},
			CurrentLoad:  0.1,
			Performance: WorkerPerformanceMetrics{
				TasksCompleted:  8,
				TasksFailed:     0,
				SuccessRate:     1.0,
				AvgDurationMins: 12.3,
			},
		},
	}

	// 步骤3: 选择Worker
	o.logger.Info("开始演示：选择Worker")
	selection, err := o.SelectWorkerWithLLM(ctx, sampleTask, analysis, availableWorkers)
	if err != nil {
		return fmt.Errorf("worker选择失败: %w", err)
	}

	// 步骤4: 生成执行计划
	o.logger.Info("开始演示：生成执行计划")
	plan, err := o.GenerateTaskExecutionPlan(ctx, sampleTask, analysis)
	if err != nil {
		return fmt.Errorf("生成执行计划失败: %w", err)
	}

	// 输出结果总结
	o.logger.Info("演示完成",
		"task_id", sampleTask.ID,
		"analysis_complexity", analysis.EstimatedComplexity,
		"requires_decomposition", analysis.RequiresDecomposition,
		"selected_worker", selection.SelectedWorkerID,
		"selection_confidence", selection.Confidence,
		"execution_steps", len(plan.Steps),
	)

	return nil
}
