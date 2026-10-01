package bot

import (
	"context"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"potatobot/internal/command"

	"github.com/bwmarrin/discordgo"
)

func TestTaggedUserMetadata(t *testing.T) {
	for _, tt := range []struct {
		name, content string
		mentions      []*discordgo.User
		wantIDs       []string
	}{
		{"standard", "<@123> is <@456> a potato?", []*discordgo.User{{ID: "123"}, {ID: "456"}}, []string{"456"}},
		{"nickname and duplicates", "<@!123> <@!456> <@456> <@789>", []*discordgo.User{{ID: "123"}, {ID: "456"}, {ID: "456"}, {ID: "789"}}, []string{"456", "789"}},
		{"before activation", "<@456> <@123> is this a potato?", []*discordgo.User{{ID: "123"}, {ID: "456"}}, nil},
		{"fake tags", "<@123> <@456> <@!789>", nil, nil},
		{"roles and plain names", "<@123> <@&456> @everyone @Someone", []*discordgo.User{{ID: "123"}, {ID: "456"}}, nil},
		{"bot as target", "<@123> is <@!123> a potato?", []*discordgo.User{nil, {}, {ID: "123"}}, []string{"123"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var got command.Request
			router := routerFunc(func(_ context.Context, req command.Request) (string, error) {
				got = req
				return "", nil
			})
			logger := slog.New(slog.NewTextHandler(io.Discard, nil))
			b := New(context.Background(), "123", nil, router, logger, time.Second, 1)
			event := testMessage(tt.content)
			event.Mentions = tt.mentions
			input, _ := MentionInput(event.Content, "123")
			b.respond(event.Message, input)
			if got.BotID != "123" || !reflect.DeepEqual(got.MentionedUserIDs, tt.wantIDs) {
				t.Errorf("mention metadata: bot=%q users=%v, want %v", got.BotID, got.MentionedUserIDs, tt.wantIDs)
			}
		})
	}
}

func TestReferencedMessageMetadata(t *testing.T) {
	var got command.Request
	router := routerFunc(func(_ context.Context, req command.Request) (string, error) {
		got = req
		return "", nil
	})
	b := New(context.Background(), "123", nil, router, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second, 1)
	event := testMessage("<@123>")
	event.Message.ReferencedMessage = &discordgo.Message{Content: "the post to judge"}
	input, _ := MentionInput(event.Content, "123")
	b.respond(event.Message, input)
	if got.Input != "" || got.ParentContent != "the post to judge" {
		t.Fatalf("reply metadata: input=%q parent=%q", got.Input, got.ParentContent)
	}
}

type fetchDiscord struct {
	discordFunc
	parent *discordgo.Message
}

func (d fetchDiscord) ChannelMessage(string, string, ...discordgo.RequestOption) (*discordgo.Message, error) {
	return d.parent, nil
}

func TestReferencedMessageFetch(t *testing.T) {
	var got command.Request
	router := routerFunc(func(_ context.Context, req command.Request) (string, error) {
		got = req
		return "", nil
	})
	discord := fetchDiscord{
		discordFunc: discordFunc(func(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error) {
			return nil, nil
		}),
		parent: &discordgo.Message{Content: "fetched post to judge"},
	}
	b := New(context.Background(), "123", discord, router, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second, 1)
	event := testMessage("<@123>")
	event.Message.MessageReference = &discordgo.MessageReference{ChannelID: "channel", MessageID: "parent"}
	input, _ := MentionInput(event.Content, "123")
	b.respond(event.Message, input)
	if got.ParentContent != "fetched post to judge" {
		t.Fatalf("fetched reply metadata: parent=%q", got.ParentContent)
	}
}
