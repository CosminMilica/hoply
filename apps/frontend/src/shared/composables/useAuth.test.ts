import { describe, it, expect, beforeEach, vi } from 'vitest'

describe('useAuth – dev mode (VITE_AUTH_DISABLED=true)', () => {
  beforeEach(() => {
    vi.resetModules()
    vi.stubEnv('VITE_AUTH_DISABLED', 'true')
  })

  async function loadUseAuth() {
    const mod = await import('./useAuth')
    return mod.useAuth()
  }

  it('init() sets user to DEV_USER and isLoading to false', async () => {
    const auth = await loadUseAuth()
    await auth.init()

    expect(auth.user.value).not.toBeNull()
    expect(auth.user.value!.username).toBe('dev-user')
    expect(auth.isLoading.value).toBe(false)
  })

  it('user has expected dev properties', async () => {
    const auth = await loadUseAuth()
    await auth.init()

    expect(auth.user.value).toMatchObject({
      id: '00000000-0000-0000-0000-000000000001',
      email: 'dev@localhost',
      username: 'dev-user',
      firstName: 'Dev',
      lastName: 'User',
      roles: ['admin', 'user'],
      customerId: '00000000-0000-0000-0000-000000000001',
      orgId: null,
    })
  })

  it('isAuthenticated is true after init', async () => {
    const auth = await loadUseAuth()
    await auth.init()

    expect(auth.isAuthenticated.value).toBe(true)
  })

  it('getToken() returns dev-token', async () => {
    const auth = await loadUseAuth()
    await auth.init()

    const token = await auth.getToken()
    expect(token).toBe('dev-token')
  })

  it('login() sets user to DEV_USER (no-op style)', async () => {
    const auth = await loadUseAuth()
    await auth.init()

    auth.login()
    expect(auth.user.value).not.toBeNull()
    expect(auth.user.value!.username).toBe('dev-user')
  })

  it('logout() sets user to null', async () => {
    const auth = await loadUseAuth()
    await auth.init()

    auth.logout()
    expect(auth.user.value).toBeNull()
  })

  it('hasRole() returns true for any role when user is DEV_USER', async () => {
    const auth = await loadUseAuth()
    await auth.init()

    expect(auth.hasRole('admin')).toBe(true)
    expect(auth.hasRole('user')).toBe(true)
  })
})


