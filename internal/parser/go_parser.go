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

// GoFileParser implements FileParser for Go source files
type GoFileParser struct {
	*BaseFileParser
}

// NewGoFileParser creates a new Go file parser
func NewGoFileParser() *GoFileParser {
	return &GoFileParser{
		BaseFileParser: NewBaseFileParser(LanguageGo),
	}
}

// CanParse checks if this parser can handle the given file
func (gp *GoFileParser) CanParse(filename string) bool {
	return gp.BaseFileParser.CanParse(filename)
}

// ParseFile parses a Go file and extracts ServiceSpec annotations
func (gp *GoFileParser) ParseFile(filepath string) ([]models.ServiceSpec, []models.ParseError) {
	return gp.BaseFileParser.ParseFile(filepath)
}