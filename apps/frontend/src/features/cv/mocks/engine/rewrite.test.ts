import { describe, expect, it } from 'vitest'
import { engineJob } from '@/test/jobs'
import { cleanFact, rewriteBullet, verbPairs } from './rewrite'

const snowflake = engineJob(0)
const bitdefender = engineJob(1)

describe('rewriteBullet', () => {
  it('uses the job’s terms and verbs, then adds the tools used in that role', () => {
    const used = new Set<string>()
    const out = rewriteBullet('Built and maintained deployment pipelines for internal services.', 0, {
      job: snowflake,
      matched: ['SQL', 'Git', 'CI/CD', 'Monitoring', 'Python'],
      tools: ['Linux', 'Docker', 'CI/CD', 'Python', 'Git'],
      used,
    })
    expect(out).toBe('Developed and maintained CI/CD pipelines for internal services, using Git.')
    expect(used.has('git')).toBe(true)
  })

  it('adds a tool only once per role, and with "with" on odd bullets', () => {
    const used = new Set<string>(['git'])
    const out = rewriteBullet('Automated server setup and configuration with scripts.', 1, {
      job: snowflake, matched: ['Git', 'Python'], tools: ['Git', 'Python'], used,
    })
    expect(out).toBe('Automated server setup and configuration with scripts, using Python.')
    const odd = rewriteBullet('Wrote reports for the team.', 1, { job: snowflake, matched: ['SQL'], tools: ['SQL'], used: new Set() })
    expect(odd).toBe('Wrote reports for the team with SQL.')
  })

  it('keeps a bullet whose verb fits no group, and adds no tool without a role', () => {
    const out = rewriteBullet('Set up backups and checked that they ran.', 0, { job: bitdefender, matched: ['Linux'], tools: [], used: null })
    expect(out).toBe('Set up backups and checked that they ran.')
  })

  it('takes one verb when the bullet already has two', () => {
    const pairs = verbPairs(bitdefender)
    expect(pairs[0]).toEqual(['designed', 'maintained'])
    const out = rewriteBullet('Maintained servers and user accounts.', 0, { job: bitdefender, matched: [], tools: [], used: null })
    expect(out).toBe('Maintained servers and user accounts.')
    const two = rewriteBullet('Managed and fixed hosts.', 0, { job: bitdefender, matched: [], tools: [], used: null })
    expect(two).toBe('Maintained and fixed hosts.')
  })
})

describe('cleanFact', () => {
  it('capitalises, ends with one full stop and names the skill', () => {
    expect(cleanFact('  wrote   terraform modules...', 'Terraform')).toBe('Wrote terraform modules.')
    expect(cleanFact('set up roles for 15 teams', 'RBAC')).toBe('Set up roles for 15 teams, using RBAC.')
    expect(cleanFact('ran the on-call rota', null)).toBe('Ran the on-call rota.')
    expect(cleanFact('   ', 'SQL')).toBe('')
  })
})
