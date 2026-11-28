import { computed } from 'vue'
import * as common from './messages'
import * as shell from '@/features/shell/messages'
import * as jobs from '@/features/jobs/messages'
import * as cv from '@/features/cv/messages'
import * as profile from '@/features/profile/messages'
import * as applications from '@/features/applications/messages'
import * as onboarding from '@/features/onboarding/messages'
import { locale } from './locale'

export * from './locale'
export * from './format'

const catalogs = {
  en: { common: common.en, shell: shell.en, jobs: jobs.en, cv: cv.en, profile: profile.en, applications: applications.en, onboarding: onboarding.en },
  ro: { common: common.ro, shell: shell.ro, jobs: jobs.ro, cv: cv.ro, profile: profile.ro, applications: applications.ro, onboarding: onboarding.ro },
}

export type Messages = typeof catalogs.en

/**
 * The app's words in the language it shows. Templates read `t.jobs.title` (a ref in setup unwraps there); scripts
 * read `t.value.jobs.title`, inside a computed or a function so a switch of language reaches them.
 */
export const t = computed<Messages>(() => catalogs[locale.value])

/** An area's name in the app's language; an area the app does not know shows as the service sent it. */
export const areaLabel = (area: string) => (t.value.common.areas as Record<string, string>)[area] ?? area
