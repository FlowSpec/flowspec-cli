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

// JavaFileParser implements FileParser for Java source files
type JavaFileParser struct {
	*BaseFileParser
}

// NewJavaFileParser creates a new Java file parser
func NewJavaFileParser() *JavaFileParser {
	return &JavaFileParser{
		BaseFileParser: NewBaseFileParser(LanguageJava),
	}
}

// CanParse checks if this parser can handle the given file
func (jp *JavaFileParser) CanParse(filename string) bool {
	return jp.BaseFileParser.CanParse(filename)
}

// ParseFile parses a Java file and extracts ServiceSpec annotations
func (jp *JavaFileParser) ParseFile(filepath string) ([]models.ServiceSpec, []models.ParseError) {
	return jp.BaseFileParser.ParseFile(filepath)
}