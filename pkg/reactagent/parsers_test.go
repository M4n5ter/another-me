package reactagent

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/m4n5ter/another-me/pkg/llminterface"
)

func TestJSONFormatParser_ParseToolCalls(t *testing.T) {
	parser := &JSONFormatParser{}

	tests := []struct {
		name              string
		text              string
		expectedToolCalls []llminterface.ToolCall
		expectedRemaining string
		expectError       bool
	}{
		{
			name: "single valid tool call",
			text: `{"action": "search", "action_input": {"query": "golang"}}`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "golang"}`},
			},
			expectedRemaining: "",
		},
		{
			name: "multiple valid tool calls",
			text: `{"action": "search", "action_input": {"query": "golang"}} Some text in between {"action": "fetch", "action_input": {"url": "example.com"}}`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "golang"}`},
				{Name: "fetch", Arguments: `{"url": "example.com"}`},
			},
			expectedRemaining: "Some text in between",
		},
		{
			name: "valid tool call with surrounding text",
			text: `Thought: I need to search. {"action": "search", "action_input": {"query": "rust"}} Result:`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "rust"}`},
			},
			expectedRemaining: "Thought: I need to search.  Result:",
		},
		{
			name: "nested JSON (should still parse outer tool call)",
			text: `{"action": "complex_tool", "action_input": {"param1": "value1", "nested_json": {"key": "value"}}}`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "complex_tool", Arguments: `{"param1": "value1", "nested_json": {"key": "value"}}`},
			},
			expectedRemaining: "",
		},
		{
			name:              "invalid JSON",
			text:              `{"action": "search", "action_input": {"query": "golang"}`, // 缺少闭合括号
			expectedToolCalls: nil,
			expectedRemaining: `{"action": "search", "action_input": {"query": "golang"}`,
		},
		{
			name:              "no action",
			text:              `{"action_input": {"query": "golang"}}`,
			expectedToolCalls: nil,
			expectedRemaining: `{"action_input": {"query": "golang"}}`,
		},
		{
			name:              "no action_input",
			text:              `{"action": "search"}`,
			expectedToolCalls: nil,
			expectedRemaining: `{"action": "search"}`,
		},
		{
			name:              "empty text",
			text:              "",
			expectedToolCalls: nil,
			expectedRemaining: "",
		},
		{
			name:              "text without JSON",
			text:              "This is just a regular sentence.",
			expectedToolCalls: nil,
			expectedRemaining: "This is just a regular sentence.",
		},
		{
			name: "tool call with escaped quotes in argument",
			text: `{"action": "echo", "action_input": {"message": "hello \"world\""}}`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "echo", Arguments: `{"message": "hello \"world\""}`},
			},
			expectedRemaining: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCalls, remaining := parser.ParseToolCalls(tt.text)

			assert.Equal(t, len(tt.expectedToolCalls), len(toolCalls), "Number of tool calls did not match")

			for i, tc := range toolCalls {
				assertToolIDFormat(t, tc.ID)
				assert.Equal(t, tt.expectedToolCalls[i].Name, tc.Name, "Tool call name did not match")
				assert.JSONEq(t, tt.expectedToolCalls[i].Arguments, tc.Arguments, "Tool call arguments did not match")
			}
			assert.Equal(t, tt.expectedRemaining, remaining, "Remaining text did not match")
		})
	}

	t.Run("ExceptFormat", func(t *testing.T) {
		expected := `{"action": "tool_name", "action_input": {...}}` + "\nExample: {\"action\": \"tool_name\", \"action_input\": {\"arg1\": \"value1\", \"arg2\": \"value2\"}}"
		assert.Equal(t, expected, parser.ExceptFormat())
	})
}

func TestMarkdownFormatParser_ParseToolCalls(t *testing.T) {
	parser := &MarkdownFormatParser{}

	tests := []struct {
		name              string
		text              string
		expectedToolCalls []llminterface.ToolCall
		expectedRemaining string
	}{
		{
			name: "single valid tool call",
			text: "```search\n{\"query\": \"golang\"}\n```",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "golang"}`},
			},
			expectedRemaining: "",
		},
		{
			name: "multiple valid tool calls",
			text: "```search\n{\"query\": \"golang\"}\n```\nSome text\n```fetch\n{\"url\": \"example.com\"}\n```",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "golang"}`},
				{Name: "fetch", Arguments: `{"url": "example.com"}`},
			},
			expectedRemaining: "Some text",
		},
		{
			name: "tool call with surrounding text",
			text: "Thought: I need to search.\n```search\n{\"query\": \"rust\"}\n```\nResult:",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "rust"}`},
			},
			expectedRemaining: "Thought: I need to search.\n\nResult:",
		},
		{
			name:              "incomplete markdown",
			text:              "```search\n{\"query\": \"golang\"}",
			expectedToolCalls: nil,
			expectedRemaining: "```search\n{\"query\": \"golang\"}",
		},
		{
			name:              "no tool name",
			text:              "```\n{\"query\": \"golang\"}\n```",
			expectedToolCalls: nil,
			expectedRemaining: "```\n{\"query\": \"golang\"}\n```",
		},
		{
			name:              "no arguments",
			text:              "```search\n\n```",
			expectedToolCalls: nil, // 当前行为，可以争论是否应该有一个带有空参数的工具调用
			expectedRemaining: "```search\n\n```",
		},
		{
			name:              "empty text",
			text:              "",
			expectedToolCalls: nil,
			expectedRemaining: "",
		},
		{
			name:              "text without markdown",
			text:              "This is just a regular sentence.",
			expectedToolCalls: nil,
			expectedRemaining: "This is just a regular sentence.",
		},
		{
			name: "tool name with hyphens and underscores",
			text: "```my-tool_v1\n{\"param\": \"value\"}\n```",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "my-tool_v1", Arguments: `{"param": "value"}`},
			},
			expectedRemaining: "",
		},
		{
			name: "arguments with leading/trailing whitespace (should be trimmed)",
			text: "```search\n  {\"query\": \"whitespace\"}  \n```",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "whitespace"}`},
			},
			expectedRemaining: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCalls, remaining := parser.ParseToolCalls(tt.text)

			assert.Equal(t, len(tt.expectedToolCalls), len(toolCalls), "Number of tool calls did not match")

			for i, tc := range toolCalls {
				assertToolIDFormat(t, tc.ID)
				assert.Equal(t, tt.expectedToolCalls[i].Name, tc.Name, "Tool call name did not match")
				assert.JSONEq(t, tt.expectedToolCalls[i].Arguments, tc.Arguments, "Tool call arguments did not match")
			}
			assert.Equal(t, tt.expectedRemaining, remaining, "Remaining text did not match")
		})
	}
	t.Run("ExceptFormat", func(t *testing.T) {
		expected := "```tool_name\nTool Arguments (JSON)\n```" + "\nExample: ```tool_name\n{\"arg1\": \"value1\", \"arg2\": \"value2\"}\n```"
		assert.Equal(t, expected, parser.ExceptFormat())
	})
}

