<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  ItemCard,
  Notice,
  PageHead,
  Tabs,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  BookOpen,
  FileCode,
  Folder,
  Link2,
  Loader2,
  Network,
  RefreshCw,
  ScrollText,
  ShieldCheck,
  Trash2,
} from 'lucide-vue-next'
import GraphWorkbench from '~/components/GraphWorkbench.vue'
import { useUp } from '~/composables/useUp'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* One folder, and everything this machine holds about it.
 *
 * The list answers what is covered; this answers what came of covering it. The
 * graph is drawn here rather than linked to, because a graph of one repository
 * is what somebody arriving from that repository's row wants to see.
 */

const route = useRoute()
const router = useRouter()
const up = useUp()
const { repositories, chosen, fetchRepositories } = useRepositories()

const name = computed(() => String(route.params.name ?? ''))
const repository = computed(() => repositories.value.find((item) => item.name === name.value))

const tab = ref('overview')
const counts = ref(null)
const knowledge = ref([])
const skills = ref([])
const attention = ref({ files: [], since: '' })
const working = ref(false)
const lastRead = ref(null)
const error = ref('')

const figures = computed(() => [
  { label: 'Files', value: counts.value?.files ?? 0, icon: FileCode },
  { label: 'Connections', value: counts.value?.links ?? 0, icon: Link2 },
  { label: 'Parts', value: counts.value?.parts ?? 0, icon: Network },
  { label: 'Recorded', value: knowledge.value.length, icon: BookOpen },
  { label: 'Rules', value: skills.value.length, icon: ScrollText },
])

// The busiest file sets the scale the rest of the bars are drawn against.
const busiest = computed(() => attention.value.files[0]?.changes || 1)

const tabs = computed(() => [
  { id: 'overview', label: 'Overview' },
  { id: 'knowledge', label: `Knowledge${knowledge.value.length ? ` ${knowledge.value.length}` : ''}` },
  { id: 'skills', label: `Skills${skills.value.length ? ` ${skills.value.length}` : ''}` },
  { id: 'graph', label: 'Graph' },
])

async function load() {
  if (!name.value) return
  // Choosing it here means the pages this links out to open on the same one.
  chosen.value = name.value
  const [graph, recorded, rules, worth] = await Promise.all([
    api.graph(name.value).catch(() => null),
    api.knowledge(name.value).catch(() => ({ items: [] })),
    api.skills(name.value).catch(() => ({ skills: [] })),
    api.attention(name.value).catch(() => ({ files: [], since: '' })),
  ])
  counts.value = graph
    ? {
        files: graph.nodes.filter((node) => node.kind === 'file').length,
        links: graph.links.length,
        parts: (graph.communities ?? []).length,
      }
    : null
  knowledge.value = recorded.items ?? []
  skills.value = rules.skills ?? []
  attention.value = worth
}

async function reindex() {
  working.value = true
  try {
    const [read] = await api.index(name.value)
    lastRead.value = read
    error.value = ''
  } catch (problem) {
    error.value = problem.message
  }
  working.value = false
  await load()
}

async function drop() {
  if (!confirm(`Stop covering ${repository.value.path}?\n\nWhat was already indexed is left alone.`))
    return
  try {
    await api.dropRepository(repository.value.path)
    await fetchRepositories()
    router.push('/repositories')
  } catch (problem) {
    error.value = problem.message
  }
}

