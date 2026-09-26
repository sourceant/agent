<script setup lang="ts">
import { Button, CodeGraph as KnowledgeGraph, Loading, Tabs, useContextKinds } from '@sourceant/design'
import { computed, ref, watch } from 'vue'
import { Crosshair, Network, Search, X } from 'lucide-vue-next'
import type { KnowledgeGraphData } from '~/types'
import { useCodeGraph, useRepos } from '~/composables/useCodeGraph'

/**
 * The graph, its controls and whatever reads what was picked on it.
 *
 * Written once and asked for differently in each place it appears. Reading a
 * record beside the knowledge it belongs to and working the whole index are
 * different jobs, and they were two implementations of the same drawing until
 * this existed. Anything a placement does not ask for is absent rather than
 * disabled, so the smaller use stays small.
 */
type Source = 'knowledge' | 'code'
type Control = 'filter' | 'kinds' | 'parts' | 'layouts' | 'depth' | 'retired'
type Mode = '2d' | '3d' | 'web' | 'tree' | 'radial' | 'layered'

const props = withDefaults(defineProps<{
  /** owner/name. Chosen by whoever mounts this, not here. */
  repository: string
  /** Offered sources. More than one puts a toggle at the top of the rail. */
  sources?: Source[]
  controls?: Control[]
  height?: string
  /** What to narrow to when it opens, for arriving from a search. */
  find?: string
}>(), {
  sources: () => ['knowledge', 'code'],
  controls: () => ['filter', 'kinds', 'parts', 'layouts', 'depth', 'retired'],
  height: '640px',
})

const emit = defineEmits<{ select: [string] }>()

const { fetchGraph, failureFor } = useRepos()
const { legend } = useContextKinds()
const {
  graph: codeGraph,
  loading: loadingCode,
  fetchCodeGraph,
  failureFor: codeFailure,
} = useCodeGraph()

const source = ref<Source>(props.sources[0] ?? 'knowledge')
const graphData = ref<KnowledgeGraphData | null>(null)
const loadingGraph = ref(false)

const drawn = computed(() => (source.value === 'code' ? codeGraph.value : graphData.value))
const busy = computed(() => (source.value === 'code' ? loadingCode.value : loadingGraph.value))
const problem = computed(() =>
  source.value === 'code' ? codeFailure('code-graph') : failureFor('graph'))

const repoLabel = computed(() => props.repository.split('/').pop() ?? props.repository)
const hasRealGraph = computed(() => !!drawn.value && drawn.value.nodes.length > 0)

function offers(control: Control) {
  return props.controls.includes(control)
}

// Searching for somewhere to start and narrowing what is drawn are different
// questions, so they are different controls. Depth only means anything once
// there is a place to walk out from, and says so rather than sitting there.
const focus = ref('')
const depth = ref(2)
const kinds = ref<string[]>([])
const term = ref(props.find ?? '')
const mode = ref<Mode>('2d')
const hiddenParts = ref<number[]>([])
const retired = ref(false)

const focused = computed(() => !!focus.value)
const focusLabel = computed(() => {
  const node = drawn.value?.nodes.find(n => n.id === focus.value)
  return node?.name ?? focus.value.replace(/^(file|symbol):/, '')
})

const parts = computed(() => drawn.value?.communities ?? [])
const partsShown = ref(50)
const visibleParts = computed(() => parts.value.slice(0, partsShown.value))
watch(parts, () => { partsShown.value = 50 })

const modes: { id: Mode, label: string }[] = [
  { id: '2d', label: '2D' },
  { id: '3d', label: '3D' },
  { id: 'tree', label: 'Tree' },
  { id: 'radial', label: 'Radial' },
  { id: 'layered', label: 'Layered' },
  { id: 'web', label: 'Force' },
]

const SOURCE_LABELS: Record<Source, string> = { knowledge: 'Knowledge', code: 'Code' }

/** The same ten the canvas uses, so the legend and the drawing agree. */
const PART_COLORS = [
  '#4E79A7', '#F28E2B', '#E15759', '#76B7B2', '#59A14F',
  '#EDC948', '#B07AA1', '#FF9DA7', '#9C755F', '#BAB0AC',
]

async function loadGraph() {
  const [owner, name] = props.repository.split('/')
  if (!owner || !name) {
    graphData.value = null
    return
  }

  if (source.value === 'code') {
    await fetchCodeGraph(owner, name, {
      focus: focus.value || undefined,
      depth: focused.value ? depth.value : undefined,
      q: term.value.trim() || undefined,
    })
    return
  }

  loadingGraph.value = true
  try {
    graphData.value = await fetchGraph(owner, name, {
      focus: focus.value || undefined,
      depth: focused.value ? depth.value : undefined,
      kind: kinds.value,
      q: term.value.trim() || undefined,
      // Everything, rather than only what is still standing. A record somebody
      // retired is the answer to why something is not the way it used to be.
      status: retired.value ? 'all' : undefined,
    })
  }
  finally {
    loadingGraph.value = false
  }
}

