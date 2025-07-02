package barertc

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/messages"
)

// Functionality for handling server-side message filtering and reporting.

// filterMessage will check an incoming user message against the configured
// server-side filters and react accordingly. This function also is
// responsible for collecting the recent contexts (10 messages per channel).
//
// Parameters: the rawMsg is their (pre-Markdown-formatted) original message
// (for the message context); the msg pointer is their post-formatted one, which
// may be modified to censor their word before returning.
//
// Returns the matching message filter (or nil) and a boolean (matched).
func (s *Server) filterMessage(sub *Subscriber, rawMsg messages.Message, msg *messages.Message) (*config.MessageFilter, bool) {
	// Collect the recent channel context first.
	if strings.HasPrefix(msg.Channel, "@") {
		// DM
		pushDirectMessageContext(sub, sub.Username, msg.Channel[1:], rawMsg)

		// If either party is an admin user, waive filtering this DM chat.
		if sub.IsAdmin() {
			return nil, false
		} else if other, err := s.GetSubscriber(msg.Channel[1:]); err == nil && other.IsAdmin() {
			return nil, false
		}
	} else {
		// Public channel
		pushMessageContext(sub, msg.Channel, rawMsg)
	}

	// Check it against the configured filters.
	var matched bool
	for _, filter := range config.Current.MessageFilters {
		if !filter.Enabled {
			continue
		}

		for _, phrase := range filter.IterPhrases() {
			m := phrase.FindAllStringSubmatch(msg.Message, -1)
			for _, match := range m {
				// Found a match!
				matched = true

				// Censor it?
				if filter.CensorMessage {
					msg.Message = strings.ReplaceAll(msg.Message, match[0], strings.Repeat("*", len(match[0])))
				}
			}
		}

		if matched {
			return filter, true
		}
	}

	return nil, false
}

// Report the filtered message along with recent context.
func (s *Server) reportFilteredMessage(sub *Subscriber, msg messages.Message) error {
	if !WebhookEnabled(WebhookReport) {
		return errors.New("report webhook is not enabled on this server")
	}

	// Prepare the report.
	var context string
	if strings.HasPrefix(msg.Channel, "@") {
		context = getDirectMessageContext(sub.Username, msg.Channel[1:])
	} else {
		context = getMessageContext(msg.Channel)
	}

	if _, err := PostWebhook(WebhookReport, WebhookRequest{
		Action: WebhookReport,
		APIKey: config.Current.AdminAPIKey,
		Report: WebhookRequestReport{
			FromUsername:  sub.Username,
			AboutUsername: sub.Username,
			Channel:       msg.Channel,
			Timestamp:     time.Now().Format(time.RFC1123),
			Reason:        "Server Side Message Filter",
			Message:       msg.Message,
			Comment: fmt.Sprintf(
				"This is an automated report via server side chat filters.\n\n"+
					"The recent context in this channel included the following conversation:\n\n"+
					"%s",
				context,
			),
		},
	}); err != nil {
		return err
	}

	return nil
}

// Message Context Caching
//
// Hold the recent (10) messages for each channel so in case of automated
// reporting, the context can be delivered in the report.
var (
	messageContexts    = map[string][]string{}
	messageContextMu   sync.RWMutex
	messageContextSize = 30
)

// Push a message onto the recent messages context.
func pushMessageContext(sub *Subscriber, channel string, msg messages.Message) {
	messageContextMu.Lock()
	defer messageContextMu.Unlock()

	// Initialize the context for new channel the first time.
	if _, ok := messageContexts[channel]; !ok {
		messageContexts[channel] = []string{}
	}

	// Append this message to it.
	messageContexts[channel] = append(messageContexts[channel], fmt.Sprintf(
		"%s [%s] %s",
		time.Now().Format("2006-01-02 15:04:05"),
		sub.Username,
		strings.TrimSpace(msg.Message),
	))

	// Trim the context to recent messages only.
	if len(messageContexts[channel]) > messageContextSize {
		messageContexts[channel] = messageContexts[channel][len(messageContexts[channel])-messageContextSize:]
	}
}

// Push a message context for DMs. A channel name will be derived consistently
// based on the sorted pair of usernames.
func pushDirectMessageContext(sub *Subscriber, username1, username2 string, msg messages.Message) {
	var names = []string{username1, username2}
	sort.Strings(names)
	pushMessageContext(
		sub,
		fmt.Sprintf("@%s", strings.Join(names, ":")),
		msg,
	)
}

// Get the recent message context, pretty printed.
func getMessageContext(channel string) string {
	messageContextMu.RLock()
	defer messageContextMu.RUnlock()

	if _, ok := messageContexts[channel]; !ok {
		return "(No recent message history in this channel)"
	}

	return strings.Join(messageContexts[channel], "\n\n")
}

func getDirectMessageContext(username1, username2 string) string {
	var names = []string{username1, username2}
	sort.Strings(names)
	return getMessageContext(
		fmt.Sprintf("@%s", strings.Join(names, ":")),
	)
}
