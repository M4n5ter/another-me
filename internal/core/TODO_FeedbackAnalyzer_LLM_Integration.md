# TODO: FeedbackAnalyzer LLM 智能增强计划

## 1. 概述

本文档旨在规划将大型语言模型 (LLM) 集成到 `FeedbackAnalyzer` 中，以增强其在执行结果分析方面的核心功能，特别是质量评估、风险预测和洞察生成。目标是让这些功能更加智能和深入。

## 2. LLM 集成通用方法

对于每个需要LLM支持的功能，其通用实现流程如下：

1.  **准备输入数据**: 从函数参数（如执行结果、系统状态、历史数据等）中提取和处理信息，构建适合LLM分析的数据结构。
2.  **构建LLM提示 (Prompt)**:
    *   **System Message**: 定义LLM的角色和任务，指示其期望的分析深度、输出格式（通常是JSON）和关键考量点。
    *   **User Message**: 提供准备好的输入数据，供LLM分析。
    *   可以根据需要使用 `llminterface.InputMessage` 组织多轮对话格式的提示。
3.  **调用LLM**: 使用类似 `SmartFeedbackAnalyzer.invokeLLMForAnalysis` 的方法，将构建好的提示消息发送给LLM。
4.  **解析LLM响应**:
    *   对LLM返回的（JSON）字符串进行清洗（例如移除markdown代码块标记）。
    *   将JSON响应反序列化为预定义的Go结构体。
    *   进行错误处理和数据校验。
5.  **返回结果**: 返回处理后的分析结果。

## 3. FeedbackAnalyzer 接口变更建议

为了支持新的和改进的功能，`FeedbackAnalyzer` 接口 (位于 `internal/core/interfaces.go`) 可能需要如下更新：

```go
package core

import (
	"context"
	"time" // 确保导入 time 包，如果 QualityAssessment 等新类型需要

	"github.com/m4n5ter/another-me/internal/core/types"
)

// QualityAssessment 执行质量评估结果
type QualityAssessment struct {
	OverallQualityScore   float64           `json:"overall_quality_score"`   // 综合质量评分 (0.0-1.0)
	DimensionScores       map[string]float64 `json:"dimension_scores"`        // 各维度评分 (如: "efficiency", "accuracy")
	Strengths             []string          `json:"strengths"`               // 主要优点
	Weaknesses            []string          `json:"weaknesses"`              // 主要不足
	DetailedReport        string            `json:"detailed_report"`         // 详细评估文本
	RecommendedImprovements []string          `json:"recommended_improvements"`// 推荐改进点
}

// GeneratedInsights LLM生成的深度洞察结果
type GeneratedInsights struct {
	KeyTakeaways              []string `json:"key_takeaways"` // 核心发现
	ActionableRecommendations []struct {
		Recommendation string `json:"recommendation"` // 建议内容
		Rationale      string `json:"rationale"`      // 理由
		Priority       string `json:"priority"`       // 优先级 (high/medium/low)
		PotentialImpact string `json:"potential_impact"` // 潜在影响
	} `json:"actionable_recommendations"`
	ImprovementOpportunities []struct {
		Opportunity    string `json:"opportunity"`     // 机会描述
		PotentialBenefit string `json:"potential_benefit"` // 潜在益处
		Difficulty     string `json:"difficulty"`      // 实现难度 (high/medium/low)
	} `json:"improvement_opportunities"`
	PredictedNextSteps []string `json:"predicted_next_steps"` // 预测的下一步行动
	ConfidenceLevel    float64  `json:"confidence_level"`    // 洞察的置信度 (0.0-1.0)
	Summary            string   `json:"summary"`             // 洞察总结
}

// FeedbackAnalysisRecord 为了在AssessRisk中使用历史记录，确保其定义对接口使用者可见或在此处引用
// type FeedbackAnalysisRecord struct { ... } // (已在 feedback_analyzer.go 中定义)


// FeedbackAnalyzer 反馈分析器接口
type FeedbackAnalyzer interface {
	// AnalyzeExecutionResults 分析执行结果 (现有方法，可能会利用新的LLM功能进行增强或其输出作为新功能输入)
	AnalyzeExecutionResults(ctx context.Context, results []types.ExecutionResult) (types.AgentOutputAnalysis, error)

	// DetectPatterns 检测执行模式 (现有方法，暂不修改其核心逻辑为LLM)
	DetectPatterns(ctx context.Context, history []types.ExecutionResult) ([]string, error)

	// PredictNextSteps 预测下一步操作 (现有方法，可以考虑未来是否也使用LLM增强)
	PredictNextSteps(ctx context.Context, currentResults []types.ExecutionResult, systemState types.SystemState) ([]string, error)

	// --- 新增或重点改造的方法 (使用LLM) ---

	// AssessExecutionQuality 评估执行质量 (新方法)
	// 基于执行结果，通过LLM进行多维度质量评估
	AssessExecutionQuality(ctx context.Context, results []types.ExecutionResult) (QualityAssessment, error)

	// AssessRisk 评估风险 (修改)
	// 通过LLM，基于当前结果、提议的行动、系统状态和历史数据进行风险预测
	AssessRisk(ctx context.Context, currentResults []types.ExecutionResult, proposedActions []types.Task, systemState types.SystemState, history []FeedbackAnalysisRecord) (types.RiskAssessment, error)

	// GenerateInsights 生成洞察 (修改)
	// 通过LLM，从分析数据、历史记录和模式中生成更深层次的可执行洞察
	GenerateInsights(ctx context.Context, baseAnalysis types.AgentOutputAnalysis, history []types.ExecutionResult, detectedPatterns []string) (GeneratedInsights, error)
}
```
*这些新的结构体 (`QualityAssessment`, `GeneratedInsights`) 需要在 `internal/core/types/` 目录下或合适的包中定义。为简化，这里直接列出结构。*

