package main

import (
	"github.com/bwmarrin/discordgo"
)

func isBotManager(i *discordgo.InteractionCreate) bool {
	if BOT_OWNER_ID != "" {
		var userID string
		if i.Member != nil {
			userID = i.Member.User.ID
		} else if i.User != nil {
			userID = i.User.ID
		}
		if userID == BOT_OWNER_ID {
			return true
		}
	}
	if i.Member == nil {
		return false
	}
	return i.Member.Permissions&discordgo.PermissionAdministrator != 0
}
