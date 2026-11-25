import type { EngineJob as Job } from './job'
import { matchJob } from '@/features/jobs/match'
import type { Profile } from '@/features/profile/api/profile.schema'
import { knownSkills } from '@/features/profile/skills'
import type { PrepQuestions } from '@/features/cv/api/cv.schema'
import { capitalize, joinList, pastPhrase } from './text'

const EXAMPLES: Record<string, string> = {
  'terraform': 'Wrote Terraform modules for our staging environments, cutting setup from 2 days to 30 minutes',
  'snowflake': 'Managed roles and warehouses in Snowflake for a team of 20 analysts',
  'rbac': 'Set up RBAC roles for 15 teams and removed shared admin accounts',
  'infrastructure as code': 'Moved 40 servers to Infrastructure as Code, so a rebuild now takes minutes',
  'kubernetes': 'Ran a Kubernetes cluster with 30 services and cut deploy time by half',
  'aws': 'Moved 12 services to AWS and reduced hosting cost by 20%',
  'java': 'Built a Java service that handles 5,000 requests per minute',
  'algorithms': 'Replaced a slow search with a better algorithm, cutting response time from 4s to 300ms',
  'distributed systems': 'Worked on a queue-based system that processes 1M events a day',
  'testing': 'Added automated tests for the checkout flow and caught 30 bugs before release',
  'windows server': 'Administered 12 Windows Servers and automated monthly patching',
  'active directory': 'Managed Active Directory for 150 users, including groups and policies',
  'networking': 'Set up VLANs and VPN access for 3 offices',
  'backup': 'Set up nightly backups and tested restores every month',
  'virtualization': 'Ran 25 virtual machines on VMware and moved 8 of them to new hosts',
}

export const exampleFor = (skill: string) => EXAMPLES[skill.toLowerCase()] ?? `Used ${skill} to ..., which improved ... by ...`

/**
 * What Prepare your CV asks: the job's keywords the profile does not show yet (in its skills, its
 * tools or a highlight that proves one), and the job's key responsibilities.
 */
export function prepQuestions(job: Job, profile: Pick<Profile, 'skills' | 'experience' | 'projects'>): PrepQuestions {
  const { matched, related, missing, score } = matchJob(job, knownSkills(profile))
  return {
    job_id: job.id,
    base_score: score,
    covered: matched,
    related: related.map(({ skill, via, category }) => ({
      skill,
      via,
      note: `You use ${joinList(via)}, a ${category} tool like ${skill}. Your CV shows your ${category} experience, without naming ${skill}.`,
    })),
    gaps: missing.map(skill => ({
      skill,
      ask: `Have you used ${skill}? Tell us where and what you did. Numbers help: how many, how much, how fast.`,
      example: `e.g. ${exampleFor(skill)}`,
    })),
    responsibilities: job.key_responsibilities
      .filter(index => job.responsibilities[index])
      .map((index) => {
        const phrase = pastPhrase(job.responsibilities[index]!)
        return { index, text: job.responsibilities[index]!, question: `Have you ${phrase}?`, prefill: capitalize(phrase) }
      }),
  }
}
