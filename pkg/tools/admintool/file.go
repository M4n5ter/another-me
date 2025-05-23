package admintool

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

const (
	// OperationRead 读取文件内容
	OperationRead = "read"
	// OperationWrite 写入文件内容
	OperationWrite = "write"
	// OperationCreate 创建文件
	OperationCreate = "create"
	// OperationDelete 删除文件
	OperationDelete = "delete"
	// OperationList 列出目录内容
	OperationList = "list"
	// OperationExists 检查文件是否存在
	OperationExists = "exists"
	// OperationMove 移动文件
	OperationMove = "move"
	// OperationCopy 复制文件
	OperationCopy = "copy"
)

// FileTool 实现 toolcore.Tool 接口，用于文件操作
type FileTool struct {
	fs      FileSystem
	i18nMgr *i18n.Manager
	runner  CommandRunner // 命令执行器
	logger  *slog.Logger
}

// FileArgs 定义了 FileTool 的参数
type FileArgs struct {
	Path            string `json:"path"`                       // 文件或目录路径
	Operation       string `json:"operation"`                  // 操作类型: read, write, create, delete, list, exists, move, copy
	Content         string `json:"content,omitempty"`          // 写入的内容
	IsDir           bool   `json:"is_dir,omitempty"`           // 是否是目录
	Recursive       bool   `json:"recursive,omitempty"`        // 是否递归操作
	Lines           *int   `json:"lines,omitempty"`            // 读取的行数，默认所有行
	FindText        string `json:"find_text,omitempty"`        // 要查找的文本
	ReplaceText     string `json:"replace_text,omitempty"`     // 替换为的文本
	DestinationPath string `json:"destination_path,omitempty"` // 移动或复制的目标路径
}

// FileResult 定义了成功结果的结构
type FileResult struct {
	Path        string   `json:"path"`                  // 文件或目录路径
	Operation   string   `json:"operation"`             // 执行的操作
	Content     string   `json:"content,omitempty"`     // 文件内容
	Exists      bool     `json:"exists,omitempty"`      // 文件是否存在
	IsDir       bool     `json:"is_dir,omitempty"`      // 是否是目录
	Permissions string   `json:"permissions,omitempty"` // 文件权限
	Size        int64    `json:"size,omitempty"`        // 文件大小
	ModTime     string   `json:"mod_time,omitempty"`    // 修改时间
	Files       []string `json:"files,omitempty"`       // 目录列表内容
	Message     string   `json:"message,omitempty"`     // 操作消息
	LineCount   *int     `json:"line_count,omitempty"`  // 行数
	Destination string   `json:"destination,omitempty"` // For move/copy
}

// NewFileTool 创建一个新的 FileTool 实例
func NewFileTool(i18nMgr *i18n.Manager) *FileTool {
	fs := &RealFileSystem{}
	return &FileTool{
		fs:      fs,
		i18nMgr: i18nMgr,
		runner:  NewRealCommandRunner(),
		logger:  slog.Default().WithGroup("file_tool"),
	}
}

// NewFileToolWithFs 创建一个新的 FileTool 实例，允许设置自定义的 FileSystem
func NewFileToolWithFs(i18nMgr *i18n.Manager, fs FileSystem) *FileTool {
	return &FileTool{
		fs:      fs,
		i18nMgr: i18nMgr,
		runner:  NewRealCommandRunner(),
		logger:  slog.Default().WithGroup("file_tool"),
	}
}

// NewFileToolWithDeps 创建一个新的 FileTool 实例，允许设置自定义的依赖项
func NewFileToolWithDeps(i18nMgr *i18n.Manager, fs FileSystem, runner CommandRunner) *FileTool {
	return &FileTool{
		fs:      fs,
		i18nMgr: i18nMgr,
		runner:  runner,
		logger:  slog.Default().WithGroup("file_tool"),
	}
}

var _ toolcore.Tool = (*FileTool)(nil)

