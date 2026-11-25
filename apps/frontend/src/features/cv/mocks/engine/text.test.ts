import { describe, expect, it } from 'vitest'
import { capitalize, hasWord, joinList, pastOf, pastPhrase } from './text'

describe('text helpers', () => {
  it('joins lists the way a sentence does', () => {
    expect(joinList([])).toBe('')
    expect(joinList(['SQL'])).toBe('SQL')
    expect(joinList(['SQL', 'Git'])).toBe('SQL and Git')
    expect(joinList(['SQL', 'Git', 'Bash'])).toBe('SQL, Git and Bash')
  })

  it('finds whole words only, case-insensitively, including ones with symbols', () => {
    expect(hasWord('Built CI/CD pipelines', 'ci/cd')).toBe(true)
    expect(hasWord('Ran SQL queries', 'SQL')).toBe(true)
    expect(hasWord('Ran MySQL queries', 'SQL')).toBe(false)
  })

  it('turns a responsibility into a past-tense phrase', () => {
    expect(pastPhrase('Build and ship features in our React web app.')).toBe('built and shipped features in a React web app')
    expect(pastPhrase('Monitor your services and fix operational issues quickly.')).toBe('monitored the services and fixed operational issues quickly')
    expect(pastPhrase('Kubernetes upgrades.')).toBe('kubernetes upgrades')
    expect(pastOf('Troubleshoot')).toBe('troubleshot')
    expect(capitalize('built it')).toBe('Built it')
  })
})
