package traffic

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	// Regex to collapse multiple consecutive slashes
	multipleSlashRegex = regexp.MustCompile(`/+`)
)

// NormalizePath normalizes a URL path according to the requirements:
// - URL decode
// - Remove trailing slash (except for root "/")
// - Collapse multiple consecutive slashes
// - Exclude query string
func NormalizePath(rawPath string) string {
	if rawPath == "" {
		return "/"
	}
	
	// Parse URL to separate path from query string
	parsedURL, err := url.Parse(rawPath)
	if err != nil {
		// If parsing fails, treat the entire string as path
		parsedURL = &url.URL{Path: rawPath}
	}
	
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}
	
	// URL decode the path
	decodedPath, err := url.QueryUnescape(path)
	if err != nil {
		// If decoding fails, use the original path
		decodedPath = path
	}
	
	// Collapse multiple consecutive slashes
	normalizedPath := multipleSlashRegex.ReplaceAllString(decodedPath, "/")
	
	// Remove trailing slash, but keep root "/"
	if len(normalizedPath) > 1 && strings.HasSuffix(normalizedPath, "/") {
		normalizedPath = strings.TrimSuffix(normalizedPath, "/")
	}
	
	// Ensure path starts with "/"
	if !strings.HasPrefix(normalizedPath, "/") {
		normalizedPath = "/" + normalizedPath
	}
	
	return normalizedPath
}

// NormalizeHeaders normalizes header keys to lowercase and supports multi-value headers
func NormalizeHeaders(headers map[string]string) map[string][]string {
	if headers == nil {
		return make(map[string][]string)
	}
	
	normalized := make(map[string][]string)
	for key, value := range headers {
		normalizedKey := strings.ToLower(key)
		
		// Split multi-value headers by comma (common HTTP practice)
		values := strings.Split(value, ",")
		for i, v := range values {
			values[i] = strings.TrimSpace(v)
		}
		
		normalized[normalizedKey] = values
	}
	
	return normalized
}

// NormalizeQuery preserves query parameter keys as-is and supports multi-value parameters
func NormalizeQuery(queryString string) map[string][]string {
	if queryString == "" {
		return make(map[string][]string)
	}
	
	// Parse query string
	values, err := url.ParseQuery(queryString)
	if err != nil {
		// If parsing fails, return empty map
		return make(map[string][]string)
	}
	
	// Convert url.Values to map[string][]string (they're the same type, but explicit conversion for clarity)
	result := make(map[string][]string)
	for key, valueList := range values {
		result[key] = valueList
	}
	
	return result
}

// ExtractQueryString extracts the query string from a raw path
func ExtractQueryString(rawPath string) string {
	parsedURL, err := url.Parse(rawPath)
	if err != nil {
		return ""
	}
	return parsedURL.RawQuery
}

// ApplyRedactionPolicy applies the specified redaction policy to sensitive fields
func ApplyRedactionPolicy(headers map[string][]string, query map[string][]string, sensitiveKeys []string, policy string) (map[string][]string, map[string][]string) {
	if len(sensitiveKeys) == 0 {
		return headers, query
	}
	
	// Create sets for faster lookup
	sensitiveSet := make(map[string]bool)
	for _, key := range sensitiveKeys {
		sensitiveSet[strings.ToLower(key)] = true
	}
	
	// Apply redaction to headers
	redactedHeaders := make(map[string][]string)
	for key, values := range headers {
		if sensitiveSet[strings.ToLower(key)] {
			switch policy {
			case "drop":
				// Skip this header entirely
				continue
			case "mask":
				redactedHeaders[key] = []string{"***"}
			case "hash":
				// Simple hash representation (in real implementation, use proper hashing)
				redactedHeaders[key] = []string{"<hashed>"}
			default:
				// Default to drop
				continue
			}
		} else {
			redactedHeaders[key] = values
		}
	}
	
	// Apply redaction to query parameters
	redactedQuery := make(map[string][]string)
	for key, values := range query {
		if sensitiveSet[strings.ToLower(key)] {
			switch policy {
			case "drop":
				// Skip this parameter entirely
				continue
			case "mask":
				redactedQuery[key] = []string{"***"}
			case "hash":
				// Simple hash representation (in real implementation, use proper hashing)
				redactedQuery[key] = []string{"<hashed>"}
			default:
				// Default to drop
				continue
			}
		} else {
			redactedQuery[key] = values
		}
	}
	
	return redactedHeaders, redactedQuery
}