<script setup lang="ts">
import { computed } from 'vue'
import HeaderSidebarShell from '@/shared/layouts/HeaderSidebarShell.vue'
import { provideMenuBadges } from '@/shared/application/menus'
import { useApplicationsQuery } from '@/features/applications/composables/useApplications'
import { useNavTabs } from '../tabs'
import AccountMenu from './AccountMenu.vue'
import BottomTabBar from './BottomTabBar.vue'
import HoplyLogo from './HoplyLogo.vue'
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
const addJob = computed(() => ({ name: 'hoply-job-add', query: tabs.value.some(tab => tab.active && tab.value === 'saved') ? { from: 'saved' } : {} }))
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
        <HoplyLogo />
      </template>
      <template #header>
        <TopNav />
      </template>
      <template #actions="{ narrow, account }">
        <UButton
          v-if="narrow"
          :to="addJob"
          icon="i-lucide-plus"
          aria-label="Add a job"
          color="neutral"
          variant="ghost"
          size="lg"
        />
        <UButton v-else :to="addJob" icon="i-lucide-plus" label="Add a job" color="neutral" variant="outline" />
        <AccountMenu :items="account" />
      </template>
    </HeaderSidebarShell>
    <BottomTabBar class="min-[900px]:hidden" />
  </UTheme>
</template>
