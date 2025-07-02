package spam

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"git.kirsle.net/apps/barertc/pkg/config"
)

/*
LinkSpamMap keeps track of link spamming behavior per username.

It is a map of usernames to their recent history of hyperlinks sent to other usernames
over DMs. The intention is to detect when one user is spamming (copy/pasting) the same
hyperlink to many many people over DMs, e.g., if they are trying to take the whole chat
room away to a competing video conference and they are sending the link by DM to everybody
on chat in order to hide from the moderators by not using the public channels.
*/
type LinkSpamMap map[string]UserLinkMap

// LinkSpam holds info about possibly spammy hyperlinks that Username has sent to multiple
// others over Direct Messages, to detect e.g. somebody spamming a link to their off-site
// video conference to everybody on chat while hiding from moderators and public channels.
type LinkSpam struct {
	Username  string
	URL       string
	SentTo    map[string]struct{} // Usernames they have sent it to
	FirstSent time.Time           // time of the first link
	LastSent  time.Time           // time of the most recently sent link
	Lock      sync.RWMutex
	Kicked    bool // user was kicked once for this spam
}

// UserLinkMap connects usernames to the set of distinct links they have sent.
//
// It is a map of the URL hash to the LinkSpam data struct.
type UserLinkMap map[string]*LinkSpam

// LinkSpamManager is the singleton global instance variable that checks and tracks link
// spam sent by users on chat. It is initialized at server startup and provides an API
// surface area to ping and test for spammy behavior from users.
var (
	LinkSpamManager LinkSpamMap
	linkSpamLock    sync.Mutex // protects the top-level map from concurrent writes.
)

var HyperlinkRegexp = regexp.MustCompile(`(?:http[s]?:\/\/.)?(?:www\.)?[-a-zA-Z0-9@%._\+~#=]{2,256}\.[a-z]{2,6}\b(?:[-a-zA-Z0-9@:%_\+.~#?&\/\/=]*)`)

func init() {
	LinkSpamManager = map[string]UserLinkMap{}
	go LinkSpamManager.expire()
}

/*
Check if the current user has been spamming a link to too many people over DMs.

This function will parse the message for any hyperlinks, and upsert/ping the
sourceUsername's spam detection struct.

If the user has pasted the same hyperlink into too many different DM threads,
this function may return one of two sentinel errors:

- ErrLinkSpamKickUser if the user should be kicked from the room.
- ErrLinkSpamBanUser if the user should be banned from the room.

The first time they trip the spam limit they are to be kicked, but if they rejoin
the chat and (still within the spam duration window), paste the same link to yet
another recipient they will be banned temporarily from chat.
*/
func (m LinkSpamMap) Check(sourceUsername, targetUsername, message string) error {

	if !config.Current.DMLinkSpamProtection.Enabled {
		return nil
	}

	linkSpamLock.Lock()
	defer linkSpamLock.Unlock()

	// Initialize data structures.
	if _, ok := m[sourceUsername]; !ok {
		m[sourceUsername] = UserLinkMap{}
	}

	// Parse all URLs from their message.
	matches := HyperlinkRegexp.FindAllStringSubmatch(message, -1)
	for _, match := range matches {
		var (
			url  = match[0]
			hash = Hash([]byte(url))
		)

		// Initialize the struct?
		if _, ok := m[sourceUsername][hash]; !ok {
			m[sourceUsername][hash] = &LinkSpam{
				Username:  sourceUsername,
				URL:       url,
				SentTo:    map[string]struct{}{},
				FirstSent: time.Now(),
			}
		}

		// Check and update information.
		spam := m[sourceUsername][hash]
		spam.Lock.Lock()
		defer spam.Lock.Unlock()

		spam.SentTo[targetUsername] = struct{}{}
		spam.LastSent = time.Now()

		// Have they sent it to too many people?
		if len(spam.SentTo) > config.Current.DMLinkSpamProtection.MaxThreads {
			// Kick or ban them.
			if spam.Kicked {
				return ErrLinkSpamBanUser
			}
			spam.Kicked = true
			return ErrLinkSpamKickUser
		}
	}

	return nil
}

// expire cleans up link spam data after the rate limit window for them had passed.
//
// It runs as a background goroutine and periodically cleans up expired link spam.
func (m LinkSpamMap) expire() {
	for {
		time.Sleep(5 * time.Minute)

		// Lock the top-level struct for cleanup.
		linkSpamLock.Lock()

		// Iterate all users who have links stored.
		var cleanupUsernames = []string{}
		for username, links := range m {
			var cleanupHashes = []string{}
			for hash, spam := range links {
				// Has this record expired based on its LastSent time?
				if time.Since(spam.LastSent) > config.Current.DMLinkSpamProtection.TimeLimit*time.Second {
					cleanupHashes = append(cleanupHashes, hash)
				}
			}

			// Clean up the hashes.
			if len(cleanupHashes) > 0 {
				for _, hash := range cleanupHashes {
					delete(links, hash)
				}
			}

			// Are any left anymore?
			if len(links) == 0 {
				cleanupUsernames = append(cleanupUsernames, username)
			}
		}

		// Clean up empty usernames?
		if len(cleanupUsernames) > 0 {
			for _, username := range cleanupUsernames {
				delete(m, username)
			}
		}

		// Unlock the struct.
		linkSpamLock.Unlock()
	}
}

// Sentinel errors returned from LinkSpamMan.Check().
var (
	ErrLinkSpamKickUser = errors.New(
		`<strong>Spam Detected:</strong> ` +
			`You have pasted the same URL link to too many different people in a row, and this has been flagged as spam.<br><br>` +
			`You will now be kicked from the chat room. You may refresh and log back in, however, if you continue to spam this ` +
			`link one more time, you <strong>will be temporarily banned from the chat room.</strong>`,
	)
	ErrLinkSpamBanUser = errors.New(
		`<strong>Spam Detected:</strong> ` +
			`You recently were kicked from the chat room because you had already pasted this link to too many different people. ` +
			`You were warned that spamming this link one more time would result in a temporary ban from the chat room.<br><br>` +
			`<strong>You are now (temporarily) banned from the chat room.</strong>`,
	)
)

// Hash a byte array as SHA256 and returns the hex string.
func Hash(input []byte) string {
	h := sha256.New()
	h.Write(input)
	return fmt.Sprintf("%x", h.Sum(nil))
}
