package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	typesafe "github.com/haileyok/typesafe-client/go"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRouterWithTypeSafeClient(t *testing.T) {
	for _, tt := range []struct {
		name, choice, input string
		confidence          float64
		status              int
		body                string
		wantHandler         bool
		wantError           bool
	}{
		{name: "dispatch", choice: "test", confidence: 0.9, wantHandler: true},
		{name: "at threshold", choice: "test", confidence: 0.7, wantHandler: true},
		{name: "low confidence", choice: "test", confidence: 0.69},
		{name: "unknown", choice: "unknown", confidence: 0.95},
		{name: "help not registered", input: "help", choice: "unknown", confidence: 0.95},
		{name: "unregistered", choice: "invented", confidence: 0.95, wantError: true},
		{name: "invalid confidence", choice: "test", confidence: 1.1, wantError: true},
		{name: "missing answer", body: `{"model":"jev-test","answers":{}}`, wantError: true},
		{name: "wrong type", body: `{"model":"jev-test","answers":{"command":{"type":"noul","noul":0.9}}}`, wantError: true},
		{name: "authentication error", status: http.StatusUnauthorized, body: `{"detail":"invalid key"}`, wantError: true},
		{name: "server error", status: http.StatusServiceUnavailable, body: `{"detail":"unavailable"}`, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := Request{Input: "please run the test", AuthorID: "author", GuildID: "guild", ChannelID: "channel", MessageID: "message"}
			if tt.input != "" {
				req.Input = tt.input
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
				}
				var payload struct {
					State     string `json:"state"`
					Model     string `json:"model"`
					Questions map[string]struct {
						Type         string            `json:"type"`
						Instructions string            `json:"instructions"`
						Criteria     map[string]string `json:"criteria"`
					} `json:"questions"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("decode request: %v", err)
				}
				q := payload.Questions["command"]
				if payload.State != req.Input || payload.Model != "jev-test" || len(payload.Questions) != 1 || q.Type != "choice" || q.Instructions == "" || len(q.Criteria) != 2 || q.Criteria["test"] != "Run the test handler" || q.Criteria["unknown"] == "" {
					t.Errorf("unexpected classification payload: %+v", payload)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-TypeSafe-Request-Id", "request-123")
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				if tt.body != "" {
					_, _ = io.WriteString(w, tt.body)
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{
					"model": "jev-test",
					"answers": map[string]any{"command": map[string]any{
						"type": "choice", "choice": tt.choice, "confidence": tt.confidence,
						"probabilities": map[string]float64{"test": 0.9, "unknown": 0.1},
					}},
				})
			}))
			defer server.Close()
			client, err := typesafe.NewClient(typesafe.WithAPIKey("test-key"), typesafe.WithBaseURL(server.URL), typesafe.WithModel("jev-test"), typesafe.WithRetryPolicy(typesafe.NoRetries()))
			if err != nil {
				t.Fatal(err)
			}
			called := false
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "value")
			router, err := NewRouter(client, []Command{{Name: "test", Description: "Run the test handler", Handle: func(handlerCtx context.Context, got Request) (string, error) {
				called = true
				if !reflect.DeepEqual(got, req) || handlerCtx.Value(contextKey{}) != "value" {
					t.Errorf("handler lost request or context: %+v", got)
				}
				return "handler reply", nil
			}}}, 0.7, testLogger())
			if err != nil {
				t.Fatal(err)
			}
			reply, err := router.Route(ctx, req)
			if (err != nil) != tt.wantError || called != tt.wantHandler {
				t.Fatalf("Route: reply=%q, err=%v, handler called=%v", reply, err, called)
			}
			if tt.wantHandler && reply != "handler reply" {
				t.Errorf("handler reply = %q", reply)
			}
			if !tt.wantHandler && !tt.wantError && !strings.Contains(reply, "help") {
				t.Errorf("missing fallback: %q", reply)
			}
		})
	}
}

type clientFunc func(context.Context, typesafe.Request) (*typesafe.Response, error)

func (f clientFunc) SystemOne(ctx context.Context, req typesafe.Request, _ ...typesafe.RequestOption) (*typesafe.Response, error) {
	return f(ctx, req)
}

func TestRouterPropagatesErrors(t *testing.T) {
	want := errors.New("handler failure")
	client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
		return &typesafe.Response{Answers: map[string]typesafe.Answer{"command": typesafe.ChoiceAnswer{Choice: "test", Confidence: 1}}}, nil
	})
	router, err := NewRouter(client, []Command{{Name: "test", Description: "Test", Handle: func(context.Context, Request) (string, error) { return "", want }}}, 0.7, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := router.Route(context.Background(), Request{Input: "test"}); !errors.Is(err, want) {
		t.Fatalf("handler error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	router.client = clientFunc(func(ctx context.Context, _ typesafe.Request) (*typesafe.Response, error) { return nil, ctx.Err() })
	if _, err := router.Route(ctx, Request{Input: "test"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation error = %v", err)
	}
	router.client = clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) { return nil, nil })
	if _, err := router.Route(context.Background(), Request{Input: "test"}); err == nil {
		t.Fatal("nil response accepted")
	}
}

func TestCommandRegistration(t *testing.T) {
	handler := func(context.Context, Request) (string, error) { return "ok", nil }
	valid := Command{Name: "test", Description: "Test", Handle: handler}
	for _, commands := range [][]Command{
		{{Name: "", Description: "Test", Handle: handler}},
		{{Name: "unknown", Description: "Test", Handle: handler}},
		{{Name: "test", Handle: handler}},
		{{Name: "test", Description: "Test"}},
		{valid, valid},
	} {
		if _, err := NewRouter(nil, commands, 0.7, testLogger()); err == nil {
			t.Errorf("invalid commands accepted: %+v", commands)
		}
	}
	for _, threshold := range []float64{-0.1, 1.1, math.NaN(), math.Inf(1)} {
		if _, err := NewRouter(nil, []Command{valid}, threshold, testLogger()); err == nil {
			t.Errorf("invalid confidence accepted: %v", threshold)
		}
	}
	commands := DefaultCommands(nil, testLogger())
	if len(commands) != 4 || commands[0].Name != "help" || commands[1].Name != "potato" || commands[2].Name != "opinion" || commands[3].Name != "hot_take" {
		t.Fatalf("expected help, potato, opinion, and hot_take commands: %+v", commands)
	}
	reply, err := commands[0].Handle(context.Background(), Request{})
	if err != nil || !strings.Contains(reply, "@potatobot") {
		t.Fatalf("help reply = %q, err=%v", reply, err)
	}
	for _, command := range []string{"`help` -", "`potato` -", "`opinion` -", "`/hot take` -"} {
		if !strings.Contains(reply, command) {
			t.Errorf("help reply missing %q: %q", command, reply)
		}
	}
}

func TestHelpShortcut(t *testing.T) {
	for _, tt := range []struct {
		input   string
		wantAPI bool
	}{
		{input: "help"},
		{input: "HELP"},
		{input: " \tHelp\n"},
		{input: "what can you do?", wantAPI: true},
		{input: "help with this", wantAPI: true},
		{input: "helpful", wantAPI: true},
	} {
		t.Run(tt.input, func(t *testing.T) {
			req := Request{Input: tt.input, AuthorID: "author", MessageID: "message"}
			calledAPI, calledHandler := false, false
			client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
				calledAPI = true
				return &typesafe.Response{Answers: map[string]typesafe.Answer{"command": typesafe.ChoiceAnswer{Choice: "help", Confidence: 1}}}, nil
			})
			wantErr := errors.New("handler error")
			type contextKey struct{}
			ctx := context.WithValue(context.Background(), contextKey{}, "value")
			commands := DefaultCommands(client, testLogger())
			commands[0].Handle = func(gotCtx context.Context, got Request) (string, error) {
				calledHandler = true
				if !reflect.DeepEqual(got, req) || gotCtx.Value(contextKey{}) != "value" {
					t.Errorf("handler lost context or request: %+v", got)
				}
				return "help reply", wantErr
			}
			router, err := NewRouter(client, commands, 0.7, testLogger())
			if err != nil {
				t.Fatal(err)
			}
			reply, err := router.Route(ctx, req)
			if calledAPI != tt.wantAPI || !calledHandler || reply != "help reply" || !errors.Is(err, wantErr) {
				t.Fatalf("API=%v handler=%v reply=%q err=%v", calledAPI, calledHandler, reply, err)
			}
		})
	}
}

func TestParentContentDispatchesOpinion(t *testing.T) {
	called := false
	router, err := NewRouter(nil, []Command{{
		Name:        "opinion",
		Description: "Analyze an opinion",
		Handle: func(context.Context, Request) (string, error) {
			called = true
			return "opinion reply", nil
		},
	}}, 0.7, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	reply, err := router.Route(context.Background(), Request{ParentContent: "the post"})
	if err != nil || reply != "opinion reply" || !called {
		t.Fatalf("parent dispatch: reply=%q err=%v called=%v", reply, err, called)
	}
}

func TestOpinionQuestionDispatchesOpinion(t *testing.T) {
	for _, input := range []string{"do you think pineapple belongs on pizza?", "what do you think about this?"} {
		t.Run(input, func(t *testing.T) {
			called := false
			router, err := NewRouter(nil, []Command{{
				Name:        "opinion",
				Description: "Analyze an opinion",
				Handle: func(context.Context, Request) (string, error) {
					called = true
					return "opinion reply", nil
				},
			}}, 0.7, testLogger())
			if err != nil {
				t.Fatal(err)
			}
			reply, err := router.Route(context.Background(), Request{Input: input})
			if err != nil || reply != "opinion reply" || !called {
				t.Fatalf("opinion question dispatch: reply=%q err=%v called=%v", reply, err, called)
			}
		})
	}
}

func TestHotTakeQuestionDispatchesHotTake(t *testing.T) {
	called := false
	router, err := NewRouter(nil, []Command{{
		Name:        "hot_take",
		Description: "Rate a take",
		Handle: func(context.Context, Request) (string, error) {
			called = true
			return "hot take reply", nil
		},
	}}, 0.7, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	reply, err := router.Route(context.Background(), Request{Input: "/hot take pineapple belongs on pizza"})
	if err != nil || reply != "hot take reply" || !called {
		t.Fatalf("hot take dispatch: reply=%q err=%v called=%v", reply, err, called)
	}
}

func TestRequestLogging(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "success", true: "API failure"}[failure], func(t *testing.T) {
			var logs bytes.Buffer
			req := Request{Input: "what can you do?\n\"please\"", AuthorID: "author", GuildID: "guild", ChannelID: "channel", MessageID: "message"}
			contextFields := map[string]any{"author_id": req.AuthorID, "guild_id": req.GuildID, "channel_id": req.ChannelID, "message_id": req.MessageID}
			client := clientFunc(func(_ context.Context, classification typesafe.Request) (*typesafe.Response, error) {
				var record map[string]any
				if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &record); err != nil {
					t.Fatalf("request must be logged before the API call as JSON: %v", err)
				}
				questionsJSON, err := json.Marshal(classification.Questions)
				if err != nil {
					t.Fatal(err)
				}
				var questions map[string]any
				if err := json.Unmarshal(questionsJSON, &questions); err != nil {
					t.Fatal(err)
				}
				if record["msg"] != "classifying request" || record["level"] != "INFO" || record["state"] != classification.State || !reflect.DeepEqual(record["questions"], questions) || !reflect.DeepEqual(record["context"], contextFields) {
					t.Errorf("request log doesn't match questions and context: %+v", record)
				}
				if failure {
					return nil, errors.New("API unavailable")
				}
				return &typesafe.Response{Model: "jev-test", RequestID: "request-123", Answers: map[string]typesafe.Answer{"command": typesafe.ChoiceAnswer{Choice: "help", Confidence: 0.9}}}, nil
			})
			router, err := NewRouter(client, DefaultCommands(client, testLogger()), 0.7, slog.New(slog.NewJSONHandler(&logs, nil)))
			if err != nil {
				t.Fatal(err)
			}
			if _, err := router.Route(context.Background(), req); (err != nil) != failure {
				t.Fatalf("Route error = %v", err)
			}
			decoder := json.NewDecoder(&logs)
			var requestLog map[string]any
			if err := decoder.Decode(&requestLog); err != nil {
				t.Fatal(err)
			}
			var decision map[string]any
			if failure {
				if err := decoder.Decode(&decision); !errors.Is(err, io.EOF) {
					t.Fatalf("unexpected classification decision after API failure: %v", err)
				}
				return
			}
			if err := decoder.Decode(&decision); err != nil {
				t.Fatal(err)
			}
			if decision["msg"] != "request classified" || decision["command"] != "help" || decision["confidence"] != 0.9 || decision["model"] != "jev-test" || decision["request_id"] != "request-123" || !reflect.DeepEqual(decision["context"], contextFields) {
				t.Errorf("classification log missing decision or correlation fields: %+v", decision)
			}
		})
	}
}

func TestLocalHelpLogging(t *testing.T) {
	var logs bytes.Buffer
	router, err := NewRouter(nil, DefaultCommands(nil, testLogger()), 0.7, slog.New(slog.NewJSONHandler(&logs, nil)))
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Input: "help", MessageID: "message"}
	if _, err := router.Route(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &record); err != nil {
		t.Fatal(err)
	}
	if record["msg"] != "request dispatched" || record["source"] != "local" || record["command"] != "help" || record["state"] != req.Input || record["questions"] != nil || record["context"].(map[string]any)["message_id"] != req.MessageID {
		t.Errorf("local help log = %+v", record)
	}
}

func TestRequestLogsCanBeSuppressed(t *testing.T) {
	var logs bytes.Buffer
	calledAPI := false
	client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
		calledAPI = true
		return &typesafe.Response{Answers: map[string]typesafe.Answer{"command": typesafe.ChoiceAnswer{Choice: "help", Confidence: 1}}}, nil
	})
	router, err := NewRouter(client, DefaultCommands(client, testLogger()), 0.7, slog.New(slog.NewJSONHandler(&logs, &slog.HandlerOptions{Level: slog.LevelWarn})))
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"help", "what can you do?"} {
		if _, err := router.Route(context.Background(), Request{Input: input}); err != nil {
			t.Fatal(err)
		}
	}
	if !calledAPI || logs.Len() != 0 {
		t.Fatalf("warn level should suppress payload logs without preventing routing: API=%v logs=%q", calledAPI, logs.String())
	}
}
