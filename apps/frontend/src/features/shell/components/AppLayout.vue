<script setup lang="ts">
import { computed } from 'vue'
import HeaderSidebarShell from '@/shared/layouts/HeaderSidebarShell.vue'
import { provideMenuBadges } from '@/shared/application/menus'
import { useApplicationsQuery } from '@/features/applications/composables/useApplications'
import BottomTabBar from './BottomTabBar.vue'
import HoplyLogo from './HoplyLogo.vue'

/**
 * The `app` layout: forge's header-sidebar shell, wide from 900px. Below that the sidebar's
 * menu button is hidden (UTheme) and the same menu renders as a bottom tab bar instead.
 */
const WIDE = '(min-width: 900px)'
const applications = useApplicationsQuery()
provideMenuBadges(computed(() => ({ saved: applications.data.value?.length || undefined })))
</script>

<template>
  <UTheme :ui="{ dashboardSidebarToggle: { base: 'hidden' } }">
    <HeaderSidebarShell
      layout="app"
      :breakpoints="{ sidebar: WIDE, header: WIDE, aside: WIDE }"
      :menu="{ ui: { link: 'text-[15px]' } }"
      :page="{ main: 'flex-1 px-4 pt-4 pb-28 sm:px-8 min-[900px]:px-12 min-[900px]:pt-8 min-[900px]:pb-12' }"
    >
      <template #logo>
        <HoplyLogo />
      </template>
    </HeaderSidebarShell>
    <BottomTabBar class="min-[900px]:hidden" />
  </UTheme>
</template>
