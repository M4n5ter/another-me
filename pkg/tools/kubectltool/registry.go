package kubectltool

import (
	"github.com/m4n5ter/another-me/pkg/i18n"
	"github.com/m4n5ter/another-me/pkg/toolcore"
)

// NewKubeTools 创建浏览器工具的实例
func NewKubeTools(i18nMgr *i18n.Manager) []toolcore.Tool {
	return []toolcore.Tool{
		NewKubectlTool(i18nMgr),
	}
}
