<script>
export default {
    props: {
        visible: Boolean,
    },
    data() {
        return {
            username: '',
        };
    },
    methods: {
        signIn() {
            this.$emit('signIn', this.username);
        }
    }
}
</script>

<template>
    <div class="modal" :class="{ 'is-active': visible }">
        <div class="modal-background"></div>
        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-info">
                    <p class="card-header-title">Connexion</p>
                </header>
                <div class="card-content">
                    <form @submit.prevent="signIn()">

                        <div v-if="autoLogin" class="content">
                            <p>
                                Bienvenue sur <span v-html="config.branding"></span> ! Veuillez simplement cliquer sur le bouton "Entrer dans le chat" ci-dessous pour vous connecter. Votre nom d'utilisateur a été pré-rempli depuis le site web qui vous a envoyé ici.
                            </p>

                            <p>
                                Cette boîte de dialogue a été ajoutée à titre expérimental pour voir si cela aide les appareils iOS (iPads et iPhones) à se connecter au chat plus facilement, en vous faisant interagir avec la page avant qu'elle ne se connecte au serveur. Faites-nous savoir dans le chat si votre iPhone ou iPad peut se connecter de cette façon !
                            </p>
                        </div>

                        <div class="field">
                            <label class="label">Nom d'utilisateur</label>
                            <input class="input" v-model="username" placeholder="Nom d'utilisateur" autocomplete="off" autofocus
                                :disabled="autoLogin" required>
                        </div>

                        <div class="field">
                            <div class="control">
                                <button class="button is-link">Entrer dans le chat</button>
                            </div>
                        </div>

                    </form>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
</style>
