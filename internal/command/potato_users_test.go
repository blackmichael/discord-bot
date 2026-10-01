package command

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestTaggedUserVerdictsAreSticky(t *testing.T) {
	// Each fresh router represents a restart: its first lookup must be a new assignment.
	for range 2 {
		var logs bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&logs, nil))
		router, err := NewRouter(nil, DefaultCommands(nil, logger), 1, logger)
		if err != nil {
			t.Fatal(err)
		}
		req := Request{Input: "is <@123> a potato?", AuthorID: "author", GuildID: "guild", ChannelID: "channel", BotID: "999", MentionedUserIDs: []string{"123"}}
		first, err := router.Route(context.Background(), req)
		if err != nil || !strings.HasPrefix(first, "<@123>: ") || (!strings.Contains(first, "**Yes**") && !strings.Contains(first, "**No**")) {
			t.Fatalf("random user verdict: reply=%q err=%v", first, err)
		}
		req.Input, req.AuthorID, req.GuildID, req.ChannelID = "what about <@!123>?", "other-author", "other-guild", "other-channel"
		second, err := router.Route(context.Background(), req)
		if err != nil || second != first {
			t.Fatalf("user-ID verdict changed: first=%q second=%q err=%v", first, second, err)
		}
		decoder := json.NewDecoder(&logs)
		for _, cached := range []bool{false, true} {
			var dispatch, verdict map[string]any
			if err := decoder.Decode(&dispatch); err != nil {
				t.Fatal(err)
			}
			if err := decoder.Decode(&verdict); err != nil {
				t.Fatal(err)
			}
			if dispatch["source"] != "local" || verdict["source"] != "sticky_random" || verdict["target_user_id"] != "123" || verdict["cached"] != cached {
				t.Errorf("sticky assignment log = %+v", verdict)
			}
			potato, ok := verdict["is_potato"].(bool)
			if !ok || potato != strings.Contains(first, "**Yes**") || verdict["model"] != nil || verdict["request_id"] != nil {
				t.Errorf("random verdict must agree with its reply, not claim a model result: %+v", verdict)
			}
		}
	}
}

func TestTaggedUsersAndPotatoBot(t *testing.T) {
	router, err := NewRouter(nil, DefaultCommands(nil, testLogger()), 1, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Input: "judge <@123> <@999> <@456>", BotID: "999", MentionedUserIDs: []string{"123", "999", "456", "123", ""}}
	reply, err := router.Route(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(reply, "\n\n")
	if len(parts) != 3 {
		t.Fatalf("expected one answer per unique user: %q", reply)
	}
	for i, id := range []string{"123", "999", "456"} {
		if !strings.HasPrefix(parts[i], "<@"+id+">: ") {
			t.Errorf("wrong target: %q", parts[i])
		}
	}
	if !strings.Contains(parts[1], "100.0%") || strings.Contains(parts[1], "Randomly assigned") {
		t.Errorf("PotatoBot must remain a known potato: %q", parts[1])
	}
	req.MentionedUserIDs = []string{"456", "123"}
	again, err := router.Route(context.Background(), req)
	if err != nil || again != parts[2]+"\n\n"+parts[0] {
		t.Fatalf("each user must keep its own answer: reply=%q err=%v", again, err)
	}
}
