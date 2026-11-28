import { afterEach, describe, expect, it } from 'vitest'
import { mount, type VueWrapper } from '@vue/test-utils'
import { setLocale } from '@/features/common/i18n'
import { button, click, testRouter } from '@/test/crashr'
import { catalogJob } from '@/test/jobs'
import { matchJob } from '../match'
import JobGrid from './JobGrid.vue'
import SwipeDeck from './SwipeDeck.vue'

const text = () => document.body.textContent ?? ''
const deck = (count: number) => Array.from({ length: count }, (_, index) => {
  const job = catalogJob(index % 2, { id: `job-${index}` })
  return { ...job, ...matchJob(job, ['Linux']) }
})

const mounted: VueWrapper[] = []
function show(component: typeof JobGrid | typeof SwipeDeck, count: number, toast: string) {
  const wrapper = mount(component, { props: { deck: deck(count), toast }, attachTo: document.body, global: { plugins: [testRouter()] } })
  mounted.push(wrapper)
  return wrapper
}

afterEach(() => {
  mounted.splice(0).forEach(wrapper => wrapper.unmount())
  document.body.innerHTML = ''
})

describe('the jobs in Romanian', () => {
  it('shows the grid, its decisions and the end of the deck in Romanian', async () => {
    setLocale('ro')
    const wrapper = show(JobGrid, 26, 'Salvat · DevOps Engineer')
    const tile = document.querySelector('[data-test="job-tile"]')!
    expect(tile.getAttribute('aria-label')).toBe(`${catalogJob(0).title} la ${catalogJob(0).company}`)
    expect(tile.textContent).toContain('Iași, la birou')
    expect(tile.textContent).toContain('DevOps, Date · Senior · de pe www.linkedin.com')
    for (const label of ['Salvează', 'Adaptează CV-ul', 'Detalii', 'Sari peste', 'Anulează']) expect(button(label)).toBeTruthy()
    await click(button('Arată încă 2'))
    expect(document.querySelectorAll('[data-test="job-tile"]')).toHaveLength(26)

    await wrapper.setProps({ deck: [] })
    expect(text()).toContain('Le-ai văzut pe toate')
    expect(button('Resetează filtrele')).toBeTruthy()
    await click(button('Adu înapoi joburile sărite'))
    expect(wrapper.emitted('resetSkipped')).toHaveLength(1)
    expect(button('Vezi joburile salvate')).toBeTruthy()
  })

  it('shows the swipe deck in Romanian, and the end of it with its own first way out', async () => {
    setLocale('ro')
    const wrapper = show(SwipeDeck, 2, 'Sărit · DevOps Engineer')
    expect(document.querySelector('[data-test="job-card"]')!.getAttribute('aria-label')).toBe(`${catalogJob(0).title} la ${catalogJob(0).company}`)
    for (const words of ['SALVEAZĂ', 'SARI PESTE', 'ADAPTEAZĂ CV-UL', 'Ce vei face', 'Glisează la dreapta', 'Trage cardul']) expect(text()).toContain(words)
    for (const label of ['Sari peste job', 'Salvează jobul', 'Detalii', 'Adaptează CV-ul', 'Anulează']) expect(button(label)).toBeTruthy()

    await wrapper.setProps({ deck: [] })
    expect(button('Schimbă filtrele')).toBeTruthy()
    setLocale('en')
    await wrapper.vm.$nextTick()
    expect(button('Change filters')).toBeTruthy()
    expect(text()).toContain('You are all caught up')
  })
})
