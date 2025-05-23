package admintool

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// 进程信息类型常量
const (
	InfoTypeBasic       = "basic"
	InfoTypeDetailed    = "detailed"
	InfoTypeEnvironment = "environment"
	InfoTypeFiles       = "files"
	InfoTypeAll         = "all"
)

// ProcInfoTool 实现 toolcore.Tool 接口，用于获取进程信息
type ProcInfoTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
	runner  CommandRunner // 命令执行器
}

// ProcInfoArgs 定义了 ProcInfoTool 的参数
type ProcInfoArgs struct {
	PID      int    `json:"pid,omitempty"`       // 进程ID
	Name     string `json:"name,omitempty"`      // 进程名称
	InfoType string `json:"info_type,omitempty"` // 信息类型: basic, detailed, environment, files, all
	Count    int    `json:"count,omitempty"`     // 最多返回的进程数
}

// ProcInfoResult 定义了成功结果的结构
type ProcInfoResult struct {
	PID          int    `json:"pid,omitempty"`           // 进程ID
	Name         string `json:"name,omitempty"`          // 进程名称
	Content      string `json:"content"`                 // 进程信息内容
	Command      string `json:"command"`                 // 执行的命令
	InfoType     string `json:"info_type,omitempty"`     // 返回的信息类型
	ProcessCount int    `json:"process_count,omitempty"` // 匹配的进程数
}

// NewProcInfoTool 创建一个新的 ProcInfoTool 实例
func NewProcInfoTool(i18nMgr *i18n.Manager) *ProcInfoTool {
	return &ProcInfoTool{
		logger:  slog.Default().WithGroup("procinfo_tool"),
		i18nMgr: i18nMgr,
		runner:  NewRealCommandRunner(),
	}
}

// NewProcInfoToolWithRunner 创建一个使用自定义命令执行器的ProcInfoTool实例（用于测试）
func NewProcInfoToolWithRunner(i18nMgr *i18n.Manager, runner CommandRunner) *ProcInfoTool {
	return &ProcInfoTool{
		logger:  slog.Default().WithGroup("procinfo_tool"),
		i18nMgr: i18nMgr,
		runner:  runner,
	}
}

