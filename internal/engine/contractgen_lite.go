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

package engine

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/flowspec/flowspec-cli/internal/ingestor"
	"github.com/flowspec/flowspec-cli/internal/ingestor/traffic"
	"github.com/flowspec/flowspec-cli/internal/models"
)

// ContractGenerator defines the interface for generating service contracts from traffic data
type ContractGenerator interface {
	// GenerateSpec processes traffic records and generates a ServiceSpec
	GenerateSpec(it ingestor.Iterator[*traffic.NormalizedRecord]) (*models.ServiceSpec, error)
	
	// SetOptions configures the generation behavior
	SetOptions(options *GenerationOptions)
}

// GenerationOptions configures contract generation behavior
type GenerationOptions struct {
	// PathClusteringThreshold defines the unique value ratio threshold for parameterization (default ≥0.8)
	PathClusteringThreshold float64 `json:"pathClusteringThreshold"`
	
	// MinSampleSize defines the minimum sample size required for parameterization (default ≥20)
	MinSampleSize int `json:"minSampleSize"`
	
	// RequiredFieldThreshold defines the appearance ratio threshold for required fields (default ≥0.95)
	RequiredFieldThreshold float64 `json:"requiredFieldThreshold"`
	
	// MinEndpointSamples defines the minimum samples required for an endpoint to be included (default ≥5)
	MinEndpointSamples int `json:"minEndpointSamples"`
	
	// StatusAggregation defines the status code aggregation strategy ("range"|"exact"|"auto")
	StatusAggregation string `json:"statusAggregation"`
	
	// MaxUniqueValues defines the maximum unique values to track per path segment (default 10000)
	MaxUniqueValues int `json:"maxUniqueValues"`
	
	// ServiceName defines the name for the generated service spec
	ServiceName string `json:"serviceName"`
	
	// ServiceVersion defines the version for the generated service spec
	ServiceVersion string `json:"serviceVersion"`
}

// DefaultGenerationOptions returns default generation options
func DefaultGenerationOptions() *GenerationOptions {
	return &GenerationOptions{
		PathClusteringThreshold: 0.8,
		MinSampleSize:          20,
		RequiredFieldThreshold: 0.95,
		MinEndpointSamples:     5,
		StatusAggregation:      "auto",
		MaxUniqueValues:        10000,
		ServiceName:            "generated-service",
		ServiceVersion:         "v1.0.0",
	}
}

// EndpointPattern represents a discovered endpoint pattern with its operations
type EndpointPattern struct {
	Pattern     string                        `json:"pattern"`
	Operations  map[string]*OperationPattern  `json:"operations"` // method -> pattern
	SampleCount int                           `json:"sampleCount"`
}

// OperationPattern represents a discovered operation pattern for a specific HTTP method
type OperationPattern struct {
	Method          string            `json:"method"`
	StatusCodes     []int             `json:"statusCodes"`
	StatusRanges    []string          `json:"statusRanges"`
	RequiredQuery   []string          `json:"requiredQuery"`
	RequiredHeaders []string          `json:"requiredHeaders"`
	OptionalQuery   []string          `json:"optionalQuery"`
	OptionalHeaders []string          `json:"optionalHeaders"`
	SampleCount     int               `json:"sampleCount"`
	FirstSeen       time.Time         `json:"firstSeen"`
	LastSeen        time.Time         `json:"lastSeen"`
	
	// Internal tracking for field analysis
	queryFieldCounts   map[string]int `json:"-"`
	headerFieldCounts  map[string]int `json:"-"`
}

// NewOperationPattern creates a new operation pattern
func NewOperationPattern(method string) *OperationPattern {
	return &OperationPattern{
		Method:             method,
		StatusCodes:        make([]int, 0),
		StatusRanges:       make([]string, 0),
		RequiredQuery:      make([]string, 0),
		RequiredHeaders:    make([]string, 0),
		OptionalQuery:      make([]string, 0),
		OptionalHeaders:    make([]string, 0),
		queryFieldCounts:   make(map[string]int),
		headerFieldCounts:  make(map[string]int),
	}
}

