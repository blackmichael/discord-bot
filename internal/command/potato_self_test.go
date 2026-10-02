package command

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"testing"

	typesafe "github.com/haileyok/typesafe-client/go"
)

func TestPotatoBotSelfQuestions(t *testing.T) {
	for _, input := range []string{
		"are you a potato?",
		"ARE YOU REALLY A POTATO?!",
		"are you yourself a potato",
		"are you also a real potato?",
		"is PotatoBot a potato?",
		"is PotatoBot itself actually a potato?",
		"is @PotatoBot definitely a potato.",
		"is this bot a potato?",
		"is the bot itself a literal potato!",
		" \tIs PotatoBot certainly a potato?\n",
	} {
		t.Run(input, func(t *testing.T) {
			var logs bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&logs, nil))
			calls := 0
			client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
				calls++
				return nil, errors.New("TypeSafe unavailable")
			})
			router, err := NewRouter(client, DefaultCommands(client, logger), 1, logger)
			if err != nil {
				t.Fatal(err)
			}
			req := Request{Input: input, AuthorID: "author", GuildID: "guild", ChannelID: "channel", MessageID: "message"}
			reply, err := router.Route(context.Background(), req)
			const suffix = "\n\npotato probability: 100.0%"
			if err != nil || calls != 0 || !strings.HasSuffix(reply, suffix) || !slices.Contains(potatoBotReplies, strings.TrimSuffix(reply, suffix)) {
				t.Fatalf("self-verdict: reply=%q calls=%d err=%v", reply, calls, err)
			}
			decoder := json.NewDecoder(&logs)
			var dispatch, verdict map[string]any
			if err := decoder.Decode(&dispatch); err != nil {
				t.Fatal(err)
			}
			if err := decoder.Decode(&verdict); err != nil {
				t.Fatal(err)
			}
			wantContext := map[string]any{"author_id": req.AuthorID, "guild_id": req.GuildID, "channel_id": req.ChannelID, "message_id": req.MessageID}
			if dispatch["msg"] != "request dispatched" || dispatch["source"] != "local" || dispatch["command"] != "potato" || dispatch["state"] != req.Input || !reflect.DeepEqual(dispatch["context"], wantContext) {
				t.Errorf("local dispatch log = %+v", dispatch)
			}
			if verdict["msg"] != "potato evaluated" || verdict["source"] != "known_fact" || verdict["potato_probability"] != 1.0 || verdict["band"] != "almost_certainly_potato" || verdict["state"] != req.Input || !reflect.DeepEqual(verdict["context"], wantContext) || verdict["model"] != nil || verdict["request_id"] != nil {
				t.Errorf("known-fact verdict log = %+v", verdict)
			}
			if err := decoder.Decode(&verdict); !errors.Is(err, io.EOF) {
				t.Fatalf("unexpected extra log: %v", err)
			}
		})
	}
	// Calling the handler directly must preserve the guarantee, independently of routing.
	reply, err := potatoCommand(nil, testLogger()).Handle(context.Background(), Request{Input: "are you a potato?"})
	if err != nil || !strings.Contains(reply, "100.0%") {
		t.Fatalf("direct self-verdict: reply=%q err=%v", reply, err)
	}
}

func TestPotatoBotFamilyQuestions(t *testing.T) {
	for _, input := range []string{
		"is your mom a potato?",
		"is your dad really a potato?",
		"is PotatoBot's brother an actual potato?",
		"are PotatoBot's parents potatoes?",
	} {
		t.Run(input, func(t *testing.T) {
			calls := 0
			client := clientFunc(func(context.Context, typesafe.Request) (*typesafe.Response, error) {
				calls++
				return nil, errors.New("TypeSafe unavailable")
			})
			router, err := NewRouter(client, DefaultCommands(client, testLogger()), 1, testLogger())
			if err != nil {
				t.Fatal(err)
			}
			reply, err := router.Route(context.Background(), Request{Input: input})
			if err != nil || calls != 0 || !strings.HasSuffix(reply, "\n\npotato probability: 100.0%") {
				t.Fatalf("family verdict: reply=%q calls=%d err=%v", reply, calls, err)
			}
			if !slices.Contains(potatoResponses[0].replies, strings.TrimSuffix(reply, "\n\npotato probability: 100.0%")) {
				t.Errorf("family verdict should use a certain potato response: %q", reply)
			}
		})
	}
}

func TestOtherSubjectsDoNotInheritPotatoBotIdentity(t *testing.T) {
	for _, input := range []string{
		"PotatoBot, is my laptop a potato?",
		"is PotatoBot's laptop a potato?",
		"is another bot a potato?",
		"is the other bot a potato?",
		"are you a potato or a laptop?",
		"do you think I am a potato?",
		"is PotatoBot a potato chip?",
	} {
		t.Run(input, func(t *testing.T) {
			calls := 0
			client := clientFunc(func(_ context.Context, req typesafe.Request) (*typesafe.Response, error) {
				calls++
				if req.State != input {
					t.Errorf("state changed: %v", req.State)
				}
				if _, routing := req.Questions["command"]; routing {
					return &typesafe.Response{Answers: map[string]typesafe.Answer{"command": typesafe.ChoiceAnswer{Choice: "potato", Confidence: 1}}}, nil
				}
				return &typesafe.Response{Answers: map[string]typesafe.Answer{"is_potato": typesafe.NoulAnswer{Noul: 0}}}, nil
			})
			router, err := NewRouter(client, DefaultCommands(client, testLogger()), 0.7, testLogger())
			if err != nil {
				t.Fatal(err)
			}
			reply, err := router.Route(context.Background(), Request{Input: input})
			if err != nil || calls != 2 {
				t.Fatalf("other subject: calls=%d reply=%q err=%v", calls, reply, err)
			}
			assertPotatoReply(t, reply, "almost_certainly_not_potato", 0)
		})
	}
}
