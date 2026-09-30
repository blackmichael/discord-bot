package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"potatobot/internal/config"

	typesafe "github.com/haileyok/typesafe-client/go"
)

func TestTypeSafeClientJSONLogging(t *testing.T) {
	const apiKey = "typesafe-test-secret-1234"
	const state = "potato \"state\"\nsecond line"
	const question = "Is this a potato?"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer "+apiKey {
			t.Error("request did not carry the unredacted credential")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"model":"jev-test","answers":{"potato":{"type":"noul","noul":0.9}}}`)
	}))
	defer server.Close()

	for _, tt := range []struct {
		level, botLevel string
		want            []string
	}{
		{"debug", "error", []string{"DEBUG typesafe: request", "INFO typesafe: response", "DEBUG typesafe: response body"}},
		{"info", "error", []string{"INFO typesafe: response"}},
		{"warn", "debug", nil},
		{"error", "debug", nil},
		{"off", "debug", nil},
	} {
		t.Run(tt.level, func(t *testing.T) {
			t.Setenv("DISCORD_TOKEN", "discord-test-token")
			t.Setenv("TYPESAFE_API_KEY", apiKey)
			for _, key := range []string{"BOT_REQUEST_TIMEOUT", "BOT_MAX_CONCURRENT", "BOT_MIN_CONFIDENCE"} {
				t.Setenv(key, "")
			}
			t.Setenv("LOG_LEVEL", tt.botLevel)
			t.Setenv("TYPESAFE_LOG_LEVEL", tt.level)
			cfg, err := config.Load()
			if err != nil {
				t.Fatal(err)
			}
			oldLogger := slog.Default()
			slog.SetDefault(slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: cfg.LogLevel})))
			t.Cleanup(func() { slog.SetDefault(oldLogger) })
			// The injected logger must win over the SDK's conflicting environment.
			t.Setenv("TYPESAFE_LOG_LEVEL", "off")
			if cfg.TypeSafeLogLevel == nil {
				t.Setenv("TYPESAFE_LOG_LEVEL", "debug")
			}
			var output bytes.Buffer
			logger := typeSafeLogger(&output, cfg.TypeSafeLogLevel)
			client, err := typesafe.NewClient(typesafe.WithAPIKey(apiKey), typesafe.WithBaseURL(server.URL),
				typesafe.WithModel("jev-test"), typesafe.WithLogger(logger), typesafe.WithRetryPolicy(typesafe.RetryPolicy{}))
			if err != nil {
				t.Fatal(err)
			}
			ctx := context.Background()
			if _, err := client.SystemOne(ctx, typesafe.Request{
				State: state, Questions: typesafe.Questions{"potato": typesafe.Noul(question)},
			}); err != nil {
				t.Fatal(err)
			}
			want := slices.Clone(tt.want)
			// The SDK emits only debug/info; probe higher levels to verify the threshold and off.
			for _, level := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
				logger.Log(ctx, level, "level probe")
				if cfg.TypeSafeLogLevel != nil && level >= *cfg.TypeSafeLogLevel {
					want = append(want, level.String()+" level probe")
				}
			}
			if strings.Contains(output.String(), apiKey) {
				t.Error("logs leaked the API key")
			}
			var got []string
			scanner := bufio.NewScanner(&output)
			for scanner.Scan() {
				var record struct {
					Level, Msg, Component, Body string
					Headers                     map[string]string
				}
				if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
					t.Fatalf("log record is not JSON: %v", err)
				}
				if record.Component != "typesafe" {
					t.Errorf("component = %q, want typesafe", record.Component)
				}
				got = append(got, record.Level+" "+record.Msg)
				if record.Msg == "typesafe: request" {
					if record.Headers["Authorization"] != "Bearer ***1234" {
						t.Error("logged Authorization header was not redacted as expected")
					}
					var body struct {
						State     string
						Questions map[string]struct{ Type, Instructions string }
					}
					if err := json.Unmarshal([]byte(record.Body), &body); err != nil {
						t.Fatalf("debug request body is not JSON: %v", err)
					}
					q := body.Questions["potato"]
					if body.State != state || q.Type != "noul" || q.Instructions != question {
						t.Error("debug request body did not preserve state and questions")
					}
				}
			}
			if err := scanner.Err(); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, want) {
				t.Errorf("records = %v, want %v", got, want)
			}
		})
	}
}
