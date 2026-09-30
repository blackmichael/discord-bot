package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gorilla/websocket"
)

func TestCancellationDuringGatewayStartup(t *testing.T) {
	for _, stage := range []string{"before hello", "before ready"} {
		t.Run(stage, func(t *testing.T) {
			t.Setenv("DISCORD_TOKEN", "test-token")
			t.Setenv("TYPESAFE_API_KEY", "test-key")
			t.Setenv("TYPESAFE_LOG_LEVEL", "off")
			for _, key := range []string{"BOT_REQUEST_TIMEOUT", "BOT_MAX_CONCURRENT", "BOT_MIN_CONFIDENCE", "LOG_LEVEL"} {
				t.Setenv(key, "")
			}
			oldLogger := slog.Default()
			t.Cleanup(func() { slog.SetDefault(oldLogger) })
			stalled := make(chan struct{})
			release := make(chan struct{})
			var gatewayURL string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/users/@me":
					_ = json.NewEncoder(w).Encode(map[string]string{"id": "123", "username": "PotatoBot"})
				case "/gateway":
					_ = json.NewEncoder(w).Encode(map[string]string{"url": gatewayURL})
				case "/ws/":
					conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
					if err != nil {
						t.Errorf("upgrade: %v", err)
						return
					}
					defer conn.Close()
					if stage == "before ready" {
						if err := conn.WriteJSON(map[string]any{"op": 10, "d": map[string]int{"heartbeat_interval": 45000}}); err != nil {
							t.Errorf("write hello: %v", err)
							return
						}
						if _, _, err := conn.ReadMessage(); err != nil {
							t.Errorf("read identify: %v", err)
							return
						}
					}
					close(stalled)
					<-release
				default:
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			defer close(release)
			gatewayURL = "ws" + strings.TrimPrefix(server.URL, "http") + "/ws"
			users, gateway := discordgo.EndpointUsers, discordgo.EndpointGateway
			discordgo.EndpointUsers = server.URL + "/users/"
			discordgo.EndpointGateway = server.URL + "/gateway"
			defer func() { discordgo.EndpointUsers, discordgo.EndpointGateway = users, gateway }()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- run(ctx) }()
			select {
			case <-stalled:
			case <-time.After(5 * time.Second):
				t.Fatal("gateway did not reach expected startup stage")
			}
			cancel()
			select {
			case err := <-result:
				if err != nil {
					t.Fatalf("startup cancellation: %v", err)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("startup did not stop on cancellation")
			}
		})
	}
}
