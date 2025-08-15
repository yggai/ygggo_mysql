package ygggo_mysql

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

// SlowQueryRecord represents a single slow query record
type SlowQueryRecord struct {
	ID          string        `json:"id"`          // Unique identifier
	Query       string        `json:"query"`       // Original SQL query
	NormalizedQuery string    `json:"normalized_query"` // Normalized query for pattern matching
	Duration    time.Duration `json:"duration"`    // Query execution duration
	Timestamp   time.Time     `json:"timestamp"`   // When the query was executed
	Args        []interface{} `json:"args,omitempty"` // Query arguments (may be sanitized)
	Error       string        `json:"error,omitempty"` // Error message if query failed
	Stack       string        `json:"stack,omitempty"` // Call stack (optional)
	Database    string        `json:"database,omitempty"` // Database name
	User        string        `json:"user,omitempty"` // Database user
	Host        string        `json:"host,omitempty"` // Database host
}

// SlowQueryStats represents statistics for slow queries
type SlowQueryStats struct {
	TotalCount       int64         `json:"total_count"`       // Total number of slow queries
	UniqueQueries    int64         `json:"unique_queries"`    // Number of unique query patterns
	AverageDuration  time.Duration `json:"average_duration"`  // Average execution duration
	MaxDuration      time.Duration `json:"max_duration"`      // Maximum execution duration
	MinDuration      time.Duration `json:"min_duration"`      // Minimum execution duration
	LastRecordTime   time.Time     `json:"last_record_time"`  // Time of last slow query
	TopQueries       []QueryPattern `json:"top_queries"`      // Most frequent slow queries
}

// QueryPattern represents a pattern of similar queries
type QueryPattern struct {
	NormalizedQuery string        `json:"normalized_query"` // Normalized query pattern
	Count           int64         `json:"count"`           // Number of occurrences
	TotalDuration   time.Duration `json:"total_duration"`  // Total execution time
	AverageDuration time.Duration `json:"average_duration"` // Average execution time
	MaxDuration     time.Duration `json:"max_duration"`    // Maximum execution time
	LastSeen        time.Time     `json:"last_seen"`       // Last occurrence time
	Examples        []string      `json:"examples,omitempty"` // Example queries (limited)
}

// SlowQueryConfig holds configuration for slow query recording
type SlowQueryConfig struct {
	Enabled           bool          `json:"enabled"`            // Enable slow query recording
	Threshold         time.Duration `json:"threshold"`          // Slow query threshold
	MaxRecords        int           `json:"max_records"`        // Maximum records to keep in memory
	MaxPatterns       int           `json:"max_patterns"`       // Maximum query patterns to track
	SanitizeArgs      bool          `json:"sanitize_args"`      // Whether to sanitize query arguments
	IncludeStack      bool          `json:"include_stack"`      // Whether to include call stack
	NormalizationMode string        `json:"normalization_mode"` // Query normalization mode
}

// DefaultSlowQueryConfig returns default configuration
func DefaultSlowQueryConfig() SlowQueryConfig {
	return SlowQueryConfig{
		Enabled:           false,
		Threshold:         100 * time.Millisecond,
		MaxRecords:        1000,
		MaxPatterns:       100,
		SanitizeArgs:      true,
		IncludeStack:      false,
		NormalizationMode: "basic",
	}
}

// SlowQueryConfigManager manages slow query configuration
type SlowQueryConfigManager struct {
	config SlowQueryConfig
	mutex  sync.RWMutex
}

// NewSlowQueryConfigManager creates a new configuration manager
func NewSlowQueryConfigManager(config SlowQueryConfig) *SlowQueryConfigManager {
	return &SlowQueryConfigManager{
		config: config,
	}
}

// GetConfig returns the current configuration
func (m *SlowQueryConfigManager) GetConfig() SlowQueryConfig {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config
}

// UpdateConfig updates the configuration
func (m *SlowQueryConfigManager) UpdateConfig(config SlowQueryConfig) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config = config
}

// SetEnabled enables or disables slow query recording
func (m *SlowQueryConfigManager) SetEnabled(enabled bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.Enabled = enabled
}

// IsEnabled returns whether slow query recording is enabled
func (m *SlowQueryConfigManager) IsEnabled() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.Enabled
}

// SetThreshold sets the slow query threshold
func (m *SlowQueryConfigManager) SetThreshold(threshold time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.Threshold = threshold
}

