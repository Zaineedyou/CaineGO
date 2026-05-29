# CaineGO 🤖

A Discord bot powered by Groq AI, built in Go. Fast, lightweight, and easy to self-host.

## Features

- 💬 **AI Chat** — context-aware conversations using Groq (supports multiple models)
- 👁️ **Vision** — analyze images sent in chat
- 🛡️ **Moderation** — kick, ban, warn, timeout, mute, clear, lock, and more
- ⭐ **XP & Leveling** — automatic XP gain with level-up notifications
- 💤 **AFK System** — set AFK status, auto-unafk on message
- 👋 **Welcome/Goodbye** — customizable join and leave messages
- 🎭 **Auto-role** — automatically assign role on join
- 🚫 **Automod** — banned words filter with auto-delete
- 📊 **Dashboard** — admin settings via slash command buttons
- 🧠 **Multi-model** — switch AI model per server (Llama 70B, GPT OSS 120B/20B, Qwen 32B)

## Requirements

- Go 1.21+
- A Discord bot token → [Discord Developer Portal](https://discord.com/developers/applications)
- A Groq API key → [console.groq.com](https://console.groq.com)

## Setup

```bash
# Clone the repo
git clone https://github.com/Zaineedyou/CaineGO.git
cd CaineGO

# Copy and fill in your credentials
cp .env.example .env
nano .env

# Download dependencies
go mod tidy

# Build and run
go build -o caine .
./caine
```

## Environment Variables

```env
DISCORD_TOKEN=your_discord_bot_token
GROQ_API_KEY=your_groq_api_key
BOT_PREFIX=Caine          # optional, default: Caine
SYSTEM_PROMPT=            # optional, custom AI persona
```

## Usage

Mention the bot or use the prefix:

```
Caine <question>
Caine help
Caine kick @user reason
Caine setmodel llama70b
/dashboard
/info
```

**Available models:**
| Alias | Model |
|-------|-------|
| `llama70b` | llama-3.3-70b-versatile *(default)* |
| `gpt120b` | openai/gpt-oss-120b |
| `gpt20b` | openai/gpt-oss-20b |
| `qwen32b` | qwen/qwen3-32b |

## Running in Background

```bash
nohup ./caine > caine.log 2>&1 &
tail -f caine.log   # check logs
pkill caine         # stop bot
```

## Built With

- [discordgo](https://github.com/bwmarrin/discordgo)
- [Groq API](https://groq.com)
- [modernc/sqlite](https://gitlab.com/cznic/sqlite) — pure Go SQLite, no CGO needed

## License

MIT
