import { t } from '@/features/common/i18n'
import type { JobDraft, UnreadableReason } from './api/jobs.schema'
import { hostOf } from './match'

const SITES: [RegExp, string][] = [[/(^|\.)linkedin\.com$/i, 'LinkedIn'], [/(^|\.)glassdoor\./i, 'Glassdoor']]

/** The site's name for a known host, else the host itself. */
export function siteName(url: string): string {
  const host = hostOf(url.trim()).replace(/^www\./i, '')
  return SITES.find(([pattern]) => pattern.test(host))?.[1] ?? (host || t.value.jobs.reading.thisSite)
}

/** Why a link could not be read, in the person's words, with what to do instead. */
export function unreadableMessage(reason: UnreadableReason | 'unknown', url: string): string {
  const words = t.value.jobs.reading
  switch (reason) {
    case 'login_required':
      return words.loginRequired(siteName(url))
    case 'no_job_data':
      return words.noJobData
    case 'fetch_failed':
      return words.fetchFailed
    case 'too_large':
      return words.tooLarge
    case 'blocked_address':
      return words.blockedAddress
    default:
      return words.unknown
  }
}

/** Where the draft came from, as the preview says it. */
export function readFromLine(draft: Pick<JobDraft, 'source' | 'read_from' | 'host' | 'url'>): string {
  const words = t.value.jobs.reading
  if (draft.source !== 'link') return words.fromText
  const host = draft.host || hostOf(draft.url)
  if (draft.read_from === 'json_ld') return words.fromJobData(host)
  if (draft.read_from === 'ats_api') return words.fromBoard(host)
  return words.fromPage(host)
}

export const isLink = (value: string) => /^https?:\/\/[^\s/]+\.[^\s]+$/i.test(value.trim())
export const MIN_AD_LENGTH = 80
