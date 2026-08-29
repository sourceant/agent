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
  Tabs,
  Textarea,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { BookOpenCheck, Copy, Eye, Pencil, Plus, ScrollText, Search, Trash2 } from 'lucide-vue-next'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* The rules a team has for its own code.
 *
 * Some of them are already written down: people have been teaching their coding
 * agents how work is done here for a while, in a folder the agent reads, and
 * nothing else in this product could see it. Those are read and never written,
 * because they are somebody's own files and are frequently a link into a
 * checkout of their own.
 *
 * The rest are in somebody's head. Those get written here, into the repository
 * the rule is about, where the rest of the team gets them by pulling.
 */

const MINE = 'repository'

const { repositories, chosen, error, fetchRepositories } = useRepositories()
const skills = ref([])
const term = ref('')
const where = ref('all')
const reading = ref(null)
const editing = ref(null)
const draft = ref({ id: '', name: '', description: '', body: '' })
const saving = ref(false)
const problem = ref('')

const wheres = [
  { id: 'all', label: 'All' },
  { id: MINE, label: "This repository's" },
  { id: 'machine', label: 'Yours' },
]

const shown = computed(() => {
  const wanted = term.value.trim().toLowerCase()
  return skills.value.filter((skill) => {
    if (where.value === MINE && skill.origin !== MINE) return false
    if (where.value === 'machine' && skill.origin === MINE) return false
    if (!wanted) return true
    return (
      skill.name.toLowerCase().includes(wanted) ||
      skill.description.toLowerCase().includes(wanted)
    )
  })
})

const ours = (skill) => skill.origin === MINE

async function load() {
  try {
    const page = await api.skills(chosen.value)
    skills.value = page.skills
    error.value = ''
  } catch (caught) {
    skills.value = []
    error.value = caught.message
  }
}

async function open(skill) {
  try {
    reading.value = await api.skill(skill.id, chosen.value)
  } catch (caught) {
    error.value = caught.message
  }
}

function compose(skill) {
  problem.value = ''
  if (!skill) {
    editing.value = { fresh: true }
    draft.value = { id: '', name: '', description: '', body: '' }
    return
  }
  editing.value = { fresh: !ours(skill), from: skill }
  draft.value = {
    // Copying somebody's own rule into this repository gives it a new home and
    // the same name, so the repository's copy is the one that gets used.
    id: skill.id.split('/').pop(),
    name: skill.name,
    description: skill.description,
    body: skill.body ?? '',
  }
}

async function edit(skill) {
  try {
    compose(await api.skill(skill.id, chosen.value))
  } catch (caught) {
    error.value = caught.message
  }
}

async function save() {
  saving.value = true
  problem.value = ''
  try {
    await api.recordSkill({ repository: chosen.value, ...draft.value })
    editing.value = null
    reading.value = null
    await load()
  } catch (caught) {
    problem.value = caught.message
  } finally {
    saving.value = false
  }
}

