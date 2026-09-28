<script setup>
import { defineAsyncComponent, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { Network } from 'lucide-vue-next'
import { Notice, PageHead, Select, Tabs } from '@sourceant/design'
import GraphWorkbench from '~/components/GraphWorkbench.vue'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'

const route = useRoute()
const { repositories, chosen, one, error, fetchRepositories } = useRepositories()
const Architecture = defineAsyncComponent(() => import('~/components/Architecture.vue'))
/* A search names what it was looking for, so it opens the view that can show it. */
const view = ref(route.query.q ? 'graph' : 'components')

/* Only the code source. Knowledge is a list on a machine: nothing serves the
 * relationships between one record and another, so there is no graph of it to
 * draw yet. */
onMounted(async () => {
  await fetchRepositories()
  // Arriving from a search, which named both.
  const named = String(route.query.repository ?? '')
  if (named && repositories.value.some((one) => one.name === named)) chosen.value = named
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <PageHead title="Graphs" sub="Your code, and how it holds together.">
      <template #icon><Network class="h-6 w-6" /></template>
      <template #actions>
        <Select v-if="repositories.length > 1" :model-value="one" size="sm" aria-label="Repository" @update:model-value="chosen = $event">
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>
      </template>
    </PageHead>

    <Notice v-if="error" tone="danger" class="mb-4">
      {{ error }}
    </Notice>

    <EmptyMachine v-if="repositories.length === 0" />

    <Tabs v-if="repositories.length" v-model="view" :tabs="[{ id: 'components', label: 'Components' }, { id: 'graph', label: 'Code graph' }]" label="Graph view" class="mb-4 self-start" />
    <Architecture v-if="repositories.length && view === 'components'" :repository="one" />
    <GraphWorkbench
      v-else-if="repositories.length"
      :key="one"
      :repository="one"
      :find="String(route.query.q ?? '')"
      :sources="['code']"
      :controls="['filter', 'kinds', 'parts', 'layouts', 'depth']"
      height="calc(100vh - 15rem)"
    />
  </div>
</template>
