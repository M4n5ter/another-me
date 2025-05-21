package guiagent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	json "github.com/json-iterator/go"

	. "github.com/m4n5ter/another-me/pkg/option"
)

// https://www.volcengine.com/docs/82379/1536429?lang=zh

// ParseActionOutput 解析LLM输出的Action文本
func ParseActionOutput(outputText string) (string, error) {
	// 初始化结果结构体
	result := ActionResult{
		Thought: "",
		Action:  "",
	}

	// 提取Thought部分
	thoughtRegex := regexp.MustCompile(`(?s)Thought:(.*?)\nAction:`) // Satety: const pattern
	thoughtMatches := thoughtRegex.FindStringSubmatch(outputText)
	if len(thoughtMatches) > 1 {
		result.Thought = strings.TrimSpace(thoughtMatches[1])
	}

	// 提取Action部分
	actionRegex := regexp.MustCompile(`(?s)Action:(.*?)(?:\n|$)`) // Satety: const pattern
	actionMatches := actionRegex.FindStringSubmatch(outputText)
	if len(actionMatches) > 1 { //nolint:nestif // ParseActionOutput 的逻辑是必要的
		actionText := strings.TrimSpace(actionMatches[1])
		if actionText == "" {
			// 没有action信息，直接返回结果
			jsonResult, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return "", fmt.Errorf("JSON编码错误: %w", err)
			}
			return string(jsonResult), nil
		}

		// 解析action类型
		actionParts := strings.SplitN(actionText, "(", 2)
		result.Action = actionParts[0]

		// 解析参数
		if len(actionParts) > 1 {
			paramsText := strings.TrimRight(actionParts[1], ")")

			// 处理键值对参数
			params := splitParams(paramsText)
			for _, param := range params {
				param = strings.TrimSpace(param)
				if strings.Contains(param, "=") {
					parts := strings.SplitN(param, "=", 2)
					key := strings.TrimSpace(parts[0])
					value := strings.TrimSpace(parts[1])

					// 去除引号
					value = strings.Trim(value, "'\"")

					// 处理不同类型的参数
					switch {
					case strings.Contains(key, "box"):
						coords := extractCoordinates(value)
						if len(coords) == 4 {
							switch key {
							case "start_box":
								result.StartBox = coords
							case "end_box":
								result.EndBox = coords
							}
						}
					case key == "key":
						result.Key = Some(value)
					case key == "content":
						// 处理转义字符
						value = processEscapeChars(value)
						result.Content = Some(value)
					case key == "direction":
						result.Direction = Some(value)
					case key == "button":
						result.Button = Some(value)
					case key == "up":
						upVal := strings.ToLower(value) == "true"
						result.Up = Some(upVal)
					case key == "keys":
						// 处理keys数组
						keys, err := parseKeysArray(value)
						if err == nil && len(keys) > 0 {
							result.Keys = Some(keys)
						}
					case key == "x":
						if num, err := strconv.Atoi(value); err == nil {
							result.X = Some(num)
						}
					case key == "y":
						if num, err := strconv.Atoi(value); err == nil {
							result.Y = Some(num)
						}
					case key == "ms":
					case key == "ms_delay":
						if num, err := strconv.Atoi(value); err == nil {
							result.MS = Some(num)
						}
					}
				}
			}
		}
	}

	// 转换为JSON
	jsonResult, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON编码错误: %w", err)
	}
	return string(jsonResult), nil
}

// 解析keys数组字符串，例如：["ctrl", "alt", "a"]
func parseKeysArray(value string) ([]string, error) {
	// 移除可能存在的方括号
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")

	// 分割字符串
	var keys []string
	for _, key := range strings.Split(value, ",") {
		key = strings.TrimSpace(key)
		key = strings.Trim(key, "\"'")
		if key != "" {
			keys = append(keys, key)
		}
	}

	if len(keys) == 0 {
		return nil, fmt.Errorf("无效的keys数组")
	}

	return keys, nil
}

// ActionResult 表示解析后的操作结果
type ActionResult struct {
	Thought   string           `json:"thought"`
	Action    string           `json:"action"`
	Key       Option[string]   `json:"key,omitempty"`
	Content   Option[string]   `json:"content,omitempty"`
	StartBox  []int            `json:"start_box,omitempty"`
	EndBox    []int            `json:"end_box,omitempty"`
	Direction Option[string]   `json:"direction,omitempty"`
	Button    Option[string]   `json:"button,omitempty"`
	Up        Option[bool]     `json:"up,omitempty"`
	Keys      Option[[]string] `json:"keys,omitempty"`
	X         Option[int]      `json:"x,omitempty"`
	Y         Option[int]      `json:"y,omitempty"`
	MS        Option[int]      `json:"ms,omitempty"`
}

// CoordinatesConvert 将相对坐标[0,1000]转换为图片上的绝对像素坐标
func CoordinatesConvert(relativeBBox []int, imgSize [2]int) ([]int, error) {
	// 参数校验
	if len(relativeBBox) != 4 {
		return nil, fmt.Errorf("相对坐标必须是4个元素: [x1,y1,x2,y2]")
	}

	// 解包图片尺寸
	imgWidth, imgHeight := imgSize[0], imgSize[1]

	// 计算绝对坐标
	absX1 := int(float64(relativeBBox[0]) * float64(imgWidth) / 1000)
	absY1 := int(float64(relativeBBox[1]) * float64(imgHeight) / 1000)
	absX2 := int(float64(relativeBBox[2]) * float64(imgWidth) / 1000)
	absY2 := int(float64(relativeBBox[3]) * float64(imgHeight) / 1000)

	return []int{absX1, absY1, absX2, absY2}, nil
}

// 辅助函数：提取坐标数字
func extractCoordinates(value string) []int {
	re := regexp.MustCompile(`\d+`)
	matches := re.FindAllString(value, -1)

	coords := make([]int, 0, 4)
	for _, match := range matches {
		num, err := strconv.Atoi(match)
		if err == nil {
			coords = append(coords, num)
		}
	}

	return coords
}

// 辅助函数：处理转义字符
func processEscapeChars(value string) string {
	value = strings.ReplaceAll(value, "\\n", "\n")
	value = strings.ReplaceAll(value, "\\\"", "\"")
	value = strings.ReplaceAll(value, "\\'", "'")
	return value
}

// 辅助函数：分割参数，处理括号内的复杂情况
func splitParams(paramsText string) []string {
	var result []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)
	depth := 0

	for _, char := range paramsText {
		switch char {
		case '\'', '"':
			if inQuote && char == quoteChar {
				inQuote = false
			} else if !inQuote {
				inQuote = true
				quoteChar = char
			}
			current.WriteRune(char)
		case '[', '(':
			depth++
			current.WriteRune(char)
		case ']', ')':
			depth--
			current.WriteRune(char)
		case ',':
			if inQuote || depth > 0 {
				current.WriteRune(char)
			} else {
				result = append(result, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(char)
		}
	}

	// 添加最后一个参数
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}
