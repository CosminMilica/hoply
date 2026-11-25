export interface TextPart {
  text: string
  hit: boolean
}

const escape = (value: string) => value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')

/** Splits text into the parts that are one of the keywords (case-insensitive) and the rest. */
export function splitKeywords(text: string, keywords: string[]): TextPart[] {
  if (!keywords.length) return text ? [{ text, hit: false }] : []
  const pattern = new RegExp(`(${keywords.map(escape).join('|')})`, 'gi')
  return text
    .split(pattern)
    .map((part, index) => ({ text: part, hit: index % 2 === 1 }))
    .filter(part => part.text !== '')
}

export function countKeywordHits(text: string, keywords: string[]): number {
  return splitKeywords(text, keywords).filter(part => part.hit).length
}
