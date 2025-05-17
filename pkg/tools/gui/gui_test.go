package gui

import (
	"context"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 创建测试用的翻译文件系统
func createTestFS(t *testing.T) fs.FS {
	t.Helper()

	testFS := fstest.MapFS{
		"en.json": &fstest.MapFile{
			Data: []byte(`{
				"tool.gui.screenshot.description": "Capture a screenshot of the default desktop.",
				"tool.gui.move_mouse.description": "Move the mouse to the specified coordinates.",
				"tool.gui.move_mouse.arg.x": "x coordinate",
				"tool.gui.move_mouse.arg.y": "y coordinate",
				"tool.gui.mouse_location.description": "Get the current mouse location."
			}`),
			Mode: 0644,
		},
		"zh.json": &fstest.MapFile{
			Data: []byte(`{
				"tool.gui.screenshot.description": "捕获一张默认桌面的截图。",
				"tool.gui.move_mouse.description": "移动鼠标到指定坐标。",
				"tool.gui.move_mouse.arg.x": "x坐标",
				"tool.gui.move_mouse.arg.y": "y坐标",
				"tool.gui.mouse_location.description": "获取鼠标当前位置。"
			}`),
			Mode: 0644,
		},
	}

	return testFS
}

// TestGUITool_Creation 测试 GUITool 的创建
func TestGUITool_Creation(t *testing.T) {
	// 创建测试环境
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err, "创建 i18n.Manager 应成功")

	// 创建 GUI 工具
	tools := NewGUITools(i18nMgr)
	require.NotEmpty(t, tools, "应创建非空的工具集合")

	// 验证各种 GUI 工具的创建
	toolNames := map[string]bool{}
	for _, tool := range tools {
		schema, err := tool.Schema(context.Background())
		require.NoError(t, err, "获取工具 Schema 不应返回错误")
		toolNames[schema.Name] = true
	}

	// 验证是否创建了几个基本工具
	assert.True(t, toolNames["screenshot"], "应包含 screenshot 工具")
	assert.True(t, toolNames["move_mouse"], "应包含 move_mouse 工具")
	assert.True(t, toolNames["mouse_location"], "应包含 mouse_location 工具")
}

// TestGUITool_Schema 测试 GUITool 的 Schema 方法
func TestGUITool_Schema(t *testing.T) {
	// 创建测试环境
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err, "创建 i18n.Manager 应成功")

	// 创建 GUITool 实例
	screenshotTool := NewGUIToolWithName(i18nMgr, "screenshot")
	require.NotNil(t, screenshotTool, "创建 GUITool 不应返回 nil")

	// 获取 Schema
	ctx := context.Background()
	schema, err := screenshotTool.Schema(ctx)
	require.NoError(t, err, "获取工具 Schema 不应返回错误")

	// 验证 Schema 内容
	assert.Equal(t, "screenshot", schema.Name, "工具名称应为 'screenshot'")
	assert.NotEmpty(t, schema.Descriptions["en"], "英文描述不应为空")
	assert.NotEmpty(t, schema.Descriptions["zh"], "中文描述不应为空")
	assert.Equal(t, "Capture a screenshot of the default desktop.", schema.Descriptions["en"], "英文描述应正确")

	// 测试另一个工具的 Schema
	moveMouseTool := NewGUIToolWithName(i18nMgr, "move_mouse")
	schema2, err := moveMouseTool.Schema(ctx)
	require.NoError(t, err, "获取工具 Schema 不应返回错误")

	// 验证 Schema 内容
	assert.Equal(t, "move_mouse", schema2.Name, "工具名称应为 'move_mouse'")
	assert.Equal(t, 2, len(schema2.InputParameters), "move_mouse 应有 2 个输入参数")

	// 验证参数
	var xParam, yParam *toolcore.ParameterDefinition
	for _, param := range schema2.InputParameters {
		if param.Name == "x" {
			xParam = &param
		} else if param.Name == "y" {
			yParam = &param
		}
	}

	require.NotNil(t, xParam, "应存在名为 'x' 的参数")
	require.NotNil(t, yParam, "应存在名为 'y' 的参数")
	assert.True(t, xParam.Required, "x 参数应为必需")
	assert.True(t, yParam.Required, "y 参数应为必需")
	assert.Equal(t, "x coordinate", xParam.Description["en"], "x 参数的英文描述应正确")
	assert.Equal(t, "x坐标", xParam.Description["zh"], "x 参数的中文描述应正确")
}

// TestGUITool_Call_BadInput 测试传入无效输入时 Call 方法的行为
func TestGUITool_Call_BadInput(t *testing.T) {
	// 跳过此测试，因为它会尝试调用robotgo，在没有显示器的环境中会崩溃
	t.Skip("在无显示器环境下跳过 GUI 工具的调用测试")

	// 以下代码在实际有显示器的环境中可以取消注释测试
	/*
		// 创建测试环境
		testFS := createTestFS(t)
		i18nMgr, err := i18n.NewManager(testFS, "en")
		require.NoError(t, err, "创建 i18n.Manager 应成功")

		// 创建 GUITool 实例
		moveMouseTool := NewGUIToolWithName(i18nMgr, "move_mouse")
		ctx := context.Background()

		// 测试无效 JSON
		_, err = moveMouseTool.Call(ctx, "{not valid json")
		assert.Error(t, err, "无效 JSON 输入应返回错误")

		// 测试参数缺失
		_, err = moveMouseTool.Call(ctx, `{"x": 100}`)
		assert.Error(t, err, "缺少必需参数应返回错误")

		// 测试参数类型错误
		_, err = moveMouseTool.Call(ctx, `{"x": "not a number", "y": 100}`)
		assert.Error(t, err, "参数类型错误应返回错误")
	*/
}