let asking: ReturnType<typeof setTimeout> | null = null
function askAgain() {
  if (asking) clearTimeout(asking)
  asking = setTimeout(loadGraph, 250)
}

watch([depth, kinds, term, retired], askAgain, { deep: true })
watch(focus, loadGraph)
watch(source, () => {
  focus.value = ''
  hiddenParts.value = []
  loadGraph()
})
watch(() => props.repository, () => {
  focus.value = ''
  hiddenParts.value = []
  loadGraph()
}, { immediate: true })

/**
 * Picking is two answers to one gesture: the drawing re-cuts around what was
 * picked, and whoever mounted this says what it was.
 */
function pick(id: string) {
  emit('select', id)
  if (offers('depth')) focus.value = focus.value === id ? '' : id
}

function drawEverything() {
  focus.value = ''
}

function toggleKind(kind: string) {
  kinds.value = kinds.value.includes(kind)
    ? kinds.value.filter(k => k !== kind)
    : [...kinds.value, kind]
}

function togglePart(id: number) {
  hiddenParts.value = hiddenParts.value.includes(id)
    ? hiddenParts.value.filter(p => p !== id)
    : [...hiddenParts.value, id]
}

const railed = computed(() =>
  props.sources.length > 1
  || offers('filter')
  || offers('kinds')
  || offers('parts')
  || offers('layouts')
  || offers('depth')
  || offers('retired'),
)

defineExpose({ reload: loadGraph })
</script>

