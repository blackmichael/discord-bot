# PotatoBot

A small Go Discord bot that uses [Jev](https://typesafe.ai) through
[`haileyok/typesafe-client`](https://github.com/haileyok/typesafe-client) to
classify natural-language requests and select a Go command handler.

Mention the bot with a request, such as `@PotatoBot what can you do?`.
Commands include `help` and a snarky potato check; unsupported or uncertain
routing requests get a fallback response. Jev chooses logic to run, not a
generated chat answer.
Exact `help` (case-insensitive, with surrounding whitespace ignored) runs
locally without a TypeSafe request. Natural-language help requests still use Jev.

## Potato Check

Ask a question after mentioning the bot:

```text
@PotatoBot is a russet a potato?
@PotatoBot does a sweet potato count as a potato?
@PotatoBot is my laptop a potato?
@PotatoBot are you a potato?
```

Jev first routes the request to the `potato` command, then evaluates a typed
yes/no (`Noul`) question about the subject. Both calls share the bot request
deadline. Ordinary potatoes and foods primarily made from them count; sweet
potatoes, yams, and figurative "potatoes" do not. Describe the subject in text:
images and earlier messages are not evaluated.

**PotatoBot itself is most certainly a potato.** Direct questions such as
`are you a potato?`, `is PotatoBot a potato?`, and `is this bot really a potato?`
get a local, snarky **100%** answer without any TypeSafe calls. This is a known
persona fact, not a model estimate. Jev's instructions also include the fact
for other phrasings. Mentioning PotatoBot while asking about a laptop or another
bot does not make that other subject a potato.

Replies are randomly selected from three snarky answers in each probability
band and include the model's potato probability:

| Potato Probability | Tone |
| --- | --- |
| Below 5% | Almost certainly not a potato |
| 5% to below 25% | Probably not a potato |
| 25% to below 40% | Leaning no |
| 40% to below 60% | Uncertain; ask for a better description |
| 60% to below 75% | Leaning yes |
| 75% to below 95% | Probably a potato |
| 95% and above | Almost certainly a potato |

For example: "No. The potato community has declined its application."

The yes/no score is the probability of **potato**, not a separate confidence
score. A value near 50% is uncertain; a value near 0% is a confident negative.
This is separate from the routing confidence controlled by `BOT_MIN_CONFIDENCE`.

## Discord Setup

1. Create an application in the [Discord Developer Portal](https://discord.com/developers/applications) and configure its bot as **PotatoBot**.
2. Obtain its **bot token** from the Bot settings. Never use a user token.
3. Use the OAuth2 URL Generator with the `bot` scope to invite it to your server. Grant **View Channels**, **Send Messages**, and **Read Message History**. For threads, also grant **Send Messages in Threads**. Administrator permission is not needed.
4. Obtain a TypeSafe API key from the [TypeSafe console](https://console.typesafe.ai/keys).

The bot requests only guild-message and direct-message gateway intents.
Discord exposes content for messages that mention the bot, so the privileged
**Message Content Intent is not required**. DMs also require a mention to activate
the bot. Select the actual bot from Discord's mention picker; plain text that
just spells `@PotatoBot` is not a mention.

## Run With Docker

Create your local configuration from the template:

```sh
cp .env.example .env
```

Set `DISCORD_TOKEN` and `TYPESAFE_API_KEY` in `.env`, then run:

```sh
docker compose up --build -d
docker compose logs -f potatobot
```

Stop it with `docker compose down`. No published ports or persistent volumes
are needed. The container runs as a non-root user with a read-only filesystem;
secrets are provided at runtime and excluded from the build context.

Without Compose:

```sh
docker build -t potatobot .
docker run --rm --init --env-file .env potatobot
```

## Configuration

| Environment Variable | Default | Purpose |
| --- | --- | --- |
| `DISCORD_TOKEN` | Required | Discord bot token, without the `Bot ` prefix |
| `TYPESAFE_API_KEY` | Required | TypeSafe API key |
| `TYPESAFE_BASE_URL` | `https://api.typesafe.ai` | API root or compatible gateway |
| `TYPESAFE_DEFAULT_MODEL` | `jev-latest` | Jev model; pin a version after tuning routing |
| `TYPESAFE_LOG_LEVEL` | `off` | Client JSON logging to stdout: `debug`, `info`, `warn`, `error`, or `off`; independent of `LOG_LEVEL` |
| `BOT_REQUEST_TIMEOUT` | `30s` | Deadline for classification and command handling, as a Go duration |
| `BOT_MAX_CONCURRENT` | `4` | Maximum in-flight requests, including replies |
| `BOT_MIN_CONFIDENCE` | `0.7` | Minimum classification confidence, from `0` to `1`, to dispatch a handler |
| `LOG_LEVEL` | `info` | Bot JSON logging: `debug`, `info`, `warn`, or `error`; info/debug include classification input and questions |

The TypeSafe library handles transient-error retries, with its default 30-second
total retry budget and 10-second per-attempt timeout. `BOT_REQUEST_TIMEOUT` can
further shorten the request deadline. Reply delivery has its own 10-second
HTTP deadline so a failed classification can still produce an error message.
DiscordGo's internal rate-limit waits can outlast that deadline; a reply may
therefore take longer during rate limiting.

When all slots are occupied, new requests are dropped and logged rather than
queued. SIGINT and SIGTERM cancel active requests and close the Discord session.
The process exits after at most 15 seconds of graceful shutdown if a handler or
DiscordGo is stuck. Gateway startup is bounded to 30 seconds. Command handlers
must honor their context to shut down promptly.

## Logging

Bot and TypeSafe client logs are newline-delimited JSON on stdout. Set
`TYPESAFE_LOG_LEVEL=info` to enable SDK response/retry logs, or `debug` to also
include HTTP request/response bodies and headers. SDK records have
`"component":"typesafe"`; the SDK redacts credential headers.

At the default `LOG_LEVEL=info`, the bot logs a `classifying request` or
`evaluating potato` record **before each TypeSafe call**, including:

- `state`: the full trimmed text after the bot mention, exactly as sent to Jev.
- `questions`: structured question instructions and command-option descriptions, exactly as sent to Jev.
- `context`: Discord author, guild, channel, and message IDs for correlation. These IDs are available to handlers but are not sent to Jev.

The `request classified` record includes the same Discord context, selected
command, confidence, model, and TypeSafe request ID. Local `help` calls instead
log `request dispatched` with `source=local`, the input, and Discord context;
no model questions are evaluated.
Recognized direct PotatoBot self-questions likewise log local dispatch and a
`potato evaluated` record with `source=known_fact` and probability `1`, without
a model or TypeSafe request ID.
The `potato evaluated` record includes the same Discord context, potato
probability, response band, model, and TypeSafe request ID.

**These logs contain users' input and Discord IDs.** Restrict access and choose
appropriate retention. `LOG_LEVEL=warn` or `error` suppresses request-payload
logs. `TYPESAFE_LOG_LEVEL=off` only disables SDK logs, not the bot's payload
logs. Tokens are not included in the bot's request records, but text users
provide is logged verbatim, so secrets pasted into a request will be logged.

Except for local `help` and recognized direct PotatoBot self-questions, the text
after a mention is sent to TypeSafe. The bot
does not send or log prior conversation history or attachments as context.

## Run Locally

Use Go 1.27 or newer. The binary reads process environment variables, not `.env`
files itself. For example:

```sh
export DISCORD_TOKEN='your-bot-token'
export TYPESAFE_API_KEY='your-typesafe-key'
go run ./cmd/potatobot
```

## Add Commands Later

Register commands in `internal/command/commands.go`. Each `Command` has a unique
`Name`, a `Description` telling Jev which requests match, and a `Handle` function:

```go
Command{
    Name:        "example",
    Description: "Describe the requests this command handles and what it does not handle.",
    Handle: func(ctx context.Context, req Request) (string, error) {
        // Implement logic using req.Input and the Discord IDs in req.
        return "Example response", nil
    },
}
```

Except for local exact-`help` and recognized direct self-questions, the router
builds a typed Jev `Choice` question from the registered commands and a reserved
`unknown` option.
It only dispatches a registered handler when the
answer meets the confidence threshold. Handlers receive the full trimmed text
after the first bot mention, along with the author, guild, channel, and message
IDs. Text before that mention is ignored. Update the help handler as commands
are added.

Classification is **not authorization**. Future privileged or destructive
commands must validate permissions and, where appropriate, ask for confirmation.
Replies suppress user, role, and everyone pings and are truncated safely to
Discord's 2,000-character limit, measured conservatively in UTF-16 units.

## Development

```sh
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/potatobot
```

Race tests require CGO and a C compiler. Tests use fake Discord transports and
a local HTTP server for the real TypeSafe client; they do not require tokens
or make paid API calls. The Docker build runs the tests before producing the
static binary.

| Path | Responsibility |
| --- | --- |
| `cmd/potatobot` | Startup, Discord connection, shutdown |
| `internal/config` | Environment loading and validation |
| `internal/bot` | Mention activation, bounded request handling, safe replies |
| `internal/command` | Jev classification and command registration/dispatch |
