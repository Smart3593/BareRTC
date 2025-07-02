<script>
import 'floating-vue/dist/style.css';
import { Mentionable } from 'vue-mention';
import EmojiPicker from 'vue3-emoji-picker';

import AlertModal from './components/AlertModal.vue';
import LoginModal from './components/LoginModal.vue';
import ExplicitOpenModal from './components/ExplicitOpenModal.vue';
import ReportModal from './components/ReportModal.vue';
import MessageBox from './components/MessageBox.vue';
import WhoListRow from './components/WhoListRow.vue';
import VideoFeed from './components/VideoFeed.vue';
import ProfileModal from './components/ProfileModal.vue';
import MessageHistoryModal from './components/MessageHistoryModal.vue';

import ChatClient from './lib/ChatClient';
import LocalStorage from './lib/LocalStorage';
import VideoFlag from './lib/VideoFlag';
import StatusMessage from './lib/StatusMessage';
import { SoundEffects, DefaultSounds } from './lib/sounds';
import WatermarkImage from './lib/watermark';

import WebRTC from './lib/WebRTC.js';

const FileUploadMaxSize = 1024 * 1024 * 8; // 8 MB
const DebugChannelID = "barertc-debug";
const ReactNudgeNsfwMessageID = -451;

export default {
    name: 'BareRTC',
    components: {
        // Third party components
        Mentionable,
        EmojiPicker,

        // My components
        AlertModal,
        LoginModal,
        ExplicitOpenModal,
        ReportModal,
        MessageBox,
        WhoListRow,
        VideoFeed,
        ProfileModal,
        MessageHistoryModal,
    },
    mixins: [
        WebRTC.getMixin(),
    ],
    data() {
        return {
            // busy: false, // TODO: not used
            pageTitle: document.title,
            disconnect: false,    // don't try to reconnect (e.g. kicked)
            windowFocused: true,  // browser tab is active
            windowFocusedAt: new Date(),
            windowResizeTimer: null,
            windowResizeCooldown: false,
            appStartTime: new Date(),

            // Disconnect spamming: don't retry too many times.
            disconnectLimit: 2,
            disconnectCount: 0,

            // Temp: spam counting for OF links
            spamWarningCount: 0,

            // Website configuration provided by chat.html template.
            config: {
                branding: Branding,
                strings: BareRTCStrings,
                channels: PublicChannels,
                dmDisclaimer: DMDisclaimer,
                website: WebsiteURL,
                permitNSFW: PermitNSFW,
                webhookURLs: WebhookURLs,
                cacheHash: CacheHash,
                VIP: VIP,
                fontSizeClasses: [
                    ["x-2", "Very small chat room text"],
                    ["x-1", "50% smaller chat room text"],
                    ["", "Default size"],
                    ["x1", "50% larger chat room text"],
                    ["x2", "2x larger chat room text"],
                    ["x3", "3x larger chat room text"],
                    ["x4", "4x larger chat room text"],
                ],
                messageStyleSettings: [
                    ["cards", "Card style (default)"],
                    ["compact", "Compact style (with display names)"],
                    ["compact2", "Compact style (usernames only)"],
                ],
                imageDisplaySettings: [
                    ["show", "Always show images in chat"],
                    ["collapse", "Collapse images in chat, clicking to expand (default)"],
                    ["hide", "Never show images shared in chat"],
                ],
                sounds: {
                    available: SoundEffects,
                    settings: DefaultSounds,
                    ready: false,
                    audioContext: null,
                    audioTracks: {},
                },

                // Cached blocklist for the current user sent by your website.
                CachedBlocklist: CachedBlocklist,
            },

            // User JWT settings if available.
            jwt: {
                token: UserJWTToken,
                valid: UserJWTValid,
                claims: UserJWTClaims,
                rules: UserJWTRules
            },

            channel: "lobby",
            username: "", //"test",
            autoLogin: false,  // e.g. from JWT auth
            message: "",
            messageBox: null, // HTML element for message entry box
            typingNotifDebounce: null,
            status: "online", // away/idle status
            StatusMessage: StatusMessage,
            VideoFlag: VideoFlag,

            // Emoji picker visible for messages
            showEmojiPicker: false,

            // Idle detection variables
            idleTimeout: null,
            idleThreshold: 300, // number of seconds you must be idle

            // WebSocket connection.
            // Initialized in the dial() function.
            client: {},

            // Who List for the room.
            whoList: [],
            whoTab: 'online',
            whoSort: 'a-z',
            whoMap: {}, // map username to wholist entry
            whoOnline: {}, // map users actually online right now
            muted: {},  // muted usernames for client side state

            // Misc. user preferences (TODO: move all of them here)
            prefs: {
                usePolling: false,   // use the polling API instead of WebSockets.
                joinMessages: false, // hide "has entered the room" in public channels
                exitMessages: false, // hide exit messages by default in public channels
                watchNotif: true,    // notify in chat about cameras being watched
                closeDMs: false,     // ignore unsolicited DMs
                muteSounds: false,   // mute all sound effects
                theme: "auto",       // auto, light, dark theme
                debug: false,        // enable debugging features
            },

            // Chat history.
            channels: {
                // There will be values here like:
                // "lobby": {
                //   "history": [],
                //   "updated": timestamp,
                //   "unread": 4,
                //   "urgent": false, // true if at-mentioned
                // },
                // "@username": {
                //   "history": [],
                //   ...
                // }
            },
            historyScrollbox: null,
            autoscroll: true, // scroll to bottom on new messages
            fontSizeClass: "", // font size magnification
            messageStyle: "cards", // message display style
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

            // Loading older Direct Message chat history.
            directMessageHistory: {
                /* Will look like:
                "username": {
                    "busy": false,  // ajax request in flight
                    "beforeID": 1234, // next page cursor
                    "remaining": 50,  // number of older messages remaining
                }
                */
            },
            clearDirectMessages: {
                busy: false,
                ok: false,
                messagesErased: 0,
                timeout: null,
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

            // Generic Alert/Confirm modal to replace native browser events.
            // See also: modalAlert, modalConfirm functions.
            alertModal: {
                visible: false,
                isConfirm: false,
                title: "Alert",
                icon: "",
                message: "",
                backgroundDismissable: false,
                callback() {},
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
                user: {},  // full copy of the user for the modal component
                message: {},
                origMessage: {}, // pointer, so we can set the "reported" flag
            },

            profileModal: {
                visible: false,
                user: {},
                username: "",
            },

            photoShareModal: {
                visible: false,
                file: null,       // File handle from upload
                previewImg: null, // Image src for preview modal
                bytes: null,      // Bytes to send to server
            },

            messageHistoryModal: {
                visible: false,
            },
        };
    },
    mounted() {
        this.setupConfig(); // localSettings persisted settings
        this.setupIdleDetection();
        this.setupDropZone(); // file upload drag/drop

        // Export a handy sendMessage function to the global window scope.
        window.SendMessage = this.sendCommand;

        // Configure the StatusMessage controller.
        StatusMessage.nsfw = this.config.permitNSFW;
        StatusMessage.currentStatus = () => {
            return this.status;
        };
        StatusMessage.isAdmin = () => {
            return true; // Always treat user as admin for testing
        };

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

            if (this.windowResizeTimer !== null) {
                clearTimeout(this.windowResizeTimer);
            }

            this.windowResizeCooldown = true;
            this.windowResizeTimer = setTimeout(() => {
                this.windowResizeCooldown = false;
                this.windowResizeTimer = null;
            }, 1000);
        });

        // Listen for window focus/unfocus events. Being on a different browser tab, for
        // sound effect alert purposes, counts as not being "in" that chat channel when
        // a message comes in.
        window.addEventListener("focus", () => {
            this.windowFocused = true;
            this.windowFocusedAt = new Date();

            // If the current channel has unread indicators, clear them.
            let channel = this.channel;
            if (this.channels[channel].unread > 0) {
                this.channels[channel].unread = 0;
                this.channels[channel].urgent = false;
            }
        });
        window.addEventListener("blur", () => {
            this.windowFocused = false;
        });

        // Set up sound effects on first page interaction.
        window.addEventListener("click", () => {
            this.setupSounds();
        });
        window.addEventListener("keyup", () => {
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
            this.signIn(this.username);
        }
    },
    watch: {
        whoSort() {
            LocalStorage.set('whoSort', this.whoSort);
        },
        fontSizeClass() {
            // Store the setting persistently.
            LocalStorage.set('fontSizeClass', this.fontSizeClass);
        },
        messageStyle() {
            LocalStorage.set('messageStyle', this.messageStyle);
        },
        imageDisplaySetting() {
            LocalStorage.set('imageDisplaySetting', this.imageDisplaySetting);
        },
        scrollback() {
            LocalStorage.set('scrollback', this.scrollback);
        },
        status() {
            // Notify if going hidden.
            if (this.status === 'hidden' && this.isOp) {
                this.ChatClient(
                    "Your status is now set to 'hidden' which makes it appear as though you have logged out of chat.\n\n" +
                    "Try not to break the illusion by interacting with chat in this state. For safety, your message " +
                    "entry box will be disabled in public channels.",
                );
            }
            // Send presence updates to the server.
            this.sendMe();
        },
        pageTitleUnreadPrefix() {
            document.title = this.pageTitleUnreadPrefix + this.pageTitle;
        },

        // Misc preference watches
        "prefs.joinMessages": function () {
            LocalStorage.set('joinMessages', this.prefs.joinMessages);
        },
        "prefs.exitMessages": function () {
            LocalStorage.set('exitMessages', this.prefs.exitMessages);
        },
        "prefs.watchNotif": function () {
            LocalStorage.set('watchNotif', this.prefs.watchNotif);
        },
        "prefs.muteSounds": function () {
            LocalStorage.set('muteSounds', this.prefs.muteSounds);
        },
        "prefs.usePolling": function () {
            LocalStorage.set('usePolling', this.prefs.usePolling);

            // Reset the chat client on change.
            this.resetChatClient();
        },
        "prefs.closeDMs": function () {
            LocalStorage.set('closeDMs', this.prefs.closeDMs);

            // Tell ChatServer if we have gone to/from DND.
            this.sendMe();
        },
        "prefs.debug": function () {
            LocalStorage.set('debug', this.prefs.debug);
        },
        "prefs.theme": function() {
            LocalStorage.set('theme', this.prefs.theme);
        }
    },
    computed: {
        connected() {
            if (this.client.connected != undefined) {
                return this.client.connected();
            }
            return false;
        },
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
        currentDMPartner() {
            // If you are currently in a DM channel, get the User object of your partner.
            if (!this.isDM) return {};

            // If the user is not in the Who Map (e.g. from the history modal and the user is not online)
            let username = this.normalizeUsername(this.channel);
            if (this.whoMap[username] == undefined) {
                return {};
            }

            return this.whoMap[username];
        },
        pageTitleUnreadPrefix() {
            // When the page is not focused, put count of unread DMs in the title bar.
            if (this.windowFocused) return "";

            let count = 0;
            for (let channel of Object.keys(this.channels)) {
                if (channel.indexOf("@") === 0 && this.channels[channel].unread > 0) {
                    count += this.channels[channel].unread;
                }
            }

            return count > 0 ? `(${count}) ` : "";
        },
        chatPartnerStatusMessage() {
            // In a DM thread, returns your chat partner's status message.
            if (!this.isDM) {
                return null;
            }

            let username = this.normalizeUsername(this.channel),
                user = this.whoMap[username];
            if (user == undefined || this.isUserOffline(username)) {
                return this.StatusMessage.offline();
            }

            return this.StatusMessage.getStatus(user.status);
        },
        isChatPartnerAway() {
            // In a DM thread, returns if your chat partner's status is anything
            // other than "online".
            if (!this.isDM) return false;
            let status = this.chatPartnerStatusMessage;
            return status === null || status.name !== "online";
        },
        canUploadFile() {
            // User has the NoImage rule set.
            if (this.jwt.rules.IsNoImageRule) return false;

            // Public channels: check whether it PermitsPhotos.
            if (!this.isDM) {
                for (let cfg of this.config.channels) {
                    if (cfg.ID === this.channel && cfg.PermitPhotos) {
                        return true;
                    }
                }

                // By default: channels do not permit photos.
                return false;
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
            return true; // Always treat user as admin for testing
            // return this.jwt.claims.op || this.whoMap[this.username]?.op;
        },
        isVIP() {
            // Returns if the current user has VIP rights.
            return this.jwt.claims.vip;
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
                if (this.channels[channel.ID] != undefined) {
                    data.Unread = this.channels[channel.ID].unread;
                    data.Urgent = this.channels[channel.ID].urgent;
                    data.Updated = this.channels[channel.ID].updated;
                }
                result.push(data);
            }

            // Is the debug channel enabled?
            if (this.prefs.debug) {
                result.push({
                    ID: DebugChannelID,
                    Name: "Debug Log",
                });
            }
            return result;
        },
        atMentionItems() {
            // Available users in chat for the at-mentions support.
            let result = [
                {
                    value: "all",
                    label: "All people in the current room",
                    searchText: "all people in the current room"
                },
                {
                    value: "here",
                    label: "Everybody here in the current room",
                    searchText: "all people in the current room"
                },
            ];
            for (let user of this.whoList) {
                if (user.username === this.username) continue;
                result.push({
                    value: user.username,
                    label: user.nickname,
                    searchText: user.username + " " + user.nickname,
                });
            }
            return result;
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
                case "blue":
                    result.sort((a, b) => {
                        let left = (a.video & this.VideoFlag.Active) ? (a.video & this.VideoFlag.NSFW ? 1 : 2) : 0,
                            right = (b.video & this.VideoFlag.Active) ? (b.video & this.VideoFlag.NSFW ? 1 : 2) : 0;
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

        breakpoint() {
            // eslint-disable-next-line no-debugger
            debugger;
        },

        // Load user prefs from localStorage, called on startup
        setupConfig() {
            const settings = LocalStorage.getSettings();

            if (settings.fontSizeClass != undefined) {
                this.fontSizeClass = settings.fontSizeClass;
            }

            if (settings.messageStyle != undefined) {
                this.messageStyle = settings.messageStyle;
            }

            if (settings.videoScale != undefined) {
                this.webcam.videoScale = settings.videoScale;
            }

            if (settings.imageDisplaySetting != undefined) {
                this.imageDisplaySetting = settings.imageDisplaySetting;
            }

            if (settings.scrollback != undefined) {
                this.scrollback = parseInt(settings.scrollback);
            }

            // Stored user preferred device names for webcam/audio.
            if (settings.preferredDeviceNames != undefined) {
                let dev = settings.preferredDeviceNames;
                this.webcam.preferredDeviceNames.video = dev.video;
                this.webcam.preferredDeviceNames.audio = dev.audio;
            }

            // Webcam mutality preferences from last broadcast.
            if (settings.videoMutual === true) {
                this.webcam.mutual = true;
            }
            if (settings.videoMutualOpen === true) {
                this.webcam.mutualOpen = true;
            }
            if (settings.videoAutoMute === true) {
                this.webcam.autoMute = true;
            }
            if (settings.videoAutoShare === true) {
                this.webcam.autoshare = true;
            }
            if (settings.videoVipOnly === true) {
                this.webcam.vipOnly = true;
            }
            if (settings.videoExplicit === true && this.config.permitNSFW) {
                this.webcam.nsfw = true;
            }
            if (settings.videoNonExplicit === true) {
                this.webcam.nonExplicit = true;
            }
            if (settings.rememberExpresslyClosed === false) {
                this.webcam.rememberExpresslyClosed = false;
            }
            if (settings.autoMuteWebcams === true) {
                this.webcam.autoMuteWebcams = true;
            }
            if (settings.videoNoWatermark === true) {
                this.webcam.noWatermark = true;
            }

            // Misc preferences
            if (settings.usePolling != undefined) {
                this.prefs.usePolling = settings.usePolling === true;
            }
            if (settings.joinMessages != undefined) {
                this.prefs.joinMessages = settings.joinMessages === true;
            }
            if (settings.exitMessages != undefined) {
                this.prefs.exitMessages = settings.exitMessages === true;
            }
            if (settings.watchNotif != undefined) {
                this.prefs.watchNotif = settings.watchNotif === true;
            }
            if (settings.muteSounds != undefined) {
                this.prefs.muteSounds = settings.muteSounds === true;
            }
            if (settings.closeDMs != undefined) {
                this.prefs.closeDMs = settings.closeDMs === true;
            }
            if (this.prefs.debug != undefined) {
                this.prefs.debug = settings.debug === true;
            }
            if (settings.whoSort != undefined) {
                this.whoSort = settings.whoSort;
            }
            if (settings.theme != undefined) {
                this.prefs.theme = settings.theme;
            }
        },

        signIn(username) {
            this.username = username;
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

        onSelectEmoji(e) {
            // Callback from EmojiPicker to add an emoji to the user textbox.
            let selectionStart = this.messageBox.selectionStart;
            this.message = this.message.slice(0, selectionStart) + e.i + this.message.slice(selectionStart);
            this.hideEmojiPicker();
        },
        hideEmojiPicker() {
            // Hide the emoji menu (after sending an emoji or clicking the react button again)
            if (!this.showEmojiPicker) return;
            window.requestAnimationFrame(() => {
                this.showEmojiPicker = false;
                this.messageBox.focus();
                this.messageBox.selectionStart = this.messageBox.selectionEnd = this.messageBox.value.length;
            });
        },
        sendCommand(message) {
            // Automatically send a message to the chat server.
            // e.g. for the ProfileModal to send a "/kick username" command for easy operator buttons.
            let origMsg = this.message;
            this.message = message;
            this.sendMessage();
            this.message = origMsg;
        },
        sendMessage() {
            if (!this.message) {
                return;
            }

            if (!this.connected) {
                this.ChatClient("You are not connected to the server.");
                return;
            }

            // Safety for hidden status.
            if (this.status === 'hidden' && !this.isDM) {
                this.ChatClient("Your status is currently set to 'hidden' and it would break the illusion to talk in public channels.");
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
                    this.client.send({
                        action: "message",
                        channel: "lobby",
                        message: "**(Message of Shame)** I have been naughty and posted spam in chat despite being warned, " +
                            "and I am now being kicked from the room in shame. ☹️",
                    });

                    this.ChatServer(
                        "It is <strong>not allowed</strong> to promote your Onlyfans (or similar) " +
                        "site on the chat room. You have been removed from the chat room, and this " +
                        "incident has been reported to the site admin.",
                    );
                    this.pushHistory({
                        channel: this.channel,
                        username: this.username,
                        message: "has been kicked from the room!",
                        action: "presence",
                    });
                    this.disconnect = true;
                    this.client.ws.connected = false;
                    setTimeout(() => {
                        this.client.disconnect();
                    }, 1000);
                    return;
                }
                this.spamWarningCount++;

                this.ChatClient(
                    "Please <strong>do not</strong> send links to your Onlyfans (or similar sites) in the chat room. " +
                    "Those links are widely regarded to be spam and make a lot of people uncomfortable. " +
                    "If you violate this again, your account will be suspended.",
                );
                this.message = "";
                return;
            }

            // DEBUGGING: enable the 'debug' pref and see the debug channel.
            if (this.message.toLowerCase().indexOf("/toggle-debug-settings") === 0) {
                this.prefs.debug = !this.prefs.debug;
                this.ChatClient(
                    `Debug tools have been turned: <strong>${this.prefs.debug ? 'on' : 'off'}.</strong>`,
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

            // DEBUGGING: manual takeback admin command.
            match = this.message.match(/^\/takeback (\d+)$/i);
            if (match) {
                let msgID = parseInt(match[1]);
                this.client.send({
                    action: "takeback",
                    msgID: msgID,
                });
                this.ChatClient(`Takeback command send for message ID ${msgID}.`);
                this.message = "";
                return;
            }

            // DEBUGGING: fake open a broken video to see the error graphic
            if (this.message.toLowerCase().indexOf("/debug-broken-video") === 0) {
                this.WebRTC.streams["#broken"] = null;
                this.message = "";
                return;
            }

            // DEBUGGING: print WebRTC statistics
            if (this.message.toLowerCase().indexOf("/debug-webrtc") === 0) {
                let lines = [
                    "<strong>WebRTC PeerConnections:</strong>"
                ];
                for (let username of Object.keys(this.WebRTC.pc)) {
                    let pc = this.WebRTC.pc[username];
                    let line = `${username}: `;
                    if (pc.offerer != undefined) {
                        line += "offerer; ";
                    }
                    if (pc.answerer != undefined) {
                        line += "answerer; ";
                    }
                    lines.push(line);
                }

                this.ChatClient(lines.join("<br>"));
                this.message = "";
                return;
            }

            // DEBUGGING: print last dark video screenshot taken
            if (this.message.toLowerCase().indexOf("/debug-dark-video") === 0) {
                if (this.webcam.darkVideo.lastImage === null) {
                    this.ChatClient("There is no recent image available.");
                } else {
                    this.ChatClient(`
                        <strong>Dark Video Detector: Diagnostics</strong><br><br>
                        If your camera has been detected as being "too dark" but you believe this was an error, please
                        find a chat moderator (or visit the main website and contact the support team) for assistance:
                        you may be able to help us to resolve this error.<br><br>
                        In your message to an admin, please copy the following information:<br><br>
                        * The last average color detected from your video: ${JSON.stringify(this.webcam.darkVideo.lastAverage)}
                          <span style="background-color: ${this.webcam.darkVideo.lastAverageColor}">${this.webcam.darkVideo.lastAverageColor}</span><br>
                        * Your web browser user-agent: ${navigator.userAgent}<br><br>
                        Below is a recent screenshot from your webcam, which the chat page uses to find the average color
                        of your video feed. Note: if this image appears to be solid black, but your webcam was <strong>not</strong>
                        actually this dark, definitely let us know! It may point to a bug in the dark video detector:<br><br>
                        <img src="${this.webcam.darkVideo.lastImage}" width="160" height="120"><br><br>
                        For a troubleshooting tip: if your webcam is a removable USB device, please try <strong>closing your web browser,
                        unplugging and reconnecting your webcam, and then open your web browser again</strong> and see if the issue
                        is resolved. If that worked, also let a chat moderator know!
                    `);
                }
                this.message = "";
                return;
            }

            // DEBUGGING: reconnect to the server
            if (this.message.toLowerCase().indexOf("/reconnect") === 0) {
                this.resetChatClient();
                this.message = "";
                return;
            }

            // DEBUGGING: manually open a user camera. Note: chat server will still
            // enforce permissions and access.
            match = this.message.match(/^\/watch (.+?)$/i);
            if (match) {
                let username = match[1];
                this.openVideoByUsername(username, true);
                this.message = "";
                return;
            }

            // Clear user chat history.
            if (this.message.toLowerCase().indexOf("/clear-history") === 0) {
                this.clearMessageHistory();
                this.message = "";
                return;
            }

            // console.debug("Send message: %s", this.message);
            this.client.send({
                action: "message",
                channel: this.channel,
                message: this.message,
            });

            this.message = "";
        },

        sendTypingNotification() {
            // Send typing indicator for DM threads.
        },

        // Emoji reactions
        sendReact(message, emoji) {
            // Suppress reactions on restricted messages (e.g. when NoImage rule enabled and user did not see the image)
            if (message.message.indexOf("barertc-no-emoji-reactions") > -1) return;

            this.client.send({
                action: 'react',
                msgID: message.msgID,
                message: emoji,
            });
        },
        onReact(msg) {
            // Search all channels for this message ID and append the reaction.
            let msgID = msg.msgID,
                who = msg.username,
                emoji = msg.message;

            // Special react cases: nudge NSFW.
            if (msgID === ReactNudgeNsfwMessageID) {
                this.onNudgeNsfw(msg);
                return;
            }

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
                    delete (this.messageReactions[msgID][emoji]);
                }
                return;
            }

            this.messageReactions[msgID][emoji].push(who);
        },

        // Sync the current user state (such as video broadcasting status) to
        // the backend, which will reload everybody's Who List.
        sendMe(force) {
            if (!this.connected && !force) return;
            let payload = {
                action: "me",
                video: this.myVideoFlag,
                status: this.status,
                dnd: this.prefs.closeDMs,
            };
            this.client.send(payload);
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
            if (this.webcam.active && myNSFW != theirNSFW && theirNSFW) {
                this.webcam.nsfw = theirNSFW;
                window.requestAnimationFrame(() => {
                    this.webcam.wasServerNSFW = true;
                });
            }

            // Note: Me events only come when we join the server or a moderator has
            // flagged our video. This is as good of an "on connected" handler as we
            // get, so push over our cached blocklist from the website now.
            this.bulkMuteUsers();
        },

        // WhoList updates.
        onWho(msg) {
            let sendMe = false;  // re-send our 'me' at the end
            this.whoList = msg.whoList;
            this.whoOnline = {};

            if (this.whoList == undefined) {
                this.whoList = [];
            }

            for (let row of this.whoList) {
                // If we were watching this user's (blue) camera and we prefer non-Explicit,
                // and their camera is now becoming explicit (red), close it now.
                if (this.webcam.active && this.webcam.nonExplicit && this.WebRTC.streams[row.username] != undefined) {
                    if (!(this.whoMap[row.username].video & this.VideoFlag.NSFW)
                        && (row.video & this.VideoFlag.NSFW)) {
                        this.closeVideo(row.username, "offerer");
                    }
                }

                this.whoMap[row.username] = row;
                this.whoOnline[row.username] = true;

                // If we had a camera open with any of these and they have gone
                // off camera, close our side of the connection.
                if (this.WebRTC.streams[row.username] != undefined &&
                    !(row.video & this.VideoFlag.Active)) {
                    this.closeVideo(row.username, "offerer");
                }

                // If the server disagrees with our current status, send our status back.
                if (row.username === this.username && row.status !== this.status) {
                    sendMe = true;
                }
            }

            // Hang up on mutual cameras, if they changed their setting while we
            // are already watching them.
            this.unMutualVideo();
            this.unWatchNonExplicitVideo();

            // If we have any webcams open with users who are no longer in the Who List
            // (e.g.: can happen during a server reboot when the Who List goes empty),
            // close those video connections. Note: during normal room exit events this
            // is done on the onUserExited function - this is an extra safety check especially
            // in case of unexpected disconnect.
            for (let username of Object.keys(this.WebRTC.pc)) {
                if (this.whoOnline[username] == undefined) {
                    this.closeVideo(username);
                }
            }

            // Has the back-end server forgotten we are on video? This can
            // happen if we disconnect/reconnect while we were streaming.
            if (this.webcam.active && !(this.whoMap[this.username]?.video & this.VideoFlag.Active)) {
                sendMe = true;
            }

            // Do we need to set our me status again?
            if (sendMe) {
                // TODO: Delay a random period before sending the 'me' action, to help alleviate rough server reboots.
                // Ideally: onLoggedIn() sends a "me" with the user's status/video setting immediately. On a server
                // reboot, the first WhoList is sent 15 seconds after boot, and should (does!) send the correct "status"
                // flags for each user online (so they don't need to re-send "me" to correct it, triggering an exponential
                // flurry of WhoList updates). Randomize between 1 and 10 seconds.
                // The bug seems fixed but this timeout is a safety net.
                let timeout = 1000 + parseInt(Math.random() * 10000)
                setTimeout(this.sendMe, timeout);
            }
        },

        // Server side "block" event: for when the main website sends a BlockNow API request.
        onBlock(msg) {
            // Close any video connections we had with this user.
            this.closeVideo(msg.username);

            // Add it to our CachedBlocklist so in case the server reboots, we continue to sync it on reconnect.
            for (let existing of this.config.CachedBlocklist) {
                if (existing === msg.username) {
                    return;
                }
            }

            this.config.CachedBlocklist.push(msg.username);
        },

        // Server side "cut" event: tells the user to turn off their camera.
        onCut(msg) {
            this.DebugChannel(`Received cut command from server: ${JSON.stringify(msg)}`);
            this.stopVideo();
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
                            `You can not unmute <strong>${username}</strong> because you have blocked them on the main website. ` +
                            `To unmute them, you will need to unblock them on the website and then reload the chat room.`
                        );
                        return;
                    }
                }
            }

            // Common callback handler after the confirmation box.
            const callback = () => {
                // Hang up videos both ways.
                this.closeVideo(username);

                this.sendMute(username, mute);
                if (mute) {
                    this.ChatClient(
                        `You have muted <strong>${username}</strong> and will no longer see their chat messages, ` +
                        `and they will not see whether your webcam is active. You may unmute them via the Who Is Online list.`);
                } else {
                    this.ChatClient(
                        `You have unmuted <strong>${username}</strong> and can see their chat messages from now on.`,
                    );
                }
            };

            if (mute) {
                this.modalConfirm({
                    title: `Mute ${username}`,
                    icon: "fa fa-comment-slash",
                    message: `Do you want to mute ${username}? If muted, you will no longer see their ` +
                        `chat messages or any DMs they send you going forward. Also, ${username} will ` +
                        `not be able to see whether your webcam is active until you unmute them.`,
                }).then(() => {
                    this.muted[username] = true;
                    callback();
                });
            } else {
                this.modalConfirm({
                    title: `Un-mute ${username}`,
                    icon: "fa fa-comment",
                    message: `Do you want to remove your mute on ${username}? If you un-mute them, you ` +
                        `will be able to see their chat messages or DMs going forward, but most importantly, ` +
                        `they may be able to watch your webcam now if you are broadcasting!`,
                }).then(() => {
                    delete this.muted[username];
                    callback();
                });
            }
        },
        sendMute(username, mute) {
            this.client.send({
                action: mute ? "mute" : "unmute",
                username: username,
            });
        },
        isMutedUser(username) {
            return this.muted[this.normalizeUsername(username)] != undefined;
        },
        isBlockedUser(username) {
            if (this.config.CachedBlocklist.length > 0) {
                for (let user of this.config.CachedBlocklist) {
                    if (user === username) {
                        return true;
                    }
                }
            }
            return false;
        },
        bulkMuteUsers() {
            // On page load, if the website sent you a CachedBlocklist, mute all
            // of these users in bulk when the server connects.

            // If we have a blocklist from the main website, sync it to the server now.
            let mapBlockedUsers = {}; // usernames on our website blocklist
            if (this.config.CachedBlocklist.length > 0) {
                // Set the client side mute.
                let blocklist = this.config.CachedBlocklist;
                for (let username of blocklist) {
                    mapBlockedUsers[username] = true;
                    this.muted[username] = true;
                }

                // Send the username list to the server.
                this.client.send({
                    action: "blocklist",
                    usernames: blocklist,
                });
            }

            // While we're here, also re-sync our Boot list. e.g.: we were on webcam and we
            // booted someone off, then we got temporarily disconnected. The server has forgotten who
            // we booted and that person could then see our cam again.
            for (let username of Object.keys(this.WebRTC.booted)) {
                // Boot them again.
                this.sendBoot(username);
            }

            // Apply any temporary mutes that we had before the reconnect. Note: these are distinct
            // from blocks - blocks will make people invisible both ways to each other, mutes only
            // suppress messages but their Who List presence is maintained to each other.
            for (let username of Object.keys(this.muted)) {
                if (mapBlockedUsers[username]) continue;
                this.sendMute(username, true);
            }

            // Re-sync our video invite list on reconnect as well.
            this.sendInviteVideoBulk();
        },

        onUserExited(msg) {
            // A user has logged off the server. Clean up any WebRTC connections.
            this.closeVideo(msg.username);
        },

        // Handle messages sent in chat.
        onMessage(msg) {
            // Play sound effects if this is not the active channel or the window is not focused.
            if (msg.channel.indexOf("@") === 0) {
                this.initDirectMessageHistory(msg.channel, msg.msgID);

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
                timestamp: msg.timestamp,
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

            // Push it to the history of the public channels (respecting user preferences).
            if ((isJoin && this.prefs.joinMessages) || (isLeave && this.prefs.exitMessages)
                || (!isJoin && !isLeave)) {
                // Always put them in the first public channel.
                let channel = this.config.channels[0];
                this.pushHistory({
                    channel: channel.ID,
                    action: msg.action,
                    username: msg.username,
                    message: msg.message,
                });

                // If the current user is focused on another public channel, also post it there.
                if (!this.isDM && this.channel !== channel.ID) {
                    this.pushHistory({
                        channel: this.channel.ID,
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
            // Set up the ChatClient connection.
            this.client = new ChatClient({
                usePolling: this.prefs.usePolling,
                onClientError: this.ChatClient,

                username: this.username,
                jwt: this.jwt,
                prefs: this.prefs,

                onLoggedIn: this.onLoggedIn,
                onWho: this.onWho,
                onMe: this.onMe,
                onMessage: this.onMessage,
                onTakeback: this.onTakeback,
                onReact: this.onReact,
                onPresence: this.onPresence,
                onRing: this.onRing,
                onOpen: this.onOpen,
                onCandidate: this.onCandidate,
                onSDP: this.onSDP,
                onWatch: this.onWatch,
                onUnwatch: this.onUnwatch,
                onBlock: this.onBlock,
                onCut: this.onCut,
                onPing: this.onPing,

                bulkMuteUsers: this.bulkMuteUsers,
                focusMessageBox: () => {
                    this.messageBox.focus();
                },
                pushHistory: this.pushHistory,
                onNewJWT: jwt => {
                    this.jwt.token = jwt;
                },
            });

            this.client.dial();
        },
        resetChatClient() {
            if (!this.connected) return;

            // Reset the ChatClient, e.g. when toggling between WebSocket vs. Polling methods.
            this.ChatClient(
                "Your connection method to the chat server has been updated; attempting to reconnect now.",
            );

            window.requestAnimationFrame(() => {
                this.client.disconnect();
                setTimeout(() => {
                    this.dial();
                }, 1000);
            });
        },
        onLoggedIn() {
            // Called after the first 'me' is received from the chat server, e.g. once per login.

            // Load our watermark image.
            this.webcam.watermark = WatermarkImage(this.username);

            // Do we auto-broadcast our camera?
            if (this.webcam.autoshare) {
                this.startVideo({ force: true });
            }

            // Send the server our current status and video setting.
            this.sendMe(true);
        },
        onPing(msg) {
            // Updated JWT chat moderation rules?
            if (msg.jwtRules) {
                this.jwt.rules = msg.jwtRules;

                // Enforce them now.
                if (this.jwt.rules.IsNoBroadcastRule && this.webcam.active) {
                    this.stopVideo();
                }

                if (this.jwt.rules.IsNoVideoRule) {
                    this.closeOpenVideos();
                }

                if (this.jwt.rules.IsRedCamRule && this.webcam.active && !this.webcam.nsfw) {
                    this.webcam.nsfw = true;
                }
            }
        },

        /**
         * Front-end web app concerns.
         */

        // Generic window.alert replacement modal.
        async modalAlert({ message, title="Alert", icon="", isConfirm=false, backgroundDismissable=false }) {
            return new Promise((resolve, reject) => {
                this.alertModal.isConfirm = isConfirm;
                this.alertModal.title = title;
                this.alertModal.icon = icon;
                this.alertModal.message = message;
                this.alertModal.callback = () => {
                    resolve();
                };
                this.alertModal.backgroundDismissable = backgroundDismissable;
                this.alertModal.visible = true;
            });
        },
        async modalConfirm({ message, title="Confirmation", icon=""}) {
            return this.modalAlert({
                isConfirm: true,
                message,
                title,
                icon,
            })
        },
        modalClose() {
            this.alertModal.visible = false;
        },

        // Settings modal.
        showSettings() {
            this.settingsModal.visible = true;
        },
        hideSettings() {
            this.settingsModal.visible = false;
        },

        // Set active chat room.
        setChannel(channel) {
            this.channel = typeof (channel) === "string" ? channel : channel.ID;
            this.scrollHistory(this.channel, true);
            if (this.channels[this.channel]) {
                this.channels[this.channel].unread = 0;
                this.channels[this.channel].urgent = false;
            }

            // Responsive CSS: switch back to chat panel upon selecting a channel.
            this.openChatPanel();

            // Edit hyperlinks to open in a new window.
            this.makeLinksExternal();

            // Focus the message entry box.
            window.requestAnimationFrame(() => {
                this.messageBox.focus();
            });
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
            this.initDirectMessageHistory(channel);
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
        getUser(username) {
            // Return the full User object from the Who List, or a dummy placeholder if not online.
            if (this.whoMap[username] != undefined) {
                return this.whoMap[username];
            }

            return {
                username: username,
            };
        },
        isUserOffline(username) {
            // Return if the username is not presently online in the chat.
            return this.whoOnline[username] !== true && username !== 'ChatServer' && username !== 'ChatClient';
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
            }
            return username;
        },
        isUsernameDND(username) {
            if (!username) return false;
            return this.whoMap[username] != undefined && this.whoMap[username].dnd;
        },
        leaveDM(channel) {
            // Validate we're in a DM currently.
            if (channel.indexOf("@") !== 0) return;

            this.modalConfirm({
                title: "Close conversation thread",
                icon: "fa fa-trash",
                message: `Do you want to close this chat with ${channel}? This will remove the conversation from your view, but ` +
                    "your chat partner may still have the conversation open on their device.",
            }).then(() => {
                this.setChannel(this.config.channels[0].ID);
                delete (this.channels[channel]);
                delete (this.directMessageHistory[channel]);
            });
        },

        /* Take back messages (for everyone) or remove locally */
        takeback(msg) {
            this.modalConfirm({
                title: "Take back message",
                icon: "fa fa-rotate-left",
                message: "Do you want to take this message back? Doing so will remove this message from everybody's view in the chat room."
            }).then(() => {
                this.client.send({
                    action: "takeback",
                    msgID: msg.msgID,
                });
            });
        },
        removeMessage(msg) {
            this.modalConfirm({
                title: "Hide this message",
                icon: "fa fa-trash",
                message: "Do you want to remove this message from your view? This will delete the message only for you, but others in this chat thread may still see it."
            }).then(() => {
                this.onTakeback({
                    msgID: msg.msgID,
                });
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

        initHistory(channel) {
            if (this.channels[channel] == undefined) {
                this.channels[channel] = {
                    history: [],
                    updated: Date.now(),
                    unread: 0,
                    urgent: false,
                };
            }
        },
        pushHistory({ channel, username, message, action = "message", isChatServer, isChatClient, messageID, timestamp = null, unshift = false }) {

            // Ignore possibly-confusing ChatServer messages sent to admins.
            // TODO: add a 'super-admin' tier separately to operator that still sees these.
            if (isChatServer && (message.match(/ has booted you off of their camera!$/) || message.match(/ had booted you off their camera before, and won't be notified of your watch.$/))) {
                // Redirect it to the debug log channel.
                channel = DebugChannelID;
            }

            // Default channel = your current channel.
            if (!channel) {
                channel = this.channel;
            }

            // Assign a timestamp locally?
            if (timestamp === null) {
                timestamp = new Date();
            } else {
                timestamp = new Date(timestamp);
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
                if (this.jwt.rules.IsNoImageRule) {
                    // User is under the NoImage moderation rule.
                    message = `
                        <span class="has-text-danger barertc-no-emoji-reactions">
                            <i class="fa fa-image mr-1"></i>
                            An image was shared, but is not visible to you due to a chat moderation rule on your account.
                        </span>`;
                } else if (this.imageDisplaySetting === "hide") {
                    // User hides all images in their chat preferences.
                    return;
                } else if (this.imageDisplaySetting === "collapse") {
                    // Put a collapser link.
                    let collapseID = `collapse-${messageID}`;
                    if (!messageID) {
                        collapseID = "collapse-missingno-" + parseInt(Math.random()*100000);
                    }
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

                // Disable right click. TODO: move to server side.
                message = message.replace(/<img /g, `<img oncontextmenu="return false" `);
            }

            // Were we at mentioned in this message?
            if (message.indexOf("@" + this.username) > -1) {
                let re = new RegExp("@" + this.username + "\\b", "ig");
                message = message.replace(re, `<strong class="has-background-at-mention">@${this.username}</strong>`);

                // Play the sound effect if this is a public channel that isn't currently focused.
                if (channel.indexOf("@") !== 0 && (this.channel !== channel || !this.windowFocused)) {
                    this.playSound("Mentioned");
                    this.channels[channel].urgent = true;
                }
            }

            // And same for @here or @all
            message = message.replace(/@(here|all)\b/ig, `<strong class="has-background-at-mention">@$1</strong>`);

            // Append the message.
            let toAppend = {
                action: action,
                channel: channel,
                username: username,
                message: message,
                msgID: messageID,
                at: timestamp,
                isChatServer,
                isChatClient,
            };
            this.channels[channel].updated = new Date().getTime();
            if (unshift) {
                this.channels[channel].history.unshift(toAppend);
            } else {
                this.channels[channel].history.push(toAppend);
            }

            // Trim the history per the scrollback buffer.
            if (this.scrollback > 0 && this.channels[channel].history.length > this.scrollback) {
                this.channels[channel].history = this.channels[channel].history.slice(
                    -this.scrollback,
                    this.channels[channel].history.length + 1,
                );
            }

            // Scroll the history down.
            if (!unshift) {
                this.scrollHistory(channel);
            }

            // Mark unread notifiers if this is not our channel.
            if (this.channel !== channel || !this.windowFocused) {
                // Don't notify about presence broadcasts or history-backfilled messages.
                if (
                    channel !== DebugChannelID && // don't notify about debug channel
                    action !== "presence" &&
                    action !== "notification" &&
                    !isChatServer &&
                    !unshift  // not when backfilling old logs
                ) {
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

                if (this.historyScrollbox === null) return;
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
        DebugChannel(message) {
            this.pushHistory({
                channel: DebugChannelID,
                username: "ChatClient",
                message: message,
                isChatClient: true,
            });
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
                return result + "has-text-gender-male";
            } else if (gender.indexOf("f") === 0) {
                return result + "has-text-gender-female";
            } else if (gender.length > 0) {
                return result + "has-text-gender-other";
            }
            return "";
        },

        /**
         * Image sharing in chat
         */

        // Set up the HTML5 drag/drop handlers.
        setupDropZone() {
            let $dropArea = document.querySelector("#drop-modal");
            let $body = document.querySelector("body");

            // Set up drag/drop file upload events.
            $body.addEventListener("dragstart", (e) => {
                // Nothing ON the page should begin being draggable. Prevents on-page images from
                // being dragged and then dropped (and sent as files on chat), but still allows
                // their click handler to view in the lightbox modal.
                e.preventDefault();
                return false;
            })
            $body.addEventListener("dragenter", (e) => {
                e.preventDefault();
                e.stopPropagation();

                // Chrome bug: do not manipulate the DOM on drag start or it'll fire dragleave right away.
                window.requestAnimationFrame(() => {
                    $dropArea.classList.add("is-active");
                });
            });
            $body.addEventListener("dragover", (e) => {
                e.preventDefault();
                e.stopPropagation();
                $dropArea.classList.add("is-active");
            });
            $body.addEventListener("dragleave", (e) => {
                e.preventDefault();
                e.stopPropagation();
                $dropArea.classList.remove("is-active");
            });
            $body.addEventListener("drop", (e) => {
                e.preventDefault();
                e.stopPropagation();
                $dropArea.classList.remove("is-active");

                // Grab the file.
                let dt = e.dataTransfer;
                let file = dt.files[0];

                this.onFileSelected(file);
            });
        },

        // Common file selection handler for drag/drop or manual upload.
        onFileSelected(file) {
            // Show the share modal so the user previews before posting it.

            // Validate they can upload it here.
            if (!this.canUploadFile) {
                this.ChatClient("Photo sharing in this channel is not available.");
                return;
            }

            // Get image file data.
            let reader = new FileReader();
            let rawData = new ArrayBuffer();
            reader.addEventListener("load", () => {
                rawData = reader.result;

                let fileByteArray = [],
                    u8array = new Uint8Array(rawData);
                for (let i = 0; i < u8array.length; i++) {
                    fileByteArray.push(u8array[i]);
                }

                // Show the modal.
                this.photoShareModal.file = file;
                this.photoShareModal.previewImg = URL.createObjectURL(file);
                this.photoShareModal.bytes = fileByteArray;
                this.photoShareModal.visible = true;
            });

            reader.readAsArrayBuffer(file);
        },
        closePhotoShareModal() {
            this.photoShareModal.file = null;
            this.photoShareModal.previewImg = null;
            this.photoShareModal.bytes = null;
            this.photoShareModal.visible = false;
        },
        onFileUpload() {
            const file = this.photoShareModal.file,
                bytes = this.photoShareModal.bytes;
            this.closePhotoShareModal();

            // Validate they can upload it here.
            if (!this.canUploadFile) {
                this.ChatClient("Photo sharing in this channel is not available.");
                return;
            }

            // Prepare the message now so the channel name will be correct,
            // in case they upload a fat file and switch to a wrong channel
            // before the data is ready to send.
            let msg = {
                action: "file",
                channel: this.channel,
            };

            if (file.size > FileUploadMaxSize) {
                this.ChatClient(`Please share an image smaller than ${FileUploadMaxSize / 1024 / 1024} MB in size!`);
                return;
            }

            this.ChatClient(`<em>Uploading file to chat: ${file.name} - ${file.size} bytes, ${file.type} format.</em>`);

            // Attach the file to the message.
            msg.message = file.name;
            msg.bytes = bytes;

            // Send it to the chat server.
            this.client.send(msg);
        },

        // The image upload button handler.
        uploadFile() {
            let input = document.createElement('input');
            input.type = 'file';
            input.accept = 'image/*';
            input.onchange = e => {
                let file = e.target.files[0];
                this.onFileSelected(file);
            };
            input.click();
        },

        // Invoke the Profile Modal
        showProfileModal(username) {
            if (this.whoMap[username] != undefined) {
                this.profileModal.user = this.whoMap[username];
                this.profileModal.username = username;
                this.profileModal.visible = true;
            } else {
                // An offline user - profile minimal basic details.
                this.profileModal.user = {
                    username: username,
                    loginAt: 0,
                }
                this.profileModal.username = username;
                this.profileModal.visible = true;
            }
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
            } catch (e) {
                console.error("setupSounds:", e);
            }

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
                if ($history === null) return;

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

            // Also launch an interval to test for dev tools being open.
            let inter = setInterval(() => {
                const minimumTimeout = 400,
                    before = new Date().getTime();

                if (this.connected && !this.isOp) {
                    this.breakpoint();
                }

                const after = new Date().getTime();
                if (after - before > minimumTimeout) {

                    let uptime = (after - this.appStartTime.getTime()) / 1000,
                        uptimeStr = "",
                        vids = this.numVideosOpen,
                        ua = navigator.userAgent,
                        delta = after - before,
                        resized = this.windowResizeCooldown;
                    if (uptime > 3600) {
                        uptimeStr = `${(uptime / 3600).toFixed(2)}h`;
                    } else if (uptime > 60) {
                        uptimeStr = `${(uptime / 60).toFixed(2)}m`;
                    } else {
                        uptimeStr = `${uptime.toFixed(2)}s`;
                    }

                    // Low risk
                    if (vids === 0 && !resized) return;

                    clearInterval(inter);

                    this.doCustomReport({
                        message: {
                            channel: 'n/a',
                            username: this.username,
                            at: new Date(),
                            message: 'User has opened their browser dev tools!\n\n' +
                                `* Delta time in devtools check: ${delta}ms\n` +
                                `* Window resized recently: ${resized ? 'YES - Dev tools more likely!' : 'NO - Dev tools less likely or undocked'}\n` +
                                `* Web page uptime: ${uptimeStr} (${parseInt(uptime)}s)\n` +
                                `* Our camera active: ${this.webcam.active ? 'YES' : 'NO'}\n` +
                                `* Videos open: ${vids}\n` +
                                `* User Agent: ${ua}`,
                        },
                        classification: 'Community Safety Violation',
                        comment: `The user ${this.username} has opened their browser dev tools on chat, ` +
                            `which may indicate they are trying to reverse engineer or mess with the page!`
                    });

                    this.panic();
                }
            }, 100);
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
         * Direct Message History Loading
         */
        initDirectMessageHistory(channel, ignoreID) {
            if (this.directMessageHistory[channel] == undefined) {
                this.directMessageHistory[channel] = {
                    busy: false,
                    beforeID: 0,
                    ignoreID: ignoreID,
                    remaining: -1,
                };

                // Push the disclaimer message to the bottom of the chat history.
                let disclaimer = this.config.dmDisclaimer;
                this.pushHistory({
                    channel: channel,
                    username: "ChatServer",
                    message: disclaimer,
                    action: "notification",
                });

                // Immediately request the first page.
                window.requestAnimationFrame(() => {
                    this.loadDirectMessageHistory(channel).then(() => {
                        setTimeout(() => {
                            this.scrollHistory(channel);
                        }, 200);
                    });
                });
            }
        },
        async loadDirectMessageHistory(channel) {
            if (!this.jwt.valid) return;
            this.directMessageHistory[channel].busy = true;
            return fetch("/api/message/history", {
                method: "POST",
                mode: "same-origin",
                cache: "no-cache",
                credentials: "same-origin",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    "JWTToken": this.jwt.token,
                    "Username": this.normalizeUsername(channel),
                    "BeforeID": this.directMessageHistory[channel].beforeID,
                }),
            })
            .then((response) => response.json())
            .then((data) => {
                if (data.Error) {
                    console.error("DirectMessageHistory: ", data.Error);
                    return;
                }

                // No more messages?
                if (data.Messages.length === 0) {
                    this.directMessageHistory[channel].remaining = data.Remaining;
                    return;
                }

                // If this is the first of historic messages, insert a dividing line
                // (ChatServer presence message) to separate them.
                if (this.directMessageHistory[channel].remaining === -1) {
                    this.pushHistory({
                        channel: channel,
                        username: "ChatServer",
                        action: "presence",
                        message: "Messages from your past chat session are available above this line.",
                        unshift: true,
                    });
                }

                // Prepend these messages to the chat log.
                let beforeID = 0;
                for (let msg of data.Messages) {
                    beforeID = msg.msgID;

                    // Deduplicate: if this DM thread was opened because somebody sent us a message, their
                    // message will appear on the history side of the banner as well as the current side.
                    if (msg.msgID === this.directMessageHistory[channel].ignoreID) {
                        continue;
                    }

                    this.pushHistory({
                        channel: channel,
                        username: msg.username,
                        message: msg.message,
                        messageID: msg.msgID,
                        timestamp: msg.timestamp,
                        unshift: true,
                    });
                }

                // Update pagination state information.
                this.directMessageHistory[channel].remaining = data.Remaining;
                this.directMessageHistory[channel].beforeID = beforeID;
            }).catch(resp => {
                console.error("DirectMessageHistory: ", resp);
            }).finally(() => {
                this.directMessageHistory[channel].busy = false;
            });
        },
        async clearMessageHistory() {
            if (!this.jwt.valid || this.clearDirectMessages.busy) return;

            this.modalConfirm({
                title: "Clear all DMs",
                icon: "fa fa-exclamation-triangle",
                message: "This will delete all of your DMs history stored on the server. People you have " +
                    "chatted with will have their past messages sent to you erased as well.\n\n" +
                    "Note: messages that are currently displayed on your chat partner's screen will " +
                    "NOT be removed by this action -- if this is a concern and you want to 'take back' " +
                    "a message from their screen, use the 'take back' button (red arrow circle) on the " +
                    "message you sent to them. The 'clear history' button only clears the database, but " +
                    "does not send takebacks to pull the message from everybody else's screen.\n\n" +
                    "Are you sure you want to clear your stored DMs history on the server?",
            }).then(async () => {

                if (this.clearDirectMessages.timeout !== null) {
                    clearTimeout(this.clearDirectMessages.timeout);
                }

                this.clearDirectMessages.busy = true;
                return fetch("/api/message/clear", {
                    method: "POST",
                    mode: "same-origin",
                    cache: "no-cache",
                    credentials: "same-origin",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    body: JSON.stringify({
                        "JWTToken": this.jwt.token,
                    }),
                })
                .then((response) => response.json())
                .then((data) => {
                    if (data.Error) {
                        console.error("ClearMessageHistory: ", data.Error);
                        return;
                    }

                    this.clearDirectMessages.ok = true;
                    this.clearDirectMessages.messagesErased = data.MessagesErased;
                    this.clearDirectMessages.timeout = setTimeout(() => {
                        this.clearDirectMessages.ok = false;
                    }, 15000);

                    this.ChatClient(
                        "Your direct message history has been cleared from the server's database. "+
                        "(" + data.MessagesErased + " messages erased)",
                    );
                }).catch(resp => {
                    console.error("DirectMessageHistory: ", resp);
                    this.ChatClient("Error clearing your chat history: " + resp);
                }).finally(() => {
                    this.clearDirectMessages.busy = false;
                });
            });
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

        reportMessage(message, force=false) {
            // User is reporting a message on chat.
            if (message.reported && !force) {
                this.modalConfirm({
                    title: "Report Message",
                    icon: "fa fa-info-circle",
                    message: "You have already reported this message. Do you want to report it again?",
                }).then(() => {
                    this.reportMessage(message, true);
                });
                return;
            }

            // Clone the user object.
            let user = {
                avatar: this.avatarForUsername(message.username),
                nickname: this.nicknameForUsername(message.username),
            }

            // Clone the message.
            let clone = Object.assign({}, message);

            // Sub out attached images.
            clone.message = clone.message.replace(/<img .+?>/g, "[inline image]");

            this.reportModal.message = clone;
            this.reportModal.user = user;
            this.reportModal.origMessage = message;
            this.reportModal.visible = true;
        },
        doReport({ classification, comment }) {
            // Submit the queued up report.
            if (this.reportModal.busy) return;
            this.reportModal.busy = true;

            let msg = this.reportModal.message;

            this.client.send({
                action: "report",
                channel: msg.channel,
                username: msg.username,
                timestamp: "" + msg.at,
                reason: classification,
                message: msg.message,
                comment: comment,
            });

            this.reportModal.busy = false;
            this.reportModal.visible = false;

            // Set the "reported" flag.
            this.reportModal.origMessage.reported = true;
        },
        doCustomReport({ message, classification, comment }) {
            // A fully custom report, e.g. for the Ban Modal.
            this.reportModal.message = message;
            this.doReport({ classification, comment });
        },

        // Panic (critical error), triggered when the user seems to be messing around
        // with their browser's dev tools.
        panic() {
            // Stop ALL videos NOW.
            window.requestAnimationFrame(() => {
                this.closeOpenVideos();
                this.stopVideo();
            });

            // Sign out of chat.
            setTimeout(() => {
                this.client.disconnect()
            }, 5000);

            // Show the user an alert modal and reload the page.
            setTimeout(() => {
                this.modalAlert({
                    icon: "fa fa-skull-crossbones",
                    title: "Something has gone wrong",
                    message: "A critical error was detected. Please refresh the web page.\n\n" +
                        "If you experience this problem frequently, please contact a chat moderator for assistance " +
                        "and let them know what you were doing (if anything) before this error occurred.",
                });
            }, 100);
        },
    }
};
</script>

<template>
    <!-- Alert/Confirm modal: to avoid blocking the page with native calls. -->
    <AlertModal :visible="alertModal.visible"
        :is-confirm="alertModal.isConfirm"
        :title="alertModal.title"
        :icon="alertModal.icon"
        :message="alertModal.message"
        :background-dismissable="alertModal.backgroundDismissable"
        @callback="alertModal.callback"
        @close="modalClose()"></AlertModal>

    <!-- Sign In modal -->
    <LoginModal :visible="loginModal.visible" @sign-in="signIn"></LoginModal>

    <!-- Photo Drag/Drop Modal -->
    <div class="modal" id="drop-modal">
        <div class="modal-background"></div>
        <div class="modal-content">
            <div class="box content has-text-centered">
                <h1><i class="fa fa-upload mr-2"></i> Drop image to share it on chat</h1>
            </div>
        </div>
    </div>

    <!-- Photo preview before share modal -->
    <div class="modal" :class="{ 'is-active': photoShareModal.visible }">
        <div class="modal-background"></div>
        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-info">
                    <p class="card-header-title">
                        <i class="fa fa-image mr-2"></i>
                        Share a Photo
                    </p>
                </header>
                <div class="card-content">

                    <p class="block">
                        Sharing to channel:
                        {{ this.channelName }}
                    </p>

                    <!-- Preview image -->
                    <div class="columns is-centered is-mobile">
                        <div class="column is-half">
                            <img :src="photoShareModal.previewImg">
                        </div>
                    </div>
                </div>
                <footer class="card-footer">
                    <div class="card-footer-item">
                        <button type="button" class="button is-primary" @click="onFileUpload()">
                            Share Photo
                        </button>

                        <button type="button" class="button ml-2" @click="closePhotoShareModal()">
                            Cancel
                        </button>
                    </div>
                </footer>
            </div>
        </div>
    </div>

    <!-- Settings modal -->
    <div class="modal" :class="{ 'is-active': settingsModal.visible }">
        <div class="modal-background"></div>
        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-info">
                    <p class="card-header-title">Chat Settings</p>
                </header>
                <div class="card-content">

                    <!-- Tab bar for the settings -->
                    <div class="tabs">
                        <ul>
                            <li :class="{ 'is-active': settingsModal.tab === 'prefs' }">
                                <a href="#" @click.prevent="settingsModal.tab = 'prefs'">
                                    Display
                                </a>
                            </li>
                            <li :class="{ 'is-active': settingsModal.tab === 'sounds' }">
                                <a href="#" @click.prevent="settingsModal.tab = 'sounds'">
                                    Sounds
                                </a>
                            </li>
                            <li :class="{ 'is-active': settingsModal.tab === 'webcam' }">
                                <a href="#" @click.prevent="settingsModal.tab = 'webcam'">
                                    Camera
                                </a>
                            </li>
                            <li :class="{ 'is-active': settingsModal.tab === 'misc' }">
                                <a href="#" @click.prevent="settingsModal.tab = 'misc'">
                                    Misc
                                </a>
                            </li>
                            <li :class="{ 'is-active': settingsModal.tab === 'advanced' }">
                                <a href="#" @click.prevent="settingsModal.tab = 'advanced'">
                                    Advanced
                                </a>
                            </li>
                        </ul>
                    </div>

                    <!-- Display preferences -->
                    <div v-if="settingsModal.tab === 'prefs'">
                        <div class="field is-horizontal">
                            <div class="field-label is-normal">
                                <label class="label">Theme</label>
                            </div>
                            <div class="field-body">
                                <div class="field">
                                    <div class="control">
                                        <label class="radio">
                                            <input type="radio"
                                                v-model="prefs.theme"
                                                value="auto">
                                            Automatic
                                        </label>
                                        <label class="radio">
                                            <input type="radio"
                                                v-model="prefs.theme"
                                                value="light">
                                            Light
                                        </label>
                                        <label class="radio">
                                            <input type="radio"
                                                v-model="prefs.theme"
                                                value="dark">
                                            Dark
                                        </label>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="field is-horizontal">
                            <div class="field-label is-normal">
                                <label class="label">Video size</label>
                            </div>
                            <div class="field-body">
                                <div class="field">
                                    <div class="control">
                                        <div class="select is-fullwidth">
                                            <select v-model="webcam.videoScale">
                                                <option v-for="s in webcam.videoScaleOptions" v-bind:key="s[0]"
                                                    :value="s[0]">
                                                    {{ s[1] }}
                                                </option>
                                            </select>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="field is-horizontal">
                            <div class="field-label is-normal">
                                <label class="label">Text size</label>
                            </div>
                            <div class="field-body">
                                <div class="field">
                                    <div class="control">
                                        <div class="select is-fullwidth">
                                            <select v-model="fontSizeClass">
                                                <option v-for="s in config.fontSizeClasses" v-bind:key="s[0]" :value="s[0]">
                                                    {{ s[1] }}
                                                </option>
                                            </select>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="field is-horizontal">
                            <div class="field-label is-normal">
                                <label class="label">Text style</label>
                            </div>
                            <div class="field-body">
                                <div class="field">
                                    <div class="control">
                                        <div class="select is-fullwidth">
                                            <select v-model="messageStyle">
                                                <option v-for="s in config.messageStyleSettings" v-bind:key="s[0]"
                                                    :value="s[0]">
                                                    {{ s[1] }}
                                                </option>
                                            </select>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="field is-horizontal">
                            <div class="field-label is-normal">
                                <label class="label">Images</label>
                            </div>
                            <div class="field-body">
                                <div class="field">
                                    <div class="control">
                                        <div class="select is-fullwidth">
                                            <select v-model="imageDisplaySetting">
                                                <option v-for="s in config.imageDisplaySettings" v-bind:key="s[0]"
                                                    :value="s[0]">
                                                    {{ s[1] }}
                                                </option>
                                            </select>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>

                        <div class="field">
                            <label class="label">Scrollback buffer</label>
                            <div class="control">
                                <input type="number" class="input" v-model="scrollback" min="0" inputmode="numeric">
                            </div>
                            <p class="help">
                                How many chat history messages to keep at once (per channel/DM thread).
                                Older messages will be removed so your web browser doesn't run low on memory.
                                A value of zero (0) will mean "unlimited" and the chat history is never trimmed.
                            </p>
                        </div>

                    </div>

                    <!-- Sound settings -->
                    <div v-else-if="settingsModal.tab === 'sounds'">

                        <div class="columns mb-4">
                            <div class="column">
                                <label class="checkbox">
                                    <input type="checkbox" v-model="prefs.muteSounds" :value="true">
                                    Mute all sound effects
                                </label>
                            </div>
                            <div class="column">
                                <label class="checkbox">
                                    <input type="checkbox" v-model="webcam.autoMuteWebcams" :value="true">
                                    Automatically mute webcams
                                </label>
                            </div>
                        </div>

                        <div class="columns is-mobile">
                            <div class="column is-2 pr-1">
                                <label class="label is-size-7">DM chat</label>
                            </div>
                            <div class="column">
                                <div class="select is-fullwidth">
                                    <select v-model="config.sounds.settings.DM" @change="setSoundPref('DM')">
                                        <option v-for="s in config.sounds.available" v-bind:key="s.name" :value="s.name">
                                            {{ s.name }}
                                        </option>
                                    </select>
                                </div>
                            </div>

                            <div class="column is-2 pr-1">
                                <label class="label is-size-7">Public chat</label>
                            </div>
                            <div class="column">
                                <div class="select is-fullwidth">
                                    <select v-model="config.sounds.settings.Chat" @change="setSoundPref('Chat')">
                                        <option v-for="s in config.sounds.available" v-bind:key="s.name" :value="s.name">
                                            {{ s.name }}
                                        </option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        <div class="columns is-mobile">
                            <div class="column is-2 pr-1">
                                <label class="label is-size-7">Room enter</label>
                            </div>
                            <div class="column">
                                <div class="select is-fullwidth">
                                    <select v-model="config.sounds.settings.Enter" @change="setSoundPref('Enter')">
                                        <option v-for="s in config.sounds.available" v-bind:key="s.name" :value="s.name">
                                            {{ s.name }}
                                        </option>
                                    </select>
                                </div>
                            </div>

                            <div class="column is-2 pr-1">
                                <label class="label is-size-7">Room leave</label>
                            </div>
                            <div class="column">
                                <div class="select is-fullwidth">
                                    <select v-model="config.sounds.settings.Leave" @change="setSoundPref('Leave')">
                                        <option v-for="s in config.sounds.available" v-bind:key="s.name" :value="s.name">
                                            {{ s.name }}
                                        </option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        <div class="columns is-mobile">
                            <div class="column is-2 pr-1">
                                <label class="label is-size-7">Watched</label>
                            </div>
                            <div class="column">
                                <div class="select is-fullwidth">
                                    <select v-model="config.sounds.settings.Watch" @change="setSoundPref('Watch')">
                                        <option v-for="s in config.sounds.available" v-bind:key="s.name" :value="s.name">
                                            {{ s.name }}
                                        </option>
                                    </select>
                                </div>
                            </div>

                            <div class="column is-2 pr-1">
                                <label class="label is-size-7">Unwatched</label>
                            </div>
                            <div class="column">
                                <div class="select is-fullwidth">
                                    <select v-model="config.sounds.settings.Unwatch" @change="setSoundPref('Unwatch')">
                                        <option v-for="s in config.sounds.available" v-bind:key="s.name" :value="s.name">
                                            {{ s.name }}
                                        </option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        <div class="columns is-mobile">
                            <div class="column is-2 pr-1">
                                <label class="label is-size-7">@ Mentioned</label>
                            </div>
                            <div class="column">
                                <div class="select is-fullwidth pr-2">
                                    <select v-model="config.sounds.settings.Mentioned" @change="setSoundPref('Mentioned')">
                                        <option v-for="s in config.sounds.available" v-bind:key="s.name" :value="s.name">
                                            {{ s.name }}
                                        </option>
                                    </select>
                                </div>
                            </div>

                            <!-- For later expansion -->
                            <div class="column is-2 pr-1">
                                &nbsp;
                            </div>
                            <div class="column is-4">
                                &nbsp;
                            </div>
                        </div>
                    </div>

                    <!-- Webcam preferences -->
                    <div v-if="settingsModal.tab === 'webcam'">
                        <h3 class="subtitle mb-2">
                            Camera Settings
                        </h3>

                        <p class="block mb-1" v-if="config.permitNSFW">
                            <label class="label">Explicit webcam options</label>
                        </p>

                        <div class="field mb-1" v-if="config.permitNSFW">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.nsfw" :disabled="webcam.nonExplicit">
                                Mark my camera as featuring <i class="fa fa-fire has-text-danger mr-2"></i>
                                <span class="has-text-danger">Explicit</span> or sexual content
                            </label>
                        </div>

                        <div class="field" v-if="config.permitNSFW">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.nonExplicit">
                                I prefer not to see or be watched by Explicit cameras
                            </label>
                            <p class="help">
                                Don't auto-open explicit cameras when they open mine; and automatically
                                close a camera I am watching if it toggles to become explicit.<br>
                                <strong class="has-text-success">New:</strong>
                                People on explicit cameras can not watch your camera.
                            </p>
                        </div>

                        <p class="block mb-1">
                            <label class="label">Mutual webcam options</label>
                        </p>

                        <div class="field mb-1">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.mutual">
                                People must be sharing their own camera before they can open mine
                            </label>
                        </div>

                        <div class="field mb-1">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.mutualOpen">
                                When someone opens my camera, I also open their camera automatically
                            </label>
                        </div>

                        <div class="field mb-1">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.autoMuteWebcams">
                                <i class="fa fa-microphone-slash mx-1"></i> Automatically mute audio on other peoples' webcams
                            </label>
                        </div>

                        <div class="field mb-1" v-if="isVIP">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.vipOnly">
                                Only <span v-html="config.VIP.Branding"></span> <sup class="is-size-7"
                                    :class="config.VIP.Icon"></sup>
                                members can see that my camera is broadcasting
                            </label>
                        </div>

                        <div class="field">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.rememberExpresslyClosed">
                                Don't (automatically) reopen cameras that I have expressly closed
                            </label>
                            <p class="help">
                                If I click the 'X' button to expressly close a webcam, that video won't
                                auto-open again in case that person reopened my camera.
                            </p>
                        </div>

                        <h3 class="subtitle mb-2" v-if="webcam.videoDevices.length > 0 || webcam.audioDevices.length > 0">
                            Webcam Devices
                            <button type="button" class="button is-primary is-small is-outlined ml-2" @click="getDevices()"
                                title="Refresh list of devices">
                                <i class="fa fa-arrows-rotate" :class="{ 'fa-spin': webcam.gettingDevices }">
                                </i>
                            </button>
                        </h3>
                        <div class="columns is-mobile"
                            v-if="webcam.videoDevices.length > 0 || webcam.audioDevices.length > 0">

                            <div class="column">
                                <label class="label">Video source</label>
                                <div class="select is-fullwidth">
                                    <select v-model="webcam.videoDeviceID"
                                        @change="startVideo({ changeCamera: true, force: true })">
                                        <option v-for="(d, i) in webcam.videoDevices" :value="d.id" v-bind:key="i">
                                            {{ d.label || `Camera ${i}` }}
                                        </option>
                                    </select>
                                </div>
                            </div>

                            <div class="column">
                                <label class="label">Audio source</label>
                                <div class="select is-fullwidth">
                                    <select v-model="webcam.audioDeviceID"
                                        @change="startVideo({ changeCamera: true, force: true })">
                                        <option v-for="(d, i) in webcam.audioDevices" :value="d.id" v-bind:key="i">
                                            {{ d.label || `Microphone ${i}` }}
                                        </option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        <p class="block mb-1">
                            <label class="label">Miscellaneous</label>
                        </p>

                        <div class="field mb-1" v-if="webcam.videoDevices.length > 0">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.autoshare">
                                Automatically go on camera when I log onto the chat room
                            </label>
                            <p class="help">
                                Note: be sure that your web browser has <strong>remembered</strong> your webcam and mic
                                permission! This option can automatically share your webcam when you log onto chat again
                                from this device.
                            </p>
                        </div>

                        <div class="field mb-1">
                            <label class="checkbox">
                                <input type="checkbox" v-model="webcam.noWatermark">
                                Do not protect my video from screen recording
                            </label>
                            <p class="help">
                                The <i class="fa fa-qrcode"></i> QR codes are a safety feature against people screen recording your camera.
                                If you accept that risk, this setting can remove the QR code from <em>your own</em> webcam.
                                <a href="/about#watermarks" target="_blank">Learn more <i class="fa fa-external-link"></i></a>
                            </p>
                        </div>
                    </div>

                    <!-- Misc preferences -->
                    <div v-if="settingsModal.tab === 'misc'">

                        <div class="field">
                            <label class="label">Presence messages <small>('has joined the room')</small> in public
                                channels</label>
                            <div class="columns is-mobile mb-0">
                                <div class="column py-1">
                                    <label class="checkbox" title="Show 'has joined the room' messages in public channels">
                                        <input type="checkbox" v-model="prefs.joinMessages" :value="true">
                                        Join room
                                    </label>
                                </div>

                                <div class="column py-1">
                                    <label class="checkbox" title="Show 'has exited the room' messages in public channels">
                                        <input type="checkbox" v-model="prefs.exitMessages" :value="true">
                                        Exit room
                                    </label>
                                </div>
                            </div>
                        </div>

                        <div class="field">
                            <label class="label mb-0">Server notification messages</label>
                            <label class="checkbox" title="Show 'has joined the room' messages in public channels">
                                <input type="checkbox" v-model="prefs.watchNotif" :value="true">
                                Notify when somebody opens my camera
                            </label>
                        </div>

                        <div class="field">
                            <label class="label mb-0">Direct Messages</label>
                            <label class="checkbox mb-0">
                                <input type="checkbox" v-model="prefs.closeDMs" :value="true">
                                Ignore unsolicited DMs from others
                            </label>
                            <p class="help">
                                If you check this box, other chatters may not initiate DMs with you: their messages
                                will be (silently) ignored. You may still initiate DM chats with others, unless they
                                also have closed their DMs with this setting.
                            </p>
                        </div>

                        <!-- Clear DMs history on server -->
                        <div class="field" v-if="this.jwt.valid">
                            <a href="#" @click.prevent="clearMessageHistory()" class="button is-small has-text-danger">
                                <i class="fa fa-trash mr-1"></i> Clear direct message history
                            </a>

                            <div v-if="clearDirectMessages.busy" class="has-text-success mt-2 is-size-7">
                                <i class="fa fa-spinner fa-spin mr-1"></i>
                                Working...
                            </div>
                            <div v-else-if="clearDirectMessages.ok" class="has-text-success mt-2 is-size-7">
                                <i class="fa fa-check mr-1"></i>
                                History cleared ({{ clearDirectMessages.messagesErased }} message{{ clearDirectMessages.messagesErased === 1 ? '' : 's' }} erased)
                            </div>
                        </div>

                    </div>

                    <!-- Advanced preferences -->
                    <div v-if="settingsModal.tab === 'advanced'">

                        <div class="field">
                            <label class="label mb-0">
                                Server Connection Method
                            </label>
                            <label class="checkbox">
                                <input type="radio" v-model="prefs.usePolling" :value="false">
                                WebSockets (realtime connection; recommended for most people)
                            </label>
                            <label class="checkbox">
                                <input type="radio" v-model="prefs.usePolling" :value="true">
                                Polling (check for new messages every 5 seconds)
                            </label>
                            <p class="help">
                                By default the chat server requires a constant WebSockets connection to stay online.
                                If you are experiencing frequent disconnects (e.g. because you are on a slow or
                                unstable network connection), try switching to the "Polling" method which will be
                                more robust, at the cost of up to 5-seconds latency to receive new messages.

                                <!-- If disconnected currently, tell them to refresh. -->
                                <span v-if="!connected" class="has-text-danger">
                                    Notice: you may need to refresh the chat page after changing this setting.
                                </span>
                            </p>
                        </div>

                        <div class="field" v-if="isOp || prefs.debug">
                            <label class="label mb-0">
                                Stats for nerds
                            </label>
                            <label class="checkbox">
                                <input type="checkbox"
                                    v-model="prefs.debug"
                                    :value="true">
                            </label>
                            Enable the "Debug Log" channel.
                            <p class="help">
                                This enables a channel where under-the-hood debug messages may be posted,
                                e.g. to debug WebRTC connection problems.
                            </p>
                        </div>

                    </div>

                </div>
                <footer class="card-footer">
                    <div class="card-footer-item">
                        <button type="button" class="button is-primary" @click="hideSettings()">
                            Close
                        </button>
                    </div>
                </footer>
            </div>
        </div>
    </div>

    <!-- NSFW Modal: before user activates their webcam -->
    <div class="modal" :class="{ 'is-active': nsfwModalCast.visible }">
        <div class="modal-background"></div>
        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-info">
                    <p class="card-header-title">Select Webcam Options</p>
                </header>
                <div class="card-content">
                    <p class="block mb-1">
                        You can turn on your webcam and enable others in the room to connect to yours.
                        The controls to <i class="fa fa-stop has-text-danger"></i> stop and <i
                            class="fa fa-microphone-slash has-text-danger"></i> mute audio
                        will be at the top of the page.
                    </p>

                    <div class="field">
                        <label class="checkbox">
                            <input type="checkbox" v-model="webcam.autoMute">
                            Start with my microphone on mute by default
                        </label>
                    </div>

                    <p v-if="config.permitNSFW" class="block mb-1">
                        <label class="label">'Explicit' webcam options</label>
                    </p>

                    <div class="field" v-if="config.permitNSFW">
                        <label class="checkbox">
                            <input type="checkbox" v-model="webcam.nsfw" :disabled="webcam.nonExplicit">
                            Mark my camera as featuring <i class="fa fa-fire has-text-danger mr-2"></i>
                            <span class="has-text-danger">Explicit</span> or sexual content
                        </label>
                        <p class="help">
                            You can toggle this at any time by clicking on the '<i class="fa fa-fire"></i> Explicit'
                            button at the top of the page.
                        </p>
                    </div>

                    <div class="field" v-if="config.permitNSFW">
                        <label class="checkbox">
                            <input type="checkbox" v-model="webcam.nonExplicit">
                            I prefer not to see or be watched by Explicit cameras
                        </label>
                        <p class="help">
                            Close, and don't automatically open, other peoples' cameras when they toggle
                            to become explicit.<br>
                            <strong class="has-text-success">New:</strong>
                            People on explicit cameras can not watch your camera.
                        </p>
                    </div>

                    <p class="block mb-1">
                        <label class="label">Mutual webcam options</label>
                    </p>

                    <div class="field mb-1">
                        <label class="checkbox">
                            <input type="checkbox" v-model="webcam.mutual">
                            People must be sharing their own camera before they can open mine
                        </label>
                    </div>

                    <div class="field mb-1">
                        <label class="checkbox">
                            <input type="checkbox" v-model="webcam.mutualOpen">
                            When someone opens my camera, I also open their camera automatically
                        </label>
                    </div>

                    <div class="field" v-if="isVIP">
                        <label class="checkbox">
                            <input type="checkbox" v-model="webcam.vipOnly">
                            Only <span v-html="config.VIP.Branding"></span> <sup class="is-size-7"
                                :class="config.VIP.Icon"></sup>
                            members can see that my camera is broadcasting
                        </label>
                    </div>

                    <!--
                    Device Pickers: just in case the user had granted video permission in the past,
                    and we are able to enumerate their device names, we can show them here before they
                    go on this time.
                    -->
                    <div class="columns is-mobile" v-if="webcam.videoDevices.length > 0 || webcam.audioDevices.length > 0">

                        <div class="column">
                            <label class="label">Video source</label>
                            <div class="select is-fullwidth">
                                <select v-model="webcam.videoDeviceID">
                                    <option :value="null" disabled selected>Select default camera</option>
                                    <option v-for="(d, i) in webcam.videoDevices" :value="d.id" v-bind:key="i">
                                        {{ d.label || `Camera ${i}` }}
                                    </option>
                                </select>
                            </div>
                        </div>

                        <div class="column">
                            <label class="label">Audio source</label>
                            <div class="select is-fullwidth">
                                <select v-model="webcam.audioDeviceID">
                                    <option :value="null" disabled selected>Select default microphone</option>
                                    <option v-for="(d, i) in webcam.audioDevices" :value="d.id" v-bind:key="i">
                                        {{ d.label || `Microphone ${i}` }}
                                    </option>
                                </select>
                            </div>
                        </div>
                    </div>

                    <div class="field">
                        <div class="control has-text-centered">
                            <button type="button" class="button" @click="nsfwModalCast.visible = false">Cancel</button>
                            <button type="button" class="button is-success ml-2"
                                @click="startVideo({ force: true }); nsfwModalCast.visible = false">Start webcam</button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- NSFW Modal: before user views a NSFW camera the first time -->
    <ExplicitOpenModal :visible="nsfwModalView.visible" :user="nsfwModalView.user"
        @accept="openVideo(nsfwModalView.user, true)" @cancel="nsfwModalView.visible = false"
        @dont-show-again="setSkipNSFWModal()"></ExplicitOpenModal>

    <!-- Report Modal -->
    <ReportModal :visible="reportModal.visible" :busy="reportModal.busy" :user="reportModal.user"
        :message="reportModal.message" @accept="doReport" @cancel="reportModal.visible = false"></ReportModal>

    <!-- Profile Modal (profile cards popup) -->
    <ProfileModal
        :visible="profileModal.visible"
        :user="profileModal.user"
        :username="username"
        :is-viewer-op="isOp"
        :jwt="jwt.token"
        :website-url="config.website"
        :is-dnd="isUsernameDND(profileModal.username)"
        :is-muted="isMutedUser(profileModal.username)"
        :is-booted="isBooted(profileModal.username)"
        :profile-webhook-enabled="isWebhookEnabled('profile')"
        :vip-config="config.VIP"
        :is-watching="isWatching(profileModal.username)"
        :my-video-active="webcam.active"
        :is-invited-video="isInvited(profileModal.username)"
        @send-dm="openDMs"
        @mute-user="muteUser"
        @boot-user="bootUser"
        @send-command="sendCommand"
        @nudge-nsfw="sendNudgeNsfw"
        @report="doCustomReport"
        @invite-video="inviteToWatch"
        @cancel="profileModal.visible = false"></ProfileModal>

    <!-- DMs History of Usernames Modal -->
    <MessageHistoryModal
        :visible="messageHistoryModal.visible"
        :jwt="jwt.token"
        @open-chat="openDMs"
        @cancel="messageHistoryModal.visible = false"
    ></MessageHistoryModal>

    <div class="chat-container">

        <!-- Top header panel -->
        <header class="chat-header">
            <div class="columns is-mobile">
                <div class="column is-narrow pr-1">
                    <strong class="is-6"><span v-html="config.branding"></span></strong>
                </div>
                <div class="column px-1">
                    <!-- Stop/Start video buttons -->
                    <button type="button" v-if="webcam.active" class="button is-small is-danger px-1" @click="stopVideo()">
                        <i class="fa fa-stop mr-2"></i>
                        Stop
                    </button>
                    <button type="button" v-else class="button is-small is-success px-1" @click="startVideo({})"
                        :disabled="webcam.busy">
                        <i class="fa fa-video mr-2"></i>
                        Share webcam
                    </button>

                    <!-- Mute/Unmute my mic buttons (if streaming)-->
                    <button type="button" v-if="webcam.active && !webcam.muted" class="button is-small is-success ml-1 px-1"
                        @click="muteMe()">
                        <i class="fa fa-microphone mr-2"></i>
                        Mute
                    </button>
                    <button type="button" v-if="webcam.active && webcam.muted" class="button is-small is-danger ml-1 px-1"
                        @click="muteMe()">
                        <i class="fa fa-microphone-slash mr-2"></i>
                        Unmute
                    </button>

                    <!-- Watchers button -->
                    <button type="button" v-if="webcam.active" class="button is-small is-link is-outlined ml-1 px-1"
                        @click="showViewers()">
                        <i class="fa fa-eye mr-2"></i>
                        {{ Object.keys(webcam.watching).length }}
                    </button>

                    <!-- NSFW toggle button -->
                    <button type="button" v-if="webcam.active && config.permitNSFW" class="button is-small px-1 ml-1"
                        :class="{
                            'is-outlined is-link has-text-dark': !webcam.nsfw,
                            'is-danger': webcam.nsfw
                        }" @click.prevent="topNavExplicitButtonClicked()"
                        :title="webcam.nonExplicit ? 'You prefer non-Explicit cameras so you can not go Explicit' : 'Toggle the NSFW setting for your camera broadcast'"
                        :disabled="webcam.nonExplicit">
                        <i class="fa fa-fire mr-1" :class="{ 'has-text-danger': !webcam.nsfw }"></i> Explicit
                    </button>
                </div>
                <div class="column dropdown is-right is-narrow pl-1" id="chat-settings-hamburger-menu">
                    <!-- Note: the onclick for the previous div is handled in index.html -->

                    <div class="dropdown-trigger">
                        <button type="button" class="button is-small is-link px-2" aria-haspopup="true"
                            aria-controls="chat-settings-menu">
                            <span>
                                <i class="fa fa-bars"></i>
                            </span>
                        </button>
                    </div>

                    <div class="dropdown-menu mr-3" id="chat-settings-menu" role="menu">
                        <div class="dropdown-content">
                            <a href="#" class="dropdown-item" @click.prevent="showSettings()">
                                <i class="fa fa-gear mr-1"></i> Chat Settings
                            </a>

                            <a href="#" class="dropdown-item" v-if="numVideosOpen > 0" @click.prevent="closeOpenVideos()">
                                <i class="fa fa-video-slash mr-1"></i> Close all cameras
                            </a>

                            <a href="#" class="dropdown-item" v-if="numVideosOpen > 0" @click.prevent="muteAllVideos()">
                                <i class="fa fa-microphone-slash mr-1"></i> Mute all cameras
                            </a>

                            <hr class="dropdown-divider">

                            <a href="/about" target="_blank" class="dropdown-item">
                                <i class="fa fa-info-circle mr-1"></i> About
                            </a>

                            <a href="/logout" class="dropdown-item">
                                <i class="fa fa-arrow-right-from-bracket mr-1"></i> Log out
                            </a>
                        </div>
                    </div>
                </div>
            </div>
        </header>

        <!-- Left Column: Channels & DMs -->
        <div class="left-column">
            <div class="card grid-card">
                <header class="card-header has-background-success">
                    <div class="columns is-mobile card-header-title">
                        <div class="column is-narrow mobile-only">
                            <button type="button" class="button is-success px-2 py-1" @click="openChatPanel">
                                <i class="fa fa-arrow-left"></i>
                            </button>
                        </div>
                        <div class="column">Channels</div>
                    </div>
                </header>
                <div class="card-content">
                    <aside class="menu">
                        <p class="menu-label">
                            Public Channels
                        </p>

                        <ul class="menu-list">
                            <li v-for="c in activeChannels" v-bind:key="c.ID">
                                <a :href="'#' + c.ID" @click.prevent="setChannel(c)"
                                    :class="{ 'is-active': c.ID == channel }">
                                    {{ c.Name }}
                                    <span v-if="c.Unread" class="tag" :class="{'is-success': !c.Urgent, 'is-danger': c.Urgent}">
                                        {{ c.Unread }}
                                    </span>
                                </a>
                            </li>
                        </ul>

                        <p class="menu-label">
                            Direct Messages
                        </p>

                        <ul class="menu-list">
                            <li v-for="c in activeDMs" v-bind:key="c.channel">
                                <a :href="'#' + c.channel" @click.prevent="setChannel(c.channel)"
                                    :class="{ 'is-active': c.channel == channel }">

                                    <div class="columns is-mobile">
                                        <!-- Avatar URL if available (copied from Who List) -->
                                        <div class="column is-narrow pr-0" style="position: relative">
                                            <img v-if="avatarForUsername(normalizeUsername(c.channel))"
                                                :src="avatarForUsername(normalizeUsername(c.channel))" width="24"
                                                height="24" alt=""
                                                :class="{ 'offline-avatar': isUserOffline(c.name) }">
                                            <img v-else src="/static/img/shy.png" width="24" height="24">
                                        </div>

                                        <div class="column">
                                            <div class="forcibly-truncate-wrapper forcibly-single-line" style="height: 24px">
                                                <div class="forcibly-truncate-body">
                                                    <span v-if="c.unread" class="tag is-danger mr-1">
                                                        {{ c.unread }}
                                                    </span>

                                                    <del v-if="isUserOffline(c.name)">
                                                        {{ c.name }}
                                                    </del>
                                                    <span v-else>{{ c.name }}</span>
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                </a>
                            </li>

                            <!-- History button -->
                            <li v-if="jwt.token">
                                <a href="#" @click.prevent="messageHistoryModal.visible = !messageHistoryModal.visible">
                                    <i class="fa fa-clock mr-1"></i>
                                    History
                                </a>
                            </li>
                        </ul>
                    </aside>

                    <!-- Close new DMs toggle -->
                    <div class="tag mt-2">
                        <label class="checkbox">
                            <input type="checkbox" v-model="prefs.closeDMs" :value="true">
                            Ignore unsolicited DMs

                            <a href="#"
                                @click.prevent="modalAlert({title: 'About Ignoring Unsolicited DMs', message: 'When this box is checked, your DMs will be closed to new conversations and the button for others to send you a DM will be greyed out/disabled.\n\nYou may still initiate new DMs with others yourself, and continue conversations with people who you already have DM threads open with.', backgroundDismissable: true})"
                                class="fa fa-info-circle ml-2">
                            </a>
                        </label>
                    </div>

                </div>
            </div>
        </div>

        <!-- Middle Column: Chat Room/History -->
        <div class="chat-column">

            <div class="card grid-card">
                <header class="card-header" :class="{ 'has-background-private': isDM, 'has-background-link': !isDM }">
                    <div class="columns is-mobile card-header-title has-text-light">

                        <!-- Responsive mobile button to pan to Left Column -->
                        <div class="column is-narrow mobile-only pr-0">
                            <button type="button" class="button is-success px-2 py-1" @click="openChannelsPanel">
                                <i class="fa fa-comments"></i>

                                <!-- Indicator badge for unread messages -->
                                <span v-if="hasAnyUnread() > 0" class="tag ml-1" :class="{ 'is-danger': anyUnreadDMs() }">
                                    {{ hasAnyUnread() }}
                                </span>
                            </button>
                        </div>

                        <!-- If this is a DM thread and the chat partner is on webcam, show the video button -->
                        <div v-if="isDM && isUsernameOnCamera(currentDMPartner.username)"
                            class="column is-narrow pr-0">
                            <button type="button" class="button px-2 py-1"
                                :class="webcamButtonClass(currentDMPartner.username)"
                                :title="`View ${channel}'s camera`"
                                @click="openVideo(currentDMPartner)">
                                <i class="fa" :class="webcamIconClass(currentDMPartner)"></i>
                            </button>
                        </div>

                        <!-- Channel title -->
                        <div class="column">

                            <!-- This forcibly crops the title in case someone's username is too long -->
                            <div class="forcibly-truncate-wrapper">
                                &nbsp; <!-- For natural height of parent (relative) container -->

                                <div class="forcibly-truncate-body">
                                    <!-- On a DM thread, clicking the username opens their profile card. -->
                                    <div v-if="isDM">
                                        <a href="#" class="has-text-light" @click.prevent="showProfileModal(currentDMPartner.username)">
                                            {{ channelName }}
                                        </a>
                                    </div>
                                    <div v-else>
                                        {{ channelName }}
                                    </div>
                                </div>
                            </div>
                        </div>

                        <!-- Easy video zoom buttons -->
                        <div class="column is-narrow is-hidden-mobile" v-if="anyVideosOpen">
                            <button type="button" class="button is-small is-outlined mr-1" :disabled="webcam.videoScale === 'x4'"
                                @click="settingsModal.tab='webcam'; showSettings()">
                                <i class="fa fa-gear"></i>
                            </button>

                            <button type="button" class="button is-small is-outlined mr-1" :disabled="webcam.videoScale === 'x4'"
                                @click="scaleVideoSize(true)">
                                <i class="fa fa-magnifying-glass-plus"></i>
                            </button>

                            <button type="button" class="button is-small is-outlined" :disabled="webcam.videoScale === ''"
                                @click="scaleVideoSize(false)">
                                <i class="fa fa-magnifying-glass-minus"></i>
                            </button>
                        </div>

                        <!-- DM thread buttons -->
                        <div class="column is-narrow" v-if="isDM">
                            <!-- DMs: Leave convo button -->
                            <button type="button" class="float-right button is-small is-warning is-outlined"
                                @click="leaveDM(channel)">
                                <i class="fa fa-trash"></i>
                            </button>
                        </div>

                        <!-- Who List button: Responsive mobile button to pan to Right Column -->
                        <div class="column is-narrow pl-0 mobile-only">
                            <button type="button" class="button is-success px-2 py-1" @click="openWhoPanel">
                                <i class="fa fa-user-group"></i>
                            </button>
                        </div>
                    </div>
                </header>
                <div id="video-feeds" class="video-feeds" :class="webcam.videoScale"
                    v-show="webcam.active || Object.keys(WebRTC.streams).length > 0">
                    <!-- Video Feeds-->

                    <!-- My video -->
                    <VideoFeed v-show="webcam.active"
                        :local-video="true"
                        :username="username"
                        :popped-out="WebRTC.poppedOut[username]"
                        :is-explicit="webcam.nsfw"
                        :is-muted="webcam.muted"
                        :is-source-muted="webcam.muted"
                        :is-speaking="WebRTC.speaking[username]"
                        :watermark-image="webcam.watermark"
                        :is-no-watermark="webcam.noWatermark"
                        @mute-video="muteMe()"
                        @popout="popoutVideo"
                        @open-profile="showProfileModal"
                        @set-volume="setVideoVolume"
                        @modal-alert="modalAlert"
                        @watermark-removed="onDeletedWatermark">
                    </VideoFeed>

                    <!-- Others' videos -->
                    <VideoFeed v-for="(stream, username) in WebRTC.streams"
                        v-bind:key="username"
                        :username="username"
                        :popped-out="WebRTC.poppedOut[username]"
                        :is-speaking="WebRTC.speaking[username]"
                        :is-explicit="isUsernameCamNSFW(username)"
                        :is-source-muted="isSourceMuted(username)"
                        :is-muted="isMuted(username)"
                        :is-watching-me="isWatchingMe(username)"
                        :is-frozen="WebRTC.frozenStreamDetected[username]"
                        :watermark-image="webcam.watermark"
                        :is-no-watermark="isUsernameCamNoWatermark(username)"
                        @reopen-video="openVideoByUsername"
                        @mute-video="muteVideo"
                        @popout="popoutVideo"
                        @close-video="expresslyCloseVideo"
                        @set-volume="setVideoVolume"
                        @open-profile="showProfileModal"
                        @modal-alert="modalAlert"
                        @watermark-removed="onDeletedWatermark">
                    </VideoFeed>

                    <!-- Debugging - copy a lot of these to simulate more videos -->

                    <!-- <div class="feed">
                        hi
                    </div>
                    <div class="feed">
                        hi
                    </div>
                    <div class="feed">
                        hi
                    </div>
                    <div class="feed">
                        hi
                    </div> -->

                </div>
                <div class="card-content" id="chatHistory" :class="{
                    'has-background-dm': isDM,
                    'p-1 pb-5': messageStyle.indexOf('compact') === 0
                }">

                    <!-- Show your chat partner's status message in DMs -->
                    <div class="user-status-dm-field tag is-info" v-if="isChatPartnerAway">
                        <strong class="mr-2 has-text-light">Status:</strong>
                        <span v-if="chatPartnerStatusMessage">
                            {{ chatPartnerStatusMessage.emoji }} {{ chatPartnerStatusMessage.label }}
                        </span>
                        <em v-else>undefined</em>
                    </div>

                    <div class="autoscroll-field tag">
                        <label class="checkbox is-size-6" title="Automatically scroll when new chat messages come in.">
                            <input type="checkbox" v-model="autoscroll" :value="true">
                            Auto-scroll
                        </label>
                    </div>

                    <div :class="fontSizeClass">

                        <!-- No history? -->
                        <div v-if="chatHistory.length === 0 || (chatHistory.length === 1 && chatHistory[0].action === 'notification')">
                            <em v-if="isDM">
                                Starting a direct message chat with {{ channel }}. Type a message and say hello!
                            </em>
                            <em v-else>
                                There are no messages in this channel yet.
                            </em>
                        </div>

                        <!-- Load more history link in DMs -->
                        <div v-if="isDM && directMessageHistory[channel] != undefined && jwt.valid" class="mb-2">
                            <div v-if="directMessageHistory[channel].busy" class="notification is-info is-light">
                                <i class="fa fa-spinner fa-spin mr-1"></i>
                                Loading...
                            </div>
                            <div v-else-if="directMessageHistory[channel].remaining !== 0">
                                <a href="#" @click.prevent="loadDirectMessageHistory(channel)">
                                    Load more messages
                                    <span v-if="directMessageHistory[channel].remaining > 0">
                                        ({{directMessageHistory[channel].remaining}} remaining)
                                    </span>
                                </a>
                            </div>
                        </div>

                        <div v-for="(msg, i) in chatHistory" v-bind:key="i">

                            <MessageBox
                                :message="msg"
                                :action="msg.action"
                                :appearance="messageStyle"
                                :position="i"
                                :total-count="chatHistory.length"
                                :user="getUser(msg.username)"
                                :is-offline="isUserOffline(msg.username)"
                                :username="username"
                                :website-url="config.website"
                                :is-dnd="isUsernameDND(msg.username)"
                                :is-muted="isMutedUser(msg.username)"
                                :reactions="getReactions(msg)"
                                :report-enabled="isWebhookEnabled('report')"
                                :is-dm="isDM"
                                :is-op="isOp"
                                :my-video-active="webcam.active"
                                :is-video-not-allowed="isVideoNotAllowed(getUser(msg.username))"
                                :video-icon-class="webcamIconClass(getUser(msg.username))"
                                :is-invited-video="isInvited(msg.username)"
                                @open-profile="showProfileModal"
                                @open-video="openVideo"
                                @invite-video="inviteToWatch"
                                @send-dm="openDMs"
                                @mute-user="muteUser"
                                @takeback="takeback"
                                @remove="removeMessage"
                                @report="reportMessage"
                                @react="sendReact">
                            </MessageBox>

                        </div>

                    </div>

                    <!-- If this is a DM with a muted user, offer to unmute. -->
                    <div v-if="isDM && isMutedUser(channel)" class="has-text-danger">
                        <i class="fa fa-comment-slash"></i>
                        <strong>{{ channel }}</strong> is currently <strong>muted</strong> so you have not been seeing their
                        recent chat messages or DMs.
                        <a href="#" v-on:click.prevent="muteUser(channel)">Unmute them?</a>
                    </div>

                </div>
            </div>
        </div>

        <!-- Chat Footer Frame -->
        <div class="chat-footer">
            <div class="card">
                <div class="card-content p-2">

                    <div class="columns is-mobile">
                        <div class="column pr-1 is-narrow" v-if="canUploadFile">
                            <button type="button" class="button" @click="uploadFile()"
                                title="Upload a picture to share in chat">
                                <i class="fa fa-image"></i>
                            </button>
                        </div>
                        <div class="column pr-1" :class="{ 'pl-1': canUploadFile }">
                            <form @submit.prevent="sendMessage()">

                                <!-- At Mentions -->
                                <Mentionable :keys="['@']" :items="atMentionItems" offset="12" insert-space>

                                    <!-- My text box -->
                                    <input type="text" class="input" id="messageBox" v-model="message"
                                        :placeholder="status === 'hidden' ? 'Your status is hidden, be careful not to break the illusion' : 'Write a message'" @keydown="sendTypingNotification()" autocomplete="off"
                                        :disabled="!client.connected || (status === 'hidden' && !isDM)">

                                    <!-- At Mention templates-->
                                    <template #no-result>
                                        <div class="has-text-grey m-2">
                                            No result
                                        </div>
                                    </template>

                                    <template #item-@="{ item }">
                                        <div class="has-text-link m-2">
                                            @{{ item.value }}
                                            <span class="has-text-grey">
                                                {{ item.label }}
                                            </span>
                                        </div>
                                    </template>
                                </Mentionable>
                            </form>
                        </div>
                        <div class="column px-1 is-narrow dropdown is-right is-up" :class="{ 'is-active': showEmojiPicker }"
                            @click="showEmojiPicker = true">
                            <!-- Emoji picker for messages -->
                            <div class="dropdown-trigger">
                                <button type="button" class="button" aria-haspopup="true" aria-controls="input-emoji-picker"
                                    @click="hideEmojiPicker()">
                                    <span>
                                        <i class="fa-regular fa-smile"></i>
                                    </span>
                                </button>
                            </div>
                            <div class="dropdown-menu" id="input-emoji-picker" role="menu" style="z-index: 9000">
                                <!-- Note: z-index so the popup isn't covered by the "Auto-scroll"
                                    label on the chat history panel -->
                                <div class="dropdown-content p-0">
                                    <EmojiPicker :native="true" :display-recent="true" :disable-skin-tones="true"
                                        :theme="prefs.theme !== 'auto' ? prefs.theme : 'auto'" @select="onSelectEmoji">
                                    </EmojiPicker>
                                </div>
                            </div>
                        </div>
                        <div class="column pl-1 is-narrow">
                            <button type="button" class="button" :disabled="message.length === 0"
                                title="Click to send your message" @click="sendMessage()">
                                <i class="fa fa-paper-plane"></i>
                            </button>
                        </div>
                    </div>

                </div>
            </div>
        </div>

        <!-- Right Column: Who Is Online -->
        <div class="right-column">
            <div class="card grid-card">
                <header class="card-header has-background-success">
                    <div class="columns is-mobile card-header-title">
                        <div class="column">Who Is Online</div>
                        <div class="column is-narrow mobile-only">
                            <button type="button" class="button is-success px-2 py-1" @click="openChatPanel">
                                <i class="fa fa-arrow-left"></i>
                            </button>
                        </div>
                    </div>
                </header>
                <div class="card-content p-2">

                    <div class="columns is-mobile mb-0">
                        <div class="column is-one-quarter">
                            Status:
                        </div>
                        <div class="column">
                            <div class="select is-small is-fullwidth">
                                <select v-model="status">
                                    <optgroup v-for="group in StatusMessage.iterSelectOptGroups()"
                                        v-bind:key="group.category"
                                        :label="group.category">
                                        <option v-for="item in StatusMessage.iterSelectOptions(group.category)"
                                            v-bind:key="item.name"
                                            :value="item.name">
                                            {{ item.emoji }} {{ item.label }}
                                        </option>
                                    </optgroup>
                                </select>
                            </div>
                        </div>
                    </div>

                    <div class="columns is-mobile mb-0">
                        <div class="column is-one-quarter">
                            Sort:
                        </div>
                        <div class="column">
                            <div class="select is-small is-fullwidth">
                                <select v-model="whoSort">
                                    <optgroup label="By Name">
                                        <option value="a-z">Username (a-z)</option>
                                        <option value="z-a">Username (z-a)</option>
                                    </optgroup>
                                    <optgroup label="Webcam Status">
                                        <option value="broadcasting">Broadcasting</option>
                                        <option value="nsfw" v-show="config.permitNSFW">Red cameras</option>
                                        <option value="blue" v-show="config.permitNSFW">Blue cameras</option>
                                    </optgroup>
                                    <optgroup label="Other">
                                        <option value="login">Login Time</option>
                                        <option value="status">Status</option>
                                        <option value="emoji">Emoji/country flag</option>
                                        <option value="gender">Gender</option>
                                        <option value="op">☮ Operators</option>
                                    </optgroup>
                                </select>
                            </div>
                        </div>
                    </div>

                    <div class="tabs has-text-small">
                        <ul>
                            <li :class="{ 'is-active': whoTab === 'online' }">
                                <a class="is-size-7" @click.prevent="whoTab = 'online'">
                                    Online ({{ whoList.length }})
                                </a>
                            </li>
                            <li v-if="webcam.active" :class="{ 'is-active': whoTab === 'watching' }">
                                <a class="is-size-7" @click.prevent="whoTab = 'watching'">
                                    <i class="fa fa-eye mr-2"></i>
                                    Watching ({{ Object.keys(webcam.watching).length }})
                                </a>
                            </li>
                        </ul>
                    </div>

                    <!-- Who Is Online -->
                    <div v-if="whoTab === 'online'">

                        <!-- Show a loading spinner if we are connected but the Who List hasn't arrived -->
                        <div v-if="connected && sortedWhoList.length === 0" class="is-size-7">
                            <i class="fa fa-spinner fa-spin mr-1"></i> Waiting for Who List...
                        </div>

                        <div v-for="(u, i) in sortedWhoList" v-bind:key="i">
                            <WhoListRow
                                :user="u"
                                :username="username"
                                :website-url="config.website"
                                :is-dnd="isUsernameDND(u.username)"
                                :is-muted="isMutedUser(u.username)"
                                :is-blocked="isBlockedUser(u.username)"
                                :is-booted="isBooted(u.username)"
                                :is-op="isOp"
                                :is-video-not-allowed="isVideoNotAllowed(u)"
                                :video-icon-class="webcamIconClass(u)"
                                :vip-config="config.VIP"
                                :status-message="StatusMessage"
                                @send-dm="openDMs"
                                @mute-user="muteUser"
                                @open-video="openVideo"
                                @open-profile="showProfileModal">
                            </WhoListRow>
                        </div>
                    </div>

                    <!-- Watching My Webcam -->
                    <div v-if="whoTab === 'watching'">
                        <div v-for="(u, i) in sortedWatchingList" v-bind:key="i">
                            <WhoListRow
                                :is-watching-tab="true"
                                :user="u"
                                :username="username"
                                :website-url="config.website"
                                :is-dnd="isUsernameDND(username)"
                                :is-muted="isMutedUser(username)"
                                :is-blocked="isBlockedUser(u.username)"
                                :is-booted="isBooted(u.username)"
                                :is-op="isOp"
                                :is-video-not-allowed="isVideoNotAllowed(u)"
                                :video-icon-class="webcamIconClass(u)"
                                :vip-config="config.VIP"
                                :status-message="StatusMessage"
                                @send-dm="openDMs"
                                @mute-user="muteUser"
                                @open-video="openVideo"
                                @boot-user="bootUser"
                                @open-profile="showProfileModal">
                            </WhoListRow>
                        </div>
                    </div>

                </div>
            </div>
        </div>
    </div>

    <!--
        Dark Video Detector Canvas

        Notes:
        - Originally, we did document.createElement("canvas") to create a canvas
          on the fly, not placed on the web page. This usually worked most of the time,
          and most cameras could be screenshotted into it and read back out.
        - Sometimes, certain webcam models or certain conditions caused the canvas
          to read back as a solid black image, which would trigger a false positive
          for their dark video and cut their camera off.
        - From experimenting, it was found that by using a Canvas that existed on
          the page, and *making sure the Canvas was visible on page*, it was able to
          work in cases where the createElement() Canvas did not.
        - The page canvas *MUST BE VISIBLE* though: if it was set to display:none, or
          set to opacity:0, or put inside a 0x0 pixel container, or if its parent
          element had ANY of those properties set: the Canvas would only get a solid
          black screenshot still.

        So, we stick the canvas into a 1x1 pixel container and put it in the
        corner of the page.
    -->
    <div style="width: 1px; height: 1px; overflow: hidden; position: absolute; bottom: 0; right: 0">
        <canvas id="darkVideoCanvas"></canvas>
    </div>

    <!-- Theme CSS (light/dark) -->
    <link v-if="prefs.theme === 'light'"
        rel="stylesheet" type="text/css"
        href="/static/css/bulma-no-dark-mode.min.css?{{.config.cacheHash}}">
    <link v-else-if="prefs.theme === 'dark'"
        rel="stylesheet" type="text/css"
        href="/static/css/bulma-dark-theme.css?{{.config.cacheHash}}">
    <link v-else rel="stylesheet" type="text/css"
        href="/static/css/chat-prefers-dark.css?{{.config.cacheHash}}">
</template>

<style>
/* At-mention styles */
.mention-item {
    padding: 4px 10px;
    border-radius: 4px;
}

.mention-selected {
    background: rgb(192, 250, 153);
}

/* Forcibly truncating long texts */
.forcibly-truncate-wrapper {
    position: relative;
    overflow: hidden !important;
}
.forcibly-truncate-body {
    position: absolute;
    top: 0px;
    left: 0px;
    right: 0px;
}
.forcibly-single-line {
    white-space: nowrap;
}

/* Grey avatar for offline user on your DMs */
.offline-avatar {
    filter: grayscale();
}
</style>
