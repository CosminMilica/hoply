import { EDUCATION_LEVELS, type Education, type Profile } from './api/profile.schema'

export function plural(count: number, one: string, many = `${one}s`): string {
  return `${count} ${count === 1 ? one : many}`
}

/** `2022-03` as `03/2022`. */
export function monthLabel(value: string): string {
  const [year, month] = value.split('-')
  return year && month ? `${month}/${year}` : value
}

/** `03/2022 – now`; nothing without a start. */
export function monthRange(start: string, end: string, ongoing = 'now'): string {
  return start ? `${monthLabel(start)} – ${end ? monthLabel(end) : ongoing}` : ''
}

export function yearRange(start: string, end: string): string {
  if (!start && !end) return ''
  return `${start || '?'} – ${end || 'now'}`
}

/** The degree's own title, else its level and field: "Bachelor’s degree in Computer Science". */
export function educationTitle(item: Pick<Education, 'degree' | 'level' | 'field'>): string {
  if (item.degree.trim()) return item.degree
  const level = EDUCATION_LEVELS[item.level as keyof typeof EDUCATION_LEVELS] ?? ''
  return [level, item.field.trim()].filter(Boolean).join(' in ') || 'Education'
}

const MONTH = /^\d{4}-\d{2}$/
const monthIndex = (value: string) => {
  const [year, month] = value.split('-').map(Number)
  return year! * 12 + (month! - 1)
}

/** Whole years from the earliest role's start month to `now`. */
export function yearsOfExperience(profile: Pick<Profile, 'experience'>, now: Date): number {
  const starts = profile.experience.map(role => role.start).filter(start => MONTH.test(start)).map(monthIndex)
  if (!starts.length) return 0
  return Math.max(0, Math.floor((now.getFullYear() * 12 + now.getMonth() - Math.min(...starts)) / 12))
}
