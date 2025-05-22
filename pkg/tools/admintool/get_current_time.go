package admintool

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// TimeTool 提供获取当前时间的功能
type TimeTool struct {
	i18nMgr *i18n.Manager
	logger  *slog.Logger
}

// InputGetCurrentTime 定义了 get_current_time 的输入参数
type InputGetCurrentTime struct {
	Format Option[string] `json:"format"`
}

// OutputGetTime 定义了 get_current_time 的输出参数
type OutputGetTime struct {
	CurrentTime string `json:"current_time"`
}

// NewTimeTool 创建 TimeTool 的实例
func NewTimeTool(i18nMgr *i18n.Manager) toolcore.Tool {
	return &TimeTool{i18nMgr: i18nMgr, logger: slog.Default().WithGroup("time_tool")}
}

var _ toolcore.Tool = (*TimeTool)(nil)

// Schema 实现 toolcore.Tool 接口的 Schema 方法
func (t *TimeTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	// 获取不同语言的描述文本
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.admin.time.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.admin.time.name", nil)
	}

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "format", toolcore.ParamTypeString, false, nil, "tool.admin.time.arg.format", nil),
	}

	return toolcore.ToolSchema{
		Name:             "get_current_time",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: t.createOutputParameters(ctx),
	}, nil
}

// Call 实现 toolcore.Tool 接口的 Call 方法
func (t *TimeTool) Call(ctx context.Context, inputJSON string) (outputJSON string, err error) {
	var input InputGetCurrentTime
	err = json.Unmarshal([]byte(inputJSON), &input)
	if err != nil {
		// 如果初始解析失败，并且输入不是一个简单的空对象 "{}"，则直接返回错误。
		// 只有当输入是 "{}" 并且解析它也失败时（理论上不太可能），我们才进入更深层的错误处理。
		// 这样可以确保像 `{"format":{}}` 或 `{invalid_json` 这样的错误被正确捕获并报告。
		if inputJSON != "{}" {
			t.logger.Error("无效的输入 JSON", "error", err)
			return "", fmt.Errorf("无效的输入 JSON: %w", err)
		}
		// 如果输入是 "{}" 但解析仍然失败（例如，Unmarshal 本身出现意外问题），则报告此特定错误。
		if e := json.Unmarshal([]byte("{}"), &input); e != nil {
			t.logger.Error("尝试解析 '{}' 作为备用输入时出错", "error", e, "original_error", err.Error())
			return "", fmt.Errorf("尝试解析 '{}' 作为备用输入时出错: %w (原始错误: %s)", e, err.Error())
		}
		// 如果输入是 "{}" 并且成功解析（这是预期的备用路径），err 将被重置为 nil。
		err = nil
	}

	format := defaultTimeFormat
	if input.Format.IsSome() {
		format = input.Format.Unwrap()
		if format == "" {
			format = defaultTimeFormat
		}
	}

	currentTime := time.Now().Format(format)
	output := OutputGetTime{CurrentTime: currentTime}

	outputBytes, err := json.Marshal(output)
	if err != nil {
		t.logger.Error("failed to marshal output for TimeTool", "error", err)
		return "", fmt.Errorf("failed to marshal output for TimeTool: %w", err)
	}
	return string(outputBytes), nil
}

func (t *TimeTool) createOutputParameters(_ context.Context) []toolcore.ParameterDefinition {
	currentTimeDesc := map[string]string{
		"en": "The current time",
		"zh": "当前时间",
	}

	return []toolcore.ParameterDefinition{
		{
			Name:        "current_time",
			Type:        toolcore.ParamTypeString,
			Description: currentTimeDesc,
			Required:    true,
		},
	}
}

const defaultTimeFormat = time.RFC3339
