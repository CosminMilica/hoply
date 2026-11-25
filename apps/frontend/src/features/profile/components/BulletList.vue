<script setup lang="ts">
import type { Bullet } from '../api/profile.schema'

defineProps<{ bullets: Bullet[] }>()
</script>

<template>
  <ul class="flex flex-col gap-2">
    <li v-for="(bullet, index) in bullets" :key="index" class="flex gap-2 text-sm leading-snug font-medium">
      <span aria-hidden="true" class="font-extrabold">·</span>
      <span class="flex flex-col gap-0.5">
        <span>{{ bullet.text }}</span>
        <span v-if="bullet.result" class="flex items-center gap-1 text-[13px] font-bold text-hoply-300">
          <UIcon name="i-lucide-trending-up" class="size-3.5 shrink-0" /><span class="sr-only">Result:</span>{{ bullet.result }}
        </span>
      </span>
    </li>
  </ul>
</template>
