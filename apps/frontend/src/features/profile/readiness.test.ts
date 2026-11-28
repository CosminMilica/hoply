import { describe, expect, it } from 'vitest'
import { setLocale } from '@/features/common/i18n'
import { emptyProfile, sampleProfile } from '@/test/profile-service'
import { educationTitle, monthLabel, monthRange, yearRange } from './format'
import { hasWork, initialsOf, isRichRole, profileReadiness, READY_PERCENT, readyForClaude } from './readiness'
import { bulletLine, knownSkills } from './skills'

describe('profileReadiness', () => {
  it('is complete for a full profile', () => {
    expect(profileReadiness(sampleProfile())).toMatchObject({ percent: 100, hint: 'All set. Time to find some jobs.' })
  })

  it('asks for the items in order', () => {
    expect(profileReadiness(emptyProfile()).checks.map(check => check.label)).toEqual([
      'your name', 'a headline', 'a short intro', 'your email', 'your phone number', 'a GitHub or LinkedIn link',
      'your city and country', 'your work authorization', 'at least 5 skills',
      'a role with a start month, tools and two highlights', 'a highlight with a result', 'your education with its level', 'a language',
    ])
    expect(profileReadiness(emptyProfile())).toMatchObject({ percent: 0, hint: 'Next up: add your name' })
    expect(profileReadiness(sampleProfile({ phone: '' }))).toMatchObject({ percent: 92, hint: 'Next up: add your phone number' })
  })

  it('counts a GitHub or a LinkedIn link as one item, and needs both city and country', () => {
    const github = profileReadiness(sampleProfile({ linkedin: '' }))
    const neither = profileReadiness(sampleProfile({ github: '', linkedin: '' }))
    expect([github.percent, neither.percent]).toEqual([100, 92])
    expect(profileReadiness(sampleProfile({ country: '' })).hint).toBe('Next up: add your city and country')
    expect(profileReadiness(sampleProfile({ work_authorization: '' })).hint).toBe('Next up: add your work authorization')
  })

  it('wants a result on some highlight and a level on some education', () => {
    const profile = sampleProfile()
    const noResult = { ...profile, experience: profile.experience.map(role => ({ ...role, bullets: role.bullets.map(b => ({ ...b, result: '' })) })) }
    expect(profileReadiness(noResult).hint).toBe('Next up: add a highlight with a result')
    const fromProject = { ...noResult, projects: [{ id: 'x', name: 'Lab', role: '', url: '', start: '', end: '', tools: [], bullets: [{ text: 'Ran it', result: '12 containers', skills: [] }] }] }
    expect(profileReadiness(fromProject).percent).toBe(100)
    expect(profileReadiness(sampleProfile({ education: [{ ...profile.education[0]!, level: '' }] })).hint).toBe('Next up: add your education with its level')
  })

  it('wants a role with a start month, three tools and two highlights', () => {
    const role = sampleProfile().experience[0]!
    expect(isRichRole(role)).toBe(true)
    expect(isRichRole({ ...role, start: '' })).toBe(false)
    expect(isRichRole({ ...role, tools: ['a', 'b'] })).toBe(false)
  })
})

describe('in Romanian', () => {
  it('names the next item, the dates and the education in Romanian', () => {
    setLocale('ro')
    expect(profileReadiness(emptyProfile())).toMatchObject({ percent: 0, hint: 'Următorul pas: adaugă numele tău' })
    expect(profileReadiness(sampleProfile()).hint).toBe('Totul e gata. E timpul să cauți joburi.')
    expect(monthRange('2022-03', '')).toBe('03/2022 – prezent')
    expect(yearRange('2016', '')).toBe('2016 – prezent')
    expect(educationTitle({ degree: '', level: 'master', field: 'Data Science' })).toBe('Master în Data Science')
    expect(educationTitle({ degree: '', level: '', field: 'Drept' })).toBe('Drept')
    expect(educationTitle({ degree: '', level: '', field: '' })).toBe('Educație')
  })
})

