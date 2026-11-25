import { describe, expect, it } from 'vitest'
import { engineJob } from '@/test/jobs'
import { sampleProfile } from '@/test/profile-service'
import { exampleFor, prepQuestions } from './questions'

describe('prepQuestions', () => {
  it('asks about the missing keywords and the key responsibilities', () => {
    const questions = prepQuestions(engineJob(0), sampleProfile({ skills: ['Python', 'SQL', 'Git', 'CI/CD', 'Monitoring'], experience: [] }))
    expect(questions.base_score).toBe(56)
    expect(questions.covered).toEqual(['SQL', 'Git', 'CI/CD', 'Monitoring', 'Python'])
    expect(questions.gaps.map(gap => gap.skill)).toEqual(['Snowflake', 'Terraform', 'Infrastructure as Code', 'RBAC'])
    expect(questions.gaps[0]!.example).toBe('e.g. Managed roles and warehouses in Snowflake for a team of 20 analysts')
    expect(questions.responsibilities.map(item => item.index)).toEqual([0, 1, 2, 3])
    expect(questions.responsibilities[3]).toMatchObject({
      question: 'Have you developed and maintained Terraform modules and reusable deployment templates?',
      prefill: 'Developed and maintained Terraform modules and reusable deployment templates',
    })
  })

  it('does not ask again about a skill the profile shows in its tools or a highlight', () => {
    const profile = sampleProfile()
    profile.experience[0]!.bullets.push({ text: 'Managed warehouses', result: '', skills: ['Snowflake'] })
    profile.projects = [{ id: 'p', name: 'Lab', role: '', url: '', start: '', end: '', tools: ['Terraform'], bullets: [] }]
    const questions = prepQuestions(engineJob(0), profile)
    expect(questions.gaps.map(gap => gap.skill)).toEqual(['Infrastructure as Code', 'RBAC'])
    expect(questions.covered).toContain('Snowflake')
    expect(questions.base_score).toBe(78)
  })

  it('falls back to a generic example', () => {
    expect(exampleFor('Rust')).toBe('Used Rust to ..., which improved ... by ...')
  })
})
