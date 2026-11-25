import type { EngineJob as Job } from './job'
import { capitalize, escapeRegExp, hasWord, joinList, lc, pastPhrase } from './text'

const VERB_GROUPS = [
  ['built', 'developed', 'created', 'shipped'],
  ['fixed', 'resolved'],
  ['maintained', 'managed', 'ran', 'administered'],
  ['wrote', 'authored'],
  ['monitored', 'tracked'],
]
const TERMS = [
  ['deployment pipelines', 'release pipelines', 'CI/CD pipelines'],
  ['servers', 'hosts', 'instances'],
  ['incidents', 'outages'],
  ['interface', 'UI', 'user interface'],
  ['platform', 'web app', 'application', 'app'],
  ['websites', 'web apps', 'sites'],
]
/** A tool is only added to a bullet whose wording fits it. */
const AFFINITY: Record<string, RegExp> = {
  'sql': /report|quer|data|dashboard/i,
  'figma': /design|screen|page|flow/i,
  'css': /style|page|screen|design|website|interface/i,
  'react': /interface|ui|component|feature|app|screen|page|platform/i,
  'typescript': /interface|ui|component|feature|app|screen|platform|api|service/i,
  'javascript': /website|site|bug|feature|page|interface/i,
  'node.js': /api|service|server|endpoint|quer/i,
  'graphql': /api|endpoint/i,
  'next.js': /page|app|interface|site/i,
  'storybook': /component/i,
  'testing': /test|check|bug/i,
  'docker': /deploy|container|service|server/i,
  'ci/cd': /pipeline|deploy|release/i,
  'linux': /server|setup|config|host|account|system/i,
  'python': /script|automat|report|data|tool|setup/i,
  'git': /release|code|deploy|version|pipeline/i,
  'bash': /script|automat|setup|server|backup/i,
  'monitoring': /monitor|incident|alert|uptime|call/i,
  'terraform': /provision|infrastructure|environment/i,
  'kubernetes': /container|cluster|deploy/i,
  'aws': /cloud|infrastructure|deploy|service/i,
}
const NO_INJECT = ['monitoring', 'testing', 'networking', 'backup', 'virtualization', 'algorithms', 'distributed systems']

type JobText = Pick<Job, 'title' | 'about' | 'responsibilities' | 'requirements'>

export function jobText(job: JobText): string {
  return lc(`${job.title} ${job.about} ${job.responsibilities.join(' ')} ${job.requirements.join(' ')}`)
}

/** The first verb of each responsibility and, after "and", the second one, in the past tense. */
export function verbPairs(job: Pick<Job, 'responsibilities'>): string[][] {
  return job.responsibilities.map((line) => {
    const words = pastPhrase(line).split(' ')
    return [words[0]!, ...(words[1] === 'and' && words[2] ? [words[2]] : [])]
  })
}

export interface RewriteContext {
  job: JobText
  /** The job keywords the person has. */
  matched: string[]
  /** The tools of the role the bullet belongs to. */
  tools: string[]
  /** Tools already added to this role's bullets; null adds none. */
  used: Set<string> | null
}

/** Rewrites a bullet in the job's words: its terms, its verbs, then the tools the person used in that role. */
export function rewriteBullet(text: string, index: number, { job, matched, tools, used }: RewriteContext): string {
  const jd = jobText(job)
  let out = text
  for (const group of TERMS) {
    const jdTerm = group.find(word => hasWord(jd, word))
    if (!jdTerm) continue
    for (const word of group) {
      if (word !== jdTerm && hasWord(out, word) && !hasWord(out, jdTerm)) {
        out = out.replace(new RegExp(`\\b${escapeRegExp(word)}\\b`, 'i'), jdTerm)
      }
    }
  }
  const lead = out.match(/^(\w+)(\s[\s\S]*)$/)
  if (lead) {
    const verb = lead[1]!.toLowerCase()
    const group = VERB_GROUPS.find(candidate => candidate.includes(verb))
    const pair = group && verbPairs(job).map(verbs => verbs.filter(v => group.includes(v))).find(verbs => verbs.length > 0)
    if (pair) {
      const verbs = out.split(' ')[1] === 'and' ? [pair[0]!] : pair
      if (verbs[0] !== verb) out = capitalize(joinList(verbs)) + lead[2]
    }
  }
  if (!used) return out
  const roleTools = tools.map(lc)
  const fits = (keyword: string) => AFFINITY[lc(keyword)]?.test(text) ?? true
  const added = matched
    .filter(k => !NO_INJECT.includes(lc(k)) && roleTools.includes(lc(k)) && fits(k) && !lc(out).includes(lc(k)) && !used.has(lc(k)))
    .slice(0, 2)
  if (!added.length) return out
  for (const keyword of added) used.add(lc(keyword))
  const joiner = / with /.test(out) || index % 2 === 0 ? ', using ' : ' with '
  return `${out.replace(/[.]$/, '')}${joiner}${joinList(added)}.`
}

/** The person's own words as a CV bullet: capitalised, one full stop, the skill named if they did not. */
export function cleanFact(text: string, keyword: string | null): string {
  let fact = text.trim().replace(/\s+/g, ' ')
  if (!fact) return ''
  fact = capitalize(fact).replace(/[.]*$/, '')
  if (keyword && !hasWord(fact, keyword)) fact += `, using ${keyword}`
  return `${fact}.`
}
