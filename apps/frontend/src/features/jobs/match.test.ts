import { describe, expect, it } from 'vitest'
import { setLocale } from '@/features/common/i18n'
import { catalogJob } from '@/test/jobs'
import { sampleProfile } from '@/test/profile-service'
import { keywordsIn, knockoutFor, levelLabel, lookFor, marketNote, matchJob, placeOf, platformLabel, postedAgo, requirementsOf, sourceLine, tagline, verdictFor, yearsLine } from './match'

describe('matchJob', () => {
  it('scores the requirements the skills cover, ignoring case, a must-have twice a nice-to-have', () => {
    const job = { must_have: ['Linux', 'Docker'], nice_to_have: ['Kubernetes', 'AWS'] }
    expect(matchJob(job, ['linux', 'kubernetes'])).toMatchObject({ matched: ['Linux', 'Kubernetes'], missing: ['Docker', 'AWS'], score: 50 })
    expect(matchJob(job, ['Linux', 'Docker'])).toMatchObject({ score: 67 })
    expect(matchJob(job, ['Kubernetes', 'AWS'])).toMatchObject({ score: 33 })
    expect(matchJob({ must_have: [], nice_to_have: [] }, ['x']).score).toBe(0)
    expect(requirementsOf(job)).toEqual(['Linux', 'Docker', 'Kubernetes', 'AWS'])
  })

  it('knows other names, umbrella terms and close tools', () => {
    const job = { must_have: ['CI/CD', 'Containers', 'Git', 'ArgoCD', 'GCP', 'GraphQL', 'k8s'], nice_to_have: ['Redis'] }
    const match = matchJob(job, ['GitHub Actions', 'Docker', 'FluxCD', 'AWS', 'Kubernetes', 'Valkey'])
    expect(match.matched).toEqual(['CI/CD', 'Containers', 'Git', 'k8s'])
    expect(match.related).toEqual([
      { skill: 'ArgoCD', via: ['FluxCD'], category: 'GitOps' },
      { skill: 'GCP', via: ['AWS'], category: 'cloud infrastructure' },
      { skill: 'Redis', via: ['Valkey'], category: 'in-memory caching' },
    ])
    expect(match.missing).toEqual(['GraphQL'])
    expect(match.score).toBe(70)
    expect(matchJob(job, ['Argo CD']).matched).toEqual(['CI/CD', 'ArgoCD'])
  })

  it('words the verdict by score and finds keywords in a text', () => {
    expect(verdictFor(80)).toMatch(/^Strong/)
    expect(verdictFor(60)).toMatch(/^Good/)
    expect(verdictFor(40)).toMatch(/^Partial/)
    expect(verdictFor(10)).toMatch(/^Light/)
    expect(keywordsIn('Ran CI/CD with Go', ['CI/CD', 'Go', 'Java'])).toEqual(['CI/CD', 'Go'])
    expect(keywordsIn('Built C++ and C# services on .NET', ['C++', 'C#', '.NET', 'C'])).toEqual(['C++', 'C#', '.NET'])
  })
})