## 4. 新增/修改功能的详细规划

### 4.1. 质量评估 (AssessExecutionQuality) - 新函数

*   **目的**: 利用LLM对Agent的执行结果进行全面的、多维度的质量评估。
*   **接口方法签名**:
    ```go
    AssessExecutionQuality(ctx context.Context, results []types.ExecutionResult) (QualityAssessment, error)
    ```
*   **输出数据结构 (`QualityAssessment`)**:
    ```json
    {
      "overall_quality_score": 0.85,
      "dimension_scores": {
        "efficiency": 0.9, // 例如，基于执行时间和资源消耗
        "accuracy": 0.8,   // 例如，基于错误率和输出正确性
        "stability": 0.75, // 例如，基于执行过程中的异常和重试次数
        "resource_utilization": 0.95 // 例如，CPU、内存使用情况
      },
      "strengths": ["执行效率高", "大部分任务成功完成"],
      "weaknesses": ["部分任务稳定性有待提高", "资源峰值消耗较高"],
      "detailed_report": "本次执行总体质量良好，效率和资源利用率表现出色。准确性尚可，但在稳定性方面，观察到少量任务出现重试，建议关注相关错误日志以优化。峰值资源使用略高，可评估是否有优化空间。",
      "recommended_improvements": ["检查并优化导致任务重试的错误", "分析资源峰值时段的操作，寻求平滑方案"]
    }
    ```
    *(此结构体需要定义在 `types` 包中)*
*   **LLM输入数据准备**:
    *   统计数据：总任务数、成功数、失败数、成功率、失败率。
    *   时间数据：平均执行时长、总执行时长、各个任务的开始/结束时间。
    *   错误信息：错误列表、错误频率。
    *   （可选）资源消耗摘要（如果可获取）。
    *   （可选）部分关键任务的输入/输出摘要。
*   **LLM 系统提示词 (System Prompt)**:
    ```
    你是一位经验丰富的AI智能应用执行质量评估专家。你的任务是基于提供的Agent执行结果数据，进行全面而深入的质量评估。
    请重点从以下几个维度进行分析：
    1.  **效率**: 任务执行速度、耗时、是否有不必要的延迟。
    2.  **准确性/正确性**: 任务成功率、错误发生情况、输出结果是否符合预期。
    3.  **稳定性**: 执行过程中是否出现异常、崩溃、重试等情况。
    4.  **资源利用率**: （如果提供相关数据）CPU、内存等资源使用是否合理，有无浪费。

    请严格按照以下JSON格式返回你的评估报告，确保所有字段都被填充：
    {
      "overall_quality_score": <综合质量评分, 0.0-1.0>,
      "dimension_scores": {
        "efficiency": <效率评分, 0.0-1.0>,
        "accuracy": <准确性评分, 0.0-1.0>,
        "stability": <稳定性评分, 0.0-1.0>,
        "resource_utilization": <资源利用率评分, 0.0-1.0, 如果无数据则为-1或省略该键>
      },
      "strengths": ["<主要优点1>", "<主要优点2>", ...],
      "weaknesses": ["<主要不足1>", "<主要不足2>", ...],
      "detailed_report": "<对整体质量的详细文字描述，包括对各维度表现的分析>",
      "recommended_improvements": ["<具体的改进建议1>", "<具体的改进建议2>", ...]
    }
    ```

