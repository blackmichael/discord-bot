package bot

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf16"
	"unicode/utf8"

	"potatobot/internal/command"

	"github.com/bwmarrin/discordgo"
)

type routerFunc func(context.Context, command.Request) (string, error)

func (f routerFunc) Route(ctx context.Context, req command.Request) (string, error) {
	return f(ctx, req)
}

type discordFunc func(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error)

func (f discordFunc) ChannelMessageSendComplex(channel string, msg *discordgo.MessageSend, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
	return f(channel, msg, opts...)
}

func testBot(t *testing.T, router Router, discord Discord, timeout time.Duration, limit int) (*Bot, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	b := New(ctx, "123", discord, router, slog.New(slog.NewTextHandler(io.Discard, nil)), timeout, limit)
	t.Cleanup(func() {
		cancel()
		b.Stop()
	})
	return b, cancel
}

func testMessage(content string) *discordgo.MessageCreate {
	return &discordgo.MessageCreate{Message: &discordgo.Message{
		ID: "message", ChannelID: "channel", GuildID: "guild", Content: content,
		Author: &discordgo.User{ID: "author"},
	}}
}

// Timeouts are deadlock guards, not synchronization for the assertions.
func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for worker")
		var zero T
		return zero
	}
}

func TestMentionInput(t *testing.T) {
	for _, tt := range []struct {
		name, content, id, want string
		mentioned               bool
	}{
		{"standard", "<@123> help", "123", "help", true},
		{"nickname", "<@!123> help", "123", "help", true},
		{"trim", "prefix <@123> \t help\n please \r\n", "123", "help\n please", true},
		{"standard first", "before <@123> first <@!123> second", "123", "first <@!123> second", true},
		{"nickname first", "before <@!123> first <@123> second", "123", "first <@123> second", true},
		{"other mention first", "<@999> ignore <@!123> help", "123", "help", true},
		{"repeated", "<@123> one <@123> two", "123", "one <@123> two", true},
		{"display name", "@PotatoBot help", "123", "", false},
		{"other IDs", "<@12> <@1234> <@!999>", "123", "", false},
		{"role", "<@&123> help", "123", "", false},
		{"malformed", "<@123 <@!123 >", "123", "", false},
		{"empty content", "", "123", "", false},
		{"empty ID", "<@> help", "", "", false},
		{"mention only", "<@123>", "123", "", true},
		{"whitespace only", "prefix <@!123> \n\t", "123", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, mentioned := MentionInput(tt.content, tt.id)
			if got != tt.want || mentioned != tt.mentioned {
				t.Fatalf("MentionInput(%q, %q) = (%q, %v), want (%q, %v)", tt.content, tt.id, got, mentioned, tt.want, tt.mentioned)
			}
		})
	}
}

func TestHandleIgnoresMessages(t *testing.T) {
	for _, name := range []string{"nil event", "nil message", "nil author", "bot", "webhook", "self", "unmentioned", "other ID", "display name"} {
		t.Run(name, func(t *testing.T) {
			event := testMessage("<@123> help")
			switch name {
			case "nil event":
				event = nil
			case "nil message":
				event.Message = nil
			case "nil author":
				event.Author = nil
			case "bot":
				event.Author.Bot = true
			case "webhook":
				event.WebhookID = "webhook"
			case "self":
				event.Author.ID = "123"
			case "unmentioned":
				event.Content = "help"
			case "other ID":
				event.Content = "<@999> help"
			case "display name":
				event.Content = "@PotatoBot help"
			}
			var routes, replies atomic.Int32
			b, _ := testBot(t, routerFunc(func(context.Context, command.Request) (string, error) {
				routes.Add(1)
				return "reply", nil
			}), discordFunc(func(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error) {
				replies.Add(1)
				return nil, nil
			}), time.Minute, 1)
			b.Handle(event)
			b.Stop()
			if routes.Load() != 0 || replies.Load() != 0 {
				t.Fatalf("ignored message caused %d routes and %d replies", routes.Load(), replies.Load())
			}
		})
	}
}

