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
const { repositories, chosen, fetchRepositories } = useRepositories()

const id = computed(() => String(route.params.id ?? ''))
const fresh = computed(() => id.value === NEW)

const skill = ref(null)
const draft = ref({ id: '', name: '', description: '', body: '', paths: [], reviews: null })
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
    draft.value.reviews !== skill.value.reviews
  )
})

const lines = computed(() => (draft.value.body ? draft.value.body.split('\n').length : 0))

async function load() {
  loading.value = true
  problem.value = ''
  if (fresh.value) {
    skill.value = null
    draft.value = { id: '', name: '', description: '', body: '', paths: [], reviews: null }
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
      scope: forEverything.value ? GLOBAL : REPOSITORY,
      repository: forEverything.value ? '' : belongsTo.value,
      id: draft.value.id || draft.value.name,
      name: draft.value.name || draft.value.id,
      description: draft.value.description,
      body: draft.value.body,
      paths: draft.value.paths,
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
        <UiButton variant="ghost" size="icon" aria-label="Back to skills" @click="router.push('/skills')">
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
          label="Used for"
          for="skill-belongs"
          :hint="forEverything
            ? 'Read for every repository you work in.'
            : 'Read only when reviewing that repository.'"
        >
          <Select
            id="skill-belongs"
            v-model="belongsTo"
            :disabled="!!skill && !theirs"
            class="w-full"
          >
            <option v-for="one in belongings" :key="one.id" :value="one.id">{{ one.label }}</option>
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
          <ListInput
            v-model="draft.paths"
            mono
            size="sm"
            noun="a pattern"
            placeholder="db/migrations/**"
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
