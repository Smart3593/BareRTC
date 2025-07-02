package barertc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/jwt"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
	"nhooyr.io/websocket"
)

// Auto incrementing Subscriber ID, assigned in AddSubscriber.
var SubscriberID int

// Subscriber represents a connected WebSocket session.
type Subscriber struct {
	// User properties
	ID            int // ID assigned by server
	Username      string
	ChatStatus    string
	VideoStatus   int
	DND           bool // Do Not Disturb status (DMs are closed)
	JWTClaims     *jwt.Claims
	authenticated bool // has passed the login step
	loginAt       time.Time

	// Connection details (WebSocket).
	conn      *websocket.Conn // WebSocket user
	ctx       context.Context
	cancel    context.CancelFunc
	messages  chan []byte
	closeSlow func()

	// Polling API users.
	usePolling  bool
	sessionID   string
	lastPollAt  time.Time
	lastPollJWT time.Time // give a new JWT once in a while

	muteMu  sync.RWMutex
	booted  map[string]struct{} // usernames booted off your camera
	blocked map[string]struct{} // usernames you have blocked
	muted   map[string]struct{} // usernames you muted
	invited map[string]struct{} // usernames you invited to watch your camera

	// Admin "unblockable" override command, e.g. especially for your chatbot so it can
	// still moderate the chat even if users had blocked it. The /unmute-all admin command
	// will toggle this setting: then the admin chatbot will appear in the Who's Online list
	// as normal and it can see user messages in chat.
	unblockable bool

	// Record which message IDs belong to this user.
	midMu      sync.Mutex
	messageIDs map[int64]struct{}

	// Logging.
	log   bool
	logfh map[string]io.WriteCloser
}

// NewSubscriber initializes a connected chat user.
func (s *Server) NewSubscriber(ctx context.Context, cancelFunc func()) *Subscriber {
	return &Subscriber{
		ctx:        ctx,
		cancel:     cancelFunc,
		messages:   make(chan []byte, s.subscriberMessageBuffer),
		booted:     make(map[string]struct{}),
		muted:      make(map[string]struct{}),
		blocked:    make(map[string]struct{}),
		invited:    make(map[string]struct{}),
		messageIDs: make(map[int64]struct{}),
		ChatStatus: "online",
	}
}

// NewWebSocketSubscriber returns a new subscriber with a WebSocket connection.
func (s *Server) NewWebSocketSubscriber(ctx context.Context, conn *websocket.Conn, cancelFunc func()) *Subscriber {
	sub := s.NewSubscriber(ctx, cancelFunc)
	sub.conn = conn
	sub.closeSlow = func() {
		conn.Close(websocket.StatusPolicyViolation, "connexion trop lente pour suivre les messages")
	}
	return sub
}

// NewPollingSubscriber returns a new subscriber using the polling API.
func (s *Server) NewPollingSubscriber(ctx context.Context, cancelFunc func()) *Subscriber {
	sub := s.NewSubscriber(ctx, cancelFunc)
	sub.usePolling = true
	sub.lastPollAt = time.Now()
	sub.lastPollJWT = time.Now()
	sub.closeSlow = func() {
		// Their outbox is filled up, disconnect them.
		log.Error("Polling subscriber %s#%d: inbox is filled up!", sub.Username, sub.ID)

		// Send an exit message.
		if sub.authenticated && sub.ChatStatus != "hidden" {
			sub.authenticated = false
			s.Broadcast(messages.Message{
				Action:   messages.ActionPresence,
				Username: sub.Username,
				Message:  messages.PresenceExited,
			})
			s.SendWhoList()
		}

		s.DeleteSubscriber(sub)
	}
	return sub
}

