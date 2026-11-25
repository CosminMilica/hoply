<script setup lang="ts">
import { computed } from 'vue'
import type { NavigationMenuItem } from '@nuxt/ui'
import { useApplicationMenu } from '@/shared/application/menus'

/** The phone's navigation: the layout's primary menu as tabs at the bottom of the screen. */
const menu = useApplicationMenu('app', 'primary')
const tabs = computed(() => menu.value.flat().filter((item): item is NavigationMenuItem & { to: NonNullable<NavigationMenuItem['to']> } => Boolean(item.to)))
</script>

<template>
  <nav
    aria-label="Main"
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
          class="absolute -top-1.5 -right-2.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-hoply-600 px-1 text-[11px] text-white"
        >
          {{ tab.badge }}<span class="sr-only"> saved</span>
        </span>
      </span>
      {{ tab.label }}
    </RouterLink>
  </nav>
</template>
