export const lc = (value: string) => value.toLowerCase()
export const escapeRegExp = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
export const capitalize = (value: string) => value.charAt(0).toUpperCase() + value.slice(1)

export function joinList(items: string[]): string {
  return items.length <= 1 ? items.join('') : `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`
}

export function hasWord(text: string, word: string): boolean {
  return new RegExp(`\\b${escapeRegExp(word)}\\b`, 'i').test(text)
}

const PAST: Record<string, string> = {
  build: 'built', ship: 'shipped', work: 'worked', review: 'reviewed', document: 'documented', add: 'added',
  write: 'wrote', design: 'designed', manage: 'managed', run: 'ran', watch: 'watched', own: 'owned', join: 'joined',
  keep: 'kept', clean: 'cleaned', fix: 'fixed', share: 'shared', maintain: 'maintained', create: 'created',
  develop: 'developed', improve: 'improved', check: 'checked', test: 'tested', administer: 'administered',
  automate: 'automated', implement: 'implemented', monitor: 'monitored', troubleshoot: 'troubleshot',
  support: 'supported', collaborate: 'collaborated', respond: 'responded', participate: 'participated',
  analyze: 'analyzed', mentor: 'mentored', turn: 'turned', install: 'installed', plan: 'planned', apply: 'applied',
  escalate: 'escalated', lead: 'led', harden: 'hardened',
}

export const pastOf = (word: string) => PAST[word.toLowerCase()] ?? word

/** "Build and ship features in our React web app." becomes "built and shipped features in a React web app". */
export function pastPhrase(line: string): string {
  let text = line.replace(/[.]$/, '').replace(/\bour\b/gi, 'a').replace(/\byour\b/gi, 'the')
  text = text.replace(/^(\w+)/, word => pastOf(word)).replace(/ and (\w+)/, (_, word: string) => ` and ${pastOf(word)}`)
  return text.charAt(0).toLowerCase() + text.slice(1)
}
