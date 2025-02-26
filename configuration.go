package treblle

import (
	"os"
	"strings"
	"time"
)

var Config internalConfiguration

// Configuration sets up and customizes communication with the Treblle API
type Configuration struct {
	APIKey                 string
	ProjectID              string
	AdditionalFieldsToMask []string
	DefaultFieldsToMask    []string
	MaskingEnabled         bool
	Endpoint               string // Custom endpoint for testing
	BatchErrorEnabled      bool   // Enable batch error collection
	BatchErrorSize         int    // Size of error batch before sending
	BatchFlushInterval     time.Duration // Interval to flush errors if batch size not reached
	SDKName                string // Defaults to "go"
	SDKVersion             string // Defaults to "1.0.0"
}

// internalConfiguration is used for communication with Treblle API and contains optimizations
type internalConfiguration struct {
	APIKey                 string
	ProjectID              string
	AdditionalFieldsToMask []string
	DefaultFieldsToMask    []string
	MaskingEnabled         bool
	Endpoint               string
	FieldsMap              map[string]bool
	serverInfo             ServerInfo
	languageInfo           LanguageInfo
	Debug                  bool
	batchErrorCollector    *BatchErrorCollector
	SDKName                string
	SDKVersion             string
}

func Configure(config Configuration) {
	if config.APIKey != "" {
		Config.APIKey = config.APIKey
	}
	if config.ProjectID != "" {
		Config.ProjectID = config.ProjectID
	}
	if config.Endpoint != "" {
		Config.Endpoint = config.Endpoint
	}

	// Initialize server and language info
	Config.serverInfo = GetServerInfo()
	Config.languageInfo = GetLanguageInfo()

	// Initialize default masking settings
	Config.MaskingEnabled = true // Enable by default

	// Set SDK Name and Version (Can be overridden via ENV)
	sdkName := SDKName
	if config.SDKName != "" {
		sdkName = config.SDKName
	}
	
	sdkVersion := SDKVersion
	if config.SDKVersion != "" {
		sdkVersion = config.SDKVersion
	}
	
	Config.SDKName = getEnvOrDefault("TREBLLE_SDK_NAME", sdkName)
	Config.SDKVersion = getEnvOrDefault("TREBLLE_SDK_VERSION", sdkVersion)

	// Initialize batch error collector if enabled
	if config.BatchErrorEnabled {
		// Close existing collector if any
		if Config.batchErrorCollector != nil {
			Config.batchErrorCollector.Close()
		}
		// Create new batch error collector
		Config.batchErrorCollector = NewBatchErrorCollector(config.BatchErrorSize, config.BatchFlushInterval)
	}
	if len(config.DefaultFieldsToMask) > 0 {
		Config.DefaultFieldsToMask = config.DefaultFieldsToMask
	} else {
		// Load from environment variable if available
		if envFields := getEnvMaskedFields(); len(envFields) > 0 {
			Config.DefaultFieldsToMask = envFields
		} else {
			Config.DefaultFieldsToMask = getDefaultFieldsToMask()
		}
	}

	if len(config.AdditionalFieldsToMask) > 0 {
		Config.AdditionalFieldsToMask = config.AdditionalFieldsToMask
	}

	Config.FieldsMap = generateFieldsToMask(Config.DefaultFieldsToMask, Config.AdditionalFieldsToMask)
}

// getEnvMaskedFields reads masked fields from environment variable
func getEnvMaskedFields() []string {
	fieldsStr := os.Getenv("TREBLLE_MASKED_FIELDS")
	if fieldsStr == "" {
		return nil
	}
	return strings.Split(fieldsStr, ",")
}

// getDefaultFieldsToMask returns the default list of fields to mask
func getDefaultFieldsToMask() []string {
	return []string{
		"password",
		"pwd",
		"secret",
		"password_confirmation",
		"passwordConfirmation",
		"cc",
		"card_number",
		"cardNumber",
		"ccv",
		"ssn",
		"credit_score",
		"creditScore",
		"api_key",
		"apiKey",
	}
}

func generateFieldsToMask(defaultFields, additionalFields []string) map[string]bool {
	fields := append(defaultFields, additionalFields...)
	fieldsToMask := make(map[string]bool)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field != "" {
			fieldsToMask[field] = true
		}
	}
	return fieldsToMask
}

// shouldMaskField checks if a field should be masked based on configuration
func shouldMaskField(field string) bool {
	_, exists := Config.FieldsMap[strings.ToLower(field)]
	return exists
}

// Utility function to get env variable or return default
func getEnvOrDefault(envKey, defaultValue string) string {
	if value := os.Getenv(envKey); value != "" {
		return value
	}
	return defaultValue
}

// GetSDKInfo returns SDK name and version (for debugging)
func GetSDKInfo() map[string]string {
	return map[string]string{
		"SDK Name":    Config.SDKName,
		"SDK Version": Config.SDKVersion,
	}
}
