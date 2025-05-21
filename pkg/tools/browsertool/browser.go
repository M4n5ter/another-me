package browsertool

import (
	"context"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// 操作类型常量
const (
	OperationNavigate   = "navigate"   // 导航到指定URL
	OperationScreenshot = "screenshot" // 截图
	OperationClick      = "click"      // 点击元素
	OperationFill       = "fill"       // 填充表单
	OperationSelect     = "select"     // 选择下拉框选项
	OperationHover      = "hover"      // 鼠标悬停
	OperationEvaluate   = "evaluate"   // 执行JavaScript
	OperationDebug      = "debug"      // 调试操作
)

// 调试操作类型常量
const (
	DebugEnableDisable    = "enable_disable"    // 启用/禁用调试
	DebugSetBreakpoint    = "set_breakpoint"    // 设置断点
	DebugRemoveBreakpoint = "remove_breakpoint" // 移除断点
	DebugPause            = "pause"             // 暂停执行
	DebugResume           = "resume"            // 恢复执行
	DebugGetCallstack     = "get_callstack"     // 获取调用堆栈
)

// BrowserTool 实现 toolcore.Tool 接口，用于浏览器自动化
type BrowserTool struct {
	logger        *slog.Logger
	i18nMgr       *i18n.Manager
	config        *BrowserConfig
	browserCtx    context.Context
	cancelBrowser context.CancelFunc
}

// BrowserArgs 定义了 BrowserTool 的参数
type BrowserArgs struct {
	Operation string         `json:"operation"`          // 操作类型
	URL       string         `json:"url,omitempty"`      // 网址
	Selector  string         `json:"selector,omitempty"` // 选择器
	Value     string         `json:"value,omitempty"`    // 值
	Script    string         `json:"script,omitempty"`   // JavaScript脚本
	Debug     *DebugArgs     `json:"debug,omitempty"`    // 调试参数
	Options   map[string]any `json:"options,omitempty"`  // 其他选项
}

// DebugArgs 定义了调试操作的参数
type DebugArgs struct {
	Action       string `json:"action"`                  // 调试操作类型
	Enable       bool   `json:"enable,omitempty"`        // 是否启用调试
	URL          string `json:"url,omitempty"`           // 断点URL
	LineNumber   int    `json:"line_number,omitempty"`   // 断点行号
	ColumnNumber int    `json:"column_number,omitempty"` // 断点列号
	Condition    string `json:"condition,omitempty"`     // 断点条件
	BreakpointID string `json:"breakpoint_id,omitempty"` // 断点ID
}

// BrowserResult 定义了操作结果的结构
type BrowserResult struct {
	Operation  string `json:"operation"`            // 执行的操作
	Success    bool   `json:"success"`              // 是否成功
	Message    string `json:"message"`              // 操作消息
	Screenshot string `json:"screenshot,omitempty"` // Base64编码的截图
	Value      string `json:"value,omitempty"`      // 返回值
	DebugInfo  string `json:"debug_info,omitempty"` // 调试信息
}

// NewBrowserTool 创建一个新的 BrowserTool 实例
func NewBrowserTool(i18nMgr *i18n.Manager) *BrowserTool {
	return &BrowserTool{
		logger:  slog.Default().WithGroup("browser_tool"),
		i18nMgr: i18nMgr,
		config:  NewBrowserConfig(),
	}
}

// NewBrowserToolWithConfig 创建一个使用自定义配置的BrowserTool实例
func NewBrowserToolWithConfig(i18nMgr *i18n.Manager, config *BrowserConfig) *BrowserTool {
	return &BrowserTool{
		logger:  slog.Default().WithGroup("browser_tool"),
		i18nMgr: i18nMgr,
		config:  config,
	}
}

var _ toolcore.Tool = (*BrowserTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *BrowserTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.browser.description", nil)
		localizedNames[lang] = "Browser Automation"
	}

	// 定义操作类型枚举
	operations := []any{
		OperationNavigate,
		// OperationScreenshot, // TODO: 暂时关闭截图功能
		OperationClick,
		OperationFill,
		OperationSelect,
		OperationHover,
		OperationEvaluate,
		OperationDebug,
	}

	// 构建参数定义
	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "operation", toolcore.ParamTypeString, true, Some(operations), "tool.browser.arg.operation", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "url", toolcore.ParamTypeString, false, nil, "tool.browser.arg.url", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "selector", toolcore.ParamTypeString, false, nil, "tool.browser.arg.selector", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "value", toolcore.ParamTypeString, false, nil, "tool.browser.arg.value", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "script", toolcore.ParamTypeString, false, nil, "tool.browser.arg.script", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "debug", toolcore.ParamTypeObject, false, nil, "tool.browser.arg.debug", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "options", toolcore.ParamTypeObject, false, nil, "tool.browser.arg.options", nil),
	}

	// 返回工具的完整模式
	return toolcore.ToolSchema{
		Name:             "browser",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *BrowserTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var args BrowserArgs
	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	// 如果浏览器未初始化，则初始化
	if t.browserCtx == nil {
		if err := t.initBrowser(); err != nil {
			t.logger.Error("初始化浏览器失败", "error", err)
			return "", fmt.Errorf("初始化浏览器失败: %w", err)
		}
	}

	// 根据操作类型执行相应的函数
	var result BrowserResult
	var err error

	switch args.Operation {
	case OperationNavigate:
		result, err = t.navigate(args)
	case OperationScreenshot:
		result, err = t.screenshot(args)
	case OperationClick:
		result, err = t.click(args)
	case OperationFill:
		result, err = t.fill(args)
	case OperationSelect:
		result, err = t.selectOption(args)
	case OperationHover:
		result, err = t.hover(args)
	case OperationEvaluate:
		result, err = t.evaluate(args)
	case OperationDebug:
		result, err = t.debug(args)
	default:
		return "", fmt.Errorf("不支持的操作类型: %s", args.Operation)
	}

	if err != nil {
		t.logger.Error("执行操作失败", "operation", args.Operation, "error", err)
		return "", fmt.Errorf("执行操作 %s 失败: %w", args.Operation, err)
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		t.logger.Error("序列化结果失败", "error", err)
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}
	return string(resultJSON), nil
}

