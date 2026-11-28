import { describe, expect, it } from 'vitest'
import { setLocale } from '@/features/common/i18n'
import { authorshipLine, letterFailure, modelLabel } from './authorship'

const cv = { written_by: 'claude' as const, model: 'claude-sonnet-5-5', refused: 0, rules_reason: '' as const, limit_until: null }

describe('authorshipLine', () => {
  it('names Claude and its model, and the sentences the checks rewrote', () => {
    expect(authorshipLine(cv)).toBe('Written by Claude (Sonnet 5.5) from your profile.')
    expect(authorshipLine({ ...cv, refused: 1 })).toBe('Written by Claude (Sonnet 5.5) from your profile. One sentence that went beyond your profile was rewritten by rules.')
    expect(authorshipLine({ ...cv, refused: 3, model: '' })).toBe('Written by Claude from your profile. 3 sentences that went beyond your profile were rewritten by rules.')
  })

  it('says why the rules wrote it', () => {
    const rules = { ...cv, written_by: 'rules' as const, model: '' }
    expect(authorshipLine({ ...rules, rules_reason: 'no_token' })).toBe('Written without AI. Claude is not connected yet.')
    expect(authorshipLine({ ...rules, rules_reason: 'login_needed' })).toBe('Written without AI. Claude needs to be connected again.')
    expect(authorshipLine({ ...rules, rules_reason: 'claude_failed' })).toBe('Written without AI. Claude could not answer this time.')
    expect(authorshipLine({ ...rules, rules_reason: 'usage_limit', limit_until: '2026-11-25T16:20:00Z' })).toMatch(/^Written without AI\. Claude reached its usage limit\. It is back at /)
    expect(authorshipLine({ ...rules, rules_reason: '' })).toBe('Written without AI.')
    expect(authorshipLine({ ...cv, written_by: null })).toBe('')
  })

  it('says who wrote it in Romanian when the app is in Romanian', () => {
    setLocale('ro')
    expect(authorshipLine(cv)).toBe('Scris de Claude (Sonnet 5.5) din profilul tău.')
    expect(authorshipLine({ ...cv, refused: 1 })).toBe('Scris de Claude (Sonnet 5.5) din profilul tău. O propoziție care depășea profilul tău a fost rescrisă de reguli.')
    expect(authorshipLine({ ...cv, refused: 21, model: '' })).toBe('Scris de Claude din profilul tău. 21 de propoziții care depășeau profilul tău au fost rescrise de reguli.')
    const rules = { ...cv, written_by: 'rules' as const, model: '' }
    expect(authorshipLine({ ...rules, rules_reason: 'usage_limit', limit_until: '2026-11-25T16:20:00Z' })).toMatch(/^Scris fără AI\. Claude și-a atins limita de utilizare\. Revine la ora \d{2}:\d{2}\.$/)
    expect(letterFailure({ reason: 'off', limit_until: null })).toBe('Serviciul e setat să scrie fără Claude, iar o scrisoare are nevoie de Claude.')
    expect(letterFailure({ reason: 'unknown', limit_until: null })).toBe('Claude nu a putut răspunde de data asta.')
  })

  it('shortens Claude model names', () => {
    expect(modelLabel('claude-opus-5-5')).toBe('Opus 5.5')
    expect(modelLabel('claude-fable-5')).toBe('Fable 5')
    expect(modelLabel('other')).toBe('other')
  })
})
