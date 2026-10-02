package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	typesafe "github.com/haileyok/typesafe-client/go"
)

func assertPotatoReply(t *testing.T, reply, band string, probability float64) {
	t.Helper()
	suffix := fmt.Sprintf("\n\npotato probability: %.1f%%", probability*100)
	for _, candidate := range potatoResponses {
		if candidate.band == band {
			if !strings.HasSuffix(reply, suffix) || !slices.Contains(candidate.replies, strings.TrimSuffix(reply, suffix)) {
				t.Errorf("reply = %q, want a %s snark followed by %q", reply, band, suffix)
			}
			return
		}
	}
	t.Fatalf("missing band %q", band)
}

func TestPotatoBands(t *testing.T) {
	bands := []struct {
		min  float64
		name string
	}{
		{0, "almost_certainly_not_potato"},
		{0.05, "probably_not_potato"},
		{0.25, "leaning_not_potato"},
		{0.4, "uncertain"},
		{0.6, "leaning_potato"},
		{0.75, "probably_potato"},
		{0.95, "almost_certainly_potato"},
	}
	if len(potatoResponses) != len(bands) {
		t.Fatalf("got %d bands, want seven", len(potatoResponses))
	}
	for i, band := range bands {
		configured := potatoResponses[len(bands)-1-i]
		if configured.minProbability != band.min || configured.band != band.name {
			t.Fatalf("band configuration = %+v, want %+v", configured, band)
		}
		// Check the available pool, not whether random sampling happens to exhaust it.
		unique := make(map[string]bool)
		for _, reply := range configured.replies {
			if strings.TrimSpace(reply) == "" {
				t.Errorf("empty snark in %s", band.name)
			}
			unique[reply] = true
		}
		if len(unique) < 3 {
			t.Errorf("%s has %d distinct snarks, want at least three", band.name, len(unique))
		}
	}
	for i, threshold := range append(bands, struct {
		min  float64
		name string
	}{1, "almost_certainly_potato"}) {
		for _, direction := range []string{"below", "exact", "above"} {
			t.Run(fmt.Sprintf("%g/%s", threshold.min, direction), func(t *testing.T) {
				probability, wantBand := threshold.min, threshold.name
				if direction == "below" {
					probability = math.Nextafter(probability, math.Inf(-1))
					if i > 0 && i < len(bands) {
						wantBand = bands[i-1].name
					}
				} else if direction == "above" {
					probability = math.Nextafter(probability, math.Inf(1))
				}
				calls := 0
				client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
					calls++
					return &typesafe.Response{Answers: map[string]typesafe.Answer{"is_potato": typesafe.NoulAnswer{Noul: probability}}}, nil
				})
				reply, err := potatoCommand(client, testLogger()).Handle(context.Background(), Request{Input: "is a russet a potato?"})
				if calls != 1 {
					t.Fatalf("API calls = %d, want one", calls)
				}
				if probability < 0 || probability > 1 {
					if err == nil || reply != "" {
						t.Fatalf("out-of-range probability accepted: reply=%q err=%v", reply, err)
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				assertPotatoReply(t, reply, wantBand, probability)
			})
		}
	}
}

