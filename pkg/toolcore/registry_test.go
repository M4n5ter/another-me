package toolcore

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewRegistry 测试 NewRegistry 函数
func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.NotNil(t, registry, "NewRegistry 返回的实例不应为 nil")
	assert.NotNil(t, registry.tools, "NewRegistry 应初始化 tools map")
	assert.Empty(t, registry.tools, "新创建的 Registry 应是空的")
}

// TestRegistry_Register 测试 Register 方法
func TestRegistry_Register(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	// 测试成功注册
	tool1 := NewMockTool("tool1", nil)
	err := registry.Register(ctx, tool1)
	assert.NoError(t, err, "Register 应成功注册一个有效的工具")

	// 测试检索刚注册的工具
	retrievedTool, exists := registry.Get("tool1")
	assert.True(t, exists, "应能找到已注册的工具")
	assert.Equal(t, tool1, retrievedTool, "检索的工具实例应与注册时相同")

	// 测试注册具有相同名称的工具
	tool1Duplicate := NewMockTool("tool1", nil)
	err = registry.Register(ctx, tool1Duplicate)
	assert.Error(t, err, "注册重名工具应返回错误")
	assert.Contains(t, err.Error(), "已存在", "错误消息应指出工具已存在")

	// 测试注册一个 Schema() 返回错误的工具
	errorTool := NewMockTool("error_tool", nil)
	errorTool.ForceSchemaError = errors.New("schema error")
	err = registry.Register(ctx, errorTool)
	assert.Error(t, err, "注册有问题的工具应返回错误")
	assert.Contains(t, err.Error(), "schema error", "错误消息应包含原始错误信息")

	// 测试注册一个 Schema.Name 为空的工具
	emptyNameTool := NewMockTool("", nil)
	err = registry.Register(ctx, emptyNameTool)
	assert.Error(t, err, "注册名称为空的工具应返回错误")
	assert.Contains(t, err.Error(), "Name 字段不能为空", "错误消息应指出名称不能为空")
}

// TestRegistry_Get 测试 Get 方法
func TestRegistry_Get(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	// 注册两个工具
	tool1 := NewMockTool("tool1", nil)
	tool2 := NewMockTool("tool2", nil)
	require.NoError(t, registry.Register(ctx, tool1), "注册 tool1 应成功")
	require.NoError(t, registry.Register(ctx, tool2), "注册 tool2 应成功")

	// 测试查找已注册的工具
	foundTool, exists := registry.Get("tool1")
	assert.True(t, exists, "已注册的工具应能被找到")
	assert.Equal(t, tool1, foundTool, "找到的工具应与注册的相同")

	foundTool, exists = registry.Get("tool2")
	assert.True(t, exists, "已注册的工具应能被找到")
	assert.Equal(t, tool2, foundTool, "找到的工具应与注册的相同")

	// 测试查找未注册的工具
	foundTool, exists = registry.Get("non_existent_tool")
	assert.False(t, exists, "未注册的工具应返回 exists=false")
	assert.Nil(t, foundTool, "未找到的工具应返回 nil")
}

// TestRegistry_ListSchemas 测试 ListSchemas 方法
func TestRegistry_ListSchemas(t *testing.T) {
	ctx := context.Background()
	registry := NewRegistry()

	// 注册两个正常工具
	tool1 := NewMockTool("tool1", nil)
	tool2 := NewMockTool("tool2", map[string]string{"en": "Tool 2 desc", "zh": "工具2描述"})
	require.NoError(t, registry.Register(ctx, tool1), "注册 tool1 应成功")
	require.NoError(t, registry.Register(ctx, tool2), "注册 tool2 应成功")

	// 测试空 Registry 的 ListSchemas
	emptyRegistry := NewRegistry()
	schemas, err := emptyRegistry.ListSchemas(ctx)
	assert.NoError(t, err, "空 Registry 的 ListSchemas 不应返回错误")
	assert.Empty(t, schemas, "空 Registry 应返回空的 schemas 切片")

	// 测试正常 Registry 的 ListSchemas
	schemas, err = registry.ListSchemas(ctx)
	assert.NoError(t, err, "正常 Registry 的 ListSchemas 不应返回错误")
	assert.Len(t, schemas, 2, "应返回 2 个 schema")

	// 验证返回的 schemas 内容
	var tool1Schema, tool2Schema ToolSchema
	for _, schema := range schemas {
		switch schema.Name {
		case "tool1":
			tool1Schema = schema
		case "tool2":
			tool2Schema = schema
		}
	}

	assert.Equal(t, "tool1", tool1Schema.Name, "应返回 tool1 的 schema")
	assert.Equal(t, "tool2", tool2Schema.Name, "应返回 tool2 的 schema")
	assert.Equal(t, "Tool 2 desc", tool2Schema.Descriptions["en"], "工具2的英文描述应正确")

	// 测试在 ListSchemas 过程中出现的错误
	errorRegistry := NewRegistry()
	normalTool := NewMockTool("normal_tool", nil)
	require.NoError(t, errorRegistry.Register(ctx, normalTool), "注册正常工具应成功")

	// 注册后修改工具对象以在 Schema() 调用时返回错误
	// 这模拟了工具在注册后某些状态变化导致 Schema() 开始失败的情况
	normalTool.ForceSchemaError = errors.New("schema error")

	schemas, err = errorRegistry.ListSchemas(ctx)
	assert.Error(t, err, "当有工具的 Schema() 返回错误时，ListSchemas 应返回错误")
	assert.Contains(t, err.Error(), "schema error", "错误消息应包含原始错误信息")
	assert.Empty(t, schemas, "由于错误发生在第一个也是唯一的工具上，应返回空的结果")
}

// TestRegistry_Concurrency 测试 Registry 在并发环境下的行为
func TestRegistry_Concurrency(t *testing.T) {
	// 注：这不是一个完整的并发测试，只是基本验证 Registry 的互斥锁机制能否工作
	// 对于更复杂的并发测试，可能需要使用 -race 检测器或更精细的并发测试工具

	ctx := context.Background()
	registry := NewRegistry()

	// 并发注册多个工具
	done := make(chan bool)
	for i := range 10 {
		go func(idx int) {
			toolName := fmt.Sprintf("tool_%d", idx)
			tool := NewMockTool(toolName, nil)
			_ = registry.Register(ctx, tool)
			done <- true
		}(i)
	}

	// 等待所有 goroutines 完成
	for range 10 {
		<-done
	}

	// 验证注册结果
	schemas, err := registry.ListSchemas(ctx)
	assert.NoError(t, err, "即使在并发环境下，ListSchemas 也不应返回错误")
	assert.True(t, len(schemas) > 0, "应至少注册了一些工具")

	// 并发获取工具
	for range 10 {
		go func() {
			for j := range 10 {
				toolName := fmt.Sprintf("tool_%d", j)
				_, _ = registry.Get(toolName)
			}
			done <- true
		}()
	}

	// 等待所有 goroutines 完成
	for range 10 {
		<-done
	}
}