var _ toolcore.Tool = (*ProcInfoTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *ProcInfoTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.admin.procinfo.description", nil)
		localizedNames[lang] = "Process Info"
	}

	// 定义信息类型枚举
	infoTypes := []any{InfoTypeBasic, InfoTypeDetailed, InfoTypeEnvironment, InfoTypeFiles, InfoTypeAll}

	// 构建参数定义
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "pid", toolcore.ParamTypeInteger, false, nil, "tool.admin.procinfo.arg.pid", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "name", toolcore.ParamTypeString, false, nil, "tool.admin.procinfo.arg.name", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "info_type", toolcore.ParamTypeString, false, Some(infoTypes), "tool.admin.procinfo.arg.info_type", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "count", toolcore.ParamTypeInteger, false, nil, "tool.admin.procinfo.arg.count", nil),
	}

	// 返回工具的完整模式
	return toolcore.ToolSchema{
		Name:             "procinfo",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *ProcInfoTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args ProcInfoArgs
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	// 设置默认值
	if args.InfoType == "" {
		args.InfoType = InfoTypeBasic
	}
	if args.Count <= 0 {
		args.Count = 10
	}

	if args.PID <= 0 && args.Name == "" {
		// 如果没有指定PID或名称，显示所有进程的基本信息
		result, err := t.getAllProcesses(ctx, args.InfoType, args.Count)
		if err != nil {
			return "", err
		}

		resultJSON, err := json.MarshalToString(result)
		if err != nil {
			t.logger.Error("序列化结果失败", "error", err)
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}
		return resultJSON, nil
	}

	if args.PID > 0 {
		// 如果指定了PID，获取该PID的进程信息
		result, err := t.getProcessInfoByPID(ctx, args.PID, args.InfoType)
		if err != nil {
			return "", err
		}

		resultJSON, err := json.MarshalToString(result)
		if err != nil {
			t.logger.Error("序列化结果失败", "error", err)
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}
		return resultJSON, nil
	}

	// 如果指定了名称，获取该名称的进程信息
	result, err := t.getProcessInfoByName(ctx, args.Name, args.InfoType, args.Count)
	if err != nil {
		return "", err
	}

	resultJSON, err := json.MarshalToString(result)
	if err != nil {
		t.logger.Error("序列化结果失败", "error", err)
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return resultJSON, nil
}

// getAllProcesses 获取所有进程的基本信息
func (t *ProcInfoTool) getAllProcesses(ctx context.Context, infoType string, count int) (ProcInfoResult, error) {
	var cmdStr string

	switch infoType {
	case InfoTypeBasic:
		cmdStr = fmt.Sprintf("ps aux --sort=-%s | head -%d", "cpu", count+1)
	case InfoTypeDetailed:
		cmdStr = fmt.Sprintf("ps -eo pid,ppid,user,stat,pcpu,pmem,comm,args --sort=-%s | head -%d", "pcpu", count+1)
	default:
		cmdStr = fmt.Sprintf("ps aux --sort=-%s | head -%d", "cpu", count+1)
	}

	output, err := t.runner.RunShell(ctx, cmdStr)
	if err != nil {
		t.logger.Error("执行命令失败", "command", cmdStr, "error", err)
		return ProcInfoResult{}, fmt.Errorf("获取进程信息失败: %w", err)
	}

	return ProcInfoResult{
		Content:      string(output),
		Command:      cmdStr,
		InfoType:     infoType,
		ProcessCount: count,
	}, nil
}

// getProcessInfoByPID 根据PID获取进程信息
func (t *ProcInfoTool) getProcessInfoByPID(ctx context.Context, pid int, infoType string) (ProcInfoResult, error) {
	var cmdStr string
	pidStr := strconv.Itoa(pid)

	switch infoType {
	case InfoTypeBasic:
		cmdStr = fmt.Sprintf("ps -p %s -o pid,ppid,user,stat,pcpu,pmem,comm,args", pidStr)
	case InfoTypeDetailed:
		cmdStr = fmt.Sprintf("ps -p %s -f", pidStr)
	case InfoTypeEnvironment:
		// 获取进程的环境变量
		cmdStr = fmt.Sprintf("cat /proc/%s/environ | tr '\\0' '\\n'", pidStr)
	case InfoTypeFiles:
		// 获取进程打开的文件
		cmdStr = fmt.Sprintf("lsof -p %s", pidStr)
	case InfoTypeAll:
		// 获取所有信息的组合
		var allOutput strings.Builder

		// 基本信息
		basicCmd := fmt.Sprintf("ps -p %s -o pid,ppid,user,stat,pcpu,pmem,comm,args", pidStr)
		basicOutput, err := t.runner.RunShell(ctx, basicCmd)
		if err == nil {
			allOutput.WriteString("===== 基本信息 =====\n")
			allOutput.WriteString(string(basicOutput))
			allOutput.WriteString("\n\n")
		}

		// 详细信息
		detailCmd := fmt.Sprintf("ps -p %s -f", pidStr)
		detailOutput, err := t.runner.RunShell(ctx, detailCmd)
		if err == nil {
			allOutput.WriteString("===== 详细信息 =====\n")
			allOutput.WriteString(string(detailOutput))
			allOutput.WriteString("\n\n")
		}

		// 尝试获取环境变量
		envCmd := fmt.Sprintf("cat /proc/%s/environ 2>/dev/null | tr '\\0' '\\n'", pidStr)
		envOutput, err := t.runner.RunShell(ctx, envCmd)
		if err == nil && len(envOutput) > 0 {
			allOutput.WriteString("===== 环境变量 =====\n")
			allOutput.WriteString(string(envOutput))
			allOutput.WriteString("\n\n")
		}

		// 尝试获取打开的文件
		filesCmd := fmt.Sprintf("lsof -p %s 2>/dev/null", pidStr)
		filesOutput, err := t.runner.RunShell(ctx, filesCmd)
		if err == nil && len(filesOutput) > 0 {
			allOutput.WriteString("===== 打开的文件 =====\n")
			allOutput.WriteString(string(filesOutput))
		}

		return ProcInfoResult{
			PID:      pid,
			Content:  allOutput.String(),
			Command:  "多个命令组合",
			InfoType: InfoTypeAll,
		}, nil

	default:
		cmdStr = fmt.Sprintf("ps -p %s -o pid,ppid,user,stat,pcpu,pmem,comm,args", pidStr)
	}

	output, err := t.runner.RunShell(ctx, cmdStr)
	if err != nil {
		t.logger.Error("执行命令失败", "command", cmdStr, "error", err, "pid", pid)
		return ProcInfoResult{}, fmt.Errorf("获取进程信息失败 (PID=%d): %w", pid, err)
	}

	// 检查输出，如果只有标题行，说明PID不存在
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) <= 1 && infoType != InfoTypeEnvironment && infoType != InfoTypeFiles {
		return ProcInfoResult{}, fmt.Errorf("找不到PID为 %d 的进程", pid)
	}

	return ProcInfoResult{
		PID:      pid,
		Content:  string(output),
		Command:  cmdStr,
		InfoType: infoType,
	}, nil
}

