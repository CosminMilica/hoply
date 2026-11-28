import { ref } from 'vue'

/** The languages the app speaks; the profile service keeps the person's as `app_language`. */
export const LOCALES = ['en', 'ro'] as const
export type Locale = (typeof LOCALES)[number]

const KEY = 'hoply.locale'

export const isLocale = (value: unknown): value is Locale => LOCALES.includes(value as Locale)

function remembered(): Locale {
  try {
    const value = localStorage.getItem(KEY)
    return isLocale(value) ? value : 'en'
  }
  catch {
    return 'en'
  }
}

/** The language the app shows; remembered on this device, so the next visit starts in it before the profile loads. */
export const locale = ref<Locale>(remembered())
markPage(locale.value)

/** Tells the browser the page's language; nothing outside a browser (test helpers that run in Node). */
function markPage(language: Locale) {
  if (typeof document !== 'undefined') document.documentElement.lang = language
}

/** Switches the app to a language; anything else is ignored. */
export function setLocale(next: unknown) {
  if (!isLocale(next)) return
  locale.value = next
  markPage(next)
  try {
    localStorage.setItem(KEY, next)
  }
  catch {
    // A browser that keeps nothing still switches; the next visit starts in English.
  }
}
