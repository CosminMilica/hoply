import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import type { Profile } from '@/features/profile/api/profile.schema'
import { LEVELS, type JobFilters } from '../api/jobs.schema'

/** What the person is looking for, kept while they move between the deck and the jobs it opens. */
export const useJobSearchStore = defineStore('job-search', () => {
  const query = ref('')
  const location = ref('')
  const includeRemote = ref(true)
  const areas = ref<string[]>([])
  const level = ref('Any')
  /** The person's country, from their profile: the search stays in it. */
  const country = ref('')
  const stage = ref<'filters' | 'deck'>('filters')
  const prefilled = ref(false)

  const filters = computed<JobFilters>(() => ({
    q: query.value,
    city: location.value,
    country: country.value,
    includeRemote: includeRemote.value,
    areas: areas.value,
    level: level.value,
  }))

  const anywhere = computed(() => (country.value ? `Anywhere in ${country.value}` : 'Anywhere'))

  const summary = computed(() => [
    query.value.trim() ? `“${query.value.trim()}”` : '',
    areas.value.length ? areas.value.join(', ') : 'All areas',
    location.value.trim() ? `${location.value.trim()}${includeRemote.value ? ' + remote' : ''}` : anywhere.value,
    level.value !== 'Any' ? level.value : '',
  ].filter(Boolean).join(' · '))

  function toggleArea(area: string) {
    areas.value = areas.value.includes(area) ? areas.value.filter(item => item !== area) : [...areas.value, area]
  }

  function reset() {
    query.value = ''
    location.value = ''
    includeRemote.value = true
    areas.value = []
    level.value = 'Any'
  }

  /** Once per session: the city, remote work and a single target seniority from the person's preferences. */
  function prefill(profile: Pick<Profile, 'city' | 'work_modes' | 'target_seniority'>) {
    if (prefilled.value) return
    prefilled.value = true
    if (profile.city.trim()) location.value = profile.city.trim()
    if (profile.work_modes.length) includeRemote.value = profile.work_modes.includes('remote')
    const [only, ...others] = profile.target_seniority
    const match = !others.length && LEVELS.find(item => item.toLowerCase() === only)
    if (match) level.value = match
  }

  return { query, location, includeRemote, areas, level, country, stage, prefilled, filters, anywhere, summary, toggleArea, reset, prefill }
})
