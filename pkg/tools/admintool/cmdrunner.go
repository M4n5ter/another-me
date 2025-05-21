package admintool

import (
	"context"
	"fmt"
	"os/exec"
)

// CommandRunner 定义命令执行接口
type CommandRunner interface {
	// Run 执行命令并返回结果
	Run(ctx context.Context, cmd string, args ...string) ([]byte, error)
	// RunShell 执行shell命令
	RunShell(ctx context.Context, cmdStr string) ([]byte, error)
}

// RealCommandRunner 真实命令执行器
type RealCommandRunner struct{}

// NewRealCommandRunner 创建真实命令执行器
func NewRealCommandRunner() CommandRunner {
	return &RealCommandRunner{}
}

// Run 运行指定的命令和参数
func (r *RealCommandRunner) Run(ctx context.Context, cmd string, args ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, cmd, args...).CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("执行命令失败: %w", err)
	}
	return output, nil
}

// RunShell 运行shell命令
func (r *RealCommandRunner) RunShell(ctx context.Context, cmdStr string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, "sh", "-c", cmdStr).CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("执行shell命令失败: %w", err)
	}
	return output, nil
}
