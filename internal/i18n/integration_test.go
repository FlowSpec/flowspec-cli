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
	"strings"
	"testing"
)

// TestI18nIntegration tests the complete internationalization workflow
func TestI18nIntegration(t *testing.T) {
	// Test scenario: Create localizer, switch languages, verify translations
	localizer := NewLocalizer(LanguageEnglish)

	// Test English
	title := localizer.T("report.title")
	if !strings.Contains(title, "FlowSpec") {
		t.Errorf("English title should contain 'FlowSpec', got: %s", title)
	}

	// Switch to Chinese
	localizer.SetLanguage(LanguageChinese)
	title = localizer.T("report.title")
	if !strings.Contains(title, "验证报告") {
		t.Errorf("Chinese title should contain '验证报告', got: %s", title)
	}

	// Switch to Japanese
	localizer.SetLanguage(LanguageJapanese)
	title = localizer.T("report.title")
	if !strings.Contains(title, "検証レポート") {
		t.Errorf("Japanese title should contain '検証レポート', got: %s", title)
	}
}

// TestEnvironmentDetectionIntegration tests environment-based language detection
func TestEnvironmentDetectionIntegration(t *testing.T) {
	// Save original environment
	originalFlowspecLang := os.Getenv("FLOWSPEC_LANG")
	originalLang := os.Getenv("LANG")

	// Clean up function
	defer func() {
		if originalFlowspecLang != "" {
			os.Setenv("FLOWSPEC_LANG", originalFlowspecLang)
		} else {
			os.Unsetenv("FLOWSPEC_LANG")
		}
		if originalLang != "" {
			os.Setenv("LANG", originalLang)
		} else {
			os.Unsetenv("LANG")
		}
	}()

	// Test Chinese environment
	os.Setenv("FLOWSPEC_LANG", "zh")
	localizer := NewLocalizerFromEnv()
	
	if localizer.GetLanguage() != LanguageChinese {
		t.Errorf("Expected Chinese language, got %s", localizer.GetLanguage())
	}

	summary := localizer.T("report.summary")
	if !strings.Contains(summary, "汇总") {
		t.Errorf("Chinese summary should contain '汇总', got: %s", summary)
	}
}

// TestParameterizedTranslations tests translations with parameters
func TestParameterizedTranslations(t *testing.T) {
	tests := []struct {
		lang     SupportedLanguage
		key      string
		params   []interface{}
		contains string
	}{
		{
			lang:     LanguageEnglish,
			key:      "summary.total",
			params:   []interface{}{42},
			contains: "42",
		},
		{
			lang:     LanguageChinese,
			key:      "summary.total",
			params:   []interface{}{42},
			contains: "42",
		},
		{
			lang:     LanguageJapanese,
			key:      "summary.total",
			params:   []interface{}{42},
			contains: "42",
		},
	}

	for _, tt := range tests {
		localizer := NewLocalizer(tt.lang)
		result := localizer.T(tt.key, tt.params...)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("Language %s: expected result to contain '%s', got: %s", 
				tt.lang, tt.contains, result)
		}
	}
}

// TestAllLanguagesCompleteness tests that all languages have complete translations
func TestAllLanguagesCompleteness(t *testing.T) {
	// Define critical keys that must exist in all languages
	criticalKeys := []string{
		"report.title",
		"report.summary",
		"report.details",
		"summary.total",
		"summary.success",
		"summary.failed",
		"summary.skipped",
		"status.success",
		"status.failed",
	}

	for _, lang := range GetSupportedLanguages() {
		localizer := NewLocalizer(lang)
		for _, key := range criticalKeys {
			translation := localizer.T(key)
			if translation == key {
				t.Errorf("Language %s missing critical translation for key: %s", lang, key)
			}
			if strings.TrimSpace(translation) == "" {
				t.Errorf("Language %s has empty translation for key: %s", lang, key)
			}
		}
	}
}

// BenchmarkTranslation benchmarks translation performance
func BenchmarkTranslation(b *testing.B) {
	localizer := NewLocalizer(LanguageEnglish)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = localizer.T("report.title")
	}
}

// BenchmarkTranslationWithParams benchmarks parameterized translation performance
func BenchmarkTranslationWithParams(b *testing.B) {
	localizer := NewLocalizer(LanguageEnglish)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = localizer.T("summary.total", 100)
	}
}

// BenchmarkLanguageSwitching benchmarks language switching performance
func BenchmarkLanguageSwitching(b *testing.B) {
	localizer := NewLocalizer(LanguageEnglish)
	languages := GetSupportedLanguages()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lang := languages[i%len(languages)]
		localizer.SetLanguage(lang)
		_ = localizer.T("report.title")
	}
}