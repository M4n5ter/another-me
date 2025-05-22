package admintool

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// SearchOptions 定义了文件搜索的内部参数。
type SearchOptions struct {
	Path          string         // 搜索的根路径
	NamePattern   string         // 文件名/目录名匹配模式 (支持 glob)
	ContentRegex  *regexp.Regexp // 文件内容正则表达式匹配对象
	MinSize       int64          // 最小文件大小 (字节)
	MaxSize       int64          // 最大文件大小 (字节), 0 表示无限制
	ModTimeAfter  time.Time      // 修改时间晚于此时间点 (如果 IsZero() 则忽略)
	ModTimeBefore time.Time      // 修改时间早于此时间点 (如果 IsZero() 则忽略)
	IsDir         *bool          // nil = 同时搜索文件和目录, true = 仅目录, false = 仅文件
	Recursive     bool           // 是否递归搜索子目录
	MaxDepth      int            // 最大搜索深度, 0 表示无限制 (仅当 Recursive 为 true 时有效)
	CaseSensitive bool           // 内容搜索和名称匹配(如果后端 glob 支持)时是否区分大小写
}

// SearchResult 代表一个单独的搜索结果条目。
type SearchResult struct {
	Path    string    `json:"path"`            // 找到的文件或目录的完整路径
	IsDir   bool      `json:"is_dir"`          // 指示此条目是否为目录
	Size    int64     `json:"size"`            // 文件大小 (以字节为单位)；目录的大小通常为0或特定值
	ModTime time.Time `json:"mod_time"`        // 文件的最后修改时间
	Error   string    `json:"error,omitempty"` // 如果在处理此特定条目时发生错误，则包含错误信息
}

const defaultSearchPath = "." // 默认搜索路径为当前目录

// SearchTool 是用于搜索文件和目录的工具。
type SearchTool struct {
	i18nMgr *i18n.Manager
}

// NewSearchTool 创建一个新的 SearchTool 实例。
func NewSearchTool(i18nMgr *i18n.Manager) toolcore.Tool {
	return &SearchTool{i18nMgr: i18nMgr}
}

