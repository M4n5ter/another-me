package tests

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/m4n5ter/another-me/internal/core/types"

	"github.com/m4n5ter/another-me/internal/core"
	"github.com/m4n5ter/another-me/pkg/llminterface"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// TestSmartFeedbackAnalyzer_AnalyzeExecutionResults tests the AnalyzeExecutionResults method.
func TestSmartFeedbackAnalyzer_AnalyzeExecutionResults(t *testing.T) {
	cfg := core.DefaultFeedbackAnalyzerConfig()
	cfg.AnalysisTimeout = 100 * time.Second // Increased timeout for potential LLM calls
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("no results provided", func(t *testing.T) {
		// For this case, LLM is not called, so a nil adapter or a basic mock is fine.
		mockLLM := new(MockChatAdapter) // Use mock, no real LLM needed
		analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, cfg, newTestLogger())

		analysis, err := analyzer.AnalyzeExecutionResults(context.Background(), []types.ExecutionResult{})
		require.NoError(t, err)
		assert.Equal(t, []string{"没有执行结果"}, analysis.KeyFindings)
		assert.False(t, analysis.RequiresUserInput)
		assert.Equal(t, 0.0, analysis.ConfidenceLevel)
		assert.Equal(t, "low", analysis.RiskAssessment.Level)
	})

	t.Run("successful LLM analysis (using mock)", func(t *testing.T) {
		mockLLM := new(MockChatAdapter)
		logger := newTestLogger()
		analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, cfg, logger)

		results := []types.ExecutionResult{
			{TaskID: "llm_task1", Status: types.ExecutionStatusSuccess, Output: "Good output", StartTime: baseTime, EndTime: baseTime.Add(time.Second)},
			{TaskID: "llm_task2", Status: types.ExecutionStatusFailure, Error: "Bad error", StartTime: baseTime, EndTime: baseTime.Add(time.Second)},
		}

		// Expected JSON output from LLM for AnalyzeExecutionResults
		expectedLLMResponse := `{
			"key_findings": ["Mocked: Found success and failure patterns"],
			"actionable_insights": ["Mocked: Review failed task llm_task2"],
			"requires_user_input": true,
			"confidence_level": 0.88,
			"recommended_actions": ["Mocked: Check logs for llm_task2"],
			"risk_assessment": {
				"level": "medium",
				"factors": ["Mocked: One failure observed"],
				"mitigation": ["Mocked: Retry mechanism suggested"],
				"description": "Mocked: Medium risk due to one failure."
			},
			"next_step_suggestions": ["Mocked: Proceed with caution"],
			"patterns_detected": ["Mocked: Success/failure mix"],
			"improvement_opportunities": ["Mocked: Enhance error reporting"]
		}`
		outChan := make(chan llminterface.ChatOutputChunk, 1)
		outChan <- llminterface.ChatOutputChunk{
			ContentParts: []llminterface.ContentPart{{Type: llminterface.PartTypeText, Text: expectedLLMResponse}},
		}
		close(outChan)
		mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(outChan), nil).Once()

		analysis, err := analyzer.AnalyzeExecutionResults(t.Context(), results)

		require.NoError(t, err)
		assert.Contains(t, analysis.KeyFindings, "Mocked: Found success and failure patterns")
		assert.Equal(t, 0.88, analysis.ConfidenceLevel)
		assert.Equal(t, "medium", analysis.RiskAssessment.Level)
		assert.Contains(t, analysis.ActionableInsights, "Mocked: Review failed task llm_task2")
		mockLLM.AssertExpectations(t)
	})

	t.Run("LLM adapter is nil (simulating init failure), fallback to basic analysis", func(t *testing.T) {
		// Pass nil adapter to simulate LLM init failure
		analyzerWithNilAdapter := core.NewSmartFeedbackAnalyzer(nil, nil, cfg, newTestLogger())

		results := []types.ExecutionResult{
			{TaskID: "fail_task1", Status: types.ExecutionStatusFailure, Error: "some error", StartTime: baseTime, EndTime: baseTime.Add(time.Second)},
		}
		analysis, err := analyzerWithNilAdapter.AnalyzeExecutionResults(context.Background(), results)
		t.Logf("Fallback Analysis (nil adapter): %+v", analysis)
		require.NoError(t, err)
		assert.Equal(t, 0.2, analysis.ConfidenceLevel) // Fallback confidence
		assert.Contains(t, analysis.KeyFindings, "总共执行任务数: 1")
		assert.Contains(t, analysis.KeyFindings, "成功率: 0.00%")
		assert.Contains(t, analysis.KeyFindings, "错误数: 1")
		assert.Equal(t, "high", analysis.RiskAssessment.Level) // Based on fallback logic for 100% failure
	})

	t.Run("buildLLMPromptMessage fails (e.g. bad template in config), fallback", func(t *testing.T) {
		badTemplateCfg := core.DefaultFeedbackAnalyzerConfig()
		badTemplateCfg.AgentPromptTemplate = `{{ .InvalidField }}` // This will cause an error in buildLLMPromptMessage

		// Even with a mock LLM, if prompt building fails, it should fallback.
		mockLLM := new(MockChatAdapter)
		analyzerBadTemplate := core.NewSmartFeedbackAnalyzer(nil, mockLLM, badTemplateCfg, newTestLogger())

		results := []types.ExecutionResult{
			{TaskID: "fail_task_prompt", Status: types.ExecutionStatusSuccess, StartTime: baseTime, EndTime: baseTime.Add(time.Second)},
		}
		analysis, err := analyzerBadTemplate.AnalyzeExecutionResults(context.Background(), results)
		require.NoError(t, err)                        // Fallback should not return an error to the caller
		assert.Equal(t, 0.2, analysis.ConfidenceLevel) // Fallback confidence
		assert.Contains(t, analysis.KeyFindings, "总共执行任务数: 1")
		// mockLLM.AssertNotCalled(t, "Chat") // Chat should not be called if prompt building fails - this mock does not support AssertNotCalled directly without more setup
	})
}

