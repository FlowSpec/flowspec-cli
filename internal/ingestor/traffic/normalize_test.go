package traffic

import (
	"reflect"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "/",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "simple path",
			input:    "/api/users",
			expected: "/api/users",
		},
		{
			name:     "path with trailing slash",
			input:    "/api/users/",
			expected: "/api/users",
		},
		{
			name:     "path with query string",
			input:    "/api/users?id=123",
			expected: "/api/users",
		},
		{
			name:     "path with multiple slashes",
			input:    "/api//users///123",
			expected: "/api/users/123",
		},
		{
			name:     "URL encoded path",
			input:    "/api/users/john%20doe",
			expected: "/api/users/john doe",
		},
		{
			name:     "path without leading slash",
			input:    "api/users",
			expected: "/api/users",
		},
		{
			name:     "complex path with query and encoding",
			input:    "/api//users/john%20doe/?include=profile&sort=name",
			expected: "/api/users/john doe",
		},
		{
			name:     "root with trailing slash should remain root",
			input:    "/",
			expected: "/",
		},
		{
			name:     "path with fragment",
			input:    "/api/users#section",
			expected: "/api/users",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePath(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizePath(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeHeaders(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected map[string][]string
	}{
		{
			name:     "nil headers",
			input:    nil,
			expected: map[string][]string{},
		},
		{
			name:     "empty headers",
			input:    map[string]string{},
			expected: map[string][]string{},
		},
		{
			name: "single value headers",
			input: map[string]string{
				"Content-Type":  "application/json",
				"Authorization": "Bearer token123",
			},
			expected: map[string][]string{
				"content-type":  {"application/json"},
				"authorization": {"Bearer token123"},
			},
		},
		{
			name: "multi-value header",
			input: map[string]string{
				"Accept": "application/json, text/html, */*",
			},
			expected: map[string][]string{
				"accept": {"application/json", "text/html", "*/*"},
			},
		},
		{
			name: "mixed case headers",
			input: map[string]string{
				"Content-TYPE": "application/json",
				"accept":       "text/html",
				"X-Custom":     "value",
			},
			expected: map[string][]string{
				"content-type": {"application/json"},
				"accept":       {"text/html"},
				"x-custom":     {"value"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeHeaders(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NormalizeHeaders(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string][]string
	}{
		{
			name:     "empty query string",
			input:    "",
			expected: map[string][]string{},
		},
		{
			name:  "single parameter",
			input: "id=123",
			expected: map[string][]string{
				"id": {"123"},
			},
		},
		{
			name:  "multiple parameters",
			input: "id=123&name=john&active=true",
			expected: map[string][]string{
				"id":     {"123"},
				"name":   {"john"},
				"active": {"true"},
			},
		},
		{
			name:  "multi-value parameter",
			input: "tags=red&tags=blue&tags=green",
			expected: map[string][]string{
				"tags": {"red", "blue", "green"},
			},
		},
		{
			name:  "URL encoded values",
			input: "name=john%20doe&message=hello%20world",
			expected: map[string][]string{
				"name":    {"john doe"},
				"message": {"hello world"},
			},
		},
		{
			name:  "parameter without value",
			input: "debug&verbose=true",
			expected: map[string][]string{
				"debug":   {""},
				"verbose": {"true"},
			},
		},
		{
			name:  "case sensitive keys",
			input: "ID=123&id=456",
			expected: map[string][]string{
				"ID": {"123"},
				"id": {"456"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeQuery(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("NormalizeQuery(%q) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestExtractQueryString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no query string",
			input:    "/api/users",
			expected: "",
		},
		{
			name:     "simple query string",
			input:    "/api/users?id=123",
			expected: "id=123",
		},
		{
			name:     "complex query string",
			input:    "/api/users?id=123&name=john&active=true",
			expected: "id=123&name=john&active=true",
		},
		{
			name:     "query string with fragment",
			input:    "/api/users?id=123#section",
			expected: "id=123",
		},
		{
			name:     "empty query string",
			input:    "/api/users?",
			expected: "",
		},
		{
			name:     "malformed URL",
			input:    "not a valid url",
			expected: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractQueryString(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractQueryString(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestApplyRedactionPolicy(t *testing.T) {
	headers := map[string][]string{
		"authorization": {"Bearer token123"},
		"content-type":  {"application/json"},
		"cookie":        {"session=abc123"},
		"accept":        {"application/json"},
	}
	
	query := map[string][]string{
		"token":    {"secret123"},
		"id":       {"123"},
		"password": {"secret"},
		"name":     {"john"},
	}
	
	sensitiveKeys := []string{"authorization", "cookie", "token", "password"}
	
	tests := []struct {
		name            string
		policy          string
		expectedHeaders map[string][]string
		expectedQuery   map[string][]string
	}{
		{
			name:   "drop policy",
			policy: "drop",
			expectedHeaders: map[string][]string{
				"content-type": {"application/json"},
				"accept":       {"application/json"},
			},
			expectedQuery: map[string][]string{
				"id":   {"123"},
				"name": {"john"},
			},
		},
		{
			name:   "mask policy",
			policy: "mask",
			expectedHeaders: map[string][]string{
				"authorization": {"***"},
				"content-type":  {"application/json"},
				"cookie":        {"***"},
				"accept":        {"application/json"},
			},
			expectedQuery: map[string][]string{
				"token":    {"***"},
				"id":       {"123"},
				"password": {"***"},
				"name":     {"john"},
			},
		},
		{
			name:   "hash policy",
			policy: "hash",
			expectedHeaders: map[string][]string{
				"authorization": {"<hashed>"},
				"content-type":  {"application/json"},
				"cookie":        {"<hashed>"},
				"accept":        {"application/json"},
			},
			expectedQuery: map[string][]string{
				"token":    {"<hashed>"},
				"id":       {"123"},
				"password": {"<hashed>"},
				"name":     {"john"},
			},
		},
		{
			name:   "unknown policy defaults to drop",
			policy: "unknown",
			expectedHeaders: map[string][]string{
				"content-type": {"application/json"},
				"accept":       {"application/json"},
			},
			expectedQuery: map[string][]string{
				"id":   {"123"},
				"name": {"john"},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resultHeaders, resultQuery := ApplyRedactionPolicy(headers, query, sensitiveKeys, tt.policy)
			
			if !reflect.DeepEqual(resultHeaders, tt.expectedHeaders) {
				t.Errorf("ApplyRedactionPolicy headers = %v, expected %v", resultHeaders, tt.expectedHeaders)
			}
			
			if !reflect.DeepEqual(resultQuery, tt.expectedQuery) {
				t.Errorf("ApplyRedactionPolicy query = %v, expected %v", resultQuery, tt.expectedQuery)
			}
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
	
	resultHeaders, resultQuery := ApplyRedactionPolicy(headers, query, []string{}, "drop")
	
	// Should return original data when no sensitive keys are specified
	if !reflect.DeepEqual(resultHeaders, headers) {
		t.Errorf("Expected headers to remain unchanged, got %v", resultHeaders)
	}
	
	if !reflect.DeepEqual(resultQuery, query) {
		t.Errorf("Expected query to remain unchanged, got %v", resultQuery)
	}
}