func TestPredefinedPatternParser_ParseToolCalls(t *testing.T) {
	defaultParser := &PredefinedPatternParser{} // Uses default "工具:" and "参数:"
	customParser := &PredefinedPatternParser{
		StartPattern: "Action:",
		ArgPattern:   "Action Input:",
	}

	tests := []struct {
		name              string
		parser            TextFormatParser
		text              string
		expectedToolCalls []llminterface.ToolCall
		expectedRemaining string
	}{
		// 默认解析器测试
		{
			name:   "default parser - single valid tool call",
			parser: defaultParser,
			text:   "工具: search\n参数: {\"query\": \"golang\"}",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "golang"}`},
			},
			expectedRemaining: "",
		},
		{
			name:   "default parser - multiple valid tool calls",
			parser: defaultParser,
			text:   "工具: search\n参数: {\"query\": \"golang\"}\nSome text\n工具: fetch\n参数: {\"url\": \"example.com\"}",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "golang"}`},
				{Name: "fetch", Arguments: `{"url": "example.com"}`},
			},
			expectedRemaining: "Some text",
		},
		{
			name:   "default parser - tool call with surrounding text",
			parser: defaultParser,
			text:   "Thought: I need to search.\n工具: search\n参数: {\"query\": \"rust\"}\nResult:",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "rust"}`},
			},
			expectedRemaining: "Thought: I need to search.\n\nResult:",
		},
		{
			name:   "default parser - no arguments provided (empty string for args)",
			parser: defaultParser,
			text:   "工具: no_args_tool\n参数: ",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "no_args_tool", Arguments: ""},
			},
			expectedRemaining: "",
		},
		{
			name:   "default parser - arguments are not JSON (should still capture but result in empty string for extracted JSON)",
			parser: defaultParser,
			text:   "工具: custom_args\\n参数: plain text arguments",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "custom_args", Arguments: ""}, // 修正：非 JSON 参数将导致空字符串，因为当前 JSON 提取逻辑
			},
			expectedRemaining: "", // 整个块 "工具: custom_args\\n参数: plain text arguments" 应该被消耗
		},
		{
			name:              "default parser - missing arg pattern",
			parser:            defaultParser,
			text:              "工具: search\n{\"query\": \"golang\"}",
			expectedToolCalls: nil,
			expectedRemaining: "工具: search\n{\"query\": \"golang\"}",
		},
		{
			name:   "default parser - tool call followed by text then another tool call",
			parser: defaultParser,
			text:   "工具: first_tool\n参数: {\"id\": 1}\nThis is some intermediate text.\n工具: second_tool\n参数: {\"id\": 2}",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "first_tool", Arguments: `{"id": 1}`},
				{Name: "second_tool", Arguments: `{"id": 2}`},
			},
			expectedRemaining: "This is some intermediate text.",
		},
		{
			name:   "default parser - tool call arguments ending a line, then more text",
			parser: defaultParser,
			text:   "工具: mytool\n参数: {\"key\":\"value\"}\nSome other text.",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "mytool", Arguments: `{"key":"value"}`},
			},
			expectedRemaining: "Some other text.",
		},
		{
			name:   "default parser - tool call arguments spanning multiple lines (regex should handle)",
			parser: defaultParser,
			text:   "工具: mytool\n参数: {\n  \"key\": \"value\",\n  \"another_key\": \"another_value\"\n}\nText after.",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "mytool", Arguments: "{\n  \"key\": \"value\",\n  \"another_key\": \"another_value\"\n}"},
			},
			expectedRemaining: "Text after.",
		},
		{
			name:   "default parser - only tool call in text, no trailing newline for args",
			parser: defaultParser,
			text:   "工具: tool_no_newline\n参数: {\"arg\": \"val\"}",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "tool_no_newline", Arguments: `{"arg": "val"}`},
			},
			expectedRemaining: "",
		},
		{
			name:   "default parser - tool call with empty JSON object args",
			parser: defaultParser,
			text:   "工具: tool_empty_json_obj\n参数: {}",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "tool_empty_json_obj", Arguments: `{}`},
			},
			expectedRemaining: "",
		},
		{
			name:   "default parser - tool call with JSON array args",
			parser: defaultParser,
			text:   "工具: tool_json_array_args\n参数: [1, \"test\", true]",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "tool_json_array_args", Arguments: `[1, "test", true]`},
			},
			expectedRemaining: "",
		},
		{
			name:   "default parser - text after args but before next tool pattern",
			parser: defaultParser,
			text:   "工具: tool1\n参数: {\"a\":1}\nSome middle text\nNot a tool line\n工具: tool2\n参数: {\"b\":2}",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "tool1", Arguments: "{\"a\":1}"},
				{Name: "tool2", Arguments: "{\"b\":2}"},
			},
			expectedRemaining: "Some middle text\nNot a tool line",
		},

		// 自定义解析器测试
		{
			name:   "custom parser - single valid tool call",
			parser: customParser,
			text:   "Action: search\nAction Input: {\"query\": \"golang\"}",
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "search", Arguments: `{"query": "golang"}`},
			},
			expectedRemaining: "",
		},
		{
			name:              "custom parser - wrong start pattern",
			parser:            customParser,
			text:              "工具: search\nAction Input: {\"query\": \"golang\"}",
			expectedToolCalls: nil,
			expectedRemaining: "工具: search\nAction Input: {\"query\": \"golang\"}",
		},
		{
			name:              "empty text",
			parser:            defaultParser,
			text:              "",
			expectedToolCalls: nil,
			expectedRemaining: "",
		},
		{
			name:              "text without predefined pattern",
			parser:            defaultParser,
			text:              "This is just a regular sentence.",
			expectedToolCalls: nil,
			expectedRemaining: "This is just a regular sentence.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCalls, remaining := tt.parser.ParseToolCalls(tt.text)

			assert.Equal(t, len(tt.expectedToolCalls), len(toolCalls), "Number of tool calls did not match")

			for i, tc := range toolCalls {
				assertToolIDFormat(t, tc.ID)
				assert.Equal(t, tt.expectedToolCalls[i].Name, tc.Name, "Tool call name did not match")
				// 对于 PredefinedPatternParser，参数可能不是 JSON，所以直接字符串比较
				if strings.HasPrefix(tt.expectedToolCalls[i].Arguments, "{") || strings.HasPrefix(tt.expectedToolCalls[i].Arguments, "[") {
					if tt.expectedToolCalls[i].Arguments != "" { // 避免空预期 JSON 的错误
						assert.JSONEq(t, tt.expectedToolCalls[i].Arguments, tc.Arguments, "Tool call arguments did not match JSON")
					} else {
						assert.Equal(t, tt.expectedToolCalls[i].Arguments, tc.Arguments, "Tool call arguments did not match empty string")
					}
				} else {
					assert.Equal(t, tt.expectedToolCalls[i].Arguments, tc.Arguments, "Tool call arguments did not match plain text")
				}
			}
			assert.Equal(t, tt.expectedRemaining, remaining, "Remaining text did not match")
		})
	}

	t.Run("Default Parser ExceptFormat", func(t *testing.T) {
		expected := fmt.Sprintf("%s ToolName\n%s Args\n", "工具:", "参数:") + "\nExample: " + "工具:" + " tool_name\n" + "参数:" + " {\"arg1\": \"value1\", \"arg2\": \"value2\"}"
		assert.Equal(t, expected, defaultParser.ExceptFormat())
	})

	t.Run("Custom Parser ExceptFormat", func(t *testing.T) {
		expected := fmt.Sprintf("%s ToolName\n%s Args\n", "Action:", "Action Input:") + "\nExample: " + "Action:" + " tool_name\n" + "Action Input:" + " {\"arg1\": \"value1\", \"arg2\": \"value2\"}"
		assert.Equal(t, expected, customParser.ExceptFormat())
	})
}

func TestGuiAgentFormatParser_ParseToolCalls(t *testing.T) {
	parser := &GuiAgentFormatParser{}

	tests := []struct {
		name              string
		text              string
		expectedToolCalls []llminterface.ToolCall
		expectedRemaining string
	}{
		{
			name: "single valid tool call",
			text: `Thought: I need to click a button.
