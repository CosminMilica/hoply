import { describe, expect, it } from 'vitest'
import { setLocale } from '@/features/common/i18n'
import type { Job } from '@/features/jobs/api/jobs.schema'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { engineJob } from '@/test/jobs'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { settle, useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import type { Profile } from '@/features/profile/api/profile.schema'
import { prepQuestions } from '@/test/cv-engine/questions'
import { OTHER } from './useNotes'
import { usePrep } from './usePrep'
import { seedId } from '@/test/jobs-seed'

const job = engineJob(0)

async function setup(person: Profile = sampleProfile(), ad: Job = job) {
  const services = useServices(person)
  const questions = prepQuestions(ad, person)
  let prep!: ReturnType<typeof usePrep>
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  mount(defineComponent({
    setup() {
      const profile = useProfileQuery()
      prep = usePrep({ job: ad, profile: () => profile.data.value, questions })
      return () => h('div')
    },
  }), { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
  await settle()
  return { prep, profile: services.profile }
}

describe('usePrep', () => {
  it('remembers a gap fact under the chosen role and takes exactly it back', async () => {
    const person = sampleProfile()
    person.experience[1]!.bullets.push({ text: 'Wrote modules for 3 environments', result: '', skills: [] })
    const { prep, profile } = await setup(person)
    expect(prep.matchLabel.value).toBe('56%')
    prep.answers.setDraft('g:Terraform', 'too short')
    await prep.addGap('Terraform')
    expect(prep.answers.answer('g:Terraform')).toBeUndefined()
    prep.answers.setDraft('g:Terraform', ' Wrote modules for 3 environments ')
    expect(prep.roleChoices('g:Terraform').map(choice => choice.label)).toEqual(['Verde Cloud', 'Bonbon Labs', 'Elsewhere'])
    prep.pickRole('g:Terraform', 'experience:r2')
    await prep.addGap('Terraform')
    await settle()
    expect(profile.experience[1]!.notes).toEqual([{ text: 'Wrote modules for 3 environments', skills: ['Terraform'] }])
    expect(profile.experience[1]!.bullets).toHaveLength(4)
    expect(prep.whereLabel('experience:r2')).toBe('Junior system administrator at Bonbon Labs')
    expect(prep.matchLabel.value).toBe('56% → 78%')

    await prep.answers.undo('g:Terraform')
    await settle()
    expect(prep.answers.answer('g:Terraform')).toBeUndefined()
    expect(profile.experience[1]!.notes).toEqual([])
    expect(profile.experience[1]!.bullets.at(-1)).toEqual({ text: 'Wrote modules for 3 environments', result: '', skills: [] })
    expect(prep.matchLabel.value).toBe('56%')
    prep.answers.skip('g:RBAC')
    await prep.answers.undo('g:RBAC')
    expect(prep.answers.answer('g:RBAC')).toBeUndefined()
  })

  it('remembers elsewhere as the person’s own notes, and in a project', async () => {
    const person = sampleProfile({ projects: [{ id: 'p1', name: 'Home lab', role: '', url: '', start: '', end: '', tools: [], bullets: [], notes: [] }] })
    const { prep, profile } = await setup(person)
    prep.answers.setDraft('g:Snowflake', 'Used Snowflake at a meetup demo')
    prep.pickRole('g:Snowflake', OTHER)
    await prep.addGap('Snowflake')
    await settle()
    expect(profile.notes).toEqual([{ text: 'Used Snowflake at a meetup demo', skills: ['Snowflake'] }])
    expect(profile.additional_info).toEqual([])
    expect(prep.whereLabel(OTHER)).toBe('Your notes')
    await prep.answers.undo('g:Snowflake')
    await settle()
    expect(profile.notes).toEqual([])

    prep.answers.setDraft('c:3', 'Built reusable Terraform modules for the lab')
    prep.pickRole('c:3', 'projects:p1')
    await prep.claimYes(3)
    await settle()
    expect(profile.projects[0]!.notes).toEqual([{ text: 'Built reusable Terraform modules for the lab', skills: ['Terraform'] }])
    expect(profile.projects[0]!.bullets).toEqual([])
    expect(prep.whereLabel('projects:p1')).toBe('the Home lab project')
    await prep.answers.undo('c:3')
    await settle()
    expect(profile.projects[0]!.notes).toEqual([])
  })

  it('offers the person’s own highlights that share a keyword with a responsibility', async () => {
    const person = sampleProfile()
    person.experience[0]!.bullets.push({ text: 'Kept Terraform state tidy', result: '', skills: [] })
    person.experience[1]!.bullets.push({ text: 'Wrote modules', result: 'saved a day a week', skills: ['terraform'] })
    const { prep } = await setup(person)
    expect(prep.suggestionsFor(job.responsibilities[3]!)).toEqual([
      { label: 'Verde Cloud', text: 'Kept Terraform state tidy' },
      { label: 'Bonbon Labs', text: 'Wrote modules, saved a day a week.' },
    ])
    expect(prep.suggestionsFor(job.responsibilities[7]!)).toEqual([])
    prep.answers.pick('c:3', 'Kept Terraform state tidy')
    prep.answers.skip('c:2')
    expect(prep.progress.value).toBe('2 of 6 questions answered')
    expect(prep.toRequest(seedId(1)).confirmed).toEqual([3])
    await prep.answers.undo('c:3')
    expect(prep.answers.answer('c:3')).toBeUndefined()
  })

  it('saves results and extra lines in the profile', async () => {
    const { prep, profile } = await setup()
    expect(prep.impactRows.value.map(row => row.key)).toEqual(['r1-0', 'r1-1', 'r1-2'])
    await prep.saveImpact('r1-1', ' cut setup from 2 days to 30 minutes ')
    await settle()
    expect(profile.experience[0]!.bullets[1]!.result).toBe('cut setup from 2 days to 30 minutes')
    expect(prep.impactRows.value).toHaveLength(3)
    await prep.saveImpact('r1-1', 'cut setup from 2 days to 30 minutes')
    await prep.saveImpact('missing', 'x')

    prep.note.value = 'CKA, 2025\n\n  Home lab  '
    await prep.saveNote()
    await settle()
    expect(profile.additional_info).toEqual(['CKA, 2025', 'Home lab'])
    expect(prep.note.value).toBe('CKA, 2025\nHome lab')
    await prep.saveNote()
  })

  it('keeps the service’s refusal next to the question', async () => {
    const { prep } = await setup()
    server.use(http.put('*/api/profile/v1/profiles/me/experience/:id', () => HttpResponse.json({ message: 'at most 10 bullets' }, { status: 422 })))
    prep.answers.setDraft('g:RBAC', 'Set up RBAC roles for 15 teams')
    await prep.addGap('RBAC')
    expect(prep.answers.state.errors['g:RBAC']).toBe('at most 10 bullets')
    expect(prep.answers.answer('g:RBAC')).toBeUndefined()
  })

  it('keeps the CV’s own choices: roles and up to three focus skills', async () => {
    const { prep } = await setup()
    expect(prep.roles.value[0]).toMatchObject({ overlap: 'Uses SQL, Git, CI/CD, Monitoring and Python', meta: 'Verde Cloud · 03/2022 – now', included: true })
    prep.toggleRole('r2')
    expect(prep.roleChoices('g:x').map(choice => choice.label)).toEqual(['Verde Cloud', 'Elsewhere'])
    expect(prep.focusChips.value.filter(chip => chip.pressed).map(chip => chip.label)).toEqual(['SQL', 'Git', 'CI/CD'])
    prep.toggleFocus('Python')
    expect(prep.state.focus).toBeNull()
    prep.toggleFocus('Git')
    prep.toggleFocus('Python')
    expect(prep.toRequest(seedId(1))).toEqual({ job_id: seedId(1), skip_roles: ['r2'], focus: ['SQL', 'CI/CD', 'Python'], confirmed: [], language: 'en' })
    prep.state.language = 'ro'
    expect(prep.toRequest(seedId(1)).language).toBe('ro')
  })

  it('writes the CV in the person’s pick, else the ad’s language, else the app’s', async () => {
    setLocale('ro')
    const english = await setup()
    expect(english.prep.language.value).toBe('en')
    english.prep.state.language = 'ro'
    expect(english.prep.toRequest(seedId(1)).language).toBe('ro')

    const neutral = await setup(sampleProfile(), { ...job, title: 'DevOps Engineer', about: '', responsibilities: [], must_have: ['Kubernetes', 'Terraform'] })
    expect(neutral.prep.language.value).toBe('ro')
    setLocale('en')
    expect(neutral.prep.language.value).toBe('en')
    neutral.prep.state.language = 'ro'
    expect(neutral.prep.language.value).toBe('ro')
  })

  it('speaks Romanian when the app does', async () => {
    setLocale('ro')
    const person = sampleProfile({ projects: [{ id: 'p1', name: 'Home lab', role: '', url: '', start: '', end: '', tools: [], bullets: [], notes: [] }] })
    const { prep } = await setup(person)
    expect(prep.progress.value).toBe('0 din 8 întrebări cu răspuns')
    expect(prep.roleChoices('g:x').at(-1)!.label).toBe('În altă parte')
    expect(prep.roles.value[0]!.overlap).toBe('Folosește SQL, Git, CI/CD, Monitoring și Python')
    expect(prep.whereLabel(OTHER)).toBe('notițele tale')
    expect(prep.whereLabel('projects:p1')).toBe('proiectul Home lab')
    expect(prep.whereLabel('experience:r2')).toBe('Junior system administrator la Bonbon Labs')
  })

  it('falls back to the person’s own notes without a role or project', async () => {
    const { prep } = await setup(sampleProfile({ experience: [] }))
    expect(prep.roleChoices('c:1')).toEqual([{ id: OTHER, label: 'Elsewhere', pressed: true }])
    expect(prep.impactRows.value).toEqual([])
  })
})