// Schema 实现 toolcore.Tool 接口，返回工具的元数据。
func (t *SearchTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.admin.search.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.admin.search.name", nil)
	}

	InputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "path", toolcore.ParamTypeString, false, nil, "tool.admin.search.arg.path", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "name_pattern", toolcore.ParamTypeString, false, nil, "tool.admin.search.arg.name_pattern", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "content_regex", toolcore.ParamTypeString, false, nil, "tool.admin.search.arg.content_regex", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "type_filter", toolcore.ParamTypeString, false, Some([]any{"file", "directory", "any"}), "tool.admin.search.arg.type_filter", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "recursive", toolcore.ParamTypeBoolean, false, nil, "tool.admin.search.arg.recursive", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "max_depth", toolcore.ParamTypeInteger, false, nil, "tool.admin.search.arg.max_depth", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "min_size_bytes", toolcore.ParamTypeInteger, false, nil, "tool.admin.search.arg.min_size_bytes", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "max_size_bytes", toolcore.ParamTypeInteger, false, nil, "tool.admin.search.arg.max_size_bytes", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "modified_after", toolcore.ParamTypeString, false, nil, "tool.admin.search.arg.modified_after", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "modified_before", toolcore.ParamTypeString, false, nil, "tool.admin.search.arg.modified_before", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "case_sensitive_match", toolcore.ParamTypeBoolean, false, nil, "tool.admin.search.arg.case_sensitive_match", nil),
	}

	return toolcore.ToolSchema{
		Name:             "search_file_system",
		Descriptions:     descriptions,
		LocalizedNames:   localizedNames,
		InputParameters:  InputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// InputSearchFileSystem 定义了 search_file_system 工具的输入参数结构。
type InputSearchFileSystem struct {
	Path               Option[string] `json:"path"`                 // 可选；搜索的根路径，默认为当前工作目录 (".").
	NamePattern        Option[string] `json:"name_pattern"`         // 可选；文件名或目录名的匹配模式 (支持 glob 语法，例如 "*.txt", "my_folder/?*.log").
	ContentRegex       Option[string] `json:"content_regex"`        // 可选；用于匹配文件内容的正则表达式 (例如 "error\\s+\\d+").
	TypeFilter         Option[string] `json:"type_filter"`          // 可选；文件类型过滤器: "file" (仅文件), "directory" (仅目录), 或 "any" (文件和目录，默认).
	Recursive          Option[bool]   `json:"recursive"`            // 可选；是否递归搜索子目录，默认为 true.
	MaxDepth           Option[int]    `json:"max_depth"`            // 可选；最大搜索深度 (0 表示无限制深度)，仅当 recursive 为 true 时有效.
	MinSizeBytes       Option[int64]  `json:"min_size_bytes"`       // 可选；文件的最小大小 (字节).
	MaxSizeBytes       Option[int64]  `json:"max_size_bytes"`       // 可选；文件的最大大小 (字节).
	ModifiedAfter      Option[string] `json:"modified_after"`       // 可选；修改时间晚于此ISO 8601/RFC3339 时间戳 (例如 "2023-01-01T15:04:05Z").
	ModifiedBefore     Option[string] `json:"modified_before"`      // 可选；修改时间早于此ISO 8601/RFC3339 时间戳.
	CaseSensitiveMatch Option[bool]   `json:"case_sensitive_match"` // 可选；文件名匹配 (如果glob库支持) 和内容正则匹配是否区分大小写，默认为 false (不区分).
}

// OutputSearchFileSystem 定义了 search_file_system 工具的输出结构。
type OutputSearchFileSystem struct {
	Results []SearchResult `json:"results"` // 包含所有匹配项的列表
}

// Call 实现 toolcore.Tool 接口，执行文件搜索操作。
func (t *SearchTool) Call(ctx context.Context, inputJSON string) (outputJSON string, err error) {
	// 步骤 1: 解析输入 JSON
	var input InputSearchFileSystem
	if err = json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return "", fmt.Errorf("无效的输入 JSON: %w", err) // 返回错误信息：无效的输入JSON
	}

	// 步骤 2: 准备 SearchOptions
	// 2a. 处理路径，如果未提供则使用默认路径
	var path string
	if input.Path.IsSome() {
		path = input.Path.Unwrap()
	} else {
		path = defaultSearchPath // 如果用户未指定路径，则默认为当前目录
	}

	// 2b. 处理名称匹配模式
	var namePattern string
	if input.NamePattern.IsSome() {
		namePattern = input.NamePattern.Unwrap()
	}

	// 2c. 处理内容正则表达式
	var contentRegex *regexp.Regexp
	if input.ContentRegex.IsSome() {
		regexStr := input.ContentRegex.Unwrap()
		if regexStr != "" { // 仅当正则表达式字符串非空时才编译
			var re *regexp.Regexp
			// 根据 CaseSensitiveMatch 决定编译区分或不区分大小写的正则表达式
			if input.CaseSensitiveMatch.OrElse(func() Option[bool] { return Some(false) }).Unwrap() {
				re, err = regexp.Compile(regexStr) // 区分大小写
			} else {
				re, err = regexp.Compile("(?i)" + regexStr) // 不区分大小写 (通过模式前缀实现)
			}
			if err != nil {
				return "", fmt.Errorf("无效的 content_regex: %w", err) // 返回错误：无效的内容正则表达式
			}
			contentRegex = re
		}
	}

	// 2d. 处理时间戳转换
	var modifiedAfterTime time.Time
	if input.ModifiedAfter.IsSome() {
		modAfterStr := input.ModifiedAfter.Unwrap()
		if modAfterStr != "" { // 仅当时间戳字符串非空时解析
			modifiedAfterTime, err = time.Parse(time.RFC3339, modAfterStr)
			if err != nil {
				return "", fmt.Errorf("无效的 modified_after 时间戳 (必须是 RFC3339 格式): %w", err)
			}
		}
	}

	var modifiedBeforeTime time.Time
	if input.ModifiedBefore.IsSome() {
		modBeforeStr := input.ModifiedBefore.Unwrap()
		if modBeforeStr != "" { // 仅当时间戳字符串非空时解析
			modifiedBeforeTime, err = time.Parse(time.RFC3339, modBeforeStr)
			if err != nil {
				return "", fmt.Errorf("无效的 modified_before 时间戳 (必须是 RFC3339 格式): %w", err)
			}
		}
	}

	// 2e. 聚合所有选项到 SearchOptions 结构体
	opts := &SearchOptions{
		Path:          path,
		Recursive:     input.Recursive.OrElse(func() Option[bool] { return Some(true) }).Unwrap(), // 默认为递归
		NamePattern:   namePattern,
		MinSize:       input.MinSizeBytes.OrElse(func() Option[int64] { return Some[int64](0) }).Unwrap(), // 默认为0 (无最小大小限制)
		MaxSize:       input.MaxSizeBytes.OrElse(func() Option[int64] { return Some[int64](0) }).Unwrap(), // 默认为0 (无最大大小限制)
		ModTimeAfter:  modifiedAfterTime,
		ModTimeBefore: modifiedBeforeTime,
		ContentRegex:  contentRegex,
		CaseSensitive: input.CaseSensitiveMatch.OrElse(func() Option[bool] { return Some(false) }).Unwrap(), // 默认为不区分大小写 (用于内容和名称匹配)
	}

	// 2f. 决定 effectiveIsDir (搜索目标是文件、目录还是两者)
	// 优先级:
	// 1. 如果 ContentRegex 存在 (非nil)，则只搜索文件。
	// 2. 否则，如果用户通过 TypeFilter 明确指定了类型 ("file" 或 "directory")，则遵循用户指定。
	// 3. 否则 (ContentRegex 为空，TypeFilter 未指定或为 "any")，
	//    如果存在其他文件特定属性筛选器 (大小、修改时间)，则默认为仅搜索文件。
	// 4. 如果以上条件都不满足，则 effectiveIsDir 保持为 nil (搜索所有匹配名称等条件的文件和目录)。
	var effectiveIsDir *bool
	if contentRegex != nil { // 内容搜索优先，并强制只搜索文件
		isDirFalse := false
		effectiveIsDir = &isDirFalse
	} else {
		if input.TypeFilter.IsSome() {
			val := input.TypeFilter.Unwrap()
			switch val {
			case "file":
				isDirFalse := false
				effectiveIsDir = &isDirFalse
			case "directory":
				isDirTrue := true
				effectiveIsDir = &isDirTrue
				// "any" 或其他值: effectiveIsDir 保持原样 (可能为 nil)
			}
		}
		// 仅当 effectiveIsDir 仍为 nil (即 TypeFilter 不是 "file" 或 "directory"，且无内容正则)
		// 才检查其他文件属性是否暗示仅搜索文件。
		if effectiveIsDir == nil {
			hasOtherFileSpecificFilters := (input.MinSizeBytes.IsSome() ||
				input.MaxSizeBytes.IsSome() ||
				input.ModifiedAfter.IsSome() ||
				input.ModifiedBefore.IsSome())
			if hasOtherFileSpecificFilters {
				isDirFalse := false
				effectiveIsDir = &isDirFalse // 其他文件属性筛选器暗示只搜索文件
			}
			// 若无上述条件，effectiveIsDir 仍为 nil，表示匹配所有类型。
		}
	}
	opts.IsDir = effectiveIsDir

	// 2g. 设置最大深度
	if input.MaxDepth.IsSome() {
		opts.MaxDepth = input.MaxDepth.Unwrap()
	} // 若未设置，opts.MaxDepth 默认为 0 (在 SearchOptions 中定义为无限制或由 SearchFiles 具体处理)

	// 步骤 3: 执行搜索
	results, err := SearchFiles(opts)
	if err != nil {
		return "", fmt.Errorf("文件搜索过程中出错: %w", err)
	}

	// 步骤 4: 格式化并返回输出 JSON
	output := OutputSearchFileSystem{Results: results}
	outputBytes, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("序列化搜索结果为 JSON 时出错: %w", err)
	}
	return string(outputBytes), nil
}

