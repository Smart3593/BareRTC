package client

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	reHTML = regexp.MustCompile(`<(.|\n)+?>`)
	reIMG  = regexp.MustCompile(`<img .+?>`)
)

// StripHTML removes HTML content from a message.
func StripHTML(s string) string {
	s = reIMG.ReplaceAllString(s, "inline embedded image")
	return strings.TrimSpace(reHTML.ReplaceAllString(s, ""))
}

// WebSocketURL converts the BareRTC base (https) URL into the WebSocket link.
func WebSocketURL(baseURL string) (string, error) {
	url, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	switch url.Scheme {
	case "https":
		return fmt.Sprintf("wss://%s/ws", url.Host), nil
	case "http":
		return fmt.Sprintf("ws://%s/ws", url.Host), nil
	case "ws", "wss":
		return fmt.Sprintf("%s//%s/ws", url.Scheme, url.Host), nil
	default:
		return "", errors.New("unsupported URL scheme")
	}
}

// AtMentioned checks if somebody has "at mentioned" your username (having your
// name at the beginning or end of their message). Returns whether the at mention
// was detected, along with the modified message without the at mention name on the
// end of it.
func AtMentioned(c *Client, message string) (bool, string) {
	// Patterns to look for.
	var (
		reAtMention = regexp.MustCompile(
			fmt.Sprintf(`(?i)^@?%s|@?%s$`, c.Username(), c.Username()),
		)
	)
	m := reAtMention.FindStringSubmatch(message)
	if m != nil {
		// Found a match! Sub off the at mentioned part and return.
		message = strings.TrimSpace(reAtMention.ReplaceAllString(message, ""))
		if len(message) > 0 {
			return true, message
		}
	}
	return false, message
}
