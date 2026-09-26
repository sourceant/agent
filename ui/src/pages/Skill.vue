<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Field,
  Input,
  ListInput,
  Markdown,
  Notice,
  PageHead,
  Select,
  Tabs,
  Textarea,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Check, Copy, Loader2, ScrollText, Trash2 } from 'lucide-vue-next'
import { useUp } from '~/composables/useUp'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* Writing a skill down.
 *
 * A skill is a document, frequently a long one, and a box in a dialog is not
 * somewhere anybody writes a document. This is a page: the whole height for the
 * text, the rendering beside it on a wide screen and behind a tab on a narrow
 * one, and nothing modal in the way.
 *
 * Who it is for is one control naming the destination, rather than a scope
 * that reads the repository somebody happened to be filtering by. That is how
 * a skill ends up filed against a project nobody meant, and the person who
 * filed it has no way of telling from the screen.
 *
 * Nothing is written into anybody's repository. A folder appearing in a
 * checkout because a tool was opened turns up in their `git status` and in a
 * review nobody asked for. A skill kept in a coding agent's own folder, or
 * committed by a team, opens read-only for the same reason: those files are
 * theirs. Saving keeps a copy of ours instead.
 */

const NEW = 'new'
const REPOSITORY = 'repository'
const GLOBAL = 'global'
const OURS = [REPOSITORY, GLOBAL]

const route = useRoute()
const router = useRouter()
const up = useUp()
const { repositories, chosen, fetchRepositories } = useRepositories()

const id = computed(() => String(route.params.id ?? ''))
const fresh = computed(() => id.value === NEW)

const skill = ref(null)
const draft = ref({
  id: '',
  name: '',
  description: '',
  body: '',
  paths: [],
  // Which uses it is kept out of. What it is used for, whether this product may
  // pick it itself, and how it is read are three separate questions.
  applications: {},
  automatic: true,
  type: 'guidance',
})
// Either GLOBAL, or the name of the repository it is for. One value, so
// there is no second place for the destination to come from.
const belongsTo = ref(GLOBAL)
const pane = ref('write')
const saving = ref(false)
const saved = ref(false)
const problem = ref('')
const loading = ref(true)

const panes = [
  { id: 'write', label: 'Write' },
  { id: 'preview', label: 'Preview' },
]

const uses = ref([])

// On until said otherwise, so a skill nobody has narrowed is available to
// everything this product does.
const on = (use) => draft.value.applications?.[use] !== false

function toggle(use) {
  const applications = { ...draft.value.applications }
  if (on(use)) applications[use] = false
  else delete applications[use]
  draft.value.applications = applications
}

// Both kept in the skill's own file, in the fields the format sets aside for
// them, so a skill carrying either stays portable.
const kinds = [
  { id: 'guidance', label: 'Guidance' },
  { id: 'review-pass', label: 'Review pass' },
  { id: 'initialization-pass', label: 'Initialization pass' },
]

const savedInto = computed(() =>
  forEverything.value
    ? 'Kept on this machine and read for every repository you work in.'
    : `Kept on this machine and read for ${belongsTo.value}. Nothing is written into the checkout.`,
)

const belongings = computed(() => [
  { id: GLOBAL, label: 'Everywhere' },
  ...repositories.value.map((one) => ({ id: one.name, label: one.name })),
])

const forEverything = computed(() => belongsTo.value === GLOBAL)

// A skill in a coding agent's own folder is read, never written. Saving makes a
// copy of ours, which is then the one that gets used.
const theirs = computed(() => !!skill.value && !OURS.includes(skill.value.origin))
const copying = computed(() => theirs.value)

const changed = computed(() => {
  if (fresh.value) return !!(draft.value.id || draft.value.description || draft.value.body)
  if (!skill.value) return false
  return (
    draft.value.name !== skill.value.name ||
    draft.value.description !== skill.value.description ||
    draft.value.body !== (skill.value.body ?? '') ||
    draft.value.paths.join('\n') !== (skill.value.paths ?? []).join('\n') ||
    draft.value.type !== (skill.value.type ?? 'guidance') ||
    draft.value.automatic !== (skill.value.automatic ?? true) ||
    JSON.stringify(draft.value.applications) !== JSON.stringify(skill.value.applications ?? {})
  )
})

