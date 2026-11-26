<script setup lang="ts">
import { computed } from 'vue'
import { bookmarklet } from '../sendToHoply'

const href = computed(() => bookmarklet(window.location.origin))
</script>

<template>
  <div class="flex flex-col gap-2 rounded-[14px] border-[1.5px] border-dashed border-ink-600 p-3.5" data-test="send-to-hoply">
    <p class="m-0 text-sm font-extrabold">Send jobs here from LinkedIn, eJobs or any site</p>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">
      Drag this button to your bookmarks bar. On a job ad, click the bookmark: the ad opens here, already read. Select part of the page first to send only that.
    </p>
    <a
      :href="href"
      class="flex items-center gap-1.5 self-start rounded-lg bg-hoply-600 px-3 py-1.5 text-sm font-extrabold text-white"
      title="Drag me to your bookmarks bar"
      @click.prevent
    >
      <UIcon name="i-lucide-bookmark-plus" class="size-4" />Send to Hoply
    </a>
  </div>
</template>
