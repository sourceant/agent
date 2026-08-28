<script setup>
import { onMounted } from 'vue'
import { Network } from 'lucide-vue-next'
import PageHead from '~/components/PageHead.vue'
import GraphWorkbench from '~/components/GraphWorkbench.vue'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'

const { repositories, chosen, error, fetchRepositories } = useRepositories()

/* Only the code source. Knowledge is a list on a machine: nothing serves the
 * relationships between one record and another, so there is no graph of it to
 * draw yet. */
onMounted(fetchRepositories)
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <PageHead :icon="Network" title="Graphs" sub="Your code, and how it holds together.">
      <template #actions>
        <select
          v-if="repositories.length > 1"
          v-model="chosen"
          class="rounded-md border bg-card px-3 py-1.5 text-sm"
          aria-label="Repository"
        >
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </select>
      </template>
    </PageHead>

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <EmptyMachine v-if="repositories.length === 0" />

    <GraphWorkbench
      v-else
      :key="chosen"
      :repository="chosen"
      :sources="['code']"
      :controls="['filter', 'kinds', 'parts', 'layouts', 'depth']"
      height="calc(100vh - 15rem)"
    />
  </div>
</template>
