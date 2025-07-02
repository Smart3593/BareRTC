package barertc

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/jwt"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
	"git.kirsle.net/apps/barertc/pkg/models"
	"git.kirsle.net/apps/barertc/pkg/spam"
	"git.kirsle.net/apps/barertc/pkg/util"
)

// OnLogin handles "login" actions from the client.
func (s *Server) OnLogin(sub *Subscriber, msg messages.Message) {
	// Using a JWT token for authentication?
	var claims = &jwt.Claims{}
	if msg.JWTToken != "" || (config.Current.JWT.Enabled && config.Current.JWT.Strict) {
		parsed, ok, err := jwt.ParseAndValidate(msg.JWTToken)
		if err != nil {
			log.Error("Error parsing JWT token in WebSocket login: %s", err)
			sub.ChatServer("Your authentication has expired. Please go back and launch the chat room again.")
			return
		}

		// Sanity check the username.
		if msg.Username != parsed.Subject {
			log.Error("JWT login had a different username: %s vs %s", parsed.Subject, msg.Username)
		}

		// Strict enforcement?
		if config.Current.JWT.Strict && !ok {
			log.Error("JWT enforcement is strict and user did not pass JWT checks")
			sub.ChatServer("Server side authentication is required. Please go back and launch the chat room from your logged-in account.")
			return
		}

		claims = parsed
		msg.Username = claims.Subject
		sub.JWTClaims = claims
	}

	if claims.Subject != "" {
		log.Debug("JWT claims: %+v", claims)
	}

	// Somehow no username?
	if msg.Username == "" {
		msg.Username = "anonymous"
	}

	// Ensure the username is unique, or rename it.
	username, err := s.UniqueUsername(msg.Username)
	if err != nil {
		// If JWT authentication was used: disconnect the original (conflicting) username.
		if claims.Subject == msg.Username {
			if other, err := s.GetSubscriber(msg.Username); err == nil {
				other.ChatServer("You have been signed out of chat because you logged in from another location.")
				other.SendJSON(messages.Message{
					Action: messages.ActionKick,
				})
				other.authenticated = false
				other.Username = ""
			}

			// They will take over their original username.
			username = msg.Username
		}

		// If JWT auth was not used: UniqueUsername already gave them a uniquely spelled name.
	}
	msg.Username = username

	// Is the username currently banned?
	if IsBanned(msg.Username) {
		sub.ChatServer(
			"You are currently banned from entering the chat room. Chat room bans are temporary and usually last for "+
				"24 hours.<br><br>Your ban expires in: %s",
			BanExpires(msg.Username),
		)
		sub.SendJSON(messages.Message{
			Action: messages.ActionKick,
		})
		return
	}

	// Use their username.
	sub.Username = msg.Username
	sub.authenticated = true
	sub.DND = msg.DND
	sub.loginAt = time.Now()
	log.Debug("OnLogin: %s joins the room", sub.Username)

	// Send the user back their settings.
	sub.SendMe()

	// Send the WhoList to everybody.
	s.SendWhoList()

	// Echo recent public channel messages to the user.
	sub.SendEchoedMessages()

	// Tell everyone they joined.
	s.Broadcast(messages.Message{
		Action:   messages.ActionPresence,
		Username: msg.Username,
		Message:  messages.PresenceJoined,
	})

	// Send the initial ChatServer messages to the public channels.
	for _, channel := range config.Current.PublicChannels {
		for _, msg := range channel.WelcomeMessages {
			sub.SendJSON(messages.Message{
				Channel:  channel.ID,
				Action:   messages.ActionError,
				Username: "ChatServer",
				Message:  RenderMarkdown(msg),
			})
		}
	}
}

