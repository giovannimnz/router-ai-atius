package i18n

import (
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var placeholderPattern = regexp.MustCompile(`\{\{[^}]+\}\}`)

func TestSupportedLanguagesIncludesPortuguese(t *testing.T) {
	require.NoError(t, Init())
	assert.Contains(t, SupportedLanguages(), LangPt)
	assert.True(t, IsSupported("pt"))
	assert.True(t, IsSupported("pt-BR"))
	assert.True(t, IsSupported("pt_BR"))
}

func TestParseAcceptLanguageNormalizesPortuguese(t *testing.T) {
	assert.Equal(t, LangPt, ParseAcceptLanguage("pt-BR,pt;q=0.9,en;q=0.8"))
	assert.Equal(t, LangPt, ParseAcceptLanguage("pt_BR"))
	assert.Equal(t, LangEn, ParseAcceptLanguage("es-ES,es;q=0.9"))
}

func TestTranslateUsesPortugueseAndFallback(t *testing.T) {
	require.NoError(t, Init())

	assert.Equal(t, "Parâmetros inválidos", Translate("pt-BR", "common.invalid_params"))
	assert.Equal(t, "Invalid parameters", Translate("es-ES", "common.invalid_params"))
}

func TestPortugueseLocaleMatchesEnglishKeysAndPlaceholders(t *testing.T) {
	en := loadLocaleYAML(t, "locales/en.yaml")
	pt := loadLocaleYAML(t, "locales/pt.yaml")

	enKeys := sortedKeys(en)
	ptKeys := sortedKeys(pt)
	assert.Equal(t, enKeys, ptKeys)

	for _, key := range enKeys {
		assert.Equal(t, placeholders(en[key]), placeholders(pt[key]), key)
	}
}

func loadLocaleYAML(t *testing.T, path string) map[string]string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var locale map[string]string
	require.NoError(t, yaml.Unmarshal(data, &locale))
	return locale
}

func sortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func placeholders(value string) []string {
	return placeholderPattern.FindAllString(value, -1)
}
