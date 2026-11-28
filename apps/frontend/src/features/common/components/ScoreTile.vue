<script setup lang="ts">
withDefaults(defineProps<{ score: number, size?: 'sm' | 'md' | 'lg', label?: boolean }>(), { size: 'md', label: false })
</script>

<template>
  <span
    class="flex shrink-0 flex-col items-center justify-center rounded-xl bg-crashr-900 font-extrabold text-crashr-200"
    :class="{ 'size-12 text-[15px]': size === 'sm', 'size-14 text-lg': size === 'md', 'size-16 text-lg': size === 'lg' }"
    :aria-label="`${score}% match`"
  >
    <span aria-hidden="true" class="leading-none">{{ score }}%</span>
    <span v-if="label" aria-hidden="true" class="mt-0.5 text-[10px] tracking-wider uppercase">match</span>
  </span>
</template>
