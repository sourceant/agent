<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Empty,
  Notice,
  PageHead,
  Table,
} from '@sourceant/design'
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  ArrowRight,
  BookOpenCheck,
  FileCode,
  LayoutDashboard,
  Lightbulb,
  Loader2,
  RefreshCw,
  ShieldCheck,
  TriangleAlert,
} from 'lucide-vue-next'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'
import { when } from '~/moments'
import { api } from '~/api'

/* What this machine knows, and the one thing worth doing about it.
 *
 * Counts alone say nothing: a number is only useful beside what it should be.
 * A folder nobody has read, a review that found something, a proposal waiting
 * for a person: those are the reasons to open this screen, so they are what it
 * leads with.
 */

const MOST = 5

const router = useRouter()
const { repositories, error, fetchRepositories } = useRepositories()

const counted = ref([])
const reviews = ref([])
const attention = ref({ files: [], since: '' })
const settings = ref([])
const loading = ref(true)

const columns = [
  { id: 'repository', label: 'Repository' },
  { id: 'files', label: 'Files', align: 'right' },
  { id: 'knowledge', label: 'Knowledge', align: 'right', narrow: true },
  { id: 'read', label: 'Last read', narrow: true },
]

const totals = computed(() => ({
  files: counted.value.reduce((sum, one) => sum + one.files, 0),
  knowledge: counted.value.reduce((sum, one) => sum + one.knowledge, 0),
  proposed: counted.value.reduce((sum, one) => sum + one.proposed, 0),
  skills: counted.value.reduce((sum, one) => sum + one.skills, 0),
}))

const unread = computed(() => repositories.value.filter((one) => !one.indexed_at))
const reading = computed(() => repositories.value.filter((one) => one.reading))
const failed = computed(() => reviews.value.filter((one) => one.status === 'failed'))
const model = computed(() => {
  const named = settings.value.find((one) => one.key === 'model.name')?.value
  const keyed = settings.value.find((one) => one.key === 'model.api_key')?.is_set
  return named && keyed ? String(named) : ''
})

/* The one thing worth doing, in the order it matters. Everything on this screen
 * is reachable anyway; this is what somebody would otherwise have to go looking
 * for. */
const next = computed(() => {
  if (!model.value) {
    return {
      tone: 'warning',
      icon: TriangleAlert,
      title: 'No model configured',
      detail: 'Reviews read a change but cannot judge it.',
      label: 'Choose a model',
      to: '/settings?group=Model',
    }
  }
  if (unread.value.length) {
    return {
      tone: 'warning',
      icon: RefreshCw,
      title: `${unread.value.length} folder${unread.value.length === 1 ? '' : 's'} never read`,
      detail: 'Nothing can be answered about a folder that has not been read.',
      label: 'Repositories',
      to: '/repositories',
    }
  }
  if (totals.value.proposed) {
    return {
      tone: 'info',
      icon: Lightbulb,
      title: `${totals.value.proposed} proposal${totals.value.proposed === 1 ? '' : 's'} waiting`,
      detail: 'Nothing proposed is used until somebody agrees to it.',
      label: 'Knowledge',
      to: '/knowledge',
    }
  }
  if (failed.value.length) {
    return {
      tone: 'warning',
      icon: TriangleAlert,
      title: `${failed.value.length} review${failed.value.length === 1 ? '' : 's'} failed`,
      detail: failed.value[0].error || 'Ask for it again.',
      label: 'Reviews',
      to: '/reviews',
    }
  }
  return {
    tone: 'success',
    icon: ShieldCheck,
    title: 'Everything is read',
    detail: `${model.value} is configured, so a change can be judged as well as read.`,
    label: 'Review a checkout',
    to: '/reviews',
  }
})

const TONES = {
  warning: 'border-warning/40 bg-warning/5',
  info: 'border-primary/30 bg-primary/5',
  success: 'border-success/40 bg-success/5',
}

