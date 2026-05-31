package main

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
)

// handleHealthCheck handles the /healthcheck slash command.
// Runs component health checks and reports results. Admin only.

func handleHealthCheck(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Admin only
	if i.Member == nil || i.Member.Permissions&discordgo.PermissionAdministrator == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "❌ Khusus admin.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	// Defer the response since health checks may take a moment
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	})

	results := runHealthChecks(s, i.GuildID)

	var fields []*discordgo.MessageEmbedField
	allOK := true
	for _, r := range results {
		icon := "✅"
		if !r.ok {
			icon = "❌"
			allOK = false
		}
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("%s %s", icon, r.name),
			Value:  r.detail,
			Inline: true,
		})
	}

	color := 0x00ff88
	title := "✅ Semua sistem normal"
	if !allOK {
		color = 0xff4444
		title = "⚠️ Ada komponen bermasalah"
	}

	s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			{
				Color:  color,
				Title:  title,
				Fields: fields,
				Footer: &discordgo.MessageEmbedFooter{
					Text: fmt.Sprintf("Checked at %s • Uptime: %s", time.Now().Format("15:04:05"), getUptime()),
				},
			},
		},
	})
}

type checkResult struct {
	name   string
	ok     bool
	detail string
}

func runHealthChecks(s *discordgo.Session, guildId string) []checkResult {
	var results []checkResult

	// SQLite read/write
	results = append(results, checkDB(guildId))

	// In-memory cache consistency
	results = append(results, checkCache(guildId))

	// Groq API connectivity
	results = append(results, checkGroq())

	// Bot permissions
	results = append(results, checkBotPerms(s, guildId))

	// Discord latency
	results = append(results, checkLatency(s))

	return results
}

func checkDB(guildId string) checkResult {
	testKey := "__healthcheck_test__"
	testVal := fmt.Sprintf("ping_%d", time.Now().UnixMilli())

	kvSet(guildId, testKey, testVal)
	got := kvGet(guildId, testKey)
	kvDel(guildId, testKey)

	if got != testVal {
		return checkResult{"Database (SQLite)", false, fmt.Sprintf("Write OK tapi read dapat `%s`", got)}
	}
	return checkResult{"Database (SQLite)", true, "Read/write OK"}
}

func checkCache(guildId string) checkResult {
	testKey := "__cache_test__"
	testVal := "cache_ping"

	// kvSet also populates the cache
	kvSet(guildId, testKey, testVal)

	// Verify cache was populated
	if v, ok := cache.get(guildId, testKey); !ok || v != testVal {
		kvDel(guildId, testKey)
		return checkResult{"In-memory Cache", false, "Cache miss setelah kvSet"}
	}

	kvDel(guildId, testKey)

	// Verify cache is invalidated after delete
	if _, ok := cache.get(guildId, testKey); ok {
		return checkResult{"In-memory Cache", false, "Cache tidak di-invalidate setelah kvDel"}
	}

	return checkResult{"In-memory Cache", true, "Hit/invalidate OK"}
}

func checkGroq() checkResult {
	start := time.Now()
	_, err := doGroqRequest(GroqRequest{
		Model: DEFAULT_MODEL,
		Messages: []GroqMessage{
			{Role: "user", Content: "ping"},
		},
		MaxTokens: 5,
	})
	elapsed := time.Since(start).Milliseconds()

	if err != nil {
		return checkResult{"Groq API", false, fmt.Sprintf("Error: %v", err)}
	}
	return checkResult{"Groq API", true, fmt.Sprintf("Response dalam %dms", elapsed)}
}

func checkBotPerms(s *discordgo.Session, guildId string) checkResult {
	if guildId == "" {
		return checkResult{"Bot Permissions", false, "Tidak bisa cek di DM"}
	}

	botID := s.State.User.ID
	guild, err := s.State.Guild(guildId)
	if err != nil {
		return checkResult{"Bot Permissions", false, fmt.Sprintf("Gagal ambil guild: %v", err)}
	}

	// Find bot member in guild
	member, err := s.GuildMember(guildId, botID)
	if err != nil {
		return checkResult{"Bot Permissions", false, fmt.Sprintf("Gagal ambil member: %v", err)}
	}

	// Calculate effective permissions from all bot roles
	var perms int64
	for _, roleID := range member.Roles {
		for _, role := range guild.Roles {
			if role.ID == roleID {
				perms |= role.Permissions
			}
		}
	}

	needed := map[string]int64{
		"Send Messages":   discordgo.PermissionSendMessages,
		"Embed Links":     discordgo.PermissionEmbedLinks,
		"Read History":    discordgo.PermissionReadMessageHistory,
		"Manage Messages": discordgo.PermissionManageMessages,
	}

	var missing []string
	for name, perm := range needed {
		if perms&perm == 0 && perms&discordgo.PermissionAdministrator == 0 {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return checkResult{"Bot Permissions", false, fmt.Sprintf("Kurang: %v", missing)}
	}
	return checkResult{"Bot Permissions", true, "Semua permission OK"}
}

func checkLatency(s *discordgo.Session) checkResult {
	latency := s.HeartbeatLatency().Milliseconds()
	if latency <= 0 {
		return checkResult{"Discord Latency", false, "Tidak bisa baca latency"}
	}
	if latency > 500 {
		return checkResult{"Discord Latency", false, fmt.Sprintf("%dms (tinggi)", latency)}
	}
	return checkResult{"Discord Latency", true, fmt.Sprintf("%dms", latency)}
}
