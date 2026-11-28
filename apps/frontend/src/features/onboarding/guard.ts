import type { QueryClient } from '@tanstack/vue-query'
import type { Router } from 'vue-router'
import { setLocale } from '@/features/common/i18n'
import { profileApi } from '@/features/profile/api/profileApi'
import { profileKeys } from '@/features/profile/composables/useProfile'

export const ONBOARDING = 'hoply-onboarding'

/** How long the guard trusts the profile it read, so moving between pages does not read it again each time. */
const FRESH_MS = 60_000

/**
 * Before a page of the app layout opens, after the sign-in guard: the app takes the person's language, and a person
 * who has picked no field of work yet goes through onboarding first. A profile that cannot be read lets the page
 * open: the page shows its own error.
 */
export function installOnboardingGuard(router: Router, client: QueryClient) {
  router.beforeEach(async (to) => {
    if (to.name === ONBOARDING || !to.matched.some(record => record.meta.layout === 'app')) return true
    const profile = await client.fetchQuery({ queryKey: profileKeys.me, queryFn: profileApi.get, staleTime: FRESH_MS }).catch(() => null)
    if (!profile) return true
    setLocale(profile.app_language)
    return profile.domains.length ? true : { name: ONBOARDING, query: { to: to.fullPath } }
  })
}