### 4.2. 风险预测 (AssessRisk) - 修改现有函数

*   **目的**: 利用LLM结合当前结果、计划行动、系统状态及历史数据，进行更智能、更全面的风险预测。
*   **接口方法签名**:
    ```go
    AssessRisk(ctx context.Context, currentResults []types.ExecutionResult, proposedActions []types.Task, systemState types.SystemState, history []FeedbackAnalysisRecord) (types.RiskAssessment, error)
    ```
    *(`types.RiskAssessment` 已在 `types` 包中定义)*
*   **LLM输入数据准备**:
    *   `currentResults` 摘要：近期成功/失败情况，错误类型。
    *   `proposedActions` 摘要：将要执行的任务类型、数量、关键参数。
    *   `systemState`：当前系统错误计数、活跃状态等。
    *   `history` 摘要：从 `FeedbackAnalysisRecord` 中提取过去相似场景下的风险评估、失败模式、高频错误等。
*   **LLM 系统提示词 (System Prompt)**:
    ```
    你是一位资深的AI系统风险管理与预测专家。你的任务是基于提供的当前执行结果、计划中的后续行动、当前系统状态以及相关的历史分析数据，全面评估潜在的风险。
    你需要识别风险因素，预测风险发生的可能性和潜在影响，并确定风险等级。

    请严格按照以下JSON格式返回你的风险评估报告，确保所有字段都被填充：
    {
      "level": "<风险级别: low/medium/high>",
      "factors": ["<识别出的风险因子1>", "<风险因子2>", ...],
      "mitigation": ["<针对风险的缓解措施1>", "<缓解措施2>", ...],
      "description": "<对整体风险情况的详细文字描述，包括对主要风险因子的分析和潜在影响的评估>",
      "predicted_impact_if_occurs": "<如果风险发生，预测的具体影响描述>",
      "probability_assessment": "<对风险发生可能性的定性评估: low/medium/high，或定量0.0-1.0>"
    }
    ```
    *(注意: `predicted_impact_if_occurs` 和 `probability_assessment` 是对现有 `types.RiskAssessment` 的潜在扩展，如果采纳，需要更新该类型定义。如果暂时不扩展，则LLM提示中省略这两个字段，并确保LLM只输出现有字段。)*

### 4.3. 洞察生成 (GenerateInsights) - 修改现有函数

*   **目的**: 利用LLM从基础分析结果、历史数据和模式中提炼更深层次、更具可操作性的洞察、建议和下一步行动。
*   **接口方法签名**:
    ```go
    GenerateInsights(ctx context.Context, baseAnalysis types.AgentOutputAnalysis, history []types.ExecutionResult, detectedPatterns []string) (GeneratedInsights, error)
    ```
*   **输出数据结构 (`GeneratedInsights`)**:
    ```json
    {
      "key_takeaways": [
        "核心发现1：近期X类型任务的成功率显著提升。",
        "核心发现2：尽管总体稳定，但Y错误模式依然间歇性出现。"
      ],
      "actionable_recommendations": [
        {
          "recommendation": "针对Y错误模式，建议增加更详细的日志记录级别，并设置告警。",
          "rationale": "有助于快速定位Y错误的根本原因，减少其影响。",
          "priority": "high",
          "potential_impact": "显著降低Y错误导致的失败率，提升用户体验。"
        },
        {
          "recommendation": "鉴于X类型任务的成功率提升，可以考虑逐步增加其负载。",
          "rationale": "系统已表现出处理该类型任务的鲁棒性。",
          "priority": "medium",
          "potential_impact": "提高整体处理效率，更快完成业务目标。"
        }
      ],
      "improvement_opportunities": [
        {
          "opportunity": "优化Z流程的数据预处理步骤。",
          "potential_benefit": "可能缩短Z流程平均执行时间15%。",
          "difficulty": "medium"
        }
      ],
      "predicted_next_steps": [
        "安排技术评审会议，讨论Y错误模式的解决方案。",
        "制定X类型任务负载的渐进式增加计划。"
      ],
      "confidence_level": 0.88,
      "summary": "系统在X任务类型上表现良好，但需关注并解决Y错误模式。通过实施推荐的措施，有望进一步提升系统稳定性和效率。"
    }
    ```
    *(此结构体需要定义在 `types` 包中)*
