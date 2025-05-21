package common

import (
	"context"

	"github.com/m4n5ter/another-me/pkg/i18n"
	. "github.com/m4n5ter/another-me/pkg/option"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// CreateParamDef 创建参数定义
func CreateParamDef(ctx context.Context, i18nMgr *i18n.Manager, name string, paramType toolcore.ParameterType, required bool, enumValues Option[[]any], descKey string, arrayItemType Option[toolcore.ParameterDefinition]) toolcore.ParameterDefinition {
	langs := i18nMgr.GetSupportedLanguages()
	descriptions := make(map[string]string, len(langs))

	for _, lang := range langs {
		langCtx := i18n.ContextWithLanguage(ctx, lang)
		descriptions[lang] = i18nMgr.T(langCtx, descKey, nil)
	}

	return toolcore.ParameterDefinition{
		Name:        name,
		Type:        paramType,
		Description: descriptions,
		Required:    required,
		EnumValues:  enumValues,
		Items:       arrayItemType,
	}
}
