package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const VERSION = "1.0.0"

var (
	DISCORD_TOKEN string
	GROQ_API_KEY  string
	BOT_PREFIX    string

	DEFAULT_SYSTEM_PROMPT string
	DEFAULT_MODEL         = "llama-3.3-70b-versatile"

	// Flag biar slash commands cuma di-register sekali, bukan tiap reconnect
	slashCommandsRegistered bool
)

const MAX_HISTORY = 30

func getEnvOrDefault(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}

func main() {
	// --version flag
	for _, arg := range os.Args[1:] {
		if arg == "--version" || arg == "-v" {
			fmt.Println("Caine Bot v" + VERSION)
			os.Exit(0)
		}
	}

	rand.Seed(time.Now().UnixNano())

	// Load .env file kalau ada
	godotenv.Load()

	// Baca env SETELAH godotenv.Load()
	DISCORD_TOKEN = os.Getenv("DISCORD_TOKEN")
	GROQ_API_KEY = os.Getenv("GROQ_API_KEY")
	BOT_PREFIX = getEnvOrDefault("BOT_PREFIX", "Caine")
	DEFAULT_SYSTEM_PROMPT = getEnvOrDefault("SYSTEM_PROMPT",
		"Kamu adalah AI asisten yang nyantai dan gaul. Jawab pake bahasa Indonesia slang yang natural, kayak ngobrol sama teman. Tetep informatif dan tepat tapi ga kaku.")

	// Validasi env wajib
	if DISCORD_TOKEN == "" || GROQ_API_KEY == "" {
		fmt.Println("❌ DISCORD_TOKEN dan GROQ_API_KEY wajib diset!")
		os.Exit(1)
	}

	// Init DB cache
	initDB()

	// Graceful shutdown
	go func() {
		sc := make(chan os.Signal, 1)
		signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM)
		<-sc
		fmt.Println("\n🛑 Shutting down, saving data...")
		flushDB()
		os.Exit(0)
	}()

	// runBot() sekarang hanya return kalau dg.Open() gagal (misal token salah, no network).
	// discordgo sudah handle reconnect otomatis untuk disconnect sementara di dalam runBot().
	// Loop ini hanya sebagai fallback kalau koneksi awal betul-betul gagal.
	retryDelay := 5 * time.Second
	for {
		runBot() // hanya return kalau Open() gagal
		fmt.Printf("⚠️ Gagal konek ke Discord, retry in %s...\n", retryDelay)
		time.Sleep(retryDelay)
		if retryDelay < 60*time.Second {
			retryDelay *= 2
		}
	}
}

func runBot() {
	dg, err := discordgo.New("Bot " + DISCORD_TOKEN)
	if err != nil {
		fmt.Println("❌ Error creating session:", err)
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
		fmt.Println("❌ Error opening connection:", err)
		return
	}
	defer dg.Close()

	fmt.Println("✅ Bot running. CTRL-C to exit.")

	// discordgo sudah handle reconnect otomatis secara internal.
	// Kita TIDAK pakai Disconnect handler untuk close channel done —
	// itu penyebab looping: setiap disconnect sementara (network blip, timeout)
	// akan menutup done lalu main() spawn runBot() baru tanpa henti.
	// Cukup block di sini selamanya, shutdown hanya lewat OS signal (SIGINT/SIGTERM).
	select {}
}

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	// Suppress error log yang ga penting dari discordgo
	s.LogLevel = discordgo.LogWarning
	fmt.Printf("✅ Bot online: %s\n", r.User.Username)
	s.UpdateCustomStatus("Property Of Caineedyou | Developed By Zaineedyou")

	// Reset retryDelay secara implisit dengan flag — kalau bot berhasil online,
	// artinya koneksi sukses. Slash commands hanya didaftarkan SEKALI.
	// Kalau didaftarkan ulang tiap reconnect, Discord akan rate-limit dan
	// langsung disconnect lagi — itulah penyebab looping cepat sebelumnya.
	if !slashCommandsRegistered {
		commands := []*discordgo.ApplicationCommand{
			{Name: "info", Description: "Lihat info dan status bot Caine"},
			{Name: "dashboard", Description: "Buka dashboard pengaturan bot (Admin only)"},
			{Name: "help", Description: "Lihat semua command yang tersedia"},
		}
		for _, cmd := range commands {
			s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		}
		slashCommandsRegistered = true
		fmt.Println("✅ Slash commands terdaftar")
	}
}

func onGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	// Auto-role
	roleId := getAutoRole(m.GuildID)
	if roleId != "" {
		s.GuildMemberRoleAdd(m.GuildID, m.User.ID, roleId)
	}

	// Welcome
	chId := getWelcomeChannel(m.GuildID)
	if chId == "" {
		return
	}
	g, _ := s.State.Guild(m.GuildID)
	guildName := ""
	memberCount := 0
	if g != nil {
		guildName = g.Name
		memberCount = g.MemberCount
	}
	msg := getWelcomeMessage(m.GuildID)
	msg = strings.ReplaceAll(msg, "{user}", fmt.Sprintf("<@%s>", m.User.ID))
	msg = strings.ReplaceAll(msg, "{username}", m.User.Username)
	msg = strings.ReplaceAll(msg, "{server}", guildName)
	msg = strings.ReplaceAll(msg, "{count}", fmt.Sprintf("%d", memberCount))

	s.ChannelMessageSendEmbed(chId, &discordgo.MessageEmbed{
		Color:       0x00ff7f,
		Title:       "👋 Member Baru!",
		Description: msg,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: m.User.AvatarURL("256")},
	})
}

func onGuildMemberRemove(s *discordgo.Session, m *discordgo.GuildMemberRemove) {
	chId := getGoodbyeChannel(m.GuildID)
	if chId == "" {
		return
	}
	g, _ := s.State.Guild(m.GuildID)
	guildName := ""
	memberCount := 0
	if g != nil {
		guildName = g.Name
		memberCount = g.MemberCount
	}
	msg := getGoodbyeMessage(m.GuildID)
	msg = strings.ReplaceAll(msg, "{user}", fmt.Sprintf("<@%s>", m.User.ID))
	msg = strings.ReplaceAll(msg, "{username}", m.User.Username)
	msg = strings.ReplaceAll(msg, "{server}", guildName)
	msg = strings.ReplaceAll(msg, "{count}", fmt.Sprintf("%d", memberCount))

	s.ChannelMessageSendEmbed(chId, &discordgo.MessageEmbed{
		Color:       0xff4444,
		Title:       "👋 Member Keluar",
		Description: msg,
		Thumbnail:   &discordgo.MessageEmbedThumbnail{URL: m.User.AvatarURL("256")},
	})
}

func onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("⚠️ Recovered panic di onMessageCreate:", r)
		}
	}()

	if m.Author == nil || m.Author.Bot {
		return
	}

	// Rate limiter — ignore kalau spam
	if isRateLimited(m.Author.ID) {
		return
	}

	// Automod
	if m.GuildID != "" {
		lower := strings.ToLower(m.Content)
		for _, word := range getBannedWords(m.GuildID) {
			if strings.Contains(lower, word) {
				s.ChannelMessageDelete(m.ChannelID, m.ID)
				logAutomod(s, m.Message, word)
				s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("⚠️ Pesan <@%s> dihapus karena mengandung kata terlarang.", m.Author.ID))
				return
			}
		}
	}

	// XP
	if m.GuildID != "" {
		handleXP(s, m.Message)
	}

	// AFK auto-unafk
	if m.GuildID != "" {
		if getAfkUser(m.Author.ID, m.GuildID) != nil {
			removeAfkUser(m.Author.ID, m.GuildID)
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("✅ Welcome back <@%s>! AFK kamu dihapus.", m.Author.ID), m.Reference())
		}

		// AFK mention check
		for _, u := range m.Mentions {
			afkData := getAfkUser(u.ID, m.GuildID)
			if afkData != nil && u.ID != m.Author.ID {
				tm, _ := s.GuildMember(m.GuildID, u.ID)
				name := u.Username
				if tm != nil && tm.Nick != "" {
					name = tm.Nick
				}
				elapsed := formatDuration(nowMs() - afkData.Time)
				s.ChannelMessageSendReply(m.ChannelID,
					fmt.Sprintf("💤 **%s** lagi AFK: *%s* (%s lalu)", name, afkData.Reason, elapsed),
					m.Reference())
				break
			}
		}
	}

	// Channel disable check
	if m.GuildID != "" && isChannelDisabled(m.GuildID, m.ChannelID) {
		return
	}

	content := strings.TrimSpace(m.Content)
	isMentioned := false
	for _, u := range m.Mentions {
		if u.ID == s.State.User.ID {
			isMentioned = true
			break
		}
	}
	// @everyone juga trigger bot
	if m.MentionEveryone {
		isMentioned = true
	}
	hasPrefix := containsWord(content, BOT_PREFIX)

	isReply := false
	if m.MessageReference != nil {
		refMsg, err := s.ChannelMessage(m.ChannelID, m.MessageReference.MessageID)
		if err == nil && refMsg.Author.ID == s.State.User.ID {
			isReply = true
		}
	}

	if !hasPrefix && !isMentioned && !isReply {
		return
	}

	userText := content
	if hasPrefix {
		idx := strings.Index(strings.ToLower(content), strings.ToLower(BOT_PREFIX))
		if idx >= 0 {
			userText = strings.TrimSpace(content[:idx] + content[idx+len(BOT_PREFIX):])
		}
	} else if isMentioned {
		userText = strings.TrimSpace(strings.ReplaceAll(content, fmt.Sprintf("<@%s>", s.State.User.ID), ""))
	}

	historyKey := "server-" + m.ChannelID
	if m.GuildID == "" {
		historyKey = "dm-" + m.Author.ID
	}

	displayName := m.Author.Username
	if m.GuildID != "" {
		mem, err := s.GuildMember(m.GuildID, m.Author.ID)
		if err == nil && mem.Nick != "" {
			displayName = mem.Nick
		}
	}

	// Simple commands
	if strings.ToLower(userText) == "reset" || strings.ToLower(userText) == "clear" {
		clearHistory(historyKey)
		s.ChannelMessageSendReply(m.ChannelID, "🧹 Memory kita udah di-reset sayang!", m.Reference())
		return
	}

	if strings.ToLower(userText) == "help" {
		helpText := "**Hai sayang! Ini cara pakai aku:**\n" +
			"`Caine <pertanyaan>` — tanya apapun\n" +
			"`Caine` + kirim gambar — analisis gambar\n" +
			"`Caine summarize [jumlah]` — rangkum chat\n" +
			"`Caine report @user alasan` — laporin user\n" +
			"`Caine reset` — hapus memory\n" +
			"`Caine afk [alasan]` — set AFK\n" +
			"`Caine afklist` — lihat siapa yang AFK\n" +
			"`Caine rank [@user]` — lihat rank/XP\n" +
			"`Caine leaderboard` — top 10 XP\n" +
			"`Caine status` — status bot\n" +
			"`Caine setmodel <alias>` — ganti model AI\n" +
			"`/info` — info bot\n" +
			"`/dashboard` — buka dashboard (admin)\n\n" +
			"**Moderasi:** kick, ban, unban, timeout, untimeout, warn, warnings, clearwarn, clear, lock, unlock, slowmode, nick, role add/remove\n\n" +
			"**Admin:** addword, removeword, words, enable, disable, setlog, setwelcome, setgoodbye, setwelcomemsg, setgoodbyemsg, autorole, removeautorole, setlevelchannel, setpersona, setmodel"
		s.ChannelMessageSendReply(m.ChannelID, helpText, m.Reference())
		return
	}

	// Moderation commands
	if handleModeration(s, m.Message, userText) {
		return
	}

	// AI chat
	var imageURL string
	for _, att := range m.Attachments {
		if strings.HasPrefix(att.ContentType, "image/") {
			imageURL = att.URL
			break
		}
	}

	s.ChannelTyping(m.ChannelID)

	var reply string
	var aiErr error

	if imageURL != "" {
		reply, aiErr = askVision(historyKey, userText, imageURL, displayName, m.GuildID)
	} else {
		prompt := userText
		if prompt == "" {
			prompt = "Seseorang baru manggil namamu. Balas dengan sapaan mesra seperti pacar, jangan pakai kata bro."
		}
		reply, aiErr = askGroq(historyKey, prompt, displayName, m.GuildID)
	}

	if aiErr != nil {
		fmt.Println("AI Error:", aiErr)
		s.ChannelMessageSendReply(m.ChannelID, "❌ Ada error sayang, coba lagi ya 🙏", m.Reference())
		return
	}

	chunks := splitMessage(reply, 1900)
	for _, chunk := range chunks {
		s.ChannelMessageSendReply(m.ChannelID, chunk, m.Reference())
	}
	logChat(s, m.Message, userText, reply)
}

