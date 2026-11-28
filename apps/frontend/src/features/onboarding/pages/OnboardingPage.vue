<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import CrashrLogo from '@/features/shell/components/CrashrLogo.vue'
import { locale, setLocale, t, type Locale } from '@/features/common/i18n'
import { useProfileMutations, useProfileQuery } from '@/features/profile/composables/useProfile'
import type { Domain } from '@/features/profile/api/profile.schema'
import DomainPicker from '../components/DomainPicker.vue'
import LanguagePicker from '../components/LanguagePicker.vue'

/** The first sign-in: the app's language, then the fields of work whose jobs Find jobs shows. */
const route = useRoute()
const router = useRouter()
const profile = useProfileQuery()
const { patch } = useProfileMutations()
const step = ref<1 | 2>(1)
const domains = ref<Domain[]>([])
const failed = ref(false)

// The page starts in the language the person picked before, if any, with the fields they picked; once, so a profile
// read again later does not undo what they pick here.
const stop = watch(() => profile.data.value, (person) => {
  if (!person) return
  setLocale(person.app_language)
  domains.value = person.domains as Domain[]
  queueMicrotask(() => stop())
}, { immediate: true })

// Picking a language switches the app to it at once, so the next step reads in it.
const language = computed<Locale>({ get: () => locale.value, set: value => setLocale(value) })

/**
 * Where the person was going when onboarding stopped them: a page inside the app, else Find jobs, as the button says.
 * The landing page is where a plain sign-in goes, not a page they chose, so it opens Find jobs too.
 */
const next = computed(() => {
  const to = route.query.to
  const inside = typeof to === 'string' && to.startsWith('/') && !to.startsWith('//')
  const landing = inside && ['/', router.resolve({ name: 'crashr-profile' }).path].includes(router.resolve(to).path)
  return inside && !landing ? to : { name: 'crashr-jobs' }
})

async function finish() {
  failed.value = false
  try {
    await patch.mutateAsync({ app_language: locale.value, domains: domains.value })
  }
  catch {
    failed.value = true
    return
  }
  await router.replace(next.value)
}
</script>

<template>
  <div class="flex min-h-dvh flex-col bg-default text-default">
    <header class="flex items-center justify-between gap-4 border-b border-default px-4 py-2 sm:px-8">
      <CrashrLogo />
      <span class="text-sm font-bold text-muted" data-test="step">{{ t.onboarding.stepOf(step, 2) }}</span>
    </header>
    <main class="flex flex-1 justify-center px-4 pt-10 pb-16 sm:pt-16">
      <div class="flex w-full flex-col gap-7" :class="step === 1 ? 'max-w-[720px]' : 'max-w-[1040px]'">
        <div class="flex max-w-[720px] gap-2" aria-hidden="true">
          <span class="h-1.5 flex-1 rounded-full bg-crashr-600" />
          <span class="h-1.5 flex-1 rounded-full" :class="step === 2 ? 'bg-crashr-600' : 'bg-accented'" />
        </div>
        <UAlert v-if="profile.isError.value" color="error" variant="subtle" :title="t.onboarding.loadFailed" />

        <template v-if="step === 1">
          <div class="flex flex-col gap-2.5">
            <h1 class="m-0 text-[34px] leading-tight font-extrabold tracking-tight sm:text-[40px]">{{ t.onboarding.welcome }}</h1>
            <p class="m-0 text-[17px] leading-relaxed font-medium text-muted">{{ t.onboarding.languageIntro }}</p>
          </div>
          <LanguagePicker v-model="language" />
          <div class="flex flex-wrap items-center justify-between gap-4">
            <span class="text-sm font-medium text-muted">{{ t.onboarding.languageLater }}</span>
            <UButton :label="t.common.actions.continue" trailing-icon="i-lucide-arrow-right" size="lg" @click="step = 2" />
          </div>
        </template>

        <template v-else>
          <div class="flex max-w-[720px] flex-col gap-2.5">
            <h1 class="m-0 text-[34px] leading-tight font-extrabold tracking-tight sm:text-[40px]">{{ t.onboarding.domainsTitle }}</h1>
            <p class="m-0 text-[17px] leading-relaxed font-medium text-muted">{{ t.onboarding.domainsIntro }}</p>
          </div>
          <DomainPicker v-model="domains" />
          <UAlert v-if="failed" color="error" variant="subtle" :title="t.onboarding.saveFailed" />
          <div class="flex flex-wrap items-center justify-between gap-4">
            <div class="flex flex-wrap items-center gap-5">
              <UButton :label="t.common.actions.back" icon="i-lucide-arrow-left" color="neutral" variant="ghost" @click="step = 1" />
              <span class="text-sm font-semibold text-muted" aria-live="polite" data-test="picked">{{ t.onboarding.picked(domains.length) }}</span>
            </div>
            <UButton
              :label="t.onboarding.finish"
              trailing-icon="i-lucide-arrow-right"
              size="lg"
              :disabled="!domains.length || !profile.data.value"
              :loading="patch.isPending.value"
              @click="finish"
            />
          </div>
        </template>
      </div>
    </main>
  </div>
</template>
