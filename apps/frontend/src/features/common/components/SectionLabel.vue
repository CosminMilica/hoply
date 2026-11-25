<script setup lang="ts">
withDefaults(defineProps<{ as?: string }>(), { as: 'h2' })
</script>

<template>
  <component :is="as" class="m-0 text-xs font-extrabold tracking-wider text-muted uppercase">
    <slot />
  </component>
</template>
