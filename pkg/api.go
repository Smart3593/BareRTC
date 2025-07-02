package barertc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
	"git.kirsle.net/apps/barertc/pkg/jwt"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
	"git.kirsle.net/apps/barertc/pkg/models"
)

// Statistics (/api/statistics) returns info about the users currently logged onto the chat,
// for your website to call via CORS. The URL to your site needs to be in the CORSHosts array
// of your settings.toml.
func (s *Server) Statistics() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle the CORS header from your trusted domains.
		if origin := r.Header.Get("Origin"); origin != "" {
			var found bool
			for _, allowed := range config.Current.CORSHosts {
				if allowed == origin {
					found = true
				}
			}

			if found {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		var result = struct {
			UserCount int
			Usernames []string
			Cameras   struct {
				Blue int
				Red  int
			}
		}{
			Usernames: []string{},
		}

		// Count all users + collect unique usernames.
		var unique = map[string]struct{}{}
		for _, sub := range s.IterSubscribers() {
			if sub.authenticated && sub.ChatStatus != "hidden" {
				result.UserCount++
				if _, ok := unique[sub.Username]; ok {
					continue
				}
				result.Usernames = append(result.Usernames, sub.Username)
				unique[sub.Username] = struct{}{}

				// Count cameras by color.
				if sub.VideoStatus&messages.VideoFlagActive == messages.VideoFlagActive {
					if sub.VideoStatus&messages.VideoFlagNSFW == messages.VideoFlagNSFW {
						result.Cameras.Red++
					} else {
						result.Cameras.Blue++
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		enc.Encode(result)
	})
}

// Authenticate (/api/authenticate) for the chatbot API.
//
// This endpoint will sign a JWT token using the claims you pass in. It requires
// the shared secret `AdminAPIKey` from your settings.toml and will sign the
// JWT claims you give it.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"APIKey": "from settings.toml",
//		"Claims": {
//			"sub": "username",
//			"nick": "Display Name",
//			"op": false,
//			"img": "/static/photos/avatar.png",
//			"url": "/users/username",
//			"emoji": "🤖",
//			"gender": "m"
//		}
//	}
//
// The return schema looks like:
//
//	{
//		"OK": true,
//		"Error": "error string, omitted if none",
//		"JWT": "jwt token string"
//	}
func (s *Server) Authenticate() http.HandlerFunc {
	type request struct {
		APIKey string
		Claims jwt.Claims
	}

	type result struct {
		OK    bool
		Error string `json:",omitempty"`
		JWT   string `json:",omitempty"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Validate the API key.
		if params.APIKey != config.Current.AdminAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			enc.Encode(result{
				Error: "Authentification refusée.",
			})
			return
		}

		// Encode the JWT token.
		var claims = params.Claims
		token, err := claims.ReSign()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			enc.Encode(result{
				Error: "Erreur lors de la signature des revendications JWT.",
			})
			return
		}

		enc.Encode(result{
			OK:  true,
			JWT: token,
		})
	})
}

// Shutdown (/api/shutdown) the chat server, hopefully to reboot it.
//
// This endpoint is equivalent to the operator '/shutdown' command but may be
// invoked by your website, or your chatbot. It requires the AdminAPIKey.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"APIKey": "from settings.toml",
//	}
//
// The return schema looks like:
//
//	{
//		"OK": true,
//		"Error": "error string, omitted if none",
//	}
func (s *Server) ShutdownAPI() http.HandlerFunc {
	type request struct {
		APIKey string
	}

	type result struct {
		OK    bool
		Error string `json:",omitempty"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Validate the API key.
		if params.APIKey != config.Current.AdminAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			enc.Encode(result{
				Error: "Authentification refusée.",
			})
			return
		}

		// Send the response.
		enc.Encode(result{
			OK: true,
		})

		// Defer a shutdown a moment later.
		go func() {
			time.Sleep(2 * time.Second)
			os.Exit(1)
		}()

		// Attempt to broadcast, but if deadlocked this might not go out.
		go func() {
			s.Broadcast(messages.Message{
				Action:   messages.ActionError,
				Username: "ChatServer",
				Message:  "Le serveur de chat va redémarrer MAINTENANT !",
			})
		}()
	})
}

// Amend JWT (/api/amend-jwt) token for a user who may be logged into chat.
//
// This endpoint allows your website to send updates to a user's JWT token, such
// as to change chat moderator rules or to swap their profile picture URL or
// change their nickname/username.
//
// It is a POST request with a json body containing the following schema:
//
//		{
//			"APIKey": "from settings.toml",
//	     "Username": "alice",
//	     "JWTToken": "new.jwt.token"
//		}
//
// The "Username" should match the username they currently would have in the
// chat room, and the JWT token is a new auth token to replace their current
// JWT with. You can change any properties in the new JWT token. For example,
// if you include a new username in the JWT token, the chat server will rename
// them (this will probably interrupt their DM threads people have open with
// them, however).
//
// The return schema looks like:
//
//	{
//		"OK": true,
//		"Error": "error string, omitted if none",
//	}
func (s *Server) AmendJWT() http.HandlerFunc {
	type request struct {
		APIKey   string
		Username string
		JWTToken string
	}

	type result struct {
		OK    bool
		Error string `json:",omitempty"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Validate the API key.
		if params.APIKey != config.Current.AdminAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			enc.Encode(result{
				Error: "Authentification refusée.",
			})
			return
		}

		// Validate the newly given JWT token.
		var claims *jwt.Claims
		if parsed, ok, err := jwt.ParseAndValidate(params.JWTToken); err != nil || !ok {
			w.WriteHeader(http.StatusUnauthorized)
			enc.Encode(result{
				Error: "Jeton JWT invalide.",
			})
			return
		} else {
			claims = parsed
		}

		// Look for this username on chat.
		sub, err := s.GetSubscriber(params.Username)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			enc.Encode(result{
				Error: fmt.Sprintf("L'utilisateur %s n'est pas connecté", params.Username),
			})
			return
		}

		// If the subscriber is JWT authenticated, send them the new token.
		if sub.JWTClaims != nil {
			sub.JWTClaims = claims
			sub.SendPing()
			sub.SendMe()
			s.SendWhoList()
		}

		// Send the response.
		enc.Encode(result{
			OK: true,
		})
	})
}

// BlockList (/api/blocklist) allows your website to pre-sync mute lists between your
// user accounts, so that when they see each other in chat they will pre-emptively mute
// or boot one another.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"APIKey": "from settings.toml",
//		"Username": "soandso",
//		"Blocklist": [ "list", "of", "other", "usernames" ],
//	}
//
// The chat server will remember these mappings (until rebooted). How they are
// used is that the blocklist is embedded in the front-end page when the username
// signs in later. As part of the On Connect handler, the front-end will send the
// list of usernames in a bulk `mute` command to the server. This way even if the
// chat server reboots while the user is connected, when it comes back up and the user
// reconnects they will retransmit their block list.
func (s *Server) BlockList() http.HandlerFunc {
	type request struct {
		APIKey    string
		Username  string
		Blocklist []string
	}

	type result struct {
		OK    bool
		Error string `json:",omitempty"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Validate the API key.
		if params.APIKey != config.Current.AdminAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			enc.Encode(result{
				Error: "Authentification refusée.",
			})
			return
		}

		// Store the cached blocklist.
		SetCachedBlocklist(params.Username, params.Blocklist)
		enc.Encode(result{
			OK: true,
		})
	})
}

