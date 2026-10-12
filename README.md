# CaineGO

Go-based Discord bot with Groq AI, moderation, leveling, server configuration, and an optional Minecraft bridge.

## Features

- AI chat with conversation history and image analysis
- Groq model selection and per-server persona settings
- Moderation, warnings, automod, welcome/goodbye messages, XP, and AFK commands
- PostgreSQL storage
- Optional Minecraft WebSocket bridge
- Docker Compose deployment

## Requirements

- Go 1.25+ for local builds
- Discord bot token
- Groq API key
- PostgreSQL for local runs; Docker Compose provides PostgreSQL 16

## Run with Docker Compose

```bash
cp .env.example .env
```

Set `DISCORD_TOKEN`, `GROQ_API_KEY`, and a non-default `POSTGRES_PASSWORD` in `.env`, then run:

```bash
docker compose up --build -d
docker compose logs -f caine
```

Compose starts PostgreSQL, waits for its health check, and configures the bot's database URL. The optional bridge port is published using `BRIDGE_PORT`.

## Run locally

Set `DISCORD_TOKEN`, `GROQ_API_KEY`, and a reachable PostgreSQL `DATABASE_URL` in `.env`, then run:

```bash
go mod download
go test ./...
go vet ./...
go run .
```

## Configuration

See `.env.example` for all supported variables.

| Variable | Purpose |
| --- | --- |
| `DISCORD_TOKEN` | Discord bot token |
| `GROQ_API_KEY` | Groq API key |
| `DATABASE_URL` | PostgreSQL connection URL |
| `POSTGRES_USER` | PostgreSQL user for Compose |
| `POSTGRES_PASSWORD` | PostgreSQL password for Compose |
| `POSTGRES_DB` | PostgreSQL database for Compose |
| `BOT_PREFIX` | Text-command prefix; default `Caine` |
| `SYSTEM_PROMPT` | Default AI persona |
| `BOT_OWNER_ID` | Bot owner Discord user ID |
| `MAX_HISTORY` | Default conversation history limit |
| `AI_RATE_MAX` | AI requests per user per rate-limit window |
| `AI_RATE_WINDOW_SEC` | AI rate-limit window in seconds |
| `BRIDGE_TOKEN` | Bearer token for the Minecraft bridge |
| `BRIDGE_PORT` | Minecraft bridge port; default `8080` |

## Commands

AI and utility commands use the configured prefix or a bot mention:

```text
Caine <question>
Caine summarize [n]
Caine reset
Caine setmodel <alias>
Caine setpersona <prompt>
Caine help
```

Available model aliases: `llama70b`, `gpt120b`, `gpt20b`, `qwen32b`.

Other commands include:

- **Moderation:** `kick`, `ban`, `unban`, `timeout`, `untimeout`, `warn`, `warnings`, `clearwarn`, `clear`, `lock`, `unlock`, `slowmode`, `nick`, `role`, `report`
- **Automod and server setup:** `addword`, `removeword`, `words`, `enable`, `disable`, `setlog`, `setwelcome`, `setgoodbye`, `setwelcomemsg`, `setgoodbyemsg`, `autorole`, `removeautorole`, `setlevelchannel`, `sethistory`
- **Community:** `rank`, `leaderboard`, `afk`, `afklist`, `status`
- **Minecraft bridge:** `setbridge`, `bridgestatus`
- **Slash commands:** `/info`, `/help`, `/dashboard`, `/healthcheck`

## Chat history and storage

PostgreSQL stores conversation history and bot feature data. In guild channels, AI history is shared across members of that channel; `Caine reset` clears the channel's shared history. In direct messages, history is scoped to the user.

Tables are initialized when the bot starts. Back up the PostgreSQL database separately.

## Checks

```bash
go test ./...
go vet ./...
go build ./...
```

GitHub Actions runs these checks for pushes and pull requests targeting `main`.

## License

MIT. See [LICENSE](LICENSE).
