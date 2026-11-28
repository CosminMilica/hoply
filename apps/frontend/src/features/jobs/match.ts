import type { Profile } from '@/features/profile/api/profile.schema'
import { yearsOfExperience } from '@/features/profile/format'
import { JOB_LEVELS, JOB_WORK_MODES, type Job } from './api/jobs.schema'
import skillMap from './skillmap.json' with { type: 'json' }

type Requirements = Pick<Job, 'must_have' | 'nice_to_have'>

/** A requirement the person has not used but knows a close equivalent of (Argo CD for someone who uses FluxCD). */
export interface RelatedSkill {
  skill: string
  via: string[]
  /** The kind of tool both are ("GitOps" for Argo CD and FluxCD): the CV names it, never the skill itself. */
  category: string
}

export interface JobMatch {
  matched: string[]
  related: RelatedSkill[]
  missing: string[]
  score: number
  verdict: string
}

/** Every requirement of the job: the must-haves, then the nice-to-haves. */
export function requirementsOf(job: Requirements): string[] {
  return [...job.must_have, ...job.nice_to_have]
}

/** A name without case, spaces or punctuation: "Argo CD" and "argocd" fold alike. */
const fold = (name: string) => name.trim().toLowerCase().replace(/[^\p{L}\p{N}+#]+/gu, '')

const sameKey = new Map<string, string>()
for (const group of skillMap.same) {
  for (const name of group) sameKey.set(fold(name), fold(group[0]!))
}
/** One key per skill, whatever name it goes by. */
export const skillKey = (name: string) => sameKey.get(fold(name)) ?? fold(name)
const coverBy = new Map(Object.entries(skillMap.covers).map(([umbrella, tools]) => [skillKey(umbrella), tools.map(skillKey)]))
const families = skillMap.close.map(family => ({ category: family.category, keys: family.tools.map(skillKey) }))

/**
 * How the job's requirements line up with a set of skills (skillmap.json, the cv service's copy): a requirement
 * is matched when the person has it under any of its names or has a tool that proves it (GitHub Actions proves
 * CI/CD), related when they have a close equivalent, missing otherwise. A must-have counts twice a
 * nice-to-have; a related one counts half.
 */
export function matchJob(job: Requirements, skills: string[]): JobMatch {
  const have = new Map<string, string>()
  for (const skill of skills) {
    const key = skillKey(skill)
    if (key && !have.has(key)) have.set(key, skill)
  }
  const nice = new Set(job.nice_to_have.map(item => item.toLowerCase()))
  const weight = (item: string) => (nice.has(item.toLowerCase()) ? 2 : 4)
  const matched: string[] = []
  const related: RelatedSkill[] = []
  const missing: string[] = []
  let total = 0
  let got = 0
  for (const item of requirementsOf(job)) {
    const key = skillKey(item)
    total += weight(item)
    const covered = have.has(key) || (coverBy.get(key) ?? []).some(tool => have.has(tool))
    const close = families.filter(family => family.keys.includes(key))
      .map(family => ({ category: family.category, via: family.keys.filter(member => member !== key && have.has(member)).map(member => have.get(member)!) }))
      .filter(family => family.via.length)
    const via = [...new Set(close.flatMap(family => family.via))]
    if (covered) {
      matched.push(item)
      got += weight(item)
    }
    else if (via.length) {
      related.push({ skill: item, via, category: close[0]!.category })
      got += weight(item) / 2
    }
    else {
      missing.push(item)
    }
  }
  const score = total ? Math.round((got / total) * 100) : 0
  return { matched, related, missing, score, verdict: verdictFor(score) }
}

const escape = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** The keywords a text names as whole words, ignoring case; "C++" and ".NET" count, and "C" is not in "C++". */
export function keywordsIn(text: string, keywords: string[]): string[] {
  return keywords.filter(keyword => new RegExp(`(^|[^\\p{L}\\p{N}_+#])${escape(keyword)}($|[^\\p{L}\\p{N}_+#])`, 'iu').test(text))
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

const PLATFORMS: Record<string, string> = {
  'greenhouse': 'Greenhouse', 'lever': 'Lever', 'lever-eu': 'Lever', 'ashby': 'Ashby', 'workday': 'Workday', 'smartrecruiters': 'SmartRecruiters', 'bamboohr': 'BambooHR', 'recruitee': 'Recruitee', 'workable': 'Workable',
}

/** "Greenhouse" for greenhouse; the platform as it is for one without a name. */
export function platformLabel(platform: string): string {
  return PLATFORMS[platform] ?? platform
}

/** "today", "yesterday", "3 days ago", or the date after a month. */
export function postedAgo(at: string, now = new Date()): string {
  const date = new Date(at)
  const days = Math.floor((now.getTime() - date.getTime()) / 86_400_000)
  if (days <= 0) return 'today'
  if (days === 1) return 'yesterday'
  if (days < 31) return `${days} days ago`
  return date.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })
}

export function sourceLine(job: Pick<Job, 'sample' | 'source' | 'url' | 'added_by_me'> & Partial<Pick<Job, 'origin' | 'posted_at'>>, now = new Date()): string {
  if (job.source === 'import') {
    const via = `via ${PLATFORMS[job.origin ?? ''] ?? hostOf(job.url)}`
    return job.posted_at ? `${via} · posted ${postedAgo(job.posted_at, now)}` : via
  }
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
