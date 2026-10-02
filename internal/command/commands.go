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
				return "tag me and write your request after the mention.\n\n" +
					"`help` - show this help without calling jev.\n" +
					"`potato` - ask whether something is a potato, for example `@potatobot is a russet a potato?`. i'll judge it, with entirely unnecessary attitude.\n" +
					"`opinion` - say how strongly i agree or disagree with an opinion, or reply to a post and tag me.\n" +
					"`/hot take` - rate how spicy a take is without saying whether i agree.\n\n" +
					"describe the subject in text; i don't inspect images or attachments.", nil
			},
		},
		potatoCommand(client, logger),
		opinionCommand(client, logger),
		hotTakeCommand(client, logger),
	}
}