// AddRecord adds a traffic record to this operation pattern
func (op *OperationPattern) AddRecord(record *traffic.NormalizedRecord) {
	op.SampleCount++
	
	// Update timestamps
	if op.FirstSeen.IsZero() || record.Timestamp.Before(op.FirstSeen) {
		op.FirstSeen = record.Timestamp
	}
	if op.LastSeen.IsZero() || record.Timestamp.After(op.LastSeen) {
		op.LastSeen = record.Timestamp
	}
	
	// Track status codes
	statusExists := false
	for _, code := range op.StatusCodes {
		if code == record.Status {
			statusExists = true
			break
		}
	}
	if !statusExists {
		op.StatusCodes = append(op.StatusCodes, record.Status)
	}
	
	// Track query parameters
	for key := range record.Query {
		op.queryFieldCounts[key]++
	}
	
	// Track headers
	for key := range record.Headers {
		op.headerFieldCounts[key]++
	}
}

// FinalizeFields analyzes field counts and determines required vs optional fields
func (op *OperationPattern) FinalizeFields(requiredThreshold float64) {
	// Clear existing field lists
	op.RequiredQuery = make([]string, 0)
	op.OptionalQuery = make([]string, 0)
	op.RequiredHeaders = make([]string, 0)
	op.OptionalHeaders = make([]string, 0)
	
	// Analyze query parameters
	for field, count := range op.queryFieldCounts {
		ratio := float64(count) / float64(op.SampleCount)
		if ratio >= requiredThreshold {
			op.RequiredQuery = append(op.RequiredQuery, field)
		} else {
			op.OptionalQuery = append(op.OptionalQuery, field)
		}
	}
	
	// Analyze headers
	for field, count := range op.headerFieldCounts {
		ratio := float64(count) / float64(op.SampleCount)
		if ratio >= requiredThreshold {
			op.RequiredHeaders = append(op.RequiredHeaders, field)
		} else {
			op.OptionalHeaders = append(op.OptionalHeaders, field)
		}
	}
}

// FinalizeStatusCodes applies status code aggregation strategy
func (op *OperationPattern) FinalizeStatusCodes(generator *ContractGeneratorLite) {
	codes, ranges := generator.aggregateStatusCodes(op.StatusCodes, generator.options.StatusAggregation)
	op.StatusCodes = codes
	op.StatusRanges = ranges
}

// ContractGeneratorLite implements the ContractGenerator interface
type ContractGeneratorLite struct {
	options *GenerationOptions
}

// NewContractGeneratorLite creates a new contract generator with default options
func NewContractGeneratorLite() *ContractGeneratorLite {
	return &ContractGeneratorLite{
		options: DefaultGenerationOptions(),
	}
}

// SetOptions configures the generation behavior
func (c *ContractGeneratorLite) SetOptions(options *GenerationOptions) {
	if options != nil {
		c.options = options
	}
}

// GenerateSpec processes traffic records and generates a ServiceSpec
func (c *ContractGeneratorLite) GenerateSpec(it ingestor.Iterator[*traffic.NormalizedRecord]) (*models.ServiceSpec, error) {
	// Collect all records for analysis
	var records []*traffic.NormalizedRecord
	for it.Next() {
		records = append(records, it.Value())
	}
	
	if err := it.Err(); err != nil {
		return nil, err
	}
	
	// Cluster paths and generate patterns
	patterns := c.clusterPaths(records)
	
	// Filter patterns by minimum sample count
	filteredPatterns := make(map[string]*EndpointPattern)
	for pattern, ep := range patterns {
		if ep.SampleCount >= c.options.MinEndpointSamples {
			filteredPatterns[pattern] = ep
		}
	}
	
	// Convert patterns to ServiceSpec
	return c.patternsToServiceSpec(filteredPatterns), nil
}