// SearchFiles 根据提供的选项执行搜索操作。
// 它会遍历文件系统，应用所有过滤器，并返回匹配结果列表。
//
//nolint:gocyclo // 该函数的复杂度较高，但这是搜索操作的必要复杂度
func SearchFiles(opts *SearchOptions) ([]SearchResult, error) {
	var results []SearchResult
	var err error // 添加缺失的 err 变量声明
	if opts.Path == "" {
		opts.Path = defaultSearchPath // 确保搜索路径不为空
	}

	walkFn := func(path string, d fs.DirEntry, errIn error) error {
		if errIn != nil {
			// 记录或处理访问错误，但通常继续遍历其他条目
			results = append(results, SearchResult{Path: path, Error: fmt.Sprintf("访问错误: %v", errIn)})
			return nil // 或者根据错误类型决定是否 filepath.SkipDir
		}

		info, err := d.Info()
		if err != nil {
			results = append(results, SearchResult{Path: path, Error: fmt.Sprintf("获取文件信息错误: %v", err)})
			return nil // 继续处理下一个
		}

		// 检查是否应跳过此条目 (基于类型过滤器 IsDir)
		if opts.IsDir != nil {
			if *opts.IsDir && !info.IsDir() { // 要求目录，但当前是文件
				return nil
			}
			if !*opts.IsDir && info.IsDir() { // 要求文件，但当前是目录
				// 如果是非递归模式，则不进一步处理此目录的子项
				// 如果是递归模式，则仍需进入目录，但目录本身不作为结果
				if !opts.Recursive {
					return filepath.SkipDir
				}
				return nil // 目录本身不符合“仅文件”的筛选，但如果是递归，则需要继续遍历其内容
			}
		}

		// 检查递归深度
		currentRelPath, err := filepath.Rel(opts.Path, path)
		if err != nil {
			return fmt.Errorf("无法获取相对路径 '%s': %w", path, err)
		}
		currentDepth := 0
		if currentRelPath != "." { // 根路径深度为0
			currentDepth = len(strings.Split(currentRelPath, string(os.PathSeparator)))
		}

		if opts.Recursive && opts.MaxDepth > 0 && currentDepth > opts.MaxDepth {
			if info.IsDir() {
				return filepath.SkipDir // 超过最大深度，跳过此目录
			}
			return nil // 文件已超过最大深度，忽略
		}

		// 名称匹配 (支持 glob)
		if opts.NamePattern != "" {
			nameToMatch := info.Name()
			pattern := opts.NamePattern // 使用原始模式

			if !opts.CaseSensitive {
				// 如果不区分大小写匹配，将文件名转换为小写
				nameToMatch = strings.ToLower(nameToMatch)
			}

			matched, err := filepath.Match(pattern, nameToMatch)
			if err != nil {
				return fmt.Errorf("名称匹配失败 '%s': %w", path, err)
			}
			if !matched {
				return nil
			}
		}

		// 对于文件，应用大小和修改时间过滤器
		if !info.IsDir() {
			if opts.MinSize > 0 && info.Size() < opts.MinSize {
				return nil
			}
			if opts.MaxSize > 0 && info.Size() > opts.MaxSize {
				return nil
			}
			if !opts.ModTimeAfter.IsZero() && !info.ModTime().After(opts.ModTimeAfter) {
				return nil
			}
			if !opts.ModTimeBefore.IsZero() && !info.ModTime().Before(opts.ModTimeBefore) {
				return nil
			}

			// 内容正则表达式匹配 (仅对文件)
			if opts.ContentRegex != nil {
				content, readErr := os.ReadFile(path)
				if readErr != nil {
					results = append(results, SearchResult{Path: path, Error: fmt.Sprintf("读取文件内容时出错: %s", readErr.Error())})
					return nil // 读取失败，但仍记录此路径的错误
				}
				if !opts.ContentRegex.Match(content) {
					return nil
				}
			}
		}

		// 如果条目通过了所有过滤器，则添加到结果列表
		results = append(results, SearchResult{
			Path:    path,
			IsDir:   info.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})

		// 如果是目录且非递归模式，则不再深入
		if info.IsDir() && !opts.Recursive && path != opts.Path { // path != opts.Path 避免根目录被跳过
			return filepath.SkipDir
		}

		return nil
	}

	if opts.Recursive {
		err = filepath.WalkDir(opts.Path, walkFn)
	} else {
		// 非递归：仅读取顶层目录条目
		entries, readDirErr := os.ReadDir(opts.Path)
		if readDirErr != nil {
			return nil, fmt.Errorf("无法读取目录 '%s': %w", opts.Path, readDirErr)
		}
		for _, entry := range entries {
			entryPath := filepath.Join(opts.Path, entry.Name())
			if errWalk := walkFn(entryPath, entry, nil); errWalk != nil {
				if errors.Is(errWalk, filepath.SkipDir) { // SkipDir 在非递归的顶层迭代中不适用，但以防万一
					continue
				}
				return nil, fmt.Errorf("处理条目 '%s' 时出错: %w", entryPath, errWalk) // 如果walkFn返回了真实错误
			}
		}
	}

	if err != nil && !errors.Is(err, filepath.SkipDir) { // WalkDir 可能返回 SkipDir，这不是一个“真正的”错误
		return results, fmt.Errorf("文件搜索过程中出错: %w", err)
	}

	return results, nil
}

