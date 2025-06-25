package humanintool

import (
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// NewHumanInTools 创建人类介入工具的实例
func NewHumanInTools(i18nMgr *i18n.Manager) []toolcore.Tool {
	return []toolcore.Tool{
		NewHumanInTool(i18nMgr),
	}
}
