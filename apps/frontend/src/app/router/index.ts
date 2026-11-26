// The application's routes, composed once (ADR-015). forge renders the shell, the menus,
// the routing contracts and the system pages from the configuration into
// src/shared/application/; the project adds its pages per layout, its public routes,
// the wrappers that fill a layout's slots, and its menu actions here.
import type { RouteRecordRaw } from 'vue-router'
import { createApplicationRouter } from '@/shared/application/router'
import AppLayout from '@/features/shell/components/AppLayout.vue'
import { auth } from '../auth'

export const hoplyRoutes: RouteRecordRaw[] = [
  { path: 'profile', name: 'hoply-profile', component: () => import('@/features/profile/pages/ProfilePage.vue'), meta: { title: 'Profile', menu: 'profile' } },
  { path: 'jobs', name: 'hoply-jobs', component: () => import('@/features/jobs/pages/JobsPage.vue'), meta: { title: 'Find jobs', menu: 'jobs' } },
  {
    path: 'jobs/add',
    name: 'hoply-job-add',
    component: () => import('@/features/jobs/pages/AddJobPage.vue'),
    // Opened from Saved (`?from=saved`), the page belongs to Saved in the menu.
    meta: { title: 'Add a job', get menu() { return router.currentRoute.value.query.from === 'saved' ? 'saved' : 'jobs' } },
  },
  { path: 'jobs/:id', name: 'hoply-job', component: () => import('@/features/jobs/pages/JobPage.vue'), props: true, meta: { title: 'Job', menu: 'saved' } },
  { path: 'jobs/:id/prepare', name: 'hoply-prep', component: () => import('@/features/cv/pages/PrepPage.vue'), props: true, meta: { title: 'Prepare your CV', menu: 'saved' } },
  { path: 'jobs/:id/cv', name: 'hoply-cv', component: () => import('@/features/cv/pages/CvPage.vue'), props: true, meta: { title: 'Your CV', menu: 'saved' } },
  { path: 'saved', name: 'hoply-saved', component: () => import('@/features/applications/pages/SavedPage.vue'), meta: { title: 'Saved jobs', menu: 'saved' } },
  { path: 'saved/skills', name: 'hoply-skills', component: () => import('@/features/applications/pages/SkillGapsPage.vue'), meta: { title: 'Skill gaps', menu: 'saved' } },
]

const router = createApplicationRouter({
  auth,
  routes: { app: hoplyRoutes },
  layouts: { app: AppLayout },
  actions: {
    'sign-out': () => auth.signOut(),
  },
})

export default router
