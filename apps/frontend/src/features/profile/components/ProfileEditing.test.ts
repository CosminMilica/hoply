import { afterEach, describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { setLocale } from '@/features/common/i18n'
import { server } from '@/shared/mocks/node'
import { button, choose, click, input, mountPage, sectionToggle, settle, type, useServices } from '@/test/hoply'
import { sampleProfile } from '@/test/profile-service'
import ProfilePage from '../pages/ProfilePage.vue'

const text = () => document.body.textContent ?? ''
const form = (name: string) => document.querySelector<HTMLFormElement>(`form[aria-label="${name}"]`)
const value = (label: string) => input(label).value
const refuse = (path: string, method: 'put' | 'post' = 'put') => server.use(http[method](`*/api/profile/v1/profiles/me/${path}${method === 'put' ? '/:id' : ''}`, () =>
  HttpResponse.json({ message: 'end must not be before start', type: 'ValidationError', detail: null }, { status: 422 })))

afterEach(() => {
  document.body.innerHTML = ''
})

describe('editing a role', () => {
  it('opens the role prefilled, changes every field and saves it in place', async () => {
    const { profile } = useServices()
    profile.experience[0]!.bullets[0] = { text: 'Wrote Terraform modules', result: '', skills: ['Terraform'] }
    await mountPage(ProfilePage)
    expect(text()).toContain('Full-time · Cloud · Iași · team of 6')
    await click(button('Edit DevOps Engineer'))
    expect([value('Role'), value('Company'), value('Location'), value('Industry'), value('Start month'), value('Team size')])
      .toEqual(['DevOps Engineer', 'Verde Cloud', 'Iași', 'Cloud', '2022-03', '6'])
    expect(() => input('End month')).toThrow()
    expect(value('Tools you used, separated by commas')).toBe('Linux, Docker, CI/CD, Python, Git, Monitoring, SQL')
    expect(value('Highlight 4')).toBe('Wrote SQL queries and reports for the data team.')

    await type(input('Role'), 'Senior DevOps Engineer')
    await type(input('Industry'), 'Fintech')
    await click(input('I still work here') as unknown as HTMLElement)
    await type(input('End month'), '2026-01')
    await choose('Employment type', 'Contract')
    await type(input('People managed'), '2')
    await type(input('Tools you used, separated by commas'), 'Linux, Terraform')
    await type(input('Result of highlight 2'), 'cut setup from 2 days to 30 minutes')
    await click(button('Remove highlight 4'))
    await click(button('Save'))

    expect(form('Edit DevOps Engineer')).toBeNull()
    expect(profile.experience.map(role => role.id)).toEqual(['r1', 'r2'])
    expect(profile.experience[0]).toMatchObject({
      role: 'Senior DevOps Engineer', industry: 'Fintech', employment_type: 'contract', end: '2026-01', people_managed: 2, team_size: 6,
      tools: ['Linux', 'Terraform'],
    })
    expect(profile.experience[0]!.bullets).toEqual([
      { text: 'Wrote Terraform modules', result: '', skills: ['Terraform'] },
      { text: 'Automated server setup and configuration with scripts.', result: 'cut setup from 2 days to 30 minutes', skills: [] },
      { text: 'Monitored services and responded to incidents on call.', result: '', skills: [] },
    ])
    expect(text()).toContain('cut setup from 2 days to 30 minutes')
    expect(text()).toContain('managed 2 people')
  })

  it('shows the notes Prepare your CV saved and removes one', async () => {
    const { profile } = useServices()
    profile.experience[0]!.notes = [{ text: 'i ran the CI for 40 services', skills: ['CI/CD'] }]
    profile.notes = [{ text: 'I like calm on-call weeks', skills: [] }]
    await mountPage(ProfilePage)
    expect(text()).toContain('Your notes for CVs')
    expect(text()).toContain('i ran the CI for 40 services')
    await click(button('Remove note 1 of DevOps Engineer'))
    await settle()
    expect(profile.experience[0]!.notes).toEqual([])
    expect(profile.experience[0]!.bullets).toHaveLength(4)
    await click(sectionToggle('Additional information'))
    expect(text()).toContain('I like calm on-call weeks')
    await click(button('Remove note 1 of your notes'))
    await settle()
    expect(profile.notes).toEqual([])
  })

  it('clears the employment type and the counts', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(button('Edit DevOps Engineer'))
    await choose('Employment type', 'Not set')
    await type(input('Team size'), '')
    await click(button('Save'))
    expect(profile.experience[0]).toMatchObject({ employment_type: '', team_size: null })
  })

  it('restores the read view on cancel, without saving', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(button('Edit DevOps Engineer'))
    await type(input('Role'), 'Changed')
    await click(button('Cancel'))
    expect(form('Edit DevOps Engineer')).toBeNull()
    expect(profile.experience[0]!.role).toBe('DevOps Engineer')
  })

  it('shows the service’s refusal in the form and keeps it open', async () => {
    useServices()
    refuse('experience')
    await mountPage(ProfilePage)
    await click(button('Edit DevOps Engineer'))
    await click(button('Save'))
    await settle()
    expect(document.querySelector('[role="alert"]')!.textContent).toBe('end must not be before start')
    expect(form('Edit DevOps Engineer')).toBeTruthy()
    await click(button('Cancel'))
    expect(document.querySelector('[role="alert"]')).toBeNull()
  })

  it('shows a refused new role in the add form', async () => {
    useServices()
    refuse('experience', 'post')
    await mountPage(ProfilePage)
    await click(button('+ Add a role'))
    await type(input('Role'), 'Engineer')
    await click(button('Save'))
    await settle()
    expect(document.querySelector('[role="alert"]')!.textContent).toBe('end must not be before start')
  })

  it('keeps highlights up to ten rows and checks their length', async () => {
    useServices()
    await mountPage(ProfilePage)
    await click(button('+ Add a role'))
    for (let i = 0; i < 9; i += 1) await click(button('+ Add a highlight'))
    expect(() => button('+ Add a highlight')).toThrow()
    expect(value('Highlight 10')).toBe('')
    await type(input('Role'), 'Engineer')
    await type(input('Highlight 1'), 'x'.repeat(301))
    await click(button('Save'))
    expect(text()).toContain('At most 300 characters')
  })

  it('shows only the labels that have content, and a hint without either', async () => {
    const empty = { id: 'r9', role: 'Intern', company: 'Acme', location: '', employment_type: '', industry: '', team_size: null, people_managed: null, start: '', end: '', tools: [], bullets: [] }
    useServices(sampleProfile({ experience: [empty, { ...empty, id: 'r8', role: 'Tester', tools: ['Jest'] }] }))
    await mountPage(ProfilePage)
    const card = (role: string) => [...document.querySelectorAll('[data-state]')].filter(element => element.textContent?.includes(role) && element.querySelector('h3, p')).at(-1)!
    expect(card('Intern').textContent).toContain('No highlights yet. Edit the role to add what you did.')
    expect(card('Intern').textContent).not.toContain('What you did')
    await click(button(/^Tester/))
    expect(card('Tester').textContent).toContain('Tools you used')
    expect(card('Tester').textContent).not.toContain('No highlights yet')
  })
})

