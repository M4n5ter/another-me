package browsertool

import (
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BrowserType 表示浏览器类型
type BrowserType string

const (
	// Chrome 浏览器
	Chrome BrowserType = "chrome"
	// Chromium 浏览器
	Chromium BrowserType = "chromium"
	// Edge 浏览器
	Edge BrowserType = "edge"
	// Brave 浏览器
	Brave BrowserType = "brave"
)

// BrowserInfo 包含浏览器的信息
type BrowserInfo struct {
	Type BrowserType // 浏览器类型
	Path string      // 浏览器可执行文件的路径
}

// FindBrowser 搜索系统上可用的浏览器
// 按照优先级顺序返回第一个找到的浏览器
func FindBrowser(logger *slog.Logger) *BrowserInfo {
	var searchPaths []BrowserInfo

	switch runtime.GOOS {
	case "darwin":
		searchPaths = []BrowserInfo{
			{Type: Chrome, Path: "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"},
			{Type: Chrome, Path: "/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary"},
			{Type: Chromium, Path: "/Applications/Chromium.app/Contents/MacOS/Chromium"},
			{Type: Edge, Path: "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"},
			{Type: Brave, Path: "/Applications/Brave Browser.app/Contents/MacOS/Brave Browser"},
		}
	case "windows":
		// Windows上的常见安装路径
		// 将检查Program Files和Program Files (x86)
		programFiles := os.Getenv("ProgramFiles")
		programFilesX86 := os.Getenv("ProgramFiles(x86)")

		if programFiles != "" {
			searchPaths = append(searchPaths, []BrowserInfo{
				{Type: Chrome, Path: filepath.Join(programFiles, "Google/Chrome/Application/chrome.exe")},
				{Type: Edge, Path: filepath.Join(programFiles, "Microsoft/Edge/Application/msedge.exe")},
				{Type: Brave, Path: filepath.Join(programFiles, "BraveSoftware/Brave-Browser/Application/brave.exe")},
			}...)
		}

		if programFilesX86 != "" {
			searchPaths = append(searchPaths, []BrowserInfo{
				{Type: Chrome, Path: filepath.Join(programFilesX86, "Google/Chrome/Application/chrome.exe")},
				{Type: Edge, Path: filepath.Join(programFilesX86, "Microsoft/Edge/Application/msedge.exe")},
				{Type: Brave, Path: filepath.Join(programFilesX86, "BraveSoftware/Brave-Browser/Application/brave.exe")},
			}...)
		}
	case "linux":
		// 在Linux上使用which命令查找安装的浏览器
		possibleBrowsers := []struct {
			Type BrowserType
			Cmds []string
		}{
			{Type: Chrome, Cmds: []string{"google-chrome", "google-chrome-stable"}},
			{Type: Chromium, Cmds: []string{"chromium", "chromium-browser"}},
			{Type: Edge, Cmds: []string{"microsoft-edge"}},
			{Type: Brave, Cmds: []string{"brave-browser"}},
		}

		for _, browser := range possibleBrowsers {
			for _, cmd := range browser.Cmds {
				if path, err := exec.LookPath(cmd); err == nil {
					searchPaths = append(searchPaths, BrowserInfo{Type: browser.Type, Path: path})
				}
			}
		}
	}

	// 检查是否有浏览器存在
	for _, browser := range searchPaths {
		if _, err := os.Stat(browser.Path); err == nil {
			logger.Info("找到可用浏览器", "type", browser.Type, "path", browser.Path)
			return &browser
		}
	}

	// 如果没有在预定义位置找到，尝试从环境变量PATH中查找Chrome
	browsers := map[BrowserType][]string{
		Chrome:   {"chrome", "google-chrome", "google-chrome-stable"},
		Chromium: {"chromium", "chromium-browser"},
		Edge:     {"msedge", "microsoft-edge"},
		Brave:    {"brave", "brave-browser"},
	}

	for browserType, cmds := range browsers {
		for _, cmd := range cmds {
			if path, err := exec.LookPath(cmd); err == nil {
				logger.Info("从PATH中找到浏览器", "type", browserType, "command", cmd, "path", path)
				return &BrowserInfo{Type: browserType, Path: path}
			}
		}
	}

	logger.Warn("未找到任何支持的浏览器")
	return nil
}

// GetChromedpExecPath 根据浏览器信息返回ChromeDP可用的执行路径
func GetChromedpExecPath(browser *BrowserInfo) string {
	if browser == nil {
		return ""
	}

	path := browser.Path

	// Windows路径处理
	if runtime.GOOS == "windows" {
		// 确保路径格式正确
		path = strings.ReplaceAll(path, `\`, `\\`)
	}

	return path
}
