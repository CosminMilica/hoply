<script setup lang="ts">
import { computed } from 'vue'
import { t } from '@/features/common/i18n'
import { bookmarklet } from '../sendToHoply'

const href = computed(() => bookmarklet(window.location.origin))
</script>

<template>
  <div class="flex flex-col gap-2 rounded-[14px] border-[1.5px] border-dashed border-ink-600 p-3.5" data-test="send-to-hoply">
    <p class="m-0 text-sm font-extrabold">{{ t.jobs.sendToHoply.title }}</p>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">
      {{ t.jobs.sendToHoply.text }}
    </p>
    <a
      :href="href"
      class="flex items-center gap-1.5 self-start rounded-lg bg-hoply-600 px-3 py-1.5 text-sm font-extrabold text-white"
      :title="t.jobs.sendToHoply.drag"
      @click.prevent
    >
      <UIcon name="i-lucide-bookmark-plus" class="size-4" />{{ t.jobs.sendToHoply.button }}
    </a>
  </div>
</template>