// OnMessage handles a chat message posted by the user.
func (s *Server) OnMessage(sub *Subscriber, msg messages.Message) {
	if !strings.HasPrefix(msg.Channel, "@") {
		log.Info("[%s to #%s] %s", sub.Username, msg.Channel, msg.Message)
	}

	if sub.Username == "" || !sub.authenticated {
		sub.ChatServer("You must log in first.")
		return
	}

	// Process commands.
	if handled := s.ProcessCommand(sub, msg); handled {
		return
	}

	// Translate their message as Markdown syntax.
	markdown := RenderMarkdown(msg.Message)
	if markdown == "" {
		return
	}

	// Detect and expand media such as YouTube videos.
	markdown = s.ExpandMedia(markdown)

	// Assign a message ID and own it to the sender.
	sub.midMu.Lock()
	var mid = messages.NextMessageID()
	sub.messageIDs[mid] = struct{}{}
	sub.midMu.Unlock()

	// Message to be echoed to the channel.
	var message = messages.Message{
		Action:    messages.ActionMessage,
		Channel:   msg.Channel,
		Username:  sub.Username,
		Message:   markdown,
		MessageID: mid,
	}

	// Run message filters.
	if filter, ok := s.filterMessage(sub, msg, &message); ok {
		// What do we do with the matched filter?

		// If we will not send this message out, do echo it back to
		// the sender (possibly with censors applied).
		if !filter.ForwardMessage {
			s.SendTo(sub.Username, message)
		}

		// Is ChatServer to say something?
		if filter.ChatServerResponse != "" {
			sub.ChatServer(filter.ChatServerResponse)
		}

		// Are we to report the message to the site admin?
		if filter.ReportMessage {
			// If the user is OP, just tell them we would.
			if sub.IsAdmin() {
				sub.ChatServer("Your recent chat context would have been reported to your main website.")
			} else if err := s.reportFilteredMessage(sub, msg); err != nil {
				// Send the report to the main website.
				log.Error("Reporting filtered message: %s", err)
			}
		}

		// If we are not forwarding this message, stop here.
		if !filter.ForwardMessage {
			return
		}
	}

	// Is this a DM?
	if strings.HasPrefix(msg.Channel, "@") {

		// Check the message for hyperlink spam: if one user is copy/pasting a link to too many others
		// over DMs (possibly trying to evade detection from moderators), kick or ban them from chat.
		if err := spam.LinkSpamManager.Check(sub.Username, msg.Channel, msg.Message); err != nil {
			log.Error("Username %s has spammed a link too often (most recently to %s): %s", sub.Username, msg.Channel, msg.Message)

			var (
				isBan = err == spam.ErrLinkSpamBanUser

				// For the admin report.
				report   = "The user was **kicked** from the room with a warning."
				username = sub.Username
			)
			if isBan {
				report = "The user was **banned** from chat."
				s.BanUser(sub.Username, config.Current.DMLinkSpamProtection.BanHours*time.Hour)
			}

			s.KickUser(sub, err.Error(), isBan)

			// Send an admin report to your main website.
			if err := PostWebhookReport(WebhookRequestReport{
				FromUsername:  username,
				AboutUsername: username,
				Channel:       msg.Channel,
				Timestamp:     time.Now().Format(time.RFC3339),
				Reason:        "Spam Detected",
				Message: fmt.Sprintf(
					"_In a DM to:_ %s<br>[%s] %s",
					msg.Channel,
					username,
					msg.Message,
				),
				Comment: report,
			}); err != nil {
				log.Error("Error delivering a report to your website about the /nsfw command by %s: %s", sub.Username, err)
			}
			return
		}

		// Echo the message only to both parties.
		s.SendTo(sub.Username, message)
		message.Channel = "@" + sub.Username

		// Don't deliver it if the receiver has muted us. Note: admin users, even if muted,
		// can still deliver a DM to the one who muted them.
		rcpt, err := s.GetSubscriber(strings.TrimPrefix(msg.Channel, "@"))
		if err != nil {
			// Recipient was no longer online: the message won't be sent.
			sub.ChatServer("Could not deliver your message: %s appears not to be online.", msg.Channel)
			return
		} else if rcpt.Mutes(sub.Username) && !sub.IsAdmin() {
			log.Debug("Do not send message to %s: they have muted or booted %s", rcpt.Username, sub.Username)
			return
		}

		// If the sender already mutes the recipient, reply back with the error.
		if sub.Mutes(rcpt.Username) && !sub.IsAdmin() {
			sub.ChatServer("You have muted %s and so your message has not been sent.", rcpt.Username)
			return
		}

		// If there is blocking happening, do not send.
		if sub.Blocks(rcpt) {
			return
		}

		// Log this conversation?
		if IsLoggingUsername(sub) && IsLoggingUsername(rcpt) {
			// Both sides are logged, copy it to both logs.
			LogMessage(sub, rcpt.Username, sub.Username, msg)
			LogMessage(rcpt, sub.Username, sub.Username, msg)
		} else if IsLoggingUsername(sub) {
			// The sender of this message is being logged.
			LogMessage(sub, rcpt.Username, sub.Username, msg)
		} else if IsLoggingUsername(rcpt) {
			// The recipient of this message is being logged.
			LogMessage(rcpt, sub.Username, sub.Username, msg)
		}

		// Add it to the DM history SQLite database.
		if err := (models.DirectMessage{}).LogMessage(sub.Username, rcpt.Username, message); err != nil && err != models.ErrNotInitialized {
			log.Error("Logging DM history to SQLite: %s", err)
		}

		if err := s.SendTo(msg.Channel, message); err != nil {
			sub.ChatServer("Your message could not be delivered: %s", err)
		}
		return
	}

	// Are we logging this public channel?
	if IsLoggingChannel(msg.Channel) {
		LogChannel(s, msg.Channel, sub.Username, msg)
	}

	// Append it to the public channel's echo buffer.
	s.EchoPushPublicMessage(sub, message.Channel, message)

	// Broadcast a chat message to the room.
	s.Broadcast(message)
}

