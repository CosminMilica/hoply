import { describe, expect, it } from 'vitest'
import { adLanguage, HEADINGS } from './language'

describe('the CV language', () => {
  it('reads an ad as Romanian when its words are, and as English otherwise', () => {
    expect(adLanguage('Contabil primar. Cerințe: experiență de minimum 2 ani în contabilitate și cunoștințe SAGA. Oferim pachet salarial.')).toBe('ro')
    expect(adLanguage('Accounts Payable Associate. You will process invoices and reconcile vendor statements with the team.')).toBe('en')
    expect(adLanguage('')).toBe('en')
  })

  it('names the sections the way each language does', () => {
    expect(HEADINGS.ro.experience).toBe('Experiență profesională')
    expect(HEADINGS.en.experience).toBe('Experience')
  })
})
