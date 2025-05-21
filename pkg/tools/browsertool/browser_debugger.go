package browsertool

import (
	"context"
	"fmt"
	"time"

	"github.com/chromedp/cdproto/debugger"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	json "github.com/json-iterator/go"
)

// debugEnableDisable 启用或禁用调试模式
func (t *BrowserTool) debugEnableDisable(args BrowserArgs) (BrowserResult, error) {
	if args.Debug == nil {
		return BrowserResult{}, fmt.Errorf("调试参数不能为空")
	}

	// 设置超时上下文
	timeoutCtx, cancel := context.WithTimeout(t.browserCtx, time.Duration(t.config.URLTimeout)*time.Second)
	defer cancel()

	var action string
	if args.Debug.Enable {
		action = "启用"
		err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			_, err := debugger.Enable().Do(ctx)
			if err != nil {
				return fmt.Errorf("无法启用调试模式: %w", err)
			}
			return nil
		}))
		if err != nil {
			t.logger.Error("启用调试失败", "error", err)
			return BrowserResult{
				Operation: OperationDebug,
				Success:   false,
				Message:   fmt.Sprintf("启用调试失败: %s", err),
			}, nil
		}
	} else {
		action = "禁用"
		err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
			err := debugger.Disable().Do(ctx)
			if err != nil {
				return fmt.Errorf("无法禁用调试模式: %w", err)
			}
			return nil
		}))
		if err != nil {
			t.logger.Error("禁用调试失败", "error", err)
			return BrowserResult{
				Operation: OperationDebug,
				Success:   false,
				Message:   fmt.Sprintf("禁用调试失败: %s", err),
			}, nil
		}
	}

	return BrowserResult{
		Operation: OperationDebug,
		Success:   true,
		Message:   fmt.Sprintf("成功%s调试模式", action),
	}, nil
}

// debugSetBreakpoint 设置断点
func (t *BrowserTool) debugSetBreakpoint(args BrowserArgs) (BrowserResult, error) {
	if args.Debug == nil {
		return BrowserResult{}, fmt.Errorf("调试参数不能为空")
	}

	if args.Debug.URL == "" || args.Debug.LineNumber <= 0 {
		return BrowserResult{}, fmt.Errorf("断点URL和行号不能为空")
	}

	// 设置超时上下文
	timeoutCtx, cancel := context.WithTimeout(t.browserCtx, time.Duration(t.config.URLTimeout)*time.Second)
	defer cancel()

	// 首先启用调试器
	if err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		_, err := debugger.Enable().Do(ctx)
		if err != nil {
			return fmt.Errorf("无法启用调试器: %w", err)
		}
		return nil
	})); err != nil {
		t.logger.Error("启用调试器失败", "error", err)
		return BrowserResult{
			Operation: OperationDebug,
			Success:   false,
			Message:   fmt.Sprintf("启用调试器失败: %s", err),
		}, nil
	}

	// 设置断点参数
	params := &debugger.SetBreakpointByURLParams{
		URL:        args.Debug.URL,
		LineNumber: int64(args.Debug.LineNumber),
	}

	if args.Debug.ColumnNumber > 0 {
		params.ColumnNumber = int64(args.Debug.ColumnNumber)
	}

	if args.Debug.Condition != "" {
		params.Condition = args.Debug.Condition
	}

	var breakpointID debugger.BreakpointID
	var locations []*debugger.Location
	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		breakpointID, locations, err = params.Do(ctx)
		if err != nil {
			return fmt.Errorf("无法设置断点: %w", err)
		}
		return nil
	}))
	if err != nil {
		t.logger.Error("设置断点失败", "url", args.Debug.URL, "line", args.Debug.LineNumber, "error", err)
		return BrowserResult{
			Operation: OperationDebug,
			Success:   false,
			Message:   fmt.Sprintf("设置断点失败: %s", err),
		}, nil
	}

	// 将断点信息格式化为JSON
	breakpointInfo := map[string]any{
		"breakpoint_id": string(breakpointID),
		"locations":     locations,
	}

	breakpointJSON, err := json.Marshal(breakpointInfo)
	if err != nil {
		breakpointJSON = []byte(fmt.Sprintf(`{"breakpoint_id":"%s"}`, breakpointID))
	}

	return BrowserResult{
		Operation: OperationDebug,
		Success:   true,
		Message:   fmt.Sprintf("在 %s 的第 %d 行成功设置断点", args.Debug.URL, args.Debug.LineNumber),
		DebugInfo: string(breakpointJSON),
	}, nil
}

