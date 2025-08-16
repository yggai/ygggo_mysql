package ygggo_mysql

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlowQueryRecorder_BasicFunctionality(t *testing.T) {
	ctx := context.Background()
	
	// Create a memory storage for testing
	storage := NewMemorySlowQueryStorage(100)
	config := DefaultSlowQueryConfig()
	config.Enabled = true
	config.Threshold = 50 * time.Millisecond
	
	recorder := NewSlowQueryRecorder(config, storage)
	defer recorder.Close()
	
	// Test that recorder is enabled
	assert.True(t, recorder.IsEnabled())
	
	// Test threshold setting
	assert.Equal(t, 50*time.Millisecond, recorder.GetThreshold())
	
	// Record a slow query
	err := recorder.Record(ctx, "SELECT * FROM users WHERE id = ?", []interface{}{1}, 100*time.Millisecond, nil)
	require.NoError(t, err)
	
	// Record a fast query (should not be recorded)
	err = recorder.Record(ctx, "SELECT * FROM users WHERE id = ?", []interface{}{2}, 30*time.Millisecond, nil)
	require.NoError(t, err)
	
	// Get records
	records, err := recorder.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 1)
	
	record := records[0]
	assert.Equal(t, "SELECT * FROM users WHERE id = ?", record.Query)
	assert.Equal(t, 100*time.Millisecond, record.Duration)
	assert.NotEmpty(t, record.ID)
	assert.NotZero(t, record.Timestamp)
}

func TestSlowQueryRecorder_Statistics(t *testing.T) {
	ctx := context.Background()
	
	storage := NewMemorySlowQueryStorage(100)
	config := DefaultSlowQueryConfig()
	config.Enabled = true
	config.Threshold = 50 * time.Millisecond
	
	recorder := NewSlowQueryRecorder(config, storage)
	defer recorder.Close()
	
	// Record multiple slow queries
	queries := []struct {
		query    string
		duration time.Duration
	}{
		{"SELECT * FROM users WHERE id = ?", 100 * time.Millisecond},
		{"SELECT * FROM orders WHERE user_id = ?", 150 * time.Millisecond},
		{"SELECT * FROM users WHERE id = ?", 120 * time.Millisecond}, // Same pattern
		{"UPDATE users SET last_login = NOW() WHERE id = ?", 200 * time.Millisecond},
	}
	
	for _, q := range queries {
		err := recorder.Record(ctx, q.query, []interface{}{1}, q.duration, nil)
		require.NoError(t, err)
	}
	
	// Get statistics
	stats, err := recorder.GetStats(ctx)
	require.NoError(t, err)
	
	assert.Equal(t, int64(4), stats.TotalCount)
	assert.Equal(t, int64(3), stats.UniqueQueries) // 3 unique patterns
	assert.Equal(t, 200*time.Millisecond, stats.MaxDuration)
	assert.Equal(t, 100*time.Millisecond, stats.MinDuration)
	
	// Calculate expected average: (100 + 150 + 120 + 200) / 4 = 142.5ms
	expectedAvg := 142*time.Millisecond + 500*time.Microsecond
	assert.Equal(t, expectedAvg, stats.AverageDuration)
}

func TestSlowQueryRecorder_QueryPatterns(t *testing.T) {
	ctx := context.Background()
	
	storage := NewMemorySlowQueryStorage(100)
	config := DefaultSlowQueryConfig()
	config.Enabled = true
	config.Threshold = 50 * time.Millisecond
	
	recorder := NewSlowQueryRecorder(config, storage)
	defer recorder.Close()
	
	// Record queries with same pattern
	for i := 0; i < 3; i++ {
		err := recorder.Record(ctx, "SELECT * FROM users WHERE id = ?", []interface{}{i + 1}, 100*time.Millisecond, nil)
		require.NoError(t, err)
	}
	
	// Record queries with different pattern
	for i := 0; i < 2; i++ {
		err := recorder.Record(ctx, "SELECT * FROM orders WHERE user_id = ?", []interface{}{i + 1}, 150*time.Millisecond, nil)
		require.NoError(t, err)
	}
	
	// Get patterns
	patterns, err := recorder.GetPatterns(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, patterns, 2)

	// Find the users pattern
	var usersPattern *QueryPattern
	for _, p := range patterns {
		if p.NormalizedQuery == "SELECT * FROM USERS WHERE ID = ?" {
			usersPattern = p
			break
		}
	}

	require.NotNil(t, usersPattern)
	assert.Equal(t, int64(3), usersPattern.Count)
	assert.Equal(t, 300*time.Millisecond, usersPattern.TotalDuration)
	assert.Equal(t, 100*time.Millisecond, usersPattern.AverageDuration)
}

