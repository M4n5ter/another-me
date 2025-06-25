package i18n

import (
	"io/fs"
	"log/slog"
	"os"
)

// GlobalManager 是一个全局的 Manager 实例，方便在应用各处直接调用。
var GlobalManager *Manager

func init() {
	var err error
	// localeFS 是在 manager.go 中定义的包级变量，内嵌了 locales 目录。
	// 它内部的路径是 "locales/en.json" 等。
	// NewManager 期望的 fsys 是直接包含 en.json, zh.json 的目录。
	actualLocalesFS, errSub := fs.Sub(localeFS, "locales")
	if errSub != nil {
		slog.Error("i18n: 初始化全局 GlobalManager 失败，无法获取 locales 子文件系统", "错误", errSub)
		os.Exit(1)
	}

	GlobalManager, err = NewManager(actualLocalesFS, "en")
	if err != nil {
		slog.Error("i18n: 初始化全局 GlobalManager 失败", "错误", err)
		os.Exit(1)
	}
	// 全局 Manager 初始化成功，NewManager 内部已有日志记录。
}
