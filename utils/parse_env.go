package utils

import (
	"os"
	"strconv"
	"time"
)

func EnvInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func EnvDuration(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func EnvString(name string, fallback string) string {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	return value
}

func EnvBool(name string, fallbask bool) bool {
	value := os.Getenv(name)
	if value == "" {
		return fallbask
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallbask
	}
	return parsed
}
