<script setup lang="ts">
import { computed, ref } from 'vue'
import KeywordChips from '@/features/common/components/KeywordChips.vue'
import ScoreTile from '@/features/common/components/ScoreTile.vue'
import SectionLabel from '@/features/common/components/SectionLabel.vue'
import type { Decision, ScoredJob } from '../composables/useDeck'
import { placeOf, sourceLine, tagline } from '../match'

const props = defineProps<{ deck: ScoredJob[], toast: string }>()
const emit = defineEmits<{
  decide: [decision: Decision, job: ScoredJob]
  details: [job: ScoredJob]
  undo: []
  changeFilters: []
  resetSkipped: []
}>()

const SWIPE = 100
const FLY_MS = 240
const current = computed(() => props.deck[0])
const next = computed(() => props.deck[1])
const drag = ref({ x: 0, y: 0, active: false, snap: false, moved: false, startX: 0, startY: 0 })
const busy = ref(false)

const dy = computed(() => Math.min(0, drag.value.y))
const vertical = computed(() => Math.abs(dy.value) > Math.abs(drag.value.x))
const progress = computed(() => Math.min(1, Math.max(Math.abs(drag.value.x), Math.abs(dy.value)) / 110))
const cardStyle = computed(() => ({
  transform: `translate(${drag.value.x}px, ${dy.value}px) rotate(${drag.value.x / 18}deg)`,
  transition: drag.value.snap ? `transform ${FLY_MS}ms ease` : 'none',
}))
const backStyle = computed(() => ({ transform: `scale(${0.95 + 0.05 * progress.value}) translateY(${12 * (1 - progress.value)}px)` }))
const stamp = computed(() => ({
  save: !vertical.value && drag.value.x > 0 ? progress.value : 0,
  skip: !vertical.value && drag.value.x < 0 ? progress.value : 0,
  tailor: vertical.value ? progress.value : 0,
}))

function reset(snap = false) {
  drag.value = { ...drag.value, x: 0, y: 0, active: false, snap, moved: false }
}

function decide(decision: Decision) {
  const job = current.value
  if (!job || busy.value) return
  busy.value = true
  const [x, y] = decision === 'tailor' ? [0, -760] : [decision === 'save' ? 520 : -520, 0]
  drag.value = { ...drag.value, active: false, snap: true, x, y }
  window.setTimeout(() => {
    reset()
    busy.value = false
    emit('decide', decision, job)
  }, FLY_MS)
}

function onDown(event: PointerEvent) {
  if (busy.value) return
  ;(event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId)
  drag.value = { x: 0, y: 0, active: true, snap: false, moved: false, startX: event.clientX, startY: event.clientY }
}
function onMove(event: PointerEvent) {
  if (!drag.value.active) return
  const x = event.clientX - drag.value.startX
  const y = event.clientY - drag.value.startY
  drag.value = { ...drag.value, x, y, moved: drag.value.moved || Math.abs(x) > 6 || Math.abs(y) > 6 }
}
function onUp() {
  const { active, x, y, moved } = drag.value
  if (!active) return
  drag.value = { ...drag.value, active: false }
  if (y < -110 && Math.abs(y) > Math.abs(x)) decide('tailor')
  else if (x > SWIPE) decide('save')
  else if (x < -SWIPE) decide('skip')
  else if (!moved && current.value) {
    reset()
    emit('details', current.value)
  }
  else reset(true)
}
</script>