describe('editing education', () => {
  it('opens the entry prefilled and saves it in place', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(sectionToggle('Education'))
    await click(button('Edit BSc Computer Science'))
    expect([value('Field of study'), value('Degree title'), value('School'), value('Start year'), value('End year')])
      .toEqual(['Computer Science', 'BSc Computer Science', 'University of Iași', '2016', '2020'])
    await type(input('End year'), '2015')
    await click(button('Save'))
    expect(text()).toContain('The end year cannot be before the start year')
    await type(input('End year'), '2019')
    await type(input('Degree title'), '')
    await click(button('Save'))
    expect(profile.education[0]).toMatchObject({ id: 'e1', level: 'bachelor', degree: '', end_year: '2019' })
    expect(text()).toContain('Bachelor’s degree in Computer Science')
  })

  it('cancels without saving and shows a refusal from the service', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(sectionToggle('Education'))
    await click(button('Edit BSc Computer Science'))
    await type(input('School'), 'Elsewhere')
    await click(button('Cancel'))
    expect(profile.education[0]!.school).toBe('University of Iași')
    refuse('education')
    await click(button('Edit BSc Computer Science'))
    await click(button('Save'))
    await settle()
    expect(document.querySelector('[role="alert"]')!.textContent).toBe('end must not be before start')
  })
})

describe('editing in Romanian', () => {
  it('edits a role, its highlights and its notes, and confirms a removal, in Romanian', async () => {
    setLocale('ro')
    const { profile } = useServices()
    profile.experience[0]!.notes = [{ text: 'i ran the CI for 40 services', skills: [] }]
    await mountPage(ProfilePage)
    expect(text()).toContain('Notițele tale pentru CV-uri')
    await click(button('Șterge notița 1 din DevOps Engineer'))
    await settle()
    expect(profile.experience[0]!.notes).toEqual([])

    await click(button('Editează DevOps Engineer'))
    expect(form('Editează DevOps Engineer')).toBeTruthy()
    expect(value('Realizarea 4')).toBe('Wrote SQL queries and reports for the data team.')
    await type(input('Rol'), '')
    await click(button('Salvează'))
    expect(text()).toContain('Adaugă rolul')
    await type(input('Rol'), 'SRE')
    await choose('Tip de angajare', 'Contract')
    await type(input('Persoane conduse'), '2')
    await type(input('Rezultatul realizării 1'), 'zero downtime')
    await click(button('Șterge realizarea 4'))
    await click(button('+ Adaugă o realizare'))
    expect(input('Realizarea 4').getAttribute('placeholder')).toBe('Altceva ce ai făcut')
    await click(button('Salvează'))
    expect(profile.experience[0]).toMatchObject({ role: 'SRE', employment_type: 'contract', people_managed: 2 })
    expect(text()).toContain('Contract · Cloud · Iași · echipă de 6 persoane · ai condus 2 persoane')
    expect(document.body.innerHTML).toContain('Rezultat:')

    const removeRole = [...document.querySelectorAll('button')].find(element => element.textContent?.trim() === 'Șterge acest rol')!
    expect(removeRole.getAttribute('aria-label')).toBe('Șterge SRE')
    await click(removeRole)
    expect(text()).toContain('Ștergi SRE?')
    await click(button('Păstrează'))
    expect(profile.experience).toHaveLength(2)
  })

  it('adds education by its Romanian level', async () => {
    setLocale('ro')
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(sectionToggle('Educație'))
    await click(button('+ Adaugă educație'))
    expect(form('Adaugă educație')).toBeTruthy()
    await click(button('Salvează'))
    expect(text()).toContain('Alege nivelul')
    await choose('Nivel', 'Master')
    await type(input('Domeniul de studiu'), 'Drept')
    await type(input('Anul de început'), '2021')
    await type(input('Anul de final'), '2020')
    await click(button('Salvează'))
    expect(text()).toContain('Anul de final nu poate fi înainte de anul de început')
    await type(input('Anul de final'), '')
    await click(button('Salvează'))
    expect(profile.education[1]).toMatchObject({ level: 'master', field: 'Drept' })
    expect(text()).toContain('Master în Drept')
    expect(text()).toContain('2021 – prezent')
    await click(button('Editează Master în Drept'))
    expect(form('Editează Drept')).toBeTruthy()
  })
})
