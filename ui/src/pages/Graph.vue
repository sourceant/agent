<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { Network, X } from 'lucide-vue-next'
import UiCard from '~/components/ui/Card.vue'
import UiButton from '~/components/ui/Button.vue'
import PageHead from '~/components/PageHead.vue'
import CodeGraph from '~/components/CodeGraph.vue'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'
import { COLOURS, MODES, OTHER, groupOf, withFolders } from '~/lib/graph'

const { repositories, chosen, error, fetchRepositories } = useRepositories()
const loaded = ref(null)
const loading = ref(false)
const mode = ref('2d')
const selected = ref(null)

const folders = ref(true)
const symbols = ref(false)
const imports = ref(false)
const tests = ref(false)

/* A repository's every function is a texture rather than a picture, so what
 * opens is its shape. Symbols and imports are there to be asked for. */
const shown = computed(() => {
  if (!loaded.value) return null
  const nodes = loaded.value.nodes.filter((node) => {
    const group = groupOf(node)
    if (group === 'import') return imports.value
    if (group === 'file') return true
    return symbols.value
  })
  const kept = new Set(nodes.map((node) => node.id))
  const links = loaded.value.links.filter((link) => kept.has(link.source) && kept.has(link.target))
  const data = { nodes: nodes.map((node) => ({ ...node })), links: links.map((link) => ({ ...link })) }
  return folders.value ? withFolders(data, chosen.value) : data
})

const legend = computed(() => {
  if (!shown.value) return []
  return [...new Set(shown.value.nodes.map(groupOf))].sort()
    .map((group) => ({ group, colour: COLOURS[group] || OTHER }))
})

const degree = computed(() => {
  if (!selected.value || !loaded.value) return 0
  return loaded.value.links.filter(
    (link) => link.source === selected.value.id || link.target === selected.value.id).length
})

async function load() {
  if (!chosen.value) return
  loading.value = true
  selected.value = null
  try {
    loaded.value = await api.graph(chosen.value, { includeTests: tests.value })
    error.value = ''
  } catch (problem) {
    loaded.value = null
    error.value = problem.message
  } finally {
    loading.value = false
  }
}

watch([chosen, tests], load)

onMounted(async () => {
  await fetchRepositories()
  await load()
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <PageHead :icon="Network" title="Knowledge graph" sub="Your code, and how it holds together.">
      <template #actions>
        <select
          v-if="repositories.length > 1"
          v-model="chosen"
          class="rounded-md border bg-card px-3 py-1.5 text-sm"
          aria-label="Repository"
        >
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </select>
      </template>
    </PageHead>

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <EmptyMachine v-if="repositories.length === 0" />

    <template v-else>
      <div class="mb-3 flex flex-wrap items-center justify-between gap-3">
        <div class="inline-flex rounded-md border bg-card p-0.5">
          <button
            v-for="option in MODES"
            :key="option.id"
            type="button"
            class="px-3 py-1 text-xs font-medium rounded transition-colors"
            :class="mode === option.id ? 'bg-primary/15 text-primary' : 'text-muted-foreground hover:text-foreground'"
            @click="mode = option.id"
          >
            {{ option.label }}
          </button>
        </div>
        <div class="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
          <label class="inline-flex items-center gap-1.5 cursor-pointer">
            <input v-model="folders" type="checkbox"> Folders
          </label>
          <label class="inline-flex items-center gap-1.5 cursor-pointer">
            <input v-model="symbols" type="checkbox"> Symbols
          </label>
          <label class="inline-flex items-center gap-1.5 cursor-pointer">
            <input v-model="imports" type="checkbox"> Imports
          </label>
          <label class="inline-flex items-center gap-1.5 cursor-pointer">
            <input v-model="tests" type="checkbox"> Tests
          </label>
          <span>{{ mode === '2d' ? 'Scroll to zoom, drag to pan' : 'Drag to rotate' }}</span>
        </div>
      </div>

      <div class="relative min-h-0 flex-1 overflow-hidden rounded-lg border bg-card">
        <CodeGraph :data="shown" :mode="mode" :repository="chosen" @select="selected = $event" />

        <div
          v-if="loading || !shown || shown.nodes.length === 0"
          class="absolute inset-0 grid place-items-center bg-card p-8 text-center text-sm text-muted-foreground"
        >
          <span v-if="loading">Reading the index…</span>
          <span v-else>Nothing here yet. Re-index it from Repositories.</span>
        </div>

        <aside
          v-if="selected"
          class="absolute right-3 top-3 w-80 max-w-[calc(100%-1.5rem)] rounded-md border bg-background/90 p-4 backdrop-blur-xl"
        >
          <button
            class="absolute right-2 top-2 rounded-md p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
            aria-label="Close"
            @click="selected = null"
          >
            <X class="h-3.5 w-3.5" />
          </button>
          <h2 class="mb-2 mr-6 text-sm font-semibold break-all">{{ selected.name }}</h2>
          <dl class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
            <dt class="text-muted-foreground">Kind</dt>
            <dd class="font-mono break-all">
              {{ selected.synthetic ? `${selected.kind} · this view's arrangement` : selected.kind }}
            </dd>
            <dt class="text-muted-foreground">Path</dt>
            <dd class="font-mono break-all">{{ selected.path || '—' }}</dd>
            <dt class="text-muted-foreground">Links</dt>
            <dd class="font-mono">{{ selected.synthetic ? '—' : degree }}</dd>
          </dl>
        </aside>
      </div>

      <p v-if="loaded?.truncated" class="mt-3 rounded-md border border-warning/35 bg-warning/10 px-4 py-2.5 text-sm">
        This repository is larger than the limit, so this is part of it, not all of it.
      </p>

      <div class="mt-3 flex flex-wrap items-center justify-between gap-4">
        <div class="flex flex-wrap gap-x-4 gap-y-2">
          <div v-for="item in legend" :key="item.group" class="flex items-center gap-2 text-xs text-muted-foreground">
            <span class="h-2.5 w-2.5 rounded-sm" :style="{ backgroundColor: item.colour }" />
            {{ item.group || 'other' }}
          </div>
        </div>
        <p v-if="shown" class="text-xs text-muted-foreground tabular-nums">
          {{ shown.nodes.length.toLocaleString() }} nodes · {{ shown.links.length.toLocaleString() }} links
        </p>
      </div>
    </template>
  </div>
</template>
