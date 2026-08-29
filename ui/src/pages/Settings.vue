<script setup>
import { Badge as UiBadge, Button as UiButton, Card as UiCard, PageHead } from '@sourceant/design'
import { onMounted, ref } from 'vue'
import { Settings as SettingsIcon, Sun, Moon } from 'lucide-vue-next'
import ModelSettings from '~/components/ModelSettings.vue'
import { useTheme } from '~/composables/useTheme'
import { api } from '~/api'

const { isDark, toggleTheme } = useTheme()
const status = ref(null)
const problem = ref('')

onMounted(async () => {
  try {
    status.value = await api.status()
  } catch (error) {
    problem.value = error.message
  }
})
</script>

<template>
  <div>
    <PageHead pillar="tokens" title="Settings" sub="What is running, and how this looks.">
      <template #icon><SettingsIcon class="h-6 w-6" /></template>
    </PageHead>

    <p v-if="problem" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ problem }}
    </p>

    <UiCard class="p-5 mb-3">
      <h2 class="font-semibold mb-3">Appearance</h2>
      <UiButton variant="outline" @click="toggleTheme">
        <component :is="isDark ? Sun : Moon" class="mr-2 h-4 w-4" />
        {{ isDark ? 'Light mode' : 'Dark mode' }}
      </UiButton>
    </UiCard>

    <UiCard class="p-5 mb-3">
      <h2 class="font-semibold mb-1">Model</h2>
      <p class="text-sm text-muted-foreground mb-4">
        Reading a repository needs none of this. Anything that proposes rather than reads
        does, and it stays off until you say which model to ask.
      </p>
      <ModelSettings />
    </UiCard>

    <UiCard v-if="status" class="p-5">
      <h2 class="font-semibold mb-3">What is running</h2>
      <dl class="grid grid-cols-[auto_1fr] gap-x-4 gap-y-2 text-sm">
        <dt class="text-muted-foreground">Agent</dt>
        <dd class="font-mono">{{ status.version }}</dd>
        <dt class="text-muted-foreground">Indexer</dt>
        <dd class="font-mono break-all">
          {{ status.core_url }}
          <UiBadge :variant="status.core_up ? 'success' : 'destructive'" class="ml-2">
            {{ status.core_up ? 'answering' : 'not answering' }}
          </UiBadge>
        </dd>
        <dt class="text-muted-foreground">Starts</dt>
        <dd class="font-mono">{{ status.core_starts }}</dd>
        <template v-if="status.last_exit">
          <dt class="text-muted-foreground">Last exit</dt>
          <dd class="font-mono break-all">{{ status.last_exit }}</dd>
        </template>
      </dl>
      <p class="mt-4 text-xs text-muted-foreground">
        Where the indexer comes from is chosen at install time and kept in
        <code class="font-mono">~/.sourceant/config.json</code>.
      </p>
    </UiCard>
  </div>
</template>
