package humanintool

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/m4n5ter/another-me/pkg/i18n"
)

// 创建测试用的翻译文件系统
func createTestFS(t *testing.T) fs.FS {
	t.Helper()

	testFS := fstest.MapFS{
		"en.json": &fstest.MapFile{
			Data: []byte(`{
				"tool.human_in.name": "Human Intervention",
				"tool.human_in.description": "Request human intervention to get input, confirmation, or decisions from a human operator.",
				"tool.human_in.arg.message": "Message to display to the human operator",
				"tool.human_in.arg.timeout": "Timeout in seconds (default: 300 seconds)",
				"tool.human_in.arg.request_type": "Type of request: input, confirmation, or decision",
				"tool.human_in.arg.options": "Options for decision-type requests"
			}`),
			Mode: 0o644,
		},
		"zh.json": &fstest.MapFile{
			Data: []byte(`{
				"tool.human_in.name": "人类介入",
				"tool.human_in.description": "请求人类介入，获取人类操作员的输入、确认或决策。",
				"tool.human_in.arg.message": "显示给人类操作员的消息",
				"tool.human_in.arg.timeout": "超时时间（秒，默认：300秒）",
				"tool.human_in.arg.request_type": "请求类型：input（输入）、confirmation（确认）或decision（决策）",
				"tool.human_in.arg.options": "决策类型请求的选项"
			}`),
			Mode: 0o644,
		},
	}

	return testFS
}

func TestHumanInTool_Schema(t *testing.T) {
	// 创建测试用的国际化管理器
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err)

	// 创建工具实例
	tool := NewHumanInTool(i18nMgr)

	// 获取Schema
	ctx := context.Background()
	schema, err := tool.Schema(ctx)
	require.NoError(t, err)

	// 验证基本信息
	assert.Equal(t, "human_in", schema.Name)
	assert.NotEmpty(t, schema.Descriptions)
	assert.NotEmpty(t, schema.LocalizedNames)

	// 验证输入参数
	assert.Len(t, schema.InputParameters, 4)

	paramNames := make(map[string]bool)
	for _, param := range schema.InputParameters {
		paramNames[param.Name] = true
	}

	assert.True(t, paramNames["message"])
	assert.True(t, paramNames["timeout"])
	assert.True(t, paramNames["request_type"])
	assert.True(t, paramNames["options"])

	// 验证输出参数
	assert.Len(t, schema.OutputParameters, 5)

	outputNames := make(map[string]bool)
	for _, param := range schema.OutputParameters {
		outputNames[param.Name] = true
	}

	assert.True(t, outputNames["status"])
	assert.True(t, outputNames["human_response"])
	assert.True(t, outputNames["response_time"])
	assert.True(t, outputNames["request_type"])
	assert.True(t, outputNames["timed_out"])
}

func TestHumanInTool_CallWithTimeout(t *testing.T) {
	// 创建测试用的国际化管理器
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err)

	// 创建通道
	commChan := make(chan HumanResponse, 1)
	pendingReq := make(chan HumanRequest, 1)

	// 创建工具实例
	tool := NewHumanInToolWithChannels(i18nMgr, commChan, pendingReq)

	// 准备输入参数（设置短超时）
	inputJSON := `{
		"message": "测试消息",
		"timeout": 1,
		"request_type": "input"
	}`

	// 调用工具（应该超时）
	ctx := context.Background()
	result, err := tool.Call(ctx, inputJSON)
	require.NoError(t, err)

	// 验证结果包含超时状态
	assert.Contains(t, result, `"status":"timeout"`)
	assert.Contains(t, result, `"timed_out":true`)
}

func TestHumanInTool_CallWithResponse(t *testing.T) {
	// 创建测试用的国际化管理器
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err)

	// 创建通道
	commChan := make(chan HumanResponse, 1)
	pendingReq := make(chan HumanRequest, 1)

	// 创建工具实例
	tool := NewHumanInToolWithChannels(i18nMgr, commChan, pendingReq)

	// 准备输入参数
	inputJSON := `{
		"message": "请输入你的名字",
		"timeout": 30,
		"request_type": "input"
	}`

	// 在另一个goroutine中模拟客户端响应
	go func() {
		// 等待请求
		select {
		case req := <-pendingReq:
			// 模拟人类响应
			response := HumanResponse{
				ID:           req.ID,
				Response:     "张三",
				Status:       "success",
				ResponseTime: 5,
			}
			// 稍微延迟后发送响应
			time.Sleep(100 * time.Millisecond)
			commChan <- response
		case <-time.After(5 * time.Second):
			t.Error("未收到预期的请求")
		}
	}()

	// 调用工具
	ctx := context.Background()
	result, err := tool.Call(ctx, inputJSON)
	require.NoError(t, err)

	// 验证结果
	assert.Contains(t, result, `"status":"success"`)
	assert.Contains(t, result, `"human_response":"张三"`)
	assert.Contains(t, result, `"timed_out":false`)
}

func TestHumanInTool_CallWithInvalidInput(t *testing.T) {
	// 创建测试用的国际化管理器
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err)

	// 创建工具实例
	tool := NewHumanInTool(i18nMgr)

	// 测试无效JSON
	ctx := context.Background()
	_, err = tool.Call(ctx, "invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的 JSON 输入")

	// 测试空消息
	_, err = tool.Call(ctx, `{"message": ""}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message 参数不能为空")

	// 测试无效请求类型
	_, err = tool.Call(ctx, `{"message": "test", "request_type": "invalid"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的请求类型")

	// 测试决策类型缺少选项
	_, err = tool.Call(ctx, `{"message": "choose", "request_type": "decision"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "决策类型请求必须提供选项")
}

func TestHumanInTool_DefaultValues(t *testing.T) {
	// 创建测试用的国际化管理器
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err)

	// 创建通道
	commChan := make(chan HumanResponse, 1)
	pendingReq := make(chan HumanRequest, 1)

	// 创建工具实例
	tool := NewHumanInToolWithChannels(i18nMgr, commChan, pendingReq)

	// 准备最小输入参数（测试默认值）
	inputJSON := `{"message": "测试默认值"}`

	// 在另一个goroutine中检查请求的默认值
	go func() {
		select {
		case req := <-pendingReq:
			// 验证默认值
			assert.Equal(t, 300, req.Timeout)         // 默认超时
			assert.Equal(t, "input", req.RequestType) // 默认请求类型

			// 发送响应以避免工具超时
			response := HumanResponse{
				ID:       req.ID,
				Response: "ok",
				Status:   "success",
			}
			time.Sleep(50 * time.Millisecond)
			commChan <- response
		case <-time.After(2 * time.Second):
			t.Error("未收到预期的请求")
		}
	}()

	// 调用工具
	ctx := context.Background()
	_, err = tool.Call(ctx, inputJSON)
	require.NoError(t, err)
}
