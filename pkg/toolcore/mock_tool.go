package toolcore

import (
	"context"
	"encoding/json"
	"fmt"
)

// MockTool 是一个用于测试的简单 Tool 实现。
type MockTool struct {
	SchemaFn         func(ctx context.Context) (ToolSchema, error)               // 可定制的 Schema 函数
	CallFn           func(ctx context.Context, inputJSON string) (string, error) // 可定制的 Call 函数
	SchemaResult     ToolSchema                                                  // 预设的 Schema 返回值
	CallResult       string                                                      // 预设的 Call 返回值
	ForceSchemaError error                                                       // 如设置，则 Schema 方法将返回此错误
	ForceCallError   error                                                       // 如设置，则 Call 方法将返回此错误
	SchemaCallCount  int                                                         // Schema 方法被调用的次数
	CallCallCount    int                                                         // Call 方法被调用的次数
	LastCallInput    any                                                         // 最后一次调用 Call 方法的解析输入参数
}

// Schema 实现 Tool 接口的 Schema 方法。
func (m *MockTool) Schema(ctx context.Context) (ToolSchema, error) {
	m.SchemaCallCount++

	if m.ForceSchemaError != nil {
		return ToolSchema{}, m.ForceSchemaError
	}

	if m.SchemaFn != nil {
		return m.SchemaFn(ctx)
	}

	return m.SchemaResult, nil
}

// Call 实现 Tool 接口的 Call 方法。
func (m *MockTool) Call(ctx context.Context, inputJSON string) (string, error) {
	m.CallCallCount++

	// 解析输入 JSON 并保存
	if inputJSON != "" {
		var inputData any
		if err := json.Unmarshal([]byte(inputJSON), &inputData); err == nil {
			m.LastCallInput = inputData
		}
	}

	if m.ForceCallError != nil {
		return "", m.ForceCallError
	}

	if m.CallFn != nil {
		return m.CallFn(ctx, inputJSON)
	}

	return m.CallResult, nil
}

// NewMockTool 创建一个新的 MockTool 实例，提供一个默认的 Schema。
func NewMockTool(name string, descriptions map[string]string) *MockTool {
	// 如果没有提供描述，使用默认值
	if descriptions == nil {
		descriptions = map[string]string{
			"en": fmt.Sprintf("Mock tool named %s for testing", name),
			"zh": fmt.Sprintf("用于测试的模拟工具 %s", name),
		}
	}

	// 创建一个基本的 Schema
	schema := ToolSchema{
		Name:           name,
		Descriptions:   descriptions,
		LocalizedNames: map[string]string{"en": "Mock Tool", "zh": "模拟工具"},
		InputParameters: []ParameterDefinition{
			{
				Name:        "message",
				Type:        ParamTypeString,
				Description: map[string]string{"en": "Message input", "zh": "消息输入"},
				Required:    true,
			},
		},
		OutputParameters: []ParameterDefinition{
			{
				Name:        "result",
				Type:        ParamTypeString,
				Description: map[string]string{"en": "Operation result", "zh": "操作结果"},
				Required:    true,
			},
		},
	}

	return &MockTool{
		SchemaResult: schema,
		CallResult:   `{"result": "mock operation completed"}`,
	}
}

// 确认 MockTool 实现了 Tool 接口
var _ Tool = (*MockTool)(nil)
