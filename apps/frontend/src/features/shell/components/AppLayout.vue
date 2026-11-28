<script setup lang="ts">
import { computed, watch } from 'vue'
import { useRoute } from 'vue-router'
import HeaderSidebarShell from '@/shared/layouts/HeaderSidebarShell.vue'
import { provideMenuBadges } from '@/shared/application/menus'
import { locale, setLocale, t } from '@/features/common/i18n'
import { useApplicationsQuery } from '@/features/applications/composables/useApplications'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { useNavTabs } from '../tabs'
import AccountMenu from './AccountMenu.vue'
import BottomTabBar from './BottomTabBar.vue'
import CrashrLogo from './CrashrLogo.vue'
import TopNav from './TopNav.vue'

/**
 * The `app` layout: forge's header-sidebar shell with no sidebar. From 900px the primary menu is tabs in the
 * header; below that it is a bottom tab bar. The avatar in the header's corner opens the account menu.
 */
const WIDE = '(min-width: 900px)'
const NEVER = 'not all'
const applications = useApplicationsQuery()
provideMenuBadges(computed(() => {
  const all = applications.data.value ?? []
  const saved = all.filter(application => application.status === 'saved').length
  return { saved: saved || undefined, applications: all.length - saved || undefined }
}))
const tabs = useNavTabs()

// The app speaks the person's language, and the browser tab's title follows a switch.
const profile = useProfileQuery()
watch(() => profile.data.value?.app_language, language => setLocale(language), { immediate: true })
const route = useRoute()
watch(locale, () => {
  const titled = [...route.matched].reverse().find(record => typeof record.meta.title === 'string' && record.meta.title)
  if (titled) document.title = titled.meta.title as string
})
const addJob = computed(() => ({ name: 'crashr-job-add', query: tabs.value.some(tab => tab.active && tab.value === 'saved') ? { from: 'saved' } : {} }))
</script>

<template>
  <UTheme :ui="{ dashboardSidebarToggle: { base: 'hidden!' } }">
    <HeaderSidebarShell
      layout="app"
      :breakpoints="{ sidebar: NEVER, header: WIDE, aside: WIDE }"
      :header="{ spacing: 'gap-6 px-4 sm:px-8 min-[900px]:px-12', actions: 'gap-3' }"
      :page="{ main: 'flex-1 px-4 pt-4 pb-28 sm:px-8 min-[900px]:px-12 min-[900px]:pt-8 min-[900px]:pb-12' }"
    >
      <template #logo>
        <CrashrLogo />
      </template>
      <template #header>
        <TopNav />
      </template>
      <template #actions="{ narrow }">
        <UButton
          v-if="narrow"
          :to="addJob"
          icon="i-lucide-plus"
          :aria-label="t.shell.addJob"
          color="neutral"
          variant="ghost"
          size="lg"
        />
        <UButton v-else :to="addJob" icon="i-lucide-plus" :label="t.shell.addJob" color="neutral" variant="outline" />
        <AccountMenu />
      </template>
    </HeaderSidebarShell>
    <BottomTabBar class="min-[900px]:hidden" />
  </UTheme>
</template>
