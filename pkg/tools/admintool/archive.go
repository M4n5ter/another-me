package admintool

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// 压缩格式常量
const (
	FormatZip    = "zip"
	FormatTar    = "tar"
	FormatTargz  = "targz"
	FormatTarbz2 = "tarbz2"
	FormatTarxz  = "tarxz"
)

// ArchiveTool 实现 toolcore.Tool 接口，用于文件压缩和解压缩
type ArchiveTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
	runner  CommandRunner // 命令执行器
	fs      FileSystem    // 文件系统操作接口
}

// ArchiveArgs 定义了 ArchiveTool 的参数
type ArchiveArgs struct {
	Operation   string   `json:"operation"`              // 操作类型: compress, extract
	SourcePath  string   `json:"source_path"`            // 源文件/目录路径
	TargetPath  string   `json:"target_path,omitempty"`  // 目标文件/目录路径
	Format      string   `json:"format,omitempty"`       // 压缩格式: zip, tar, targz, tarbz2, tarxz
	SourcePaths []string `json:"source_paths,omitempty"` // 多个源文件/目录路径 (用于压缩多个文件/目录)
	Verbose     bool     `json:"verbose,omitempty"`      // 是否显示详细信息
}

// ArchiveResult 定义了成功结果的结构
type ArchiveResult struct {
	Operation   string   `json:"operation"`              // 执行的操作
	SourcePath  string   `json:"source_path"`            // 源文件/目录路径
	TargetPath  string   `json:"target_path"`            // 目标文件/目录路径
	Format      string   `json:"format"`                 // 使用的格式
	Command     string   `json:"command"`                // 执行的命令
	Message     string   `json:"message"`                // 操作消息
	Output      string   `json:"output"`                 // 命令输出
	SourcePaths []string `json:"source_paths,omitempty"` // 多个源文件/目录路径
}

// NewArchiveTool 创建一个新的 ArchiveTool 实例
func NewArchiveTool(i18nMgr *i18n.Manager) *ArchiveTool {
	return &ArchiveTool{
		logger:  slog.Default().WithGroup("archive_tool"),
		i18nMgr: i18nMgr,
		runner:  NewRealCommandRunner(),
		fs:      NewRealFileSystem(),
	}
}

// NewArchiveToolWithDeps 创建一个使用自定义依赖的ArchiveTool实例（用于测试）
func NewArchiveToolWithDeps(i18nMgr *i18n.Manager, runner CommandRunner, fs FileSystem) *ArchiveTool {
	return &ArchiveTool{
		logger:  slog.Default().WithGroup("archive_tool"),
		i18nMgr: i18nMgr,
		runner:  runner,
		fs:      fs,
	}
}

