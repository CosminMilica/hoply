import { describe, expect, it } from 'vitest'
import { server } from '@/shared/mocks/node'
import { emptyProfile, profileServiceHandlers } from '@/test/profile-service'
import { profileApi } from './profileApi'
import {
  certificationInputSchema,
  cleanList,
  educationInputSchema,
  experienceInputSchema,
  profilePatchSchema,
  profileSchema,
  projectInputSchema,
  withoutId,
} from './profile.schema'

const role = { role: 'SRE', company: 'A', location: '', employment_type: '' as const, industry: '', team_size: null, people_managed: null, start: '2020-01', end: '', tools: [], bullets: [] }

describe('profileApi', () => {
  it('reads and changes the profile and every list by the contract', async () => {
    const service = profileServiceHandlers(emptyProfile())
    server.use(...service.handlers)
    expect((await profileApi.get()).skills).toEqual([])
    expect((await profileApi.patch({ headline: 'SRE', notice_period_weeks: null, work_modes: ['remote'] })).work_modes).toEqual(['remote'])
    expect((await profileApi.putSkills(['Go'])).skills).toEqual(['Go'])
    const added = await profileApi.experience.add(role)
    expect((await profileApi.experience.update(added.id, { ...role, role: 'Lead SRE' })).role).toBe('Lead SRE')
    const project = await profileApi.projects.add({ name: 'Lab', role: '', url: '', start: '', end: '', tools: [], bullets: [{ text: 'Ran it', result: '', skills: [] }] })
    expect((await profileApi.projects.update(project.id, { ...withoutId(project), name: 'Home lab' })).name).toBe('Home lab')
    const degree = await profileApi.education.add({ level: 'bachelor', field: '', degree: 'BSc', school: '', start_year: '', end_year: '' })
    const certification = await profileApi.certifications.add({ name: 'CKA', issuer: '', issued: '', expires: '', credential_id: '', url: '' })
    expect((await profileApi.certifications.update(certification.id, { ...withoutId(certification), issuer: 'CNCF' })).issuer).toBe('CNCF')
    const language = await profileApi.languages.add({ name: 'English', level: 'C1' })
    for (const remove of [
      () => profileApi.experience.remove(added.id), () => profileApi.projects.remove(project.id), () => profileApi.education.remove(degree.id),
      () => profileApi.certifications.remove(certification.id), () => profileApi.languages.remove(language.id),
    ]) await remove()
    const profile = await profileApi.get()
    expect([profile.experience, profile.projects, profile.education, profile.certifications, profile.languages]).toEqual([[], [], [], [], []])
    await expect(profileApi.education.update('none', { level: 'other', field: '', degree: '', school: '', start_year: '', end_year: '' })).rejects.toThrow()
  })
})

describe('profile schemas', () => {
  it('reads numeric ids, nulls and missing lists the service may send', () => {
    const parsed = profileSchema.parse({ id: 7, name: null, skills: null, open_to_relocation: null, experience: [{ id: 3, role: 'x', bullets: [{ text: 'a' }] }] })
    expect(parsed).toMatchObject({ id: '7', name: '', skills: [], projects: [], certifications: [], open_to_relocation: false, salary_min: null })
    expect(parsed.experience[0]).toMatchObject({ id: '3', tools: [], team_size: null, bullets: [{ text: 'a', result: '', skills: [] }] })
  })

  it('checks the profile rules before sending', () => {
    expect(profilePatchSchema.safeParse({ email: 'nope' }).success).toBe(false)
    expect(profilePatchSchema.safeParse({ notice_period_weeks: 53 }).success).toBe(false)
    expect(profilePatchSchema.safeParse({ notice_period_weeks: null }).success).toBe(true)
    expect(profilePatchSchema.safeParse({ available_from: '2026-13' }).success).toBe(false)
    expect(profilePatchSchema.safeParse({ work_modes: ['office'] }).success).toBe(false)
    expect(profilePatchSchema.parse({ salary_currency: 'eur', name: ' Ana ' })).toEqual({ salary_currency: 'EUR', name: 'Ana' })
    expect(profilePatchSchema.safeParse({ salary_currency: 'EURO' }).success).toBe(false)
    expect(profilePatchSchema.safeParse({ additional_info: Array(11).fill('x') }).success).toBe(false)
  })

  it('checks the entries before sending', () => {
    expect(experienceInputSchema.safeParse({ ...role, start: '2022-05', end: '2022-01' }).success).toBe(false)
    expect(experienceInputSchema.safeParse({ ...role, start: '2022' }).success).toBe(false)
    expect(experienceInputSchema.safeParse({ ...role, bullets: [{ text: 'x', result: 'r'.repeat(201), skills: [] }] }).success).toBe(false)
    expect(experienceInputSchema.safeParse(role).success).toBe(true)
    expect(projectInputSchema.safeParse({ name: '', role: '', url: '', start: '', end: '', tools: [], bullets: [] }).success).toBe(false)
    expect(educationInputSchema.safeParse({ level: undefined, field: '', degree: '', school: '', start_year: '', end_year: '' }).success).toBe(false)
    expect(educationInputSchema.safeParse({ level: 'master', field: '', degree: '', school: '', start_year: '2020', end_year: '2019' }).success).toBe(false)
    expect(certificationInputSchema.safeParse({ name: 'CKA', issuer: '', issued: '2024-05', expires: '2023-05', credential_id: '', url: '' }).success).toBe(false)
  })

  it('cleans lists as the service does', () => {
    expect(cleanList([' React ', '', 'react', 'SQL'])).toEqual(['React', 'SQL'])
  })
})