// BlockNow (/api/block/now) allows your website to add to a current online chatter's
// blocked list immediately.
//
// For example: the BlockList endpoint does a bulk sync of the blocklist at the time
// a user joins the chat room, but if users are already on chat when the blocking begins,
// it doesn't take effect until one or the other re-joins the room. This API endpoint
// can apply the blocking immediately to the currently online users.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"APIKey": "from settings.toml",
//		"Usernames": [ "source", "target" ]
//	}
//
// The pair of usernames will be the two users who block one another (in any order).
// If any of the users are currently connected to the chat, they will all mutually
// block one another immediately.
func (s *Server) BlockNow() http.HandlerFunc {
	type request struct {
		APIKey    string
		Usernames []string
	}

	type result struct {
		OK    bool
		Error string `json:",omitempty"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Validate the API key.
		if params.APIKey != config.Current.AdminAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			enc.Encode(result{
				Error: "Authentification refusée.",
			})
			return
		}

		// Check if any of these users are online, and update their blocklist accordingly.
		var changed bool
		for _, username := range params.Usernames {
			if sub, err := s.GetSubscriber(username); err == nil {
				for _, otherName := range params.Usernames {
					if username == otherName {
						continue
					}
					log.Info("BlockNow API: %s is currently on chat, add block for %+v", username, otherName)

					sub.muteMu.Lock()
					sub.muted[otherName] = struct{}{}
					sub.blocked[otherName] = struct{}{}
					sub.muteMu.Unlock()

					// Changes have been made to online users.
					changed = true

					// Send a server-side "block" command to the subscriber, so their front-end page might
					// update the cachedBlocklist so there's no leakage in case of chat server rebooting.
					sub.SendJSON(messages.Message{
						Action:   messages.ActionBlock,
						Username: otherName,
					})
				}
			}
		}

		// If any changes to blocklists were made: send the Who List.
		if changed {
			s.SendWhoList()
		}

		enc.Encode(result{
			OK: true,
		})
	})
}

// DisconnectNow (/api/disconnect/now) allows your website to remove a user from
// the chat room if they are currently online.
//
// For example: a user on your website has deactivated their account, and so
// should not be allowed to remain in the chat room.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"APIKey": "from settings.toml",
//		"Usernames": [ "alice", "bob" ],
//		"Message": "An optional ChatServer message to send them first.",
//		"Kick": false,
//	}
//
// The `Message` parameter, if provided, will be sent to that user as a
// ChatServer error before they are removed from the room. You can use this
// to provide them context as to why they are being kicked. For example:
// "You have been logged out of chat because you deactivated your profile on
// the main website."
//
// The `Kick` boolean is whether the removal should manifest to other users
// in chat as a "kick" (sending a presence message of "has been kicked from
// the room!"). By default (false), BareRTC will tell the user to disconnect
// and it will manifest as a regular "has left the room" event to other online
// chatters.
func (s *Server) DisconnectNow() http.HandlerFunc {
	type request struct {
		APIKey    string
		Usernames []string
		Message   string
		Kick      bool
	}

	type result struct {
		OK      bool
		Removed int
		Error   string `json:",omitempty"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Validate the API key.
		if params.APIKey != config.Current.AdminAPIKey {
			w.WriteHeader(http.StatusUnauthorized)
			enc.Encode(result{
				Error: "Authentification refusée.",
			})
			return
		}

		// Check if any of these users are online, and disconnect them from the chat.
		var removed int
		for _, username := range params.Usernames {
			if sub, err := s.GetSubscriber(username); err == nil {
				// Broadcast to everybody that the user left the chat.
				message := messages.PresenceExited
				if params.Kick {
					message = messages.PresenceKicked
				}
				s.Broadcast(messages.Message{
					Action:   messages.ActionPresence,
					Username: username,
					Message:  message,
				})

				// Custom message to send to them?
				if params.Message != "" {
					sub.ChatServer(params.Message)
				}

				// Disconnect them.
				sub.SendJSON(messages.Message{
					Action: messages.ActionKick,
				})
				sub.authenticated = false
				sub.Username = ""

				removed++
			}
		}

		// If any changes to blocklists were made: send the Who List.
		if removed > 0 {
			s.SendWhoList()
		}

		enc.Encode(result{
			OK:      true,
			Removed: removed,
		})
	})
}