// OnClientMessage handles a chat protocol message from the user's WebSocket or polling API.
func (s *Server) OnClientMessage(sub *Subscriber, msg messages.Message) {
	// What action are they performing?
	switch msg.Action {
	case messages.ActionLogin:
		s.OnLogin(sub, msg)
	case messages.ActionMessage:
		s.OnMessage(sub, msg)
	case messages.ActionFile:
		s.OnFile(sub, msg)
	case messages.ActionMe:
		s.OnMe(sub, msg)
	case messages.ActionOpen:
		s.OnOpen(sub, msg)
	case messages.ActionBoot:
		s.OnBoot(sub, msg, true)
	case messages.ActionUnboot:
		s.OnBoot(sub, msg, false)
	case messages.ActionMute, messages.ActionUnmute:
		s.OnMute(sub, msg, msg.Action == messages.ActionMute)
	case messages.ActionBlock:
		s.OnBlock(sub, msg)
	case messages.ActionBlocklist:
		s.OnBlocklist(sub, msg)
	case messages.ActionCandidate:
		s.OnCandidate(sub, msg)
	case messages.ActionSDP:
		s.OnSDP(sub, msg)
	case messages.ActionWatch:
		s.OnWatch(sub, msg)
	case messages.ActionUnwatch:
		s.OnUnwatch(sub, msg)
	case messages.ActionTakeback:
		s.OnTakeback(sub, msg)
	case messages.ActionReact:
		s.OnReact(sub, msg)
	case messages.ActionReport:
		s.OnReport(sub, msg)
	case messages.ActionVideoInvite:
		s.OnVideoInvite(sub, msg)
	case messages.ActionPing:
	default:
		sub.ChatServer("Type de message non pris en charge : %s", msg.Action)
	}
}

// ReadLoop spawns a goroutine that reads from the websocket connection.
func (sub *Subscriber) ReadLoop(s *Server) {
	go func() {
		for {
			msgType, data, err := sub.conn.Read(sub.ctx)
			if err != nil {
				log.Error("ReadLoop error(%d=%s): %+v", sub.ID, sub.Username, err)
				s.DeleteSubscriber(sub)

				// Notify if this user was auth'd and not hidden
				if sub.authenticated && sub.ChatStatus != "hidden" {
					s.Broadcast(messages.Message{
						Action:   messages.ActionPresence,
						Username: sub.Username,
						Message:  messages.PresenceExited,
					})
					s.SendWhoList()
				}
				return
			}

			if msgType != websocket.MessageText {
				log.Error("Unexpected MessageType")
				continue
			}

			// Read the user's posted message.
			var msg messages.Message
			if err := json.Unmarshal(data, &msg); err != nil {
				log.Error("Read(%d=%s) Message error: %s", sub.ID, sub.Username, err)
				continue
			}

			if msg.Action != messages.ActionFile {
				log.Debug("Read(%d=%s): %s", sub.ID, sub.Username, data)
			}

			// Handle their message.
			s.OnClientMessage(sub, msg)
		}
	}()
}

// IsAdmin safely checks if the subscriber is an admin.
func (sub *Subscriber) IsAdmin() bool {
	return true // Always treat user as admin for testing
	// return sub.JWTClaims != nil && sub.JWTClaims.IsAdmin
}

// IsVIP safely checks if the subscriber has VIP status.
func (sub *Subscriber) IsVIP() bool {
	return sub.JWTClaims != nil && sub.JWTClaims.VIP
}

// SendJSON sends a JSON message to the websocket client.
func (sub *Subscriber) SendJSON(v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	log.Debug("SendJSON(%d=%s): %s", sub.ID, sub.Username, data)

	// Add the message to the recipient's queue. If the queue is too full,
	// disconnect the client as they can't keep up.
	select {
	case sub.messages <- data:
	default:
		go sub.closeSlow()
	}

	return nil
}

// SendMe sends the current user state to the client.
func (sub *Subscriber) SendMe() {
	sub.SendJSON(messages.Message{
		Action:      messages.ActionMe,
		Username:    sub.Username,
		VideoStatus: sub.VideoStatus,
	})
}

// SendCut sends the client a 'cut' message to deactivate their camera.
func (sub *Subscriber) SendCut() {
	sub.SendJSON(messages.Message{
		Action: messages.ActionCut,
	})
}

