import { reactive, toValue, type MaybeRefOrGetter } from 'vue'
import { serviceErrorMessage } from '@/features/common/errors'
import type { Profile } from '@/features/profile/api/profile.schema'
import type { ProbeQuestion } from '../api/cv.schema'
import type { ProbeAnswer } from '../components/ProbeQuestionCard.vue'
import { MIN_ANSWER } from './usePrep'
import { useNotes, type Remembered } from './useNotes'

type Kept = { status: 'added', remembered: Remembered } | { status: 'skipped' }

/** Answers to Claude's questions, each kept as a note of the role or project it is about. */
export function useProbeAnswers(profile: MaybeRefOrGetter<Profile | undefined>) {
  const { whereLabel, remember, forget } = useNotes(() => toValue(profile))
  const state = reactive({
    drafts: {} as Record<number, string>,
    answers: {} as Record<number, Kept>,
    saving: {} as Record<number, boolean>,
    errors: {} as Record<number, string | undefined>,
  })
  const draft = (index: number) => state.drafts[index] ?? ''
  const canAdd = (index: number) => draft(index).trim().length >= MIN_ANSWER && !state.saving[index]

  async function saving(index: number, work: () => Promise<void>) {
    state.saving[index] = true
    state.errors[index] = undefined
    try {
      await work()
    }
    catch (error) {
      state.errors[index] = await serviceErrorMessage(error)
    }
    finally {
      state.saving[index] = false
    }
  }

  return {
    state,
    draft,
    canAdd,
    setDraft: (index: number, value: string) => { state.drafts[index] = value },
    answerOf(index: number): ProbeAnswer {
      const kept = state.answers[index]
      if (!kept) return undefined
      return kept.status === 'added' ? { status: 'added', text: kept.remembered.text, where: whereLabel(kept.remembered.place) } : kept
    },
    async answer(index: number, question: ProbeQuestion) {
      if (!canAdd(index)) return
      const text = draft(index).trim()
      await saving(index, async () => {
        state.answers[index] = { status: 'added', remembered: await remember(question.place, text, []) }
      })
    },
    skip(index: number) {
      state.answers[index] = { status: 'skipped' }
    },
    async undo(index: number) {
      const kept = state.answers[index]
      if (kept?.status !== 'added') {
        delete state.answers[index]
        return
      }
      await saving(index, async () => {
        await forget(kept.remembered)
        delete state.answers[index]
      })
    },
  }
}
