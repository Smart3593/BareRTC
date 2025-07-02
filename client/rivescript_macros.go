package client

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
	"github.com/aichaos/rivescript-go"
	"github.com/aichaos/rivescript-go/lang/javascript"
)

// Set up object macros for RiveScript.
func (h *BotHandlers) setObjectMacros() {
	// Reload the bot's RiveScript brain.
	h.rs.SetSubroutine("reload", func(rs *rivescript.RiveScript, args []string) string {
		var bot = rivescript.New(&rivescript.Config{
			UTF8:  true,
			Debug: rs.Debug,
		})
		bot.SetHandler("javascript", javascript.New(bot))

		if err := bot.LoadDirectory("brain"); err != nil {
			return fmt.Sprintf("Error on LoadDirectory: %s", err)
		}
		if err := bot.SortReplies(); err != nil {
			return fmt.Sprintf("Error on SortReplies: %s", err)
		}

		// Install the new bot and set object macros on it.
		h.rs = bot
		h.setObjectMacros()

		return "The RiveScript brain has been reloaded!"
	})

	// React to a message.
	h.rs.SetSubroutine("react", func(rs *rivescript.RiveScript, args []string) string {
		if len(args) >= 2 {
			if msgID, err := strconv.Atoi(args[0]); err == nil {
				// With a small delay.
				go func() {
					time.Sleep(2500 * time.Millisecond)
					h.client.Send(messages.Message{
						Action:    messages.ActionReact,
						MessageID: int64(msgID),
						Message:   args[1],
					})
				}()
			} else {
				return fmt.Sprintf("[react: %s]", err)
			}
			return ""
		}
		return "[react: invalid number of parameters]"
	})

	// Mark a camera NSFW for a username.
	h.rs.SetSubroutine("nsfw", func(rs *rivescript.RiveScript, args []string) string {
		if len(args) >= 1 {
			var username = strings.TrimPrefix(args[0], "@")
			h.client.Send(messages.Message{
				Action:  messages.ActionMessage,
				Message: fmt.Sprintf("/nsfw %s", username),
			})
			return ""
		}
		return "[nsfw: invalid number of parameters]"
	})

	// Takeback a message (admin action especially)
	h.rs.SetSubroutine("takeback", func(rs *rivescript.RiveScript, args []string) string {
		if len(args) >= 1 {
			if msgID, err := strconv.Atoi(args[0]); err == nil {
				// Take it back.
				h.client.Send(messages.Message{
					Action:    messages.ActionTakeback,
					MessageID: int64(msgID),
				})
			} else {
				return fmt.Sprintf("[takeback: %s]", err)
			}
			return ""
		}
		return "[takeback: invalid number of parameters]"
	})

	// Flag (report) a message on chat.
	h.rs.SetSubroutine("report", func(rs *rivescript.RiveScript, args []string) string {
		if len(args) >= 2 {
			if msgID, err := strconv.Atoi(args[0]); err == nil {
				var comment = strings.Join(args[1:], " ")

				// Look up this message.
				if msg, ok := h.getMessageByID(int64(msgID)); ok {
					// Report it with the custom comment.
					h.client.Send(messages.Message{
						Action:    messages.ActionReport,
						Channel:   msg.Channel,
						Username:  msg.Username,
						Timestamp: "not recorded",
						Reason:    "Automated chatbot flag",
						Message:   msg.Message,
						Comment:   comment,
					})
					return ""
				}

				return "[msgID not found]"
			} else {
				return fmt.Sprintf("[report: %s]", err)
			}
		}
		return "[report: invalid number of parameters]"
	})

	// Send a user a Direct Message.
	h.rs.SetSubroutine("dm", func(rs *rivescript.RiveScript, args []string) string {
		if len(args) >= 2 {
			var (
				username = args[0]
				message  = strings.Join(args[1:], " ")
			)

			// Slide into their DMs.
			log.Error("Send DM to [%s]: %s", username, message)
			h.client.Send(messages.Message{
				Action:  messages.ActionMessage,
				Channel: "@" + username,
				Message: message,
			})
		} else {
			return "[dm: invalid number of parameters]"
		}
		return ""
	})

	// Send a public chat message to a channel name.
	h.rs.SetSubroutine("send-message", func(rs *rivescript.RiveScript, args []string) string {
		if len(args) >= 2 {
			var (
				channel = args[0]
				message = strings.Join(args[1:], " ")
			)

			// Slide into their DMs.
			log.Error("Send chat to [%s]: %s", channel, message)
			h.client.Send(messages.Message{
				Action:  messages.ActionMessage,
				Channel: channel,
				Message: message,
			})
		} else {
			return "[send-message: invalid number of parameters]"
		}
		return ""
	})

	// Send a public chat message to a channel name.
	h.rs.SetSubroutine("ban", func(rs *rivescript.RiveScript, args []string) string {
		if len(args) >= 1 {
			var (
				username = args[0]
				duration = "24"
			)
			if len(args) > 1 {
				duration = args[1]
			}

			log.Error("Macro: /ban %s %s", username, duration)
			h.client.Send(messages.Message{
				Action:  messages.ActionMessage,
				Channel: "barertc-debug",
				Message: fmt.Sprintf("/ban %s %s", username, duration),
			})
		} else {
			return "[ban: invalid number of parameters]"
		}
		return ""
	})
}
