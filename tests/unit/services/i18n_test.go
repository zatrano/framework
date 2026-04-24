package services_test

import (
	"testing"

	"github.com/zatrano/framework/packages/i18n"

	"github.com/stretchr/testify/assert"
)

func TestI18n_Init(t *testing.T) {
	i18n.Init()

	tr := i18n.Translate(i18n.LangTR, "auth.login_success")
	assert.NotEqual(t, "auth.login_success", tr, "TR çevirisi bulunmalı")
	assert.Contains(t, tr, "giriş")

	en := i18n.Translate(i18n.LangEN, "auth.login_success")
	assert.NotEqual(t, "auth.login_success", en, "EN çevirisi bulunmalı")
	assert.Contains(t, en, "logged")
}

func TestI18n_MissingKey_ReturnKey(t *testing.T) {
	i18n.Init()
	result := i18n.Translate(i18n.LangTR, "this.key.does.not.exist")
	assert.Equal(t, "this.key.does.not.exist", result)
}

func TestI18n_FallbackToDefault(t *testing.T) {
	i18n.Init()
	// Desteklenmeyen dil → default (TR) kullanılmalı
	result := i18n.Translate("jp", "auth.login_success")
	trResult := i18n.Translate(i18n.LangTR, "auth.login_success")
	assert.Equal(t, trResult, result)
}

func TestI18n_FormatArgs(t *testing.T) {
	i18n.Init()
	result := i18n.Translate(i18n.LangTR, "validation.min_length", 8)
	assert.Contains(t, result, "8")
}

func TestI18n_AllKeysExistInBothLangs(t *testing.T) {
	i18n.Init()
	// TR'deki her key EN'de de olmalı
	criticalKeys := []string{
		"error.internal",
		"error.not_found",
		"error.unauthorized",
		"auth.login_success",
		"auth.login_failed",
		"crud.created",
		"crud.updated",
		"crud.deleted",
	}
	for _, key := range criticalKeys {
		tr := i18n.Translate(i18n.LangTR, key)
		en := i18n.Translate(i18n.LangEN, key)
		assert.NotEqual(t, key, tr, "TR key eksik: %s", key)
		assert.NotEqual(t, key, en, "EN key eksik: %s", key)
		assert.NotEqual(t, tr, en, "TR ve EN aynı olmamalı: %s", key)
	}
}
