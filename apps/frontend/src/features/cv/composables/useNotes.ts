import { computed, toValue, type MaybeRefOrGetter } from 'vue'
import { withoutId, type Experience, type Note, type Profile, type Project } from '@/features/profile/api/profile.schema'
import { useProfileMutations } from '@/features/profile/composables/useProfile'

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

/** Answers kept as notes of the role or project they are about, or of the person. */
export function useNotes(profile: MaybeRefOrGetter<Profile | undefined>) {
  const mutations = useProfileMutations()
  const current = () => toValue(profile)
  const experience = computed(() => current()?.experience ?? [])
  const projects = computed(() => current()?.projects ?? [])

  function entryOf(place: Place): { list: List, entry: Experience | Project } | null {
    const [list, id] = place.split(':') as [List, string]
    const entries: (Experience | Project)[] = list === 'experience' ? experience.value : list === 'projects' ? projects.value : []
    const entry = entries.find(item => item.id === id)
    return entry ? { list, entry } : null
  }
  async function putEntry(list: List, entry: Experience | Project, change: Partial<Pick<Experience, 'bullets' | 'notes'>>) {
    const body = { ...withoutId(entry), ...change }
    if (list === 'experience') await mutations.experience.update.mutateAsync({ id: entry.id, body: body as Omit<Experience, 'id'> as never })
    else await mutations.projects.update.mutateAsync({ id: entry.id, body: body as Omit<Project, 'id'> as never })
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
    const target = entryOf(place)
    if (!target) {
      await mutations.patch.mutateAsync({ notes: [...(current()?.notes ?? []), note] })
      return { place: OTHER, text, skills }
    }
    await putEntry(target.list, target.entry, { notes: [...target.entry.notes, note] })
    return { place, text, skills }
  }
  async function forget({ place, text, skills }: Remembered) {
    const target = entryOf(place)
    const notes = target ? target.entry.notes : (current()?.notes ?? [])
    let at = -1
    notes.forEach((note, index) => {
      if (note.text === text && sameSkills(note.skills, skills)) at = index
    })
    if (at < 0) return
    const kept = notes.filter((_, index) => index !== at)
    if (target) await putEntry(target.list, target.entry, { notes: kept })
    else await mutations.patch.mutateAsync({ notes: kept })
  }
  return { experience, projects, entryOf, putEntry, whereLabel, remember, forget, mutations }
}
