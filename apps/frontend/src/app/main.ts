// First import: it configures zod before any schema is built.
import { cspNonce } from './providers/csp'
import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { VueQueryPlugin } from '@tanstack/vue-query'
import ui from '@nuxt/ui/vue-plugin'

import App from './App.vue'
import router from './router'
import { auth } from './auth'
import { queryClient } from './providers/query-client'
import { createAppHead } from './providers/head'
import { configureApiClient } from '@/shared/api/client'
import { preferDarkColorMode } from '@/custom/theme/color-mode'

import './styles/main.css'

async function bootstrap() {
  // Start MSW in development when enabled
  if (import.meta.env.DEV && import.meta.env.VITE_MSW_ENABLED === 'true') {
    // The mocked services (jobs, applications, cv); everything else, the profile
    // service among it, passes through to the network.
    const { worker } = await import('@/custom/mocks/browser')
    await worker.start({ onUnhandledFrame: 'bypass', quiet: true })
  }

  const app = createApp(App)

  app.use(createPinia())
  app.use(router)
  app.use(VueQueryPlugin, { queryClient })
  // Nuxt UI renders its palette through the head installed before it.
  app.use(createAppHead(cspNonce()))
  preferDarkColorMode()
  app.use(ui)

  // The authentication contract (./auth.ts) starts here, and the app does not wait for it:
  // the guards wait (a bounded time), so a sign-in that never answers ends on the
  // auth_stuck page instead of a blank one.
  auth.init?.().catch((error: unknown) => {
    console.error('The authentication contract failed to start.', error)
  })
  configureApiClient({
    getToken: async () => null,
    onUnauthorized: () => {},
  })

  app.mount('#app')
}

bootstrap()
