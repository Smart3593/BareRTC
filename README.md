# BareRTC

BareRTC is a simple WebRTC-based chat room application. It is especially designed to be plugged into any existing website, with or without a pre-existing base of users.

**Live demo:** [BareRTC Demo Chat](https://chat.kirsle.net/)

![Screenshot of BareRTC](screenshot.png)

It is very much in the style of the old-school Flash based webcam chat rooms of the early 2000's: a multi-user chat room with DMs and _some_ users may broadcast video and others may watch multiple video feeds in an asynchronous manner. I thought that this should be such an obvious free and open source app that should exist, but it did not and so I had to write it myself.

- [BareRTC](#barertc)
- [Installation](#installation)
- [Features](#features)
- [Configuration](#configuration)
- [Authentication](#authentication)
- [Moderator Commands](#moderator-commands)
- [JSON APIs](#json-apis)
- [Webhook URLs](#webhook-urls)
- [Chatbot](#chatbot)
- [Tour of the Codebase](#tour-of-the-codebase)
  - [Backend files](#backend-files)
  - [Frontend files](#frontend-files)
- [Deploying This App](#deploying-this-app)
- [Developing This App](#developing-this-app)
- [License](#license)

# Installation

See the [Install.md](./Install.md) for installation help.

# Features

* Specify multiple Public Channels that all users have access to.
* Users can open direct message (one-on-one) conversations with each other.
* No long-term server side state: messages are pushed out as they come in.
* Users may share pictures and GIFs from their computer, which are pushed out as `data:` URLs (images scaled and metadata stripped by server) directly to connected chatters with no storage required.
* Users may broadcast their webcam which shows a camera icon by their name in the Who List. Users may click on those icons to open multiple camera feeds of other users they are interested in.
    * Mutual webcam options: users may opt that anyone who views their cam must also be sharing their own camera first.
    * Users may mark their own cameras as explicit/NSFW which marks the icon in red so other users can get a warning before clicking in (if NSFW is enabled in the settings.toml)
    * Users may boot people off their camera, and to the booted person it appears the same as if the broadcaster had turned their camera off completely - the chat server lies about the camera status so the booted user can't easily tell they'd been booted.
* Mobile friendly: works best on iPads and above but adapts to smaller screens well.
* WebRTC means peer-to-peer video streaming so cheap on hosting costs!
* Simple integration with your existing userbase via signed JWT tokens.
* User configurable sound effects to be notified of DMs or users entering/exiting the room.
* Operator commands to kick, ban users, mark cameras NSFW, etc.

The BareRTC project also includes a [Chatbot implementation](docs/Chatbot.md) so you can provide an official chatbot for fun & games & to auto moderate your chat room!

# Configuration

On first run it will create the default settings.toml file for you which you may then customize to your liking:

```toml
Version = 2
Title = "BareRTC"
Branding = "BareRTC"
WebsiteURL = "https://www.example.com"
CORSHosts = ["https://www.example.com"]
PermitNSFW = true
UseXForwardedFor = true
WebSocketReadLimit = 41943040
MaxImageWidth = 1280
PreviewImageWidth = 360

[JWT]
  Enabled = false
  Strict = true
  SecretKey = ""

[[PublicChannels]]
  ID = "lobby"
  Name = "Lobby"
  Icon = "fa fa-gavel"
  WelcomeMessages = ["Welcome to the chat server!", "Please follow the basic rules:\n\n1. Have fun\n2. Be kind"]

[[PublicChannels]]
  ID = "offtopic"
  Name = "Off Topic"
  WelcomeMessages = ["Welcome to the Off Topic channel!"]
```

See [Configuration](docs/Configuration.md) for in-depth explanations on the available config settings and what they do.

# Authentication

BareRTC supports custom (user-defined) authentication with your app in the form of JSON Web Tokens (JWTs). JWTs will allow your existing app to handle authentication for users by signing a token that vouches for them, and the BareRTC app will trust your signed token.

See [Authentication](docs/Authentication.md) for more information.

# Moderator Commands

If you authenticate an Op user via JWT they can enter IRC-style chat commands to moderate the server. Current commands include:

* `/kick <username>` to disconnect a user's chat session.
* `/ban <username> [hours]` to ban a user from chat (temporary - time-based or until the next server reboot, default 24 hours)
* `/nsfw <username>` to tag a user's video feed as NSFW (if your settings.toml has PermitNSFW enabled).
* `/cut <username>` to 'cut' their webcam feed (instruct their web page to turn off their camera automatically)

There are easy buttons for the above commonly used actions in a user's pop-up "profile card" on the chat room.

Additional operator commands include:

* `/unban <username>` to lift the ban on a user.
* `/bans` to list all of the currently banned users.
* `/op <username>` to grant operator controls to a user (temporary, until they log off)
* `/deop <username>` to remove operator controls
* `/unmute-all` removes the mute flag on all users for the current operator (intended especially for the [Chatbot](docs/Chatbot.md) so it can still moderate public chat messages from users who have blocked it from your main website).

And there are some advanced commands intended for the server system administrator (these can be 'dangerous' and disruptive to users in the chat room):

* `/shutdown` will shut down the chat server (and hopefully, reboot it if your process supervisor is configured as such)
* `/reconfigure` will reload the server config file without needing to reboot.
* `/kickall` will kick ALL users from the room, with a message asking them to refresh the page (useful to deploy backwards-incompatible server updates where the new front-end is required to be loaded).

In case your operators forget, the `/help` command will list the common moderator commands and `/help-advanced` will list the more advanced/dangerous ones. **Note:** there is only one level of admin rights currently, so it will be a matter of policy to instruct your moderators not to play with the advanced commands.

# JSON APIs

BareRTC provides some API endpoints that your website can call over HTTP for better integration with your site. See [API](docs/API.md) for more information.

# Webhook URLs

BareRTC supports setting up webhook URLs so the chat server can call out to _your_ website in response to certain events, such as allowing users to send you reports about messages they receive on chat.

See [Webhooks](docs/Webhooks.md) for more information.

# Chatbot

The BareRTC project also comes with a chatbot program named BareBot which you can use to create your own bots for fun, games, and auto-moderator capabilities.

See [Chatbot](docs/Chatbot.md) for more information.

# Tour of the Codebase

This app uses WebSockets and WebRTC at the very simplest levels, without using a framework like `Socket.io`. Here is a tour of the codebase with the more interesting modules listed first.

## Backend files

* `cmd/BareRTC/main.go`: the entry point for the Go back-end application (parses command-line flags and starts the web server)
* `pkg/` contains the Go source code for the server side (the application).
    * `config/` handles the settings.toml config file for the app.
    * `jwt/` handles the JWT authentication logic
    * `log/` is an internal logger library - not very interesting
    * `util/` houses some miscellaneous utility functions, such as generating random strings or getting the user's IP address (w/ X-Forwarded-For support, etc.)
* `pkg/server.go` sets up the Go HTTP server and all the endpoint routes (e.g.: the /about page, static files, the WebSockets endpoint)
* `pkg/websocket.go` handles the WebSockets endpoint which drives 99% of the chat app (all the login, text chat, who list portions - not webcams). Some related files to this include:
    * `pkg/messages.go` is where I define the JSON message schema for the WebSockets protocol. Client and server messages marshal into the Message struct.
    * `pkg/handlers.go` is where I write "high level" chat event handlers (OnLogin, OnMessage, etc.) - the WebSocket read loop parses their message and then nicely calls my event handler based on action.
    * `pkg/commands.go` handles commands like /kick from moderators.
* `pkg/api.go` handles the JSON API endpoints from the web server.
* `pkg/pages.go` handles the index (w/ jwt parsing) and about pages.

## Frontend files

The `web/` folder holds front-end files and templates used by the Go app.

* `web/templates` holds Go html/template sources that are rendered server-side.
    * `chat.html` is the template for the main chat page (index route, `/`).
    * `about.html` is the template for the `/about` page.
* `web/static` holds the static files (scripts, stylesheets, images) for the site.
    * `js/BareRTC.js` does the whole front-end Vue.js app for BareRTC. The portions of the code that handle the WebSockets and WebRTC features are marked off with comment banners so you can scroll until you find them.
    * `js/sounds.js` handles the sound effects for the chat room.
    * `css/chat.css` has custom CSS for the chat room UI (mainly a lot of CSS Grid stuff for the full-screen layout).

Other front-end files are all vendored libraries or frameworks used by this app:

* [Bulma](https://bulma.io) CSS framework
* [Font Awesome](https://fontawesome.com) for icons.

# Deploying This App

It is recommended to use a reverse proxy such as nginx in front of this app. You will need to configure nginx to forward WebSocket related headers:

```nginx
server {
    server_name chat.example.com;
    listen 443 ssl http2;
    listen [::]:443 ssl http2;

    ssl_certificate /etc/letsencrypt/live/chat.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/chat.example.com/privkey.pem;

    # Proxy pass to BareRTC.
    location / {
        proxy_pass http://127.0.0.1:9000;

        # WebSocket headers to forward along.
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "Upgrade";
        proxy_set_header Host $host;
    }
}
```

You can run the BareRTC app itself using any service supervisor you like. I use [Supervisor](http://supervisord.org/introduction.html) and you can configure BareRTC like so:

```ini
# /etc/supervisor/conf.d/barertc.conf
[program:barertc]
command = /home/user/git/BareRTC/BareRTC -address 127.0.0.1:9000
directory = /home/user/git/BareRTC
user = user
```

Then `sudo supervisorctl reread && sudo supervisorctl add barertc` to start the app.

# Developing This App

In local development you'll probably run two processes in your terminal: one to `npm run watch` the Vue.js app and the other to run the Go server.

Building and running the front-end app:

```bash
# Install dependencies
npm install

# Build the front-end
npm run build

# Run the front-end in watch mode for local dev
npm run watch
```

And `make run` to run the Go server.

# License

GPLv3.