// UserProfile (/api/profile) fetches profile information about a user.
//
// This endpoint will proxy to your WebhookURL for the "profile" endpoint.
// If your webhook is not configured or not reachable, this endpoint returns
// an error to the caller.
//
// Authentication: the caller must send their current chat JWT token when
// hitting this endpoint.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"JWTToken": "the caller's jwt token",
//		"Username": [ "soandso" ]
//	}
//
// The response JSON will look like the following (this also mirrors the
// response json as sent by your site's webhook URL):
//
//	{
//		"OK": true,
//	    "Error": "only on errors",
//	    "ProfileFields": [
//			{
//				"Name": "Age",
//				"Value": "30yo",
//			},
//			{
//				"Name": "Gender",
//				"Value": "Man",
//			},
//			...
//		]
//	}
func (s *Server) UserProfile() http.HandlerFunc {
	type request struct {
		JWTToken string
		Username string
	}

	type profileField struct {
		Name  string
		Value string
	}
	type result struct {
		OK            bool
		Error         string         `json:",omitempty"`
		Headline      string         `json:",omitempty"`
		ProfileFields []profileField `json:",omitempty"`
	}

	type webhookRequest struct {
		Action   string
		APIKey   string
		Username string
	}

	type webhookResponse struct {
		StatusCode int
		Data       result
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Are JWT tokens enabled on the server?
		if !config.Current.JWT.Enabled || params.JWTToken == "" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "JWT authentication is not available.",
			})
			return
		}

		// Validate the user's JWT token.
		_, _, err := jwt.ParseAndValidate(params.JWTToken)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Fetch the profile data from your website.
		data, err := PostWebhook("profile", webhookRequest{
			Action:   "profile",
			APIKey:   config.Current.AdminAPIKey,
			Username: params.Username,
		})
		if err != nil {
			log.Error("Couldn't get profile information: %s", err)
		}

		// Success? Try and parse the response into our expected format.
		var resp webhookResponse
		if err := json.Unmarshal(data, &resp); err != nil {
			w.WriteHeader(http.StatusInternalServerError)

			// A nice error message?
			if resp.Data.Error != "" {
				enc.Encode(result{
					Error: resp.Data.Error,
				})
			} else {
				enc.Encode(result{
					Error: fmt.Sprintf("Didn't get expected response for profile data: %s", err),
				})
			}
			return
		}

		// At this point the expected resp mirrors our own, so return it.
		if resp.StatusCode != http.StatusOK || resp.Data.Error != "" {
			w.WriteHeader(http.StatusInternalServerError)
		}
		enc.Encode(resp.Data)
	})
}

