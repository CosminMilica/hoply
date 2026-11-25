<script setup lang="ts">
defineProps<{ title: string, icon: string, summary: string }>()
const open = defineModel<boolean>('open', { default: false })
</script>

<template>
  <section class="border-t-[1.5px] border-ink-700 pt-0.5">
    <UCollapsible v-model:open="open" :unmount-on-hide="false">
      <button type="button" class="flex min-h-13 w-full items-center gap-2.5 text-left text-highlighted">
        <span class="flex size-8 shrink-0 items-center justify-center rounded-lg bg-accented text-hoply-400">
          <UIcon :name="icon" class="size-[18px]" />
        </span>
        <h2 class="m-0 flex-1 text-[17px] font-extrabold tracking-tight">{{ title }}</h2>
        <span class="text-[13px] font-bold text-muted">{{ summary }}</span>
        <UIcon name="i-lucide-chevron-down" class="size-[18px] text-muted transition-transform" :class="{ 'rotate-180': open }" />
      </button>
      <template #content>
        <div class="flex flex-col gap-2.5 pt-0.5 pb-3.5">
          <slot />
        </div>
      </template>
    </UCollapsible>
  </section>
</template>
