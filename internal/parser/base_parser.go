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

package parser

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/flowspec/flowspec-cli/internal/models"
	"gopkg.in/yaml.v3"
)

// File extension constants
const (
	javaExtension = ".java"
)

// BaseFileParser provides common functionality for parsing ServiceSpec annotations
type BaseFileParser struct {
	language        SupportedLanguage
	commentPatterns []CommentPattern
}

// CommentPattern defines how comments are structured in different languages
type CommentPattern struct {
	StartPattern string // e.g., "/**", "//"
	EndPattern   string // e.g., "*/", "" (empty for single-line)
	LinePrefix   string // e.g., " * ", "// "
	IsMultiLine  bool
}

// ServiceSpecAnnotation represents a parsed ServiceSpec annotation
type ServiceSpecAnnotation struct {
	OperationID    string
	Description    string
	Preconditions  interface{}
	Postconditions interface{}
	StartLine      int
	EndLine        int
}

// NewBaseFileParser creates a new base file parser
func NewBaseFileParser(language SupportedLanguage) *BaseFileParser {
	return &BaseFileParser{
		language:        language,
		commentPatterns: getCommentPatterns(language),
	}
}

// getCommentPatterns returns comment patterns for different languages
func getCommentPatterns(language SupportedLanguage) []CommentPattern {
	switch language {
	case LanguageJava:
		return []CommentPattern{
			{
				StartPattern: "/**",
				EndPattern:   "*/",
				LinePrefix:   " * ",
				IsMultiLine:  true,
			},
			{
				StartPattern: "/*",
				EndPattern:   "*/",
				LinePrefix:   " * ",
				IsMultiLine:  true,
			},
		}
	case LanguageTypeScript:
		return []CommentPattern{
			{
				StartPattern: "/**",
				EndPattern:   "*/",
				LinePrefix:   " * ",
				IsMultiLine:  true,
			},
			{
				StartPattern: "/*",
				EndPattern:   "*/",
				LinePrefix:   " * ",
				IsMultiLine:  true,
			},
			{
				StartPattern: "//",
				EndPattern:   "",
				LinePrefix:   "// ",
				IsMultiLine:  false,
			},
		}
	case LanguageGo:
		return []CommentPattern{
			{
				StartPattern: "/*",
				EndPattern:   "*/",
				LinePrefix:   " * ",
				IsMultiLine:  true,
			},
			{
				StartPattern: "//",
				EndPattern:   "",
				LinePrefix:   "// ",
				IsMultiLine:  false,
			},
		}
	default:
		return []CommentPattern{}
	}
}

// ParseFile parses a file and extracts ServiceSpec annotations
func (bp *BaseFileParser) ParseFile(filepath string) ([]models.ServiceSpec, []models.ParseError) {
	errorCollector := NewErrorCollector()

	file, err := os.Open(filepath)
	if err != nil {
		errorCollector.AddError(filepath, 0, fmt.Sprintf("failed to open file: %s", err.Error()))
		return []models.ServiceSpec{}, errorCollector.GetErrors()
	}
	defer file.Close()

	// Read file line by line
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		errorCollector.AddError(filepath, 0, fmt.Sprintf("failed to read file: %s", err.Error()))
		return []models.ServiceSpec{}, errorCollector.GetErrors()
	}

	// Extract ServiceSpec annotations
	annotations := bp.extractServiceSpecAnnotations(lines, filepath, errorCollector)

	// Convert annotations to ServiceSpec objects
	var specs []models.ServiceSpec
	for _, annotation := range annotations {
		spec, err := bp.convertAnnotationToSpec(annotation, filepath)
		if err != nil {
			errorCollector.AddError(filepath, annotation.StartLine, err.Error())
			continue
		}
		specs = append(specs, spec)
	}

	return specs, errorCollector.GetErrors()
}

// extractServiceSpecAnnotations finds and extracts ServiceSpec annotations from file lines
func (bp *BaseFileParser) extractServiceSpecAnnotations(lines []string, filepath string, errorCollector *ErrorCollector) []ServiceSpecAnnotation {
	var annotations []ServiceSpecAnnotation

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Look for ServiceSpec annotation start
		if bp.isServiceSpecStart(line) {
			annotation, endLine := bp.parseServiceSpecAnnotation(lines, i, filepath, errorCollector)
			if annotation != nil {
				annotations = append(annotations, *annotation)
			}
			// Move to the end line to avoid re-processing
			if endLine > i {
				i = endLine
			}
		}
	}

	return annotations
}

