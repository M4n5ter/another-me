package i18n

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"sync"
)

//go:embed all:locales
var localeFS embed.FS // localeFS 内嵌了 locales 目录下的所有翻译文件。

// langCtxKey 是用于在 context.Context 中存储和检索语言代码的私有类型键。
type langCtxKeyType struct{}

var langCtxKey = langCtxKeyType{} // langCtxKey 的实例。

// Manager 结构体负责管理国际化翻译的加载和检索。
type Manager struct {
	mu              sync.RWMutex                 // mu 用于保护对 translations 和 supportedLangs 的并发访问。
	translations    map[string]map[string]string // translations 存储所有加载的翻译，格式为：语言代码 -> 翻译键 -> 翻译文本。
	defaultLanguage string                       // defaultLanguage 是当请求的语言或键不存在时的回退语言。
	supportedLangs  map[string]struct{}          // supportedLangs 是一个集合，存储所有已加载并支持的语言代码。
}

// NewManager 创建并初始化一个新的 Manager 实例。
// fsys 参数是包含本地化文件的文件系统，例如 embed.FS。
// defaultLang 参数指定默认的回退语言代码（例如 "en", "zh"）。
// 它会尝试从 fsys 的 "locales" 目录加载所有 .json 格式的翻译文件。
func NewManager(fsys fs.FS, defaultLang string) (*Manager, error) {
	// 初始化 Manager 结构体
	m := &Manager{
		translations:    make(map[string]map[string]string),
		defaultLanguage: strings.ToLower(defaultLang), // 语言代码统一转换为小写处理
		supportedLangs:  make(map[string]struct{}),
	}

	// 从提供的文件系统加载翻译
	if err := m.loadTranslationsFromFS(fsys, "."); err != nil {
		return nil, fmt.Errorf("i18n: 初始化失败，加载翻译时出错: %w", err)
	}

	// 检查默认语言是否已成功加载
	if _, ok := m.supportedLangs[m.defaultLanguage]; !ok {
		if len(m.supportedLangs) > 0 {
			// 如果指定的默认语言未加载，但有其他语言已加载，则选择第一个加载的语言作为新的默认语言。
			for lang := range m.supportedLangs {
				slog.Warn("i18n: 指定的默认语言未找到，将使用第一个加载的语言作为回退",
					"原始默认语言", defaultLang, "新的默认语言", lang)
				m.defaultLanguage = lang
				break
			}
		} else {
			// 如果没有任何语言被加载，则返回错误。
			return nil, fmt.Errorf("i18n: 初始化失败，没有加载任何翻译，且默认语言 '%s' 不可用", defaultLang)
		}
	}

	slog.Info("i18n: Manager 初始化成功",
		"默认语言", m.defaultLanguage,
		"支持的语言", m.GetSupportedLanguages())
	return m, nil
}

// loadTranslationsFromFS 从给定的 fsys 文件系统中加载指定目录下的所有 .json 翻译文件。
// 文件名（去除 .json 后缀）被用作语言代码。
func (m *Manager) loadTranslationsFromFS(fsys fs.FS, dirPath string) error {
	m.mu.Lock() // 写锁定，因为会修改 m.translations 和 m.supportedLangs
	defer m.mu.Unlock()

	entries, err := fs.ReadDir(fsys, dirPath) // 使用传入的 fsys 和 dirPath
	if err != nil {
		return fmt.Errorf("无法读取目录 %s: %w", dirPath, err)
	}

	loadedAny := false // 标记是否成功加载了至少一个翻译文件
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			lang := strings.ToLower(strings.TrimSuffix(entry.Name(), ".json")) // 从文件名提取语言代码
			filePath := entry.Name()                                            // 直接使用 entry.Name() 作为路径
			if dirPath != "." { // 如果 dirPath 不是根目录，则拼接
				filePath = strings.Join([]string{dirPath, entry.Name()}, "/")
			}

			content, err := fs.ReadFile(fsys, filePath) // 使用传入的 fsys 读取文件
			if err != nil {
				slog.Error("i18n: 读取语言文件失败", "路径", filePath, "错误", err)
				continue // 跳过此文件，继续尝试加载其他文件
			}

			var langTranslations map[string]string
			if err := json.Unmarshal(content, &langTranslations); err != nil {
				slog.Error("i18n: 解析语言文件失败 (非JSON格式或格式错误)", "路径", filePath, "错误", err)
				continue // 跳过此文件
			}

			m.translations[lang] = langTranslations
			m.supportedLangs[lang] = struct{}{}
			loadedAny = true
			slog.Debug("i18n: 成功加载翻译文件", "语言", lang, "键数量", len(langTranslations))
		}
	}

	if !loadedAny {
		slog.Warn("i18n: 在 'locales' 目录中没有找到或加载任何 .json 翻译文件。")
		// 根据需求，这里可以选择返回错误，但当前设计为允许 Manager 在无翻译的情况下启动。
	}
	return nil
}