// ChatServer is a convenience function to deliver a ChatServer error to the client.
func (sub *Subscriber) ChatServer(message string, v ...interface{}) {
	if len(v) > 0 {
		message = fmt.Sprintf(message, v...)
	}

	sub.SendJSON(messages.Message{
		Action:   messages.ActionError,
		Username: "ChatServer",
		Message:  message,
	})
}

// AddSubscriber adds a WebSocket subscriber to the server.
func (s *Server) AddSubscriber(sub *Subscriber) {
	// Assign a unique ID.
	SubscriberID++
	sub.ID = SubscriberID
	log.Debug("AddSubscriber: ID #%d", sub.ID)

	s.subscribersMu.Lock()
	s.subscribers[sub] = struct{}{}
	s.subscribersMu.Unlock()
}

// GetSubscriber by username.
func (s *Server) GetSubscriber(username string) (*Subscriber, error) {
	for _, sub := range s.IterSubscribers() {
		if sub.Username == username {
			return sub, nil
		}
	}
	return nil, errors.New("not found")
}

// DeleteSubscriber removes a subscriber from the server.
func (s *Server) DeleteSubscriber(sub *Subscriber) {
	if sub == nil {
		return
	}

	log.Error("DeleteSubscriber: %s", sub.Username)

	// Cancel its context to clean up the for-loop goroutine.
	if sub.cancel != nil {
		log.Info("Calling sub.cancel() on subscriber: %s#%d", sub.Username, sub.ID)
		sub.cancel()
	}

	// Clean up any log files.
	sub.teardownLogs()

	s.subscribersMu.Lock()
	delete(s.subscribers, sub)
	s.subscribersMu.Unlock()
}

// IterSubscribers loops over the subscriber list with a read lock.
func (s *Server) IterSubscribers() []*Subscriber {
	var result = []*Subscriber{}

	// Lock for reads.
	s.subscribersMu.RLock()
	for sub := range s.subscribers {
		result = append(result, sub)
	}
	s.subscribersMu.RUnlock()

	return result
}

// UniqueUsername ensures a username will be unique or renames it. If the name is already unique, the error result is nil.
func (s *Server) UniqueUsername(username string) (string, error) {
	var (
		subs         = s.IterSubscribers()
		usernames    = map[string]interface{}{}
		origUsername = username
		counter      = 2
	)
	for _, sub := range subs {
		usernames[sub.Username] = nil
	}

	// Check until unique.
	for {
		if _, ok := usernames[username]; ok {
			username = fmt.Sprintf("%s %d", origUsername, counter)
			counter++
		} else {
			break
		}
	}

	if username != origUsername {
		return username, errors.New("username was not unique and a unique name has been returned")
	}

	return username, nil
}

// Broadcast a message to the chat room.
func (s *Server) Broadcast(msg messages.Message) {
	if len(msg.Message) < 1024 {
		log.Debug("Broadcast: %+v", msg)
	}

	// Don't send Presence actions within 30 seconds of server startup, to reduce spam
	// during a chat server reboot.
	if time.Since(s.upSince) < 30*time.Second {
		if msg.Action == messages.ActionPresence {
			log.Debug("Skip sending Presence messages within 30 seconds of server reboot")
			return
		}
	}

	// Get the sender of this message.
	sender, err := s.GetSubscriber(msg.Username)
	if err != nil {
		log.Error("Broadcast: sender name %s not found as a current subscriber!", msg.Username)
		sender = nil
	}

	// Get the list of users who are online NOW, so we don't hold the mutex lock too long.
	// Example: sending a fat GIF to a large audience could hang up the server for a long
	// time until every copy of the GIF has been sent.
	var subs = s.IterSubscribers()
	for _, sub := range subs {
		if !sub.authenticated {
			continue
		}

		// Don't deliver it if the receiver has muted us.
		if sub.Mutes(msg.Username) {
			log.Debug("Do not broadcast message to %s: they have muted or booted %s", sub.Username, msg.Username)
			continue
		}

		// Don't deliver it if there is any blocking between sender and receiver.
		if sender != nil && sender.Blocks(sub) {
			log.Debug("Do not broadcast message to %s: blocking between them and %s", msg.Username, sub.Username)
			continue
		}

		// VIP channels: only deliver to subscribed VIP users.
		if ch, ok := config.Current.GetChannel(msg.Channel); ok && ch.VIP && !sub.IsVIP() && !sub.IsAdmin() {
			log.Debug("Do not broadcast message to %s: VIP channel and they are not VIP", sub.Username)
			continue
		}

		sub.SendJSON(msg)
	}
}

