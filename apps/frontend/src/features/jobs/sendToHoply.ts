/** What the Send to Hoply bookmark sends in the link's hash: the ad's text, its page, and the title and company the page shows. */
export interface SentJob {
  text: string
  url: string
  title: string
  company: string
}

/** The job a Send to Hoply link carries in its hash, or null for any other hash. */
export function sentJob(hash: string): SentJob | null {
  const params = new URLSearchParams(hash.replace(/^#/, ''))
  const text = (params.get('text') ?? '').trim()
  if (!text) return null
  const field = (name: string, limit: number) => (params.get(name) ?? '').trim().slice(0, limit)
  return { text: text.slice(0, 50_000), url: field('url', 2000), title: field('title', 160), company: field('company', 120) }
}

/**
 * The Send to Hoply bookmark. On a job page it opens Add a job at origin with the selected text, else the page's job
 * description, plus the page's link and the title and company it shows (LinkedIn's first, then any page's). It only
 * reads the page and opens a tab; it sends nothing anywhere else.
 */
export function bookmarklet(origin: string): string {
  const script = [
    '(()=>{',
    'const pick=(...s)=>s.map(x=>document.querySelector(x)).find(e=>e&&e.innerText.trim());',
    'const text=String(getSelection()).trim()||(pick("#job-details",".jobs-description__content","[class*=job-description]","[class*=jobDescription]","main","article")||document.body).innerText;',
    'const title=pick(".job-details-jobs-unified-top-card__job-title","h1");',
    'const company=pick(".job-details-jobs-unified-top-card__company-name","[class*=company-name]","[class*=companyName]");',
    'const p=new URLSearchParams({text:text.trim().slice(0,50000),url:location.href});',
    'if(title)p.set("title",title.innerText.trim());',
    'if(company)p.set("company",company.innerText.trim());',
    `open(${JSON.stringify(`${origin}/jobs/add`)}+"#"+p,"_blank");`,
    '})()',
  ].join('')
  return `javascript:${encodeURIComponent(script)}`
}
