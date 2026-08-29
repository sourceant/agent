<script setup>
import { onMounted, ref, computed } from 'vue'
import { LayoutDashboard, FileCode, Lightbulb, Folder } from 'lucide-vue-next'
import { Card as UiCard } from '@sourceant/design'
import { Badge as UiBadge } from '@sourceant/design'
import { Button as UiButton } from '@sourceant/design'
import PageHead from '~/components/PageHead.vue'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

const { repositories, error, fetchRepositories } = useRepositories()
const counted = ref([])

const totals = computed(() => ({
  repositories: repositories.value.length,
  files: counted.value.reduce((sum, item) => sum + item.files, 0),
  nodes: counted.value.reduce((sum, item) => sum + item.nodes, 0),
  knowledge: counted.value.reduce((sum, item) => sum + item.knowledge, 0),
}))

onMounted(async () => {
  await fetchRepositories()
  counted.value = await Promise.all(repositories.value.map(async (repository) => {
    const [graph, knowledge] = await Promise.all([
      api.graph(repository.name).catch(() => null),
      api.knowledge(repository.name).catch(() => null),
    ])
    return {
      repository,
      files: graph ? graph.nodes.filter((node) => node.kind === 'file').length : 0,
      nodes: graph ? graph.nodes.length : 0,
      knowledge: knowledge ? knowledge.total : 0,
    }
  }))
})
</script>

<template>
  <div>
    <PageHead
      :icon="LayoutDashboard"
      pillar="memory"
      title="Overview"
      sub="What SourceAnt has on this machine."
    />

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <EmptyMachine v-if="!error && repositories.length === 0" />

    <template v-else-if="repositories.length">
      <div class="grid gap-3 mb-6 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard v-for="(value, label) in totals" :key="label" class="p-5">
          <p class="text-xs text-muted-foreground capitalize">{{ label }}</p>
          <p class="mt-1 text-2xl font-semibold tabular-nums">{{ value.toLocaleString() }}</p>
        </UiCard>
      </div>

      <div class="grid gap-3">
        <UiCard v-for="item in counted" :key="item.repository.name" hover class="p-5">
          <div class="flex items-start gap-4">
            <div class="h-11 w-11 shrink-0 rounded-lg bg-pillar-graph/15 text-pillar-graph flex items-center justify-center">
              <Folder class="h-5 w-5" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-2 mb-1">
                <h3 class="font-semibold">{{ item.repository.name }}</h3>
                <UiBadge :variant="item.files ? 'success' : 'warning'">
                  {{ item.files ? 'Indexed' : 'Not indexed' }}
                </UiBadge>
              </div>
              <p class="text-sm text-muted-foreground font-mono break-all">{{ item.repository.path }}</p>
              <div class="flex flex-wrap items-center gap-4 mt-2 text-sm text-muted-foreground">
                <span class="flex items-center gap-1.5">
                  <FileCode class="h-3.5 w-3.5" />{{ item.files.toLocaleString() }} files
                </span>
                <span class="flex items-center gap-1.5">
                  <Lightbulb class="h-3.5 w-3.5" />{{ item.knowledge.toLocaleString() }} recorded
                </span>
              </div>
            </div>
            <UiButton as="a" href="#/graph" variant="ghost" size="sm" class="shrink-0">Graph</UiButton>
          </div>
        </UiCard>
      </div>

      <p class="mt-4 text-xs text-muted-foreground">
        Reviews are not here. A review reads a pull request, and nothing on this machine
        produces one.
      </p>
    </template>
  </div>
</template>