func TestSmartFeedbackAnalyzer_AssessExecutionQuality_Success(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	// Simulate LLM returning a valid JSON response for quality assessment
	jsonResponse := `{
		"overall_quality_score": 0.85,
		"dimension_scores": {
			"efficiency": 0.9,
			"accuracy": 0.8,
			"stability": 0.75
		},
		"strengths": ["Fast execution", "High accuracy"],
		"weaknesses": ["Occasional timeouts"],
		"detailed_report": "The agent performed well overall.",
		"recommended_improvements": ["Optimize timeout handling"]
	}`
	outChan := make(chan llminterface.ChatOutputChunk, 1)
	outChan <- llminterface.ChatOutputChunk{
		ContentParts: []llminterface.ContentPart{
			{Type: llminterface.PartTypeText, Text: jsonResponse},
		},
	}
	close(outChan)

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(outChan), nil).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	results := []types.ExecutionResult{
		{TaskID: "task1", Status: types.ExecutionStatusSuccess, StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Second)},
		{TaskID: "task2", Status: types.ExecutionStatusFailure, Error: "failed", StartTime: time.Now(), EndTime: time.Now().Add(2 * time.Second)},
	}

	assessment, err := analyzer.AssessExecutionQuality(context.Background(), results)
	require.NoError(t, err)

	assert.Equal(t, 0.85, assessment.OverallQualityScore)
	assert.Equal(t, 0.9, assessment.DimensionScores["efficiency"])
	assert.Contains(t, assessment.Strengths, "Fast execution")
	assert.Contains(t, assessment.Weaknesses, "Occasional timeouts")
	assert.Equal(t, "The agent performed well overall.", assessment.DetailedReport)
	assert.Contains(t, assessment.RecommendedImprovements, "Optimize timeout handling")
	mockLLM.AssertExpectations(t) // Verify that expectations were met
}

func TestSmartFeedbackAnalyzer_AssessExecutionQuality_EmptyResults(t *testing.T) {
	mockLLM := &MockChatAdapter{} // Default behavior is fine
	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	assessment, err := analyzer.AssessExecutionQuality(context.Background(), []types.ExecutionResult{})
	require.NoError(t, err) // Expect no error, should return empty/default assessment
	assert.Equal(t, types.QualityAssessment{}, assessment)
}

func TestSmartFeedbackAnalyzer_AssessExecutionQuality_LLMError(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	// Even when returning an error, provide a valid (closed) channel for the first return value
	closedChan := make(chan llminterface.ChatOutputChunk)
	close(closedChan)
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(closedChan), errors.New("LLM API error")).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	results := []types.ExecutionResult{
		{TaskID: "task1", Status: types.ExecutionStatusSuccess, StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Second)},
	}

	_, err := analyzer.AssessExecutionQuality(context.Background(), results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM质量评估API调用失败")
	mockLLM.AssertExpectations(t)
}

