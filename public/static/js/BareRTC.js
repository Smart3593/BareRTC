// console.log("BareRTC!");

// WebRTC configuration.
const configuration = {
    iceServers: TURN.URLs.map(val => {
        let row = {
            urls: val,
        };

        if (val.indexOf('turn:') === 0) {
            row.username = TURN.Username;
            row.credential = TURN.Credential;
        }

        return row;
    })
};

const FileUploadMaxSize = 1024 * 1024 * 8; // 8 MB


function setModalImage(url) {
    let $modalImg = document.querySelector("#modalImage"),
        $modal = document.querySelector("#photo-modal");
    $modalImg.src = url;
    $modal.classList.add("is-active");
    return false;
}

// Popped-out video drag functions.



const app = Vue.createApp({
    delimiters: ['[[', ']]'],
    data() {
        return {
            // busy: false, // TODO: not used
            disconnect: false,    // don't try to reconnect (e.g. kicked)
            windowFocused: true,  // browser tab is active
            windowFocusedAt: new Date(),

            // Disconnect spamming: don't retry too many times.
            disconnectLimit: 3,
            disconnectCount: 0,

            // Temp: spam counting for OF links
            spamWarningCount: 0,

            // Website configuration provided by chat.html template.
            config: {
                channels: PublicChannels,
                website: WebsiteURL,
                permitNSFW: PermitNSFW,
                webhookURLs: WebhookURLs,
                VIP: VIP,
                fontSizeClasses: [
                    [ "x-2", "Very small chat room text" ],
                    [ "x-1", "50% smaller chat room text" ],
                    [ "", "Default size" ],
                    [ "x1", "50% larger chat room text" ],
                    [ "x2", "2x larger chat room text" ],
                    [ "x3", "3x larger chat room text" ],
                    [ "x4", "4x larger chat room text" ],
                ],
                imageDisplaySettings: [
                    [ "show", "Always show images in chat" ],
                    [ "collapse", "Collapse images in chat, clicking to expand (default)" ],
                    [ "hide", "Never show images shared in chat" ],
                ],
                reportClassifications: [
                    "It's spam",
                    "It's abusive (racist, homophobic, etc.)",
                    "It's malicious (e.g. link to a malware website, phishing)",
                    "It's illegal (e.g. controlled substances, violence)",
                    "It's child abuse (CP, CSAM, pedophilia, etc.)",
                    "Other (please describe)",
                ],
                sounds: {
                    available: SoundEffects,
                    settings: DefaultSounds,
                    ready: false,
                    audioContext: null,
                    audioTracks: {},
                },
                reactions: [
                    ['❤️', '👍', '😂', '😉', '😢', '😡', '🥰'],
                    ['😘', '👎', '☹️', '😭', '🤔', '🙄', '🤩'],
                    ['👋', '🔥', '😈', '🍑', '🍆', '💦', '🍌'],
                    ['😋', '⭐', '😇', '😴', '😱', '👀', '🎃'],
                    ['🤮', '🥳', '🙏', '🤦', '💩', '🤯', '💯'],
                    ['😏', '🙈', '🙉', '🙊', '☀️', '🌈', '🎂'],
                ],

                // Cached blocklist for the current user sent by your website.
                CachedBlocklist: CachedBlocklist,
            },

            // User JWT settings if available.
            jwt: {
                token: UserJWTToken,
                valid: UserJWTValid,
                claims: UserJWTClaims
            },

            channel: "lobby",
            username: "", //"test",
            autoLogin: false,  // e.g. from JWT auth
            message: "",
            messageBox: null, // HTML element for message entry box
            typingNotifDebounce: null,
            status: "online", // away/idle status

            // Idle detection variables
            idleTimeout: null,
            idleThreshold: 300, // number of seconds you must be idle

            // WebSocket connection.
            ws: {
                conn: null,
                connected: false,
            },

            // Who List for the room.
            whoList: [],
            whoTab: 'online',
            whoSort: 'a-z',
            whoMap: {}, // map username to wholist entry
            muted: {},  // muted usernames for client side state

            // Misc. user preferences (TODO: move all of them here)
            prefs: {
                joinMessages: true,  // show "has entered the room" in public channels
                exitMessages: false, // hide exit messages by default in public channels
                watchNotif: true,    // notify in chat about cameras being watched
                closeDMs: false,     // ignore unsolicited DMs
                muteSounds: false,   // mute all sound effects
            },

            // My video feed.
            webcam: {
                busy: false,
                active: false,
                elem: null,   // <video id="localVideo"> element
                stream: null, // MediaStream object
                muted: false, // our outgoing mic is muted, not by default
                autoMute: false, // mute our mic automatically when going live (user option)
                nsfw: false,  // user has flagged their camera to be NSFW
                mutual: false, // user wants viewers to share their own videos
                mutualOpen: false, // user wants to open video mutually
                vipOnly: false, // only show camera to fellow VIP users

                // Who all is watching me? map of users.
                watching: {},

                // Scaling setting for the videos drawer, so the user can
                // embiggen the webcam sizes so a suitable size.
                videoScale: "",
                videoScaleOptions: [
                    [ "", "Default size" ],
                    [ "x1", "50% larger videos" ],
                    [ "x2", "2x larger videos" ],
                    [ "x3", "3x larger videos" ],
                    [ "x4", "4x larger videos (not recommended)" ],
                ],

                // Available cameras and microphones for the Settings modal.
                gettingDevices: false, // busy indicator for refreshing devices
                videoDevices: [],
                videoDeviceID: null,
                audioDevices: [],
                audioDeviceID: null,

                // After we get a device selected, remember it (by name) so that we
                // might hopefully re-select it by default IF we are able to enumerate
                // devices before they go on camera the first time.
                preferredDeviceNames: {
                    video: null,
                    audio: null,
                },
            },

            // Video flag constants (sync with values in messages.go)
            VideoFlag: {
                Active:         1 << 0,
                NSFW:           1 << 1,
                Muted:          1 << 2,
                IsTalking:      1 << 3,
                MutualRequired: 1 << 4,
                MutualOpen:     1 << 5,
                VipOnly:        1 << 6,
            },

            // WebRTC sessions with other users.
            WebRTC: {
                // Streams per username.
                streams: {},
                muted: {}, // muted bool per username
                booted: {}, // booted bool per username
                poppedOut: {}, // popped-out video per username

                // RTCPeerConnections per username.
                pc: {},

                // Video stream freeze detection.
                frozenStreamInterval: {}, // map usernames to intervals
                frozenStreamDetected: {}, // map usernames to bools

                // Debounce connection attempts since now every click = try to connect.
                debounceOpens: {}, // map usernames to bools

                // Timeouts for open camera attempts. e.g.: when you click to view
                // a camera, the icon changes to a spinner for a few seconds to see
                // whether the video goes on to open.
                openTimeouts: {}, // map usernames to timeouts
            },

            // Chat history.
            history: [],
            channels: {
                // There will be values here like:
                // "lobby": {
                //   "history": [],
                //   "updated": timestamp,
                //   "unread": 4,
                // },
                // "@username": {
                //   "history": [],
                //   ...
                // }
            },
            historyScrollbox: null,
            autoscroll: true, // scroll to bottom on new messages
            fontSizeClass: "", // font size magnification
            imageDisplaySetting: "collapse", // image show/hide setting
            scrollback: 1000,  // scrollback buffer (messages to keep per channel)
            DMs: {},
            messageReactions: {
                // Will look like:
                // "123": {    (message ID)
                //    "❤️": [  (reaction emoji)
                //        "username"  // users who reacted
                //    ]
                // }
            },

            // Responsive CSS controls for mobile.
            responsive: {
                leftDrawerOpen: false,
                rightDrawerOpen: false,
                nodes: {
                    // DOM nodes for the CSS grid cells.
                    $container: null,
                    $left: null,
                    $center: null,
                    $right: null,
                }
            },

            loginModal: {
                visible: false,
            },

            settingsModal: {
                visible: false,
                tab: 'prefs', // selected setting tab
            },

            nsfwModalCast: {
                visible: false,
            },

            nsfwModalView: {
                visible: false,
                dontShowAgain: false,
                user: null, // staged User we wanted to open
            },

            reportModal: {
                visible: false,
                busy: false,
                message: {},
                origMessage: {}, // pointer, so we can set the "reported" flag
                classification: "It's spam",
                comment: "",
            },
        }
    },
    mounted() {
        this.setupConfig(); // localSettings persisted settings
        this.setupIdleDetection();

        this.webcam.elem = document.querySelector("#localVideo");
        this.historyScrollbox = document.querySelector("#chatHistory");

        this.responsive.nodes = {
            $container: document.querySelector(".chat-container"),
            $left: document.querySelector(".left-column"),
            $center: document.querySelector(".chat-column"),
            $right: document.querySelector(".right-column"),
        };
        this.messageBox = document.getElementById("messageBox");

        // Reset CSS overrides for responsive display on any window size change. In effect,
        // making the chat panel the current screen again on phone rotation.
        window.addEventListener("resize", () => {
            this.resetResponsiveCSS();
        });

        // Listen for window focus/unfocus events. Being on a different browser tab, for
        // sound effect alert purposes, counts as not being "in" that chat channel when
        // a message comes in.
        window.addEventListener("focus", () => {
            this.windowFocused = true;
            this.windowFocusedAt = new Date();
        });
        window.addEventListener("blur", () => {
            this.windowFocused = false;
        });

        // Set up sound effects on first page interaction.
        window.addEventListener("click", () => {
            this.setupSounds();
        });
        window.addEventListener("keydown", () => {
            this.setupSounds();
        });

        for (let channel of this.config.channels) {
            this.initHistory(channel.ID);
        }

        this.ChatClient("Welcome to BareRTC!");

        // Auto login with JWT token?
        // TODO: JWT validation on the WebSocket as well.
        if (this.jwt.valid && this.jwt.claims.sub) {
            this.username = this.jwt.claims.sub;
            this.autoLogin = true;
        }

        // Scrub JWT token from query string parameters.
        history.pushState(null, "", location.href.split("?")[0]);

        // XX: always show login dialog to test if this helps iOS devices.
        // this.loginModal.visible = true;
        if (!this.username) {
            this.loginModal.visible = true;
        } else {
            this.signIn();
        }
    },
    watch: {
        "webcam.videoScale": function() {
            document.querySelectorAll(".video-feeds > .feed").forEach(node => {
                node.style.width = null;
                node.style.height = null;
            });
            localStorage.videoScale = this.webcam.videoScale;
        },
        fontSizeClass() {
            // Store the setting persistently.
            localStorage.fontSizeClass = this.fontSizeClass;
        },
        imageDisplaySetting() {
            localStorage.imageDisplaySetting = this.imageDisplaySetting;
        },
        scrollback() {
            localStorage.scrollback = this.scrollback;
        },
        status() {
            // Send presence updates to the server.
            this.sendMe();
        },

        // Webcam preferences that the user can edit while live.
        "webcam.nsfw": function() {
            if (this.webcam.active) {
                this.sendMe();
            }
        },
        "webcam.mutual": function() {
            if (this.webcam.active) {
                this.sendMe();
            }
        },
        "webcam.mutualOpen": function() {
            if (this.webcam.active) {
                this.sendMe();
            }
        },
        "webcam.vipOnly": function() {
            if (this.webcam.active) {
                this.sendMe();
            }

            // If we have toggled this while already connected to people:
            // Hang up on any that have a mutual viewership requirement, if
            // they can not see our VIP-only camera.
            for (username of Object.keys(this.WebRTC.pc)) {
                if (this.whoMap[username] != undefined && this.isVideoNotAllowed(this.whoMap[username])) {
                    this.closeVideo(username);
                }
            }
        },

        // Misc preference watches
        "prefs.joinMessages": function() {
            localStorage.joinMessages = this.prefs.joinMessages;
        },
        "prefs.exitMessages": function() {
            localStorage.exitMessages = this.prefs.exitMessages;
        },
        "prefs.watchNotif": function() {
            localStorage.watchNotif = this.prefs.watchNotif;
        },
        "prefs.muteSounds": function() {
            localStorage.muteSounds = this.prefs.muteSounds;
        },
        "prefs.closeDMs": function() {
            localStorage.closeDMs = this.prefs.closeDMs;

            // Tell ChatServer if we have gone to/from DND.
            this.sendMe();
        },
    },
    computed: {
        chatHistory() {
            if (this.channels[this.channel] == undefined) {
                return [];
            }

            let history = this.channels[this.channel].history;

            // How channels work:
            // - Everything going to a public channel like "lobby" goes
            //   into the "lobby" channel in the front-end
            // - Direct messages are different: they are all addressed
            //   "to" the channel of the current @user, but they are
            //   divided into DM threads based on the username.
            if (this.channel.indexOf("@") === 0) {
                // DM thread, divide them by sender.
                // let username = this.channel.substring(1);
                // return history.filter(v => {
                //     return v.username === username;
                // });
            }
            return history;
        },
        activeDMs() {
            // List your currently open DM threads, sorted by most recent.
            let result = [];
            for (let channel of Object.keys(this.channels)) {
                // @mentions only
                if (channel.indexOf("@") !== 0) {
                    continue;
                }

                result.push({
                    channel: channel,
                    name: channel.substring(1),
                    updated: this.channels[channel].updated,
                    unread: this.channels[channel].unread,
                });
            }

            result.sort((a, b) => b.updated - a.updated);
            return result;
        },
        channelName() {
            // Return a suitable channel title.
            if (this.channel.indexOf("@") === 0) {
                // A DM, return it directly as is.
                return this.channel;
            }

            // Find the friendly name from our config.
            for (let channel of this.config.channels) {
                if (channel.ID === this.channel) {
                    return channel.Name;
                }
            }

            return this.channel;
        },
        isDM() {
            // Is the current channel a DM?
            return this.channel.indexOf("@") === 0;
        },
        canUploadFile() {
            // Public channels: OK
            if (!this.channel.indexOf('@') === 0) {
                return true;
            }

            // User is an admin?
            if (this.jwt.claims.op) {
                return true;
            }

            // User is in a DM thread with an admin?
            if (this.isDM) {
                let partner = this.normalizeUsername(this.channel);
                if (this.whoMap[partner] != undefined && this.whoMap[partner].op) {
                    return true;
                }
            }

            return !this.isDM;
        },
        isOp() {
            // Returns if the current user has operator rights
            return this.jwt.claims.op;
        },
        isVIP() {
            // Returns if the current user has VIP rights.
            return this.jwt.claims.vip;
        },
        myVideoFlag() {
            // Compute the current user's video status flags.
            let status = 0;
            if (!this.webcam.active) return 0; // unset all flags if not active now
            if (this.webcam.active) status |= this.VideoFlag.Active;
            if (this.webcam.muted) status |= this.VideoFlag.Muted;
            if (this.webcam.nsfw) status |= this.VideoFlag.NSFW;
            if (this.webcam.mutual) status |= this.VideoFlag.MutualRequired;
            if (this.webcam.mutualOpen) status |= this.VideoFlag.MutualOpen;
            if (this.webcam.vipOnly && this.isVIP) status |= this.VideoFlag.VipOnly;
            return status;
        },
        sortedWhoList() {
            let result = [...this.whoList];

            switch (this.whoSort) {
                case "broadcasting":
                    result.sort((a, b) => {
                        return (b.video & this.VideoFlag.Active) - (a.video & this.VideoFlag.Active);
                    });
                    break;
                case "nsfw":
                    result.sort((a, b) => {
                        let left = (a.video & (this.VideoFlag.Active | this.VideoFlag.NSFW)),
                            right = (b.video & (this.VideoFlag.Active | this.VideoFlag.NSFW));
                        return right - left;
                    });
                    break;
                case "status":
                    result.sort((a, b) => {
                        if (a.status === b.status) return 0;
                        return b.status < a.status ? -1 : 1;
                    });
                    break;
                case "op":
                    result.sort((a, b) => {
                        return b.op - a.op;
                    });
                    break;
                case "emoji":
                    result.sort((a, b) => {
                        if (a.emoji === b.emoji) return 0;
                        return a.emoji < b.emoji ? -1 : 1;
                    })
                    break;
                case "login":
                    result.sort((a, b) => {
                        return b.loginAt - a.loginAt;
                    });
                    break;
                case "gender":
                    result.sort((a, b) => {
                        if (a.gender === b.gender) return 0;
                        let left = a.gender || 'z',
                            right = b.gender || 'z';
                        return left < right ? -1 : 1;
                    })
                    break;
                case "z-a":
                    result = result.reverse();
            }

            // Default ordering from ChatServer = a-z
            return result;
        },
    },
    methods: {
        // Load user prefs from localStorage, called on startup
        setupConfig() {
            if (localStorage.fontSizeClass != undefined) {
                this.fontSizeClass = localStorage.fontSizeClass;
            }

            if (localStorage.videoScale != undefined) {
                this.webcam.videoScale = localStorage.videoScale;
            }

            if (localStorage.imageDisplaySetting != undefined) {
                this.imageDisplaySetting = localStorage.imageDisplaySetting;
            }

            if (localStorage.scrollback != undefined) {
                this.scrollback = parseInt(localStorage.scrollback);
            }

            // Stored user preferred device names for webcam/audio.
            if (localStorage.preferredDeviceNames != undefined) {
                let dev = JSON.parse(localStorage.preferredDeviceNames);
                this.webcam.preferredDeviceNames.video = dev.video;
                this.webcam.preferredDeviceNames.audio = dev.audio;
            }

            // Webcam mutality preferences from last broadcast.
            if (localStorage.videoMutual === "true") {
                this.webcam.mutual = true;
            }
            if (localStorage.videoMutualOpen === "true") {
                this.webcam.mutualOpen = true;
            }
            if (localStorage.videoAutoMute === "true") {
                this.webcam.autoMute = true;
            }
            if (localStorage.videoVipOnly === "true") {
                this.webcam.vipOnly = true;
            }

            // Misc preferences
            if (localStorage.joinMessages != undefined) {
                this.prefs.joinMessages = localStorage.joinMessages === "true";
            }
            if (localStorage.exitMessages != undefined) {
                this.prefs.exitMessages = localStorage.exitMessages === "true";
            }
            if (localStorage.watchNotif != undefined) {
                this.prefs.watchNotif = localStorage.watchNotif === "true";
            }
            if (localStorage.muteSounds != undefined) {
                this.prefs.muteSounds = localStorage.muteSounds === "true";
            }
            if (localStorage.closeDMs != undefined) {
                this.prefs.closeDMs = localStorage.closeDMs === "true";
            }
        },

        signIn() {
            this.loginModal.visible = false;
            this.dial();
        },

        // Normalize a DM channel name into a username (remove the @ prefix)
        normalizeUsername(channel) {
            return channel.replace(/^@+/, '');
        },

        /**
         * Chat API Methods (WebSocket packets sent/received)
         */

        sendMessage() {
            if (!this.message) {
                return;
            }

            if (!this.ws.connected) {
                this.ChatClient("You are not connected to the server.");
                return;
            }

            // Spammy links.
            if (this.message.toLowerCase().indexOf("onlyfans.com") > -1 ||
                this.message.toLowerCase().indexOf("justfor.fans") > -1 ||
                this.message.toLowerCase().indexOf("justforfans") > -1 ||
                this.message.toLowerCase().match(/fans[^A-Za-z0-9]+dot/)) {

                // If they do it twice, kick them from the room.
                if (this.spamWarningCount >= 1) {
                    // Walk of shame.
                    this.ws.conn.send(JSON.stringify({
                        action: "message",
                        channel: "lobby",
                        message: "**(Message of Shame)** I have been naughty and posted spam in chat despite being warned, "+
                            "and I am now being kicked from the room in shame. ☹️",
                    }));

                    this.ChatServer(
                        "It is <strong>not allowed</strong> to promote your Onlyfans (or similar) "+
                        "site on the chat room. You have been removed from the chat room, and this "+
                        "incident has been reported to the site admin.",
                    );
                    this.pushHistory({
                        channel: this.channel,
                        username: this.username,
                        message: "has been kicked from the room!",
                        action: "presence",
                    });
                    this.disconnect = true;
                    this.ws.connected = false;
                    setTimeout(() => {
                        this.ws.conn.close();
                    }, 1000);
                    return;
                }
                this.spamWarningCount++;

                this.ChatClient(
                    "Please <strong>do not</strong> send links to your Onlyfans (or similar sites) in the chat room. "+
                    "Those links are widely regarded to be spam and make a lot of people uncomfortable. "+
                    "If you violate this again, your account will be suspended.",
                );
                this.message = "";
                return;
            }

            // DEBUGGING: fake set the freeze indicator.
            let match = this.message.match(/^\/freeze (.+?)$/i);
            if (match) {
                let username = match[1];
                this.WebRTC.frozenStreamDetected[username] = true;
                this.ChatClient(`DEBUG: Marked ${username} stream as frozen.`);
                this.message = "";
                return;
            }

            // DEBUGGING: test whether the page thinks you're Apple Webkit.
            if (this.message.toLowerCase().indexOf("/ipad") === 0) {
                if (this.isAppleWebkit()) {
                    this.ChatClient("I have detected that you are probably an iPad or iPhone browser.");
                } else {
                    this.ChatClient("I have detected that you <strong>are not</strong> an iPad or iPhone browser.");
                }
                this.message = "";
                return;
            }

            // console.debug("Send message: %s", this.message);
            this.ws.conn.send(JSON.stringify({
                action: "message",
                channel: this.channel,
                message: this.message,
            }));

            this.message = "";
        },

        sendTypingNotification() {
            // TODO
        },

        // Emoji reactions
        sendReact(message, emoji) {
            this.ws.conn.send(JSON.stringify({
                action: 'react',
                msgID: message.msgID,
                message: emoji,
            }));
        },
        onReact(msg) {
            // Search all channels for this message ID and append the reaction.
            let msgID = msg.msgID,
                who = msg.username,
                emoji = msg.message;

            if (this.messageReactions[msgID] == undefined) {
                this.messageReactions[msgID] = {};
            }
            if (this.messageReactions[msgID][emoji] == undefined) {
                this.messageReactions[msgID][emoji] = [];
            }

            // if they double sent the same reaction, it counts as a removal
            let unreact = false;
            for (let i = 0; i < this.messageReactions[msgID][emoji].length; i++) {
                let reactor = this.messageReactions[msgID][emoji][i];
                if (reactor === who) {
                    this.messageReactions[msgID][emoji].splice(i, 1);
                    unreact = true;
                }
            }

            // if this emoji reaction is empty, clean it up
            if (unreact) {
                if (this.messageReactions[msgID][emoji].length === 0) {
                    delete(this.messageReactions[msgID][emoji]);
                }
                return;
            }

            this.messageReactions[msgID][emoji].push(who);
        },

        // Sync the current user state (such as video broadcasting status) to
        // the backend, which will reload everybody's Who List.
        sendMe() {
            if (!this.ws.connected) return;
            this.ws.conn.send(JSON.stringify({
                action: "me",
                video: this.myVideoFlag,
                status: this.status,
                dnd: this.prefs.closeDMs,
            }));
        },
        onMe(msg) {
            // We have had settings pushed to us by the server, such as a change
            // in our choice of username.
            if (this.username != msg.username) {
                this.ChatServer(`Your username has been changed to ${msg.username}.`);
                this.username = msg.username;
            }

            // The server can set our webcam NSFW flag.
            let myNSFW = this.webcam.nsfw;
            let theirNSFW = (msg.video & this.VideoFlag.NSFW) > 0;
            if (myNSFW != theirNSFW) {
                this.webcam.nsfw = theirNSFW;
            }

            // Note: Me events only come when we join the server or a moderator has
            // flagged our video. This is as good of an "on connected" handler as we
            // get, so push over our cached blocklist from the website now.
            this.bulkMuteUsers();
        },

        // WhoList updates.
        onWho(msg) {
            this.whoList = msg.whoList;
            this.whoMap = {};

            if (this.whoList == undefined) {
                this.whoList = [];
            }

            // If we had a camera open with any of these and they have gone
            // off camera, close our side of the connection.
            for (let row of this.whoList) {
                this.whoMap[row.username] = row;
                if (this.WebRTC.streams[row.username] != undefined &&
                    !(row.video & this.VideoFlag.Active)) {
                    this.closeVideo(row.username, "offerer");
                }
            }

            // Hang up on mutual cameras, if they changed their setting while we
            // are already watching them.
            this.unMutualVideo();

            // Has the back-end server forgotten we are on video? This can
            // happen if we disconnect/reconnect while we were streaming.
            if (this.webcam.active && !(this.whoMap[this.username]?.video & this.VideoFlag.Active)) {
                this.sendMe();
            }
        },

        // Mute or unmute a user.
        muteUser(username) {
            username = this.normalizeUsername(username);
            let mute = this.muted[username] == undefined;

            // If the user is muted because they were blocked on your main website (CachedBlocklist),
            // do not allow a temporary unmute in chat: make them live with their choice.
            if (this.config.CachedBlocklist.length > 0) {
                for (let user of this.config.CachedBlocklist) {
                    if (user === username) {
                        this.ChatClient(
                            `You can not unmute <strong>${username}</strong> because you have blocked them on the main website. `+
                            `To unmute them, you will need to unblock them on the website and then reload the chat room.`
                        );
                        return;
                    }
                }
            }

            if (mute) {
                if (!window.confirm(
                    `Do you want to mute ${username}? If muted, you will no longer see their `+
                    `chat messages or any DMs they send you going forward. Also, ${username} will `+
                    `not be able to see whether your webcam is active until you unmute them.`
                )) {
                    return;
                }
                this.muted[username] = true;
            } else {
                if (!window.confirm(
                    `Do you want to remove your mute on ${username}? If you un-mute them, you `+
                    `will be able to see their chat messages or DMs going forward, but most importantly, `+
                    `they may be able to watch your webcam now if you are broadcasting!\n\n`+
                    `Note: currently you can only re-mute them the next time you see one of their `+
                    `chat messages, or you can only boot them off your cam after they have already `+
                    `opened it. If you are concerned about this, click Cancel and do not remove `+
                    `the mute on ${username}.`
                )) {
                    return;
                }
                delete this.muted[username];
            }

            // Hang up videos both ways.
            this.closeVideo(username);

            this.sendMute(username, mute);
            if (mute) {
                this.ChatClient(
                    `You have muted <strong>${username}</strong> and will no longer see their chat messages, `+
                    `and they will not see whether your webcam is active. You may unmute them via the Who Is Online list.`);
            } else {
                this.ChatClient(
                    `You have unmuted <strong>${username}</strong> and can see their chat messages from now on.`,
                );
            }
        },
        sendMute(username, mute) {
            this.ws.conn.send(JSON.stringify({
                action: mute ? "mute" : "unmute",
                username: username,
            }));
        },
        isMutedUser(username) {
            return this.muted[this.normalizeUsername(username)] != undefined;
        },
        bulkMuteUsers() {
            // On page load, if the website sent you a CachedBlocklist, mute all
            // of these users in bulk when the server connects.
            // this.ChatClient("BulkMuteUsers: sending our blocklist " + this.config.CachedBlocklist);

            if (this.config.CachedBlocklist.length === 0) {
                return; // nothing to do
            }

            // Set the client side mute.
            let blocklist = this.config.CachedBlocklist;
            for (let username of blocklist) {
                this.muted[username] = true;
            }

            // Send the username list to the server.
            this.ws.conn.send(JSON.stringify({
                action: "blocklist",
                usernames: blocklist,
            }))
        },

        // Send a video request to access a user's camera.
        sendOpen(username) {
            this.ws.conn.send(JSON.stringify({
                action: "open",
                username: username,
            }));
        },
        sendBoot(username) {
            this.ws.conn.send(JSON.stringify({
                action: "boot",
                username: username,
            }));
        },
        onOpen(msg) {
            // Response for the opener to begin WebRTC connection.
            this.startWebRTC(msg.username, true);
        },
        onRing(msg) {
            // Request from a viewer to see our broadcast.
            this.startWebRTC(msg.username, false);
        },
        onUserExited(msg) {
            // A user has logged off the server. Clean up any WebRTC connections.
            this.closeVideo(msg.username);
        },

        // Handle messages sent in chat.
        onMessage(msg) {
            // Play sound effects if this is not the active channel or the window is not focused.
            if (msg.channel.indexOf("@") === 0) {
                if (msg.channel !== this.channel || !this.windowFocused) {
                    // If we are ignoring unsolicited DMs, don't play the sound effect here.
                    if (this.prefs.closeDMs && this.channels[msg.channel] == undefined) {
                        console.log("Unsolicited DM received");
                    } else {
                        this.playSound("DM");
                    }
                }
            } else if (msg.channel !== this.channel || !this.windowFocused) {
                this.playSound("Chat");
            }

            this.pushHistory({
                channel: msg.channel,
                username: msg.username,
                message: msg.message,
                messageID: msg.msgID,
            });
        },

        // A user deletes their message for everybody
        onTakeback(msg) {
            // Search all channels for this message ID and remove it.
            for (let channel of Object.keys(this.channels)) {
                for (let i = 0; i < this.channels[channel].history.length; i++) {
                    let cmp = this.channels[channel].history[i];
                    if (cmp.msgID === msg.msgID) {
                        this.channels[channel].history.splice(i, 1);
                        return;
                    }
                }
            }

            console.error("Got a takeback for msgID %d but did not find it!", msg.msgID);
        },

        // User logged in or out.
        onPresence(msg) {
            // TODO: make a dedicated leave event
            let isLeave = false,
                isJoin = false;
            if (msg.message.indexOf("has exited the room!") > -1) {
                // Clean up data about this user.
                this.onUserExited(msg);
                this.playSound("Leave");
                isLeave = true;
            } else if (msg.message.indexOf("has joined the room!") > -1) {
                this.playSound("Enter");
                isJoin = true;
            }

            // Push it to the history of all public channels (depending on user preference).
            if ((isJoin && this.prefs.joinMessages) || (isLeave && this.prefs.exitMessages)
                || (!isJoin && !isLeave)) {
                for (let channel of this.config.channels) {
                    this.pushHistory({
                        channel: channel.ID,
                        action: msg.action,
                        username: msg.username,
                        message: msg.message,
                    });
                }
            }

            // Push also to any DM channels for this user (leave events do push to DM thread for visibility).
            let channel = "@" + msg.username;
            if (this.channels[channel] != undefined) {
                this.pushHistory({
                    channel: channel,
                    action: msg.action,
                    username: msg.username,
                    message: msg.message,
                });
            }
        },

        // Dial the WebSocket connection.
        dial() {
            this.ChatClient("Establishing connection to server...");

            const proto = location.protocol === 'https:' ? 'wss' : 'ws';
            const conn = new WebSocket(`${proto}://${location.host}/ws`);

            conn.addEventListener("close", ev => {
                // Lost connection to server - scrub who list.
                this.onWho({whoList: []});
                this.muted = {};

                this.ws.connected = false;
                this.ChatClient(`WebSocket Disconnected code: ${ev.code}, reason: ${ev.reason}`);

                this.disconnectCount++;
                if (this.disconnectCount > this.disconnectLimit) {
                    this.ChatClient(`It seems there's a problem connecting to the server. Please try some other time.`);
                    return;
                }

                if (!this.disconnect) {
                    if (ev.code !== 1001) {
                        this.ChatClient("Reconnecting in 5s");
                        setTimeout(this.dial, 5000);
                    }
                }
            });

            conn.addEventListener("open", ev => {
                this.ws.connected = true;
                this.ChatClient("Websocket connected!");

                // Tell the server our username.
                this.ws.conn.send(JSON.stringify({
                    action: "login",
                    username: this.username,
                    jwt: this.jwt.token,
                    dnd: this.prefs.closeDMs,
                }));

                // Focus the message entry box.
                window.requestAnimationFrame(() => {
                    this.messageBox.focus();
                });
            });

            conn.addEventListener("message", ev => {
                if (typeof ev.data !== "string") {
                    console.error("unexpected message type", typeof ev.data);
                    return;
                }

                let msg = JSON.parse(ev.data);
                try {
                    // Cast timestamp to date.
                    msg.at = new Date(msg.at);
                } catch(e) {
                    console.error("Parsing timestamp '%s' on msg: %s", msg.at, e);
                }

                switch (msg.action) {
                    case "who":
                        this.onWho(msg);
                        break;
                    case "me":
                        this.onMe(msg);
                        break;
                    case "message":
                        this.onMessage(msg);
                        break;
                    case "takeback":
                        this.onTakeback(msg);
                        break;
                    case "react":
                        this.onReact(msg);
                        break;
                    case "presence":
                        this.onPresence(msg);
                        break;
                    case "ring":
                        this.onRing(msg);
                        break;
                    case "open":
                        this.onOpen(msg);
                        break;
                    case "candidate":
                        this.onCandidate(msg);
                        break;
                    case "sdp":
                        this.onSDP(msg);
                        break;
                    case "watch":
                        this.onWatch(msg);
                        break;
                    case "unwatch":
                        this.onUnwatch(msg);
                        break;
                    case "error":
                        this.pushHistory({
                            channel: msg.channel,
                            username: msg.username || 'Internal Server Error',
                            message: msg.message,
                            isChatServer: true,
                        });
                        break;
                    case "disconnect":
                        this.disconnect = true;
                        break;
                    case "ping":
                        // New JWT token?
                        if (msg.jwt) {
                            this.jwt.token = msg.jwt;
                        }

                        // Reset disconnect retry counter: if we were on long enough to get
                        // a ping, we're well connected and can reconnect no matter how many
                        // times the chat server is rebooted.
                        this.disconnectCount = 0;
                        break;
                    default:
                        console.error("Unexpected action: %s", JSON.stringify(msg));
                }
            });

            this.ws.conn = conn;
        },

        /**
         * WebRTC concerns.
         */

        // Start WebRTC with the other username.
        startWebRTC(username, isOfferer) {
            // this.ChatClient(`Begin WebRTC with ${username}.`);
            let pc = new RTCPeerConnection(configuration);

            // Store uni-directional PeerConnections:
            // - If we are reading video from the other (offerer)
            // - If we are sending video to the other (answerer)
            if (this.WebRTC.pc[username] == undefined) {
                this.WebRTC.pc[username] = {};
            }
            if (isOfferer) {
                this.WebRTC.pc[username].offerer = pc;
            } else {
                this.WebRTC.pc[username].answerer = pc;
            }

            // Keep a pointer to the current channel being established (for candidate/SDP).
            this.WebRTC.pc[username].connecting = pc;

            // Create a data channel so we have something to connect over even if
            // the local user is not broadcasting their own camera.
            // TODO: adding a dummy data channel might allow iPad to open single directional video
            let dataChannel = pc.createDataChannel("data");
            dataChannel.addEventListener("open", event => {
                // beginTransmission(dataChannel);
            })

            // 'onicecandidate' notifies us whenever an ICE agent needs to deliver a
            // message to the other peer through the signaling server.
            pc.onicecandidate = event => {
                if (event.candidate) {
                    this.ws.conn.send(JSON.stringify({
                        action: "candidate",
                        username: username,
                        candidate: JSON.stringify(event.candidate),
                    }));
                }
            };

            // If the user is offerer let the 'negotiationneeded' event create the offer.
            if (isOfferer) {
                pc.onnegotiationneeded = () => {
                    pc.createOffer().then(this.localDescCreated(pc, username)).catch(this.ChatClient);
                };
            }

            // When a remote stream arrives.
            pc.ontrack = event => {
                const stream = event.streams[0];

                // We've received a video! If we had an "open camera spinner timeout",
                // clear it before it expires.
                if (this.WebRTC.openTimeouts[username] != undefined) {
                    clearTimeout(this.WebRTC.openTimeouts[username]);
                    delete(this.WebRTC.openTimeouts[username]);
                }

                // Do we already have it?
                // this.ChatClient(`Received a video stream from ${username}.`);
                if (this.WebRTC.streams[username] == undefined ||
                    this.WebRTC.streams[username].id !== stream.id) {
                    this.WebRTC.streams[username] = stream;
                }

                window.requestAnimationFrame(() => {
                    let $ref = document.getElementById(`videofeed-${username}`);
                    $ref.srcObject = stream;
                });

                // Inform them they are being watched.
                this.sendWatch(username, true);

                // Set a mute video handler to detect freezes.
                stream.getVideoTracks().forEach(videoTrack => {
                    let freezeDetected = () => {
                        console.log("FREEZE DETECTED:", username);
                        // Wait some seconds to see if the stream has recovered on its own
                        setTimeout(() => {
                            // Flag it as likely frozen.
                            if (videoTrack.muted) {
                                this.WebRTC.frozenStreamDetected[username] = true;
                            }
                        }, 7500); // 7.5s
                    };

                    console.log("Apply onmute handler for", username);
                    videoTrack.onmute = freezeDetected;

                    // Double check for frozen streams on an interval.
                    this.WebRTC.frozenStreamInterval[username] = setInterval(() => {
                        if (videoTrack.muted) freezeDetected();
                    }, 3000);
                })
            };

            // ANSWERER: add our video to the connection so that the offerer (the one who
            // clicked on our video icon to watch us) can receive it.
            if (!isOfferer && this.webcam.active) {
                let stream = this.webcam.stream;
                stream.getTracks().forEach(track => {
                    pc.addTrack(track, stream)
                });
            }

            // OFFERER: If we were already broadcasting our own video, and the answerer
            // has the "auto-open your video" setting enabled, attach our video to the initial
            // offer right now.
            //
            // NOTE: this will force open our video on the answerer's side, and this workflow
            // is also the only way that iPads/iPhones/Safari browsers can make a call
            // (two-way video is the only option for them; send-only/receive-only channels seem
            // not to work in Safari).
            if (isOfferer && (this.whoMap[username].video & this.VideoFlag.MutualOpen) && this.webcam.active) {
                let stream = this.webcam.stream;
                stream.getTracks().forEach(track => {
                    pc.addTrack(track, stream)
                });
            }

            // If we are the offerer, begin the connection.
            if (isOfferer) {
                pc.createOffer({
                    offerToReceiveVideo: true,
                    offerToReceiveAudio: true,
                }).then(this.localDescCreated(pc, username)).catch(this.ChatClient);
            }
        },

        // Common handler function for
        localDescCreated(pc, username) {
            return (desc) => {
                pc.setLocalDescription(desc).then(() => {
                    this.ws.conn.send(JSON.stringify({
                        action: "sdp",
                        username: username,
                        description: JSON.stringify(pc.localDescription),
                    }));
                }).catch(e => {
                    console.error(`Error sending WebRTC negotiation message (SDP): ${e}`);
                });
            };
        },

        // Handle inbound WebRTC signaling messages proxied by the websocket.
        onCandidate(msg) {
            if (this.WebRTC.pc[msg.username] == undefined || !this.WebRTC.pc[msg.username].connecting) {
                return;
            }
            let pc = this.WebRTC.pc[msg.username].connecting;

            // XX: WebRTC candidate/SDP messages JSON stringify their inner payload so that the
            // Go back-end server won't re-order their json keys (Safari on Mac OS is very sensitive
            // to the keys being re-ordered during the handshake, in ways that NO OTHER BROWSER cares
            // about at all). Re-parse the JSON stringified object here.
            let candidate = JSON.parse(msg.candidate);

            // Add the new ICE candidate.
            pc.addIceCandidate(candidate).catch(e => {
                this.ChatClient(`addIceCandidate: ${e}`);
            });
        },
        onSDP(msg) {
            if (this.WebRTC.pc[msg.username] == undefined || !this.WebRTC.pc[msg.username].connecting) {
                return;
            }
            let pc = this.WebRTC.pc[msg.username].connecting;

            // XX: WebRTC candidate/SDP messages JSON stringify their inner payload so that the
            // Go back-end server won't re-order their json keys (Safari on Mac OS is very sensitive
            // to the keys being re-ordered during the handshake, in ways that NO OTHER BROWSER cares
            // about at all). Re-parse the JSON stringified object here.
            let message = JSON.parse(msg.description);

            // Add the new ICE candidate.
            // this.ChatClient(`Received a Remote Description from ${msg.username}: ${JSON.stringify(msg.description)}.`);
            pc.setRemoteDescription(new RTCSessionDescription(message), () => {
                // When receiving an offer let's answer it.
                if (pc.remoteDescription.type === 'offer') {
                    pc.createAnswer().then(this.localDescCreated(pc, msg.username)).catch(this.ChatClient);
                }
            }, console.error);
        },
        onWatch(msg) {
            // The user has our video feed open now.
            if (this.isBootedAdmin(msg.username)) return;

            // Notify in chat if this was the first watch (viewer may send multiple per each track they received)
            if (this.prefs.watchNotif && this.webcam.watching[msg.username] != true) {
                this.ChatServer(
                    `<strong>${msg.username}</strong> is now watching your camera.`,
                );
            }

            this.webcam.watching[msg.username] = true;
            this.playSound("Watch");
        },
        onUnwatch(msg) {
            // The user has closed our video feed.
            delete(this.webcam.watching[msg.username]);
            this.playSound("Unwatch");
        },
        sendWatch(username, watching) {
            // Send the watch or unwatch message to backend.
            this.ws.conn.send(JSON.stringify({
                action: watching ? "watch" : "unwatch",
                username: username,
            }));
        },
        isWatchingMe(username) {
            // Return whether the user is watching your camera
            return this.webcam.watching[username] === true;
        },

        /**
         * Front-end web app concerns.
         */

        // Settings modal.
        showSettings() {
            this.settingsModal.visible = true;
        },
        hideSettings() {
            this.settingsModal.visible = false;
        },

        // Set active chat room.
        setChannel(channel) {
            this.channel = typeof(channel) === "string" ? channel : channel.ID;
            this.scrollHistory(this.channel, true);
            this.channels[this.channel].unread = 0;

            // Responsive CSS: switch back to chat panel upon selecting a channel.
            this.openChatPanel();

            // Edit hyperlinks to open in a new window.
            this.makeLinksExternal();

            // Focus the message entry box.
            this.messageBox.focus();
        },
        hasUnread(channel) {
            if (this.channels[channel] == undefined) {
                return 0;
            }
            return this.channels[channel].unread;
        },
        hasAnyUnread() {
            // Returns total unread count (for mobile responsive view to show in the left drawer button)
            let count = 0;
            for (let channel of Object.keys(this.channels)) {
                count += this.channels[channel].unread;
            }
            return count;
        },
        anyUnreadDMs() {
            // Returns true if any unread messages are DM threads
            for (let channel of Object.keys(this.channels)) {
                if (channel.indexOf("@") === 0 && this.channels[channel].unread > 0) {
                    return true;
                }
            }
            return false;
        },
        openDMs(user) {
            let channel = "@" + user.username;
            this.initHistory(channel);
            this.setChannel(channel);

            // Responsive CSS: switch back to chat panel upon opening a DM.
            this.openChatPanel();
        },
        openProfile(user) {
            let url = this.profileURLForUsername(user.username);
            if (url) {
                window.open(url);
            }
        },
        avatarURL(user) {
            // Resolve the avatar URL of this user.
            if (user.avatar.match(/^https?:/i)) {
                return user.avatar;
            } else if (user.avatar.indexOf("/") === 0) {
                return this.config.website.replace(/\/+$/, "") + user.avatar;
            }
            return "";
        },
        isUsernameOnline(username) {
            return this.whoMap[username] != undefined;
        },
        avatarForUsername(username) {
            if (this.whoMap[username] != undefined && this.whoMap[username].avatar) {
                return this.avatarURL(this.whoMap[username]);
            }
            return null;
        },
        profileURLForUsername(username) {
            if (!username) return;
            username = username.replace(/^@/, "");
            if (this.whoMap[username] != undefined && this.whoMap[username].profileURL) {
                let url = this.whoMap[username].profileURL;
                if (url.match(/^https?:/i)) {
                    return url;
                } else if (url.indexOf("/") === 0) {
                    // Subdirectory relative to our WebsiteURL
                    return this.config.website.replace(/\/+$/, "") + url;
                } else {
                    this.ChatClient("Didn't know how to open profile URL: " + url);
                }
                return url;
            }
            return null;
        },
        nicknameForUsername(username) {
            if (!username) return;
            username = username.replace(/^@/, "");
            if (this.whoMap[username] != undefined && this.whoMap[username].profileURL) {
                let nick = this.whoMap[username].nickname;
                if (nick) {
                    return nick;
                }
            } else if (this.whoMap[username] == undefined && username !== 'ChatServer' && username !== 'ChatClient') {
                // User is not even logged in! Add this note to their name
                username += " (offline)";
            }
            return username;
        },
        isUsernameDND(username) {
            if (!username) return false;
            return this.whoMap[username] != undefined && this.whoMap[username].dnd;
        },
        isUsernameCamNSFW(username) {
            // returns true if the username is broadcasting and NSFW, false otherwise.
            // (used for the color coding of their nickname on their video box - so assumed they are broadcasting)
            if (this.whoMap[username] != undefined && this.whoMap[username].video & this.VideoFlag.NSFW) {
                return true;
            }
            return false;
        },
        leaveDM() {
            // Validate we're in a DM currently.
            if (this.channel.indexOf("@") !== 0) return;

            if (!window.confirm(
                "Do you want to close this chat thread? Your conversation history will " +
                "be forgotten on your computer, but your chat partner may still have " +
                "your chat thread open on their end."
            )) {
                return;
            }

            let channel = this.channel;
            this.setChannel(this.config.channels[0].ID);
            delete(this.channels[channel]);
        },

        /* Take back messages (for everyone) or remove locally */
        takeback(msg) {
            if (!window.confirm(
                "Do you want to take this message back? Doing so will remove this message from everybody's view in the chat room."
            )) return;

            this.ws.conn.send(JSON.stringify({
                action: "takeback",
                msgID: msg.msgID,
            }));
        },
        removeMessage(msg) {
            if (!window.confirm(
                "Do you want to remove this message from your view? This will delete the message only for you, but others in this chat thread may still see it."
            )) return;

            this.onTakeback({
                msgID: msg.msgID,
            })
        },

        /* message reaction emojis */
        hasReactions(msg) {
            return this.messageReactions[msg.msgID] != undefined;
        },
        getReactions(msg) {
            if (!this.hasReactions(msg)) return [];
            return this.messageReactions[msg.msgID];
        },
        iReacted(msg, emoji) {
            // test whether the current user has reacted
            if (this.messageReactions[msg.msgID] != undefined && this.messageReactions[msg.msgID][emoji] != undefined) {
                for (let reactor of this.messageReactions[msg.msgID][emoji]) {
                    if (reactor === this.username) {
                        return true;
                    }
                }
            }
            return false;
        },

        activeChannels() {
            // List of current channels, unread indicators etc.
            let result = [];
            for (let channel of this.config.channels) {
                // VIP room we can't see?
                if (channel.VIP && !this.isVIP) continue;

                let data = {
                    ID: channel.ID,
                    Name: channel.Name,
                };
                if (this.channels[channel] != undefined) {
                    data.Unread = this.channels[channel].unread;
                    data.Updated = this.channels[channel].updated;
                }
                result.push(data);
            }
            return result;
        },

        // Start broadcasting my webcam.
        // - force=true to skip the NSFW modal prompt (this param is passed by the button in that modal)
        // - changeCamera=true to re-negotiate WebRTC connections with a new camera device (invoked by the Settings modal)
        startVideo({force=false, changeCamera=false}) {
            if (this.webcam.busy) return;

            // Before they go on cam the first time, ATTEMPT to get their device names.
            // - If they had never granted permission, we won't get the names of
            //   the devices and no big deal.
            // - If they had given permission before, we can present a nicer experience
            //   for them and enumerate their devices before they go on originally.
            if (!changeCamera && !force) {
                // Initial broadcast: did they select device IDs?
                this.getDevices();
            }

            // If we are running in PermitNSFW mode, show the user the modal.
            if (this.config.permitNSFW && !force) {
                this.nsfwModalCast.visible = true;
                return;
            }

            let mediaParams = {
                audio: true,
                video: {
                    width: { max: 640 },
                    height: { max: 480 },
                },
            };

            // Name the specific devices chosen by the user.
            if (this.webcam.videoDeviceID) {
                mediaParams.video.deviceId = { exact: this.webcam.videoDeviceID };
            }
            if (this.webcam.audioDeviceID) {
                mediaParams.audio = {
                    deviceId: { exact: this.webcam.audioDeviceID },
                };
            }

            this.webcam.busy = true;
            navigator.mediaDevices.getUserMedia(mediaParams).then(stream => {
                this.webcam.active = true;
                this.webcam.elem.srcObject = stream;
                this.webcam.stream = stream;

                // Save our mutuality prefs.
                localStorage.videoMutual = this.webcam.mutual;
                localStorage.videoMutualOpen = this.webcam.mutualOpen;
                localStorage.videoAutoMute = this.webcam.autoMute;
                localStorage.videoVipOnly = this.webcam.vipOnly;

                // Auto-mute our camera? Two use cases:
                // 1. The user marked their cam as muted but then changed video device,
                //    so we set the mute to match their preference as shown on their UI.
                // 2. The user opted to auto-mute their camera from the get to on their
                //    NSFW broadcast modal popup.
                if (this.webcam.muted || this.webcam.autoMute) {
                    // Mute their audio tracks.
                    this.webcam.stream.getAudioTracks().forEach(track => {
                        track.enabled = false;
                    });

                    // Set their front-end mute toggle to match (in case of autoMute).
                    this.webcam.muted = true;
                }

                // Tell backend the camera is ready.
                this.sendMe();

                // Record the selected device IDs.
                this.webcam.videoDeviceID = stream.getVideoTracks()[0].getSettings().deviceId;
                this.webcam.audioDeviceID = stream.getAudioTracks()[0].getSettings().deviceId;
                console.log("device IDs:", this.webcam.videoDeviceID, this.webcam.audioDeviceID);

                // Collect video and audio devices to let the user change them in their settings.
                this.getDevices().then(() => {
                    // Store their names on localStorage in case we can reselect them by name
                    // on the user's next visit.
                    this.storePreferredDeviceNames();
                });

                // If we have changed devices, reconnect everybody's WebRTC channels for your existing watchers.
                if (changeCamera) {
                    this.updateWebRTCStreams();
                }
            }).catch(err => {
                this.ChatClient(`Webcam error: ${err}`);
            }).finally(() => {
                this.webcam.busy = false;
            })
        },
        getDevices() {
            // Collect video and audio devices.
            if (!navigator.mediaDevices?.enumerateDevices) {
                console.log("enumerateDevices() not supported.");
                return;
            }

            this.webcam.gettingDevices = true;

            return navigator.mediaDevices.enumerateDevices().then(devices => {
                this.webcam.videoDevices = [];
                this.webcam.audioDevices = [];
                devices.forEach(device => {
                    // If we can't get the device label, disregard it.
                    // It can happen if the user has not yet granted permission.
                    if (!device.label) {
                        return;
                    };

                    if (device.kind === 'videoinput') {
                        // console.log(`Video device ${device.deviceId} ${device.label}`);
                        this.webcam.videoDevices.push({
                            id: device.deviceId,
                            label: device.label,
                        });
                    } else if (device.kind === 'audioinput') {
                        // console.log(`Audio device ${device.deviceId} ${device.label}`);
                        this.webcam.audioDevices.push({
                            id: device.deviceId,
                            label: device.label,
                        });
                    }
                });

                // If we don't have the user's current active device IDs (e.g., they have
                // not yet started their video the first time), see if we can pre-select
                // by their preferred device names.
                if (!this.webcam.videoDeviceID && this.webcam.preferredDeviceNames.video) {
                    for (let dev of this.webcam.videoDevices) {
                        if (dev.label === this.webcam.preferredDeviceNames.video) {
                            this.webcam.videoDeviceID = dev.id;
                        }
                    }
                }
                if (!this.webcam.audioDeviceID && this.webcam.preferredDeviceNames.audio) {
                    for (let dev of this.webcam.audioDevices) {
                        if (dev.label === this.webcam.preferredDeviceNames.audio) {
                            this.webcam.audioDeviceID = dev.id;
                        }
                    }
                }
            }).catch(err => {
                this.ChatClient(`Error listing your cameras and microphones: ${err.name}: ${err.message}`);
            }).finally(() => {
                // In the settings modal, let the spinner spin for a moment.
                setTimeout(() => {
                    this.webcam.gettingDevices = false;
                }, 500);
            })
        },
        storePreferredDeviceNames() {
            // This function looks up the names of the user's selected video/audio device.
            // When they come back later, IF we are able to enumerate devices before they
            // go on for the first time, we might pre-select their last device by name.
            // NOTE: on iOS apparently device IDs shuffle every single time so only names
            // may be reliable!
            if (this.webcam.videoDeviceID) {
                for (let dev of this.webcam.videoDevices) {
                    if (dev.id === this.webcam.videoDeviceID && dev.label) {
                        this.webcam.preferredDeviceNames.video = dev.label;
                    }
                }
            }

            if (this.webcam.audioDeviceID) {
                for (let dev of this.webcam.audioDevices) {
                    if (dev.id === this.webcam.audioDeviceID && dev.label) {
                        this.webcam.preferredDeviceNames.audio = dev.label;
                    }
                }
            }

            // Put them on localStorage.
            localStorage.preferredDeviceNames = JSON.stringify(this.webcam.preferredDeviceNames);
        },

        // Replace your video/audio streams for your watchers (on camera changes)
        updateWebRTCStreams() {
            console.log("Re-negotiating video and audio channels to your watchers.");
            for (let username of Object.keys(this.WebRTC.pc)) {
                let pc = this.WebRTC.pc[username];
                if (pc.answerer != undefined) {
                    let oldTracks = pc.answerer.getSenders();
                    let newTracks = this.webcam.stream.getTracks();

                    // Remove and replace the tracks.
                    for (let old of oldTracks) {
                        if (old.track.kind === 'audio') {
                            for (let replace of newTracks) {
                                if (replace.kind === 'audio') {
                                    old.replaceTrack(replace);
                                }
                            }
                        }
                        else if (old.track.kind === 'video') {
                            for (let replace of newTracks) {
                                if (replace.kind === 'video') {
                                    old.replaceTrack(replace);
                                }
                            }
                        }
                    }
                }
            }
        },

        // Begin connecting to someone else's webcam.
        openVideoByUsername(username, force) {
            if (this.whoMap[username] != undefined) {
                this.openVideo(this.whoMap[username], force);
                return;
            }
            this.ChatClient("Couldn't open video by username: not found.");
        },
        openVideo(user, force) {
            if (user.username === this.username) {
                this.ChatClient("You can already see your own webcam.");
                return;
            }

            // If we have muted the target, we shouldn't view their video.
            if (this.isMutedUser(user.username)) {
                this.ChatClient(`You have muted <strong>${user.username}</strong> and so should not see their camera.`);
                return;
            }

            // Is the target user NSFW? Go thru the modal.
            let dontShowAgain = localStorage["skip-nsfw-modal"] == "true";
            if ((user.video & this.VideoFlag.NSFW) && !dontShowAgain && !force) {
                this.nsfwModalView.user = user;
                this.nsfwModalView.visible = true;
                return;
            }
            if (this.nsfwModalView.dontShowAgain) {
                // user doesn't want to see the modal again.
                localStorage["skip-nsfw-modal"] = "true";
            }

            // Debounce so we don't spam too much for the same user.
            if (this.WebRTC.debounceOpens[user.username]) return;
            this.WebRTC.debounceOpens[user.username] = true;
            setTimeout(() => {
                delete(this.WebRTC.debounceOpens[user.username]);
            }, 5000);

            // Camera is already open? Then disconnect the connection.
            if (this.WebRTC.pc[user.username] != undefined && this.WebRTC.pc[user.username].offerer != undefined) {
                this.closeVideo(user.username, "offerer");
            }

            // If this user requests mutual viewership...
            if (this.isVideoNotAllowed(user)) {
                this.ChatClient(
                    `<strong>${user.username}</strong> has requested that you should share your own camera too before opening theirs.`
                );
                return;
            }

            // Set a timeout: the video icon becomes a spinner and we wait a while
            // to see if the connection went thru. This gives the user feedback and we
            // can avoid a spammy 'ChatClient' notification message.
            if (this.WebRTC.openTimeouts[user.username] != undefined) {
                clearTimeout(this.WebRTC.openTimeouts[user.username]);
                delete(this.WebRTC.openTimeouts[user.username]);
            }
            this.WebRTC.openTimeouts[user.username] = setTimeout(() => {
                // It timed out. If they are on an iPad, offer additional hints on
                // how to have better luck connecting their cameras.
                if (this.isAppleWebkit()) {
                    this.ChatClient(
                        `There was an error opening <strong>${user.username}</strong>'s camera.<br><br>` +
                        "<strong>Advice:</strong> You appear to be on an iPad-style browser. Webcam sharing " +
                        "may be limited and only work if:<br>A) You are sharing your own camera first, and<br>B) "+
                        "The person you view has the setting to auto-open your camera in return.<br>Best of luck!",
                    );
                } else {
                    this.ChatClient(
                        `There was an error opening <strong>${user.username}</strong>'s camera.`,
                    );
                }
                delete(this.WebRTC.openTimeouts[user.username]);
            }, 10000);

            // Send the ChatServer 'open' command.
            this.sendOpen(user.username);

            // Responsive CSS -> go to chat panel to see the camera
            this.openChatPanel();
        },
        closeVideo(username, name) {
            // Clean up any lingering camera freeze states.
            delete (this.WebRTC.frozenStreamDetected[username]);
            if (this.WebRTC.frozenStreamInterval[username]) {
                clearInterval(this.WebRTC.frozenStreamInterval);
                delete(this.WebRTC.frozenStreamInterval[username]);
            }

            if (name === "offerer") {
                // We are closing another user's video stream.
                delete (this.WebRTC.streams[username]);
                delete (this.WebRTC.muted[username]);
                delete (this.WebRTC.poppedOut[username]);
                if (this.WebRTC.pc[username] != undefined && this.WebRTC.pc[username].offerer != undefined) {
                    this.WebRTC.pc[username].offerer.close();
                    delete (this.WebRTC.pc[username]);
                }

                // Inform backend we have closed it.
                this.sendWatch(username, false);
                return;
            } else if (name === "answerer") {
                // We have turned off our camera, kick off viewers.
                if (this.WebRTC.pc[username] != undefined && this.WebRTC.pc[username].answerer != undefined) {
                    this.WebRTC.pc[username].answerer.close();
                    delete (this.WebRTC.pc[username]);
                }
                return;
            }

            // A user has logged off the server. Clean up any WebRTC connections.
            delete (this.WebRTC.streams[username]);
            delete (this.webcam.watching[username]);
            if (this.WebRTC.pc[username] != undefined) {
                if (this.WebRTC.pc[username].offerer) {
                    this.WebRTC.pc[username].offerer.close();
                }
                if (this.WebRTC.pc[username].answerer) {
                    this.WebRTC.pc[username].answerer.close();
                }
                delete (this.WebRTC.pc[username]);
                delete (this.WebRTC.muted[username]);
                delete (this.WebRTC.poppedOut[username]);
            }

            // Clean up any lingering camera freeze states.
            delete (this.WebRTC.frozenStreamDetected[username]);
            if (this.WebRTC.frozenStreamInterval[username]) {
                clearInterval(this.WebRTC.frozenStreamInterval);
                delete(this.WebRTC.frozenStreamInterval[username]);
            }

            // Inform backend we have closed it.
            this.sendWatch(username, false);
        },
        unMutualVideo() {
            // If we had our camera on to watch a video of someone who wants mutual cameras,
            // and then we turn ours off: we should unfollow the ones with mutual video.
            if (this.webcam.active) return;
            for (let row of this.whoList) {
                let username = row.username;
                if ((row.video & this.VideoFlag.MutualRequired) && this.WebRTC.pc[username] != undefined) {
                    this.closeVideo(username);
                }
            }
        },
        webcamIconClass(user) {
            // Return the icon to show on a video button.
            // - Usually a video icon
            // - May be a crossed-out video if isVideoNotAllowed
            // - Or an eyeball for cameras already opened
            // - Or a spinner if we are actively trying to open the video

            // Current user sees their own self camera always.
            if (user.username === this.username && this.webcam.active) {
                return 'fa-eye';
            }

            // In spinner mode? (Trying to open the video)
            if (this.WebRTC.openTimeouts[user.username] != undefined) {
                return 'fa-spinner fa-spin';
            }

            // Already opened?
            if (this.WebRTC.pc[user.username] != undefined && this.WebRTC.streams[user.username] != undefined) {
                return 'fa-eye';
            }

            // iPad test: they will have very limited luck opening videos unless
            // A) the iPad camera is already on, and
            // B) the person they want to watch has mutual auto-open enabled.
            if (this.isAppleWebkit()) {
                if (!this.webcam.active) {
                    return 'fa-video-slash';  // can not open any cam w/o local video on
                }
                if (!(this.whoMap[user.username].video & this.VideoFlag.MutualOpen)) {
                    // the user must have mutual auto-open on: the iPad has to offer
                    // their video which will force open their cam on the other side,
                    // and this is only if the user expects it.
                    return 'fa-video-slash';
                }
            }

            if (this.isVideoNotAllowed(user)) return 'fa-video-slash';
            return 'fa-video';
        },
        isVideoNotAllowed(user) {
            // Returns whether the video button to open a user's cam will be not allowed (crossed out)

            // Mutual video sharing is required on this camera, and ours is not active
            if ((user.video & this.VideoFlag.Active) && (user.video & this.VideoFlag.MutualRequired)) {
                // A nuance to the mutual video required: if we DO have our cam on, but ours is VIP only, and the
                // user we want to watch can't see that our cam is on, then honor their wishes.
                if (this.webcam.active && this.isVIP && this.webcam.vipOnly && this.whoMap[user.username] != undefined && !this.whoMap[user.username].vip) {
                    // Our cam is active, but the non-VIP user won't see it on, so they won't expect
                    // us to be able to open their camera.
                    return true;
                }

                if (!this.webcam.active) {
                    // Our cam is not broadcasting, but they requested it should be: not allowed.
                    return true;
                }
            }

            // We have muted them and it wouldn't be appropriate to still watch their video but not get their messages.
            if (this.isMutedUser(user.username)) {
                return true;
            }

            return false;
        },

        // Show who watches our video.
        showViewers() {
            // TODO: for now, ChatClient is our bro.
            let users = Object.keys(this.webcam.watching);
            if (users.length === 0) {
                this.ChatClient("There is currently nobody viewing your camera.");
            } else {
                this.ChatClient("Your current webcam viewers are:<br><br>" + users.join(", "));
            }

            // Also focus the Watching list.
            this.whoTab = 'watching';

            // TODO: if mobile, show the panel - this width matches
            // the media query in chat.css
            if (screen.width < 1024) {
                this.openWhoPanel();
            }
        },

        // Boot someone off your video.
        bootUser(username) {
            if (!window.confirm(
                `Kick ${username} off your camera? This will also prevent them `+
                `from seeing that your camera is active for the remainder of your `+
                `chat session.`)) {
                return;
            }

            this.sendBoot(username);
            this.WebRTC.booted[username] = true;

            // Close the WebRTC peer connection.
            if (this.WebRTC.pc[username] != undefined) {
                this.closeVideo(username, "answerer");
            }

            this.ChatClient(
                `You have booted ${username} off your camera. They will no longer be able `+
                `to connect to your camera, or even see that your camera is active at all -- `+
                `to them it appears as though you had turned yours off.<br><br>This will be `+
                `in place for the remainder of your current chat session.`
            );
        },
        isBootedAdmin(username) {
            return (this.WebRTC.booted[username] === true || this.muted[username] === true) &&
                this.whoMap[username] != undefined &&
                this.whoMap[username].op;
        },

        // Stop broadcasting.
        stopVideo() {
            // Close all WebRTC sessions.
            for (username of Object.keys(this.WebRTC.pc)) {
                this.closeVideo(username, "answerer");
            }

            // Hang up on mutual cameras.
            this.unMutualVideo();

            // Close the local camera devices completely.
            this.webcam.stream.getTracks().forEach(track => {
                track.stop();
            });

            // Reset all front-end state.
            this.webcam.elem.srcObject = null;
            this.webcam.stream = null;
            this.webcam.active = false;
            this.webcam.muted = false;
            this.whoTab = "online";

            // Tell backend our camera state.
            this.sendMe();
        },

        // Mute my microphone if broadcasting.
        muteMe() {
            this.webcam.muted = !this.webcam.muted;
            this.webcam.stream.getAudioTracks().forEach(track => {
                track.enabled = !this.webcam.muted;
            });

            // Communicate our local mute to others.
            this.sendMe();
        },
        isSourceMuted(username) {
            // See if the webcam broadcaster muted their mic at the source
            if (this.whoMap[username] != undefined && this.whoMap[username].video & this.VideoFlag.Muted) {
                return true;
            }
            return false;
        },
        isMuted(username) {
            return this.WebRTC.muted[username] === true;
        },
        muteVideo(username) {
            this.WebRTC.muted[username] = !this.isMuted(username);

            // Find the <video> tag to mute it.
            let $ref = document.getElementById(`videofeed-${username}`);
            if ($ref) {
                $ref.muted = this.WebRTC.muted[username];
            }
        },

        // Pop out a user's video.
        popoutVideo(username) {
            this.WebRTC.poppedOut[username] = !this.WebRTC.poppedOut[username];

            // If not popped out, reset CSS positioning.
            window.requestAnimationFrame(this.makeDraggableVideos);
        },

        // Outside of Vue, attach draggable video scripts to DOM.
        makeDraggableVideos() {
            let $panel = document.querySelector("#video-feeds");

            interact('.popped-in').unset();

            // Give popped out videos to the root of the DOM so they can
            // be dragged anywhere on the page.
            window.requestAnimationFrame(() => {
                document.querySelectorAll('.popped-out').forEach(node => {
                    // $panel.removeChild(node);
                    document.body.appendChild(node);
                });

                document.querySelectorAll('.popped-in').forEach(node => {
                    // document.body.removeChild(node);
                    $panel.appendChild(node);
                    node.style.top = null;
                    node.style.left = null;
                    node.setAttribute('data-x', 0);
                    node.setAttribute('data-y', 0);
                });
            });

            interact('.popped-out').draggable({
                // enable inertial throwing
                inertia: true,
                // keep the element within the area of it's parent
                modifiers: [
                interact.modifiers.restrictRect({
                    restriction: 'parent',
                    endOnly: true
                })
                ],

                listeners: {
                // call this function on every dragmove event
                move(event) {
                    let target = event.target;
                    let x = (parseFloat(target.getAttribute('data-x')) || 0) + event.dx
                    let y = (parseFloat(target.getAttribute('data-y')) || 0) + event.dy

                    target.style.top = `${y}px`;
                    target.style.left = `${x}px`;

                    target.setAttribute('data-x', x);
                    target.setAttribute('data-y', y);
                },

                // call this function on every dragend event
                end (event) {
                    console.log(
                    'moved a distance of ' +
                    (Math.sqrt(Math.pow(event.pageX - event.x0, 2) +
                                Math.pow(event.pageY - event.y0, 2) | 0))
                        .toFixed(2) + 'px')
                }
                }
            }).resizable({
                edges: { left: true, right: true, bottom: true, right: true },
                listeners: {
                    move (event) {
                      var target = event.target
                      var x = (parseFloat(target.getAttribute('data-x')) || 0)
                      var y = (parseFloat(target.getAttribute('data-y')) || 0)

                      // update the element's style
                      target.style.width = event.rect.width + 'px'
                      target.style.height = event.rect.height + 'px'

                      // translate when resizing from top or left edges
                      x += event.deltaRect.left
                      y += event.deltaRect.top

                      target.style.top = `${y}px`;
                      target.style.left = `${x}px`;

                      target.setAttribute('data-x', x)
                      target.setAttribute('data-y', y)
                    }
                  },
                  modifiers: [
                    // keep the edges inside the parent
                    interact.modifiers.restrictEdges({
                      outer: 'parent'
                    }),

                    // minimum size
                    interact.modifiers.restrictSize({
                      min: { width: 100, height: 50 }
                    })
                  ],

                  inertia: true
            })
        },

        initHistory(channel) {
            if (this.channels[channel] == undefined) {
                this.channels[channel] = {
                    history: [],
                    updated: Date.now(),
                    unread: 0,
                };
            }
        },
        pushHistory({ channel, username, message, action = "message", isChatServer, isChatClient, messageID }) {
            // Default channel = your current channel.
            if (!channel) {
                channel = this.channel;
            }

            // Are we ignoring DMs?
            if (this.prefs.closeDMs && channel.indexOf('@') === 0) {
                // Don't allow an (incoming) DM to initialize a new chat room for us.
                // Unless the user is an operator.
                let isSenderOp = this.whoMap[username] != undefined && this.whoMap[username].op;
                if (username !== this.username && this.channels[channel] == undefined && !isSenderOp) return;
            }

            // Initialize this channel's history?
            this.initHistory(channel);

            // Image handling per the user's preference.
            if (message.indexOf("<img") > -1) {
                if (this.imageDisplaySetting === "hide") {
                    return;
                } else if (this.imageDisplaySetting === "collapse") {
                    // Put a collapser link.
                    let collapseID = `collapse-${messageID}`;
                    message = `
                        <a href="#" id="img-show-${collapseID}"
                            class="button is-outlined is-small is-info"
                            onclick="document.querySelector('#img-${collapseID}').style.display = 'block';
                                     document.querySelector('#img-show-${collapseID}').style.display = 'none';
                                     return false">
                            <i class="fa fa-image mr-1"></i>
                            Image attachment - click to expand
                        </a>
                        <div id="img-${collapseID}" style="display: none">${message}</div>`;
                }
            }


            // Append the message.
            this.channels[channel].updated = new Date().getTime();
            this.channels[channel].history.push({
                action: action,
                channel: channel,
                username: username,
                message: message,
                msgID: messageID,
                at: new Date(),
                isChatServer,
                isChatClient,
            });

            // Trim the history per the scrollback buffer.
            if (this.scrollback > 0 && this.channels[channel].history.length > this.scrollback) {
                this.channels[channel].history = this.channels[channel].history.slice(
                    -this.scrollback,
                    this.channels[channel].history.length+1,
                );
            }

            this.scrollHistory(channel);

            // Mark unread notifiers if this is not our channel.
            if (this.channel !== channel) {
                // Don't notify about presence broadcasts.
                if (action !== "presence" && !isChatServer) {
                    this.channels[channel].unread++;
                }
            }

            // Edit hyperlinks to open in a new window.
            this.makeLinksExternal();
        },

        scrollHistory(channel, force) {
            if (!this.autoscroll && !force) return;

            window.requestAnimationFrame(() => {
                // Only scroll if it's the current channel.
                if (channel !== this.channel) return;

                this.historyScrollbox.scroll({
                    top: this.historyScrollbox.scrollHeight,
                    behavior: 'smooth',
                });
            });

        },

        // Responsive CSS controls for mobile. Notes for maintenance:
        // - The chat.css has responsive CSS to hide the left/right panels
        //   and set the grid-template-columns for devices < 1024px width
        // - These functions override w/ style tags to force the drawer to
        //   be visible and change the grid-template-columns.
        // - On window resize (e.g. device rotation) or when closing one
        //   of the side drawers, we reset our CSS overrides to default so
        //   the main chat window reappears.
        openChannelsPanel() {
            // Open the left drawer
            let $container = this.responsive.nodes.$container,
                $drawer = this.responsive.nodes.$left;

            $container.style.gridTemplateColumns = "1fr 0 0";
            $drawer.style.display = "block";
        },
        openWhoPanel() {
            // Open the right drawer
            let $container = this.responsive.nodes.$container,
                $drawer = this.responsive.nodes.$right;

            $container.style.gridTemplateColumns = "0 0 1fr";
            $drawer.style.display = "block";
        },
        openChatPanel() {
            this.resetResponsiveCSS();
        },
        resetResponsiveCSS() {
            let $container = this.responsive.nodes.$container,
                $left = this.responsive.nodes.$left,
                $right = this.responsive.nodes.$right;

            $left.style.removeProperty("display");
            $right.style.removeProperty("display");
            $container.style.removeProperty("grid-template-columns");
        },

        // Send a chat message as ChatServer
        ChatServer(message) {
            this.pushHistory({
                username: "ChatServer",
                message: message,
                isChatServer: true,
            });
        },
        ChatClient(message) {
            this.pushHistory({
                username: "ChatClient",
                message: message,
                isChatClient: true,
            });
        },

        // Format a datetime nicely for chat timestamp.
        prettyDate(date) {
            if (date == undefined) return '';
            let hours = date.getHours(),
                minutes = String(date.getMinutes()).padStart(2, '0'),
                ampm = hours >= 11 ? "pm" : "am";

            let hour = hours%12 || 12;
            return `${(hour)}:${minutes} ${ampm}`;
        },

        // CSS classes for the profile button (color coded genders)
        profileButtonClass(user) {
            // VIP background.
            let result = "";
            if (user.vip) {
                result = "has-background-vip ";
            }

            let gender = (user.gender || "").toLowerCase();
            if (gender.indexOf("m") === 0) {
                return result+"has-text-gender-male";
            } else if (gender.indexOf("f") === 0) {
                return result+"has-text-gender-female";
            } else if (gender.length > 0) {
                return result+"has-text-gender-other";
            }
            return "";
        },

        /**
         * Image sharing in chat
         */

        // The image upload button handler.
        uploadFile() {
            let input = document.createElement('input');
            input.type = 'file';
            input.accept = 'image/*';
            input.onchange = e => {
                let file = e.target.files[0];
                if (file.size > FileUploadMaxSize) {
                    this.ChatClient(`Please share an image smaller than ${FileUploadMaxSize / 1024 / 1024} MB in size!`);
                    return;
                }

                this.ChatClient(`<em>Uploading file to chat: ${file.name} - ${file.size} bytes, ${file.type} format.</em>`);

                // Get image file data.
                let reader = new FileReader();
                let rawData = new ArrayBuffer();
                reader.onload = e => {
                    rawData = e.target.result;

                    let fileByteArray = [],
                        u8array = new Uint8Array(rawData);
                    for (let i = 0; i < u8array.length; i++) {
                        fileByteArray.push(u8array[i]);
                    }

                    let msg = JSON.stringify({
                        action: "file",
                        channel: this.channel,
                        message: file.name,
                        bytes: fileByteArray, //btoa(fileByteArray),
                    });

                    // Send it to the chat server.
                    this.ws.conn.send(msg);
                };

                reader.readAsArrayBuffer(file);
            };
            input.click();
        },

        /**
         * Sound effect concerns.
         */

        setupSounds() {
            // Note: setupSounds had to be called on a page gesture (mouse click) before browsers
            // allow it to set up the AudioContext. If we've successfully set one up before, exit
            // this function immediately.
            if (this.config.sounds.audioContext) {
                if (this.config.sounds.audioContext.state === 'suspended') {
                    this.config.sounds.audioContext.resume();
                }
                return;
            }

            try {
                if (AudioContext) {
                    this.config.sounds.audioContext = new AudioContext();
                } else {
                    this.config.sounds.audioContext = window.AudioContext || window.webkitAudioContext;
                }
            } catch {}

            if (!this.config.sounds.audioContext) {
                console.error("Couldn't set up AudioContext! No sound effects will be supported.");
                return;
            }

            // Create <audio> elements for all the sounds.
            for (let effect of this.config.sounds.available) {
                if (!effect.filename) continue; // 'Quiet' has no audio

                let elem = document.createElement("audio");
                elem.autoplay = false;
                elem.src = `/static/sfx/${effect.filename}`;
                document.body.appendChild(elem);

                let track = this.config.sounds.audioContext.createMediaElementSource(elem);
                track.connect(this.config.sounds.audioContext.destination);
                this.config.sounds.audioTracks[effect.name] = elem;
            }

            // Apply the user's saved preferences if any.
            for (let setting of Object.keys(this.config.sounds.settings)) {
                if (localStorage[`sound:${setting}`] != undefined) {
                    this.config.sounds.settings[setting] = localStorage[`sound:${setting}`];
                }
            }
        },

        playSound(event) {
            // Muting all SFX?
            if (this.prefs.muteSounds) return;
            let filename = this.config.sounds.settings[event];
            // Do we have an audio track?
            if (this.config.sounds.audioTracks[filename] != undefined) {
                let track = this.config.sounds.audioTracks[filename];
                track.play();
            }
        },

        setSoundPref(event) {
            this.playSound(event);

            // Store the user's setting in localStorage.
            localStorage[`sound:${event}`] = this.config.sounds.settings[event];
        },

        // Make all links in chat open in new windows
        makeLinksExternal() {
            window.requestAnimationFrame(() => {
                let $history = document.querySelector("#chatHistory");
                // Make all <a> links appearing in chat into external links.
                ($history.querySelectorAll("a") || []).forEach(node => {
                    let href = node.attributes.href,
                        target = node.attributes.target;
                    if (href == undefined || target != undefined) return;
                    node.target = "_blank";
                });
            })
        },

        /*
         * Idle Detection methods
         */

        setupIdleDetection() {
            window.addEventListener("keypress", this.deidle);
            window.addEventListener("mousemove", this.deidle);
        },

        // Common "de-idle" event handler
        deidle(e) {
            if (this.status === "idle") {
                this.status = "online";
            }

            if (this.idleTimeout !== null) {
                clearTimeout(this.idleTimeout);
            }

            this.idleTimeout = setTimeout(this.goIdle, 1000 * this.idleThreshold);
        },
        goIdle() {
            // only if we aren't already set on away
            if (this.status === "online") {
                this.status = "idle";
            }
        },

        /*
         * Webhook methods
         */
        isWebhookEnabled(name) {
            for (let webhook of this.config.webhookURLs) {
                if (webhook.Name === name && webhook.Enabled) {
                    return true;
                }
            }
            return false;
        },

        reportMessage(message) {
            // User is reporting a message on chat.
            if (message.reported) {
                if (!window.confirm("You have already reported this message. Do you want to report it again?")) return;
            }

            // Clone the message.
            let clone = Object.assign({}, message);

            // Sub out attached images.
            clone.message = clone.message.replace(/<img .+?>/g, "[inline image]");

            this.reportModal.message = clone;
            this.reportModal.origMessage = message;
            this.reportModal.classification = this.config.reportClassifications[0];
            this.reportModal.comment = "";
            this.reportModal.visible = true;
        },
        doReport() {
            // Submit the queued up report.
            if (this.reportModal.busy) return;
            this.reportModal.busy = true;

            let msg = this.reportModal.message;

            this.ws.conn.send(JSON.stringify({
                action: "report",
                channel: msg.channel,
                username: msg.username,
                timestamp: ""+msg.at,
                reason: this.reportModal.classification,
                message: msg.message,
                comment: this.reportModal.comment,
            }));

            this.reportModal.busy = false;
            this.reportModal.visible = false;

            // Set the "reported" flag.
            this.reportModal.origMessage.reported = true;
        },

        // Miscellaneous utility methods.
        isAppleWebkit() {
            // Try and detect whether the user is on an Apple Safari browser, which has
            // special nuances in their WebRTC video sharing support. This is intended to
            // detect: iPads, iPhones, and Safari on macOS.

            // By User-Agent.
            if (/iPad|iPhone|iPod/.test(navigator.userAgent)) {
                return true;
            }

            // By (deprecated) navigator.platform.
            if (navigator.platform === 'iPad' || navigator.platform === 'iPhone' || navigator.platform === 'iPod') {
                return true;
            }

            return false;
        },
    }
});

app.mount("#BareRTC-App");
