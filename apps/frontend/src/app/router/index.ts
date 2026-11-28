// The application's routes, composed once (ADR-015). forge renders the shell, the menus,
// the routing contracts and the system pages from the configuration into
// src/shared/application/; the project adds its pages per layout, its public routes,
// the wrappers that fill a layout's slots, and its menu actions here.
import type { RouteRecordRaw } from 'vue-router'
import { createApplicationRouter } from '@/shared/application/router'
import AppLayout from '@/features/shell/components/AppLayout.vue'
import { t } from '@/features/common/i18n'
import { installOnboardingGuard, ONBOARDING } from '@/features/onboarding/guard'
import { auth } from '../auth'
import { queryClient } from '../providers/query-client'

/** A route's meta with its title in the app's language, read each time the router sets the page title. */
function titled<T extends object>(key: keyof typeof t.value.shell.titles, meta: T = {} as T): T & { title: string } {
  return Object.defineProperty(meta, 'title', { get: () => t.value.shell.titles[key], enumerable: true }) as T & { title: string }
}

export const crashrRoutes: RouteRecordRaw[] = [
  { path: 'profile', name: 'crashr-profile', component: () => import('@/features/profile/pages/ProfilePage.vue'), meta: titled('profile', { menu: 'profile' }) },
  { path: 'jobs', name: 'crashr-jobs', component: () => import('@/features/jobs/pages/JobsPage.vue'), meta: titled('jobs', { menu: 'jobs' }) },
  {
    path: 'jobs/add',
    name: 'crashr-job-add',
    component: () => import('@/features/jobs/pages/AddJobPage.vue'),
    // Opened from Saved (`?from=saved`), the page belongs to Saved in the menu.
    meta: titled('addJob', { get menu() { return router.currentRoute.value.query.from === 'saved' ? 'saved' : 'jobs' } }),
  },
  { path: 'jobs/:id', name: 'crashr-job', component: () => import('@/features/jobs/pages/JobPage.vue'), props: true, meta: titled('job', { menu: 'saved' }) },
  { path: 'jobs/:id/prepare', name: 'crashr-prep', component: () => import('@/features/cv/pages/PrepPage.vue'), props: true, meta: titled('prepare', { menu: 'saved' }) },
  { path: 'jobs/:id/cv', name: 'crashr-cv', component: () => import('@/features/cv/pages/CvPage.vue'), props: true, meta: titled('cv', { menu: 'applications' }) },
  { path: 'saved', name: 'crashr-saved', component: () => import('@/features/applications/pages/SavedPage.vue'), meta: titled('saved', { menu: 'saved' }) },
  { path: 'applications', name: 'crashr-applications', component: () => import('@/features/applications/pages/ApplicationsPage.vue'), meta: titled('applications', { menu: 'applications' }) },
  { path: 'skills', name: 'crashr-skills', component: () => import('@/features/applications/pages/SkillGapsPage.vue'), meta: titled('skills', { menu: 'skills' }) },
  { path: 'preferences', name: 'crashr-preferences', component: () => import('@/features/onboarding/pages/PreferencesPage.vue'), meta: titled('preferences') },
]

/** Onboarding fills the screen, outside the layout, and needs the person signed in. */
export const onboardingRoute: RouteRecordRaw = {
  path: '/welcome',
  name: ONBOARDING,
  component: () => import('@/features/onboarding/pages/OnboardingPage.vue'),
  meta: { requiresAuth: true, title: 'crashr' },
}

const router = createApplicationRouter({
  auth,
  routes: { app: crashrRoutes },
  layouts: { app: AppLayout },
  publicRoutes: [onboardingRoute],
  actions: {
    'sign-out': () => auth.signOut(),
  },
})

installOnboardingGuard(router, queryClient)

export default router
