package command

import (
	"context"
	"log/slog"
)

// DefaultCommands is the registration point for future commands and their Jev descriptions.
func DefaultCommands(client Client, logger *slog.Logger) []Command {
	return []Command{
		{
			Name:        "help",
			Description: "The user asks what PotatoBot can do, requests help, or asks how to use the bot or list its commands.",
			Handle: func(_ context.Context, _ Request) (string, error) {
				return "Tag me and write your request after the mention.\n\n" +
					"`help` - Show this help without calling Jev.\n" +
					"Potato check - Ask whether something is a potato, for example `@PotatoBot is a russet a potato?`. I'll judge it, with entirely unnecessary attitude.\n\n" +
					"Describe the subject in text; I don't inspect images or earlier messages.", nil
			},
		},
		potatoCommand(client, logger),
	}
}
