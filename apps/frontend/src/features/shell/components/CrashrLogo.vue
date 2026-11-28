<script setup lang="ts">
import { useApplicationRouting } from '@/shared/application/context'
import { t } from '@/features/common/i18n'

const landing = useApplicationRouting().landing
</script>

<template>
  <RouterLink :to="landing ? { name: landing } : '/'" class="flex min-h-11 items-center gap-2 text-highlighted" :aria-label="t.shell.logo">
    <svg width="28" height="28" viewBox="0 0 28 28" aria-hidden="true">
      <circle cx="14" cy="9" r="6" fill="#2A4BFF" />
      <path d="M3 22q11-9 22 0" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" />
    </svg>
    <span class="text-[22px] font-extrabold tracking-tight">crashr<span class="text-sm text-crashr-400">.xyz</span></span>
  </RouterLink>
</template>
