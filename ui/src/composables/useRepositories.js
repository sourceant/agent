import { computed, ref } from 'vue'
import { api } from '~/api'

/* One list of repositories for the whole app, and one choice of which is being
 * looked at, so moving between pages does not lose it. */
const repositories = ref([])
const chosen = ref('')
const error = ref('')
const loading = ref(false)

// Every repository at once. Empty, because that is what the API already means
// by "not narrowed to one", so nothing below has to translate it.
export const EVERY = ''

export function useRepositories({ all = false } = {}) {
  // Whether the view is showing more than one repository's worth, which is
  // what decides if a card has to say which one it came from.
  const mixed = computed(() => all && !chosen.value && repositories.value.length > 1)

  async function fetchRepositories() {
    loading.value = true
    try {
      repositories.value = await api.repositories()
      error.value = ''
    } catch (problem) {
      repositories.value = []
      error.value = `${problem.message}. Is sourceant-agent running?`
    } finally {
      loading.value = false
    }
    if (chosen.value === EVERY && all) return
    if (!repositories.value.some((item) => item.name === chosen.value)) {
      chosen.value = repositories.value[0]?.name ?? ''
    }
  }

  return { repositories, chosen, error, loading, mixed, fetchRepositories }
}