const lines = computed(() => (draft.value.body ? draft.value.body.split('\n').length : 0))

async function load() {
  loading.value = true
  problem.value = ''
  if (!uses.value.length) uses.value = await api.uses().catch(() => [])
  if (fresh.value) {
    skill.value = null
    draft.value = {
      id: '',
      name: '',
      description: '',
      body: '',
      paths: [],
      applications: {},
      automatic: true,
      type: 'guidance',
    }
    // What the list was showing, so writing one for the project being
    // looked at takes no thought, and is still named on the screen.
    belongsTo.value = route.query.for || chosen.value || GLOBAL
    loading.value = false
    return
  }
  try {
    const found = await api.skill(id.value, chosen.value)
    skill.value = found
    belongsTo.value =
      found.origin === REPOSITORY ? chosen.value : found.origin === GLOBAL ? GLOBAL : chosen.value || GLOBAL
    draft.value = {
      id: found.id.split('/').pop(),
      name: found.name,
      description: found.description,
      body: found.body ?? '',
      paths: [...(found.paths ?? [])],
      applications: { ...(found.applications ?? {}) },
      automatic: found.automatic ?? true,
      type: found.type ?? 'guidance',
    }
  } catch (caught) {
    problem.value = caught.message
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  saved.value = false
  problem.value = ''
  try {
    const written = await api.recordSkill({
      scope: forEverything.value ? GLOBAL : REPOSITORY,
      repository: forEverything.value ? '' : belongsTo.value,
      id: draft.value.id || draft.value.name,
      name: draft.value.name || draft.value.id,
      description: draft.value.description,
      body: draft.value.body,
      paths: draft.value.paths,
      applications: draft.value.applications,
      automatic: draft.value.automatic,
      type: draft.value.type,
    })
    saved.value = true
    if (fresh.value || written.id !== id.value) {
      router.replace(`/skills/${written.id}`)
    } else {
      await load()
    }
  } catch (caught) {
    problem.value = caught.message
  } finally {
    saving.value = false
  }
}

async function forget() {
  const from = forEverything.value ? 'everywhere' : belongsTo.value
  if (!confirm(`Forget ${draft.value.name}?\n\nIt stops being read for ${from}.`)) return
  try {
    await api.forgetSkill(
      forEverything.value ? '' : belongsTo.value,
      forEverything.value ? GLOBAL : REPOSITORY,
      id.value,
    )
    router.push('/skills')
  } catch (caught) {
    problem.value = caught.message
  }
}

watch(id, load)
onMounted(async () => {
  await fetchRepositories()
  await load()
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <PageHead
      pillar="review"
      :title="fresh ? 'A new skill' : draft.name || id"
      :sub="skill?.path || savedInto"
      :mono="!!skill?.path"
    >
      <template #back>
        <UiButton variant="ghost" size="icon" aria-label="Back to skills" @click="up('/skills')">
          <ArrowLeft class="h-4 w-4" />
        </UiButton>
      </template>
      <template #icon><ScrollText class="h-5 w-5" /></template>
      <template #badges>
        <UiBadge v-if="skill" :variant="theirs ? 'outline' : 'success'">
          {{ theirs ? skill.origin : forEverything ? 'everywhere' : belongsTo }}
        </UiBadge>
      </template>
      <template #actions>
        <span v-if="saved && !changed" class="text-xs text-success">Saved.</span>
        <UiButton
          v-if="skill && !theirs"
          variant="ghost"
          size="icon"
          aria-label="Forget this skill"
          @click="forget"
        >
          <Trash2 class="h-4 w-4" />
        </UiButton>
        <UiButton size="sm" :disabled="saving || !changed" @click="save">
          <Loader2 v-if="saving" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
          <component :is="copying ? Copy : Check" v-else class="mr-1.5 h-3.5 w-3.5" />
          {{ copying ? 'Save your own copy' : 'Save' }}
        </UiButton>
      </template>
    </PageHead>

    <Notice v-if="copying" tone="info" class="mb-4">
      This one is not ours to change: it belongs to your coding agent, or your team committed
      it to the repository. Saving keeps a copy of our own, for whatever you choose below, and
      the copy is then the one that gets used.
    </Notice>

    <Notice v-if="problem" tone="danger" class="mb-4">
      {{ problem }}
    </Notice>

    <p v-if="loading" class="py-10 text-center text-sm text-muted-foreground">Reading it.</p>

    <template v-else>
      <UiCard class="mb-3 p-5">
        <div class="grid gap-4 lg:grid-cols-3">
          <Field label="Name" for="skill-id" hint="Lower case words joined by hyphens.">
            <Input
              id="skill-id"
              v-model="draft.id"
              :readonly="!!skill && !theirs"
              placeholder="retry-limit"
            />
          </Field>
          <Field
            class="lg:col-span-2"
            label="Description"
            for="skill-description"
            hint="Matched against a change when SourceAnt picks skills itself."
          >
            <Input
              id="skill-description"
              v-model="draft.description"
              placeholder="Use when a change adds or edits a database migration."
            />
          </Field>
        </div>
        <Field class="mt-4" label="Files" for="skill-paths" hint="Globs, one a line. Named, only these count.">
          <ListInput
            v-model="draft.paths"
            mono
            size="sm"
            noun="a pattern"
            placeholder="db/migrations/**"
          />
        </Field>
      </UiCard>

      <UiCard class="mb-3 p-5">
        <div class="grid gap-4 lg:grid-cols-3">
          <Field class="lg:col-span-2" label="Skill uses">
            <div class="flex flex-wrap gap-1.5">
              <button
                v-for="use in uses"
                :key="use.id"
                type="button"
                :aria-pressed="on(use.id)"
                :class="[
                  'rounded-full border px-2.5 py-0.5 text-xs transition-colors',
                  on(use.id)
                    ? 'border-primary/30 bg-primary/10 text-foreground'
                    : 'border-border text-muted-foreground line-through',
                  'hover:border-primary/50',
                ]"
                @click="toggle(use.id)"
              >
                {{ use.label }}
              </button>
            </div>
            <label class="mt-3 flex items-center gap-2 text-sm">
              <input v-model="draft.automatic" type="checkbox" class="h-3.5 w-3.5 rounded border">
              SourceAnt may pick it itself
            </label>
          </Field>
          <div class="grid gap-4">
            <Field label="Read as">
              <Select v-model="draft.type" class="w-full">
                <option v-for="one in kinds" :key="one.id" :value="one.id">{{ one.label }}</option>
              </Select>
            </Field>
            <Field label="Kept for" for="skill-belongs">
              <Select
                id="skill-belongs"
                v-model="belongsTo"
                :disabled="!!skill && !theirs"
                class="w-full"
              >
                <option v-for="one in belongings" :key="one.id" :value="one.id">{{ one.label }}</option>
              </Select>
            </Field>
          </div>
        </div>
      </UiCard>

      <div class="mb-2 flex items-center justify-between gap-3">
        <Tabs v-model="pane" :tabs="panes" label="Write or preview" class="lg:hidden" />
        <p class="hidden text-xs uppercase tracking-wider text-muted-foreground lg:block">
          What it says
        </p>
        <p class="text-xs text-muted-foreground">
          {{ lines }} line{{ lines === 1 ? '' : 's' }} · markdown
        </p>
      </div>

      <!-- Side by side where there is room for both, one at a time where there
           is not. Both are always rendered so switching keeps the scroll. -->
      <div class="grid min-h-0 flex-1 gap-3 lg:grid-cols-2">
        <Textarea
          v-model="draft.body"
          class="h-full min-h-[24rem] resize-none font-mono leading-relaxed"
          :class="pane === 'write' ? '' : 'hidden lg:block'"
          placeholder="Never edit a migration that has already run. Add a new one instead."
          aria-label="What the skill says"
        />
        <UiCard
          class="h-full min-h-[24rem] overflow-auto p-5"
          :class="pane === 'preview' ? '' : 'hidden lg:block'"
        >
          <Markdown v-if="draft.body" :source="draft.body" />
          <p v-else class="text-sm text-muted-foreground">
            Nothing written yet. What appears here is what a person reads, and what a model is
            given when your work is checked against this skill.
          </p>
        </UiCard>
      </div>
    </template>
  </div>
</template>
