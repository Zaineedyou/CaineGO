package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const VERSION = "1.0.0"

var (
	DISCORD_TOKEN  string
	GROQ_API_KEY   string
	BOT_PREFIX     string
	BOT_OWNER_ID   string

	DEFAULT_SYSTEM_PROMPT string
	DEFAULT_MODEL         = "llama-3.3-70b-versatile"

	slashCommandsMu         sync.Mutex
	slashCommandsRegistered bool
)

// MAX_HISTORY is the default conversation history limit per channel.
// Can be overridden per guild via the dashboard or sethistory command.
var MAX_HISTORY int

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Println("Caine Bot v" + VERSION)
			os.Exit(0)
		}
	}

	godotenv.Load()

	DISCORD_TOKEN = os.Getenv("DISCORD_TOKEN")
	GROQ_API_KEY = os.Getenv("GROQ_API_KEY")
	BOT_PREFIX = getEnvOrDefault("BOT_PREFIX", "Caine")
	DEFAULT_SYSTEM_PROMPT = getEnvOrDefault("SYSTEM_PROMPT",
		"Kamu adalah AI asisten yang nyantai dan gaul. Jawab pake bahasa Indonesia slang yang natural, kayak ngobrol sama teman. Tetep informatif dan tepat tapi ga kaku.")

	BOT_OWNER_ID = os.Getenv("BOT_OWNER_ID")
	MAX_HISTORY = getEnvInt("MAX_HISTORY", 30)
	if v := getEnvInt("AI_RATE_MAX", 0); v > 0 {
		AI_RATE_MAX = v
	}
	if v := getEnvInt("AI_RATE_WINDOW_SEC", 0); v > 0 {
		AI_RATE_WINDOW = int64(v * 1000)
	}

	if DISCORD_TOKEN == "" || GROQ_API_KEY == "" {
		fmt.Println("❌ DISCORD_TOKEN and GROQ_API_KEY are required")
		os.Exit(1)
	}

	initDB()

	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
		<-sc
		fmt.Println("\n🛑 Shutting down...")
		flushDB()
		os.Exit(0)
	}()

	retryDelay := 5 * time.Second
	for {
		runBot() // hanya return kalau Open() gagal
		fmt.Printf("⚠️ Failed to connect to Discord, retrying in %s...\n", retryDelay)
		time.Sleep(retryDelay)
		if retryDelay < 60*time.Second {
			retryDelay *= 2
		}
	}
}

func runBot() {
	dg, err := discordgo.New("Bot " + DISCORD_TOKEN)
	if err != nil {
		fmt.Println("❌ Failed to create session:", err)
		return
	}

	dg.Identify.Intents = discordgo.IntentsGuilds |
		discordgo.IntentsGuildMessages |
		discordgo.IntentsMessageContent |
		discordgo.IntentsDirectMessages |
		discordgo.IntentsGuildMembers

	dg.AddHandler(onReady)
	dg.AddHandler(onMessageCreate)
	dg.AddHandler(onGuildMemberAdd)
	dg.AddHandler(onGuildMemberRemove)
	dg.AddHandler(onInteractionCreate)

	err = dg.Open()
	if err != nil {
		fmt.Println("❌ Failed to open connection:", err)
		return
	}
	defer dg.Close()

	fmt.Println("✅ Bot running. Press CTRL-C to exit.")

	select {}
}

