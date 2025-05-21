package reactagent

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

// 通用解析器实现

// JSONFormatParser 解析JSON格式的工具调用
// 期望格式: {"action": "tool_name", "action_input": {...}}
type JSONFormatParser struct{}

// ParseToolCalls 解析JSON格式的工具调用
func (p *JSONFormatParser) ParseToolCalls(text string) ([]llminterface.ToolCall, string) {
	// 提取所有可能的JSON对象
	jsonRegex := regexp.MustCompile(`\{(?:[^{}]|(?:\{(?:[^{}]|(?:\{[^{}]*\}))*\}))*\}`)
	matches := jsonRegex.FindAllString(text, -1)

	var toolCalls []llminterface.ToolCall
	for i, match := range matches {
		var parsed struct {
			Action      string          `json:"action"`
			ActionInput json.RawMessage `json:"action_input"`
		}

		if err := json.Unmarshal([]byte(match), &parsed); err != nil {
			continue
		}

		if parsed.Action != "" && len(parsed.ActionInput) > 0 {
			toolCalls = append(toolCalls, llminterface.ToolCall{
				ID:        fmt.Sprintf("tool-%d-%s", i, time.Now().Format("150405.000")),
				Name:      parsed.Action,
				Arguments: string(parsed.ActionInput),
			})

			// 从文本中移除已处理的JSON
			text = strings.Replace(text, match, "", 1)
		}
	}

	return toolCalls, strings.TrimSpace(text)
}

// MarkdownFormatParser 解析Markdown代码块格式的工具调用
// 期望格式:
// ```tool_name
// JSON参数
// ```
type MarkdownFormatParser struct{}

// ParseToolCalls 解析Markdown代码块格式的工具调用
func (p *MarkdownFormatParser) ParseToolCalls(text string) ([]llminterface.ToolCall, string) {
	// 提取Markdown代码块 ```tool_name ... ```
	mdRegex := regexp.MustCompile("```([a-zA-Z0-9_-]+)\\s*\\n([\\s\\S]*?)```")
	matches := mdRegex.FindAllStringSubmatch(text, -1)

	var toolCalls []llminterface.ToolCall
	for i, match := range matches {
		if len(match) < 3 {
			continue
		}

		toolName := strings.TrimSpace(match[1])
		arguments := strings.TrimSpace(match[2])

		if toolName != "" && arguments != "" {
			toolCalls = append(toolCalls, llminterface.ToolCall{
				ID:        fmt.Sprintf("tool-%d-%s", i, time.Now().Format("150405.000")),
				Name:      toolName,
				Arguments: arguments,
			})

			// 从文本中移除已处理的代码块
			text = strings.Replace(text, match[0], "", 1)
		}
	}

	return toolCalls, strings.TrimSpace(text)
}

// PredefinedPatternParser 基于预定义格式解析工具调用
// 格式例如：
// "工具: tool_name
// 参数: {...}"
type PredefinedPatternParser struct {
	// 工具调用开始模式
	StartPattern string
	// 参数开始模式
	ArgPattern string
}

// ParseToolCalls 解析预定义模式的工具调用
func (p *PredefinedPatternParser) ParseToolCalls(text string) ([]llminterface.ToolCall, string) {
	startPattern := p.StartPattern
	if startPattern == "" {
		startPattern = "工具:"
	}

	argPattern := p.ArgPattern
	if argPattern == "" {
		argPattern = "参数:"
	}

	// 构建正则表达式
	pattern := fmt.Sprintf("%s\\s*([a-zA-Z0-9_-]+)\\s*(?:\\n|\\r\\n?)%s\\s*([\\s\\S]*?)(?:(?:\\n|\\r\\n?)(?:%s|$)|$)",
		regexp.QuoteMeta(startPattern),
		regexp.QuoteMeta(argPattern),
		regexp.QuoteMeta(startPattern))

	regex := regexp.MustCompile(pattern)
	matches := regex.FindAllStringSubmatch(text, -1)

	var toolCalls []llminterface.ToolCall
	for i, match := range matches {
		if len(match) < 3 {
			continue
		}

		toolName := strings.TrimSpace(match[1])
		arguments := strings.TrimSpace(match[2])

		if toolName != "" {
			toolCalls = append(toolCalls, llminterface.ToolCall{
				ID:        fmt.Sprintf("tool-%d-%s", i, time.Now().Format("150405.000")),
				Name:      toolName,
				Arguments: arguments,
			})

			// 从文本中移除已处理的匹配
			text = strings.Replace(text, match[0], "", 1)
		}
	}

	return toolCalls, strings.TrimSpace(text)
}