describe('knockoutFor', () => {
  const person = sampleProfile({ city: 'Iași', work_modes: ['remote', 'hybrid'], open_to_relocation: false })

  it('warns about a work mode the person does not want', () => {
    expect(knockoutFor({ city: 'Iași', work_mode: 'onsite' }, sampleProfile({ city: 'Iași', work_modes: ['remote'] }))).toBe('')
    expect(knockoutFor({ city: 'Cluj-Napoca', work_mode: 'onsite' }, person))
      .toBe('On-site in Cluj-Napoca. Your profile asks for remote or hybrid work, and you are not open to relocation.')
    expect(knockoutFor({ city: '', work_mode: 'remote' }, sampleProfile({ work_modes: ['onsite'] })))
      .toBe('Remote. Your profile asks for on-site work.')
  })

  it('warns about a city elsewhere when the person does not relocate', () => {
    expect(knockoutFor({ city: 'Cluj-Napoca', work_mode: 'hybrid' }, person)).toBe('Based in Cluj-Napoca, hybrid. You live in Iași and are not open to relocation.')
    expect(knockoutFor({ city: 'Cluj-Napoca', work_mode: '' }, person)).toBe('Based in Cluj-Napoca. You live in Iași and are not open to relocation.')
    expect(knockoutFor({ city: 'Cluj-Napoca', work_mode: 'remote' }, person)).toBe('')
    expect(knockoutFor({ city: 'Cluj-Napoca', work_mode: 'hybrid' }, { ...person, open_to_relocation: true })).toBe('')
    expect(knockoutFor({ city: 'Iasi', work_mode: 'hybrid' }, person)).toBe('')
  })

  it('says nothing the profile does not support', () => {
    expect(knockoutFor({ city: 'Cluj-Napoca', work_mode: 'onsite' }, sampleProfile({ city: '', work_modes: [] }))).toBe('')
  })
})

describe('job lines', () => {
  it('says how many years the job asks for, against the person’s', () => {
    expect(yearsLine({ years_required: 4 }, sampleProfile(), new Date('2026-11-25'))).toBe('Asks for 4+ years of experience. You have about 6.')
    expect(yearsLine({ years_required: null }, sampleProfile())).toBe('')
  })

  it('describes where a job comes from, where it is and what it asks', () => {
    expect(sourceLine(catalogJob(0))).toBe('via www.linkedin.com')
    expect(sourceLine(catalogJob(1))).toBe('Sample listing, not a real posting')
    expect(sourceLine(catalogJob(1, { sample: false }))).toBe('From the catalog')
    expect(sourceLine(catalogJob(1, { source: 'link', added_by_me: true }))).toBe('Added by you · from a link')
    expect(sourceLine(catalogJob(1, { source: 'text', added_by_me: true }))).toBe('Added by you · from pasted text')
    const now = new Date('2026-11-26T12:00:00Z')
    const imported = catalogJob(1, { source: 'import', origin: 'greenhouse', url: 'https://job-boards.eu.greenhouse.io/x/jobs/1', posted_at: '2026-11-23T09:00:00Z' })
    expect(sourceLine(imported, now)).toBe('via Greenhouse · posted 3 days ago')
    expect(sourceLine({ ...imported, origin: 'jobvite', url: 'https://careers.ing.com/en/job/1', posted_at: null })).toBe('via careers.ing.com')
    expect([platformLabel('smartrecruiters'), platformLabel('bamboohr'), platformLabel('workable'), platformLabel('jobvite')]).toEqual(['SmartRecruiters', 'BambooHR', 'Workable', 'jobvite'])
    expect(['2026-11-26T08:00:00Z', '2026-11-25T08:00:00Z', '2026-09-01T08:00:00Z'].map(at => postedAgo(at, now))).toEqual(['today', 'yesterday', '1 Sept'])
    expect(placeOf(catalogJob(0))).toBe('Iași, on-site')
    expect(placeOf({ city: '', work_mode: '' })).toBe('Location not given')
    expect(tagline(catalogJob(0))).toBe('DevOps, Data · Senior')
    expect(tagline({ areas: [], level: '' })).toBe('')
    expect(levelLabel('lead')).toBe('Lead')
    expect(lookFor({ must_have: ['Linux'], nice_to_have: ['AWS'] })).toEqual(['Experience with Linux.', 'AWS is a plus.'])
  })

  it('reminds the person of their market', () => {
    expect(marketNote(sampleProfile({ work_modes: ['hybrid', 'onsite'] }))).toBe('From your profile: Iași, Romania · hybrid or on-site · not relocating.')
    expect(marketNote(sampleProfile({ city: '', country: 'Romania', work_modes: [], open_to_relocation: true }))).toBe('From your profile: Romania · open to relocation.')
    expect(marketNote(sampleProfile({ city: '', country: '', work_modes: [] }))).toBe('')
  })
})