// isServiceSpecStart checks if a line starts a ServiceSpec annotation
func (bp *BaseFileParser) isServiceSpecStart(line string) bool {
	// Look for @ServiceSpec in various comment formats
	patterns := []string{
		`@ServiceSpec`,
		`\*\s*@ServiceSpec`,
		`//\s*@ServiceSpec`,
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, line)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// parseServiceSpecAnnotation parses a complete ServiceSpec annotation
func (bp *BaseFileParser) parseServiceSpecAnnotation(lines []string, startLine int, filepath string, errorCollector *ErrorCollector) (*ServiceSpecAnnotation, int) {
	annotation := &ServiceSpecAnnotation{
		StartLine: startLine + 1, // Convert to 1-based line numbers
	}

	// Look backwards to find the start of the comment block
	commentStart := startLine
	for commentStart > 0 {
		prevLine := strings.TrimSpace(lines[commentStart-1])
		if strings.Contains(prevLine, "/**") || strings.Contains(prevLine, "/*") {
			commentStart = commentStart - 1
			break
		}
		if strings.HasPrefix(prevLine, "*") || strings.HasPrefix(prevLine, " *") {
			commentStart = commentStart - 1
		} else {
			break
		}
	}

	// Determine comment pattern from the actual comment start
	pattern := bp.detectCommentPattern(lines[commentStart])
	if pattern == nil {
		// Try to detect from the current line
		pattern = bp.detectCommentPattern(lines[startLine])
		if pattern == nil {
			errorCollector.AddError(filepath, startLine+1, "unable to detect comment pattern")
			return nil, startLine
		}
	}

	// Extract annotation content starting from the comment start
	content, endLine := bp.extractAnnotationContent(lines, commentStart, pattern)
	annotation.EndLine = endLine + 1

	// Parse the content
	if err := bp.parseAnnotationContent(content, annotation, filepath); err != nil {
		errorCollector.AddError(filepath, startLine+1, err.Error())
		return nil, endLine
	}

	return annotation, endLine
}

// detectCommentPattern determines which comment pattern is being used
func (bp *BaseFileParser) detectCommentPattern(line string) *CommentPattern {
	trimmedLine := strings.TrimSpace(line)

	// Check patterns in order of specificity (longer patterns first)
	for _, pattern := range bp.commentPatterns {
		if strings.Contains(trimmedLine, pattern.StartPattern) {
			// Return a copy of the pattern to avoid modifying the original
			patternCopy := pattern
			return &patternCopy
		}
	}
	return nil
}

// extractAnnotationContent extracts the content of a ServiceSpec annotation
func (bp *BaseFileParser) extractAnnotationContent(lines []string, startLine int, pattern *CommentPattern) (string, int) {
	var content strings.Builder
	currentLine := startLine

	if pattern.IsMultiLine {
		// Multi-line comment (/* */ or /** */)
		inAnnotation := false

		for currentLine < len(lines) {
			line := lines[currentLine]

			if strings.Contains(line, "@ServiceSpec") {
				inAnnotation = true
				currentLine++
				continue // Skip the @ServiceSpec line itself
			}

			if inAnnotation {
				// Check for end of comment first
				if strings.Contains(line, pattern.EndPattern) {
					break
				}

				// Clean the line
				cleaned := bp.cleanCommentLine(line, pattern)
				if cleaned != "" {
					content.WriteString(cleaned)
					content.WriteString("\n")
				}
			}

			currentLine++
		}
	} else {
		// Single-line comments (//)
		foundServiceSpec := false

		for currentLine < len(lines) {
			line := lines[currentLine]
			trimmedLine := strings.TrimSpace(line)

			// Check if this line is still part of the comment
			if !strings.HasPrefix(trimmedLine, strings.TrimSpace(pattern.StartPattern)) {
				break
			}

			if strings.Contains(line, "@ServiceSpec") {
				foundServiceSpec = true
				currentLine++
				continue // Skip the @ServiceSpec line itself
			}

			if foundServiceSpec {
				cleaned := bp.cleanCommentLine(line, pattern)
				if cleaned != "" {
					content.WriteString(cleaned)
					content.WriteString("\n")
				}
			}

			currentLine++
		}
		currentLine-- // Adjust for the last line that wasn't part of the comment
	}

	return content.String(), currentLine
}

// cleanCommentLine removes comment markers from a line while preserving indentation
func (bp *BaseFileParser) cleanCommentLine(line string, pattern *CommentPattern) string {
	cleaned := line

	// Remove start pattern
	if pattern.StartPattern != "" {
		cleaned = strings.Replace(cleaned, pattern.StartPattern, "", 1)
	}

	// Remove end pattern
	if pattern.EndPattern != "" {
		cleaned = strings.Replace(cleaned, pattern.EndPattern, "", 1)
	}

	// For multi-line comments, remove the line prefix but preserve indentation after it
	if pattern.IsMultiLine && pattern.LinePrefix != "" {
		// Find the line prefix and remove it, but keep any indentation after it
		prefixIndex := strings.Index(cleaned, strings.TrimSpace(pattern.LinePrefix))
		if prefixIndex >= 0 {
			// Remove everything up to and including the prefix
			cleaned = cleaned[prefixIndex+len(strings.TrimSpace(pattern.LinePrefix)):]
		}
	} else if pattern.LinePrefix != "" {
		// For single-line comments, remove the prefix but preserve indentation
		prefixToRemove := strings.TrimSpace(pattern.LinePrefix)
		if strings.HasPrefix(strings.TrimSpace(cleaned), prefixToRemove) {
			// Find the prefix position and remove it, keeping leading whitespace
			trimmed := strings.TrimSpace(cleaned)
			prefixIndex := strings.Index(trimmed, prefixToRemove)
			if prefixIndex == 0 {
				// Remove the prefix and any immediately following space
				remaining := trimmed[len(prefixToRemove):]
				if strings.HasPrefix(remaining, " ") {
					remaining = remaining[1:]
				}
				// Preserve the original indentation structure for YAML
				leadingSpaces := len(cleaned) - len(strings.TrimLeft(cleaned, " 	"))
				if leadingSpaces > 0 && remaining != "" {
					// Keep some indentation for YAML structure
					cleaned = strings.Repeat(" ", max(0, leadingSpaces-2)) + remaining
				} else {
					cleaned = remaining
				}
			}
		}
	}

	// Remove @ServiceSpec marker if present
	cleaned = regexp.MustCompile(`@ServiceSpec\s*`).ReplaceAllString(cleaned, "")

	// Don't trim the result to preserve indentation
	return cleaned
}

// parseAnnotationContent parses the extracted annotation content
func (bp *BaseFileParser) parseAnnotationContent(content string, annotation *ServiceSpecAnnotation, filepath string) error {
	if content == "" {
		return fmt.Errorf("empty ServiceSpec annotation")
	}

	// Try to parse as YAML first (more flexible)
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		// If YAML fails, try JSON
		if jsonErr := json.Unmarshal([]byte(content), &data); jsonErr != nil {
			return fmt.Errorf("failed to parse annotation content as YAML or JSON: %s", err.Error())
		}
	}

	// Extract required fields
	if operationID, ok := data["operationId"].(string); ok {
		annotation.OperationID = operationID
	} else {
		return fmt.Errorf("missing or invalid operationId field")
	}

	if description, ok := data["description"].(string); ok {
		annotation.Description = description
	} else {
		return fmt.Errorf("missing or invalid description field")
	}

	// Extract optional fields
	if preconditions, ok := data["preconditions"]; ok {
		annotation.Preconditions = preconditions
	}

	if postconditions, ok := data["postconditions"]; ok {
		annotation.Postconditions = postconditions
	}

	return nil
}

