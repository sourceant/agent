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
import { ArrowLeft, Check, ChevronDown, ChevronRight, Copy, Loader2, ScrollText } from 'lucide-vue-next'
import { useUp } from '~/composables/useUp'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* Adding a skill, or changing one.
 *
 * Its own route rather than the screen that reads one: a read view that is also
 * a form leaves somebody looking at boxes when they came to read a skill.
 *
 * A name, a line saying what it is about, and the skill itself. Everything else
 * has an answer already and sits behind one line, because six decisions before
 * the first word is how a form stops being filled in.
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
const fresh = computed(() => !id.value || id.value === NEW)

const skill = ref(null)
const draft = ref({
  name: '',
  description: '',
  body: '',
  paths: [],
  // Which uses it is kept out of, and whether it may be selected without being
  // named.
  applications: {},
  automatic: true,
  type: '',
})
// Either GLOBAL, or the name of the repository it is for.
const belongsTo = ref(GLOBAL)
const pane = ref('write')
const details = ref(false)
const saving = ref(false)
const problem = ref('')
const loading = ref(true)
const uses = ref([])

const panes = [
  { id: 'write', label: 'Write' },
  { id: 'preview', label: 'Preview' },
]

/* The folder a skill is saved in takes the name, so the name is typed once and
 * the folder follows it. */
const slug = (text) =>
  text
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, '-')
    .replace(/^-+|-+$/g, '')
    .slice(0, 64)

const named = computed(() => (fresh.value ? slug(draft.value.name) : id.value))

const on = (use) => draft.value.applications?.[use] !== false

function toggle(use) {
  const applications = { ...draft.value.applications }
  if (on(use)) applications[use] = false
  else delete applications[use]
  draft.value.applications = applications
}

const forEverything = computed(() => belongsTo.value === GLOBAL)

const belongings = computed(() => [
  { id: GLOBAL, label: 'Everywhere' },
  ...repositories.value.map((one) => ({ id: one.name, label: one.name })),
])

// What the details say without being opened, so nothing has to be opened to
// find out what it is set to.
const settled = computed(() => {
  const off = uses.value.filter((use) => !on(use.id)).map((use) => use.label)
  return [
    forEverything.value ? 'Everywhere' : belongsTo.value,
    off.length ? `not for ${off.join(', ').toLowerCase()}` : 'all uses',
    draft.value.automatic ? 'automatic' : 'only when named',
    draft.value.paths.length ? `${draft.value.paths.length} file pattern${draft.value.paths.length === 1 ? '' : 's'}` : '',
  ]
    .filter(Boolean)
    .join(' · ')
})

const theirs = computed(() => !!skill.value && !OURS.includes(skill.value.origin))
const copying = computed(() => theirs.value)

const ready = computed(() => !!named.value && !!draft.value.description.trim())

