package fetchtool

import (
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// NewFetchTools 创建一个fetch工具列表
func NewFetchTools(i18nMgr *i18n.Manager) []toolcore.Tool {
	return []toolcore.Tool{
		NewFetchTool(i18nMgr),
	}
}
