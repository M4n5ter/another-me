package decisionmaker

import (
	"context"
	"fmt"
	"log/slog"

	json "github.com/json-iterator/go"

	"github.com/m4n5ter/another-me/internal/core/types"
	"github.com/m4n5ter/another-me/pkg/common"
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// AgentSelection Agent选择结构体
type AgentSelection struct {
	SelectedAgentType types.AgentType `json:"selected_agent_type" jsonschema:"title=selected_agent_type,description=Selected agent type,enum=gui,enum=react,default=react"`
	Reasoning         string          `json:"reasoning" jsonschema:"title=reasoning,description=Reasoning for the selection"`
	Confidence        float64         `json:"confidence" jsonschema:"title=confidence,description=Confidence level,minimum=0,maximum=1"`
}

// SelectAgentTool Agent选择工具
type SelectAgentTool struct {
	logger  *slog.Logger
	i18nMgr *i18n.Manager
}

// NewSelectAgentTool 创建Agent选择工具实例
func NewSelectAgentTool(i18nMgr *i18n.Manager) *SelectAgentTool {
	return &SelectAgentTool{
		logger:  slog.Default().WithGroup("select_agent_tool"),
		i18nMgr: i18nMgr,
	}
}

var _ toolcore.Tool = (*SelectAgentTool)(nil)

// Schema 返回工具的完整模式
func (t *SelectAgentTool) Schema(ctx context.Context) (toolcore.ToolSchema, error) {
	langs := t.i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))
	localizedNames := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = t.i18nMgr.T(langCtx, "tool.select_agent.description", nil)
		localizedNames[lang] = t.i18nMgr.T(langCtx, "tool.select_agent.name", nil)
	}

	inputParameters := []toolcore.ParameterDefinition{
		common.CreateParamDef(ctx, t.i18nMgr, "selected_agent_type", toolcore.ParamTypeString, true, nil, "tool.select_agent.arg.selected_agent_type", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "reasoning", toolcore.ParamTypeString, true, nil, "tool.select_agent.arg.reasoning", nil),
		common.CreateParamDef(ctx, t.i18nMgr, "confidence", toolcore.ParamTypeNumber, true, nil, "tool.select_agent.arg.confidence", nil),
	}

	return toolcore.ToolSchema{
		Name:             "select_agent",
		LocalizedNames:   localizedNames,
		Descriptions:     descriptions,
		InputParameters:  inputParameters,
		OutputParameters: []toolcore.ParameterDefinition{},
	}, nil
}

// Call 工具调用方法
func (t *SelectAgentTool) Call(ctx context.Context, inputJSON string) (string, error) {
	var selection AgentSelection
	if err := json.Unmarshal([]byte(inputJSON), &selection); err != nil {
		t.logger.Error("解析参数失败", "error", err, "input", inputJSON)
		return "", fmt.Errorf("无效的 JSON 输入: %w", err)
	}

	return inputJSON, nil
}
