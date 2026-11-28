<script setup lang="ts">
import { t } from '@/features/common/i18n'
import type { Profile } from '../api/profile.schema'
import { useProfileMutations } from '../composables/useProfile'
import TagListEditor from './TagListEditor.vue'

defineProps<{ profile: Profile }>()
const { setSkills } = useProfileMutations()
</script>

<template>
  <TagListEditor :items="profile.skills" :label="t.profile.skills.add" :placeholder="t.profile.skills.placeholder" @change="skills => setSkills.mutate(skills)" />
</template>
