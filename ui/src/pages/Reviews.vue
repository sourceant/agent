<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Chip,
  Diff,
  ItemCard,
  Markdown,
  Notice,
  PageHead,
  Select,
  Tabs,
} from '@sourceant/design'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  Check,
  Clock,
  FileCode,
  GitBranch,
  Loader2,
  Plus,
  ShieldCheck,
  TriangleAlert,
  Wand2,
} from 'lucide-vue-next'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* Work read before anybody else has been asked to read it.
 *
 * Laid out the way a pull request is, because that is the thing somebody is
 * about to open and the shape they already read in: the change on the left,
 * what changed in the file they picked on the right, and anything said about a
 * line drawn against that line rather than in a list somewhere else.
 */

const route = useRoute()
const router = useRouter()
const { repositories, chosen, error, fetchRepositories } = useRepositories()
const running = ref(false)
const judging = ref(false)
const result = ref(null)
const reading = ref(null)
const hasModel = ref(false)
const looking = ref('')
const picked = ref([])
const adding = ref('')
const skills = ref([])
const past = ref([])
const tab = ref('overview')

const files = computed(() => result.value?.changed ?? [])

// Named in the link, so an agent can hand somebody one and it opens here.
const named = computed(() => String(route.params.id ?? ''))

const spare = computed(() =>
  skills.value.filter((skill) => !picked.value.includes(skill.id)),
)

const byId = computed(() => Object.fromEntries(skills.value.map((one) => [one.id, one])))
const nameOf = (id) => byId.value[id]?.name ?? id

// What a skill made of this change, or undefined where it has not been asked.
const verdictFor = (id) => verdicts.value.find((one) => one.skill === id)?.passed

function drop(id) {
  picked.value = picked.value.filter((one) => one !== id)
}

function add(id) {
  if (id && !picked.value.includes(id)) picked.value = [...picked.value, id]
  adding.value = ''
}
const where = computed(() => result.value?.where ?? null)

const verdicts = computed(() => result.value?.verdicts ?? [])
const findings = computed(() =>
  verdicts.value.flatMap((verdict) =>
    verdict.findings.map((finding) => ({ ...finding, skill: verdict.skill })),
  ),
)
const blocking = computed(() => findings.value.filter((one) => one.severity === 'blocking'))
const advisory = computed(() => findings.value.filter((one) => one.severity !== 'blocking'))

// Said about the file somebody is looking at, so the diff can draw it in place.


// How loudly a file is asking to be looked at, so the worst sorts first.
const weight = computed(() => {
  const by = {}
  for (const one of findings.value) {
    if (!one.path) continue
    by[one.path] = (by[one.path] ?? 0) + (one.severity === 'blocking' ? 10 : 1)
  }
  for (const one of suggestions.value) {
    by[one.path] = (by[one.path] ?? 0) + 1
  }
  return by
})

const listed = computed(() =>
  [...files.value].sort(
    (a, b) => (weight.value[b.path] ?? 0) - (weight.value[a.path] ?? 0),
  ),
)

const showing = computed(
  () => files.value.find((file) => file.path === looking.value) ?? listed.value[0] ?? null,
)

// Anything said about no file in particular: a commit message, a missing
// description. That belongs at the top, not against a line of code.
const overall = computed(() => findings.value.filter((one) => !one.path))

// The review proper, as opposed to what the skills made of it.
const read = computed(() => result.value?.review ?? null)
const summary = computed(() => read.value?.summary ?? null)
const suggestions = computed(() => read.value?.suggestions ?? [])

const VERDICTS = {
  APPROVE: { label: 'Looks good', tone: 'success' },
  REQUEST_CHANGES: { label: 'Change this first', tone: 'danger' },
  COMMENT: { label: 'Worth a look', tone: 'warning' },
}
const verdict = computed(() => VERDICTS[read.value?.verdict] ?? null)

const tabs = computed(() => [
  { id: 'overview', label: 'Overview' },
  {
    id: 'details',
    label: `Files${files.value.length ? ` ${files.value.length}` : ''}`,
  },
])

