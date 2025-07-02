# Configuration

On first run it will create the default settings.toml file for you which you may then customize to your liking:

```toml
Version = 7
Title = "BareRTC"
Branding = "BareRTC"
WebsiteURL = "http://www.example.com"
CORSHosts = ["http://www.example.com"]
AdminAPIKey = "e635e463-7987-4788-94f3-671a5c2a589f"
PermitNSFW = true
UseXForwardedFor = false
WebSocketReadLimit = 40971520
MaxImageWidth = 1280
PreviewImageWidth = 360

[JWT]
  Enabled = true
  Strict = true
  SecretKey = "05c45344-1c52-430b-beb9-c3f64ff7ed12"
  LandingPageURL = "https://www.example.com/enter-chat"

[TURN]
  URLs = ["stun:stun.l.google.com:19302"]
  Username = ""
  Credential = ""

[[PublicChannels]]
  ID = "lobby"
  Name = "Lobby"
  Icon = "fa fa-gavel"
  WelcomeMessages = ["Welcome to the chat server!", "Please follow the basic rules:\n\n1. Have fun\n2. Be kind"]

[[PublicChannels]]
  ID = "offtopic"
  Name = "Off Topic"
  WelcomeMessages = ["Welcome to the Off Topic channel!"]

[[WebhookURLs]]
  Name = "report"
  Enabled = true
  URL = "https://www.example.com/v1/barertc/report"

[[WebhookURLs]]
  Name = "profile"
  Enabled = true
  URL = "https://www.example.com/v1/barertc/profile"

[VIP]
  Name = "VIP"
  Branding = "<em>VIP Members</em>"
  Icon = "fa fa-circle"
  MutuallySecret = false

[[MessageFilters]]
  Enabled = true
  PublicChannels = true
  PrivateChannels = true
  KeywordPhrases = [
    "\\bswear words\\b",
    "\\b(swearing|cursing)\\b",
    "suck my ([^\\s]+)"
  ]
  CensorMessage = true
  ForwardMessage = false
  ReportMessage = false
  ChatServerResponse = "Watch your language."

[[ModerationRule]]
  Username = "example"
  CameraAlwaysNSFW = true
  NoBroadcast = false
  NoVideo = false
  NoImage = false

[DirectMessageHistory]
  Enabled = true
  SQLiteDatabase = "database.sqlite"
  RetentionDays = 90
  DisclaimerMessage = "Reminder: please conduct yourself honorable in DMs."

[Logging]
  Enabled = true
  Directory = "./logs"
  Channels = ["lobby"]
  Usernames = []
```

A description of the config directives includes:

## Website Settings