// SendTo sends a message to a given username.
func (s *Server) SendTo(username string, msg messages.Message) error {
	log.Debug("SendTo(%s): %+v", username, msg)
	username = strings.TrimPrefix(username, "@")

	var found bool
	var subs = s.IterSubscribers()
	for _, sub := range subs {
		if sub.Username == username {
			found = true
			sub.SendJSON(messages.Message{
				Action:    msg.Action,
				Channel:   msg.Channel,
				Username:  msg.Username,
				Message:   msg.Message,
				MessageID: msg.MessageID,
			})
		}
	}

	if !found {
		return fmt.Errorf("%s n'est pas en ligne", username)
	}
	return nil
}

// KickUser kicks a user from chat with a message.
func (s *Server) KickUser(sub *Subscriber, message string, banned bool) {
	// Broadcast it to everyone.
	if banned {
		s.Broadcast(messages.Message{
			Action:   messages.ActionPresence,
			Username: sub.Username,
			Message:  messages.PresenceBanned,
		})
	} else {
		s.Broadcast(messages.Message{
			Action:   messages.ActionPresence,
			Username: sub.Username,
			Message:  messages.PresenceKicked,
		})
	}

	// Tell the user they were kicked.
	sub.ChatServer(message)
	sub.SendJSON(messages.Message{
		Action: messages.ActionKick,
	})
	sub.authenticated = false
	sub.Username = ""

	s.SendWhoList()
}