// initBrowser 初始化浏览器
func (t *BrowserTool) initBrowser() error {
	// 确保配置有效
	if err := t.config.Check(); err != nil {
		return fmt.Errorf("配置检查失败: %w", err)
	}

	// 创建浏览器数据目录
	userDataDir := filepath.Join(t.config.DataPath, t.config.BrowserDataPath)
	if err := os.MkdirAll(userDataDir, 0o755); err != nil {
		return fmt.Errorf("创建浏览器数据目录失败: %w", err)
	}

	// 设置Chrome选项
	opts := []chromedp.ExecAllocatorOption{
		chromedp.NoFirstRun,
		chromedp.NoDefaultBrowserCheck,
		chromedp.UserDataDir(userDataDir),
		chromedp.UserAgent(t.config.UserAgent),
		chromedp.WindowSize(t.config.WindowWidth, t.config.WindowHeight),
		chromedp.Flag("disable-popup-blocking", true),
		chromedp.Flag("blink-settings", "imagesEnabled=true"),
	}

	if t.config.DefaultLanguage != "" {
		opts = append(opts, chromedp.Flag("lang", t.config.DefaultLanguage))
	}

	if t.config.Proxy != "" {
		opts = append(opts, chromedp.Flag("proxy-server", t.config.Proxy))
	}

	if t.config.Headless {
		opts = append(opts, chromedp.Headless)
	}

	// 查找系统中可用的浏览器
	if t.config.BrowserPath != "" {
		// 优先使用配置中指定的浏览器路径
		if _, err := os.Stat(t.config.BrowserPath); err == nil {
			t.logger.Info("使用配置中指定的浏览器", "path", t.config.BrowserPath)
			opts = append(opts, chromedp.ExecPath(t.config.BrowserPath))
		} else {
			t.logger.Warn("配置中指定的浏览器路径无效，将尝试自动搜索", "path", t.config.BrowserPath, "error", err)
		}
	}

	// 如果没有指定浏览器路径，则尝试自动搜索
	if t.config.BrowserPath == "" {
		browser := FindBrowser(t.logger)
		if browser != nil {
			browserPath := GetChromedpExecPath(browser)
			if browserPath != "" {
				t.logger.Info("使用已发现的浏览器", "type", browser.Type, "path", browserPath)
				opts = append(opts, chromedp.ExecPath(browserPath))
			}
		} else {
			t.logger.Warn("未找到浏览器，将使用ChromeDP默认浏览器")
		}
	}

	// 创建浏览器上下文
	allocCtx, _ := chromedp.NewExecAllocator(context.Background(), opts...)

	// 使用自定义日志处理
	logHandler := func(format string, args ...any) {
		// 使用常量字符串处理日志
		t.logger.Info("ChromeDP日志", "message", fmt.Sprintf(format, args...))
	}

	browserCtx, browserCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(logHandler))

	// 保存浏览器上下文和取消函数，不设置全局超时
	t.browserCtx = browserCtx
	t.cancelBrowser = browserCancel

	return nil
}

