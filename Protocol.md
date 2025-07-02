# Chat Protocol

The primary communication channel for the chat is WebSockets between the ChatServer and ChatClient (the front-end web page).

The protocol was made up as it went and here is some (hopefully current) documentation of what the different message types and contents look like.

Messages are delivered as JSON objects in both directions.

# WebRTC Workflow

WebRTC only kicks in when a user wants to see the webcam stream shared by a broadcaster. Simply turning on your webcam doesn't start any WebRTC stuff - it's when somebody clicks to see your cam, or you click to see somebody else's.

Since the WebRTC workflow is always triggered by _somebody_ clicking on the video icon to open a broadcaster's camera we will start there.

The one who initiates the connection is called the **offerer** (they send the first offer to connect) and the one sharing video is the **answerer** in WebRTC parlance.

1. The offerer clicks the video button to begin the process. This sends an [open](#open) message to the server.
2. The server echoes the [open](#open) back to the offerer and sends a [ring](#ring) to the answerer, to let them know that the offerer wants to connect.
    * For the answerer, the `ring` message triggers the "has opened your camera" notice in chat.
3. Both the offerer and answerer will use the server to negotiate a WebRTC peer connection.
    * WebRTC is a built-in browser standard and the two browsers will negotiate "ICE candidates" and "session description protocol" (SDP) messages.
    * The [candidate](#candidate) and [sdp](#sdp) actions on the chat server allow simple relaying of these messages between browsers.
    * The answerer adds their video stream to the RTC PeerConnection so that once they are established, the offerer receives the video.
4. When connectivity is established, the offerer sends a [watch](#watch) message which the server passes to the answerer, so that their username apprars in the Watching list.

The video stream can be interrupted and closed via various methods:

* When the answerer turns off their camera, they close all RTC PeerConnections with the offerers.
* When the PeerConnection is closed, the offerer deletes the `<video>` widget from the page (turning off the camera feed).
* Also, a `who` update that says a person's videoActive went false will instruct all clients who had the video open, to close it (in case the PeerConnection closure didn't already do this).
* If a user exits the room, e.g. exited their browser abruptly without gracefully closing PeerConnections, any client who had their video open will close it immediately.

# Video Flags

The various video settings sent on Who List updates are now consolidated
to a bit flag field:

```javascript
VideoFlag: {
    Active:         1 << 0,  // or 00000001 in binary
    NSFW:           1 << 1,  // or 00000010
    Muted:          1 << 2,  // or 00000100, etc.
    NonExplicit:    1 << 3,
    MutualRequired: 1 << 4,
    MutualOpen:     1 << 5,
    OnlyVIP:        1 << 6,
    Invited:        1 << 7,
}
```

# WebSocket Message Actions

Every message has an "action" and may have other fields depending on the action type.

## Login

Sent by: Client.

Log in to the chat room. Looks like:

```javascript
// Client login
{
    "action": "login",
    "username": "soandso",
    "jwt": "jwt token string (if used)"
}
```

If JWT authentication is enabled on the server, the ChatClient sends the JWT token to the server for validation.

## Disconnect

Sent by: Server.

The server tells the client to disconnect and not come back, e.g.: they have been kicked or banned by an operator.

```javascript
// Server disconnect
{
    "action": "disconnect"
}
```

## Ping

Sent by: Server, Client.

Just a keep-alive message to prevent the WebSocket connection from closing.

```javascript
{
    "action": "ping"
}
```

When the server sends a ping to a user who authenticated via JWT token, it will also provide a refreshed JWT token with an updated expiration date (in case the user is disconnected, so they can log back in gracefully), along with a mapping of the chat moderation rules in their token. Example:

```javascript
{
    "action": "ping",
    "jwt": "new.jwt.token",
    "jwtRules": {
        "IsRedCamRule": false,
        "IsNoVideoRule": false,
        //...
    }
}
```

## Error

Sent by: Server.

Send an error message which will appear as a ChatServer error in chat.

```javascript
{
    "action": "error",
    "message": "Something went wrong!",
    "username": "ChatServer",
    "channel": "lobby"
}
```

## Message

Sent by: Client, Server.

The client sends this to post a text message to a chat channel:

```javascript
// Client message to public channel
{
    "action": "message",
    "channel": "lobby",
    "message": "Hello everyone!"
}

// Client message to DM
{
    "action": "message",
    "channel": "@target",
    "message": "Hi"
}
```

If this is a DM, the channel will begin with an `@` symbol followed by the username, like `"channel": "@target"`

The server sends a similar message to push chats to the client:

```javascript
// Server message
{
    "action": "message",
    "channel": "lobby",
    "username": "senderName",
    "message": "Hello!",
    "msgID": 123
}
```

If the message is a DM, the channel will be the username prepended by an @ symbol and the ChatClient will add it to the appropriate DM thread (creating a new DM thread if needed).

Every message or file share originated from a user has a "msgID" attached
which is useful for [takebacks](#takeback).

## Echo

Sent by: Server.

This supports the feature for the server to echo recent public messages that took place before a user joined chat.

It is basically a wrapper around a list of Messages that happened in these channels.

```javascript
// Server message
{
    "action": "echo",
    "messages": [
        {
            "action": "message",
            "channel": "lobby",
            "username": "senderName",
            "message": "Hello!",
            "msgID": 123,
            "timestamp": "2024-01-01 00:00:00"
        }
    ]
}
```

A notable feature compared to how those Messages originally were sent is that the echoed ones carry a "timestamp" since it is sending outdated messages to the client.

## File

Sent by: Client.

The client is posting an image to share in chat.

```javascript
// Client file.
{
    "action": "file",
    "channel": "lobby",
    "bytes": new Uint8Array()
}
```

The server will massage and validate the image data and then send it to others in the chat via a normal `message` containing an `<img>` tag with a data: URL - directly passing the image data to other chatters without needing to store it somewhere with a public URL.

## Takeback

Sent by: Client, Server.

The takeback message is how a user can delete their previous message from
everybody else's display. Operators may also take back messages sent by
other users.

```javascript
{
    "action": "takeback",
    "msgID": 123
}
```

Every message or file share initiated by a real user (not ChatClient or
ChatServer) is assigned an auto-incrementing message ID, and the chat
server records which message IDs "belong" to which user (so that a
modded chat client or bot can't takeback other peoples' messages without
operator rights).

When the front-end receives a takeback, it searches all channels to
delete the message with that ID.

## Presence

Sent by: Server.

The `presence` message is just like a `message` but is designed for join/exit chat events.

```javascript
// Server message
{
    "action": "presence",
    "username": "soandso",
    "message": "has joined the room!"
}
```

## Me

Sent by: Client, Server.

The "me" action communicates the user's current state and settings to the server. It will usually also trigger a "who" action to refresh the Who List for all chatters.

The client sends "me" messages to send their webcam broadcast status and NSFW flag:

```javascript
// Client Me
{
    "action": "me",
    "video": 1,
}
```

The server may also push "me" messages to the user: for example if there is a conflict in username and the server has changed your username:

```javascript
// Server Me
{
    "action": "me",
    "username": "soandso 12345",
    "video": 1,
}
```

## Who

Sent by: Server.

The `who` action sends the Who Is Online list to all connected chatters.

```javascript
// Server Who
{
    "action": "who",
    "whoList": [
        {
            "username": "soandso",
            "op": false, // operator status
            "avatar": "/picture/soandso.png",
            "profileURL": "/u/soandso",
            "video": 0,
        }
    ]
}
```

## Open

Sent by: Client, Server.

This command is sent when a viewer wants to **open** the webcam of a broadcaster and see their video.

```javascript
// Client Open
{
    "action": "open",
    "username": "target"
}
```

The server echos the `open` command back at the person who initiated it:

```javascript
// Server Open
{
    "action": "open",
    "username": "target",
    "openSecret": "random string (not actually used)"
}
```

And for the one sharing their webcam, sends a `ring` message.

## Ring

Sent by: Server.

This is sent to the user who is sharing their webcam, to notify them that a viewer wants to connect.

```javascript
// Server Ring
{
    "action": "ring",
    "username": "viewer",
    "openSecret": "random string (not actually used)"
}
```

The user will then initiate a WebRTC peer-to-peer connection with the viewer to share their video to them.

## Watch, Unwatch

Sent by: Client, Server.

When a viewing client successfully receives video frames from the sender, they send a `watch` command to update the sender's Watching list, and will send an `unwatch` command when they close the video.

The server passes the watch/unwatch message to the broadcaster.

```javascript
{
    "action": "watch",
    "username": "viewer"
}
```

## Cut

Sent by: Server.

The server tells the client to turn off their camera. This is done in response to a `/cut` command being sent by an admin user: to remotely cause another user on chat to turn off their camera and stop broadcasting.

```javascript
// Server Cut
{
    "action": "cut"
}
```

## Video Invite

Sent by: Client.

The invite-video command allows the client to whitelist permission for other users on chat to watch their webcam, even if normally those other users would not be allowed.

For example: if the user has the mutual webcam option enabled (to require their viewers to be on camera too before watching), the invite-video command will allow a user who is not on webcam to watch anyway.

On the front-end app, this manifests as showing a green outline around the video button (with the video icon itself being blue or red as normal).

Note: multiple usernames can be sent. This is so if the user experiences a temporary disconnect, their page can re-sync the list over when they reconnect.

```javascript
// Client Video Invite
{
    "action": "video-invite",
    "usernames": [ "alice", "bob" ]
}
```

The server does not send any response to this message. Instead, to the target users, the sender's video flag will have the Invited bit set on WhoList updates.

## Mute, Unmute

Sent by: Client.

The mute command tells the server that you are muting the user.

```javascript
// Client Mute
{
    "action": "mute",
    "username": "target"
}
```

When the user is muted:

* The server will lie about your camera status on `who` messages to that user, always showing your camera as not active.
* If they were already watching your camera, they see that you have turned your camera off and they disconnect.
* The server will not send you any `message` from that user.

The `unmute` action does the opposite and removes the mute status:

```javascript
// Client Unmute
{
    "action": "unmute",
    "username": "target"
}
```

## Block

Sent by: Client, Server.

The block command places a hard block between the current user and the target.

When either user blocks the other:

* They do not see each other in the Who's Online list at all.
* They can not see each other's messages, including presence messages.

**Note:** the chat page currently does not have a front-end button to block a user. This feature is currently used by the Blocklist feature to apply a block to a set of users at once upon join.

```javascript
// Client Block
{
    "action": "block",
    "username": "target"
}
```

The server may send a "block" message to the client in response to the BlockNow API endpoint: your main website can communicate that a block was just added, so if either user is currently in chat the block can apply immediately instead of at either user's next re-join of the room.

The server "block" message follows the same format, having the username of the other party.

## Blocklist

Sent by: Client.

The blocklist command is basically a bulk block for (potentially) many usernames at once.

```javascript
// Client blocklist
{
    "action": "blocklist",
    "usernames": [ "target1", "target2", "target3" ]
}
```

How this works: if you have an existing website and use JWT authentication to sign users into chat, your site can pre-emptively sync the user's block list **before** the user enters the room, using the `/api/blocklist` endpoint (see the README.md for BareRTC).

The chat server holds onto blocklists temporarily in memory: when that user loads the chat room (with a JWT token!), the front-end page receives the cached blocklist. As part of the "on connected" handler, the chat page sends the `blocklist` command over WebSocket to perform a mass block on these users in one go.

The reason for this workflow is in case the chat server is rebooted _while_ the user is in the room. The cached blocklist pushed by your website is forgotten by the chat server back-end, but the client's page was still open with the cached blocklist already, and it will send the `blocklist` command to the server when it reconnects, eliminating any gaps.

## Boot, Unboot

Sent by: Client.

This command is to kick a viewer off of your webcam and block them from opening your webcam again.

```javascript
// Client Boot
{
    "action": "boot",
    "username": "target"
}
```

When a user is booted:

* They are kicked off your camera.
* The chat server lies to them about your camera status on future `who` messages - showing that your camera is not running.

Note: it is designed that the person being booted off can not detect that they have been booted. They will see your RTC PeerConnection close + get a Who List that says you are not sharing video - exactly the same as if you had simply turned off your camera completely.

There is also a client side Unboot command, to undo the effects of a boot:

```javascript
// Client Unboot
{
    "action": "unboot",
    "username": "target"
}
```

## WebRTC Signaling

Sent by: Client, Server.

The `candidate` and `sdp` actions are used as part of WebRTC signaling negotiations where the two browsers (the broadcaster and viewer) try and connect to share video.

```javascript
// Candidate
{
    "action": "candidate",
    "username": "otherUser",
    "candidate": "..."
}

// SDP
{
    "action": "sdp",
    "username": "otherUser",
    "description": "..."
}
```

The server simply proxies the message between the two parties.
