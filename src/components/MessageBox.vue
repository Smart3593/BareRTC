<script>
import EmojiPicker from 'vue3-emoji-picker';
import LocalStorage from '../lib/LocalStorage';
import ScamDetection from './ScamDetection.vue';
import VideoFlag from '../lib/VideoFlag';
import WebRTC from '../lib/WebRTC';
import 'vue3-emoji-picker/css';

export default {
    props: {
        message: Object,     // chat Message object
        action: String,      // presence, notification, or (default) normal chat message
        appearance: String,  // message style appearance (cards, compact, etc.)
        user: Object,        // User object of the Message author
        isOffline: Boolean,  // user is not currently online
        username: String,    // current username logged in
        websiteUrl: String,  // Base URL to website (for profile/avatar URLs)
        isDnd: Boolean,      // user is not accepting DMs
        isMuted: Boolean,    // user is muted by current user
        reactions: Object,   // emoji reactions on the message
        reportEnabled: Boolean, // Report Message webhook is available
        position: Number,    // position of the message (0 to n), for the emoji menu to know which side to pop
        totalCount: Number,  // total count of messages
        isDm: Boolean,       // is in a DM thread (hide DM buttons)
        isOp: Boolean,       // current user is Operator (always show takeback button)
        noButtons: Boolean,  // hide all message buttons (e.g. for Report Modal)

        // User webcam settings
        myVideoActive: Boolean, // local user's camera is on
        isVideoNotAllowed: Boolean,
        videoIconClass: String,
        isInvitedVideo: Boolean, // we had already invited them to watch
    },
    components: {
        EmojiPicker,
        ScamDetection,
    },
    data() {
        return {
            VideoFlag: VideoFlag,

            // Emoji picker visible
            showEmojiPicker: false,

            // Message menu (compact displays, and overflow button on card layout)
            menuVisible: false,

            // Favorite emojis
            customEmojiGroups: {
                frequently_used: [
                    { n: ['heart'], u: '2764-fe0f' },
                    { n: ['+1', 'thumbs_up'], u: '1f44d' },
                    { n: ['-1', 'thumbs_down'], u: '1f44e' },
                    { n: ['rolling_on_the_floor_laughing'], u: '1f923' },
                    { n: ['wink'], u: '1f609' },
                    { n: ['cry'], u: '1f622' },
                    { n: ['angry'], u: '1f620' },
                    { n: ['heart_eyes'], u: '1f60d' },

                    { n: ['kissing_heart'], u: '1f618' },
                    { n: ['wave'], u: '1f44b' },
                    { n: ['fire'], u: '1f525' },
                    { n: ['smiling_imp'], u: '1f608' },
                    { n: ['peach'], u: '1f351' },
                    { n: ['eggplant', 'aubergine'], u: '1f346' },
                    { n: ['splash', 'sweat_drops'], u: '1f4a6' },
                    { n: ['banana'], u: '1f34c' },
                ]
            },

            // Emoji reactions are toggled fully spelled out (for mobile)
            showReactions: false,
        };
    },
    computed: {
        profileURL() {
            if (this.user.profileURL) {
                return this.urlFor(this.user.profileURL);
            }
            return null;
        },
        avatarURL() {
            if (this.user.avatar) {
                return this.urlFor(this.user.avatar);
            }
            return null;
        },
        nickname() {
            if (this.user.nickname) {
                return this.user.nickname;
            }
            return this.user.username;
        },
        hasReactions() {
            return this.reactions != undefined && Object.keys(this.reactions).length > 0;
        },

        // Compactify a message (remove paragraph breaks added by Markdown renderer)
        compactMessage() {
            return this.message.message.replace(/<\/p>\s*<p>/g, "<br><br>").replace(/<\/?p>/g, "");
        },

        emojiPickerTheme() {
            let theme = LocalStorage.get('theme');
            if (theme === 'light' || theme === 'dark') {
                return theme;
            }
            return 'auto';
        },

        // Unique ID component for dropdown menus.
        uniqueID() {
            // Messages sent by users always have a msgID, use that.
            if (this.message.msgID) {
                return this.message.msgID;
            }

            // Others (e.g. ChatServer messages), return something unique.
            return `${this.position}-${this.user.username}-${this.message.at}-${Math.random()*99999}`;
        },

        videoButtonClass() {
            return WebRTC.videoButtonClass(this.user, this.isVideoNotAllowed);
        },
        videoButtonTitle() {
            return WebRTC.videoButtonTitle(this.user);
        },
    },
    methods: {
        openProfile() {
            this.$emit('open-profile', this.message.username);
        },

        openDMs() {
            this.$emit('send-dm', {
                username: this.message.username,
            });
        },

        openVideo() {
            this.$emit('open-video', this.user);
        },

        inviteVideo() {
            this.$emit('invite-video', this.user.username);
        },

        muteUser() {
            this.$emit('mute-user', this.message.username);
        },

        takeback() {
            this.$emit('takeback', this.message);
        },

        removeMessage() {
            this.$emit('remove', this.message);
        },

        reportMessage() {
            this.$emit('report', this.message);
        },

        sendReact(emoji) {
            this.$emit('react', this.message, emoji);
        },

        // Vue3-emoji-picker callback
        onSelectEmoji(emoji) {
            this.sendReact(emoji.i);
            this.hideEmojiPicker();
        },

        // Hide the emoji menu (after sending an emoji or clicking the react button again)
        hideEmojiPicker() {
            if (!this.showEmojiPicker) return;
            window.requestAnimationFrame(() => {
                this.showEmojiPicker = false;
            });
        },

        urlFor(url) {
            // Prepend the base websiteUrl if the given URL is relative.
            if (url.match(/^https?:/i)) {
                return url;
            }
            return this.websiteUrl.replace(/\/+$/, "") + url;
        },

        // Current user has reacted to the message.
        iReacted(emoji) {
            if (!this.hasReactions) return false;

            // test whether the current user has reacted
            if (this.reactions[emoji] != undefined) {
                for (let reactor of this.reactions[emoji]) {
                    if (reactor === this.username) {
                        return true;
                    }
                }
            }
            return false;
        },

        // Google Translate link.
        translate() {
            let message = this.message?.message.replace(/<(.|\n)+?>/g, "");
            let url = `https://translate.google.com/?sl=auto&tl=en&text=${encodeURIComponent(message)}&op=translate`;
            window.open(url);
        },

        // TODO: DRY
        prettyDate(date) {
            if (date == undefined) return '';
            let hours = date.getHours(),
                minutes = String(date.getMinutes()).padStart(2, '0'),
                ampm = hours >= 12 ? "pm" : "am";

            let hour = hours % 12 || 12;
            return `${(hour)}:${minutes} ${ampm}`;
        },

        prettyDateCompact(date) {
            if (date == undefined) return '';
            let hour = date.getHours(),
                minutes = String(date.getMinutes()).padStart(2, '0');
            return `${hour}:${minutes}`;
        },
    }
}
</script>

