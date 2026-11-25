import { describe, expect, it } from 'vitest'
import { isLink, readFromLine, siteName, unreadableMessage } from './reading'

describe('reading an ad', () => {
  it('names known sites and accepts only full links', () => {
    expect(siteName('https://www.linkedin.com/jobs/view/1')).toBe('LinkedIn')
    expect(siteName('https://ro.glassdoor.com/job')).toBe('Glassdoor')
    expect(siteName('https://jobs.example.org/a')).toBe('jobs.example.org')
    expect(siteName('')).toBe('This site')
    expect(isLink('https://jobs.example.org/a')).toBe(true)
    expect(isLink('jobs.example.org/a')).toBe(false)
    expect(isLink('https://localhost')).toBe(false)
  })

  it('explains each reason a page could not be read', () => {
    const url = 'https://www.linkedin.com/jobs/view/1'
    expect(unreadableMessage('login_required', url)).toBe('LinkedIn shows job pages only to signed-in members, so we cannot read this one. Copy the ad from LinkedIn and paste its text; we keep the link with it.')
    expect(unreadableMessage('no_job_data', url)).toMatch(/^We found no job ad on this page/)
    expect(unreadableMessage('fetch_failed', url)).toMatch(/^The page did not answer/)
    expect(unreadableMessage('too_large', url)).toMatch(/^The page is too big/)
    expect(unreadableMessage('blocked_address', url)).toMatch(/private or local address/)
    expect(unreadableMessage('unknown', url)).toMatch(/^Something on this page stopped us/)
  })

  it('says where a draft was read from', () => {
    expect(readFromLine({ source: 'link', read_from: 'json_ld', host: 'jobs.lever.co', url: '' })).toBe('Read from the job data on jobs.lever.co')
    expect(readFromLine({ source: 'link', read_from: 'page_text', host: '', url: 'https://acme.ro/careers/1' })).toBe('Read from the page on acme.ro')
    expect(readFromLine({ source: 'link', read_from: 'ats_api', host: 'boards.greenhouse.io', url: '' })).toBe('Read from the job board\'s data on boards.greenhouse.io')
    expect(readFromLine({ source: 'link', read_from: 'something_new', host: 'a.example', url: '' })).toBe('Read from the page on a.example')
    expect(readFromLine({ source: 'text', read_from: 'text', host: '', url: '' })).toBe('Read from the text you pasted')
  })
})
