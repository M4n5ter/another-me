package common

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

// SanitizeValue 递归地清洗数据结构（结构体、切片、数组、映射），将 NaN/Inf 浮点值替换为 nil。将结构体和映射转换为 map[string]any 和切片/数组转换为 []any。
func SanitizeValue(val any) any {
	if val == nil {
		return nil
	}

	v := reflect.ValueOf(val)

	// 解引用指针，直到获取到实际值或遇到 nil 指针
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		// 特殊处理 time.Time 等自带 MarshalJSON 的类型，如果它们没有 NaN 问题
		// if t, ok := v.Interface().(time.Time); ok {
		//  return t // 或者其标准序列化结果
		// }

		outMap := make(map[string]any)
		structType := v.Type()
		for i := range v.NumField() {
			fieldVal := v.Field(i)
			fieldType := structType.Field(i)

			// 跳过未导出的字段
			if fieldType.PkgPath != "" {
				continue
			}

			// 获取 json 标签名，如果不存在则用字段名
			fieldName := fieldType.Name
			jsonTag := fieldType.Tag.Get("json")
			if jsonTag != "" {
				parts := strings.Split(jsonTag, ",")
				if parts[0] != "" && parts[0] != "-" {
					fieldName = parts[0]
				} else if parts[0] == "-" { // 字段被忽略
					continue
				}
			}

			// 针对 *float64 和 float64 的特殊处理
			isFloatPtr := fieldVal.Kind() == reflect.Ptr && fieldVal.Type().Elem().Kind() == reflect.Float64
			isFloat := fieldVal.Kind() == reflect.Float64

			switch {
			case isFloatPtr:
				if fieldVal.IsNil() {
					outMap[fieldName] = nil
				} else {
					fVal := fieldVal.Elem().Float()
					if math.IsNaN(fVal) || math.IsInf(fVal, 0) {
						outMap[fieldName] = nil // 将 NaN/Inf 替换为 null
					} else {
						outMap[fieldName] = fVal
					}
				}
			case isFloat:
				fVal := fieldVal.Float()
				if math.IsNaN(fVal) || math.IsInf(fVal, 0) {
					outMap[fieldName] = nil // 将 NaN/Inf 替换为 null
				} else {
					outMap[fieldName] = fVal
				}
			default:
				// 对其他所有字段递归调用 sanitizeValue
				outMap[fieldName] = SanitizeValue(fieldVal.Interface())
			}
		}
		return outMap

	case reflect.Slice, reflect.Array:
		// 如果原始slice是nil，json.Marshal会正确处理为null，但这里我们构建新slice
		if v.IsNil() {
			return nil
		}
		outSlice := make([]any, v.Len())
		for i := range v.Len() {
			outSlice[i] = SanitizeValue(v.Index(i).Interface())
		}
		return outSlice

	case reflect.Map:
		if v.IsNil() {
			return nil
		}
		outMap := make(map[string]any)
		iter := v.MapRange()
		for iter.Next() {
			key := fmt.Sprintf("%v", iter.Key().Interface()) // 将 map key 转为 string
			outMap[key] = SanitizeValue(iter.Value().Interface())
		}
		return outMap

	// 如果原始数据中直接有 float64 (非指针) 且可能是 NaN/Inf
	case reflect.Float64, reflect.Float32:
		fVal := v.Float()
		if math.IsNaN(fVal) || math.IsInf(fVal, 0) {
			return nil // 将 NaN/Inf 替换为 null
		}
		return fVal

	default:
		// 对于其他基础类型 (string, int, bool 等)，直接返回它们的值
		// 确保值是有效的并且可以被接口化
		if v.IsValid() && v.CanInterface() {
			return v.Interface()
		}
		return nil
	}
}
