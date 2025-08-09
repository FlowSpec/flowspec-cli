// Copyright 2024-2025 FlowSpec
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package i18n

import (
	"os"
	"testing"
)

func TestNewLocalizer(t *testing.T) {
	localizer := NewLocalizer(LanguageEnglish)
	if localizer.GetLanguage() != LanguageEnglish {
		t.Errorf("Expected language %s, got %s", LanguageEnglish, localizer.GetLanguage())
	}
}

func TestNewLocalizerFromEnv(t *testing.T) {
	// Test with FLOWSPEC_LANG
	os.Setenv("FLOWSPEC_LANG", "zh")
	localizer := NewLocalizerFromEnv()
	if localizer.GetLanguage() != LanguageChinese {
		t.Errorf("Expected language %s, got %s", LanguageChinese, localizer.GetLanguage())
	}
	os.Unsetenv("FLOWSPEC_LANG")

	// Test with LANG
	os.Setenv("LANG", "ja_JP.UTF-8")
	localizer = NewLocalizerFromEnv()
	if localizer.GetLanguage() != LanguageJapanese {
		t.Errorf("Expected language %s, got %s", LanguageJapanese, localizer.GetLanguage())
	}
	os.Unsetenv("LANG")

	// Test default (English)
	localizer = NewLocalizerFromEnv()
	if localizer.GetLanguage() != LanguageEnglish {
		t.Errorf("Expected default language %s, got %s", LanguageEnglish, localizer.GetLanguage())
	}
}

func TestTranslation(t *testing.T) {
	tests := []struct {
		name     string
		lang     SupportedLanguage
		key      string
		expected string
	}{
		{
			name:     "English report title",
			lang:     LanguageEnglish,
			key:      "report.title",
			expected: "FlowSpec Validation Report",
		},
		{
			name:     "Chinese report title",
			lang:     LanguageChinese,
			key:      "report.title",
			expected: "FlowSpec 验证报告",
		},
		{
			name:     "Missing key fallback",
			lang:     LanguageChinese,
			key:      "nonexistent.key",
			expected: "nonexistent.key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localizer := NewLocalizer(tt.lang)
			result := localizer.T(tt.key)
			if result != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestTranslationWithParams(t *testing.T) {
	localizer := NewLocalizer(LanguageEnglish)
	result := localizer.T("summary.total", 5)
	expected := "Total: 5 ServiceSpecs"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestSetLanguage(t *testing.T) {
	localizer := NewLocalizer(LanguageEnglish)
	localizer.SetLanguage(LanguageChinese)
	if localizer.GetLanguage() != LanguageChinese {
		t.Errorf("Expected language %s, got %s", LanguageChinese, localizer.GetLanguage())
	}
}

func TestIsSupported(t *testing.T) {
	tests := []struct {
		lang     SupportedLanguage
		expected bool
	}{
		{LanguageEnglish, true},
		{LanguageChinese, true},
		{LanguageJapanese, true},
		{SupportedLanguage("unsupported"), false},
	}

	for _, tt := range tests {
		result := IsSupported(tt.lang)
		if result != tt.expected {
			t.Errorf("IsSupported(%s) = %v, expected %v", tt.lang, result, tt.expected)
		}
	}
}

func TestGetSupportedLanguages(t *testing.T) {
	langs := GetSupportedLanguages()
	if len(langs) != 8 {
		t.Errorf("Expected 8 supported languages, got %d", len(langs))
	}

	// Check if English is included
	found := false
	for _, lang := range langs {
		if lang == LanguageEnglish {
			found = true
			break
		}
	}
	if !found {
		t.Error("English should be in supported languages")
	}
}
func TestDetectLanguageFromEnv(t *testing.T) {
	tests := []struct {
		name         string
		flowspecLang string
		langVar      string
		expected     SupportedLanguage
	}{
		{
			name:         "FLOWSPEC_LANG takes precedence",
			flowspecLang: "fr",
			langVar:      "zh_CN.UTF-8",
			expected:     LanguageFrench,
		},
		{
			name:         "LANG Chinese",
			flowspecLang: "",
			langVar:      "zh_CN.UTF-8",
			expected:     LanguageChinese,
		},
		{
			name:         "LANG Chinese Traditional",
			flowspecLang: "",
			langVar:      "zh_TW.UTF-8",
			expected:     LanguageChineseTraditional,
		},
		{
			name:         "LANG Japanese",
			flowspecLang: "",
			langVar:      "ja_JP.UTF-8",
			expected:     LanguageJapanese,
		},
		{
			name:         "LANG Korean",
			flowspecLang: "",
			langVar:      "ko_KR.UTF-8",
			expected:     LanguageKorean,
		},
		{
			name:         "LANG German",
			flowspecLang: "",
			langVar:      "de_DE.UTF-8",
			expected:     LanguageGerman,
		},
		{
			name:         "LANG Spanish",
			flowspecLang: "",
			langVar:      "es_ES.UTF-8",
			expected:     LanguageSpanish,
		},
		{
			name:         "LANG French",
			flowspecLang: "",
			langVar:      "fr_FR.UTF-8",
			expected:     LanguageFrench,
		},
		{
			name:         "Unsupported language defaults to English",
			flowspecLang: "",
			langVar:      "xx_XX.UTF-8",
			expected:     LanguageEnglish,
		},
		{
			name:         "No environment variables defaults to English",
			flowspecLang: "",
			langVar:      "",
			expected:     LanguageEnglish,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			os.Unsetenv("FLOWSPEC_LANG")
			os.Unsetenv("LANG")

			// Set test environment
			if tt.flowspecLang != "" {
				os.Setenv("FLOWSPEC_LANG", tt.flowspecLang)
			}
			if tt.langVar != "" {
				os.Setenv("LANG", tt.langVar)
			}

			result := detectLanguageFromEnv()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}

			// Clean up
			os.Unsetenv("FLOWSPEC_LANG")
			os.Unsetenv("LANG")
		})
	}
}

func TestAllLanguagesHaveBasicTranslations(t *testing.T) {
	// Test that all supported languages have basic translations
	basicKeys := []string{
		"report.title",
		"report.summary",
		"summary.total",
		"summary.success",
		"summary.failed",
	}

	for _, lang := range GetSupportedLanguages() {
		localizer := NewLocalizer(lang)
		for _, key := range basicKeys {
			translation := localizer.T(key)
			// Translation should not be the key itself (except for unsupported keys)
			if translation == key {
				t.Errorf("Language %s missing translation for key %s", lang, key)
			}
		}
	}
}
