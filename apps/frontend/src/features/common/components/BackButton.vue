<script setup lang="ts">
import { useRouter, type RouteLocationRaw } from 'vue-router'

const props = withDefaults(defineProps<{ fallback: RouteLocationRaw, label?: string }>(), { label: 'Back' })
const router = useRouter()

function back() {
  if (window.history.state?.back) router.back()
  else router.push(props.fallback)
}
</script>

<template>
  <UButton variant="ghost" color="neutral" icon="i-lucide-chevron-left" :label="label" class="self-start px-1 text-toned" @click="back" />
</template>
