package admintool

import (
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// NewAdminTools 创建所有运维工具的实例
func NewAdminTools(i18nMgr *i18n.Manager) []toolcore.Tool {
	return []toolcore.Tool{
		NewSyslogTool(i18nMgr),
		NewSysstatTool(i18nMgr),
		NewProcInfoTool(i18nMgr),
		NewFileTool(i18nMgr),
		NewArchiveTool(i18nMgr),
	}
}
