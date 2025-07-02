package barertc

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

/* Functions to handle banned users */

// Ban is an entry on the ban list.
type Ban struct {
	Username  string
	ExpiresAt time.Time
}

// Global storage for banned users in memory.
var (
	banList   = map[string]Ban{}
	banListMu sync.RWMutex
)

// BanUser adds a user to the ban list.
func (s *Server) BanUser(username string, duration time.Duration) {
	banListMu.Lock()
	defer banListMu.Unlock()
	banList[username] = Ban{
		Username:  username,
		ExpiresAt: time.Now().Add(duration),
	}
}

// UnbanUser lifts the ban of a user early.
func UnbanUser(username string) bool {
	banListMu.RLock()
	defer banListMu.RUnlock()
	_, ok := banList[username]
	if ok {
		delete(banList, username)
	}
	return ok
}

// StringifyBannedUsers returns a stringified list of all the current banned users.
func StringifyBannedUsers() string {
	var lines = []string{}
	banListMu.RLock()
	defer banListMu.RUnlock()
	for username, ban := range banList {
		lines = append(lines, fmt.Sprintf(
			"* `%s` banned until %s",
			username,
			ban.ExpiresAt.Format(time.RFC3339),
		))
	}
	return strings.Join(lines, "\n")
}

// IsBanned returns whether the username is currently banned.
func IsBanned(username string) bool {
	banListMu.Lock()
	defer banListMu.Unlock()
	ban, ok := banList[username]
	if ok {
		// Has the ban expired?
		if time.Now().After(ban.ExpiresAt) {
			delete(banList, username)
			return false
		}
	}
	return ok
}

// BanExpires returns the time.Duration string until the user's ban expires.
func BanExpires(username string) string {
	banListMu.RLock()
	defer banListMu.RUnlock()
	ban, ok := banList[username]
	if ok {
		return time.Until(ban.ExpiresAt).String()
	}
	return ""
}
