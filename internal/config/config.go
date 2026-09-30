package config

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DiscordToken     string
	RequestTimeout   time.Duration
	MaxConcurrent    int
	MinConfidence    float64
	LogLevel         slog.Level
	TypeSafeLogLevel *slog.Level // nil disables client logs.
}

func Load() (Config, error) {
	c := Config{
		DiscordToken:   strings.TrimSpace(os.Getenv("DISCORD_TOKEN")),
		RequestTimeout: 30 * time.Second,
		MaxConcurrent:  4,
		MinConfidence:  0.7,
		LogLevel:       slog.LevelInfo,
	}
	if c.DiscordToken == "" {
		return Config{}, fmt.Errorf("DISCORD_TOKEN is required")
	}
	if strings.TrimSpace(os.Getenv("TYPESAFE_API_KEY")) == "" {
		return Config{}, fmt.Errorf("TYPESAFE_API_KEY is required")
	}
	if value := strings.TrimSpace(os.Getenv("BOT_REQUEST_TIMEOUT")); value != "" {
		timeout, err := time.ParseDuration(value)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("BOT_REQUEST_TIMEOUT must be a positive Go duration, e.g. 30s")
		}
		c.RequestTimeout = timeout
	}
	if value := strings.TrimSpace(os.Getenv("BOT_MAX_CONCURRENT")); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil || limit < 1 {
			return Config{}, fmt.Errorf("BOT_MAX_CONCURRENT must be a positive integer")
		}
		c.MaxConcurrent = limit
	}
	if value := strings.TrimSpace(os.Getenv("BOT_MIN_CONFIDENCE")); value != "" {
		confidence, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(confidence) || confidence < 0 || confidence > 1 {
			return Config{}, fmt.Errorf("BOT_MIN_CONFIDENCE must be between 0 and 1")
		}
		c.MinConfidence = confidence
	}
	if value := strings.TrimSpace(os.Getenv("LOG_LEVEL")); value != "" {
		if err := c.LogLevel.UnmarshalText([]byte(value)); err != nil {
			return Config{}, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
		}
	}
	switch value := strings.ToLower(strings.TrimSpace(os.Getenv("TYPESAFE_LOG_LEVEL"))); value {
	case "", "off":
	case "debug", "info", "warn", "warning", "error":
		var level slog.Level
		if value == "warning" {
			value = "warn"
		}
		_ = level.UnmarshalText([]byte(value))
		c.TypeSafeLogLevel = &level
	default:
		return Config{}, fmt.Errorf("TYPESAFE_LOG_LEVEL must be debug, info, warn, error, or off")
	}
	return c, nil
}
