import { ref } from 'vue'
import { api } from '~/api'

/** The code itself: what is defined, and what calls what. */
export function useCodeGraph() {
  const graph = ref(null)
  const loading = ref(false)
  const problem = ref(null)

  async function fetchCodeGraph(owner, repo, ask = {}) {
    loading.value = true
    try {
      graph.value = await api.graph(`${owner}/${repo}`, ask)
      problem.value = null
    } catch (caught) {
      graph.value = null
      problem.value = caught
    } finally {
      loading.value = false
    }
  }

  return {
    graph,
    loading,
    fetchCodeGraph,
    failureFor: () => problem.value,
  }
}

/** Knowledge as a graph, which a machine does not answer yet.
 *
 * Locally knowledge is a list: nothing serves the relationships between one
 * record and another, so there is no graph to draw. Saying so is better than
 * drawing an empty one and letting a reader conclude they have recorded
 * nothing.
 */
export function useRepos() {
  const graph = ref(null)

  async function fetchGraph() {
    graph.value = { nodes: [], links: [] }
    return graph.value
  }

  return {
    fetchGraph,
    failureFor: () => null,
  }
}
