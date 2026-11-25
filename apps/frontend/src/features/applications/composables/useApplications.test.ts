import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { delay, http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { settle, useServices } from '@/test/hoply'
import { applicationKeys, useApplicationMutations, useApplicationsQuery, useSkipsQuery } from './useApplications'

function setup() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  let api!: ReturnType<typeof useApplicationMutations>
  mount(defineComponent({
    setup() {
      useApplicationsQuery()
      useSkipsQuery()
      api = useApplicationMutations()
      return () => h('div')
    },
  }), { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
  return { queryClient, api: () => api }
}

describe('useApplicationMutations', () => {
  it('puts the cache back when the service refuses', async () => {
    useServices()
    const { queryClient, api } = setup()
    await settle()
    server.use(
      http.put('*/api/applications/v1/applications/:id', async () => {
        await delay(30)
        return HttpResponse.json({ message: 'no' }, { status: 500 })
      }),
      http.put('*/api/applications/v1/skips/:id', async () => {
        await delay(30)
        return HttpResponse.json({ message: 'no' }, { status: 500 })
      }),
      http.delete('*/api/applications/v1/skips', () => HttpResponse.json({ message: 'no' }, { status: 500 })),
    )
    api().setStatus.mutate({ jobId: '1', status: 'saved' })
    api().skip.mutate('1')
    await settle(1)
    expect(queryClient.getQueryData(applicationKeys.list)).toHaveLength(1)
    expect(queryClient.getQueryData(applicationKeys.skips)).toEqual(['1'])
    await new Promise(resolve => setTimeout(resolve, 60))
    await settle()
    expect(queryClient.getQueryData(applicationKeys.list)).toEqual([])
    expect(queryClient.getQueryData(applicationKeys.skips)).toEqual([])
    api().clearSkips.mutate()
    await settle()
    expect(queryClient.getQueryData(applicationKeys.skips)).toEqual([])
  })

  it('changes nothing before the lists are loaded', async () => {
    useServices()
    const queryClient = new QueryClient({ defaultOptions: { mutations: { retry: false } } })
    let api!: ReturnType<typeof useApplicationMutations>
    mount(defineComponent({ setup: () => { api = useApplicationMutations(); return () => h('div') } }), { global: { plugins: [[VueQueryPlugin, { queryClient }]] } })
    api.remove.mutate('9')
    await settle()
    expect(queryClient.getQueryData(applicationKeys.list)).toBeUndefined()
  })
})
