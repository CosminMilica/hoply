<script setup lang="ts">
import ToggleChip from '@/features/common/components/ToggleChip.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'

defineProps<{ choices: { id: string, label: string, pressed: boolean }[], name: string }>()
defineEmits<{ pick: [id: string] }>()
</script>

<template>
  <div class="flex flex-col gap-1.5" role="group" :aria-label="`Where did you do this? ${name}`">
    <SectionLabel as="span">Where did you do this?</SectionLabel>
    <div class="flex flex-wrap gap-1.5">
      <ToggleChip v-for="choice in choices" :key="choice.id" :label="choice.label" :pressed="choice.pressed" @toggle="$emit('pick', choice.id)" />
    </div>
  </div>
</template>