func TestHandleReplies(t *testing.T) {
	for _, tt := range []struct {
		name, content, response, want string
		err                           error
		routes                        int32
	}{
		{"request", "prefix <@!123> \thelp\n", "Hello @Everyone <@999>", "hello @everyone <@999>", nil, 1},
		{"empty", "<@123>", "unused", "write a request after my mention. try `@potatobot help`.", nil, 0},
		{"whitespace", "<@!123> \n\t", "unused", "write a request after my mention. try `@potatobot help`.", nil, 0},
		{"failure", "<@123> help", "unsafe partial reply", "i couldn't process that request right now. please try again shortly.", errors.New("secret upstream failure"), 1},
		{"empty response", "<@123> help", "", "", nil, 1},
		{"blank response", "<@123> help", " \n\t", "", nil, 1},
		{"long ASCII", "<@123> help", strings.Repeat("a", 2001), strings.Repeat("a", 1997) + "...", nil, 1},
		{"long UTF16", "<@123> help", strings.Repeat("\U0001f954", 1001), strings.Repeat("\U0001f954", 998) + "...", nil, 1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var routes atomic.Int32
			replies := make(chan *discordgo.MessageSend, 2)
			type contextKey struct{}
			b, _ := testBot(t, routerFunc(func(ctx context.Context, req command.Request) (string, error) {
				routes.Add(1)
				want := command.Request{Input: "help", AuthorID: "author", GuildID: "guild", ChannelID: "channel", MessageID: "message", BotID: "123"}
				if !reflect.DeepEqual(req, want) {
					t.Errorf("router request = %+v, want %+v", req, want)
				}
				if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > time.Minute || ctx.Err() != nil {
					t.Errorf("router context has invalid deadline or error: %v, %v, %v", deadline, ok, ctx.Err())
				}
				if ctx.Value(contextKey{}) != "root value" {
					t.Error("router context lost root value")
				}
				return tt.response, tt.err
			}), discordFunc(func(channel string, msg *discordgo.MessageSend, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
				if channel != "channel" {
					t.Errorf("reply channel = %q", channel)
				}
				cfg := &discordgo.RequestConfig{Request: &http.Request{}}
				for _, opt := range opts {
					opt(cfg)
				}
				ctx := cfg.Request.Context()
				if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 10*time.Second || ctx.Err() != nil {
					t.Errorf("reply context has invalid deadline or error: %v, %v, %v", deadline, ok, ctx.Err())
				}
				if ctx.Value(contextKey{}) != "root value" {
					t.Error("reply context lost root value")
				}
				replies <- msg
				return nil, nil
			}), time.Minute, 1)
			b.ctx = context.WithValue(b.ctx, contextKey{}, "root value")
			b.Handle(testMessage(tt.content))
			b.Stop()
			if routes.Load() != tt.routes {
				t.Errorf("router calls = %d, want %d", routes.Load(), tt.routes)
			}
			if tt.want == "" {
				if len(replies) != 0 {
					t.Fatal("blank response sent a reply")
				}
				return
			}
			if len(replies) != 1 {
				t.Fatalf("reply count = %d, want 1", len(replies))
			}
			msg := <-replies
			if msg.Content != tt.want {
				t.Errorf("reply content = %q, want %q", msg.Content, tt.want)
			}
			ref := msg.Reference
			if ref == nil || ref.MessageID != "message" || ref.ChannelID != "channel" || ref.GuildID != "guild" || ref.FailIfNotExists == nil || *ref.FailIfNotExists {
				t.Errorf("reply reference = %+v", ref)
			}
			mentions := msg.AllowedMentions
			if mentions == nil || mentions.Parse == nil || len(mentions.Parse) != 0 || len(mentions.Users) != 0 || len(mentions.Roles) != 0 || mentions.RepliedUser {
				t.Errorf("mentions not explicitly suppressed: %+v", mentions)
			}
		})
	}
}

func TestRequestDeadlineAllowsSafeReply(t *testing.T) {
	replied := make(chan error, 1)
	b, _ := testBot(t, routerFunc(func(ctx context.Context, _ command.Request) (string, error) {
		if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			t.Errorf("expired router context error = %v", ctx.Err())
		}
		return "", ctx.Err()
	}), discordFunc(func(_ string, msg *discordgo.MessageSend, opts ...discordgo.RequestOption) (*discordgo.Message, error) {
		cfg := &discordgo.RequestConfig{Request: &http.Request{}}
		for _, opt := range opts {
			opt(cfg)
		}
		if msg.Content != "i couldn't process that request right now. please try again shortly." {
			t.Errorf("timeout reply = %q", msg.Content)
		}
		replied <- cfg.Request.Context().Err()
		return nil, errors.New("Discord unavailable")
	}), 0, 1)
	b.Handle(testMessage("<@123> help"))
	b.Stop()
	if err := receive(t, replied); err != nil {
		t.Fatalf("reply inherited expired routing context: %v", err)
	}
}

func TestCancellationAndStop(t *testing.T) {
	started, canceled, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var routes, replies atomic.Int32
	b, cancel := testBot(t, routerFunc(func(ctx context.Context, _ command.Request) (string, error) {
		routes.Add(1)
		close(started)
		<-ctx.Done()
		if !errors.Is(ctx.Err(), context.Canceled) {
			t.Errorf("router context error = %v, want cancellation", ctx.Err())
		}
		close(canceled)
		<-release
		return "must not reply after cancellation", nil
	}), discordFunc(func(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error) {
		replies.Add(1)
		return nil, nil
	}), time.Minute, 1)
	t.Cleanup(func() { close(release) })
	b.Handle(testMessage("<@123> help"))
	receive(t, started)
	cancel()
	stopped := make(chan struct{})
	go func() {
		b.Stop()
		close(stopped)
	}()
	receive(t, canceled)
	select {
	case <-stopped:
		t.Fatal("Stop returned while router was still active")
	default:
	}
	release <- struct{}{}
	receive(t, stopped)
	b.Handle(testMessage("<@123> more"))
	b.Stop()
	if routes.Load() != 1 || replies.Load() != 0 {
		t.Fatalf("routes = %d, replies = %d after cancellation", routes.Load(), replies.Load())
	}
}

