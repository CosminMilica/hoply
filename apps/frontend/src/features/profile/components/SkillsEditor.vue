<script setup lang="ts">
import type { Profile } from '../api/profile.schema'
import { useProfileMutations } from '../composables/useProfile'
import TagListEditor from './TagListEditor.vue'

defineProps<{ profile: Profile }>()
const { setSkills } = useProfileMutations()
</script>

<template>
  <TagListEditor :items="profile.skills" label="Add a skill" placeholder="Add a skill, e.g. Next.js" @change="skills => setSkills.mutate(skills)" />
</template>
