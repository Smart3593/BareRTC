package barertc

import (
	"sync"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
)

// Functionality for storing recent public channel messages and echo them to new joiners.

/*
Echo Messages On Join

This feature stores recent public messages to channels (in memory) to echo them
back to new users when they join the room.
*/
var (
	echoMessages = map[string][]messages.Message{} // map channel ID -> messages
	echoLock     sync.RWMutex
)

// SendEchoedMessages will repeat recent public messages in public channels to the newly
// connecting subscriber as echoed messages.
func (sub *Subscriber) SendEchoedMessages() {
	var echoes []messages.Message

	// Gather the subscriber's block list, so we don't echo users who are on it.
	var blocks = map[string]interface{}{}
	sub.muteMu.RLock()
	for username := range sub.blocked {
		blocks[username] = nil
	}
	for username := range sub.muted {
		blocks[username] = nil
	}
	sub.muteMu.RUnlock()

	// Read lock to collect the messages.
	echoLock.RLock()

	for _, msgs := range echoMessages {
		for _, msg := range msgs {
			if _, ok := blocks[msg.Username]; ok {
				continue
			}
			echoes = append(echoes, msg)
		}
	}

	// Release the lock.
	echoLock.RUnlock()

	// Send all of these in one Echo message.
	sub.SendJSON(messages.Message{
		Action:   messages.ActionEcho,
		Messages: echoes,
	})
}

// EchoPushPublicMessage pushes a message into the recent message history of the channel ID.
//
// The buffer of recent messages (the size configured in settings.toml) is echoed to
// a new user when they join the chat so they can catch up.
func (s *Server) EchoPushPublicMessage(sub *Subscriber, channel string, msg messages.Message) {

	// Get the channel from settings to see its capacity.
	ch, ok := config.Current.GetChannel(channel)
	if !ok {
		return
	}

	echoLock.Lock()
	defer echoLock.Unlock()

	// Initialize the storage for this channel?
	if _, ok := echoMessages[channel]; !ok {
		echoMessages[channel] = []messages.Message{}
	}

	// Timestamp it and append this message.
	msg.Timestamp = time.Now().Format(time.RFC3339)
	echoMessages[channel] = append(echoMessages[channel], msg)

	// Trim the history to the configured window size.
	if ln := len(echoMessages[channel]); ln > ch.EchoMessagesOnJoin {
		echoMessages[channel] = echoMessages[channel][ln-ch.EchoMessagesOnJoin:]
	}
}

// EchoTakebackMessage will remove any taken-back message that was cached
// in the echo buffer for new joiners.
func (s *Server) EchoTakebackMessage(msgID int64) {

	// Takebacks are relatively uncommon enough, write lock while we read and/or
	// maybe remove messages from the echo cache.
	echoLock.Lock()
	defer echoLock.Unlock()

	// Find matching messages in each channel.
	for _, ch := range config.Current.PublicChannels {
		for i, msg := range echoMessages[ch.ID] {
			if msg.MessageID == msgID {
				log.Error("EchoTakebackMessage: message ID %d removed from channel %s", msgID, ch.ID)

				// Remove this message.
				echoMessages[ch.ID] = append(echoMessages[ch.ID][:i], echoMessages[ch.ID][i+1:]...)
			}
		}
	}
}
