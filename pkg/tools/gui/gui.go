package gui

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"log/slog"
	"strings"

	"github.com/go-vgo/robotgo"
	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// Tool 实现 toolcore.Tool 接口，提供各种 GUI 操作功能
type Tool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
	name    string // 工具的名称，如 "screenshot", "move_mouse" 等
}

// NewGUITools 创建一个新的 GUITool 实例
func NewGUITools(i18nMgr *i18n.Manager) []toolcore.Tool {
	// 创建各种 GUI 工具的实例
	tools := []toolcore.Tool{
		NewGUIToolWithName(i18nMgr, "screenshot"),
		NewGUIToolWithName(i18nMgr, "move_mouse"),
		NewGUIToolWithName(i18nMgr, "mouse_location"),
		NewGUIToolWithName(i18nMgr, "drag"),
		NewGUIToolWithName(i18nMgr, "scroll"),
		NewGUIToolWithName(i18nMgr, "scroll_relative"),
		NewGUIToolWithName(i18nMgr, "scroll_direction"),
		NewGUIToolWithName(i18nMgr, "click"),
		NewGUIToolWithName(i18nMgr, "toggle_mouse_button"),
		NewGUIToolWithName(i18nMgr, "toggle_key"),
		NewGUIToolWithName(i18nMgr, "key_tap"),
		NewGUIToolWithName(i18nMgr, "type_string"),
		NewGUIToolWithName(i18nMgr, "key_sleep_milli"),
		NewGUIToolWithName(i18nMgr, "sleep_milli"),
	}
	return tools
}

// NewGUIToolWithName 创建指定名称的 GUI 工具实例
func NewGUIToolWithName(i18nMgr *i18n.Manager, name string) *Tool {
	return &Tool{
		logger:  slog.Default().WithGroup("gui_tool").With("tool_name", name),
		i18nMgr: i18nMgr,
		name:    name,
	}
}

var _ toolcore.Tool = (*Tool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *Tool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取支持的语言
	langs := t.i18nMgr.GetSupportedLanguages()

	// 初始化描述映射
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	// 工具名称的i18n键
	descKey := fmt.Sprintf("tool.gui.%s.description", t.name)

	// 获取不同语言的描述
	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, descKey, nil)
		// 本地化名称可以是简单的大写首字母
		localizedNames[lang] = toTitleCase(t.name) // 例如：将 "screenshot" 转为 "Screenshot"
	}

	// 根据工具类型定义输入参数
	var inputParams []toolcore.ParameterDefinition
	switch t.name {
	case "screenshot":
		// 截图工具不需要输入参数
	case "move_mouse":
		inputParams = t.createMoveMouseParams(ctx)
	case "mouse_location":
		// 不需要输入参数
	case "drag":
		inputParams = t.createDragParams(ctx)
	case "scroll":
		inputParams = t.createScrollParams(ctx)
	case "scroll_relative":
		inputParams = t.createScrollRelativeParams(ctx)
	case "scroll_direction":
		inputParams = t.createScrollDirectionParams(ctx)
	case "click":
		inputParams = t.createClickParams(ctx)
	case "toggle_mouse_button":
		inputParams = t.createToggleMouseButtonParams(ctx)
	case "toggle_key":
		inputParams = t.createToggleKeyParams(ctx)
	case "key_tap":
		inputParams = t.createKeyTapParams(ctx)
	case "type_string":
		inputParams = t.createTypeStringParams(ctx)
	case "key_sleep_milli":
		inputParams = t.createKeySleepMilliParams(ctx)
	case "sleep_milli":
		inputParams = t.createSleepMilliParams(ctx)
	}

	// 返回工具模式
	return toolcore.ToolSchema{
		Name:            t.name,
		LocalizedNames:  localizedNames,
		Descriptions:    descriptions,
		InputParameters: inputParams,
		// 输出参数通常是一个简单的字符串消息
		OutputParameters: []toolcore.ParameterDefinition{
			{
				Name: "result",
				Type: toolcore.ParamTypeString,
				Description: map[string]string{
					"en": "Operation result",
					"zh": "操作结果",
				},
				Required: true,
			},
		},
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *Tool) Call(ctx context.Context, inputJSON string) (string, error) {
	switch t.name {
	case "screenshot":
		return t.Screenshot()
	case "move_mouse":
		return t.MoveMouse(inputJSON)
	case "mouse_location":
		return t.MouseLocation()
	case "drag":
		return t.Drag(inputJSON)
	case "scroll":
		return t.Scroll(inputJSON)
	case "scroll_relative":
		return t.ScrollRelative(inputJSON)
	case "scroll_direction":
		return t.ScrollDirection(inputJSON)
	case "click":
		return t.Click(inputJSON)
	case "toggle_mouse_button":
		return t.ToggleMouseButton(inputJSON)
	case "toggle_key":
		return t.ToggleKey(inputJSON)
	case "key_tap":
		return t.KeyTap(inputJSON)
	case "type_string":
		return t.TypeString(inputJSON)
	case "key_sleep_milli":
		return t.KeySleepMilli(inputJSON)
	case "sleep_milli":
		return t.SleepMilli(inputJSON)
	default:
		return "", fmt.Errorf("未知的 GUI 工具: %s", t.name)
	}
}

