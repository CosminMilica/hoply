import type { JobDraft, UnreadableReason } from './api/jobs.schema'
import { hostOf } from './match'

const SITES: [RegExp, string][] = [[/(^|\.)linkedin\.com$/i, 'LinkedIn'], [/(^|\.)glassdoor\./i, 'Glassdoor']]

/** The site's name for a known host, else the host itself. */
export function siteName(url: string): string {
  const host = hostOf(url.trim()).replace(/^www\./i, '')
  return SITES.find(([pattern]) => pattern.test(host))?.[1] ?? (host || 'This site')
}

/** Why a link could not be read, in the person's words, with what to do instead. */
export function unreadableMessage(reason: UnreadableReason | 'unknown', url: string): string {
  const site = siteName(url)
  switch (reason) {
    case 'login_required':
      return `${site} shows job pages only to signed-in members, so we cannot read this one. Copy the ad from ${site} and paste its text; we keep the link with it.`
    case 'no_job_data':
      return 'We found no job ad on this page. It may be a list of jobs, or a page that loads the ad later. Open the ad itself and copy its link, or paste its text.'
    case 'fetch_failed':
      return 'The page did not answer, or it is not a web page. Check the link and try again later, or paste the ad’s text.'
    case 'too_large':
      return 'The page is too big for us to read. Paste the ad’s text instead.'
    case 'blocked_address':
      return 'This link points to a private or local address, which we do not open. Use the public link to the ad, or paste its text.'
    default:
      return 'Something on this page stopped us from reading it. Paste the ad’s text instead.'
  }
}

/** Where the draft came from, as the preview says it. */
export function readFromLine(draft: Pick<JobDraft, 'source' | 'read_from' | 'host' | 'url'>): string {
  if (draft.source !== 'link') return 'Read from the text you pasted'
  const host = draft.host || hostOf(draft.url)
  if (draft.read_from === 'json_ld') return `Read from the job data on ${host}`
  if (draft.read_from === 'ats_api') return `Read from the job board's data on ${host}`
  return `Read from the page on ${host}`
}

export const isLink = (value: string) => /^https?:\/\/[^\s/]+\.[^\s]+$/i.test(value.trim())
export const MIN_AD_LENGTH = 80