// convertAnnotationToSpec converts a ServiceSpecAnnotation to a models.ServiceSpec
func (bp *BaseFileParser) convertAnnotationToSpec(annotation ServiceSpecAnnotation, filepath string) (models.ServiceSpec, error) {
	spec := models.ServiceSpec{
		OperationID: annotation.OperationID,
		Description: annotation.Description,
		SourceFile:  filepath,
		LineNumber:  annotation.StartLine,
	}

	// Convert preconditions to map[string]interface{}
	if annotation.Preconditions != nil {
		if preconditions, ok := annotation.Preconditions.(map[string]interface{}); ok {
			spec.Preconditions = preconditions
		} else {
			return spec, fmt.Errorf("preconditions must be a map/object")
		}
	} else {
		spec.Preconditions = make(map[string]interface{})
	}

	// Convert postconditions to map[string]interface{}
	if annotation.Postconditions != nil {
		if postconditions, ok := annotation.Postconditions.(map[string]interface{}); ok {
			spec.Postconditions = postconditions
		} else {
			return spec, fmt.Errorf("postconditions must be a map/object")
		}
	} else {
		spec.Postconditions = make(map[string]interface{})
	}

	// Validate the spec
	if err := spec.Validate(); err != nil {
		return spec, fmt.Errorf("invalid ServiceSpec: %s", err.Error())
	}

	return spec, nil
}

// CanParse checks if this parser can handle the given file
func (bp *BaseFileParser) CanParse(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	switch bp.language {
	case LanguageJava:
		return ext == javaExtension
	case LanguageTypeScript:
		return ext == ".ts" || ext == ".tsx"
	case LanguageGo:
		return ext == ".go"
	default:
		return false
	}
}

// GetLanguage returns the language this parser handles
func (bp *BaseFileParser) GetLanguage() SupportedLanguage {
	return bp.language
}

// ValidateJSONLogic validates that the given data can be processed by JSONLogic
func (bp *BaseFileParser) ValidateJSONLogic(data interface{}) error {
	// Basic validation - check if it's a valid JSON structure
	if data == nil {
		return nil // Empty conditions are allowed
	}

	// Try to marshal and unmarshal to ensure it's valid JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("invalid JSON structure: %s", err.Error())
	}

	var result interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		return fmt.Errorf("invalid JSON structure: %s", err.Error())
	}

	return nil
}

// Helper function to convert line numbers (for debugging)
func (bp *BaseFileParser) formatLineNumber(line int) string {
	return strconv.Itoa(line)
}

// Helper function to get the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