*   **LLM输入数据准备**:
    *   `baseAnalysis`: `AnalyzeExecutionResults` 函数返回的 `types.AgentOutputAnalysis` 对象，包含初步的关键发现、风险评估等。
    *   `history` 摘要：近期执行结果的关键统计数据、性能趋势。
    *   `detectedPatterns`: 由 `DetectPatterns` 函数识别出的模式列表。
*   **LLM 系统提示词 (System Prompt)**:
    ```
    你是一名顶级的AI系统分析与策略顾问。你的任务是基于提供的初步分析报告 (`baseAnalysis`)、历史执行数据摘要以及已识别的关键执行模式，进行深度分析，并生成一份富有洞察力的报告。
    这份报告需要包含：
    1.  **核心发现 (Key Takeaways)**: 从所有信息中提炼出的最关键、最重要的结论。
    2.  **可执行的建议 (Actionable Recommendations)**: 具体的、可操作的改进建议，每条建议需包含理由、优先级评估以及潜在影响。
    3.  **改进机会 (Improvement Opportunities)**: 指出潜在的可以优化或提升的方面，包括潜在益处和预估的实现难度。
    4.  **预测的下一步行动 (Predicted Next Steps)**: 基于你的洞察，建议接下来应该采取哪些具体步骤。
    5.  **洞察置信度 (Confidence Level)**: 你对这份洞察报告整体准确性和价值的置信度评分 (0.0-1.0)。
    6.  **洞察总结 (Summary)**: 对整个洞察报告的简短总结。

    请严格按照以下JSON格式返回你的洞察报告：
    {
      "key_takeaways": ["<核心发现1>", "<核心发现2>", ...],
      "actionable_recommendations": [
        {
          "recommendation": "<建议内容>",
          "rationale": "<理由>",
          "priority": "<high/medium/low>",
          "potential_impact": "<潜在影响描述>"
        },
        ...
      ],
      "improvement_opportunities": [
        {
          "opportunity": "<机会描述>",
          "potential_benefit": "<潜在益处>",
          "difficulty": "<high/medium/low>"
        },
        ...
      ],
      "predicted_next_steps": ["<预测的下一步1>", "<下一步2>", ...],
      "confidence_level": <置信度评分, 0.0-1.0>,
      "summary": "<洞察总结文本>"
    }
    ```

## 5. 实现TODOs

*   [x] 在 `internal/core/types/` 定义新的数据结构: `QualityAssessment`, `GeneratedInsights`。
*   [x] 更新 `internal/core/interfaces.go` 中的 `FeedbackAnalyzer` 接口定义，添加/修改上述方法签名。
*   [x] 在 `internal/core/feedback_analyzer.go` 中的 `SmartFeedbackAnalyzer` 结构体中实现新的/修改后的方法：
    *   [x] 实现 `AssessExecutionQuality` 方法，包括数据准备、提示构建、LLM调用和响应解析逻辑。
    *   [x] 修改 `AssessRisk` 方法，集成LLM调用逻辑，替换或增强现有规则。
    *   [x] 修改 `GenerateInsights` 方法，集成LLM调用逻辑，替换或增强现有规则。
*   [x] 为每个新的LLM调用逻辑创建或调整相应的提示词模板常量 (类似 `agentPromptTemplateDefault`)。
*   [x] 编写单元测试以验证新功能和LLM集成是否按预期工作，包括对LLM响应解析的健壮性测试。
*   [ ] （可选）如果 `types.RiskAssessment` 被扩展，更新其在 `types` 包中的定义。
*   [ ] 在 `NewSmartFeedbackAnalyzer` 中初始化可能需要的与新功能相关的字段。 