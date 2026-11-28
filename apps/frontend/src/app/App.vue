<script setup lang="ts">
import { computed } from 'vue'
import { en, ro } from '@nuxt/ui/locale'
import ConfirmHost from '@/shared/components/ConfirmHost.vue'
import { locale } from '@/features/common/i18n'
import { cspNonce } from './providers/csp'
import { useQueryErrorToasts } from './providers/query-client'

// UApp provides toasts, tooltips and overlays; its nonce reaches the <style>
// elements Reka UI inserts (scrollbar hiding, the splitter's drag cursor).
const nonce = cspNonce()

useQueryErrorToasts()

// Nuxt UI's own words (close buttons, empty selects) follow the app's language.
const uiLocale = computed(() => (locale.value === 'ro' ? ro : en))
</script>

<template>
  <UApp :toaster="{ position: 'bottom-right', expand: true }" :nonce="nonce" :locale="uiLocale">
    <RouterView />
    <ConfirmHost />
  </UApp>
</template>