// OnTakeback handles takebacks (delete your message for everybody)
func (s *Server) OnTakeback(sub *Subscriber, msg messages.Message) {
	// In case we're in a DM thread, remove this message ID from the history table
	// if the username matches.
	wasRemovedFromHistory, err := (models.DirectMessage{}).TakebackMessage(sub.Username, msg.MessageID, sub.IsAdmin())
	if err != nil && err != models.ErrNotInitialized {
		log.Error("Error taking back DM history message (%s, %d): %s", sub.Username, msg.MessageID, err)
	}

	// Permission check.
	if sub.JWTClaims == nil || !sub.JWTClaims.IsAdmin {
		sub.midMu.Lock()
		_, ok := sub.messageIDs[msg.MessageID]
		sub.midMu.Unlock()

		if !ok {
			// The messageID is not found in the current chat session, but did we remove
			// it from past DM history for the correct current user?
			if !wasRemovedFromHistory {
				sub.ChatServer("That is not your message to take back.")
				return
			}
		}
	}

	// Remove it from cached echo buffers for public channels.
	s.EchoTakebackMessage(msg.MessageID)

	// Broadcast to everybody to remove this message.
	s.Broadcast(messages.Message{
		Action:    messages.ActionTakeback,
		MessageID: msg.MessageID,
	})
}

// OnReact handles emoji reactions for chat messages.
func (s *Server) OnReact(sub *Subscriber, msg messages.Message) {
	// Forward the reaction to everybody.
	s.Broadcast(messages.Message{
		Action:    messages.ActionReact,
		Username:  sub.Username,
		Message:   msg.Message,
		MessageID: msg.MessageID,
	})
}

// OnFile handles a picture shared in chat with a channel.
func (s *Server) OnFile(sub *Subscriber, msg messages.Message) {
	if sub.Username == "" {
		sub.ChatServer("You must log in first.")
		return
	}

	// Moderation rules?
	if rule := sub.GetModerationRule(); rule != nil {

		// Are they barred from watching cameras on chat?
		if rule.NoImage {
			sub.ChatServer(config.Current.Strings.ModRuleErrorNoImage)
			return
		}

	}

	// Detect image type and convert it into an <img src="data:"> tag.
	var (
		filename = msg.Message
		ext      = filepath.Ext(filename)
		filetype string
	)
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		filetype = "image/jpeg"
	case ".gif":
		filetype = "image/gif"
	case ".png":
		filetype = "image/png"
	default:
		sub.ChatServer("Unsupported image type, should be a jpeg, GIF or png.")
		return
	}

	// Process the image: scale it down, strip metadata, etc.
	img, pvWidth, pvHeight := ProcessImage(filetype, msg.Bytes)
	var dataURL = fmt.Sprintf("data:%s;base64,%s", filetype, base64.StdEncoding.EncodeToString(img))

	// Assign a message ID and own it to the sender.
	sub.midMu.Lock()
	var mid = messages.NextMessageID()
	sub.messageIDs[mid] = struct{}{}
	sub.midMu.Unlock()

	// Message to be echoed to the channel.
	var message = messages.Message{
		Action:    messages.ActionMessage,
		Channel:   msg.Channel,
		Username:  sub.Username,
		MessageID: mid,

		// Their image embedded via a data: URI - no server storage needed!
		Message: fmt.Sprintf(
			`<img src="%s" width="%d" height="%d" onclick="setModalImage(this.src)" style="cursor: pointer">`,
			dataURL,
			pvWidth, pvHeight,
		),
	}

	// Is this a DM?
	if strings.HasPrefix(msg.Channel, "@") {
		// Echo the message only to both parties.
		s.SendTo(sub.Username, message)
		message.Channel = "@" + sub.Username

		// Don't deliver it if the receiver has muted us.
		rcpt, err := s.GetSubscriber(strings.TrimPrefix(msg.Channel, "@"))
		if err == nil && rcpt.Mutes(sub.Username) {
			log.Debug("Do not send message to %s: they have muted or booted %s", rcpt.Username, sub.Username)
			return
		}

		// If the sender already mutes the recipient, reply back with the error.
		if sub.Mutes(rcpt.Username) {
			sub.ChatServer("You have muted %s and so your message has not been sent.", rcpt.Username)
			return
		}

		// If there is blocking happening, do not send.
		if sub.Blocks(rcpt) {
			return
		}

		if err := s.SendTo(msg.Channel, message); err != nil {
			sub.ChatServer("Your message could not be delivered: %s", err)
		}
		return
	}

	// Broadcast a chat message to the room.
	s.Broadcast(message)
}

