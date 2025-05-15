package tool

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"log/slog"

	"github.com/go-vgo/robotgo"
	"github.com/m4n5ter/another-me/src/locale"
	"github.com/mark3labs/mcp-go/mcp"
)

var ScreenshotTool = mcp.NewTool("screenshot",
	mcp.WithDescription(locale.ScreenshotDescription()),
)

// TODO: 其它MCP工具定义

type GUITool struct {
	logger *slog.Logger
}

func NewGUITool(logger *slog.Logger) GUITool {
	return GUITool{logger: logger.WithGroup("gui_tool")}
}

// Screenshot 捕获一张默认桌面的截图: png base64 url
func (c *GUITool) Screenshot() (base64PNGURL string, rect image.Rectangle, err error) {
	rgba, err := robotgo.Capture()
	if err != nil {
		c.logger.Error("Failed to capture screen", "error", err)
		return "", image.Rectangle{}, fmt.Errorf("failed to capture screen: %w", err)
	}

	rect = rgba.Bounds()

	buf := new(bytes.Buffer)
	err = png.Encode(buf, rgba)
	if err != nil {
		c.logger.Error("Failed to encode screenshot", "error", err)
		return "", rect, fmt.Errorf("failed to encode screenshot: %w", err)
	}

	base64PNG := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:image/png;base64,%s", base64PNG), rect, nil
}

// MoveMouse 移动鼠标
func (c *GUITool) MoveMouse(x, y int) {
	robotgo.MoveSmooth(x, y)
}

// MouseLocation 获取鼠标当前坐标位置
func (c *GUITool) MouseLocation() (x, y int) {
	return robotgo.Location()
}

// Drag 拖动
func (c *GUITool) Drag(x, y int) {
	robotgo.DragSmooth(x, y)
}

// Scroll 滚动，滚动到y轴具体位置，滚动几次，每次间隔多久，滚动到x轴具体位置
func (c *GUITool) Scroll(toy, num, msSleep, tox int) {
	robotgo.ScrollSmooth(toy, num, msSleep, tox)
}

// ScrollRelative 相对坐标滚动，滚动的x，y以及毫秒延迟
func (c *GUITool) ScrollRelative(x, y, msDeplay int) {
	robotgo.ScrollRelative(x, y, msDeplay)
}

// ScrollDirection 选择指定方向滚动
func (c *GUITool) ScrollDirection(x int, direction Direction) {
	switch direction {
	case DirectionUp, DirectionDown, DirectionLeft, DirectionRight:
	default:
		direction = DirectionUp
	}

	robotgo.ScrollDir(x, string(direction))
}

type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

// Click 点击（按下+松开）
func (c *GUITool) Click(button MouseButton, double bool) {
	switch button {
	case MouseButtonLeft, MouseButtonRight, MouseButtonWheelLeft, MouseButtonWheelRight:
	default:
		button = MouseButtonLeft
	}

	robotgo.Click(string(button), double)
}

// Toggle 操作鼠标按键
func (c *GUITool) ToggleMouseButton(button MouseButton, up bool) {
	switch button {
	case MouseButtonLeft, MouseButtonRight, MouseButtonWheelLeft, MouseButtonWheelRight:
	default:
		button = MouseButtonLeft
	}

	direction := "down"
	if up {
		direction = "up"
	}

	robotgo.Toggle(string(button), direction)
}

// ToggleKey 操作键盘按键（切换）
func (c *GUITool) ToggleKey(up bool, keys ...Key) {
	if len(keys) < 1 {
		return
	}

	direction := "down"
	if up {
		direction = "up"
	}

	k0 := string(keys[0])

	if len(keys) == 1 {
		robotgo.KeyToggle(k0, direction)
		return
	}

	keysLeft := make([]string, 0, len(keys)-1)
	for _, key := range keys[1:] {
		keysLeft = append(keysLeft, string(key))
	}

	robotgo.KeyToggle(k0, direction, keysLeft)
}

// KeyTap 操作键盘按键（点按）
func (c *GUITool) KeyTap(keys ...Key) {
	if len(keys) < 1 {
		return
	}

	k0 := string(keys[0])

	if len(keys) == 1 {
		robotgo.KeyTap(k0)
		return
	}

	keysLeft := make([]string, 0, len(keys)-1)
	for _, key := range keys[1:] {
		keysLeft = append(keysLeft, string(key))
	}

	robotgo.KeyTap(k0, keysLeft)
}

// KeySleepMilli 设置按键睡眠时间（毫秒）
func (c *GUITool) KeySleepMilli(ms int) {
	robotgo.KeySleep = ms
}

// ---- MCP 工具定义开始 ----

func (c *GUITool) ScreenshotMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	base64PNGURL, _, err := c.Screenshot()
	if err != nil {
		return nil, err
	}

	return mcp.NewToolResultText(base64PNGURL), nil
}

func (c *GUITool) MoveMouseMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	x, ok := request.Params.Arguments["x"].(int)
	if !ok {
		return nil, fmt.Errorf("x is not an int")
	}

	y, ok := request.Params.Arguments["y"].(int)
	if !ok {
		return nil, fmt.Errorf("y is not an int")
	}

	c.MoveMouse(x, y)

	newX, newY := c.MouseLocation()

	return mcp.NewToolResultText(fmt.Sprintf("current mouse location: x: %d, y: %d", newX, newY)), nil
}

