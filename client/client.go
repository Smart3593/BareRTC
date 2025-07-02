// Package client provides Go WebSocket client support for BareRTC.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"git.kirsle.net/apps/barertc/client/config"
	"git.kirsle.net/apps/barertc/pkg/jwt"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// HandlerFunc for WebSocket chat protocol events.
type HandlerFunc func(messages.Message)

// Client represents a WebSocket client connection to BareRTC.
type Client struct {
	// Event handlers for your app to respond to.
	OnWho        HandlerFunc // Who's Online
	OnMe         HandlerFunc // Status updates for current user sent by server
	OnMessage    HandlerFunc
	OnTakeback   HandlerFunc
	OnReact      HandlerFunc
	OnPresence   HandlerFunc
	OnRing       HandlerFunc
	OnOpen       HandlerFunc
	OnWatch      HandlerFunc
	OnUnwatch    HandlerFunc
	OnCut        HandlerFunc
	OnError      HandlerFunc
	OnDisconnect HandlerFunc
	OnPing       HandlerFunc
	OnCandidate  HandlerFunc
	OnSDP        HandlerFunc

	// Private state variables.
	url    string
	jwt    string // JWT token
	claims jwt.Claims
	ctx    context.Context
	conn   *websocket.Conn
}

// NewClient initializes the WebSocket connection (JWT claims required).
//
// URL is like ws://localhost:9000/ws
func NewClient(url string, claims jwt.Claims) (*Client, error) {
	// Sanity check the claims.
	if claims.Subject == "" {
		return nil, errors.New("missing Subject field of JWT claims")
	}
	return &Client{
		url:    url,
		claims: claims,
	}, nil
}

// Run the client, connecting to the WebSocket and returning only on error or disconnect.
func (c *Client) Run() error {
	// Authenticate.
	if token, err := c.Authenticate(); err != nil {
		return fmt.Errorf("n'a pas obtenu de jeton JWT de BareRTC : %s", err)
	} else {
		c.jwt = token
	}

	// Get the WebSocket URL.
	wss, err := WebSocketURL(c.url)
	if err != nil {
		return fmt.Errorf("impossible d'obtenir l'URL WebSocket depuis %s : %s", c.url, err)
	}

	ctx := context.Background()
	c.ctx = ctx

	conn, _, err := websocket.Dial(ctx, wss, nil)
	if err != nil {
		return fmt.Errorf("composition de l'URL WebSocket (%s) : %s", c.url, err)
	}
	c.conn = conn
	defer conn.Close(websocket.StatusInternalError, "the sky is falling")

	conn.SetReadLimit(config.Current.WebSocketReadLimit)

	// Authenticate via JWT token.
	if err := c.Send(messages.Message{
		Action:   messages.ActionLogin,
		Username: "testbot",
		JWTToken: c.jwt,
	}); err != nil {
		return fmt.Errorf("envoi du message de connexion : %s", err)
	}

	// Enter the Read Loop
	for {
		var msg messages.Message
		err := wsjson.Read(c.ctx, c.conn, &msg)
		if err != nil {
			log.Error("wsjson.Read: %s", err)
			break
		}

		// Handle the various protocol messages.
		switch msg.Action {
		case messages.ActionWhoList:
			c.Handle(msg, c.OnWho)
		case messages.ActionMe:
			c.Handle(msg, c.OnMe)
		case messages.ActionMessage:
			c.Handle(msg, c.OnMessage)
		case messages.ActionReact:
			c.Handle(msg, c.OnReact)
		case messages.ActionPresence:
			c.Handle(msg, c.OnPresence)
		case messages.ActionRing:
			c.Handle(msg, c.OnRing)
		case messages.ActionOpen:
			c.Handle(msg, c.OnOpen)
		case messages.ActionWatch:
			c.Handle(msg, c.OnWatch)
		case messages.ActionUnwatch:
			c.Handle(msg, c.OnUnwatch)
		case messages.ActionCut:
			c.Handle(msg, c.OnCut)
		case messages.ActionError:
			c.Handle(msg, c.OnError)
		case messages.ActionKick:
			c.Handle(msg, c.OnDisconnect)
		case messages.ActionPing:
			c.Handle(msg, c.OnPing)
		case messages.ActionCandidate:
			c.Handle(msg, c.OnCandidate)
		case messages.ActionSDP:
			c.Handle(msg, c.OnSDP)
		default:
			log.Error("Unsupported chat protocol message type: %s", msg.Action)
		}
	}

	conn.Close(websocket.StatusNormalClosure, "")

	return errors.New("disconnected")
}

// Send a WebSocket message.
func (c *Client) Send(msg messages.Message) error {
	return wsjson.Write(c.ctx, c.conn, msg)
}

// Username returns the bot's username.
func (c *Client) Username() string {
	return c.claims.Subject
}

// Handle a WebSocket message. This is called internally on the read loop.
// It basically passes the message into the HandlerFunc, or returns an
// error if the HandlerFunc is nil (not defined).
//
// Note: handler funcs are run on a background goroutine, so they can be
// free to use time.Sleep and delay message sending if needed.
func (c *Client) Handle(msg messages.Message, fn HandlerFunc) error {
	if fn == nil {
		return fmt.Errorf("aucun gestionnaire défini pour les messages '%s'", msg.Action)
	}
	go fn(msg)
	return nil
}

// Authenticate with the BareRTC server, returning a signed JWT token.
//
// This posts to the /api/authenticate endpoint on the BareRTC Web API. It
// is called automatically as part of the logon process in Run().
func (c *Client) Authenticate() (string, error) {
	// API request struct for BareRTC /api/blocklist endpoint.
	var request = struct {
		APIKey string
		Claims jwt.Claims
	}{
		APIKey: config.Current.BareRTC.AdminAPIKey,
		Claims: c.claims,
	}

	// Response struct
	type response struct {
		OK    bool
		Error string
		JWT   string
	}

	// JSON request body.
	jsonStr, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Make the API request to BareRTC.
	var url = strings.TrimSuffix(config.Current.BareRTC.URL, "/") + "/api/authenticate"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("SendBlocklist : erreur de synchronisation de la liste de blocage avec BareRTC : statut %d corps %s", resp.StatusCode, body)
	}

	// Return the signed JWT token.
	var result response
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if result.JWT == "" {
		return "", errors.New("did not get JWT token from BareRTC")
	}

	return result.JWT, err
}
