package browsertool

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	json "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// MockLogger 用于测试的日志记录器
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Error(msg string, args ...any) {
	m.Called(msg, args)
}

func (m *MockLogger) Info(msg string, args ...any) {
	m.Called(msg, args)
}

// TestBrowserConfig 测试浏览器配置
func TestBrowserConfig(t *testing.T) {
	// 测试创建默认配置
	config := NewBrowserConfig()
	assert.NotNil(t, config)
	assert.True(t, config.Headless)
	assert.Equal(t, 60, config.Timeout)
	assert.NotEmpty(t, config.UserAgent)

	// 测试配置检查
	tempDir := t.TempDir()
	config.DataPath = tempDir
	err := config.Check()
	assert.NoError(t, err)

	// 测试配置检查创建目录
	browserDir := filepath.Join(tempDir, config.BrowserDataPath)
	_, err = os.Stat(browserDir)
	assert.NoError(t, err)
	assert.True(t, filepath.IsAbs(browserDir))

	// 测试提示加载
	assert.NotEmpty(t, config.GetPrompt(), "提示内容不应为空")

	// 测试使用自定义提示文件
	promptFile := filepath.Join(tempDir, "custom_prompt.txt")
	customPrompt := "Custom browser prompt"
	err = os.WriteFile(promptFile, []byte(customPrompt), 0o644)
	assert.NoError(t, err)

	config.PromptFile = promptFile
	err = config.Check()
	assert.NoError(t, err)
	assert.Equal(t, customPrompt, config.GetPrompt())

	// 测试提示文件不存在的情况
	config.PromptFile = filepath.Join(tempDir, "nonexistent.txt")
	err = config.Check()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "提示文件不存在")
}

// TestBrowserTool_Schema 测试Schema函数
func TestBrowserTool_Schema(t *testing.T) {
	// 使用真实的i18n管理器（在实际项目中可能需要mock）
	tool := NewBrowserTool(i18n.GlobalManager)
	ctx := context.Background()

	// 获取模式
	schema, err := tool.Schema(ctx)

	// 验证结果
	assert.NoError(t, err)
	assert.Equal(t, "browser", schema.Name)
	assert.NotEmpty(t, schema.InputParameters)
	assert.NotEmpty(t, schema.OutputParameters)

	// 验证必要的参数存在
	foundOperation := false
	foundURL := false
	foundSelector := false
	foundValue := false
	foundScript := false
	foundDebug := false
	foundOptions := false

	for _, param := range schema.InputParameters {
		switch param.Name {
		case "operation":
			foundOperation = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
			assert.True(t, param.Required)
		case "url":
			foundURL = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
		case "selector":
			foundSelector = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
		case "value":
			foundValue = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
		case "script":
			foundScript = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
		case "debug":
			foundDebug = true
			assert.Equal(t, toolcore.ParamTypeObject, param.Type)
		case "options":
			foundOptions = true
			assert.Equal(t, toolcore.ParamTypeObject, param.Type)
		}
	}

	assert.True(t, foundOperation, "Schema should include 'operation' parameter")
	assert.True(t, foundURL, "Schema should include 'url' parameter")
	assert.True(t, foundSelector, "Schema should include 'selector' parameter")
	assert.True(t, foundValue, "Schema should include 'value' parameter")
	assert.True(t, foundScript, "Schema should include 'script' parameter")
	assert.True(t, foundDebug, "Schema should include 'debug' parameter")
	assert.True(t, foundOptions, "Schema should include 'options' parameter")

	// 验证输出参数
	successFound := false
	messageFound := false
	screenshotFound := false
	valueFound := false
	debugInfoFound := false

	for _, param := range schema.OutputParameters {
		switch param.Name {
		case "success":
			successFound = true
			assert.Equal(t, toolcore.ParamTypeBoolean, param.Type)
		case "message":
			messageFound = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
		case "screenshot":
			screenshotFound = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
		case "value":
			valueFound = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
		case "debug_info":
			debugInfoFound = true
			assert.Equal(t, toolcore.ParamTypeString, param.Type)
		}
	}

	assert.True(t, successFound, "Schema should include 'success' output parameter")
	assert.True(t, messageFound, "Schema should include 'message' output parameter")
	assert.True(t, screenshotFound, "Schema should include 'screenshot' output parameter")
	assert.True(t, valueFound, "Schema should include 'value' output parameter")
	assert.True(t, debugInfoFound, "Schema should include 'debug_info' output parameter")

	// 检查Schema的多语言支持
	assert.NotEmpty(t, schema.Descriptions)
	assert.NotEmpty(t, schema.LocalizedNames)
}

