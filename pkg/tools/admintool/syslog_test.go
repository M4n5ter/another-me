package admintool_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/tools/admintool"
)

// TestSyslogTool_GetLogs 测试获取系统日志功能
func TestSyslogTool_GetLogs(t *testing.T) {
	// 准备测试环境
	mockRunner := new(MockCommandRunner)
	tool := admintool.NewSyslogToolWithRunner(i18n.GlobalManager, mockRunner)
	ctx := context.Background()

	// 测试场景1: 获取系统日志
	t.Run("GetSystemLogs", func(t *testing.T) {
		// 模拟执行命令
		mockOutput := `May 21 15:10:00 localhost systemd[1]: Starting System Logging Service...
May 21 15:10:01 localhost systemd[1]: Started System Logging Service.
May 21 15:10:05 localhost systemd[1]: Stopping System Logging Service...
May 21 15:10:06 localhost systemd[1]: Stopped System Logging Service.
May 21 15:10:10 localhost systemd[1]: Starting System Logging Service...`

		mockRunner.On("Run", ctx, "journalctl", "--lines", "5", "-u", "systemd").
			Return([]byte(mockOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"lines":5,"log_type":"system"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"lines":5`)
		assert.Contains(t, result, "Starting System Logging Service")
		mockRunner.AssertExpectations(t)
	})

	// 测试场景2: journalctl失败，使用tail回退
	t.Run("JournalctlFailFallbackToTail", func(t *testing.T) {
		// 模拟journalctl失败
		mockRunner.On("Run", ctx, "journalctl", "--lines", "3").
			Return([]byte{}, errors.New("command failed")).Once()

		// 模拟tail成功
		tailOutput := `May 21 15:20:00 localhost kernel: Process terminated
May 21 15:20:05 localhost kernel: New process started
May 21 15:20:10 localhost kernel: System event detected`

		mockRunner.On("Run", ctx, "tail", "-3", "/var/log/syslog").
			Return([]byte(tailOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"lines":3}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"lines":3`)
		assert.Contains(t, result, "System event detected")
		mockRunner.AssertExpectations(t)
	})

	// 测试场景3: 获取内核日志
	t.Run("GetKernelLogs", func(t *testing.T) {
		// 模拟执行dmesg命令
		mockOutput := `[    0.000000] Linux version 5.10.0
[    0.000123] Command line: BOOT_IMAGE=/boot/vmlinuz`

		mockRunner.On("Run", ctx, "dmesg", "-l", "kern", "--nlines", "2").
			Return([]byte(mockOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"lines":2,"log_type":"kernel"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"lines":2`)
		assert.Contains(t, result, "Linux version")
		mockRunner.AssertExpectations(t)
	})
}
