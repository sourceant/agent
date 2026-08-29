<script setup>
import { Button as UiButton, Card as UiCard, ItemCard, PageHead } from '@sourceant/design'
import { onMounted, ref } from 'vue'
import { Boxes, Plus, Trash2, RefreshCw, FileCode, Link2, Folder, Loader2 } from 'lucide-vue-next'
import FolderPicker from '~/components/FolderPicker.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

const { repositories, error, fetchRepositories } = useRepositories()
const counts = ref({})
const working = ref('')
const picking = ref(false)
// What the last read found, so pressing the button says something. Reading a
// repository that has not changed reports nothing changed, which is a result;
// showing no result at all is indistinguishable from the button not working.
const lastRead = ref({})

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
  await countAll()
}

function readingSaid(read) {
  if (!read) return ''
  if (read.indexed) return `Read ${read.indexed.toLocaleString()} files just now.`
  return 'Nothing had changed.'
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

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <UiCard v-if="repositories.length === 0" class="text-center py-16 px-6">
      <h2 class="text-lg font-semibold mb-1">No folders yet</h2>
      <p class="text-muted-foreground text-sm mb-4">
        Point SourceAnt at a repository and it reads the files into a graph.
      </p>
      <UiButton variant="glow" @click="picking = true">
        <Plus class="mr-2 h-4 w-4" />
        Add a folder
      </UiButton>
    </UiCard>

    <div v-else class="grid gap-3">
      <ItemCard
        v-for="repository in repositories"
        :key="repository.path"
        :title="repository.name"
        :subtitle="repository.path"
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
          <span v-else-if="counts[repository.name] === null">Not read yet. Re-index to read it.</span>
          <span v-else>Reading…</span>
          <span v-if="lastRead[repository.name]" class="text-primary">
            {{ readingSaid(lastRead[repository.name]) }}
          </span>
        </template>
        <template #actions>
          <UiButton
            variant="outline"
            size="sm"
            :disabled="working === repository.name"
            @click="reindex(repository.name)"
          >
            <Loader2 v-if="working === repository.name" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
            <RefreshCw v-else class="mr-1.5 h-3.5 w-3.5" />
            {{ working === repository.name ? 'Reading…' : 'Re-index' }}
          </UiButton>
          <UiButton variant="ghost" size="icon" :aria-label="`Remove ${repository.name}`" @click="drop(repository)">
            <Trash2 class="h-4 w-4" />
          </UiButton>
        </template>
      </ItemCard>
    </div>

    <FolderPicker :open="picking" @close="picking = false" @added="refresh" />
  </div>
</template>