// GetThreshold returns the current slow query threshold
func (m *SlowQueryConfigManager) GetThreshold() time.Duration {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.Threshold
}

// SetMaxRecords sets the maximum number of records to keep
func (m *SlowQueryConfigManager) SetMaxRecords(maxRecords int) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.MaxRecords = maxRecords
}

// GetMaxRecords returns the maximum number of records to keep
func (m *SlowQueryConfigManager) GetMaxRecords() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.MaxRecords
}

// SetSanitizeArgs sets whether to sanitize query arguments
func (m *SlowQueryConfigManager) SetSanitizeArgs(sanitize bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.SanitizeArgs = sanitize
}

// ShouldSanitizeArgs returns whether to sanitize query arguments
func (m *SlowQueryConfigManager) ShouldSanitizeArgs() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.SanitizeArgs
}

// SetIncludeStack sets whether to include call stack
func (m *SlowQueryConfigManager) SetIncludeStack(include bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.config.IncludeStack = include
}

// ShouldIncludeStack returns whether to include call stack
func (m *SlowQueryConfigManager) ShouldIncludeStack() bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.config.IncludeStack
}

// SlowQueryStorage defines the interface for storing slow query records
type SlowQueryStorage interface {
	// Store stores a slow query record
	Store(ctx context.Context, record *SlowQueryRecord) error
	
	// GetRecords retrieves slow query records with optional filtering
	GetRecords(ctx context.Context, filter SlowQueryFilter) ([]*SlowQueryRecord, error)
	
	// GetStats returns statistics about slow queries
	GetStats(ctx context.Context) (*SlowQueryStats, error)
	
	// GetPatterns returns query patterns
	GetPatterns(ctx context.Context, limit int) ([]*QueryPattern, error)
	
	// Clear removes all stored records
	Clear(ctx context.Context) error
	
	// Close closes the storage
	Close() error
}

// SlowQueryFilter defines filtering options for retrieving records
type SlowQueryFilter struct {
	StartTime    *time.Time    `json:"start_time,omitempty"`
	EndTime      *time.Time    `json:"end_time,omitempty"`
	MinDuration  *time.Duration `json:"min_duration,omitempty"`
	MaxDuration  *time.Duration `json:"max_duration,omitempty"`
	QueryPattern string        `json:"query_pattern,omitempty"`
	Database     string        `json:"database,omitempty"`
	Limit        int           `json:"limit,omitempty"`
	Offset       int           `json:"offset,omitempty"`
}

// SlowQueryRecorder manages slow query recording
type SlowQueryRecorder struct {
	configManager *SlowQueryConfigManager
	storage       SlowQueryStorage
}

// NewSlowQueryRecorder creates a new slow query recorder
func NewSlowQueryRecorder(config SlowQueryConfig, storage SlowQueryStorage) *SlowQueryRecorder {
	return &SlowQueryRecorder{
		configManager: NewSlowQueryConfigManager(config),
		storage:       storage,
	}
}

// IsEnabled returns whether slow query recording is enabled
func (r *SlowQueryRecorder) IsEnabled() bool {
	return r.configManager.IsEnabled()
}

// SetEnabled enables or disables slow query recording
func (r *SlowQueryRecorder) SetEnabled(enabled bool) {
	r.configManager.SetEnabled(enabled)
}

// SetThreshold sets the slow query threshold
func (r *SlowQueryRecorder) SetThreshold(threshold time.Duration) {
	r.configManager.SetThreshold(threshold)
}

// GetThreshold returns the current slow query threshold
func (r *SlowQueryRecorder) GetThreshold() time.Duration {
	return r.configManager.GetThreshold()
}

// GetConfig returns the current configuration
func (r *SlowQueryRecorder) GetConfig() SlowQueryConfig {
	return r.configManager.GetConfig()
}

// UpdateConfig updates the configuration
func (r *SlowQueryRecorder) UpdateConfig(config SlowQueryConfig) {
	r.configManager.UpdateConfig(config)
}

