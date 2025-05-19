package toolcore

import "context"

// Tool 是任何可供 LLM 代理使用的核心工具接口。
type Tool interface {
	// Schema 返回工具的元数据 (ToolSchema)。
	// ctx 可用于传递请求范围的值，例如用于国际化描述的语言偏好。
	Schema(ctx context.Context) (ToolSchema, error)

	// Call 执行工具的核心逻辑。
	// inputJSON 是一个 JSON 字符串，代表工具的输入参数，其结构应与 Schema() 返回的 InputParameters 匹配。
	// outputJSON 是一个 JSON 字符串，代表工具执行的结果，其结构应与 Schema() 返回的 OutputParameters 匹配。
	// ctx 可用于取消、截止时间、携带请求范围的值 (如 Trace ID、语言偏好等)。
	Call(ctx context.Context, inputJSON string) (outputJSON string, err error)
}

// ToolSchema 定义了一个工具的元数据，包括其名称、国际化描述、以及输入输出参数的定义。
// 名称 (Name) 是工具的规范化、非本地化调用名称，建议使用 snake_case。
// 本地化名称 (LocalizedNames) 和描述 (Descriptions) 字段的键是语言代码。
type ToolSchema struct {
	Name             string                `json:"name"`                        // 工具的规范名称 (snake_case)，用于调用
	LocalizedNames   map[string]string     `json:"localized_names,omitempty"`   // 工具的国际化可读名称 (键: 语言代码)
	Descriptions     map[string]string     `json:"descriptions"`                // 工具的国际化详细描述 (键: 语言代码)
	InputParameters  []ParameterDefinition `json:"input_parameters,omitempty"`  // 输入参数定义列表
	OutputParameters []ParameterDefinition `json:"output_parameters,omitempty"` // 输出参数定义列表，描述 Call() 方法输出的 JSON 字符串结构
}

// ParameterDefinition 描述了一个工具的单个参数。
// 名称建议使用 snake_case 以便与 LLM API 实践保持一致。
// 描述字段 (Description) 是国际化的，键为语言代码 (例如 "en", "zh")。
type ParameterDefinition struct {
	Name        string                `json:"name"`                  // 参数名称 (snake_case)
	Type        ParameterType         `json:"type"`                  // 参数类型
	Description map[string]string     `json:"description"`           // 参数的国际化描述 (键: 语言代码)
	Required    bool                  `json:"required"`              // 此参数是否必需
	EnumValues  []any                 `json:"enum_values,omitempty"` // 可选：枚举值列表，用于具有固定允许值的参数
	Properties  []ParameterDefinition `json:"properties,omitempty"`  // 当 Type = ParamTypeObject 时，定义对象的属性
	Items       *ParameterDefinition  `json:"items,omitempty"`       // 当 Type = ParamTypeArray 时，描述数组元素的类型
}

// ParameterType 定义了工具参数的类型。
// 这些类型旨在与 JSON Schema 的基本类型以及一些常见扩展对齐。
type ParameterType string

// ParameterType 的常量定义
const (
	ParamTypeString  ParameterType = "string"  // 字符串类型
	ParamTypeNumber  ParameterType = "number"  // 数字类型 (可以是整数或浮点数)
	ParamTypeInteger ParameterType = "integer" // 整数类型 (明确指定为整数)
	ParamTypeBoolean ParameterType = "boolean" // 布尔类型
	ParamTypeObject  ParameterType = "object"  // 对象类型 (嵌套结构)
	ParamTypeArray   ParameterType = "array"   // 数组类型
	ParamTypeNull    ParameterType = "null"    // Null 类型 (用于可以为 null 的参数)
	ParamTypeAny     ParameterType = "any"     // 任意类型 (当类型真正动态或不确定时使用)
)
