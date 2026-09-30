package config

import (
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

const (
	testToken  = "discord-token-DO-NOT-LEAK"
	testAPIKey = "typesafe-key-DO-NOT-LEAK"
)

func configEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"DISCORD_TOKEN": testToken, "TYPESAFE_API_KEY": testAPIKey,
		"BOT_REQUEST_TIMEOUT": "", "BOT_MAX_CONCURRENT": "", "BOT_MIN_CONFIDENCE": "", "LOG_LEVEL": "", "TYPESAFE_LOG_LEVEL": "",
	} {
		t.Setenv(key, value)
	}
}

func TestLoadDefaults(t *testing.T) {
	for _, optional := range []string{"unset", "", " \t\n"} {
		t.Run("optional="+optional, func(t *testing.T) {
			configEnv(t)
			t.Setenv("DISCORD_TOKEN", " \t"+testToken+"\n")
			t.Setenv("TYPESAFE_API_KEY", " \t"+testAPIKey+"\n")
			for _, key := range []string{"BOT_REQUEST_TIMEOUT", "BOT_MAX_CONCURRENT", "BOT_MIN_CONFIDENCE", "LOG_LEVEL", "TYPESAFE_LOG_LEVEL"} {
				if optional == "unset" {
					if err := os.Unsetenv(key); err != nil {
						t.Fatal(err)
					}
				} else {
					t.Setenv(key, optional)
				}
			}
			got, err := Load()
			want := Config{DiscordToken: testToken, RequestTimeout: 30 * time.Second, MaxConcurrent: 4, MinConfidence: 0.7, LogLevel: slog.LevelInfo}
			if err != nil || got != want {
				t.Fatalf("defaults differ (error=%v)", err)
			}
		})
	}
}

func TestLoadOverrides(t *testing.T) {
	configEnv(t)
	t.Setenv("BOT_REQUEST_TIMEOUT", " 1m250ms ")
	t.Setenv("BOT_MAX_CONCURRENT", " 7 ")
	t.Setenv("BOT_MIN_CONFIDENCE", " 0.85 ")
	t.Setenv("LOG_LEVEL", " DEBUG ")
	got, err := Load()
	want := Config{DiscordToken: testToken, RequestTimeout: time.Minute + 250*time.Millisecond, MaxConcurrent: 7, MinConfidence: 0.85, LogLevel: slog.LevelDebug}
	if err != nil || got != want {
		t.Fatalf("overrides differ (error=%v)", err)
	}
}

func TestLoadTypeSafeLogLevels(t *testing.T) {
	for _, tt := range []struct {
		value string
		level slog.Level
	}{
		{"off", 0},
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
	} {
		for _, value := range []string{tt.value, " \t" + strings.ToUpper(tt.value) + "\n"} {
			t.Run(value, func(t *testing.T) {
				configEnv(t)
				t.Setenv("TYPESAFE_LOG_LEVEL", value)
				t.Setenv("LOG_LEVEL", "error")
				got, err := Load()
				if err != nil {
					t.Fatal(err)
				}
				if tt.value == "off" {
					if got.TypeSafeLogLevel != nil {
						t.Error("off must leave TypeSafeLogLevel nil")
					}
				} else if got.TypeSafeLogLevel == nil || *got.TypeSafeLogLevel != tt.level {
					t.Errorf("TypeSafeLogLevel = %v, want %s", got.TypeSafeLogLevel, tt.level)
				}
				if got.LogLevel != slog.LevelError {
					t.Errorf("LOG_LEVEL changed to %s", got.LogLevel)
				}
			})
		}
	}
}

