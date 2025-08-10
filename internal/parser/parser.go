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
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/flowspec/flowspec-cli/internal/models"
)

// Hash format constants
const (
	hexFormat = "%x"
)

// SpecParser defines the interface for parsing ServiceSpecs from source code
type SpecParser interface {
	ParseFromSource(sourcePath string) (*models.ParseResult, error)
}

// FileParser defines the interface for parsing individual files
type FileParser interface {
	CanParse(filename string) bool
	ParseFile(filepath string) ([]models.ServiceSpec, []models.ParseError)
}

// SupportedLanguage represents a supported programming language
type SupportedLanguage string

const (
	LanguageJava       SupportedLanguage = "java"
	LanguageTypeScript SupportedLanguage = "typescript"
	LanguageGo         SupportedLanguage = "go"
)

// FileType represents the mapping between file extensions and languages
type FileType struct {
	Extensions []string
	Language   SupportedLanguage
}

// DefaultSpecParser implements the SpecParser interface
type DefaultSpecParser struct {
	fileParsers    map[SupportedLanguage]FileParser
	supportedTypes []FileType
	maxWorkers     int
	cache          *ParseCache
	mu             sync.RWMutex
}

// ParserConfig holds configuration for the parser
type ParserConfig struct {
	MaxWorkers    int
	SkipHidden    bool
	SkipVendor    bool
	MaxFileSize   int64 // Maximum file size in bytes
	AllowedDirs   []string
	ExcludedDirs  []string
	EnableCache   bool
	CacheSize     int
	EnableMetrics bool
}

// ParseCache provides caching for parsed files
type ParseCache struct {
	entries map[string]*CacheEntry
	maxSize int
	mu      sync.RWMutex
}

// CacheEntry represents a cached parse result
type CacheEntry struct {
	FilePath     string
	FileHash     string
	ModTime      time.Time
	Specs        []models.ServiceSpec
	Errors       []models.ParseError
	LastAccessed time.Time
}

// ParseMetrics tracks parsing performance
type ParseMetrics struct {
	TotalFiles     int
	ProcessedFiles int
	CacheHits      int
	CacheMisses    int
	TotalSpecs     int
	TotalErrors    int
	ParseDuration  time.Duration
	StartTime      time.Time
	mu             sync.RWMutex
}

// DefaultParserConfig returns a default parser configuration
func DefaultParserConfig() *ParserConfig {
	return &ParserConfig{
		MaxWorkers:    4,
		SkipHidden:    true,
		SkipVendor:    true,
		MaxFileSize:   10 * 1024 * 1024, // 10MB
		AllowedDirs:   []string{},
		ExcludedDirs:  []string{"node_modules", "vendor", ".git", "build", "dist", "target"},
		EnableCache:   true,
		CacheSize:     1000,
		EnableMetrics: true,
	}
}

// NewParseCache creates a new parse cache
func NewParseCache(maxSize int) *ParseCache {
	return &ParseCache{
		entries: make(map[string]*CacheEntry),
		maxSize: maxSize,
	}
}

// NewParseMetrics creates a new parse metrics tracker
func NewParseMetrics() *ParseMetrics {
	return &ParseMetrics{
		StartTime: time.Now(),
	}
}

// NewSpecParser creates a new SpecParser with default configuration
func NewSpecParser() *DefaultSpecParser {
	config := DefaultParserConfig()
	parser := &DefaultSpecParser{
		fileParsers:    make(map[SupportedLanguage]FileParser),
		supportedTypes: getSupportedFileTypes(),
		maxWorkers:     config.MaxWorkers,
		cache:          NewParseCache(config.CacheSize),
	}

	// Register default file parsers
	parser.RegisterFileParser(LanguageJava, NewJavaFileParser())
	parser.RegisterFileParser(LanguageTypeScript, NewTypeScriptFileParser())
	parser.RegisterFileParser(LanguageGo, NewGoFileParser())

	return parser
}

// NewSpecParserWithConfig creates a new SpecParser with custom configuration
func NewSpecParserWithConfig(config *ParserConfig) *DefaultSpecParser {
	parser := &DefaultSpecParser{
		fileParsers:    make(map[SupportedLanguage]FileParser),
		supportedTypes: getSupportedFileTypes(),
		maxWorkers:     config.MaxWorkers,
	}

	if config.EnableCache {
		parser.cache = NewParseCache(config.CacheSize)
	}

	// Register default file parsers
	parser.RegisterFileParser(LanguageJava, NewJavaFileParser())
	parser.RegisterFileParser(LanguageTypeScript, NewTypeScriptFileParser())
	parser.RegisterFileParser(LanguageGo, NewGoFileParser())

	return parser
}

// RegisterFileParser registers a file parser for a specific language
func (p *DefaultSpecParser) RegisterFileParser(language SupportedLanguage, parser FileParser) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.fileParsers[language] = parser
}