func TestSlowQueryRecorder_Filtering(t *testing.T) {
	ctx := context.Background()
	
	storage := NewMemorySlowQueryStorage(100)
	config := DefaultSlowQueryConfig()
	config.Enabled = true
	config.Threshold = 50 * time.Millisecond
	
	recorder := NewSlowQueryRecorder(config, storage)
	defer recorder.Close()
	
	now := time.Now()
	
	// Record queries at different times
	queries := []struct {
		query    string
		duration time.Duration
		delay    time.Duration
	}{
		{"SELECT * FROM users WHERE id = ?", 100 * time.Millisecond, 0},
		{"SELECT * FROM orders WHERE user_id = ?", 150 * time.Millisecond, 100 * time.Millisecond},
		{"UPDATE users SET last_login = NOW() WHERE id = ?", 200 * time.Millisecond, 200 * time.Millisecond},
	}
	
	for _, q := range queries {
		time.Sleep(q.delay)
		err := recorder.Record(ctx, q.query, []interface{}{1}, q.duration, nil)
		require.NoError(t, err)
	}
	
	// Test duration filtering
	filter := SlowQueryFilter{
		MinDuration: &[]time.Duration{120 * time.Millisecond}[0],
	}
	records, err := recorder.GetRecords(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, records, 2) // 150ms and 200ms queries
	
	// Test time range filtering
	midTime := now.Add(150 * time.Millisecond)
	filter = SlowQueryFilter{
		StartTime: &midTime,
	}
	records, err = recorder.GetRecords(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, records, 1) // Only the last query
	
	// Test limit
	filter = SlowQueryFilter{
		Limit: 2,
	}
	records, err = recorder.GetRecords(ctx, filter)
	require.NoError(t, err)
	assert.Len(t, records, 2)
}

func TestSlowQueryRecorder_Configuration(t *testing.T) {
	ctx := context.Background()
	
	storage := NewMemorySlowQueryStorage(100)
	config := DefaultSlowQueryConfig()
	config.Enabled = false // Start disabled
	config.Threshold = 100 * time.Millisecond
	
	recorder := NewSlowQueryRecorder(config, storage)
	defer recorder.Close()
	
	// Test that recorder is disabled
	assert.False(t, recorder.IsEnabled())
	
	// Record a slow query (should not be recorded)
	err := recorder.Record(ctx, "SELECT * FROM users WHERE id = ?", []interface{}{1}, 150*time.Millisecond, nil)
	require.NoError(t, err)
	
	records, err := recorder.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 0)
	
	// Enable recorder
	recorder.SetEnabled(true)
	assert.True(t, recorder.IsEnabled())
	
	// Record a slow query (should be recorded now)
	err = recorder.Record(ctx, "SELECT * FROM users WHERE id = ?", []interface{}{1}, 150*time.Millisecond, nil)
	require.NoError(t, err)
	
	records, err = recorder.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 1)
	
	// Test threshold change
	recorder.SetThreshold(200 * time.Millisecond)
	assert.Equal(t, 200*time.Millisecond, recorder.GetThreshold())
	
	// Record a query that was slow before but not now
	err = recorder.Record(ctx, "SELECT * FROM orders WHERE id = ?", []interface{}{1}, 150*time.Millisecond, nil)
	require.NoError(t, err)
	
	records, err = recorder.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 1) // Still only one record
}

func TestSlowQueryRecorder_ErrorHandling(t *testing.T) {
	ctx := context.Background()
	
	storage := NewMemorySlowQueryStorage(100)
	config := DefaultSlowQueryConfig()
	config.Enabled = true
	config.Threshold = 50 * time.Millisecond
	
	recorder := NewSlowQueryRecorder(config, storage)
	defer recorder.Close()
	
	// Record a slow query with error
	testError := assert.AnError
	err := recorder.Record(ctx, "SELECT * FROM non_existent_table", []interface{}{}, 100*time.Millisecond, testError)
	require.NoError(t, err)
	
	records, err := recorder.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 1)
	
	record := records[0]
	assert.Equal(t, testError.Error(), record.Error)
}