// GuiAgentFormatParser 解析类似GUI Agent格式的工具调用
// 期望格式:
// Thought: ...
// Action: action_name(arg1="value1", arg2="value2")
type GuiAgentFormatParser struct{}

// ParseToolCalls 解析GUI Agent格式的工具调用
func (p *GuiAgentFormatParser) ParseToolCalls(text string) ([]llminterface.ToolCall, string) {
	// 提取GUI Agent格式的工具调用
	actionRegex := regexp.MustCompile(`Action:?\s*([a-zA-Z0-9_-]+)\(([^)]*)\)`)
	matches := actionRegex.FindAllStringSubmatch(text, -1)

	var toolCalls []llminterface.ToolCall
	for i, match := range matches {
		if len(match) < 3 {
			continue
		}

		toolName := strings.TrimSpace(match[1])
		argStr := strings.TrimSpace(match[2])

		// 将参数转换为JSON格式
		args := make(map[string]any)

		// 提取key="value"格式的参数
		argPairs := regexp.MustCompile(`([a-zA-Z0-9_-]+)\s*=\s*"([^"]*)"`)
		argMatches := argPairs.FindAllStringSubmatch(argStr, -1)
		for _, argMatch := range argMatches {
			if len(argMatch) >= 3 {
				key := argMatch[1]
				value := argMatch[2]
				args[key] = value
			}
		}

		// 提取key=[x1, y1, x2, y2]格式的参数
		boxRegex := regexp.MustCompile(`([a-zA-Z0-9_-]+)\s*=\s*\[([^\]]*)\]`)
		boxMatches := boxRegex.FindAllStringSubmatch(argStr, -1)
		for _, boxMatch := range boxMatches {
			if len(boxMatch) >= 3 {
				key := boxMatch[1]
				valueStr := boxMatch[2]

				// 解析数组值
				valueParts := strings.Split(valueStr, ",")
				var values []any
				for _, part := range valueParts {
					part = strings.TrimSpace(part)
					// 尝试解析为数字
					if num, err := strconv.ParseInt(part, 10, 64); err == nil {
						values = append(values, num)
					} else if fnum, err := strconv.ParseFloat(part, 64); err == nil {
						values = append(values, fnum)
					} else {
						// 作为字符串处理
						values = append(values, part)
					}
				}
				args[key] = values
			}
		}

		// 将参数转换为JSON字符串
		argsJSON, err := json.Marshal(args)
		if err != nil {
			continue
		}

		if toolName != "" {
			toolCalls = append(toolCalls, llminterface.ToolCall{
				ID:        fmt.Sprintf("tool-%d-%s", i, time.Now().Format("150405.000")),
				Name:      toolName,
				Arguments: string(argsJSON),
			})
		}
	}

	// 如果没有工具调用，保留整个文本
	if len(toolCalls) == 0 {
		return nil, text
	}

	// 提取思考部分
	thoughtRegex := regexp.MustCompile(`Thought:?\s*([\s\S]*?)(?:Action:|$)`)
	thoughtMatch := thoughtRegex.FindStringSubmatch(text)
	if len(thoughtMatch) >= 2 {
		return toolCalls, strings.TrimSpace(thoughtMatch[1])
	}

	// 如果没有明确的思考部分，移除Action部分后返回剩余文本
	for _, match := range matches {
		text = strings.Replace(text, match[0], "", 1)
	}

	return toolCalls, strings.TrimSpace(text)
}
