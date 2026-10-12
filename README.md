# CaineGO

[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![CI](https://github.com/Zaineedyou/CaineGO/actions/workflows/ci.yml/badge.svg)](https://github.com/Zaineedyou/CaineGO/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

CaineGO is a self-hosted Discord bot written in Go. It combines Groq-powered chat, server moderation, community features, and an optional Minecraft WebSocket bridge.

## Features

- **AI chat:** Groq model selection, image analysis, conversation summaries, and configurable personas
- **Moderation:** warnings, kick/ban, timeouts, channel controls, and moderation reports
- **Server utilities:** XP and rankings, AFK status, welcome/goodbye messages, auto-roles, and word filtering
- **Minecraft bridge:** optional token-authenticated WebSocket connection
- **Persistence:** PostgreSQL-backed conversation history and server data
- **Deployment:** Docker Compose configuration with PostgreSQL health checks and persistent storage

## Requirements

- Go 1.25+ for local builds
- Discord bot token
- Groq API key
- PostgreSQL 16+ for local runs; Docker Compose provides PostgreSQL 16

## Quick start: Docker Compose

```bash
git clone https://github.com/Zaineedyou/CaineGO.git
cd CaineGO
cp .env.example .env
```

Set `DISCORD_TOKEN`, `GROQ_API_KEY`, and a strong `POSTGRES_PASSWORD` in `.env`, then start the services:

```bash
docker compose up --build -d
docker compose ps
docker compose logs -f caine
```

Compose starts PostgreSQL, waits for its health check, and configures the bot's database connection. Database files persist in the `postgres_data` volume.

## Local development

Start a PostgreSQL instance and set `DISCORD_TOKEN`, `GROQ_API_KEY`, and `DATABASE_URL` in `.env`. Then run:

```bash
go mod download
go test ./...
go vet ./...
go run .
```

To build a binary:

```bash
go build -o caine .
```

## Configuration

`.env.example` lists the supported environment variables.

| Variable | Required | Description |
| --- | --- | --- |
| `DISCORD_TOKEN` | Yes | Discord bot token |
| `GROQ_API_KEY` | Yes | Groq API key |
| `DATABASE_URL` | Yes for local runs | PostgreSQL connection URL; Compose sets the container URL |
| `POSTGRES_USER` | Compose | PostgreSQL user |
| `POSTGRES_PASSWORD` | Compose | PostgreSQL password |
| `POSTGRES_DB` | Compose | PostgreSQL database name |
| `BOT_PREFIX` | No | Text-command prefix; default `Caine` |
| `SYSTEM_PROMPT` | No | Default AI persona |
| `BOT_OWNER_ID` | No | Discord user ID for owner-only commands |
| `MAX_HISTORY` | No | Default conversation history limit; default `30` |
| `AI_RATE_MAX` | No | AI requests per user in one rate-limit window; default `10` |
| `AI_RATE_WINDOW_SEC` | No | Rate-limit window in seconds; default `60` |
| `BRIDGE_TOKEN` | No | Bearer token for Minecraft bridge connections |
| `BRIDGE_PORT` | No | Minecraft bridge port; default `8080` |

Use a unique database password for deployments. For remote PostgreSQL, configure TLS in the connection URL. Keep `.env` out of source control.

## AI models

| Alias | Groq model |
| --- | --- |
| `llama70b` | `llama-3.3-70b-versatile` |
| `gpt120b` | `openai/gpt-oss-120b` |
| `gpt20b` | `openai/gpt-oss-20b` |
| `qwen32b` | `qwen/qwen3-32b` |

## Commands

AI commands use the configured text prefix or a bot mention:

```text
Caine <question>
Caine summarize [count]
Caine reset
Caine setmodel <alias>
Caine setpersona <prompt>
Caine help
```

Other text commands cover:

- **Moderation:** `kick`, `ban`, `unban`, `timeout`, `untimeout`, `warn`, `warnings`, `clearwarn`, `clear`, `lock`, `unlock`, `slowmode`, `nick`, `role`, `report`
- **Automod and configuration:** `addword`, `removeword`, `words`, `enable`, `disable`, `setlog`, `setwelcome`, `setgoodbye`, `setwelcomemsg`, `setgoodbyemsg`, `autorole`, `removeautorole`, `setlevelchannel`, `sethistory`
- **Community:** `rank`, `leaderboard`, `afk`, `afklist`, `status`
- **Minecraft bridge:** `setbridge`, `bridgestatus`

Slash commands: `/info`, `/help`, `/dashboard`, `/healthcheck`.

## Chat history and data

PostgreSQL stores conversation history and feature data. In guild channels, AI history is shared across members of the channel; `Caine reset` clears that channel's shared history. Direct-message history is scoped to the user. Database tables are initialized at startup.

## Minecraft bridge

Set `BRIDGE_TOKEN` and `BRIDGE_PORT` in `.env`, then configure the bot with `Caine setbridge #channel`. The bridge accepts authenticated WebSocket connections on the configured port. Docker Compose publishes this port.

## Development checks

```bash
go test ./...
go vet ./...
go build ./...
```

GitHub Actions runs these checks for pushes and pull requests targeting `main`.

## License

MIT. See [LICENSE](LICENSE).
