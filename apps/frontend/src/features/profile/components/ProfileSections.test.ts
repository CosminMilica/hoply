import { afterEach, describe, expect, it } from 'vitest'
import { button, choose, click, input, mountPage, sectionToggle, settle, type, useServices } from '@/test/hoply'
import { emptyProfile, sampleProfile } from '@/test/profile-service'
import ProfilePage from '../pages/ProfilePage.vue'

const text = () => document.body.textContent ?? ''
const submit = async (label: string) => {
  input(label).closest('form')!.dispatchEvent(new Event('submit', { cancelable: true }))
  await settle()
}

afterEach(() => {
  document.body.innerHTML = ''
})

describe('work preferences', () => {
  it('saves every preference as it changes', async () => {
    const { profile } = useServices(emptyProfile())
    await mountPage(ProfilePage)
    expect(text()).toContain('Work preferences')
    await click(sectionToggle('Work preferences'))

    await click(button('Remote'))
    await click(button('On-site'))
    await click(button('Remote'))
    expect(profile.work_modes).toEqual(['onsite'])
    expect(button('On-site').getAttribute('aria-pressed')).toBe('true')
    await click(input('Open to relocation') as unknown as HTMLElement)
    expect(profile.open_to_relocation).toBe(true)
    await choose('Work authorization', 'Needs sponsorship')
    expect(profile.work_authorization).toBe('needs_sponsorship')
    await type(input('Where it applies'), 'EU')
    await type(input('Notice period (weeks)'), '60')
    expect(text()).toContain('At most 52')
    await type(input('Notice period (weeks)'), '6')
    await type(input('Available from'), '2027-02')
    expect(profile).toMatchObject({ work_authorization_region: 'EU', notice_period_weeks: 6, available_from: '2027-02' })
    await type(input('Notice period (weeks)'), '')
    expect(profile.notice_period_weeks).toBeNull()

    await type(input('Add a target role'), 'Platform engineer')
    await submit('Add a target role')
    expect(profile.target_roles).toEqual(['Platform engineer'])
    await click(button('Remove Platform engineer'))
    expect(profile.target_roles).toEqual([])
    await click(button('Senior'))
    await click(button('Contract'))
    expect(profile).toMatchObject({ target_seniority: ['senior'], employment_types: ['contract'] })

    await type(input('Minimum salary (optional)'), '60000')
    await type(input('Currency'), 'eur')
    await choose('Period', 'per year')
    expect(profile).toMatchObject({ salary_min: 60000, salary_currency: 'EUR', salary_period: 'year' })
    await choose('Period', 'Not set')
    await choose('Work authorization', 'Not set')
    expect(profile).toMatchObject({ salary_period: '', work_authorization: '' })
    await type(input('Currency'), 'euro')
    expect(text()).toContain('Use a three-letter code, e.g. EUR')
    expect(text()).toContain('Set')
  })
})

describe('contact data', () => {
  it('saves the city and the country', async () => {
    const { profile } = useServices(sampleProfile({ city: '', country: '' }))
    await mountPage(ProfilePage)
    expect(text()).toContain('Next up: add your city and country')
    await click(sectionToggle('Contact data'))
    await type(input('City'), 'Cluj-Napoca')
    await type(input('Country'), 'Romania')
    expect(profile).toMatchObject({ city: 'Cluj-Napoca', country: 'Romania' })
    expect(text()).toContain('6/6')
  })
})

describe('projects', () => {
  it('adds, edits and removes a project, with results on its highlights', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(sectionToggle('Projects'))
    expect(text()).toContain('Side projects, open source or a home lab count too.')
    await click(button('+ Add a project'))
    await click(button('Save'))
    expect(text()).toContain('Add the project name')
    await type(input('Project name'), 'Home lab')
    await type(input('Your role in it'), 'Maintainer')
    await type(input('Link'), 'github.com/andrei/lab')
    await type(input('Start month'), '2023-06')
    await click(input('Still ongoing') as unknown as HTMLElement)
    await type(input('Tools you used, separated by commas'), 'Docker, Grafana')
    await type(input('Highlight 1'), 'Ran 12 containers on one host')
    await type(input('Result of highlight 1'), 'kept uptime above 99.9%')
    await click(button('Save'))
    expect(profile.projects[0]).toMatchObject({
      name: 'Home lab', role: 'Maintainer', url: 'github.com/andrei/lab', start: '2023-06', end: '', tools: ['Docker', 'Grafana'],
      bullets: [{ text: 'Ran 12 containers on one host', result: 'kept uptime above 99.9%', skills: [] }],
    })
    expect(text()).toContain('Maintainer · 06/2023 – now')
    expect(text()).toContain('kept uptime above 99.9%')
    expect(document.querySelector('a[href="https://github.com/andrei/lab"]')).toBeTruthy()

    await click(button('Edit Home lab'))
    await type(input('Project name'), 'Homelab')
    await click(button('Save'))
    expect(profile.projects[0]!.name).toBe('Homelab')
    await click(button('Remove Homelab'))
    await click(button('Remove'))
    expect(profile.projects).toEqual([])
  })
})

describe('certifications', () => {
  it('adds, edits and removes a certification', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(sectionToggle('Certifications'))
    await click(button('+ Add a certification'))
    await type(input('Certification'), 'Terraform Associate')
    await type(input('Issuer'), 'HashiCorp')
    await type(input('Issued'), '2025-04')
    await type(input('Expires'), '2024-04')
    await click(button('Save'))
    expect(text()).toContain('It cannot expire before it was issued')
    await type(input('Expires'), '2027-04')
    await type(input('Credential ID'), 'HC-123')
    await click(button('Save'))
    expect(profile.certifications[0]).toMatchObject({ name: 'Terraform Associate', issuer: 'HashiCorp', issued: '2025-04', expires: '2027-04', credential_id: 'HC-123' })
    expect(text()).toContain('HashiCorp · issued 04/2025 · expires 04/2027 · ID HC-123')
    await click(button('Edit Terraform Associate'))
    await type(input('Credential link'), 'credly.com/x')
    await click(button('Save'))
    expect(profile.certifications[0]!.url).toBe('credly.com/x')
    await click(button('Remove Terraform Associate'))
    expect(profile.certifications).toEqual([])
  })
})

describe('additional information', () => {
  it('adds and removes short lines', async () => {
    const { profile } = useServices()
    await mountPage(ProfilePage)
    await click(sectionToggle('Additional information'))
    await type(input('Add a line'), 'Open to relocating to Berlin')
    await submit('Add a line')
    await type(input('Add a line'), '  ')
    await submit('Add a line')
    expect(profile.additional_info).toEqual(['Open to relocating to Berlin'])
    expect(text()).toContain('1 line')
    await click(button('Remove Open to relocating to Berlin'))
    expect(profile.additional_info).toEqual([])
  })
})
