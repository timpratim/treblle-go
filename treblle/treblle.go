package treblle

import (
	"context"
	"math/rand"
	"time"

	"github.com/timpratim/treblle-go/internal"
	"github.com/timpratim/treblle-go/models"
)

const (
	timeoutDuration = 2 * time.Second
)

type BaseUrlOptions struct {
	Debug bool
}

type Configuration struct {
	APIKey                  string
	ProjectID               string
	AdditionalFieldsToMask  []string
	MaskingEnabled          bool
	AsyncProcessingEnabled  bool
	MaxConcurrentProcessing int
	AsyncShutdownTimeout    time.Duration
	IgnoredEnvironments     []string
}

// Configure sets up the Treblle SDK with the provided configuration
func Configure(config Configuration) {
	internal.Configure(internal.Configuration{
		ApiKey:                  config.APIKey,
		ProjectID:               config.ProjectID,
		AdditionalFieldsToMask:  config.AdditionalFieldsToMask,
		MaskingEnabled:          config.MaskingEnabled,
		AsyncProcessingEnabled:  config.AsyncProcessingEnabled,
		MaxConcurrentProcessing: config.MaxConcurrentProcessing,
		AsyncShutdownTimeout:    config.AsyncShutdownTimeout,
		IgnoredEnvironments:     config.IgnoredEnvironments,
	})
}

func getTreblleBaseUrl() string {
	// If custom endpoint is set, use it
	if models.Config.Endpoint != "" {
		return models.Config.Endpoint
	}

	// For debug mode
	if models.Config.Debug {
		return "https://debug.treblle.com/"
	}

	// Default Treblle endpoints
	treblleBaseUrls := []string{
		"https://rocknrolla.treblle.com",
		"https://punisher.treblle.com",
		"https://sicario.treblle.com",
	}

	rand.Seed(time.Now().Unix())
	randomUrlIndex := rand.Intn(len(treblleBaseUrls))

	return treblleBaseUrls[randomUrlIndex]
}

// SendToTreblle sends data to Treblle
func SendToTreblle(treblleInfo models.MetaData) {
	// Use the context-aware version with a default timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
	defer cancel()

	SendToTreblleWithContext(ctx, treblleInfo)
}

// SendToTreblleWithContext sends data to Treblle with context support
func SendToTreblleWithContext(ctx context.Context, treblleInfo models.MetaData) {
	// Call the models package implementation directly
	models.SendToTreblleWithContext(ctx, treblleInfo)
}
