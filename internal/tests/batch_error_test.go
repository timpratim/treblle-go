package treblle

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
)

func TestBatchErrorCollector(t *testing.T) {
	// Configure Treblle with test settings
	internal.Configure(internal.Configuration{
		ApiKey:    "test-api-key",
		ProjectID: "test-project-id",
		Endpoint:  "http://localhost:8080",
	})

	// Create a batch collector with small batch size and interval for testing
	collector := internal.NewBatchErrorCollector(2, 100*time.Millisecond)
	defer collector.Close()

	// Create test errors
	testErrors := []models.ErrorInfo{
		{
			Message:  "Test error 1",
			Type:    string(models.RequestError),
			Source:  "test",
			Line:    42,
			File:    "test.go",
		},
		{
			Message:  "Test error 2",
			Type:    string(models.ResponseError),
			Source:  "test",
			Line:    43,
			File:    "test.go",
		},
	}

	// Test batch size trigger
	t.Run("BatchSizeTrigger", func(t *testing.T) {
		// Add errors up to batch size
		for _, err := range testErrors {
			collector.Add(err)
		}

		// Allow time for batch processing
		time.Sleep(50 * time.Millisecond)

		// Verify the batch was sent (errors cleared)
		assert.Equal(t, 0, collector.GetErrorCount(), "Batch should be cleared after reaching batch size")
	})

	// Test interval trigger
	t.Run("IntervalTrigger", func(t *testing.T) {
		// Add one error
		collector.Add(testErrors[0])

		// Wait for flush interval
		time.Sleep(150 * time.Millisecond)

		// Verify the batch was sent
		assert.Equal(t, 0, collector.GetErrorCount(), "Batch should be cleared after interval")
	})

	// Test close functionality
	t.Run("CloseFlush", func(t *testing.T) {
		// Add an error
		collector.Add(testErrors[0])

		// Close the collector
		collector.Close()

		// Verify all errors were flushed
		assert.Equal(t, 0, collector.GetErrorCount(), "All errors should be flushed on close")
	})
}
