import { computed, ref, watch } from 'vue'
import { defineStore } from 'pinia'
import { useLocalStorage } from '@vueuse/core'
import { areaLabel, t } from '@/features/common/i18n'
import type { Domain, Profile } from '@/features/profile/api/profile.schema'
import { CITIES, DOMAIN_AREAS, LEVELS, type JobFilters } from '../api/jobs.schema'

/** A city without case or diacritics, its Romanian name folded onto the English one: "București" is "bucharest". */
const cityKey = (city: string) => {
  const folded = city.normalize('NFD').replace(/\p{Diacritic}/gu, '').trim().toLowerCase()
  return folded === 'bucuresti' ? 'bucharest' : folded
}

/** What the person is looking for, kept while they move between the deck and the jobs it opens. */
export const useJobSearchStore = defineStore('job-search', () => {
  const query = ref('')
  const location = ref('')
  const includeRemote = ref(true)
  const areas = ref<string[]>([])
  const level = ref('Any')
  /** The person's country, from their profile: the search stays in it. */
  const country = ref('')
  /** The fields of work the person picked, from their profile: the search stays in them. */
  const domains = ref<Domain[]>([])
  /** The field of work the filters show; '' for all the person picked. */
  const domain = ref<Domain | ''>('')
  const stage = ref<'filters' | 'deck'>('filters')
  /** How the jobs are shown: a grid of cards, or the swipe deck; remembered on this device. */
  const view = useLocalStorage<'grid' | 'swipe'>('hoply.jobs.view', 'grid')
  const prefilled = ref(false)

  /** The field whose areas the filters offer: the one chosen, or the only one the person picked. */
  const shownDomain = computed<Domain | ''>(() => domain.value || (domains.value.length === 1 ? domains.value[0]! : ''))

  // A field the person no longer picks leaves the filters.
  watch(domains, (picked) => {
    if (domain.value && !picked.includes(domain.value)) pickDomain('')
  })

  const filters = computed<JobFilters>(() => ({
    q: query.value,
    city: location.value,
    country: country.value,
    includeRemote: includeRemote.value,
    areas: areas.value,
    domains: domain.value ? [domain.value] : domains.value,
    level: level.value,
  }))

  const anywhere = computed(() => (country.value ? t.value.jobs.search.anywhereIn(country.value) : t.value.jobs.search.anywhere))
  /** The city chips: anywhere in the country, then the cities Find jobs offers, the chosen one pressed. */
  const cityOptions = computed(() => [{ label: anywhere.value, value: '' }, ...CITIES.map(city => ({ label: t.value.jobs.search.cities[city], value: city }))]
    .map(city => ({ ...city, pressed: cityKey(location.value) === cityKey(city.value) })))

  /** The field tabs: all the person picked, then each of them; none when they picked one. */
  const domainOptions = computed(() => (domains.value.length > 1
    ? [{ value: '' as const, label: t.value.jobs.search.allDomains }, ...domains.value.map(value => ({ value, label: t.value.common.domains[value] }))]
        .map(option => ({ ...option, pressed: domain.value === option.value }))
    : []))
  /** The area chips of the field shown. */
  const areaOptions = computed(() => (shownDomain.value ? DOMAIN_AREAS[shownDomain.value] : [])
    .map(area => ({ value: area, label: areaLabel(area), pressed: areas.value.includes(area) })))
  const levelOptions = computed(() => LEVELS.map(value => ({ value, label: t.value.jobs.levels[value], pressed: level.value === value })))

  const summary = computed(() => [
    query.value.trim() ? `“${query.value.trim()}”` : '',
    areas.value.length
      ? areas.value.map(areaLabel).join(', ')
      : shownDomain.value ? t.value.common.domains[shownDomain.value] : t.value.jobs.search.allAreas,
    location.value.trim() ? `${location.value.trim()}${includeRemote.value ? t.value.jobs.search.plusRemote : ''}` : anywhere.value,
    level.value !== 'Any' ? t.value.jobs.levels[level.value as (typeof LEVELS)[number]] : '',
  ].filter(Boolean).join(' · '))

  function pickDomain(value: Domain | '') {
    domain.value = value
    areas.value = []
  }

  function toggleArea(area: string) {
    areas.value = areas.value.includes(area) ? areas.value.filter(item => item !== area) : [...areas.value, area]
  }

  function reset() {
    query.value = ''
    location.value = ''
    includeRemote.value = true
    pickDomain('')
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

  return {
    query, location, includeRemote, areas, level, country, domains, domain, shownDomain, stage, view, prefilled, filters, anywhere,
    cityOptions, domainOptions, areaOptions, levelOptions, summary, pickDomain, toggleArea, reset, prefill,
  }
})