// A suggestion is about a line, so it is drawn against that line beside
// whatever a skill said about the same place.
const notesOn = (path) => [
  ...findings.value
    .filter((one) => one.path === path)
    .map((one) => ({ ...one, from: one.skill })),
  ...suggestions.value
    .filter((one) => one.path === path)
    .map((one) => ({
      line: one.start_line,
      severity: 'suggestion',
      detail: one.comment,
      code: one.suggested_code,
      from: one.category || 'review',
    })),
]

async function loadPast() {
  try {
    past.value = await api.reviews(chosen.value)
  } catch {
    past.value = []
  }
}

function when(stamp) {
  if (!stamp) return ''
  const at = new Date(stamp)
  const ago = Math.round((Date.now() - at.getTime()) / 60000)
  if (ago < 1) return 'just now'
  if (ago < 60) return `${ago} minute${ago === 1 ? '' : 's'} ago`
  const hours = Math.round(ago / 60)
  if (hours < 24) return `${hours} hour${hours === 1 ? '' : 's'} ago`
  return at.toLocaleDateString()
}

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

async function loadSkills() {
  try {
    skills.value = (await api.skills(chosen.value)).skills
  } catch {
    skills.value = []
  }
}

const ASKING_AGAIN = 1500
let asking = null

function stopAsking() {
  if (asking) clearTimeout(asking)
  asking = null
}

async function run(useModel) {
  stopAsking()
  const flag = useModel ? judging : running
  flag.value = true
  error.value = ''
  try {
    const started = await api.startReview(chosen.value, {
      useModel,
      skills: [...picked.value],
    })
    // The name goes in the address, so this review can be come back to and
    // handed to somebody else.
    router.replace(`/reviews/${started.id}`)
    await collect(started.id, useModel)
  } catch (caught) {
    error.value = caught.message
    flag.value = false
  }
}

async function collect(id, useModel = true) {
  const flag = useModel ? judging : running
  let answered
  try {
    answered = await api.reviewed(id)
  } catch (caught) {
    error.value = caught.message
    flag.value = false
    return
  }

  reading.value = answered
  if (answered.status === 'running') {
    flag.value = true
    asking = setTimeout(() => collect(id, useModel), ASKING_AGAIN)
    return
  }

  flag.value = false
  if (answered.status === 'failed') {
    // What was already read stays on screen. Losing a good reading of the
    // change because the judging failed is two steps backwards for one problem.
    error.value = answered.error
    return
  }
  result.value = answered.review
  looking.value = ''
  tab.value = 'overview'
  loadPast()
  if (answered.repository) chosen.value = answered.repository
  // What it was actually read against, so removing one and running again is
  // the obvious next move rather than a form to fill in.
  picked.value = (answered.review.skills ?? []).map((one) => one.id)
}

watch(chosen, () => {
  stopAsking()
  loadSkills()
  loadPast()
})

// A link somebody was handed opens the review it names.
watch(named, (id) => {
  if (!id) return
  stopAsking()
  collect(id)
})

onUnmounted(stopAsking)

onMounted(async () => {
  await fetchRepositories()
  await Promise.all([checkModel(), loadSkills(), loadPast()])
  if (named.value) await collect(named.value)
})
</script>

