import { describe, expect, it } from 'vitest'
import { sampleProfile } from '@/test/profile-service'
import { experienceInput, projectInput } from './profile.schema'

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
