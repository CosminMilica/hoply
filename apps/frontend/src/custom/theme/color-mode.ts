import { useColorMode } from '@vueuse/core'

/** Dark unless the person chose otherwise; run before Nuxt UI's colour mode starts. */
export function preferDarkColorMode(): void {
  useColorMode({ initialValue: 'dark' })
}