describe('initialsOf', () => {
  it('takes the first letters of the first and last names', () => {
    expect(initialsOf('Andrei Popescu')).toBe('AP')
    expect(initialsOf('  maria   ioana pop ')).toBe('MP')
    expect(initialsOf('Cher')).toBe('C')
    expect(initialsOf('')).toBe('')
  })
})

describe('formatting', () => {
  it('prints dates', () => {
    expect(monthLabel('2022-03')).toBe('03/2022')
    expect(monthLabel('soon')).toBe('soon')
    expect(monthRange('2022-03', '')).toBe('03/2022 – now')
    expect(monthRange('2022-03', '2024-01')).toBe('03/2022 – 01/2024')
    expect(monthRange('', '2024-01')).toBe('')
    expect(yearRange('2016', '2020')).toBe('2016 – 2020')
    expect(yearRange('', '2020')).toBe('? – 2020')
    expect(yearRange('2016', '')).toBe('2016 – now')
    expect(yearRange('', '')).toBe('')
  })

  it('titles education by its degree, else its level and field', () => {
    expect(educationTitle({ degree: 'BSc CS', level: 'bachelor', field: 'CS' })).toBe('BSc CS')
    expect(educationTitle({ degree: '', level: 'master', field: 'Data Science' })).toBe('Master’s degree in Data Science')
    expect(educationTitle({ degree: '', level: 'doctorate', field: '' })).toBe('Doctorate')
    expect(educationTitle({ degree: '', level: '', field: '' })).toBe('Education')
  })
})

describe('skills', () => {
  it('knows the skills, the tools and the skills highlights prove, once each', () => {
    const profile = sampleProfile({
      skills: ['Python', 'Git'],
      experience: [{ ...sampleProfile().experience[0]!, tools: ['git', 'Docker'], bullets: [{ text: 'x', result: '', skills: ['Terraform', ' '] }] }],
      projects: [{ id: 'p', name: 'Lab', role: '', url: '', start: '', end: '', tools: ['Grafana'], bullets: [] }],
    })
    expect(knownSkills(profile)).toEqual(['Python', 'Git', 'Docker', 'Terraform', 'Grafana'])
  })

  it('reads a highlight with its result', () => {
    expect(bulletLine({ text: 'Built pipelines.', result: 'Cut release time to 15 minutes.' })).toBe('Built pipelines, cut release time to 15 minutes.')
    expect(bulletLine({ text: 'Built pipelines', result: ' ' })).toBe('Built pipelines')
  })
})

describe('readyForClaude', () => {
  // The cv service keeps the same rule and cases (cvdoc.ReadyForClaude, readiness_test.go).
  const contact = emptyProfile()
  Object.assign(contact, { name: 'Ana Pop', headline: 'Accountant', about: 'About', email: 'a@b.ro', phone: '+40', linkedin: 'li',
    city: 'Iași', country: 'Romania', work_authorization: 'citizen' })

  it('needs the percent and some work', () => {
    expect(READY_PERCENT).toBe(60)
    expect(readyForClaude(emptyProfile())).toBe(false)
    expect(profileReadiness(contact).percent).toBe(62)
    expect(hasWork(contact)).toBe(false)
    expect(readyForClaude(contact)).toBe(false)
    const working = { ...contact, projects: [{ id: 'p1', name: 'Books', role: '', url: '', start: '', end: '', tools: [], notes: [],
      bullets: [{ text: 'Kept the books of 12 clients', result: '', skills: [] }] }] }
    expect(hasWork(working)).toBe(true)
    expect(readyForClaude(working)).toBe(true)
    expect(profileReadiness({ ...working, headline: '', about: '' }).percent).toBe(46)
    expect(readyForClaude({ ...working, headline: '', about: '' })).toBe(false)
    expect(readyForClaude(sampleProfile())).toBe(true)
  })
})
