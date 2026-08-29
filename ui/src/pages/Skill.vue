<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Field,
  Input,
  Markdown,
  PageHead,
  Select,
  Tabs,
  Textarea,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Check, Copy, Loader2, ScrollText, Trash2 } from 'lucide-vue-next'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* Writing a skill down.
 *
 * A skill is a document, frequently a long one, and a box in a dialog is not
 * somewhere anybody writes a document. This is a page: the whole height for the
 * text, the rendering beside it on a wide screen and behind a tab on a narrow
 * one, and nothing modal in the way.
 *
 * It goes in one of two places. A repository, when it is about that project and
 * the team should get it by pulling. This machine, when it is how you work
 * everywhere. A skill kept in a coding agent's own folder opens read-only,
 * because those files are that agent's and are frequently a link into a
 * checkout of their own; saving writes a copy into one of ours instead.
 */

const NEW = 'new'
const REPOSITORY = 'repository'
const MACHINE = 'machine'
const OURS = [REPOSITORY, MACHINE]

const route = useRoute()
const router = useRouter()
const { repositories, chosen, fetchRepositories } = useRepositories()

const id = computed(() => String(route.params.id ?? ''))
const fresh = computed(() => id.value === NEW)

const skill = ref(null)
const draft = ref({ id: '', name: '', description: '', body: '', paths: '', reviews: null })
const scope = ref(REPOSITORY)
const pane = ref('write')
const saving = ref(false)
const saved = ref(false)
const problem = ref('')
const loading = ref(true)

const panes = [
  { id: 'write', label: 'Write' },
  { id: 'preview', label: 'Preview' },
]

// Kept in the skill's own frontmatter, in the map the format sets aside for
// whatever a client wants to record, so a skill carrying it stays portable.
const choices = [
  { id: null, label: 'When it looks relevant' },
  { id: true, label: 'Always' },
  { id: false, label: 'Never' },
]

const saying = computed(() => {
  if (draft.value.reviews === true) return 'Read against every change here.'
  if (draft.value.reviews === false) return 'Left out of reviews entirely.'
  return 'Picked when what it says matches what a change touches.'
})

const scopes = computed(() => [
  { id: REPOSITORY, label: chosen.value || 'This repository' },
  { id: MACHINE, label: 'Global' },
])

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
    draft.value.paths !== (skill.value.paths ?? []).join('\n') ||
    draft.value.reviews !== skill.value.reviews
  )
})

const lines = computed(() => (draft.value.body ? draft.value.body.split('\n').length : 0))

const savedInto = computed(() =>
  scope.value === MACHINE
    ? 'Saved on this machine, so it applies to every repository you work in.'
    : `Saved into ${chosen.value} as a file, so the team gets it by pulling.`,
)

async function load() {
  loading.value = true
  problem.value = ''
  if (fresh.value) {
    skill.value = null
    draft.value = { id: '', name: '', description: '', body: '', paths: '', reviews: null }
    scope.value = route.query.scope === MACHINE ? MACHINE : REPOSITORY
    loading.value = false
    return
  }
  try {
    const found = await api.skill(id.value, chosen.value)
    skill.value = found
    scope.value = found.origin === MACHINE ? MACHINE : REPOSITORY
    draft.value = {
      id: found.id.split('/').pop(),
      name: found.name,
      description: found.description,
      body: found.body ?? '',
      paths: (found.paths ?? []).join('\n'),
      reviews: found.reviews,
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
      scope: scope.value,
      repository: scope.value === REPOSITORY ? chosen.value : '',
      id: draft.value.id || draft.value.name,
      name: draft.value.name || draft.value.id,
      description: draft.value.description,
      body: draft.value.body,
      paths: draft.value.paths.split('\n').map((one) => one.trim()).filter(Boolean),
      reviews: draft.value.reviews,
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
  const from = scope.value === MACHINE ? 'global' : chosen.value
  if (!confirm(`Forget ${draft.value.name}?\n\nThe file is removed from ${from}.`)) return
  try {
    await api.forgetSkill(chosen.value, scope.value, id.value)
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
        <UiButton variant="ghost" size="icon" aria-label="Back to skills" @click="router.push('/skills')">
          <ArrowLeft class="h-4 w-4" />
        </UiButton>
      </template>
      <template #icon><ScrollText class="h-5 w-5" /></template>
      <template #badges>
        <UiBadge v-if="skill" :variant="theirs ? 'outline' : 'success'">
          {{ theirs ? skill.origin : scope === MACHINE ? 'global' : 'this repository' }}
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
          {{ copying ? `Save into ${scope === MACHINE ? 'global' : chosen}` : 'Save' }}
        </UiButton>
      </template>
    </PageHead>

    <p
      v-if="copying"
      class="mb-4 rounded-md border border-primary/30 bg-primary/10 px-4 py-3 text-sm"
    >
      This one belongs to your coding agent, kept wherever it keeps its own, so editing it here
      would change a file outside this repository. Saving writes a copy where you choose below,
      and the copy is then the one that gets used.
    </p>

    <p v-if="problem" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ problem }}
    </p>

    <p v-if="loading" class="py-10 text-center text-sm text-muted-foreground">Reading it.</p>

    <template v-else>
      <UiCard class="mb-3 grid gap-4 p-5 lg:grid-cols-3">
        <Field
          label="Name"
          for="skill-id"
          hint="Lower case words joined by hyphens. It names the folder the skill is saved in."
        >
          <Input
            id="skill-id"
            v-model="draft.id"
            :readonly="!!skill && !theirs"
            placeholder="retry-limit"
          />
        </Field>
        <Field
          label="Where it belongs"
          for="skill-scope"
          :hint="scope === MACHINE
            ? 'Kept on this machine, so it applies to every repository you work in.'
            : 'Kept in the repository as a file, so the team gets it by pulling.'"
        >
          <Select
            id="skill-scope"
            v-model="scope"
            :disabled="!!skill && !theirs"
            class="w-full"
          >
            <option v-for="one in scopes" :key="one.id" :value="one.id">{{ one.label }}</option>
          </Select>
        </Field>
        <Field
          label="When it applies"
          for="skill-description"
          hint="One sentence. It decides whether a change gets read against this skill."
        >
          <Input
            id="skill-description"
            v-model="draft.description"
            placeholder="Use when a change adds or edits a database migration."
          />
        </Field>
      </UiCard>

      <UiCard class="mb-3 grid gap-4 p-5 lg:grid-cols-2">
        <Field
          label="Files it is about"
          for="skill-paths"
          hint="Globs, one to a line. Named here, a change is read against this only when it
                touches one of them, whatever the wording says. Left empty, the wording decides."
        >
          <Textarea
            id="skill-paths"
            v-model="draft.paths"
            rows="3"
            class="font-mono"
            placeholder="db/migrations/**&#10;**/*.sql"
          />
        </Field>

        <Field label="Use in reviews" hint="Not everything you teach an agent is about judging a change.">
          <div class="flex flex-wrap gap-1.5">
            <UiButton
              v-for="one in choices"
              :key="String(one.id)"
              size="sm"
              :variant="draft.reviews === one.id ? 'default' : 'outline'"
              @click="draft.reviews = one.id"
            >
              {{ one.label }}
            </UiButton>
          </div>
          <p class="mt-2 text-xs text-muted-foreground">{{ saying }}</p>
        </Field>
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
