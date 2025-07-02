/*
WebRTC and Webcam related functionality for BareRTC.

Most methods here are called by App.vue with some funcs reused by sub-components, e.g.
for showing the webcam video buttons on the Who List and other places.
*/

import interact from 'interactjs';
import hark from 'hark';

import VideoFlag from './VideoFlag.js';
import LocalStorage from './LocalStorage.js';

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

// Webcam sizes: ideal is 640x480 as this is the most friendly for end users, e.g. if everyone
// broadcasted at 720p users on weaker hardware would run into problems sooner.
const WebcamWidth = 640,
    WebcamHeight = 480;

// For webcams that can not transmit at 640x480 (e.g. ultra widescreen), allow them to choose
// the nearest resolution but no more than 720p.
const WebcamMaxWidth = 1280,
    WebcamMaxHeight = 720;

const ReactNudgeNsfwMessageID = -451;

// An API surface layer of functions.
class WebRTCController {
    // The caller configures:
    // - nsfw (bool): the BareRTC PermitNSFW setting, which controls some status options.
    // - isAdmin (func): return a boolean if the current user is operator.
    // - currentStatus (func): return the name of the user's current status.
    constructor() {

    }

    // Vue mixin for the main App.vue
    getMixin() {
        return {
            data: this.getData(),
            watch: this.getWatches(),
            computed: this.getComputed(),
            methods: this.getMethods(),
        }
    }

