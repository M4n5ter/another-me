package i18n

import (
	"context"
	"io/fs"
	"sort"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 创建测试用的翻译文件系统
func createTestFS(t *testing.T, files map[string]string) fs.FS {
	t.Helper()

	testFS := fstest.MapFS{}
	for name, content := range files {
		testFS[name] = &fstest.MapFile{
			Data: []byte(content),
			Mode: 0644,
		}
	}

	return testFS
}

// TestNewManager_Success 测试 NewManager 成功初始化的场景
func TestNewManager_Success(t *testing.T) {
	// 创建一个包含 en.json 和 zh.json 的测试文件系统
	files := map[string]string{
		"en.json": `{
			"greeting": "Hello",
			"farewell": "Goodbye"
		}`,
		"zh.json": `{
			"greeting": "你好",
			"farewell": "再见"
		}`,
	}
	testFS := createTestFS(t, files)

	// 测试创建 Manager
	m, err := NewManager(testFS, "en")
	require.NoError(t, err, "NewManager 应成功创建")
	require.NotNil(t, m, "创建的 Manager 不应为 nil")

	// 验证默认语言
	assert.Equal(t, "en", m.defaultLanguage, "Manager 的默认语言应为 en")

	// 验证支持的语言
	supportedLangs := m.GetSupportedLanguages()
	sort.Strings(supportedLangs)
	expectedLangs := []string{"en", "zh"}
	sort.Strings(expectedLangs)
	assert.Equal(t, expectedLangs, supportedLangs, "Manager 应支持 en 和 zh")

	// 验证 Manager 能正确加载翻译
	enCtx := ContextWithLanguage(context.Background(), "en")
	zhCtx := ContextWithLanguage(context.Background(), "zh")

	assert.Equal(t, "Hello", m.T(enCtx, "greeting", nil), "应返回英文 greeting")
	assert.Equal(t, "你好", m.T(zhCtx, "greeting", nil), "应返回中文 greeting")

	// 测试使用其他默认语言
	m2, err := NewManager(testFS, "zh")
	require.NoError(t, err, "使用 zh 作为默认语言创建 Manager 应成功")
	assert.Equal(t, "zh", m2.defaultLanguage, "Manager 的默认语言应为 zh")
}

// TestNewManager_NoLocalesLoaded 测试没有任何翻译文件加载时 NewManager 的行为
func TestNewManager_NoLocalesLoaded(t *testing.T) {
	// 案例1：空的文件系统
	emptyFS := createTestFS(t, map[string]string{})

	// 创建 Manager 应该失败，但错误消息是关于找不到目录
	m, err := NewManager(emptyFS, "en")
	assert.Error(t, err, "在没有任何翻译文件的情况下，NewManager 应返回错误")
	assert.Nil(t, m, "在没有任何翻译文件的情况下，Manager 应为 nil")
	assert.Contains(t, err.Error(), "没有加载任何翻译", "错误信息应包含 '没有加载任何翻译'")

	// 案例2：包含 locales 目录但没有任何 .json 文件
	nonJSONFS := createTestFS(t, map[string]string{
		"notes.txt": "some notes",
	})

	m, err = NewManager(nonJSONFS, "en")
	assert.Error(t, err, "当 FS 中没有 json 文件时，NewManager 应返回错误")
	assert.Nil(t, m, "当 FS 中没有 json 文件时，Manager 应为 nil")
	assert.Contains(t, err.Error(), "没有加载任何翻译", "错误信息应包含 '没有加载任何翻译'")
}

// TestNewManager_FallbackDefaultLanguage 测试当指定的默认语言不存在时，是否回退到第一个加载的语言
func TestNewManager_FallbackDefaultLanguage(t *testing.T) {
	// 创建一个仅包含 fr.json 的测试文件系统（不包含默认的 en.json）
	files := map[string]string{
		"fr.json": `{
			"greeting": "Bonjour",
			"farewell": "Au revoir"
		}`,
	}
	testFS := createTestFS(t, files)

	// 尝试使用 "en" 作为默认语言创建 Manager
	m, err := NewManager(testFS, "en")
	require.NoError(t, err, "尽管默认语言不存在，NewManager 应该能成功创建")
	require.NotNil(t, m, "创建的 Manager 不应为 nil")

	// 验证默认语言是否已回退到 "fr"
	assert.Equal(t, "fr", m.defaultLanguage, "当 en 不存在时，Manager 应回退到 fr 作为默认语言")

	// 创建一个包含多种语言但不包含默认语言的测试文件系统
	multiFiles := map[string]string{
		"fr.json": `{
			"greeting": "Bonjour",
			"farewell": "Au revoir"
		}`,
		"de.json": `{
			"greeting": "Hallo",
			"farewell": "Auf Wiedersehen"
		}`,
	}
	multiFS := createTestFS(t, multiFiles)

	// 再次尝试使用 "en" 作为默认语言创建 Manager
	m2, err := NewManager(multiFS, "en")
	require.NoError(t, err, "尽管默认语言不存在，NewManager 应该能成功创建")

	// 验证默认语言是否已回退到其中一个可用语言
	defaultLang := m2.defaultLanguage
	assert.Contains(t, []string{"fr", "de"}, defaultLang, "Manager 应将默认语言回退到 fr 或 de")

	// 验证 Manager 是否正确加载翻译
	ctx := context.Background()

	// 使用回退的默认语言
	if defaultLang == "fr" {
		assert.Equal(t, "Bonjour", m2.T(ctx, "greeting", nil), "使用回退的默认语言 fr 时应返回法语 greeting")
	} else {
		assert.Equal(t, "Hallo", m2.T(ctx, "greeting", nil), "使用回退的默认语言 de 时应返回德语 greeting")
	}
}

// TestContextHandling 测试 ContextWithLanguage 和 LanguageFromContext 函数
func TestContextHandling(t *testing.T) {
	// 测试基本设置和获取
	ctx := context.Background()
	langCtx := ContextWithLanguage(ctx, "en")
	lang := LanguageFromContext(langCtx)
	assert.Equal(t, "en", lang, "应从 context 中获取到正确的语言代码")

	// 测试大小写不敏感
	langCtxUpper := ContextWithLanguage(ctx, "EN")
	langUpper := LanguageFromContext(langCtxUpper)
	assert.Equal(t, "en", langUpper, "语言代码应转换为小写")

	// 测试空语言代码
	emptyLangCtx := ContextWithLanguage(ctx, "")
	emptyLang := LanguageFromContext(emptyLangCtx)
	assert.Equal(t, "", emptyLang, "应从 context 中获取到空语言代码")

	// 测试从未设置语言的 context 获取语言
	plainLang := LanguageFromContext(ctx)
	assert.Equal(t, "", plainLang, "从未设置语言的 context 获取语言应返回空字符串")
}

// TestManager_T_Basic 测试 T 方法的基本翻译功能
func TestManager_T_Basic(t *testing.T) {
	// 创建一个包含基本翻译的测试文件系统
	files := map[string]string{
		"en.json": `{
			"greeting": "Hello",
			"welcome": "Welcome, {name}!",
			"count": "You have {count} items"
		}`,
		"zh.json": `{
			"greeting": "你好",
			"welcome": "欢迎，{name}！",
			"count": "您有 {count} 个物品"
		}`,
	}
	testFS := createTestFS(t, files)

	m, err := NewManager(testFS, "en")
	require.NoError(t, err, "NewManager 应成功创建")

	// 创建英文和中文 context
	enCtx := ContextWithLanguage(context.Background(), "en")
	zhCtx := ContextWithLanguage(context.Background(), "zh")

	// 测试简单翻译
	assert.Equal(t, "Hello", m.T(enCtx, "greeting", nil), "应返回英文 greeting")
	assert.Equal(t, "你好", m.T(zhCtx, "greeting", nil), "应返回中文 greeting")

	// 测试带参数的翻译
	params := map[string]any{"name": "John"}
	assert.Equal(t, "Welcome, John!", m.T(enCtx, "welcome", params), "应替换英文翻译中的参数")

	params = map[string]any{"name": "张三"}
	assert.Equal(t, "欢迎，张三！", m.T(zhCtx, "welcome", params), "应替换中文翻译中的参数")

	// 测试数值参数
	countParams := map[string]any{"count": 5}
	assert.Equal(t, "You have 5 items", m.T(enCtx, "count", countParams), "应替换英文翻译中的数值参数")
	assert.Equal(t, "您有 5 个物品", m.T(zhCtx, "count", countParams), "应替换中文翻译中的数值参数")
}

// TestManager_T_Fallback 测试 T 方法的语言回退逻辑
func TestManager_T_Fallback(t *testing.T) {
	// 创建测试文件系统，其中某些键在某语言中缺失，需要回退到默认语言
	files := map[string]string{
		"en.json": `{
			"common": "Common String",
			"only_in_en": "Only in English"
		}`,
		"zh.json": `{
			"common": "通用字符串",
			"only_in_zh": "仅在中文中"
		}`,
	}
	testFS := createTestFS(t, files)

	// 使用英文作为默认语言创建 Manager
	m, err := NewManager(testFS, "en")
	require.NoError(t, err, "NewManager 应成功创建")

	// 创建英文和中文 context
	enCtx := ContextWithLanguage(context.Background(), "en")
	zhCtx := ContextWithLanguage(context.Background(), "zh")

	// 测试从非默认语言到默认语言的回退 (应该执行回退)
	assert.Equal(t, "Only in English", m.T(zhCtx, "only_in_en", nil),
		"当中文翻译缺失时，应回退到英文（默认语言）")

	// 测试默认语言缺失键的行为 (不会回退到其他语言)
	result := m.T(enCtx, "only_in_zh", nil)
	assert.Contains(t, result, "翻译缺失",
		"当默认语言中键缺失时，不应回退到其他语言，而应返回缺失提示")
	assert.Contains(t, result, "only_in_zh",
		"返回的缺失提示应包含键名")

	// 使用中文作为默认语言创建另一个 Manager
	m2, err := NewManager(testFS, "zh")
	require.NoError(t, err, "使用中文作为默认语言的 NewManager 应成功创建")

	// 测试回退逻辑（现在默认语言是中文）
	// 默认语言中键缺失 (不会回退)
	result = m2.T(zhCtx, "only_in_en", nil)
	assert.Contains(t, result, "翻译缺失",
		"当默认语言中键缺失时，不应回退到其他语言，而应返回缺失提示")

	// 从非默认语言到默认语言的回退
	assert.Equal(t, "仅在中文中", m2.T(enCtx, "only_in_zh", nil),
		"当英文翻译缺失时，应回退到中文（当前的默认语言）")
}

// TestManager_T_MissingKeys 测试当键在所有语言中都缺失时的行为
func TestManager_T_MissingKeys(t *testing.T) {
	// 创建测试文件系统
	files := map[string]string{
		"en.json": `{
			"existing": "Existing String"
		}`,
		"zh.json": `{
			"existing": "已存在的字符串"
		}`,
	}
	testFS := createTestFS(t, files)

	m, err := NewManager(testFS, "en")
	require.NoError(t, err, "NewManager 应成功创建")

	// 创建英文和中文 context
	enCtx := ContextWithLanguage(context.Background(), "en")
	zhCtx := ContextWithLanguage(context.Background(), "zh")

	// 测试缺失的键
	missingKey := "non_existent_key"
	enResult := m.T(enCtx, missingKey, nil)
	zhResult := m.T(zhCtx, missingKey, nil)

	assert.Contains(t, enResult, "翻译缺失", "当键在所有语言中都缺失时，应返回带有提示的字符串")
	assert.Contains(t, enResult, missingKey, "返回的字符串应包含缺失的键")
	assert.Contains(t, zhResult, "翻译缺失", "当键在所有语言中都缺失时，应返回带有提示的字符串")
	assert.Contains(t, zhResult, missingKey, "返回的字符串应包含缺失的键")

	// 测试带参数的缺失键
	params := map[string]any{"name": "Test"}
	paramResult := m.T(enCtx, missingKey, params)
	assert.Contains(t, paramResult, "翻译缺失", "当键在所有语言中都缺失时，应返回带有提示的字符串")
	assert.Contains(t, paramResult, missingKey, "返回的字符串应包含缺失的键")
	assert.Contains(t, paramResult, "Test", "返回的字符串应包含参数值")
}

// TestManager_T_UnsupportedLangInContext 测试当 context 中的语言不受支持时的行为
func TestManager_T_UnsupportedLangInContext(t *testing.T) {
	// 创建测试文件系统
	files := map[string]string{
		"en.json": `{
			"greeting": "Hello"
		}`,
		"zh.json": `{
			"greeting": "你好"
		}`,
	}
	testFS := createTestFS(t, files)

	m, err := NewManager(testFS, "en")
	require.NoError(t, err, "NewManager 应成功创建")

	// 创建包含不支持语言的 context
	unsupportedLangCtx := ContextWithLanguage(context.Background(), "fr")

	// 测试应该回退到默认语言
	assert.Equal(t, "Hello", m.T(unsupportedLangCtx, "greeting", nil),
		"当 context 中的语言不受支持时，应使用默认语言")
}

// TestManager_GetSupportedLanguages 测试 GetSupportedLanguages 方法
func TestManager_GetSupportedLanguages(t *testing.T) {
	// 创建测试文件系统
	files := map[string]string{
		"en.json": `{}`,
		"zh.json": `{}`,
		"fr.json": `{}`,
	}
	testFS := createTestFS(t, files)

	m, err := NewManager(testFS, "en")
	require.NoError(t, err, "NewManager 应成功创建")

	// 获取并验证支持的语言
	langs := m.GetSupportedLanguages()
	sort.Strings(langs)

	expected := []string{"en", "fr", "zh"}
	assert.Equal(t, expected, langs, "GetSupportedLanguages 应返回所有已加载的语言")
}
