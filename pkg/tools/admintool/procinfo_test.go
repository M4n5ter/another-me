package admintool_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/tools/admintool"
)

// TestProcInfoTool_GetAllProcesses 测试获取所有进程信息功能
func TestProcInfoTool_GetAllProcesses(t *testing.T) {
	// 准备测试环境
	mockRunner := new(MockCommandRunner)
	tool := admintool.NewProcInfoToolWithRunner(i18n.GlobalManager, mockRunner)
	ctx := context.Background()

	// 测试场景1: 获取基本进程信息
	t.Run("GetBasicProcInfo", func(t *testing.T) {
		// 模拟执行命令
		mockCmd := "ps aux --sort=-cpu | head -6"
		mockOutput := `USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root      1234  2.0  0.5 123456 54321 ?        Ss   10:00   0:30 /usr/bin/process1
user1     2345  1.5  0.3  98765 32198 ?        S    11:00   0:20 /usr/bin/process2
user2     3456  1.0  0.2  65432 21098 ?        R    12:00   0:10 /usr/bin/process3
user3     4567  0.5  0.1  43210 10987 ?        S    13:00   0:05 /usr/bin/process4
user4     5678  0.3  0.1  32109  8765 ?        S    14:00   0:03 /usr/bin/process5`

		mockRunner.On("RunShell", ctx, mockCmd).
			Return([]byte(mockOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"info_type":"basic","count":5}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"info_type":"basic"`)
		assert.Contains(t, result, `"process_count":5`)
		assert.Contains(t, result, `/usr/bin/process1`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景2: 获取进程信息失败
	t.Run("GetProcInfoFail", func(t *testing.T) {
		// 模拟命令执行失败
		mockCmd := "ps aux --sort=-cpu | head -6"
		mockRunner.On("RunShell", ctx, mockCmd).
			Return([]byte{}, errors.New("command failed")).Once()

		// 执行被测函数
		inputJSON := `{"info_type":"basic","count":5}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "获取进程信息失败")
		mockRunner.AssertExpectations(t)
	})
}

// TestProcInfoTool_GetProcessByPID 测试根据PID获取进程信息功能
func TestProcInfoTool_GetProcessByPID(t *testing.T) {
	// 准备测试环境
	mockRunner := new(MockCommandRunner)
	tool := admintool.NewProcInfoToolWithRunner(i18n.GlobalManager, mockRunner)
	ctx := context.Background()

	// 测试场景1: 获取PID为1的进程信息
	t.Run("GetProcessByPID", func(t *testing.T) {
		// 模拟执行命令
		mockCmd := "ps -p 1 -o pid,ppid,user,stat,pcpu,pmem,comm,args"
		mockOutput := `  PID  PPID USER     STAT  %CPU %MEM COMMAND         COMMAND
    1     0 root     Ss     0.0  0.1 /sbin/init      /sbin/init`

		mockRunner.On("RunShell", ctx, mockCmd).
			Return([]byte(mockOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"pid":1,"info_type":"basic"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"pid":1`)
		assert.Contains(t, result, `"info_type":"basic"`)
		assert.Contains(t, result, `/sbin/init`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景2: PID不存在
	t.Run("PIDNotExist", func(t *testing.T) {
		// 模拟执行命令返回只有标题行
		mockCmd := "ps -p 999999 -o pid,ppid,user,stat,pcpu,pmem,comm,args"
		mockOutput := `  PID  PPID USER     STAT  %CPU %MEM COMMAND         COMMAND`

		mockRunner.On("RunShell", ctx, mockCmd).
			Return([]byte(mockOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"pid":999999,"info_type":"basic"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "找不到PID为 999999 的进程")
		mockRunner.AssertExpectations(t)
	})
}