// Record records a slow query if it exceeds the threshold
func (r *SlowQueryRecorder) Record(ctx context.Context, query string, args []interface{}, duration time.Duration, err error) error {
	if !r.IsEnabled() {
		return nil
	}

	threshold := r.GetThreshold()
	if duration <= threshold {
		return nil
	}

	config := r.GetConfig()
	record := &SlowQueryRecord{
		ID:              generateID(),
		Query:           query,
		NormalizedQuery: r.normalizeQuery(query, config.NormalizationMode),
		Duration:        duration,
		Timestamp:       time.Now(),
		Args:            r.sanitizeArgs(args, config.SanitizeArgs),
	}

	if err != nil {
		record.Error = err.Error()
	}

	if config.IncludeStack {
		record.Stack = captureStack()
	}

	return r.storage.Store(ctx, record)
}

// GetRecords retrieves slow query records
func (r *SlowQueryRecorder) GetRecords(ctx context.Context, filter SlowQueryFilter) ([]*SlowQueryRecord, error) {
	return r.storage.GetRecords(ctx, filter)
}

// GetStats returns slow query statistics
func (r *SlowQueryRecorder) GetStats(ctx context.Context) (*SlowQueryStats, error) {
	return r.storage.GetStats(ctx)
}

// GetPatterns returns query patterns
func (r *SlowQueryRecorder) GetPatterns(ctx context.Context, limit int) ([]*QueryPattern, error) {
	return r.storage.GetPatterns(ctx, limit)
}

// Clear clears all slow query records
func (r *SlowQueryRecorder) Clear(ctx context.Context) error {
	return r.storage.Clear(ctx)
}

// Close closes the recorder
func (r *SlowQueryRecorder) Close() error {
	if r.storage != nil {
		return r.storage.Close()
	}
	return nil
}

// Helper functions implementation
func generateID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (r *SlowQueryRecorder) normalizeQuery(query string, mode string) string {
	if mode == "none" {
		return query
	}

	// Basic normalization: replace values with placeholders
	normalized := query

	// Replace string literals
	stringLiteralRe := regexp.MustCompile(`'[^']*'`)
	normalized = stringLiteralRe.ReplaceAllString(normalized, "?")

	// Replace numeric literals
	numericRe := regexp.MustCompile(`\b\d+\b`)
	normalized = numericRe.ReplaceAllString(normalized, "?")

	// Normalize whitespace
	whitespaceRe := regexp.MustCompile(`\s+`)
	normalized = whitespaceRe.ReplaceAllString(strings.TrimSpace(normalized), " ")

	// Convert to uppercase for consistency
	normalized = strings.ToUpper(normalized)

	return normalized
}

func (r *SlowQueryRecorder) sanitizeArgs(args []interface{}, shouldSanitize bool) []interface{} {
	if !shouldSanitize {
		return args
	}

	sanitized := make([]interface{}, len(args))
	for i, arg := range args {
		switch v := arg.(type) {
		case string:
			if len(v) > 50 {
				sanitized[i] = v[:50] + "..."
			} else {
				sanitized[i] = "[string]"
			}
		case []byte:
			sanitized[i] = fmt.Sprintf("[bytes:%d]", len(v))
		default:
			sanitized[i] = fmt.Sprintf("[%T]", v)
		}
	}
	return sanitized
}

func captureStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}

// MemorySlowQueryStorage implements SlowQueryStorage using in-memory storage
type MemorySlowQueryStorage struct {
	records   []*SlowQueryRecord
	patterns  map[string]*QueryPattern
	maxSize   int
	mutex     sync.RWMutex
}

// NewMemorySlowQueryStorage creates a new memory-based slow query storage
func NewMemorySlowQueryStorage(maxSize int) *MemorySlowQueryStorage {
	return &MemorySlowQueryStorage{
		records:  make([]*SlowQueryRecord, 0),
		patterns: make(map[string]*QueryPattern),
		maxSize:  maxSize,
	}
}

// Store stores a slow query record
func (s *MemorySlowQueryStorage) Store(ctx context.Context, record *SlowQueryRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Add to records
	s.records = append(s.records, record)

	// Maintain max size (ring buffer behavior)
	if len(s.records) > s.maxSize {
		s.records = s.records[1:]
	}

	// Update patterns
	pattern, exists := s.patterns[record.NormalizedQuery]
	if !exists {
		pattern = &QueryPattern{
			NormalizedQuery: record.NormalizedQuery,
			Count:           0,
			TotalDuration:   0,
			MaxDuration:     0,
			Examples:        make([]string, 0, 3),
		}
		s.patterns[record.NormalizedQuery] = pattern
	}

	pattern.Count++
	pattern.TotalDuration += record.Duration
	pattern.AverageDuration = time.Duration(int64(pattern.TotalDuration) / pattern.Count)
	pattern.LastSeen = record.Timestamp

	if record.Duration > pattern.MaxDuration {
		pattern.MaxDuration = record.Duration
	}

	// Add example (limit to 3)
	if len(pattern.Examples) < 3 {
		pattern.Examples = append(pattern.Examples, record.Query)
	}

	return nil
}