async function forget(skill) {
  if (!confirm(`Forget ${skill.name}?\n\nThe file is removed from this repository.`)) return
  try {
    await api.forgetSkill(chosen.value, skill.id)
    await load()
  } catch (caught) {
    error.value = caught.message
  }
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
      pillar="review"
      title="Skills"
      sub="The rules your work is read against, yours and this repository's."
    >
      <template #icon><BookOpenCheck class="h-6 w-6" /></template>
      <template #actions>
        <Select v-if="repositories.length > 1" v-model="chosen" aria-label="Repository">
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="term" placeholder="Find a rule" class="w-48 pl-8" aria-label="Find a rule" />
        </div>
        <UiButton v-if="repositories.length" variant="glow" @click="compose(null)">
          <Plus class="mr-2 h-4 w-4" />
          Write one down
        </UiButton>
      </template>
    </PageHead>

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <EmptyMachine v-if="repositories.length === 0" />

    <template v-else>
      <Tabs v-model="where" :tabs="wheres" label="Whose rules" class="mb-4" />

      <div v-if="shown.length" class="space-y-3">
        <ItemCard
          v-for="skill in shown"
          :key="skill.id"
          :title="skill.name"
          :subtitle="skill.path"
          pillar="review"
        >
          <template #icon><ScrollText class="h-5 w-5" /></template>
          <template #badges>
            <UiBadge :variant="ours(skill) ? 'success' : 'outline'">
              {{ ours(skill) ? 'this repository' : skill.origin }}
            </UiBadge>
          </template>
          <p class="text-sm text-muted-foreground">{{ skill.description }}</p>
          <template #actions>
            <UiButton variant="ghost" size="icon" :aria-label="`Read ${skill.name}`" @click="open(skill)">
              <Eye class="h-4 w-4" />
            </UiButton>
            <UiButton
              variant="ghost"
              size="icon"
              :aria-label="ours(skill) ? `Edit ${skill.name}` : `Copy ${skill.name} into this repository`"
              @click="edit(skill)"
            >
              <component :is="ours(skill) ? Pencil : Copy" class="h-4 w-4" />
            </UiButton>
            <UiButton
              v-if="ours(skill)"
              variant="ghost"
              size="icon"
              :aria-label="`Forget ${skill.name}`"
              @click="forget(skill)"
            >
              <Trash2 class="h-4 w-4" />
            </UiButton>
          </template>
        </ItemCard>
      </div>

      <UiCard v-else class="p-10 text-center">
        <ScrollText class="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
        <p class="font-medium">Nothing written down here yet.</p>
        <p class="mx-auto mt-1 max-w-lg text-sm text-muted-foreground">
          A rule says when it applies and what it requires. This repository's live in
          <code class="font-mono">.sourceant/skills</code>, so the rest of the team gets them by
          pulling. Anything you already taught Claude or Codex is read from your own folders.
        </p>
        <UiButton class="mt-4" variant="glow" @click="compose(null)">
          <Plus class="mr-2 h-4 w-4" />
          Write one down
        </UiButton>
      </UiCard>
    </template>

    <UiModal :open="!!reading" max-width="2xl" @close="reading = null">
      <div v-if="reading">
        <h2 class="pr-8 text-lg font-semibold">{{ reading.name }}</h2>
        <p class="mt-1 text-sm text-muted-foreground">{{ reading.description }}</p>
        <p class="mt-1 break-all font-mono text-xs text-muted-foreground">{{ reading.path }}</p>
        <pre class="mt-4 max-h-[50vh] overflow-auto whitespace-pre-wrap rounded-md bg-muted/50 p-4 text-sm">{{ reading.body }}</pre>
      </div>
    </UiModal>

    <UiModal :open="!!editing" max-width="2xl" @close="editing = null">
      <div v-if="editing" class="space-y-4">
        <div>
          <h2 class="pr-8 text-lg font-semibold">
            {{ editing.from ? (editing.fresh ? 'Copy it into this repository' : 'Edit this rule') : 'Write a rule down' }}
          </h2>
          <p class="mt-1 text-sm text-muted-foreground">
            <template v-if="editing.fresh && editing.from">
              This one is yours, kept wherever your coding agent keeps it, so it is not edited
              here. Saved into <span class="font-mono">{{ chosen }}</span> it becomes the
              team's, and the copy here is the one that gets used.
            </template>
            <template v-else>
              It is saved into <span class="font-mono">{{ chosen }}</span> as a file, so the rest
              of the team gets it by pulling and your coding agent reads it too.
            </template>
          </p>
        </div>

        <Field
          label="Name"
          for="skill-id"
          hint="Lower case words joined by hyphens. It names the folder the rule is saved in."
        >
          <Input id="skill-id" v-model="draft.id" :readonly="!!editing.from && !editing.fresh" placeholder="retry-limit" />
        </Field>

        <Field
          label="When it applies"
          for="skill-description"
          hint="One sentence. This is what decides whether a change gets read against this rule."
        >
          <Textarea
            id="skill-description"
            v-model="draft.description"
            placeholder="Use when a change adds or edits a database migration."
          />
        </Field>

        <Field label="What it requires" for="skill-body" hint="What somebody has to do, or not do.">
          <Textarea
            id="skill-body"
            v-model="draft.body"
            rows="8"
            placeholder="Never edit a migration that has already run. Add a new one instead."
          />
        </Field>

        <p v-if="problem" class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm">
          {{ problem }}
        </p>

        <div class="flex justify-end gap-2">
          <UiButton variant="ghost" @click="editing = null">Cancel</UiButton>
          <UiButton :disabled="saving" @click="save">
            {{ saving ? 'Saving…' : 'Save' }}
          </UiButton>
        </div>
      </div>
    </UiModal>
  </div>
</template>
