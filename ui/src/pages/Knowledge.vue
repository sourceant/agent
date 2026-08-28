<script setup>
import { onMounted, ref, watch } from 'vue'
import { Lightbulb, Plus, Pencil, Trash2, Check } from 'lucide-vue-next'
import UiCard from '~/components/ui/Card.vue'
import UiButton from '~/components/ui/Button.vue'
import UiBadge from '~/components/ui/Badge.vue'
import UiModal from '~/components/ui/Modal.vue'
import PageHead from '~/components/PageHead.vue'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

const KINDS = ['decision', 'convention', 'constraint', 'pattern', 'workaround', 'requirement']

const { repositories, chosen, error, fetchRepositories } = useRepositories()
const items = ref([])
const editing = ref(null)
const draft = ref({ id: '', kind: 'decision', summary: '', why: '' })
const problem = ref('')
const saving = ref(false)

async function load() {
  if (!chosen.value) return
  try {
    const page = await api.knowledge(chosen.value)
    items.value = page.items
    error.value = ''
  } catch (caught) {
    items.value = []
    error.value = caught.message
  }
}

function open(item) {
  editing.value = item ?? { fresh: true }
  draft.value = item
    ? { id: item.id, kind: item.kind, summary: item.summary, why: item.properties?.why ?? '' }
    : { id: '', kind: 'decision', summary: '', why: '' }
  problem.value = ''
}

async function save() {
  if (!draft.value.id.trim() || !draft.value.summary.trim()) {
    problem.value = 'A name and what is true are both needed.'
    return
  }
  saving.value = true
  try {
    await api.recordKnowledge({
      repository: chosen.value,
      id: draft.value.id.trim(),
      kind: draft.value.kind,
      status: editing.value?.status ?? 'accepted',
      summary: draft.value.summary.trim(),
      properties: draft.value.why.trim()
        ? { ...(editing.value?.properties ?? {}), why: draft.value.why.trim() }
        : (editing.value?.properties ?? {}),
    })
    editing.value = null
    await load()
  } catch (caught) {
    problem.value = caught.message
  } finally {
    saving.value = false
  }
}

async function forget(item) {
  if (!confirm(`Forget ${item.id}?`)) return
  try {
    await api.forgetKnowledge(chosen.value, item.id)
  } catch (caught) {
    error.value = caught.message
  }
  await load()
}

watch(chosen, load)
onMounted(async () => {
  await fetchRepositories()
  await load()
})
</script>

<template>
  <div>
    <PageHead
      :icon="Lightbulb"
      pillar="memory"
      title="Knowledge"
      sub="The decisions, conventions and constraints behind this code."
    >
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
        <UiButton v-if="repositories.length" variant="glow" @click="open(null)">
          <Plus class="mr-2 h-4 w-4" />
          Record something
        </UiButton>
      </template>
    </PageHead>

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <EmptyMachine v-if="repositories.length === 0" />

    <UiCard v-else-if="items.length === 0" class="text-center py-16 px-6">
      <h2 class="text-lg font-semibold mb-1">Nothing recorded yet</h2>
      <p class="text-muted-foreground text-sm mb-4 max-w-lg mx-auto">
        Why a thing is the way it is outlives the code that does it. Write one down and every
        agent reading this repository over MCP gets it too.
      </p>
      <UiButton variant="glow" @click="open(null)">
        <Plus class="mr-2 h-4 w-4" />
        Record something
      </UiButton>
    </UiCard>

    <div v-else class="grid gap-3">
      <UiCard v-for="item in items" :key="item.id" class="p-5">
        <div class="flex items-start gap-4">
          <div class="h-11 w-11 shrink-0 rounded-lg bg-pillar-memory/15 text-pillar-memory flex items-center justify-center">
            <Lightbulb class="h-5 w-5" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="flex flex-wrap items-center gap-2 mb-1">
              <h3 class="font-semibold break-all">{{ item.id }}</h3>
              <UiBadge variant="secondary">{{ item.kind }}</UiBadge>
              <UiBadge v-if="item.status" variant="outline">{{ item.status }}</UiBadge>
            </div>
            <p class="text-sm text-muted-foreground">{{ item.summary }}</p>
            <dl v-if="item.properties?.why" class="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
              <dt class="text-muted-foreground">why</dt>
              <dd class="font-mono break-all">{{ item.properties.why }}</dd>
            </dl>
          </div>
          <div class="flex shrink-0 items-center gap-1">
            <UiButton variant="ghost" size="icon" aria-label="Edit" @click="open(item)">
              <Pencil class="h-4 w-4" />
            </UiButton>
            <UiButton variant="ghost" size="icon" aria-label="Remove" @click="forget(item)">
              <Trash2 class="h-4 w-4" />
            </UiButton>
          </div>
        </div>
      </UiCard>
    </div>

    <UiModal :open="!!editing" max-width="xl" @close="editing = null">
      <h2 class="text-lg font-semibold mb-4">{{ editing?.fresh ? 'Record something' : 'Edit' }}</h2>
      <div class="space-y-4">
        <div>
          <label class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1.5 block">Name</label>
          <input
            v-model="draft.id"
            :readonly="!editing?.fresh"
            placeholder="retry-limit"
            class="w-full bg-muted/50 border rounded-md px-3 py-2 text-sm outline-none focus:border-primary/50 text-foreground"
          >
        </div>
        <div>
          <label class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1.5 block">Kind</label>
          <select
            v-model="draft.kind"
            class="w-full bg-muted/50 border rounded-md px-3 py-2 text-sm outline-none focus:border-primary/50 text-foreground"
          >
            <option v-for="kind in KINDS" :key="kind">{{ kind }}</option>
          </select>
        </div>
        <div>
          <label class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1.5 block">What is true</label>
          <textarea
            v-model="draft.summary"
            rows="3"
            placeholder="Charges retry three times, then stop."
            class="w-full bg-muted/50 border rounded-md px-3 py-2 text-sm outline-none focus:border-primary/50 text-foreground"
          />
        </div>
        <div>
          <label class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1.5 block">Why</label>
          <textarea
            v-model="draft.why"
            rows="3"
            placeholder="The provider rate limits after four."
            class="w-full bg-muted/50 border rounded-md px-3 py-2 text-sm outline-none focus:border-primary/50 text-foreground"
          />
        </div>
        <p v-if="problem" class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm">
          {{ problem }}
        </p>
        <div class="flex items-center gap-2 pt-2">
          <UiButton :disabled="saving" @click="save">
            <Check class="mr-1.5 h-3.5 w-3.5" />
            Save
          </UiButton>
          <UiButton variant="outline" @click="editing = null">Cancel</UiButton>
        </div>
      </div>
    </UiModal>
  </div>
</template>
