import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { useQueryClient } from '@tanstack/vue-query'
import { experienceInput, projectInput, type Experience, type Note, type Profile, type Project } from '@/features/profile/api/profile.schema'
import { profileKeys, useProfileMutations } from '@/features/profile/composables/useProfile'

export const OTHER = 'other'

export type List = 'experience' | 'projects'
/** Where an answer goes: `experience:<id>`, `projects:<id>`, or `other` (the person's own notes). */
export type Place = string
/** What an answer added to the profile, so that Change takes exactly that back out. */
export interface Remembered {
  place: Place
  text: string
  skills: string[]
}

const sameSkills = (a: string[], b: string[]) => a.length === b.length && a.every((skill, index) => skill === b[index])

// Saves to the profile run one at a time, each on the profile the save before it left in the cache: two answers
// about the same role never overwrite each other.
let queue: Promise<unknown> = Promise.resolve()
function serial<T>(work: () => Promise<T>): Promise<T> {
  const run = queue.then(work)
  queue = run.catch(() => undefined)
  return run
}

type EntryChange = Partial<Pick<Experience, 'bullets' | 'notes'>>
type Target = { list: 'experience', entry: Experience } | { list: 'projects', entry: Project }

/** Answers kept as notes of the role or project they are about, or of the person. */
export function useNotes(profile: MaybeRefOrGetter<Profile | undefined>) {
  const client = useQueryClient()
  const mutations = useProfileMutations()
  const current = () => toValue(profile)
  const latest = () => client.getQueryData<Profile>(profileKeys.me) ?? current()
  const experience = computed(() => current()?.experience ?? [])
  const projects = computed(() => current()?.projects ?? [])

  function find(person: Profile | undefined, place: Place): Target | null {
    const [list, id] = place.split(':') as [List, string]
    const role = list === 'experience' ? person?.experience.find(item => item.id === id) : undefined
    if (role) return { list: 'experience', entry: role }
    const project = list === 'projects' ? person?.projects.find(item => item.id === id) : undefined
    return project ? { list: 'projects', entry: project } : null
  }
  const entryOf = (place: Place) => find(current(), place)

  /** Changes one role or project, from its latest saved state; nothing when it is gone. */
  function updateEntry(place: Place, change: (entry: Experience | Project) => EntryChange) {
    return serial(async () => {
      const target = find(latest(), place)
      if (target?.list === 'experience') {
        await mutations.experience.update.mutateAsync({ id: target.entry.id, body: experienceInput(target.entry, change(target.entry)) })
      }
      else if (target?.list === 'projects') {
        await mutations.projects.update.mutateAsync({ id: target.entry.id, body: projectInput(target.entry, change(target.entry)) })
      }
    })
  }
  function whereLabel(place: Place): string {
    const target = entryOf(place)
    if (!target) return 'Your notes'
    if (target.list === 'projects') return `the ${(target.entry as Project).name} project`
    const role = target.entry as Experience
    return role.company ? `${role.role} at ${role.company}` : role.role
  }
  /** An answer becomes a note of the role or project it is about, or of the person: material Claude writes from. */
  async function remember(place: Place, text: string, skills: string[]): Promise<Remembered> {
    const note: Note = { text, skills }
    if (!find(latest(), place)) {
      await serial(() => mutations.patch.mutateAsync({ notes: [...(latest()?.notes ?? []), note] }))
      return { place: OTHER, text, skills }
    }
    await updateEntry(place, entry => ({ notes: [...entry.notes, note] }))
    return { place, text, skills }
  }
  async function forget({ place, text, skills }: Remembered) {
    const without = (notes: Note[]) => {
      let at = -1
      notes.forEach((note, index) => {
        if (note.text === text && sameSkills(note.skills, skills)) at = index
      })
      return at < 0 ? null : notes.filter((_, index) => index !== at)
    }
    if (find(latest(), place)) {
      await updateEntry(place, entry => ({ notes: without(entry.notes) ?? entry.notes }))
      return
    }
    await serial(async () => {
      const kept = without(latest()?.notes ?? [])
      if (kept) await mutations.patch.mutateAsync({ notes: kept })
    })
  }
  return { experience, projects, entryOf, updateEntry, whereLabel, remember, forget, mutations }
}
