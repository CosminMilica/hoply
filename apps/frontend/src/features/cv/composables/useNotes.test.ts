import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { settle, useServices } from '@/test/crashr'
import { sampleProfile } from '@/test/profile-service'
import { OTHER, useNotes } from './useNotes'

async function setup() {
  const services = useServices(sampleProfile())
  let notes!: ReturnType<typeof useNotes>
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  mount(defineComponent({
    setup() {
      const profile = useProfileQuery()
      notes = useNotes(() => profile.data.value)
      return () => h('div')
    },
  }), { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
  await settle()
  return { notes, profile: services.profile }
}

describe('useNotes', () => {
  it('keeps every answer saved at the same time to one role, and takes back exactly the one asked', async () => {
    const { notes, profile } = await setup()
    const [first, second, third] = await Promise.all([
      notes.remember('experience:r1', 'Cut deploys from 2 hours to 15 minutes', ['CI/CD']),
      notes.remember('experience:r1', 'Ran 40 services on Kubernetes', ['Kubernetes']),
      notes.updateEntry('experience:r1', entry => ({ bullets: entry.bullets.map((bullet, index) => (index === 0 ? { ...bullet, result: 'Saved 20% a month' } : bullet)) })),
    ])
    expect(third).toBeUndefined()
    expect(profile.experience[0]!.notes.map(note => note.text)).toEqual(['Cut deploys from 2 hours to 15 minutes', 'Ran 40 services on Kubernetes'])
    expect(profile.experience[0]!.bullets[0]!.result).toBe('Saved 20% a month')

    await Promise.all([notes.forget(first), notes.forget(second)])
    expect(profile.experience[0]!.notes).toEqual([])
    await notes.updateEntry('experience:gone', () => ({ notes: [] }))
  })

  it('keeps answers about no role as the person’s own notes', async () => {
    const { notes, profile } = await setup()
    const [kept] = await Promise.all([notes.remember(OTHER, 'Mentored two juniors', []), notes.remember(OTHER, 'Spoke at a meetup', [])])
    expect(kept.place).toBe(OTHER)
    expect(profile.notes.map(note => note.text)).toEqual(['Mentored two juniors', 'Spoke at a meetup'])
    await notes.forget(kept)
    await notes.forget(kept)
    expect(profile.notes.map(note => note.text)).toEqual(['Spoke at a meetup'])
  })
})
