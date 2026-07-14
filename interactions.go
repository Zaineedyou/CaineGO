package main

import (
	"github.com/bwmarrin/discordgo"
)

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

	case "healthcheck":
		handleHealthCheck(s, i)

	case "dashboard":
		if !isBotManager(i) {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Khusus admin atau bot owner.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}
		respondDashboardMain(s, i)
	}
}
