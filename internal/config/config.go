package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	AppPort             string
	AppEnv              string
	ReadTimeoutSeconds  int
	WriteTimeoutSeconds int
	MySQLHost           string
	MySQLPort           string
	MySQLUser           string
	MySQLPassword       string
	MySQLDatabase       string
	MySQLParams         string
	SourceAPIURL        string
	SourceQueries       []string
	SourceTimeout       int
	SyncIntervalMinutes int
}


func Load() (Config, error) {
	cfg := Config{
		AppPort:             getEnv("APP_PORT", "8080"),
		AppEnv:              getEnv("APP_ENV", "development"),
		MySQLHost:           getEnv("MYSQL_HOST", "mysql"),
		MySQLPort:           getEnv("MYSQL_PORT", "3306"),
		MySQLUser:           getEnv("MYSQL_USER", "stockvacancy"),
		MySQLPassword:       getEnv("MYSQL_PASSWORD", "stockvacancy"),
		MySQLDatabase:       getEnv("MYSQL_DATABASE", "stockvacancy"),
		MySQLParams:         getEnv("MYSQL_PARAMS", "parseTime=true&multiStatements=true"),
		SourceAPIURL:        getEnv("SOURCE_API_URL", ""),
		SourceQueries:       getEnvAsCSV("SOURCE_QUERIES", []string{"software", "backend", "frontend", "mobile", "data", "devops", "golang", "java", "python", "qa"}),
		SyncIntervalMinutes: 0, // set below
	}

	var err error
	cfg.ReadTimeoutSeconds, err = getEnvAsInt("READ_TIMEOUT_SECONDS", 10)
	if err != nil {
		return Config{}, fmt.Errorf("invalid READ_TIMEOUT_SECONDS: %w", err)
	}
	cfg.WriteTimeoutSeconds, err = getEnvAsInt("WRITE_TIMEOUT_SECONDS", 10)
	if err != nil {
		return Config{}, fmt.Errorf("invalid WRITE_TIMEOUT_SECONDS: %w", err)
	}
	cfg.SourceTimeout, err = getEnvAsInt("SOURCE_TIMEOUT_SECONDS", 20)
	if err != nil {
		return Config{}, fmt.Errorf("invalid SOURCE_TIMEOUT_SECONDS: %w", err)
	}
	cfg.SyncIntervalMinutes, err = getEnvAsInt("SYNC_INTERVAL_MINUTES", 60)
	if err != nil {
		return Config{}, fmt.Errorf("invalid SYNC_INTERVAL_MINUTES: %w", err)
	}

	return cfg, nil
}

func (c Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?%s", c.MySQLUser, c.MySQLPassword, c.MySQLHost, c.MySQLPort, c.MySQLDatabase, c.MySQLParams)
}

func getEnv(key string, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getEnvAsInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	return parsed, nil
}

func getEnvAsCSV(key string, fallback []string) []string {
	value := os.Getenv(key)
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]struct{})
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		if _, exists := seen[lower]; exists {
			continue
		}
		seen[lower] = struct{}{}
		result = append(result, trimmed)
	}
	if len(result) == 0 {
		return fallback
	}
	return result
}
