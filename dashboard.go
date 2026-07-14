package main

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

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