<template>
  <template v-if="current">
    <div class="relative min-h-[360px] flex-1 min-[900px]:max-h-[700px]">
      <div v-if="next" aria-hidden="true" class="absolute inset-0 rounded-[18px] border border-default bg-accented" :style="backStyle" />
      <article
        :aria-label="`${current.title} at ${current.company}`"
        data-test="job-card"
        class="absolute inset-0 flex cursor-grab touch-none flex-col gap-3.5 overflow-hidden rounded-[18px] border-[1.5px] border-ink-700 bg-elevated p-[22px] select-none"
        :style="cardStyle"
        @pointerdown="onDown"
        @pointermove="onMove"
        @pointerup="onUp"
        @pointercancel="onUp"
      >
        <span aria-hidden="true" class="pointer-events-none absolute top-[18px] left-[18px] z-[2] -rotate-[8deg] rounded-xl bg-hoply-600 px-3.5 py-1.5 text-base font-extrabold tracking-wider text-white" :style="{ opacity: stamp.save }">SAVE</span>
        <span aria-hidden="true" class="pointer-events-none absolute top-[18px] right-[18px] z-[2] rotate-[8deg] rounded-xl bg-ember-500 px-3.5 py-1.5 text-base font-extrabold tracking-wider text-ink-950" :style="{ opacity: stamp.skip }">SKIP</span>
        <span aria-hidden="true" class="pointer-events-none absolute top-[18px] left-1/2 z-[2] -translate-x-1/2 rounded-xl border-[1.5px] border-hoply-400 bg-accented px-3.5 py-1.5 text-base font-extrabold tracking-wider whitespace-nowrap text-hoply-400" :style="{ opacity: stamp.tailor }">TAILOR CV</span>

        <div class="flex items-center justify-between gap-3">
          <span class="text-[13px] font-bold">{{ sourceLine(current) }}</span>
          <ScoreTile :score="current.score" size="lg" label />
        </div>
        <div class="flex flex-col gap-1">
          <h2 class="m-0 text-[28px] leading-tight font-extrabold tracking-tight">{{ current.title }}</h2>
          <span class="text-base font-medium text-muted">{{ [current.company, placeOf(current)].filter(Boolean).join(' · ') }}</span>
          <span class="text-[13px] font-bold text-toned">{{ tagline(current) }}</span>
        </div>
        <p class="m-0 line-clamp-4 text-[15px] leading-normal font-medium">{{ current.about }}</p>
        <div class="flex flex-col gap-1.5">
          <SectionLabel as="h3">What you will do</SectionLabel>
          <ul class="flex flex-col gap-1.5">
            <li v-for="item in current.responsibilities.slice(0, 3)" :key="item" class="flex gap-2 text-sm leading-snug font-medium">
              <span aria-hidden="true" class="font-extrabold">·</span><span class="line-clamp-2">{{ item }}</span>
            </li>
          </ul>
        </div>
        <KeywordChips :matched="current.matched.slice(0, 4)" :missing="current.missing.slice(0, 3)" />
        <span class="mt-auto text-[13px] font-bold text-ink-500">
          <span class="pointer-fine:hidden">Swipe right to save, left to skip, up to tailor your CV. Tap for details.</span>
          <span class="hidden pointer-fine:inline">Drag the card or use the buttons below. Click for details.</span>
        </span>
      </article>

      <div
        v-if="toast"
        role="status"
        class="absolute inset-x-3 bottom-3 z-[3] flex min-h-11 items-center justify-between gap-3 rounded-[10px] bg-ink-800 ps-4 pe-1.5 text-sm font-bold"
      >
        <span class="truncate">{{ toast }}</span>
        <UButton variant="ghost" color="neutral" label="Undo" class="text-hoply-400" @click="emit('undo')" />
      </div>
    </div>

    <div class="flex items-center justify-center gap-3.5 py-0.5">
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="outline"
        aria-label="Skip this job"
        class="size-12 justify-center rounded-full"
        :ui="{ leadingIcon: 'size-6' }"
        @click="decide('skip')"
      />
      <UButton color="neutral" variant="outline" label="Details" @click="emit('details', current)" />
      <UButton color="neutral" label="Tailor CV" @click="decide('tailor')" />
      <UButton
        icon="i-lucide-heart"
        aria-label="Save this job"
        class="size-12 justify-center rounded-full"
        :ui="{ leadingIcon: 'size-6' }"
        @click="decide('save')"
      />
    </div>
  </template>

  <div v-else class="mt-1 flex flex-col gap-2.5 rounded-[18px] border border-default bg-elevated px-[22px] py-7">
    <h2 class="m-0 text-2xl leading-tight font-extrabold tracking-tight">You are all caught up</h2>
    <p class="m-0 text-[15px] leading-normal font-medium text-muted">No more jobs for these filters. Change them, or bring back the ones you skipped.</p>
    <UButton label="Change filters" size="lg" block class="mt-1.5 justify-center" @click="emit('changeFilters')" />
    <UButton label="Bring back skipped jobs" color="neutral" variant="outline" size="lg" block class="justify-center" @click="emit('resetSkipped')" />
    <UButton :to="{ name: 'hoply-saved' }" label="See saved jobs" variant="link" color="neutral" class="justify-center text-muted underline" />
  </div>
</template>