// OnMe handles current user state updates.
func (s *Server) OnMe(sub *Subscriber, msg messages.Message) {
	// Reflect a 'me' message back at them? (e.g. if server forces their camera NSFW)
	var reflect bool

	if msg.VideoStatus&messages.VideoFlagActive == messages.VideoFlagActive {
		log.Debug("User %s turns on their video feed", sub.Username)

		// Moderation rules?
		if rule := sub.GetModerationRule(); rule != nil {

			// Are they barred from sharing their camera on chat?
			if rule.NoBroadcast || rule.NoVideo {
				sub.SendCut()
				sub.ChatServer(config.Current.Strings.ModRuleErrorNoBroadcast)
				msg.VideoStatus = 0
			}

			// Is their camera forced to always be explicit?
			if rule.CameraAlwaysNSFW && !(msg.VideoStatus&messages.VideoFlagNSFW == messages.VideoFlagNSFW) {
				msg.VideoStatus |= messages.VideoFlagNSFW
				reflect = true // send them a 'me' echo afterward to inform the front-end page properly of this
				sub.ChatServer(config.Current.Strings.ModRuleErrorCameraAlwaysNSFW)
			}

		}
	}

	// Hidden status: for operators only, + fake a join/exit chat message.
	if sub.JWTClaims != nil && sub.JWTClaims.IsAdmin {
		if sub.ChatStatus != "hidden" && msg.ChatStatus == "hidden" {
			// Going hidden - fake leave message
			s.Broadcast(messages.Message{
				Action:   messages.ActionPresence,
				Username: sub.Username,
				Message:  messages.PresenceExited,
			})
		} else if sub.ChatStatus == "hidden" && msg.ChatStatus != "hidden" {
			// Leaving hidden - fake join message
			s.Broadcast(messages.Message{
				Action:   messages.ActionPresence,
				Username: sub.Username,
				Message:  messages.PresenceJoined,
			})
		}
	} else if msg.ChatStatus == "hidden" {
		// normal users can not set this status
		msg.ChatStatus = "away"
	}

	sub.VideoStatus = msg.VideoStatus
	sub.ChatStatus = msg.ChatStatus
	sub.DND = msg.DND

	// Sync the WhoList to everybody.
	s.SendWhoList()

	// Reflect a 'me' message back?
	if reflect {
		sub.SendMe()
	}
}

// OnVideoInvite is a client inviting another to watch their camera.
func (s *Server) OnVideoInvite(sub *Subscriber, msg messages.Message) {
	sub.muteMu.Lock()
	for _, username := range msg.Usernames {
		sub.invited[username] = struct{}{}
	}
	sub.muteMu.Unlock()

	s.SendWhoList()
}

