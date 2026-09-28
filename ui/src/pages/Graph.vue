<script setup>
import { onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Network } from 'lucide-vue-next'
import { Notice, PageHead, Select } from '@sourceant/design'
import GraphWorkbench from '~/components/GraphWorkbench.vue'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'

const route = useRoute()
const { repositories, chosen, one, error, fetchRepositories } = useRepositories()

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

    <GraphWorkbench
      v-else
      :key="one"
      :repository="one"
      :find="String(route.query.q ?? '')"
      :sources="['code']"
      :controls="['filter', 'kinds', 'parts', 'layouts', 'depth']"
      height="calc(100vh - 15rem)"
    />
  </div>
</template>