// navigate 实现导航功能
func (t *BrowserTool) navigate(args BrowserArgs) (BrowserResult, error) {
	if args.URL == "" {
		return BrowserResult{}, fmt.Errorf("URL不能为空")
	}

	// 导航到URL
	err := chromedp.Run(t.browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		if err := chromedp.Navigate(args.URL).Do(ctx); err != nil {
			return fmt.Errorf("导航到URL失败: %w", err)
		}
		if err := chromedp.WaitReady("body", chromedp.ByQuery).Do(ctx); err != nil {
			return fmt.Errorf("等待页面加载完成失败: %w", err)
		}
		return nil
	}))
	if err != nil {
		t.logger.Error("导航失败", "url", args.URL, "error", err)
		return BrowserResult{
			Operation: OperationNavigate,
			Success:   false,
			Message:   fmt.Sprintf("导航到 %s 失败: %s", args.URL, err),
		}, nil
	}

	return BrowserResult{
		Operation: OperationNavigate,
		Success:   true,
		Message:   fmt.Sprintf("成功导航到 %s", args.URL),
	}, nil
}

// screenshot 实现截图功能
func (t *BrowserTool) screenshot(args BrowserArgs) (BrowserResult, error) {
	var buf []byte
	var err error

	if args.Selector != "" {
		// 截取特定元素
		err = chromedp.Run(t.browserCtx,
			chromedp.WaitReady(args.Selector, chromedp.ByQuery),
			chromedp.Screenshot(args.Selector, &buf, chromedp.NodeVisible, chromedp.ByQuery),
		)
	} else {
		// 截取整个页面
		captureScreenshot := func(ctx context.Context) ([]byte, error) {
			var buf []byte
			if err := chromedp.Run(ctx, chromedp.FullScreenshot(&buf, 100)); err != nil {
				return nil, fmt.Errorf("截取全屏截图失败: %w", err)
			}
			return buf, nil
		}
		buf, err = captureScreenshot(t.browserCtx)
	}

	if err != nil {
		t.logger.Error("截图失败", "selector", args.Selector, "error", err)
		return BrowserResult{
			Operation: OperationScreenshot,
			Success:   false,
			Message:   fmt.Sprintf("截图失败: %s", err),
		}, nil
	}

	// 处理输出形式
	var outputMsg string
	var b64Screenshot string

	// 如果有目标文件名，保存到文件
	if args.Value != "" {
		// 确保输出目录存在
		outputDir := filepath.Join(t.config.DataPath, "screenshots")
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			return BrowserResult{}, fmt.Errorf("创建截图输出目录失败: %w", err)
		}

		// 拼接文件路径，确保文件名以.png结尾
		filename := args.Value
		if !strings.HasSuffix(filename, ".png") {
			filename += ".png"
		}
		outputPath := filepath.Join(outputDir, filename)

		// 保存截图
		if err := os.WriteFile(outputPath, buf, 0o644); err != nil {
			return BrowserResult{}, fmt.Errorf("保存截图失败: %w", err)
		}

		outputMsg = fmt.Sprintf("截图已保存到文件: %s", outputPath)
	} else {
		// 不指定文件名时，直接返回Base64编码的截图
		b64Screenshot = fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(buf))
		outputMsg = "截图完成，返回Base64编码"
	}

	return BrowserResult{
		Operation:  OperationScreenshot,
		Success:    true,
		Message:    outputMsg,
		Screenshot: b64Screenshot,
	}, nil
}