describe('in Romanian', () => {
  it('words the match, the warnings and the job lines in Romanian', () => {
    setLocale('ro')
    expect([verdictFor(90), verdictFor(65), verdictFor(45), verdictFor(5)].map(verdict => verdict.split('.')[0]!.split(',')[0]))
      .toEqual(['Potrivire puternică', 'Potrivire bună', 'Potrivire parțială', 'Potrivire slabă'])
    expect(matchJob({ must_have: ['Linux'], nice_to_have: [] }, ['Linux']).verdict).toBe('Potrivire puternică. Ai deja aproape tot ce cer.')
    const person = sampleProfile({ city: 'Iași', work_modes: ['remote', 'hybrid'], open_to_relocation: false })
    expect(knockoutFor({ city: 'Cluj-Napoca', work_mode: 'onsite' }, person))
      .toBe('La birou în Cluj-Napoca. În profil ai ales să lucrezi remote sau hibrid și nu vrei să te muți.')
    expect(knockoutFor({ city: '', work_mode: 'remote' }, sampleProfile({ work_modes: ['onsite'] }))).toBe('Remote. În profil ai ales să lucrezi la birou.')
    expect(knockoutFor({ city: 'Cluj-Napoca', work_mode: 'hybrid' }, person)).toBe('Jobul e în Cluj-Napoca, hibrid. Tu locuiești în Iași și nu vrei să te muți.')
    expect(yearsLine({ years_required: 4 }, sampleProfile(), new Date('2026-11-25'))).toBe('Cere cel puțin 4 ani de experiență. Tu ai cam 6 ani.')
    expect(placeOf(catalogJob(0))).toBe('Iași, la birou')
    expect(placeOf({ city: '', work_mode: '' })).toBe('Locație nespecificată')
    expect(tagline(catalogJob(0))).toBe('DevOps, Date · Senior')
    expect(lookFor({ must_have: ['Linux'], nice_to_have: ['AWS'] })).toEqual(['Experiență cu Linux.', 'AWS e un avantaj.'])
    expect(marketNote(sampleProfile({ work_modes: ['hybrid', 'onsite'] }))).toBe('Din profilul tău: Iași, Romania · hibrid sau la birou · fără relocare.')
    expect(marketNote(sampleProfile({ city: '', work_modes: [], open_to_relocation: true }))).toBe('Din profilul tău: Romania · cu relocare.')
  })

  it('says when and where a job was posted in Romanian', () => {
    setLocale('ro')
    const now = new Date('2026-11-26T12:00:00Z')
    expect(['2026-11-26T08:00:00Z', '2026-11-25T08:00:00Z', '2026-11-23T08:00:00Z', '2026-11-05T08:00:00Z', '2026-09-01T08:00:00Z'].map(at => postedAgo(at, now)))
      .toEqual(['azi', 'ieri', 'acum 3 zile', 'acum 21 de zile', '1 sept.'])
    const imported = catalogJob(1, { source: 'import', origin: 'greenhouse', url: 'https://job-boards.eu.greenhouse.io/x/jobs/1', posted_at: '2026-11-25T09:00:00Z' })
    expect(sourceLine(imported, now)).toBe('de pe Greenhouse · publicat ieri')
    expect(sourceLine(catalogJob(0))).toBe('de pe www.linkedin.com')
    expect(sourceLine(catalogJob(1))).toBe('Anunț de exemplu, nu unul real')
    expect(sourceLine(catalogJob(1, { sample: false }))).toBe('Din catalog')
    expect(sourceLine(catalogJob(1, { source: 'link', added_by_me: true }))).toBe('Adăugat de tine · dintr-un link')
    expect(sourceLine(catalogJob(1, { source: 'text', added_by_me: true }))).toBe('Adăugat de tine · din text lipit')
    expect(levelLabel('lead')).toBe('Lead')
  })
})
