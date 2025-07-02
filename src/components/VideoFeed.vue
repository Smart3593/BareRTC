<script>
import Slider from 'vue3-slider';

export default {
    props: {
        localVideo: Boolean,  // is our local webcam (not other's camera)
        poppedOut: Boolean,   // Video is popped-out and draggable
        username: String,     // username related to this video
        isExplicit: Boolean,  // camera is marked Explicit
        isMuted: Boolean,     // camera is muted on our end
        isSourceMuted: Boolean, // camera is muted on the broadcaster's end
        isWatchingMe: Boolean, // other video is watching us back
        isFrozen: Boolean,     // video is detected as frozen
        isSpeaking: Boolean,   // video is registering audio
        watermarkImage: Image, // watermark image to overlay (nullable)
        isNoWatermark: Boolean, // camera does not want watermark protection
    },
    components: {
        Slider,
    },
    data() {
        return {
            // Volume slider
            volume: 100,

            // Volume change debounce
            volumeDebounce: null,

            // Mouse over status
            mouseOver: false,

            // Mutation Observer, to detect if the user deletes the QR code via dev tools.
            observer: null,

            // Window resize debouncer: temporarily hide videos after a resize event, so e.g.,
            // if dev tools are opened and the debugger detector kicks in, videos may be hidden
            // while the page is paused.
            windowResizeTimer: null,
            windowResizeCooldown: false,
        };
    },
    mounted() {
        this.initWatermarkObserver();
        this.initWindowResizer();
    },
    beforeUnmount() {
        this.observer?.disconnect();
    },
    computed: {
        containerID() {
            return this.videoID + '-container';
        },
        videoID() {
            return this.localVideo ? 'localVideo' : `videofeed-${this.username}`;
        },
        textColorClass() {
            return this.isExplicit ? 'has-text-camera-red' : 'has-text-camera-blue';
        },
        muteButtonClass() {
            let classList = [
                'button is-small ml-1 p-2',
            ]

            if (this.isMuted) {
                classList.push('is-danger');
            } else {
                classList.push('is-success is-outlined');
            }

            if (!this.mouseOver) {
                classList.push('seethru');
            }

            return classList.join(' ');
        },
        muteIconClass() {
            if (this.localVideo) {
                return this.isMuted ? 'fa-microphone-slash' : 'fa-microphone';
            }
            return this.isMuted ? 'fa-volume-xmark' : 'fa-volume-high';
        }
    },
    methods: {
        closeVideo() {
            // Note: closeVideo only available for OTHER peoples cameras.
            // Closes the WebRTC connection as the offerer.
            this.$emit('close-video', this.username, 'offerer');
        },

        reopenVideo() {
            // Note: goes into openVideo(username, force)
            this.$emit('reopen-video', this.username, true);
        },

        openProfile() {
            this.$emit('open-profile', this.username);
        },

        // Toggle the Mute button
        muteVideo() {
            this.$emit('mute-video', this.username);
        },

        popoutVideo() {
            this.$emit('popout', this.username);
        },

        fullscreen(force=false) {
            // If we are popped-out, pop back in before full screen.
            if (this.poppedOut && !force) {
                this.popoutVideo();
                window.requestAnimationFrame(() => {
                    this.fullscreen(true);
                });
                return;
            }

            let $elem = document.getElementById(this.containerID);
            if ($elem) {
                if (document.fullscreenElement) {
                    document.exitFullscreen();
                } else if ($elem.requestFullscreen) {
                    $elem.requestFullscreen();
                } else {
                    window.alert("Le mode plein écran n'est pas supporté par votre navigateur.");
                }
            }
        },

        volumeChanged() {
            if (this.volumeDebounce !== null) {
                clearTimeout(this.volumeDebounce);
            }
            this.volumeDebounce = setTimeout(() => {
                this.$emit('set-volume', this.username, this.volume);
            }, 200);
        },

        // Show info about the watermark when the corner one is clicked.
        showWatermarkInfo() {
            this.$emit('modal-alert', {
                icon: "fa-solid fa-qrcode",
                title: "Que sont les QR codes sur les webcams ?",
                message: "Ce QR code est une fonctionnalité de sécurité pour dissuader les gens d'enregistrer l'écran des webcams sur le chat. " +
                    "Le QR code contient le nom d'utilisateur du spectateur actuel, le nom du site web et la date/heure actuelle. " +
                    "L'idée est que si une webcam est enregistrée puis diffusée en ligne, le QR code permettrait de relier directement la vidéo à la personne qui l'a enregistrée, et quand.\n\n" +
                    "Remarque : si ce risque ne vous concerne pas, vous pouvez retirer le QR code de *votre propre* vidéo afin qu'il ne gêne pas l'action lorsque d'autres personnes regardent votre caméra. Ce paramètre se trouve dans les paramètres du chat, onglet 'Webcam'.",
                backgroundDismissable: true,
            });
        },

        // Mutation Observer to detect if the watermark is tampered with by the viewer.
        initWatermarkObserver() {
            const $container = this.$refs.videoContainer,
                $layer = this.$refs.watermarkLayer,
                $watermark = this.$refs.watermarkImage,
                $smallImage = this.$refs.watermarkSmallImage;

            if (!$container || !$watermark || !$smallImage) return;

            // Helper function to check CSS properties that may hide the element visually without deleting it from the page.
            const isInvisible = (elem) => {
                let style = window.getComputedStyle(elem);
                return style.display === "none" || style.visibility === "hidden" || parseFloat(style.opacity) < 0.02;
            };
            const isImageMissing = (elem) => {
                try {
                    let src = elem.src;
                    if (src.indexOf("data:image/svg") !== 0) {
                        return true;
                    }
                } catch (e) {
                    console.error("isImageMissing", elem, e);
                }
                return false;
            }

            this.observer = new MutationObserver(() => {
                const stillPresent = $container.contains($layer) && $container.contains($watermark) && $container.contains($smallImage);
                if (!stillPresent) {
                    this.onWatermarkRemoved();
                }

                // Check for CSS hacking: this looks at the computed style, so only check if the top-level VideoFeed is not hidden.
                // TODO: this may generate false positives on some browsers, more testing needed.
                try {
                    if ($container.style.display !== "none") {
                        if (isInvisible($watermark) || isInvisible($smallImage) || isInvisible($layer)) {
                            console.error("Watermark CSS appears hidden");
                            // this.onWatermarkRemoved();
                        } else if (isImageMissing($watermark) || isImageMissing($smallImage)) {
                            console.error("Watermark img src appears missing");
                            // this.onWatermarkRemoved();
                        }
                    }
                } catch(e) {
                    console.error("checking container:", e);
                }
            });

            this.observer.observe($container, {
                childList: true,
                subtree: true,
            });
        },
        onWatermarkRemoved() {
            this.$emit('watermark-removed', this.username);
        },
        initWindowResizer() {
            window.addEventListener('resize', this.onWindowResized);
        },
        onWindowResized() {
            if (this.windowResizeTimer !== null) {
                clearTimeout(this.windowResizeTimer);
            }

            this.windowResizeCooldown = true;
            this.windowResizeTimer = setTimeout(() => {
                this.windowResizeCooldown = false;
                this.windowResizetimer = null;
            }, 1000);
        },
    }
}
</script>

