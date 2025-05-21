package admintool

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// 资源类型常量
const (
	ResourceCPU    = "cpu"
	ResourceMemory = "memory"
	ResourceDisk   = "disk"
	ResourceAll    = "all"
)

// 输出格式常量
const (
	FormatText = "text"
	FormatJSON = "json"
)

// SysstatTool 实现 toolcore.Tool 接口，用于获取系统资源状态
type SysstatTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
	runner  CommandRunner // 命令执行器
}

// SysstatArgs 定义了 SysstatTool 的参数
type SysstatArgs struct {
	Resource string `json:"resource"`         // 资源类型: cpu, memory, disk, all
	Format   string `json:"format,omitempty"` // 输出格式: text, json
}

// SysstatResult 定义了成功结果的结构
type SysstatResult struct {
	ResourceType string `json:"resource_type"` // 资源类型
	Content      string `json:"content"`       // 资源状态内容
	Command      string `json:"command"`       // 执行的命令
}

// NewSysstatTool 创建一个新的 SysstatTool 实例
func NewSysstatTool(i18nMgr *i18n.Manager) *SysstatTool {
	return &SysstatTool{
		logger:  slog.Default().WithGroup("sysstat_tool"),
		i18nMgr: i18nMgr,
		runner:  NewRealCommandRunner(),
	}
}

// NewSysstatToolWithRunner 创建一个使用自定义命令执行器的SysstatTool实例（用于测试）
func NewSysstatToolWithRunner(i18nMgr *i18n.Manager, runner CommandRunner) *SysstatTool {
	return &SysstatTool{
		logger:  slog.Default().WithGroup("sysstat_tool"),
		i18nMgr: i18nMgr,
		runner:  runner,
	}
}