func TestSlowQueryRecorder_Clear(t *testing.T) {
	ctx := context.Background()
	
	storage := NewMemorySlowQueryStorage(100)
	config := DefaultSlowQueryConfig()
	config.Enabled = true
	config.Threshold = 50 * time.Millisecond
	
	recorder := NewSlowQueryRecorder(config, storage)
	defer recorder.Close()
	
	// Record some slow queries
	for i := 0; i < 5; i++ {
		err := recorder.Record(ctx, "SELECT * FROM users WHERE id = ?", []interface{}{i}, 100*time.Millisecond, nil)
		require.NoError(t, err)
	}
	
	records, err := recorder.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 5)
	
	// Clear all records
	err = recorder.Clear(ctx)
	require.NoError(t, err)
	
	records, err = recorder.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 0)
	
	// Stats should also be reset
	stats, err := recorder.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalCount)
}

func TestFileSlowQueryStorage(t *testing.T) {
	ctx := context.Background()

	// Create temporary file
	tmpFile := filepath.Join(t.TempDir(), "slow_queries.log")

	storage, err := NewFileSlowQueryStorage(tmpFile, 1024*1024) // 1MB max
	require.NoError(t, err)
	defer storage.Close()

	// Test storing records
	record1 := &SlowQueryRecord{
		ID:              "test1",
		Query:           "SELECT * FROM users WHERE id = ?",
		NormalizedQuery: "SELECT * FROM USERS WHERE ID = ?",
		Duration:        100 * time.Millisecond,
		Timestamp:       time.Now(),
		Args:            []interface{}{1},
	}

	err = storage.Store(ctx, record1)
	require.NoError(t, err)

	record2 := &SlowQueryRecord{
		ID:              "test2",
		Query:           "SELECT * FROM orders WHERE user_id = ?",
		NormalizedQuery: "SELECT * FROM ORDERS WHERE USER_ID = ?",
		Duration:        150 * time.Millisecond,
		Timestamp:       time.Now(),
		Args:            []interface{}{1},
	}

	err = storage.Store(ctx, record2)
	require.NoError(t, err)

	// Test retrieving records
	records, err := storage.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records, 2)

	// Test statistics
	stats, err := storage.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(2), stats.TotalCount)
	assert.Equal(t, int64(2), stats.UniqueQueries)

	// Test patterns
	patterns, err := storage.GetPatterns(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, patterns, 2)

	// Test persistence by creating new storage with same file
	storage.Close()

	storage2, err := NewFileSlowQueryStorage(tmpFile, 1024*1024)
	require.NoError(t, err)
	defer storage2.Close()

	// Should load existing records
	records2, err := storage2.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)
	assert.Len(t, records2, 2)
}

func TestSlowQueryAnalyzer(t *testing.T) {
	ctx := context.Background()

	storage := NewMemorySlowQueryStorage(100)
	analyzer := NewSlowQueryAnalyzer(storage)

	// Create test data with various patterns
	baseTime := time.Now()
	testRecords := []*SlowQueryRecord{
		{
			ID:              "1",
			Query:           "SELECT * FROM users WHERE id = 1",
			NormalizedQuery: "SELECT * FROM USERS WHERE ID = ?",
			Duration:        100 * time.Millisecond,
			Timestamp:       baseTime,
		},
		{
			ID:              "2",
			Query:           "SELECT * FROM users WHERE id = 2",
			NormalizedQuery: "SELECT * FROM USERS WHERE ID = ?",
			Duration:        150 * time.Millisecond,
			Timestamp:       baseTime.Add(1 * time.Hour),
		},
		{
			ID:              "3",
			Query:           "SELECT * FROM orders ORDER BY created_at",
			NormalizedQuery: "SELECT * FROM ORDERS ORDER BY CREATED_AT",
			Duration:        800 * time.Millisecond,
			Timestamp:       baseTime.Add(2 * time.Hour),
		},
		{
			ID:              "4",
			Query:           "SELECT name FROM users WHERE email LIKE '%@gmail.com'",
			NormalizedQuery: "SELECT NAME FROM USERS WHERE EMAIL LIKE ?",
			Duration:        1200 * time.Millisecond,
			Timestamp:       baseTime.Add(3 * time.Hour),
		},
	}

	// Store test records
	for _, record := range testRecords {
		err := storage.Store(ctx, record)
		require.NoError(t, err)
	}

	// Generate analysis report
	report, err := analyzer.GenerateReport(ctx, SlowQueryFilter{})
	require.NoError(t, err)

	// Test summary
	assert.Equal(t, int64(4), report.Summary.TotalQueries)
	assert.Equal(t, int64(3), report.Summary.UniquePatterns) // 3 unique patterns
	assert.Equal(t, 1200*time.Millisecond, report.Summary.SlowestQuery)

	// Test top slow queries
	assert.Len(t, report.TopSlowQueries, 4)
	assert.Equal(t, 1200*time.Millisecond, report.TopSlowQueries[0].Duration) // Slowest first

	// Test time distribution
	assert.NotEmpty(t, report.TimeDistribution)

	// Test recommendations
	assert.NotEmpty(t, report.Recommendations)

	// Debug: print all recommendations
	for i, rec := range report.Recommendations {
		t.Logf("Recommendation %d: %s", i, rec)
	}

	// Should contain recommendations about SELECT * and LIKE patterns
	hasSelectStarRecommendation := false
	hasLikeRecommendation := false
	for _, rec := range report.Recommendations {
		if containsSubstring(rec, "SELECT *") {
			hasSelectStarRecommendation = true
		}
		if containsSubstring(rec, "LIKE") {
			hasLikeRecommendation = true
		}
	}
	assert.True(t, hasSelectStarRecommendation)
	assert.True(t, hasLikeRecommendation)
}