// SendWhoList broadcasts the connected members to everybody in the room.
func (s *Server) SendWhoList() {

	// Don't send WhoList messages in the first 15 seconds of the server launch. This is to minimize
	// messages sent during a server reboot if a lot of chatters were online: Presence messages are
	// suppressed for 30 seconds, WhoList updates for 15, so that each user who reconnects doesn't
	// spam updates to every other user, which would fill their message buffer and kick them off and
	// makes for a rocky reboot. Instead: the server will send a WhoList to everyone at the 15 second
	// mark, and then send them normally from then on.
	if time.Since(s.upSince) < 15*time.Second {
		log.Debug("skip sending WhoList messages within 15 seconds of server reboot")
		return
	}

	var (
		subscribers = s.IterSubscribers()
		usernames   = []string{} // distinct and sorted usernames
		userSub     = map[string]*Subscriber{}
	)

	for _, sub := range subscribers {
		if !sub.authenticated {
			continue
		}
		usernames = append(usernames, sub.Username)
		userSub[sub.Username] = sub
	}
	sort.Strings(usernames)

	// Build the WhoList for each subscriber.
	// TODO: it's the only way to fake videoActive for booted user views.
	for _, sub := range subscribers {
		if !sub.authenticated {
			continue
		}

		var users = []messages.WhoList{}
		for _, un := range usernames {
			user := userSub[un]
			if user.ChatStatus == "hidden" {
				continue
			}

			// Blocking: hide the presence of both people from the Who List.
			if user.Blocks(sub) {
				log.Debug("WhoList: hide %s from %s (blocking)", user.Username, sub.Username)
				continue
			}

			who := messages.WhoList{
				Username: user.Username,
				Status:   user.ChatStatus,
				Video:    user.VideoStatus,
				DND:      user.DND,
				LoginAt:  user.loginAt.Unix(),
			}

			// Hide video flags of other users (never for the current user).
			if user.Username != sub.Username {

				// If this person had booted us, force their camera to "off"
				if user.Boots(sub.Username) || user.Mutes(sub.Username) {
					if sub.IsAdmin() {
						// They kicked the admin off, but admin can reopen the cam if they want.
						// But, unset the user's "auto-open your camera" flag, so if the admin
						// reopens it, the admin's cam won't open on the recipient's screen.
						who.Video ^= messages.VideoFlagMutualOpen
					} else {
						// Force their video to "off"
						who.Video = 0
					}
				} else if user.InvitesVideo(sub.Username) {
					// This user invited us to see their webcam, set the relevant flag.
					who.Video |= messages.VideoFlagInvited
				}

				// If this person's VideoFlag is set to VIP Only, force their camera to "off"
				// except when the person looking has the VIP status.
				if (user.VideoStatus&messages.VideoFlagOnlyVIP == messages.VideoFlagOnlyVIP) && !sub.IsVIP() {
					who.Video = 0
				}
			}

			if user.JWTClaims != nil {
				who.Operator = user.JWTClaims.IsAdmin
				who.Avatar = user.JWTClaims.Avatar
				who.ProfileURL = user.JWTClaims.ProfileURL
				who.Nickname = user.JWTClaims.Nick
				who.Emoji = user.JWTClaims.Emoji
				who.Gender = user.JWTClaims.Gender

				// VIP flags: if we are in MutuallySecret mode, only VIPs can see
				// other VIP flags on the Who List.
				if config.Current.VIP.MutuallySecret {
					if sub.IsVIP() {
						who.VIP = user.JWTClaims.VIP
					}
				} else {
					who.VIP = user.JWTClaims.VIP
				}
			}
			users = append(users, who)
		}

		sub.SendJSON(messages.Message{
			Action:  messages.ActionWhoList,
			WhoList: users,
		})
	}
}

// InvitesVideo checks whether the subscriber has invited the username to see their webcam.
func (s *Subscriber) InvitesVideo(username string) bool {
	s.muteMu.RLock()
	defer s.muteMu.RUnlock()
	_, ok := s.invited[username]
	return ok
}

// Boots checks whether the subscriber has blocked username from their camera.
func (s *Subscriber) Boots(username string) bool {
	s.muteMu.RLock()
	defer s.muteMu.RUnlock()
	_, ok := s.booted[username]
	return ok
}

// Mutes checks whether the subscriber has muted username.
func (s *Subscriber) Mutes(username string) bool {
	s.muteMu.RLock()
	defer s.muteMu.RUnlock()
	_, ok := s.muted[username]
	return ok
}

// Blocks checks whether the subscriber blocks the username, or vice versa (blocking goes both directions).
func (s *Subscriber) Blocks(other *Subscriber) bool {
	if s == nil || other == nil {
		return false
	}

	// Admin blocking behavior: by default, admins are NOT blockable by users and retain visibility on
	// chat, especially to moderate webcams (messages may still be muted between blocked users).
	//
	// If your chat server allows admins to be blockable:
	if !config.Current.BlockableAdmins && (s.IsAdmin() || other.IsAdmin()) {
		return false
	} else {
		// Admins are blockable, unless they have the unblockable flag - e.g. if you have an admin chatbot on
		// your server it will send the `/unmute-all` command to still retain visibility into user messages for
		// auto-moderation. The `/unmute-all` sets the unblockable flag, so your admin chatbot still appears
		// on the Who's Online list as well.
		unblockable := (s.IsAdmin() && s.unblockable) || (other.IsAdmin() && other.unblockable)
		if unblockable {
			return false
		}
	}

	s.muteMu.RLock()
	defer s.muteMu.RUnlock()

	// Forward block?
	if _, ok := s.blocked[other.Username]; ok {
		return true
	}

	// Reverse block?
	other.muteMu.RLock()
	defer other.muteMu.RUnlock()
	_, ok := other.blocked[s.Username]
	return ok
}