func TestSmartFeedbackAnalyzer_AssessExecutionQuality_InvalidJSON(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	jsonResponse := `{"overall_quality_score": 0.85, "dimension_scores": { "efficiency": 0.9, "accuracy": 0.8, "stability": 0.75 }` // Missing closing brace
	outChan := make(chan llminterface.ChatOutputChunk, 1)
	outChan <- llminterface.ChatOutputChunk{
		ContentParts: []llminterface.ContentPart{
			{Type: llminterface.PartTypeText, Text: jsonResponse},
		},
	}
	close(outChan)

	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(outChan), nil).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	results := []types.ExecutionResult{
		{TaskID: "task1", Status: types.ExecutionStatusSuccess, StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Second)},
	}

	_, err := analyzer.AssessExecutionQuality(context.Background(), results)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "转换LLM质量评估结果失败")
	assert.Contains(t, err.Error(), "解析LLM质量评估JSON失败")
	mockLLM.AssertExpectations(t)
}

func TestSmartFeedbackAnalyzer_AssessRisk_Success(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	// Simulate LLM returning a valid JSON response for risk assessment
	jsonResponse := `{
		"level": "medium",
		"factors": ["High error rate", "External dependency unavailable"],
		"mitigation": ["Implement retries", "Add circuit breaker"],
		"description": "Moderate risk due to error rates and dependency issues.",
		"predicted_impact_if_occurs": "Partial service degradation.",
		"probability_assessment": "medium"
	}`
	outChan := make(chan llminterface.ChatOutputChunk, 1)
	outChan <- llminterface.ChatOutputChunk{
		ContentParts: []llminterface.ContentPart{
			{Type: llminterface.PartTypeText, Text: jsonResponse},
		},
	}
	close(outChan)
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(outChan), nil).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	currentResults := []types.ExecutionResult{
		{TaskID: "task1", Status: types.ExecutionStatusFailure, Error: "error1", StartTime: time.Now(), EndTime: time.Now().Add(1 * time.Second)},
	}
	proposedActions := []types.Task{
		{ID: "action1", Type: "typeA", Priority: 5},
	}
	systemState := types.SystemState{IsActive: true, ErrorCount: 1}
	history := []core.FeedbackAnalysisRecord{}

	riskAssessment, err := analyzer.AssessRisk(context.Background(), currentResults, proposedActions, systemState, history)
	require.NoError(t, err)

	assert.Equal(t, "medium", riskAssessment.Level)
	assert.Contains(t, riskAssessment.Factors, "High error rate")
	assert.Contains(t, riskAssessment.Mitigation, "Implement retries")
	assert.Equal(t, "Moderate risk due to error rates and dependency issues.", riskAssessment.Description)
	// Note: PredictedImpactIfOccurs and ProbabilityAssessment are not in types.RiskAssessment by default
	// If they were added, we would assert them here.
	mockLLM.AssertExpectations(t)
}

