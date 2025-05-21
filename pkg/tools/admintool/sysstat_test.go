package admintool_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/tools/admintool"
)

// TestSysstatTool_GetResourceStats 测试获取系统资源状态功能
func TestSysstatTool_GetResourceStats(t *testing.T) {
	// 准备测试环境
	mockRunner := new(MockCommandRunner)
	tool := admintool.NewSysstatToolWithRunner(i18n.GlobalManager, mockRunner)
	ctx := context.Background()

	// 测试场景1: 获取CPU状态（使用mpstat成功）
	t.Run("GetCPUStatsWithMpstat", func(t *testing.T) {
		// 模拟mpstat命令成功
		mockOutput := `Linux 5.10.0-15-amd64 (test-server) 	03/15/2023 	_x86_64_	(4 CPU)

		10:00:01 AM  CPU    %usr   %nice    %sys %iowait    %irq   %soft  %steal  %guest  %gnice   %idle
		10:00:01 AM  all    5.12    0.00    1.94    0.57    0.03    0.18    0.00    0.00    0.00   92.16`

		mockRunner.On("Run", ctx, "mpstat").
			Return([]byte(mockOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"resource":"cpu"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"resource_type":"cpu"`)
		assert.Contains(t, result, `"command":"mpstat"`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景2: 获取CPU状态（mpstat失败，使用top成功）
	t.Run("GetCPUStatsWithTop", func(t *testing.T) {
		// 模拟mpstat命令失败
		mockRunner.On("Run", ctx, "mpstat").
			Return([]byte{}, errors.New("command not found")).Once()

		// 模拟top命令成功
		topOutput := `top - 10:05:01 up  3:45,  2 users,  load average: 0.08, 0.15, 0.10
Tasks: 205 total,   1 running, 204 sleeping,   0 stopped,   0 zombie
%Cpu(s):  4.2 us,  1.8 sy,  0.0 ni, 93.7 id,  0.3 wa,  0.0 hi,  0.1 si,  0.0 st
MiB Mem :  7950.2 total,  4589.6 free,  1756.1 used,  1604.5 buff/cache
MiB Swap:  2048.0 total,  2048.0 free,     0.0 used.  5850.3 avail Mem`

		mockRunner.On("Run", ctx, "top", "-bn1").
			Return([]byte(topOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"resource":"cpu"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"resource_type":"cpu"`)
		assert.Contains(t, result, `"command":"top -bn1"`)
		assert.Contains(t, result, `%Cpu(s):  4.2 us`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景3: 获取内存状态
	t.Run("GetMemoryStats", func(t *testing.T) {
		// 模拟free命令成功
		freeOutput := `             total        used        free      shared  buff/cache   available
Mem:           7.8G        1.7G        4.5G        286M        1.6G        5.7G
Swap:          2.0G          0B        2.0G`

		mockRunner.On("Run", ctx, "free", "-h").
			Return([]byte(freeOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"resource":"memory"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"resource_type":"memory"`)
		assert.Contains(t, result, `"command":"free -h"`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景4: 获取磁盘状态
	t.Run("GetDiskStats", func(t *testing.T) {
		// 模拟df命令成功
		dfOutput := `Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        50G   15G   33G  32% /
/dev/sda2       250G   75G  163G  32% /home
tmpfs           4.0G     0  4.0G   0% /dev/shm`

		mockRunner.On("Run", ctx, "df", "-h").
			Return([]byte(dfOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"resource":"disk"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"resource_type":"disk"`)
		assert.Contains(t, result, `"command":"df -h"`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景5: 获取所有资源状态
	t.Run("GetAllResourceStats", func(t *testing.T) {
		// 模拟mpstat命令成功
		mpstatOutput := `Linux 5.10.0-15-amd64 (test-server) 	03/15/2023 	_x86_64_	(4 CPU)
10:00:01 AM  CPU    %usr   %nice    %sys %iowait    %irq   %soft  %steal  %guest  %gnice   %idle
10:00:01 AM  all    5.12    0.00    1.94    0.57    0.03    0.18    0.00    0.00    0.00   92.16`

		mockRunner.On("Run", ctx, "mpstat").
			Return([]byte(mpstatOutput), nil).Once()

		// 模拟free命令成功
		freeOutput := `             total        used        free      shared  buff/cache   available
Mem:           7.8G        1.7G        4.5G        286M        1.6G        5.7G
Swap:          2.0G          0B        2.0G`

		mockRunner.On("Run", ctx, "free", "-h").
			Return([]byte(freeOutput), nil).Once()

		// 模拟df命令成功
		dfOutput := `Filesystem      Size  Used Avail Use% Mounted on
/dev/sda1        50G   15G   33G  32% /
/dev/sda2       250G   75G  163G  32% /home
tmpfs           4.0G     0  4.0G   0% /dev/shm`

		mockRunner.On("Run", ctx, "df", "-h").
			Return([]byte(dfOutput), nil).Once()

		// 执行被测函数
		inputJSON := `{"resource":"all"}`
		result, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.NoError(t, err)
		assert.Contains(t, result, `"resource_type":"cpu"`)
		assert.Contains(t, result, `"resource_type":"memory"`)
		assert.Contains(t, result, `"resource_type":"disk"`)
		mockRunner.AssertExpectations(t)
	})

	// 测试场景6: 获取不支持的资源类型
	t.Run("GetUnsupportedResourceType", func(t *testing.T) {
		// 执行被测函数
		inputJSON := `{"resource":"network"}`
		_, err := tool.Call(ctx, inputJSON)

		// 验证结果
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "不支持的资源类型")
	})
}
