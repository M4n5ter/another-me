package fetchtool

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
				"tool.fetch.description": "Fetches content from a given URL.",
				"tool.fetch.arg.url": "The URL to fetch content from."
			}`),
			Mode: 0644,
		},
		"zh.json": &fstest.MapFile{
			Data: []byte(`{
				"tool.fetch.description": "从给定的 URL 获取内容。",
				"tool.fetch.arg.url": "要从中获取内容的 URL。"
			}`),
			Mode: 0644,
		},
	}

	return testFS
}

// TestFetchTool_Schema 测试 FetchTool 的 Schema 方法
func TestFetchTool_Schema(t *testing.T) {
	// 创建测试环境
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err, "创建 i18n.Manager 应成功")

	// 创建 FetchTool 实例
	fetchTool := NewFetchTool(i18nMgr)
	require.NotNil(t, fetchTool, "创建 FetchTool 不应返回 nil")

	// 获取 Schema
	ctx := context.Background()
	schema, err := fetchTool.Schema(ctx)
	require.NoError(t, err, "获取工具 Schema 不应返回错误")

	// 验证 Schema 内容
	assert.Equal(t, "fetch", schema.Name, "工具名称应为 'fetch'")
	assert.NotEmpty(t, schema.Descriptions["en"], "英文描述不应为空")
	assert.NotEmpty(t, schema.Descriptions["zh"], "中文描述不应为空")
	assert.GreaterOrEqual(t, len(schema.InputParameters), 1, "至少应有一个输入参数")

	// 验证 URL 参数
	var urlParam *toolcore.ParameterDefinition
	for _, param := range schema.InputParameters {
		if param.Name == "url" {
			urlParam = &param
			break
		}
	}
	require.NotNil(t, urlParam, "应存在名为 'url' 的参数")
	assert.True(t, urlParam.Required, "URL 参数应为必需")
}

// TestFetchTool_Call_BadInput 测试传入无效输入时 Call 方法的行为
func TestFetchTool_Call_BadInput(t *testing.T) {
	// 创建测试环境
	testFS := createTestFS(t)
	i18nMgr, err := i18n.NewManager(testFS, "en")
	require.NoError(t, err, "创建 i18n.Manager 应成功")

	// 创建 FetchTool 实例
	fetchTool := NewFetchTool(i18nMgr)
	ctx := context.Background()

	// 测试空 JSON
	_, err = fetchTool.Call(ctx, "")
	assert.Error(t, err, "空 JSON 输入应返回错误")

	// 测试无效 JSON
	_, err = fetchTool.Call(ctx, "{not valid json")
	assert.Error(t, err, "无效 JSON 输入应返回错误")

	// 测试缺少 URL 参数
	_, err = fetchTool.Call(ctx, `{"max_length": 100}`)
	assert.Error(t, err, "缺少 URL 参数应返回错误")

	// 测试无效 URL
	_, err = fetchTool.Call(ctx, `{"url": "not-a-url"}`)
	assert.Error(t, err, "无效 URL 应返回错误")
} 