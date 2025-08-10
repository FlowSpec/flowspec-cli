package traffic

import (
	"time"

	"github.com/flowspec/flowspec-cli/internal/ingestor"
)

// NormalizedRecord represents a normalized traffic record
type NormalizedRecord struct {
	Method    string                 `json:"method"`
	Path      string                 `json:"path"`         // Normalized path
	RawPath   string                 `json:"rawPath"`      // Original path
	Status    int                    `json:"status"`
	Timestamp time.Time              `json:"timestamp"`    // RFC3339 format
	Query     map[string][]string    `json:"query"`        // Keys preserved as-is, supports multi-value
	Headers   map[string][]string    `json:"headers"`      // Keys normalized to lowercase, supports multi-value
	Host      string                 `json:"host"`
	Scheme    string                 `json:"scheme"`
	BodyBytes int64                  `json:"bodyBytes,omitempty"` // Optional
}

// IngestMetrics tracks ingestion statistics and error samples
type IngestMetrics struct {
	TotalLines   int64         `json:"totalLines"`
	ParsedLines  int64         `json:"parsedLines"`
	ErrorLines   int64         `json:"errorLines"`
	Duration     time.Duration `json:"duration"`
	ErrorSamples []string      `json:"errorSamples"` // Limited collection, max 10 by default
}

// TimeRange defines a time filter for ingestion
type TimeRange struct {
	Since *time.Time `json:"since,omitempty"`
	Until *time.Time `json:"until,omitempty"`
}

// IngestOptions configures the ingestion process
type IngestOptions struct {
	LogFormat         string     `json:"logFormat"`         // e.g., "combined", "common"
	CustomRegex       string     `json:"customRegex"`       // Custom regex pattern
	SampleRate        float64    `json:"sampleRate"`        // 0.0-1.0, default 1.0
	TimeFilter        *TimeRange `json:"timeFilter"`        // Optional time range filter
	SensitiveKeys     []string   `json:"sensitiveKeys"`     // Keys to redact
	RedactionPolicy   string     `json:"redactionPolicy"`   // "drop"|"mask"|"hash"
	MaxErrorSamples   int        `json:"maxErrorSamples"`   // Max error samples to collect, default 10
}

// TrafficIngestor defines the interface for traffic log ingestion
type TrafficIngestor interface {
	// Supports checks if the ingestor can handle the given file path
	Supports(filePath string) bool
	
	// Ingest processes the input files and returns an iterator of normalized records
	Ingest(inputs []string, options *IngestOptions) (ingestor.Iterator[*NormalizedRecord], error)
	
	// Metrics returns the current ingestion metrics
	Metrics() *IngestMetrics
	
	// Close releases any resources held by the ingestor
	Close() error
}

// DefaultIngestOptions returns default ingestion options
func DefaultIngestOptions() *IngestOptions {
	return &IngestOptions{
		LogFormat:       "combined",
		SampleRate:      1.0,
		SensitiveKeys:   []string{"authorization", "cookie", "set-cookie", "token", "password", "api_key"},
		RedactionPolicy: "drop",
		MaxErrorSamples: 10,
	}
}

// NewIngestMetrics creates a new metrics instance
func NewIngestMetrics() *IngestMetrics {
	return &IngestMetrics{
		ErrorSamples: make([]string, 0),
	}
}

// AddError adds an error sample to the metrics, respecting the max limit
func (m *IngestMetrics) AddError(errorLine string, maxSamples int) {
	m.ErrorLines++
	
	// Only collect samples up to the limit
	if len(m.ErrorSamples) < maxSamples {
		m.ErrorSamples = append(m.ErrorSamples, errorLine)
	}
}

// AddParsed increments the parsed lines counter
func (m *IngestMetrics) AddParsed() {
	m.ParsedLines++
}

// AddTotal increments the total lines counter
func (m *IngestMetrics) AddTotal() {
	m.TotalLines++
}

// SetDuration sets the processing duration
func (m *IngestMetrics) SetDuration(duration time.Duration) {
	m.Duration = duration
}

// ErrorRate returns the error rate as a percentage
func (m *IngestMetrics) ErrorRate() float64 {
	if m.TotalLines == 0 {
		return 0.0
	}
	return float64(m.ErrorLines) / float64(m.TotalLines)
}

// IsIncomplete returns true if the error rate exceeds 10%
func (m *IngestMetrics) IsIncomplete() bool {
	return m.ErrorRate() > 0.1
}