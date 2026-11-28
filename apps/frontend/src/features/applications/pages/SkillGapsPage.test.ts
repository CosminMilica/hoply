import { afterEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { setLocale } from '@/features/common/i18n'
import { mountPage, useServices } from '@/test/hoply'
import { seedId } from '@/test/jobs-seed'
import { sampleProfile } from '@/test/profile-service'
import SkillGapsPage from './SkillGapsPage.vue'

const text = () => document.body.textContent ?? ''
const section = (id: string) => document.querySelector(`[data-test="gaps-${id}"]`)?.textContent ?? ''

afterEach(() => {
  document.body.innerHTML = ''
})

describe('SkillGapsPage', () => {
  it('lists what the saved jobs ask for and the profile lacks, the most asked first', async () => {
    const profile = sampleProfile()
    profile.skills.push('Azure')
    const { applications } = useServices(profile)
    applications.track(seedId(1), 'saved')
    applications.track(seedId(2), 'cv_ready')
    applications.track(seedId(4), 'applied')
    const { router } = await mountPage(SkillGapsPage)
    expect(section('missing')).toMatch(/^Not in your profile \d+TerraformAsked in 2 of 3 saved jobs, always a must-have\.DevOps Engineer · Bitdefender/)
    expect(section('missing')).toContain('Windows ServerAsked in 1 of 3 saved jobs, always a must-have.System Administrator · Carpathia Systems')
    expect(section('missing')).not.toContain('AWS')
    expect(section('close')).toContain('AWSYou know Azure (cloud infrastructure).Asked in 1 of 3 saved jobs')
    const link = [...document.querySelectorAll('a')].find(item => item.textContent?.trim() === 'Snowflake Administrator DevOps · Capgemini Romania')!
    expect(link.getAttribute('href')).toBe(`/jobs/${seedId(1)}`)
    expect(text()).toContain('Learned one? Add it to your profile and it leaves this list.')
    await router.push(link.getAttribute('href')!)
    expect(router.currentRoute.value.name).toBe('hoply-job')
  })

  it('speaks Romanian', async () => {
    setLocale('ro')
    const profile = sampleProfile()
    profile.skills.push('Azure')
    const { applications } = useServices(profile)
    applications.track(seedId(1), 'saved')
    applications.track(seedId(2), 'cv_ready')
    applications.track(seedId(4), 'applied')
    await mountPage(SkillGapsPage)
    expect(text()).toContain('Competențe lipsă')
    expect(section('missing')).toMatch(/^Nu apar în profilul tău \d+TerraformCerută în 2 din 3 joburi salvate, mereu obligatorie\./)
    expect(section('close')).toMatch(/^Apropiate de ce știi \d+AWSȘtii Azure \(cloud infrastructure\)\.Cerută în 1 din 3 joburi salvate/)
    expect(text()).toContain('Ai învățat una? Adaug-o în profilul tău și dispare din listă.')
  })

  it('says in Romanian when there is nothing to show', async () => {
    setLocale('ro')
    useServices()
    await mountPage(SkillGapsPage)
    expect(text()).toContain('Salvează întâi câteva joburi.')
    document.body.innerHTML = ''

    const { applications, jobs } = useServices(sampleProfile({ skills: ['Python', 'Git', 'Docker'] }))
    const job = jobs.jobs.find(item => item.id === seedId(1))!
    job.must_have = ['Python', 'Git']
    job.nice_to_have = ['Docker']
    applications.track(seedId(1), 'saved')
    await mountPage(SkillGapsPage)
    expect(text()).toContain('Profilul tău acoperă toate cerințele joburilor salvate.')
  })

  it('says when the saved jobs cannot be loaded', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'saved')
    server.use(http.get('*/api/jobs/v1/jobs', () => HttpResponse.json({ message: 'down' }, { status: 503 })))
    await mountPage(SkillGapsPage)
    await vi.waitFor(() => expect(text()).toContain('We could not load your saved jobs'))
    expect(text()).not.toContain('Save a few jobs first')
  })

  it('asks for saved jobs first, and says when the profile covers them all', async () => {
    useServices()
    await mountPage(SkillGapsPage)
    expect(text()).toContain('Save a few jobs first.')
    document.body.innerHTML = ''

    const { applications, jobs } = useServices(sampleProfile({ skills: ['Python', 'SQL', 'Git', 'Linux', 'Docker', 'CI/CD', 'Bash', 'Monitoring'] }))
    const job = jobs.jobs.find(item => item.id === seedId(1))!
    job.must_have = ['Python', 'Git']
    job.nice_to_have = ['Docker']
    applications.track(seedId(1), 'saved')
    await mountPage(SkillGapsPage)
    expect(text()).toContain('Your profile covers every requirement of your saved jobs.')
  })
})
