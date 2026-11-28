import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { setLocale } from '@/features/common/i18n'
import { cvApi } from '../api/cvApi'
import { button, click, input, mountPage, settle, type, useServices } from '@/test/hoply'
import { seedId } from '@/test/jobs-seed'
import CvPage from '../pages/CvPage.vue'

const request = { job_id: seedId(1), skip_roles: [], focus: [], confirmed: [] }
const card = () => document.querySelector('[data-test="letter"]')!
const text = () => card().textContent ?? ''
const field = () => document.querySelector<HTMLTextAreaElement>('[data-test="letter-text"] textarea, textarea[data-test="letter-text"]')!
const writeText = vi.fn(async () => {})

beforeEach(() => {
  Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
  URL.createObjectURL = vi.fn(() => 'blob:letter')
  URL.revokeObjectURL = vi.fn()
  HTMLAnchorElement.prototype.click = vi.fn()
})

afterEach(() => {
  document.body.innerHTML = ''
  localStorage.clear()
  vi.clearAllMocks()
})

async function openCv(options: Parameters<typeof useServices>[1] = {}) {
  const services = useServices(undefined, options)
  await cvApi.generate(request)
  const page = await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
  return { ...services, ...page, id: services.cvs.cvs[0]!.cv.id }
}

async function refetch(queryClient: Awaited<ReturnType<typeof openCv>>['queryClient']) {
  await queryClient.invalidateQueries({ queryKey: ['cv', 'letter'] })
  await settle()
}

describe('LetterCard', () => {
  it('writes the letter and the messages for the recruiter, and lets the person edit, copy and download them', async () => {
    const { cvs, queryClient, id } = await openCv({ letterWriting: 1 })
    expect(text()).toContain('A cover letter, a LinkedIn note, a message for after you apply and a follow-up')
    await type(input("Recruiter's name (optional)"), '  Ana   Pop ')
    await click(button('Write them'))
    expect(card().querySelector('[data-test="letter-writing"]')).not.toBeNull()
    await refetch(queryClient)
    await refetch(queryClient)

    expect(field().value).toBe(`Dear Ana Pop,\n\nYou need someone for the ${cvs.cvs[0]!.cv.job_title} role, and I have done this work.\n\nI would welcome a conversation.\n\nKind regards,\nAndrei Popescu`)
    expect(text()).toContain('Written by Claude (Sonnet 5.5) from this CV and your profile.')
    expect(card().querySelector('[data-test="letter-count"]')!.textContent).toBe('27 words')
    expect(input("Recruiter's name (optional)").value).toBe('Ana Pop')

    await type(field(), 'Dear Ana,\n\nI can help.')
    await click(button('Undo my edits'))
    expect(field().value.startsWith('Dear Ana Pop,')).toBe(true)
    await type(field(), 'Dear Ana,\n\nI can help.')
    await click(button('Elegant'))
    await click(button('Download the cover letter as PDF'))
    expect(cvs.letterFiles).toEqual([{ id, format: 'pdf', template: 'elegant', text: 'Dear Ana,\n\nI can help.' }])
    expect(HTMLAnchorElement.prototype.click).toHaveBeenCalledOnce()
    await click(button('Download the cover letter as DOCX'))
    expect(cvs.letterFiles[1]!.format).toBe('docx')

    await click(button('LinkedIn note'))
    expect(text()).toContain('Send it with a connection request')
    expect(card().querySelector('[data-test="letter-count"]')!.textContent).toMatch(/^\d+ \/ 300 characters$/)
    await type(field(), 'x'.repeat(301))
    expect(card().querySelector('[data-test="letter-count"]')!.className).toContain('text-error')

    await click(button('Message'))
    expect(input('Subject').value).toBe(`Application for ${cvs.cvs[0]!.cv.job_title}`)
    await click(button('Copy the subject'))
    expect(writeText).toHaveBeenLastCalledWith(`Application for ${cvs.cvs[0]!.cv.job_title}`)
    expect(button('Copy the subject').textContent?.trim()).toBe('Copied')
    await click(button('Copy the message'))
    expect(writeText).toHaveBeenLastCalledWith(expect.stringMatching(/^Hi Ana,\n\nI applied/))

    await click(button('Follow-up'))
    expect(text()).toContain('Send it once, about a week after you apply')
    expect(input('Subject').value).toContain('Following up')

    await click(button('Write them again'))
    expect(cvs.letters.get(id)?.recruiter).toBe('Ana Pop')
  })

  it('speaks Romanian when the app does', async () => {
    setLocale('ro')
    const { queryClient } = await openCv()
    expect(text()).toContain('Scrisoare de intenție și mesaje')
    await click(button('Scrie-le'))
    await refetch(queryClient)
    expect(text()).toContain('Scrise de Claude (Sonnet 5.5) pe baza acestui CV')
    expect(card().querySelector('[data-test="letter-count"]')!.textContent).toMatch(/^\d+ (de )?cuvinte$/)
    expect(button('Copiază scrisoarea de intenție').textContent?.trim()).toBe('Copiază')
    await click(button('Mesaj de conectare'))
    expect(card().querySelector('[data-test="letter-count"]')!.textContent).toMatch(/^\d+ \/ 300 caractere$/)
    await click(button('Mesaj de revenire'))
    expect(input('Subiect')).toBeTruthy()
    expect(button('Scrie-le din nou')).toBeTruthy()
  })

  it('says why a letter could not be written and offers to try again', async () => {
    const { queryClient } = await openCv({ letterFailure: 'usage_limit' })
    await click(button('Write them'))
    await refetch(queryClient)
    expect(card().querySelector('[data-test="letter-failed"]')!.textContent).toBe('Claude reached its usage limit.')
    expect(button('Try again')).toBeTruthy()
  })

  it('says when the file could not be made and when nothing passed the checks', async () => {
    const { queryClient } = await openCv()
    await click(button('Write them'))
    await refetch(queryClient)
    server.use(http.post('*/api/cv/v1/cvs/:id/letter/file', () => HttpResponse.json({}, { status: 503 })))
    await click(button('Download the cover letter as PDF'))
    expect(text()).toContain('The file could not be made.')
    writeText.mockRejectedValueOnce(new Error('denied'))
    await click(button('Copy the cover letter'))
    expect(button('Copy the cover letter').textContent?.trim()).toBe('Copy')
    queryClient.setQueryData(['cv', 'letter', queryClient.getQueryCache().findAll({ queryKey: ['cv', 'letter'] })[0]!.queryKey[2]], {
      status: 'ready', recruiter: '', model: '', refused: 3, reason: '', limit_until: null,
      texts: { cover_letter: '', connection_note: '', message: { subject: '', body: '' }, follow_up: { subject: '', body: '' } },
    })
    await settle()
    expect(card().querySelector('[data-test="letter-empty"]')).not.toBeNull()
    expect(text()).toContain('Written by Claude from this CV')
  })

  it('sends the person to complete their profile instead of writing, while it says too little', async () => {
    const services = useServices()
    await cvApi.generate(request)
    services.profile.experience = []
    services.profile.projects = []
    await mountPage(CvPage, { at: `/jobs/${seedId(1)}/cv`, props: { id: seedId(1) } })
    expect(card().querySelector('[data-test="profile-gate"]')).not.toBeNull()
    expect(text()).toContain('Complete your profile')
    expect([...card().querySelectorAll('button')].some(element => element.textContent?.trim() === 'Write them')).toBe(false)
  })
})
