package client

import (
	"fmt"
	"strings"

	"git.kirsle.net/apps/barertc/pkg/messages"
)

// SetUserVariables prepares RiveScript user variables before handling a message.
//
// Example: it will set the user's `name` to their WhoList nickname, and other such flags.
//
// User variables set include:
//
// * name (nickname or username)
// * isAdmin (boolean operator status)
// * messageID (BareRTC MessageID)
//
// Global variables (`<env>`) are also set here:
//
// * numUsersOnline (int): length of who list
func (h *BotHandlers) SetUserVariables(msg messages.Message) {
	var (
		username = msg.Username
	)

	// Defaults
	var vars = map[string]string{
		"name":      username,
		"isAdmin":   "false",
		"messageID": fmt.Sprint(msg.MessageID),
		"channelID": msg.Channel,
	}

	// Set global variables.
	h.rs.SetGlobal("numUsersOnline", fmt.Sprint(len(h.whoList)))

	// Are they on the Who List?
	if who, ok := h.GetUser(username); ok {
		if who.Nickname != "" {
			vars["name"] = who.Nickname
		}

		if who.Operator {
			vars["isAdmin"] = "true"
		}
	}

	if len(vars) > 0 {
		h.rs.SetUservars(username, vars)
	}
}

// GetUser looks up a username from the Who List.
func (h *BotHandlers) GetUser(username string) (*messages.WhoList, bool) {
	h.whoMu.RLock()
	defer h.whoMu.RUnlock()
	for _, user := range h.whoList {
		if user.Username == username {
			return &user, true
		}
	}
	return nil, false
}

// NoReply checks if a bot's reply contains the noreply tag.
func NoReply(message string) bool {
	return strings.Contains(message, "<noreply>") || strings.TrimSpace(message) == ""
}
