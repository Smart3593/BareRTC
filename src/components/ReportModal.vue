<script>
import MessageBox from './MessageBox.vue';

export default {
    props: {
        visible: Boolean,
        busy: Boolean,
        user: Object,
        message: Object,
    },
    components: {
        MessageBox,
    },
    data() {
        return {
            // Configuration
            reportClassifications: [
                "C'est du spam",
                "C'est abusif (raciste, homophobe, etc.)",
                "C'est malveillant (ex : lien vers un site malveillant, phishing)",
                "C'est illégal (ex : substances contrôlées, violence)",
                "C'est de la maltraitance d'enfants (CP, CSAM, pédophilie, etc.)",
                "Autre (veuillez décrire)",
            ],

            // Our settings.
            classification: "It's spam",
            comment: "",
        };
    },
    methods: {
        reset() {
            this.classification = this.reportClassifications[0];
            this.comment = "";
        },

        accept() {
            this.$emit('accept', {
                classification: this.classification,
                comment: this.comment,
            });
            this.reset();
        },
        cancel() {
            this.$emit('cancel');
            this.reset();
        },
    }
}
</script>

<template>
    <!-- Report Modal -->
    <div class="modal" :class="{ 'is-active': visible }">
        <div class="modal-background"></div>
        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-warning">
                    <p class="card-header-title has-text-dark">Signaler un message</p>
                </header>
                <div class="card-content">

                    <!-- Message preview we are reporting on -->
                    <MessageBox
                        :message="message"
                        :user="user"
                        :no-buttons="true"
                    ></MessageBox>

                    <div class="field mb-1">
                        <label class="label" for="classification">Catégorie du signalement :</label>
                        <div class="select is-fullwidth">
                            <select id="classification" v-model="classification" :disabled="busy">
                                <option v-for="i in reportClassifications" v-bind:key="i" :value="i">{{ i }}</option>
                            </select>
                        </div>
                    </div>

                    <div class="field">
                        <label class="label" for="reportComment">Commentaire :</label>
                        <textarea class="textarea" v-model="comment" :disabled="busy" cols="80"
                            rows="2" placeholder="Optionnel : décrivez le problème"></textarea>
                    </div>

                    <div class="field">
                        <div class="control has-text-centered">
                            <button type="button" class="button is-link mr-4" :disabled="busy"
                                @click="accept()">Envoyer le signalement</button>
                            <button type="button" class="button" @click="cancel()">Annuler</button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<style scoped>
</style>