// clusterPaths analyzes traffic records and clusters similar paths into parameterized patterns
func (c *ContractGeneratorLite) clusterPaths(records []*traffic.NormalizedRecord) map[string]*EndpointPattern {
	// First pass: collect all unique path segments and their values
	segmentAnalysis := c.analyzePathSegments(records)
	
	// Second pass: determine parameterization for each path
	pathPatterns := make(map[string]string) // original path -> pattern
	for _, record := range records {
		if _, exists := pathPatterns[record.Path]; !exists {
			pathPatterns[record.Path] = c.parameterizePath(record.Path, segmentAnalysis)
		}
	}
	
	// Third pass: group records by pattern and build endpoint patterns
	patterns := make(map[string]*EndpointPattern)
	for _, record := range records {
		pattern := pathPatterns[record.Path]
		
		if _, exists := patterns[pattern]; !exists {
			patterns[pattern] = &EndpointPattern{
				Pattern:     pattern,
				Operations:  make(map[string]*OperationPattern),
				SampleCount: 0,
			}
		}
		
		ep := patterns[pattern]
		ep.SampleCount++
		
		// Add to operation pattern
		if _, exists := ep.Operations[record.Method]; !exists {
			ep.Operations[record.Method] = NewOperationPattern(record.Method)
		}
		
		ep.Operations[record.Method].AddRecord(record)
	}
	
	// Fourth pass: finalize field analysis and status codes for all operations
	for _, ep := range patterns {
		for _, op := range ep.Operations {
			op.FinalizeFields(c.options.RequiredFieldThreshold)
			op.FinalizeStatusCodes(c)
		}
	}
	
	// Fifth pass: resolve conflicts (more specific patterns take precedence)
	return c.resolvePatternConflicts(patterns)
}

// PathSegmentAnalysis holds analysis data for a path segment
type PathSegmentAnalysis struct {
	UniqueValues map[string]int // value -> count
	TotalCount   int
	IsLimited    bool // true if we hit the MaxUniqueValues limit
}

// analyzePathSegments analyzes all path segments to determine parameterization candidates
func (c *ContractGeneratorLite) analyzePathSegments(records []*traffic.NormalizedRecord) map[int]*PathSegmentAnalysis {
	// segmentAnalysis[segmentIndex] -> analysis (across all paths with same segment count)
	segmentAnalysis := make(map[int]*PathSegmentAnalysis)
	
	for _, record := range records {
		segments := c.splitPath(record.Path)
		
		for i, segment := range segments {
			if _, exists := segmentAnalysis[i]; !exists {
				segmentAnalysis[i] = &PathSegmentAnalysis{
					UniqueValues: make(map[string]int),
					TotalCount:   0,
					IsLimited:    false,
				}
			}
			
			analysis := segmentAnalysis[i]
			analysis.TotalCount++
			
			// Only track unique values if we haven't hit the limit
			if !analysis.IsLimited {
				if len(analysis.UniqueValues) < c.options.MaxUniqueValues {
					analysis.UniqueValues[segment]++
				} else {
					// Hit the limit, mark as limited and clear the map to save memory
					analysis.IsLimited = true
					analysis.UniqueValues = nil
				}
			}
		}
	}
	
	return segmentAnalysis
}

// parameterizePath converts a path to a parameterized pattern based on segment analysis
func (c *ContractGeneratorLite) parameterizePath(path string, segmentAnalysis map[int]*PathSegmentAnalysis) string {
	segments := c.splitPath(path)
	parameterizedSegments := make([]string, len(segments))
	
	for i, segment := range segments {
		analysis, exists := segmentAnalysis[i]
		if !exists {
			parameterizedSegments[i] = segment
			continue
		}
		
		if c.shouldParameterize(segment, analysis) {
			parameterizedSegments[i] = c.generateParameterName(segment, analysis)
		} else {
			parameterizedSegments[i] = segment
		}
	}
	
	return "/" + strings.Join(parameterizedSegments, "/")
}

