package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// loadFallbacks reads the fallback file and returns a map of keys to values.
func loadFallbacks(filePath string) (map[string]string, error) {
	fallbacks := make(map[string]string)
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") { // Skip empty lines or comments
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			fallbacks[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return fallbacks, nil
}

// getEnvOrFallback retrieves the environment variable value or falls back to the given map.
func getEnvOrFallback(key string, fallbacks map[string]string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallbacks[key]
}

// getVariables retrieves the environment variables or their fallback values.
func getVariables() (string, string, string, string, string, string, error) {
	// Load fallbacks from the file
	fallbacks, err := loadFallbacks("fallback.txt")
	if err != nil {
		return "", "", "", "", "", "", fmt.Errorf("error loading fallback file: %w", err)
	}

	// Retrieve variables
	apiToken := getEnvOrFallback("QASE_API_TOKEN", fallbacks)
	sourceProject := getEnvOrFallback("QASE_SOURCE_PROJECT", fallbacks)
	targetProject := getEnvOrFallback("QASE_TARGET_PROJECT", fallbacks)
	customFieldIDStr := getEnvOrFallback("QASE_CF_ID", fallbacks)
	sourceRunIDStr := getEnvOrFallback("QASE_SOURCE_RUN", fallbacks)
	targetRunIDStr := getEnvOrFallback("QASE_TARGET_RUN", fallbacks)

	return apiToken, sourceProject, targetProject, customFieldIDStr, sourceRunIDStr, targetRunIDStr, nil
}
