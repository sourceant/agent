<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Field,
  Input,
  Markdown,
  PageHead,
  Tabs,
  Textarea,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Check, Copy, Loader2, ScrollText, Trash2 } from 'lucide-vue-next'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* Writing a rule down.
 *
 * A rule is a document, frequently a long one, and a box in a dialog is not
 * somewhere anybody writes a document. This is a page: the whole height for the
 * text, the rendering beside it on a wide screen and behind a tab on a narrow
 * one, and nothing modal in the way.
 *
 * A rule kept in somebody's own agent folder opens read-only, because those
 * files are theirs and are frequently a link into a checkout of their own.
 * Saving it here saves a copy into the repository instead.
 */

const NEW = 'new'
const MINE = 'repository'

const route = useRoute()
const router = useRouter()
const { repositories, chosen, fetchRepositories } = useRepositories()

const id = computed(() => String(route.params.id ?? ''))
const fresh = computed(() => id.value === NEW)

const skill = ref(null)
const draft = ref({ id: '', name: '', description: '', body: '' })
const pane = ref('write')
const saving = ref(false)
const saved = ref(false)
const problem = ref('')
const loading = ref(true)

const panes = [
  { id: 'write', label: 'Write' },
  { id: 'preview', label: 'Preview' },
]

// Somebody else's rule is read, never written. Saving makes the repository a
// copy, which is then the one that gets used.
const theirs = computed(() => !!skill.value && skill.value.origin !== MINE)
const copying = computed(() => theirs.value)

const changed = computed(() => {
  if (fresh.value) return !!(draft.value.id || draft.value.description || draft.value.body)
  if (!skill.value) return false
  return (
    draft.value.name !== skill.value.name ||
    draft.value.description !== skill.value.description ||
    draft.value.body !== (skill.value.body ?? '')
  )
})

const lines = computed(() => (draft.value.body ? draft.value.body.split('\n').length : 0))

async function load() {
  loading.value = true
  problem.value = ''
  if (fresh.value) {
    skill.value = null
    draft.value = { id: '', name: '', description: '', body: '' }
    loading.value = false
    return
  }
  try {
    const found = await api.skill(id.value, chosen.value)
    skill.value = found
    draft.value = {
      id: found.id.split('/').pop(),
      name: found.name,
      description: found.description,
      body: found.body ?? '',
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
      repository: chosen.value,
      id: draft.value.id || draft.value.name,
      name: draft.value.name || draft.value.id,
      description: draft.value.description,
      body: draft.value.body,
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
  if (!confirm(`Forget ${draft.value.name}?\n\nThe file is removed from this repository.`)) return
  try {
    await api.forgetSkill(chosen.value, id.value)
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
      :title="fresh ? 'Write a rule down' : draft.name || id"
      :sub="skill?.path || `Saved into ${chosen} as a file, so the team gets it by pulling.`"
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
          {{ theirs ? skill.origin : 'this repository' }}
        </UiBadge>
      </template>
      <template #actions>
        <span v-if="saved && !changed" class="text-xs text-success">Saved.</span>
        <UiButton
          v-if="skill && !theirs"
          variant="ghost"
          size="icon"
          aria-label="Forget this rule"
          @click="forget"
        >
          <Trash2 class="h-4 w-4" />
        </UiButton>
        <UiButton size="sm" :disabled="saving || !changed" @click="save">
          <Loader2 v-if="saving" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
          <component :is="copying ? Copy : Check" v-else class="mr-1.5 h-3.5 w-3.5" />
          {{ copying ? 'Save into this repository' : 'Save' }}
        </UiButton>
      </template>
    </PageHead>

    <p
      v-if="copying"
      class="mb-4 rounded-md border border-primary/30 bg-primary/10 px-4 py-3 text-sm"
    >
      This one is yours, kept wherever your coding agent keeps it, so editing it here would
      change a file outside this repository. Saving writes a copy into
      <span class="font-mono">{{ chosen }}</span>, and the copy is then the one that gets used.
    </p>

    <p v-if="problem" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ problem }}
    </p>

    <p v-if="loading" class="py-10 text-center text-sm text-muted-foreground">Reading it.</p>

    <template v-else>
      <UiCard class="mb-3 grid gap-4 p-5 lg:grid-cols-2">
        <Field
          label="Name"
          for="skill-id"
          hint="Lower case words joined by hyphens. It names the folder the rule is saved in."
        >
          <Input
            id="skill-id"
            v-model="draft.id"
            :readonly="!!skill && !theirs"
            placeholder="retry-limit"
          />
        </Field>
        <Field
          label="When it applies"
          for="skill-description"
          hint="One sentence. This is what decides whether a change gets read against this rule."
        >
          <Input
            id="skill-description"
            v-model="draft.description"
            placeholder="Use when a change adds or edits a database migration."
          />
        </Field>
      </UiCard>

      <div class="mb-2 flex items-center justify-between gap-3">
        <Tabs v-model="pane" :tabs="panes" label="Write or preview" class="lg:hidden" />
        <p class="hidden text-xs uppercase tracking-wider text-muted-foreground lg:block">
          What it requires
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
          aria-label="What the rule requires"
        />
        <UiCard
          class="h-full min-h-[24rem] overflow-auto p-5"
          :class="pane === 'preview' ? '' : 'hidden lg:block'"
        >
          <Markdown v-if="draft.body" :source="draft.body" />
          <p v-else class="text-sm text-muted-foreground">
            Nothing written yet. What appears here is what a person reads, and what a model is
            given when your work is checked against this rule.
          </p>
        </UiCard>
      </div>
    </template>
  </div>
</template>
