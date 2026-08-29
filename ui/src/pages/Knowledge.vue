<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Field,
  Input,
  ItemCard,
  Modal as UiModal,
  PageHead,
  Select,
  Textarea,
} from '@sourceant/design'
import { onMounted, ref, watch } from 'vue'
import { Lightbulb, Plus, Pencil, Trash2, Check, Sparkles, Loader2, Wand2 } from 'lucide-vue-next'
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
const reading = ref(false)
const asking = ref(false)
const readOff = ref(null)
const hasModel = ref(false)

// Asking costs whatever the model costs, so it is only offered where there is
// one to ask.
async function checkModel() {
  try {
    const settings = await api.settings()
    const name = settings.find((s) => s.key === 'model.name')
    const key = settings.find((s) => s.key === 'model.api_key')
    hasModel.value = !!name?.value && !!key?.is_set
  } catch {
    hasModel.value = false
  }
}

/* What a repository already states about itself: decision records, conventions
 * and constraints in a contributing guide, rules written for whatever works on
 * it. It is knowledge already, just nowhere a tool can reach. Nothing is judged
 * or summarised, so it all arrives proposed. */
async function initialize(useModel = false) {
  const flag = useModel ? asking : reading
  flag.value = true
  readOff.value = null
  try {
    const found = await api.initialize(chosen.value, { useModel })
    readOff.value = found.recorded
    error.value = ''
    await load()
  } catch (caught) {
    error.value = caught.message
  } finally {
    flag.value = false
  }
}

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
  await Promise.all([load(), checkModel()])
})
</script>

<template>
  <div>
    <PageHead pillar="memory" title="Knowledge" sub="The decisions, conventions and constraints behind this code.">
      <template #icon><Lightbulb class="h-6 w-6" /></template>
      <template #actions>
        <Select v-if="repositories.length > 1" v-model="chosen" size="sm" aria-label="Repository">
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>
        <UiButton
          size="sm"
          v-if="repositories.length"
          variant="outline"
          :disabled="reading"
          @click="initialize"
        >
          <Loader2 v-if="reading" class="mr-2 h-4 w-4 animate-spin" />
          <Sparkles v-else class="mr-2 h-4 w-4" />
          {{ reading ? 'Reading…' : 'Read what the repo states' }}
        </UiButton>
        <UiButton
          size="sm"
          v-if="repositories.length && hasModel"
          variant="outline"
          :disabled="asking"
          @click="initialize(true)"
        >
          <Loader2 v-if="asking" class="mr-2 h-4 w-4 animate-spin" />
          <Wand2 v-else class="mr-2 h-4 w-4" />
          {{ asking ? 'Asking…' : 'Ask the model' }}
        </UiButton>
        <UiButton v-if="repositories.length" size="sm" variant="glow" @click="open(null)">
          <Plus class="mr-2 h-4 w-4" />
          Record something
        </UiButton>
      </template>
    </PageHead>

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <p v-if="readOff !== null" class="mb-4 rounded-md border border-primary/30 bg-primary/10 px-4 py-3 text-sm">
      <template v-if="readOff">
        Read {{ readOff }} thing{{ readOff === 1 ? '' : 's' }} this repository already states.
        Nobody has agreed to any of it, so it is all proposed.
      </template>
      <template v-else>
        This repository does not state anything in the places projects usually write these
        down: a decision record, or a conventions section in a contributing guide.
      </template>
    </p>

    <EmptyMachine v-if="repositories.length === 0" />

    <UiCard v-else-if="items.length === 0" class="text-center py-16 px-6">
      <h2 class="text-lg font-semibold mb-1">Nothing recorded yet</h2>
      <p class="text-muted-foreground text-sm mb-4 max-w-lg mx-auto">
        Why a thing is the way it is outlives the code that does it. Write one down and every
        agent reading this repository over MCP gets it too.
      </p>
      <div class="flex items-center justify-center gap-2">
        <UiButton variant="glow" @click="open(null)">
          <Plus class="mr-2 h-4 w-4" />
          Record something
        </UiButton>
        <UiButton variant="outline" :disabled="reading" @click="initialize">
          <Loader2 v-if="reading" class="mr-2 h-4 w-4 animate-spin" />
          <Sparkles v-else class="mr-2 h-4 w-4" />
          Read what the repo states
        </UiButton>
      </div>
    </UiCard>

    <div v-else class="grid gap-3">
      <ItemCard v-for="item in items" :key="item.id" :title="item.id" pillar="memory">
        <template #icon><Lightbulb class="h-5 w-5" /></template>
        <template #badges>
          <UiBadge variant="secondary">{{ item.kind }}</UiBadge>
          <UiBadge v-if="item.status" variant="outline">{{ item.status }}</UiBadge>
        </template>

        <p class="text-sm text-muted-foreground">{{ item.summary }}</p>
        <dl v-if="item.properties?.why" class="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 text-xs">
          <dt class="text-muted-foreground">why</dt>
          <dd class="break-all font-mono">{{ item.properties.why }}</dd>
        </dl>

        <template #actions>
          <UiButton variant="ghost" size="icon" aria-label="Edit" @click="open(item)">
            <Pencil class="h-4 w-4" />
          </UiButton>
          <UiButton variant="ghost" size="icon" aria-label="Remove" @click="forget(item)">
            <Trash2 class="h-4 w-4" />
          </UiButton>
        </template>
      </ItemCard>
    </div>

    <UiModal :open="!!editing" max-width="xl" @close="editing = null">
      <h2 class="text-lg font-semibold mb-4">{{ editing?.fresh ? 'Record something' : 'Edit' }}</h2>
      <div class="space-y-4">
        <Field label="Name" hint="What this will be called, and how it is found again.">
          <Input v-model="draft.id" :readonly="!editing?.fresh" placeholder="retry-limit" />
        </Field>
        <Field label="Kind">
          <Select v-model="draft.kind" class="w-full">
            <option v-for="kind in KINDS" :key="kind">{{ kind }}</option>
          </Select>
        </Field>
        <Field label="What is true">
          <Textarea v-model="draft.summary" placeholder="Charges retry three times, then stop." />
        </Field>
        <Field label="Why" hint="What stops somebody undoing it next year.">
          <Textarea v-model="draft.why" placeholder="The provider rate limits after four." />
        </Field>
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