// Screenshot 捕获一张默认桌面的截图: png base64 url
func (t *Tool) Screenshot() (string, error) {
	rgba, err := robotgo.Capture()
	if err != nil {
		t.logger.Error("Failed to capture screen", "error", err)
		return "", fmt.Errorf("failed to capture screen: %w", err)
	}

	rect := rgba.Bounds()

	buf := new(bytes.Buffer)
	err = png.Encode(buf, rgba)
	if err != nil {
		t.logger.Error("Failed to encode screenshot", "error", err)
		return "", fmt.Errorf("failed to encode screenshot: %w", err)
	}

	base64PNG := base64.StdEncoding.EncodeToString(buf.Bytes())
	base64PNGURL := fmt.Sprintf("data:image/png;base64,%s", base64PNG)

	result := map[string]any{
		"result": base64PNGURL,
		"width":  rect.Max.X,
		"height": rect.Max.Y,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// MoveMouse 移动鼠标
func (t *Tool) MoveMouse(inputJSON string) (string, error) {
	var args struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	robotgo.MoveSmooth(args.X, args.Y)

	newX, newY := robotgo.Location()

	result := map[string]any{
		"result": fmt.Sprintf("移动鼠标到位置 x: %d, y: %d", newX, newY),
		"x":      newX,
		"y":      newY,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// MouseLocation 获取鼠标当前坐标位置
func (t *Tool) MouseLocation() (string, error) {
	x, y := robotgo.Location()

	result := map[string]any{
		"result": fmt.Sprintf("鼠标当前位置 x: %d, y: %d", x, y),
		"x":      x,
		"y":      y,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// Drag 拖动
func (t *Tool) Drag(inputJSON string) (string, error) {
	var args struct {
		X int `json:"x"`
		Y int `json:"y"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	robotgo.DragSmooth(args.X, args.Y)

	newX, newY := robotgo.Location()

	result := map[string]any{
		"result": fmt.Sprintf("拖动鼠标到位置 x: %d, y: %d", newX, newY),
		"x":      newX,
		"y":      newY,
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// Scroll 滚动
func (t *Tool) Scroll(inputJSON string) (string, error) {
	var args struct {
		ToY     int `json:"toy"`
		Num     int `json:"num"`
		MsSleep int `json:"ms_sleep"`
		ToX     int `json:"tox"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	robotgo.ScrollSmooth(args.ToY, args.Num, args.MsSleep, args.ToX)

	result := map[string]any{
		"result": fmt.Sprintf("滚动完成，目标: y=%d, x=%d, 次数=%d, 间隔=%dms", args.ToY, args.ToX, args.Num, args.MsSleep),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// ScrollRelative 相对坐标滚动
func (t *Tool) ScrollRelative(inputJSON string) (string, error) {
	var args struct {
		X       int `json:"x"`
		Y       int `json:"y"`
		MsDelay int `json:"ms_delay"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	robotgo.ScrollRelative(args.X, args.Y, args.MsDelay)

	result := map[string]any{
		"result": fmt.Sprintf("相对滚动完成，x=%d, y=%d, 延迟=%dms", args.X, args.Y, args.MsDelay),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// ScrollDirection 选择指定方向滚动
func (t *Tool) ScrollDirection(inputJSON string) (string, error) {
	var args struct {
		X         int    `json:"x"`
		Direction string `json:"direction"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	// 验证方向值是否有效
	validDirection := false
	for _, dir := range directionEnum() {
		if args.Direction == dir {
			validDirection = true
			break
		}
	}

	if !validDirection {
		args.Direction = "up" // 默认方向
	}

	robotgo.ScrollDir(args.X, args.Direction)

	result := map[string]any{
		"result": fmt.Sprintf("方向滚动完成，x=%d, 方向=%s", args.X, args.Direction),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// Click 点击
func (t *Tool) Click(inputJSON string) (string, error) {
	var args struct {
		Button string `json:"button"`
		Double bool   `json:"double"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	// 验证按钮值是否有效
	validButton := false
	for _, btn := range mouseButtonEnum() {
		if args.Button == btn {
			validButton = true
			break
		}
	}

	if !validButton {
		args.Button = "left" // 默认按钮
	}

	robotgo.Click(args.Button, args.Double)

	doubleStr := "单次"
	if args.Double {
		doubleStr = "双"
	}

	result := map[string]any{
		"result": fmt.Sprintf("%s点击完成，按钮=%s", doubleStr, args.Button),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// ToggleMouseButton 操作鼠标按键
func (t *Tool) ToggleMouseButton(inputJSON string) (string, error) {
	var args struct {
		Button string `json:"button"`
		Up     bool   `json:"up"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	// 验证按钮值是否有效
	validButton := false
	for _, btn := range mouseButtonEnum() {
		if args.Button == btn {
			validButton = true
			break
		}
	}

	if !validButton {
		args.Button = "left" // 默认按钮
	}

	direction := "down"
	if args.Up {
		direction = "up"
	}

	err := robotgo.Toggle(args.Button, direction)
	if err != nil {
		return "", fmt.Errorf("操作鼠标按键失败: %w", err)
	}

	actionStr := "按下"
	if args.Up {
		actionStr = "释放"
	}

	result := map[string]any{
		"result": fmt.Sprintf("%s鼠标按钮=%s", actionStr, args.Button),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// ToggleKey 操作键盘按键
func (t *Tool) ToggleKey(inputJSON string) (string, error) {
	var args struct {
		Up   bool     `json:"up"`
		Keys []string `json:"keys"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	if len(args.Keys) < 1 {
		return "", fmt.Errorf("必须提供至少一个按键")
	}

	direction := "down"
	if args.Up {
		direction = "up"
	}

	k0 := args.Keys[0]

	if len(args.Keys) == 1 {
		err := robotgo.KeyToggle(k0, direction)
		if err != nil {
			return "", fmt.Errorf("操作按键失败: %w", err)
		}
	} else {
		keysLeft := args.Keys[1:]
		err := robotgo.KeyToggle(k0, direction, keysLeft)
		if err != nil {
			return "", fmt.Errorf("操作按键失败: %w", err)
		}
	}

	actionStr := "按下"
	if args.Up {
		actionStr = "释放"
	}

	result := map[string]any{
		"result": fmt.Sprintf("%s按键=%v", actionStr, args.Keys),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// KeyTap 点按键盘按键
func (t *Tool) KeyTap(inputJSON string) (string, error) {
	var args struct {
		Keys []string `json:"keys"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	if len(args.Keys) < 1 {
		return "", fmt.Errorf("必须提供至少一个按键")
	}

	k0 := args.Keys[0]

	if len(args.Keys) == 1 {
		err := robotgo.KeyTap(k0)
		if err != nil {
			return "", fmt.Errorf("点按按键失败: %w", err)
		}
	} else {
		keysLeft := args.Keys[1:]
		err := robotgo.KeyTap(k0, keysLeft)
		if err != nil {
			return "", fmt.Errorf("点按按键失败: %w", err)
		}
	}

	result := map[string]any{
		"result": fmt.Sprintf("点按按键=%v", args.Keys),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// TypeString 输入字符串
func (t *Tool) TypeString(inputJSON string) (string, error) {
	var args struct {
		Content string `json:"content"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	robotgo.TypeStr(args.Content)

	result := map[string]any{
		"result": fmt.Sprintf("输入字符串=%s", args.Content),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// KeySleepMilli 设置按键睡眠时间
func (t *Tool) KeySleepMilli(inputJSON string) (string, error) {
	var args struct {
		Ms int `json:"ms"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	robotgo.KeySleep = args.Ms

	result := map[string]any{
		"result": fmt.Sprintf("设置按键睡眠时间=%dms", args.Ms),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// SleepMilli 睡眠 ms 毫秒
func (t *Tool) SleepMilli(inputJSON string) (string, error) {
	var args struct {
		Ms int `json:"ms"`
	}

	if err := json.Unmarshal([]byte(inputJSON), &args); err != nil {
		return "", fmt.Errorf("解析参数失败: %w", err)
	}

	robotgo.MilliSleep(args.Ms)

	result := map[string]any{
		"result": fmt.Sprintf("睡眠=%dms", args.Ms),
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("序列化结果失败: %w", err)
	}

	return string(resultJSON), nil
}

// 以下是各种参数定义方法

func (t *Tool) createMoveMouseParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "x", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.move_mouse.arg.x", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "y", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.move_mouse.arg.y", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createDragParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "x", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.drag.arg.x", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "y", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.drag.arg.y", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createScrollParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "toy", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.scroll.arg.toy", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "num", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.scroll.arg.num", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "ms_sleep", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.scroll.arg.ms_sleep", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "tox", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.scroll.arg.tox", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createScrollRelativeParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "x", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.scroll_relative.arg.x", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "y", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.scroll_relative.arg.y", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "ms_delay", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.scroll_relative.arg.ms_delay", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createScrollDirectionParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "x", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.scroll_direction.arg.x", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "direction", toolcore.ParamTypeString, true, Some(directionEnum()), "tool.gui.scroll_direction.arg.direction", Some(toolcore.ParameterDefinition{
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "The direction to scroll",
				"zh": "要滚动的方向",
			},
		})),
	}
}

func (t *Tool) createClickParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "button", toolcore.ParamTypeString, true, Some(mouseButtonEnum()), "tool.gui.click.arg.button", Some(toolcore.ParameterDefinition{
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "The mouse button to click",
				"zh": "要点击的鼠标按钮",
			},
		})),
		t.createParamDef(ctx, "double", toolcore.ParamTypeBoolean, true, None[[]any](), "tool.gui.click.arg.double", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createToggleMouseButtonParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "button", toolcore.ParamTypeString, true, Some(mouseButtonEnum()), "tool.gui.toggle_mouse_button.arg.button", Some(toolcore.ParameterDefinition{
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "The mouse button to toggle",
				"zh": "要切换的鼠标按钮",
			},
		})),
		t.createParamDef(ctx, "up", toolcore.ParamTypeBoolean, true, None[[]any](), "tool.gui.toggle_mouse_button.arg.up", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createToggleKeyParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "up", toolcore.ParamTypeBoolean, true, None[[]any](), "tool.gui.toggle_key.arg.up", None[toolcore.ParameterDefinition]()),
		t.createParamDef(ctx, "keys", toolcore.ParamTypeArray, true, Some(stringSliceToAnySlice(KeyEnum())), "tool.gui.toggle_key.arg.keys", Some(toolcore.ParameterDefinition{
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "The keys to toggle",
				"zh": "要切换的按键",
			},
		})),
	}
}

func (t *Tool) createKeyTapParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "keys", toolcore.ParamTypeArray, true, Some(stringSliceToAnySlice(KeyEnum())), "tool.gui.key_tap.arg.keys", Some(toolcore.ParameterDefinition{
			Type: toolcore.ParamTypeString,
			Description: map[string]string{
				"en": "The keys to tap",
				"zh": "要按下的按键",
			},
		})),
	}
}

func (t *Tool) createTypeStringParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "content", toolcore.ParamTypeString, true, None[[]any](), "tool.gui.type_string.arg.content", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createKeySleepMilliParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "ms", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.key_sleep_milli.arg.ms", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createSleepMilliParams(ctx context.Context) []toolcore.ParameterDefinition {
	return []toolcore.ParameterDefinition{
		t.createParamDef(ctx, "ms", toolcore.ParamTypeInteger, true, None[[]any](), "tool.gui.sleep_milli.arg.ms", None[toolcore.ParameterDefinition]()),
	}
}

func (t *Tool) createParamDef(ctx context.Context, name string, paramType toolcore.ParameterType,
	required bool, enumValues Option[[]any], descKey string, arrayItemType Option[toolcore.ParameterDefinition],
) toolcore.ParameterDefinition {
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, descKey, nil)
	}

	return toolcore.ParameterDefinition{
		Name:        name,
		Type:        paramType,
		Description: descriptions,
		Required:    required,
		EnumValues:  enumValues,
		Items:       arrayItemType,
	}
}

// 辅助函数

// toTitleCase 将首字母大写，下划线转为空格
func toTitleCase(s string) string {
	if s == "" {
		return ""
	}

	// 可以替换下划线为空格然后转换，但这里简单实现
	result := strings.ToUpper(s[:1]) + strings.ReplaceAll(s[1:], "_", " ")
	return result
}

// 方向枚举值
func directionEnum() []any {
	return []any{"up", "down", "left", "right"}
}

// 鼠标按钮枚举值
func mouseButtonEnum() []any {
	return []any{"left", "right", "wheelLeft", "wheelRight", "wheelDown", "wheelUp"}
}

// KeyEnum 返回所有键盘按键的枚举值
func KeyEnum() []string {
	return []string{string(KeyA), string(KeyB), string(KeyC), string(KeyD), string(KeyE), string(KeyF), string(KeyG), string(KeyH), string(KeyI), string(KeyJ), string(KeyK), string(KeyL), string(KeyM), string(KeyN), string(KeyO), string(KeyP), string(KeyQ), string(KeyR), string(KeyS), string(KeyT), string(KeyU), string(KeyV), string(KeyW), string(KeyX), string(KeyY), string(KeyZ), string(Keya), string(Keyb), string(Keyc), string(Keyd), string(Keye), string(Keyf), string(Keyg), string(Keyh), string(Keyi), string(Keyj), string(Keyk), string(Keyl), string(Keym), string(Keyn), string(Keyo), string(Keyp), string(Keyq), string(Keyr), string(Keys), string(Keyt), string(Keyu), string(Keyv), string(Keyw), string(Keyx), string(Keyy), string(Keyz), string(Key0), string(Key1), string(Key2), string(Key3), string(Key4), string(Key5), string(Key6), string(Key7), string(Key8), string(Key9), string(KeyBackspace), string(KeyDelete), string(KeyEnter), string(KeyTab), string(KeyEsc), string(KeyEscape), string(KeyUp), string(KeyDown), string(KeyRight), string(KeyLeft), string(KeyHome), string(KeyEnd), string(KeyPageUp), string(KeyPageDown), string(KeyF1), string(KeyF2), string(KeyF3), string(KeyF4), string(KeyF5), string(KeyF6), string(KeyF7), string(KeyF8), string(KeyF9), string(KeyF10), string(KeyF11), string(KeyF12), string(KeyF13), string(KeyF14), string(KeyF15), string(KeyF16), string(KeyF17), string(KeyF18), string(KeyF19), string(KeyF20), string(KeyF21), string(KeyF22), string(KeyF23), string(KeyF24), string(KeyCmd), string(KeyLCmd), string(KeyRCmd), string(KeyAlt), string(KeyLAlt), string(KeyRAlt), string(KeyCtrl), string(KeyLCtrl), string(KeyRCtrl), string(KeyControl), string(KeyShift), string(KeyLShift), string(KeyRShift), string(KeyCapsLock), string(KeySpace), string(KeyPrint), string(KeyPrintScreen), string(KeyInsert), string(KeyMenu), string(KeyAudioMute), string(KeyAudioVolDown), string(KeyAudioVolUp), string(KeyAudioPlay), string(KeyAudioStop), string(KeyAudioPause), string(KeyAudioPrev), string(KeyAudioNext), string(KeyAudioRewind), string(KeyAudioForward), string(KeyAudioRepeat), string(KeyAudioRandom), string(KeyNum0), string(KeyNum1), string(KeyNum2), string(KeyNum3), string(KeyNum4), string(KeyNum5), string(KeyNum6), string(KeyNum7), string(KeyNum8), string(KeyNum9), string(KeyNumLock), string(KeyNumDot), string(KeyNumPlus), string(KeyNumMinus), string(KeyNumMultiply), string(KeyNumDivide), string(KeyNumClear), string(KeyNumEnter), string(KeyNumEqual), string(KeyNumpad0), string(KeyNumpad1), string(KeyNumpad2), string(KeyNumpad3), string(KeyNumpad4), string(KeyNumpad5), string(KeyNumpad6), string(KeyNumpad7), string(KeyNumpad8), string(KeyNumpad9), string(KeyLightsMonUp), string(KeyLightsMonDown), string(KeyLightsKbdToggle), string(KeyLightsKbdUp), string(KeyLightsKbdDown)}
}

// 辅助函数，用于将 []string 转换为 []any
func stringSliceToAnySlice(s []string) []any {
	a := make([]any, len(s))
	for i, v := range s {
		a[i] = v
	}
	return a
}

// Key 键盘按键类型
type Key string

//nolint:revive // 键盘按键常量
const (
	// 普通字符键 A-Z (大写), a-z (小写), 0-9
	KeyA Key = "A" // 字母键 A
	KeyB Key = "B" // 字母键 B
	KeyC Key = "C" // 字母键 C
	KeyD Key = "D" // 字母键 D
	KeyE Key = "E" // 字母键 E
	KeyF Key = "F" // 字母键 F
	KeyG Key = "G" // 字母键 G
	KeyH Key = "H" // 字母键 H
	KeyI Key = "I" // 字母键 I
	KeyJ Key = "J" // 字母键 J
	KeyK Key = "K" // 字母键 K
	KeyL Key = "L" // 字母键 L
	KeyM Key = "M" // 字母键 M
	KeyN Key = "N" // 字母键 N
	KeyO Key = "O" // 字母键 O
	KeyP Key = "P" // 字母键 P
	KeyQ Key = "Q" // 字母键 Q
	KeyR Key = "R" // 字母键 R
	KeyS Key = "S" // 字母键 S
	KeyT Key = "T" // 字母键 T
	KeyU Key = "U" // 字母键 U
	KeyV Key = "V" // 字母键 V
	KeyW Key = "W" // 字母键 W
	KeyX Key = "X" // 字母键 X
	KeyY Key = "Y" // 字母键 Y
	KeyZ Key = "Z" // 字母键 Z

	Keya Key = "a" // 字母键 a
	Keyb Key = "b" // 字母键 b
	Keyc Key = "c" // 字母键 c
	Keyd Key = "d" // 字母键 d
	Keye Key = "e" // 字母键 e
	Keyf Key = "f" // 字母键 f
	Keyg Key = "g" // 字母键 g
	Keyh Key = "h" // 字母键 h
	Keyi Key = "i" // 字母键 i
	Keyj Key = "j" // 字母键 j
	Keyk Key = "k" // 字母键 k
	Keyl Key = "l" // 字母键 l
	Keym Key = "m" // 字母键 m
	Keyn Key = "n" // 字母键 n
	Keyo Key = "o" // 字母键 o
	Keyp Key = "p" // 字母键 p
	Keyq Key = "q" // 字母键 q
	Keyr Key = "r" // 字母键 r
	Keys Key = "s" // 字母键 s
	Keyt Key = "t" // 字母键 t
	Keyu Key = "u" // 字母键 u
	Keyv Key = "v" // 字母键 v
	Keyw Key = "w" // 字母键 w
	Keyx Key = "x" // 字母键 x
	Keyy Key = "y" // 字母键 y
	Keyz Key = "z" // 字母键 z

	Key0 Key = "0" // 数字键 0
	Key1 Key = "1" // 数字键 1
	Key2 Key = "2" // 数字键 2
	Key3 Key = "3" // 数字键 3
	Key4 Key = "4" // 数字键 4
	Key5 Key = "5" // 数字键 5
	Key6 Key = "6" // 数字键 6
	Key7 Key = "7" // 数字键 7
	Key8 Key = "8" // 数字键 8
	Key9 Key = "9" // 数字键 9

	KeyBackspace Key = "backspace" // 退格键
	KeyDelete    Key = "delete"    // 删除键
	KeyEnter     Key = "enter"     // 回车键
	KeyTab       Key = "tab"       // Tab 键
	KeyEsc       Key = "esc"       // Esc 键
	KeyEscape    Key = "escape"    // Esc 键 (别名)
	KeyUp        Key = "up"        // 向上箭头键
	KeyDown      Key = "down"      // 向下箭头键
	KeyRight     Key = "right"     // 向右箭头键
	KeyLeft      Key = "left"      // 向左箭头键
	KeyHome      Key = "home"      // Home 键
	KeyEnd       Key = "end"       // End 键
	KeyPageUp    Key = "pageup"    // Page Up 键
	KeyPageDown  Key = "pagedown"  // Page Down 键

	KeyF1  Key = "f1"  // 功能键 F1
	KeyF2  Key = "f2"  // 功能键 F2
	KeyF3  Key = "f3"  // 功能键 F3
	KeyF4  Key = "f4"  // 功能键 F4
	KeyF5  Key = "f5"  // 功能键 F5
	KeyF6  Key = "f6"  // 功能键 F6
	KeyF7  Key = "f7"  // 功能键 F7
	KeyF8  Key = "f8"  // 功能键 F8
	KeyF9  Key = "f9"  // 功能键 F9
	KeyF10 Key = "f10" // 功能键 F10
	KeyF11 Key = "f11" // 功能键 F11
	KeyF12 Key = "f12" // 功能键 F12
	KeyF13 Key = "f13" // 功能键 F13
	KeyF14 Key = "f14" // 功能键 F14
	KeyF15 Key = "f15" // 功能键 F15
	KeyF16 Key = "f16" // 功能键 F16
	KeyF17 Key = "f17" // 功能键 F17
	KeyF18 Key = "f18" // 功能键 F18
	KeyF19 Key = "f19" // 功能键 F19
	KeyF20 Key = "f20" // 功能键 F20
	KeyF21 Key = "f21" // 功能键 F21
	KeyF22 Key = "f22" // 功能键 F22
	KeyF23 Key = "f23" // 功能键 F23
	KeyF24 Key = "f24" // 功能键 F24

	KeyCmd Key = "cmd" // Command 键 (Windows 上的 Win 键)
	// "command"
	KeyLCmd Key = "lcmd" // 左 Command 键
	KeyRCmd Key = "rcmd" // 右 Command 键

	KeyAlt  Key = "alt"  // Alt 键
	KeyLAlt Key = "lalt" // 左 Alt 键
	KeyRAlt Key = "ralt" // 右 Alt 键

	KeyCtrl    Key = "ctrl"    // Ctrl 键
	KeyLCtrl   Key = "lctrl"   // 左 Ctrl 键
	KeyRCtrl   Key = "rctrl"   // 右 Ctrl 键
	KeyControl Key = "control" // Control 键 (别名)

	KeyShift  Key = "shift"  // Shift 键
	KeyLShift Key = "lshift" // 左 Shift 键
	KeyRShift Key = "rshift" // 右 Shift 键
	// "right_shift"

	KeyCapsLock Key = "capslock" // 大写锁定键
	KeySpace    Key = "space"    // 空格键

	KeyPrint       Key = "print"       // Print 键
	KeyPrintScreen Key = "printscreen" // PrintScreen 键 (Mac 不支持)
	KeyInsert      Key = "insert"      // Insert 键
	KeyMenu        Key = "menu"        // 菜单键 (仅 Windows)

	KeyAudioMute    Key = "audio_mute"     // 静音
	KeyAudioVolDown Key = "audio_vol_down" // 降低音量
	KeyAudioVolUp   Key = "audio_vol_up"   // 提高音量
	KeyAudioPlay    Key = "audio_play"     // 播放
	KeyAudioStop    Key = "audio_stop"     // 停止
	KeyAudioPause   Key = "audio_pause"    // 暂停
	KeyAudioPrev    Key = "audio_prev"     // 上一曲
	KeyAudioNext    Key = "audio_next"     // 下一曲
	KeyAudioRewind  Key = "audio_rewind"   // 快退 (仅 Linux)
	KeyAudioForward Key = "audio_forward"  // 快进 (仅 Linux)
	KeyAudioRepeat  Key = "audio_repeat"   // 重复 (仅 Linux)
	KeyAudioRandom  Key = "audio_random"   // 随机播放 (仅 Linux)

	KeyNum0 Key = "num0" // 小键盘数字键 0
	KeyNum1 Key = "num1" // 小键盘数字键 1
	KeyNum2 Key = "num2" // 小键盘数字键 2
	KeyNum3 Key = "num3" // 小键盘数字键 3
	KeyNum4 Key = "num4" // 小键盘数字键 4
	KeyNum5 Key = "num5" // 小键盘数字键 5
	KeyNum6 Key = "num6" // 小键盘数字键 6
	KeyNum7 Key = "num7" // 小键盘数字键 7
	KeyNum8 Key = "num8" // 小键盘数字键 8
	KeyNum9 Key = "num9" // 小键盘数字键 9

	KeyNumLock Key = "num_lock" // 数字锁定键

	KeyNumDot      Key = "num."      // 小键盘小数点
	KeyNumPlus     Key = "num+"      // 小键盘加号
	KeyNumMinus    Key = "num-"      // 小键盘减号
	KeyNumMultiply Key = "num*"      // 小键盘乘号
	KeyNumDivide   Key = "num/"      // 小键盘除号
	KeyNumClear    Key = "num_clear" // 小键盘清空键
	KeyNumEnter    Key = "num_enter" // 小键盘回车键
	KeyNumEqual    Key = "num_equal" // 小键盘等号键

	KeyNumpad0    Key = "numpad_0"    // Linux 不支持
	KeyNumpad1    Key = "numpad_1"    // Linux 不支持
	KeyNumpad2    Key = "numpad_2"    // Linux 不支持
	KeyNumpad3    Key = "numpad_3"    // Linux 不支持
	KeyNumpad4    Key = "numpad_4"    // Linux 不支持
	KeyNumpad5    Key = "numpad_5"    // Linux 不支持
	KeyNumpad6    Key = "numpad_6"    // Linux 不支持
	KeyNumpad7    Key = "numpad_7"    // Linux 不支持
	KeyNumpad8    Key = "numpad_8"    // Linux 不支持
	KeyNumpad9    Key = "numpad_9"    // Linux 不支持
	KeyNumpadLock Key = "numpad_lock" // Linux 不支持

	KeyLightsMonUp     Key = "lights_mon_up"     // 提高显示器亮度 (Windows 不支持)
	KeyLightsMonDown   Key = "lights_mon_down"   // 降低显示器亮度 (Windows 不支持)
	KeyLightsKbdToggle Key = "lights_kbd_toggle" // 切换键盘背光开/关 (Windows 不支持)
	KeyLightsKbdUp     Key = "lights_kbd_up"     // 提高键盘背光亮度 (Windows 不支持)
	KeyLightsKbdDown   Key = "lights_kbd_down"   // 降低键盘背光亮度 (Windows 不支持)
)