// OnOpen is a client wanting to start WebRTC with another, e.g. to see their camera.
func (s *Server) OnOpen(sub *Subscriber, msg messages.Message) {
	// Moderation rules?
	if rule := sub.GetModerationRule(); rule != nil {

		// Are they barred from watching cameras on chat?
		if rule.NoVideo {
			sub.ChatServer(config.Current.Strings.ModRuleErrorNoVideo)
			return
		}

	}

	// Look up the other subscriber.
	other, err := s.GetSubscriber(msg.Username)
	if err != nil {
		sub.ChatServer(
			"Could not open that video: %s appears to be offline.", msg.Username,
		)
		return
	}

	// Enforce whether the viewer has permission to see this camera.
	if ok, reason := s.IsVideoNotAllowed(sub, other); !ok {
		sub.ChatServer(
			"Could not open that video: %s", reason,
		)
		return
	}

	// Make up a WebRTC shared secret and send it to both of them.
	secret := util.RandomString(16)
	log.Info("WebRTC: %s opens %s with secret %s", sub.Username, other.Username, secret)

	// If the current user is an admin and was booted or muted, inform them.
	if sub.IsAdmin() {
		if other.Boots(sub.Username) {
			sub.ChatServer("Note: %s had booted you off their camera before, and won't be notified of your watch.", other.Username)
		} else if other.Mutes(sub.Username) {
			sub.ChatServer("Note: %s had muted you before, and won't be notified of your watch.", other.Username)
		}
	}

	// Ring the target of this request and give them the secret.
	other.SendJSON(messages.Message{
		Action:     messages.ActionRing,
		Username:   sub.Username,
		OpenSecret: secret,
	})

	// To the caller, echo back the Open along with the secret.
	sub.SendJSON(messages.Message{
		Action:     messages.ActionOpen,
		Username:   other.Username,
		OpenSecret: secret,
	})
}

// IsVideoNotAllowed verifies whether a viewer can open a broadcaster's camera.
//
// Returns a boolean and an error message to return if false.
func (s *Server) IsVideoNotAllowed(sub *Subscriber, other *Subscriber) (bool, string) {
	var (
		ourVideoActive      = (sub.VideoStatus & messages.VideoFlagActive) == messages.VideoFlagActive
		theirVideoActive    = (other.VideoStatus & messages.VideoFlagActive) == messages.VideoFlagActive
		theirMutualRequired = (other.VideoStatus & messages.VideoFlagMutualRequired) == messages.VideoFlagMutualRequired
		theirVIPRequired    = (other.VideoStatus & messages.VideoFlagOnlyVIP) == messages.VideoFlagOnlyVIP
		theyInvitedUs       = other.InvitesVideo(sub.Username)
	)

	// Conditions in which we can not watch their video.
	var conditions = []struct {
		If    bool
		Error string
	}{
		{
			If:    !theirVideoActive,
			Error: "Their video is not currently enabled.",
		},
		{
			If:    !theyInvitedUs && (theirMutualRequired && !ourVideoActive),
			Error: fmt.Sprintf("%s has requested that you should share your own camera too before opening theirs.", other.Username),
		},
		{
			If:    !theyInvitedUs && (theirVIPRequired && !sub.IsVIP() && !sub.IsAdmin()),
			Error: "You do not have permission to view that camera.",
		},
		{
			If:    (other.Mutes(sub.Username) || other.Boots(sub.Username) || other.Blocks(sub)) && !sub.IsAdmin(),
			Error: "You do not have permission to view that camera.",
		},
	}

	for _, c := range conditions {
		if c.If {
			return false, c.Error
		}
	}

	return true, ""
}

// OnBoot is a user kicking you off their video stream.
func (s *Server) OnBoot(sub *Subscriber, msg messages.Message, boot bool) {
	sub.muteMu.Lock()

	if boot {
		log.Info("%s boots %s off their camera", sub.Username, msg.Username)
		sub.booted[msg.Username] = struct{}{}

		// If the subject of the boot is an admin, inform them they have been booted.
		if other, err := s.GetSubscriber(msg.Username); err == nil && other.IsAdmin() {
			other.ChatServer(
				"%s has booted you off of their camera!",
				sub.Username,
			)
		}
	} else {
		log.Info("%s unboots %s from their camera", sub.Username, msg.Username)
		delete(sub.booted, msg.Username)
	}

	sub.muteMu.Unlock()

	s.SendWhoList()
}

