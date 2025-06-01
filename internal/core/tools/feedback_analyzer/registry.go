package feedbackanalyzer

import (
	"context"
	"fmt"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// RegistFeedbackAnalyzerTools 注册反馈分析工具
func RegistFeedbackAnalyzerTools(ctx context.Context, registry *toolcore.Registry) error {
	// 注册Agent反馈分析工具
	err := registry.Register(ctx, NewAgentFeedbackAnalysisTool(i18n.GlobalManager))
	if err != nil {
		return fmt.Errorf("注册决策工具失败: %w", err)
	}

	// 注册Agent洞察工具
	err = registry.Register(ctx, NewAgentInsightsTool(i18n.GlobalManager))
	if err != nil {
		return fmt.Errorf("注册Agent选择工具失败: %w", err)
	}

	// 注册Agent风险评估工具
	err = registry.Register(ctx, NewAgentRiskAssessmentTool(i18n.GlobalManager))
	if err != nil {
		return fmt.Errorf("注册任务优先级评估工具失败: %w", err)
	}

	// 注册Agent质量评估工具
	err = registry.Register(ctx, NewAgentQualityAssessmentTool(i18n.GlobalManager))
	if err != nil {
		return fmt.Errorf("注册监控定义工具失败: %w", err)
	}

	return nil
}
