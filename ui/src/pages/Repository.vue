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
        parts: graph.communities ?? 0,
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

    <div v-if="tab === 'overview'" class="grid gap-3 sm:grid-cols-3">
      <UiCard class="p-5">
        <p class="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
          <FileCode class="h-3.5 w-3.5" />Files
        </p>
        <p class="mt-1 text-2xl font-semibold">{{ (counts?.files ?? 0).toLocaleString() }}</p>
      </UiCard>
      <UiCard class="p-5">
        <p class="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
          <Link2 class="h-3.5 w-3.5" />Connections
        </p>
        <p class="mt-1 text-2xl font-semibold">{{ (counts?.links ?? 0).toLocaleString() }}</p>
      </UiCard>
      <UiCard class="p-5">
        <p class="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
          <Network class="h-3.5 w-3.5" />Parts
        </p>
        <p class="mt-1 text-2xl font-semibold">{{ (counts?.parts ?? 0).toLocaleString() }}</p>
      </UiCard>

      <UiCard class="p-5 sm:col-span-3">
        <p v-if="lastRead" class="text-sm text-primary">
          <template v-if="lastRead.indexed">Read {{ lastRead.indexed.toLocaleString() }} files just now.</template>
          <template v-else>Nothing had changed.</template>
        </p>
        <p v-else-if="counts === null" class="text-sm text-muted-foreground">
          Not read yet. Re-index to read it.
        </p>
        <p v-else class="text-sm text-muted-foreground">
          Everything on this page comes off this folder. Nothing about it has left this machine.
        </p>
        <div class="mt-3 flex flex-wrap gap-2">
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
        <p class="font-medium">No rules on hand for this repository.</p>
        <UiButton class="mt-3" variant="outline" @click="router.push('/skills')">Write one down</UiButton>
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
