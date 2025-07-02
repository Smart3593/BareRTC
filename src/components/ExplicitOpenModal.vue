<script>
export default {
    props: {
        visible: Boolean,
        user: Object,
    },
    data() {
        return {
            dontShowAgain: false,
        };
    },
    mounted() {
        window.addEventListener('keyup', (e) => {
            if (!this.visible) return;

            if (e.key === 'Enter') {
                return this.accept();
            }

            if (e.key == 'Escape') {
                return this.cancel();
            }
        })
    },
    methods: {
        accept() {
            if (this.dontShowAgain) {
                this.$emit('dont-show-again');
            }
            this.$emit('accept');
        },
        cancel() {
            this.$emit('cancel');
        },
    }
}
</script>

<template>
    <!-- NSFW Modal: before user views a NSFW camera the first time -->
    <div class="modal" :class="{ 'is-active': visible }">
        <div class="modal-background"></div>
        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-info">
                    <p class="card-header-title">Cette caméra peut contenir du contenu explicite</p>
                </header>
                <div class="card-content">
                    <p class="block">
                        Cette caméra a été marquée comme "Explicite/<abbr title="Not Safe For Work">NSFW</abbr>" et peut contenir des scènes de sexualité. Si vous ne souhaitez pas voir cela, cherchez les caméras avec une icône bleue plutôt que les rouges.
                    </p>

                    <div class="field">
                        <label class="checkbox">
                            <input type="checkbox" v-model="dontShowAgain">
                            Ne plus afficher ce message
                        </label>
                    </div>

                    <div class="field">
                        <div class="control has-text-centered">
                            <button type="button" class="button is-link mr-4"
                                @click="accept()">
                                Ouvrir la webcam
                            </button>
                            <button type="button" class="button"
                                @click="cancel()">
                                Annuler
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
</style>
