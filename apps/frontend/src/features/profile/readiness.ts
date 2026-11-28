import { t } from '@/features/common/i18n'
import type { Experience, Profile } from './api/profile.schema'

export interface ReadinessCheck {
  ok: boolean
  label: string
}

export interface Readiness {
  percent: number
  hint: string
  checks: ReadinessCheck[]
}

const filled = (value: string) => value.trim().length > 0

/** A role the tailoring can work with: a start month, at least three tools and two highlights. */
export function isRichRole(role: Experience): boolean {
  return role.tools.length >= 3 && role.bullets.length >= 2 && filled(role.start)
}

/** How far the profile is from a good tailored CV, with its checks and hint in the app's language. */
export function profileReadiness(profile: Profile): Readiness {
  const words = t.value.profile.readiness
  const label = words.checks
  const bullets = [...profile.experience, ...profile.projects].flatMap(entry => entry.bullets)
  const checks: ReadinessCheck[] = [
    { ok: filled(profile.name), label: label.name },
    { ok: filled(profile.headline), label: label.headline },
    { ok: filled(profile.about), label: label.about },
    { ok: filled(profile.email), label: label.email },
    { ok: filled(profile.phone), label: label.phone },
    { ok: filled(profile.github) || filled(profile.linkedin), label: label.link },
    { ok: filled(profile.city) && filled(profile.country), label: label.place },
    { ok: filled(profile.work_authorization), label: label.workAuthorization },
    { ok: profile.skills.length >= 5, label: label.skills },
    { ok: profile.experience.some(isRichRole), label: label.richRole },
    { ok: bullets.some(bullet => filled(bullet.result)), label: label.result },
    { ok: profile.education.some(item => filled(item.level)), label: label.education },
    { ok: profile.languages.length >= 1, label: label.language },
  ]
  const percent = Math.round((checks.filter(check => check.ok).length / checks.length) * 100)
  const next = checks.find(check => !check.ok)
  return { percent, checks, hint: next ? words.next(next.label) : words.allSet }
}

/**
 * How complete a profile must be before crashr asks Claude anything for it (questions, a CV, a letter): the cv service
 * keeps the same rule (cvdoc.ReadyForClaude), so the app never offers what the service would refuse.
 */
export const READY_PERCENT = 60

/** Whether the profile tells what the person did: a role or a project with a highlight. */
export function hasWork(profile: Profile): boolean {
  return [...profile.experience, ...profile.projects].some(entry => entry.bullets.some(bullet => filled(bullet.text)))
}

/** Whether the profile gives Claude enough to work with: READY_PERCENT complete and some work. */
export function readyForClaude(profile: Profile): boolean {
  return profileReadiness(profile).percent >= READY_PERCENT && hasWork(profile)
}

/** "Andrei Popescu" gives "AP"; one word gives its first letter. */
export function initialsOf(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean)
  if (!words.length) return ''
  const first = words[0]!.charAt(0)
  const last = words.length > 1 ? words[words.length - 1]!.charAt(0) : ''
  return (first + last).toUpperCase()
}
