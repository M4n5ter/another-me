package toolcore

import (
	"fmt"
	"slices"
	"sort"

	"github.com/mark3labs/mcp-go/mcp"
	"google.golang.org/genai"

	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/schema"
)

// ConvertMCPInputSchemaToParams 转换 MCP 的输入 schema 为 ParameterDefinition 列表
func ConvertMCPInputSchemaToParams(schema mcp.ToolInputSchema) []ParameterDefinition {
	// mcp.ToolInputSchema.Properties 是 map[string]any, 其中 'any' 通常是每个属性定义的 map[string]any
	// mcp.ToolInputSchema.Required 是 []string
	// 这个结构与 ConvertRawSchemaPropertiesToParams 直接兼容
	if schema.Properties == nil { // 确保 Properties 不为 nil 以避免 make 时的 panic
		return []ParameterDefinition{}
	}
	return ConvertRawSchemaPropertiesToParams(schema.Properties, schema.Required)
}

// ConvertJSONSchemaToParams 转换 JSON 的输入 schema 为 ParameterDefinition 列表
func ConvertJSONSchemaToParams(fullSchema map[string]any) []ParameterDefinition {
	properties, ok := fullSchema["properties"].(map[string]any)
	if !ok {
		// 如果没有属性，则是一个空对象模式，或者不是参数的模式
		// 对于空对象 {"type": "object"}，没有参数返回
		return []ParameterDefinition{}
	}
	var required []string
	if reqVal, found := fullSchema["required"]; found {
		switch reqAsserted := reqVal.(type) {
		case []any:
			for _, v := range reqAsserted {
				if str, ok := v.(string); ok {
					required = append(required, str)
				}
			}
		case []string:
			required = reqAsserted
		}
	}
	return ConvertRawSchemaPropertiesToParams(properties, required)
}

// ConvertRawSchemaPropertiesToParams 转换 JSON 模式的 "properties" 部分为 ParameterDefinition 列表
// rawSchemaProperties: JSON 模式的 "properties" 部分的 map
// requiredList: 当前级别必需的属性名称列表
func ConvertRawSchemaPropertiesToParams(rawSchemaProperties map[string]any, requiredList []string) []ParameterDefinition {
	// 先获取所有属性名并排序，确保参数顺序一致
	names := make([]string, 0, len(rawSchemaProperties))
	for name := range rawSchemaProperties {
		names = append(names, name)
	}
	sort.Strings(names)

	params := make([]ParameterDefinition, 0, len(rawSchemaProperties))
	for _, name := range names {
		propRaw := rawSchemaProperties[name]
		propSchema, ok := propRaw.(map[string]any)
		if !ok {
			continue
		}
		param := convertSinglePropertySchemaToParamDef(name, propSchema, requiredList)
		params = append(params, param)
	}
	return params
}

// convertSinglePropertySchemaToParamDef 转换单个属性的模式为 ParameterDefinition
// 递归处理嵌套对象和数组
func convertSinglePropertySchemaToParamDef(propName string, propSchema map[string]any, parentRequiredList []string) ParameterDefinition {
	param := ParameterDefinition{
		Name: propName,
		Description: map[string]string{
			"en": getStringFromMap(propSchema, "description", ""),
			"zh": getStringFromMap(propSchema, "description", ""), // 假设 en 描述为 zh 如果未指定
		},
		Required: isNameInList(propName, parentRequiredList),
	}

	typeStr := getStringFromMap(propSchema, "type", "")
	param.Type = parameterTypeFromString(typeStr)

	if enumValues, ok := propSchema["enum"]; ok {
		if enumArr, ok := enumValues.([]any); ok && len(enumArr) > 0 {
			param.EnumValues = Some(enumArr)
		}
	}

	// 处理默认值
	if defaultValue, ok := propSchema["default"]; ok {
		param.DefaultValue = Some(defaultValue)
	}

	switch param.Type {
	case ParamTypeObject:
		if nestedRawProps, ok := propSchema["properties"].(map[string]any); ok {
			var nestedRequiredList []string
			if reqVal, found := propSchema["required"]; found {
				switch reqAsserted := reqVal.(type) {
				case []any:
					for _, v := range reqAsserted {
						if str, ok := v.(string); ok {
							nestedRequiredList = append(nestedRequiredList, str)
						}
					}
				case []string:
					nestedRequiredList = reqAsserted
				}
			}
			param.Properties = Some(ConvertRawSchemaPropertiesToParams(nestedRawProps, nestedRequiredList))
		} else {
			// 处理没有特定属性定义的对象类型 (例如 "type": "object" 本身)
			// 这意味着它可以是任何对象，或者是一个空对象
			// 我们表示为 Some([]ParameterDefinition{}) 来表示它是一个对象，但没有任何 *定义* 的属性
			param.Properties = Some([]ParameterDefinition{})
		}
	case ParamTypeArray:
		if itemsSchemaRaw, ok := propSchema["items"].(map[string]any); ok {
			var itemPropsRequiredList []string
			if _, itemIsObject := itemsSchemaRaw["properties"].(map[string]any); itemIsObject {
				if reqVal, found := itemsSchemaRaw["required"]; found {
					switch reqAsserted := reqVal.(type) {
					case []any:
						for _, v := range reqAsserted {
							if str, ok := v.(string); ok {
								itemPropsRequiredList = append(itemPropsRequiredList, str)
							}
						}
					case []string:
						itemPropsRequiredList = reqAsserted
					}
				}
			}
			itemParamDef := convertSinglePropertySchemaToParamDef("item", itemsSchemaRaw, itemPropsRequiredList)
			param.Items = Some(itemParamDef)
		}
	}
	return param
}