// OnMute is a user kicking setting the mute flag for another user.
func (s *Server) OnMute(sub *Subscriber, msg messages.Message, mute bool) {
	log.Info("%s mutes or unmutes %s: %v", sub.Username, msg.Username, mute)

	sub.muteMu.Lock()

	if mute {
		sub.muted[msg.Username] = struct{}{}
	} else {
		delete(sub.muted, msg.Username)
	}

	sub.muteMu.Unlock()

	// If the subject of the mute is an admin, inform them they have been booted.
	if other, err := s.GetSubscriber(msg.Username); err == nil && other.IsAdmin() {
		other.ChatServer(
			"%s has muted you! Your new mute status is: %v",
			sub.Username, mute,
		)
	}

	// Send the Who List in case our cam will show as disabled to the muted party.
	s.SendWhoList()
}

// OnBlock is a user placing a hard block (hide from) another user.
func (s *Server) OnBlock(sub *Subscriber, msg messages.Message) {
	log.Info("%s blocks %s", sub.Username, msg.Username)

	// If the subject of the block is an admin, return an error.
	if other, err := s.GetSubscriber(msg.Username); err == nil && other.IsAdmin() {
		sub.ChatServer(
			"You are not allowed to block a chat operator.",
		)
		return
	}

	sub.muteMu.Lock()
	sub.blocked[msg.Username] = struct{}{}
	sub.muteMu.Unlock()

	// Send the Who List so the blocker/blockee can disappear from each other's list.
	s.SendWhoList()
}

// OnBlocklist is a bulk user mute from the CachedBlocklist sent by the website.
func (s *Server) OnBlocklist(sub *Subscriber, msg messages.Message) {
	log.Info("[%s] syncs their blocklist: %s", sub.Username, msg.Usernames)

	sub.muteMu.Lock()
	for _, username := range msg.Usernames {
		sub.muted[username] = struct{}{}
		sub.blocked[username] = struct{}{}
	}

	sub.muteMu.Unlock()

	// Send the Who List in case our cam will show as disabled to the muted party.
	s.SendWhoList()
}

// OnReport handles a user's report of a message.
func (s *Server) OnReport(sub *Subscriber, msg messages.Message) {
	if !WebhookEnabled(WebhookReport) {
		sub.ChatServer("Unfortunately, the report webhook is not enabled so your report could not be received!")
		return
	}

	// Attach recent message context to DMs.
	if strings.HasPrefix(msg.Channel, "@") {
		context := getDirectMessageContext(sub.Username, msg.Username)
		msg.Message += "\n\nRecent message context:\n\n" + context
	}

	// Post to the report webhook.
	if err := PostWebhookReport(WebhookRequestReport{
		FromUsername:  sub.Username,
		AboutUsername: msg.Username,
		Channel:       msg.Channel,
		Timestamp:     msg.Timestamp,
		Reason:        msg.Reason,
		Message:       msg.Message,
		Comment:       msg.Comment,
	}); err != nil {
		sub.ChatServer("Error sending the report to the website: %s", err)
	} else {
		sub.ChatServer("Your report has been delivered successfully.")
	}
}

// OnCandidate handles WebRTC candidate signaling.
func (s *Server) OnCandidate(sub *Subscriber, msg messages.Message) {
	// Look up the other subscriber.
	other, err := s.GetSubscriber(msg.Username)
	if err != nil {
		return
	}

	other.SendJSON(messages.Message{
		Action:    messages.ActionCandidate,
		Username:  sub.Username,
		Candidate: msg.Candidate,
	})
}

// OnSDP handles WebRTC sdp signaling.
func (s *Server) OnSDP(sub *Subscriber, msg messages.Message) {
	// Look up the other subscriber.
	other, err := s.GetSubscriber(msg.Username)
	if err != nil {
		return
	}

	other.SendJSON(messages.Message{
		Action:      messages.ActionSDP,
		Username:    sub.Username,
		Description: msg.Description,
	})
}

// OnWatch communicates video watching status between users.
func (s *Server) OnWatch(sub *Subscriber, msg messages.Message) {
	// Look up the other subscriber.
	other, err := s.GetSubscriber(msg.Username)
	if err != nil {
		return
	}

	other.SendJSON(messages.Message{
		Action:   messages.ActionWatch,
		Username: sub.Username,
	})
}

// OnUnwatch communicates video Unwatching status between users.
func (s *Server) OnUnwatch(sub *Subscriber, msg messages.Message) {
	// Look up the other subscriber.
	other, err := s.GetSubscriber(msg.Username)
	if err != nil {
		return
	}

	other.SendJSON(messages.Message{
		Action:   messages.ActionUnwatch,
		Username: sub.Username,
	})
}
