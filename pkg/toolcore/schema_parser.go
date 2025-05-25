package toolcore

import (
	"slices"
	"sort"

	"github.com/mark3labs/mcp-go/mcp"

	. "github.com/m4n5ter/another-me/pkg/option"
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
