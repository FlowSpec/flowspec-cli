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
	"fmt"
	"os"
	"strings"
)

// SupportedLanguage represents a supported language
type SupportedLanguage string

const (
	LanguageEnglish            SupportedLanguage = "en"
	LanguageChinese            SupportedLanguage = "zh"
	LanguageChineseTraditional SupportedLanguage = "zh-TW"
	LanguageJapanese           SupportedLanguage = "ja"
	LanguageKorean             SupportedLanguage = "ko"
	LanguageFrench             SupportedLanguage = "fr"
	LanguageGerman             SupportedLanguage = "de"
	LanguageSpanish            SupportedLanguage = "es"
)

// Localizer handles internationalization
type Localizer struct {
	language SupportedLanguage
	messages map[string]string
}

// NewLocalizer creates a new localizer with the specified language
func NewLocalizer(lang SupportedLanguage) *Localizer {
	l := &Localizer{
		language: lang,
		messages: make(map[string]string),
	}
	l.loadMessages()
	return l
}

// NewLocalizerFromEnv creates a new localizer based on environment variables
func NewLocalizerFromEnv() *Localizer {
	lang := detectLanguageFromEnv()
	return NewLocalizer(lang)
}

// T translates a message key with optional parameters
func (l *Localizer) T(key string, params ...interface{}) string {
	if message, exists := l.messages[key]; exists {
		if len(params) > 0 {
			return fmt.Sprintf(message, params...)
		}
		return message
	}
	// Fallback to English if key not found
	if l.language != LanguageEnglish {
		englishLocalizer := NewLocalizer(LanguageEnglish)
		return englishLocalizer.T(key, params...)
	}
	// If English also doesn't have the key, return the key itself
	return key
}

// GetLanguage returns the current language
func (l *Localizer) GetLanguage() SupportedLanguage {
	return l.language
}

// SetLanguage changes the current language
func (l *Localizer) SetLanguage(lang SupportedLanguage) {
	l.language = lang
	l.loadMessages()
}

// detectLanguageFromEnv detects language from environment variables
func detectLanguageFromEnv() SupportedLanguage {
	// Check FLOWSPEC_LANG first
	if lang := os.Getenv("FLOWSPEC_LANG"); lang != "" {
		return SupportedLanguage(lang)
	}

	// Check LANG environment variable
	if lang := os.Getenv("LANG"); lang != "" {
		lang = strings.ToLower(lang)
		if strings.HasPrefix(lang, "zh") {
			if strings.Contains(lang, "tw") || strings.Contains(lang, "hk") {
				return LanguageChineseTraditional
			}
			return LanguageChinese
		}
		if strings.HasPrefix(lang, "ja") {
			return LanguageJapanese
		}
		if strings.HasPrefix(lang, "ko") {
			return LanguageKorean
		}
		if strings.HasPrefix(lang, "fr") {
			return LanguageFrench
		}
		if strings.HasPrefix(lang, "de") {
			return LanguageGerman
		}
		if strings.HasPrefix(lang, "es") {
			return LanguageSpanish
		}
	}

	// Default to English
	return LanguageEnglish
}

// loadMessages loads messages for the current language
func (l *Localizer) loadMessages() {
	switch l.language {
	case LanguageEnglish:
		l.messages = englishMessages
	case LanguageChinese:
		l.messages = chineseMessages
	case LanguageChineseTraditional:
		l.messages = chineseTraditionalMessages
	case LanguageJapanese:
		l.messages = japaneseMessages
	case LanguageKorean:
		l.messages = koreanMessages
	case LanguageFrench:
		l.messages = frenchMessages
	case LanguageGerman:
		l.messages = germanMessages
	case LanguageSpanish:
		l.messages = spanishMessages
	default:
		l.messages = englishMessages
	}
}

// IsSupported checks if a language is supported
func IsSupported(lang SupportedLanguage) bool {
	supportedLangs := []SupportedLanguage{
		LanguageEnglish,
		LanguageChinese,
		LanguageChineseTraditional,
		LanguageJapanese,
		LanguageKorean,
		LanguageFrench,
		LanguageGerman,
		LanguageSpanish,
	}
	
	for _, supported := range supportedLangs {
		if lang == supported {
			return true
		}
	}
	return false
}

// GetSupportedLanguages returns all supported languages
func GetSupportedLanguages() []SupportedLanguage {
	return []SupportedLanguage{
		LanguageEnglish,
		LanguageChinese,
		LanguageChineseTraditional,
		LanguageJapanese,
		LanguageKorean,
		LanguageFrench,
		LanguageGerman,
		LanguageSpanish,
	}
}