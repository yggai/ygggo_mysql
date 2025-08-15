package ygggo_mysql

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FileSlowQueryStorage implements SlowQueryStorage using file-based storage
type FileSlowQueryStorage struct {
	filePath    string
	maxFileSize int64
	records     []*SlowQueryRecord
	patterns    map[string]*QueryPattern
	mutex       sync.RWMutex
	file        *os.File
}

// NewFileSlowQueryStorage creates a new file-based slow query storage
func NewFileSlowQueryStorage(filePath string, maxFileSize int64) (*FileSlowQueryStorage, error) {
	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	
	storage := &FileSlowQueryStorage{
		filePath:    filePath,
		maxFileSize: maxFileSize,
		records:     make([]*SlowQueryRecord, 0),
		patterns:    make(map[string]*QueryPattern),
	}
	
	// Load existing records
	if err := storage.loadRecords(); err != nil {
		return nil, fmt.Errorf("failed to load existing records: %w", err)
	}
	
	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	storage.file = file
	
	return storage, nil
}

// loadRecords loads existing records from file
func (s *FileSlowQueryStorage) loadRecords() error {
	file, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's OK
		}
		return err
	}
	defer file.Close()
	
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record SlowQueryRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue // Skip invalid lines
		}
		
		s.records = append(s.records, &record)
		s.updatePattern(&record)
	}
	
	return scanner.Err()
}

// updatePattern updates the pattern statistics for a record
func (s *FileSlowQueryStorage) updatePattern(record *SlowQueryRecord) {
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
}

// Store stores a slow query record
func (s *FileSlowQueryStorage) Store(ctx context.Context, record *SlowQueryRecord) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	// Add to memory
	s.records = append(s.records, record)
	s.updatePattern(record)
	
	// Write to file
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}
	
	if _, err := s.file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}
	
	// Check file size and rotate if necessary
	if err := s.checkAndRotateFile(); err != nil {
		return fmt.Errorf("failed to rotate file: %w", err)
	}
	
	return nil
}

// checkAndRotateFile rotates the log file if it exceeds max size
func (s *FileSlowQueryStorage) checkAndRotateFile() error {
	if s.maxFileSize <= 0 {
		return nil
	}
	
	stat, err := s.file.Stat()
	if err != nil {
		return err
	}
	
	if stat.Size() > s.maxFileSize {
		// Close current file
		s.file.Close()
		
		// Rename current file
		backupPath := s.filePath + ".old"
		if err := os.Rename(s.filePath, backupPath); err != nil {
			return err
		}
		
		// Open new file
		file, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		s.file = file
		
		// Keep only recent records in memory
		if len(s.records) > 1000 {
			s.records = s.records[len(s.records)-1000:]
			s.rebuildPatterns()
		}
	}
	
	return nil
}

// rebuildPatterns rebuilds pattern statistics from current records
func (s *FileSlowQueryStorage) rebuildPatterns() {
	s.patterns = make(map[string]*QueryPattern)
	for _, record := range s.records {
		s.updatePattern(record)
	}
}

// GetRecords retrieves slow query records with optional filtering
func (s *FileSlowQueryStorage) GetRecords(ctx context.Context, filter SlowQueryFilter) ([]*SlowQueryRecord, error) {
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

func (s *FileSlowQueryStorage) matchesFilter(record *SlowQueryRecord, filter SlowQueryFilter) bool {
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
	
	if filter.QueryPattern != "" && !stringContains(record.NormalizedQuery, filter.QueryPattern) {
		return false
	}
	
	if filter.Database != "" && record.Database != filter.Database {
		return false
	}
	
	return true
}

// GetStats returns statistics about slow queries
func (s *FileSlowQueryStorage) GetStats(ctx context.Context) (*SlowQueryStats, error) {
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
func (s *FileSlowQueryStorage) GetPatterns(ctx context.Context, limit int) ([]*QueryPattern, error) {
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

func (s *FileSlowQueryStorage) getTopPatterns(limit int) []QueryPattern {
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
func (s *FileSlowQueryStorage) Clear(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	s.records = make([]*SlowQueryRecord, 0)
	s.patterns = make(map[string]*QueryPattern)
	
	// Close and recreate file
	s.file.Close()
	if err := os.Remove(s.filePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	
	file, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	s.file = file
	
	return nil
}

// Close closes the storage
func (s *FileSlowQueryStorage) Close() error {
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

// Helper function for string contains check
func stringContains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
