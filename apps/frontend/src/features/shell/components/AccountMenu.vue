<script setup lang="ts">
import { computed } from 'vue'
import type { DropdownMenuItem } from '@nuxt/ui'
import { useProfileQuery } from '@/features/profile/composables/useProfile'
import { initialsOf, profileReadiness } from '@/features/profile/readiness'

/** The avatar in the header's corner: who you are, how complete the profile is, and the account menu's links. */
const props = defineProps<{ items: DropdownMenuItem[][] }>()

const profile = useProfileQuery()
const person = computed(() => profile.data.value)
const initials = computed(() => (person.value ? initialsOf(person.value.name) : ''))
const readiness = computed(() => (person.value ? profileReadiness(person.value).percent : 0))
const role = computed(() => [person.value?.headline, person.value?.city].filter(Boolean).join(' · '))
const menu = computed<DropdownMenuItem[][]>(() => [...(person.value ? [[{ type: 'label' as const, slot: 'person' as const }]] : []), ...props.items])
</script>

<template>
  <UDropdownMenu :items="menu" :content="{ align: 'end' }" :ui="{ content: 'w-72', item: 'min-h-11 text-[15px] font-bold' }">
    <button
      type="button"
      aria-label="Your profile and menu"
      class="flex min-h-11 items-center gap-2 rounded-full border border-default py-[3px] pr-2.5 pl-[3px] text-toned hover:border-accented"
      data-test="account"
    >
      <span class="flex size-9 items-center justify-center rounded-full border-2 border-hoply-500 bg-hoply-900 text-sm font-extrabold text-hoply-200" data-test="account-avatar">
        <template v-if="initials">{{ initials }}</template>
        <UIcon v-else name="i-lucide-user" class="size-4" />
      </span>
      <UIcon name="i-lucide-chevron-down" class="size-4" />
    </button>
    <template #person>
      <span class="flex w-full flex-col gap-3 py-1 text-default">
        <span class="flex flex-col gap-0.5">
          <span class="text-[15px] font-extrabold text-highlighted">{{ person?.name || 'Your profile' }}</span>
          <span v-if="role" class="text-[13px] font-semibold text-muted">{{ role }}</span>
        </span>
        <span class="flex flex-col gap-1.5">
          <span class="flex justify-between text-xs font-bold text-muted">
            <span>Profile complete</span>
            <span class="text-hoply-200" data-test="account-readiness">{{ readiness }}%</span>
          </span>
          <span class="h-1.5 rounded-full bg-accented">
            <span class="block h-1.5 rounded-full bg-hoply-500" :style="{ width: `${readiness}%` }" />
          </span>
        </span>
      </span>
    </template>
  </UDropdownMenu>
</template>