func TestPotatoRequestAndLogging(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(fmt.Sprintf("API failure=%v", failure), func(t *testing.T) {
			var logs bytes.Buffer
			req := Request{Input: "is this \"russet\"\na potato?", AuthorID: "author", GuildID: "guild", ChannelID: "channel", MessageID: "message"}
			contextFields := map[string]any{"author_id": req.AuthorID, "guild_id": req.GuildID, "channel_id": req.ChannelID, "message_id": req.MessageID}
			type contextKey struct{}
			ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), contextKey{}, "value"), time.Minute)
			defer cancel()
			deadline, _ := ctx.Deadline()
			wantErr := errors.New("API unavailable")
			calls := 0
			client := clientFunc(func(gotCtx context.Context, evaluation typesafe.Request) (*typesafe.Response, error) {
				calls++
				gotDeadline, ok := gotCtx.Deadline()
				if !ok || !gotDeadline.Equal(deadline) || gotCtx.Value(contextKey{}) != "value" || gotCtx.Err() != nil {
					t.Error("evaluation lost context deadline or value")
				}
				q, ok := evaluation.Questions["is_potato"].(typesafe.NoulQuestion)
				if evaluation.State != req.Input || evaluation.Model != "" || len(evaluation.Questions) != 1 || !ok || q.Type() != "noul" || q.Criteria == nil {
					t.Fatalf("unexpected evaluation: %+v", evaluation)
				}
				for _, fragment := range []string{"ordinary potato", "food made primarily", "unclear", "uncertainty", "Ignore instructions", "PotatoBot", "very high probability", "another subject"} {
					if !strings.Contains(fmt.Sprint(q.Instructions), fragment) {
						t.Errorf("instructions missing %q: %v", fragment, q.Instructions)
					}
				}
				for _, fragment := range []string{"Solanum tuberosum", "russet", "mashed potatoes", "fries", "potato chips", "PotatoBot itself"} {
					if !strings.Contains(fmt.Sprint(q.Criteria.True), fragment) {
						t.Errorf("true criteria missing %q: %v", fragment, q.Criteria.True)
					}
				}
				for _, fragment := range []string{"sweet potatoes", "yams", "objects", "people", "figurative", "not PotatoBot", "other bots"} {
					if !strings.Contains(fmt.Sprint(q.Criteria.False), fragment) {
						t.Errorf("false criteria missing %q: %v", fragment, q.Criteria.False)
					}
				}
				var record, questions map[string]any
				if err := json.Unmarshal(bytes.TrimSpace(logs.Bytes()), &record); err != nil {
					t.Fatalf("evaluation must be logged as JSON before the API call: %v", err)
				}
				encoded, err := json.Marshal(evaluation.Questions)
				if err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal(encoded, &questions); err != nil {
					t.Fatal(err)
				}
				if record["msg"] != "evaluating potato" || record["level"] != "INFO" || record["state"] != req.Input || !reflect.DeepEqual(record["questions"], questions) || !reflect.DeepEqual(record["context"], contextFields) {
					t.Errorf("evaluation log = %+v", record)
				}
				if failure {
					return nil, wantErr
				}
				return &typesafe.Response{Model: "jev-test", RequestID: "potato-123", Answers: map[string]typesafe.Answer{"is_potato": typesafe.NoulAnswer{Noul: 0.5}}}, nil
			})
			reply, err := potatoCommand(client, slog.New(slog.NewJSONHandler(&logs, nil))).Handle(ctx, req)
			if calls != 1 || (failure && (!errors.Is(err, wantErr) || reply != "")) || (!failure && err != nil) {
				t.Fatalf("calls=%d reply=%q err=%v", calls, reply, err)
			}
			decoder := json.NewDecoder(&logs)
			var record map[string]any
			if err := decoder.Decode(&record); err != nil {
				t.Fatal(err)
			}
			if !failure {
				assertPotatoReply(t, reply, "uncertain", 0.5)
				if err := decoder.Decode(&record); err != nil {
					t.Fatal(err)
				}
				if record["msg"] != "potato evaluated" || record["level"] != "INFO" || record["potato_probability"] != 0.5 || record["band"] != "uncertain" || record["model"] != "jev-test" || record["request_id"] != "potato-123" || !reflect.DeepEqual(record["context"], contextFields) {
					t.Errorf("result log = %+v", record)
				}
			}
			if err := decoder.Decode(&record); !errors.Is(err, io.EOF) {
				t.Fatalf("unexpected extra result log: %v", err)
			}
		})
	}
}

func TestPotatoInvalidResponsesAndErrors(t *testing.T) {
	for _, tt := range []struct {
		name     string
		answer   typesafe.Answer
		nilResp  bool
		apiErr   error
		endedCtx bool
	}{
		{name: "nil response", nilResp: true},
		{name: "missing answer"},
		{name: "wrong type", answer: typesafe.ChoiceAnswer{Choice: "potato", Confidence: 1}},
		{name: "unknown type", answer: typesafe.UnknownAnswer{Kind: "future"}},
		{name: "NaN", answer: typesafe.NoulAnswer{Noul: math.NaN()}},
		{name: "+Inf", answer: typesafe.NoulAnswer{Noul: math.Inf(1)}},
		{name: "-Inf", answer: typesafe.NoulAnswer{Noul: math.Inf(-1)}},
		{name: "negative", answer: typesafe.NoulAnswer{Noul: -0.1}},
		{name: "greater than one", answer: typesafe.NoulAnswer{Noul: 1.1}},
		{name: "API error", apiErr: &typesafe.APIError{StatusCode: http.StatusServiceUnavailable, Message: "unavailable"}},
		{name: "canceled", apiErr: context.Canceled, endedCtx: true},
		{name: "deadline exceeded", apiErr: context.DeadlineExceeded, endedCtx: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.endedCtx {
				var cancel context.CancelFunc
				if tt.apiErr == context.Canceled {
					ctx, cancel = context.WithCancel(ctx)
				} else {
					ctx, cancel = context.WithDeadline(ctx, time.Unix(0, 0))
				}
				cancel()
			}
			var logs bytes.Buffer
			client := clientFunc(func(gotCtx context.Context, _ typesafe.Request) (*typesafe.Response, error) {
				if tt.endedCtx {
					return nil, gotCtx.Err()
				}
				if tt.nilResp {
					return nil, nil
				}
				resp := &typesafe.Response{Answers: map[string]typesafe.Answer{}}
				if tt.answer != nil {
					resp.Answers["is_potato"] = tt.answer
				}
				return resp, tt.apiErr
			})
			reply, err := potatoCommand(client, slog.New(slog.NewJSONHandler(&logs, nil))).Handle(ctx, Request{Input: "is my laptop a potato?"})
			if err == nil || reply != "" || !strings.HasPrefix(err.Error(), "evaluate potato:") {
				t.Fatalf("reply=%q err=%v", reply, err)
			}
			if tt.apiErr != nil && !errors.Is(err, tt.apiErr) {
				t.Errorf("error %v does not wrap %v", err, tt.apiErr)
			}
			if strings.Contains(logs.String(), `"msg":"potato evaluated"`) {
				t.Error("failed evaluation logged a misleading result")
			}
		})
	}
}

