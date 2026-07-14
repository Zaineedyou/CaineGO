package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type modalData struct {
	CustomID   string
	Title      string
	Components []discordgo.MessageComponent
}

func handleButtonInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !isBotManager(i) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "❌ Khusus admin atau bot owner.", Flags: discordgo.MessageFlagsEphemeral},
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
			rows, _ := db.Query(`SELECT channel_id FROM disabled_channels WHERE guild_id=$1`, guildId)
			if rows == nil {
				return nil
			}
			defer rows.Close()
			var ids []string
			for rows.Next() {
				var id string
				rows.Scan(&id)
				ids = append(ids, id)
			}
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
		kvDel(guildId, "system_prompt")
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
					{Name: "📜 History Limit", Value: fmt.Sprintf("%d messages", getGuildMaxHistory(guildId)), Inline: true},
				},
			},
			[]discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.Button{Label: "Set History Limit", CustomID: "dash_sethistory", Style: discordgo.PrimaryButton},
				}},
				backRow(),
			},
		)

	case "dash_sethistory":
		showModal(modalData{
			CustomID: "modal_sethistory", Title: "Set History Limit",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{Components: []discordgo.MessageComponent{
					discordgo.TextInput{CustomID: "limit", Label: "Limit History (min 5, max 50)", Style: discordgo.TextInputShort, Required: true, Placeholder: "Contoh: 15", Value: strconv.Itoa(getGuildMaxHistory(guildId))},
				}},
			},
		})
	}
}

func backRow() discordgo.ActionsRow {
	return discordgo.ActionsRow{Components: []discordgo.MessageComponent{
		discordgo.Button{Label: "⬅️ Kembali", CustomID: "dash_back", Style: discordgo.SecondaryButton},
	}}
}
