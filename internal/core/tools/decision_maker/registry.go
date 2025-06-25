package decisionmaker

import (
	"context"
	"fmt"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// RegistDecisionMakerTools 注册决策工具
func RegistDecisionMakerTools(ctx context.Context, registry *toolcore.Registry) error {
	// 注册主要决策工具
	err := registry.Register(ctx, NewMakeDecisionTool(i18n.GlobalManager))
	if err != nil {
		return fmt.Errorf("注册决策工具失败: %w", err)
	}

	// 注册Agent选择工具
	err = registry.Register(ctx, NewSelectAgentTool(i18n.GlobalManager))
	if err != nil {
		return fmt.Errorf("注册Agent选择工具失败: %w", err)
	}

	// 注册任务优先级评估工具
	err = registry.Register(ctx, NewEvaluateTaskPriorityTool(i18n.GlobalManager))
	if err != nil {
		return fmt.Errorf("注册任务优先级评估工具失败: %w", err)
	}

	// 注册监控定义工具
	err = registry.Register(ctx, NewDefineMonitoringTool(i18n.GlobalManager))
	if err != nil {
		return fmt.Errorf("注册监控定义工具失败: %w", err)
	}

	return nil
}
