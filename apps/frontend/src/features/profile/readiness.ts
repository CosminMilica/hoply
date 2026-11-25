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

export function profileReadiness(profile: Profile): Readiness {
  const bullets = [...profile.experience, ...profile.projects].flatMap(entry => entry.bullets)
  const checks: ReadinessCheck[] = [
    { ok: filled(profile.name), label: 'your name' },
    { ok: filled(profile.headline), label: 'a headline' },
    { ok: filled(profile.about), label: 'a short intro' },
    { ok: filled(profile.email), label: 'your email' },
    { ok: filled(profile.phone), label: 'your phone number' },
    { ok: filled(profile.github) || filled(profile.linkedin), label: 'a GitHub or LinkedIn link' },
    { ok: filled(profile.city) && filled(profile.country), label: 'your city and country' },
    { ok: filled(profile.work_authorization), label: 'your work authorization' },
    { ok: profile.skills.length >= 5, label: 'at least 5 skills' },
    { ok: profile.experience.some(isRichRole), label: 'a role with a start month, tools and two highlights' },
    { ok: bullets.some(bullet => filled(bullet.result)), label: 'a highlight with a result' },
    { ok: profile.education.some(item => filled(item.level)), label: 'your education with its level' },
    { ok: profile.languages.length >= 1, label: 'a language' },
  ]
  const percent = Math.round((checks.filter(check => check.ok).length / checks.length) * 100)
  const next = checks.find(check => !check.ok)
  return { percent, checks, hint: next ? `Next up: add ${next.label}` : 'All set. Time to find some jobs.' }
}

/** "Andrei Popescu" gives "AP"; one word gives its first letter. */
export function initialsOf(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean)
  if (!words.length) return ''
  const first = words[0]!.charAt(0)
  const last = words.length > 1 ? words[words.length - 1]!.charAt(0) : ''
  return (first + last).toUpperCase()
}