var _ toolcore.Tool = (*SysstatTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *SysstatTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.admin.sysstat.description", nil)
		localizedNames[lang] = "System Status"
	}

	// 定义资源类型枚举
	resourceTypes := []any{ResourceCPU, ResourceMemory, ResourceDisk, ResourceAll}
	formatTypes := []any{FormatText, FormatJSON}

	// 构建参数定义
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "resource", toolcore.ParamTypeString, true, Some(resourceTypes), "tool.admin.sysstat.arg.resource", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "format", toolcore.ParamTypeString, false, Some(formatTypes), "tool.admin.sysstat.arg.format", nil),
	}

	// 返回工具的完整模式
	return toolcore.ToolSchema{
		Name:             "sysstat",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *SysstatTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args SysstatArgs
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	if args.Resource == "" {
		args.Resource = ResourceAll
	}

	if args.Format == "" {
		args.Format = FormatText
	}

	var results []SysstatResult

	// 根据资源类型执行相应的命令
	switch args.Resource {
	case ResourceCPU:
		result, err := t.getCPUStats(ctx)
		if err != nil {
			return "", err
		}
		results = append(results, result)
	case ResourceMemory:
		result, err := t.getMemoryStats(ctx)
		if err != nil {
			return "", err
		}
		results = append(results, result)
	case ResourceDisk:
		result, err := t.getDiskStats(ctx)
		if err != nil {
			return "", err
		}
		results = append(results, result)
	case ResourceAll:
		// 获取所有资源的状态
		cpuResult, err := t.getCPUStats(ctx)
		if err != nil {
			t.logger.Warn("获取CPU状态失败", "error", err)
		} else {
			results = append(results, cpuResult)
		}

		memResult, err := t.getMemoryStats(ctx)
		if err != nil {
			t.logger.Warn("获取内存状态失败", "error", err)
		} else {
			results = append(results, memResult)
		}

		diskResult, err := t.getDiskStats(ctx)
		if err != nil {
			t.logger.Warn("获取磁盘状态失败", "error", err)
		} else {
			results = append(results, diskResult)
		}
	default:
		return "", fmt.Errorf("不支持的资源类型: %s", args.Resource)
	}

	if len(results) == 0 {
		return "", fmt.Errorf("未能获取任何资源状态")
	}

	// 如果只获取一种资源，直接返回该资源的结果
	if len(results) == 1 {
		resultJSON, err := json.Marshal(results[0])
		if err != nil {
			t.logger.Error("序列化结果失败", "error", err)
			return "", fmt.Errorf("序列化结果失败: %w", err)
		}
		return string(resultJSON), nil
	}

	// 否则，返回所有资源的结果数组
	resultJSON, err := json.Marshal(results)
	if err != nil {
		t.logger.Error("序列化结果失败", "error", err)
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return string(resultJSON), nil
}

// getCPUStats 获取CPU状态
func (t *SysstatTool) getCPUStats(ctx context.Context) (SysstatResult, error) {
	// 使用 mpstat 或 top 命令获取 CPU 使用情况
	// 尝试使用 mpstat
	output, err := t.runner.Run(ctx, "mpstat")
	if err == nil {
		return SysstatResult{
			ResourceType: ResourceCPU,
			Content:      string(output),
			Command:      "mpstat",
		}, nil
	}

	// mpstat 失败，尝试使用 top 命令批处理模式
	output, err = t.runner.Run(ctx, "top", "-bn1")
	if err == nil {
		// 提取 top 输出中的 CPU 相关行
		lines := strings.Split(string(output), "\n")
		var cpuLines []string
		for _, line := range lines {
			if strings.Contains(line, "Cpu") {
				cpuLines = append(cpuLines, line)
			}
		}
		return SysstatResult{
			ResourceType: ResourceCPU,
			Content:      strings.Join(cpuLines, "\n"),
			Command:      "top -bn1",
		}, nil
	}

	// 如果 top 也失败，尝试使用 ps 命令
	output, err = t.runner.Run(ctx, "ps", "aux", "--sort", "-%cpu")
	if err != nil {
		return SysstatResult{}, fmt.Errorf("获取CPU状态失败: %w", err)
	}

	return SysstatResult{
		ResourceType: ResourceCPU,
		Content:      string(output),
		Command:      "ps aux --sort -%cpu",
	}, nil
}

// getMemoryStats 获取内存状态
func (t *SysstatTool) getMemoryStats(ctx context.Context) (SysstatResult, error) {
	// 使用 free 命令获取内存使用情况
	output, err := t.runner.Run(ctx, "free", "-h")
	if err == nil {
		return SysstatResult{
			ResourceType: ResourceMemory,
			Content:      string(output),
			Command:      "free -h",
		}, nil
	}

	// free 失败，尝试使用 vmstat
	output, err = t.runner.Run(ctx, "vmstat")
	if err == nil {
		return SysstatResult{
			ResourceType: ResourceMemory,
			Content:      string(output),
			Command:      "vmstat",
		}, nil
	}

	// 如果 vmstat 也失败，尝试读取 /proc/meminfo
	output, err = t.runner.Run(ctx, "cat", "/proc/meminfo")
	if err != nil {
		return SysstatResult{}, fmt.Errorf("获取内存状态失败: %w", err)
	}

	return SysstatResult{
		ResourceType: ResourceMemory,
		Content:      string(output),
		Command:      "cat /proc/meminfo",
	}, nil
}

// getDiskStats 获取磁盘状态
func (t *SysstatTool) getDiskStats(ctx context.Context) (SysstatResult, error) {
	// 使用 df 命令获取磁盘使用情况
	output, err := t.runner.Run(ctx, "df", "-h")
	if err != nil {
		return SysstatResult{}, fmt.Errorf("获取磁盘状态失败: %w", err)
	}

	return SysstatResult{
		ResourceType: ResourceDisk,
		Content:      string(output),
		Command:      "df -h",
	}, nil
}

// createOutputParameters 创建输出参数定义
func (t *SysstatTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	resourceTypeDesc := map[string]string{
		"en": "Type of resource (cpu, memory, disk)",
		"zh": "资源类型(cpu, memory, disk)",
	}

	contentDesc := map[string]string{
		"en": "Resource status content",
		"zh": "资源状态内容",
	}

	commandDesc := map[string]string{
		"en": "The command executed",
		"zh": "执行的命令",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "resource_type",
			Type:        toolcore.ParamTypeString,
			Description: resourceTypeDesc,
			Required:    true,
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
	}
}
