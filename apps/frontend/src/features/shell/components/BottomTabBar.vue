<script setup lang="ts">
import { t } from '@/features/common/i18n'
import { useNavTabs } from '../tabs'

/** The phone's navigation: the layout's primary menu as tabs at the bottom of the screen. */
const tabs = useNavTabs()
</script>

<template>
  <nav
    :aria-label="t.shell.mainMenu"
    data-shell="tabs"
    class="fixed inset-x-0 bottom-0 z-30 flex border-t border-default bg-elevated px-3 pt-1.5 pb-[calc(0.625rem+env(safe-area-inset-bottom))]"
  >
    <RouterLink
      v-for="tab in tabs"
      :key="String(tab.value)"
      :to="tab.to"
      :aria-current="tab.active ? 'page' : undefined"
      class="flex min-h-13 flex-1 flex-col items-center justify-center gap-0.5 rounded-lg text-xs font-extrabold"
      :class="tab.active ? 'bg-accented text-hoply-400' : 'text-ink-200'"
    >
      <span class="relative flex">
        <UIcon v-if="tab.icon" :name="tab.icon" class="size-[22px]" />
        <span
          v-if="tab.badge !== undefined"
          class="absolute -top-1.5 -right-2.5 flex h-4 min-w-4 items-center justify-center rounded-full px-1 text-[11px]"
          :class="tab.value === 'saved' ? 'bg-hoply-600 text-white' : 'bg-ink-700 text-ink-50'"
        >
          {{ tab.badge }}<span class="sr-only">{{ ` ${String(tab.label).toLowerCase()}` }}</span>
        </span>
      </span>
      {{ tab.label }}
    </RouterLink>
  </nav>
</template>
