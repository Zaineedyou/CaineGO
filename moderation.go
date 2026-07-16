package main

import (
	"fmt"
	"crypto/rand"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// channelMentionRegex menangkap channel mention dalam bentuk <#123456789>
// langsung dari raw text pesan. Dipakai sebagai fallback karena m.MentionChannels
// dari discordgo tidak selalu terisi untuk mention channel biasa.
var channelMentionRegex = regexp.MustCompile(`<#(\d+)>`)

// extractChannelMention mengambil channel ID pertama yang di-mention dalam pesan,
// coba dari m.MentionChannels dulu, fallback ke regex parsing dari raw content.
func extractChannelMention(m *discordgo.Message) string {
	if len(m.MentionChannels) > 0 {
		return m.MentionChannels[0].ID
	}
	matches := channelMentionRegex.FindStringSubmatch(m.Content)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

var modCmds = map[string]bool{
	"kick": true, "ban": true, "unban": true, "timeout": true, "untimeout": true,
	"warn": true, "warnings": true, "clearwarn": true, "clear": true,
	"lock": true, "unlock": true, "slowmode": true, "nick": true, "role": true,
	"report": true, "addword": true, "removeword": true, "words": true,
	"enable": true, "disable": true, "setlog": true, "setwelcome": true,
	"setgoodbye": true, "setwelcomemsg": true, "setgoodbyemsg": true,
	"autorole": true, "removeautorole": true, "setlevelchannel": true,
	"setpersona": true, "setmodel": true, "sethistory": true, "rank": true, "leaderboard": true,
	"afk": true, "afklist": true, "status": true, "summarize": true,
	"setbridge": true, "bridgestatus": true,
}

func handleModeration(s *discordgo.Session, m *discordgo.Message, userText string) bool {
	if m.GuildID == "" {
		return false
	}
	args := strings.Fields(userText)
	if len(args) == 0 {
		return false
	}
	cmd := strings.ToLower(args[0])
	if !modCmds[cmd] {
		return false
	}

	guildId := m.GuildID
	member, _ := s.GuildMember(guildId, m.Author.ID)

	hasPerm := func(perm int64) bool {
		if BOT_OWNER_ID != "" && m.Author.ID == BOT_OWNER_ID {
			return true
		}
		if member == nil {
			return false
		}
		// Owner server selalu bypass
		g, err := s.State.Guild(guildId)
		if err == nil && g.OwnerID == m.Author.ID {
			return true
		}
		perms, err := s.State.UserChannelPermissions(m.Author.ID, m.ChannelID)
		if err != nil {
			return false
		}
		return perms&perm != 0
	}

	getMention := func() string {
		for _, u := range m.Mentions {
			return u.ID
		}
		return ""
	}

	replyMsg := func(content string) {
		s.ChannelMessageSendReply(m.ChannelID, content, m.Reference())
	}

	// STATUS
	if cmd == "status" {
		replyMsg(fmt.Sprintf("📊 **Status Bot**\n⏱️ Uptime: %s\n🌐 Servers: %d", getUptime(), len(s.State.Guilds)))
		return true
	}

	// AFK
	if cmd == "afk" {
		reason := "AFK"
		if len(args) > 1 {
			reason = strings.Join(args[1:], " ")
		}
		setAfkUser(m.Author.ID, guildId, reason)
		replyMsg(fmt.Sprintf("💤 **%s** sekarang AFK: *%s*", member.Nick, reason))
		return true
	}
	if cmd == "afklist" {
		afkData := getAllAfk(guildId)
		if len(afkData) == 0 {
			replyMsg("✅ Ga ada yang AFK sekarang.")
			return true
		}
		var lines []string
		for uid, d := range afkData {
			elapsed := formatDuration(nowMs() - d.Time)
			lines = append(lines, fmt.Sprintf("<@%s> — *%s* (%s lalu)", uid, d.Reason, elapsed))
		}
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Color:       0x778899,
			Title:       "💤 Daftar AFK",
			Description: strings.Join(lines, "\n"),
		})
		return true
	}

	// RANK
	if cmd == "rank" {
		targetId := getMention()
		targetName := m.Author.Username
		if targetId == "" {
			targetId = m.Author.ID
		} else {
			tm, _ := s.GuildMember(guildId, targetId)
			if tm != nil {
				targetName = tm.User.Username
			}
		}
		data := getUserXP(targetId, guildId)
		needed := xpToNextLevel(data.Level)
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Color: 0x7289da,
			Title: fmt.Sprintf("📊 Rank — %s", targetName),
			Fields: []*discordgo.MessageEmbedField{
				{Name: "Level", Value: strconv.Itoa(data.Level), Inline: true},
				{Name: "XP", Value: fmt.Sprintf("%d / %d", data.XP, needed), Inline: true},
			},
		})
		return true
	}

	// LEADERBOARD
	if cmd == "leaderboard" {
		rows := getAllXP(guildId)
		if len(rows) == 0 {
			replyMsg("📭 Belum ada data XP.")
			return true
		}
		var lines []string
		for i, r := range rows {
			lines = append(lines, fmt.Sprintf("**%d.** <@%s> — Level **%d** (%d XP)", i+1, r.UserID, r.Data.Level, r.Data.XP))
		}
		s.ChannelMessageSendEmbed(m.ChannelID, &discordgo.MessageEmbed{
			Color:       0xffd700,
			Title:       "🏆 Leaderboard XP",
			Description: strings.Join(lines, "\n"),
		})
		return true
	}

	// SETLEVELCHANNEL
	if cmd == "setlevelchannel" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(m.MentionChannels) == 0 {
			replyMsg("❌ Mention channel-nya.")
			return true
		}
		chId := m.MentionChannels[0].ID
		setLevelChannel(guildId, chId)
		replyMsg(fmt.Sprintf("✅ Notif level up akan dikirim ke <#%s>!", chId))
		return true
	}

	// SETPERSONA
	if cmd == "setpersona" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(args) < 2 {
			replyMsg("❌ Masukin persona-nya.")
			return true
		}
		persona := strings.Join(args[1:], " ")
		setGuildSystemPrompt(guildId, persona)
		replyMsg("✅ Persona Caine di server ini udah diupdate!")
		return true
	}

	// SETMODEL
	if cmd == "setmodel" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(args) < 2 {
			currentModel := getGuildModel(guildId)
			replyMsg(fmt.Sprintf("**Model saat ini:** `%s`\n\n**Pilihan model:**\n%s\n\nContoh: `Caine setmodel llama70b`", currentModel, modelListText()))
			return true
		}
		alias := strings.ToLower(args[1])
		modelId, ok := availableModels[alias]
		if !ok {
			replyMsg(fmt.Sprintf("❌ Model ga valid. Pilihan:\n%s", modelListText()))
			return true
		}
		setGuildModel(guildId, modelId)
		replyMsg(fmt.Sprintf("✅ Model diganti ke `%s` (`%s`)!", alias, modelId))
		return true
	}

	if cmd == "sethistory" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(args) < 2 {
			current := getGuildMaxHistory(guildId)
			replyMsg(fmt.Sprintf("📜 History limit sekarang: **%d** pesan. Untuk mengubah: `%s sethistory <angka>`", current, BOT_PREFIX))
			return true
		}
		limit, err := strconv.Atoi(args[1])
		if err != nil || limit < 5 || limit > 100 {
			replyMsg("❌ Masukkan angka antara 5-100.")
			return true
		}
		setGuildMaxHistory(guildId, limit)
		replyMsg(fmt.Sprintf("✅ History limit diset ke **%d** pesan.", limit))
		return true
	}

	// AUTOROLE
	if cmd == "autorole" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(m.MentionRoles) == 0 {
			replyMsg("❌ Mention role-nya. Contoh: `Caine autorole @Member`")
			return true
		}
		roleId := m.MentionRoles[0]
		setAutoRole(guildId, roleId)
		replyMsg(fmt.Sprintf("✅ Auto-role diset ke <@&%s>!", roleId))
		return true
	}
	if cmd == "removeautorole" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		removeAutoRole(guildId)
		replyMsg("✅ Auto-role dihapus.")
		return true
	}

	// WELCOME/GOODBYE
	if cmd == "setwelcome" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(m.MentionChannels) == 0 {
			replyMsg("❌ Mention channel-nya.")
			return true
		}
		setWelcomeChannel(guildId, m.MentionChannels[0].ID)
		replyMsg(fmt.Sprintf("✅ Channel welcome diset ke <#%s>!", m.MentionChannels[0].ID))
		return true
	}
	if cmd == "setgoodbye" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(m.MentionChannels) == 0 {
			replyMsg("❌ Mention channel-nya.")
			return true
		}
		setGoodbyeChannel(guildId, m.MentionChannels[0].ID)
		replyMsg(fmt.Sprintf("✅ Channel goodbye diset ke <#%s>!", m.MentionChannels[0].ID))
		return true
	}
	if cmd == "setwelcomemsg" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(args) < 2 {
			replyMsg("❌ Masukin pesannya. Variabel: `{user}` `{username}` `{server}` `{count}`")
			return true
		}
		setWelcomeMessage(guildId, strings.Join(args[1:], " "))
		replyMsg("✅ Pesan welcome diupdate!")
		return true
	}
	if cmd == "setgoodbyemsg" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(args) < 2 {
			replyMsg("❌ Masukin pesannya.")
			return true
		}
		setGoodbyeMessage(guildId, strings.Join(args[1:], " "))
		replyMsg("✅ Pesan goodbye diupdate!")
		return true
	}

	// SETLOG
	if cmd == "setlog" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(m.MentionChannels) == 0 {
			replyMsg("❌ Mention channel log-nya.")
			return true
		}
		setGuildLogChannel(guildId, m.MentionChannels[0].ID)
		replyMsg(fmt.Sprintf("✅ Channel log diset ke <#%s>!", m.MentionChannels[0].ID))
		return true
	}

	// REPORT
	if cmd == "report" {
		targetId := getMention()
		if targetId == "" {
			replyMsg("❌ Mention siapa yang mau di-report.")
			return true
		}
		reason := ""
		if len(args) > 2 {
			reason = strings.Join(args[2:], " ")
		}
		target, _ := s.User(targetId)
		targetTag := targetId
		if target != nil {
			targetTag = target.Username
		}
		logReport(s, m.Author.Username, targetTag, reason, m.ChannelID, guildId)
		replyMsg("✅ Report dikirim ke admin sayang!")
		return true
	}

	// KICK
	if cmd == "kick" {
		if !hasPerm(discordgo.PermissionKickMembers) {
			replyMsg("❌ No permission.")
			return true
		}
		targetId := getMention()
		if targetId == "" {
			replyMsg("❌ Mention siapa.")
			return true
		}
		reason := orDefault(strings.Join(args[2:], " "), "Tidak ada alasan")
		if err := s.GuildMemberDeleteWithReason(guildId, targetId, reason); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal kick: %v", err))
			return true
		}
		target, _ := s.User(targetId)
		tag := targetId
		if target != nil {
			tag = target.Username
		}
		logMod(s, "Kick", m.Author.Username, tag, reason, guildId)
		replyMsg(fmt.Sprintf("✅ **%s** di-kick.", tag))
		return true
	}

	// BAN
	if cmd == "ban" {
		if !hasPerm(discordgo.PermissionBanMembers) {
			replyMsg("❌ No permission.")
			return true
		}
		targetId := getMention()
		if targetId == "" {
			replyMsg("❌ Mention siapa.")
			return true
		}
		reason := orDefault(strings.Join(args[2:], " "), "Tidak ada alasan")
		if err := s.GuildBanCreateWithReason(guildId, targetId, reason, 0); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal ban: %v", err))
			return true
		}
		target, _ := s.User(targetId)
		tag := targetId
		if target != nil {
			tag = target.Username
		}
		logMod(s, "Ban", m.Author.Username, tag, reason, guildId)
		replyMsg(fmt.Sprintf("✅ **%s** di-ban.", tag))
		return true
	}

	// UNBAN
	if cmd == "unban" {
		if !hasPerm(discordgo.PermissionBanMembers) {
			replyMsg("❌ No permission.")
			return true
		}
		if len(args) < 2 {
			replyMsg("❌ Masukin user ID.")
			return true
		}
		userId := args[1]
		if err := s.GuildBanDelete(guildId, userId); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal unban: %v", err))
			return true
		}
		logMod(s, "Unban", m.Author.Username, userId, "-", guildId)
		replyMsg(fmt.Sprintf("✅ **%s** di-unban.", userId))
		return true
	}

	// TIMEOUT
	if cmd == "timeout" {
		if !hasPerm(discordgo.PermissionModerateMembers) {
			replyMsg("❌ No permission.")
			return true
		}
		targetId := getMention()
		if targetId == "" || len(args) < 3 {
			replyMsg("❌ `Caine timeout @user <durasi_menit> [alasan]`")
			return true
		}
		minutes, err := strconv.Atoi(args[2])
		if err != nil {
			replyMsg("❌ Durasi harus angka (menit).")
			return true
		}
		reason := ""
		if len(args) > 3 {
			reason = strings.Join(args[3:], " ")
		}
		until := time.Now().Add(time.Duration(minutes) * time.Minute)
		if err := s.GuildMemberTimeout(guildId, targetId, &until); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal timeout: %v", err))
			return true
		}
		target, _ := s.User(targetId)
		tag := targetId
		if target != nil {
			tag = target.Username
		}
		logMod(s, "Timeout", m.Author.Username, tag, orDefault(reason, "Tidak ada alasan"), guildId)
		replyMsg(fmt.Sprintf("✅ **%s** di-timeout selama %d menit.", tag, minutes))
		return true
	}

	// UNTIMEOUT
	if cmd == "untimeout" {
		if !hasPerm(discordgo.PermissionModerateMembers) {
			replyMsg("❌ No permission.")
			return true
		}
		targetId := getMention()
		if targetId == "" {
			replyMsg("❌ Mention siapa.")
			return true
		}
		if err := s.GuildMemberTimeout(guildId, targetId, nil); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal untimeout: %v", err))
			return true
		}
		target, _ := s.User(targetId)
		tag := targetId
		if target != nil {
			tag = target.Username
		}
		replyMsg(fmt.Sprintf("✅ Timeout **%s** dihapus.", tag))
		return true
	}

	// WARN
	if cmd == "warn" {
		if !hasPerm(discordgo.PermissionKickMembers) {
			replyMsg("❌ No permission.")
			return true
		}
		targetId := getMention()
		if targetId == "" {
			replyMsg("❌ Mention siapa.")
			return true
		}
		reason := orDefault(strings.Join(args[2:], " "), "Tidak ada alasan")
		count := addWarning(targetId, guildId, reason)
		target, _ := s.User(targetId)
		tag := targetId
		if target != nil {
			tag = target.Username
		}
		logMod(s, "Warn", m.Author.Username, tag, reason, guildId)
		replyMsg(fmt.Sprintf("⚠️ **%s** diwarn. Total warning: **%d**", tag, count))
		return true
	}

	// WARNINGS
	if cmd == "warnings" {
		targetId := getMention()
		if targetId == "" {
			targetId = m.Author.ID
		}
		warns := getWarnings(targetId, guildId)
		if len(warns) == 0 {
			replyMsg(fmt.Sprintf("✅ <@%s> ga punya warning.", targetId))
			return true
		}
		var lines []string
		for i, w := range warns {
			lines = append(lines, fmt.Sprintf("**%d.** %s — `%s`", i+1, w.Reason, w.Time))
		}
		replyMsg(fmt.Sprintf("⚠️ **Warning <@%s>:**\n%s", targetId, strings.Join(lines, "\n")))
		return true
	}

	// CLEARWARN
	if cmd == "clearwarn" {
		if !hasPerm(discordgo.PermissionKickMembers) {
			replyMsg("❌ No permission.")
			return true
		}
		targetId := getMention()
		if targetId == "" {
			replyMsg("❌ Mention siapa.")
			return true
		}
		clearWarnings(targetId, guildId)
		replyMsg(fmt.Sprintf("✅ Warning <@%s> dihapus.", targetId))
		return true
	}

	// CLEAR
	if cmd == "clear" {
		if !hasPerm(discordgo.PermissionManageMessages) {
			replyMsg("❌ No permission.")
			return true
		}
		amount := 10
		if len(args) > 1 {
			if n, err := strconv.Atoi(args[1]); err == nil {
				amount = n
			}
		}
		messages, err := s.ChannelMessages(m.ChannelID, amount+1, "", "", "")
		if err != nil {
			replyMsg("❌ Gagal ambil pesan.")
			return true
		}
		var ids []string
		for _, msg := range messages {
			ids = append(ids, msg.ID)
		}
		if err := s.ChannelMessagesBulkDelete(m.ChannelID, ids); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal hapus pesan: %v", err))
			return true
		}
		replyMsg(fmt.Sprintf("🗑️ %d pesan dihapus.", len(ids)-1))
		return true
	}

	// LOCK
	if cmd == "lock" {
		if !hasPerm(discordgo.PermissionManageChannels) {
			replyMsg("❌ No permission.")
			return true
		}
		deny := int64(discordgo.PermissionSendMessages)
		if err := s.ChannelPermissionSet(m.ChannelID, guildId, discordgo.PermissionOverwriteTypeRole, 0, deny); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal lock: %v", err))
			return true
		}
		replyMsg("🔒 Channel dikunci.")
		return true
	}

	// UNLOCK
	if cmd == "unlock" {
		if !hasPerm(discordgo.PermissionManageChannels) {
			replyMsg("❌ No permission.")
			return true
		}
		if err := s.ChannelPermissionDelete(m.ChannelID, guildId); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal unlock: %v", err))
			return true
		}
		replyMsg("🔓 Channel dibuka.")
		return true
	}

	// SLOWMODE
	if cmd == "slowmode" {
		if !hasPerm(discordgo.PermissionManageChannels) {
			replyMsg("❌ No permission.")
			return true
		}
		seconds := 0
		if len(args) > 1 {
			if n, err := strconv.Atoi(args[1]); err == nil {
				seconds = n
			}
		}
		if _, err := s.ChannelEditComplex(m.ChannelID, &discordgo.ChannelEdit{RateLimitPerUser: &seconds}); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal set slowmode: %v", err))
			return true
		}
		if seconds == 0 {
			replyMsg("✅ Slowmode dimatiin.")
		} else {
			replyMsg(fmt.Sprintf("✅ Slowmode diset ke %d detik.", seconds))
		}
		return true
	}

	// NICK
	if cmd == "nick" {
		if !hasPerm(discordgo.PermissionManageNicknames) {
			replyMsg("❌ No permission.")
			return true
		}
		targetId := getMention()
		if targetId == "" || len(args) < 3 {
			replyMsg("❌ `Caine nick @user <nickname baru>`")
			return true
		}
		newNick := strings.Join(args[2:], " ")
		if err := s.GuildMemberNickname(guildId, targetId, newNick); err != nil {
			replyMsg(fmt.Sprintf("❌ Gagal ganti nickname: %v", err))
			return true
		}
		replyMsg(fmt.Sprintf("✅ Nickname <@%s> diubah ke **%s**.", targetId, newNick))
		return true
	}

	// ROLE ADD/REMOVE
	if cmd == "role" {
		if !hasPerm(discordgo.PermissionManageRoles) {
			replyMsg("❌ No permission.")
			return true
		}
		if len(args) < 3 || len(m.Mentions) == 0 || len(m.MentionRoles) == 0 {
			replyMsg("❌ `Caine role add/remove @user @role`")
			return true
		}
		action := strings.ToLower(args[1])
		targetId := m.Mentions[0].ID
		roleId := m.MentionRoles[0]
		if action == "add" {
			if err := s.GuildMemberRoleAdd(guildId, targetId, roleId); err != nil {
				replyMsg(fmt.Sprintf("❌ Gagal tambah role: %v", err))
				return true
			}
			replyMsg(fmt.Sprintf("✅ Role <@&%s> ditambah ke <@%s>.", roleId, targetId))
		} else if action == "remove" {
			if err := s.GuildMemberRoleRemove(guildId, targetId, roleId); err != nil {
				replyMsg(fmt.Sprintf("❌ Gagal hapus role: %v", err))
				return true
			}
			replyMsg(fmt.Sprintf("✅ Role <@&%s> dihapus dari <@%s>.", roleId, targetId))
		} else {
			replyMsg("❌ Pakai `add` atau `remove`.")
		}
		return true
	}

	// ADDWORD
	if cmd == "addword" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(args) < 2 {
			replyMsg("❌ `Caine addword <kata>`")
			return true
		}
		word := strings.ToLower(args[1])
		addBannedWord(guildId, word)
		replyMsg(fmt.Sprintf("✅ **%s** ditambah ke blacklist!", word))
		return true
	}

	// REMOVEWORD
	if cmd == "removeword" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		if len(args) < 2 {
			replyMsg("❌ `Caine removeword <kata>`")
			return true
		}
		word := strings.ToLower(args[1])
		removeBannedWord(guildId, word)
		replyMsg(fmt.Sprintf("✅ **%s** dihapus dari blacklist!", word))
		return true
	}

	// WORDS
	if cmd == "words" {
		words := getBannedWords(guildId)
		if len(words) == 0 {
			replyMsg("📭 Belum ada kata terlarang.")
			return true
		}
		replyMsg(fmt.Sprintf("🚫 **Kata terlarang:**\n%s", strings.Join(words, ", ")))
		return true
	}

	// ENABLE/DISABLE
	if cmd == "enable" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		enableChannel(guildId, m.ChannelID)
		replyMsg("✅ Bot diaktifkan di channel ini.")
		return true
	}
	if cmd == "disable" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		disableChannel(guildId, m.ChannelID)
		replyMsg("✅ Bot dinonaktifkan di channel ini.")
		return true
	}

	// SUMMARIZE
	if cmd == "summarize" {
		limit := 30
		if len(args) > 1 {
			if n, err := strconv.Atoi(args[1]); err == nil {
				limit = n
			}
		}
		go summarizeChannel(s, m, limit)
		return true
	}

	// SETBRIDGE
	if cmd == "setbridge" {
		if !hasPerm(discordgo.PermissionAdministrator) {
			replyMsg("❌ Khusus admin.")
			return true
		}
		chId := extractChannelMention(m)
		if chId == "" {
			replyMsg("❌ Mention channel-nya. Contoh: `Caine setbridge #minecraft-chat`")
			return true
		}
		setBridgeChannel(guildId, chId)
		replyMsg(fmt.Sprintf("✅ Chat Minecraft akan di-bridge ke <#%s>! Pastikan plugin Minecraft-nya sudah connect ya.", chId))
		return true
	}

	// BRIDGESTATUS
	if cmd == "bridgestatus" {
		chId := getBridgeChannel(guildId)
		if chId == "" {
			replyMsg("⚠️ Bridge channel belum di-set. Pakai `Caine setbridge #channel` dulu.")
			return true
		}
		if isBridgeConnected(guildId) {
			replyMsg(fmt.Sprintf("✅ Bridge aktif — server Minecraft terhubung. Channel: <#%s>", chId))
		} else {
			replyMsg(fmt.Sprintf("🔴 Bridge channel: <#%s>, tapi server Minecraft belum/tidak terhubung.", chId))
		}
		return true
	}

	return false
}

