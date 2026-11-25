<script setup lang="ts">
import { computed } from 'vue'
import { splitKeywords } from '../highlight'

const props = defineProps<{ text: string, keywords: string[] }>()
const parts = computed(() => splitKeywords(props.text, props.keywords))
</script>

<template>
  <span><template v-for="(part, index) in parts" :key="index"><mark v-if="part.hit" class="rounded-[3px] bg-hoply-200 px-0.5 font-bold text-hoply-950">{{ part.text }}</mark><template v-else>{{ part.text }}</template></template></span>
</template>
