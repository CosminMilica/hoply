import { locale, type Locale } from './locale'

/** The BCP 47 tag each language formats dates and numbers with. */
const TAGS: Record<Locale, string> = { en: 'en-GB', ro: 'ro-RO' }

export const languageTag = () => TAGS[locale.value]

export function formatDate(value: Date, options: Intl.DateTimeFormatOptions) {
  return value.toLocaleDateString(languageTag(), options)
}

export function formatTime(value: Date) {
  return value.toLocaleTimeString(languageTag(), { hour: '2-digit', minute: '2-digit' })
}

/** An English count: "1 job", "2 jobs". */
export function countEn(n: number, one: string, other: string) {
  return `${n} ${n === 1 ? one : other}`
}

/** A Romanian count: "1 job", "2 joburi", "20 de joburi"; "de" comes after 20 to 99 and their hundreds. */
export function countRo(n: number, one: string, other: string) {
  if (n === 1) return `1 ${one}`
  const rest = n % 100
  return n !== 0 && (rest === 0 || rest >= 20) ? `${n} de ${other}` : `${n} ${other}`
}