// shouldParameterize determines if a path segment should be parameterized
func (c *ContractGeneratorLite) shouldParameterize(segment string, analysis *PathSegmentAnalysis) bool {
	// If we hit the limit, assume high cardinality and parameterize
	if analysis.IsLimited {
		return analysis.TotalCount >= c.options.MinSampleSize
	}
	
	// Check if we have enough samples
	if analysis.TotalCount < c.options.MinSampleSize {
		return false
	}
	
	// Check unique value ratio
	uniqueRatio := float64(len(analysis.UniqueValues)) / float64(analysis.TotalCount)
	return uniqueRatio >= c.options.PathClusteringThreshold
}

// generateParameterName generates an appropriate parameter name based on the segment characteristics
func (c *ContractGeneratorLite) generateParameterName(segment string, analysis *PathSegmentAnalysis) string {
	// If we hit the limit, we can't analyze the values, so use generic {var}
	if analysis.IsLimited {
		return "{var}"
	}
	
	// Analyze the values to determine the best parameter name
	numericCount := 0
	uuidCount := 0
	totalValues := len(analysis.UniqueValues)
	
	for value := range analysis.UniqueValues {
		if c.isNumeric(value) {
			numericCount++
		}
		if c.isUUIDLike(value) {
			uuidCount++
		}
	}
	
	// If ≥90% are numeric, use {num}
	if float64(numericCount)/float64(totalValues) >= 0.9 {
		return "{num}"
	}
	
	// If any are UUID-like, use {id}
	if uuidCount > 0 {
		return "{id}"
	}
	
	// Default to {var}
	return "{var}"
}

// resolvePatternConflicts resolves conflicts between overlapping patterns
// More specific patterns (with more literal segments) take precedence
func (c *ContractGeneratorLite) resolvePatternConflicts(patterns map[string]*EndpointPattern) map[string]*EndpointPattern {
	// Convert to slice for easier processing
	patternList := make([]*EndpointPattern, 0, len(patterns))
	for _, pattern := range patterns {
		patternList = append(patternList, pattern)
	}
	
	// Sort by specificity (more literal segments = more specific)
	sort.Slice(patternList, func(i, j int) bool {
		specificityI := c.calculateSpecificity(patternList[i].Pattern)
		specificityJ := c.calculateSpecificity(patternList[j].Pattern)
		
		// More specific patterns first
		if specificityI != specificityJ {
			return specificityI > specificityJ
		}
		
		// If same specificity, prefer higher sample count
		return patternList[i].SampleCount > patternList[j].SampleCount
	})
	
	// Keep track of which patterns to include
	result := make(map[string]*EndpointPattern)
	
	for _, pattern := range patternList {
		// Check if this pattern conflicts with any already included pattern
		conflicts := false
		for includedPattern := range result {
			if c.patternsConflict(pattern.Pattern, includedPattern) {
				conflicts = true
				break
			}
		}
		
		// If no conflicts, include this pattern
		if !conflicts {
			result[pattern.Pattern] = pattern
		}
	}
	
	return result
}

// calculateSpecificity returns the number of literal (non-parameterized) segments in a pattern
func (c *ContractGeneratorLite) calculateSpecificity(pattern string) int {
	segments := c.splitPath(pattern)
	specificity := 0
	
	for _, segment := range segments {
		if !strings.HasPrefix(segment, "{") || !strings.HasSuffix(segment, "}") {
			specificity++
		}
	}
	
	return specificity
}

// patternsConflict checks if two patterns would match overlapping sets of paths
func (c *ContractGeneratorLite) patternsConflict(pattern1, pattern2 string) bool {
	segments1 := c.splitPath(pattern1)
	segments2 := c.splitPath(pattern2)
	
	// Different number of segments means no conflict
	if len(segments1) != len(segments2) {
		return false
	}
	
	// Check each segment pair
	for i := 0; i < len(segments1); i++ {
		seg1 := segments1[i]
		seg2 := segments2[i]
		
		// If both are literal and different, no conflict
		if !c.isParameter(seg1) && !c.isParameter(seg2) && seg1 != seg2 {
			return false
		}
	}
	
	// If we get here, the patterns could potentially match overlapping paths
	return true
}

// Helper functions

func (c *ContractGeneratorLite) splitPath(path string) []string {
	// Remove leading slash and split
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	
	if path == "" {
		return []string{}
	}
	
	return strings.Split(path, "/")
}

