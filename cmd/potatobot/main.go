package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"potatobot/internal/bot"
	"potatobot/internal/command"
	"potatobot/internal/config"

	"github.com/bwmarrin/discordgo"
	typesafe "github.com/haileyok/typesafe-client/go"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx); err != nil {
		slog.Error("bot stopped", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(logger)
	client, err := typesafe.NewClient(typesafe.WithLogger(typeSafeLogger(os.Stdout, cfg.TypeSafeLogLevel)))
	if err != nil {
		return fmt.Errorf("configure TypeSafe: %w", err)
	}
	router, err := command.NewRouter(client, command.DefaultCommands(client, logger), cfg.MinConfidence, logger)
	if err != nil {
		return err
	}
	session, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		return fmt.Errorf("configure Discord: %w", err)
	}
	// Mentioned messages expose content without the privileged Message Content intent.
	session.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages
	session.SyncEvents = true
	session.Client.Timeout = 10 * time.Second
	user, err := session.User("@me", discordgo.WithContext(ctx), discordgo.WithRetryOnRatelimit(false), discordgo.WithRestRetries(0))
	if err != nil {
		return fmt.Errorf("authenticate Discord bot: %w", err)
	}
	b := bot.New(ctx, user.ID, session, router, logger, cfg.RequestTimeout, cfg.MaxConcurrent)
	session.AddHandler(func(_ *discordgo.Session, event *discordgo.MessageCreate) { b.Handle(event) })
	// DiscordGo's initial gateway reads have no deadline; don't let them stall the process.
	opened := make(chan error, 1)
	go func() { opened <- session.Open() }()
	select {
	case err := <-opened:
		if err != nil {
			return fmt.Errorf("connect Discord gateway: %w", err)
		}
	case <-ctx.Done():
		return nil
	case <-time.After(30 * time.Second):
		return fmt.Errorf("Discord gateway startup exceeded 30s")
	}
	logger.Info("PotatoBot ready", "bot_id", user.ID, "username", user.Username)
	<-ctx.Done()
	logger.Info("shutting down")
	// DiscordGo's rate-limit waits can outlive request contexts. Bound graceful shutdown.
	stopped := make(chan error, 1)
	go func() {
		b.Stop()
		stopped <- session.Close()
	}()
	select {
	case err := <-stopped:
		if err != nil {
			return fmt.Errorf("close Discord gateway: %w", err)
		}
	case <-time.After(15 * time.Second):
		return fmt.Errorf("graceful shutdown exceeded 15s")
	}
	return nil
}

func typeSafeLogger(output io.Writer, level *slog.Level) *slog.Logger {
	options := &slog.HandlerOptions{}
	if level == nil {
		output = io.Discard
	} else {
		options.Level = *level
	}
	return slog.New(slog.NewJSONHandler(output, options)).With("component", "typesafe")
}
