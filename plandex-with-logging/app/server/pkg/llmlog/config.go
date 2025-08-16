// Package llmlog provides configuration and log capture for LLM call instrumentation.
//
// This file defines the config struct and types for logging control.
package llmlog

import (
	"os"
	"strconv"
	"strings"
)

// Config defines the configuration for LLM call logging.
// Matches the YAML schema in the product roadmap.
type Config struct {
	Enabled          bool     `yaml:"enabled" json:"enabled"`
	FilePath         string   `yaml:"filePath" json:"filePath"`
	Format           string   `yaml:"format" json:"format"` // "json", "text", "csv"
	Level            string   `yaml:"level" json:"level"`   // "basic", "detailed"
	RetentionDays    int      `yaml:"retentionDays" json:"retentionDays"`
	MaxFileSize      string   `yaml:"maxFileSize" json:"maxFileSize"` // e.g. "100MB"
	IncludeContent   bool     `yaml:"includeContent" json:"includeContent"`
	IncludeSensitive bool     `yaml:"includeSensitive" json:"includeSensitive"`
	Filters          Filters  `yaml:"filters" json:"filters"`
}

// Filters indicates what roles/providers to log.
type Filters struct {
	Roles     []string `yaml:"roles" json:"roles"`         // ["*"] or specific roles
	Providers []string `yaml:"providers" json:"providers"` // ["*"] or specific providers
}

// LoadConfigFromEnv loads config from environment variables, falling back to defaults.
func LoadConfigFromEnv() *Config {
	cfg := &Config{
		Enabled:          getEnvBool("PLANDEX_LLM_LOGGING", false),
		FilePath:         getEnvString("PLANDEX_LLM_LOG_FILE", ""),
		Format:           getEnvString("PLANDEX_LLM_LOG_FORMAT", "json"),
		Level:            getEnvString("PLANDEX_LLM_LOG_LEVEL", "basic"),
		RetentionDays:    getEnvInt("PLANDEX_LLM_LOG_RETENTION", 7),
		MaxFileSize:      getEnvString("PLANDEX_LLM_LOG_MAX_FILE_SIZE", "100MB"),
		IncludeContent:   getEnvBool("PLANDEX_LLM_LOG_INCLUDE_CONTENT", true),
		IncludeSensitive: getEnvBool("PLANDEX_LLM_LOG_INCLUDE_SENSITIVE", false),
		Filters: Filters{
			Roles:     getEnvCSV("PLANDEX_LLM_LOG_FILTER_ROLES", []string{"*"}),
			Providers: getEnvCSV("PLANDEX_LLM_LOG_FILTER_PROVIDERS", []string{"*"}),
		},
	}

	// Set default log file path if not provided
	if cfg.FilePath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			cfg.FilePath = "./llm_calls.log" // fallback if home dir can't be determined
		} else {
			cfg.FilePath = home + "/.plandex/llm_calls.log"
		}
	}
	return cfg
}

// Helper: getEnvString returns env var or default.
func getEnvString(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

// Helper: getEnvBool parses a bool from env var or returns default.
func getEnvBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

// Helper: getEnvInt parses an int from env var or returns default.
func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

// Helper: getEnvCSV parses a comma-separated string as a []string, or returns default.
func getEnvCSV(key string, def []string) []string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	parts := strings.Split(v, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return def
	}
	return out
}