func (c *ContractGeneratorLite) isParameter(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

func (c *ContractGeneratorLite) isNumeric(value string) bool {
	if value == "" {
		return false
	}
	
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}

func (c *ContractGeneratorLite) isUUIDLike(value string) bool {
	// Simple UUID pattern check: 8-4-4-4-12 hex characters with dashes
	// or 32 hex characters without dashes
	if len(value) == 36 {
		// Check pattern: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
		parts := strings.Split(value, "-")
		if len(parts) == 5 && 
		   len(parts[0]) == 8 && len(parts[1]) == 4 && len(parts[2]) == 4 && 
		   len(parts[3]) == 4 && len(parts[4]) == 12 {
			return c.isHex(strings.Join(parts, ""))
		}
	} else if len(value) == 32 {
		// Check if all characters are hex
		return c.isHex(value)
	}
	
	return false
}

func (c *ContractGeneratorLite) isHex(value string) bool {
	for _, char := range strings.ToLower(value) {
		if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
			return false
		}
	}
	return true
}

// aggregateStatusCodes applies the configured status code aggregation strategy
func (c *ContractGeneratorLite) aggregateStatusCodes(statusCodes []int, strategy string) ([]int, []string) {
	if len(statusCodes) == 0 {
		return statusCodes, nil
	}
	
	// Sort status codes for consistent processing
	sort.Ints(statusCodes)
	
	switch strategy {
	case "exact":
		return statusCodes, nil
	case "range":
		return nil, c.statusCodesToRanges(statusCodes)
	case "auto":
		return c.autoAggregateStatusCodes(statusCodes)
	default:
		// Default to auto
		return c.autoAggregateStatusCodes(statusCodes)
	}
}

// statusCodesToRanges converts status codes to range format (2xx, 4xx, etc.)
func (c *ContractGeneratorLite) statusCodesToRanges(statusCodes []int) []string {
	rangeSet := make(map[string]bool)
	
	for _, code := range statusCodes {
		class := code / 100
		if class >= 1 && class <= 5 {
			rangeSet[fmt.Sprintf("%dxx", class)] = true
		}
	}
	
	// Convert to sorted slice
	ranges := make([]string, 0, len(rangeSet))
	for r := range rangeSet {
		ranges = append(ranges, r)
	}
	sort.Strings(ranges)
	
	return ranges
}

// autoAggregateStatusCodes automatically determines the best aggregation strategy
func (c *ContractGeneratorLite) autoAggregateStatusCodes(statusCodes []int) ([]int, []string) {
	if len(statusCodes) <= 1 {
		return statusCodes, nil
	}
	
	// Group by status class (1xx, 2xx, 3xx, 4xx, 5xx)
	classes := make(map[int][]int) // class -> codes in that class
	for _, code := range statusCodes {
		class := code / 100
		classes[class] = append(classes[class], code)
	}
	
	// If all codes are in the same class, use range format
	if len(classes) == 1 {
		return nil, c.statusCodesToRanges(statusCodes)
	}
	
	// Check if we have continuous ranges within classes
	canUseRanges := true
	for class, codes := range classes {
		if len(codes) > 1 {
			// Check if codes in this class are continuous or represent the whole class
			if !c.isClassWellRepresented(class, codes) {
				canUseRanges = false
				break
			}
		}
	}
	
	if canUseRanges {
		return nil, c.statusCodesToRanges(statusCodes)
	}
	
	// Use exact codes for non-continuous or sparse distributions
	return statusCodes, nil
}

