<script setup>
import WelcomeItem from './WelcomeItem.vue'
import DocumentationIcon from './icons/IconDocumentation.vue'
import ToolingIcon from './icons/IconTooling.vue'
import EcosystemIcon from './icons/IconEcosystem.vue'
import CommunityIcon from './icons/IconCommunity.vue'
import SupportIcon from './icons/IconSupport.vue'
</script>

<template>
  <WelcomeItem>
    <template #icon>
      <DocumentationIcon />
    </template>
    <template #heading>Documentation</template>

    La <a href="https://vuejs.org/" target="_blank" rel="noopener">documentation officielle</a> de Vue vous fournit toutes les informations nécessaires pour commencer.
  </WelcomeItem>

  <WelcomeItem>
    <template #icon>
      <ToolingIcon />
    </template>
    <template #heading>Outils</template>

    Ce projet est servi et assemblé avec <a href="https://vitejs.dev/guide/features.html" target="_blank" rel="noopener">Vite</a>. L'environnement de développement recommandé est <a href="https://code.visualstudio.com/" target="_blank" rel="noopener">VSCode</a> + <a href="https://github.com/johnsoncodehk/volar" target="_blank" rel="noopener">Volar</a>. Si vous avez besoin de tester vos composants et pages web, consultez <a href="https://www.cypress.io/" target="_blank" rel="noopener">Cypress</a> et <a href="https://on.cypress.io/component" target="_blank">Cypress Component Testing</a>.

    <br />

    Plus d'instructions sont disponibles dans <code>README.md</code>.
  </WelcomeItem>

  <WelcomeItem>
    <template #icon>
      <EcosystemIcon />
    </template>
    <template #heading>Écosystème</template>

    Obtenez des outils et bibliothèques officiels pour votre projet : <a href="https://pinia.vuejs.org/" target="_blank" rel="noopener">Pinia</a>, <a href="https://router.vuejs.org/" target="_blank" rel="noopener">Vue Router</a>, <a href="https://test-utils.vuejs.org/" target="_blank" rel="noopener">Vue Test Utils</a>, et <a href="https://github.com/vuejs/devtools" target="_blank" rel="noopener">Vue Dev Tools</a>. Si vous avez besoin de plus de ressources, nous vous suggérons de visiter <a href="https://github.com/vuejs/awesome-vue" target="_blank" rel="noopener">Awesome Vue</a>.
  </WelcomeItem>

  <WelcomeItem>
    <template #icon>
      <CommunityIcon />
    </template>
    <template #heading>Communauté</template>

    Vous êtes bloqué ? Posez votre question sur <a href="https://chat.vuejs.org" target="_blank" rel="noopener">Vue Land</a>, notre serveur Discord officiel, ou <a href="https://stackoverflow.com/questions/tagged/vue.js" target="_blank" rel="noopener">StackOverflow</a>. Vous devriez aussi vous abonner à <a href="https://news.vuejs.org" target="_blank" rel="noopener">notre newsletter</a> et suivre le compte officiel <a href="https://twitter.com/vuejs" target="_blank" rel="noopener">@vuejs</a> sur Twitter pour les dernières nouvelles du monde Vue.
  </WelcomeItem>

  <WelcomeItem>
    <template #icon>
      <SupportIcon />
    </template>
    <template #heading>Soutenir Vue</template>

    En tant que projet indépendant, Vue compte sur le soutien de la communauté pour sa pérennité. Vous pouvez nous aider en <a href="https://vuejs.org/sponsor/" target="_blank" rel="noopener">devenant sponsor</a>.
  </WelcomeItem>
</template>
