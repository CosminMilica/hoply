import type { Cv } from './api/cv.schema'

/** "claude-sonnet-5-5" as "Sonnet 5.5"; any other name as it is. */
export function modelLabel(model: string): string {
  const parts = model.match(/^claude-([a-z]+)-(\d+)(?:-(\d+))?/)
  if (!parts) return model
  const name = parts[1]!.charAt(0).toUpperCase() + parts[1]!.slice(1)
  return `${name} ${parts[2]}${parts[3] ? `.${parts[3]}` : ''}`
}

const time = (value: string) => new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })

const RULES_REASONS: Record<Exclude<Cv['rules_reason'], ''>, string> = {
  off: 'The service is set to write with rules.',
  no_token: 'Claude is not connected yet.',
  login_needed: 'Claude needs to be connected again.',
  usage_limit: 'Claude reached its usage limit.',
  claude_failed: 'Claude could not answer this time.',
}

/** Why a cover letter could not be written. */
export function letterFailure(letter: { reason: string, limit_until: string | null }): string {
  if (letter.reason === 'off') return 'The service is set to write without Claude, and a letter needs Claude.'
  const reason = RULES_REASONS[letter.reason as keyof typeof RULES_REASONS] ?? RULES_REASONS.claude_failed
  const back = letter.reason === 'usage_limit' && letter.limit_until ? ` It is back at ${time(letter.limit_until)}.` : ''
  return `${reason}${back}`
}

/** Who wrote the CV, and why the rules did when Claude did not. */
export function authorshipLine(cv: Pick<Cv, 'written_by' | 'model' | 'refused' | 'rules_reason' | 'limit_until'>): string {
  if (cv.written_by === 'claude') {
    const by = cv.model ? `Written by Claude (${modelLabel(cv.model)}) from your profile.` : 'Written by Claude from your profile.'
    if (!cv.refused) return by
    return `${by} ${cv.refused === 1 ? 'One sentence that went' : `${cv.refused} sentences that went`} beyond your profile ${cv.refused === 1 ? 'was' : 'were'} rewritten by rules.`
  }
  if (cv.written_by !== 'rules') return ''
  const reason = cv.rules_reason ? RULES_REASONS[cv.rules_reason] : ''
  const back = cv.rules_reason === 'usage_limit' && cv.limit_until ? ` It is back at ${time(cv.limit_until)}.` : ''
  return `Written without AI. ${reason}${back}`.trim()
}
