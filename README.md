# CaineGO 🤖

A Discord bot powered by Groq AI — AI chat, moderation, leveling, and more. Built in Go.

![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)
![Discord](https://img.shields.io/badge/Discord-Bot-5865F2?style=flat&logo=discord)
![License](https://img.shields.io/badge/License-MIT-green?style=flat)

## Features

- 💬 **AI Chat** — context-aware conversations using Groq (supports multiple models)
- 👁️ **Vision** — analyze images sent in chat
- 🛡️ **Moderation** — kick, ban, warn, timeout, mute, clear, lock, and more
- ⭐ **XP & Leveling** — automatic XP gain with level-up notifications
- 💤 **AFK System** — set AFK status, auto-unafk on message
- 👋 **Welcome/Goodbye** — customizable join and leave messages
- 🎭 **Auto-role** — automatically assign role on join
- 🚫 **Automod** — banned words filter with auto-delete
- ⚙️ **Dashboard** — admin settings via slash command buttons
- 🧠 **Multi-model** — switch AI model per server (Llama 70B, GPT OSS 120B/20B, Qwen 32B)

## Requirements

- Go 1.21+
- A Discord bot token → [Discord Developer Portal](https://discord.com/developers/applications)
- A Groq API key → [console.groq.com](https://console.groq.com)

## Setup

### Option A — Run directly

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

### Option B — Docker

```bash
# Clone the repo
git clone https://github.com/Zaineedyou/CaineGO.git
cd CaineGO

# Copy and fill in your credentials
cp .env.example .env
nano .env

# Run
docker compose up -d

# Logs
docker compose logs -f
```

## Environment Variables

```env
DISCORD_TOKEN=your_discord_bot_token   # required
GROQ_API_KEY=your_groq_api_key         # required
BOT_PREFIX=Caine                        # optional, default: Caine
SYSTEM_PROMPT=                          # optional, custom AI persona
DB_PATH=./caine.db                      # optional, default: ./caine.db
```

## Usage

Mention the bot, use the prefix, or reply to its messages:

```
Caine <question>
Caine help
Caine kick @user reason
Caine setmodel llama70b
/dashboard
/info
/healthcheck
```

### Available Models

| Alias | Model |
|-------|-------|
| `llama70b` | llama-3.3-70b-versatile (default) |
| `gpt120b` | openai/gpt-oss-120b |
| `gpt20b` | openai/gpt-oss-20b |
| `qwen32b` | qwen/qwen3-32b |

## Commands

### AI
| Command | Description |
|---------|-------------|
| `Caine <text>` | Chat with AI |
| `Caine` + image | Analyze image |
| `Caine reset` | Clear conversation memory |
| `Caine summarize [n]` | Summarize last n messages |
| `Caine setmodel <alias>` | Change AI model for this server |
| `Caine setpersona <prompt>` | Set custom AI persona |

### Moderation
| Command | Description |
|---------|-------------|
| `Caine kick @user [reason]` | Kick member |
| `Caine ban @user [reason]` | Ban member |
| `Caine unban <userID>` | Unban member |
| `Caine timeout @user <minutes> [reason]` | Timeout member |
| `Caine untimeout @user` | Remove timeout |
| `Caine warn @user [reason]` | Warn member |
| `Caine warnings [@user]` | View warnings |
| `Caine clearwarn @user` | Clear warnings |
| `Caine clear [n]` | Delete last n messages (default: 10) |
| `Caine lock` | Lock channel |
| `Caine unlock` | Unlock channel |
| `Caine slowmode [seconds]` | Set slowmode (0 to disable) |
| `Caine nick @user <name>` | Change nickname |
| `Caine role add/remove @user @role` | Add or remove role |
| `Caine report @user [reason]` | Report user to admins |
| `Caine addword <word>` | Add word to automod blacklist |
| `Caine removeword <word>` | Remove word from blacklist |
| `Caine words` | List banned words |

### Leveling & AFK
| Command | Description |
|---------|-------------|
| `Caine rank [@user]` | View XP and level |
| `Caine leaderboard` | Top 10 XP leaderboard |
| `Caine afk [reason]` | Set AFK status |
| `Caine afklist` | View who's AFK |

### Admin
| Command | Description |
|---------|-------------|
| `Caine setlog #channel` | Set mod log channel |
| `Caine setwelcome #channel` | Set welcome channel |
| `Caine setgoodbye #channel` | Set goodbye channel |
| `Caine setwelcomemsg <msg>` | Set welcome message |
| `Caine setgoodbyemsg <msg>` | Set goodbye message |
| `Caine autorole @role` | Set auto-role on join |
| `Caine removeautorole` | Remove auto-role |
| `Caine setlevelchannel #channel` | Set level-up notification channel |
| `Caine enable #channel` | Enable bot in channel |
| `Caine disable #channel` | Disable bot in channel |
| `Caine status` | View bot status |

### Slash Commands
| Command | Description |
|---------|-------------|
| `/info` | Bot info and uptime |
| `/help` | Command list |
| `/dashboard` | Admin settings panel (admin only) |
| `/healthcheck` | Check status of all bot components (admin only) |

### Welcome/Goodbye Variables

Use these in welcome/goodbye messages:

| Variable | Description |
|----------|-------------|
| `{user}` | Mention the user |
| `{username}` | Plain username |
| `{server}` | Server name |
| `{count}` | Member count |

## Running in Background

```bash
# Start
nohup ./caine > caine.log 2>&1 &

# Check logs
tail -f caine.log

# Stop
pkill caine
```

## Built With

- [discordgo](https://github.com/bwmarrin/discordgo)
- [Groq API](https://console.groq.com)
- [modernc/sqlite](https://gitlab.com/cznic/sqlite) — pure Go SQLite, no CGO needed

## License

MIT
