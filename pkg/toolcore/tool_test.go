package toolcore

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParameterTypes 测试参数类型常量定义是否正确
func TestParameterTypes(t *testing.T) {
	assert.Equal(t, ParameterType("string"), ParamTypeString, "ParamTypeString 应为 'string'")
	assert.Equal(t, ParameterType("number"), ParamTypeNumber, "ParamTypeNumber 应为 'number'")
	assert.Equal(t, ParameterType("integer"), ParamTypeInteger, "ParamTypeInteger 应为 'integer'")
	assert.Equal(t, ParameterType("boolean"), ParamTypeBoolean, "ParamTypeBoolean 应为 'boolean'")
	assert.Equal(t, ParameterType("object"), ParamTypeObject, "ParamTypeObject 应为 'object'")
	assert.Equal(t, ParameterType("array"), ParamTypeArray, "ParamTypeArray 应为 'array'")
	assert.Equal(t, ParameterType("null"), ParamTypeNull, "ParamTypeNull 应为 'null'")
	assert.Equal(t, ParameterType("any"), ParamTypeAny, "ParamTypeAny 应为 'any'")
}

// TestParameterDefinition 测试 ParameterDefinition 结构体
func TestParameterDefinition(t *testing.T) {
	// 测试基本字符串参数
	strParam := ParameterDefinition{
		Name:        "test_param",
		Type:        ParamTypeString,
		Description: map[string]string{"en": "Test parameter", "zh": "测试参数"},
		Required:    true,
	}

	assert.Equal(t, "test_param", strParam.Name, "参数名称应为 'test_param'")
	assert.Equal(t, ParamTypeString, strParam.Type, "参数类型应为 string")
	assert.Equal(t, "Test parameter", strParam.Description["en"], "英文描述应正确")
	assert.Equal(t, "测试参数", strParam.Description["zh"], "中文描述应正确")
	assert.True(t, strParam.Required, "参数应标记为必需")

	// 测试带枚举值的参数
	enumParam := ParameterDefinition{
		Name:        "color",
		Type:        ParamTypeString,
		Description: map[string]string{"en": "Color choice", "zh": "颜色选择"},
		Required:    false,
		EnumValues:  []any{"red", "green", "blue"},
	}

	assert.Equal(t, 3, len(enumParam.EnumValues), "枚举值数量应为 3")
	assert.Equal(t, "red", enumParam.EnumValues[0], "第一个枚举值应为 'red'")

	// 测试数组参数
	arrayParam := ParameterDefinition{
		Name:        "items",
		Type:        ParamTypeArray,
		Description: map[string]string{"en": "List of items", "zh": "物品列表"},
		Required:    true,
		Items: &ParameterDefinition{
			Type:        ParamTypeString,
			Description: map[string]string{"en": "Item name", "zh": "物品名称"},
		},
	}

	assert.NotNil(t, arrayParam.Items, "数组参数的 Items 不应为 nil")
	assert.Equal(t, ParamTypeString, arrayParam.Items.Type, "数组元素类型应为 string")

	// 测试对象参数
	objectParam := ParameterDefinition{
		Name:        "user",
		Type:        ParamTypeObject,
		Description: map[string]string{"en": "User info", "zh": "用户信息"},
		Required:    true,
		Properties: []ParameterDefinition{
			{
				Name:        "name",
				Type:        ParamTypeString,
				Description: map[string]string{"en": "User name", "zh": "用户名"},
				Required:    true,
			},
			{
				Name:        "age",
				Type:        ParamTypeInteger,
				Description: map[string]string{"en": "User age", "zh": "用户年龄"},
				Required:    false,
			},
		},
	}

	assert.Equal(t, 2, len(objectParam.Properties), "对象属性数量应为 2")
	assert.Equal(t, "name", objectParam.Properties[0].Name, "第一个属性名应为 'name'")
	assert.Equal(t, ParamTypeInteger, objectParam.Properties[1].Type, "第二个属性类型应为 integer")
}

