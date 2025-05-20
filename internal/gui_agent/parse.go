package guiagent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	json "github.com/json-iterator/go"
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
	thoughtRegex := regexp.MustCompile(`(?s)Thought:(.*?)\nAction:`)
	thoughtMatches := thoughtRegex.FindStringSubmatch(outputText)
	if len(thoughtMatches) > 1 {
		result.Thought = strings.TrimSpace(thoughtMatches[1])
	}

	// 提取Action部分
	actionRegex := regexp.MustCompile(`(?s)Action:(.*?)(?:\n|$)`)
	actionMatches := actionRegex.FindStringSubmatch(outputText)
	if len(actionMatches) > 1 {
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
							if key == "start_box" {
								result.StartBox = coords
							} else if key == "end_box" {
								result.EndBox = coords
							}
						}
					case key == "key":
						result.Key = &value
					case key == "content":
						// 处理转义字符
						value = processEscapeChars(value)
						result.Content = &value
					case key == "direction":
						result.Direction = &value
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

// ActionResult 表示解析后的操作结果
type ActionResult struct {
	Thought   string  `json:"thought"`
	Action    string  `json:"action"`
	Key       *string `json:"key"`
	Content   *string `json:"content"`
	StartBox  []int   `json:"start_box"`
	EndBox    []int   `json:"end_box"`
	Direction *string `json:"direction"`
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
