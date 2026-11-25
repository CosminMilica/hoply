import type { Bullet, Experience, Profile, Project } from './api/profile.schema'

/** Every skill the profile shows: its skills, the tools of each role and project, and the skills its highlights and notes prove. */
export function knownSkills(profile: Pick<Profile, 'skills' | 'experience' | 'projects'> & Partial<Pick<Profile, 'notes'>>): string[] {
  const entries: (Experience | Project)[] = [...profile.experience, ...profile.projects]
  const all = [
    ...profile.skills,
    ...entries.flatMap(entry => [...entry.tools, ...entry.bullets.flatMap(bullet => bullet.skills), ...(entry.notes ?? []).flatMap(note => note.skills)]),
    ...(profile.notes ?? []).flatMap(note => note.skills),
  ]
  const seen = new Set<string>()
  return all.filter((skill) => {
    const key = skill.trim().toLowerCase()
    if (!key || seen.has(key)) return false
    seen.add(key)
    return true
  })
}

/** A highlight as one line: its text, then its result. */
export function bulletLine(bullet: Pick<Bullet, 'text' | 'result'>): string {
  const result = bullet.result.trim()
  if (!result) return bullet.text
  return `${bullet.text.trim().replace(/[.]+$/, '')}, ${result.charAt(0).toLowerCase()}${result.slice(1).replace(/[.]+$/, '')}.`
}