// getStringFromMap 从 map 中安全地获取字符串属性
func getStringFromMap(prop map[string]any, key, defaultValue string) string {
	if val, ok := prop[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return defaultValue
}

// isNameInList 检查一个名称是否存在于字符串列表中
func isNameInList(name string, list []string) bool {
	return slices.Contains(list, name)
}

// parameterTypeFromString 将字符串类型转换为 ParameterType
func parameterTypeFromString(typeStr string) ParameterType {
	switch typeStr {
	case "string":
		return ParamTypeString
	case "number":
		return ParamTypeNumber
	case "integer":
		return ParamTypeInteger
	case "boolean":
		return ParamTypeBoolean
	case "object":
		return ParamTypeObject
	case "array":
		return ParamTypeArray
	case "null":
		return ParamTypeNull
	default:
		return ParamTypeAny
	}
}

// ConvertParamsToRawSchema 将 ParameterDefinition 列表转换为完整的原始 JSON Schema map
// 顶层模式始终被视为对象
// title 和 description 用于顶层模式对象
func ConvertParamsToRawSchema(params []ParameterDefinition, title, description string) map[string]any {
	rawSchema := map[string]any{
		"type": "object",
	}
	if title != "" {
		rawSchema["title"] = title
	}
	if description != "" {
		rawSchema["description"] = description
	}

	properties := make(map[string]any)
	var requiredProps []string

	if len(params) > 0 {
		for _, paramDef := range params {
			propSchema := convertSingleParamDefToRawSchema(&paramDef)
			properties[paramDef.Name] = propSchema
			if paramDef.Required {
				requiredProps = append(requiredProps, paramDef.Name)
			}
		}
		rawSchema["properties"] = properties
		if len(requiredProps) > 0 {
			sort.Strings(requiredProps) // 确保required数组顺序一致
			rawSchema["required"] = requiredProps
		}
	} else {
		// 如果没有参数，则是一个没有定义属性的对象
		rawSchema["properties"] = make(map[string]any) // 空属性 map
	}

	return rawSchema
}

// convertSingleParamDefToRawSchema 将单个 ParameterDefinition 转换为原始 JSON Schema 部分 (map[string]any)
func convertSingleParamDefToRawSchema(paramDef *ParameterDefinition) map[string]any {
	propSchema := map[string]any{
		"type": string(paramDef.Type), // Convert ParameterType to string
	}

	if desc, ok := paramDef.Description["en"]; ok && desc != "" {
		propSchema["description"] = desc
	} else if desc, ok := paramDef.Description["zh"]; ok && desc != "" { // Fallback to zh if en is empty
		propSchema["description"] = desc
	}

	if paramDef.EnumValues.IsSome() {
		if enumVals := paramDef.EnumValues.Unwrap(); len(enumVals) > 0 {
			propSchema["enum"] = enumVals
		}
	}

	// 处理序列化时的默认值
	if paramDef.DefaultValue.IsSome() {
		propSchema["default"] = paramDef.DefaultValue.Unwrap()
	}

	switch paramDef.Type {
	case ParamTypeObject:
		// 对于对象，我们创建它的 "properties" 和 "required" 字段
		// 如果 Properties 为 None 或空切片，则结果为空 "properties": {} map 用于模式
		nestedPropsMap := make(map[string]any)
		var nestedRequiredList []string
		if paramDef.Properties.IsSome() {
			nestedParams := paramDef.Properties.Unwrap()
			if len(nestedParams) > 0 {
				for _, nestedParam := range nestedParams {
					nestedPropsMap[nestedParam.Name] = convertSingleParamDefToRawSchema(&nestedParam)
					if nestedParam.Required {
						nestedRequiredList = append(nestedRequiredList, nestedParam.Name)
					}
				}
			}
		}
		// 即使 nestedParams 为空，对象类型也应该有一个 "properties" 字段 (可能是空的)
		propSchema["properties"] = nestedPropsMap
		if len(nestedRequiredList) > 0 {
			sort.Strings(nestedRequiredList) // 确保嵌套对象的required数组顺序一致
			propSchema["required"] = nestedRequiredList
		}

	case ParamTypeArray:
		if paramDef.Items.IsSome() {
			itemDef := paramDef.Items.Unwrap()
			propSchema["items"] = convertSingleParamDefToRawSchema(&itemDef)
		}
	}
	return propSchema
}

// ===== 转换层: genai.Schema 与 ParameterDefinition/ToolSchema 互相转换 =====

// ConvertGenaiSchemaToParams 将 genai.Schema 转换为 ParameterDefinition 列表
// schema: genai.Schema 对象，通常是 object 类型的根模式
func ConvertGenaiSchemaToParams(genaiSchema *schema.Schema) []ParameterDefinition {
	if genaiSchema == nil || string(genaiSchema.Type) != "OBJECT" {
		return []ParameterDefinition{}
	}

	// 获取属性和必需字段
	properties := genaiSchema.Properties
	required := genaiSchema.Required

	if properties == nil {
		return []ParameterDefinition{}
	}

	// 获取所有属性名并排序
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	params := make([]ParameterDefinition, 0, len(properties))
	for _, name := range names {
		propSchema := properties[name]
		if propSchema == nil {
			continue
		}
		param := convertGenaiSchemaToParamDef(name, propSchema, required)
		params = append(params, param)
	}
	return params
}

// convertGenaiSchemaToParamDef 将单个 genai.Schema 转换为 ParameterDefinition
func convertGenaiSchemaToParamDef(propName string, propSchema *schema.Schema, parentRequiredList []string) ParameterDefinition {
	param := ParameterDefinition{
		Name: propName,
		Description: map[string]string{
			"en": propSchema.Description,
			"zh": propSchema.Description, // 默认使用英文描述作为中文
		},
		Required: isNameInList(propName, parentRequiredList),
	}

	// 转换类型
	param.Type = convertGenaiTypeToParameterType(propSchema.Type)

	// 处理枚举值
	if len(propSchema.Enum) > 0 {
		enumValues := make([]any, len(propSchema.Enum))
		for i, v := range propSchema.Enum {
			enumValues[i] = v
		}
		param.EnumValues = Some(enumValues)
	}

	// 处理默认值
	if propSchema.Default != nil {
		param.DefaultValue = Some(propSchema.Default)
	}

	switch param.Type {
	case ParamTypeObject:
		if propSchema.Properties != nil {
			nestedParams := ConvertGenaiSchemaToParams(propSchema)
			param.Properties = Some(nestedParams)
		} else {
			param.Properties = Some([]ParameterDefinition{})
		}
	case ParamTypeArray:
		if propSchema.Items != nil {
			itemParamDef := convertGenaiSchemaToParamDef("item", propSchema.Items, []string{})
			param.Items = Some(itemParamDef)
		}
	}
	return param
}

// convertGenaiTypeToParameterType 将 genai.Type 转换为 ParameterType
func convertGenaiTypeToParameterType(genaiType genai.Type) ParameterType {
	switch string(genaiType) {
	case "STRING":
		return ParamTypeString
	case "NUMBER":
		return ParamTypeNumber
	case "INTEGER":
		return ParamTypeInteger
	case "BOOLEAN":
		return ParamTypeBoolean
	case "OBJECT":
		return ParamTypeObject
	case "ARRAY":
		return ParamTypeArray
	default:
		return ParamTypeAny
	}
}

// ConvertParamsToGenaiSchema 将 ParameterDefinition 列表转换为 genai.Schema
// params: 参数定义列表
// title: 模式标题
// description: 模式描述
func ConvertParamsToGenaiSchema(params []ParameterDefinition, title, description string) *schema.Schema {
	genaiSchema := &schema.Schema{
		Type:        genai.TypeObject,
		Title:       title,
		Description: description,
		Properties:  make(map[string]*schema.Schema),
	}

	var requiredProps []string

	for _, paramDef := range params {
		propSchema := convertParamDefToGenaiSchema(&paramDef)
		genaiSchema.Properties[paramDef.Name] = propSchema
		if paramDef.Required {
			requiredProps = append(requiredProps, paramDef.Name)
		}
	}

	if len(requiredProps) > 0 {
		sort.Strings(requiredProps)
		genaiSchema.Required = requiredProps
	}

	return genaiSchema
}

// convertParamDefToGenaiSchema 将单个 ParameterDefinition 转换为 genai.Schema
func convertParamDefToGenaiSchema(paramDef *ParameterDefinition) *schema.Schema {
	propSchema := &schema.Schema{
		Type: convertParameterTypeToGenaiType(paramDef.Type),
	}

	// 使用英文描述，如果没有则使用中文
	if desc, ok := paramDef.Description["en"]; ok && desc != "" {
		propSchema.Description = desc
	} else if desc, ok := paramDef.Description["zh"]; ok && desc != "" {
		propSchema.Description = desc
	}

	// 处理枚举值
	if paramDef.EnumValues.IsSome() {
		enumVals := paramDef.EnumValues.Unwrap()
		enumStrings := make([]string, len(enumVals))
		for i, v := range enumVals {
			if str, ok := v.(string); ok {
				enumStrings[i] = str
			} else {
				enumStrings[i] = fmt.Sprintf("%v", v)
			}
		}
		propSchema.Enum = enumStrings
	}

	// 处理默认值
	if paramDef.DefaultValue.IsSome() {
		propSchema.Default = paramDef.DefaultValue.Unwrap()
	}

	switch paramDef.Type {
	case ParamTypeObject:
		propSchema.Properties = make(map[string]*schema.Schema)
		var nestedRequired []string

		if paramDef.Properties.IsSome() {
			nestedParams := paramDef.Properties.Unwrap()
			for _, nestedParam := range nestedParams {
				propSchema.Properties[nestedParam.Name] = convertParamDefToGenaiSchema(&nestedParam)
				if nestedParam.Required {
					nestedRequired = append(nestedRequired, nestedParam.Name)
				}
			}
		}

		if len(nestedRequired) > 0 {
			sort.Strings(nestedRequired)
			propSchema.Required = nestedRequired
		}

	case ParamTypeArray:
		if paramDef.Items.IsSome() {
			itemDef := paramDef.Items.Unwrap()
			propSchema.Items = convertParamDefToGenaiSchema(&itemDef)
		}
	}

	return propSchema
}

// convertParameterTypeToGenaiType 将 ParameterType 转换为 genai.Type
func convertParameterTypeToGenaiType(paramType ParameterType) genai.Type {
	switch paramType {
	case ParamTypeString:
		return genai.TypeString
	case ParamTypeNumber:
		return genai.TypeNumber
	case ParamTypeInteger:
		return genai.TypeInteger
	case ParamTypeBoolean:
		return genai.TypeBoolean
	case ParamTypeObject:
		return genai.TypeObject
	case ParamTypeArray:
		return genai.TypeArray
	default:
		return genai.TypeUnspecified
	}
}

// ConvertToolSchemaToGenaiSchema 将 ToolSchema 转换为 genai.Schema
// toolSchema: 工具模式
func ConvertToolSchemaToGenaiSchema(toolSchema *ToolSchema) *schema.Schema {
	// 获取英文描述，如果没有则使用中文
	description := ""
	if desc, ok := toolSchema.Descriptions["en"]; ok && desc != "" {
		description = desc
	} else if desc, ok := toolSchema.Descriptions["zh"]; ok && desc != "" {
		description = desc
	}

	// 获取英文名称，如果没有则使用中文
	title := ""
	if name, ok := toolSchema.LocalizedNames["en"]; ok && name != "" {
		title = name
	} else if name, ok := toolSchema.LocalizedNames["zh"]; ok && name != "" {
		title = name
	} else {
		title = toolSchema.Name
	}

	return ConvertParamsToGenaiSchema(toolSchema.InputParameters, title, description)
}

// ConvertGenaiSchemaToToolSchema 将 genai.Schema 转换为 ToolSchema
// genaiSchema: genai.Schema 对象
// name: 工具名称
// localizedNames: 本地化名称映射
// descriptions: 本地化描述映射
func ConvertGenaiSchemaToToolSchema(genaiSchema *schema.Schema, name string, localizedNames, descriptions map[string]string) *ToolSchema {
	inputParams := ConvertGenaiSchemaToParams(genaiSchema)

	return &ToolSchema{
		Name:            name,
		LocalizedNames:  localizedNames,
		Descriptions:    descriptions,
		InputParameters: inputParams,
		// OutputParameters 需要单独处理，因为 genai.Schema 通常只描述输入
		OutputParameters: []ParameterDefinition{},
	}
}