// getProcessInfoByName 根据名称获取进程信息
func (t *ProcInfoTool) getProcessInfoByName(ctx context.Context, name, infoType string, count int) (ProcInfoResult, error) {
	// 首先，根据名称获取匹配的PID列表
	findPIDCmd := fmt.Sprintf("pgrep -f %s", name)
	output, err := t.runner.RunShell(ctx, findPIDCmd)
	if err != nil {
		// 尝试使用ps grep组合
		findPIDCmd = fmt.Sprintf("ps aux | grep '%s' | grep -v grep | awk '{print $2}'", name)
		output, err = t.runner.RunShell(ctx, findPIDCmd)

		if err != nil || len(strings.TrimSpace(string(output))) == 0 {
			t.logger.Error("查找进程失败", "name", name, "error", err)
			return ProcInfoResult{}, fmt.Errorf("找不到名称包含 '%s' 的进程", name)
		}
	}

	// 解析PID列表
	pidStrs := strings.Split(strings.TrimSpace(string(output)), "\n")
	pids := make([]int, 0, len(pidStrs))
	for _, pidStr := range pidStrs {
		if pidStr == "" {
			continue
		}
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}

	if len(pids) == 0 {
		return ProcInfoResult{}, fmt.Errorf("找不到名称包含 '%s' 的进程", name)
	}

	// 限制处理的进程数
	if len(pids) > count {
		pids = pids[:count]
	}

	// 对于单个进程，直接返回详细信息
	if len(pids) == 1 {
		return t.getProcessInfoByPID(ctx, pids[0], infoType)
	}

	// 对于多个进程，获取基本信息并组合
	var pidList strings.Builder
	for i, pid := range pids {
		if i > 0 {
			pidList.WriteString(",")
		}
		pidList.WriteString(strconv.Itoa(pid))
	}

	var cmdStr string
	switch infoType {
	case InfoTypeBasic:
		cmdStr = fmt.Sprintf("ps -p %s -o pid,ppid,user,stat,pcpu,pmem,comm,args", pidList.String())
	case InfoTypeDetailed:
		cmdStr = fmt.Sprintf("ps -p %s -f", pidList.String())
	default:
		cmdStr = fmt.Sprintf("ps -p %s -o pid,ppid,user,stat,pcpu,pmem,comm,args", pidList.String())
	}

	output, err = t.runner.RunShell(ctx, cmdStr)
	if err != nil {
		t.logger.Error("执行命令失败", "command", cmdStr, "error", err)
		return ProcInfoResult{}, fmt.Errorf("获取进程信息失败: %w", err)
	}

	return ProcInfoResult{
		Name:         name,
		Content:      string(output),
		Command:      cmdStr,
		InfoType:     infoType,
		ProcessCount: len(pids),
	}, nil
}

// createOutputParameters 创建输出参数定义
func (t *ProcInfoTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	pidDesc := map[string]string{
		"en": "Process ID",
		"zh": "进程ID",
	}

	nameDesc := map[string]string{
		"en": "Process name",
		"zh": "进程名称",
	}

	contentDesc := map[string]string{
		"en": "Process information content",
		"zh": "进程信息内容",
	}

	commandDesc := map[string]string{
		"en": "The command executed",
		"zh": "执行的命令",
	}

	infoTypeDesc := map[string]string{
		"en": "Type of information returned",
		"zh": "返回的信息类型",
	}

	processCountDesc := map[string]string{
		"en": "Number of processes matched",
		"zh": "匹配的进程数量",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "pid",
			Type:        toolcore.ParamTypeInteger,
			Description: pidDesc,
			Required:    false,
		},
		{
			Name:        "name",
			Type:        toolcore.ParamTypeString,
			Description: nameDesc,
			Required:    false,
		},
		{
			Name:        "content",
			Type:        toolcore.ParamTypeString,
			Description: contentDesc,
			Required:    true,
		},
		{
			Name:        "command",
			Type:        toolcore.ParamTypeString,
			Description: commandDesc,
			Required:    true,
		},
		{
			Name:        "info_type",
			Type:        toolcore.ParamTypeString,
			Description: infoTypeDesc,
			Required:    false,
		},
		{
			Name:        "process_count",
			Type:        toolcore.ParamTypeInteger,
			Description: processCountDesc,
			Required:    false,
		},
	}
}
