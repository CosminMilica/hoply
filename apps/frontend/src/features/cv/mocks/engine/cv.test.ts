import { describe, expect, it } from 'vitest'
import { engineJob } from '@/test/jobs'
import { sampleProfile } from '@/test/profile-service'
import type { CvRequest } from '../../api/cv.schema'
import { atsChecks, cvDates, cvFileName, generateCv, rewritePreview, yearsOfExperience } from './cv'

const job = engineJob(0)
const now = new Date('2026-11-25T10:00:00Z')
const request = (overrides: Partial<CvRequest> = {}): CvRequest => ({ job_id: job.id, skip_roles: [], focus: [], confirmed: [], ...overrides })

describe('generateCv', () => {
  it('keeps the person’s headline and puts the target title in the summary', () => {
    const cv = generateCv(sampleProfile(), job, request(), now)
    expect(cv.tailored.headline).toBe('DevOps Engineer | SQL, Git, CI/CD')
    expect(cv.profile.headline).toBe('DevOps Engineer')
    expect(cv.tailored.summary).toBe(
      'DevOps Engineer with 6 years of experience, strongest in SQL, Git and CI/CD. I like reliable systems, clear documentation and calm on-call weeks. '
      + 'Looking to join Capgemini Romania as Snowflake Administrator DevOps and work on its enterprise Knowledge Platform on Snowflake.',
    )
    expect(cv.file_name).toBe('Andrei_Popescu_CV_Capgemini_Romania.pdf')
    expect(cv.tailored.name).toBe('Andrei Popescu')
    expect(cv.tailored.contact).toEqual(['andrei@example.com', '+40 700 000 000', 'Iași, Romania'])
    expect(cv.tailored.skills.slice(0, 5)).toEqual(['SQL', 'Git', 'CI/CD', 'Monitoring', 'Python'])
    expect(cv.profile.skills).toEqual(sampleProfile().skills)
  })

  it('prints month dates, results and the education title', () => {
    const cv = generateCv(sampleProfile(), job, request(), now)
    expect(cv.profile.experience.map(role => role.meta)).toEqual(['Verde Cloud | Iași | 03/2022 – Present', 'Bonbon Labs | Iași | 09/2020 – 02/2022'])
    expect(cv.profile.experience[1]!.bullets[1]).toBe('Set up backups and checked that they ran every night, restored every test backup within an hour.')
    expect(cv.profile.education).toEqual([{ degree: 'BSc Computer Science', meta: 'University of Iași | 2016 – 2020' }])
    const byLevel = generateCv(sampleProfile({ education: [{ id: 'e', level: 'master', field: 'Data', degree: '', school: '', start_year: '2021', end_year: '' }] }), job, request(), now)
    expect(byLevel.profile.education).toEqual([{ degree: 'Master’s degree in Data', meta: '2021 – Present' }])
  })

  it('uses the facts remembered in the profile, wherever they are', () => {
    const profile = sampleProfile({
      additional_info: ['HashiCorp Terraform Associate, 2025'],
      projects: [{ id: 'p1', name: 'Home lab', role: 'Maintainer', url: 'github.com/a/lab', start: '2023-06', end: '', tools: ['Docker', 'Terraform'], bullets: [{ text: 'Provisioned 12 containers with Terraform', result: '', skills: ['Terraform'] }] }],
      certifications: [{ id: 'c1', name: 'CKA', issuer: 'CNCF', issued: '2024-05', expires: '2027-05', credential_id: 'X1', url: '' }],
    })
    profile.experience[1]!.bullets.push({ text: 'managed warehouses for 20 analysts', result: '', skills: ['Snowflake'] })
    const cv = generateCv(profile, job, request({ confirmed: [1, 3, 99] }), now)
    expect(cv.highlights).toEqual(expect.arrayContaining(['Snowflake', 'Terraform']))
    expect(cv.tailored.experience[1]!.bullets[0]).toBe('Administered warehouses for 20 analysts.')
    expect(cv.tailored.projects).toEqual([{ name: 'Home lab', meta: 'Maintainer | github.com/a/lab | 06/2023 – Present', bullets: ['Provisioned 12 containers with Terraform.'] }])
    expect(cv.tailored.certifications).toEqual(['CKA | CNCF | Issued 05/2024 | Expires 05/2027 | ID X1'])
    expect(cv.tailored.additional).toEqual(['HashiCorp Terraform Associate, 2025.'])
    expect(cv.profile.additional).toEqual(['HashiCorp Terraform Associate, 2025.'])
    expect(cv.ats.checks.find(check => check.label.startsWith('Job responsibilities'))).toMatchObject({ ok: true, label: 'Job responsibilities mirrored: 2 of 4' })
    expect(cv.ats.checks.find(check => check.label.startsWith('Keywords from the job'))!.label).toBe('Keywords from the job: 7 of 9')
  })

  it('leaves out skipped roles and follows the chosen focus', () => {
    const cv = generateCv(sampleProfile(), job, request({ skip_roles: ['r2'], focus: ['Python', 'Snowflake'] }), now)
    expect(cv.tailored.experience).toHaveLength(1)
    expect(cv.profile.experience).toHaveLength(2)
    expect(cv.tailored.headline).toBe('DevOps Engineer | Python')
  })

  it('keeps the whole profile as it was', () => {
    const profile = sampleProfile()
    const cv = generateCv(profile, job, request(), now)
    profile.name = 'Someone Else'
    profile.experience[0]!.role = 'Changed'
    expect(cv.snapshot.name).toBe('Andrei Popescu')
    expect(cv.snapshot.experience[0]!.role).toBe('DevOps Engineer')
    expect(cv.snapshot.city).toBe('Iași')
  })

  it('copes with a thin profile', () => {
    const cv = generateCv(sampleProfile({ name: '', headline: '', about: 'Short.', skills: [], experience: [], email: '', phone: '', city: '', country: '' }), job, request(), now)
    expect(cv.tailored.summary.startsWith('Engineer with hands-on experience.')).toBe(true)
    expect(cv.preview).toBeNull()
    expect(cv.file_name).toBe('CV_Capgemini_Romania.pdf')
    const failed = cv.ats.checks.filter(check => !check.ok).map(check => check.label)
    expect(failed).toEqual(expect.arrayContaining([
      'Email and phone in the body, not in a header', 'Location in the header', 'Dates with month and year', 'Every bullet starts with an action verb',
    ]))
  })
})

