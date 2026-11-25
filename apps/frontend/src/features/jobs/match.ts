import type { Profile } from '@/features/profile/api/profile.schema'
import { yearsOfExperience } from '@/features/profile/format'
import { JOB_LEVELS, JOB_WORK_MODES, type Job } from './api/jobs.schema'

type Requirements = Pick<Job, 'must_have' | 'nice_to_have'>

export interface JobMatch {
  matched: string[]
  missing: string[]
  score: number
  verdict: string
}

/** Every requirement of the job: the must-haves, then the nice-to-haves. */
export function requirementsOf(job: Requirements): string[] {
  return [...job.must_have, ...job.nice_to_have]
}

/** How the job's requirements line up with a set of skills (case-insensitive); a must-have counts twice a nice-to-have. */
export function matchJob(job: Requirements, skills: string[]): JobMatch {
  const have = new Set(skills.map(skill => skill.toLowerCase()))
  const nice = new Set(job.nice_to_have.map(item => item.toLowerCase()))
  const all = requirementsOf(job)
  const weight = (item: string) => (nice.has(item.toLowerCase()) ? 1 : 2)
  const matched = all.filter(item => have.has(item.toLowerCase()))
  const missing = all.filter(item => !have.has(item.toLowerCase()))
  const total = all.reduce((sum, item) => sum + weight(item), 0)
  const got = matched.reduce((sum, item) => sum + weight(item), 0)
  const score = total ? Math.round((got / total) * 100) : 0
  return { matched, missing, score, verdict: verdictFor(score) }
}

const escape = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** The keywords a text names as whole words, ignoring case. */
export function keywordsIn(text: string, keywords: string[]): string[] {
  return keywords.filter(keyword => new RegExp(`\\b${escape(keyword)}\\b`, 'i').test(text))
}

export function verdictFor(score: number): string {
  if (score >= 80) return 'Strong match. You already have most of what they ask for.'
  if (score >= 60) return 'Good match, with a couple of things to pick up.'
  if (score >= 40) return 'Partial match. Worth a try if the role excites you.'
  return 'Light match, in a new direction for you.'
}

const plain = (value: string) => value.normalize('NFD').replace(/\p{Diacritic}/gu, '').trim().toLowerCase()
const modeLabel = (mode: string) => JOB_WORK_MODES[mode as keyof typeof JOB_WORK_MODES] ?? ''

/**
 * Why the job may not work for the person, from their preferences: a work mode they do not want, or a
 * city they do not live in while they are not open to relocation. Empty when nothing stands out.
 */
export function knockoutFor(job: Pick<Job, 'city' | 'work_mode'>, profile: Pick<Profile, 'city' | 'work_modes' | 'open_to_relocation'>): string {
  const far = Boolean(job.city && profile.city.trim() && plain(job.city) !== plain(profile.city) && !profile.open_to_relocation)
  const mode = job.work_mode
  if (mode && profile.work_modes.length && !profile.work_modes.includes(mode) && !(mode === 'onsite' && !far)) {
    const wanted = profile.work_modes.map(item => modeLabel(item).toLowerCase()).join(' or ')
    return `${modeLabel(mode)}${job.city ? ` in ${job.city}` : ''}. Your profile asks for ${wanted} work${far ? ', and you are not open to relocation.' : '.'}`
  }
  if (far && mode !== 'remote') {
    return `Based in ${job.city}${mode === 'hybrid' ? ', hybrid' : ''}. You live in ${profile.city} and are not open to relocation.`
  }
  return ''
}

/** "Asks for 4+ years of experience. You have about 6." Empty when the job asks for no number. */
export function yearsLine(job: Pick<Job, 'years_required'>, profile: Pick<Profile, 'experience'>, now = new Date()): string {
  if (!job.years_required) return ''
  return `Asks for ${job.years_required}+ years of experience. You have about ${yearsOfExperience(profile, now)}.`
}

const hostOf = (url: string) => url.replace(/^https?:\/\//i, '').split('/')[0] ?? ''

export function sourceLine(job: Pick<Job, 'sample' | 'source' | 'url' | 'added_by_me'>): string {
  if (job.added_by_me || job.source === 'link' || job.source === 'text') return `Added by you · ${job.source === 'link' ? 'from a link' : 'from pasted text'}`
  if (job.sample) return 'Sample listing, not a real posting'
  return job.url ? `via ${hostOf(job.url)}` : 'From the catalog'
}

/** "Iași, hybrid", from the city and the work mode. */
export function placeOf(job: Pick<Job, 'city' | 'work_mode'>): string {
  return [job.city, modeLabel(job.work_mode).toLowerCase()].filter(Boolean).join(', ') || 'Location not given'
}

export function levelLabel(level: string): string {
  return JOB_LEVELS[level as keyof typeof JOB_LEVELS] ?? ''
}

export function tagline(job: Pick<Job, 'areas' | 'level'>): string {
  return [job.areas.join(', '), levelLabel(job.level)].filter(Boolean).join(' · ')
}

/** The job's requirements as the lines a recruiter would write. */
export function lookFor(job: Requirements): string[] {
  return [...job.must_have.map(item => `Experience with ${item}.`), ...job.nice_to_have.map(item => `${item} is a plus.`)]
}

/** Where and how the person wants to work, as Find jobs reminds them; empty without a place or a work mode. */
export function marketNote(profile: Pick<Profile, 'city' | 'country' | 'work_modes' | 'open_to_relocation'>): string {
  const place = [profile.city, profile.country].map(part => part.trim()).filter(Boolean).join(', ')
  const modes = profile.work_modes.map(mode => modeLabel(mode).toLowerCase()).filter(Boolean).join(' or ')
  if (!place && !modes) return ''
  return `From your profile: ${[place, modes, profile.open_to_relocation ? 'open to relocation' : 'not relocating'].filter(Boolean).join(' · ')}.`
}

export { hostOf }