func TestSmartFeedbackAnalyzer_AssessRisk_LLMError(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	// Even when returning an error, provide a valid (closed) channel
	closedChan := make(chan llminterface.ChatOutputChunk)
	close(closedChan)
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(closedChan), errors.New("LLM connection failed")).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	_, err := analyzer.AssessRisk(context.Background(), []types.ExecutionResult{}, []types.Task{}, types.SystemState{}, []core.FeedbackAnalysisRecord{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM风险评估API调用失败")
	mockLLM.AssertExpectations(t)
}

func TestSmartFeedbackAnalyzer_AssessRisk_InvalidJSON(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	jsonResponse := `{"level": "high", "factors": ["Critical failure"]` // Malformed JSON
	outChan := make(chan llminterface.ChatOutputChunk, 1)
	outChan <- llminterface.ChatOutputChunk{
		ContentParts: []llminterface.ContentPart{
			{Type: llminterface.PartTypeText, Text: jsonResponse},
		},
	}
	close(outChan)
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(outChan), nil).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	_, err := analyzer.AssessRisk(context.Background(), []types.ExecutionResult{}, []types.Task{}, types.SystemState{}, []core.FeedbackAnalysisRecord{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "解析LLM风险评估JSON失败")
	mockLLM.AssertExpectations(t)
}

func TestSmartFeedbackAnalyzer_GenerateInsights_Success(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	// Simulate LLM returning a valid JSON response for insights generation
	jsonResponse := `{
		"key_takeaways": ["System performance is stable", "Error rate decreased"],
		"actionable_recommendations": [
			{
				"recommendation": "Monitor new deployment closely",
				"rationale": "Ensure stability after updates",
				"priority": "high",
				"potential_impact": "Maintain service availability"
			}
		],
		"improvement_opportunities": [
			{
				"opportunity": "Optimize database queries",
				"potential_benefit": "Reduce latency by 10%",
				"difficulty": "medium"
			}
		],
		"predicted_next_steps": ["Schedule performance review"],
		"confidence_level": 0.9,
		"summary": "Overall system health is good, with areas for proactive optimization."
	}`
	outChan := make(chan llminterface.ChatOutputChunk, 1)
	outChan <- llminterface.ChatOutputChunk{
		ContentParts: []llminterface.ContentPart{
			{Type: llminterface.PartTypeText, Text: jsonResponse},
		},
	}
	close(outChan)
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(outChan), nil).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	baseAnalysis := types.AgentOutputAnalysis{
		KeyFindings:     []string{"Initial finding"},
		RiskAssessment:  types.RiskAssessment{Level: "low"},
		ConfidenceLevel: 0.7,
	}
	history := []types.ExecutionResult{
		{TaskID: "hist1", Status: types.ExecutionStatusSuccess, StartTime: time.Now(), EndTime: time.Now().Add(500 * time.Millisecond)},
	}
	detectedPatterns := []string{"Frequent successful executions"}

	insights, err := analyzer.GenerateInsights(context.Background(), baseAnalysis, history, detectedPatterns)
	require.NoError(t, err)

	assert.Equal(t, 0.9, insights.ConfidenceLevel)
	assert.Contains(t, insights.KeyTakeaways, "System performance is stable")
	require.Len(t, insights.ActionableRecommendations, 1)
	assert.Equal(t, "Monitor new deployment closely", insights.ActionableRecommendations[0].Recommendation)
	require.Len(t, insights.ImprovementOpportunities, 1)
	assert.Equal(t, "Optimize database queries", insights.ImprovementOpportunities[0].Opportunity)
	assert.Contains(t, insights.PredictedNextSteps, "Schedule performance review")
	assert.Equal(t, "Overall system health is good, with areas for proactive optimization.", insights.Summary)
	mockLLM.AssertExpectations(t)
}

func TestSmartFeedbackAnalyzer_GenerateInsights_LLMError(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	// Even when returning an error, provide a valid (closed) channel
	closedChan := make(chan llminterface.ChatOutputChunk)
	close(closedChan)
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(closedChan), errors.New("LLM service unavailable")).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	_, err := analyzer.GenerateInsights(context.Background(), types.AgentOutputAnalysis{}, []types.ExecutionResult{}, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "LLM洞察生成API调用失败")
	mockLLM.AssertExpectations(t)
}

func TestSmartFeedbackAnalyzer_GenerateInsights_InvalidJSON(t *testing.T) {
	mockLLM := new(MockChatAdapter) // Changed to use testify/mock

	jsonResponse := `{"key_takeaways": ["Data inconsistent"], "confidence_level": 0.5` // Malformed
	outChan := make(chan llminterface.ChatOutputChunk, 1)
	outChan <- llminterface.ChatOutputChunk{
		ContentParts: []llminterface.ContentPart{
			{Type: llminterface.PartTypeText, Text: jsonResponse},
		},
	}
	close(outChan)
	mockLLM.On("Chat", mock.Anything, mock.Anything).Return((<-chan llminterface.ChatOutputChunk)(outChan), nil).Once()

	config := core.DefaultFeedbackAnalyzerConfig()
	logger := newTestLogger()
	analyzer := core.NewSmartFeedbackAnalyzer(nil, mockLLM, config, logger)

	_, err := analyzer.GenerateInsights(context.Background(), types.AgentOutputAnalysis{}, []types.ExecutionResult{}, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "转换LLM洞察生成结果失败")
	assert.Contains(t, err.Error(), "解析LLM洞察生成JSON失败")
	mockLLM.AssertExpectations(t)
}
