<script setup lang="ts">
defineProps<{ label: string, tone: 'missing' | 'added' | 'muted' }>()
</script>

<template>
  <span
    class="rounded-md px-2 py-0.5 text-[11px] font-extrabold tracking-wider uppercase"
    :class="{ 'bg-ember-500 text-ink-950': tone === 'missing', 'bg-hoply-600 text-white': tone === 'added', 'bg-accented text-highlighted': tone === 'muted' }"
  >{{ label }}</span>
</template>
