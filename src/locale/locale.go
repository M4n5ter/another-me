package locale

import (
	"strings"
	"sync"

	"github.com/m4n5ter/another-me/src/locale/en"
	"github.com/m4n5ter/another-me/src/locale/zh"
)

type Locale uint8

const (
	LocaleZH Locale = iota
	LocaleEN
)

var (
	locale       = LocaleZH
	locale_mutex sync.RWMutex
)

func SetLocaleFromStr(str string) {
	locale_mutex.Lock()
	defer locale_mutex.Unlock()

	str = strings.ToLower(str)
	switch str {
	case "zh":
		locale = LocaleZH
	case "en":
		locale = LocaleEN
	default:
		locale = LocaleZH
	}
}

func SetLocale(l Locale) {
	locale_mutex.Lock()
	defer locale_mutex.Unlock()
	locale = l
}

func GetLocale() Locale {
	locale_mutex.RLock()
	defer locale_mutex.RUnlock()
	return locale
}

func AnotherMeSystemPrompt() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.AnotherMeSystemPrompt
	case LocaleEN:
		return en.AnotherMeSystemPrompt
	default:
		return zh.AnotherMeSystemPrompt
	}
}

// --- 任务评估工具 ---

func TaskEvaluatorDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.TaskEvaluatorDescription
	case LocaleEN:
		return en.TaskEvaluatorDescription
	default:
		return zh.TaskEvaluatorDescription
	}
}

func TaskEvaluatorArgIsCompleteDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.TaskEvaluatorArgIsCompleteDescription
	case LocaleEN:
		return en.TaskEvaluatorArgIsCompleteDescription
	default:
		return zh.TaskEvaluatorArgIsCompleteDescription
	}
}

func TaskEvaluatorArgContextDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.TaskEvaluatorArgContextDescription
	case LocaleEN:
		return en.TaskEvaluatorArgContextDescription
	default:
		return zh.TaskEvaluatorArgContextDescription
	}
}

// --- Fetch 工具 ---

func FetchToolDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolDescription
	case LocaleEN:
		return en.FetchToolDescription
	default:
		return zh.FetchToolDescription
	}
}

func FetchToolArgURLDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolArgURLDescription
	case LocaleEN:
		return en.FetchToolArgURLDescription
	default:
		return zh.FetchToolArgURLDescription
	}
}

func FetchToolArgMaxLengthDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolArgMaxLengthDescription
	case LocaleEN:
		return en.FetchToolArgMaxLengthDescription
	default:
		return zh.FetchToolArgMaxLengthDescription
	}
}

func FetchToolArgStartIndexDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolArgStartIndexDescription
	case LocaleEN:
		return en.FetchToolArgStartIndexDescription
	default:
		return zh.FetchToolArgStartIndexDescription
	}
}

func FetchToolArgRawDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolArgRawDescription
	case LocaleEN:
		return en.FetchToolArgRawDescription
	default:
		return zh.FetchToolArgRawDescription
	}
}

func FetchToolArgIgnoreRobotsTxtDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolArgIgnoreRobotsTxtDescription
	case LocaleEN:
		return en.FetchToolArgIgnoreRobotsTxtDescription
	default:
		return zh.FetchToolArgIgnoreRobotsTxtDescription
	}
}

func FetchToolArgUserAgentDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolArgUserAgentDescription
	case LocaleEN:
		return en.FetchToolArgUserAgentDescription
	default:
		return zh.FetchToolArgUserAgentDescription
	}
}

func FetchToolArgProxyURLDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolArgProxyURLDescription
	case LocaleEN:
		return en.FetchToolArgProxyURLDescription
	default:
		return zh.FetchToolArgProxyURLDescription
	}
}

func FetchToolArgIsManualRequestDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.FetchToolArgIsManualRequestDescription
	case LocaleEN:
		return en.FetchToolArgIsManualRequestDescription
	default:
		return zh.FetchToolArgIsManualRequestDescription
	}
}