<template>
  <!-- Controls beside the graph, not stacked above it: the drawing is the
       point, and pushing it down the page to make room for its own settings
       gets that backwards. -->
  <div
    class="grid gap-4 lg:items-start"
    :class="railed ? 'lg:grid-cols-[16rem_1fr]' : ''"
  >
    <aside
      v-if="railed"
      class="space-y-4 overflow-y-auto rounded-lg border bg-card p-3"
      :style="{ height: height }"
    >
      <div v-if="sources.length > 1">
        <p class="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Draw</p>
        <div class="inline-flex w-full rounded-md border bg-muted/40 p-0.5">
          <button
            v-for="s in sources"
            :key="s"
            type="button"
            class="flex-1 rounded px-2 py-1 text-xs font-medium transition-colors"
            :class="source === s ? 'bg-card text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'"
            @click="source = s"
          >{{ SOURCE_LABELS[s] }}</button>
        </div>
        <p class="mt-1.5 text-[11px] text-muted-foreground">
          {{ source === 'code'
            ? 'What is defined and what calls what, read from the code.'
            : 'What your team decided, and the files it covers.' }}
        </p>
      </div>

      <div v-if="offers('filter')">
        <label class="mb-1.5 block text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          Only what mentions
        </label>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <input
            v-model="term"
            type="search"
            placeholder="a word"
            class="w-full rounded-md border bg-background py-1.5 pl-8 pr-2 text-sm"
          >
        </div>
      </div>

      <div v-if="offers('parts') && source === 'code' && parts.length">
        <p class="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Parts</p>
        <ul class="space-y-0.5">
          <li v-for="p in visibleParts" :key="p.id">
            <button
              type="button"
              class="flex w-full items-center gap-2 rounded-md px-1 py-1 text-left text-[11px] transition-colors hover:bg-muted"
              :class="hiddenParts.includes(p.id) ? 'opacity-40' : ''"
              @click="togglePart(p.id)"
            >
              <span class="h-2.5 w-2.5 shrink-0 rounded-full" :style="{ backgroundColor: PART_COLORS[p.id % PART_COLORS.length] }" />
              <span class="flex-1 truncate text-foreground">{{ p.name }}</span>
              <span class="tabular-nums text-muted-foreground">{{ p.size }}</span>
            </button>
          </li>
        </ul>
        <Button
          v-if="partsShown < parts.length"
          variant="ghost"
          size="sm"
          class="mt-2 w-full"
          @click="partsShown += 50"
        >Show more parts ({{ partsShown }} of {{ parts.length }})</Button>
        <p class="mt-1.5 text-[11px] text-muted-foreground">
          Grouped by what calls what. Click one to hide it.
        </p>
      </div>

      <div v-if="offers('kinds') && source === 'knowledge'">
        <p class="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Kinds</p>
        <div class="flex flex-wrap gap-1.5">
          <button
            v-for="l in legend"
            :key="l.label"
            type="button"
            class="inline-flex items-center gap-1.5 rounded-full border px-2 py-0.5 text-[11px] transition-colors"
            :class="kinds.includes(l.label.toLowerCase())
              ? 'border-primary/40 bg-primary/10 text-primary'
              : 'text-muted-foreground hover:text-foreground'"
            @click="toggleKind(l.label.toLowerCase())"
          >
            <span class="h-2 w-2 rounded-sm" :style="{ backgroundColor: l.color }" />
            {{ l.label }}
          </button>
          <button
            v-if="kinds.length"
            type="button"
            class="text-[11px] text-muted-foreground underline-offset-2 hover:underline"
            @click="kinds = []"
          >Any</button>
        </div>
      </div>

      <div v-if="offers('retired') && source === 'knowledge'">
        <label class="flex cursor-pointer items-center gap-2 text-[11px] text-muted-foreground">
          <input v-model="retired" type="checkbox" class="h-3 w-3 rounded border">
          Include retired records
        </label>
      </div>

      <div v-if="offers('layouts')">
        <p class="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">Shape</p>
        <Tabs v-model="mode" :tabs="modes" label="Shape" class="grid w-full grid-cols-2" />
      </div>

      <div v-if="offers('depth')">
        <div class="mb-1.5 flex items-center justify-between">
          <span class="text-[11px] font-medium uppercase tracking-wide" :class="focused ? 'text-muted-foreground' : 'text-muted-foreground/50'">
            Steps out
          </span>
          <span class="font-mono text-xs" :class="focused ? '' : 'opacity-50'">{{ depth }}</span>
        </div>
        <input v-model.number="depth" type="range" min="1" max="5" :disabled="!focused" class="w-full">
        <p v-if="!focused" class="mt-1 text-[11px] text-muted-foreground">
          Click anything on the graph to walk out from it.
        </p>
        <div v-else class="mt-1.5 space-y-1.5">
          <p class="flex items-start gap-1.5 text-[11px]">
            <Crosshair class="mt-0.5 h-3 w-3 shrink-0 text-primary" />
            <span class="min-w-0 break-words">From <span class="font-medium">{{ focusLabel }}</span></span>
          </p>
          <button
            type="button"
            class="inline-flex w-full items-center justify-center gap-1 rounded-md border px-2 py-1 text-[11px] hover:bg-muted"
            @click="drawEverything"
          >
            <X class="h-3 w-3" /> Draw everything
          </button>
        </div>
      </div>
    </aside>

    <div class="min-w-0">
      <div class="grid gap-4" :class="$slots.inspector ? 'lg:grid-cols-[1fr_20rem]' : ''">
        <div class="min-w-0">
          <div class="overflow-hidden rounded-lg border bg-card">
            <div v-if="busy" class="flex" :style="{ height }">
              <Loading
                :label="source === 'code' ? 'Reading the code' : 'Reading the knowledge graph'"
                note="Large repositories can take longer to load."
                size="sm"
              />
            </div>

            <div v-else-if="problem" role="alert" class="flex flex-col items-center justify-center gap-3 px-6 text-center" :style="{ height }">
              <p class="text-sm font-medium">Could not load this graph</p>
              <p class="text-xs text-muted-foreground">{{ problem.message }}</p>
              <Button variant="outline" size="sm" @click="loadGraph">Try again</Button>
            </div>

            <div v-else-if="!hasRealGraph" class="flex flex-col items-center justify-center gap-3 px-6 text-center" :style="{ height }">
              <Network class="h-7 w-7 text-muted-foreground" />
              <div class="space-y-1">
                <p class="text-sm font-medium">Nothing to draw</p>
                <p class="mx-auto max-w-sm text-xs text-muted-foreground">
                  {{ source === 'code'
                    ? 'This repository has not been read yet, so there is no code map for it.'
                    : 'Initialize this repository and approve what it proposes, and its decisions appear here.' }}
                </p>
              </div>
            </div>

            <template v-else>
              <KnowledgeGraph
                :repo="repoLabel"
                :mode="mode"
                :data="drawn"
                :hidden="hiddenParts"
                :height="height"
                @select="pick"
              />
            </template>
          </div>

          <p v-if="drawn?.truncated" class="mt-2 text-xs text-warning">
            More than fits in one drawing, so this is the most connected part of it.
            Narrow it{{ railed ? ' on the left' : '' }}, or click something to walk out from it.
          </p>
          <p v-if="hasRealGraph" class="mt-2 text-xs text-muted-foreground">
            {{ drawn?.nodes.length }} symbols, {{ drawn?.links.length }} connections.
          </p>
          <p v-if="hasRealGraph && !drawn?.links.length" class="mt-2 text-xs text-muted-foreground">
            No connections were found between these files.
          </p>
        </div>

        <div v-if="$slots.inspector" class="overflow-y-auto rounded-lg border bg-card" :style="{ height }">
          <slot name="inspector" />
        </div>
      </div>
    </div>
  </div>
</template>
