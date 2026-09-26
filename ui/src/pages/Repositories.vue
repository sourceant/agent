<script setup>
import {
  Empty,
  Button as UiButton,
  Card as UiCard,
  ItemCard,
  Notice,
  PageHead,
} from '@sourceant/design'
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { Boxes, Plus, Trash2, RefreshCw, FileCode, Link2, Folder, Loader2 } from 'lucide-vue-next'
import FolderPicker from '~/components/FolderPicker.vue'
import { useRepositories } from '~/composables/useRepositories'
import { when } from '~/moments'
import { api } from '~/api'

const router = useRouter()
const { repositories, error, fetchRepositories } = useRepositories()
const counts = ref({})
const working = ref('')
const picking = ref(false)
// What the last read found, so pressing the button says something. Reading a
// repository that has not changed reports nothing changed, which is a result;
// showing no result at all is indistinguishable from the button not working.
const lastRead = ref({})
const reading = computed(() => repositories.value.some((one) => one.reading))
let asking = null

async function countAll() {
  for (const repository of repositories.value) {
    const graph = await api.graph(repository.name).catch(() => null)
    counts.value = {
      ...counts.value,
      [repository.name]: graph
        ? { files: graph.nodes.filter((node) => node.kind === 'file').length, links: graph.links.length }
        : null,
    }
  }
}

async function refresh() {
  await fetchRepositories()
  await countAll()
  keepAsking()
}

// A read that somebody else started, by adding a folder or on the schedule,
// finishes without anybody pressing anything here. Only the list is asked for
// while it runs: counting means a call per repository.
function keepAsking() {
  clearTimeout(asking)
  if (!reading.value) return
  asking = setTimeout(async () => {
    await fetchRepositories()
    if (!reading.value) await countAll()
    keepAsking()
  }, 2000)
}

async function reindex(name) {
  working.value = name
  try {
    const [read] = await api.index(name)
    lastRead.value = { ...lastRead.value, [name]: read }
    error.value = ''
  } catch (problem) {
    error.value = problem.message
  }
  working.value = ''
  await refresh()
}

function readingSaid(read) {
  if (!read) return ''
  if (read.indexed) return `Read ${read.indexed.toLocaleString()} files just now.`
  return 'Nothing had changed.'
}

function freshness(repository) {
  if (repository.reading) return 'Reading…'
  if (!repository.indexed_at) return 'Not read yet'
  return `Read ${when(repository.indexed_at)}`
}

async function drop(repository) {
  if (!confirm(`Stop covering ${repository.path}?\n\nWhat was already indexed is left alone.`)) return
  try {
    await api.dropRepository(repository.path)
  } catch (problem) {
    error.value = problem.message
  }
  await refresh()
}

onMounted(refresh)
onUnmounted(() => clearTimeout(asking))
</script>

<template>
  <div>
    <PageHead title="Repositories" sub="The folders SourceAnt reads on this machine.">
      <template #icon><Boxes class="h-6 w-6" /></template>
      <template #actions>
        <UiButton variant="glow" @click="picking = true">
          <Plus class="mr-2 h-4 w-4" />
          Add a folder
        </UiButton>
      </template>
    </PageHead>

    <Notice v-if="error" tone="danger" class="mb-4">
      {{ error }}
    </Notice>

    <Empty v-if="repositories.length === 0" title="No folders yet">
      Point SourceAnt at a repository and it reads the files into a graph.
      <template #actions>
        <UiButton variant="glow" @click="picking = true">
          <Plus class="mr-2 h-4 w-4" />
          Add a folder
        </UiButton>
      </template>
    </Empty>

    <div v-else class="grid gap-3">
      <ItemCard
        v-for="repository in repositories"
        :key="repository.path"
        :title="repository.name"
        :subtitle="repository.path"
        hover
        class="cursor-pointer"
        @click="router.push(`/repositories/${repository.name}`)"
      >
        <template #icon><Folder class="h-5 w-5" /></template>
        <template #meta>
          <template v-if="counts[repository.name]">
            <span class="flex items-center gap-1.5">
              <FileCode class="h-3.5 w-3.5" />{{ counts[repository.name].files.toLocaleString() }} files
            </span>
            <span class="flex items-center gap-1.5">
              <Link2 class="h-3.5 w-3.5" />{{ counts[repository.name].links.toLocaleString() }} links
            </span>
          </template>
          <span class="flex items-center gap-1.5">
            <Loader2 v-if="repository.reading" class="h-3.5 w-3.5 animate-spin" />
            {{ freshness(repository) }}
          </span>
          <span v-if="lastRead[repository.name]" class="text-primary">
            {{ readingSaid(lastRead[repository.name]) }}
          </span>
        </template>
        <template #actions>
          <UiButton
            variant="outline"
            size="sm"
            :disabled="working === repository.name || repository.reading"
            @click.stop="reindex(repository.name)"
          >
            <Loader2 v-if="working === repository.name" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
            <RefreshCw v-else class="mr-1.5 h-3.5 w-3.5" />
            {{ working === repository.name ? 'Reading…' : 'Re-index' }}
          </UiButton>
          <UiButton variant="ghost" size="icon" :aria-label="`Remove ${repository.name}`" @click.stop="drop(repository)">
            <Trash2 class="h-4 w-4" />
          </UiButton>
        </template>
      </ItemCard>
    </div>

    <FolderPicker :open="picking" @close="picking = false" @added="refresh" />
  </div>
</template>
