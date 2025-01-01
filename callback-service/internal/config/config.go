package config

import (
	"log"
	"os"
	"strconv"
)

func GetRequired(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required but not set", key)
	}
	return value
}

func GetEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	value, err := strconv.Atoi(valueStr)
	if err != nil || valueStr == "" {
		return defaultValue
	}
	return value
}
