import { describe, expect, it } from 'vitest'
import { jobSchema } from '@/features/jobs/api/jobs.schema'
import { askedLine, skillGaps } from './gaps'

const job = (id: string, must_have: string[], nice_to_have: string[]) => jobSchema.parse({ id, title: `Job ${id}`, company: 'Acme', must_have, nice_to_have })

describe('skillGaps', () => {
  it('counts each skill once per job under any of its names, must-haves first', () => {
    const jobs = [
      job('a', ['Argo CD', 'Terraform', 'Docker'], ['GCP', 'Azure']),
      job('b', ['ArgoCD'], ['terraform', 'Snowflake']),
      job('c', ['Terraform', 'Snowflake'], ['Snowflake']),
    ]
    const { missing, close } = skillGaps(jobs, ['FluxCD', 'Docker', 'Git'])
    expect(missing.map(gap => [gap.skill, gap.jobs.map(item => item.id).join(''), gap.must])).toEqual([
      ['Terraform', 'abc', 2],
      ['Snowflake', 'bc', 1],
      ['Azure', 'a', 0],
      ['GCP', 'a', 0],
    ])
    expect(close).toEqual([{
      skill: 'Argo CD',
      via: ['FluxCD'],
      category: 'GitOps',
      must: 2,
      jobs: [{ id: 'a', title: 'Job a', company: 'Acme', must: true }, { id: 'b', title: 'Job b', company: 'Acme', must: true }],
    }])
    expect(missing[1]!.jobs.map(item => item.must)).toEqual([false, true])

    expect(askedLine(missing[0]!, 3)).toBe('Asked in 3 of 3 saved jobs, a must-have in 2.')
    expect(askedLine(close[0]!, 3)).toBe('Asked in 2 of 3 saved jobs, always a must-have.')
    expect(askedLine(missing[3]!, 3)).toBe('Asked in 1 of 3 saved jobs, as a plus.')
    expect(askedLine({ jobs: missing[3]!.jobs, must: 1 }, 1)).toBe('Asked in 1 of 1 saved job, always a must-have.')
  })
})