// click 实现点击功能
func (t *BrowserTool) click(args BrowserArgs) (BrowserResult, error) {
	if args.Selector == "" {
		return BrowserResult{}, fmt.Errorf("选择器不能为空")
	}

	// 点击元素
	err := chromedp.Run(t.browserCtx,
		chromedp.WaitReady(args.Selector, chromedp.ByQuery),
		chromedp.Click(args.Selector, chromedp.ByQuery),
	)
	if err != nil {
		t.logger.Error("点击元素失败", "selector", args.Selector, "error", err)
		return BrowserResult{
			Operation: OperationClick,
			Success:   false,
			Message:   fmt.Sprintf("点击元素 %s 失败: %s", args.Selector, err),
		}, nil
	}

	return BrowserResult{
		Operation: OperationClick,
		Success:   true,
		Message:   fmt.Sprintf("成功点击元素 %s", args.Selector),
	}, nil
}

// fill 实现填充表单功能
func (t *BrowserTool) fill(args BrowserArgs) (BrowserResult, error) {
	if args.Selector == "" {
		return BrowserResult{}, fmt.Errorf("选择器不能为空")
	}

	// 填充表单
	err := chromedp.Run(t.browserCtx,
		chromedp.WaitReady(args.Selector, chromedp.ByQuery),
		chromedp.Clear(args.Selector, chromedp.ByQuery),
		chromedp.SendKeys(args.Selector, args.Value, chromedp.ByQuery),
	)
	if err != nil {
		t.logger.Error("填充表单失败", "selector", args.Selector, "value", args.Value, "error", err)
		return BrowserResult{
			Operation: OperationFill,
			Success:   false,
			Message:   fmt.Sprintf("填充表单 %s 失败: %s", args.Selector, err),
		}, nil
	}

	return BrowserResult{
		Operation: OperationFill,
		Success:   true,
		Message:   fmt.Sprintf("成功填充 %s 为 %s", args.Selector, args.Value),
	}, nil
}

// selectOption 实现选择下拉框选项功能
func (t *BrowserTool) selectOption(args BrowserArgs) (BrowserResult, error) {
	if args.Selector == "" {
		return BrowserResult{}, fmt.Errorf("选择器不能为空")
	}

	// 选择下拉框选项
	err := chromedp.Run(t.browserCtx,
		chromedp.WaitReady(args.Selector, chromedp.ByQuery),
		chromedp.SetValue(args.Selector, args.Value, chromedp.ByQuery),
	)
	if err != nil {
		t.logger.Error("选择下拉框选项失败", "selector", args.Selector, "value", args.Value, "error", err)
		return BrowserResult{
			Operation: OperationSelect,
			Success:   false,
			Message:   fmt.Sprintf("选择下拉框 %s 中的选项 %s 失败: %s", args.Selector, args.Value, err),
		}, nil
	}

	return BrowserResult{
		Operation: OperationSelect,
		Success:   true,
		Message:   fmt.Sprintf("成功选择 %s 中的选项 %s", args.Selector, args.Value),
	}, nil
}

// hover 实现鼠标悬停功能
func (t *BrowserTool) hover(args BrowserArgs) (BrowserResult, error) {
	if args.Selector == "" {
		return BrowserResult{}, fmt.Errorf("选择器不能为空")
	}

	// 鼠标悬停，使用input包来模拟鼠标移动
	err := chromedp.Run(t.browserCtx,
		chromedp.WaitReady(args.Selector, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			// 获取元素的位置和尺寸
			var rect map[string]any
			err := chromedp.Evaluate(`
				(function() {
					const element = document.querySelector(`+"`"+args.Selector+"`"+`);
					if (!element) return null;
					const rect = element.getBoundingClientRect();
					return {
						x: rect.left + rect.width/2,
						y: rect.top + rect.height/2,
						width: rect.width,
						height: rect.height
					};
				})()
			`, &rect).Do(ctx)
			if err != nil {
				return fmt.Errorf("获取元素位置失败: %w", err)
			}

			if rect == nil {
				return fmt.Errorf("找不到元素 %s", args.Selector)
			}

			// 执行鼠标移动
			if err := input.DispatchMouseEvent(
				input.MouseMoved,
				float64(rect["x"].(float64)),
				float64(rect["y"].(float64)),
			).Do(ctx); err != nil {
				return fmt.Errorf("鼠标移动事件分发失败: %w", err)
			}
			return nil
		}),
	)
	if err != nil {
		t.logger.Error("鼠标悬停失败", "selector", args.Selector, "error", err)
		return BrowserResult{
			Operation: OperationHover,
			Success:   false,
			Message:   fmt.Sprintf("鼠标悬停在 %s 上失败: %s", args.Selector, err),
		}, nil
	}

	return BrowserResult{
		Operation: OperationHover,
		Success:   true,
		Message:   fmt.Sprintf("成功将鼠标悬停在 %s 上", args.Selector),
	}, nil
}

