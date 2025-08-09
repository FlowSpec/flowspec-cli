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

package main

import (
	"fmt"
	"runtime"
)

// Version information
var (
	// Version is the current version of FlowSpec CLI
	Version = "0.2.0"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// BuildDate is the build date
	BuildDate = "unknown"

	// GoVersion is the Go version used to build
	GoVersion = runtime.Version()

	// Platform is the target platform
	Platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// GetVersionInfo returns formatted version information
func GetVersionInfo() string {
	return fmt.Sprintf(`FlowSpec CLI %s
Git Commit: %s
Build Date: %s
Go Version: %s
Platform: %s`, Version, GitCommit, BuildDate, GoVersion, Platform)
}

// GetShortVersion returns just the version number
func GetShortVersion() string {
	return Version
}
