/**
 * Nuxt UI configuration: the project's half of the design system.
 * `vite.config.ts` passes these values to the Nuxt UI plugin. The palettes
 * hoply and ember are defined in tokens.css. Controls default to the compact
 * `md` size; a page's main action sets `lg` (44px minimum height).
 */
import type { NuxtUIOptions } from '@nuxt/ui/unplugin'

const touch = { lg: { base: 'min-h-11' }, xl: { base: 'min-h-12' } }

export const ui: NuxtUIOptions['ui'] = {
  colors: {
    primary: 'hoply',
    // Nuxt UI takes a Tailwind neutral here; dark mode's surfaces and text use ink (tokens.css).
    neutral: 'stone',
    error: 'ember',
  },
  button: {
    slots: { base: 'font-bold' },
    variants: { size: touch },
    compoundVariants: [
      {
        color: 'primary',
        variant: 'solid',
        class: 'bg-hoply-600 text-white hover:bg-hoply-700 active:bg-hoply-700 disabled:bg-hoply-600 aria-disabled:bg-hoply-600',
      },
      {
        color: 'error',
        variant: 'solid',
        class: 'text-ink-950',
      },
    ],
    defaultVariants: { size: 'md' },
  },
  input: { variants: { size: touch }, defaultVariants: { size: 'md' } },
  select: { variants: { size: touch }, defaultVariants: { size: 'md' } },
  textarea: { defaultVariants: { size: 'md' } },
}

export const theme: NuxtUIOptions['theme'] = {
  colors: ['primary', 'secondary', 'success', 'info', 'warning', 'error'],
}
