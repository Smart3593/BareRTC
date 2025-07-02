<script>
import AlertModal from './AlertModal.vue';
import VideoFlag from '../lib/VideoFlag';

export default {
    props: {
        visible: Boolean,
        jwt: String, // caller's JWT token for authorization
        user: Object, // the user we are viewing
        username: String, // the local user
        isViewerOp: Boolean, // the viewer is an operator (show buttons)
        websiteUrl: String,
        isWatching: Boolean, // chat user is currently watching the profile user
        isDnd: Boolean,
        isMuted: Boolean,
        isBooted: Boolean,
        profileWebhookEnabled: Boolean,
        vipConfig: Object,  // VIP config settings for BareRTC
        myVideoActive: Boolean, // local user's camera is on
        isInvitedVideo: Boolean, // we had already invited them to watch
    },
    components: {
        AlertModal,
    },
    data() {
        return {
            busy: false,

            // Profile data
            headline: "",
            profileFields: [],

            // Ban account data
            banModalVisible: false,
            banReason: "",
            banDuration: 24,

            // Kick user modal.
            kickModalVisible: false,
            kickReason: "",

            // Alert modal
            alertModal: {
                visible: false,
                isConfirm: false,
                title: "Alerte",
                icon: "fa-exclamation-triangle",
                message: "",
                callback() {},
            },

            // The local user has sent a NSFW nudge already.
            sentNsfwNudge: false,

            // Error messaging from backend
            error: null,
        };
    },
    watch: {
        visible() {
            if (this.visible) {
                this.refresh();
            } else {
                this.profileFields = [];
                this.error = null;
                this.busy = false;
            }
        }
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
        isOnBlueCam() {
            // User is broadcasting a cam and is not NSFW.
            if ((this.user.video & VideoFlag.Active) && !(this.user.video & VideoFlag.NSFW)) {
                return true;
            }
            return false;
        },
        isOnCamera() {
            // User's camera is enabled.
            return (this.user.video & VideoFlag.Active);
        },
        onlineSince() {
            let date = new Date(this.user.loginAt * 1000),
                year = date.getFullYear(),
                month = String(date.getMonth() + 1).padStart(2, '0'),
                day = String(date.getDate()).padStart(2, '0'),
                hours = String(date.getHours()).padStart(2, '0'),
                minutes = String(date.getMinutes()).padStart(2, '0'),
                seconds = String(date.getSeconds()).padStart(2, '0'),
                ampm = 'AM';
            if (hours >= 12) {
                if (hours > 12) {
                    hours -= 12;
                }
                ampm = 'PM';
            }
            return `${year}-${month}-${day} @ ${hours}:${minutes}:${seconds} ${ampm}`;
        },
    },
    methods: {
        refresh() {
            if (!this.profileWebhookEnabled) return;
            if (!this.user || !this.user?.username) return;
            this.busy = true;
            this.sentNsfwNudge = false;
            return fetch("/api/profile", {
                method: "POST",
                mode: "same-origin",
                cache: "no-cache",
                credentials: "same-origin",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    "JWTToken": this.jwt,
                    "Username": this.user?.username,
                }),
            })
            .then((response) => response.json())
            .then((data) => {
                if (data.Error) {
                    this.error = data.Error;
                    return;
                }

                this.headline = data.Headline ? data.Headline : "";

                if (data.ProfileFields != undefined) {
                    this.profileFields = data.ProfileFields;
                }
            }).catch(resp => {
                this.error = resp;
            }).finally(() => {
                this.busy = false;
            })
        },

        cancel() {
            this.$emit("cancel");
        },

        openProfile() {
            let url = this.profileURL;
            if (url) {
                window.open(url);
            }
        },

        openDMs() {
            this.cancel();
            this.$emit('send-dm', {
                username: this.user.username,
            });
        },

        muteUser() {
            this.$emit('mute-user', this.user.username);
        },

        bootUser() {
            this.$emit('boot-user', this.user.username);
        },

        inviteVideo() {
            this.$emit('invite-video', this.user.username);
        },

        // Operator commands (may be rejected by server if not really Op)
        markNsfw() {
            this.modalConfirm({
                message: "Marquer la webcam de cet utilisateur comme 'Explicite' ?\n\n" +
                    `Si @${this.user.username} a un comportement sexuel sur une caméra Bleue, cliquez sur OK pour confirmer ` +
                    "que sa caméra doit être marquée comme Rouge (explicite).",
                title: "Marquer une webcam comme Explicite",
                icon: "fa fa-fire",
            }).then(() => {
                this.$emit('send-command', `/nsfw ${this.user.username}`);

                // Close the modal immediately: our view of the user's cam data is a copy
                // and we can't follow the current value.
                this.cancel();
            });
        },
        nudgeNsfw() {
            // Gentler, public user-facing version to gently remind the user that
            // their webcam should probably be marked as Explicit.
            if (this.sentNsfwNudge) return;

            this.modalConfirm({
                title: "Mark a webcam as Explicit",
                message: `Should @${this.user.username}'s webcam be marked as Explicit?\n\n` +
                    `If their webcam is 'blue' and they are behaving sexually on camera, you may send them a gentle reminder ` +
                    `and ask them to tag their webcam as Explicit.\n\n` +
                    `They will not know for sure that it was you who did this.`,
                icon: "fa fa-fire",
            }).then(() => {
                this.$emit('nudge-nsfw', this.user.username);
                this.sentNsfwNudge = true;
            });
        },
        cutCamera() {
            this.modalConfirm({
                message: "Make this user stop broadcasting their camera?",
                title: "Cut Camera",
                icon: "fa fa-video-slash",
            }).then(() => {
                this.$emit('send-command', `/cut ${this.user.username}`);
                this.cancel();
            });
        },
        kickUser() {
            this.kickModalVisible = true;
            this.kickReason = "";
            window.requestAnimationFrame(() => {
                let reason = document.querySelector("#kick_reason");
                if (reason) {
                    reason.focus();
                }
            });
        },
        confirmKick() {
            this.$emit('send-command', `/kick ${this.user.username}`);

            // Also send an admin report to the main website.
            this.$emit('report', {
                message: {
                    channel: `n/a`,
                    username: this.user.username,
                    at: new Date(),
                    message: 'Kick reason: ' + this.kickReason,
                },
                classification: 'User kicked by admin',
                comment: `The chat admin @${this.username} has kicked ${this.user.username} from the room!\n\n` +
                    `* Chat admin: <a href="/u/${this.username}">${this.username}</a>\n` +
                    `* Reason: ${this.kickReason}`,
            });

            this.kickModalVisible = false;
            this.cancel();
        },
        banUser() {
            this.banModalVisible = true;
            this.banReason = "";
            this.banDuration = 24;
            window.requestAnimationFrame(() => {
                let reason = document.querySelector("#ban_reason");
                if (reason) {
                    reason.focus();
                }
            });
        },
        confirmBan() {
            // Send the ban command.
            this.$emit('send-command', `/ban ${this.user.username} ${this.banDuration}`);

            // Also send an admin report to the main website.
            this.$emit('report', {
                message: {
                    channel: `n/a`,
                    username: this.user.username,
                    at: new Date(),
                    message: 'Ban reason: ' + this.banReason,
                },
                classification: 'User banned by admin',
                comment: `A chat admin has banned ${this.user.username} from the chat room!\n\n` +
                    `* Chat admin: <a href="/u/${this.username}">${this.username}</a>\n` +
                    `* Reason: ${this.banReason}\n` +
                    `* Duration: ${this.banDuration} hours`,
            });

            this.banModalVisible = false;
            this.cancel();
        },

        urlFor(url) {
            // Prepend the base websiteUrl if the given URL is relative.
            if (url.match(/^https?:/i)) {
                return url;
            }
            return this.websiteUrl.replace(/\/+$/, "") + url;
        },

        // Alert Modal funcs, copied from/the same as App.vue (TODO: make it D.R.Y.)
        async modalAlert({ message, title="Alert", icon="", isConfirm=false }) {
            return new Promise((resolve, reject) => {
                this.alertModal.isConfirm = isConfirm;
                this.alertModal.title = title;
                this.alertModal.icon = icon;
                this.alertModal.message = message;
                this.alertModal.callback = () => {
                    resolve();
                };
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
    },
}
</script>

<template>
    <!-- Profile Card Modal -->
    <div class="modal" :class="{ 'is-active': visible }">
        <div class="modal-background" @click="cancel()"></div>
        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-success">
                    <p class="card-header-title">Profile Card</p>
                </header>
                <div class="card-content">

                    <!-- Avatar and name/username media -->
                    <div class="media mb-0">
                        <div class="media-left">
                            <a :href="profileURL"
                                @click.prevent="openProfile()"
                                :class="{ 'cursor-default': !profileURL }">
                                <figure class="image is-96x96">
                                    <img v-if="avatarURL"
                                        :src="avatarURL">
                                    <img v-else src="/static/img/shy.png">
                                </figure>
                            </a>
                        </div>
                        <div class="media-content">
                            <strong>
                                <!-- User nickname/display name -->
                                {{ nickname }}
                            </strong>
                            <div>
                                <small>
                                    <a v-if="profileURL"
                                        :href="profileURL" target="_blank"
                                        class="has-text-grey">
                                        @{{ user.username }}
                                    </a>
                                    <span v-else class="has-text-grey">@{{ user.username }}</span>
                                </small>
                            </div>

                            <!-- User badges -->
                            <div v-if="user.op || user.vip || user.emoji" class="mt-4">
                                <!-- Emoji icon -->
                                <span v-if="user.emoji" class="mr-2">
                                    {{ user.emoji }}
                                </span>

                                <!-- Operator? -->
                                <span v-if="user.op" class="tag is-warning is-light mr-2">
                                    <i class="fa fa-peace mr-1"></i> Operator
                                </span>

                                <!-- VIP? -->
                                <span v-if="vipConfig && user.vip" class="tag is-success is-light mr-2"
                                    :title="vipConfig.Name">
                                    <i class="mr-1" :class="vipConfig.Icon"></i>
                                    {{ vipConfig.Name }}
                                </span>
                            </div>
                        </div>
                    </div>

                    <!-- Action buttons -->
                    <div v-if="user.username !== username" class="mt-4">
                        <!-- DMs button -->
                        <button type="button"
                            class="button is-small px-2 mb-1"
                            @click="openDMs()"
                            :title="isDnd ? 'This person is not accepting new DMs' : 'Open a Direct Message (DM) thread'"
                            :disabled="isDnd">
                            <i class="fa mr-1" :class="{'fa-comment': !isDnd, 'fa-comment-slash': isDnd}"></i>
                            Direct Message
                        </button>

                        <!-- Mute button -->
                        <button type="button"
                            class="button is-small px-2 ml-1 mb-1"
                            @click="muteUser()" title="Mute user">
                            <i class="fa fa-comment-slash mr-1" :class="{
                                'has-text-success': isMuted,
                                'has-text-danger': !isMuted
                            }"></i>
                            {{ isMuted ? "Unmute" : "Mute" }} Messages
                        </button>

                        <!-- Un-Boot button -->
                        <button type="button"
                            class="button is-small px-2 ml-1 mb-1"
                            @click="bootUser()" title="Boot user off your webcam">
                            <i class="fa fa-user-xmark mr-1" :class="{
                                'has-text-danger': !isBooted,
                                'has-text-success': isBooted,
                            }"></i>
                            {{  isBooted ? 'Unboot from my webcam' : "Boot from my webcam" }}
                        </button>

                        <!-- Invite to watch me button -->
                        <button type="button"
                            v-if="myVideoActive && !isBooted && !isInvitedVideo"
                            class="button is-small px-2 ml-1 mb-1"
                            @click="inviteVideo()" title="Invite to watch my webcam">
                            <i class="fa fa-video has-text-success mr-1"></i>
                            Invite to watch me
                        </button>

                        <!-- Admin actions -->
                        <div v-if="isViewerOp" class="mt-1">
                            <!-- Mark camera NSFW -->
                            <button v-if="isOnBlueCam"
                                type="button"
                                class="button is-small is-outlined is-danger has-text-dark px-2 mr-1 mb-1"
                                @click="markNsfw()" title="Mark their camera as Explicit (red).">
                                <i class="fa fa-video mr-1 has-text-danger"></i>
                                Mark camera as Explicit
                            </button>

                            <!-- Cut camera -->
                            <button v-if="isOnCamera"
                                type="button"
                                class="button is-small is-outlined is-danger has-text-dark px-2 mr-1 mb-1"
                                @click="cutCamera()" title="Turn their camera off.">
                                <i class="fa fa-stop mr-1 has-text-danger"></i>
                                Cut camera
                            </button>

                            <!-- Kick user -->
                            <button type="button"
                                class="button is-small is-outlined is-danger has-text-dark px-2 mr-1 mb-1"
                                @click="kickUser()" title="Kick this user from the chat room.">
                                <i class="fa fa-shoe-prints mr-1 has-text-danger"></i>
                                Kick from the room
                            </button>

                            <!-- Ban user -->
                            <button type="button"
                                class="button is-small is-outlined is-danger has-text-dark px-2 mb-1"
                                @click="banUser()" title="Ban this user from the chat room for 24 hours.">
                                <i class="fa fa-clock mr-1 has-text-danger"></i>
                                Ban from chat
                            </button>
                        </div>
                    </div>

                    <!-- Headline (optional) -->
                    <div v-if="headline" class="mt-3 is-size-7 notification is-info is-light py-1 px-2 mb-3" style="display: table; table-layout: fixed; width: 100%">
                        <div style="display: table-cell; white-space: nowrap; text-overflow: ellipsis; overflow: hidden" :title="headline">
                            <span>{{headline}}</span>
                        </div>
                    </div>
                    <div v-else class="mt-4"></div>

                    <!-- Login At -->
                    <div class="is-size-7">
                        <div class="columns is-multiline is-mobile">
                            <div class="column is-half">
                                <em>Online since: {{ onlineSince }}</em>
                            </div>

                            <!-- Public user button to 'gently nudge this blue camera as NSFW' -->
                            <div class="column is-half" v-if="isOnBlueCam && isWatching">
                                <a href="#" v-if="isOnBlueCam && isWatching"
                                    type="button"
                                    class="has-text-danger"
                                    @click="nudgeNsfw()" title="Should their webcam be marked as Explicit?">
                                    <i class="fa fa-fire mr-1"></i>
                                    {{ sentNsfwNudge ? 'Thank you for helping tag this camera as Explicit!' : 'Should their camera be marked as Explicit?' }}
                                </a>
                            </div>
                        </div>
                    </div>

                    <!-- Profile Fields spinner/error -->
                    <div class="notification is-success is-light p-2 my-2" v-if="busy">
                        <i class="fa fa-spinner fa-spin mr-2"></i>
                        Loading profile details...
                    </div>
                    <div class="notification is-danger is-light p-2 my-2" v-else-if="error">
                        <i class="fa fa-exclamation-triangle mr-2"></i>
                        Error loading profile details:
                        {{ error }}
                    </div>

                    <!-- Profile Fields -->
                    <div class="columns is-multiline is-mobile mt-3"
                        v-else-if="profileFields.length > 0">
                        <div class="column py-1"
                            v-for="(field, i) in profileFields"
                            v-bind:key="field.Name"
                            :class="{'is-half': i < profileFields.length-1}">
                            <strong>{{ field.Name }}:</strong>
                            {{ field.Value }}
                        </div>
                    </div>
                </div>
                <footer class="card-footer">
                    <a :href="profileURL" target="_blank"
                        v-if="profileURL" class="card-footer-item"
                        @click="cancel()">
                        Full profile <i class="fa fa-external-link ml-2"></i>
                    </a>
                    <a href="#" @click.prevent="cancel()" class="card-footer-item">
                        Close
                    </a>
                </footer>
            </div>
        </div>
    </div>

    <!-- Alert modal (for alert/confirm prompts) -->
    <AlertModal :visible="alertModal.visible"
        :is-confirm="alertModal.isConfirm"
        :title="alertModal.title"
        :icon="alertModal.icon"
        :message="alertModal.message"
        @callback="alertModal.callback"
        @close="modalClose()"></AlertModal>

    <!-- Ban User Modal (for chat admins) -->
    <div class="modal" :class="{ 'is-active': banModalVisible }">
        <div class="modal-background" @click="banModalVisible = false"></div>
        <div class="modal-content">
            <form @submit.prevent="confirmBan">
                <div class="card">
                    <header class="card-header has-background-danger">
                        <p class="card-header-title">Ban User</p>
                    </header>
                    <div class="card-content">
                        <div class="field">
                            <label class="label" for="ban_reason">Reason for the ban:</label>
                            <input type="text" class="input"
                                id="ban_reason"
                                placeholder="Please describe why this user will be banned."
                                v-model="banReason"
                                required>
                            <p class="help">
                                This reason is NOT shown to the banned user, but will be sent to the main website
                                in an admin report so that it may be documented in this user's history.
                            </p>
                        </div>

                        <div class="field">
                            <label class="label" for="ban_duration">How long for the ban? (1-96 hours)</label>
                            <input type="number" min="1" max="96" v-model="banDuration" class="input">
                        </div>

                        <div class="field has-text-centered">
                            <button type="submit" class="button is-danger">
                                Confirm Ban
                            </button>
                            <a href="#" @click.prevent="banModalVisible = false" class="button ml-2">
                                Cancel
                            </a>
                        </div>
                    </div>
                </div>
            </form>
        </div>
    </div>

    <!-- Kick User Modal (for chat admins) -->
    <div class="modal" :class="{ 'is-active': kickModalVisible }">
        <div class="modal-background" @click="kickModalVisible = false"></div>
        <div class="modal-content">
            <form @submit.prevent="confirmKick">
                <div class="card">
                    <header class="card-header has-background-danger">
                        <p class="card-header-title">Kick User</p>
                    </header>
                    <div class="card-content">
                        <div class="field">
                            <label class="label" for="kick_reason">Reason for the kick:</label>
                            <input type="text" class="input"
                                id="kick_reason"
                                placeholder="Please describe why this user will be kicked from the room."
                                v-model="kickReason"
                                required>
                            <p class="help">
                                This reason is NOT shown to the kicked user, but will be sent to the main website
                                in an admin report so that it may be documented in this user's history.
                            </p>
                        </div>

                        <div class="field has-text-centered">
                            <button type="submit" class="button is-danger">
                                Confirm Kick
                            </button>
                            <a href="#" @click.prevent="kickModalVisible = false" class="button ml-2">
                                Cancel
                            </a>
                        </div>
                    </div>
                </div>
            </form>
        </div>
    </div>
</template>

<style scoped>
</style>