watch(name, load)
onMounted(async () => {
  await fetchRepositories()
  await load()
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <PageHead :title="name" :sub="repository?.path" mono>
      <template #back>
        <UiButton variant="ghost" size="icon" aria-label="Back to repositories" @click="up('/repositories')">
          <ArrowLeft class="h-4 w-4" />
        </UiButton>
      </template>
      <template #icon><Folder class="h-5 w-5" /></template>
      <template #badges>
        <UiBadge :variant="counts?.files ? 'success' : 'warning'">
          {{ counts?.files ? 'Indexed' : 'Not indexed' }}
        </UiBadge>
      </template>
      <template #actions>
        <UiButton variant="outline" :disabled="working" @click="reindex">
          <Loader2 v-if="working" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
          <RefreshCw v-else class="mr-1.5 h-3.5 w-3.5" />
          {{ working ? 'Reading…' : 'Re-index' }}
        </UiButton>
        <UiButton variant="ghost" size="icon" aria-label="Stop covering this folder" @click="drop">
          <Trash2 class="h-4 w-4" />
        </UiButton>
      </template>
    </PageHead>

    <Notice v-if="error" tone="danger" class="mb-4">
      {{ error }}
    </Notice>

    <Tabs v-model="tab" :tabs="tabs" label="What to look at" class="mb-4 w-fit" />

    <div v-if="tab === 'overview'" class="space-y-3">
      <UiCard class="p-5">
        <dl class="flex flex-wrap gap-x-8 gap-y-3">
          <div v-for="figure in figures" :key="figure.label">
            <dt class="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
              <component :is="figure.icon" class="h-3.5 w-3.5" />{{ figure.label }}
            </dt>
            <dd class="mt-0.5 text-lg font-semibold">{{ figure.value.toLocaleString() }}</dd>
          </div>
        </dl>

        <p v-if="lastRead" class="mt-4 text-sm text-primary">
          <template v-if="lastRead.indexed">Read {{ lastRead.indexed.toLocaleString() }} files just now.</template>
          <template v-else>Nothing had changed.</template>
        </p>
        <p v-else-if="counts === null" class="mt-4 text-sm text-muted-foreground">
          Not read yet. Re-index to read it.
        </p>

        <div class="mt-4 flex flex-wrap gap-2">
          <UiButton variant="outline" size="sm" @click="router.push('/reviews')">
            <ShieldCheck class="mr-1.5 h-3.5 w-3.5" />
            Review what has changed
          </UiButton>
          <UiButton variant="outline" size="sm" @click="router.push('/knowledge')">
            <BookOpen class="mr-1.5 h-3.5 w-3.5" />
            Record something
          </UiButton>
        </div>
      </UiCard>

      <UiCard v-if="attention.files.length" class="p-5">
        <h2 class="font-semibold">Where to look first</h2>
        <p class="mb-4 mt-0.5 text-sm text-muted-foreground">
          Files that have been changing in the last {{ attention.since }} and that the rest of the
          code leans on. Either on its own says little: something everything imports and nobody
          touches is settled, and something nothing imports that changes daily is a scratch pad.
          Where they meet is where a change is most likely to catch somebody out, and is the
          shortest list worth reading first.
        </p>
        <ul class="space-y-2">
          <li v-for="file in attention.files" :key="file.path" class="flex items-center gap-3">
            <span class="min-w-0 flex-1 truncate font-mono text-sm" :title="file.path">
              {{ file.path }}
            </span>
            <span class="hidden h-1.5 w-24 shrink-0 overflow-hidden rounded-full bg-muted sm:block">
              <span
                class="block h-full rounded-full bg-pillar-graph"
                :style="{ width: `${Math.max(4, (file.changes / busiest) * 100)}%` }"
              />
            </span>
            <span class="w-28 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
              {{ file.changes }} change{{ file.changes === 1 ? '' : 's' }}
            </span>
            <span class="w-24 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
              {{ file.dependants }} depend{{ file.dependants === 1 ? 's' : '' }}
            </span>
          </li>
        </ul>
      </UiCard>

    </div>

    <div v-else-if="tab === 'knowledge'" class="space-y-3">
      <ItemCard v-for="item in knowledge" :key="item.id" :title="item.id" pillar="memory">
        <template #icon><BookOpen class="h-5 w-5" /></template>
        <template #badges>
          <UiBadge variant="secondary">{{ item.kind }}</UiBadge>
          <UiBadge v-if="item.status" variant="outline">{{ item.status }}</UiBadge>
        </template>
        <p class="text-sm text-muted-foreground">{{ item.summary }}</p>
      </ItemCard>
      <UiCard v-if="!knowledge.length" class="p-10 text-center">
        <p class="font-medium">Nothing recorded about this repository yet.</p>
        <UiButton class="mt-3" variant="outline" @click="router.push('/knowledge')">Record something</UiButton>
      </UiCard>
    </div>

    <div v-else-if="tab === 'skills'" class="space-y-3">
      <ItemCard
        v-for="skill in skills"
        :key="skill.id"
        :title="skill.name"
        :subtitle="skill.path"
        pillar="review"
      >
        <template #icon><ScrollText class="h-5 w-5" /></template>
        <template #badges>
          <UiBadge :variant="skill.origin === 'repository' ? 'success' : 'outline'">
            {{ skill.origin === 'repository' ? 'this repository' : skill.origin }}
          </UiBadge>
        </template>
        <p class="text-sm text-muted-foreground">{{ skill.description }}</p>
      </ItemCard>
      <UiCard v-if="!skills.length" class="p-10 text-center">
        <p class="font-medium">Nothing written down for this repository yet.</p>
        <UiButton class="mt-3" variant="outline" @click="router.push('/skills/new')">Add skill</UiButton>
      </UiCard>
    </div>

    <GraphWorkbench
      v-else
      :key="name"
      :repository="name"
      :sources="['code']"
      :controls="['filter', 'kinds', 'parts', 'layouts', 'depth']"
      height="calc(100vh - 19rem)"
    />
  </div>
</template>
