<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Notice,
  PageHead,
  Tabs,
} from '@sourceant/design'
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { CircleCheck, Loader2, Settings as SettingsIcon, Sun, Moon } from 'lucide-vue-next'
import McpPanel from '~/components/McpPanel.vue'
import SettingsPanel from '~/components/SettingsPanel.vue'
import { useTheme } from '~/composables/useTheme'
import { api } from '~/api'

/* Overview first, because the question somebody arriving here has is usually
 * whether the thing is working.
 *
 * The tabs after it are not chosen here. The core declares which group each
 * setting belongs to, so a group it starts declaring becomes a tab without a
 * change in this file. */

const OVERVIEW = 'overview'
const MCP = 'mcp'

const route = useRoute()
const { isDark, toggleTheme } = useTheme()
const status = ref(null)
const repositories = ref([])
const groups = ref([])
const problem = ref('')
const tab = ref(OVERVIEW)
const accepts = ref(null)
const checking = ref(false)

const tabs = computed(() => [
  { id: OVERVIEW, label: 'Overview' },
  ...groups.value.map((name) => ({ id: name, label: name })),
  { id: MCP, label: 'MCP' },
])

const model = computed(() => {
  const named = status.value?.model
  return named || 'None chosen'
})

/* Whether the pair that was just saved actually works, asked of the provider.
 * A key that is wrong otherwise shows up as a review failing much later, by
 * which time nobody connects the two. Nothing to check is nothing said. */
async function onSaved() {
  accepts.value = null
  if (tab.value !== 'Model') return
  checking.value = true
  try {
    accepts.value = await api.checkModel()
  } catch {
    accepts.value = null
  }
  checking.value = false
}

onMounted(async () => {
  try {
    const [running, settings, folders] = await Promise.all([
      api.status(),
      api.settings(),
      api.repositories().catch(() => []),
    ])
    status.value = {
      ...running,
      model: settings.find((s) => s.key === 'model.name')?.value ?? '',
      keySet: !!settings.find((s) => s.key === 'model.api_key')?.is_set,
    }
    repositories.value = folders
    groups.value = [...new Set(settings.map((s) => s.group).filter(Boolean))].sort()
    // Opened from somewhere that knows which group it wants, such as a screen
    // saying no model is configured.
    const asked = String(route.query.group ?? '')
    if (groups.value.includes(asked)) tab.value = asked
  } catch (error) {
    problem.value = error.message
  }
})
</script>

<template>
  <div>
    <PageHead pillar="tokens" title="Settings" sub="What is running, and how this looks.">
      <template #icon><SettingsIcon class="h-6 w-6" /></template>
      <template #actions>
        <Tabs v-model="tab" :tabs="tabs" label="Settings" />
      </template>
    </PageHead>

    <Notice v-if="problem" tone="danger" class="mb-4">
      {{ problem }}
    </Notice>

    <template v-if="tab === OVERVIEW">
      <UiCard class="mb-3 p-5">
        <h2 class="mb-3 font-semibold">This machine</h2>
        <dl v-if="status" class="grid gap-x-4 gap-y-2 text-sm sm:grid-cols-[auto_1fr]">
          <dt class="text-muted-foreground">Agent</dt>
          <dd class="font-mono">{{ status.version }}</dd>

          <dt class="text-muted-foreground">Indexer</dt>
          <dd class="break-all font-mono">
            {{ status.core_url }}
            <UiBadge :variant="status.core_up ? 'success' : 'destructive'" class="ml-2">
              {{ status.core_up ? 'answering' : 'not answering' }}
            </UiBadge>
          </dd>

          <dt class="text-muted-foreground">Starts</dt>
          <dd class="font-mono">
            {{ status.core_starts }}
            <span v-if="status.core_starts > 1" class="ml-2 text-xs text-muted-foreground">
              a number that keeps climbing is an indexer that keeps dying
            </span>
          </dd>

          <template v-if="status.last_exit">
            <dt class="text-muted-foreground">Last exit</dt>
            <dd class="break-all font-mono">{{ status.last_exit }}</dd>
          </template>

          <dt class="text-muted-foreground">Folders read</dt>
          <dd class="font-mono">{{ repositories.length }}</dd>

          <dt class="text-muted-foreground">Model</dt>
          <dd class="font-mono">
            {{ model }}
            <UiBadge v-if="status.model && status.keySet" variant="success" class="ml-2">key set</UiBadge>
            <UiBadge v-else-if="status.model" variant="warning" class="ml-2">no key</UiBadge>
          </dd>
        </dl>
        <p class="mt-4 text-xs text-muted-foreground">
          Where the indexer comes from is chosen at install time and kept in
          <code class="font-mono">~/.sourceant/config.json</code>. Nothing on this page has left
          this machine.
        </p>
      </UiCard>

      <UiCard class="p-5">
        <h2 class="mb-1 font-semibold">Appearance</h2>
        <p class="mb-4 text-sm text-muted-foreground">
          Kept in this browser, so it follows the screen rather than the machine.
        </p>
        <UiButton variant="outline" @click="toggleTheme">
          <component :is="isDark ? Sun : Moon" class="mr-2 h-4 w-4" />
          {{ isDark ? 'Light mode' : 'Dark mode' }}
        </UiButton>
      </UiCard>
    </template>

    <McpPanel v-else-if="tab === MCP" />

    <UiCard v-else class="p-5">
      <h2 class="mb-1 font-semibold">{{ tab }}</h2>
      <p v-if="tab === 'Model'" class="mb-4 text-sm text-muted-foreground">
        Reading a repository needs none of this. Anything that proposes or judges rather than
        reads does, and it stays off until you say which model to ask.
      </p>
      <SettingsPanel :key="tab" :group="tab" @saved="onSaved" />

      <p v-if="checking" class="mt-4 flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 class="h-3.5 w-3.5 animate-spin" />
        Asking the provider whether that key can use that model.
      </p>
      <Notice v-else-if="accepts" :tone="accepts.usable ? 'success' : 'danger'" class="mt-4">
        <template v-if="accepts.usable">
          <CircleCheck class="mr-1.5 inline h-3.5 w-3.5" />
          That key can use that model.
        </template>
        <template v-else>{{ accepts.reason }}</template>
      </Notice>
    </UiCard>
  </div>
</template>
