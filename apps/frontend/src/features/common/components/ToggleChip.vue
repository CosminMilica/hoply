<script setup lang="ts">
defineProps<{ label: string, pressed: boolean }>()
defineEmits<{ toggle: [] }>()
</script>

<template>
  <UButton
    :label="label"
    :aria-pressed="pressed"
    :color="pressed ? 'primary' : 'neutral'"
    :variant="pressed ? 'solid' : 'outline'"
    class="px-4"
    @click="$emit('toggle')"
  />
</template>
