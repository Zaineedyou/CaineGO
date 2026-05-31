package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// sendGuildLog sends an embed to the configured log channel for the guild.
func sendGuildLog(s *discordgo.Session, guildId string, embed *discordgo.MessageEmbed) {
	if guildId == "" {
		return
	}
	channelId := getGuildLogChannel(guildId)
	if channelId == "" {
		return
	}
	s.ChannelMessageSendEmbed(channelId, embed)
}

// logChat logs an AI interaction to the guild log channel. DMs are never logged.
func logChat(s *discordgo.Session, m *discordgo.Message, userText, reply string) {
	if m.GuildID == "" {
		return // DMs are not logged
	}
	embed := &discordgo.MessageEmbed{
		Color: 0x5865f2,
		Title: "💬 Chat Log",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "User", Value: m.Author.Username, Inline: true},
			{Name: "Channel", Value: fmt.Sprintf("<#%s>", m.ChannelID), Inline: true},
			{Name: "Pertanyaan", Value: truncate(userText, 1000)},
			{Name: "Jawaban", Value: truncate(reply, 1000)},
		},
	}
	sendGuildLog(s, m.GuildID, embed)
}

func logMod(s *discordgo.Session, action, moderatorTag, targetTag, reason, guildId string) {
	embed := &discordgo.MessageEmbed{
		Color: 0xff0000,
		Title: fmt.Sprintf("🔨 Moderasi — %s", action),
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Moderator", Value: moderatorTag, Inline: true},
			{Name: "Target", Value: targetTag, Inline: true},
			{Name: "Alasan", Value: orDefault(reason, "Tidak ada alasan")},
		},
	}
	sendGuildLog(s, guildId, embed)
}

func logReport(s *discordgo.Session, reporterTag, targetTag, reason, channelId, guildId string) {
	logChannelId := getGuildLogChannel(guildId)
	if logChannelId == "" {
		return
	}
	embed := &discordgo.MessageEmbed{
		Color: 0xff6600,
		Title: "🚨 Report Masuk",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "Reporter", Value: reporterTag, Inline: true},
			{Name: "Target", Value: targetTag, Inline: true},
			{Name: "Alasan", Value: orDefault(reason, "Tidak ada alasan")},
			{Name: "Channel", Value: fmt.Sprintf("<#%s>", channelId)},
		},
	}
	g, err := s.State.Guild(guildId)
	if err != nil {
		s.ChannelMessageSendEmbed(logChannelId, embed)
		return
	}
	var adminMentions string
	for _, m := range g.Members {
		perms, _ := s.State.UserChannelPermissions(m.User.ID, logChannelId)
		if perms&discordgo.PermissionAdministrator != 0 && !m.User.Bot {
			adminMentions += fmt.Sprintf("<@%s> ", m.User.ID)
		}
	}
	s.ChannelMessageSendComplex(logChannelId, &discordgo.MessageSend{
		Content: "📢 **Report baru!** " + adminMentions,
		Embeds:  []*discordgo.MessageEmbed{embed},
	})
}

func logAutomod(s *discordgo.Session, m *discordgo.Message, word string) {
	embed := &discordgo.MessageEmbed{
		Color: 0xffaa00,
		Title: "🤖 Automod",
		Fields: []*discordgo.MessageEmbedField{
			{Name: "User", Value: m.Author.Username, Inline: true},
			{Name: "Channel", Value: fmt.Sprintf("<#%s>", m.ChannelID), Inline: true},
			{Name: "Kata Terlarang", Value: fmt.Sprintf("||%s||", word)},
			{Name: "Pesan", Value: truncate(m.Content, 500)},
		},
	}
	sendGuildLog(s, m.GuildID, embed)
}

func truncate(s string, max int) string {
	if s == "" {
		return "(kosong)"
	}
	if len(s) > max {
		return s[:max]
	}
	return s
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
