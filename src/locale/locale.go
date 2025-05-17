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

// --- GUI 工具 ---

func ScreenshotDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScreenshotDescription
	case LocaleEN:
		return en.ScreenshotDescription
	default:
		return zh.ScreenshotDescription
	}
}

func MoveMouseDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.MoveMouseDescription
	case LocaleEN:
		return en.MoveMouseDescription
	default:
		return zh.MoveMouseDescription
	}
}

func MoveMouseArgXDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.MoveMouseArgXDescription
	case LocaleEN:
		return en.MoveMouseArgXDescription
	default:
		return zh.MoveMouseArgXDescription
	}
}

func MoveMouseArgYDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.MoveMouseArgYDescription
	case LocaleEN:
		return en.MoveMouseArgYDescription
	default:
		return zh.MoveMouseArgYDescription
	}
}

func MouseLocationDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.MouseLocationDescription
	case LocaleEN:
		return en.MouseLocationDescription
	default:
		return zh.MouseLocationDescription
	}
}

func DragDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.DragDescription
	case LocaleEN:
		return en.DragDescription
	default:
		return zh.DragDescription
	}
}

func DragArgXDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.DragArgXDescription
	case LocaleEN:
		return en.DragArgXDescription
	default:
		return zh.DragArgXDescription
	}
}

func DragArgYDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.DragArgYDescription
	case LocaleEN:
		return en.DragArgYDescription
	default:
		return zh.DragArgYDescription
	}
}

func ScrollDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollDescription
	case LocaleEN:
		return en.ScrollDescription
	default:
		return zh.ScrollDescription
	}
}

func ScrollArgToyDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollArgToyDescription
	case LocaleEN:
		return en.ScrollArgToyDescription
	default:
		return zh.ScrollArgToyDescription
	}
}

func ScrollArgNumDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollArgNumDescription
	case LocaleEN:
		return en.ScrollArgNumDescription
	default:
		return zh.ScrollArgNumDescription
	}
}

func ScrollArgMsSleepDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollArgMsSleepDescription
	case LocaleEN:
		return en.ScrollArgMsSleepDescription
	default:
		return zh.ScrollArgMsSleepDescription
	}
}

func ScrollArgToxDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollArgToxDescription
	case LocaleEN:
		return en.ScrollArgToxDescription
	default:
		return zh.ScrollArgToxDescription
	}
}

func ScrollRelativeDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollRelativeDescription
	case LocaleEN:
		return en.ScrollRelativeDescription
	default:
		return zh.ScrollRelativeDescription
	}
}

func ScrollRelativeArgXDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollRelativeArgXDescription
	case LocaleEN:
		return en.ScrollRelativeArgXDescription
	default:
		return zh.ScrollRelativeArgXDescription
	}
}

func ScrollRelativeArgYDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollRelativeArgYDescription
	case LocaleEN:
		return en.ScrollRelativeArgYDescription
	default:
		return zh.ScrollRelativeArgYDescription
	}
}

func ScrollRelativeArgMsDeplayDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollRelativeArgMsDeplayDescription
	case LocaleEN:
		return en.ScrollRelativeArgMsDeplayDescription
	default:
		return zh.ScrollRelativeArgMsDeplayDescription
	}
}

func ScrollDirectionDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollDirectionDescription
	case LocaleEN:
		return en.ScrollDirectionDescription
	default:
		return zh.ScrollDirectionDescription
	}
}

func ScrollDirectionArgXDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollDirectionArgXDescription
	case LocaleEN:
		return en.ScrollDirectionArgXDescription
	default:
		return zh.ScrollDirectionArgXDescription
	}
}

func ScrollDirectionArgDirectionDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ScrollDirectionArgDirectionDescription
	case LocaleEN:
		return en.ScrollDirectionArgDirectionDescription
	default:
		return zh.ScrollDirectionArgDirectionDescription
	}
}

func ClickDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ClickDescription
	case LocaleEN:
		return en.ClickDescription
	default:
		return zh.ClickDescription
	}
}

func ClickArgButtonDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ClickArgButtonDescription
	case LocaleEN:
		return en.ClickArgButtonDescription
	default:
		return zh.ClickArgButtonDescription
	}
}

func ClickArgDoubleDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ClickArgDoubleDescription
	case LocaleEN:
		return en.ClickArgDoubleDescription
	default:
		return zh.ClickArgDoubleDescription
	}
}

func ToggleMouseButtonDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ToggleMouseButtonDescription
	case LocaleEN:
		return en.ToggleMouseButtonDescription
	default:
		return zh.ToggleMouseButtonDescription
	}
}

func ToggleMouseButtonArgButtonDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ToggleMouseButtonArgButtonDescription
	case LocaleEN:
		return en.ToggleMouseButtonArgButtonDescription
	default:
		return zh.ToggleMouseButtonArgButtonDescription
	}
}

func ToggleMouseButtonArgUpDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ToggleMouseButtonArgUpDescription
	case LocaleEN:
		return en.ToggleMouseButtonArgUpDescription
	default:
		return zh.ToggleMouseButtonArgUpDescription
	}
}

func ToggleKeyDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ToggleKeyDescription
	case LocaleEN:
		return en.ToggleKeyDescription
	default:
		return zh.ToggleKeyDescription
	}
}

func ToggleKeyArgUpDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ToggleKeyArgUpDescription
	case LocaleEN:
		return en.ToggleKeyArgUpDescription
	default:
		return zh.ToggleKeyArgUpDescription
	}
}

func ToggleKeyArgKeysDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.ToggleKeyArgKeysDescription
	case LocaleEN:
		return en.ToggleKeyArgKeysDescription
	default:
		return zh.ToggleKeyArgKeysDescription
	}
}

func KeyTapDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.KeyTapDescription
	case LocaleEN:
		return en.KeyTapDescription
	default:
		return zh.KeyTapDescription
	}
}

func KeyTapArgKeysDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.KeyTapArgKeysDescription
	case LocaleEN:
		return en.KeyTapArgKeysDescription
	default:
		return zh.KeyTapArgKeysDescription
	}
}

func KeySleepMilliDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.KeySleepMilliDescription
	case LocaleEN:
		return en.KeySleepMilliDescription
	default:
		return zh.KeySleepMilliDescription
	}
}

func KeySleepMilliArgMsDescription() string {
	locale := GetLocale()
	switch locale {
	case LocaleZH:
		return zh.KeySleepMilliArgMsDescription
	case LocaleEN:
		return en.KeySleepMilliArgMsDescription
	default:
		return zh.KeySleepMilliArgMsDescription
	}
}
