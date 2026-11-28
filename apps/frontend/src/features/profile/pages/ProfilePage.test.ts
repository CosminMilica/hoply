import { afterEach, describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setLocale } from '@/features/common/i18n'
import { server } from '@/shared/mocks/node'
import { button, choose, click, input, mountPage, sectionToggle, settle, type, useServices } from '@/test/hoply'
import { emptyProfile, sampleProfile } from '@/test/profile-service'
import ProfilePage from './ProfilePage.vue'

const text = () => document.body.textContent ?? ''

afterEach(() => {
  document.body.innerHTML = ''
})

describe('ProfilePage', () => {
  it('shows how ready the profile is and the section summaries', async () => {
    useServices(sampleProfile({ phone: '' }))
    await mountPage(ProfilePage)
    expect(document.querySelector('[data-test="readiness-percent"]')!.textContent).toBe('92%')
    expect(document.querySelector('h1')!.textContent).toBe('Andrei Popescu')
    expect(document.querySelector('[data-test="avatar"]')!.textContent?.trim()).toBe('AP')
    expect(text()).toContain('Next up: add your phone number')
    expect(text()).toContain('5/6')
    expect(text()).toContain('0 projects')
    expect(text()).toContain('0 certifications')
    expect(text()).toContain('0 lines')
    expect(text()).toContain('Set')
    expect(text()).toContain('8 skills')
    expect(text()).toContain('2 roles')
    expect(text()).toContain('1 degree')
    expect(text()).toContain('2 languages')
    expect(sectionToggle('Skills').getAttribute('aria-expanded')).toBe('true')
    expect(sectionToggle('About you').getAttribute('aria-expanded')).toBe('false')
    expect(button('Add a job').getAttribute('href')).toBe('/jobs/add')
  })

  it('saves the headline, the intro and contact data, and refuses a bad email', async () => {
    const { profile } = useServices(emptyProfile())
    await mountPage(ProfilePage)
    expect(document.querySelector('h1')!.textContent).toBe('Your profile')
    expect(document.querySelector('[data-test="avatar"]')!.textContent?.trim()).toBe('')
    await click(sectionToggle('About you'))
    await type(input('Name'), '  Maria Ionescu ')
    expect(profile.name).toBe('Maria Ionescu')
    expect(document.querySelector('h1')!.textContent).toBe('Maria Ionescu')
    expect(document.querySelector('[data-test="avatar"]')!.textContent?.trim()).toBe('MI')
    await type(input('Name'), 'Maria Ionescu')
    await type(input('Headline'), 'Site reliability engineer')
    await type(input('About you'), 'I keep things running.')
    expect(profile.headline).toBe('Site reliability engineer')
    expect(profile.about).toBe('I keep things running.')
    expect(text()).toContain('22/280')
    await click(sectionToggle('Contact data'))
    await type(input('Email'), 'not-an-email')
    expect(text()).toContain('Enter a valid email address')
    expect(profile.email).toBe('')
    await type(input('Email'), 'me@example.com')
    await type(input('LinkedIn'), 'linkedin.com/in/me')
    expect(profile.email).toBe('me@example.com')
    expect(profile.linkedin).toBe('linkedin.com/in/me')
    expect(text()).not.toContain('Enter a valid email address')
    expect(text()).toContain('Next up: add your phone number')
  })

  it('adds and removes skills, ignoring duplicates', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    const add = async (value: string) => {
      await type(input('Add a skill'), value)
      input('Add a skill').closest('form')!.dispatchEvent(new Event('submit', { cancelable: true }))
      await settle()
    }
    await add('Terraform')
    await add('terraform')
    await add(' ')
    expect(profile.skills.filter(skill => skill.toLowerCase() === 'terraform')).toHaveLength(1)
    await click(button('Remove SQL'))
    expect(profile.skills).not.toContain('SQL')
    expect(text()).toContain('8 skills')
  })

  it('adds a role newest first and removes one after confirming', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(button('+ Add a role'))
    await click(button('Save'))
    expect(text()).toContain('Add the role')
    await type(input('Role'), 'Platform engineer')
    await type(input('Company'), 'Nimbus')
    await type(input('Start month'), '2024-05')
    await type(input('End month'), '2023-01')
    await type(input('Team size'), '8')
    await click(button('Save'))
    expect(text()).toContain('The end cannot be before the start')
    await type(input('End month'), '')
    await type(input('Tools you used, separated by commas'), 'Kubernetes, Go, kubernetes')
    await type(input('Highlight 1'), 'Ran the platform')
    await type(input('Result of highlight 1'), 'cut costs by 20%')
    await click(button('+ Add a highlight'))
    await type(input('Highlight 2'), 'Mentored two engineers')
    await click(button('Save'))
    expect(profile.experience[0]).toMatchObject({
      role: 'Platform engineer', start: '2024-05', end: '', team_size: 8, people_managed: null, tools: ['Kubernetes', 'Go'],
      bullets: [{ text: 'Ran the platform', result: 'cut costs by 20%', skills: [] }, { text: 'Mentored two engineers', result: '', skills: [] }],
    })
    expect(text()).toContain('3 roles')

    await click(button('+ Add a role'))
    await click(button('Cancel'))
    expect(text()).toContain('+ Add a role')

    const removeButtons = () => [...document.querySelectorAll('button')].filter(element => element.textContent?.trim() === 'Remove this role')
    await click(removeButtons()[0]!)
    expect(text()).toContain('Remove Platform engineer?')
    await click(button('Keep it'))
    await click(removeButtons()[0]!)
    await click(button('Remove'))
    expect(profile.experience.map(role => role.role)).not.toContain('Platform engineer')
  })

  it('adds and removes education and languages', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(sectionToggle('Education'))
    await click(button('+ Add education'))
    await click(button('Save'))
    expect(text()).toContain('Choose the level')
    await choose('Level', 'Master’s degree')
    await type(input('Field of study'), 'Distributed Systems')
    await type(input('School'), 'TU Delft')
    await type(input('Start year'), '2021')
    await click(button('Save'))
    expect(profile.education[1]).toMatchObject({ level: 'master', field: 'Distributed Systems', degree: '', school: 'TU Delft', start_year: '2021', end_year: '' })
    expect(text()).toContain('Master’s degree in Distributed Systems')
    await click(button('Remove BSc Computer Science'))
    expect(profile.education.map(item => item.field)).toEqual(['Distributed Systems'])

    await click(sectionToggle('Languages'))
    const form = input('Language').closest('form')!
    form.dispatchEvent(new Event('submit', { cancelable: true }))
    await settle()
    expect(text()).toContain('Add the language')
    await type(input('Language'), 'German')
    form.dispatchEvent(new Event('submit', { cancelable: true }))
    await settle()
    expect(profile.languages.at(-1)).toMatchObject({ name: 'German', level: 'B2' })
    await click(button('Remove Romanian'))
    expect(profile.languages.map(item => item.name)).toEqual(['English', 'German'])
  })

  it('says so when the profile cannot be loaded, and tries again', async () => {
    let fail = true
    server.use(http.get('*/api/profile/v1/profiles/me', () => (fail ? HttpResponse.json({ message: 'down' }, { status: 503 }) : HttpResponse.json(sampleProfile()))))
    await mountPage(ProfilePage)
    expect(text()).toContain('We could not load your profile')
    fail = false
    await click(button('Try again'))
    expect(text()).toContain('Ready to tailor')
  })

  it('switches to Romanian live, with the summaries counted in Romanian', async () => {
    useServices(sampleProfile({ phone: '' }))
    await mountPage(ProfilePage)
    expect(text()).toContain('Next up: add your phone number')
    setLocale('ro')
    await settle()
    expect(text()).toContain('Următorul pas: adaugă numărul de telefon')
    expect(text()).toContain('Gata de adaptare')
    for (const summary of ['0 proiecte', '0 certificări', '0 rânduri', 'Setat', '8 competențe', '2 roluri', '1 diplomă', '2 limbi', '4 realizări']) {
      expect(text()).toContain(summary)
    }
    expect(sectionToggle('Competențe').getAttribute('aria-expanded')).toBe('true')
    expect(sectionToggle('Despre tine').getAttribute('aria-expanded')).toBe('false')
    expect(text()).toContain('Normă întreagă · Cloud · Iași · echipă de 6 persoane')
    expect(text()).toContain('Ce ai făcut')
    expect(text()).toContain('Unelte folosite')
    expect(button('Adaugă un job').getAttribute('href')).toBe('/jobs/add')
    await click(sectionToggle('Despre tine'))
    expect(text()).toContain('Două-trei rânduri sunt de ajuns.')
    expect(input('Titlu').getAttribute('placeholder')).toBe('Titlu, de ex. Frontend developer')
  })

  it('names an empty profile and a failed load in Romanian', async () => {
    setLocale('ro')
    useServices(emptyProfile())
    await mountPage(ProfilePage)
    expect(document.querySelector('h1')!.textContent).toBe('Profilul tău')
    expect(text()).toContain('Adaugă un titlu mai jos')
    expect(text()).toContain('De completat')
    document.body.innerHTML = ''
    server.use(http.get('*/api/profile/v1/profiles/me', () => HttpResponse.json({ message: 'down' }, { status: 503 })))
    await mountPage(ProfilePage)
    expect(text()).toContain('Nu ți-am putut încărca profilul')
    expect(button('Încearcă din nou')).toBeTruthy()
  })
})
