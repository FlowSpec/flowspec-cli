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

package renderer

import (
	"fmt"
	"os"
)

// LogoRenderer handles ASCII logo and branding elements
type LogoRenderer struct {
	colorOutput bool
	isTTY       bool
}

// NewLogoRenderer creates a new logo renderer
func NewLogoRenderer(colorOutput, isTTY bool) *LogoRenderer {
	return &LogoRenderer{
		colorOutput: colorOutput,
		isTTY:       isTTY,
	}
}

// ShouldShowLogo determines if ASCII logo should be displayed
// Only shows logo in TTY with color support and respects NO_COLOR environment variable
func (l *LogoRenderer) ShouldShowLogo() bool {
	// Check NO_COLOR environment variable
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Only show logo when we have TTY and color support
	return l.isTTY && l.colorOutput
}

// GetASCIILogo returns the FlowSpec ASCII logo with branding
func (l *LogoRenderer) GetASCIILogo() string {
	if !l.ShouldShowLogo() {
		return ""
	}

	logo := fmt.Sprintf(`%s
 ███████╗██╗      ██████╗ ██╗    ██╗███████╗██████╗ ███████╗ ██████╗
 ██╔════╝██║     ██╔═══██╗██║    ██║██╔════╝██╔══██╗██╔════╝██╔════╝
 █████╗  ██║     ██║   ██║██║ █╗ ██║███████╗██████╔╝█████╗  ██║     
 ██╔══╝  ██║     ██║   ██║██║███╗██║╚════██║██╔═══╝ ██╔══╝  ██║     
 ██║     ███████╗╚██████╔╝╚███╔███╔╝███████║██║     ███████╗╚██████╗
 ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ ╚══════╝╚═╝     ╚══════╝ ╚═════╝
%s
 %sEnsure your services behave as specified%s
`, l.getColor("green"), l.getColor("reset"), l.getColor("dim"), l.getColor("reset"))

	return logo
}

// GetSuccessLogo returns a success-themed logo variant
func (l *LogoRenderer) GetSuccessLogo() string {
	if !l.ShouldShowLogo() {
		return ""
	}

	logo := fmt.Sprintf(`%s
 ███████╗██╗      ██████╗ ██╗    ██╗███████╗██████╗ ███████╗ ██████╗
 ██╔════╝██║     ██╔═══██╗██║    ██║██╔════╝██╔══██╗██╔════╝██╔════╝
 █████╗  ██║     ██║   ██║██║ █╗ ██║███████╗██████╔╝█████╗  ██║     
 ██╔══╝  ██║     ██║   ██║██║███╗██║╚════██║██╔═══╝ ██╔══╝  ██║     
 ██║     ███████╗╚██████╔╝╚███╔███╔╝███████║██║     ███████╗╚██████╗
 ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ ╚══════╝╚═╝     ╚══════╝ ╚═════╝
%s
 %s✅ All specifications validated successfully%s
`, l.getColor("green"), l.getColor("reset"), l.getColor("green"), l.getColor("reset"))

	return logo
}

// GetFailureLogo returns a failure-themed logo variant
func (l *LogoRenderer) GetFailureLogo() string {
	if !l.ShouldShowLogo() {
		return ""
	}

	logo := fmt.Sprintf(`%s
 ███████╗██╗      ██████╗ ██╗    ██╗███████╗██████╗ ███████╗ ██████╗
 ██╔════╝██║     ██╔═══██╗██║    ██║██╔════╝██╔══██╗██╔════╝██╔════╝
 █████╗  ██║     ██║   ██║██║ █╗ ██║███████╗██████╔╝█████╗  ██║     
 ██╔══╝  ██║     ██║   ██║██║███╗██║╚════██║██╔═══╝ ██╔══╝  ██║     
 ██║     ███████╗╚██████╔╝╚███╔███╔╝███████║██║     ███████╗╚██████╗
 ╚═╝     ╚══════╝ ╚═════╝  ╚══╝╚══╝ ╚══════╝╚═╝     ╚══════╝ ╚═════╝
%s
 %s❌ Specification validation failed%s
`, l.getColor("red"), l.getColor("reset"), l.getColor("red"), l.getColor("reset"))

	return logo
}

// GetBrandingMessage returns the value proposition tagline
func (l *LogoRenderer) GetBrandingMessage() string {
	if !l.colorOutput {
		return "Ensure your services behave as specified"
	}

	return fmt.Sprintf("%sEnsure your services behave as specified%s",
		l.getColor("dim"), l.getColor("reset"))
}

// getColor returns ANSI color codes if color output is enabled
func (l *LogoRenderer) getColor(colorName string) string {
	if !l.colorOutput {
		return ""
	}

	colors := map[string]string{
		"reset":   "\033[0m",
		"bold":    "\033[1m",
		"dim":     "\033[2m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
	}

	if color, exists := colors[colorName]; exists {
		return color
	}
	return ""
}