package admintool

import (
	"context"
	"strings"
	"testing"
	"time"

	json "github.com/json-iterator/go"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

func TestTimeTool_Schema(t *testing.T) {
	tool := NewTimeTool(i18n.GlobalManager)

	schema, err := tool.Schema(context.Background())
	require.NoError(t, err)

	assert.Equal(t, "get_current_time", schema.Name)
	assert.NotEmpty(t, schema.LocalizedNames["en"])
	assert.NotEmpty(t, schema.LocalizedNames["zh"])
	assert.NotEmpty(t, schema.Descriptions["en"])
	assert.NotEmpty(t, schema.Descriptions["zh"])

	require.Len(t, schema.InputParameters, 1)
	assert.Equal(t, "format", schema.InputParameters[0].Name)
	assert.Equal(t, toolcore.ParamTypeString, schema.InputParameters[0].Type)
	assert.False(t, schema.InputParameters[0].Required)

	require.Len(t, schema.OutputParameters, 1)
	assert.Equal(t, "current_time", schema.OutputParameters[0].Name)
	assert.Equal(t, toolcore.ParamTypeString, schema.OutputParameters[0].Type)
	assert.True(t, schema.OutputParameters[0].Required)
}

func TestTimeTool_Call(t *testing.T) {
	tool := NewTimeTool(i18n.GlobalManager)
	ctx := context.Background()

	tests := []struct {
		name           string
		inputJSON      string
		expectError    bool
		validateOutput func(t *testing.T, outputJSON string)
	}{
		{
			name:        "默认格式",
			inputJSON:   `{}`, // 未指定格式
			expectError: false,
			validateOutput: func(t *testing.T, outputJSON string) {
				var out OutputGetTime
				err := json.Unmarshal([]byte(outputJSON), &out)
				require.NoError(t, err)
				_, err = time.Parse(defaultTimeFormat, out.CurrentTime)
				assert.NoError(t, err, "输出时间应为默认格式")
			},
		},
		{
			name:        "自定义格式 RFC822",
			inputJSON:   `{"format": "` + time.RFC822 + `"}`,
			expectError: false,
			validateOutput: func(t *testing.T, outputJSON string) {
				var out OutputGetTime
				err := json.Unmarshal([]byte(outputJSON), &out)
				require.NoError(t, err)
				_, err = time.Parse(time.RFC822, out.CurrentTime)
				assert.NoError(t, err, "输出时间应为 RFC822 格式")
			},
		},
		{
			name:        "自定义格式 Kitchen",
			inputJSON:   `{"format": "` + time.Kitchen + `"}`,
			expectError: false,
			validateOutput: func(t *testing.T, outputJSON string) {
				var out OutputGetTime
				err := json.Unmarshal([]byte(outputJSON), &out)
				require.NoError(t, err)
				// Kitchen 格式的时间解析可能比较棘手，因为它可能不包含日期上下文的 AM/PM
				// 我们将检查它是否包含 AM 或 PM 作为基本验证
				assert.True(t, strings.Contains(out.CurrentTime, "AM") || strings.Contains(out.CurrentTime, "PM"), "Kitchen 格式的输出时间应包含 AM 或 PM")
			},
		},
		{
			name:        "无效的输入 JSON",
			inputJSON:   `{invalid_json`,
			expectError: true,
		},
		{
			name:        "空的输入 JSON 对象作为格式",
			inputJSON:   `{"format": {}}`,
			expectError: true, // 格式应为字符串
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outputJSON, err := tool.Call(ctx, tc.inputJSON)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				tc.validateOutput(t, outputJSON)
			}
		})
	}
}
