<script setup>
import { onMounted, ref } from 'vue'
import { Boxes, Plus, Trash2, RefreshCw, FileCode, Link2, Folder, Loader2 } from 'lucide-vue-next'
import UiCard from '~/components/ui/Card.vue'
import UiButton from '~/components/ui/Button.vue'
import PageHead from '~/components/PageHead.vue'
import FolderPicker from '~/components/FolderPicker.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'
import { groupOf } from '~/lib/graph'

const { repositories, error, fetchRepositories } = useRepositories()
const counts = ref({})
const working = ref('')
const picking = ref(false)

async function countAll() {
  for (const repository of repositories.value) {
    const graph = await api.graph(repository.name).catch(() => null)
    counts.value = {
      ...counts.value,
      [repository.name]: graph
        ? { files: graph.nodes.filter((node) => groupOf(node) === 'file').length, links: graph.links.length }
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
    await api.index(name)
    error.value = ''
  } catch (problem) {
    error.value = problem.message
  }
  working.value = ''
  await countAll()
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
    <PageHead :icon="Boxes" title="Repositories" sub="The folders SourceAnt reads on this machine.">
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
      <UiCard v-for="repository in repositories" :key="repository.path" class="p-5">
        <div class="flex items-start gap-4">
          <div class="h-11 w-11 shrink-0 rounded-lg bg-pillar-graph/15 text-pillar-graph flex items-center justify-center">
            <Folder class="h-5 w-5" />
          </div>
          <div class="flex-1 min-w-0">
            <h3 class="font-semibold mb-1">{{ repository.name }}</h3>
            <p class="text-sm text-muted-foreground font-mono break-all">{{ repository.path }}</p>
            <div class="flex flex-wrap items-center gap-4 mt-2 text-sm text-muted-foreground">
              <template v-if="counts[repository.name]">
                <span class="flex items-center gap-1.5">
                  <FileCode class="h-3.5 w-3.5" />{{ counts[repository.name].files.toLocaleString() }} files
                </span>
                <span class="flex items-center gap-1.5">
                  <Link2 class="h-3.5 w-3.5" />{{ counts[repository.name].links.toLocaleString() }} links
                </span>
              </template>
              <span v-else-if="counts[repository.name] === null">Not indexed yet. Re-index to read it.</span>
              <span v-else>Reading…</span>
            </div>
          </div>
          <div class="flex shrink-0 items-center gap-1">
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
          </div>
        </div>
      </UiCard>
    </div>

    <FolderPicker :open="picking" @close="picking = false" @added="refresh" />
  </div>
</template>