describe('dates', () => {
  it('prints months and counts whole years from the earliest start', () => {
    expect(cvDates('2022-03', '')).toBe('03/2022 – Present')
    expect(cvDates('2022-03', '2023-11')).toBe('03/2022 – 11/2023')
    expect(cvDates('', '2023-11')).toBe('')
    expect(yearsOfExperience(sampleProfile(), new Date('2026-08-31'))).toBe(5)
    expect(yearsOfExperience(sampleProfile(), new Date('2026-09-01'))).toBe(6)
    expect(yearsOfExperience(sampleProfile({ experience: [] }), now)).toBe(0)
  })
})

describe('atsChecks', () => {
  it('passes a CV that has it all, and fails undated roles', () => {
    const profile = sampleProfile()
    const document = {
      name: '', headline: 'x', contact: [], links: [], summary: `Snowflake Administrator DevOps using ${job.keywords.join(', ')}.`,
      skills: job.keywords, experience: [{ role: 'r', meta: '', bullets: ['Built 3 things.'] }], projects: [], education: [],
      certifications: [], languages: '', additional: [],
    }
    expect(atsChecks(document, job, profile, request({ confirmed: [1] })).every(check => check.ok)).toBe(true)
    const undated = sampleProfile({ experience: [{ ...profile.experience[0]!, start: '' }] })
    expect(atsChecks(document, job, undated, request({ confirmed: [1] })).find(check => !check.ok)!.label).toBe('Dates with month and year')
  })
})

describe('rewritePreview', () => {
  it('rewrites the first highlight of the first role kept', () => {
    expect(rewritePreview(sampleProfile(), job, request())).toEqual({
      from: 'Built and maintained deployment pipelines for internal services.',
      to: 'Developed and maintained CI/CD pipelines for internal services, using Git.',
    })
    expect(rewritePreview(sampleProfile(), job, request({ skip_roles: ['r1'] }))!.from).toBe('Maintained servers and user accounts for a small office.')
  })
})

describe('cvFileName', () => {
  it('starts with the name, with anything but letters and digits as underscores', () => {
    expect(cvFileName('Andrei Popescu', 'Capgemini Romania')).toBe('Andrei_Popescu_CV_Capgemini_Romania.pdf')
    expect(cvFileName(' Ana-Maria  Ștefan ', 'A&B Co.')).toBe('Ana_Maria_tefan_CV_A_B_Co_.pdf')
    expect(cvFileName('  ', 'Capgemini Romania')).toBe('CV_Capgemini_Romania.pdf')
  })
})
