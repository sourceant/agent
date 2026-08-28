import { ref } from 'vue'
import { api } from '~/api'

/* One list of repositories for the whole app, and one choice of which is being
 * looked at, so moving between pages does not lose it. */
const repositories = ref([])
const chosen = ref('')
const error = ref('')
const loading = ref(false)

export function useRepositories() {
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
    if (!repositories.value.some((item) => item.name === chosen.value)) {
      chosen.value = repositories.value[0]?.name ?? ''
    }
  }

  return { repositories, chosen, error, loading, fetchRepositories }
}