func (t *SearchTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		{
			Name: "results", Type: toolcore.ParamTypeArray, Required: true, Description: map[string]string{
				"en": "Search results",
				"zh": "搜索结果",
			},
			Items: Some(toolcore.ParameterDefinition{
				Type: toolcore.ParamTypeObject,
				Properties: Some([]toolcore.ParameterDefinition{
					{Name: "path", Type: toolcore.ParamTypeString, Required: true, Description: map[string]string{
						"en": "Path of the file or directory",
						"zh": "文件或目录的路径",
					}},
					{Name: "is_dir", Type: toolcore.ParamTypeBoolean, Required: true, Description: map[string]string{
						"en": "Whether the path is a directory",
						"zh": "路径是否为目录",
					}},
					{Name: "size", Type: toolcore.ParamTypeInteger, Required: false, Description: map[string]string{
						"en": "Size of the file in bytes",
						"zh": "文件大小（字节）",
					}},
					{Name: "mod_time", Type: toolcore.ParamTypeString, Required: false, Description: map[string]string{
						"en": "Last modified time of the file",
						"zh": "文件的最后修改时间",
					}},
					{Name: "error", Type: toolcore.ParamTypeString, Required: false, Description: map[string]string{
						"en": "Error message if any",
						"zh": "错误信息（如果有）",
					}},
				}),
			}),
		},
	}
}
