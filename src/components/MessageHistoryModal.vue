<script>
export default {
    props: {
        visible: Boolean,
        jwt: String, // caller's JWT token for authorization
    },
    data() {
        return {
            busy: false,

            sort: "newest",
            page: 1,
            pages: 0,
            count: 0,
            usernames: [],

            // Error messaging from backend
            error: null,
        };
    },
    watch: {
        visible() {
            if (this.visible) {
                this.refresh();
            } else {
                this.error = null;
                this.busy = false;
            }
        },
        sort() {
            this.page = 1;
            this.refresh();
        },
    },
    methods: {
        refresh() {
            this.busy = true;
            return fetch("/api/message/usernames", {
                method: "POST",
                mode: "same-origin",
                cache: "no-cache",
                credentials: "same-origin",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    "JWTToken": this.jwt,
                    "Sort": this.sort,
                    "Page": this.page,
                }),
            })
            .then((response) => response.json())
            .then((data) => {
                if (data.Error) {
                    this.error = data.Error;
                    return;
                }

                this.pages = data.Pages;
                this.count = data.Count;
                this.usernames = data.Usernames;
            }).catch(resp => {
                this.error = resp;
            }).finally(() => {
                this.busy = false;
            })
        },

        gotoPrevious() {
            this.page--;
            if (this.page < 1) {
                this.page = 1;
            }
            this.refresh();
        },
        gotoNext() {
            this.page++;
            if (this.page > this.pages) {
                this.page = this.pages;
            }
            if (this.page < 1) {
                this.page = 1;
            }
            this.refresh();
        },

        openChat(username) {
            this.$emit("open-chat", {
                username: username,
            });
        },

        cancel() {
            this.$emit("cancel");
        },
    },
}
</script>

<template>
    <!-- DM Username History Modal -->
    <div class="modal" :class="{ 'is-active': visible }">
        <div class="modal-background" @click="cancel()"></div>
        <div class="modal-content">
            <div class="card">
                <header class="card-header has-background-success">
                    <p class="card-header-title">Historique des messages privés</p>
                </header>
                <div class="card-content">
                    <div v-if="busy">
                        <i class="fa fa-spinner fa-spin mr-2"></i>
                        Chargement...
                    </div>
                    <div v-else-if="error" class="has-text-danger">
                        <i class="fa fa-exclamation-triangle mr-2"></i>
                        <strong class="has-text-danger">Erreur :</strong>
                        {{ error }}
                    </div>
                    <div v-else>
                        <p class="block">
                            {{ count }} nom d'utilisateur{{ count === 1 ? '' : 's' }} trouvé(s) avec qui vous avez discuté
                            (page {{ page }} sur {{ pages }}).
                        </p>

                        <!-- Pagination row -->
                        <div class="columns block is-mobile">
                            <div class="column">
                                <button type="button" class="button is-small mr-2"
                                    :disabled="page === 1"
                                    @click="gotoPrevious">
                                    Précédent
                                </button>
                                <button type="button" class="button is-small"
                                    :disabled="page >= pages"
                                    @click="gotoNext">
                                    Page suivante
                                </button>
                            </div>
                            <div class="column is-narrow">
                                <div class="select is-small">
                                    <select v-model="sort">
                                        <option value="newest">Le plus récent</option>
                                        <option value="oldest">Le plus ancien</option>
                                        <option value="a-z">Nom d'utilisateur (a-z)</option>
                                        <option value="z-a">Nom d'utilisateur (z-a)</option>
                                    </select>
                                </div>
                            </div>
                        </div>

                        <div class="columns block is-multiline">
                            <div class="column is-one-third is-clipped nowrap"
                                v-for="username in usernames"
                                v-bind:key="username">
                                <a href="#" @click.prevent="openChat(username); cancel()" class="truncate-text-line">
                                    <img src="/static/img/shy.png" class="mr-1" width="12" height="12">
                                    {{ username }}
                                </a>
                            </div>
                        </div>
                    </div>
                </div>
                <footer class="card-footer">
                    <a href="#" @click.prevent="cancel()" class="card-footer-item">
                        Fermer
                    </a>
                </footer>
            </div>
        </div>
    </div>
</template>

<style scoped>
.nowrap {
    white-space: nowrap;
}
</style>