// MessageHistory (/api/message/history) fetches past direct messages for a user.
//
// This endpoint looks up earlier chat messages between the current user and a target.
// It will only run with a valid JWT auth token, to protect users' privacy.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"JWTToken": "the caller's jwt token",
//		"Username": "other party",
//		"BeforeID": 1234,
//	}
//
// The "BeforeID" parameter is for pagination and is optional: by default the most
// recent page of messages are returned. To retrieve an older page, the BeforeID will
// contain the MessageID of the oldest message you received so far, so that the message
// before that will be the first returned on the next page.
//
// The response JSON will look like the following:
//
//	{
//		"OK": true,
//		"Error": "only on error responses",
//		"Messages": [
//			{
//				// Standard BareRTC Message objects...
//				"MessageID": 1234,
//				"Username": "other party",
//				"Message": "hello!",
//			}
//		],
//		"Remaining": 42,
//	}
//
// The Remaining value is how many older messages still exist to be loaded.
func (s *Server) MessageHistory() http.HandlerFunc {
	type request struct {
		JWTToken string
		Username string
		BeforeID int64
	}

	type result struct {
		OK        bool
		Error     string `json:",omitempty"`
		Messages  []messages.Message
		Remaining int
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Are JWT tokens enabled on the server?
		if !config.Current.JWT.Enabled || params.JWTToken == "" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "JWT authentication is not available.",
			})
			return
		}

		// Validate the user's JWT token.
		claims, _, err := jwt.ParseAndValidate(params.JWTToken)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Get the user from the chat roster.
		sub, err := s.GetSubscriber(claims.Subject)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "You are not logged into the chat room.",
			})
			return
		}

		// Fetch a page of message history.
		messages, remaining, err := models.PaginateDirectMessages(sub.Username, params.Username, params.BeforeID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		enc.Encode(result{
			OK:        true,
			Messages:  messages,
			Remaining: remaining,
		})
	})
}

// MessageUsernameHistory (/api/message/usernames) fetches past conversation threads for a user.
//
// This endpoint will paginate the distinct usernames that the current user has
// conversation history with. It drives the 'History' button that enables users to
// load previous chats even if their chat partner is not presently online.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"JWTToken": "the caller's jwt token",
//		"Sort": "newest",
//		"Page": 1
//	}
//
// The "Sort" parameter takes the following options:
//
// - "newest": the most recently updated threads are first (default).
// - "oldest": the oldest threads are first.
// - "a-z": sort usernames ascending.
// - "z-a": sort usernames descending.
//
// The response JSON will look like the following:
//
//	{
//		"OK": true,
//		"Error": "only on error responses",
//		"Usernames": [
//			"alice",
//			"bob"
//		],
//		"Pages": 42,
//	}
//
// The Remaining value is how many older messages still exist to be loaded.
func (s *Server) MessageUsernameHistory() http.HandlerFunc {
	type request struct {
		JWTToken string
		Sort     string
		Page     int
	}

	type result struct {
		OK        bool
		Error     string `json:",omitempty"`
		Usernames []string
		Count     int
		Pages     int
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Are JWT tokens enabled on the server?
		if !config.Current.JWT.Enabled || params.JWTToken == "" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "JWT authentication is not available.",
			})
			return
		}

		// Validate the user's JWT token.
		claims, _, err := jwt.ParseAndValidate(params.JWTToken)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Get the user from the chat roster.
		sub, err := s.GetSubscriber(claims.Subject)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "You are not logged into the chat room.",
			})
			return
		}

		// Fetch a page of message history.
		usernames, count, pages, err := models.PaginateUsernames(sub.Username, params.Sort, params.Page, 9)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		enc.Encode(result{
			OK:        true,
			Usernames: usernames,
			Count:     count,
			Pages:     pages,
		})
	})
}