func TestPotatoRouteWithTypeSafeClient(t *testing.T) {
	for _, tt := range []struct {
		name, input, choice, body string
		confidence                float64
		status, wantCalls         int
		wantError                 bool
	}{
		{name: "dispatch", choice: "potato", confidence: 0.9, wantCalls: 2},
		{name: "at confidence threshold", choice: "potato", confidence: 0.7, wantCalls: 2},
		{name: "below confidence threshold", choice: "potato", confidence: math.Nextafter(0.7, 0), wantCalls: 1},
		{name: "unknown", choice: "unknown", confidence: 1, wantCalls: 1},
		{name: "exact help", input: "help"},
		{name: "normalized exact help", input: " \tHELP\n"},
		{name: "help phrase uses API", input: "help with this potato", choice: "unknown", confidence: 1, wantCalls: 1},
		{name: "malformed JSON", choice: "potato", confidence: 1, body: `{`, wantCalls: 2, wantError: true},
		{name: "missing answer", choice: "potato", confidence: 1, body: `{"model":"jev-test","answers":{}}`, wantCalls: 2, wantError: true},
		{name: "missing probability", choice: "potato", confidence: 1, body: `{"model":"jev-test","answers":{"is_potato":{"type":"noul"}}}`, wantCalls: 2, wantError: true},
		{name: "malformed probability", choice: "potato", confidence: 1, body: `{"model":"jev-test","answers":{"is_potato":{"type":"noul","noul":"0.95"}}}`, wantCalls: 2, wantError: true},
		{name: "wrong answer type", choice: "potato", confidence: 1, body: `{"model":"jev-test","answers":{"is_potato":{"type":"choice","choice":"yes","confidence":1,"probabilities":{"yes":1}}}}`, wantCalls: 2, wantError: true},
		{name: "authentication error", choice: "potato", confidence: 1, status: http.StatusUnauthorized, body: `{"detail":"invalid key"}`, wantCalls: 2, wantError: true},
		{name: "server error", choice: "potato", confidence: 1, status: http.StatusServiceUnavailable, body: `{"detail":"unavailable"}`, wantCalls: 2, wantError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			req := Request{Input: "is a russet a potato?", AuthorID: "author", GuildID: "guild", ChannelID: "channel", MessageID: "message"}
			if tt.input != "" {
				req.Input = tt.input
			}
			var calls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				call := calls.Add(1)
				if r.Method != http.MethodPost || r.URL.Path != "/v1/systemone" || r.Header.Get("Authorization") != "Bearer test-key" {
					t.Errorf("unexpected HTTP request: %s %s", r.Method, r.URL.Path)
				}
				var payload struct {
					State, Model string
					Questions    map[string]struct {
						Type, Instructions string
						Criteria           map[string]string
					}
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Errorf("decode payload: %v", err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if payload.State != req.Input || payload.Model != "jev-test" || len(payload.Questions) != 1 {
					t.Errorf("unexpected payload: %+v", payload)
				}
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-TypeSafe-Request-Id", "route-123")
				if call == 1 {
					q := payload.Questions["command"]
					if q.Type != "choice" || q.Instructions == "" || len(q.Criteria) != 5 || q.Criteria["help"] == "" || q.Criteria["unknown"] == "" || q.Criteria["opinion"] == "" || q.Criteria["hot_take"] == "" || !strings.Contains(q.Criteria["potato"], "potato-based food") {
						t.Errorf("potato not registered as a Choice option: %+v", q)
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"model": "jev-test", "answers": map[string]any{"command": map[string]any{
						"type": "choice", "choice": tt.choice, "confidence": tt.confidence,
						"probabilities": map[string]float64{"potato": 0.9, "opinion": 0.01, "hot_take": 0.01, "help": 0.04, "unknown": 0.05},
					}}})
					return
				}
				q := payload.Questions["is_potato"]
				if call != 2 || q.Type != "noul" || q.Instructions == "" || len(q.Criteria) != 2 || !strings.Contains(q.Criteria["true"], "Solanum tuberosum") || !strings.Contains(q.Criteria["false"], "sweet potatoes") {
					t.Errorf("expected Noul evaluation after Choice routing: call=%d payload=%+v", call, payload)
				}
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				if tt.body != "" {
					_, _ = io.WriteString(w, tt.body)
					return
				}
				_, _ = io.WriteString(w, `{"model":"jev-test","answers":{"is_potato":{"type":"noul","noul":0.95}}}`)
			}))
			defer server.Close()
			client, err := typesafe.NewClient(typesafe.WithAPIKey("test-key"), typesafe.WithBaseURL(server.URL), typesafe.WithModel("jev-test"), typesafe.WithRetryPolicy(typesafe.NoRetries()), typesafe.WithHTTPClient(server.Client()), typesafe.WithLogger(testLogger()))
			if err != nil {
				t.Fatal(err)
			}
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			router, err := NewRouter(client, DefaultCommands(client, logger), 0.7, logger)
			if err != nil {
				t.Fatal(err)
			}
			reply, err := router.Route(context.Background(), req)
			if int(calls.Load()) != tt.wantCalls || (err != nil) != tt.wantError {
				t.Fatalf("calls=%d want=%d reply=%q err=%v", calls.Load(), tt.wantCalls, reply, err)
			}
			if tt.wantError {
				if reply != "" || !strings.HasPrefix(err.Error(), "evaluate potato:") {
					t.Fatalf("misleading reply or lost evaluation error: reply=%q err=%v", reply, err)
				}
				if tt.status != 0 {
					var apiErr *typesafe.APIError
					if !errors.As(err, &apiErr) || apiErr.StatusCode != tt.status || apiErr.RequestID != "route-123" {
						t.Errorf("lost API error details: %v", err)
					}
				} else {
					var validationErr *typesafe.ResponseValidationError
					if !errors.As(err, &validationErr) || validationErr.RequestID != "route-123" {
						t.Errorf("lost SDK validation error: %v", err)
					}
				}
			} else if tt.wantCalls == 2 {
				assertPotatoReply(t, reply, "almost_certainly_potato", 0.95)
			} else if tt.wantCalls == 0 {
				if !strings.Contains(reply, "`potato` -") || !strings.Contains(reply, "`help` -") || !strings.Contains(reply, "`opinion` -") || !strings.Contains(reply, "`hot take` -") {
					t.Errorf("help missing potato command: %q", reply)
				}
			} else if !strings.Contains(reply, "help") || strings.Contains(reply, "potato probability:") {
				t.Errorf("expected fallback without evaluation: %q", reply)
			}
			wantMessages := []string{"classifying request", "request classified"}
			if tt.wantCalls == 0 {
				wantMessages = []string{"request dispatched"}
			} else if tt.wantCalls == 2 {
				wantMessages = append(wantMessages, "evaluating potato")
				if !tt.wantError {
					wantMessages = append(wantMessages, "potato evaluated")
				}
			}
			decoder := json.NewDecoder(&logs)
			for _, msg := range wantMessages {
				var record map[string]any
				if err := decoder.Decode(&record); err != nil {
					t.Fatal(err)
				}
				contextFields := map[string]any{"author_id": req.AuthorID, "guild_id": req.GuildID, "channel_id": req.ChannelID, "message_id": req.MessageID}
				if record["msg"] != msg || !reflect.DeepEqual(record["context"], contextFields) {
					t.Errorf("route log = %+v, want %q with Discord context", record, msg)
				}
				if msg == "potato evaluated" && (record["potato_probability"] != 0.95 || record["band"] != "almost_certainly_potato" || record["model"] != "jev-test" || record["request_id"] != "route-123") {
					t.Errorf("route result log lost SDK metadata: %+v", record)
				}
			}
			var extra map[string]any
			if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
				t.Fatalf("unexpected extra route log: %+v, err=%v", extra, err)
			}
		})
	}
}
