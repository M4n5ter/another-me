package admintool

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// SyslogTool 实现 toolcore.Tool 接口，用于查看系统日志
type SyslogTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
	runner  CommandRunner // 命令执行器
}

// SyslogArgs 定义了 SyslogTool 的参数
type SyslogArgs struct {
	Lines   int    `json:"lines,omitempty"`    // 要显示的行数
	Filter  string `json:"filter,omitempty"`   // 过滤关键词
	Since   string `json:"since,omitempty"`    // 从指定时间开始，例如 "1 hour ago"
	Until   string `json:"until,omitempty"`    // 直到指定时间，例如 "now"
	LogType string `json:"log_type,omitempty"` // 日志类型: kernel, system, all
}

// SyslogResult 定义了成功结果的结构
type SyslogResult struct {
	Content string `json:"content"` // 日志内容
	Lines   int    `json:"lines"`   // 返回的行数
	Command string `json:"command"` // 执行的命令
}

// NewSyslogTool 创建一个新的 SyslogTool 实例
func NewSyslogTool(i18nMgr *i18n.Manager) *SyslogTool {
	return &SyslogTool{
		logger:  slog.Default().WithGroup("syslog_tool"),
		i18nMgr: i18nMgr,
		runner:  NewRealCommandRunner(),
	}
}

// NewSyslogToolWithRunner 创建一个使用自定义命令执行器的SyslogTool实例（用于测试）
func NewSyslogToolWithRunner(i18nMgr *i18n.Manager, runner CommandRunner) *SyslogTool {
	return &SyslogTool{
		logger:  slog.Default().WithGroup("syslog_tool"),
		i18nMgr: i18nMgr,
		runner:  runner,
	}
}

var _ toolcore.Tool = (*SyslogTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *SyslogTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.admin.syslog.description", nil)
		localizedNames[lang] = "Syslog"
	}

	// 构建参数定义
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "lines", toolcore.ParamTypeInteger, false, nil, "tool.admin.syslog.arg.lines", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "filter", toolcore.ParamTypeString, false, nil, "tool.admin.syslog.arg.filter", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "since", toolcore.ParamTypeString, false, nil, "tool.admin.syslog.arg.since", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "until", toolcore.ParamTypeString, false, nil, "tool.admin.syslog.arg.until", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "log_type", toolcore.ParamTypeString, false, nil, "tool.admin.syslog.arg.log_type", nil),
	}

	// 返回工具的完整模式
	return toolcore.ToolSchema{
		Name:             "syslog",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *SyslogTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args SyslogArgs
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	// 设置默认值
	if args.Lines <= 0 {
		args.Lines = 50
	}

	var cmdStr string

	switch args.LogType {
	case "kernel":
		// 使用 dmesg 命令获取内核日志
		cmdArgs := []string{"-l", "kern"}

		if args.Lines > 0 {
			cmdArgs = append(cmdArgs, "--nlines", fmt.Sprintf("%d", args.Lines))
		}

		cmdStr = fmt.Sprintf("dmesg %s", strings.Join(cmdArgs, " "))
		output, err := t.runner.Run(ctx, "dmesg", cmdArgs...)
		if err != nil {
			t.logger.Error("执行命令失败", "command", cmdStr, "error", err)
			return "", fmt.Errorf("获取内核日志失败: %w", err)
		}

		content := string(output)
		lines := len(strings.Split(content, "\n"))

		result := SyslogResult{
			Content: content,
			Lines:   lines,
			Command: cmdStr,
		}

		resultJSON, err := json.MarshalToString(result)
		if err != nil {
			t.logger.Error("序列化结果失败", "error", err)
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}

		return resultJSON, nil

	default:
		// 默认使用 journalctl 或 cat /var/log/syslog
		// 先尝试 journalctl（适用于使用systemd的系统）
		cmdArgs := []string{}

		// 添加时间范围
		if args.Since != "" {
			cmdArgs = append(cmdArgs, "--since", args.Since)
		}
		if args.Until != "" {
			cmdArgs = append(cmdArgs, "--until", args.Until)
		}

		// 添加过滤
		if args.Filter != "" {
			cmdArgs = append(cmdArgs, fmt.Sprintf("--grep=%s", args.Filter))
		}

		// 限制行数
		cmdArgs = append(cmdArgs, "--lines", fmt.Sprintf("%d", args.Lines))

		// 根据日志类型添加额外参数
		if args.LogType == "system" {
			cmdArgs = append(cmdArgs, "-u", "systemd")
		}

		cmdStr = fmt.Sprintf("journalctl %s", strings.Join(cmdArgs, " "))
		t.logger.Info("执行命令", "command", cmdStr)

		output, err := t.runner.Run(ctx, "journalctl", cmdArgs...)
		if err != nil {
			t.logger.Error("执行命令失败", "command", cmdStr, "error", err)

			// 如果journalctl失败，尝试退回到传统日志文件（如/var/log/syslog）
			var fallbackOutput []byte
			var fallbackErr error
			var fallbackCmdStr string

			if args.Filter != "" {
				fallbackCmdStr = fmt.Sprintf("tail -%d /var/log/syslog | grep %s", args.Lines, args.Filter)
				fallbackOutput, fallbackErr = t.runner.RunShell(ctx, fallbackCmdStr)
			} else {
				fallbackCmdStr = fmt.Sprintf("tail -%d /var/log/syslog", args.Lines)
				fallbackOutput, fallbackErr = t.runner.Run(ctx, "tail", fmt.Sprintf("-%d", args.Lines), "/var/log/syslog")
			}

			t.logger.Info("尝试备用命令", "command", fallbackCmdStr)

			if fallbackErr != nil {
				t.logger.Error("备用命令也失败", "command", fallbackCmdStr, "error", fallbackErr)
				return "", fmt.Errorf("获取系统日志失败: %w", fallbackErr)
			}

			output = fallbackOutput
			cmdStr = fallbackCmdStr
		}

		content := string(output)
		lines := len(strings.Split(content, "\n"))

		result := SyslogResult{
			Content: content,
			Lines:   lines,
			Command: cmdStr,
		}

		resultJSON, err := json.MarshalToString(result)
		if err != nil {
			t.logger.Error("序列化结果失败", "error", err)
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}

		return resultJSON, nil
	}
}

// createOutputParameters 创建输出参数定义
func (t *SyslogTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	contentDesc := map[string]string{
		"en": "The log content",
		"zh": "日志内容",
	}

	linesDesc := map[string]string{
		"en": "Number of lines returned",
		"zh": "返回的行数",
	}

	commandDesc := map[string]string{
		"en": "The command executed",
		"zh": "执行的命令",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "content",
			Type:        toolcore.ParamTypeString,
			Description: contentDesc,
			Required:    true,
		},
		{
			Name:        "lines",
			Type:        toolcore.ParamTypeInteger,
			Description: linesDesc,
			Required:    true,
		},
		{
			Name:        "command",
			Type:        toolcore.ParamTypeString,
			Description: commandDesc,
			Required:    true,
		},
	}
}