func TestLoadValidBoundaries(t *testing.T) {
	for _, tt := range []struct {
		key, value string
		check      func(Config) bool
	}{
		{"BOT_REQUEST_TIMEOUT", "1ns", func(c Config) bool { return c.RequestTimeout == time.Nanosecond }},
		{"BOT_MAX_CONCURRENT", "1", func(c Config) bool { return c.MaxConcurrent == 1 }},
		{"BOT_MIN_CONFIDENCE", "0", func(c Config) bool { return c.MinConfidence == 0 }},
		{"BOT_MIN_CONFIDENCE", "1", func(c Config) bool { return c.MinConfidence == 1 }},
		{"BOT_MIN_CONFIDENCE", "7e-1", func(c Config) bool { return c.MinConfidence == 0.7 }},
		{"LOG_LEVEL", "debug", func(c Config) bool { return c.LogLevel == slog.LevelDebug }},
		{"LOG_LEVEL", "INFO", func(c Config) bool { return c.LogLevel == slog.LevelInfo }},
		{"LOG_LEVEL", "Warn", func(c Config) bool { return c.LogLevel == slog.LevelWarn }},
		{"LOG_LEVEL", "error", func(c Config) bool { return c.LogLevel == slog.LevelError }},
	} {
		t.Run(tt.key+"="+tt.value, func(t *testing.T) {
			configEnv(t)
			t.Setenv(tt.key, tt.value)
			got, err := Load()
			if err != nil || !tt.check(got) {
				t.Fatalf("valid boundary rejected or parsed incorrectly (error=%v)", err)
			}
		})
	}
}

func assertConfigError(t *testing.T, key string) {
	t.Helper()
	got, err := Load()
	if err == nil {
		t.Fatal("invalid configuration accepted")
	}
	if got != (Config{}) {
		t.Error("failed load returned partial configuration")
	}
	if !strings.Contains(err.Error(), key) {
		t.Errorf("error does not identify %s: %v", key, err)
	}
	for _, secret := range []string{testToken, testAPIKey} {
		if strings.Contains(err.Error(), secret) {
			t.Error("configuration error leaked a credential")
		}
	}
}

func TestLoadMissingCredentials(t *testing.T) {
	for _, key := range []string{"DISCORD_TOKEN", "TYPESAFE_API_KEY"} {
		for _, value := range []string{"unset", "", " \t\n"} {
			t.Run(key+"="+value, func(t *testing.T) {
				configEnv(t)
				if value == "unset" {
					if err := os.Unsetenv(key); err != nil {
						t.Fatal(err)
					}
				} else {
					t.Setenv(key, value)
				}
				assertConfigError(t, key)
			})
		}
	}
	t.Run("both missing", func(t *testing.T) {
		configEnv(t)
		t.Setenv("DISCORD_TOKEN", "")
		t.Setenv("TYPESAFE_API_KEY", "")
		assertConfigError(t, "DISCORD_TOKEN")
	})
}

func TestLoadInvalidValues(t *testing.T) {
	for _, tt := range []struct {
		key    string
		values []string
	}{
		{"BOT_REQUEST_TIMEOUT", []string{"0", "0s", "-1s", "30", "tomorrow", "1e3s", "999999999999999999999h", testToken}},
		{"BOT_MAX_CONCURRENT", []string{"0", "-1", "1.5", "1e2", "many", "999999999999999999999", testAPIKey}},
		{"BOT_MIN_CONFIDENCE", []string{"-0.01", "1.01", "NaN", "nan", "Inf", "+Inf", "-Inf", "Infinity", "1e999", "certain", testToken}},
		{"LOG_LEVEL", []string{"trace", "verbose", "fatal", "123", "info debug", testAPIKey}},
		{"TYPESAFE_LOG_LEVEL", []string{"trace", "verbose", "fatal", "123", "info debug", "info+1", "debug-1", "disabled", testToken, testAPIKey}},
	} {
		for _, value := range tt.values {
			t.Run(tt.key+"="+value, func(t *testing.T) {
				configEnv(t)
				t.Setenv(tt.key, " \t"+value+"\n")
				assertConfigError(t, tt.key)
			})
		}
	}
}