    // Vue.js data mixin for App.vue to store all our webcam-related state.
    getData() {
        return {
            // The user's local webcam and settings.
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
                nonExplicit: false, // user prefers not to see explicit cameras
                vipOnly: false, // only show camera to fellow VIP users
                noWatermark: false, // user opts out of their own QR code watermark
                rememberExpresslyClosed: true,  // remember cams we expressly closed
                autoMuteWebcams: false, // auto-mute other cameras' audio channels

                // My watermark image for screen recording protection.
                // Set after login in setWatermark.
                watermark: null,

                // Who all is watching me? map of users.
                watching: {},

                // Scaling setting for the videos drawer, so the user can
                // embiggen the webcam sizes so a suitable size.
                videoScale: "",
                videoScaleOptions: [
                    ["", "Default size"],
                    ["x1", "50% larger videos"],
                    ["x2", "2x larger videos"],
                    ["x3", "3x larger videos"],
                    ["x4", "4x larger videos (not recommended)"],
                ],

                // Available cameras and microphones for the Settings modal.
                gettingDevices: false, // busy indicator for refreshing devices
                videoDevices: [],
                videoDeviceID: null,
                audioDevices: [],
                audioDeviceID: null,

                // Advanced: automatically share your webcam when the page loads.
                autoshare: false,

                // After we get a device selected, remember it (by name) so that we
                // might hopefully re-select it by default IF we are able to enumerate
                // devices before they go on camera the first time.
                preferredDeviceNames: {
                    video: null,
                    audio: null,
                },

                // Detect dark video streams.
                darkVideo: {
                    canvas: null,    // <canvas> element to screenshot video into
                    ctx: null,       // Canvas context2d
                    interval: null,  // interval loop
                    lastImage: null, // data: uri of last screenshot taken
                    lastAverage: [], // last average RGB color
                    lastAverageColor: "rgba(255, 0, 255, 1)",
                    tooDarkFrames: 0,  // frame counter for dark videos
                    tooDarkFramesLimit: 4, // frames in a row of too dark before cut

                    // Configuration thresholds: how dark is too dark? (0-255)
                    // NOTE: 0=disable the feature.
                    threshold: 10,
                },
            },

            // WebRTC sessions with other users.
            WebRTC: {
                // Streams per username.
                streams: {},
                muted: {}, // muted bool per username
                booted: {}, // booted bool per username
                invited: {}, // usernames we had invited
                poppedOut: {}, // popped-out video per username
                speaking: {}, // speaking boolean per username

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

                // Map of usernames whose cameras were expressly closed by us.
                // Example: we have mutualOpen enabled, someone opens our cam
                // so we open theirs back, and then we expressly 'X' out and
                // close their camera. If they close and re-open ours, we don't
                // want to auto-open their cam back because we had previously
                // closed out of it manually.
                //
                // Usernames are added when we 'X' out of their video,
                // and usernames are removed when we expressly re-open their
                // video (e.g., by clicking their Who List video button).
                expresslyClosed: {},  // bool per username

                // Rate limit pre-emptive boots: to avoid a user coming thru the entire Who List
                // and kicking everyone off their cam when they aren't even watching yet.
                preemptBootRateLimit: {
                    counter: 0,       // how many people have they booted so far
                    cooldownAt: null, // Date for cooldown once they reach the threshold

                    // Configuration settings:
                    maxFreeBoots: 10, // first 10 are free
                    cooldownTTL: 60, // then must wait this number of seconds before each boot
                }
            },

            /* For internal use only */
            // VideoFlag: VideoFlag,
            // LocalStorage: LocalStorage,
        };
    }

    // Watchers to be imported for App.vue
    getWatches() {
        return {
            "webcam.videoScale": function () {
                document.querySelectorAll(".video-feeds > .feed").forEach(node => {
                    node.style.width = null;
                    node.style.height = null;
                });
                LocalStorage.set('videoScale', this.webcam.videoScale);
            },

            // Webcam preferences that the user can edit while live.
            "webcam.nsfw": function () {
                this.webcam.wasServerNSFW = false;
                LocalStorage.set('videoExplicit', this.webcam.nsfw);
                if (this.webcam.active) {
                    this.sendMe();
                }

                // Respect users who have the NonExplicit cam setting checked, and close
                // their cameras when you toggle to red.
                this.unWatchNonExplicitVideo();
            },
            "webcam.mutual": function () {
                LocalStorage.set('videoMutual', this.webcam.mutual);
                if (this.webcam.active) {
                    this.sendMe();
                }
            },
            "webcam.mutualOpen": function () {
                LocalStorage.set('videoMutualOpen', this.webcam.mutualOpen);
                if (this.webcam.active) {
                    this.sendMe();
                }
            },
            "webcam.nonExplicit": function () {
                LocalStorage.set('videoNonExplicit', this.webcam.nonExplicit);

                // Turn off NSFW if this is on.
                if (this.webcam.nonExplicit && this.webcam.nsfw) {
                    // Note: this toggle will also sendMe().
                    this.webcam.nsfw = false;
                } else if (this.webcam.active) {
                    this.sendMe();
                }
            },
            "webcam.vipOnly": function () {
                LocalStorage.set('videoVipOnly', this.webcam.vipOnly);
                if (this.webcam.active) {
                    this.sendMe();
                }

                // If we have toggled this while already connected to people:
                // Hang up on any that have a mutual viewership requirement, if
                // they can not see our VIP-only camera.
                for (let username of Object.keys(this.WebRTC.pc)) {
                    if (this.whoMap[username] != undefined && this.isVideoNotAllowed(this.whoMap[username])) {
                        this.closeVideo(username);
                    }
                }
            },
            "webcam.noWatermark": function () {
                LocalStorage.set('videoNoWatermark', this.webcam.noWatermark);
                if (this.webcam.active) {
                    this.sendMe();
                }
            },
            "webcam.rememberExpresslyClosed": function () {
                LocalStorage.set('rememberExpresslyClosed', this.webcam.rememberExpresslyClosed);
            },
            "webcam.autoMuteWebcams": function () {
                LocalStorage.set('autoMuteWebcams', this.webcam.autoMuteWebcams);
            },
            "webcam.autoshare": function () {
                LocalStorage.set('videoAutoShare', this.webcam.autoshare);
            },
        };
    }

    // Vue.js mixin for the App.vue
    getComputed() {
        return {
            myVideoFlag() {
                // Compute the current user's video status flags.
                let status = 0;
                if (!this.webcam.active) return 0; // unset all flags if not active now
                if (this.webcam.active) status |= this.VideoFlag.Active;
                if (this.webcam.muted) status |= this.VideoFlag.Muted;
                if (this.webcam.nsfw && this.config.permitNSFW) status |= this.VideoFlag.NSFW;
                if (this.webcam.mutual) status |= this.VideoFlag.MutualRequired;
                if (this.webcam.mutualOpen) status |= this.VideoFlag.MutualOpen;
                if (this.webcam.nonExplicit) status |= this.VideoFlag.NonExplicit;
                if (this.webcam.vipOnly && this.isVIP) status |= this.VideoFlag.VipOnly;
                if (this.webcam.noWatermark) status |= this.VideoFlag.NoWatermark;
                return status;
            },
            anyVideosOpen() {
                // Return if any videos are open.
                return this.webcam.active || this.numVideosOpen > 0;
            },
            numVideosOpen() {
                // Return the count of other peoples videos we have open.
                return Object.keys(this.WebRTC.streams).length;
            },
            sortedWatchingList() {
                let result = [];
                for (let username of Object.keys(this.webcam.watching)) {
                    let user = this.getUser(username);
                    result.push(user);
                }
    
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
        };
    }

    // Vue.js mixin for the App.vue
    getMethods() {
        return {
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

                this.DebugChannel(`[WebRTC] Starting WebRTC with: ${username} (I am the: ${isOfferer ? 'offerer' : 'answerer'})`);

                // Keep a pointer to the current channel being established (for candidate/SDP).
                this.WebRTC.pc[username].connecting = pc;

                // 'onicecandidate' notifies us whenever an ICE agent needs to deliver a
                // message to the other peer through the signaling server.
                pc.onicecandidate = event => {
                    if (event.candidate) {
                        this.client.send({
                            action: "candidate",
                            username: username,
                            candidate: JSON.stringify(event.candidate),
                        });
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
                    // If we were not expecting to receive this video (e.g. somebody is requesting our
                    // cam and sending their video on the offer, but we don't want to auto-open their
                    // video, so don't use it)
                    if (!isOfferer && !this.webcam.mutualOpen) {
                        this.DebugChannel(`[WebRTC] The offerer ${username} gave us a video, but we don't auto-open their video.`);
                        return;
                    }

                    const stream = event.streams[0];

                    // Had we expressly closed this user's cam before? e.g.: if we have auto-open their
                    // video enabled, and we 'X' out, and they reopen ours - we may be receiving their
                    // video right now. If we had expressly closed it, do not accept their video
                    // and hang up the connection.
                    if (this.WebRTC.expresslyClosed[username] && this.webcam.rememberExpresslyClosed) {
                        if (!isOfferer) {
                            return;
                        }
                    }

                    // A booted admin?
                    if (this.isBootedAdmin(username)) {
                        return;
                    }

                    // We've received a video! If we had an "open camera spinner timeout",
                    // clear it before it expires.
                    if (this.WebRTC.openTimeouts[username] != undefined) {
                        clearTimeout(this.WebRTC.openTimeouts[username]);
                        delete (this.WebRTC.openTimeouts[username]);
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

                        // Are we muting their videos by default?
                        if (this.webcam.autoMuteWebcams) {
                            this.WebRTC.muted[username] = true;
                            $ref.muted = true;
                        }

                        // Set up the speech detector.
                        this.initSpeakingEvents(username, $ref);
                    });

                    // Inform them they are being watched.
                    this.sendWatch(username, true);

                    // Set a mute video handler to detect freezes.
                    stream.getVideoTracks().forEach(videoTrack => {
                        let freezeDetected = () => {
                            this.DebugChannel("[WebRTC] A video freeze was detected from:", username);
                            // Wait some seconds to see if the stream has recovered on its own
                            setTimeout(() => {
                                // Flag it as likely frozen.
                                if (videoTrack.muted) {
                                    this.WebRTC.frozenStreamDetected[username] = true;
                                }
                            }, 7500); // 7.5s
                        };

                        videoTrack.onmute = freezeDetected;

                        // Double check for frozen streams on an interval.
                        if (this.WebRTC.frozenStreamInterval[username]) {
                            // Clear the existing interval (e.g. audio+video track sets up the
                            // interval twice right now, don't overwrite and lose the interval)
                            clearInterval(this.WebRTC.frozenStreamInterval[username]);
                        }
                        this.WebRTC.frozenStreamInterval[username] = setInterval(() => {
                            if (videoTrack.muted) freezeDetected();
                        }, 3000);
                    });
                };

                // ANSWERER: add our video to the connection so that the offerer (the one who
                // clicked on our video icon to watch us) can receive it.
                if (!isOfferer && this.webcam.active) {
                    this.DebugChannel(`[WebRTC] Answerer: attaching my video to the connection with: ${username}`);
                    let stream = this.webcam.stream;
                    stream.getTracks().forEach(track => {
                        pc.addTrack(track, stream)
                    });
                }

                // OFFERER: If we were already broadcasting our own video, and the answerer
                // has the "auto-open your video" setting enabled, attach our video to the initial
                // offer right now.
                if (isOfferer) {
                    let shouldOfferVideo = (
                        (this.whoMap[username].video & this.VideoFlag.MutualOpen) // They auto-open us
                        && this.webcam.active             // Our camera is active (to add it)
                        && !this.isBooted(username)       // We had not booted them off ours before
                        && !this.isMutedUser(username)    // We had not muted them before

                        // If our webcam is NSFW and the viewer prefers not to see explicit,
                        // do not send our camera on this offer.
                        && (!this.webcam.nsfw || !(this.whoMap[username].video & this.VideoFlag.NonExplicit))
                    );

                    // Attach our video on the outgoing offer, so that on the answerer's side our
                    // local video pops up on their screen.
                    if (shouldOfferVideo) {
                        this.DebugChannel(`[WebRTC] Offerer: I am attaching my video to the connection with: ${username}`)
                        let stream = this.webcam.stream;
                        stream.getTracks().forEach(track => {
                            pc.addTrack(track, stream)
                        });
                    } else {
                        // We aren't offering video, but still want to receive audio/video. Add a receive-only
                        // transceiver to this offer. NOTE: in the legacy WebRTC API we could put offerToReceiveVideo
                        // and offerToReceiveAudio in the createOffer() call later, but the modern WebRTC has removed
                        // those options and Safari only supports the modern way. Adding a receive-only transceiver
                        // here is the modern way to do it that Safari will be happy with.
                        this.DebugChannel(`[WebRTC] Offer: I am attaching a receive-only video/audio transceiver to the connection with: ${username}`);
                        pc.addTransceiver('video', { direction: 'recvonly' });
                        pc.addTransceiver('audio', { direction: 'recvonly' });
                    }
                }

                // If we are the offerer, begin the connection.
                if (isOfferer) {
                    this.DebugChannel(`[WebRTC] Offerer: create the offer and send it to ${username}`);
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
                        this.DebugChannel(`[WebRTC] Local description created; sending SDP message to ${username}:<br><br>${JSON.stringify(pc.localDescription)}`);
                        this.client.send({
                            action: "sdp",
                            username: username,
                            description: JSON.stringify(pc.localDescription),
                        });
                    }).catch(e => {
                        console.error(`Error sending WebRTC negotiation message (SDP): ${e}`);
                    });
                };
            },

            // BareRTC Protocol Handlers relating to WebRTC
            sendOpen(username) {
                // Sending an open message to connect to someone's camera
                this.DebugChannel(`[WebRTC] Sending "open" message to ask to connect to: ${username}`);
                this.client.send({
                    action: "open",
                    username: username,
                });
            },
            onOpen(msg) {
                // Response for the opener to begin WebRTC connection.
                this.DebugChannel(`[WebRTC] Received "open" echo from chat server to connect to: ${msg.username}`);
                this.startWebRTC(msg.username, true);
            },
            onRing(msg) {
                // Request from a viewer to see our broadcast.
                this.DebugChannel(`[WebRTC] Received "ring" message from chat server to share my video with: ${msg.username}`);
                this.startWebRTC(msg.username, false);
            },
            sendBoot(username) {
                this.client.send({
                    action: "boot",
                    username: username,
                });
            },
            sendUnboot(username) {
                this.client.send({
                    action: "unboot",
                    username: username,
                });
            },
            sendInviteVideo(username) {
                this.WebRTC.invited[username] = true;
                this.client.send({
                    action: "video-invite",
                    usernames: [username],
                });
            },
            sendInviteVideoBulk() {
                let usernames = Object.keys(this.WebRTC.invited);
                if (usernames.length === 0) return;

                // Re-send the invite list on reconnect to server.
                this.client.send({
                    action: "video-invite",
                    usernames: usernames,
                });
            },
            isInvited(username) {
                return this.WebRTC.invited[username] != undefined;
            },
            onCandidate(msg) {
                // Handle inbound WebRTC signaling messages proxied by the websocket.
                if (this.WebRTC.pc[msg.username] == undefined || !this.WebRTC.pc[msg.username].connecting) {
                    return;
                }
                let pc = this.WebRTC.pc[msg.username].connecting;

                // XX: WebRTC candidate/SDP messages JSON stringify their inner payload so that the
                // Go back-end server won't re-order their json keys (Safari on Mac OS is very sensitive
                // to the keys being re-ordered during the handshake, in ways that NO OTHER BROWSER cares
                // about at all). Re-parse the JSON stringified object here.
                let candidate = JSON.parse(msg.candidate);

                this.DebugChannel(`[WebRTC] ICE candidate from ${msg.username}:<br><br>${msg.candidate}`);

                // Add the new ICE candidate.
                pc.addIceCandidate(candidate).catch(e => {
                    console.error(`addIceCandidate: ${e}`);
                });
            },
            onSDP(msg) {
                // WebRTC Session Description Protocol messages proxied by the websocket.
                if (this.WebRTC.pc[msg.username] == undefined || !this.WebRTC.pc[msg.username].connecting) {
                    return;
                }
                let pc = this.WebRTC.pc[msg.username].connecting;

                this.DebugChannel(`[WebRTC] Received SDP message from ${msg.username}:<br><br>${msg.description}`);

                // XX: WebRTC candidate/SDP messages JSON stringify their inner payload so that the
                // Go back-end server won't re-order their json keys (Safari on Mac OS is very sensitive
                // to the keys being re-ordered during the handshake, in ways that NO OTHER BROWSER cares
                // about at all). Re-parse the JSON stringified object here.
                let message = JSON.parse(msg.description);

                // Add the new ICE candidate.
                pc.setRemoteDescription(new RTCSessionDescription(message)).then(() => {
                    this.DebugChannel(`[WebRTC] <strong>setRemoteDescription</strong> called back OK!<br>Our pc.remoteDescription.type is: ${pc.remoteDescription.type}`);
                    // When receiving an offer let's answer it.
                    if (pc.remoteDescription.type === 'offer') {
                        this.DebugChannel(`[WebRTC] Answerer: create SDP answer message for ${msg.username}`);
                        pc.createAnswer().then(this.localDescCreated(pc, msg.username)).catch(this.ChatClient);
                    } else {
                        this.DebugChannel(`[WebRTC] pc.remoteDescription.type was not 'offer', we do not need to create an SDP Answer message.`);
                    }
                }).catch(this.DebugChannel);
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
                this.DebugChannel(`<strong>BareRTC Unwatch</strong> received from: ${msg.username}<br><br>` +
                    `Current watching: ${JSON.stringify(Object.keys(this.webcam.watching))}<br>` +
                    `Was in watch list: ${this.webcam.watching[msg.username] == undefined ? 'NO!' : 'YES'}<br>` +
                    `My camera is: ${this.webcam.active ? 'active': 'not active'}`,
                );

                // If the user leaves our watching list, play the sound.
                if (this.webcam.watching[msg.username] != undefined) {
                    this.playSound("Unwatch");
                }

                // The user has closed our video feed.
                delete (this.webcam.watching[msg.username]);
                this.cleanupPeerConnections();
            },
            sendWatch(username, watching) {
                // Send the watch or unwatch message to backend.
                this.client.send({
                    action: watching ? "watch" : "unwatch",
                    username: username,
                });
            },
            isWatching(username) {
                // Is the current user watching this target user?
                if (this.WebRTC.pc[username] != undefined && this.WebRTC.streams[username] != undefined) {
                    return true;
                }
                return false;
            },
            isWatchingMe(username) {
                // Return whether the user is watching your camera
                return this.webcam.watching[username] === true;
            },

            isUsernameCamNSFW(username) {
                // returns true if the username is broadcasting and NSFW, false otherwise.
                // (used for the color coding of their nickname on their video box - so assumed they are broadcasting)
                if (this.whoMap[username] != undefined && this.whoMap[username].video & this.VideoFlag.NSFW) {
                    return true;
                }
                return false;
            },

            isUsernameCamNoWatermark(username) {
                // returns true if the user is broadcasting and has the NoWatermark flag set.
                if (this.whoMap[username] != undefined && this.whoMap[username].video & this.VideoFlag.NoWatermark) {
                    return true;
                }
                return false;
            },

            /**********************************
             * Front-end methods and handlers *
             **********************************/

            // Start broadcasting my webcam.
            // - force=true to skip the NSFW modal prompt (this param is passed by the button in that modal)
            // - changeCamera=true to re-negotiate WebRTC connections with a new camera device (invoked by the Settings modal)
            startVideo({ force = false, changeCamera = false }) {
                if (this.webcam.busy) return;

                // Is a moderation rule in place?
                if (this.jwt.rules.IsNoBroadcastRule) {
                    return this.modalAlert({
                        title: "Broadcasting video is not allowed for you",
                        message: this.config.strings.ModRuleErrorNoBroadcast || "A chat room moderation rule is currently in place which restricts your ability to broadcast your webcam.\n\nPlease contact a chat operator for more information.",
                    });
                }

                // Before they go on cam the first time, ATTEMPT to get their device names.
                // - If they had never granted permission, we won't get the names of
                //   the devices and no big deal.
                // - If they had given permission before, we can present a nicer experience
                //   for them and enumerate their devices before they go on originally.
                if (!changeCamera && !force) {
                    // Initial broadcast: did they select device IDs?
                    this.getDevices();
                }

                // Show the broadcast settings modal the first time.
                if (!force) {
                    this.nsfwModalCast.visible = true;
                    return;
                }

                let mediaParams = {
                    audio: true,
                    video: {
                        width: { ideal: WebcamWidth, max: WebcamMaxWidth },
                        height: { ideal: WebcamHeight, max: WebcamMaxHeight },
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
                    LocalStorage.set('videoMutual', this.webcam.mutual);
                    LocalStorage.set('videoMutualOpen', this.webcam.mutualOpen);
                    LocalStorage.set('videoAutoMute', this.webcam.autoMute);
                    LocalStorage.set('videoVipOnly', this.webcam.vipOnly);
                    LocalStorage.set('videoExplicit', this.webcam.nsfw);

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

                    // If we started as Explicit, hang up on any NonExplicit camera we had open.
                    this.unWatchNonExplicitVideo();

                    // Tell backend the camera is ready.
                    this.sendMe();

                    // Record the selected device IDs.
                    this.webcam.videoDeviceID = stream.getVideoTracks()[0].getSettings().deviceId;
                    this.webcam.audioDeviceID = stream.getAudioTracks()[0].getSettings().deviceId;

                    // For debugging: log to the debug channel what the chosen resolution is for the user.
                    let videoSettings = stream.getVideoTracks()[0].getSettings();
                    this.DebugChannel(`navigator.getUserMedia(): chosen video resolution is ${videoSettings.width}x${videoSettings.height}`);

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

                    // Begin dark video detection as soon as the video is ready to capture frames.
                    this.webcam.elem.addEventListener("canplaythrough", this.initDarkVideoDetection);

                    // Begin monitoring for speaking events.
                    this.initSpeakingEvents(this.username, this.webcam.elem);
                }).catch(err => {
                    this.ChatClient(`Webcam error: ${err}<br><br>Please see the <a href="/about#troubleshooting">troubleshooting guide</a> for help.`);
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
                        }

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
                LocalStorage.set('preferredDeviceNames', this.webcam.preferredDeviceNames);
            },

            // The 'Explicit' button at the top of the page: toggle the user's NSFW setting but
            // with some smarts in case the user was just marked NSFW by an admin.
            topNavExplicitButtonClicked() {
                if (this.webcam.wasServerNSFW) {
                    this.webcam.wasServerNSFW = false;
                    this.ChatClient(
                        `Notice: your webcam was already marked as "Explicit" recently by the chat server.<br><br>` +
                        `If you were recently notified that a chat moderator has marked your camera as 'explicit' (red) for you, then ` +
                        `you do not need to do anything: your camera is marked Explicit already. Please leave it as Explicit if you are ` +
                        `being sexual on camera.<br><br>` +
                        `If you really mean to <strong>remove</strong> the Explicit label (and turn your camera 'blue'), then click on the ` +
                        `Explicit button at the top of the page one more time to confirm.`,
                    );
                    return;
                }
                this.webcam.nsfw = !this.webcam.nsfw;
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
            setSkipNSFWModal() {
                // Set the "don't show again" on NSFW modal.
                LocalStorage.set("skip-nsfw-modal", true);
            },
            openVideo(user, force) {
                // Close the NSFW warning modal if it was open.
                this.nsfwModalView.visible = false;

                if (user.username === this.username) {
                    this.ChatClient("You can already see your own webcam.");
                    return;
                }

                // Operator safety when hidden.
                if (this.status === 'hidden') {
                    this.ChatClient("Your chat status is currently set to 'hidden' and you would break the illusion by opening this camera.");
                    return;
                }

                // A chat moderation rule?
                if (this.jwt.rules.IsNoVideoRule) {
                    return this.modalAlert({
                        title: "Videos are not available to you",
                        message: this.config.strings.ModRuleErrorNoVideo || "A chat room moderation rule is currently in place which restricts your ability to watch webcams.\n\n" +
                            "Please contact a chat operator for more information.",
                    });
                }

                // If we have muted the target, we shouldn't view their video.
                if (this.isMutedUser(user.username) && !this.isOp) {
                    this.ChatClient(`You have muted <strong>${user.username}</strong> and so should not see their camera.`);
                    return;
                }

                // If we have booted the target off our cam, we shouldn't view their video.
                if (this.isBooted(user.username) && !this.isOp) {
                    this.ChatClient(`You had kicked <strong>${user.username}</strong> off your camera and so it wouldn't be right to still watch their camera.`);
                    return;
                }

                // Is the target user NSFW? Go thru the modal.
                let dontShowAgain = LocalStorage.get("skip-nsfw-modal") === true;
                if ((user.video & this.VideoFlag.NSFW) && !dontShowAgain && !force) {
                    this.nsfwModalView.user = user;
                    this.nsfwModalView.visible = true;
                    return;
                }

                // If the local user had expressly closed this user's camera before, forget
                // this action because the user now is expressly OPENING this camera on purpose.
                delete (this.WebRTC.expresslyClosed[user.username]);

                // Debounce so we don't spam too much for the same user.
                if (this.WebRTC.debounceOpens[user.username]) return;
                this.WebRTC.debounceOpens[user.username] = true;
                setTimeout(() => {
                    delete (this.WebRTC.debounceOpens[user.username]);
                }, 5000);

                // Camera is already open? Then disconnect the connection.
                if (this.WebRTC.pc[user.username] != undefined && this.WebRTC.pc[user.username].offerer != undefined) {
                    this.DebugChannel(`OpenVideo(${user.username}): already had a connection open, closing it first.`);
                    this.closeVideo(user.username, "offerer");
                }

                // If this user is NonExplicit and your camera is red...
                let theirVideo = this.whoMap[user.username].video;
                if (this.webcam.active && this.webcam.nsfw && (theirVideo & VideoFlag.Active) && (theirVideo & VideoFlag.NonExplicit)) {
                    this.ChatClient(
                        `<strong>${user.username}</strong> prefers not to have Explicit cammers watch their video, and your camera is currently marked as Explicit.`
                    );
                    return;
                }

                // Conversely: if we are NonExplicit we should not open NSFW videos.
                if (this.webcam.active && this.webcam.nonExplicit && (theirVideo & VideoFlag.Active) && (theirVideo & VideoFlag.NSFW)) {
                    this.ChatClient(
                        `You have said you do not want to see Explicit videos and <strong>${user.username}</strong> has an Explicit camera.`
                    );
                    return;
                }

                // If this user requests mutual viewership...
                if (this.isVideoNotAllowed(user) && !this.isOp) {
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
                    delete (this.WebRTC.openTimeouts[user.username]);
                }
                this.WebRTC.openTimeouts[user.username] = setTimeout(() => {
                    this.ChatClient(
                        `There was an error opening <strong>${user.username}</strong>'s camera.`,
                    );
                    delete (this.WebRTC.openTimeouts[user.username]);
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
                    clearInterval(this.WebRTC.frozenStreamInterval[username]);
                    delete (this.WebRTC.frozenStreamInterval[username]);
                }

                if (name === "offerer") {
                    // We are closing another user's video stream.
                    delete (this.WebRTC.streams[username]);
                    delete (this.WebRTC.muted[username]);
                    delete (this.WebRTC.poppedOut[username]);

                    // Should we close the WebRTC PeerConnection? If they were watching our video back, closing
                    // the connection MAY cause our video to freeze on their side: if we have the "auto-open my viewer's
                    // camera" option set, and our viewer sent their video on their open request, and they still have
                    // our camera open, do not close the connection so we don't freeze their side of the video.
                    if (this.WebRTC.pc[username] != undefined && this.WebRTC.pc[username].offerer != undefined) {
                        if (this.webcam.mutualOpen && this.isWatchingMe(username)) {
                            console.log(`OFFERER(${username}): Close video locally only: do not hang up the connection.`);
                        } else {
                            this.WebRTC.pc[username].offerer.close();
                            delete (this.WebRTC.pc[username]);
                        }
                    }

                    // Inform backend we have closed it.
                    this.sendWatch(username, false);
                    this.cleanupPeerConnections();
                    return;
                } else if (name === "answerer") {
                    // Should we close the WebRTC PeerConnection? If they were watching our video back, closing
                    // the connection MAY cause our video to freeze on their side: if we have the "auto-open my viewer's
                    // camera" option set, and our viewer sent their video on their open request, and they still have
                    // our camera open, do not close the connection so we don't freeze their side of the video.
                    if (this.WebRTC.pc[username] != undefined && this.WebRTC.pc[username].answerer != undefined) {
                        if (this.webcam.mutualOpen && this.isWatchingMe(username)) {
                            console.log(`ANSWERER(${username}): Close video locally only: do not hang up the connection.`);
                        } else {
                            this.WebRTC.pc[username].answerer.close();
                            delete (this.WebRTC.pc[username]);
                        }
                    }
                    this.cleanupPeerConnections();
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
                    clearInterval(this.WebRTC.frozenStreamInterval[username]);
                    delete (this.WebRTC.frozenStreamInterval[username]);
                }

                // Inform backend we have closed it.
                this.sendWatch(username, false);
                this.cleanupPeerConnections();
            },
            expresslyCloseVideo(username, name) {
                // Like closeVideo but communicates the user's intent to expressly
                // close the video for a user. e.g. they clicked the 'X' icon to
                // close it out. As opposed to a user went off camera and were closed
                // out automatically.
                this.WebRTC.expresslyClosed[username] = true;
                return this.closeVideo(username, name);
            },
            closeOpenVideos() {
                // Close all videos open of other users.
                for (let username of Object.keys(this.WebRTC.streams)) {
                    this.closeVideo(username, "offerer");
                }
            },
            cleanupPeerConnections() {
                // Helper function to check and clean up WebRTC PeerConnections.
                //
                // This is fired on Unwatch and CloseVideo events, to double check
                // which videos the local user has open + who online is watching our
                // video, to close out any lingering WebRTC connections.
                for (let username of Object.keys(this.WebRTC.pc)) {
                    let pc = this.WebRTC.pc[username];

                    // Is their video on our screen?
                    if (this.WebRTC.streams[username] != undefined) {
                        continue;
                    }

                    // Are they watching us?
                    if (this.isWatchingMe(username)) {
                        continue;
                    }

                    // Are they an admin that we booted?
                    if (this.isBootedAdmin(username)) {
                        continue;
                    }

                    // The WebRTC connections should be closed out.
                    if (pc.answerer != undefined) {
                        console.log("Clean up WebRTC answerer connection with " + username);
                        pc.answerer.close();
                        delete (this.WebRTC.pc[username]);
                    }
                    if (pc.offerer != undefined) {
                        console.log("Clean up WebRTC offerer connection with " + username);
                        pc.offerer.close();
                        delete (this.WebRTC.pc[username]);
                    }
                }
            },
            muteAllVideos() {
                // Mute the mic of all open videos.
                let count = 0;
                for (let username of Object.keys(this.WebRTC.streams)) {
                    if (this.WebRTC.muted[username]) continue;

                    // Find the <video> tag to mute it.
                    this.WebRTC.muted[username] = true;
                    let $ref = document.getElementById(`videofeed-${username}`);
                    if ($ref) {
                        $ref.muted = this.WebRTC.muted[username];
                    }

                    count++;
                }

                if (count > 0) {
                    this.ChatClient(`You have muted the audio on ${count} video${count === 1 ? '' : 's'}.`);
                }
            },
            unMutualVideo() {
                // If we had our camera on to watch a video of someone who wants mutual cameras,
                // and then we turn ours off: we should unfollow the ones with mutual video.
                if (this.webcam.active) return;
                for (let row of this.whoList) {
                    let username = row.username;

                    // If this user expressly invited us to watch, skip.
                    if ((row.video & this.VideoFlag.Active) && (row.video & this.VideoFlag.Invited)) {
                        continue;
                    }

                    if ((row.video & this.VideoFlag.MutualRequired) && this.WebRTC.pc[username] != undefined) {
                        this.closeVideo(username);
                    }
                }
            },
            unWatchNonExplicitVideo() {
                if (!this.webcam.active) return;

                // If we are watching cameras with the NonExplicit setting, and our camera has become
                // explicit, excuse ourselves from their watch list.
                if (this.webcam.nsfw) {
                    for (let username of Object.keys(this.WebRTC.streams)) {
                        let user = this.whoMap[username];

                        // Our video is Explicit and theirs is NonExplicit.
                        if ((user.video & VideoFlag.Active) && (user.video & VideoFlag.NonExplicit)) {
                            this.closeVideo(username);
                        }
                    }
                }

                // Conversely: if we are NonExplicit we do not watch Explicit videos.
                if (this.webcam.nonExplicit) {
                    for (let username of Object.keys(this.WebRTC.streams)) {
                        let user = this.whoMap[username];

                        if ((user.video & VideoFlag.Active) && (user.video & VideoFlag.NSFW)) {
                            this.closeVideo(username);
                        }
                    }
                }
            },

            // Invite a user to watch your camera even if they normally couldn't see.
            inviteToWatch(username) {
                this.modalConfirm({
                    title: "Invite to watch my webcam",
                    icon: "fa fa-video",
                    message: `Do you want to invite @${username} to watch your camera?\n\n` +
                        `This will give them permission to see your camera even if, normally, they would not be able to. ` +
                        `For example, if you have the option enabled to "require my viewer to be on webcam too" and @${username} is not on camera, ` +
                        `by inviting them to watch they will be allowed to see your camera anyway.\n\n` +
                        `This permission will be granted for the remainder of your chat session, but you can boot them ` +
                        `off your camera if you change your mind later.`,
                    buttons: ['Invite to watch', 'Cancel'],
                }).then(() => {
                    this.sendInviteVideo(username);

                    this.ChatClient(
                        `You have granted <strong>@${username}</strong> permission to watch your webcam. This will be in effect for the remainder of your chat session. ` +
                            "Note: if you change your mind later, you can boot them from your camera or block them from watching by using the button " +
                            "in their profile card.",
                    );
                });
            },

            isUsernameOnCamera(username) {
                return this.whoMap[username]?.video & VideoFlag.Active;
            },

            // Video button classes for a user.
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
        
                if (this.isVideoNotAllowed(user)) return 'fa-video-slash';
                return 'fa-video';
            },
            webcamButtonClass(username) {
                // This styles the convenient video button that appears in the header bar
                // of DM threads if your chat partner is on camera.
                let video = this.whoMap[username].video;
    
                if (!(video & VideoFlag.Active)) {
                    return "";
                }
    
                if (video & VideoFlag.NSFW) {
                    return "is-danger";
                }
    
                return "is-link";
            },

            isVideoNotAllowed(user) {
                // Returns whether the video button to open a user's cam will be not allowed (crossed out)
    
                // If the user is under the NoVideo rule, always cross it out.
                if (this.jwt.rules.IsNoVideoRule) return true;

                // If this user expressly invited us to watch.
                if ((user.video & this.VideoFlag.Active) && (user.video & this.VideoFlag.Invited)) {
                    return false;
                }

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
                if (this.isMutedUser(user.username) || this.isBooted(user.username)) {
                    return true;
                }
    
                // This person is NonExplicit and our camera is Explicit.
                if (this.webcam.active && this.webcam.nsfw && (user.video & VideoFlag.Active) && (user.video & VideoFlag.NonExplicit)) {
                    return true;
                }
    
                // Conversely: if we are NonExplicit we should not be able to watch Explicit videos.
                if (this.webcam.active && this.webcam.nonExplicit && (user.video & VideoFlag.Active) && (user.video & VideoFlag.NSFW)) {
                    return true;
                }
    
                return false;
            },
    
            // Functions to help users 'nudge' each other into marking their cams as Explicit.
            sendNudgeNsfw(username) {
                // Send a nudge to the username. This is triggered by their profile modal: if the user is
                // on blue camera, others on chat can anonymously nudge them into marking their camera as red.
    
                // Nudges are a special kind of emoji reaction (so we could add this feature in frontend only without
                // a server side deployment to support it).
                this.client.send({
                    action: 'react',
                    msgID: ReactNudgeNsfwMessageID,
                    message: username,
                });
            },
            onNudgeNsfw(msg) {
                // Handler for a nudge NSFW react message.
                if (msg.message !== this.username) {
                    return; // Not for us
                }
    
                // Sanity check that we are on blue camera.
                if (!this.webcam.active || this.webcam.nsfw) {
                    return;
                }
    
                // Only show this if we have at least 2 watchers.
                let watchers = Object.keys(this.webcam.watching).length;
                if (watchers < 2) {
                    return;
                }
    
                // Show a nice message on chat.
                this.ChatServer(
                    `<strong>Your webcam is <span class="has-text-danger">Hot!</span></strong> <i class="fa fa-fire has-text-danger"></i><br><br>` +
                        `Somebody who is watching your camera thinks that your webcam should be tagged as <span class="has-text-danger"><i class="fa fa-fire mx-1"></i> Explicit</span>.<br><br>` +
                        `In case you forgot to do so, please click on the '<i class="fa fa-fire has-text-danger"></i> Explicit' button at the top of the page to turn your camera 'red.' Thank you! <i class="fa fa-heart has-text-danger"></i>`,
                );
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
                // Un-boot?
                if (this.isBooted(username)) {
                    this.modalConfirm({
                        title: "Unboot user",
                        icon: "fa fa-user-xmark",
                        message: `Allow ${username} to watch your webcam again?`
                    }).then(() => {
                        this.sendUnboot(username);
                        delete (this.WebRTC.booted[username]);
                    })
                    return;
                }
    
                // Check if they are currently rate limited from pre-emptive boots of folks
                // who are not even watching their camera yet.
                if (this.rateLimitPreemptiveBoots(username, false)) return;
    
                // Boot them off our webcam.
                this.modalConfirm({
                    title: "Boot user",
                    icon: "fa fa-user-xmark",
                    message: `Kick ${username} off your camera? This will also prevent them ` +
                        `from seeing that your camera is active for the remainder of your ` +
                        `chat session.`
                }).then(() => {
                    // Ping their rate limiter.
                    if (this.rateLimitPreemptiveBoots(username, true)) return;
    
                    this.doBootUser(username);
    
                    this.ChatClient(
                        `You have booted ${username} off your camera. They will no longer be able ` +
                        `to connect to your camera, or even see that your camera is active at all -- ` +
                        `to them it appears as though you had turned yours off.<br><br>This will be ` +
                        `in place for the remainder of your time in chat, until you log off. ` +
                        `<strong>Note:</strong> If you wish to undo this, you can allow them to watch again ` +
                        `by opening their profile card.`
                    );
                });
            },
            doBootUser(username) {
                // Inner function to actually boot a user, bypassing any confirm or rate limit modals.
                this.sendBoot(username);
                this.WebRTC.booted[username] = true;
    
                // Close the WebRTC peer connections.
                if (this.WebRTC.pc[username] != undefined) {
                    this.closeVideo(username);
                }
    
                // Remove them from our list.
                delete (this.webcam.watching[username]);
            },
            isBooted(username) {
                return this.WebRTC.booted[username] === true;
            },
            isBootedAdmin(username) {
                return (this.WebRTC.booted[username] === true || this.muted[username] === true) &&
                    this.whoMap[username] != undefined &&
                    this.whoMap[username].op;
            },
            rateLimitPreemptiveBoots(username, ping=false) {
                // Rate limit abusive pre-emptive booting behavior: if the target is not even currently watching
                // your camera, limit how many and how frequently you can boot them off.
                //
                // Returns true if limited, false otherwise.
    
                let cooldownAt = this.WebRTC.preemptBootRateLimit.cooldownAt,
                    cooldownTTL = this.WebRTC.preemptBootRateLimit.cooldownTTL,
                    now = new Date().getTime();
    
                if (!this.isWatchingMe(username)) {
                    // Within the 'free' boot limits?
                    if (this.WebRTC.preemptBootRateLimit.counter < this.WebRTC.preemptBootRateLimit.maxFreeBoots) {
                        if (!ping) return false;
                        this.WebRTC.preemptBootRateLimit.counter++;
                    }
    
                    // Begin enforcing a cooldown TTL after a while.
                    if (this.WebRTC.preemptBootRateLimit.counter >= this.WebRTC.preemptBootRateLimit.maxFreeBoots) {
    
                        // Currently throttled?
                        if (cooldownAt !== null) {
                            if (now < cooldownAt) {
                                let delta = cooldownAt - now;
                                this.modalAlert({
                                    title: "You are doing that too often",
                                    message: "You have been pre-emptively booting an unusual number of people who weren't even watching your camera yet.\n\n" +
                                        `Please wait ${cooldownTTL} seconds between any additional pre-emptive boots.\n\n` +
                                        `You may try again after ${delta/1000} seconds.`,
                                });
                                return true;
                            }
                        }
    
                        // Refresh the timer on pings.
                        if (ping) {
                            this.WebRTC.preemptBootRateLimit.cooldownAt = now + (cooldownTTL * 1000);
                        }
                    }
                }
    
                return false;
            },
    
            // Stop broadcasting.
            stopVideo() {
                this.stopDarkVideoDetection();
    
                // Close all WebRTC sessions.
                for (let username of Object.keys(this.WebRTC.pc)) {
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
            setVideoVolume(username, volume) {
                // Set the volume on their video.
                let $ref = document.getElementById(`videofeed-${username}`);
                if ($ref) {
                    $ref.volume = volume / 100;
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
                        end(event) {
                            console.log(
                                'moved a distance of ' +
                                (Math.sqrt(Math.pow(event.pageX - event.x0, 2) +
                                    Math.pow(event.pageY - event.y0, 2) | 0))
                                    .toFixed(2) + 'px')
                        }
                    }
                }).resizable({
                    edges: { left: true, right: true, bottom: true },
                    listeners: {
                        move(event) {
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

            // Webcam "is speaking" functions.
            initSpeakingEvents(username, element) {
                // element is the <video> element, with the video stream
                // (whether from getUserMedia or WebRTC) on srcObject.

                let stream = element.srcObject,
                    feedElem = element.closest('div.feed'),
                    options = {},
                    speechEvents = hark(stream, options);

                speechEvents.on('speaking', () => {
                    feedElem.classList.add('is-speaking');
                    this.WebRTC.speaking[username] = true;
                });

                speechEvents.on('stopped_speaking', () => {
                    feedElem.classList.remove('is-speaking');
                    this.WebRTC.speaking[username] = false;
                });
            },

            // Dark video detection.
            initDarkVideoDetection() {
                if (this.webcam.darkVideo.canvas === null) {
                    let canvas = document.querySelector("#darkVideoCanvas"),
                        ctx = canvas.getContext('2d');
                    canvas.width = WebcamWidth;
                    canvas.height = WebcamHeight;
                    this.webcam.darkVideo.canvas = canvas;
                    this.webcam.darkVideo.ctx = ctx;
                }

                // Reset the dark frame counter.
                this.webcam.darkVideo.tooDarkFrames = 0;

                if (this.webcam.darkVideo.interval !== null) {
                    clearInterval(this.webcam.darkVideo.interval);
                }
                this.webcam.darkVideo.interval = setInterval(() => {
                    this.darkVideoInterval();
                }, 5000);
            },
            stopDarkVideoDetection() {
                if (this.webcam.darkVideo.interval !== null) {
                    clearInterval(this.webcam.darkVideo.interval);
                }

                // Remove the canplaythrough event, it will be re-added if the user restarts their cam.
                this.webcam.elem.removeEventListener("canplaythrough", this.initDarkVideoDetection);
            },
            darkVideoInterval() {
                if (!this.webcam.active) { // safety
                    this.stopDarkVideoDetection();
                    return;
                }

                // Take a screenshot from the user's local webcam.
                let canvas = this.webcam.darkVideo.canvas,
                    ctx = this.webcam.darkVideo.ctx;
                ctx.drawImage(this.webcam.elem, 0, 0, canvas.width, canvas.height);

                // Debugging: export the screenshot to a data URI.
                let img = canvas.toDataURL('image/jpeg');
                this.webcam.darkVideo.lastImage = img;

                // Get average RGB value.
                let rgb = this.getAverageRGB(ctx);
                if (rgb === null) {
                    return;
                }

                this.webcam.darkVideo.lastAverage = rgb;
                this.webcam.darkVideo.lastAverageColor = `rgba(${rgb[0]}, ${rgb[1]}, ${rgb[2]}, 1)`;

                // If they are exempt from the dark video rule, do not check their camera color.
                if (this.jwt.rules.IsNoDarkVideoRule) return;

                // If the average total color is below the threshold (too dark of a video).
                let averageBrightness = Math.floor((rgb[0] + rgb[1] + rgb[2]) / 3);
                if (averageBrightness < this.webcam.darkVideo.threshold) {

                    // Count for how many frames their camera is too dark.
                    this.webcam.darkVideo.tooDarkFrames++;

                    // After too long, cut their camera.
                    if (this.webcam.darkVideo.tooDarkFrames >= this.webcam.darkVideo.tooDarkFramesLimit) {
                        this.stopVideo();
                        this.ChatClient(`
                            Your webcam was too dark to see anything and has been turned off.<br><br>
                            <strong>Note:</strong> if your camera did not look dark to you and you believe there
                            may have been an error, please
                            <button type="button" onclick="SendMessage('/debug-dark-video')" class="button is-small is-link is-outlined">click here</button> to see
                            diagnostic information and contact a chat room moderator for assistance.
                        `);
                    }
                } else {
                    this.webcam.darkVideo.tooDarkFrames = 0;
                }
            },
            getAverageRGB(ctx) {
                // Helper function to compute the average color of a <canvas>.
                // Ref: https://stackoverflow.com/a/2541680
                const blockSize = 16; // only visit every N pixels
                let img = null,
                    rgb = [0, 0, 0];

                try {
                    img = ctx.getImageData(0, 0, WebcamWidth, WebcamHeight);
                } catch(e) {
                    // Not supported.
                    return null;
                }

                let length = img.data.length,
                    i = 0,
                    count = 0,
                    firstColor = [],
                    allSame = true;
                while ((i += blockSize * 4) < length) {
                    count++;
                    let thisColor = [
                        img.data[i],
                        img.data[i+1],
                        img.data[i+2]
                    ]

                    rgb[0] += thisColor[0];
                    rgb[1] += thisColor[1];
                    rgb[2] += thisColor[2];

                    // Also check whether every sampled pixel is THE SAME color,
                    // to detect users broadcasting a solid (bright) color.
                    if (firstColor.length === 0) {
                        firstColor = [ rgb[0], rgb[1], rgb[2] ];
                    } else if (allSame) {
                        if (firstColor[0] !== thisColor[0] ||
                            firstColor[1] !== thisColor[1] ||
                            firstColor[2] !== thisColor[2]
                        ) {
                            allSame = false;
                        }
                    }
                }

                // If all sampled colors were the same solid image: red flag!
                if (allSame) {
                    return [0, 0, 0];
                }

                rgb[0] = Math.floor(rgb[0]/count);
                rgb[1] = Math.floor(rgb[1]/count);
                rgb[2] = Math.floor(rgb[2]/count);

                return rgb;
            },

            // Scale the video zoom setting up or down.
            scaleVideoSize(bigger) {
                // Find the current size index.
                let currentSize = 0;
                for (let option of this.webcam.videoScaleOptions) {
                    if (option[0] === this.webcam.videoScale) {
                        break;
                    }
                    currentSize++;
                }

                // Adjust it.
                if (bigger) {
                    currentSize++;
                } else {
                    currentSize--;
                }

                // Constrain it.
                if (currentSize < 0) {
                    currentSize = 0;
                } else if (currentSize >= this.webcam.videoScaleOptions.length) {
                    currentSize = this.webcam.videoScale.length - 1;
                }

                // Set it.
                this.webcam.videoScale = this.webcam.videoScaleOptions[currentSize][0];
            },

            // Safety feature: if the user deletes the QR code watermark in their browser's dev
            // tools, this function is called.
            onDeletedWatermark(username) {
                // Report this to your main website.
                this.doCustomReport({
                    message: {
                        channel: 'n/a',
                        username: this.username,
                        at: new Date(),
                        message: 'User has deleted the webcam watermark QR code image!',
                    },
                    classification: 'Community Safety Violation',
                    comment: `The user ${this.username} is trying to circumvent the webcam watermarking system!\n\n` +
                        `It has been detected that the QR code overlay had been removed from the page, possibly via ${this.username} ` +
                        `using their browser's dev tools to do so.\n\nSpecifically, the webcam being broadcast by @${username} has had its ` +
                        `watermark element removed from the web page.`
                });

                // Broadcast our shame.
                this.client.send({
                    action: "message",
                    channel: "lobby",
                    message: `Hey @all I have been very naughty and have been caught red-handed tampering with the webcam QR code watermarks ` +
                        `here on the chat room. A special apology to @${username} whose camera I was messing with. I will now see myself out ` +
                        `of the chat room as I hang my head in shame.`,
                });

                this.panic();
            },
        }
    }

    /***************************
     * Common, stand-alone functions used by multiple components.
     * Important: these do not use Vue properties but are instead
     * self contained and safe to call from anywhere.
     ***************************/

    // Very similar to webcamIconClass but stand-alone and shared by
    // multiple components like WhoListRow and MessageBox.
    videoButtonClass(user, isVideoNotAllowed) {
        let result = "";

        // VIP background if their cam is set to VIPs only
        if ((user.video & VideoFlag.Active) && (user.video & VideoFlag.VipOnly)) {
            result = "has-background-vip ";
        }

        // Invited to watch: green border but otherwise red/blue icon.
        if ((user.video & VideoFlag.Active) && (user.video & VideoFlag.Invited)) {
            // Invited to watch; green border but blue/red icon.
            result += "video-button-invited ";
        }

        // Colors and/or cursors.
        if ((user.video & VideoFlag.Active) && (user.video & VideoFlag.NSFW)) {
            // Explicit camera: red border
            result += "is-danger is-outlined";
        } else if ((user.video & VideoFlag.Active) && !(user.video & VideoFlag.NSFW)) {
            // Normal camera: blue border
            result += "is-link is-outlined";
        } else if (isVideoNotAllowed) {
            // Default: grey border and not-allowed cursor.
            result += "cursor-notallowed";
        }

        return result;
    }

    videoButtonTitle(user) {
        // Mouse-over title text for the video button.
        let parts = ["Open video stream"];

        if (user.video & VideoFlag.MutualRequired) {
            parts.push("mutual video sharing required");
        }

        if (user.video & VideoFlag.MutualOpen) {
            parts.push("will auto-open your video");
        }

        if (user.video & VideoFlag.VipOnly) {
            parts.push(`${this.vipConfig.Name} only`);
        }

        if (user.video & VideoFlag.NonExplicit) {
            parts.push("prefers non-explicit video");
        }

        return parts.join("; ");
    }
}

const WebRTC = new WebRTCController();
export default WebRTC;