// GetRecords retrieves slow query records with optional filtering
func (s *MemorySlowQueryStorage) GetRecords(ctx context.Context, filter SlowQueryFilter) ([]*SlowQueryRecord, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var filtered []*SlowQueryRecord

	for _, record := range s.records {
		if s.matchesFilter(record, filter) {
			filtered = append(filtered, record)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Timestamp.After(filtered[j].Timestamp)
	})

	// Apply limit and offset
	start := 0
	if filter.Offset > 0 {
		start = filter.Offset
		if start >= len(filtered) {
			return []*SlowQueryRecord{}, nil
		}
	}

	end := len(filtered)
	if filter.Limit > 0 && start+filter.Limit < end {
		end = start + filter.Limit
	}

	return filtered[start:end], nil
}

func (s *MemorySlowQueryStorage) matchesFilter(record *SlowQueryRecord, filter SlowQueryFilter) bool {
	if filter.StartTime != nil && record.Timestamp.Before(*filter.StartTime) {
		return false
	}

	if filter.EndTime != nil && record.Timestamp.After(*filter.EndTime) {
		return false
	}

	if filter.MinDuration != nil && record.Duration < *filter.MinDuration {
		return false
	}

	if filter.MaxDuration != nil && record.Duration > *filter.MaxDuration {
		return false
	}

	if filter.QueryPattern != "" && !strings.Contains(record.NormalizedQuery, filter.QueryPattern) {
		return false
	}

	if filter.Database != "" && record.Database != filter.Database {
		return false
	}

	return true
}

// GetStats returns statistics about slow queries
func (s *MemorySlowQueryStorage) GetStats(ctx context.Context) (*SlowQueryStats, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if len(s.records) == 0 {
		return &SlowQueryStats{}, nil
	}

	var totalDuration time.Duration
	var maxDuration time.Duration
	minDuration := s.records[0].Duration
	var lastRecordTime time.Time

	for _, record := range s.records {
		totalDuration += record.Duration

		if record.Duration > maxDuration {
			maxDuration = record.Duration
		}

		if record.Duration < minDuration {
			minDuration = record.Duration
		}

		if record.Timestamp.After(lastRecordTime) {
			lastRecordTime = record.Timestamp
		}
	}

	avgDuration := time.Duration(int64(totalDuration) / int64(len(s.records)))

	// Get top patterns
	topPatterns := s.getTopPatterns(10)

	return &SlowQueryStats{
		TotalCount:      int64(len(s.records)),
		UniqueQueries:   int64(len(s.patterns)),
		AverageDuration: avgDuration,
		MaxDuration:     maxDuration,
		MinDuration:     minDuration,
		LastRecordTime:  lastRecordTime,
		TopQueries:      topPatterns,
	}, nil
}

// GetPatterns returns query patterns
func (s *MemorySlowQueryStorage) GetPatterns(ctx context.Context, limit int) ([]*QueryPattern, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	patterns := make([]*QueryPattern, 0, len(s.patterns))
	for _, pattern := range s.patterns {
		patterns = append(patterns, pattern)
	}

	// Sort by count (most frequent first)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	if limit > 0 && limit < len(patterns) {
		patterns = patterns[:limit]
	}

	return patterns, nil
}

func (s *MemorySlowQueryStorage) getTopPatterns(limit int) []QueryPattern {
	patterns := make([]QueryPattern, 0, len(s.patterns))
	for _, pattern := range s.patterns {
		patterns = append(patterns, *pattern)
	}

	// Sort by count (most frequent first)
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Count > patterns[j].Count
	})

	if limit > 0 && limit < len(patterns) {
		patterns = patterns[:limit]
	}

	return patterns
}

// Clear removes all stored records
func (s *MemorySlowQueryStorage) Clear(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.records = make([]*SlowQueryRecord, 0)
	s.patterns = make(map[string]*QueryPattern)

	return nil
}

// Close closes the storage
func (s *MemorySlowQueryStorage) Close() error {
	// Nothing to close for memory storage
	return nil
}
