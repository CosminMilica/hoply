import { describe, expect, it } from 'vitest'
import { defineComponent, h } from 'vue'
import { mount } from '@vue/test-utils'
import { QueryClient, VueQueryPlugin } from '@tanstack/vue-query'
import { delay, http, HttpResponse } from 'msw'
import { server } from '@/shared/mocks/node'
import { settle, useServices } from '@/test/hoply'
import type { Application } from '../api/applications.schema'
import { applicationKeys, useApplicationMutations, useApplicationsQuery, useSkipsQuery } from './useApplications'
import { seedId } from '@/test/jobs-seed'

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
    api().setStatus.mutate({ jobId: seedId(1), status: 'saved' })
    api().skip.mutate(seedId(1))
    await settle(1)
    expect(queryClient.getQueryData(applicationKeys.list)).toHaveLength(1)
    expect(queryClient.getQueryData(applicationKeys.skips)).toEqual([seedId(1)])
    await new Promise(resolve => setTimeout(resolve, 60))
    await settle()
    expect(queryClient.getQueryData(applicationKeys.list)).toEqual([])
    expect(queryClient.getQueryData(applicationKeys.skips)).toEqual([])
    api().clearSkips.mutate()
    await settle()
    expect(queryClient.getQueryData(applicationKeys.skips)).toEqual([])
  })

  it('shows the new status at once, by the service’s rules', async () => {
    const { applications } = useServices()
    applications.track(seedId(1), 'applied')
    const { queryClient, api } = setup()
    await settle()
    server.use(http.put('*/api/applications/v1/applications/:id', async () => {
      await delay(30)
      return HttpResponse.json({ message: 'no' }, { status: 500 })
    }))
    api().setStatus.mutate({ jobId: seedId(1), status: 'cv_ready' })
    api().setStatus.mutate({ jobId: seedId(2), status: 'saved' })
    await settle(1)
    const [newest, applied] = queryClient.getQueryData<Application[]>(applicationKeys.list)!
    expect(newest).toMatchObject({ job_id: seedId(2), status: 'saved', cv_ready_at: null, applied_at: null })
    expect(newest!.saved_at).toEqual(expect.any(String))
    expect(applied).toMatchObject({ job_id: seedId(1), status: 'applied', applied_at: '2026-11-25T09:00:01.000Z' })
    expect(applied!.cv_ready_at).toEqual(expect.any(String))
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