// Schema 返回工具的模式定义
func (t *FileTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.admin.file.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.admin.file.name", nil)
	}

	operationEnum := []any{OperationRead, OperationWrite, OperationCreate, OperationDelete, OperationList, OperationExists, OperationMove, OperationCopy}

	inputParams := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "path", toolcore.ParamTypeString, true, nil, t.i18nMgr.T(ctx, "tool.admin.file.arg.path", nil), nil),
		common.CreateParamDef(ctx, t.i18nMgr, "operation", toolcore.ParamTypeString, true, option.Some(operationEnum), t.i18nMgr.T(ctx, "tool.admin.file.arg.operation", nil), nil),
		common.CreateParamDef(ctx, t.i18nMgr, "destination_path", toolcore.ParamTypeString, false, nil, t.i18nMgr.T(ctx, "tool.admin.file.arg.destination_path", nil), nil),
		common.CreateParamDef(ctx, t.i18nMgr, "content", toolcore.ParamTypeString, false, nil, t.i18nMgr.T(ctx, "tool.admin.file.arg.content", nil), nil),
		common.CreateParamDef(ctx, t.i18nMgr, "is_dir", toolcore.ParamTypeBoolean, false, nil, t.i18nMgr.T(ctx, "tool.admin.file.arg.is_dir", nil), nil),
		common.CreateParamDef(ctx, t.i18nMgr, "recursive", toolcore.ParamTypeBoolean, false, nil, t.i18nMgr.T(ctx, "tool.admin.file.arg.recursive", nil), nil),
		common.CreateParamDef(ctx, t.i18nMgr, "lines", toolcore.ParamTypeInteger, false, nil, t.i18nMgr.T(ctx, "tool.admin.file.arg.lines", nil), nil),
		common.CreateParamDef(ctx, t.i18nMgr, "find_text", toolcore.ParamTypeString, false, nil, t.i18nMgr.T(ctx, "tool.admin.file.arg.find_text", nil), nil),
		common.CreateParamDef(ctx, t.i18nMgr, "replace_text", toolcore.ParamTypeString, false, nil, t.i18nMgr.T(ctx, "tool.admin.file.arg.replace_text", nil), nil),
	}

	return toolcore.ToolSchema{
		Name:             "file",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParams,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 执行文件操作
func (t *FileTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args FileArgs
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("无效的 JSON 输入", "error", err)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	if args.Path == "" {
		t.logger.Error("必须提供文件或目录路径")
		return "", fmt.Errorf("必须提供文件或目录路径")
	}

	args.Path = filepath.Clean(args.Path)

	var result FileResult
	var err error

	switch args.Operation {
	case OperationRead:
		result, err = t.readFile(ctx, args)
	case OperationWrite:
		result, err = t.writeFile(ctx, args)
	case OperationCreate:
		result, err = t.createFile(ctx, args)
	case OperationDelete:
		result, err = t.deleteFile(ctx, args)
	case OperationList:
		result, err = t.listDirectory(ctx, args)
	case OperationExists:
		result, err = t.fileExists(ctx, args)
	case OperationMove:
		result, err = t.moveFile(ctx, args)
	case OperationCopy:
		result, err = t.copyFile(ctx, args)
	default:
		return "", fmt.Errorf("不支持的操作类型: %s", args.Operation)
	}

	if err != nil {
		t.logger.Error("文件操作失败", "error", err)
		return "", err
	}

	resultJSON, err := json.MarshalToString(result)
	if err != nil {
		t.logger.Error("序列化结果失败", "error", err)
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return resultJSON, nil
}

// readFile 读取文件内容
func (t *FileTool) readFile(ctx context.Context, args FileArgs) (FileResult, error) {
	if args.Lines != nil && *args.Lines != 0 {
		return t.readFileByLines(ctx, args)
	}

	info, err := t.fs.Stat(args.Path)
	if err != nil {
		if t.fs.IsNotExist(err) {
			return FileResult{}, fmt.Errorf("文件不存在: %s", args.Path)
		}
		t.logger.Error("获取文件信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取文件信息失败: %w", err)
	}

	if info.IsDir() {
		return FileResult{}, fmt.Errorf("指定的路径是一个目录，不是文件: %s", args.Path)
	}

	if args.Lines != nil && *args.Lines != 0 {
		return t.readFileByLines(ctx, args)
	}

	if info.Size() > 10*1024*1024 {
		return FileResult{}, fmt.Errorf("文件过大 (%.2f MB)，请使用 lines 参数限制读取行数", float64(info.Size())/(1024*1024))
	}

	content, err := t.fs.ReadFile(args.Path)
	if err != nil {
		t.logger.Error("读取文件失败", "error", err)
		return FileResult{}, fmt.Errorf("读取文件失败: %w", err)
	}

	lines := strings.Count(string(content), "\n")

	return FileResult{
		Path:      args.Path,
		Operation: OperationRead,
		Content:   string(content),
		Size:      info.Size(),
		LineCount: &lines,
		ModTime:   info.ModTime().String(),
	}, nil
}

// readFileByLines 通过head/tail命令读取文件的指定行数
func (t *FileTool) readFileByLines(ctx context.Context, args FileArgs) (FileResult, error) {
	var cmdStr string
	if *args.Lines > 0 {
		cmdStr = fmt.Sprintf("head -n %d '%s'", *args.Lines, args.Path)
	} else {
		cmdStr = fmt.Sprintf("tail -n %d '%s'", *args.Lines*-1, args.Path)
	}

	output, err := t.runner.RunShell(ctx, cmdStr)
	if err != nil {
		t.logger.Error("读取文件失败", "error", err)
		return FileResult{}, fmt.Errorf("读取文件失败: %w", err)
	}

	content := string(output)
	lines := strings.Count(content, "\n")

	return FileResult{
		Path:      args.Path,
		Operation: OperationRead,
		Content:   content,
		Size:      int64(len(content)),
		LineCount: &lines,
		ModTime:   time.Now().String(),
	}, nil
}

// writeFile 写入或更新文件内容
func (t *FileTool) writeFile(_ context.Context, args FileArgs) (FileResult, error) {
	if args.FindText != "" {
		content, err := t.fs.ReadFile(args.Path)
		if err != nil {
			t.logger.Error("读取文件失败", "error", err)
			return FileResult{}, fmt.Errorf("读取文件失败: %w", err)
		}

		newContent := strings.ReplaceAll(string(content), args.FindText, args.ReplaceText)

		err = t.fs.WriteFile(args.Path, []byte(newContent), 0o644)
		if err != nil {
			t.logger.Error("写入文件失败", "error", err)
			return FileResult{}, fmt.Errorf("写入文件失败: %w", err)
		}

		info, err := t.fs.Stat(args.Path)
		if err != nil {
			t.logger.Error("获取文件信息失败", "error", err)
			return FileResult{}, fmt.Errorf("获取文件信息失败: %w", err)
		}

		return FileResult{
			Path:      args.Path,
			Operation: OperationWrite,
			Message:   fmt.Sprintf("文件更新成功，替换了文本'%s'", args.FindText),
			Size:      info.Size(),
			ModTime:   info.ModTime().String(),
		}, nil
	}

	if args.Content == "" {
		return FileResult{}, fmt.Errorf("更新文件需要提供content参数")
	}

	err := t.fs.WriteFile(args.Path, []byte(args.Content), 0o644)
	if err != nil {
		t.logger.Error("写入文件失败", "error", err)
		return FileResult{}, fmt.Errorf("写入文件失败: %w", err)
	}

	info, err := t.fs.Stat(args.Path)
	if err != nil {
		t.logger.Error("获取文件信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return FileResult{
		Path:      args.Path,
		Operation: OperationWrite,
		Message:   "文件更新成功",
		Size:      info.Size(),
		ModTime:   info.ModTime().String(),
	}, nil
}

// createFile 创建文件或目录
func (t *FileTool) createFile(ctx context.Context, args FileArgs) (FileResult, error) {
	if args.IsDir {
		var cmdStr string
		if args.Recursive {
			cmdStr = fmt.Sprintf("mkdir -p '%s'", args.Path)
		} else {
			cmdStr = fmt.Sprintf("mkdir '%s'", args.Path)
		}

		_, err := t.runner.RunShell(ctx, cmdStr)
		if err != nil {
			t.logger.Error("创建目录失败", "error", err)
			return FileResult{}, fmt.Errorf("创建目录失败: %w", err)
		}

		return FileResult{
			Path:      args.Path,
			Operation: OperationCreate,
			IsDir:     true,
			Message:   "目录创建成功",
		}, nil
	}

	parent := filepath.Dir(args.Path)
	if parent != "." && parent != "/" {
		mkdirCmd := fmt.Sprintf("mkdir -p '%s'", parent)
		_, err := t.runner.RunShell(ctx, mkdirCmd)
		if err != nil {
			t.logger.Error("创建父目录失败", "error", err)
			return FileResult{}, fmt.Errorf("创建父目录失败: %w", err)
		}
	}

	_, err := t.runner.Run(ctx, "touch", args.Path)
	if err != nil {
		t.logger.Error("创建文件失败", "error", err)
		return FileResult{}, fmt.Errorf("创建文件失败: %w", err)
	}

	info, err := t.fs.Stat(args.Path)
	if err != nil {
		t.logger.Error("获取文件信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取文件信息失败: %w", err)
	}

	return FileResult{
		Path:      args.Path,
		Operation: OperationCreate,
		IsDir:     false,
		Message:   "文件创建成功",
		Size:      info.Size(),
		ModTime:   info.ModTime().String(),
	}, nil
}

// deleteFile 删除文件或目录
func (t *FileTool) deleteFile(_ context.Context, args FileArgs) (FileResult, error) {
	info, err := t.fs.Stat(args.Path)
	if err != nil {
		if t.fs.IsNotExist(err) {
			return FileResult{}, fmt.Errorf("文件或目录不存在: %s", args.Path)
		}
		t.logger.Error("获取文件信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取文件信息失败: %w", err)
	}

	isDir := info.IsDir()

	if isDir && !args.Recursive {
		entries, err := t.fs.ReadDir(args.Path)
		if err != nil {
			t.logger.Error("读取目录失败", "error", err)
			return FileResult{}, fmt.Errorf("读取目录失败: %w", err)
		}

		if len(entries) > 0 {
			t.logger.Error("目录不为空，需要设置recursive=true才能删除非空目录")
			return FileResult{}, fmt.Errorf("目录不为空，需要设置recursive=true才能删除非空目录")
		}
	}

	if isDir && args.Recursive {
		err = t.fs.RemoveAll(args.Path)
	} else {
		err = t.fs.Remove(args.Path)
	}

	if err != nil {
		t.logger.Error("删除失败", "error", err)
		return FileResult{}, fmt.Errorf("删除失败: %w", err)
	}

	message := "文件删除成功"
	if isDir {
		message = "目录删除成功"
	}

	return FileResult{
		Path:      args.Path,
		Operation: OperationDelete,
		IsDir:     isDir,
		Message:   message,
	}, nil
}

// listDirectory 列出目录内容，支持递归
func (t *FileTool) listDirectory(_ context.Context, args FileArgs) (FileResult, error) {
	var files []string

	info, err := t.fs.Stat(args.Path)
	if err != nil {
		if t.fs.IsNotExist(err) {
			return FileResult{}, fmt.Errorf("目录不存在: %s", args.Path)
		}
		t.logger.Error("获取目录信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取目录信息失败: %w", err)
	}
	if !info.IsDir() {
		t.logger.Error("路径不是目录", "path", args.Path)
		return FileResult{}, fmt.Errorf("路径不是目录: %s", args.Path)
	}

	if args.Recursive {
		err = filepath.WalkDir(args.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				t.logger.Error("递归列出目录失败", "error", err)
				return err
			}
			if path != args.Path {
				relPath, err := filepath.Rel(args.Path, path)
				if err != nil {
					t.logger.Error("获取相对路径失败", "error", err)
					return fmt.Errorf("获取相对路径失败: %w", err)
				}
				if d.IsDir() {
					files = append(files, relPath+"/")
				} else {
					files = append(files, relPath)
				}
			}
			return nil
		})
		if err != nil {
			t.logger.Error("递归列出目录失败", "error", err)
			return FileResult{}, fmt.Errorf("递归列出目录失败: %w", err)
		}
	} else {
		dirEntries, err := t.fs.ReadDir(args.Path)
		if err != nil {
			t.logger.Error("读取目录内容失败", "error", err)
			return FileResult{}, fmt.Errorf("读取目录内容失败: %w", err)
		}
		for _, entry := range dirEntries {
			if entry.IsDir() {
				files = append(files, entry.Name()+"/")
			} else {
				files = append(files, entry.Name())
			}
		}
	}

	return FileResult{
		Path:      args.Path,
		Operation: OperationList,
		IsDir:     true,
		Files:     files,
		ModTime:   info.ModTime().String(),
		Size:      info.Size(),
	}, nil
}

// fileExists 检查文件或目录是否存在
func (t *FileTool) fileExists(_ context.Context, args FileArgs) (FileResult, error) {
	info, err := t.fs.Stat(args.Path)
	if err != nil {
		if t.fs.IsNotExist(err) {
			return FileResult{
				Path:      args.Path,
				Operation: OperationExists,
				Exists:    false,
				Message:   "文件或目录不存在",
			}, nil
		}
		t.logger.Error("获取文件信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取文件信息失败: %w", err)
	}

	isDir := info.IsDir()
	mode := info.Mode().String()

	var fileType string
	if isDir {
		fileType = "目录"
	} else {
		fileType = "文件"
	}

	return FileResult{
		Path:        args.Path,
		Operation:   OperationExists,
		Exists:      true,
		IsDir:       isDir,
		Size:        info.Size(),
		ModTime:     info.ModTime().String(),
		Permissions: mode,
		Message:     fmt.Sprintf("%s 存在，类型: %s", args.Path, fileType),
	}, nil
}

// moveFile 移动文件或目录
func (t *FileTool) moveFile(_ context.Context, args FileArgs) (FileResult, error) {
	if args.DestinationPath == "" {
		return FileResult{}, fmt.Errorf("移动文件需要提供destination_path参数")
	}

	_, err := t.fs.Stat(args.Path)
	if err != nil {
		if t.fs.IsNotExist(err) {
			return FileResult{}, fmt.Errorf("文件或目录不存在: %s", args.Path)
		}
		t.logger.Error("获取文件信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取文件信息失败: %w", err)
	}

	_, err = t.fs.Stat(args.DestinationPath)
	if err != nil && !t.fs.IsNotExist(err) {
		return FileResult{}, fmt.Errorf("获取目标路径信息失败: %w", err)
	}

	err = t.fs.Rename(args.Path, args.DestinationPath)
	if err != nil {
		return FileResult{}, fmt.Errorf("移动文件失败: %w", err)
	}

	return FileResult{
		Path:        args.Path,
		Operation:   OperationMove,
		Destination: args.DestinationPath,
		Message:     "文件移动成功",
	}, nil
}

// copyFile 复制文件或目录
func (t *FileTool) copyFile(_ context.Context, args FileArgs) (FileResult, error) {
	if args.DestinationPath == "" {
		return FileResult{}, fmt.Errorf("复制文件需要提供destination_path参数")
	}

	info, err := t.fs.Stat(args.Path)
	if err != nil {
		if t.fs.IsNotExist(err) {
			return FileResult{}, fmt.Errorf("文件或目录不存在: %s", args.Path)
		}
		t.logger.Error("获取文件信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取文件信息失败: %w", err)
	}

	_, err = t.fs.Stat(args.DestinationPath)
	if err != nil && !t.fs.IsNotExist(err) {
		t.logger.Error("获取目标路径信息失败", "error", err)
		return FileResult{}, fmt.Errorf("获取目标路径信息失败: %w", err)
	}

	if info.IsDir() {
		err = t.fs.CopyDir(args.Path, args.DestinationPath)
	} else {
		err = t.fs.CopyFile(args.Path, args.DestinationPath)
	}
	if err != nil {
		t.logger.Error("复制文件失败", "error", err)
		return FileResult{}, fmt.Errorf("复制文件失败: %w", err)
	}

	return FileResult{
		Path:        args.Path,
		Operation:   OperationCopy,
		Destination: args.DestinationPath,
		Message:     "文件复制成功",
	}, nil
}

func (t *FileTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	outputParams := []toolcore.ParameterDefinition{
		{
			Name: "path",
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "File or directory path",
				"zh": "文件或目录路径",
			},
			Required: true,
		},
		{
			Name: "operation",
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "Operation performed",
				"zh": "执行的操作",
			},
			Required: true,
		},
		{
			Name: "content",
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "File content (for read operations) or command output (for list)",
				"zh": "文件内容（对于读取操作）或命令输出（对于列表）",
			},
			Required: false,
		},
		{
			Name: "exists",
			Type: toolcore.ParamTypeBoolean,
			Description: map[string]string{
				"en": "Whether the file or directory exists",
				"zh": "文件或目录是否存在",
			},
			Required: false,
		},
		{
			Name: "is_dir",
			Type: toolcore.ParamTypeBoolean,
			Description: map[string]string{
				"en": "Is the path a directory",
				"zh": "路径是否为目录",
			},
			Required: false,
		},
		{
			Name: "permissions",
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "File permissions",
				"zh": "文件权限",
			},
			Required: false,
		},
		{
			Name: "size",
			Type: toolcore.ParamTypeInteger,
			Description: map[string]string{
				"en": "File size in bytes",
				"zh": "文件大小（字节）",
			},
			Required: false,
		},
		{
			Name: "mod_time",
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "Modification time",
				"zh": "修改时间",
			},
			Required: false,
		},
		{
			Name: "files",
			Type: toolcore.ParamTypeArray,
			Description: map[string]string{
				"en": "List of files in the directory (for list operation)",
				"zh": "目录中的文件列表（对于列表操作）",
			},
			Required: false,
			Items: option.Some(toolcore.ParameterDefinition{
				Type: toolcore.ParamTypeString,
			}),
		},
		{
			Name: "message",
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "General success/info message",
				"zh": "一般成功/信息消息",
			},
			Required: false,
		},
		{
			Name: "line_count",
			Type: toolcore.ParamTypeInteger,
			Description: map[string]string{
				"en": "Number of lines read (for read operation)",
				"zh": "读取的行数（对于读取操作）",
			},
			Required: false,
		},
		{
			Name: "destination",
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "Destination path for move or copy operations",
				"zh": "移动或复制操作的目标路径",
			},
			Required: false,
		},
	}
	return outputParams
}