const rows = computed(() =>
  counted.value.map((one) => ({
    id: one.repository.name,
    repository: one.repository,
    files: one.files,
    knowledge: one.knowledge,
  })),
)

function freshness(repository) {
  if (repository.reading) return 'Reading…'
  if (!repository.indexed_at) return 'Never'
  return when(repository.indexed_at)
}

onMounted(async () => {
  await fetchRepositories()
  const [past, configured] = await Promise.all([
    api.reviews().catch(() => []),
    api.settings().catch(() => []),
  ])
  reviews.value = past.slice(0, MOST)
  settings.value = configured
  // A count rather than a graph: the number of files in a repository does not
  // need the repository.
  counted.value = await Promise.all(
    repositories.value.map(async (repository) => {
      const [files, knowledge, skills] = await Promise.all([
        // The label the index files them under, which is capitalised there.
        api.nodes(repository.name, { labels: ['File'] }).catch(() => ({ total: 0 })),
        api.knowledge(repository.name).catch(() => ({ items: [], total: 0 })),
        api.skills(repository.name).catch(() => ({ total: 0 })),
      ])
      return {
        repository,
        files: files.total,
        knowledge: knowledge.total,
        proposed: (knowledge.items ?? []).filter((one) => one.status === 'proposed').length,
        skills: skills.total,
      }
    }),
  )
  const first = repositories.value[0]
  if (first) attention.value = await api.attention(first.name).catch(() => ({ files: [], since: '' }))
  loading.value = false
})
</script>

