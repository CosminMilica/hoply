import { ref } from 'vue'
import { serviceErrorMessage } from '@/features/common/errors'

interface Mutation<V> {
  mutate: (variables: V, options?: { onSuccess?: (entry: { id: string }) => void, onError?: (error: Error) => void }) => void
}

/** Which entry of a list is being added or edited, and what the service said about the last save. */
export function useEntryEditor<Input>(mutations: { add: Mutation<Input>, update: Mutation<{ id: string, body: Input }> }) {
  const adding = ref(false)
  const editing = ref<string | null>(null)
  const serverError = ref<string>()

  const showError = async (error: unknown) => {
    serverError.value = await serviceErrorMessage(error)
  }
  function open(id: string | null) {
    serverError.value = undefined
    editing.value = id
    adding.value = id === null
  }
  function close() {
    adding.value = false
    editing.value = null
    serverError.value = undefined
  }
  function add(body: Input, onAdded?: (entry: { id: string }) => void) {
    serverError.value = undefined
    mutations.add.mutate(body, {
      onSuccess: (entry: { id: string }) => {
        onAdded?.(entry)
        close()
      },
      onError: showError,
    })
  }
  function update(id: string, body: Input) {
    serverError.value = undefined
    mutations.update.mutate({ id, body }, { onSuccess: close, onError: showError })
  }

  return { adding, editing, serverError, open, close, add, update }
}
