import { afterEach, describe, expect, it } from 'vitest'
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
