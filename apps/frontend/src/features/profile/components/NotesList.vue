<script setup lang="ts">
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Note } from '../api/profile.schema'

defineProps<{ notes: Note[], of: string, pending?: boolean }>()
defineEmits<{ remove: [index: number] }>()
</script>

<template>
  <div v-if="notes.length" class="flex flex-col gap-1.5" data-test="notes">
    <SectionLabel as="h3">Your notes for CVs</SectionLabel>
    <p class="m-0 text-[13px] leading-snug font-medium text-muted">What you told Prepare your CV. Claude writes your CV from these; they never appear as they are.</p>
    <ul class="flex flex-col gap-1.5">
      <li v-for="(note, index) in notes" :key="index" class="flex items-start gap-1 rounded-[10px] border border-default bg-elevated py-1.5 ps-3 pe-1 text-sm font-medium">
        <span class="min-w-0 flex-1 whitespace-pre-line">{{ note.text }}<span v-if="note.skills.length" class="ms-1.5 text-xs font-bold text-hoply-300">{{ note.skills.join(', ') }}</span></span>
        <UButton color="neutral" variant="ghost" icon="i-lucide-x" :aria-label="`Remove note ${index + 1} of ${of}`" :disabled="pending" class="size-9 shrink-0 justify-center text-muted" @click="$emit('remove', index)" />
      </li>
    </ul>
  </div>
</template>