func (c *GUITool) MouseLocationMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	x, y := c.MouseLocation()
	return mcp.NewToolResultText(fmt.Sprintf("current mouse location: x: %d, y: %d", x, y)), nil
}

func (c *GUITool) DragMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	x, ok := request.Params.Arguments["x"].(int)
	if !ok {
		return nil, fmt.Errorf("x is not an int")
	}

	y, ok := request.Params.Arguments["y"].(int)
	if !ok {
		return nil, fmt.Errorf("y is not an int")
	}

	c.Drag(x, y)

	newX, newY := c.MouseLocation()

	return mcp.NewToolResultText(fmt.Sprintf("current mouse location: x: %d, y: %d", newX, newY)), nil
}

func (c *GUITool) ScrollMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	toy, ok := request.Params.Arguments["toy"].(int)
	if !ok {
		return nil, fmt.Errorf("toy is not an int")
	}

	num, ok := request.Params.Arguments["num"].(int)
	if !ok {
		return nil, fmt.Errorf("num is not an int")
	}

	msSleep, ok := request.Params.Arguments["msSleep"].(int)
	if !ok {
		return nil, fmt.Errorf("msSleep is not an int")
	}

	tox, ok := request.Params.Arguments["tox"].(int)
	if !ok {
		return nil, fmt.Errorf("tox is not an int")
	}

	c.Scroll(toy, num, msSleep, tox)

	return mcp.NewToolResultText(fmt.Sprintf("scroll success")), nil
}

func (c *GUITool) ScrollRelativeMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	x, ok := request.Params.Arguments["x"].(int)
	if !ok {
		return nil, fmt.Errorf("x is not an int")
	}

	y, ok := request.Params.Arguments["y"].(int)
	if !ok {
		return nil, fmt.Errorf("y is not an int")
	}

	msDeplay, ok := request.Params.Arguments["msDeplay"].(int)
	if !ok {
		return nil, fmt.Errorf("msDeplay is not an int")
	}

	c.ScrollRelative(x, y, msDeplay)

	return mcp.NewToolResultText(fmt.Sprintf("scroll success")), nil
}

func (c *GUITool) ScrollDirectionMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	x, ok := request.Params.Arguments["x"].(int)
	if !ok {
		return nil, fmt.Errorf("x is not an int")
	}

	direction, ok := request.Params.Arguments["direction"].(string)
	if !ok {
		return nil, fmt.Errorf("direction is not a string")
	}

	c.ScrollDirection(x, Direction(direction))

	return mcp.NewToolResultText(fmt.Sprintf("scroll success")), nil
}

func (c *GUITool) ClickMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	button, ok := request.Params.Arguments["button"].(string)
	if !ok {
		return nil, fmt.Errorf("button is not a string")
	}

	double, ok := request.Params.Arguments["double"].(bool)
	if !ok {
		return nil, fmt.Errorf("double is not a bool")
	}

	c.Click(MouseButton(button), double)

	return mcp.NewToolResultText(fmt.Sprintf("click success")), nil
}

func (c *GUITool) ToggleMouseButtonMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	button, ok := request.Params.Arguments["button"].(string)
	if !ok {
		return nil, fmt.Errorf("button is not a string")
	}

	up, ok := request.Params.Arguments["up"].(bool)
	if !ok {
		return nil, fmt.Errorf("up is not a bool")
	}

	c.ToggleMouseButton(MouseButton(button), up)

	return mcp.NewToolResultText(fmt.Sprintf("toggle mouse button success")), nil
}

func (c *GUITool) ToggleKeyMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	up, ok := request.Params.Arguments["up"].(bool)
	if !ok {
		return nil, fmt.Errorf("up is not a bool")
	}

	keys, ok := request.Params.Arguments["keys"].([]string)
	if !ok {
		return nil, fmt.Errorf("keys is not a []string")
	}

	keys2 := make([]Key, 0, len(keys))
	for _, key := range keys {
		keys2 = append(keys2, Key(key))
	}

	c.ToggleKey(up, keys2...)

	return mcp.NewToolResultText(fmt.Sprintf("toggle key success")), nil
}

func (c *GUITool) KeyTapMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	keys, ok := request.Params.Arguments["keys"].([]string)
	if !ok {
		return nil, fmt.Errorf("keys is not a []string")
	}

	keys2 := make([]Key, 0, len(keys))
	for _, key := range keys {
		keys2 = append(keys2, Key(key))
	}

	c.KeyTap(keys2...)

	return mcp.NewToolResultText(fmt.Sprintf("key tap success")), nil
}

func (c *GUITool) KeySleepMilliMCP(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ms, ok := request.Params.Arguments["ms"].(int)
	if !ok {
		return nil, fmt.Errorf("ms is not an int")
	}

	c.KeySleepMilli(ms)

	return mcp.NewToolResultText(fmt.Sprintf("key sleep milli success")), nil
}

// ---- MCP 工具定义结束 ----

type MouseButton string

const (
	MouseButtonLeft       MouseButton = "left"       // 鼠标左键
	MouseButtonRight      MouseButton = "right"      // 鼠标右键
	MouseButtonWheelLeft  MouseButton = "wheelLeft"  // 鼠标滚轮左键
	MouseButtonWheelRight MouseButton = "wheelRight" // 鼠标滚轮右键
	MouseButtonWheelDown  MouseButton = "wheelDown"  // 鼠标滚轮下键?
	MouseButtonWheelUp    MouseButton = "wheelUp"    // 鼠标滚轮上键?
)

type Key string

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