Action: click(button_name="submit", confidence=0.9)`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "click", Arguments: `{"button_name":"submit", "confidence":0.9}`},
			},
			expectedRemaining: "I need to click a button.",
		},
		{
			name: "tool call with coordinates array",
			text: `Thought: Target a specific area.
Action: select_area(coordinates=[10, 20, 100, 50])`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "select_area", Arguments: `{"coordinates":[10, 20, 100, 50]}`},
			},
			expectedRemaining: "Target a specific area.",
		},
		{
			name: "tool call with mixed string and number args",
			text: `Action: type_text(text="hello world", delay_ms=100)`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "type_text", Arguments: `{"text":"hello world", "delay_ms":100}`},
			},
			expectedRemaining: "", // 没有思考
		},
		{
			name: "no thought, just action",
			text: `Action: press_key(key_name="enter")`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "press_key", Arguments: `{"key_name":"enter"}`},
			},
			expectedRemaining: "",
		},
		{
			name: "action with no arguments",
			text: `Action: finish()`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "finish", Arguments: `{}`},
			},
			expectedRemaining: "",
		},
		{
			name: "multiple actions", // 更新名称以提高清晰度
			text: `Thought: Step 1
Action: action1(param="value1")
Thought: Step 2
Action: action2(param="value2")`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "action1", Arguments: `{"param":"value1"}`},
				{Name: "action2", Arguments: `{"param":"value2"}`}, // Expect both actions
			},
			// 如果 thoughtRegex 成功提取 "Step 1"，则剩余文本将是 "Step 1"。
			// 第二个 "Thought: Step 2" 不会被当前的 thoughtRegex 逻辑提取，因为当第一个思考存在时。
			expectedRemaining: "Step 1",
		},
		{
			name: "action with spaces around equals and in args",
			text: `Action: spaced_tool(  arg1 = " value with spaces " , arg2 = [ 1 , 2 ] )`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "spaced_tool", Arguments: `{"arg1":" value with spaces ", "arg2":[1,2]}`},
			},
			expectedRemaining: "",
		},
		{
			name:              "just thought, no action",
			text:              `Thought: I am thinking.`,
			expectedToolCalls: nil,
			// 如果没有动作，ParseToolCalls 返回 (nil, text)
			expectedRemaining: "Thought: I am thinking.",
		},
		{
			name:              "malformed action (missing parenthesis)",
			text:              `Action: click(button_name="submit"`,
			expectedToolCalls: nil,
			expectedRemaining: `Action: click(button_name="submit"`,
		},
		{
			name:              "malformed action (bad arg format)",
			text:              `Action: click(button_name "submit")`,
			expectedToolCalls: nil,
			expectedRemaining: `Action: click(button_name "submit")`,
		},
		{
			name:              "empty text",
			text:              "",
			expectedToolCalls: nil,
			expectedRemaining: "",
		},
		{
			name:              "text without GUI agent pattern",
			text:              "This is just a regular sentence.",
			expectedToolCalls: nil,
			expectedRemaining: "This is just a regular sentence.",
		},
		{
			name:              "Action without thought prefix",
			text:              `action_without_thought(p1="v1")`,
			expectedToolCalls: nil, // 因为期望 "Action:" 前缀
			expectedRemaining: "action_without_thought(p1=\"v1\")",
		},
		{
			name: "Thought and Action with colon",
			text: `Thought: Thinking...
Action: tool_name(arg="val")`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "tool_name", Arguments: `{"arg":"val"}`},
			},
			expectedRemaining: "Thinking...",
		},
		{
			name: "Thought and Action without colon on Action",
			text: `Thought: Thinking...
Action tool_name(arg="val")`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "tool_name", Arguments: `{"arg":"val"}`},
			},
			expectedRemaining: "Thinking...",
		},
		{
			name: "Action with float arguments",
			text: `Action: plot_point(x=10.5, y=20.25, z=-3.0)`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "plot_point", Arguments: `{"x":10.5, "y":20.25, "z":-3.0}`},
			},
			expectedRemaining: "",
		},
		{
			name: "Action with string looking like number, but quoted",
			text: `Action: set_id(id_val="12345")`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "set_id", Arguments: `{"id_val":"12345"}`},
			},
			expectedRemaining: "",
		},
		{
			name: "Action with boolean arguments",
			text: `Action: configure(enable=true, disable=false, verbose=true)`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "configure", Arguments: `{"enable":true, "disable":false, "verbose":true}`},
			},
			expectedRemaining: "",
		},
		{
			name: "Action with mixed quoted strings, numbers, booleans in args",
			text: `Action: complex_action(name="test", count=123, active=true, ratio=0.5, id="item-001")`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "complex_action", Arguments: `{"name":"test", "count":123, "active":true, "ratio":0.5, "id":"item-001"}`},
			},
			expectedRemaining: "",
		},
		{
			name: "Action with array containing mixed types including bool and string",
			text: `Action: process_items(items=[1, "text", true, 2.5, false, "another"])`,
			expectedToolCalls: []llminterface.ToolCall{
				{Name: "process_items", Arguments: `{"items":[1, "text", true, 2.5, false, "another"]}`},
			},
			expectedRemaining: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolCalls, remaining := parser.ParseToolCalls(tt.text)

			assert.Equal(t, len(tt.expectedToolCalls), len(toolCalls), "Number of tool calls did not match")

			for i, tc := range toolCalls {
				assertToolIDFormat(t, tc.ID)
				assert.Equal(t, tt.expectedToolCalls[i].Name, tc.Name, "Tool call name did not match")
				if len(toolCalls) > i && tt.expectedToolCalls[i].Arguments != "" {
					assert.JSONEq(t, tt.expectedToolCalls[i].Arguments, tc.Arguments, "Tool call arguments did not match JSON")
				} else if len(toolCalls) > i {
					assert.Equal(t, tt.expectedToolCalls[i].Arguments, tc.Arguments, "Tool call arguments should be empty or match if expected empty")
				}
			}

			// 修订后的剩余文本断言，用于GuiAgentFormatParser
			if len(tt.expectedToolCalls) == 0 && tt.expectedRemaining == tt.text {
				// 如果预期没有工具调用，且剩余文本与原始文本相同，则直接断言
				assert.Equal(t, tt.expectedRemaining, remaining, "Remaining text did not match (no tool calls, full text)")
			} else if len(tt.expectedToolCalls) > 0 && strings.HasPrefix(tt.text, "Thought:") {
				// 如果工具调用被生成，且原始文本以 "Thought:" 开头，则假设剩余文本是提取的思考。
				assert.Equal(t, tt.expectedRemaining, remaining, "Remaining thought did not match")
			} else if len(tt.expectedToolCalls) > 0 && !strings.HasPrefix(tt.text, "Thought:") {
				// 如果工具调用被生成，且没有 "Thought:" 前缀，剩余文本应为空（或解析器留下的内容）
				assert.Equal(t, tt.expectedRemaining, remaining, "Remaining text (action, no thought) did not match")
			} else {
				// 默认/回退情况，用于其他没有工具调用的情况
				assert.Equal(t, tt.expectedRemaining, remaining, "Remaining text did not match (general case)")
			}
		})
	}
	t.Run("ExceptFormat", func(t *testing.T) {
		expected := "Thought: ...\nAction: action_name(arg1=\"value1\", arg2=\"value2\")" + "\nExample: Thought: ...\nAction: action_name(arg1=\"value1\", arg2=\"value2\")"
		assert.Equal(t, expected, parser.ExceptFormat())
	})
}

func TestMain(m *testing.M) {
	m.Run()
}

// 辅助函数，检查工具调用ID是否符合预期格式
func assertToolIDFormat(t *testing.T, id string) {
	t.Helper()
	// Example ID: tool-0-150405.000
	parts := strings.Split(id, "-")
	assert.Len(t, parts, 3, "Tool ID should have 3 parts")
	assert.Equal(t, "tool", parts[0], "Tool ID should start with 'tool'")
	_, err := time.Parse("150405.000", parts[2])
	assert.NoError(t, err, "Tool ID time part should be in '150405.000' format")
}