var _ toolcore.Tool = (*ArchiveTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *ArchiveTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.admin.archive.description", nil)
		localizedNames[lang] = "Archive"
	}

	// 定义操作类型枚举
	operations := []any{"compress", "extract"}
	formats := []any{FormatZip, FormatTar, FormatTargz, FormatTarbz2, FormatTarxz}

	// 构建参数定义
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "operation", toolcore.ParamTypeString, true, Some(operations), "tool.admin.archive.arg.operation", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "source_path", toolcore.ParamTypeString, true, nil, "tool.admin.archive.arg.source_path", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "target_path", toolcore.ParamTypeString, false, nil, "tool.admin.archive.arg.target_path", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "format", toolcore.ParamTypeString, false, Some(formats), "tool.admin.archive.arg.format", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "source_paths", toolcore.ParamTypeArray, false, nil, "tool.admin.archive.arg.source_paths", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "verbose", toolcore.ParamTypeBoolean, false, nil, "tool.admin.archive.arg.verbose", nil),
	}

	// 返回工具的完整模式
	return toolcore.ToolSchema{
		Name:             "archive",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *ArchiveTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args ArchiveArgs
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	// 验证基本参数
	if args.SourcePath == "" && len(args.SourcePaths) == 0 {
		return "", fmt.Errorf("必须提供source_path或source_paths参数")
	}

	// 根据操作类型执行相应的函数
	var result ArchiveResult
	var err error

	switch args.Operation {
	case "compress":
		result, err = t.compressFiles(ctx, args)
	case "extract":
		result, err = t.extractFile(ctx, args)
	default:
		return "", fmt.Errorf("不支持的操作类型: %s", args.Operation)
	}

	if err != nil {
		return "", err
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.logger.Error("序列化结果失败", "error", err)
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return string(resultJSON), nil
}

// compressFiles 压缩文件/目录
func (t *ArchiveTool) compressFiles(ctx context.Context, args ArchiveArgs) (ArchiveResult, error) {
	// 处理参数
	var sourcePaths []string
	if args.SourcePath != "" {
		sourcePaths = append(sourcePaths, args.SourcePath)
	}
	if len(args.SourcePaths) > 0 {
		sourcePaths = append(sourcePaths, args.SourcePaths...)
	}

	// 验证源路径是否全部存在
	for _, path := range sourcePaths {
		path = filepath.Clean(path)
		_, err := t.fs.Stat(path)
		if err != nil {
			return ArchiveResult{}, fmt.Errorf("源路径不存在: %s", path)
		}
	}

	// 确定目标路径
	targetPath := args.TargetPath
	if targetPath == "" {
		// 如果只有一个源路径，使用它的名称作为目标文件名
		if len(sourcePaths) == 1 {
			base := filepath.Base(sourcePaths[0])
			switch args.Format {
			case FormatZip:
				targetPath = base + ".zip"
			case FormatTar:
				targetPath = base + ".tar"
			case FormatTargz:
				targetPath = base + ".tar.gz"
			case FormatTarbz2:
				targetPath = base + ".tar.bz2"
			case FormatTarxz:
				targetPath = base + ".tar.xz"
			default:
				targetPath = base + ".zip" // 默认使用 zip 格式
			}
		} else {
			// 如果有多个源路径，使用 archive.xxx 作为目标文件名
			switch args.Format {
			case FormatZip:
				targetPath = "archive.zip"
			case FormatTar:
				targetPath = "archive.tar"
			case FormatTargz:
				targetPath = "archive.tar.gz"
			case FormatTarbz2:
				targetPath = "archive.tar.bz2"
			case FormatTarxz:
				targetPath = "archive.tar.xz"
			default:
				targetPath = "archive.zip" // 默认使用 zip 格式
			}
		}
	}

	// 确定使用的压缩格式
	format := args.Format
	if format == "" {
		// 从目标路径的扩展名猜测格式
		ext := filepath.Ext(targetPath)
		switch ext {
		case ".zip":
			format = FormatZip
		case ".tar":
			format = FormatTar
		case ".gz":
			if strings.HasSuffix(targetPath, ".tar.gz") {
				format = FormatTargz
			}
		case ".bz2":
			if strings.HasSuffix(targetPath, ".tar.bz2") {
				format = FormatTarbz2
			}
		case ".xz":
			if strings.HasSuffix(targetPath, ".tar.xz") {
				format = FormatTarxz
			}
		default:
			format = FormatZip // 默认使用 zip 格式
		}
	}

	// 构建命令和参数
	var cmdArgs []string
	var cmdStr string
	var output []byte
	var err error

	switch format {
	case FormatZip:
		verboseFlag := ""
		if args.Verbose {
			verboseFlag = "-v"
		}

		// 使用 zip 命令
		cmdArgs = append([]string{verboseFlag, targetPath}, sourcePaths...)
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("zip %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "zip", cmdArgs...)

	case FormatTar:
		verboseFlag := ""
		if args.Verbose {
			verboseFlag = "-v"
		}

		// 使用 tar 命令
		cmdArgs = append([]string{verboseFlag + "cf", targetPath}, sourcePaths...)
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("tar %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "tar", cmdArgs...)

	case FormatTargz:
		verboseFlag := ""
		if args.Verbose {
			verboseFlag = "-v"
		}

		// 使用 tar 命令并使用 gzip 压缩
		cmdArgs = append([]string{verboseFlag + "czf", targetPath}, sourcePaths...)
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("tar %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "tar", cmdArgs...)

	case FormatTarbz2:
		verboseFlag := ""
		if args.Verbose {
			verboseFlag = "-v"
		}

		// 使用 tar 命令并使用 bzip2 压缩
		cmdArgs = append([]string{verboseFlag + "cjf", targetPath}, sourcePaths...)
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("tar %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "tar", cmdArgs...)

	case FormatTarxz:
		verboseFlag := ""
		if args.Verbose {
			verboseFlag = "-v"
		}

		// 使用 tar 命令并使用 xz 压缩
		cmdArgs = append([]string{verboseFlag + "cJf", targetPath}, sourcePaths...)
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("tar %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "tar", cmdArgs...)

	default:
		return ArchiveResult{}, fmt.Errorf("不支持的压缩格式: %s", format)
	}

	// 执行命令
	if err != nil {
		t.logger.Error("执行压缩命令失败", "command", cmdStr, "error", err)
		return ArchiveResult{}, fmt.Errorf("压缩文件失败: %w, 输出: %s", err, string(output))
	}

	return ArchiveResult{
		Operation:   "compress",
		SourcePath:  args.SourcePath,
		SourcePaths: args.SourcePaths,
		TargetPath:  targetPath,
		Format:      format,
		Command:     cmdStr,
		Output:      string(output),
		Message:     fmt.Sprintf("成功创建压缩文件: %s", targetPath),
	}, nil
}

// extractFile 解压缩文件
func (t *ArchiveTool) extractFile(ctx context.Context, args ArchiveArgs) (ArchiveResult, error) {
	// 处理参数
	sourcePath := args.SourcePath
	if sourcePath == "" {
		return ArchiveResult{}, fmt.Errorf("必须提供source_path参数")
	}

	// 确保源文件存在
	sourcePath = filepath.Clean(sourcePath)
	_, err := t.fs.Stat(sourcePath)
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("源文件不存在: %s", sourcePath)
	}

	// 确定目标路径
	targetPath := args.TargetPath
	if targetPath == "" {
		// 默认使用去除扩展名的源文件名
		ext := filepath.Ext(sourcePath)
		targetPath = strings.TrimSuffix(sourcePath, ext)
		if ext == ".gz" || ext == ".bz2" || ext == ".xz" {
			// 对于可能的双扩展名，例如 .tar.gz，再去掉一层扩展名
			targetPath = strings.TrimSuffix(targetPath, filepath.Ext(targetPath))
		}
	}

	// 确保目标目录存在
	err = t.fs.MkdirAll(targetPath, 0o755)
	if err != nil {
		return ArchiveResult{}, fmt.Errorf("创建目标目录失败: %w", err)
	}

	// 确定使用的解压方法
	var cmdStr string
	var format string
	var output []byte
	hasFormat := (args.Format != "")

	// 如果未指定格式，从文件扩展名猜测
	if !hasFormat {
		switch {
		case strings.HasSuffix(sourcePath, ".zip"):
			format = FormatZip
		case strings.HasSuffix(sourcePath, ".tar"):
			format = FormatTar
		case strings.HasSuffix(sourcePath, ".tar.gz") || strings.HasSuffix(sourcePath, ".tgz"):
			format = FormatTargz
		case strings.HasSuffix(sourcePath, ".tar.bz2") || strings.HasSuffix(sourcePath, ".tbz2"):
			format = FormatTarbz2
		case strings.HasSuffix(sourcePath, ".tar.xz") || strings.HasSuffix(sourcePath, ".txz"):
			format = FormatTarxz
		default:
			return ArchiveResult{}, fmt.Errorf("无法从文件名 %s 确定解压方法，请指定format参数", sourcePath)
		}
	} else {
		format = args.Format
	}

	// 配置解压命令
	verboseFlag := ""
	if args.Verbose {
		verboseFlag = "-v"
	}

	switch format {
	case FormatZip:
		// 使用 unzip 命令
		cmdArgs := []string{verboseFlag, sourcePath, "-d", targetPath}
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("unzip %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "unzip", cmdArgs...)

	case FormatTar:
		// 使用 tar 命令
		cmdArgs := []string{verboseFlag + "xf", sourcePath, "-C", targetPath}
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("tar %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "tar", cmdArgs...)

	case FormatTargz:
		// 使用 tar 命令并使用 gzip 解压
		cmdArgs := []string{verboseFlag + "xzf", sourcePath, "-C", targetPath}
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("tar %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "tar", cmdArgs...)

	case FormatTarbz2:
		// 使用 tar 命令并使用 bzip2 解压
		cmdArgs := []string{verboseFlag + "xjf", sourcePath, "-C", targetPath}
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("tar %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "tar", cmdArgs...)

	case FormatTarxz:
		// 使用 tar 命令并使用 xz 解压
		cmdArgs := []string{verboseFlag + "xJf", sourcePath, "-C", targetPath}
		cmdArgs = removeEmpty(cmdArgs)
		cmdStr = fmt.Sprintf("tar %s", strings.Join(cmdArgs, " "))
		output, err = t.runner.Run(ctx, "tar", cmdArgs...)

	default:
		return ArchiveResult{}, fmt.Errorf("不支持的解压格式: %s", format)
	}

	// 执行命令
	if err != nil {
		t.logger.Error("执行解压命令失败", "command", cmdStr, "error", err)
		return ArchiveResult{}, fmt.Errorf("解压文件失败: %w, 输出: %s", err, string(output))
	}

	return ArchiveResult{
		Operation:  "extract",
		SourcePath: sourcePath,
		TargetPath: targetPath,
		Format:     format,
		Command:    cmdStr,
		Output:     string(output),
		Message:    fmt.Sprintf("成功解压文件到: %s", targetPath),
	}, nil
}

// removeEmpty 移除字符串切片中的空字符串
func removeEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

// createOutputParameters 创建输出参数定义
func (t *ArchiveTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	operationDesc := map[string]string{
		"en": "Operation performed",
		"zh": "执行的操作",
	}

	sourcePathDesc := map[string]string{
		"en": "Source file or directory path",
		"zh": "源文件或目录路径",
	}

	targetPathDesc := map[string]string{
		"en": "Target file or directory path",
		"zh": "目标文件或目录路径",
	}

	formatDesc := map[string]string{
		"en": "Archive format",
		"zh": "归档格式",
	}

	commandDesc := map[string]string{
		"en": "The command executed",
		"zh": "执行的命令",
	}

	messageDesc := map[string]string{
		"en": "Operation message",
		"zh": "操作消息",
	}

	outputDesc := map[string]string{
		"en": "Command output",
		"zh": "命令输出",
	}

	sourcePathsDesc := map[string]string{
		"en": "Multiple source paths",
		"zh": "多个源路径",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "operation",
			Type:        toolcore.ParamTypeString,
			Description: operationDesc,
			Required:    true,
		},
		{
			Name:        "source_path",
			Type:        toolcore.ParamTypeString,
			Description: sourcePathDesc,
			Required:    true,
		},
		{
			Name:        "target_path",
			Type:        toolcore.ParamTypeString,
			Description: targetPathDesc,
			Required:    true,
		},
		{
			Name:        "format",
			Type:        toolcore.ParamTypeString,
			Description: formatDesc,
			Required:    true,
		},
		{
			Name:        "command",
			Type:        toolcore.ParamTypeString,
			Description: commandDesc,
			Required:    true,
		},
		{
			Name:        "message",
			Type:        toolcore.ParamTypeString,
			Description: messageDesc,
			Required:    true,
		},
		{
			Name:        "output",
			Type:        toolcore.ParamTypeString,
			Description: outputDesc,
			Required:    true,
		},
		{
			Name:        "source_paths",
			Type:        toolcore.ParamTypeArray,
			Description: sourcePathsDesc,
			Required:    false,
		},
	}
}