// ============================================================
// SLASH COMMANDS + DASHBOARD
// ============================================================

func onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		handleSlashCommand(s, i)
	case discordgo.InteractionMessageComponent:
		handleButtonInteraction(s, i)
	case discordgo.InteractionModalSubmit:
		handleModalSubmit(s, i)
	}
}

func handleSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.ApplicationCommandData().Name {
	case "info":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Color: 0xff69b4,
						Title: "💕 Caine — AI Discord Bot",
						Description: "Halo! Aku Caine, AI asisten yang siap bantu kamu di server ini~",
						Fields: []*discordgo.MessageEmbedField{
							{Name: "👨‍💻 Developer", Value: "Zaineedyou", Inline: true},
							{Name: "🖥️ Infrastructure", Value: "Zaineedyou", Inline: true},
							{Name: "🤖 Default Model", Value: "Llama 3.3 70B", Inline: true},
							{Name: "👁️ Vision Model", Value: "Llama 4 Scout 17B", Inline: true},
							{Name: "🏷️ Versi", Value: "v" + VERSION, Inline: true},
						{Name: "⏱️ Uptime", Value: getUptime(), Inline: true},
							{Name: "📡 Status", Value: "🟢 Online", Inline: true},
						},
						Footer: &discordgo.MessageEmbedFooter{Text: "Property Of Caineedyou | Developed by Zaineedyou"},
					},
				},
			},
		})

	case "help":
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{
					{
						Color: 0x5865f2,
						Title: "📖 Help — Caine Bot",
						Fields: []*discordgo.MessageEmbedField{
							{Name: "💬 AI Chat", Value: "`Caine <pertanyaan>` atau mention\nKirim gambar + teks untuk analisis visual\n`Caine reset` - hapus memory"},
							{Name: "🛡️ Moderation", Value: "`Caine kick/ban/unban @user`\n`Caine warn/warnings/clearwarn @user`\n`Caine timeout @user <menit>`\n`Caine clear <jumlah>`\n`Caine lock/unlock/slowmode`"},
							{Name: "⭐ Leveling & AFK", Value: "`Caine rank [@user]`\n`Caine leaderboard`\n`Caine afk [alasan]`\n`Caine afklist`"},
							{Name: "⚙️ Admin", Value: "`Caine setlog #channel`\n`Caine setwelcome/setgoodbye #channel`\n`Caine autorole @role`\n`Caine addword/removeword <kata>`\n`Caine setpersona <prompt>`\n`Caine setmodel <alias>`"},
							{Name: "🧠 Model Tersedia", Value: "`llama70b` `gpt120b` `gpt20b` `qwen32b`"},
							{Name: "🔧 Slash Commands", Value: "`/info` `/dashboard` `/help`"},
						},
						Footer: &discordgo.MessageEmbedFooter{Text: "Prefix: " + BOT_PREFIX + " | Bisa juga di-mention atau reply"},
					},
				},
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})

	case "dashboard":
		if i.Member == nil {
			return
		}
		perms := i.Member.Permissions
		if perms&discordgo.PermissionAdministrator == 0 {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Khusus admin.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		respondDashboardMain(s, i)
	}
}

