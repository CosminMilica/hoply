import { computed, reactive, ref, toValue, watch, type MaybeRefOrGetter } from 'vue'
import { serviceErrorMessage } from '@/features/common/errors'
import type { Job } from '@/features/jobs/api/jobs.schema'
import { keywordsIn, matchJob, requirementsOf } from '@/features/jobs/match'
import { withoutId, type Bullet, type Experience, type Note, type Profile, type Project } from '@/features/profile/api/profile.schema'
import { useProfileMutations } from '@/features/profile/composables/useProfile'
import { monthRange } from '@/features/profile/format'
import { bulletLine, knownSkills } from '@/features/profile/skills'
import type { CvRequest, PrepQuestions, ProbeQuestion } from '../api/cv.schema'

export const OTHER = 'other'
const MIN_ANSWER = 10

type List = 'experience' | 'projects'
/** Where an answer goes: `experience:<id>`, `projects:<id>`, or `other` (the person's own notes). */
type Place = string
/** What an answer added to the profile, so that Change takes exactly that back out. */
interface Remembered {
  place: Place
  text: string
  skills: string[]
}
type GapAnswer = { status: 'added', remembered: Remembered } | { status: 'skipped' }
type ClaimAnswer = { status: 'added', remembered: Remembered } | { status: 'picked', text: string } | { status: 'no' }

const joinList = (items: string[]) =>
  items.length <= 1 ? items.join('') : `${items.slice(0, -1).join(', ')} and ${items[items.length - 1]}`
const sameSkills = (a: string[], b: string[]) => a.length === b.length && a.every((skill, index) => skill === b[index])

interface PrepInputs {
  job: MaybeRefOrGetter<Job | undefined>
  profile: MaybeRefOrGetter<Profile | undefined>
  questions: MaybeRefOrGetter<PrepQuestions | undefined>
  /** Claude's questions for the job, once ready. */
  probe?: MaybeRefOrGetter<ProbeQuestion[] | undefined>
}

/**
 * Prepare your CV for one job. The facts the person confirms are saved in their profile, where every
 * later CV finds them: a highlight under the role or project they pick, or a line of additional
 * information. What stays with this CV is the roles to feature, the skills up front and the
 * responsibilities confirmed.
 */
