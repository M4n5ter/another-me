package browsertool

import (
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// NewBrowserTools 创建浏览器工具的实例
func NewBrowserTools(i18nMgr *i18n.Manager) []toolcore.Tool {
	return []toolcore.Tool{
		NewBrowserTool(i18nMgr),
	}
}