// TestToolSchema 测试 ToolSchema 结构体
func TestToolSchema(t *testing.T) {
	schema := ToolSchema{
		Name:           "test_tool",
		LocalizedNames: map[string]string{"en": "Test Tool", "zh": "测试工具"},
		Descriptions:   map[string]string{"en": "A tool for testing", "zh": "用于测试的工具"},
		InputParameters: []ParameterDefinition{
			{
				Name:        "message",
				Type:        ParamTypeString,
				Description: map[string]string{"en": "Message to process", "zh": "要处理的消息"},
				Required:    true,
			},
		},
		OutputParameters: []ParameterDefinition{
			{
				Name:        "result",
				Type:        ParamTypeString,
				Description: map[string]string{"en": "Processing result", "zh": "处理结果"},
				Required:    true,
			},
			{
				Name:        "status",
				Type:        ParamTypeInteger,
				Description: map[string]string{"en": "Status code", "zh": "状态码"},
				Required:    true,
			},
		},
	}

	assert.Equal(t, "test_tool", schema.Name, "工具名称应为 'test_tool'")
	assert.Equal(t, "Test Tool", schema.LocalizedNames["en"], "英文本地化名称应正确")
	assert.Equal(t, "测试工具", schema.LocalizedNames["zh"], "中文本地化名称应正确")
	assert.Equal(t, "A tool for testing", schema.Descriptions["en"], "英文描述应正确")
	assert.Equal(t, 1, len(schema.InputParameters), "输入参数数量应为 1")
	assert.Equal(t, 2, len(schema.OutputParameters), "输出参数数量应为 2")
}

// TestMockTool 测试 MockTool 的基本功能
func TestMockTool(t *testing.T) {
	ctx := context.Background()
	mockTool := NewMockTool("test_tool", nil)

	// 测试 Schema 方法
	schema, err := mockTool.Schema(ctx)
	assert.NoError(t, err, "Schema 方法不应返回错误")
	assert.Equal(t, "test_tool", schema.Name, "Schema 的名称应为 'test_tool'")
	assert.Equal(t, 1, mockTool.SchemaCallCount, "Schema 方法应被调用一次")

	// 测试 Call 方法
	input := `{"message": "Hello, world!"}`
	output, err := mockTool.Call(ctx, input)
	assert.NoError(t, err, "Call 方法不应返回错误")
	assert.Equal(t, `{"result": "mock operation completed"}`, output, "输出应为预设的结果")
	assert.Equal(t, 1, mockTool.CallCallCount, "Call 方法应被调用一次")

	// 验证参数解析
	expectedInput := map[string]any{"message": "Hello, world!"}
	assert.Equal(t, expectedInput, mockTool.LastCallInput, "应正确保存解析后的输入")

	// 测试强制错误
	errorTool := NewMockTool("error_tool", nil)
	errorTool.ForceSchemaError = assert.AnError
	_, err = errorTool.Schema(ctx)
	assert.Error(t, err, "当设置 ForceSchemaError 时，Schema 方法应返回错误")

	errorTool.ForceCallError = assert.AnError
	_, err = errorTool.Call(ctx, input)
	assert.Error(t, err, "当设置 ForceCallError 时，Call 方法应返回错误")

	// 测试自定义函数
	customTool := NewMockTool("custom_tool", nil)
	customTool.SchemaFn = func(ctx context.Context) (ToolSchema, error) {
		return ToolSchema{Name: "custom_schema"}, nil
	}
	schema, err = customTool.Schema(ctx)
	assert.NoError(t, err, "自定义 SchemaFn 不应返回错误")
	assert.Equal(t, "custom_schema", schema.Name, "应使用自定义 SchemaFn 的结果")

	customTool.CallFn = func(ctx context.Context, inputJSON string) (string, error) {
		return `{"result": "custom result"}`, nil
	}
	output, err = customTool.Call(ctx, input)
	assert.NoError(t, err, "自定义 CallFn 不应返回错误")
	assert.Equal(t, `{"result": "custom result"}`, output, "应使用自定义 CallFn 的结果")
}