// evaluate 实现执行JavaScript功能
func (t *BrowserTool) evaluate(args BrowserArgs) (BrowserResult, error) {
	if args.Script == "" {
		return BrowserResult{}, fmt.Errorf("脚本不能为空")
	}

	// 执行JavaScript
	var result any
	err := chromedp.Run(t.browserCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			err := chromedp.Evaluate(args.Script, &result).Do(ctx)
			if err != nil {
				return fmt.Errorf("执行JavaScript评估失败: %w", err)
			}
			return nil
		}),
	)
	if err != nil {
		t.logger.Error("执行JavaScript失败", "script", args.Script, "error", err)
		return BrowserResult{
			Operation: OperationEvaluate,
			Success:   false,
			Message:   fmt.Sprintf("执行JavaScript失败: %s", err),
		}, nil
	}

	// 将结果转换为字符串
	var resultStr string
	if result != nil {
		resultBytes, err := json.Marshal(result)
		if err != nil {
			resultStr = fmt.Sprintf("%v", result)
		} else {
			resultStr = string(resultBytes)
		}
	}

	return BrowserResult{
		Operation: OperationEvaluate,
		Success:   true,
		Message:   "JavaScript执行成功",
		Value:     resultStr,
	}, nil
}

// debug 封装调试相关功能
func (t *BrowserTool) debug(args BrowserArgs) (BrowserResult, error) {
	if args.Debug == nil {
		return BrowserResult{}, fmt.Errorf("调试参数不能为空")
	}

	// 根据调试操作类型执行相应的函数
	switch args.Debug.Action {
	case DebugEnableDisable:
		return t.debugEnableDisable(args)
	case DebugSetBreakpoint:
		return t.debugSetBreakpoint(args)
	case DebugRemoveBreakpoint:
		return t.debugRemoveBreakpoint(args)
	case DebugPause:
		return t.debugPause()
	case DebugResume:
		return t.debugResume()
	case DebugGetCallstack:
		return t.debugGetCallstack()
	default:
		return BrowserResult{}, fmt.Errorf("不支持的调试操作类型: %s", args.Debug.Action)
	}
}

// Close 关闭浏览器
func (t *BrowserTool) Close() error {
	if t.cancelBrowser != nil {
		t.cancelBrowser()
		t.cancelBrowser = nil
		t.browserCtx = nil
	}
	return nil
}

// createOutputParameters 创建输出参数定义
func (t *BrowserTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	operationDesc := map[string]string{
		"en": "Operation performed",
		"zh": "执行的操作",
	}

	successDesc := map[string]string{
		"en": "Whether the operation succeeded",
		"zh": "操作是否成功",
	}

	messageDesc := map[string]string{
		"en": "Operation message",
		"zh": "操作消息",
	}

	screenshotDesc := map[string]string{
		"en": "Base64 encoded screenshot",
		"zh": "Base64编码的截图",
	}

	valueDesc := map[string]string{
		"en": "Return value",
		"zh": "返回值",
	}

	debugInfoDesc := map[string]string{
		"en": "Debug information",
		"zh": "调试信息",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "operation",
			Type:        toolcore.ParamTypeString,
			Description: operationDesc,
			Required:    true,
		},
		{
			Name:        "success",
			Type:        toolcore.ParamTypeBoolean,
			Description: successDesc,
			Required:    true,
		},
		{
			Name:        "message",
			Type:        toolcore.ParamTypeString,
			Description: messageDesc,
			Required:    true,
		},
		{
			Name:        "screenshot",
			Type:        toolcore.ParamTypeString,
			Description: screenshotDesc,
			Required:    false,
		},
		{
			Name:        "value",
			Type:        toolcore.ParamTypeString,
			Description: valueDesc,
			Required:    false,
		},
		{
			Name:        "debug_info",
			Type:        toolcore.ParamTypeString,
			Description: debugInfoDesc,
			Required:    false,
		},
	}
}