// isClassWellRepresented checks if the status codes well represent the entire class
func (c *ContractGeneratorLite) isClassWellRepresented(class int, codes []int) bool {
	// For 2xx class, if we have 200, 201, 204, etc., it's well represented
	// For 4xx class, if we have 400, 404, etc., it's well represented
	// This is a heuristic - we consider a class well represented if:
	// 1. We have at least 3 different codes in the class, OR
	// 2. We have the most common codes for that class
	
	if len(codes) >= 3 {
		return true
	}
	
	// Check for common codes in each class
	commonCodes := map[int][]int{
		2: {200, 201, 204},
		3: {301, 302, 304},
		4: {400, 401, 403, 404},
		5: {500, 502, 503},
	}
	
	if common, exists := commonCodes[class]; exists {
		// Check if we have at least 2 common codes
		commonCount := 0
		for _, code := range codes {
			for _, commonCode := range common {
				if code == commonCode {
					commonCount++
					break
				}
			}
		}
		return commonCount >= 2
	}
	
	// For unknown classes or edge cases, be conservative
	return len(codes) >= 2
}

// patternsToServiceSpec converts endpoint patterns to a ServiceSpec
func (c *ContractGeneratorLite) patternsToServiceSpec(patterns map[string]*EndpointPattern) *models.ServiceSpec {
	// Create the service spec with new YAML format
	spec := &models.ServiceSpec{
		APIVersion: "flowspec/v1alpha1",
		Kind:       "ServiceSpec",
		Metadata: &models.ServiceSpecMetadata{
			Name:    c.options.ServiceName,
			Version: c.options.ServiceVersion,
		},
		Spec: &models.ServiceSpecDefinition{
			Endpoints: make([]models.EndpointSpec, 0, len(patterns)),
		},
	}
	
	// Convert patterns to endpoints
	for pattern, ep := range patterns {
		endpoint := models.EndpointSpec{
			Path:       pattern,
			Operations: make([]models.OperationSpec, 0, len(ep.Operations)),
			Stats: &models.EndpointStats{
				SupportCount: ep.SampleCount,
				FirstSeen:    c.calculateEndpointFirstSeen(ep),
				LastSeen:     c.calculateEndpointLastSeen(ep),
			},
		}
		
		// Convert operations
		for _, op := range ep.Operations {
			operation := models.OperationSpec{
				Method: op.Method,
				Responses: models.ResponseSpec{
					StatusCodes:  op.StatusCodes,
					StatusRanges: op.StatusRanges,
					Aggregation:  c.options.StatusAggregation,
				},
				Required: models.RequiredFieldsSpec{
					Query:   op.RequiredQuery,
					Headers: op.RequiredHeaders,
				},
				Optional: models.OptionalFieldsSpec{
					Query:   op.OptionalQuery,
					Headers: op.OptionalHeaders,
				},
				Stats: &models.OperationStats{
					SupportCount: op.SampleCount,
					FirstSeen:    op.FirstSeen,
					LastSeen:     op.LastSeen,
				},
			}
			
			endpoint.Operations = append(endpoint.Operations, operation)
		}
		
		// Sort operations by method for consistent output
		sort.Slice(endpoint.Operations, func(i, j int) bool {
			return endpoint.Operations[i].Method < endpoint.Operations[j].Method
		})
		
		spec.Spec.Endpoints = append(spec.Spec.Endpoints, endpoint)
	}
	
	// Sort endpoints by path for consistent output
	sort.Slice(spec.Spec.Endpoints, func(i, j int) bool {
		return spec.Spec.Endpoints[i].Path < spec.Spec.Endpoints[j].Path
	})
	
	return spec
}

// calculateEndpointFirstSeen calculates the earliest timestamp across all operations
func (c *ContractGeneratorLite) calculateEndpointFirstSeen(ep *EndpointPattern) time.Time {
	var earliest time.Time
	
	for _, op := range ep.Operations {
		if earliest.IsZero() || (!op.FirstSeen.IsZero() && op.FirstSeen.Before(earliest)) {
			earliest = op.FirstSeen
		}
	}
	
	return earliest
}

// calculateEndpointLastSeen calculates the latest timestamp across all operations
func (c *ContractGeneratorLite) calculateEndpointLastSeen(ep *EndpointPattern) time.Time {
	var latest time.Time
	
	for _, op := range ep.Operations {
		if latest.IsZero() || (!op.LastSeen.IsZero() && op.LastSeen.After(latest)) {
			latest = op.LastSeen
		}
	}
	
	return latest
}