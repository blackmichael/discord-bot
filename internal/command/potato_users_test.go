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
		if err != nil || !strings.HasPrefix(first, "<@123>: ") || !strings.Contains(first, "\n\nPotato probability: ") || strings.Contains(first, "Potato verdict:") {
			t.Fatalf("tagged user format: reply=%q err=%v", first, err)
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
			potato, potatoOK := verdict["is_potato"].(bool)
			probability, probabilityOK := verdict["potato_probability"].(float64)
			if !potatoOK || !probabilityOK || (potato && probability < taggedUserPotatoChance) || (!potato && probability >= 1-taggedUserPotatoChance) || verdict["model"] != nil || verdict["request_id"] != nil {
				t.Errorf("random verdict must use a matching probability, not claim a model result: %+v", verdict)
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
	parts := strings.Split(reply, "\n\n<@")
	for i := 1; i < len(parts); i++ {
		parts[i] = "<@" + parts[i]
	}
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

func TestAlwaysPotatoUsers(t *testing.T) {
	ids := []string{"314389700179918850", "314475455057231882", "323955224979177472"}
	router, err := NewRouter(nil, DefaultCommands(nil, testLogger()), 1, testLogger())
	if err != nil {
		t.Fatal(err)
	}
	req := Request{Input: "judge these users", MentionedUserIDs: ids}
	reply, err := router.Route(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(reply, "\n\n<@")
	for i := 1; i < len(parts); i++ {
		parts[i] = "<@" + parts[i]
	}
	if len(parts) != len(ids) {
		t.Fatalf("expected one answer per override: %q", reply)
	}
	for i, id := range ids {
		if !strings.HasPrefix(parts[i], "<@"+id+">: ") || !strings.Contains(parts[i], "Potato probability: 100.0%.") {
			t.Errorf("override reply = %q", parts[i])
		}
	}
	again, err := router.Route(context.Background(), req)
	if err != nil || again != reply {
		t.Fatalf("override reply changed: first=%q second=%q err=%v", reply, again, err)
	}
}