export function usePrep({ job, profile, questions, probe }: PrepInputs) {
  const mutations = useProfileMutations()
  const state = reactive({
    skip: {} as Record<string, boolean>,
    focus: null as string[] | null,
    drafts: {} as Record<string, string>,
    places: {} as Record<string, Place>,
    gaps: {} as Record<string, GapAnswer>,
    claims: {} as Record<number, ClaimAnswer>,
    probes: {} as Record<number, GapAnswer>,
    saving: {} as Record<string, boolean>,
    errors: {} as Record<string, string | undefined>,
  })

  const current = () => toValue(profile)
  const experience = computed(() => current()?.experience ?? [])
  const projects = computed(() => current()?.projects ?? [])
  const keywords = computed(() => { const current = toValue(job); return current ? requirementsOf(current) : [] })
  const known = computed(() => (current() ? knownSkills(current()!) : []))
  const match = computed(() => matchJob(toValue(job) ?? { must_have: [], nice_to_have: [] }, known.value))
  const includedRoles = computed(() => experience.value.filter(role => !state.skip[role.id]))

  const draft = (key: string) => state.drafts[key] ?? ''
  const setDraft = (key: string, value: string) => {
    state.drafts[key] = value
  }
  const canAdd = (key: string) => draft(key).trim().length >= MIN_ANSWER && !state.saving[key]

  const choices = computed(() => [
    ...includedRoles.value.map(role => ({ id: `experience:${role.id}`, label: role.company || role.role, where: `${role.role}${role.company ? ` at ${role.company}` : ''}` })),
    ...projects.value.map(project => ({ id: `projects:${project.id}`, label: project.name, where: `the ${project.name} project` })),
  ])
  function whereOf(key: string): Place {
    const chosen = state.places[key]
    if (chosen === OTHER || choices.value.some(choice => choice.id === chosen)) return chosen!
    return choices.value[0]?.id ?? OTHER
  }
  function whereLabel(place: Place): string {
    const target = entryOf(place)
    if (!target) return 'Your notes'
    if (target.list === 'projects') return `the ${(target.entry as Project).name} project`
    const role = target.entry as Experience
    return role.company ? `${role.role} at ${role.company}` : role.role
  }
  function roleChoices(key: string) {
    const where = whereOf(key)
    return [...choices.value, { id: OTHER, label: 'Elsewhere' }].map(choice => ({ id: choice.id, label: choice.label, pressed: choice.id === where }))
  }
  function pickRole(key: string, id: Place) {
    state.places[key] = id
  }

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
  const putBullets = (list: List, entry: Experience | Project, bullets: Bullet[]) => putEntry(list, entry, { bullets })

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

  /** Runs one save for a question, keeping its error next to it. */
  async function saving(key: string, work: () => Promise<void>) {
    state.saving[key] = true
    state.errors[key] = undefined
    try {
      await work()
    }
    catch (error) {
      state.errors[key] = await serviceErrorMessage(error)
    }
    finally {
      state.saving[key] = false
    }
  }

  async function addGap(skill: string) {
    const key = `g:${skill}`
    if (!canAdd(key)) return
    await saving(key, async () => {
      const remembered = await remember(whereOf(key), draft(key).trim(), [skill])
      state.gaps[skill] = { status: 'added', remembered }
    })
  }
  function skipGap(skill: string) {
    state.gaps[skill] = { status: 'skipped' }
  }
  async function undoGap(skill: string) {
    const answer = state.gaps[skill]
    if (answer?.status !== 'added') {
      delete state.gaps[skill]
      return
    }
    await saving(`g:${skill}`, async () => {
      await forget(answer.remembered)
      delete state.gaps[skill]
    })
  }

  /** An answer to one of Claude's questions becomes a note of the role or project it is about. */
  async function answerProbe(index: number, question: ProbeQuestion) {
    const key = `p:${index}`
    if (!canAdd(key)) return
    const text = draft(key).trim()
    await saving(key, async () => {
      const remembered = await remember(question.place, text, keywordsIn(text, keywords.value))
      state.probes[index] = { status: 'added', remembered }
    })
  }
  function skipProbe(index: number) {
    state.probes[index] = { status: 'skipped' }
  }
  async function undoProbe(index: number) {
    const answer = state.probes[index]
    if (answer?.status !== 'added') {
      delete state.probes[index]
      return
    }
    await saving(`p:${index}`, async () => {
      await forget(answer.remembered)
      delete state.probes[index]
    })
  }

  async function claimYes(index: number) {
    const key = `c:${index}`
    if (!canAdd(key)) return
    const text = draft(key).trim()
    await saving(key, async () => {
      const remembered = await remember(whereOf(key), text, keywordsIn(text, keywords.value))
      state.claims[index] = { status: 'added', remembered }
    })
  }
  function claimNo(index: number) {
    state.claims[index] = { status: 'no' }
  }
  function claimPick(index: number, text: string) {
    state.claims[index] = { status: 'picked', text }
  }
  async function claimUndo(index: number) {
    const answer = state.claims[index]
    if (answer?.status !== 'added') {
      delete state.claims[index]
      return
    }
    await saving(`c:${index}`, async () => {
      await forget(answer.remembered)
      delete state.claims[index]
    })
  }

  /** The person's own highlights that share one of the job's keywords with a responsibility. */
  function suggestionsFor(responsibility: string) {
    const shared = keywordsIn(responsibility, keywords.value).map(keyword => keyword.toLowerCase())
    if (!shared.length) return []
    const entries: { label: string, bullets: Bullet[] }[] = [
      ...experience.value.map(role => ({ label: role.company || role.role, bullets: role.bullets })),
      ...projects.value.map(project => ({ label: project.name, bullets: project.bullets })),
    ]
    return entries
      .flatMap(entry => entry.bullets.map(bullet => ({ label: entry.label, bullet })))
      .filter(({ bullet }) => bullet.skills.some(skill => shared.includes(skill.toLowerCase()))
        || keywordsIn(bulletLine(bullet), shared).length > 0)
      .slice(0, 3)
      .map(({ label, bullet }) => ({ text: bulletLine(bullet), label }))
  }

  const baseScore = computed(() => toValue(questions)?.base_score ?? 0)
  const matchLabel = computed(() => (match.value.score !== baseScore.value ? `${baseScore.value}% → ${match.value.score}%` : `${match.value.score}%`))
  const progress = computed(() => {
    const gaps = toValue(questions)?.gaps ?? []
    const asked = toValue(probe) ?? []
    // Claude's questions cover the job's responsibilities; the rules' questions about them are asked only without Claude.
    const claims = asked.length ? [] : (toValue(questions)?.responsibilities ?? [])
    const answered = gaps.filter(gap => state.gaps[gap.skill]).length + claims.filter(claim => state.claims[claim.index]).length
      + asked.filter((_, index) => state.probes[index]).length
    return `${answered} of ${gaps.length + claims.length + asked.length} questions answered`
  })

  // The highlights to strengthen are chosen once, so that a saved result does not move them.
  const impactRows = ref<{ key: string, roleId: string, index: number, bullet: string }[] | null>(null)
  const impacts = reactive<Record<string, string>>({})
  watch(() => includedRoles.value[0], (role) => {
    if (impactRows.value || !current()) return
    impactRows.value = (role?.bullets ?? [])
      .map((bullet, index) => ({ key: `${role!.id}-${index}`, roleId: role!.id, index, bullet: bullet.text, result: bullet.result }))
      .filter(row => !/\d/.test(row.bullet) && !row.result.trim())
      .slice(0, 3)
      .map(({ result: _result, ...row }) => row)
  }, { immediate: true })
  async function saveImpact(key: string, value: string) {
    const row = impactRows.value?.find(item => item.key === key)
    const target = row && entryOf(`experience:${row.roleId}`)
    const bullet = target?.entry.bullets[row!.index]
    if (!target || !bullet || bullet.text !== row!.bullet || bullet.result === value.trim()) return
    await saving(`i:${key}`, () => putBullets('experience', target.entry, target.entry.bullets.map((item, index) =>
      (index === row!.index ? { ...item, result: value.trim() } : item))))
  }

  const note = ref('')
  watch(() => current()?.additional_info.join('\n') ?? '', (value) => { note.value = value }, { immediate: true })
  async function saveNote() {
    const lines = note.value.split('\n').map(line => line.trim()).filter(Boolean).slice(0, 10)
    if (lines.join('\n') === (current()?.additional_info ?? []).join('\n')) return
    await saving('note', async () => {
      await mutations.patch.mutateAsync({ additional_info: lines })
    })
  }

  const roles = computed(() => experience.value.map((role) => {
    const tools = role.tools.map(tool => tool.toLowerCase())
    const overlap = match.value.matched.filter(keyword => tools.includes(keyword.toLowerCase()))
    return {
      id: role.id,
      role: role.role,
      meta: `${role.company} · ${monthRange(role.start, role.end) || 'dates not set'}`,
      overlap: overlap.length ? `Uses ${joinList(overlap)}` : 'No tools in common with this job',
      included: !state.skip[role.id],
    }
  }))
  function toggleRole(id: string) {
    state.skip[id] = !state.skip[id]
  }

  const focusSelection = computed(() => (state.focus?.length ? state.focus : match.value.matched.slice(0, 3)))
  const focusChips = computed(() => match.value.matched.map(skill => ({ label: skill, pressed: focusSelection.value.includes(skill) })))
  function toggleFocus(skill: string) {
    const selected = focusSelection.value
    if (selected.includes(skill)) state.focus = selected.filter(item => item !== skill)
    else if (selected.length < 3) state.focus = [...selected, skill]
  }

  function toRequest(jobId: string): CvRequest {
    return {
      job_id: jobId,
      skip_roles: Object.keys(state.skip).filter(id => state.skip[id]),
      focus: state.focus ?? [],
      confirmed: Object.entries(state.claims)
        .filter(([, answer]) => answer.status !== 'no')
        .map(([index]) => Number(index)),
    }
  }

  return {
    state,
    draft,
    setDraft,
    canAdd,
    whereLabel,
    roleChoices,
    pickRole,
    score: computed(() => match.value.score),
    matched: computed(() => match.value.matched),
    known,
    matchLabel,
    addGap,
    skipGap,
    undoGap,
    answerProbe,
    skipProbe,
    undoProbe,
    claimYes,
    claimNo,
    claimPick,
    claimUndo,
    suggestionsFor,
    progress,
    impactRows: computed(() => impactRows.value ?? []),
    impacts,
    saveImpact,
    note,
    saveNote,
    roles,
    toggleRole,
    focusChips,
    toggleFocus,
    toRequest,
  }
}