<template>
  <div class="flex h-full min-h-0 flex-col">
    <PageHead
      pillar="review"
      title="Reviews"
      sub="Your work read here, before anybody else is asked to read it."
    >
      <template #icon><ShieldCheck class="h-6 w-6" /></template>
      <template #actions>
        <Select v-if="repositories.length > 1" v-model="chosen" size="sm" aria-label="Repository">
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>

        <UiButton v-if="repositories.length" size="sm" variant="outline" :disabled="running" @click="run(false)">
          <Loader2 v-if="running" class="mr-2 h-4 w-4 animate-spin" />
          <FileCode v-else class="mr-2 h-4 w-4" />
          {{ running ? 'Reading…' : 'Read what changed' }}
        </UiButton>
        <UiButton
          v-if="repositories.length && hasModel"
          size="sm"
          variant="glow"
          :disabled="judging"
          @click="run(true)"
        >
          <Loader2 v-if="judging" class="mr-2 h-4 w-4 animate-spin" />
          <Wand2 v-else class="mr-2 h-4 w-4" />
          {{ judging ? 'Reviewing…' : 'Review it' }}
        </UiButton>
      </template>
    </PageHead>

    <Notice v-if="error" tone="danger" class="mb-4">{{ error }}</Notice>

    <!-- What it is read against. Whatever applied comes back here after a
         review, so taking one off and running again is the obvious next move
         rather than a form to fill in. -->
    <UiCard v-if="repositories.length" class="mb-3 flex flex-wrap items-center gap-2 p-3">
      <span class="text-xs uppercase tracking-wider text-muted-foreground">Read against</span>

      <Chip
        v-for="id in picked"
        :key="id"
        removable
        :label="nameOf(id)"
        :tone="verdictFor(id) === undefined ? 'default' : verdictFor(id) ? 'success' : 'danger'"
        @remove="drop(id)"
      >
        {{ nameOf(id) }}
      </Chip>

      <Chip v-if="!picked.length" tone="muted">Whatever applies</Chip>

      <Select
        v-if="spare.length"
        :model-value="adding"
        size="sm"
        class="ml-auto"
        aria-label="Add a skill"
        @update:model-value="add"
      >
        <option value="">Add a skill…</option>
        <option v-for="skill in spare" :key="skill.id" :value="skill.id">{{ skill.name }}</option>
      </Select>
    </UiCard>

    <EmptyMachine v-if="repositories.length === 0" />

    <template v-else>
      <Notice v-if="!hasModel" tone="info" class="mb-4">
        No model is configured, so nothing here can be judged. Reading what changed needs nothing.
        Choose a model in Settings to have the work read against your skills.
      </Notice>

      <template v-if="!result">
        <UiCard
          v-if="reading?.status === 'running'"
          class="flex flex-1 flex-col items-center justify-center p-12 text-center"
        >
          <Loader2 class="mb-3 h-8 w-8 animate-spin text-muted-foreground" />
          <p class="font-medium">Reading it.</p>
          <p class="mx-auto mt-1 max-w-md text-sm text-muted-foreground">
            This keeps going whether or not anybody is watching, and the link to it keeps working.
          </p>
        </UiCard>

        <template v-else>
          <UiCard class="mb-3 p-8 text-center">
            <ShieldCheck class="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
            <p class="font-medium">Read this checkout's work.</p>
            <p class="mx-auto mt-1 max-w-md text-sm text-muted-foreground">
              Everything comes off the checkout, so work you have not pushed, or not committed,
              still gets a review.
            </p>
            <UiButton
              v-if="repositories.length && hasModel"
              class="mt-4"
              variant="glow"
              :disabled="judging"
              @click="run(true)"
            >
              <Loader2 v-if="judging" class="mr-2 h-4 w-4 animate-spin" />
              <Plus v-else class="mr-2 h-4 w-4" />
              Review it
            </UiButton>
          </UiCard>

          <!-- Older ones, because a review has a name and somebody may have
               been handed one, or run one this morning. -->
          <template v-if="past.length">
            <p class="mb-2 text-xs uppercase tracking-wider text-muted-foreground">Earlier</p>
            <div class="space-y-2">
              <ItemCard
                v-for="one in past"
                :key="one.id"
                :title="one.title || one.repository"
                :subtitle="one.id"
                pillar="review"
                hover
                class="cursor-pointer"
                @click="router.push(`/reviews/${one.id}`)"
              >
                <template #icon><Clock class="h-5 w-5" /></template>
                <template #badges>
                  <UiBadge
                    :variant="one.status === 'done' ? 'success' : one.status === 'failed' ? 'destructive' : 'secondary'"
                  >
                    {{ one.status }}
                  </UiBadge>
                </template>
                <template #meta>
                  <span>{{ when(one.started) }}</span>
                  <span class="font-mono">{{ one.repository }}</span>
                </template>
              </ItemCard>
            </div>
          </template>
        </template>
      </template>

      <template v-else>
        <!-- What is being reviewed, against what, and how it went. -->
        <UiCard class="mb-3 flex flex-wrap items-center gap-x-4 gap-y-2 p-4">
          <component
            :is="result.ready ? Check : TriangleAlert"
            class="h-5 w-5 shrink-0"
            :class="result.ready ? 'text-success' : 'text-destructive'"
          />
          <p class="font-medium">
            <template v-if="blocking.length">
              {{ blocking.length }} thing{{ blocking.length === 1 ? '' : 's' }} to fix first
            </template>
            <template v-else-if="verdicts.length">Nothing here says this is not ready.</template>
            <template v-else>{{ result.note || 'Read, not judged.' }}</template>
          </p>

          <span v-if="where" class="flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
            <GitBranch class="h-3.5 w-3.5" />
            <span class="font-mono">{{ where.branch || 'no branch' }}</span>
            <span aria-hidden="true">→</span>
            <span class="font-mono">{{ where.against }}</span>
            <UiBadge variant="outline">
              {{ where.commits }} commit{{ where.commits === 1 ? '' : 's' }} ahead
            </UiBadge>
            <UiBadge variant="outline">
              {{ files.length }} file{{ files.length === 1 ? '' : 's' }}
            </UiBadge>
            <UiBadge v-if="advisory.length" variant="secondary">
              {{ advisory.length }} suggestion{{ advisory.length === 1 ? '' : 's' }}
            </UiBadge>
          </span>
        </UiCard>

        <UiCard v-if="verdicts.length" class="mb-3 p-4">
          <p class="mb-2 text-xs uppercase tracking-wider text-muted-foreground">
            What each one made of it
          </p>
          <div class="flex flex-wrap gap-1.5">
            <Chip
              v-for="verdict in verdicts"
              :key="verdict.skill"
              :tone="verdict.passed ? 'success' : 'danger'"
              :title="verdict.note"
            >
              <template #mark>
                <component
                  :is="verdict.passed ? Check : TriangleAlert"
                  class="h-3 w-3 shrink-0"
                  :class="verdict.passed ? 'text-success' : 'text-destructive'"
                />
              </template>
              {{ verdict.skill }}
              <span v-if="!verdict.passed" class="ml-1 opacity-70">
                {{ verdict.findings.length }}
              </span>
            </Chip>
          </div>
        </UiCard>

        <Notice
          v-for="(one, index) in overall"
          :key="index"
          :tone="one.severity === 'blocking' ? 'danger' : 'warning'"
          class="mb-3"
        >
          {{ one.detail }}
          <span class="text-muted-foreground">— {{ one.skill }}</span>
        </Notice>

        <Tabs v-model="tab" :tabs="tabs" label="What to look at" class="mb-3 w-fit" />

        <!-- The review, in the order somebody reads one. -->
        <div v-if="tab === 'overview'" class="min-h-0 flex-1 space-y-3 overflow-y-auto">
          <UiCard v-if="summary?.overview" class="p-5">
            <div class="mb-2 flex flex-wrap items-center gap-2">
              <h2 class="font-semibold">What this change does</h2>
              <UiBadge v-if="verdict" :variant="verdict.tone === 'danger' ? 'destructive' : verdict.tone">
                {{ verdict.label }}
              </UiBadge>
            </div>
            <Markdown :source="summary.overview" />
          </UiCard>

          <UiCard v-if="summary?.critical_issues?.length" class="border-destructive/40 p-5">
            <h2 class="mb-2 font-semibold">Worth stopping for</h2>
            <ul class="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
              <li v-for="(one, index) in summary.critical_issues" :key="index">{{ one }}</li>
            </ul>
          </UiCard>

          <UiCard v-if="summary?.key_improvements?.length" class="p-5">
            <h2 class="mb-2 font-semibold">Worth changing</h2>
            <ul class="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
              <li v-for="(one, index) in summary.key_improvements" :key="index">{{ one }}</li>
            </ul>
          </UiCard>

          <UiCard v-if="summary?.minor_suggestions?.length" class="p-5">
            <h2 class="mb-2 font-semibold">Nice to have</h2>
            <ul class="list-disc space-y-1 pl-5 text-sm text-muted-foreground">
              <li v-for="(one, index) in summary.minor_suggestions" :key="index">{{ one }}</li>
            </ul>
          </UiCard>

          <UiCard v-if="suggestions.length" class="p-5">
            <h2 class="mb-1 font-semibold">
              {{ suggestions.length }} suggestion{{ suggestions.length === 1 ? '' : 's' }}
            </h2>
            <p class="mb-3 text-sm text-muted-foreground">
              Each one is drawn against the line it is about, under Files.
            </p>
            <ul class="space-y-1.5 text-sm">
              <li v-for="(one, index) in suggestions" :key="index">
                <button
                  type="button"
                  class="text-left hover:underline"
                  @click="looking = one.path; tab = 'details'"
                >
                  <span class="font-mono text-xs">{{ one.path }}:{{ one.start_line }}</span>
                  <span class="ml-2 text-muted-foreground">{{ one.comment }}</span>
                </button>
              </li>
            </ul>
          </UiCard>

          <UiCard v-for="(text, name) in read?.notes ?? {}" :key="name" class="p-5">
            <h2 class="mb-2 font-semibold capitalize">{{ name.replace(/_/g, ' ') }}</h2>
            <Markdown :source="text" />
          </UiCard>

          <UiCard v-if="!summary?.overview" class="p-8 text-center text-sm text-muted-foreground">
            {{ result.note || 'Read, not judged. Ask for a review to have it read properly.' }}
          </UiCard>
        </div>

        <div v-else class="grid min-h-0 flex-1 gap-3 lg:grid-cols-[18rem_1fr]">
          <!-- The change on the left, the way a pull request lists it. -->
          <UiCard class="flex min-h-0 flex-col overflow-hidden">
            <div class="border-b px-3 py-2 text-xs uppercase tracking-wider text-muted-foreground">
              Changed
            </div>
            <ul class="min-h-0 flex-1 overflow-y-auto py-1">
              <li v-for="file in listed" :key="file.path">
                <button
                  type="button"
                  class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-sm transition-colors"
                  :class="showing?.path === file.path ? 'bg-primary/10 text-primary' : 'hover:bg-muted'"
                  @click="looking = file.path"
                >
                  <span class="min-w-0 flex-1 truncate font-mono text-xs" :title="file.path">
                    {{ file.path }}
                  </span>
                  <UiBadge
                    v-if="notesOn(file.path).length"
                    :variant="weight[file.path] >= 10 ? 'destructive' : 'secondary'"
                  >
                    {{ notesOn(file.path).length }}
                  </UiBadge>
                  <span class="shrink-0 text-[10px] uppercase text-muted-foreground">
                    {{ file.change.slice(0, 3) }}
                  </span>
                </button>
              </li>
            </ul>

          </UiCard>

          <!-- What changed in it, with anything said about a line against it. -->
          <div class="min-h-0 overflow-y-auto">
            <template v-if="showing">
              <div class="mb-2 flex flex-wrap items-center gap-2">
                <span class="break-all font-mono text-sm">{{ showing.path }}</span>
                <UiBadge variant="secondary">{{ showing.change }}</UiBadge>
              </div>
              <Diff :patch="showing.patch" :notes="notesOn(showing.path)" />
            </template>
            <UiCard v-else class="p-10 text-center text-sm text-muted-foreground">
              Nothing has changed in this checkout.
            </UiCard>
          </div>
        </div>
      </template>
    </template>
  </div>
</template>
