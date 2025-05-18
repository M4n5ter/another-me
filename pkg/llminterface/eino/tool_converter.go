package eino

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/cloudwego/eino/schema"

	"github.com/m4n5ter/another-me/pkg/toolcore"
)

const defaultLanguageForDesc = "en"

// ToolCoreSchemaToEinoToolInfo converts a toolcore.ToolSchema to an eino.schema.ToolInfo.
// lang 参数指定了用于描述的语言。
func ToolCoreSchemaToEinoToolInfo(tcSchema *toolcore.ToolSchema, lang string) (*schema.ToolInfo, error) {
	if tcSchema == nil {
		return nil, errors.New("ToolCoreSchemaToEinoToolInfo: tcSchema cannot be nil")
	}
	if lang == "" {
		lang = defaultLanguageForDesc
		slog.Debug("ToolCoreSchemaToEinoToolInfo: lang not provided, using default.", "defaultLang", defaultLanguageForDesc)
	}

	var translatedToolDesc string
	if desc, ok := tcSchema.Descriptions[lang]; ok {
		translatedToolDesc = desc
	} else if desc, ok := tcSchema.Descriptions[defaultLanguageForDesc]; ok {
		translatedToolDesc = desc
	} else if len(tcSchema.Descriptions) > 0 {
		for _, d := range tcSchema.Descriptions {
			translatedToolDesc = d
			slog.Debug("Target/default language description not found for tool, using first available.", "toolName", tcSchema.Name, "targetLang", lang)
			break
		}
	} else {
		translatedToolDesc = tcSchema.Name // Fallback
		slog.Warn("No description found for tool, using tool name as fallback.", "toolName", tcSchema.Name)
	}

	parameterProperties := make(map[string]*schema.ParameterInfo)
	// requiredParamsList := make([]string, 0) // Linter indicates NewParamsOneOfByParams does not take this directly

	if len(tcSchema.InputParameters) > 0 {
		for i := range tcSchema.InputParameters {
			paramDef := tcSchema.InputParameters[i]
			einoParamInfo, err := convertToolCoreParamDefToEinoParamInfo(&paramDef, lang)
			if err != nil {
				slog.Error("Failed to convert parameter definition for Eino", "tool", tcSchema.Name, "param", paramDef.Name, "error", err)
				continue
			}
			parameterProperties[paramDef.Name] = einoParamInfo
			// paramDef.Required is set within each einoParamInfo.Required
			// if paramDef.Required {
			// 	requiredParamsList = append(requiredParamsList, paramDef.Name)
			// }
		}
	}

	// schema.NewParamsOneOfByParams expects map[string]*schema.ParameterInfo and returns *schema.ParamsOneOf (no error)
	// The handling of the overall "required" list masih belum jelas jika hanya map properties yang dilewatkan.
	// Kita asumsikan SDK Eino mungkin menyimpulkannya dari ParameterInfo.Required masing-masing.
	paramsOneOf := schema.NewParamsOneOfByParams(parameterProperties)
	// Tidak ada error yang dikembalikan oleh NewParamsOneOfByParams berdasarkan linter.

	// // Old code that tried to build a more complex params object:
	// paramsObjectForEino := map[string]interface{}{
	// 	"type":       "object",
	// 	"properties": parameterProperties,
	// }
	// if len(requiredParamsList) > 0 {
	// 	paramsObjectForEino["required"] = requiredParamsList
	// }
	// paramsOneOf, err := schema.NewParamsOneOfByParams(paramsObjectForEino)
	// if err != nil {
	// 	slog.Error("schema.NewParamsOneOfByParams failed", "toolName", tcSchema.Name, "error", err, "constructedParamsObject", fmt.Sprintf("%+v", paramsObjectForEino))
	// 	return nil, fmt.Errorf("failed to create ParamsOneOf for tool '%s' using NewParamsOneOfByParams: %w", tcSchema.Name, err)
	// }

	return &schema.ToolInfo{
		Name:        tcSchema.Name,
		Desc:        translatedToolDesc,
		ParamsOneOf: paramsOneOf,
	}, nil
}