* **Version** number for the settings file itself. When new features are added, the Version will increment and your settings.toml will be written to disk with sensible defaults filled in for the new options.
* **Title** goes in the title bar of the chat page.
* **Branding** is the title shown in the corner of the page. HTML is permitted here! You may write an `<img>` tag to embed an image or use custom markup to color and prettify your logo.
* **WebsiteURL** is the base URL of your actual website which is used in a couple of places:
    * The About page will link to your website.
    * If using [JWT authentication](#authentication), avatar and profile URLs may be relative (beginning with a "/") and will append to your website URL to safe space on the JWT token size!
* **CORSHosts** names HTTP hosts for Cross Origin Resource Sharing. Usually, this will be the same as your WebsiteURL. This feature is used with the [Web API](API.md) if your front-end page needs to call e.g. the /api/statistics endpoint on BareRTC.
* **AdminAPIKey** is a shared secret authentication key for the admin API endpoints.
* **PermitNSFW**: for user webcam streams, expressly permit "NSFW" content if the user opts in to mark their feed as such. Setting this will enable pop-up modals regarding NSFW video and give broadcasters an opt-in button, which will warn other users before they click in to watch.
* **UseXForwardedFor**: set it to true and (for logging) the user's remote IP will use the X-Real-IP header or the first address in X-Forwarded-For. Set this if you run the app behind a proxy like nginx if you want IPs not to be all localhost.
* **WebSocketReadLimit**: sets a size limit for WebSocket messages - it essentially also caps the max upload size for shared images (add a buffer as images will be base64 encoded on upload).
* **MaxImageWidth**: for pictures shared in chat the server will resize them down to no larger than this width for the full size view.
* **PreviewImageWidth**: to not flood the chat, the image in chat is this wide and users can click it to see the MaxImageWidth in a lightbox modal.

## JWT Authentication

Settings for JWT [Authentication](#authentication):

* **Enabled** (bool): activate the JWT token authentication feature.
* **Strict** (bool): if true, **only** valid signed JWT tokens may log in. If false, users with no/invalid token can enter their own username without authentication.
* **SecretKey** (string): the JWT signing secret shared with your back-end app.

## Public Channels

Settings for the default public text channels of your room.

* **ID** (string): an arbitrary 'username' for the chat channel, like "lobby".
* **Name** (string): the user friendly name for the channel, like "Off Topic"
* **Icon** (string, optional): CSS class names for FontAwesome icon for the channel, like "fa fa-message"
* **WelcomeMessages** ([]string, optional): messages that are delivered by ChatServer to the user when they connect to the server. Useful to give an introduction to each channel, list its rules, etc.

## VIP Status

If using JWT authentication, your website can mark some users as VIPs when sending them over to the chat. The `[VIP]` section of settings.toml lets you customize the branding and behavior in BareRTC:

* **Name** (string): what you call your VIP users, used in mouse-over tooltips.
* **Branding** (string): HTML supported, this will appear in webcam sharing modals to "make my cam only visible to fellow VIP users"
* **Icon** (string): icon CSS name from Font Awesome.
* **MutuallySecret** (bool): if true, the VIP features are hidden and only visible to people who are, themselves, VIP. For example, the icon on the Who List will only show to VIP users but non-VIP will not see the icon.

## Message Filters

BareRTC supports optional server-side filtering of messages. These can be applied to monitor public channels, Direct Messages, or both; and provide a variety of options how you want to handle filtered messages.

You can configure multiple sets of filters to treat different sets of keywords with different behaviors.

Options for the `[[MessageFilters]]` section include:

* **Enabled** (bool): whether to enable this filter. The default settings.toml has a filter template example by default, but it's not enabled.
* **PublicChannels** (bool): whether to apply the filter to public channel messages.
* **PrivateChannels** (bool): whether to apply the filter to private (Direct Message) channels.
* **KeywordPhrases** ([]string): a listing of regular expression compatible strings to search the user's message again.
    * Tip: use word-boundary `\b` metacharacters to detect whole words and reduce false positives from partial word matches.
* **CensorMessage** (bool): if true, the matching keywords will be substituted with asterisks in the user's message when it appears in chat.
* **ForwardMessage** (bool): whether to repeat the message to the other chatters. If false, the sender will see their own message echo (possibly censored) but other chatters will not get their message at all.
* **ReportMessage** (bool): if true, report the message along with the recent context (previous 10 messages in that conversation) to your website's report webhook (if configured).
* **ChatServerResponse** (str): optional - you can have ChatServer send a message to the sender (in the same channel) after the filter has been run. An empty string will not send a ChatServer message.

## Moderation Rules

This section of the config file allows you to place certain moderation rules on specific users of your chat room. For example: if somebody perpetually needs to be reminded to label their camera as NSFW, you can enforce a moderation rule on that user which _always_ forces their camera to be NSFW.

Settings in the `[[ModerationRule]]` array include:

* **Username** (string): the username on chat to apply the rule to.
* **CameraAlwaysNSFW** (bool): if true, the user's camera is forced to NSFW and they will receive a ChatServer message when they try and remove the flag themselves.
* **NoBroadcast** (bool): if true, the user is not allowed to share their webcam and the server will send them a 'cut' message any time they go live, along with a ChatServer message informing them of this.
* **NoVideo** (bool): if true, the user is not allowed to broadcast their camera OR watch any camera on chat.
* **NoImage** (bool): if true, the user is not allowed to share images or see images shared by others on chat.

### JWT Moderation Rules

Rather than in the server-side settings.toml, you can enable these moderation rules from your website's side as well by including them in the "rules" custom key of your JWT token.

The "rules" key is a string array with short labels representing each of the rules:

| Moderation Rule  | JWT "Rules" Value |
|------------------|-------------------|
| CameraAlwaysNSFW | redcam            |
| NoBroadcast      | nobroadcast       |
| NoVideo          | novideo           |
| NoImage          | noimage           |

An example JWT token claims object may look like:

```javascript
{
    "sub": "username",                    // Username for chat
    "nick": "Display name",               // Friendly name
    "img": "/static/photos/username.jpg", // user picture URL
    "url": "/u/username",                 // user profile URL
    "rules": ["redcam", "noimage"],       // moderation rules
}
```

## Direct Message History

You can allow BareRTC to retain temporary DM history for your users so they can remember where they left off with people.

Settings for this include:

* **Enabled** (bool): set to true to log chat DMs history.
* **SQLiteDatabase** (string): the name of the .sqlite DB file to store their DMs in.
* **RetentionDays** (int): how many days of history to record before old chats are erased. Set to zero for no limit.
* **DisclaimerMessage** (string): a custom banner message to show at the top of DM threads. HTML is supported. A good use is to remind your users of your local site rules.

## Logging

This feature can enable logging of public channels and user DMs to text files on disk. It is useful to keep a log of your public channels so you can look back at the context of a reported public chat if you weren't available when it happened, or to selectively log the DMs of specific users to investigate a problematic user.

Settings include:

* **Enabled** (bool): to enable or disable the logging feature.
* **Directory** (string): a folder on disk to save logs into. Public channels will save directly as text files here (e.g. "lobby.txt"), while DMs will create a subfolder for the monitored user.
* **Channels** ([]string): array of public channel IDs to monitor.
* **Usernames** ([]string): array of chat usernames to monitor.