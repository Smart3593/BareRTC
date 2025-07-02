package barertc

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
	"git.kirsle.net/apps/barertc/pkg/util"
	"github.com/google/uuid"
)

// Polling user timeout before disconnecting them.
const PollingUserTimeout = time.Minute

// JSON payload structure for polling API.
type PollMessage struct {
	// Send the username after authenticated.
	Username string `json:"username,omitempty"`

	// SessionID for authentication.
	SessionID string `json:"session_id,omitempty"`

	// BareRTC protocol message.
	Message messages.Message `json:"msg"`
}

type PollResponse struct {
	// Session ID.
	Username  string `json:"username,omitempty"`
	SessionID string `json:"session_id,omitempty"`

	// Pending messages.
	Messages []messages.Message `json:"messages"`
}

// Helper method to send an error as a PollResponse.
func PollResponseError(message string) PollResponse {
	return PollResponse{
		Messages: []messages.Message{
			{
				Action:   messages.ActionError,
				Username: "ChatServer",
				Message:  message,
			},
		},
	}
}

// KickIdlePollUsers is a goroutine that will disconnect polling API users
// who haven't been seen in a while.
func (s *Server) KickIdlePollUsers() {
	log.Debug("KickIdlePollUsers goroutine engaged")
	for {
		time.Sleep(10 * time.Second)
		for _, sub := range s.IterSubscribers() {
			if sub.usePolling && time.Since(sub.lastPollAt) > PollingUserTimeout {
				// Send an exit message.
				if sub.authenticated && sub.ChatStatus != "hidden" {
					log.Error("KickIdlePollUsers: %s last seen %s ago", sub.Username, sub.lastPollAt)

					sub.authenticated = false
					s.Broadcast(messages.Message{
						Action:   messages.ActionPresence,
						Username: sub.Username,
						Message:  messages.PresenceTimedOut,
					})
					s.SendWhoList()
				}

				s.DeleteSubscriber(sub)
			}
		}
	}
}

// FlushPollResponse returns a response for the polling API that will flush
// all pending messages sent to the client.
func (sub *Subscriber) FlushPollResponse() PollResponse {
	var msgs = []messages.Message{}

	// Drain the messages from the outbox channel.
	for len(sub.messages) > 0 {
		message := <-sub.messages
		var msg messages.Message
		json.Unmarshal(message, &msg)
		msgs = append(msgs, msg)
	}

	return PollResponse{
		Username:  sub.Username,
		SessionID: sub.sessionID,
		Messages:  msgs,
	}
}

// Functions for the Polling API as an alternative to WebSockets.
func (s *Server) PollingAPI() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := util.IPAddress(r)

		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(PollResponseError("Only POST methods allowed"))
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(PollResponseError("Only application/json content-types allowed"))
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params PollMessage
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(PollResponseError(err.Error()))
			return
		}

		// Debug logging.
		log.Debug("Polling connection from %s - %s", ip, r.Header.Get("User-Agent"))

		// Are they resuming an authenticated session?
		var sub *Subscriber
		if params.Username != "" || params.SessionID != "" {
			if params.Username == "" || params.SessionID == "" {
				w.WriteHeader(http.StatusBadRequest)
				enc.Encode(PollResponseError("Authentication error: SessionID and Username both required."))
				return
			}

			log.Debug("Polling API: check if %s (%s) is authenticated", params.Username, params.SessionID)

			// Look up the subscriber.
			var (
				authOK bool
				err    error
			)
			sub, err = s.GetSubscriber(params.Username)
			if err == nil {
				// Validate the SessionID.
				if sub.sessionID == params.SessionID {
					authOK = true
				}
			}

			// Authentication error.
			if !authOK {
				s.DeleteSubscriber(sub)
				w.WriteHeader(http.StatusBadRequest)
				enc.Encode(PollResponse{
					Messages: []messages.Message{
						{
							Action:   messages.ActionError,
							Username: "ChatServer",
							Message:  "Your authentication has expired, please log back into the chat again.",
						},
						{
							Action: messages.ActionKick,
						},
					},
				})
				return
			}

			// Ping their last seen time.
			sub.lastPollAt = time.Now()
		}

		// If they are authenticated, handle this message.
		if sub != nil && sub.authenticated {
			s.OnClientMessage(sub, params.Message)

			// If they use JWT authentication, give them a ping back with an updated
			// JWT once in a while. Equivalent to the WebSockets pinger channel.
			if time.Since(sub.lastPollJWT) > PingInterval {
				sub.lastPollJWT = time.Now()
				sub.SendPing()
			}

			enc.Encode(sub.FlushPollResponse())
			return
		}

		// Not authenticated: the only acceptable message is login.
		if params.Message.Action != messages.ActionLogin {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(PollResponseError("Not logged in."))
			return
		}

		// Prepare a Subscriber object for them. Do not add it to the server
		// roster unless their login succeeds.
		ctx, cancel := context.WithCancel(r.Context())
		sub = s.NewPollingSubscriber(ctx, cancel)

		// Tentatively add them to the server. If they don't pass authentication,
		// remove their subscriber immediately. Note: they need added here so they
		// will receive their own "has entered the room" and WhoList updates.
		s.AddSubscriber(sub)

		s.OnLogin(sub, params.Message)

		// Are they authenticated?
		if sub.authenticated {
			// Generate a SessionID number.
			sessionID := uuid.New().String()
			sub.sessionID = sessionID

			log.Debug("Polling API: new user authenticated in: %s (sid %s)", sub.Username, sub.sessionID)
		} else {
			s.DeleteSubscriber(sub)
		}

		enc.Encode(sub.FlushPollResponse())
	})
}
