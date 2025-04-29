package locale

import (
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
