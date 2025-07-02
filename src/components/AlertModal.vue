<script>
export default {
    props: {
        visible: Boolean,
        isConfirm: Boolean,
        title: String,
        icon: String,
        message: String,
        backgroundDismissable: Boolean,
    },
    data() {
        return {
            username: '',
        };
    },
    mounted() {
        window.addEventListener('keyup', (e) => {
            if (!this.visible) return;

            if (e.key === 'Enter') {
                return this.callback();
            }

            if (e.key == 'Escape') {
                return this.close();
            }
        })
    },
    methods: {
        callback() {
            this.$emit('close');
            this.$emit('callback');
        },
        close() {
            this.$emit('close');
        },
        backgroundClicked() {
            if (this.backgroundDismissable) {
                this.close();
            }
        },
    }
}
</script>

<template>
    <div class="modal" :class="{ 'is-active': visible }">
        <div class="modal-background" @click="backgroundClicked()"></div>

        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-info">
                    <p class="card-header-title">
                        <i v-if="icon" :class="icon" class="mr-2"></i>
                        {{ title }}
                    </p>
                    <button class="delete mr-3 mt-3" aria-label="close" @click.prevent="close"></button>
                </header>

                <div class="card-content">
                    <form @submit.prevent="callback()">

                        <p class="literal mb-4">{{ message }}</p>

                        <div class="columns is-centered is-mobile">
                            <div class="column is-narrow">
                                <button type="submit"
                                    class="button is-success px-5">
                                    OK
                                </button>
                                <button v-if="isConfirm"
                                    type="button"
                                    class="button is-link ml-3 px-5"
                                    @click="close">
                                    Annuler
                                </button>
                            </div>
                        </div>
                    </form>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
    .modal {
        /* a high priority modal over other modals. note: bulma's default z-index is 40 for modals */
        z-index: 42;
    }

    .literal {
        white-space: pre-wrap;
    }
</style>
