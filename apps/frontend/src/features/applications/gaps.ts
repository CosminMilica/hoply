import type { JobCard } from '@/features/jobs/api/jobs.schema'
import { matchJob, skillKey } from '@/features/jobs/match'
import { t } from '@/features/common/i18n'

export interface GapJob {
  id: string
  title: string
  company: string
  must: boolean
}

/** A skill the jobs ask for and the profile does not show; `via` names the close equivalents the person has. */
export interface SkillGap {
  skill: string
  via: string[]
  category: string
  jobs: GapJob[]
  must: number
}

/**
 * What the jobs ask for that the skills do not cover, one entry per skill whatever name each ad gives it,
 * the most asked first (a must-have counts twice a nice-to-have). Missing skills and those the person
 * knows a close equivalent of come apart.
 */
export function skillGaps(jobs: JobCard[], skills: string[]): { missing: SkillGap[], close: SkillGap[] } {
  const gaps = new Map<string, SkillGap>()
  for (const job of jobs) {
    const match = matchJob(job, skills)
    const must = new Set(job.must_have.map(skillKey))
    const entries = [...match.missing.map(skill => ({ skill, via: [] as string[], category: '' })), ...match.related]
    for (const entry of entries) {
      const key = skillKey(entry.skill)
      const gap = gaps.get(key) ?? { skill: entry.skill, via: entry.via, category: entry.category, jobs: [], must: 0 }
      gaps.set(key, gap)
      if (gap.jobs.some(item => item.id === job.id)) continue
      gap.jobs.push({ id: job.id, title: job.title, company: job.company, must: must.has(key) })
      if (must.has(key)) gap.must++
    }
  }
  const weight = (gap: SkillGap) => gap.must + gap.jobs.length
  const ranked = [...gaps.values()].sort((a, b) => weight(b) - weight(a) || b.jobs.length - a.jobs.length || a.skill.localeCompare(b.skill))
  return { missing: ranked.filter(gap => !gap.via.length), close: ranked.filter(gap => gap.via.length) }
}

/** "Asked in 3 of 5 saved jobs, a must-have in 2." */
export function askedLine(gap: Pick<SkillGap, 'jobs' | 'must'>, total: number): string {
  const words = t.value.applications.asked
  if (gap.must === gap.jobs.length) return words.always(gap.jobs.length, total)
  return gap.must ? words.some(gap.jobs.length, total, gap.must) : words.plus(gap.jobs.length, total)
}
