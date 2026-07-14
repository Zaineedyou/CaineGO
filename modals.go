package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
)

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
	case "modal_sethistory":
		limitStr := getField("limit")
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 5 || limit > 100 {
			reply("❌ Masukkan angka antara 5-100.")
			return
		}
		setGuildMaxHistory(guildId, limit)
		reply(fmt.Sprintf("✅ History limit diset ke **%d** pesan.", limit))
	}
}
