# 🤖 CaineGO

A powerful, production-ready Discord bot built in **Go**, powered by **Groq AI** — delivering lightning-fast AI conversations, intelligent moderation, leveling systems, and more to your community.

<div align="center">

![Go](https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![Discord](https://img.shields.io/badge/Discord-5865F2?style=for-the-badge&logo=discord&logoColor=white)
![Groq AI](https://img.shields.io/badge/Groq%20AI-000000?style=for-the-badge)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)

[![Add to Discord](https://img.shields.io/badge/Add%20to%20Discord-5865F2?style=for-the-badge&logo=discord&logoColor=white)](https://discord.com/oauth2/authorize?client_id=1503728763416875118)

</div>

## ✨ Features

### 🧠 AI Chat
- **Groq-Powered Intelligence** — Ultra-fast AI responses with context awareness
- **Multi-Model Support** — Choose from Llama 70B, GPT OSS 120B/20B, Qwen 32B
- **Image Analysis** — Vision capabilities to analyze images in chat
- **Conversational Memory** — Bot remembers conversation context for natural interactions
- **Custom Persona** — Set custom system prompts to define bot personality
- **Message Summarization** — Condense long conversations on demand

### 🛡️ Moderation
- **Smart Content Filtering** — Automatic detection and handling of unwanted content
- **User Management** — Kick, ban, warn, timeout, mute users with detailed logging
- **Moderation Logs** — Full audit trail of all moderation actions with reasons
- **Channel Management** — Lock/unlock channels, enable slowmode
- **Report System** — Members can report issues to moderators
- **Role Management** — Add/remove roles, change nicknames, assign auto-roles

### 📊 Leveling System
- **Automatic XP Tracking** — Members earn XP for participation and messages
- **Leaderboards** — Compete with other community members and rank up
- **Level-Up Notifications** — Custom notifications when members reach new levels
- **Role Rewards** — Assign special roles at specific level thresholds

### 💤 Additional Features
- **AFK System** — Set AFK status with custom reasons, auto-unafk on message
- **Welcome/Goodbye Messages** — Customizable join and leave messages with variables
- **Auto-Role Assignment** — Automatically assign roles when members join
- **Channel-Specific Toggling** — Enable/disable bot per channel
- **Word Blacklist** — Automod filtering for banned words

### ⚡ Performance & Reliability
- **Built in Go** — Blazing-fast execution and minimal resource usage
- **SQLite Database** — Lightweight, zero-configuration persistent storage
- **Docker Ready** — Easy deployment with included Docker configuration
- **Health Checks** — Built-in monitoring and self-diagnostics
- **Pure Go SQLite** — No CGO dependencies, truly portable

### 🔒 Security & Privacy
- **Secure API Handling** — Safe credential management with environment variables
- **User Privacy** — Minimal data collection
- **Permission System** — Granular control over who can use moderation features
- **No Special User Permissions Required** — Regular users can use AI chat without admin perms

---

## 🚀 Getting Started

### Prerequisites

- **Go** 1.21 or higher
- **Discord Bot Token** (from [Discord Developer Portal](https://discord.com/developers/applications))
- **Groq API Key** (free tier available at [Groq Console](https://console.groq.com))

### Quick Installation

#### Option 1: Direct Run

1. **Clone the repository:**
   ```bash
   git clone https://github.com/Zaineedyou/CaineGO.git
   cd CaineGO
   ```

2. **Copy and configure environment variables:**
   ```bash
   cp .env.example .env
   nano .env  # or your favorite editor
   ```

3. **Install dependencies and run:**
   ```bash
   go mod tidy
   go build -o caine .
   ./caine
   ```

#### Option 2: Docker (Recommended for Production)

1. **Clone and configure:**
   ```bash
   git clone https://github.com/Zaineedyou/CaineGO.git
   cd CaineGO
   cp .env.example .env
   nano .env
   ```

2. **Run with Docker Compose:**
   ```bash
   docker compose up -d
   
   # View logs
   docker compose logs -f
   ```

### Environment Variables

```env
# Required
DISCORD_TOKEN=your_discord_bot_token
GROQ_API_KEY=your_groq_api_key

# Optional
BOT_PREFIX=Caine                    # Command prefix (default: Caine)
SYSTEM_PROMPT=                      # Custom AI persona
DB_PATH=./caine.db                  # Database location (default: ./caine.db)
LOG_LEVEL=info                      # Log verbosity (default: info)
```

### First Steps After Installation

1. **Invite the Bot to Your Server:**
   - [Use this link](https://discord.com/oauth2/authorize?client_id=1503728763416875118) or generate your own from Discord Developer Portal
   - Required permissions: `Send Messages`, `Read Message History`
   - Optional for moderation: `Manage Messages`, `Kick Members`, `Ban Members`, `Manage Roles`

2. **Test the Bot:**
   ```
   Caine hello
   @CaineGO what can you do?
   ```

3. **Configure Server (Optional):**
   ```
   Caine setlog #mod-logs
   Caine setwelcome #welcome
   Caine autorole @member
   ```

---

## 📖 Usage & Commands

### AI Chat Commands
Simply use the prefix or mention the bot:

```
Caine <question>              # Ask anything
Caine help                    # Show help
Caine reset                   # Clear conversation memory
Caine summarize [n]           # Summarize last n messages
Caine setmodel <alias>        # Change AI model
Caine setpersona <prompt>     # Set custom AI personality
```

**Send an image** → bot will analyze it with vision capabilities

### Available AI Models

| Alias | Model | Speed | Quality |
|-------|-------|-------|---------|
| `llama70b` | llama-3.3-70b-versatile | ⚡⚡⚡ | ⭐⭐⭐⭐⭐ |
| `gpt120b` | openai/gpt-oss-120b | ⚡⚡ | ⭐⭐⭐⭐ |
| `gpt20b` | openai/gpt-oss-20b | ⚡⚡⚡⚡ | ⭐⭐⭐ |
| `qwen32b` | qwen/qwen3-32b | ⚡⚡⚡ | ⭐⭐⭐⭐ |

### Moderation Commands
**Bot needs appropriate permissions. Users need moderator role.**

```
Caine kick @user [reason]             # Kick member
Caine ban @user [reason]              # Ban member
Caine unban <userID>                  # Unban member
Caine timeout @user <minutes> [reason] # Timeout member
Caine untimeout @user                 # Remove timeout
Caine warn @user [reason]             # Warn member
Caine warnings [@user]                # View warnings
Caine clearwarn @user                 # Clear warnings
Caine clear [n]                       # Delete last n messages
Caine lock                            # Lock channel
Caine unlock                          # Unlock channel
Caine slowmode [seconds]              # Set slowmode (0 to disable)
Caine nick @user <name>               # Change nickname
Caine role add/remove @user @role     # Manage roles
Caine report @user [reason]           # Report user to admins
Caine addword <word>                  # Add word to blacklist
Caine removeword <word>               # Remove from blacklist
Caine words                           # Show banned words
```

### Leveling & AFK

```
Caine rank [@user]            # View your XP and level
Caine leaderboard             # Top 10 members
Caine afk [reason]            # Set AFK status
Caine afklist                 # See who's AFK
```

### Admin/Configuration

```
Caine setlog #channel         # Mod log channel
Caine setwelcome #channel     # Welcome message channel
Caine setgoodbye #channel     # Goodbye message channel
Caine setwelcomemsg <msg>     # Set custom welcome message
Caine setgoodbyemsg <msg>     # Set custom goodbye message
Caine autorole @role          # Auto-assign role on join
Caine removeautorole          # Remove auto-role
Caine setlevelchannel #chan   # Level-up notification channel
Caine enable #channel         # Enable bot in channel
Caine disable #channel        # Disable bot in channel
Caine status                  # Bot status info
```

### Slash Commands

```
/info                         # Bot info and uptime
/help                         # Full command list
/dashboard                    # Settings panel (admin only)
/healthcheck                  # System diagnostics (admin only)
```

### Welcome/Goodbye Message Variables

Use these in custom messages:

```
{user}      → Mention the user (@username)
{username}  → Plain username
{server}    → Server name
{count}     → Current member count
```

Example:
```
Caine setwelcomemsg Welcome to {server}, {user}! We now have {count} members! 🎉
```

---

## 🔧 Configuration & Deployment

### Running in Background

```bash
# Start as daemon
nohup ./caine > caine.log 2>&1 &

# View logs
tail -f caine.log

# Stop
pkill caine
```

### Using Docker

```bash
# Start
docker compose up -d

# View logs
docker compose logs -f

# Stop
docker compose down

# Rebuild
docker compose up -d --build
```

### Database

- **Type:** SQLite (pure Go, no external dependencies)
- **Location:** `./caine.db` (configurable via `DB_PATH`)
- **Stores:** User XP/levels, mod logs, server settings, AFK statuses, banned words
- **Auto-created:** On first run

---

## 🚨 Troubleshooting

### Bot doesn't respond to commands

**Check:**
- ✅ `DISCORD_TOKEN` is correct in `.env`
- ✅ Bot has `Send Messages` permission
- ✅ Bot is online (Discord Developer Portal → Applications)
- ✅ Using correct prefix (default: `Caine`)

### AI responses are slow or rate-limited

- This is normal during high Groq load
- Upgrade your Groq plan for higher rate limits
- Responses typically arrive within 1-3 seconds with free tier

### Moderation commands return "Missing Permissions"

- ✅ Bot needs: `Manage Messages`, `Kick Members`, `Ban Members` permissions
- ✅ Bot's role must be **above** target user's role in hierarchy
- ✅ User executing command must have moderator/admin role

### Database errors

- ✅ Check write permissions in bot's working directory: `chmod 755 .`
- ✅ Verify disk space available
- ✅ Check SQLite isn't locked by another process

### Bot keeps crashing

- Check logs: `docker compose logs` or `tail -f caine.log`
- Verify all environment variables are set
- Ensure Go 1.21+ installed (for direct run)

---

## 📁 Project Structure

```
CaineGO/
├── main.go              # Bot initialization & event handlers
├── ai.go                # Groq AI integration & chat logic
├── moderation.go        # Moderation system & enforcement
├── db.go                # SQLite database & data models
├── healthcheck.go       # Health monitoring & diagnostics
├── logging.go           # Structured logging setup
├── utils.go             # Helper functions & utilities
├── Dockerfile           # Docker container configuration
├── docker-compose.yml   # Docker Compose orchestration
├── go.mod               # Go module dependencies
├── go.sum               # Dependency checksums
├── .env.example         # Example environment configuration
└── LICENSE              # MIT License
```

---

## 🤝 Contributing

We welcome contributions! To contribute:

1. **Fork the repository**
2. **Create a feature branch:**
   ```bash
   git checkout -b feature/amazing-feature
   ```
3. **Commit your changes:**
   ```bash
   git commit -m 'Add amazing feature'
   ```
4. **Push to the branch:**
   ```bash
   git push origin feature/amazing-feature
   ```
5. **Open a Pull Request**

---

## 📝 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.

---

## 🙏 Built With

- **[discordgo](https://github.com/bwmarrin/discordgo)** — Go Discord API wrapper
- **[Groq API](https://console.groq.com)** — Ultra-fast AI inference
- **[modernc/sqlite](https://gitlab.com/cznic/sqlite)** — Pure Go SQLite (no CGO needed)

---

## 🙋 Support & Community

- 📄 **Found a bug?** [Open an issue](https://github.com/Zaineedyou/CaineGO/issues)
- 💡 **Have a suggestion?** [Create a discussion](https://github.com/Zaineedyou/CaineGO/discussions)
- 🤖 **Add bot to your server:** [Click here](https://discord.com/oauth2/authorize?client_id=1503728763416875118)

---

## 🎯 Roadmap

- [ ] Web dashboard for server management
- [ ] Advanced analytics and insights
- [ ] Custom AI model selection per channel
- [ ] Giveaway and contests system
- [ ] Advanced anti-spam detection
- [ ] Multi-language support

---

<div align="center">

**Made with ❤️ by [Zaineedyou](https://github.com/Zaineedyou)**

[⭐ Star us on GitHub](https://github.com/Zaineedyou/CaineGO) | [🤖 Add to Your Server](https://discord.com/oauth2/authorize?client_id=1503728763416875118) | [📧 Report an Issue](https://github.com/Zaineedyou/CaineGO/issues)

</div>