<template>
    <div class="feed" ref="videoContainer" :id="containerID" :class="{
        'popped-out': poppedOut,
        'popped-in': !poppedOut,
    }" @mouseover="mouseOver = true" @mouseleave="mouseOver = false">
        <video
            :id="videoID"
            autoplay
            disablepictureinpicture
            playsinline
            oncontextmenu="return false;"
            :muted="localVideo"
            :class="{'invisible': windowResizeCooldown}"></video>

        <!-- Watermark layer -->
        <div :class="{'is-no-watermark': isNoWatermark}" ref="watermarkLayer">
            <img :src="watermarkImage" class="watermark" ref="watermarkImage" oncontextmenu="return false">
            <img :src="watermarkImage" class="corner-watermark seethru invert-color" ref="watermarkSmallImage" @click="showWatermarkInfo()" oncontextmenu="return false">
        </div>

        <!-- Caption -->
        <div class="caption" :class="textColorClass">
            <i class="fa fa-microphone-slash mr-1 has-text-grey" v-if="isSourceMuted"></i>
            <a href="#" @click.prevent="openProfile" :class="textColorClass">{{ username }}</a>
            <i class="fa fa-people-arrows ml-1 has-text-grey is-size-7" :title="username + ' is watching your camera too'"
                v-if="isWatchingMe"></i>

            <!-- Frozen stream detection -->
            <a class="fa fa-mountain ml-1" href="#" v-if="!localVideo && isFrozen" style="color: #00FFFF"
                @click.prevent="reopenVideo()" title="Vidéo figée détectée !"></a>

            <!-- Is speaking -->
            <span v-if="isSpeaking" class="ml-1" title="Parle">
                <i class="fa fa-volume-high has-text-info"></i>
            </span>
        </div>

        <!-- Close button (others' videos only) -->
        <div class="close" v-if="!localVideo" :class="{'seethru': !mouseOver}">
            <a href="#" class="button is-small is-danger is-outlined px-2" title="Fermer la vidéo" @click.prevent="closeVideo()">
                <i class="fa fa-close"></i>
            </a>
        </div>

        <!-- Controls -->
        <div class="controls">
            <!-- Mute Button -->
            <button type="button" :class="muteButtonClass"
                @click="muteVideo()">
                <i class="fa" :class="muteIconClass"></i>
            </button>

            <!-- Pop-out Video -->
            <button type="button" class="button is-small is-light is-outlined p-2 ml-2" title="Détacher la vidéo"
                :class="{'seethru': !mouseOver}"
                @click="popoutVideo()">
                <i class="fa fa-up-right-from-square"></i>
            </button>

            <!-- Full screen. -->
            <button type="button" class="button is-small is-light is-outlined p-2 ml-2" title="Passer en plein écran"
                :class="{'seethru': !mouseOver}"
                @click="fullscreen()">
                <i class="fa fa-expand"></i>
            </button>
        </div>

        <!-- Volume slider -->
        <div class="volume-slider" v-show="!localVideo && !isMuted && mouseOver">
            <Slider v-model="volume" color="#00FF00" track-color="#006600" :min="0" :max="100" :step="1" :height="7"
                orientation="vertical" @change="volumeChanged">

            </Slider>
        </div>
    </div>
</template>

<style scoped>
.volume-slider {
    position: absolute;
    left: 18px;
    top: 30px;
    bottom: 44px;
}

/* A background image behind video elements in case they don't load properly */
video {
    background-image: url(/static/img/connection-error.png);
    background-position: center center;
    background-repeat: no-repeat;
}

/* Translucent controls until mouse over */
.seethru {
    opacity: 0.4;
}
.invisible {
    visibility: hidden;
}

/* Watermark image */
.is-no-watermark {
    display: none;
}
.watermark {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    margin: auto;
    width: 40%;
    height: 40%;
    opacity: 0.02;
    animation-name: subtle-pulsate;
    animation-duration: 20s;
    animation-iteration-count: infinite;
}
.corner-watermark {
    position: absolute;
    right: 4px;
    bottom: 4px;
    width: 20%;
    min-width: 32px;
    min-height: 32px;
    max-height: 20%;
    cursor: pointer;
}
.invert-color {
    filter: invert(100%);
}

/* Animate the primary watermark to pulsate in opacity */
@keyframes subtle-pulsate {
    0% { opacity: 0.02; filter: invert(0%); }
    25% { opacity: 0.05; }
    50% { opacity: 0.02; }
    75% { opacity: 0.05; }
    90% { opacity: 0.06 }
    91% { opacity: 0.08; filter: invert(0%); }
    95% { opacity: 0.08; filter: invert(100%); }
    99% { opacity: 0.06; filter: invert(0%); }
    100% { opacity: 0.02; }
}
</style>
