import { describe, expect, it } from 'vitest'
import { countKeywordHits, splitKeywords } from './highlight'

describe('splitKeywords', () => {
  it('marks every keyword, ignoring case', () => {
    expect(splitKeywords('Ran sql and CI/CD jobs', ['SQL', 'CI/CD'])).toEqual([
      { text: 'Ran ', hit: false }, { text: 'sql', hit: true }, { text: ' and ', hit: false }, { text: 'CI/CD', hit: true }, { text: ' jobs', hit: false },
    ])
    expect(splitKeywords('plain', [])).toEqual([{ text: 'plain', hit: false }])
    expect(splitKeywords('', [])).toEqual([])
    expect(countKeywordHits('Git, git and GitHub', ['Git'])).toBe(3)
  })
})
