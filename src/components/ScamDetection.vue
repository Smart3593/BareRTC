<script>
export default {
    props: {
        username: String,
        message: String,
    },
    data() {
        return {
        };
    },
    computed: {
        // Message sans HTML tags, so we don't false positive on base64-encoded image data.
        filteredMessage() {
            return this.message.replace(/<(.|\n)+?>/g, "");
        },

        // Scam/spam detection and warning.
        maybeWhatsAppScam() {
            return this.filteredMessage.match(/whats\s*app/i);
        },
        maybePhoneNumberScam() {
            return this.filteredMessage.match(/\b(phone number|phone|digits|cell number|your number|ur number|text me)\b/i);
        },
        maybeOffPlatformScam() {
            return this.filteredMessage.match(/\b(telegram|signal|kik)\b/i);
        },
    },
    methods: {
    }
}
</script>

<template>
    <div v-if="maybeWhatsAppScam" class="notification is-danger is-light px-3 py-2 my-2">
        <strong class="has-text-danger">
            <i class="fa fa-exclamation-triangle mr-1"></i>
            Faites attention aux arnaques potentielles !
        </strong>
        Il semble que @{{ username }} parle de déplacer cette conversation vers <i class="fab fa-whatsapp"></i> WhatsApp.
        Si cela s'est produit dans les premiers messages, soyez prudent ! Il est bien connu que les escrocs essaient de déplacer la conversation vers une autre plateforme dès que possible pour échapper à la détection du site web.
        <br><br>
        <strong>Soyez particulièrement méfiant envers <i class="fab fa-whatsapp"></i> WhatsApp</strong> ou l'échange de numéros de téléphone. Les arnaqueurs peuvent faire <strong>beaucoup</strong> de dégâts avec seulement votre numéro de téléphone, par exemple en le saisissant sur un site de recherche de personnes et en obtenant beaucoup d'informations personnelles sur vous.
    </div>
    <div v-else-if="maybeOffPlatformScam" class="notification is-danger is-light px-3 py-2 my-2">
        <strong class="has-text-danger">
            <i class="fa fa-exclamation-triangle mr-1"></i>
            Faites attention aux arnaques potentielles !
        </strong>
        Il semble que @{{ username }} parle de déplacer cette conversation vers une autre plateforme de messagerie.
        Si cela s'est produit dans les premiers messages, soyez prudent ! Il est bien connu que les escrocs essaient de déplacer la conversation vers une autre plateforme dès que possible pour échapper à la détection du site web.
    </div>
    <div v-else-if="maybePhoneNumberScam" class="notification is-danger is-light px-3 py-2 my-2">
        <strong class="has-text-danger">
            <i class="fa fa-exclamation-triangle mr-1"></i>
            Faites attention aux arnaques potentielles !
        </strong>
        Il semble que @{{ username }} souhaite obtenir votre numéro de téléphone. Si cela s'est produit dans les premiers messages, soyez prudent ! Les arnaqueurs peuvent faire <strong>beaucoup</strong> de dégâts avec seulement votre numéro de téléphone, par exemple en le saisissant sur un site de recherche de personnes et en obtenant beaucoup d'informations personnelles sur vous.
    </div>
</template>

<style scoped>
</style>