<template>
  <div>
    <PageHead pillar="memory" title="Overview" sub="What SourceAnt has on this machine.">
      <template #icon><LayoutDashboard class="h-6 w-6" /></template>
    </PageHead>

    <Notice v-if="error" tone="danger" class="mb-4">
      {{ error }}
    </Notice>

    <EmptyMachine v-if="!error && repositories.length === 0" />

    <template v-else-if="repositories.length">
      <UiCard :class="['mb-3 flex flex-wrap items-center gap-4 p-5', TONES[next.tone]]">
        <component :is="next.icon" class="h-5 w-5 shrink-0" />
        <div class="min-w-0 flex-1">
          <p class="font-semibold">{{ next.title }}</p>
          <p class="text-sm text-muted-foreground">{{ next.detail }}</p>
        </div>
        <UiButton size="sm" variant="outline" @click="router.push(next.to)">
          {{ next.label }}
          <ArrowRight class="ml-1.5 h-3.5 w-3.5" />
        </UiButton>
      </UiCard>

      <div class="mb-3 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard class="p-4">
          <p class="text-xs text-muted-foreground">Repositories</p>
          <p class="mt-1 text-2xl font-semibold tabular-nums">{{ repositories.length }}</p>
          <p class="mt-0.5 text-xs text-muted-foreground">
            <template v-if="reading.length">{{ reading.length }} reading now</template>
            <template v-else-if="unread.length">{{ unread.length }} never read</template>
            <template v-else>all read</template>
          </p>
        </UiCard>
        <UiCard class="p-4">
          <p class="text-xs text-muted-foreground">Files</p>
          <p class="mt-1 text-2xl font-semibold tabular-nums">
            <Loader2 v-if="loading" class="h-5 w-5 animate-spin" />
            <template v-else>{{ totals.files.toLocaleString() }}</template>
          </p>
          <p class="mt-0.5 text-xs text-muted-foreground">indexed</p>
        </UiCard>
        <UiCard class="p-4">
          <p class="text-xs text-muted-foreground">Knowledge</p>
          <p class="mt-1 text-2xl font-semibold tabular-nums">{{ totals.knowledge.toLocaleString() }}</p>
          <p class="mt-0.5 text-xs text-muted-foreground">
            {{ totals.proposed ? `${totals.proposed} waiting for you` : 'nothing waiting' }}
          </p>
        </UiCard>
        <UiCard class="p-4">
          <p class="text-xs text-muted-foreground">Skills</p>
          <p class="mt-1 text-2xl font-semibold tabular-nums">{{ totals.skills.toLocaleString() }}</p>
          <p class="mt-0.5 text-xs text-muted-foreground">read against your work</p>
        </UiCard>
      </div>

      <div class="mb-3 grid gap-3 lg:grid-cols-2">
        <UiCard class="p-5">
          <div class="mb-3 flex items-center justify-between gap-3">
            <h2 class="flex items-center gap-2 text-sm font-semibold">
              <ShieldCheck class="h-4 w-4" />
              Recent reviews
            </h2>
            <UiButton variant="ghost" size="sm" @click="router.push('/reviews')">All</UiButton>
          </div>
          <Empty v-if="!reviews.length" title="No reviews yet" compact>
            A review reads what a checkout has that its default branch does not.
          </Empty>
          <ul v-else class="divide-y">
            <li v-for="one in reviews" :key="one.id">
              <button
                type="button"
                class="flex w-full items-baseline gap-2 py-2 text-left text-sm hover:text-primary"
                @click="router.push(`/reviews/${one.id}`)"
              >
                <UiBadge
                  :variant="one.status === 'done' ? 'success' : one.status === 'failed' ? 'destructive' : 'secondary'"
                >
                  {{ one.status }}
                </UiBadge>
                <span class="min-w-0 flex-1 truncate">{{ one.title || one.repository }}</span>
                <span class="shrink-0 text-xs text-muted-foreground">{{ when(one.started) }}</span>
              </button>
            </li>
          </ul>
        </UiCard>

        <UiCard class="p-5">
          <div class="mb-3 flex items-center justify-between gap-3">
            <h2 class="flex items-center gap-2 text-sm font-semibold">
              <FileCode class="h-4 w-4" />
              Worth reading first
            </h2>
            <span v-if="attention.since" class="text-xs text-muted-foreground">
              last {{ attention.since }}
            </span>
          </div>
          <Empty v-if="!attention.files.length" title="Nothing standing out" compact>
            Where recent change lands on what the rest of the code leans on appears here.
          </Empty>
          <ul v-else class="divide-y">
            <li
              v-for="file in attention.files.slice(0, MOST)"
              :key="file.path"
              class="flex items-baseline gap-2 py-2 text-sm"
            >
              <span class="min-w-0 flex-1 truncate font-mono text-xs">{{ file.path }}</span>
              <span class="shrink-0 text-xs text-muted-foreground">
                {{ file.dependants }} leaning on it · {{ file.changes }} changes
              </span>
            </li>
          </ul>
        </UiCard>
      </div>

      <Table :columns="columns" :rows="rows" row-key="id" label="Repositories" class="mb-3">
        <template #repository="{ row }">
          <button
            type="button"
            class="block max-w-md truncate text-left font-medium hover:text-primary"
            @click="router.push(`/repositories/${row.repository.name}`)"
          >
            {{ row.repository.name }}
          </button>
        </template>
        <template #files="{ row }">
          <span class="tabular-nums">{{ row.files.toLocaleString() }}</span>
        </template>
        <template #knowledge="{ row }">
          <span class="tabular-nums">{{ row.knowledge.toLocaleString() }}</span>
        </template>
        <template #read="{ row }">
          <span class="text-muted-foreground">{{ freshness(row.repository) }}</span>
        </template>
      </Table>

      <div class="flex flex-wrap gap-2">
        <UiButton variant="outline" size="sm" @click="router.push('/reviews')">
          <ShieldCheck class="mr-1.5 h-3.5 w-3.5" />
          Review a checkout
        </UiButton>
        <UiButton variant="outline" size="sm" @click="router.push('/skills')">
          <BookOpenCheck class="mr-1.5 h-3.5 w-3.5" />
          Skills
        </UiButton>
      </div>
    </template>
  </div>
</template>