func TestHandleAfterStopOrCancellation(t *testing.T) {
	for _, state := range []string{"stopped", "canceled"} {
		t.Run(state, func(t *testing.T) {
			var routes, replies atomic.Int32
			b, cancel := testBot(t, routerFunc(func(context.Context, command.Request) (string, error) {
				routes.Add(1)
				return "reply", nil
			}), discordFunc(func(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error) {
				replies.Add(1)
				return nil, nil
			}), time.Minute, 1)
			if state == "stopped" {
				b.Stop()
			} else {
				cancel()
			}
			b.Handle(testMessage("<@123> help"))
			b.Stop()
			if routes.Load() != 0 || replies.Load() != 0 {
				t.Fatalf("%s bot caused %d routes and %d replies", state, routes.Load(), replies.Load())
			}
		})
	}
}

func TestBoundedConcurrency(t *testing.T) {
	const limit = 2
	started, release := make(chan struct{}, limit+1), make(chan struct{})
	var routes, replies, active, peak atomic.Int32
	b, _ := testBot(t, routerFunc(func(ctx context.Context, _ command.Request) (string, error) {
		routes.Add(1)
		n := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); n > old; old = peak.Load() {
			if peak.CompareAndSwap(old, n) {
				break
			}
		}
		started <- struct{}{}
		select {
		case <-release:
		case <-ctx.Done():
		}
		return "reply", nil
	}), discordFunc(func(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error) {
		replies.Add(1)
		return nil, nil
	}), time.Minute, limit)
	t.Cleanup(func() { close(release) })
	for range limit {
		b.Handle(testMessage("<@123> help"))
	}
	for range limit {
		receive(t, started)
	}
	// Handle must not wait for a slot, and excess messages must not be queued.
	returned := make(chan struct{})
	go func() {
		b.Handle(testMessage("<@123> excess"))
		close(returned)
	}()
	receive(t, returned)
	for range limit {
		release <- struct{}{}
	}
	b.wg.Wait()
	if routes.Load() != limit || replies.Load() != limit || peak.Load() != limit || len(b.slots) != 0 {
		t.Fatalf("routes=%d replies=%d peak=%d slots=%d, want %d completed requests", routes.Load(), replies.Load(), peak.Load(), len(b.slots), limit)
	}
	// Completion frees capacity for a subsequent request.
	b.Handle(testMessage("<@123> again"))
	receive(t, started)
	release <- struct{}{}
	b.Stop()
	if routes.Load() != limit+1 || replies.Load() != limit+1 {
		t.Fatalf("capacity not reusable: routes=%d replies=%d", routes.Load(), replies.Load())
	}
}

func TestLimitReplyUTF16(t *testing.T) {
	for _, tt := range []struct {
		name, input, want string
	}{
		{"empty", "", ""},
		{"short", "hello", "hello"},
		{"1997 ASCII", strings.Repeat("a", 1997), strings.Repeat("a", 1997)},
		{"1998 ASCII", strings.Repeat("a", 1998), strings.Repeat("a", 1998)},
		{"1999 ASCII", strings.Repeat("a", 1999), strings.Repeat("a", 1999)},
		{"2000 ASCII", strings.Repeat("a", 2000), strings.Repeat("a", 2000)},
		{"2001 ASCII", strings.Repeat("a", 2001), strings.Repeat("a", 1997) + "..."},
		{"BMP", strings.Repeat("\u754c", 2000), strings.Repeat("\u754c", 2000)},
		{"2000 astral units", strings.Repeat("\U0001f954", 1000), strings.Repeat("\U0001f954", 1000)},
		{"astral truncation", strings.Repeat("\U0001f954", 1001), strings.Repeat("\U0001f954", 998) + "..."},
		{"mixed boundary", strings.Repeat("a", 1996) + "\U0001f954" + "abc", strings.Repeat("a", 1996) + "..."},
		{"exact truncation budget", strings.Repeat("a", 1995) + "\U0001f954" + "abcd", strings.Repeat("a", 1995) + "\U0001f954" + "..."},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := limitReply(tt.input)
			if !utf8.ValidString(got) || len(utf16.Encode([]rune(got))) > 2000 {
				t.Fatal("reply is invalid UTF-8 or exceeds 2000 UTF-16 units")
			}
			if got != tt.want {
				t.Errorf("reply changed incorrectly: got %d UTF-16 units, want %d (ellipsis=%v)", len(utf16.Encode([]rune(got))), len(utf16.Encode([]rune(tt.want))), strings.HasSuffix(got, "..."))
			}
		})
	}
}
