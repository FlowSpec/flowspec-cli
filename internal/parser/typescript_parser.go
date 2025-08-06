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
	"github.com/flowspec/flowspec-cli/internal/models"
)

// TypeScriptFileParser implements FileParser for TypeScript source files
type TypeScriptFileParser struct {
	*BaseFileParser
}

// NewTypeScriptFileParser creates a new TypeScript file parser
func NewTypeScriptFileParser() *TypeScriptFileParser {
	return &TypeScriptFileParser{
		BaseFileParser: NewBaseFileParser(LanguageTypeScript),
	}
}

// CanParse checks if this parser can handle the given file
func (tp *TypeScriptFileParser) CanParse(filename string) bool {
	return tp.BaseFileParser.CanParse(filename)
}

// ParseFile parses a TypeScript file and extracts ServiceSpec annotations
func (tp *TypeScriptFileParser) ParseFile(filepath string) ([]models.ServiceSpec, []models.ParseError) {
	return tp.BaseFileParser.ParseFile(filepath)
}
