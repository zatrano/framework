package notification

import (
	"sync"
)

// StringTranslator resolves localization keys (compatible with packages/localization.Translator).
type StringTranslator interface {
	Get(key string, replace ...map[string]string) string
	GetFor(locale, key string, replace ...map[string]string) string
}

var (
	mailMu         sync.RWMutex
	mailTranslator StringTranslator
	mailLocale     string
	mailAppName    string
)

// SetTranslator configures the translator used by built-in auth mail notifications.
func SetTranslator(tr StringTranslator) {
	mailMu.Lock()
	defer mailMu.Unlock()
	mailTranslator = tr
}

// SetMailDefaults sets the default locale and app name for auth mail notifications.
func SetMailDefaults(locale, appName string) {
	mailMu.Lock()
	defer mailMu.Unlock()
	mailLocale = locale
	mailAppName = appName
}

// SetTranslator attaches a translator used by built-in auth emails (and package defaults).
func (m *Manager) SetTranslator(tr StringTranslator) {
	if m == nil {
		return
	}
	SetTranslator(tr)
}

// SetMailDefaults stores locale/app name used when building localized auth emails.
func (m *Manager) SetMailDefaults(locale, appName string) {
	if m == nil {
		return
	}
	SetMailDefaults(locale, appName)
}

func mailTranslate(locale, key string, replace map[string]string) string {
	mailMu.RLock()
	tr, defLocale := mailTranslator, mailLocale
	mailMu.RUnlock()
	if tr == nil {
		return key
	}
	loc := locale
	if loc == "" {
		loc = defLocale
	}
	if loc != "" {
		return tr.GetFor(loc, key, replace)
	}
	return tr.Get(key, replace)
}

func resolveMailAppName(override string) string {
	if override != "" {
		return override
	}
	mailMu.RLock()
	defer mailMu.RUnlock()
	if mailAppName != "" {
		return mailAppName
	}
	return "App"
}
