<script setup lang="ts">
defineProps<{ value: number, label: string }>()
</script>

<template>
  <div
    role="progressbar"
    :aria-label="label"
    :aria-valuenow="value"
    aria-valuemin="0"
    aria-valuemax="100"
    class="h-1.5 overflow-hidden rounded-full bg-ink-800"
  >
    <div class="h-full rounded-full bg-crashr-500 transition-[width]" :style="{ width: `${Math.min(100, Math.max(0, value))}%` }" />
  </div>
</template>
