package messages

import (
	"sync"
	"time"
)

// Auto incrementing Message ID for anything pushed out by the server.
var (
	messageID = time.Now().Unix()
	mu        sync.Mutex
)

// NextMessageID atomically increments and returns a new MessageID.
func NextMessageID() int64 {
	mu.Lock()
	defer mu.Unlock()
	messageID++
	var mid = messageID
	return mid
}

/*
Message is the basic carrier of WebSocket chat protocol actions.

Every message (client or server) has an Action and the rest of the
fields may vary depending on the action. Many messages target (or carry)
a Username, chat Channel and carry an arbitrary Message.
*/
type Message struct {
	Action   string `json:"action,omitempty"`
	Channel  string `json:"channel,omitempty"`
	Username string `json:"username,omitempty"`
	Message  string `json:"message,omitempty"`

	// JWT token for `login` actions.
	JWTToken string `json:"jwt,omitempty"`

	// WhoList for `who` actions
	WhoList []WhoList `json:"whoList,omitempty"`

	// Sent on `me` actions along with Username
	VideoStatus int    `json:"video,omitempty"`  // user video flags
	ChatStatus  string `json:"status,omitempty"` // online vs. away
	DND         bool   `json:"dnd,omitempty"`    // Do Not Disturb, e.g. DMs are closed

	// Message ID to support takebacks/local deletions
	MessageID int64 `json:"msgID,omitempty"`

	// Sent on `open` actions along with the (other) Username.
	OpenSecret string `json:"openSecret,omitempty"`

	// Send on `file` actions, passing e.g. image data.
	Bytes []byte `json:"bytes,omitempty"`

	// Send on `blocklist` actions, for doing a `mute` on a list of users
	Usernames []string `json:"usernames,omitempty"`

	// Sent on `report` actions.
	Timestamp string `json:"timestamp,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Comment   string `json:"comment,omitempty"`

	// Sent on `echo` actions to condense multiple messages into one packet.
	Messages []Message `json:"messages,omitempty"`

	// Sent on `ping` messages (along with updated JWTToken)
	JWTRules map[string]bool `json:"jwtRules,omitempty"`

	// WebRTC negotiation messages: proxy their signaling messages
	// between the two users to negotiate peer connection.
	Candidate   string `json:"candidate,omitempty"`   // candidate
	Description string `json:"description,omitempty"` // sdp
}

const (
	// Actions sent by the client side only
	ActionLogin       = "login"  // post the username to backend
	ActionBoot        = "boot"   // boot a user off your video feed
	ActionUnboot      = "unboot" // unboot a user
	ActionMute        = "mute"   // mute a user's chat messages
	ActionUnmute      = "unmute"
	ActionBlock       = "block"        // hard block another user
	ActionBlocklist   = "blocklist"    // mute in bulk for usernames
	ActionReport      = "report"       // user reports a message
	ActionVideoInvite = "video-invite" // user invites another to watch their webcam

	// Actions sent by server or client
	ActionMessage  = "message"  // post a message to the room
	ActionEcho     = "echo"     // echo recent public message on join
	ActionMe       = "me"       // user self-info sent by FE or BE
	ActionOpen     = "open"     // user wants to view a webcam (open WebRTC)
	ActionRing     = "ring"     // receiver of a WebRTC open request
	ActionWatch    = "watch"    // user has received video and is watching you
	ActionUnwatch  = "unwatch"  // user has closed your video
	ActionFile     = "file"     // image sharing in chat
	ActionTakeback = "takeback" // user takes back (deletes) their message for everybody
	ActionReact    = "react"    // emoji reaction to a chat message
	ActionTyping   = "typing"   // typing indicator for DM threads

	// Actions sent by server only
	ActionPing     = "ping"
	ActionWhoList  = "who"        // server pushes the Who List
	ActionPresence = "presence"   // a user joined or left the room
	ActionCut      = "cut"        // tell the client to turn off their webcam
	ActionError    = "error"      // ChatServer errors
	ActionKick     = "disconnect" // client should disconnect (e.g. have been kicked).

	// WebRTC signaling messages.
	ActionCandidate = "candidate"
	ActionSDP       = "sdp"
)

// WhoList is a member entry in the chat room.
type WhoList struct {
	Username string `json:"username"`
	Nickname string `json:"nickname,omitempty"`
	Status   string `json:"status"`
	Video    int    `json:"video"`
	DND      bool   `json:"dnd,omitempty"`
	LoginAt  int64  `json:"loginAt"`

	// JWT auth extra settings.
	Operator   bool   `json:"op"`
	VIP        bool   `json:"vip,omitempty"`
	Avatar     string `json:"avatar,omitempty"`
	ProfileURL string `json:"profileURL,omitempty"`
	Emoji      string `json:"emoji,omitempty"`
	Gender     string `json:"gender,omitempty"`
}

// VideoFlags to convey the state and setting of users' cameras concisely.
// Also see the VideoFlag object in BareRTC.js for front-end sync.
const (
	VideoFlagActive         int = 1 << iota // user's camera is enabled/broadcasting
	VideoFlagNSFW                           // viewer's camera is marked as NSFW
	VideoFlagMuted                          // user source microphone is muted
	VideoFlagNonExplicit                    // viewer prefers not to see NSFW cameras (don't auto-open red cams/auto-close blue cams going red)
	VideoFlagMutualRequired                 // video wants viewers to share their camera too
	VideoFlagMutualOpen                     // viewer wants to auto-open viewers' cameras
	VideoFlagOnlyVIP                        // can only shows as active to VIP members
	VideoFlagInvited                        // user invites another to watch their camera
	VideoFlagNoWatermark                    // user opts out of QR code watermark protection
)

// Presence message templates.
const (
	PresenceJoined   = "has joined the room!"
	PresenceExited   = "has exited the room!"
	PresenceKicked   = "has been kicked from the room!"
	PresenceBanned   = "has been banned!"
	PresenceTimedOut = "has timed out!"
)