func summarizeChannel(s *discordgo.Session, m *discordgo.Message, limit int) {
	msgs, err := s.ChannelMessages(m.ChannelID, limit, "", "", "")
	if err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "❌ Gagal ambil pesan.", m.Reference())
		return
	}
	var lines []string
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		if msg.Author.Bot || msg.Content == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", msg.Author.Username, msg.Content))
	}
	if len(lines) == 0 {
		s.ChannelMessageSendReply(m.ChannelID, "📭 Ga ada pesan yang bisa dirangkum.", m.Reference())
		return
	}
	prompt := fmt.Sprintf("Rangkum percakapan berikut dalam beberapa poin penting:\n\n%s", strings.Join(lines, "\n"))
	reply, err := askGroq("summarize-"+m.ChannelID, prompt, "System", m.GuildID)
	if err != nil {
		s.ChannelMessageSendReply(m.ChannelID, "❌ Gagal merangkum.", m.Reference())
		return
	}
	chunks := splitMessage(reply, 1900)
	for _, chunk := range chunks {
		s.ChannelMessageSendReply(m.ChannelID, chunk, m.Reference())
	}
}

func handleXP(s *discordgo.Session, m *discordgo.Message) {
	if m.GuildID == "" || m.Author.Bot {
		return
	}
	now := nowMs()
	data := getUserXP(m.Author.ID, m.GuildID)
	if now-data.LastMessage < 60000 {
		return
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(10))
	xpGain := int(n.Int64()) + 5
	data.XP += xpGain
	data.LastMessage = now
	needed := xpToNextLevel(data.Level)
	if data.XP >= needed {
		data.XP -= needed
		data.Level++
		setUserXP(m.Author.ID, m.GuildID, data)
		levelChId := getLevelChannel(m.GuildID)
		targetCh := m.ChannelID
		if levelChId != "" {
			targetCh = levelChId
		}
		s.ChannelMessageSendEmbed(targetCh, &discordgo.MessageEmbed{
			Color:       0xffd700,
			Title:       "🎉 Level Up!",
			Description: fmt.Sprintf("Selamat <@%s>! Kamu naik ke **Level %d**! 🚀", m.Author.ID, data.Level),
		})
	} else {
		setUserXP(m.Author.ID, m.GuildID, data)
	}
}