// GetFileParser returns the file parser for a specific language
func (p *DefaultSpecParser) GetFileParser(language SupportedLanguage) (FileParser, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	parser, exists := p.fileParsers[language]
	return parser, exists
}

// ParseFromSource implements the SpecParser interface
func (p *DefaultSpecParser) ParseFromSource(sourcePath string) (*models.ParseResult, error) {
	startTime := time.Now()
	metrics := NewParseMetrics()

	// Validate source path
	if sourcePath == "" {
		return nil, fmt.Errorf("source path cannot be empty")
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to access source path %s: %w", sourcePath, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("source path %s is not a directory", sourcePath)
	}

	// Scan for supported files
	files, err := p.scanFiles(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan files: %w", err)
	}

	metrics.TotalFiles = len(files)

	if len(files) == 0 {
		return &models.ParseResult{
			Specs:  []models.ServiceSpec{},
			Errors: []models.ParseError{},
		}, nil
	}

	// Parse files concurrently
	result, err := p.parseFilesWithMetrics(files, metrics)
	if err != nil {
		return nil, err
	}

	// Update metrics
	metrics.ProcessedFiles = len(files)
	metrics.TotalSpecs = len(result.Specs)
	metrics.TotalErrors = len(result.Errors)
	metrics.SetParseDuration(time.Since(startTime))

	// Add metrics to result if available
	if result != nil {
		result.Metrics = metrics.GetSummary()
	}

	return result, nil
}

// scanFiles recursively scans the directory for supported source files
func (p *DefaultSpecParser) scanFiles(rootPath string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log error but continue walking
			return nil
		}

		// Skip directories
		if info.IsDir() {
			// Check if directory should be excluded
			if p.shouldSkipDirectory(path, info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file should be processed
		if p.shouldProcessFile(path, info) {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// shouldSkipDirectory determines if a directory should be skipped during scanning
func (p *DefaultSpecParser) shouldSkipDirectory(path, name string) bool {
	// Skip hidden directories
	if strings.HasPrefix(name, ".") && name != "." {
		return true
	}

	// Skip common vendor/build directories
	excludedDirs := []string{
		"node_modules", "vendor", ".git", "build", "dist", "target",
		".idea", ".vscode", "__pycache__", ".gradle", ".mvn",
	}

	for _, excluded := range excludedDirs {
		if name == excluded {
			return true
		}
	}

	return false
}

// shouldProcessFile determines if a file should be processed
func (p *DefaultSpecParser) shouldProcessFile(path string, info os.FileInfo) bool {
	// Skip if file is too large
	if info.Size() > 10*1024*1024 { // 10MB limit
		return false
	}

	// Check if file extension is supported
	return p.isSupportedFile(path)
}

// isSupportedFile checks if a file has a supported extension
func (p *DefaultSpecParser) isSupportedFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))

	for _, fileType := range p.supportedTypes {
		for _, supportedExt := range fileType.Extensions {
			if ext == supportedExt {
				return true
			}
		}
	}

	return false
}

// getLanguageFromFile determines the programming language from file extension
func (p *DefaultSpecParser) getLanguageFromFile(filename string) (SupportedLanguage, error) {
	ext := strings.ToLower(filepath.Ext(filename))

	for _, fileType := range p.supportedTypes {
		for _, supportedExt := range fileType.Extensions {
			if ext == supportedExt {
				return fileType.Language, nil
			}
		}
	}

	return "", fmt.Errorf("unsupported file extension: %s", ext)
}

// parseFiles parses multiple files concurrently
func (p *DefaultSpecParser) parseFiles(files []string) (*models.ParseResult, error) {
	return p.parseFilesWithMetrics(files, nil)
}

// parseFilesWithMetrics parses multiple files concurrently with metrics tracking
func (p *DefaultSpecParser) parseFilesWithMetrics(files []string, metrics *ParseMetrics) (*models.ParseResult, error) {
	if len(files) == 0 {
		return &models.ParseResult{
			Specs:  []models.ServiceSpec{},
			Errors: []models.ParseError{},
		}, nil
	}

	// Create channels for results
	specsChan := make(chan []models.ServiceSpec, len(files))
	errorsChan := make(chan []models.ParseError, len(files))

	// Create worker pool
	workers := p.maxWorkers
	if workers > len(files) {
		workers = len(files)
	}

	fileChan := make(chan string, len(files))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			p.parseWorkerWithMetrics(fileChan, specsChan, errorsChan, metrics)
		}()
	}

	// Send files to workers
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Wait for workers to complete
	wg.Wait()
	close(specsChan)
	close(errorsChan)

	// Collect results
	var allSpecs []models.ServiceSpec
	var allErrors []models.ParseError

	for specs := range specsChan {
		allSpecs = append(allSpecs, specs...)
	}

	for errors := range errorsChan {
		allErrors = append(allErrors, errors...)
	}

	return &models.ParseResult{
		Specs:  allSpecs,
		Errors: allErrors,
	}, nil
}

