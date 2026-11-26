import { afterEach, describe, expect, it, vi } from 'vitest'
import { bookmarklet, sentJob } from './sendToHoply'

/** Runs the bookmark on the current document and returns the link it opened. */
function run(origin = 'http://app.localhost') {
  const open = vi.spyOn(window, 'open').mockReturnValue(null)
  const href = bookmarklet(origin)
  expect(href.startsWith('javascript:')).toBe(true)
  // eslint-disable-next-line no-new-func -- the bookmark's own code, run as the browser would
  new Function(decodeURIComponent(href.slice('javascript:'.length)))()
  expect(open).toHaveBeenCalledWith(expect.any(String), '_blank')
  return new URL(String(open.mock.calls.at(-1)![0]))
}

afterEach(() => {
  document.body.innerHTML = ''
  window.getSelection()?.removeAllRanges()
  vi.restoreAllMocks()
})

describe('Send to Hoply', () => {
  it('sends a LinkedIn ad with its title and company to Add a job', () => {
    document.body.innerHTML = `<main><h1 class="job-details-jobs-unified-top-card__job-title">DevOps Engineer</h1>
      <div class="job-details-jobs-unified-top-card__company-name"> spektr </div>
      <div id="job-details">About the job\nWe run Kubernetes on AWS.</div></main>`
    const opened = run()
    expect(opened.origin + opened.pathname).toBe('http://app.localhost/jobs/add')
    expect(sentJob(opened.hash)).toEqual({ text: 'About the job\nWe run Kubernetes on AWS.', url: window.location.href, title: 'DevOps Engineer', company: 'spektr' })
  })

  it('sends the selected text when there is one, and any page’s main text otherwise', () => {
    document.body.innerHTML = '<main><p id="part">Only this part.</p><p>And the rest.</p></main>'
    const range = document.createRange()
    range.selectNodeContents(document.getElementById('part')!)
    window.getSelection()!.addRange(range)
    expect(sentJob(run().hash)?.text).toBe('Only this part.')
    window.getSelection()!.removeAllRanges()
    const whole = sentJob(run().hash)!
    expect([whole.text.includes('And the rest.'), whole.title, whole.company]).toEqual([true, '', ''])
  })

  it('reads only the hash a bookmark sent', () => {
    expect(sentJob('')).toBeNull()
    expect(sentJob('#section')).toBeNull()
    expect(sentJob(`#text=${'a'.repeat(60_000)}&title=${'t'.repeat(200)}`)).toMatchObject({ text: 'a'.repeat(50_000), title: 't'.repeat(160), url: '', company: '' })
  })
})