func respondDashboardMain(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildName := ""
	if i.GuildID != "" {
		g, _ := s.State.Guild(i.GuildID)
		if g != nil {
			guildName = g.Name
		}
	}
	embed := &discordgo.MessageEmbed{
		Color:       0x5865f2,
		Title:       "⚙️ Dashboard Bot Caine",
		Description: fmt.Sprintf("Server: **%s**\nPilih menu di bawah:", guildName),
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "📋 General", CustomID: "dash_general", Style: discordgo.PrimaryButton},
				discordgo.Button{Label: "👋 Welcome/Goodbye", CustomID: "dash_welcome", Style: discordgo.SuccessButton},
				discordgo.Button{Label: "🎭 Auto-role", CustomID: "dash_autorole", Style: discordgo.SecondaryButton},
				discordgo.Button{Label: "⭐ Leveling", CustomID: "dash_leveling", Style: discordgo.SecondaryButton},
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{Label: "🤖 Persona", CustomID: "dash_persona", Style: discordgo.PrimaryButton},
				discordgo.Button{Label: "🧠 Model AI", CustomID: "dash_model", Style: discordgo.PrimaryButton},
				discordgo.Button{Label: "🛡️ Moderation", CustomID: "dash_moderation", Style: discordgo.DangerButton},
				discordgo.Button{Label: "📊 Status Bot", CustomID: "dash_status", Style: discordgo.PrimaryButton},
			},
		},
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
			Flags:      discordgo.MessageFlagsEphemeral,
		},
	})
}

func backRow() discordgo.ActionsRow {
	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.Button{Label: "◀ Kembali", CustomID: "dash_back", Style: discordgo.SecondaryButton},
		},
	}
}

type modalData struct {
	CustomID   string
	Title      string
	Components []discordgo.MessageComponent
}

func handleButtonInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Khusus admin.", Flags: discordgo.MessageFlagsEphemeral},
		})
		return
	}

	guildId := i.GuildID

	updateMsg := func(embed *discordgo.MessageEmbed, components []discordgo.MessageComponent) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}

	showModal := func(modal modalData) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				CustomID:   modal.CustomID,
				Title:      modal.Title,
				Components: modal.Components,
			},
		})
	}

	ephemeralReply := func(content string) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
		})
	}

	switch i.MessageComponentData().CustomID {
	case "dash_back":
		guildName := ""
		g, _ := s.State.Guild(guildId)
		if g != nil {
			guildName = g.Name
		}
		updateMsg(
			&discordgo.MessageEmbed{
				Color:       0x5865f2,
				Title:       "⚙️ Dashboard Bot Caine",
				Description: fmt.Sprintf("Server: **%s**\nPilih menu di bawah:", guildName),
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "📋 General", CustomID: "dash_general", Style: discordgo.PrimaryButton},
					discordgo.Button{Label: "👋 Welcome/Goodbye", CustomID: "dash_welcome", Style: discordgo.SuccessButton},
					discordgo.Button{Label: "🎭 Auto-role", CustomID: "dash_autorole", Style: discordgo.SecondaryButton},
					discordgo.Button{Label: "⭐ Leveling", CustomID: "dash_leveling", Style: discordgo.SecondaryButton},
				}},
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "🤖 Persona", CustomID: "dash_persona", Style: discordgo.PrimaryButton},
					discordgo.Button{Label: "🧠 Model AI", CustomID: "dash_model", Style: discordgo.PrimaryButton},
					discordgo.Button{Label: "🛡️ Moderation", CustomID: "dash_moderation", Style: discordgo.DangerButton},
					discordgo.Button{Label: "📊 Status Bot", CustomID: "dash_status", Style: discordgo.PrimaryButton},
				}},
			},
		)

	case "dash_general":
		logCh := getGuildLogChannel(guildId)
		logVal := "❌ Belum diset"
		if logCh != "" {
			logVal = fmt.Sprintf("<#%s>", logCh)
		}
		var disabledChs []string
		for _, id := range func() []string {
			rows, _ := db.Query(`SELECT channel_id FROM disabled_channels WHERE guild_id=?`, guildId)
			if rows == nil { return nil }
			defer rows.Close()
			var ids []string
			for rows.Next() { var id string; rows.Scan(&id); ids = append(ids, id) }
			return ids
		}() {
			disabledChs = append(disabledChs, fmt.Sprintf("<#%s>", id))
		}
		disabledVal := "Tidak ada"
		if len(disabledChs) > 0 {
			disabledVal = strings.Join(disabledChs, ", ")
		}
		updateMsg(
			&discordgo.MessageEmbed{
				Color: 0x5865f2, Title: "📋 General Settings",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "📝 Log Channel", Value: logVal},
					{Name: "🚫 Disabled Channels", Value: disabledVal},
				},
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Set Log Channel", CustomID: "dash_setlog", Style: discordgo.PrimaryButton},
				}},
				backRow(),
			},
		)

	case "dash_setlog":
		showModal(modalData{
			CustomID: "modal_setlog", Title: "Set Log Channel",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "channel_id", Label: "Channel ID", Style: discordgo.TextInputShort, Required: true, Placeholder: "Contoh: 1234567890123456789"},
				}},
			},
		})

	case "dash_welcome":
		welcomeCh := getWelcomeChannel(guildId)
		goodbyeCh := getGoodbyeChannel(guildId)
		wChVal := "❌ Belum diset"
		if welcomeCh != "" {
			wChVal = fmt.Sprintf("<#%s>", welcomeCh)
		}
		gChVal := "❌ Belum diset"
		if goodbyeCh != "" {
			gChVal = fmt.Sprintf("<#%s>", goodbyeCh)
		}
		updateMsg(
			&discordgo.MessageEmbed{
				Color: 0x00ff7f, Title: "👋 Welcome / Goodbye",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Welcome Channel", Value: wChVal},
					{Name: "Welcome Message", Value: fmt.Sprintf("`%s`", getWelcomeMessage(guildId))},
					{Name: "Goodbye Channel", Value: gChVal},
					{Name: "Goodbye Message", Value: fmt.Sprintf("`%s`", getGoodbyeMessage(guildId))},
					{Name: "ℹ️ Variabel", Value: "`{user}` `{username}` `{server}` `{count}`"},
				},
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Set Welcome Channel", CustomID: "dash_setwelcome", Style: discordgo.SuccessButton},
					discordgo.Button{Label: "Set Welcome Message", CustomID: "dash_setwelcomemsg", Style: discordgo.PrimaryButton},
					discordgo.Button{Label: "Set Goodbye Channel", CustomID: "dash_setgoodbye", Style: discordgo.DangerButton},
					discordgo.Button{Label: "Set Goodbye Message", CustomID: "dash_setgoodbyemsg", Style: discordgo.PrimaryButton},
				}},
				backRow(),
			},
		)

	case "dash_setwelcome":
		showModal(modalData{
			CustomID: "modal_setwelcome", Title: "Set Welcome Channel",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "channel_id", Label: "Channel ID", Style: discordgo.TextInputShort, Required: true},
				}},
			},
		})
	case "dash_setgoodbye":
		showModal(modalData{
			CustomID: "modal_setgoodbye", Title: "Set Goodbye Channel",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "channel_id", Label: "Channel ID", Style: discordgo.TextInputShort, Required: true},
				}},
			},
		})
	case "dash_setwelcomemsg":
		showModal(modalData{
			CustomID: "modal_setwelcomemsg", Title: "Set Welcome Message",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "msg", Label: "Pesan Welcome", Style: discordgo.TextInputParagraph, Required: true, Value: getWelcomeMessage(guildId)},
				}},
			},
		})
	case "dash_setgoodbyemsg":
		showModal(modalData{
			CustomID: "modal_setgoodbyemsg", Title: "Set Goodbye Message",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "msg", Label: "Pesan Goodbye", Style: discordgo.TextInputParagraph, Required: true, Value: getGoodbyeMessage(guildId)},
				}},
			},
		})

	case "dash_autorole":
		autoRole := getAutoRole(guildId)
		roleVal := "❌ Belum diset"
		if autoRole != "" {
			roleVal = fmt.Sprintf("<@&%s>", autoRole)
		}
		updateMsg(
			&discordgo.MessageEmbed{
				Color: 0xffa500, Title: "🎭 Auto-role",
				Fields: []*discordgo.MessageEmbedField{{Name: "Role Saat Join", Value: roleVal}},
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Set Auto-role", CustomID: "dash_setautorole", Style: discordgo.PrimaryButton},
					discordgo.Button{Label: "Hapus Auto-role", CustomID: "dash_removeautorole", Style: discordgo.DangerButton},
				}},
				backRow(),
			},
		)
	case "dash_setautorole":
		showModal(modalData{
			CustomID: "modal_setautorole", Title: "Set Auto-role",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "role_id", Label: "Role ID", Style: discordgo.TextInputShort, Required: true},
				}},
			},
		})
	case "dash_removeautorole":
		removeAutoRole(guildId)
		ephemeralReply("✅ Auto-role dihapus.")

	case "dash_leveling":
		levelCh := getLevelChannel(guildId)
		levelChVal := "❌ (notif di channel chat)"
		if levelCh != "" {
			levelChVal = fmt.Sprintf("<#%s>", levelCh)
		}
		updateMsg(
			&discordgo.MessageEmbed{
				Color: 0xffd700, Title: "⭐ Leveling",
				Fields: []*discordgo.MessageEmbedField{{Name: "Level-up Channel", Value: levelChVal}},
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Set Level Channel", CustomID: "dash_setlevelchannel", Style: discordgo.PrimaryButton},
				}},
				backRow(),
			},
		)
	case "dash_setlevelchannel":
		showModal(modalData{
			CustomID: "modal_setlevelchannel", Title: "Set Level-up Channel",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "channel_id", Label: "Channel ID", Style: discordgo.TextInputShort, Required: true},
				}},
			},
		})

	case "dash_persona":
		persona := getGuildSystemPrompt(guildId)
		updateMsg(
			&discordgo.MessageEmbed{
				Color: 0xda70d6, Title: "🤖 Persona Bot",
				Fields: []*discordgo.MessageEmbedField{{Name: "System Prompt Sekarang", Value: truncate(persona, 1000)}},
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Edit Persona", CustomID: "dash_setpersona", Style: discordgo.PrimaryButton},
					discordgo.Button{Label: "Reset ke Default", CustomID: "dash_resetpersona", Style: discordgo.DangerButton},
				}},
				backRow(),
			},
		)
	case "dash_setpersona":
		currentPersona := getGuildSystemPrompt(guildId)
		if len(currentPersona) > 4000 {
			currentPersona = currentPersona[:4000]
		}
		showModal(modalData{
			CustomID: "modal_setpersona", Title: "Edit Persona Bot",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "prompt", Label: "System Prompt", Style: discordgo.TextInputParagraph, Required: true, Value: currentPersona},
				}},
			},
		})
	case "dash_resetpersona":
		db.Exec(`DELETE FROM kv WHERE guild_id=? AND key='system_prompt'`, guildId)
		ephemeralReply("✅ Persona direset ke default.")

	case "dash_model":
		currentModel := getGuildModel(guildId)
		updateMsg(
			&discordgo.MessageEmbed{
				Color: 0x00bfff, Title: "🧠 Model AI",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "Model Saat Ini", Value: fmt.Sprintf("`%s`", currentModel)},
					{Name: "Pilihan Model", Value: modelListText()},
				},
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Llama 3.3 70B", CustomID: "dash_model_llama70b", Style: discordgo.PrimaryButton},
					discordgo.Button{Label: "GPT OSS 120B", CustomID: "dash_model_gpt120b", Style: discordgo.SecondaryButton},
					discordgo.Button{Label: "GPT OSS 20B", CustomID: "dash_model_gpt20b", Style: discordgo.SecondaryButton},
					discordgo.Button{Label: "Qwen 32B", CustomID: "dash_model_qwen32b", Style: discordgo.SecondaryButton},
				}},
				backRow(),
			},
		)
	case "dash_model_llama70b":
		setGuildModel(guildId, "llama-3.3-70b-versatile")
		ephemeralReply("✅ Model diganti ke `llama-3.3-70b-versatile`!")
	case "dash_model_gpt120b":
		setGuildModel(guildId, "openai/gpt-oss-120b")
		ephemeralReply("✅ Model diganti ke `openai/gpt-oss-120b`!")
	case "dash_model_gpt20b":
		setGuildModel(guildId, "openai/gpt-oss-20b")
		ephemeralReply("✅ Model diganti ke `openai/gpt-oss-20b`!")
	case "dash_model_qwen32b":
		setGuildModel(guildId, "qwen/qwen3-32b")
		ephemeralReply("✅ Model diganti ke `qwen/qwen3-32b`!")

	case "dash_moderation":
		bannedWords := getBannedWords(guildId)
		bwVal := "Tidak ada"
		if len(bannedWords) > 0 {
			bwVal = strings.Join(bannedWords, ", ")
		}
		updateMsg(
			&discordgo.MessageEmbed{
				Color: 0xff0000, Title: "🛡️ Moderation",
				Fields: []*discordgo.MessageEmbedField{{Name: "🚫 Banned Words", Value: bwVal}},
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Tambah Banned Word", CustomID: "dash_addword", Style: discordgo.DangerButton},
					discordgo.Button{Label: "Hapus Banned Word", CustomID: "dash_removeword", Style: discordgo.SecondaryButton},
				}},
				backRow(),
			},
		)
	case "dash_addword":
		showModal(modalData{
			CustomID: "modal_addword", Title: "Tambah Banned Word",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "word", Label: "Kata", Style: discordgo.TextInputShort, Required: true},
				}},
			},
		})
	case "dash_removeword":
		showModal(modalData{
			CustomID: "modal_removeword", Title: "Hapus Banned Word",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "word", Label: "Kata yang dihapus", Style: discordgo.TextInputShort, Required: true},
				}},
			},
		})

	case "dash_status":
		updateMsg(
			&discordgo.MessageEmbed{
				Color: 0x00ff88, Title: "📊 Status Bot",
				Fields: []*discordgo.MessageEmbedField{
					{Name: "⏱️ Uptime", Value: getUptime(), Inline: true},
					{Name: "🌐 Servers", Value: fmt.Sprintf("%d", len(s.State.Guilds)), Inline: true},
					{Name: "📡 Status", Value: "🟢 Online", Inline: true},
				},
			},
			[]discordgo.MessageComponent{backRow()},
		)
	}
}

func handleModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildId := i.GuildID
	data := i.ModalSubmitData()

	getField := func(id string) string {
		for _, row := range data.Components {
			for _, comp := range row.(*discordgo.ActionsRow).Components {
				if ti, ok := comp.(*discordgo.TextInput); ok && ti.CustomID == id {
					return strings.TrimSpace(ti.Value)
				}
			}
		}
		return ""
	}

	reply := func(content string) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: content, Flags: discordgo.MessageFlagsEphemeral},
		})
	}

	switch data.CustomID {
	case "modal_setlog":
		chId := getField("channel_id")
		setGuildLogChannel(guildId, chId)
		reply(fmt.Sprintf("✅ Log channel diset ke <#%s>!", chId))
	case "modal_setwelcome":
		chId := getField("channel_id")
		setWelcomeChannel(guildId, chId)
		reply(fmt.Sprintf("✅ Welcome channel diset ke <#%s>!", chId))
	case "modal_setgoodbye":
		chId := getField("channel_id")
		setGoodbyeChannel(guildId, chId)
		reply(fmt.Sprintf("✅ Goodbye channel diset ke <#%s>!", chId))
	case "modal_setwelcomemsg":
		msg := getField("msg")
		setWelcomeMessage(guildId, msg)
		reply("✅ Pesan welcome diupdate!")
	case "modal_setgoodbyemsg":
		msg := getField("msg")
		setGoodbyeMessage(guildId, msg)
		reply("✅ Pesan goodbye diupdate!")
	case "modal_setautorole":
		roleId := getField("role_id")
		setAutoRole(guildId, roleId)
		reply(fmt.Sprintf("✅ Auto-role diset ke <@&%s>!", roleId))
	case "modal_setlevelchannel":
		chId := getField("channel_id")
		setLevelChannel(guildId, chId)
		reply(fmt.Sprintf("✅ Level-up channel diset ke <#%s>!", chId))
	case "modal_setpersona":
		prompt := getField("prompt")
		setGuildSystemPrompt(guildId, prompt)
		reply("✅ Persona diupdate!")
	case "modal_addword":
		word := strings.ToLower(getField("word"))
		addBannedWord(guildId, word)
		reply(fmt.Sprintf("✅ **%s** ditambah ke blacklist!", word))
	case "modal_removeword":
		word := strings.ToLower(getField("word"))
		removeBannedWord(guildId, word)
		reply(fmt.Sprintf("✅ **%s** dihapus dari blacklist!", word))
	}
}