// toolcoreParamTypeToEinoDataType 将 toolcore.ParameterType 转换为 eino.schema.DataType。
func toolcoreParamTypeToEinoDataType(tcType toolcore.ParameterType) schema.DataType {
	switch tcType {
	case toolcore.ParamTypeString:
		return schema.String
	case toolcore.ParamTypeNumber:
		return schema.Number
	case toolcore.ParamTypeInteger:
		return schema.Integer
	case toolcore.ParamTypeBoolean:
		return schema.Boolean
	case toolcore.ParamTypeObject:
		return schema.Object
	case toolcore.ParamTypeArray:
		return schema.Array
	case toolcore.ParamTypeNull:
		return schema.Null
	case toolcore.ParamTypeAny:
		// eino.schema.DataType 没有直接的 "any" 类型。
		// 根据实际需求，可能需要映射为 string 或 object，或者报错。
		// 这里暂时映射为 String 并记录警告。
		slog.Warn("toolcore.ParamTypeAny is being mapped to eino.schema.String. Review if this is appropriate.", "originalType", tcType)
		return schema.String
	default:
		slog.Warn("Unknown toolcore.ParameterType encountered during conversion to eino.schema.DataType, defaulting to string.", "unknownType", tcType)
		return schema.String
	}
}

// convertToolCoreParamDefToEinoParamInfo 将单个 toolcore.ParameterDefinition 转换为 eino.schema.ParameterInfo。
// lang 参数用于从多语言描述中选择一个。
func convertToolCoreParamDefToEinoParamInfo(paramDef *toolcore.ParameterDefinition, lang string) (*schema.ParameterInfo, error) {
	if paramDef == nil {
		return nil, errors.New("convertToolCoreParamDefToEinoParamInfo: paramDef cannot be nil")
	}

	einoParamInfo := &schema.ParameterInfo{
		Type:     toolcoreParamTypeToEinoDataType(paramDef.Type),
		Required: paramDef.Required,
	}

	var translatedParamDesc string
	if desc, ok := paramDef.Description[lang]; ok {
		translatedParamDesc = desc
	} else if desc, ok := paramDef.Description[defaultLanguageForDesc]; ok {
		translatedParamDesc = desc
	} else if len(paramDef.Description) > 0 {
		for _, d := range paramDef.Description {
			translatedParamDesc = d
			slog.Debug("Target/default language description not found for param, using first available.", "paramName", paramDef.Name, "targetLang", lang)
			break
		}
	}
	einoParamInfo.Desc = translatedParamDesc

	// 处理枚举值 (eino 需要 []string)
	if len(paramDef.EnumValues) > 0 {
		einoParamInfo.Enum = make([]string, 0, len(paramDef.EnumValues))
		for _, enumVal := range paramDef.EnumValues {
			if strVal, ok := enumVal.(string); ok {
				einoParamInfo.Enum = append(einoParamInfo.Enum, strVal)
			} else {
				slog.Warn("Non-string enum value encountered and skipped during conversion.", "paramName", paramDef.Name, "value", enumVal)
			}
		}
	}

	// 处理对象类型的属性
	if paramDef.Type == toolcore.ParamTypeObject {
		if len(paramDef.Properties) > 0 {
			einoParamInfo.SubParams = make(map[string]*schema.ParameterInfo, len(paramDef.Properties))
			for i := range paramDef.Properties { // 迭代属性
				subParamDef := paramDef.Properties[i]                                           // 获取单个属性定义
				einoSubParam, err := convertToolCoreParamDefToEinoParamInfo(&subParamDef, lang) // 传递 lang
				if err != nil {
					return nil, fmt.Errorf("error converting sub-parameter '%s' for object '%s': %w", subParamDef.Name, paramDef.Name, err)
				}
				einoParamInfo.SubParams[subParamDef.Name] = einoSubParam
			}
		}
	}

	// 处理数组类型的元素类型
	if paramDef.Type == toolcore.ParamTypeArray {
		if paramDef.Items != nil {
			einoItemInfo, err := convertToolCoreParamDefToEinoParamInfo(paramDef.Items, lang) // 传递 lang
			if err != nil {
				return nil, fmt.Errorf("error converting items for array '%s': %w", paramDef.Name, err)
			}
			einoParamInfo.ElemInfo = einoItemInfo
		} else {
			slog.Warn("Array parameter definition missing 'Items' specification.", "paramName", paramDef.Name)
		}
	}
	return einoParamInfo, nil
}
