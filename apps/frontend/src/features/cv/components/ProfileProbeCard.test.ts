import { afterEach, describe, expect, it } from 'vitest'
import { server } from '@/shared/mocks/node'
import { button, click, input, mountPage, settle, type, useServices } from '@/test/hoply'
import { setLocale } from '@/features/common/i18n'
import { sampleProfile } from '@/test/profile-service'
import ProfileProbeCard from './ProfileProbeCard.vue'
import ProfilePage from '@/features/profile/pages/ProfilePage.vue'

const card = () => document.querySelector<HTMLElement>('[data-test="profile-probe"]')
const question = {
  theme: 'Pipelines', question: 'How many services ship through your pipelines?', why: 'Every CV leads with this.',
  example: 'Used by <number> services', about: 'Built and maintained deployment pipelines for internal services.',
  place: 'experience:r1', choices: ['Under 10 services', '10–50 services', 'Over 50 services'],
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ProfileProbeCard', () => {
  it('asks what makes every CV stronger and keeps a picked range as a note of the role', async () => {
    const { profile } = useServices(undefined, { profileProbe: { status: 'ready', questions: [question, { ...question, question: 'What did it replace?', choices: [] }] } })
    await mountPage(ProfilePage)
    expect(card()!.textContent).toContain('Make every CV stronger')
    expect(card()!.textContent).toContain('an honest range is fine')
    expect(input('What did it replace?').getAttribute('placeholder')).toBe('Used by <number> services')
    expect(input(question.question).getAttribute('placeholder')).toBe('Pick one above, or answer in your own words')

    await click(button('10–50 services'))
    expect(button('10–50 services').getAttribute('aria-pressed')).toBe('true')
    expect((input(question.question) as HTMLTextAreaElement).value).toBe('10–50 services')
    await click(button('Add to my profile'))
    await settle()
    expect(profile.experience.find(role => role.id === 'r1')!.notes.at(-1)).toEqual({ text: '10–50 services', skills: [] })
    expect(card()!.textContent).toContain('Saved in your profile under DevOps Engineer at Verde Cloud')

    await click(button('Change'))
    await settle()
    expect(profile.experience.find(role => role.id === 'r1')!.notes.some(note => note.text === '10–50 services')).toBe(false)

    const second = () => document.querySelectorAll<HTMLElement>('[data-test="probe"]')[1]!
    const inSecond = (label: string) => [...second().querySelectorAll('button')].find(element => element.textContent?.trim() === label)!
    await type(input('What did it replace?'), 'short')
    expect(inSecond('Add to my profile').disabled).toBe(true)
    await click(inSecond('Skip'))
    expect(second().textContent).toContain('Skipped')
    await click(button('Change my mind'))
    expect(card()!.textContent).not.toContain('Skipped')
  })

  it('speaks Romanian when the app does', async () => {
    useServices(undefined, { profileProbe: { status: 'ready', questions: [question] } })
    setLocale('ro')
    await mountPage(ProfileProbeCard, { props: { profile: sampleProfile() } })
    expect(card()!.textContent).toContain('Fă fiecare CV mai puternic')
    expect(input(question.question).getAttribute('placeholder')).toBe('Alege unul de mai sus sau răspunde cu cuvintele tale')
    await click(button('Sari peste'))
    expect(card()!.textContent).toContain('Omis')
    expect(button('M-am răzgândit')).toBeTruthy()
  })

  it('says it is reading while asked for, and stays out of the way otherwise', async () => {
    useServices(undefined, { profileProbe: { status: 'pending', questions: [] } })
    await mountPage(ProfilePage)
    expect(document.querySelector('[data-test="profile-probe-pending"]')!.textContent).toContain('Reading your profile')
    document.body.innerHTML = ''
    useServices()
    await mountPage(ProfilePage)
    expect(card()).toBeNull()
    expect(document.querySelector('[data-test="profile-probe-pending"]')).toBeNull()
  })

  it('says when the questions will come while the profile says too little, and asks for none', async () => {
    const services = useServices(sampleProfile({ experience: [], projects: [] }))
    const asked: string[] = []
    const listen = ({ request }: { request: Request }) => asked.push(new URL(request.url).pathname)
    server.events.on('request:start', listen)
    await mountPage(ProfileProbeCard, { props: { profile: services.profile } })
    server.events.removeListener('request:start', listen)
    expect(document.querySelector('[data-test="profile-probe-later"]')!.textContent).toContain('Once your profile is at least 60% complete')
    expect(asked.some(path => path.endsWith('/probe'))).toBe(false)
    setLocale('ro')
    await settle()
    expect(document.querySelector('[data-test="profile-probe-later"]')!.textContent).toContain('Când profilul tău e completat cel puțin 60%')
  })
})
