import { afterEach, describe, expect, it } from 'vitest'
import { button, click, mountPage, settle } from '@/test/crashr'
import BackButton from './BackButton.vue'

afterEach(() => {
  document.body.innerHTML = ''
  window.history.replaceState({}, '')
})

describe('BackButton', () => {
  it('goes back in history when there is somewhere to go back to', async () => {
    const { router } = await mountPage(BackButton, { at: '/saved', props: { fallback: { name: 'crashr-jobs' }, label: 'Saved jobs' } })
    await router.push('/jobs/1')
    window.history.replaceState({ back: '/saved' }, '')
    const back = new Promise(resolve => router.afterEach(resolve))
    await click(button('Saved jobs'))
    await back
    await settle()
    expect(router.currentRoute.value.path).toBe('/saved')
  })
})