<template>
    <!-- Presence message banners -->
    <div v-if="action === 'presence'" class="notification is-success is-light py-1 px-3 mb-2">

        <!-- Tiny avatar next to name and action buttons -->
        <div class="columns is-mobile">
            <div class="column is-narrow pr-0 pt-4">
                <a :href="profileURL" @click.prevent="openProfile()" :class="{ 'cursor-default': !profileURL }">
                    <figure class="image is-16x16">
                        <img v-if="avatarURL" :src="avatarURL">
                        <img v-else src="/static/img/shy.png">
                    </figure>
                </a>
            </div>
            <div class="column">
                <!-- Timestamp on the right -->
                <span class="float-right is-size-7" :title="message.at">
                    {{ prettyDate(message.at) }}
                </span>

                <span @click="openProfile()" class="cursor-pointer"
                    :class="{ 'strikethru': isOffline }">
                    <strong>{{ nickname }}</strong>
                    <small class="ml-1">(@{{ message.username }})</small>
                </span>
                {{ message.message }}
            </div>
        </div>

    </div>

    <!-- Notification message banners (e.g. DM disclaimer) -->
    <div v-else-if="action === 'notification'" class="notification is-warning is-light mb-2">
        <span v-html="message.message"></span>
    </div>

    <!-- Card Style (default) -->
    <div v-else-if="appearance === 'cards' || !appearance" class="box mb-2 px-4 pt-3 pb-1 position-relative">

        <!-- Profile picture, name and buttons row -->
        <div class="columns is-mobile mb-0">
            <div class="column is-narrow pr-0">
                <a :href="profileURL" @click.prevent="openProfile()">
                    <figure class="image is-48x48">
                        <img v-if="message.isChatServer" src="/static/img/server.png">
                        <img v-else-if="message.isChatClient" src="/static/img/client.png">
                        <img v-else-if="avatarURL" :src="avatarURL" :class="{'offline-avatar': isOffline}">
                        <img v-else src="/static/img/shy.png" :class="{'offline-avatar': isOffline}">
                    </figure>
                </a>
            </div>
            <div class="column is-narrow pb-0 px-4">
                <div class="user-nickname mb-4">
                    <strong :class="{
                        'has-text-success is-dark': message.isChatServer,
                        'has-text-warning is-dark': message.isAdmin,
                        'has-text-danger': message.isChatClient
                    }">

                        <!-- User nickname/display name -->
                        <span :class="{ 'strikethru': isOffline }">{{ nickname }}</span>

                        <span v-if="isOffline" class="ml-2">(offline)</span>
                    </strong>
                </div>

                <!-- User @username below it which may link to a profile URL if JWT -->
                <div class="columns is-mobile" v-if="(message.isChatClient || message.isChatServer)">
                    <div class="column is-narrow pt-0">
                        <small v-if="!(message.isChatClient || message.isChatServer)">
                            <a v-if="profileURL" :href="profileURL" target="_blank" @click.prevent="openProfile()"
                                class="has-text-grey">
                                @{{ message.username }}
                            </a>
                            <span v-else class="has-text-grey">@{{ message.username }}</span>
                        </small>
                        <small v-else class="has-text-grey">internal</small>
                    </div>
                </div>
                <div v-else class="columns is-mobile py-0">
                    <div class="column is-narrow py-0">
                        <small v-if="!(message.isChatClient || message.isChatServer)">
                            <a :href="profileURL || '#'" target="_blank" @click.prevent="openProfile()"
                                class="has-text-grey">
                                @{{ message.username }}
                            </a>

                            <!-- User badges -->
                            <sup v-if="user.op" class="fa fa-peace ml-1 has-text-warning is-size-7"></sup>
                        </small>
                        <small v-else class="has-text-grey">internal</small>
                    </div>

                    <div class="column is-narrow px-1 py-0" v-if="!noButtons">
                        <!-- DMs button -->
                        <button type="button" v-if="!(message.username === username || isDm)"
                            class="button is-small px-2" @click="openDMs()"
                            :title="isDnd ? 'This person is not accepting new DMs' : 'Open a Direct Message (DM) thread'"
                            :disabled="isDnd">
                            <i class="fa fa-comment"></i>
                        </button>

                        <!-- Webcam button -->
                        <button type="button" v-if="(user.video & VideoFlag.Active)"
                            class="button is-small ml-1 px-2" :class="videoButtonClass"
                            :title="videoButtonTitle"
                            @click.prevent="openVideo()">
                            <i class="fa" :class="videoIconClass"></i>
                        </button>
                    </div>

                    <!-- Overflow menu for lesser used options -->
                    <div class="column is-narrow pl-0 py-0 dropdown"
                        :class="{ 'is-up': position === totalCount-1, 'is-active': menuVisible }"
                        @click="menuVisible = !menuVisible">
                        <div class="dropdown-trigger">
                            <button type="button" class="button is-small" aria-haspopup="true"
                                :aria-controls="`msg-overflow-menu-${uniqueID}`">
                                <i class="fa fa-ellipsis-vertical"></i>
                            </button>
                        </div>
                        <div class="dropdown-menu" :id="`msg-overflow-menu-${uniqueID}`">
                            <div class="dropdown-content" role="menu">

                                <!-- Invite this user to watch my camera -->
                                <a href="#" class="dropdown-item" v-if="!(message.username === username) && myVideoActive && !isInvitedVideo"
                                    @click.prevent="inviteVideo()">
                                    <i class="fa fa-video mr-1 has-text-success"></i>
                                    Invite to watch my webcam
                                </a>

                                <!-- Mute/Unmute User -->
                                <a href="#" class="dropdown-item" v-if="!(message.username === username)"
                                    @click.prevent="muteUser()">
                                    <i class="fa fa-comment-slash mr-1" :class="{
                                        'has-text-success': isMuted,
                                        'has-text-danger': !isMuted
                                    }"></i>
                                    <span v-if="isMuted">Unmute user</span>
                                    <span v-else>Mute user</span>
                                </a>

                                <!-- Owner/admin: take back message -->
                                <a href="#" class="dropdown-item" v-if="message.msgID && (message.username === username || isOp)"
                                    @click.prevent="takeback()" :data-msgid="message.msgID">
                                    <i class="fa fa-rotate-left has-text-danger mr-1"></i>
                                    Take back
                                </a>

                                <!-- Everyone else: hide message instead -->
                                <a href="#" class="dropdown-item" v-if="message.username !== username"
                                    @click.prevent="removeMessage()">
                                    <i class="fa fa-trash mr-1"></i>
                                    Hide message
                                </a>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            <div class="column has-text-right pb-0 pl-0">
                <small class="has-text-grey is-size-7" :title="message.at">{{ prettyDate(message.at) }}</small>
            </div>
        </div>

        <!-- Report & Emoji buttons -->
        <div v-if="!noButtons" class="emoji-button columns is-mobile is-gapless mb-0">
            <!-- Translate message button -->
            <div class="column">
                <button class="button is-small mr-1 has-text-success"
                    title="Translate this message using Google Translate"
                    @click.prevent="translate()">
                    <i class="fab fa-google has-text-success"></i>
                    <span class="has-text-success ml-2">Translate</span>
                </button>
            </div>

            <!-- Report message button -->
            <div class="column" v-if="message.msgID && reportEnabled && message.username !== username">
                <button class="button is-small is-outlined mr-1 py-2" :class="{
                    'is-danger': !message.reported,
                    'has-text-grey': message.reported
                }" title="Report this message" @click.prevent="reportMessage()">
                    <i class="fa fa-flag"></i>
                    <i class="fa fa-check ml-1" v-if="message.reported"></i>
                </button>
            </div>

            <div class="column dropdown is-right" v-if="message.msgID"
                :class="{ 'is-up': position >= 2, 'is-active': showEmojiPicker }"
                @click="showEmojiPicker = true">
                <div class="dropdown-trigger">
                    <button type="button" class="button is-small px-2" aria-haspopup="true"
                        :aria-controls="`react-menu-${uniqueID}`" @click.prevent="hideEmojiPicker()">
                        <span>
                            <i class="fa fa-heart has-text-grey"></i>
                            <i class="fa fa-plus has-text-grey pl-1"></i>
                        </span>
                    </button>
                </div>
                <div class="dropdown-menu" :id="`react-menu-${uniqueID}`" role="menu">
                    <div class="dropdown-content p-0">
                        <!-- Emoji reactions menu -->
                        <EmojiPicker v-if="showEmojiPicker" :native="true" :display-recent="true" :disable-skin-tones="true"
                            :additional-groups="customEmojiGroups" :group-names="{ frequently_used: 'Frequently Used' }"
                            :theme="emojiPickerTheme" @select="onSelectEmoji"></EmojiPicker>
                    </div>
                </div>
            </div>
        </div>

        <!-- Message box -->
        <div class="content pl-5 pb-3 pt-1 mb-5">
            <em v-if="message.action === 'presence'">{{ message.message }}</em>
            <div v-else v-html="message.message"></div>

            <!-- Possible scam message disclaimer -->
            <ScamDetection v-if="message.username !== username"
                :username="message.username"
                :message="message.message">
            </ScamDetection>

            <!-- Reactions so far? -->
            <div v-if="hasReactions" class="mt-1">
                <span v-for="(users, emoji) in reactions" v-bind:key="emoji" class="tag mr-1 cursor-pointer"
                    :class="{ 'has-text-weight-bold': iReacted(emoji), 'is-secondary': !iReacted(emoji) }"
                    :title="emoji + ' by: ' + users.join(', ')" @click="sendReact(emoji)">
                    {{ emoji }}

                    <small v-if="showReactions" class="ml-1">
                        {{ users.join(', ') }}
                    </small>
                    <small v-else class="ml-1">{{ users.length }}</small>
                </span>

                <!-- Mobile helper to show all -->
                <a href="#" class="tag is-secondary cursor-pointer" @click.prevent="showReactions = !showReactions">
                    <i class="fa mr-1"
                        :class="{'fa-angles-left': showReactions,
                                 'fa-angles-right': !showReactions,
                        }"></i> {{ showReactions ? 'Less' : 'More' }}
                </a>
            </div>
        </div>

    </div>

    <!-- Compact styles (with or without usernames) -->
    <div v-else-if="appearance.indexOf('compact') === 0" class="columns is-mobile">
        <!-- Timestamp -->
        <div class="column is-narrow pr-0">
            <small class="has-text-grey is-size-7" :title="message.at">{{ prettyDateCompact(message.at) }}</small>
        </div>

        <!-- Avatar icon -->
        <div class="column is-narrow px-1">
            <a :href="profileURL" @click.prevent="openProfile()" class="p-0">
                <img v-if="avatarURL" :src="avatarURL" width="16" height="16" alt="" :class="{'offline-avatar': isOffline}">
                <img v-else src="/static/img/shy.png" width="16" height="16" :class="{'offline-avatar': isOffline}">
            </a>
        </div>

        <!-- Name/username/message -->
        <div class="column px-1">
            <div class="content mb-2">
                <strong :class="{
                    'has-text-success is-dark': message.isChatServer,
                    'has-text-warning is-dark': message.isAdmin,
                    'has-text-danger': message.isChatClient
                }">
                    [<a :href="profileURL" @click.prevent="openProfile()" class="has-text-dark"
                        :class="{ 'cursor-default': !profileURL }">
                        <!-- Display name? -->
                        <span v-if="(message.isChatServer || message.isChatClient || message.isAdmin)
                            || (appearance === 'compact' && nickname !== message.username)" :class="{
        'has-text-success is-dark': message.isChatServer,
        'has-text-warning is-dark': message.isAdmin,
        'has-text-danger': message.isChatClient
    }">
                            <span :class="{ 'strikethru': isOffline }">{{ nickname }}</span>
                        </span>

                        <small class="has-text-grey"
                            :class="{ 'ml-1': appearance === 'compact' && nickname !== message.username, 'strikethru': isOffline }"
                            v-if="!(message.isChatServer || message.isChatClient || message.isAdmin)">
                            @{{ message.username}}

                            <!-- User badges -->
                            <sup v-if="user.op" class="fa fa-peace has-text-warning is-size-7"></sup>
                        </small>
                    </a>]
                </strong>

                <span v-html="compactMessage"></span>

                <!-- Possible scam message disclaimer -->
                <ScamDetection v-if="message.username !== username"
                    :username="message.username"
                    :message="message.message">
                </ScamDetection>
            </div>

            <!-- Reactions so far? -->
            <div v-if="hasReactions" class="mb-2">
                <span v-for="(users, emoji) in reactions" v-bind:key="emoji" class="tag mr-1 cursor-pointer"
                    :class="{ 'has-text-weight-bold': iReacted(emoji), 'is-secondary': !iReacted(emoji) }"
                    :title="emoji + ' by: ' + users.join(', ')" @click="sendReact(emoji)">
                    {{ emoji }}

                    <small v-if="showReactions" class="ml-1">
                        {{ users.join(', ') }}
                    </small>
                    <small v-else class="ml-1">{{ users.length }}</small>
                </span>

                <!-- Mobile helper to show all -->
                <a href="#" class="tag is-secondary cursor-pointer" @click.prevent="showReactions = !showReactions">
                    <i class="fa mr-1"
                        :class="{'fa-angles-left': showReactions,
                                 'fa-angles-right': !showReactions,
                        }"></i> {{ showReactions ? 'Less' : 'More' }}
                </a>
            </div>

        </div>

        <!-- Emoji/Menu button -->
        <div v-if="!noButtons" class="column is-narrow pl-1">

            <div class="columns is-mobile is-gapless mb-0">
                <!-- More buttons menu (DM, mute, report, etc.) -->
                <div class="column dropdown is-right"
                    :class="{ 'is-up': position >= 2, 'is-active': menuVisible }"
                    @click="menuVisible = !menuVisible">
                    <div class="dropdown-trigger">
                        <button type="button" class="button is-small px-2 mr-1" aria-haspopup="true"
                            :aria-controls="`msg-menu-${uniqueID}`">
                            <small>
                                <i class="fa fa-ellipsis-vertical"></i>
                            </small>
                        </button>
                    </div>
                    <div class="dropdown-menu" :id="`msg-menu-${uniqueID}`" role="menu">
                        <div class="dropdown-content">
                            <a href="#" class="dropdown-item" v-if="message.msgID && message.username !== username"
                                @click.prevent="openDMs()">
                                <i class="fa fa-comment mr-1"></i> Direct Message
                            </a>

                            <!-- Invite this user to watch my camera -->
                            <a href="#" class="dropdown-item" v-if="!(message.username === username) && myVideoActive && !isInvitedVideo"
                                @click.prevent="inviteVideo()">
                                <i class="fa fa-video mr-1 has-text-success"></i>
                                Invite to watch my webcam
                            </a>

                            <a href="#" class="dropdown-item" v-if="message.msgID && !message.username !== username"
                                @click.prevent="muteUser()">
                                <i class="fa fa-comment-slash mr-1" :class="{
                                    'has-text-success': isMuted,
                                    'has-text-danger': !isMuted
                                }"></i>
                                <span v-if="isMuted">Unmute user</span>
                                <span v-else>Mute user</span>
                            </a>

                            <a href="#" class="dropdown-item" v-if="message.msgID && message.username === username || isOp"
                                @click.prevent="takeback()" :data-msgid="message.msgID">
                                <i class="fa fa-rotate-left has-text-danger mr-1"></i>
                                Take back
                            </a>

                            <a href="#" class="dropdown-item" v-if="message.username !== username"
                                @click.prevent="removeMessage()">
                                <i class="fa fa-trash mr-1"></i>
                                Hide message
                            </a>

                            <!-- Google Translate -->
                            <a href="#" class="dropdown-item"
                                @click.prevent="translate()">
                                <i class="fab fa-google has-text-success mr-1"></i>
                                Google Translate <i class="fa fa-external-link ml-1"></i>
                            </a>

                            <!-- Report button -->
                            <a href="#" class="dropdown-item" v-if="message.msgID && reportEnabled && message.username !== username"
                                @click.prevent="reportMessage()">
                                <i class="fa fa-flag mr-1" :class="{ 'has-text-danger': !message.reported }"></i>
                                <span v-if="message.reported">Reported</span>
                                <span v-else>Report</span>
                            </a>
                        </div>
                    </div>
                </div>

                <!-- Webcam button -->
                <div class="column" v-if="(user.video & VideoFlag.Active)">
                    <button type="button"
                        class="button is-small mr-1 px-2" :class="videoButtonClass"
                        :title="videoButtonTitle"
                        @click.prevent="openVideo()">
                        <small>
                            <i class="fa" :class="videoIconClass"></i>
                        </small>
                    </button>
                </div>

                <!-- Emoji reactions -->
                <div class="column dropdown is-right" v-if="message.msgID"
                    :class="{ 'is-up': position >= 2, 'is-active': showEmojiPicker }"
                    @click="showEmojiPicker = true">
                    <div class="dropdown-trigger">
                        <button type="button" class="button is-small px-2" aria-haspopup="true"
                            :aria-controls="`react-menu-${uniqueID}`" @click="hideEmojiPicker()">
                            <small>
                                <i class="fa fa-heart has-text-grey"></i>
                            </small>
                        </button>
                    </div>
                    <div class="dropdown-menu" :id="`react-menu-${uniqueID}`" role="menu">
                        <div class="dropdown-content p-0">
                            <!-- Emoji reactions menu -->
                            <EmojiPicker v-if="showEmojiPicker" :native="true" :display-recent="true"
                                :disable-skin-tones="true" :additional-groups="customEmojiGroups"
                                :group-names="{ frequently_used: 'Frequently Used' }" theme="auto" @select="onSelectEmoji">
                            </EmojiPicker>
                        </div>
                    </div>
                </div>
            </div>
        </div>

    </div>
</template>

<style scoped>
/* Trim display name lines on very small screens */
.user-nickname {
    max-width: calc(150px + (100vw - 380px));
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
}
@media (min-width: 768px) {
    .user-nickname {
        max-width: none;
    }
}

/* Offline user styles */
.offline-avatar {
    filter: grayscale();
}
.strikethru {
    text-decoration: line-through;
}
</style>
