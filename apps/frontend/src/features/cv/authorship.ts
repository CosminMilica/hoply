import { formatTime, t } from '@/features/common/i18n'
import type { Cv } from './api/cv.schema'

/** "claude-sonnet-5-5" as "Sonnet 5.5"; any other name as it is. */
export function modelLabel(model: string): string {
  const parts = model.match(/^claude-([a-z]+)-(\d+)(?:-(\d+))?/)
  if (!parts) return model
  const name = parts[1]!.charAt(0).toUpperCase() + parts[1]!.slice(1)
  return `${name} ${parts[2]}${parts[3] ? `.${parts[3]}` : ''}`
}

const backAt = (reason: string, until: string | null) =>
  (reason === 'usage_limit' && until ? t.value.cv.authorship.backAt(formatTime(new Date(until))) : '')

const sentences = (...parts: string[]) => parts.filter(Boolean).join(' ')

/** Why a cover letter could not be written. */
export function letterFailure(letter: { reason: string, limit_until: string | null }): string {
  const words = t.value.cv.authorship
  if (letter.reason === 'off') return words.letterOff
  const reason = words.reasons[letter.reason as keyof typeof words.reasons] ?? words.reasons.claude_failed
  return sentences(reason, backAt(letter.reason, letter.limit_until))
}

/** Who wrote the CV, and why the rules did when Claude did not. */
export function authorshipLine(cv: Pick<Cv, 'written_by' | 'model' | 'refused' | 'rules_reason' | 'limit_until'>): string {
  const words = t.value.cv.authorship
  if (cv.written_by === 'claude') {
    const by = cv.model ? words.byModel(modelLabel(cv.model)) : words.by
    return sentences(by, cv.refused ? words.refused(cv.refused) : '')
  }
  if (cv.written_by !== 'rules') return ''
  return sentences(words.withoutAi, cv.rules_reason ? words.reasons[cv.rules_reason] : '', backAt(cv.rules_reason, cv.limit_until))
}
