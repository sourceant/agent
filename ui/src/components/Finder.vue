<script setup>
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { CommandSearch } from '@sourceant/design'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* Finding a thing rather than a screen.
 *
 * The navigation answers "where is Skills". Nobody wants Skills; they want the
 * one about migrations, the decision about retries, the file that holds the
 * charge. Those live behind a screen and a repository, which is two guesses
 * before anything is found.
 *
 * What a machine holds is small enough to ask for once and narrow here. The
 * code is not: a graph is thousands of nodes, so the core narrows that one.
 */

const props = defineProps({
  pages: { type: Array, required: true },
})

const MOST = 5
const SHORTEST = 2

const router = useRouter()
const { repositories, chosen, fetchRepositories } = useRepositories()

const open = ref(false)
const term = ref('')
const loading = ref(false)
const skills = ref([])
const knowledge = ref([])
const code = ref([])
let held = ''
let asking = null

const wanted = computed(() => term.value.trim().toLowerCase())
const has = (text) => String(text ?? '').toLowerCase().includes(wanted.value)

const groups = computed(() => {
  if (wanted.value.length < SHORTEST) return []
  const found = [
    {
      id: 'skills',
      label: 'Skills',
      items: skills.value
        .filter((one) => has(one.name) || has(one.description))
        .slice(0, MOST)
        .map((one) => ({ id: one.id, label: one.name, detail: one.description })),
    },
    {
      id: 'knowledge',
      label: 'Knowledge',
      items: knowledge.value
        .filter((one) => has(one.id) || has(one.summary))
        .slice(0, MOST)
        .map((one) => ({ id: one.id, label: one.id, detail: one.summary })),
    },
    {
      id: 'code',
      label: 'Code',
      items: code.value
        .slice(0, MOST)
        .map((one) => ({ id: one.id, label: one.name || one.id, detail: one.path })),
    },
    {
      id: 'repositories',
      label: 'Repositories',
      items: repositories.value
        .filter((one) => has(one.name) || has(one.path))
        .slice(0, MOST)
        .map((one) => ({ id: one.name, label: one.name, detail: one.path })),
    },
    {
      id: 'pages',
      label: 'Go to',
      items: props.pages
        .filter((one) => has(one.name))
        .map((one) => ({ id: one.href, label: one.name })),
    },
  ]
  return found.filter((group) => group.items.length)
})

// Everything this machine holds of the smaller kinds, once per repository.
async function hold() {
  if (held === chosen.value) return
  held = chosen.value
  const [page, recorded] = await Promise.all([
    api.skills(chosen.value).catch(() => ({ skills: [] })),
    chosen.value ? api.knowledge(chosen.value).catch(() => ({ items: [] })) : { items: [] },
  ])
  skills.value = page.skills ?? []
  knowledge.value = recorded.items ?? []
}

// The graph is asked rather than filtered here: it is the one thing on a
// machine too large to hold in a screen.
async function search() {
  clearTimeout(asking)
  if (wanted.value.length < SHORTEST || !chosen.value) {
    code.value = []
    return
  }
  asking = setTimeout(async () => {
    loading.value = true
    try {
      const graph = await api.graph(chosen.value, { q: term.value.trim(), nodeLimit: 60 })
      code.value = graph.nodes ?? []
    } catch {
      code.value = []
    }
    loading.value = false
  }, 250)
}

watch(open, async (showing) => {
  if (!showing) return
  if (!repositories.value.length) await fetchRepositories()
  await hold()
})
watch(term, search)

function go(item, group) {
  const found = term.value.trim()
  term.value = ''
  if (group === 'pages') router.push(item.id)
  else if (group === 'repositories') router.push(`/repositories/${item.id}`)
  else if (group === 'skills') router.push(`/skills/${item.id}`)
  else if (group === 'knowledge') router.push({ path: '/knowledge', query: { find: found } })
  else if (group === 'code') {
    router.push({ path: '/graph', query: { repository: chosen.value, q: found } })
  }
}
</script>

<template>
  <CommandSearch
    v-model="term"
    v-model:open="open"
    :groups="groups"
    :loading="loading"
    :context="chosen ? `This machine · ${chosen}` : 'This machine'"
    placeholder="Find a skill, a decision, a file…"
    label="Find anything on this machine"
    :empty-label="term.trim().length < 2 ? 'Type at least two letters.' : 'Nothing here matches that.'"
    @select="go"
  />
</template>