// parseWorker is a worker function that processes files from the channel
func (p *DefaultSpecParser) parseWorker(fileChan <-chan string, specsChan chan<- []models.ServiceSpec, errorsChan chan<- []models.ParseError) {
	p.parseWorkerWithMetrics(fileChan, specsChan, errorsChan, nil)
}

// parseWorkerWithMetrics is a worker function that processes files from the channel with metrics tracking
func (p *DefaultSpecParser) parseWorkerWithMetrics(fileChan <-chan string, specsChan chan<- []models.ServiceSpec, errorsChan chan<- []models.ParseError, metrics *ParseMetrics) {
	for file := range fileChan {
		specs, errors := p.parseFileWithMetrics(file, metrics)
		specsChan <- specs
		errorsChan <- errors
	}
}

// parseFileWithMetrics parses a single file with caching support and metrics tracking
func (p *DefaultSpecParser) parseFileWithMetrics(filepath string, metrics *ParseMetrics) ([]models.ServiceSpec, []models.ParseError) {
	// Get file info for caching
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return nil, []models.ParseError{{
			File:    filepath,
			Line:    0,
			Message: fmt.Sprintf("failed to stat file: %s", err.Error()),
		}}
	}

	// Check cache first
	if p.cache != nil {
		if entry, found := p.cache.Get(filepath, fileInfo.ModTime()); found {
			if metrics != nil {
				metrics.IncrementCacheHit()
			}
			return entry.Specs, entry.Errors
		}
		if metrics != nil {
			metrics.IncrementCacheMiss()
		}
	}

	// Determine language from file extension
	language, err := p.getLanguageFromFile(filepath)
	if err != nil {
		return nil, []models.ParseError{{
			File:    filepath,
			Line:    0,
			Message: fmt.Sprintf("unsupported file type: %s", err.Error()),
		}}
	}

	// Get appropriate file parser
	fileParser, exists := p.GetFileParser(language)
	if !exists {
		return nil, []models.ParseError{{
			File:    filepath,
			Line:    0,
			Message: fmt.Sprintf("no parser registered for language: %s", language),
		}}
	}

	// Parse the file
	specs, errors := fileParser.ParseFile(filepath)

	// Cache the result
	if p.cache != nil {
		p.cache.Put(filepath, fileInfo.ModTime(), specs, errors)
	}

	return specs, errors
}

// getSupportedFileTypes returns the list of supported file types
func getSupportedFileTypes() []FileType {
	return []FileType{
		{
			Extensions: []string{".java"},
			Language:   LanguageJava,
		},
		{
			Extensions: []string{".ts", ".tsx"},
			Language:   LanguageTypeScript,
		},
		{
			Extensions: []string{".go"},
			Language:   LanguageGo,
		},
	}
}

// GetSupportedLanguages returns a list of all supported languages
func (p *DefaultSpecParser) GetSupportedLanguages() []SupportedLanguage {
	var languages []SupportedLanguage
	for _, fileType := range p.supportedTypes {
		languages = append(languages, fileType.Language)
	}
	return languages
}

// GetSupportedExtensions returns a list of all supported file extensions
func (p *DefaultSpecParser) GetSupportedExtensions() []string {
	var extensions []string
	for _, fileType := range p.supportedTypes {
		extensions = append(extensions, fileType.Extensions...)
	}
	return extensions
}

// IsLanguageSupported checks if a language is supported
func (p *DefaultSpecParser) IsLanguageSupported(language SupportedLanguage) bool {
	for _, fileType := range p.supportedTypes {
		if fileType.Language == language {
			return true
		}
	}
	return false
}

// GetFileCount returns the number of registered file parsers
func (p *DefaultSpecParser) GetFileCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.fileParsers)
}

// GetCacheSize returns the current cache size
func (p *DefaultSpecParser) GetCacheSize() int {
	if p.cache == nil {
		return 0
	}
	return p.cache.Size()
}

// ClearCache clears the parse cache
func (p *DefaultSpecParser) ClearCache() {
	if p.cache != nil {
		p.cache.Clear()
	}
}

// SetMaxWorkers updates the maximum number of worker goroutines
func (p *DefaultSpecParser) SetMaxWorkers(maxWorkers int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if maxWorkers > 0 {
		p.maxWorkers = maxWorkers
	}
}

// ErrorCollector provides utilities for collecting and managing parse errors
type ErrorCollector struct {
	errors []models.ParseError
	mu     sync.Mutex
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]models.ParseError, 0),
	}
}

