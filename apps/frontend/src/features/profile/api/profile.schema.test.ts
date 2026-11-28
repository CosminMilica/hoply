import { describe, expect, it } from 'vitest'
import { setLocale, t } from '@/features/common/i18n'
import { sampleProfile } from '@/test/profile-service'
import {
  bulletInputSchema,
  certificationInputSchema,
  educationInputSchema,
  experienceInput,
  experienceInputSchema,
  languageInputSchema,
  profilePatchSchema,
  profileSchema,
  projectInput,
  projectInputSchema,
} from './profile.schema'

describe('entry inputs', () => {
  it('sends a role or a project back with a change, and leaves an unknown employment type empty', () => {
    const profile = sampleProfile()
    const role = profile.experience[0]!
    expect(experienceInput(role, { notes: [] })).toMatchObject({ role: role.role, employment_type: 'full_time', notes: [] })
    expect(experienceInput({ ...role, employment_type: 'apprenticeship' }).employment_type).toBe('')
    expect('id' in experienceInput(role)).toBe(false)
    const project = { id: 'p1', name: 'Hoply', role: '', url: '', start: '', end: '', tools: [], bullets: [], notes: [] }
    expect(projectInput(project, { bullets: [{ text: 'Built it', result: '', skills: [] }] })).toEqual({ ...project, id: undefined, bullets: [{ text: 'Built it', result: '', skills: [] }] })
  })
})

const messages = (result: { success: boolean, error?: { issues: { message: string }[] } }) => result.error?.issues.map(issue => issue.message) ?? []

describe('validation messages', () => {
  const role = { role: '', company: '', location: '', employment_type: '', industry: '', team_size: 1.5, people_managed: -1, start: '2024-05', end: '2023-01', tools: Array(31).fill('x'), bullets: Array(11).fill({ text: 'x', result: '', skills: [] }) }

  it('follows the language the app shows when the value is checked', () => {
    const email = profilePatchSchema.safeParse({ email: 'not-an-email' })
    expect(messages(email)).toEqual(['Enter a valid email address'])
    setLocale('ro')
    expect(messages(profilePatchSchema.safeParse({ email: 'not-an-email' }))).toEqual(['Introdu o adresă de email validă'])
    setLocale('en')
    expect(messages(profilePatchSchema.safeParse({ email: 'not-an-email' }))).toEqual(['Enter a valid email address'])
  })

  it('says the list limits in English by default', () => {
    expect(messages(projectInputSchema.safeParse({ name: 'Lab', role: '', url: '', start: '', end: '', tools: role.tools, bullets: role.bullets }))).toEqual(['At most 30 tools', 'At most 10 highlights'])
    expect(messages(bulletInputSchema.safeParse({ text: 'x', result: '', skills: Array(11).fill('Go') }))).toEqual(['At most 10 skills'])
    expect(messages(profilePatchSchema.safeParse({
      target_roles: Array(11).fill('Dev'),
      additional_info: Array(11).fill('x'),
      notes: Array(31).fill({ text: 'x', skills: [] }),
    }))).toEqual(['At most 10 target roles', 'At most 10 lines', 'At most 30 notes'])
  })

  it('says every rule in Romanian', () => {
    setLocale('ro')
    expect(messages(experienceInputSchema.safeParse(role))).toEqual([
      'Adaugă rolul', 'Folosește un număr întreg', 'Folosește 0 sau mai mult', 'Cel mult 30 de unelte', 'Cel mult 10 realizări',
    ])
    expect(messages(experienceInputSchema.safeParse({ ...role, role: 'Dev', team_size: 1, people_managed: 0, tools: [], bullets: [] }))).toEqual(['Finalul nu poate fi înainte de început'])
    expect(messages(projectInputSchema.safeParse({ name: ' ', role: '', url: '', start: '2024', end: '', tools: [], bullets: [] }))).toEqual(['Adaugă numele proiectului', 'Folosește o lună, de ex. 2022-03'])
    expect(messages(educationInputSchema.safeParse({ level: 'phd', field: '', degree: '', school: '', start_year: '1900', end_year: '' }))).toEqual(['Alege nivelul', 'Folosește un an între 1950 și 2100, de ex. 2022'])
    expect(messages(educationInputSchema.safeParse({ level: 'master', field: '', degree: '', school: '', start_year: '2020', end_year: '2019' }))).toEqual(['Anul de final nu poate fi înainte de anul de început'])
    expect(messages(certificationInputSchema.safeParse({ name: '', issuer: '', issued: '2025-04', expires: '2024-04', credential_id: '', url: '' })))
      .toEqual(['Adaugă certificarea', 'Nu poate expira înainte de data emiterii'])
    expect(messages(languageInputSchema.safeParse({ name: '', level: 'B2' }))).toEqual(['Adaugă limba'])
    expect(messages(bulletInputSchema.safeParse({ text: 'x'.repeat(301), result: '', skills: Array(11).fill('Go') }))).toEqual([t.value.common.validation.atMost(300), 'Cel mult 10 competențe'])
    expect(messages(profilePatchSchema.safeParse({
      notice_period_weeks: 60,
      target_roles: Array(11).fill('Dev'),
      salary_currency: 'euro',
      additional_info: Array(11).fill('x'),
      notes: Array(31).fill({ text: 'x', skills: [] }),
    }))).toEqual([
      'Cel mult 52', 'Cel mult 10 roluri dorite', 'Folosește un cod din trei litere, de ex. EUR', 'Cel mult 10 rânduri', 'Cel mult 30 de notițe',
    ])
  })
})

describe('language and domains', () => {
  it('reads them from the service, empty when it sends none, and sends only the known ones', () => {
    const { app_language, domains, ...rest } = sampleProfile()
    expect(profileSchema.parse(rest)).toMatchObject({ app_language: '', domains: [] })
    expect(profileSchema.parse({ ...rest, app_language: 'ro', domains: ['it', 'hr'] })).toMatchObject({ app_language: 'ro', domains: ['it', 'hr'] })
    expect(app_language).toBe('en')
    expect(domains).toEqual(['it'])
    expect(profilePatchSchema.safeParse({ app_language: 'ro', domains: ['banking'] }).success).toBe(true)
    expect(profilePatchSchema.safeParse({ app_language: '' }).success).toBe(true)
    expect(profilePatchSchema.safeParse({ app_language: 'de' }).success).toBe(false)
    expect(profilePatchSchema.safeParse({ domains: ['legal'] }).success).toBe(false)
  })
})
