package guiagent

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/llminterface"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/tools/gui"
)

// GUIAgent 是一个用于 GUI 操作的 Agent
type GUIAgent struct {
	llm    llminterface.ChatAdapter
	tool   *gui.Tool
	logger *slog.Logger
}

// NewGUIAgent 创建一个新的GUIAgent实例
func NewGUIAgent(ctx context.Context, llm llminterface.ChatAdapter) (*GUIAgent, error) {
	logger := slog.Default().WithGroup("gui_agent")

	return &GUIAgent{
		llm:    llm,
		tool:   gui.NewGUITool(i18n.GlobalManager),
		logger: logger,
	}, nil
}

// Execute 执行 GUI 操作
//
// 输入应该是一条 GUI 指令，比如 "移动鼠标到(100, 100)"，一般是较小的指令
func (a *GUIAgent) Execute(ctx context.Context, instruction, imageURL string) (*ExecutionResult, error) {
	llmResponse, err := llminterface.ChatAndGetFullResponse(ctx, a.llm, llminterface.ChatInput{
		Messages: []llminterface.InputMessage{
			{
				Role: llminterface.RoleSystem,
				Content: []llminterface.ContentPart{
					{Type: llminterface.PartTypeText, Text: i18n.GlobalManager.T(ctx, "assistant.gui.prompt", nil)},
				},
			},
			{
				Role: llminterface.RoleUser,
				Content: []llminterface.ContentPart{
					{Type: llminterface.PartTypeText, Text: instruction},
					{Type: llminterface.PartTypeImageURL, ImageURL: Some(llminterface.ImageURLContent{
						URL: imageURL,
					})},
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("GUIAgent: failed to execute: %w", err)
	}

	parsedJSONResult, err := ParseActionOutput(llmResponse.FullText)
	if err != nil {
		return nil, fmt.Errorf("GUIAgent: failed to parse action output: %w", err)
	}

	// 将解析结果从JSON字符串转换为ActionResult结构体
	var actionResult ActionResult
	if err := json.Unmarshal([]byte(parsedJSONResult), &actionResult); err != nil {
		return nil, fmt.Errorf("GUIAgent: failed to unmarshal action result: %w", err)
	}

	// 准备返回结果
	result := &ExecutionResult{
		ActionResult: actionResult,
	}

	// 根据动作类型执行相应的GUI操作
	executeResult, err := a.executeAction(ctx, actionResult)
	if err != nil {
		// 即使执行失败，也返回解析的结构，但附带错误信息
		result.ExecutionOutput = fmt.Sprintf("执行失败: %s", err.Error())
		return result, fmt.Errorf("GUIAgent: failed to execute action: %w", err)
	}

	// 设置执行结果
	result.ExecutionOutput = executeResult

	return result, nil
}

// ExecutionResult 表示GUI操作的最终执行结果
type ExecutionResult struct {
	ActionResult
	ExecutionOutput string `json:"execution_result,omitempty"`
}

// executeAction 根据动作类型执行相应的GUI操作
//
//nolint:gocyclo // 确实需要那么长
func (a *GUIAgent) executeAction(_ context.Context, action ActionResult) (string, error) {
	var screenshotResultMap map[string]any
	var imgWidth, imgHeight int

	// 如果操作需要坐标信息，获取截图尺寸
	if action.StartBox != nil || action.EndBox != nil {
		// 重新截图以获取当前屏幕尺寸，或者从传入的图像URL提取尺寸
		screenshot, err := a.tool.Screenshot()
		if err != nil {
			return "", fmt.Errorf("获取屏幕尺寸失败: %w", err)
		}

		if err := json.Unmarshal([]byte(screenshot), &screenshotResultMap); err != nil {
			return "", fmt.Errorf("解析截图信息失败: %w", err)
		}

		imgWidth = int(screenshotResultMap["width"].(float64))
		imgHeight = int(screenshotResultMap["height"].(float64))
	}

	// 根据动作类型执行相应的操作
	switch action.Action {
	case "click":
		if action.StartBox == nil || len(action.StartBox) != 4 {
			return "", fmt.Errorf("点击操作需要有效的start_box参数")
		}

		// 将相对坐标转换为绝对像素坐标
		absCoords, err := CoordinatesConvert(action.StartBox, [2]int{imgWidth, imgHeight})
		if err != nil {
			return "", fmt.Errorf("坐标转换失败: %w", err)
		}

		// 计算点击的中心点
		centerX := (absCoords[0] + absCoords[2]) / 2
		centerY := (absCoords[1] + absCoords[3]) / 2

		// 首先移动到目标位置
		moveInput := fmt.Sprintf(`{"x": %d, "y": %d}`, centerX, centerY)
		moveResult, err := a.tool.MoveMouse(moveInput)
		if err != nil {
			return "", fmt.Errorf("移动鼠标失败: %w", err)
		}

		// 然后执行单击
		clickInput := `{"button": "left", "double": false}`
		clickResult, err := a.tool.Click(clickInput)
		if err != nil {
			return "", fmt.Errorf("点击失败: %w", err)
		}

		return fmt.Sprintf("移动鼠标并点击: %s, %s", moveResult, clickResult), nil

	case "left_double":
		if action.StartBox == nil || len(action.StartBox) != 4 {
			return "", fmt.Errorf("双击操作需要有效的start_box参数")
		}

		absCoords, err := CoordinatesConvert(action.StartBox, [2]int{imgWidth, imgHeight})
		if err != nil {
			return "", fmt.Errorf("坐标转换失败: %w", err)
		}

		centerX := (absCoords[0] + absCoords[2]) / 2
		centerY := (absCoords[1] + absCoords[3]) / 2

		moveInput := fmt.Sprintf(`{"x": %d, "y": %d}`, centerX, centerY)
		_, err = a.tool.MoveMouse(moveInput)
		if err != nil {
			return "", fmt.Errorf("移动鼠标失败: %w", err)
		}

		clickInput := `{"button": "left", "double": true}`
		clickResult, err := a.tool.Click(clickInput)
		if err != nil {
			return "", fmt.Errorf("双击失败: %w", err)
		}

		return fmt.Sprintf("双击: %s", clickResult), nil

	case "right_single":
		if action.StartBox == nil || len(action.StartBox) != 4 {
			return "", fmt.Errorf("右键单击操作需要有效的start_box参数")
		}

		absCoords, err := CoordinatesConvert(action.StartBox, [2]int{imgWidth, imgHeight})
		if err != nil {
			return "", fmt.Errorf("坐标转换失败: %w", err)
		}

		centerX := (absCoords[0] + absCoords[2]) / 2
		centerY := (absCoords[1] + absCoords[3]) / 2

		moveInput := fmt.Sprintf(`{"x": %d, "y": %d}`, centerX, centerY)
		_, err = a.tool.MoveMouse(moveInput)
		if err != nil {
			return "", fmt.Errorf("移动鼠标失败: %w", err)
		}

		clickInput := `{"button": "right", "double": false}`
		clickResult, err := a.tool.Click(clickInput)
		if err != nil {
			return "", fmt.Errorf("右键点击失败: %w", err)
		}

		return fmt.Sprintf("右键点击: %s", clickResult), nil

	case "drag":
		if action.StartBox == nil || action.EndBox == nil || len(action.StartBox) != 4 || len(action.EndBox) != 4 {
			return "", fmt.Errorf("拖拽操作需要有效的start_box和end_box参数")
		}

		startCoords, err := CoordinatesConvert(action.StartBox, [2]int{imgWidth, imgHeight})
		if err != nil {
			return "", fmt.Errorf("起始坐标转换失败: %w", err)
		}

		endCoords, err := CoordinatesConvert(action.EndBox, [2]int{imgWidth, imgHeight})
		if err != nil {
			return "", fmt.Errorf("结束坐标转换失败: %w", err)
		}

		startX := (startCoords[0] + startCoords[2]) / 2
		startY := (startCoords[1] + startCoords[3]) / 2
		endX := (endCoords[0] + endCoords[2]) / 2
		endY := (endCoords[1] + endCoords[3]) / 2

		// 先移动到起始位置
		moveInput := fmt.Sprintf(`{"x": %d, "y": %d}`, startX, startY)
		_, err = a.tool.MoveMouse(moveInput)
		if err != nil {
			return "", fmt.Errorf("移动鼠标到起始位置失败: %w", err)
		}

		// 拖动到目标位置（包含先按下鼠标左键，等待一会，然后移动到目标位置，再释放鼠标左键）
		dragInput := fmt.Sprintf(`{"x": %d, "y": %d}`, endX, endY)
		dragResult, err := a.tool.Drag(dragInput)
		if err != nil {
			// 如果拖动失败，确保释放鼠标按钮
			_, err = a.tool.ToggleMouseButton(`{"button": "left", "up": true}`)
			if err != nil {
				return "", fmt.Errorf("拖动失败: %w", err)
			}
			return "", fmt.Errorf("拖动失败: %w", err)
		}

		return fmt.Sprintf("拖拽: %s", dragResult), nil

	case "key_tap":
		if action.Action == "key_tap" && action.Keys.IsNone() {
			return "", fmt.Errorf("按键操作需要有效的key或keys参数")
		}

		keys := action.Keys.Unwrap()

		keyTapInput := fmt.Sprintf(`{"keys": %s}`, MustMarshalJSONWithoutPanic(keys))
		keyTapResult, err := a.tool.KeyTap(keyTapInput)
		if err != nil {
			return "", fmt.Errorf("按键失败: %w", err)
		}

		return fmt.Sprintf("按键: %s", keyTapResult), nil

	case "type":
		if action.Content == nil || action.Content.IsNone() {
			return "", fmt.Errorf("输入操作需要有效的content参数")
		}

		typeInput := fmt.Sprintf(`{"content": %s}`, MustMarshalJSONWithoutPanic(action.Content.Unwrap()))
		typeResult, err := a.tool.TypeString(typeInput)
		if err != nil {
			return "", fmt.Errorf("输入文本失败: %w", err)
		}

		return fmt.Sprintf("输入: %s", typeResult), nil

	case "scroll":
		if action.StartBox == nil || action.Direction.IsNone() || len(action.StartBox) != 4 || action.Direction.IsNone() {
			return "", fmt.Errorf("滚动操作需要有效的start_box和direction参数")
		}

		absCoords, err := CoordinatesConvert(action.StartBox, [2]int{imgWidth, imgHeight})
		if err != nil {
			return "", fmt.Errorf("坐标转换失败: %w", err)
		}

		centerX := (absCoords[0] + absCoords[2]) / 2
		centerY := (absCoords[1] + absCoords[3]) / 2

		// 先移动到目标位置
		moveInput := fmt.Sprintf(`{"x": %d, "y": %d}`, centerX, centerY)
		_, err = a.tool.MoveMouse(moveInput)
		if err != nil {
			return "", fmt.Errorf("移动鼠标失败: %w", err)
		}

		// 执行滚动
		scrollInput := fmt.Sprintf(`{"x": 10, "direction": %s}`, MustMarshalJSONWithoutPanic(action.Direction.Unwrap()))
		scrollResult, err := a.tool.ScrollDirection(scrollInput)
		if err != nil {
			return "", fmt.Errorf("滚动失败: %w", err)
		}

		return fmt.Sprintf("滚动: %s", scrollResult), nil

	case "toggle_key":
		if action.Keys.IsNone() || action.Up.IsNone() {
			return "", fmt.Errorf("切换键操作需要有效的keys和up参数")
		}

		keys := action.Keys.Unwrap()
		up := action.Up.Unwrap()

		toggleKeyInput := fmt.Sprintf(`{"keys": %s, "up": %t}`, MustMarshalJSONWithoutPanic(keys), up)
		toggleKeyResult, err := a.tool.ToggleKey(toggleKeyInput)
		if err != nil {
			return "", fmt.Errorf("切换键操作失败: %w", err)
		}

		return fmt.Sprintf("切换键: %s", toggleKeyResult), nil

	case "toggle_mouse":
		if action.Button.IsNone() || action.Up.IsNone() {
			return "", fmt.Errorf("切换鼠标按钮操作需要有效的button和up参数")
		}

		button := action.Button.Unwrap()
		up := action.Up.Unwrap()

		toggleMouseInput := fmt.Sprintf(`{"button": %s, "up": %t}`, MustMarshalJSONWithoutPanic(button), up)
		toggleMouseResult, err := a.tool.ToggleMouseButton(toggleMouseInput)
		if err != nil {
			return "", fmt.Errorf("切换鼠标按钮操作失败: %w", err)
		}

		return fmt.Sprintf("切换鼠标按钮: %s", toggleMouseResult), nil

	case "mouse_location":
		locationResult, err := a.tool.MouseLocation()
		if err != nil {
			return "", fmt.Errorf("获取鼠标位置失败: %w", err)
		}

		return fmt.Sprintf("鼠标位置: %s", locationResult), nil

	case "scroll_relative":
		if action.X.IsNone() || action.Y.IsNone() {
			return "", fmt.Errorf("相对滚动操作需要有效的x和y参数")
		}

		x := action.X.Unwrap()
		y := action.Y.Unwrap()
		msDelay := 10 // 默认值
		if action.MS.IsSome() {
			msDelay = action.MS.Unwrap()
		}

		scrollRelativeInput := fmt.Sprintf(`{"x": %d, "y": %d, "ms_delay": %d}`, x, y, msDelay)
		scrollRelativeResult, err := a.tool.ScrollRelative(scrollRelativeInput)
		if err != nil {
			return "", fmt.Errorf("相对滚动操作失败: %w", err)
		}

		return fmt.Sprintf("相对滚动: %s", scrollRelativeResult), nil

	case "key_sleep":
		if action.MS.IsNone() {
			return "", fmt.Errorf("按键睡眠操作需要有效的ms参数")
		}

		ms := action.MS.Unwrap()
		keySleepInput := fmt.Sprintf(`{"ms": %d}`, ms)
		keySleepResult, err := a.tool.KeySleepMilli(keySleepInput)
		if err != nil {
			return "", fmt.Errorf("设置按键睡眠时间失败: %w", err)
		}

		return fmt.Sprintf("设置按键睡眠时间: %s", keySleepResult), nil

	case "wait":
		// 默认等待5秒，但如果指定了时间则使用指定的时间
		waitTime := 5000 // 默认值，毫秒
		if action.MS.IsSome() {
			waitTime = action.MS.Unwrap()
		}

		sleepInput := fmt.Sprintf(`{"ms": %d}`, waitTime)
		sleepResult, err := a.tool.SleepMilli(sleepInput)
		if err != nil {
			return "", fmt.Errorf("等待失败: %w", err)
		}

		// 截图检查变化
		screenshot, err := a.tool.Screenshot()
		if err != nil {
			return "", fmt.Errorf("截图失败: %w", err)
		}

		// 可以根据需要调整这里的返回信息，包含截图内容的长度或其他信息
		var screenshotResult map[string]any
		if err := json.Unmarshal([]byte(screenshot), &screenshotResult); err != nil {
			return "", fmt.Errorf("解析截图信息失败: %w", err)
		}

		return fmt.Sprintf("等待: %s, 并截图检查变化 (截图宽度: %v, 高度: %v)",
			sleepResult,
			screenshotResult["width"],
			screenshotResult["height"]), nil

	case "finished":
		content := "任务完成"
		if action.Content.IsSome() {
			content = action.Content.Unwrap()
		}
		return fmt.Sprintf("任务完成: %s", content), nil

	default:
		return "", fmt.Errorf("未知的动作类型: %s", action.Action)
	}
}

// MustMarshalJSONWithoutPanic 将任何值转换为JSON字符串
func MustMarshalJSONWithoutPanic(v any) string {
	json, err := json.MarshalToString(v)
	if err != nil {
		slog.Error("failed to marshal json", "error", err)
	}
	return json
}
