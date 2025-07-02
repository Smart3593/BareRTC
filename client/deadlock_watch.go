package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"git.kirsle.net/apps/barertc/client/config"
	"git.kirsle.net/apps/barertc/pkg/log"
	"git.kirsle.net/apps/barertc/pkg/messages"
)

const deadlockTTL = time.Minute

/*
Deadlock detection for the chat server.

Part of the chatbot handlers. The bot will send DMs to itself on an interval
and test whether the server is responsive; if it goes down, it will issue the
/api/shutdown command to reboot the server automatically.

This function is a goroutine spawned in the background.
*/
func (h *BotHandlers) watchForDeadlock() {
	log.Info("Deadlock monitor engaged!")
	h.deadlockLastOK = time.Now()

	for {
		time.Sleep(15 * time.Second)
		go func() {
			h.client.Send(messages.Message{
				Action:  messages.ActionMessage,
				Channel: "@" + h.client.Username(),
				Message: fmt.Sprintf("deadlock ping %s", time.Now().Format(time.RFC3339)),
			})
		}()

		// Has it been a while since our last ping?
		if time.Since(h.deadlockLastOK) > deadlockTTL {
			log.Error("Deadlock detected! Rebooting the chat server!")
			h.deadlockLastOK = time.Now()
			h.rebootChatServer()
		}
	}
}

// onMessageFromSelf handles DMs sent to ourself, e.g. for deadlock detection.
func (h *BotHandlers) onMessageFromSelf(msg messages.Message) {
	// If it is our own DM channel thread, it's for deadlock detection.
	if msg.Channel == "@"+h.client.Username() {
		log.Info("(Deadlock test) got echo from self, server still seems OK: %s", msg.Message)
		h.deadlockLastOK = time.Now()
	}
}

// Reboot the chat server via web API, in case of deadlock.
func (h *BotHandlers) rebootChatServer() error {
	// API request struct for BareRTC /api/shutdown endpoint.
	var request = struct {
		APIKey string
	}{
		APIKey: config.Current.BareRTC.AdminAPIKey,
	}

	// JSON request body.
	jsonStr, err := json.Marshal(request)
	if err != nil {
		return err
	}

	// Make the API request to BareRTC.
	var url = strings.TrimSuffix(config.Current.BareRTC.URL, "/") + "/api/shutdown"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("RebootChatServer : erreur lors de la publication sur BareRTC : statut %d corps %s", resp.StatusCode, body)
	}

	return nil
}
