import browserCoverage from './scripts/browser-coverage.ts';
import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import ui from '@nuxt/ui/vite'
import { msw } from 'msw/vite'
import { theme, ui as uiConfig } from './src/custom/theme/nuxt-ui.config.ts'

export default defineConfig({
  plugins: [
    ...(process.env.FORGE_COVERAGE === 'true' ? [browserCoverage()] : []),
    // The dev server serves MSW's worker script (src/app/main.ts starts it); builds hold none.
    { ...msw({ mode: 'worker-only' }), apply: 'serve' },
    vue(),
    // Nuxt UI brings Tailwind CSS and the auto-import plugins with it; their
    // options are set here, the design system in src/custom/theme/.
    ui({
      ui: uiConfig,
      theme,
      autoImport: {
        imports: [
          'vue',
          'vue-router',
          'pinia',
          {
            '@tanstack/vue-query': [
              'useQuery',
              'useMutation',
              'useQueryClient',
              'useInfiniteQuery',
            ],
          },
        ],
        dts: 'src/auto-imports.d.ts',
        dirs: [
          'src/shared/composables',
          'src/shared/stores',
          'src/features/*/composables',
          'src/features/*/api',
        ],
        vueTemplate: true,
      },
      components: { dts: 'src/components.d.ts' },
      // Bundle every icon the sources name, so nothing is fetched at runtime.
      icon: { clientBundle: { scan: { globInclude: ['src/**/*.{vue,ts}'] } } },
      experimental: { componentDetection: true },
    }),
  ],
  // security.csp: Vite stamps this placeholder on its own tags and into a
  // csp-nonce meta tag; nginx replaces it with each response's nonce.
  html: { cspNonce: '__CSP_NONCE__' },
  define: {
    __DEV_VALIDATION__: JSON.stringify(process.env.NODE_ENV !== 'production'),
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  build: {
    chunkSizeWarningLimit: 800,
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            { name: 'vendor-vue', test: /node_modules\/(?:@vue|vue|vue-router|pinia)\// },
            { name: 'vendor-tanstack', test: /node_modules\/@tanstack\// },
            { name: 'vendor-nuxt-ui', test: /node_modules\/(?:@nuxt\/ui|reka-ui)\// },
            // security.csp: zod and the module that sets it jitless (main.ts's first
            // import) load as one chunk. Rolldown runs the chunks the entry imports
            // before the entry's own body; a schema one of them built would make zod
            // probe for eval, which the policy reports.
            { name: 'vendor-zod', test: /node_modules\/zod\/|\/src\/app\/providers\/csp\.ts$/ },
          ],
        },
      },
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api/profile': {
        target: 'http://localhost:5010',
        changeOrigin: true,
        rewrite: (path: string) => path.replace(/^\/api\/profile/, '/api'),
      },
      '/api/jobs': {
        target: 'http://localhost:5020',
        changeOrigin: true,
        rewrite: (path: string) => path.replace(/^\/api\/jobs/, '/api'),
      },
      '/api/applications': {
        target: 'http://localhost:5030',
        changeOrigin: true,
        rewrite: (path: string) => path.replace(/^\/api\/applications/, '/api'),
      },
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'lcov', 'html', 'json-summary'],
      reportsDirectory: './coverage',
      include: ['src/**/*.{ts,vue}'],
      exclude: [
        'src/test/**',
        'src/auto-imports.d.ts',
        'src/env.d.ts',
        '**/*.d.ts',
        '**/*.test.ts',
        '**/*.spec.ts',
      ],
    },
  },
})
