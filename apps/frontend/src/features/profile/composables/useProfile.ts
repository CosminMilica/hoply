import { useMutation, useQuery, useQueryClient } from '@tanstack/vue-query'
import { profileApi } from '../api/profileApi'
import type { Profile } from '../api/profile.schema'

export const profileKeys = { me: ['profile', 'me'] as const }

export function useProfileQuery() {
  return useQuery({ queryKey: profileKeys.me, queryFn: profileApi.get })
}

type ListKey = 'experience' | 'projects' | 'education' | 'certifications' | 'languages'
type Entry<K extends ListKey> = Profile[K][number]

/** The profile's mutations; each writes the service's answer into the cached profile. */
export function useProfileMutations() {
  const client = useQueryClient()
  const replace = (profile: Profile) => client.setQueryData(profileKeys.me, profile)
  const update = (change: (profile: Profile) => Profile) =>
    client.setQueryData<Profile>(profileKeys.me, current => (current ? change(current) : current))

  // The entry forms show the service's refusal next to their fields, not as a toast.
  const inline = { onError: () => {} }

  /** Add, update and remove for one list; new entries go first or last as the service orders them. */
  function list<K extends ListKey, Input>(
    key: K,
    api: { add: (body: Input) => Promise<Entry<K>>, update: (id: string, body: Input) => Promise<Entry<K>>, remove: (id: string) => Promise<void> },
    newest: 'first' | 'last',
  ) {
    const set = (change: (items: Entry<K>[]) => Entry<K>[]) => update(p => ({ ...p, [key]: change(p[key] as Entry<K>[]) }))
    return {
      add: useMutation({
        ...inline,
        mutationFn: api.add,
        onSuccess: (added: Entry<K>) => set(items => (newest === 'first' ? [added, ...items] : [...items, added])),
      }),
      update: useMutation({
        ...inline,
        mutationFn: ({ id, body }: { id: string, body: Input }) => api.update(id, body),
        onSuccess: (saved: Entry<K>) => set(items => items.map(item => (item.id === saved.id ? saved : item))),
      }),
      remove: useMutation({
        mutationFn: api.remove,
        onSuccess: (_: void, id: string) => set(items => items.filter(item => item.id !== id)),
      }),
    }
  }

  return {
    patch: useMutation({ mutationFn: profileApi.patch, onSuccess: replace }),
    setSkills: useMutation({ mutationFn: profileApi.putSkills, onSuccess: replace }),
    experience: list('experience', profileApi.experience, 'first'),
    projects: list('projects', profileApi.projects, 'first'),
    education: list('education', profileApi.education, 'last'),
    certifications: list('certifications', profileApi.certifications, 'last'),
    languages: {
      add: useMutation({
        mutationFn: profileApi.languages.add,
        onSuccess: added => update(p => ({ ...p, languages: [...p.languages, added] })),
      }),
      remove: useMutation({
        mutationFn: profileApi.languages.remove,
        onSuccess: (_, id) => update(p => ({ ...p, languages: p.languages.filter(item => item.id !== id) })),
      }),
    },
  }
}