// debugRemoveBreakpoint 移除断点
func (t *BrowserTool) debugRemoveBreakpoint(args BrowserArgs) (BrowserResult, error) {
	if args.Debug == nil || args.Debug.BreakpointID == "" {
		return BrowserResult{}, fmt.Errorf("断点ID不能为空")
	}

	// 设置超时上下文
	timeoutCtx, cancel := context.WithTimeout(t.browserCtx, time.Duration(t.config.URLTimeout)*time.Second)
	defer cancel()

	// 移除断点
	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return debugger.RemoveBreakpoint(debugger.BreakpointID(args.Debug.BreakpointID)).Do(ctx)
	}))
	if err != nil {
		t.logger.Error("移除断点失败", "breakpointID", args.Debug.BreakpointID, "error", err)
		return BrowserResult{
			Operation: OperationDebug,
			Success:   false,
			Message:   fmt.Sprintf("移除断点失败: %s", err),
		}, nil
	}

	return BrowserResult{
		Operation: OperationDebug,
		Success:   true,
		Message:   fmt.Sprintf("成功移除断点 %s", args.Debug.BreakpointID),
	}, nil
}

// debugPause 暂停JavaScript执行
func (t *BrowserTool) debugPause() (BrowserResult, error) {
	// 设置超时上下文
	timeoutCtx, cancel := context.WithTimeout(t.browserCtx, time.Duration(t.config.URLTimeout)*time.Second)
	defer cancel()

	// 暂停执行
	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return debugger.Pause().Do(ctx)
	}))
	if err != nil {
		t.logger.Error("暂停执行失败", "error", err)
		return BrowserResult{
			Operation: OperationDebug,
			Success:   false,
			Message:   fmt.Sprintf("暂停执行失败: %s", err),
		}, nil
	}

	return BrowserResult{
		Operation: OperationDebug,
		Success:   true,
		Message:   "成功暂停JavaScript执行",
	}, nil
}

// debugResume 恢复JavaScript执行
func (t *BrowserTool) debugResume() (BrowserResult, error) {
	// 设置超时上下文
	timeoutCtx, cancel := context.WithTimeout(t.browserCtx, time.Duration(t.config.URLTimeout)*time.Second)
	defer cancel()

	// 恢复执行
	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return debugger.Resume().Do(ctx)
	}))
	if err != nil {
		t.logger.Error("恢复执行失败", "error", err)
		return BrowserResult{
			Operation: OperationDebug,
			Success:   false,
			Message:   fmt.Sprintf("恢复执行失败: %s", err),
		}, nil
	}

	return BrowserResult{
		Operation: OperationDebug,
		Success:   true,
		Message:   "成功恢复JavaScript执行",
	}, nil
}

// debugGetCallstack 获取调用堆栈
func (t *BrowserTool) debugGetCallstack() (BrowserResult, error) {
	// 设置超时上下文
	timeoutCtx, cancel := context.WithTimeout(t.browserCtx, time.Duration(t.config.URLTimeout)*time.Second)
	defer cancel()

	// 获取调用堆栈
	var callFrames []*runtime.CallFrame
	err := chromedp.Run(timeoutCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		// 创建一个空的StackTraceID
		stackTraceID := &runtime.StackTraceID{}
		stackTraceResult, err := debugger.GetStackTrace(stackTraceID).Do(ctx)
		if err != nil {
			return fmt.Errorf("获取堆栈追踪失败: %w", err)
		}
		callFrames = stackTraceResult.CallFrames
		return nil
	}))
	if err != nil {
		t.logger.Error("获取调用堆栈失败", "error", err)
		return BrowserResult{
			Operation: OperationDebug,
			Success:   false,
			Message:   fmt.Sprintf("获取调用堆栈失败: %s", err),
		}, nil
	}

	// 将调用堆栈信息格式化为JSON
	callstackJSON, err := json.Marshal(callFrames)
	if err != nil {
		callstackJSON = []byte("[]")
	}

	return BrowserResult{
		Operation: OperationDebug,
		Success:   true,
		Message:   "成功获取调用堆栈",
		DebugInfo: string(callstackJSON),
	}, nil
}