// ContextWithLanguage 返回一个新的 context，其中包含了指定的语言代码。
// langCode 会被转换为小写。
func ContextWithLanguage(ctx context.Context, langCode string) context.Context {
	return context.WithValue(ctx, langCtxKey, strings.ToLower(langCode))
}

// LanguageFromContext 从 context 中检索之前设置的语言代码。
// 如果 context 中没有设置语言代码，则返回空字符串。
func LanguageFromContext(ctx context.Context) string {
	if lang, ok := ctx.Value(langCtxKey).(string); ok {
		return lang
	}
	return ""
}

// T 是核心的翻译方法。它根据给定的键和参数，从 context 中获取当前语言设置，
// 然后查找对应的翻译文本。如果当前语言的翻译不存在，则尝试回退到默认语言。
// 支持使用 {placeholder} 格式的命名参数进行插值。
// key: 需要翻译的键，例如 "greeting.user"。
// params: 一个 map，包含需要替换到翻译文本中的参数，例如 map[string]interface{}{"name": "张三"}。
func (m *Manager) T(ctx context.Context, key string, params map[string]interface{}) string {
	m.mu.RLock() // 读锁定，因为只读取 m.translations 等字段

	currentLang := LanguageFromContext(ctx)
	// 如果 context 中没有语言或语言不被支持，则使用默认语言
	if _, supported := m.supportedLangs[currentLang]; currentLang == "" || !supported {
		currentLang = m.defaultLanguage
	}

	translationSet, langExists := m.translations[currentLang]
	// 在 RLock 作用域内完成对 m.translations 的访问
	// 如果 map 的查找结果需要在锁外使用，应复制出来

	if !langExists { // 理论上，如果 defaultLanguage 正确设置，这里不应发生
		m.mu.RUnlock() // 解锁后才能调用其他可能也带锁的方法或打日志
		slog.Error("i18n: 当前语言或默认语言的翻译集未加载", "尝试的语言", currentLang, "键", key)
		return m.formatMissingKey(key, params, currentLang, true) // 指示整个语言包缺失
	}

	format, keyExists := translationSet[key]
	m.mu.RUnlock() // 解锁，后续的字符串格式化不需要锁

	if !keyExists {
		// 如果在当前语言中未找到键，并且当前语言不是默认语言，则尝试在默认语言中查找
		if currentLang != m.defaultLanguage {
			slog.Debug("i18n: 在当前语言中未找到键，尝试默认语言",
				"键", key, "当前语言", currentLang, "默认语言", m.defaultLanguage)

			m.mu.RLock() // 重新获取读锁以访问默认语言的翻译
			defaultTranslationSet, defaultLangExists := m.translations[m.defaultLanguage]
			if defaultLangExists {
				format, keyExists = defaultTranslationSet[key]
			}
			m.mu.RUnlock()
		}

		if !keyExists {
			slog.Warn("i18n: 在任何语言（包括默认语言）中均未找到翻译键",
				"键", key, "尝试的语言", currentLang, "默认语言", m.defaultLanguage)
			return m.formatMissingKey(key, params, currentLang, false) // 指示键本身缺失
		}
	}

	// 执行命名参数替换
	// 例如，将 "Hello, {userName}!" 中的 "{userName}" 替换为 params["userName"] 的值
	// 这个替换逻辑比较简单，对于复杂的场景可能需要更健壮的模板引擎
	// 这里我们逐个替换，避免一个替换影响另一个占位符（虽然在这个简单场景下不太可能）
	tempFormat := format
	for pKey, pVal := range params {
		placeholder := fmt.Sprintf("{%s}", pKey) // Go 1.22 以下可以用 "{" + pKey + "}"
		tempFormat = strings.ReplaceAll(tempFormat, placeholder, fmt.Sprintf("%v", pVal))
	}
	return tempFormat
}

// formatMissingKey 是一个辅助方法，用于在翻译键缺失时生成提示信息。
// key: 缺失的翻译键。
// params: 传递给 T 方法的参数，也会显示在提示信息中以帮助调试。
// lang: 尝试获取翻译时使用的语言代码。
// langCompletelyMissing: 布尔值，指示是否是整个语言的翻译数据都缺失了。
func (m *Manager) formatMissingKey(key string, params map[string]interface{}, lang string, langCompletelyMissing bool) string {
	var suffix string
	if langCompletelyMissing {
		suffix = fmt.Sprintf(" (语言 '%s' 的翻译文件缺失或无法加载)", lang)
	} else {
		suffix = fmt.Sprintf(" (在语言 '%s' 中)", lang)
	}

	if len(params) > 0 {
		// 为了可读性，简单地将参数 map 格式化
		// 注意：对于复杂的参数结构，%v 可能输出很长
		return fmt.Sprintf("翻译缺失[%s, 参数:%v]%s", key, params, suffix)
	}
	return fmt.Sprintf("翻译缺失[%s]%s", key, suffix)
}

// GetSupportedLanguages 返回一个包含所有已加载并支持的语言代码的切片。
func (m *Manager) GetSupportedLanguages() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	langs := make([]string, 0, len(m.supportedLangs))
	for lang := range m.supportedLangs {
		langs = append(langs, lang)
	}
	return langs
}
