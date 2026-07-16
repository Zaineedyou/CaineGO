package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func onReady(s *discordgo.Session, r *discordgo.Ready) {
	s.LogLevel = discordgo.LogError
	fmt.Printf("✅ Bot online: %s\n", r.User.Username)
	s.UpdateCustomStatus("Property Of Caineedyou | Developed By Zaineedyou")

	slashCommandsMu.Lock()
	if !slashCommandsRegistered {
		commands := []*discordgo.ApplicationCommand{
			{Name: "info", Description: "Lihat info dan status bot Caine"},
			{Name: "dashboard", Description: "Buka dashboard pengaturan bot (Admin only)"},
			{Name: "help", Description: "Lihat semua command yang tersedia"},
			{Name: "healthcheck", Description: "Cek status semua komponen bot (Admin only)"},
		}
		for _, cmd := range commands {
			s.ApplicationCommandCreate(s.State.User.ID, "", cmd)
		}
		slashCommandsRegistered = true
		fmt.Println("✅ Slash commands registered")
	}
	slashCommandsMu.Unlock()
}

func onGuildMemberAdd(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	roleId := getAutoRole(m.GuildID)
	if roleId != "" {
		s.GuildMemberRoleAdd(m.GuildID, m.User.ID, roleId)
	}

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
			fmt.Println("⚠️ Panic recovered in onMessageCreate:", r)
		}
	}()

	if m.Author == nil || m.Author.Bot {
		return
	}

	if m.GuildID != "" {
		bridgeCh := getBridgeChannel(m.GuildID)
		if bridgeCh != "" && m.ChannelID == bridgeCh && strings.TrimSpace(m.Content) != "" {
			displayName := m.Author.Username
			if mem, err := s.GuildMember(m.GuildID, m.Author.ID); err == nil && mem.Nick != "" {
				displayName = mem.Nick
			}
			SendToMinecraft(m.GuildID, displayName, m.Content)
		}
	}

	if isRateLimited(m.Author.ID) {
		return
	}

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

	if m.GuildID != "" {
		handleXP(s, m.Message)
	}

	if m.GuildID != "" {
		if getAfkUser(m.Author.ID, m.GuildID) != nil {
			removeAfkUser(m.Author.ID, m.GuildID)
			s.ChannelMessageSendReply(m.ChannelID, fmt.Sprintf("✅ Welcome back <@%s>! AFK kamu dihapus.", m.Author.ID), m.Reference())
		}

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
			"`Caine sethistory <angka>` — set batas history chat\n" +
			"`/info` — info bot\n" +
			"`/dashboard` — buka dashboard (admin)\n\n" +
			"**Moderasi:** kick, ban, unban, timeout, untimeout, warn, warnings, clearwarn, clear, lock, unlock, slowmode, nick, role add/remove\n\n" +
			"**Admin:** addword, removeword, words, enable, disable, setlog, setwelcome, setgoodbye, setwelcomemsg, setgoodbyemsg, autorole, removeautorole, setlevelchannel, setpersona, setmodel, sethistory, setbridge, bridgestatus"
		s.ChannelMessageSendReply(m.ChannelID, helpText, m.Reference())
		return
	}

	if handleModeration(s, m.Message, userText) {
		return
	}

	var imageURL string
	for _, att := range m.Attachments {
		if strings.HasPrefix(att.ContentType, "image/") {
			imageURL = att.URL
			break
		}
	}

	if isAIRateLimited(m.Author.ID) {
		s.ChannelMessageSendReply(m.ChannelID, "⏱️ Slow down! Kamu terlalu banyak ngirim pesan ke Caine. Tunggu sebentar ya.", m.Reference())
		return
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
		fmt.Println("AI error:", aiErr)
		s.ChannelMessageSendReply(m.ChannelID, "❌ Ada error sayang, coba lagi ya 🙏", m.Reference())
		return
	}

	chunks := splitMessage(reply, 1900)
	for _, chunk := range chunks {
		s.ChannelMessageSendReply(m.ChannelID, chunk, m.Reference())
	}
	logChat(s, m.Message, userText, reply)
}
