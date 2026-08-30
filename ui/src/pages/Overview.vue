<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  ItemCard,
  Notice,
  PageHead,
} from '@sourceant/design'
import { onMounted, ref, computed } from 'vue'
import { LayoutDashboard, FileCode, Lightbulb, Folder } from 'lucide-vue-next'
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
    <PageHead pillar="memory" title="Overview" sub="What SourceAnt has on this machine.">
      <template #icon><LayoutDashboard class="h-6 w-6" /></template>
    </PageHead>

    <Notice v-if="error" tone="danger" class="mb-4">
      {{ error }}
    </Notice>

    <EmptyMachine v-if="!error && repositories.length === 0" />

    <template v-else-if="repositories.length">
      <div class="grid gap-3 mb-6 sm:grid-cols-2 lg:grid-cols-4">
        <UiCard v-for="(value, label) in totals" :key="label" class="p-5">
          <p class="text-xs text-muted-foreground capitalize">{{ label }}</p>
          <p class="mt-1 text-2xl font-semibold tabular-nums">{{ value.toLocaleString() }}</p>
        </UiCard>
      </div>

      <div class="grid gap-3">
        <ItemCard
          v-for="item in counted"
          :key="item.repository.name"
          :title="item.repository.name"
          :subtitle="item.repository.path"
          hover
        >
          <template #icon><Folder class="h-5 w-5" /></template>
          <template #badges>
            <UiBadge :variant="item.files ? 'success' : 'warning'">
              {{ item.files ? 'Indexed' : 'Not indexed' }}
            </UiBadge>
          </template>
          <template #meta>
            <span class="flex items-center gap-1.5">
              <FileCode class="h-3.5 w-3.5" />{{ item.files.toLocaleString() }} files
            </span>
            <span class="flex items-center gap-1.5">
              <Lightbulb class="h-3.5 w-3.5" />{{ item.knowledge.toLocaleString() }} recorded
            </span>
          </template>
          <template #actions>
            <UiButton as="a" href="/graph" variant="ghost" size="sm">Graph</UiButton>
          </template>
        </ItemCard>
      </div>

      <p class="mt-4 text-xs text-muted-foreground">
        Reviews are not here. A review reads a pull request, and nothing on this machine
        produces one.
      </p>
    </template>
  </div>
</template>
