package kubectltool

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

const (
	defaultBashBin = "/bin/bash"
)

// KubeTool 实现 toolcore.Tool 接口
type KubeTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

// Args 定义了 kubectl 命令的参数结构
type Args struct {
	Command          string `json:"command"`
	ModifiesResource string `json:"modifies_resource"`
	WorkDir          string `json:"workdir"`
	Kubeconfig       string `json:"kubeconfig"`
}

// NewKubectlTool 创建一个新的 KubeTool 实例
func NewKubectlTool(i18nMgr *i18n.Manager) *KubeTool {
	return &KubeTool{
		logger:  slog.Default().WithGroup("kube_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*KubeTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *KubeTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "kube.prompt.default", nil)
		localizedNames[lang] = "kubectl"
	}

	// 构建参数定义
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "command", toolcore.ParamTypeString, true, nil, "tool.kubectl.arg.command", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "modifies_resource", toolcore.ParamTypeString, true, nil, "tool.kubectl.arg.modifies_resource", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "workdir", toolcore.ParamTypeString, false, nil, "tool.kubectl.arg.workdir", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "kubeconfig", toolcore.ParamTypeString, false, nil, "tool.kubectl.arg.kubeconfig", nil),
	}

	outputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "error", toolcore.ParamTypeString, false, nil, "tool.kubectl.result.error", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "stdout", toolcore.ParamTypeString, false, nil, "tool.kubectl.result.stdout", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "stderr", toolcore.ParamTypeString, false, nil, "tool.kubectl.result.stderr", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "exit_code", toolcore.ParamTypeInteger, false, nil, "tool.kubectl.result.exit_code", nil),
	}

	// 返回工具的完整模式
	return toolcore.ToolSchema{
		Name:             "kubectl",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: outputParameters,
	}, nil
}

// ExecResult 表示命令执行的结果
type ExecResult struct {
	Error    string `json:"error,omitempty"`
	Stdout   string `json:"stdout,omitempty"`
	Stderr   string `json:"stderr,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *KubeTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args Args
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	// 检查命令参数
	if args.Command == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}

	// 设置默认值
	if args.WorkDir == "" {
		args.WorkDir = homeDir
	}
	if args.Kubeconfig == "" {
		args.Kubeconfig = filepath.Join(homeDir, ".kube", "config")
	}

	// 执行 kubectl 命令
	result, err := t.runKubectlCommand(ctx, args.Command, args.WorkDir, args.Kubeconfig)
	if err != nil {
		t.logger.Error("执行 kubectl 命令失败", "error", err, "command", args.Command)
		return "", fmt.Errorf("执行命令失败: %w", err)
	}

	// 序列化结果
	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.logger.Error("序列化结果失败", "error", err)
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return string(resultJSON), nil
}

// runKubectlCommand 执行 kubectl 命令
func (t *KubeTool) runKubectlCommand(ctx context.Context, command, workDir, kubeconfig string) (*ExecResult, error) {
	// 检查不允许的命令
	if strings.Contains(command, "kubectl edit") {
		result := &ExecResult{
			Error: "不支持交互式模式，请使用非交互式命令",
		}
		return result, nil
	}
	if strings.Contains(command, "kubectl port-forward") {
		result := &ExecResult{
			Error: "不允许端口转发，因为助手在非交互模式下运行，请尝试其他替代方案",
		}
		return result, nil
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.CommandContext(ctx, os.Getenv("COMSPEC"), "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, t.lookupBashBin(), "-c", command)
	}
	cmd.Env = os.Environ()
	cmd.Dir = workDir
	if kubeconfig != "" {
		kubeconfig, err := expandShellVar(kubeconfig)
		if err != nil {
			return nil, err
		}
		cmd.Env = append(cmd.Env, "KUBECONFIG="+kubeconfig)
	}

	return executeCommand(cmd)
}

// expandShellVar expands shell variables and syntax using bash
func expandShellVar(value string) (string, error) {
	if strings.Contains(value, "~") {
		if len(value) >= 2 && value[0] == '~' && os.IsPathSeparator(value[1]) {
			if runtime.GOOS == "windows" {
				value = filepath.Join(os.Getenv("USERPROFILE"), value[2:])
			} else {
				value = filepath.Join(os.Getenv("HOME"), value[2:])
			}
		}
	}
	return os.ExpandEnv(value), nil
}

// Find the bash executable path using exec.LookPath.
// On some systems (like NixOS), executables might not be in standard locations like /bin/bash.
func (t *KubeTool) lookupBashBin() string {
	actualBashPath, err := exec.LookPath("bash")
	if err != nil {
		t.logger.Warn("'bash' not found in PATH, defaulting to /bin/bash", "error", err)
		return defaultBashBin
	}
	return actualBashPath
}

func executeCommand(cmd *exec.Cmd) (*ExecResult, error) {
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	results := &ExecResult{}
	if err := cmd.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			results.ExitCode = exitError.ExitCode()
		} else {
			return nil, fmt.Errorf("执行命令失败: %w", err)
		}
	}
	results.Stdout = stdout.String()
	results.Stderr = stderr.String()
	return results, nil
}
