package browsertool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/m4n5ter/another-me/pkg/i18n"
)

// BrowserPromptKey 是浏览器工具的默认提示信息键
const BrowserPromptKey = "browser.prompt.default"

// BrowserConfig 定义了浏览器工具的配置项
type BrowserConfig struct {
	PromptFile           string `json:"prompt_file"` // 提示文件
	prompt               string // 提示内容
	Headless             bool   `json:"headless"`               // 是否使用无头模式
	Timeout              int    `json:"timeout"`                // 全局超时时间（秒）
	Proxy                string `json:"proxy"`                  // 代理设置
	UserAgent            string `json:"user_agent"`             // 用户代理字符串
	DefaultLanguage      string `json:"default_language"`       // 默认语言
	URLTimeout           int    `json:"url_timeout"`            // URL加载超时时间（秒）
	SelectorQueryTimeout int    `json:"selector_query_timeout"` // CSS选择器查询超时时间（秒）
	DataPath             string `json:"data_path"`              // 数据目录路径
	BrowserDataPath      string `json:"browser_data_path"`      // 浏览器数据目录路径
	WindowWidth          int    `json:"window_width"`           // 窗口宽度
	WindowHeight         int    `json:"window_height"`          // 窗口高度
	BrowserPath          string `json:"browser_path"`           // 浏览器可执行文件的路径，如果为空则自动搜索
}

// Check 检查配置是否有效
func (cfg *BrowserConfig) Check() error {
	// 检查提示文件
	if cfg.PromptFile != "" {
		if _, err := os.Stat(cfg.PromptFile); err != nil {
			return fmt.Errorf("提示文件不存在或不可访问: %w", err)
		}

		// 读取提示文件内容
		content, err := os.ReadFile(cfg.PromptFile)
		if err != nil {
			return fmt.Errorf("读取提示文件失败: %w", err)
		}
		cfg.prompt = string(content)
	} else {
		// 使用默认提示（从i18n系统获取）
		// 使用全局i18n管理器获取当前设置语言的提示
		ctx := context.Background()
		// 可以根据cfg.DefaultLanguage设置语言上下文
		if cfg.DefaultLanguage != "" {
			ctx = i18n.ContextWithLanguage(ctx, cfg.DefaultLanguage)
		}
		cfg.prompt = i18n.GlobalManager.T(ctx, BrowserPromptKey, nil)
	}

	// 确保数据目录存在
	if cfg.DataPath != "" {
		if err := os.MkdirAll(cfg.DataPath, 0o755); err != nil {
			return fmt.Errorf("创建数据目录失败: %w", err)
		}
	}

	// 确保浏览器数据目录存在
	if cfg.BrowserDataPath != "" {
		browserDataPath := filepath.Join(cfg.DataPath, cfg.BrowserDataPath)
		if err := os.MkdirAll(browserDataPath, 0o755); err != nil {
			return fmt.Errorf("创建浏览器数据目录失败: %w", err)
		}
	}

	return nil
}

// GetPrompt 获取提示内容
func (cfg *BrowserConfig) GetPrompt() string {
	return cfg.prompt
}

// NewBrowserConfig 创建一个带有默认值的浏览器配置
func NewBrowserConfig() *BrowserConfig {
	return &BrowserConfig{
		Headless:             true,
		Timeout:              60,
		UserAgent:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		DefaultLanguage:      "zh-CN,zh",
		URLTimeout:           30,
		SelectorQueryTimeout: 15,
		DataPath:             "./data",
		BrowserDataPath:      "browser",
		WindowWidth:          1920,
		WindowHeight:         1080,
	}
}