// TestBrowserTool_Call_InvalidInput 测试Call函数处理无效输入
func TestBrowserTool_Call_InvalidInput(t *testing.T) {
	// 创建工具实例
	tool := NewBrowserTool(i18n.GlobalManager)
	ctx := context.Background()

	// 测试无效的JSON输入
	_, err := tool.Call(ctx, "{invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "无效的 JSON 输入")

	// 测试缺少必需参数的情况
	_, err = tool.Call(ctx, `{}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的操作类型")

	// 测试不支持的操作类型
	_, err = tool.Call(ctx, `{"operation":"invalid_operation"}`)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的操作类型")
}

// TestBrowserArgs 测试BrowserArgs结构体的JSON解析
func TestBrowserArgs(t *testing.T) {
	// 测试基本操作
	jsonData := `{
		"operation": "navigate",
		"url": "https://example.com",
		"selector": "#element",
		"value": "test value",
		"script": "console.log('test')",
		"options": {"option1": "value1"}
	}`

	var args BrowserArgs
	err := json.Unmarshal([]byte(jsonData), &args)
	assert.NoError(t, err)
	assert.Equal(t, "navigate", args.Operation)
	assert.Equal(t, "https://example.com", args.URL)
	assert.Equal(t, "#element", args.Selector)
	assert.Equal(t, "test value", args.Value)
	assert.Equal(t, "console.log('test')", args.Script)
	assert.NotNil(t, args.Options)
	assert.Equal(t, "value1", args.Options["option1"])

	// 测试调试操作
	jsonData = `{
		"operation": "debug",
		"debug": {
			"action": "enable_disable",
			"enable": true,
			"url": "https://example.com/script.js",
			"line_number": 10,
			"column_number": 5,
			"condition": "x > 10",
			"breakpoint_id": "123456"
		}
	}`

	err = json.Unmarshal([]byte(jsonData), &args)
	assert.NoError(t, err)
	assert.Equal(t, "debug", args.Operation)
	assert.NotNil(t, args.Debug)
	assert.Equal(t, "enable_disable", args.Debug.Action)
	assert.True(t, args.Debug.Enable)
	assert.Equal(t, "https://example.com/script.js", args.Debug.URL)
	assert.Equal(t, 10, args.Debug.LineNumber)
	assert.Equal(t, 5, args.Debug.ColumnNumber)
	assert.Equal(t, "x > 10", args.Debug.Condition)
	assert.Equal(t, "123456", args.Debug.BreakpointID)
}

// TestBrowserResult 测试BrowserResult结构体的JSON生成
func TestBrowserResult(t *testing.T) {
	// 测试基本结果
	result := BrowserResult{
		Operation: OperationNavigate,
		Success:   true,
		Message:   "成功导航到https://example.com",
	}

	jsonData, err := json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), `"operation":"navigate"`)
	assert.Contains(t, string(jsonData), `"success":true`)
	assert.Contains(t, string(jsonData), `"message":"成功导航到https://example.com"`)

	// 测试带截图的结果
	result = BrowserResult{
		Operation:  OperationScreenshot,
		Success:    true,
		Message:    "截图完成",
		Screenshot: "data:image/png;base64,abc123",
	}

	jsonData, err = json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), `"operation":"screenshot"`)
	assert.Contains(t, string(jsonData), `"screenshot":"data:image/png;base64,abc123"`)

	// 测试带有脚本执行结果的结果
	result = BrowserResult{
		Operation: OperationEvaluate,
		Success:   true,
		Message:   "JavaScript执行成功",
		Value:     `{"result":42}`,
	}

	jsonData, err = json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), `"operation":"evaluate"`)
	assert.Contains(t, string(jsonData), `"value":"{\"result\":42}"`)

	// 测试带有调试信息的结果
	result = BrowserResult{
		Operation: OperationDebug,
		Success:   true,
		Message:   "成功设置断点",
		DebugInfo: `{"breakpoint_id":"123456"}`,
	}

	jsonData, err = json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(jsonData), `"operation":"debug"`)
	assert.Contains(t, string(jsonData), `"debug_info":"{\"breakpoint_id\":\"123456\"}"`)
}

// TestNewBrowserTools 测试工具注册函数
func TestNewBrowserTools(t *testing.T) {
	tools := NewBrowserTools(i18n.GlobalManager)
	assert.NotEmpty(t, tools)
	assert.Len(t, tools, 1)

	// 验证返回的工具是否为BrowserTool
	tool, ok := tools[0].(*BrowserTool)
	assert.True(t, ok)
	assert.NotNil(t, tool)
}

// TestBrowserTool_Close 测试浏览器关闭功能
func TestBrowserTool_Close(t *testing.T) {
	tool := NewBrowserTool(i18n.GlobalManager)

	// 测试在浏览器未初始化时关闭
	err := tool.Close()
	assert.NoError(t, err)

	// 注意：测试已初始化浏览器的关闭需要实际浏览器环境
	// 如果要在单元测试中覆盖，需要修改代码以便于注入mock
}

// TestBase64Encoding 测试Base64编码功能（辅助测试）
func TestBase64Encoding(t *testing.T) {
	// 创建一个简单的测试文件
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := []byte("Hello, Browser Tool!")
	err := os.WriteFile(testFile, testContent, 0o644)
	assert.NoError(t, err)

	// 读取并编码
	data, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	encoded := base64.StdEncoding.EncodeToString(data)
	assert.NotEmpty(t, encoded)

	// 解码并验证
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	assert.NoError(t, err)
	assert.Equal(t, testContent, decoded)
}

// TestDebugArgsValidation 测试调试参数验证
func TestDebugArgsValidation(t *testing.T) {
	// 创建一个测试工具
	tool := NewBrowserTool(i18n.GlobalManager)

	// 测试缺少调试参数的情况
	args := BrowserArgs{
		Operation: OperationDebug,
	}
	_, err := tool.debug(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "调试参数不能为空")

	// 测试不支持的调试操作类型
	args.Debug = &DebugArgs{
		Action: "unsupported_action",
	}
	_, err = tool.debug(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "不支持的调试操作类型")

	// 测试设置断点缺少必要参数的情况
	args.Debug.Action = DebugSetBreakpoint
	_, err = tool.debugSetBreakpoint(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "断点URL和行号不能为空")
}

// TestWithCustomConfig 测试使用自定义配置
func TestWithCustomConfig(t *testing.T) {
	// 创建自定义配置
	config := NewBrowserConfig()
	config.Headless = false
	config.Timeout = 120
	config.WindowWidth = 1280
	config.WindowHeight = 720

	// 使用自定义配置创建工具
	tool := NewBrowserToolWithConfig(i18n.GlobalManager, config)
	assert.NotNil(t, tool)

	// 验证配置是否正确设置
	assert.Equal(t, config, tool.config)
	assert.False(t, tool.config.Headless)
	assert.Equal(t, 120, tool.config.Timeout)
	assert.Equal(t, 1280, tool.config.WindowWidth)
	assert.Equal(t, 720, tool.config.WindowHeight)
}

// 测试操作参数验证
func TestOperationArgsValidation(t *testing.T) {
	// 创建测试工具
	tool := NewBrowserTool(i18n.GlobalManager)

	// 测试导航操作缺少URL
	args := BrowserArgs{
		Operation: OperationNavigate,
	}
	_, err := tool.navigate(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "URL不能为空")

	// 测试点击操作缺少选择器
	args = BrowserArgs{
		Operation: OperationClick,
	}
	_, err = tool.click(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "选择器不能为空")

	// 测试填充表单缺少选择器
	args = BrowserArgs{
		Operation: OperationFill,
	}
	_, err = tool.fill(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "选择器不能为空")

	// 测试选择下拉框缺少选择器
	args = BrowserArgs{
		Operation: OperationSelect,
	}
	_, err = tool.selectOption(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "选择器不能为空")

	// 测试悬停操作缺少选择器
	args = BrowserArgs{
		Operation: OperationHover,
	}
	_, err = tool.hover(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "选择器不能为空")

	// 测试执行JavaScript缺少脚本
	args = BrowserArgs{
		Operation: OperationEvaluate,
	}
	_, err = tool.evaluate(args)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "脚本不能为空")
}