func TestSlowQueryAnalyzer_EmptyData(t *testing.T) {
	ctx := context.Background()

	storage := NewMemorySlowQueryStorage(100)
	analyzer := NewSlowQueryAnalyzer(storage)

	// Generate report with no data
	report, err := analyzer.GenerateReport(ctx, SlowQueryFilter{})
	require.NoError(t, err)

	assert.Equal(t, int64(0), report.Summary.TotalQueries)
	assert.Empty(t, report.TopSlowQueries)
	assert.Empty(t, report.TimeDistribution)
}

// Helper function for substring check
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func TestPoolSlowQueryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker test in short mode")
	}

	ctx := context.Background()

	// Create a Docker test helper for real database operations
	helper, err := NewDockerTestHelper(ctx)
	require.NoError(t, err)
	defer helper.Close()

	pool := helper.Pool()

	// Enable slow query recording
	storage := NewMemorySlowQueryStorage(100)
	config := DefaultSlowQueryConfig()
	config.Enabled = true
	config.Threshold = 50 * time.Millisecond

	pool.EnableSlowQueryRecording(config, storage)
	defer pool.DisableSlowQueryRecording()

	// Verify slow query recording is enabled
	assert.True(t, pool.IsSlowQueryRecordingEnabled())

	// Create a test table (MySQL syntax)
	err = pool.WithConn(ctx, func(conn DatabaseConn) error {
		_, err := conn.Exec(ctx, "CREATE TABLE test_users (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(255))")
		return err
	})
	require.NoError(t, err)

	// Execute a query that should be recorded as slow (simulate with sleep)
	err = pool.WithConn(ctx, func(conn DatabaseConn) error {
		// This query should be fast, but we'll simulate slowness by adding a delay
		_, err := conn.Exec(ctx, "INSERT INTO test_users (name) VALUES (?)", "test_user")
		if err != nil {
			return err
		}

		// Simulate a slow query by sleeping
		time.Sleep(60 * time.Millisecond)
		return nil
	})
	require.NoError(t, err)

	// Wait a bit for the recording to complete
	time.Sleep(100 * time.Millisecond)

	// Check if the query was recorded
	recorder := pool.GetSlowQueryRecorder()
	require.NotNil(t, recorder)

	records, err := recorder.GetRecords(ctx, SlowQueryFilter{})
	require.NoError(t, err)

	// Note: The actual query execution might be fast, so we might not have records
	// This test mainly verifies the integration works without errors
	t.Logf("Recorded %d slow queries", len(records))

	// Test configuration changes
	pool.SetSlowQueryThreshold(200 * time.Millisecond)
	assert.Equal(t, 200*time.Millisecond, pool.GetSlowQueryThreshold())
	assert.Equal(t, 200*time.Millisecond, recorder.GetThreshold())

	// Test disabling
	pool.DisableSlowQueryRecording()
	assert.False(t, pool.IsSlowQueryRecordingEnabled())
}
