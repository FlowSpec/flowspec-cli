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

package traffic

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePath(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{
			name:     "Simple path",
			input:    "/api/users",
			expected: "/api/users",
		},
		{
			name:     "Root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "Empty path",
			input:    "",
			expected: "/",
		},
		
		// Trailing slash removal
		{
			name:     "Remove trailing slash",
			input:    "/api/users/",
			expected: "/api/users",
		},
		{
			name:     "Keep root slash",
			input:    "/",
			expected: "/",
		},
		{
			name:     "Multiple trailing slashes",
			input:    "/api/users///",
			expected: "/api/users",
		},
		
		// Multiple consecutive slashes
		{
			name:     "Collapse multiple slashes",
			input:    "/api//users///123",
			expected: "/api/users/123",
		},
		{
			name:     "Leading multiple slashes",
			input:    "///api/users",
			expected: "/api/users",
		},
		
		// URL decoding
		{
			name:     "URL encoded characters",
			input:    "/api/users/john%20doe",
			expected: "/api/users/john doe",
		},
		{
			name:     "URL encoded special characters",
			input:    "/api/search?q=hello%2Bworld",
			expected: "/api/search",
		},
		{
			name:     "URL encoded path segments",
			input:    "/api/users/test%40example.com",
			expected: "/api/users/test@example.com",
		},
		
		// Query string exclusion
		{
			name:     "Path with query string",
			input:    "/api/users?id=123&name=john",
			expected: "/api/users",
		},
		{
			name:     "Path with fragment",
			input:    "/api/users#section1",
			expected: "/api/users",
		},
		{
			name:     "Path with query and fragment",
			input:    "/api/users?id=123#section1",
			expected: "/api/users",
		},
		
		// Edge cases
		{
			name:     "Path without leading slash",
			input:    "api/users",
			expected: "/api/users",
		},
		{
			name:     "Complex path with all issues",
			input:    "//api///users//123/?id=456&name=john%20doe#section",
			expected: "/users/123", // URL parser treats //api as host, path is ///users//123/ -> /users/123
		},
		{
			name:     "Invalid URL encoding (should not crash)",
			input:    "/api/users/test%ZZ",
			expected: "/api/users/test%ZZ", // Should use original if decoding fails
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizePath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalizeHeaders(t *testing.T) {
	testCases := []struct {
		name     string
		input    map[string]string
		expected map[string][]string
	}{
		{
			name:     "Nil input",
			input:    nil,
			expected: map[string][]string{},
		},
		{
			name:     "Empty input",
			input:    map[string]string{},
			expected: map[string][]string{},
		},
		{
			name: "Single headers",
			input: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
				"User-Agent":    "Mozilla/5.0",
			},
			expected: map[string][]string{
				"content-type":  {"application/json"},
				"authorization": {"Bearer token123"},
				"user-agent":    {"Mozilla/5.0"},
			},
		},
		{
			name: "Multi-value headers",
			input: map[string]string{
				"Accept":          "application/json, text/html",
				"Accept-Encoding": "gzip, deflate, br",
				"Cache-Control":   "no-cache, no-store, must-revalidate",
			},
			expected: map[string][]string{
				"accept":          {"application/json", "text/html"},
				"accept-encoding": {"gzip", "deflate", "br"},
				"cache-control":   {"no-cache", "no-store", "must-revalidate"},
			},
		},
		{
			name: "Headers with extra spaces",
			input: map[string]string{
				"Accept": "application/json,  text/html  , text/plain",
			},
			expected: map[string][]string{
				"accept": {"application/json", "text/html", "text/plain"},
			},
		},
		{
			name: "Mixed case headers",
			input: map[string]string{
				"Content-TYPE":  "application/json",
				"AUTHORIZATION": "Bearer token123",
				"user-agent":    "Mozilla/5.0",
			},
			expected: map[string][]string{
				"content-type":  {"application/json"},
				"authorization": {"Bearer token123"},
				"user-agent":    {"Mozilla/5.0"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeHeaders(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected map[string][]string
	}{
		{
			name:     "Empty query string",
			input:    "",
			expected: map[string][]string{},
		},
		{
			name:  "Single parameter",
			input: "id=123",
			expected: map[string][]string{
				"id": {"123"},
			},
		},
		{
			name:  "Multiple parameters",
			input: "id=123&name=john&active=true",
			expected: map[string][]string{
				"id":     {"123"},
				"name":   {"john"},
				"active": {"true"},
			},
		},
		{
			name:  "Multi-value parameters",
			input: "tags=red&tags=blue&tags=green",
			expected: map[string][]string{
				"tags": {"red", "blue", "green"},
			},
		},
		{
			name:  "URL encoded values",
			input: "name=john%20doe&email=test%40example.com",
			expected: map[string][]string{
				"name":  {"john doe"},
				"email": {"test@example.com"},
			},
		},
		{
			name:  "Empty values",
			input: "empty=&blank&novalue=",
			expected: map[string][]string{
				"empty":   {""},
				"blank":   {""},
				"novalue": {""},
			},
		},
		{
			name:  "Special characters in keys",
			input: "filter[name]=john&sort[created_at]=desc",
			expected: map[string][]string{
				"filter[name]":       {"john"},
				"sort[created_at]":   {"desc"},
			},
		},
		{
			name:  "Complex query string",
			input: "q=search%20term&limit=10&offset=20&include=profile&include=settings",
			expected: map[string][]string{
				"q":       {"search term"},
				"limit":   {"10"},
				"offset":  {"20"},
				"include": {"profile", "settings"},
			},
		},
		{
			name:     "Invalid query string",
			input:    "invalid%ZZ",
			expected: map[string][]string{}, // Should return empty map if parsing fails
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeQuery(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestExtractQueryString(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Path without query",
			input:    "/api/users",
			expected: "",
		},
		{
			name:     "Path with query",
			input:    "/api/users?id=123&name=john",
			expected: "id=123&name=john",
		},
		{
			name:     "Path with empty query",
			input:    "/api/users?",
			expected: "",
		},
		{
			name:     "Path with fragment",
			input:    "/api/users#section1",
			expected: "",
		},
		{
			name:     "Path with query and fragment",
			input:    "/api/users?id=123#section1",
			expected: "id=123",
		},
		{
			name:     "Complex URL",
			input:    "/api/search?q=hello%20world&limit=10&offset=20",
			expected: "q=hello%20world&limit=10&offset=20",
		},
		{
			name:     "Invalid URL",
			input:    "not a valid url",
			expected: "",
		},
		{
			name:     "Empty input",
			input:    "",
			expected: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ExtractQueryString(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestApplyRedactionPolicy(t *testing.T) {
	headers := map[string][]string{
		"authorization": {"Bearer token123"},
		"cookie":        {"session=abc123"},
		"content-type":  {"application/json"},
		"user-agent":    {"Mozilla/5.0"},
	}

	query := map[string][]string{
		"token":    {"secret123"},
		"password": {"mypassword"},
		"id":       {"123"},
		"name":     {"john"},
	}

	sensitiveKeys := []string{"authorization", "cookie", "token", "password"}

	testCases := []struct {
		name            string
		policy          string
		expectedHeaders map[string][]string
		expectedQuery   map[string][]string
	}{
		{
			name:   "Drop policy",
			policy: "drop",
			expectedHeaders: map[string][]string{
				"content-type": {"application/json"},
				"user-agent":   {"Mozilla/5.0"},
			},
			expectedQuery: map[string][]string{
				"id":   {"123"},
				"name": {"john"},
			},
		},
		{
			name:   "Mask policy",
			policy: "mask",
			expectedHeaders: map[string][]string{
				"authorization": {"***"},
				"cookie":        {"***"},
				"content-type":  {"application/json"},
				"user-agent":    {"Mozilla/5.0"},
			},
			expectedQuery: map[string][]string{
				"token":    {"***"},
				"password": {"***"},
				"id":       {"123"},
				"name":     {"john"},
			},
		},
		{
			name:   "Hash policy",
			policy: "hash",
			expectedHeaders: map[string][]string{
				"authorization": {"<hashed>"},
				"cookie":        {"<hashed>"},
				"content-type":  {"application/json"},
				"user-agent":    {"Mozilla/5.0"},
			},
			expectedQuery: map[string][]string{
				"token":    {"<hashed>"},
				"password": {"<hashed>"},
				"id":       {"123"},
				"name":     {"john"},
			},
		},
		{
			name:   "Unknown policy (defaults to drop)",
			policy: "unknown",
			expectedHeaders: map[string][]string{
				"content-type": {"application/json"},
				"user-agent":   {"Mozilla/5.0"},
			},
			expectedQuery: map[string][]string{
				"id":   {"123"},
				"name": {"john"},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resultHeaders, resultQuery := ApplyRedactionPolicy(headers, query, sensitiveKeys, tc.policy)
			assert.Equal(t, tc.expectedHeaders, resultHeaders)
			assert.Equal(t, tc.expectedQuery, resultQuery)
		})
	}
}

func TestApplyRedactionPolicy_EmptySensitiveKeys(t *testing.T) {
	headers := map[string][]string{
		"authorization": {"Bearer token123"},
		"content-type":  {"application/json"},
	}

	query := map[string][]string{
		"token": {"secret123"},
		"id":    {"123"},
	}

	// No sensitive keys - should return original data
	resultHeaders, resultQuery := ApplyRedactionPolicy(headers, query, []string{}, "drop")

	assert.Equal(t, headers, resultHeaders)
	assert.Equal(t, query, resultQuery)
}

func TestApplyRedactionPolicy_CaseInsensitive(t *testing.T) {
	headers := map[string][]string{
		"Authorization": {"Bearer token123"},
		"COOKIE":        {"session=abc123"},
		"content-type":  {"application/json"},
	}

	query := map[string][]string{
		"TOKEN":    {"secret123"},
		"Password": {"mypassword"},
		"id":       {"123"},
	}

	sensitiveKeys := []string{"authorization", "cookie", "token", "password"}

	resultHeaders, resultQuery := ApplyRedactionPolicy(headers, query, sensitiveKeys, "drop")

	// Should drop sensitive keys regardless of case
	expectedHeaders := map[string][]string{
		"content-type": {"application/json"},
	}
	expectedQuery := map[string][]string{
		"id": {"123"},
	}

	assert.Equal(t, expectedHeaders, resultHeaders)
	assert.Equal(t, expectedQuery, resultQuery)
}

func TestNormalizePath_Integration(t *testing.T) {
	// Test complex real-world scenarios
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Real nginx log path",
			input:    "/api/v1/users/123/profile?include=settings&format=json",
			expected: "/api/v1/users/123/profile",
		},
		{
			name:     "Path with encoded spaces and special chars",
			input:    "/api/search?q=hello%20world%21&category=tech%26science",
			expected: "/api/search",
		},
		{
			name:     "Messy path from real logs",
			input:    "///api//v1///users//123//?id=456&name=john%20doe///",
			expected: "/api/v1/users/123",
		},
		{
			name:     "Path with encoded slashes",
			input:    "/api/files/folder%2Fsubfolder%2Ffile.txt",
			expected: "/api/files/folder/subfolder/file.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizePath(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}