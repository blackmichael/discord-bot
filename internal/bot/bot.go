package bot

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf16"

	"potatobot/internal/command"

	"github.com/bwmarrin/discordgo"
)

type Router interface {
	Route(context.Context, command.Request) (string, error)
}

type Discord interface {
	ChannelMessageSendComplex(string, *discordgo.MessageSend, ...discordgo.RequestOption) (*discordgo.Message, error)
}

type Bot struct {
	ctx     context.Context
	id      string
	discord Discord
	router  Router
	logger  *slog.Logger
	timeout time.Duration
	slots   chan struct{}
	mu      sync.Mutex
	stopped bool
	wg      sync.WaitGroup
}

func New(ctx context.Context, id string, discord Discord, router Router, logger *slog.Logger, timeout time.Duration, maxConcurrent int) *Bot {
	return &Bot{ctx: ctx, id: id, discord: discord, router: router, logger: logger, timeout: timeout, slots: make(chan struct{}, maxConcurrent)}
}

// Handle must return quickly because Discord gateway events are dispatched synchronously.
func (b *Bot) Handle(event *discordgo.MessageCreate) {
	if event == nil || event.Message == nil || event.Author == nil || event.Author.Bot || event.WebhookID != "" || event.Author.ID == b.id {
		return
	}
	input, mentioned := MentionInput(event.Content, b.id)
	if !mentioned {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopped || b.ctx.Err() != nil {
		return
	}
	select {
	case b.slots <- struct{}{}:
	default:
		b.logger.Warn("request dropped: bot at concurrency limit", "message_id", event.ID)
		return
	}
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		defer func() { <-b.slots }()
		b.respond(event.Message, input)
	}()
}

// Stop prevents new work and waits for active requests. Cancel the root context first.
func (b *Bot) Stop() {
	b.mu.Lock()
	b.stopped = true
	b.mu.Unlock()
	b.wg.Wait()
}

func (b *Bot) respond(msg *discordgo.Message, input string) {
	ctx, cancel := context.WithTimeout(b.ctx, b.timeout)
	defer cancel()
	response := "Write a request after my mention. Try `@PotatoBot help`."
	if input != "" {
		var err error
		req := command.Request{
			Input: input, AuthorID: msg.Author.ID, GuildID: msg.GuildID, ChannelID: msg.ChannelID, MessageID: msg.ID, BotID: b.id,
		}
		for _, user := range msg.Mentions {
			if user == nil || user.ID == "" || slices.Contains(req.MentionedUserIDs, user.ID) {
				continue
			}
			if strings.Contains(input, "<@"+user.ID+">") || strings.Contains(input, "<@!"+user.ID+">") {
				req.MentionedUserIDs = append(req.MentionedUserIDs, user.ID)
			}
		}
		response, err = b.router.Route(ctx, req)
		if err != nil {
			b.logger.Error("request failed", "message_id", msg.ID, "error", err)
			response = "I couldn't process that request right now. Please try again shortly."
		}
	}
	if b.ctx.Err() != nil || strings.TrimSpace(response) == "" {
		return
	}
	// Allow a short error reply even if classification used the whole request deadline.
	replyCtx, replyCancel := context.WithTimeout(b.ctx, 10*time.Second)
	defer replyCancel()
	_, err := b.discord.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
		Content:         limitReply(response),
		Reference:       &discordgo.MessageReference{MessageID: msg.ID, ChannelID: msg.ChannelID, GuildID: msg.GuildID, FailIfNotExists: new(false)},
		AllowedMentions: &discordgo.MessageAllowedMentions{Parse: []discordgo.AllowedMentionType{}, RepliedUser: false},
	}, discordgo.WithContext(replyCtx))
	if err != nil {
		b.logger.Error("reply failed", "message_id", msg.ID, "error", err)
	}
}

// MentionInput uses the authenticated bot ID, not its display name. Both Discord mention forms are valid.
func MentionInput(content, id string) (string, bool) {
	if id == "" {
		return "", false
	}
	start, length := -1, 0
	for _, mention := range []string{"<@" + id + ">", "<@!" + id + ">"} {
		if index := strings.Index(content, mention); index >= 0 && (start < 0 || index < start) {
			start, length = index, len(mention)
		}
	}
	if start < 0 {
		return "", false
	}
	return strings.TrimSpace(content[start+length:]), true
}

func limitReply(text string) string {
	units := 0
	cutoff := len(text)
	for index, r := range text {
		units += utf16.RuneLen(r)
		if units > 1997 && cutoff == len(text) {
			cutoff = index
		}
		if units > 2000 {
			return text[:cutoff] + "..."
		}
	}
	return text
}
