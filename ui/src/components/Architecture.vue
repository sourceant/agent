<script setup>
import { computed, ref, watch } from 'vue'
import { Button, Notice, Select } from '@sourceant/design'
import { api } from '~/api'

const props = defineProps({ repository: { type: String, required: true } })
const depth = ref(1)
const snapshot = ref(null)
const loading = ref(false)
const error = ref('')
const picked = ref('')
const baseline = ref(null)
const difference = ref(null)
const comparing = ref(false)
const visibleLimit = ref(100)
let generation = 0

const names = computed(() => Object.fromEntries((snapshot.value?.components ?? []).map(part => [part.id, part.name])))
const visibleComponents = computed(() => (snapshot.value?.components ?? []).slice(0, visibleLimit.value))
const connections = computed(() => (snapshot.value?.relationships ?? []).filter(edge => edge.source === picked.value || edge.target === picked.value))
const incomplete = computed(() => snapshot.value && (snapshot.value.coverage.truncated || snapshot.value.coverage.unplaced_nodes || snapshot.value.coverage.unresolved_edges))

async function load() {
  const current = ++generation
  loading.value = true
  error.value = ''
  difference.value = null
  try {
    const result = await api.architecture(props.repository, depth.value)
    if (current !== generation) return
    snapshot.value = result
    if (!result.components.some(part => part.id === picked.value)) picked.value = result.components[0]?.id ?? ''
  } catch (caught) {
    if (current === generation) {
      snapshot.value = null
      error.value = caught.message
    }
  } finally {
    if (current === generation) loading.value = false
  }
}

async function compare() {
  const current = generation
  comparing.value = true
  error.value = ''
  try {
    const result = await api.compareArchitecture(baseline.value)
    if (current === generation) difference.value = result
  } catch (caught) {
    if (current === generation) error.value = caught.message
  } finally {
    comparing.value = false
  }
}

function exportSnapshot() {
  const blob = new Blob([JSON.stringify(snapshot.value, null, 2) + '\n'], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'architecture.json'
  link.click()
  URL.revokeObjectURL(url)
}

watch(() => [props.repository, depth.value], () => {
  baseline.value = null
  visibleLimit.value = 100
  snapshot.value = null
  load()
}, { immediate: true })
</script>

<template>
  <section class="space-y-4" aria-label="Code components">
    <div class="flex flex-wrap items-center gap-2">
      <label class="flex items-center gap-2 text-sm">
        Directory depth
        <Select v-model.number="depth" aria-label="Directory depth">
          <option v-for="level in 4" :key="level" :value="level">{{ level }}</option>
        </Select>
      </label>
      <Button variant="ghost" :disabled="loading || comparing" @click="load">Read current index</Button>
      <template v-if="snapshot">
        <Button variant="ghost" :disabled="loading" @click="exportSnapshot">Export JSON</Button>
        <Button variant="ghost" :disabled="loading || incomplete || comparing" @click="baseline = snapshot; difference = null">Use as baseline</Button>
        <Button v-if="baseline" :disabled="loading || comparing" @click="compare">{{ comparing ? 'Comparing…' : 'Compare with baseline' }}</Button>
      </template>
    </div>
    <p class="text-sm text-muted-foreground">Components are grouped by directory in the current index. Connections come from code relationships, not a model.</p>
    <Notice v-if="error" tone="danger">{{ error }}</Notice>
    <p v-if="loading" role="status" class="text-sm text-muted-foreground">Reading indexed components…</p>
    <template v-else-if="snapshot">
      <Notice v-if="incomplete" tone="warning">This reading is incomplete. Missing connections do not prove independence, and this snapshot cannot be used for comparison.</Notice>
      <div v-if="!snapshot.components.length" class="rounded-lg border bg-card p-5">
        <p>No components have been indexed yet.</p>
        <p class="mt-1 text-sm text-muted-foreground">Index this repository from Repositories, then read it again.</p>
        <Button as="a" href="/repositories" variant="ghost" class="mt-3">Repositories</Button>
      </div>
      <template v-else>
        <p class="text-sm text-muted-foreground">{{ snapshot.coverage.files }} indexed files · {{ snapshot.components.length }} components · {{ snapshot.relationships.length }} dependencies</p>
        <div class="grid gap-4 lg:grid-cols-2">
          <div class="max-h-[32rem] overflow-auto rounded-lg border bg-card">
            <table class="w-full text-left text-sm">
              <caption class="sr-only">Components in the current index</caption>
              <thead><tr class="border-b text-muted-foreground"><th class="px-3 py-2">Component</th><th class="px-3 py-2">Files</th><th class="px-3 py-2">In / out</th></tr></thead>
              <tbody>
                <tr v-for="part in visibleComponents" :key="part.id" class="border-b last:border-0" :class="picked === part.id ? 'bg-primary/10' : ''">
                  <td class="px-3 py-2"><button type="button" class="text-left font-mono underline decoration-transparent hover:decoration-current" :aria-pressed="picked === part.id" @click="picked = part.id">{{ part.name }}</button></td>
                  <td class="px-3 py-2 tabular-nums">{{ part.files }}</td>
                  <td class="px-3 py-2 tabular-nums">{{ part.incoming }} / {{ part.outgoing }}</td>
                </tr>
              </tbody>
            </table>
            <Button v-if="visibleLimit < snapshot.components.length" variant="ghost" class="m-3" @click="visibleLimit += 100">Show 100 more components</Button>
          </div>
          <div class="max-h-[32rem] space-y-3 overflow-auto rounded-lg border bg-card p-4">
            <h3 class="font-medium">Connections for {{ names[picked] }}</h3>
            <p v-if="!connections.length" class="text-sm text-muted-foreground">No connections to other components were found in this index reading.</p>
            <div v-for="edge in connections" :key="`${edge.source}:${edge.target}:${edge.type}`" class="border-t pt-3 text-sm">
              <p>{{ names[edge.source] }} → {{ names[edge.target] }}</p>
              <p class="text-xs text-muted-foreground">{{ edge.type }} · {{ edge.count }} references</p>
              <ul class="mt-2 space-y-1">
                <li v-for="(item, position) in edge.evidence" :key="position" class="break-all font-mono text-xs text-muted-foreground">{{ item.source.path }} → {{ item.target.path }} ({{ item.origin }})</li>
              </ul>
            </div>
          </div>
        </div>
      </template>
      <div v-if="difference" class="rounded-lg border bg-card p-4" aria-live="polite">
        <h3 class="font-medium">Changes since the baseline</h3>
        <p class="mt-1 text-sm text-muted-foreground">{{ difference.components.length }} changed components · {{ difference.relationships.length }} changed dependencies</p>
        <ul class="mt-2 space-y-1 text-sm"><li v-for="part in difference.components" :key="part.id">{{ part.name }}: {{ part.status }}</li></ul>
        <ul class="mt-2 space-y-1 text-sm"><li v-for="edge in difference.relationships" :key="`${edge.source}:${edge.target}:${edge.type}`">{{ edge.source_name }} → {{ edge.target_name }}: {{ edge.type }} {{ edge.status }}</li></ul>
      </div>
    </template>
  </section>
</template>
