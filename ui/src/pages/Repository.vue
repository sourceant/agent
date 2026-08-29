<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  ItemCard,
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
const { repositories, chosen, fetchRepositories } = useRepositories()

const name = computed(() => String(route.params.name ?? ''))
const repository = computed(() => repositories.value.find((item) => item.name === name.value))

const tab = ref('overview')
const counts = ref(null)
const knowledge = ref([])
const skills = ref([])
const working = ref(false)
const lastRead = ref(null)
const error = ref('')

const figures = computed(() => [
  { label: 'Files', value: counts.value?.files ?? 0, icon: FileCode },
  { label: 'Connections', value: counts.value?.links ?? 0, icon: Link2 },
  { label: 'Parts', value: counts.value?.parts?.length ?? 0, icon: Network },
  { label: 'Recorded', value: knowledge.value.length, icon: BookOpen },
  { label: 'Rules', value: skills.value.length, icon: ScrollText },
])

const biggestPart = computed(() => counts.value?.parts?.[0]?.size || 1)

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
  const [graph, recorded, rules] = await Promise.all([
    api.graph(name.value).catch(() => null),
    api.knowledge(name.value).catch(() => ({ items: [] })),
    api.skills(name.value).catch(() => ({ skills: [] })),
  ])
  counts.value = graph
    ? {
        files: graph.nodes.filter((node) => node.kind === 'file').length,
        links: graph.links.length,
        // Each part is named after what it holds, which is more use than
        // counting them.
        parts: [...(graph.communities ?? [])].sort((a, b) => b.size - a.size),
      }
    : null
  knowledge.value = recorded.items ?? []
  skills.value = rules.skills ?? []
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
        <UiButton variant="ghost" size="icon" aria-label="Back to repositories" @click="router.push('/repositories')">
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

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

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

      <UiCard v-if="counts?.parts?.length" class="p-5">
        <h2 class="font-semibold">What it is made of</h2>
        <p class="mb-4 mt-0.5 text-sm text-muted-foreground">
          The parts the code falls into, named after what each one holds. Found by how tightly
          the files in them refer to each other, not by which folder they sit in.
        </p>
        <ul class="space-y-1.5">
          <li v-for="part in counts.parts" :key="part.id" class="flex items-center gap-3">
            <span class="w-40 shrink-0 truncate text-sm font-medium">{{ part.name }}</span>
            <span class="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
              <span
                class="block h-full rounded-full bg-pillar-graph"
                :style="{ width: `${Math.max(2, (part.size / biggestPart) * 100)}%` }"
              />
            </span>
            <span class="w-16 shrink-0 text-right text-sm tabular-nums text-muted-foreground">
              {{ part.size.toLocaleString() }}
            </span>
          </li>
        </ul>
        <UiButton class="mt-4" variant="outline" size="sm" @click="tab = 'graph'">
          <Network class="mr-1.5 h-3.5 w-3.5" />
          See them drawn
        </UiButton>
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
        <UiButton class="mt-3" variant="outline" @click="router.push('/skills/new')">Write one down</UiButton>
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