// ClearMessages (/api/message/clear) deletes all the stored direct messages for a user.
//
// It can be called by the authenticated user themself (with JWTToken), or from your website
// (with APIKey) in which case you can remotely clear history for a user.
//
// It is a POST request with a json body containing the following schema:
//
//	{
//		"JWTToken": "the caller's jwt token",
//		"APIKey": "your website's admin API key"
//		"Username": "if using your APIKey to specify a user to delete",
//	}
//
// The response JSON will look like the following:
//
//	{
//		"OK": true,
//		"Error": "only on error responses",
//		"MessagesErased": 123,
//	}
//
// The Remaining value is how many older messages still exist to be loaded.
func (s *Server) ClearMessages() http.HandlerFunc {
	type request struct {
		JWTToken string
		APIKey   string
		Username string
	}

	type result struct {
		OK             bool
		Error          string `json:",omitempty"`
		MessagesErased int    `json:""`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// JSON writer for the response.
		w.Header().Set("Content-Type", "application/json")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")

		// Parse the request.
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seules les méthodes POST sont autorisées",
			})
			return
		} else if r.Header.Get("Content-Type") != "application/json" {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: "Seuls les types de contenu application/json sont autorisés",
			})
			return
		}

		defer r.Body.Close()

		// Parse the request payload.
		var (
			params request
			dec    = json.NewDecoder(r.Body)
		)
		if err := dec.Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		// Authenticate this request.
		if params.APIKey != "" {
			// By admin API key.
			if params.APIKey != config.Current.AdminAPIKey {
				w.WriteHeader(http.StatusUnauthorized)
				enc.Encode(result{
					Error: "Authentification refusée.",
				})
				return
			}
		} else {
			// Are JWT tokens enabled on the server?
			if !config.Current.JWT.Enabled || params.JWTToken == "" {
				w.WriteHeader(http.StatusBadRequest)
				enc.Encode(result{
					Error: "JWT authentication is not available.",
				})
				return
			}

			// Validate the user's JWT token.
			claims, _, err := jwt.ParseAndValidate(params.JWTToken)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				enc.Encode(result{
					Error: err.Error(),
				})
				return
			}

			// Set the username to clear.
			params.Username = claims.Subject
		}

		// Erase their message history.
		count, err := (models.DirectMessage{}).ClearMessages(params.Username)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			enc.Encode(result{
				Error: err.Error(),
			})
			return
		}

		enc.Encode(result{
			OK:             true,
			MessagesErased: count,
		})
	})
}

// Blocklist cache sent over from your website.
var (
	// Map of username to the list of usernames they block.
	cachedBlocklist   map[string][]string
	cachedBlocklistMu sync.RWMutex
)

func init() {
	cachedBlocklist = map[string][]string{}
}

// GetCachedBlocklist returns the blocklist for a username.
func GetCachedBlocklist(username string) []string {
	cachedBlocklistMu.RLock()
	defer cachedBlocklistMu.RUnlock()
	if list, ok := cachedBlocklist[username]; ok {
		log.Debug("GetCachedBlocklist(%s) blocks %s", username, list)
		return list
	}
	log.Debug("GetCachedBlocklist(%s): no blocklist stored", username)
	return []string{}
}

// SetCachedBlocklist sets the blocklist cache for a user.
func SetCachedBlocklist(username string, blocklist []string) {
	log.Info("SetCachedBlocklist: %s mutes users %s", username, strings.Join(blocklist, ", "))
	cachedBlocklistMu.Lock()
	defer cachedBlocklistMu.Unlock()
	cachedBlocklist[username] = blocklist
}