// AddError adds a parse error to the collector
func (ec *ErrorCollector) AddError(file string, line int, message string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.errors = append(ec.errors, models.ParseError{
		File:    file,
		Line:    line,
		Message: message,
	})
}

// AddErrorf adds a formatted parse error to the collector
func (ec *ErrorCollector) AddErrorf(file string, line int, format string, args ...interface{}) {
	ec.AddError(file, line, fmt.Sprintf(format, args...))
}

// GetErrors returns all collected errors
func (ec *ErrorCollector) GetErrors() []models.ParseError {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	// Return a copy to prevent external modification
	result := make([]models.ParseError, len(ec.errors))
	copy(result, ec.errors)
	return result
}

// HasErrors returns true if any errors have been collected
func (ec *ErrorCollector) HasErrors() bool {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	return len(ec.errors) > 0
}

// Clear removes all collected errors
func (ec *ErrorCollector) Clear() {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.errors = ec.errors[:0]
}

// Count returns the number of collected errors
func (ec *ErrorCollector) Count() int {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	return len(ec.errors)
}

// Cache methods

// Get retrieves a cached entry if it exists and is still valid
func (pc *ParseCache) Get(filePath string, modTime time.Time) (*CacheEntry, bool) {
	if pc == nil {
		return nil, false
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	entry, exists := pc.entries[filePath]
	if !exists {
		return nil, false
	}

	// Check if file has been modified
	if !entry.ModTime.Equal(modTime) {
		return nil, false
	}

	// Update last accessed time
	entry.LastAccessed = time.Now()
	return entry, true
}

// Put stores a parse result in the cache
func (pc *ParseCache) Put(filePath string, modTime time.Time, specs []models.ServiceSpec, errors []models.ParseError) {
	if pc == nil {
		return
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()

	// If cache is full, remove least recently used entry
	if len(pc.entries) >= pc.maxSize {
		pc.evictLRU()
	}

	// Calculate file hash for integrity check
	hash, err := pc.calculateFileHash(filePath)
	if err != nil {
		// If hash calculation fails, use empty string but still cache the result
		hash = ""
	}

	entry := &CacheEntry{
		FilePath:     filePath,
		FileHash:     hash,
		ModTime:      modTime,
		Specs:        specs,
		Errors:       errors,
		LastAccessed: time.Now(),
	}

	pc.entries[filePath] = entry
}

// evictLRU removes the least recently used entry from cache
func (pc *ParseCache) evictLRU() {
	var oldestPath string
	var oldestTime time.Time

	for path, entry := range pc.entries {
		if oldestPath == "" || entry.LastAccessed.Before(oldestTime) {
			oldestPath = path
			oldestTime = entry.LastAccessed
		}
	}

	if oldestPath != "" {
		delete(pc.entries, oldestPath)
	}
}

// calculateFileHash calculates MD5 hash of file content
func (pc *ParseCache) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf(hexFormat, hash.Sum(nil)), nil
}

// Clear removes all entries from the cache
func (pc *ParseCache) Clear() {
	if pc == nil {
		return
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.entries = make(map[string]*CacheEntry)
}

// Size returns the current number of cached entries
func (pc *ParseCache) Size() int {
	if pc == nil {
		return 0
	}

	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return len(pc.entries)
}

// Metrics methods

// IncrementCacheHit increments the cache hit counter
func (pm *ParseMetrics) IncrementCacheHit() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.CacheHits++
}

// IncrementCacheMiss increments the cache miss counter
func (pm *ParseMetrics) IncrementCacheMiss() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.CacheMisses++
}

// AddFile increments the total file counter
func (pm *ParseMetrics) AddFile() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.TotalFiles++
}

// AddProcessedFile increments the processed file counter
func (pm *ParseMetrics) AddProcessedFile() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.ProcessedFiles++
}

// AddSpecs adds to the total specs counter
func (pm *ParseMetrics) AddSpecs(count int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.TotalSpecs += count
}

// AddErrors adds to the total errors counter
func (pm *ParseMetrics) AddErrors(count int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.TotalErrors += count
}

// SetParseDuration sets the total parse duration
func (pm *ParseMetrics) SetParseDuration(duration time.Duration) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.ParseDuration = duration
}

// GetSummary returns a summary of parsing metrics
func (pm *ParseMetrics) GetSummary() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return map[string]interface{}{
		"total_files":      pm.TotalFiles,
		"processed_files":  pm.ProcessedFiles,
		"cache_hits":       pm.CacheHits,
		"cache_misses":     pm.CacheMisses,
		"total_specs":      pm.TotalSpecs,
		"total_errors":     pm.TotalErrors,
		"parse_duration":   pm.ParseDuration.String(),
		"files_per_second": float64(pm.ProcessedFiles) / pm.ParseDuration.Seconds(),
	}
}
