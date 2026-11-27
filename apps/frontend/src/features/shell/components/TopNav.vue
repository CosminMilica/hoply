<script setup lang="ts">
import { useNavTabs } from '../tabs'

/** The wide screen's navigation: the layout's primary menu as tabs in the header. */
const tabs = useNavTabs()
</script>

<template>
  <nav aria-label="Main" data-shell="top-nav" class="flex w-full gap-1">
    <RouterLink
      v-for="tab in tabs"
      :key="String(tab.value)"
      :to="tab.to"
      :aria-current="tab.active ? 'page' : undefined"
      class="flex min-h-11 items-center gap-2 rounded-[10px] px-3.5 text-[15px] whitespace-nowrap"
      :class="tab.active ? 'bg-accented font-extrabold text-highlighted' : 'font-bold text-muted hover:text-highlighted'"
    >
      <UIcon v-if="tab.icon" :name="tab.icon" class="size-[18px]" :class="tab.active ? 'text-hoply-400' : ''" />
      {{ tab.label }}
      <span
        v-if="tab.badge !== undefined"
        class="flex h-5 min-w-5 items-center justify-center rounded-full px-1.5 text-xs font-extrabold"
        :class="tab.value === 'saved' ? 'bg-hoply-600 text-white' : 'bg-ink-700 text-ink-50'"
      >
        {{ tab.badge }}<span class="sr-only">{{ ` ${String(tab.label).toLowerCase()}` }}</span>
      </span>
    </RouterLink>
  </nav>
</template>