const changed = computed(() => {
  if (fresh.value) return !!(draft.value.name || draft.value.description || draft.value.body)
  if (!skill.value) return false
  return (
    draft.value.name !== skill.value.name ||
    draft.value.description !== skill.value.description ||
    draft.value.body !== (skill.value.body ?? '') ||
    draft.value.paths.join('\n') !== (skill.value.paths ?? []).join('\n') ||
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
      name: '',
      description: '',
      body: '',
      paths: [],
      applications: {},
      automatic: true,
      type: '',
    }
    // What the list was showing, so adding one for the project being looked at
    // takes no thought, and is still named on the screen.
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
      name: found.name,
      description: found.description,
      body: found.body ?? '',
      paths: [...(found.paths ?? [])],
      applications: { ...(found.applications ?? {}) },
      automatic: found.automatic ?? true,
      type: found.type ?? '',
    }
  } catch (caught) {
    problem.value = caught.message
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  problem.value = ''
  try {
    const written = await api.recordSkill({
      scope: forEverything.value ? GLOBAL : REPOSITORY,
      repository: forEverything.value ? '' : belongsTo.value,
      id: named.value,
      name: draft.value.name || named.value,
      description: draft.value.description,
      body: draft.value.body,
      paths: draft.value.paths,
      applications: draft.value.applications,
      automatic: draft.value.automatic,
      type: draft.value.type,
    })
    router.replace(`/skills/${written.id}`)
  } catch (caught) {
    problem.value = caught.message
  } finally {
    saving.value = false
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
      :title="fresh ? 'Add skill' : draft.name || id"
      :sub="skill?.path"
      :mono="!!skill?.path"
    >
      <template #back>
        <UiButton
          variant="ghost"
          size="icon"
          aria-label="Back"
          @click="up(skill ? `/skills/${id}` : '/skills')"
        >
          <ArrowLeft class="h-4 w-4" />
        </UiButton>
      </template>
      <template #icon><ScrollText class="h-5 w-5" /></template>
      <template #badges>
        <UiBadge v-if="named" variant="outline" class="font-mono">{{ named }}</UiBadge>
      </template>
      <template #actions>
        <UiButton variant="outline" size="sm" @click="up(skill ? `/skills/${id}` : '/skills')">
          Cancel
        </UiButton>
        <UiButton size="sm" :disabled="saving || !changed || !ready" @click="save">
          <Loader2 v-if="saving" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
          <component :is="copying ? Copy : Check" v-else class="mr-1.5 h-3.5 w-3.5" />
          {{ copying ? 'Copy' : fresh ? 'Add' : 'Save' }}
        </UiButton>
      </template>
    </PageHead>

    <Notice v-if="copying" tone="info" class="mb-4">
      Read-only file. Copy keeps your own, which is then the one used.
    </Notice>

    <Notice v-if="problem" tone="danger" class="mb-4">
      {{ problem }}
    </Notice>

    <p v-if="loading" class="py-10 text-center text-sm text-muted-foreground">Loading.</p>

    <template v-else>
      <UiCard class="mb-3 p-5">
        <div class="grid gap-4 lg:grid-cols-3">
          <Field label="Name" for="skill-name">
            <Input id="skill-name" v-model="draft.name" placeholder="Retry limit" />
          </Field>
          <Field
            class="lg:col-span-2"
            label="Description"
            for="skill-description"
            hint="Matched against a change during automatic selection."
          >
            <Input
              id="skill-description"
              v-model="draft.description"
              placeholder="Use when a change touches how a charge is retried."
            />
          </Field>
        </div>

        <div class="mt-4 border-t pt-3">
          <button
            type="button"
            class="flex w-full items-center gap-1.5 text-left text-sm text-muted-foreground hover:text-foreground"
            @click="details = !details"
          >
            <ChevronDown v-if="details" class="h-3.5 w-3.5 shrink-0" />
            <ChevronRight v-else class="h-3.5 w-3.5 shrink-0" />
            <span class="truncate">{{ settled }}</span>
          </button>

          <div v-show="details" class="mt-4 grid gap-4 lg:grid-cols-3">
            <Field label="Scope" for="skill-belongs">
              <Select
                id="skill-belongs"
                v-model="belongsTo"
                :disabled="!!skill && !theirs"
                class="w-full"
              >
                <option v-for="one in belongings" :key="one.id" :value="one.id">{{ one.label }}</option>
              </Select>
            </Field>
            <Field class="lg:col-span-2" label="Skill uses">
              <div class="flex flex-wrap gap-1.5">
                <button
                  v-for="use in uses"
                  :key="use.id"
                  type="button"
                  :aria-pressed="on(use.id)"
                  :class="[
                    'rounded-full border px-2.5 py-0.5 text-xs transition-colors hover:border-primary/50',
                    on(use.id)
                      ? 'border-primary/30 bg-primary/10 text-foreground'
                      : 'border-border text-muted-foreground line-through',
                  ]"
                  @click="toggle(use.id)"
                >
                  {{ use.label }}
                </button>
              </div>
              <label class="mt-3 flex items-center gap-2 text-sm">
                <input v-model="draft.automatic" type="checkbox" class="h-3.5 w-3.5 rounded border">
                Allow automatic selection
              </label>
              <p class="mt-1 text-xs text-muted-foreground">
                Off: read only when something names it.
              </p>
            </Field>
            <Field
              class="lg:col-span-3"
              label="Files"
              for="skill-paths"
              hint="Globs, one a line. Limits selection to these."
            >
              <ListInput
                v-model="draft.paths"
                mono
                size="sm"
                noun="a pattern"
                placeholder="db/migrations/**"
              />
            </Field>
          </div>
        </div>
      </UiCard>

      <div class="mb-2 flex items-center justify-between gap-3">
        <Tabs v-model="pane" :tabs="panes" label="Write or preview" />
        <p class="text-xs text-muted-foreground">{{ lines }} line{{ lines === 1 ? '' : 's' }} · markdown</p>
      </div>

      <Textarea
        v-if="pane === 'write'"
        v-model="draft.body"
        class="min-h-[24rem] flex-1 resize-none font-mono leading-relaxed"
        placeholder="Never edit a migration that has already run. Add a new one instead."
        aria-label="The skill"
      />
      <UiCard v-else class="min-h-[24rem] flex-1 overflow-auto p-5">
        <Markdown v-if="draft.body" :source="draft.body" />
        <p v-else class="text-sm text-muted-foreground">Nothing written yet.</p>
      </UiCard>
    </template>
  </div>
</template>
