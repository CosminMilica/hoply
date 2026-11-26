<script setup lang="ts">
/** filtersLabel names the first way out: changing the filters, or resetting them where the filters are in view. */
withDefaults(defineProps<{ filtersLabel?: string }>(), { filtersLabel: 'Change filters' })
const emit = defineEmits<{ changeFilters: [], resetSkipped: [] }>()
</script>

<template>
  <div class="mt-1 flex flex-col gap-2.5 rounded-[18px] border border-default bg-elevated px-[22px] py-7">
    <h2 class="m-0 text-2xl leading-tight font-extrabold tracking-tight">You are all caught up</h2>
    <p class="m-0 text-[15px] leading-normal font-medium text-muted">No more jobs for these filters. Change them, or bring back the ones you skipped.</p>
    <UButton :label="filtersLabel" size="lg" block class="mt-1.5 justify-center" @click="emit('changeFilters')" />
    <UButton label="Bring back skipped jobs" color="neutral" variant="outline" size="lg" block class="justify-center" @click="emit('resetSkipped')" />
    <UButton :to="{ name: 'hoply-saved' }" label="See saved jobs" variant="link" color="neutral" class="justify-center text-muted underline" />
  </div>
</template>
