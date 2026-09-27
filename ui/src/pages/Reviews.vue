<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Chip,
  Diff,
  DotIndicator,
  Input,
  Markdown,
  Empty,
  Loading,
  Notice,
  PageHead,
  Section,
  Select,
  Status,
  Table,
  Tabs,
} from '@sourceant/design'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  ArrowLeft,
  Boxes,
  Bug,
  CircleAlert,
  CircleCheck,
  MessageSquare,
  FileCode,
  FileText,
  GitCommit,
  Lightbulb,
  Loader2,
  Search,
  ShieldCheck,
  Sparkles,
  TriangleAlert,
  Wand2,
} from 'lucide-vue-next'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useUp } from '~/composables/useUp'
import { EVERY, useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'
import { when } from '~/moments'

/* Work read before anybody else has been asked to read it.
 *
 * Laid out the way a pull request is, because that is the thing somebody is
 * about to open and the shape they already read in: the change on the left,
 * what changed in the file they picked on the right, and anything said about a
 * line drawn against that line rather than in a list somewhere else.
 */

const route = useRoute()
const router = useRouter()
const up = useUp()
const { repositories, chosen, error, fetchRepositories } = useRepositories({
  all: true,
})
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
const term = ref('')

const columns = [
  { id: 'status', label: 'Status', width: '6rem' },
  { id: 'review', label: 'Review' },
  { id: 'repository', label: 'Repository', narrow: true },
  { id: 'started', label: 'Started', narrow: true },
  { id: 'took', label: 'Took', align: 'right', narrow: true },
]

// How long it took, for telling a review that ran from one that gave up at once.
function took(one) {
  if (!one.finished || !one.started) return ''
  const seconds = Math.round((new Date(one.finished) - new Date(one.started)) / 1000)
  if (seconds < 1) return 'instant'
  if (seconds < 60) return `${seconds}s`
  return `${Math.round(seconds / 60)}m`
}

const earlier = computed(() => {
  const wanted = term.value.trim().toLowerCase()
  if (!wanted) return past.value
  return past.value.filter((one) =>
    `${one.title ?? ''} ${one.repository ?? ''} ${one.status ?? ''}`.toLowerCase().includes(wanted),
  )
})
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

// The rules this change does not meet. What it does meet is the ordinary case
// and needs no announcement.
const unmet = computed(() => verdicts.value.filter((one) => !one.passed))

const commits = computed(() => result.value?.commits ?? [])

// Whether the overview has anything to be a document about.
const anything = computed(
  () =>
    Boolean(summary.value?.overview) ||
    unmet.value.length > 0 ||
    overall.value.length > 0 ||
    suggestions.value.length > 0 ||
    Object.keys(read.value?.notes ?? {}).length > 0,
)

const tabs = computed(() => {
  const listing = [
    { id: 'overview', label: 'Overview' },
    {
      id: 'details',
      label: `Files${files.value.length ? ` ${files.value.length}` : ''}`,
    },
  ]
  listing.push({
    id: 'commits',
    label: `Commits${commits.value.length ? ` ${commits.value.length}` : ''}`,
  })
  return listing
})

// What a reviewer said, in the words a reviewer says them in. "Nothing here
// says this is not ready" is not a thing anybody says out loud.
const VERDICTS = {
  APPROVE: { label: 'Approved', tone: 'success' },
  REQUEST_CHANGES: { label: 'Changes requested', tone: 'danger' },
  COMMENT: { label: 'Commented', tone: 'warning' },
}

const verdict = computed(() => {
  if (blocking.value.length) {
    return {
      label: 'Changes requested',
      tone: 'danger',
      count: blocking.value.length,
    }
  }
  const said = read.value?.verdict
  if (said && VERDICTS[said]) {
    return { ...VERDICTS[said], count: suggestions.value.length || null }
  }
  if (!verdicts.value.length && !read.value) {
    return { label: 'Not reviewed', tone: 'neutral', count: null }
  }
  return { label: 'Commented', tone: 'warning', count: advisory.value.length || null }
})

// The icon says the same thing as the verdict. A shield with a tick on it,
// tinted with the review pillar's green, said "approved" on every review.
const MARKS = {
  success: CircleCheck,
  danger: CircleAlert,
  warning: MessageSquare,
  neutral: ShieldCheck,
}
const mark = computed(() => (result.value ? MARKS[verdict.value.tone] : ShieldCheck))

const CATEGORIES = {
  BUG: 'danger',
  SECURITY: 'danger',
  PERFORMANCE: 'warning',
  REFACTOR: 'info',
  STYLE: 'neutral',
  CLARITY: 'neutral',
  TEST: 'info',
  DOCUMENTATION: 'neutral',
}

const toneOf = (category) => CATEGORIES[String(category || '').toUpperCase()] ?? 'info'

// A suggestion is about a line, so it is drawn against that line beside
// whatever a skill said about the same place.
const notesOn = (path) => [
  ...findings.value
    .filter((one) => one.path === path)
    .map((one) => ({
      ...one,
      from: one.skill,
      tone: one.severity === 'blocking' ? 'danger' : 'warning',
    })),
  ...suggestions.value
    .filter((one) => one.path === path)
    .map((one) => ({
      line: one.start_line,
      severity: 'suggestion',
      detail: one.comment,
      code: one.suggested_code,
      replacing: one.existing_code,
      from: one.category || 'review',
      tone: toneOf(one.category),
    })),
]

// Choosing one narrows the page to it. It does not start a review: asking for
// work to be done is a click somebody makes deliberately, on the button that
// says so, not a side effect of picking from a list.
async function narrowTo(name) {
  if (!name) return
  chosen.value = name
  await Promise.all([loadSkills(), loadPast()])
}

async function loadPast() {
  try {
    past.value = await api.reviews(chosen.value)
  } catch {
    past.value = []
  }
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
  // Opening one does not narrow the page to its repository: coming back would
  // then show a listing of one, and the choice is shared with every screen.
  // What it was actually read against, so removing one and running again is
  // the obvious next move rather than a form to fill in.
  picked.value = (answered.review.skills ?? []).map((one) => one.id)
}

watch(chosen, () => {
  stopAsking()
  loadSkills()
  loadPast()
})

// A link somebody was handed opens the review it names. Going back to the
// listing has to put the review away, or the URL changes and the page does
// not, which reads as a back button that does nothing.
watch(named, (id) => {
  stopAsking()
  if (!id) {
    result.value = null
    reading.value = null
    loadPast()
    return
  }
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
      :title="where ? `${where.branch || 'no branch'} → ${where.against}` : 'Reviews'"
      :sub="where ? where.path : 'Your work read here, before anybody else is asked to read it.'"
      :mono="Boolean(where)"
      :tone="result ? verdict.tone : undefined"
    >
      <template v-if="named" #back>
        <UiButton variant="ghost" size="icon" aria-label="Back to reviews" @click="up('/reviews')">
          <ArrowLeft class="h-4 w-4" />
        </UiButton>
      </template>
      <template #icon><component :is="mark" class="h-6 w-6" /></template>
      <template v-if="result" #meta>
        <Status :label="verdict.label" :tone="verdict.tone" :count="verdict.count" />
        <span v-if="where">{{ where.commits }} commit{{ where.commits === 1 ? '' : 's' }} ahead</span>
        <span>{{ files.length }} file{{ files.length === 1 ? '' : 's' }}</span>

      </template>
      <template #actions>
        <Select v-if="repositories.length > 1" v-model="chosen" size="sm" aria-label="Repository">
          <option :value="EVERY">All repositories</option>
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>

        <UiButton
          v-if="repositories.length && chosen"
          size="sm"
          variant="outline"
          :disabled="running"
          @click="run(false)"
        >
          <Loader2 v-if="running" class="mr-2 h-4 w-4 animate-spin" />
          <FileCode v-else class="mr-2 h-4 w-4" />
          {{ running ? 'Reading…' : 'Read what changed' }}
        </UiButton>
        <UiButton
          v-if="repositories.length && hasModel && chosen"
          size="sm"
          variant="glow"
          :disabled="judging"
          @click="run(true)"
        >
          <Loader2 v-if="judging" class="mr-2 h-4 w-4 animate-spin" />
          <Wand2 v-else class="mr-2 h-4 w-4" />
          {{ judging ? 'Reviewing…' : 'Review it' }}
        </UiButton>

        <Select
          v-else-if="repositories.length && hasModel"
          :model-value="''"
          size="sm"
          aria-label="Choose a repository to review"
          @update:model-value="narrowTo"
        >
          <option value="">Choose a repository…</option>
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>
      </template>
    </PageHead>

    <Notice v-if="error" tone="danger" class="mb-4">{{ error }}</Notice>

    <EmptyMachine v-if="repositories.length === 0" />

    <template v-else>
      <Notice v-if="!hasModel && chosen" tone="info" class="mb-4">
        No model is configured, so nothing here can be judged. Reading what changed needs nothing.
        <RouterLink to="/settings?group=Model" class="font-medium underline">
          Choose a model
        </RouterLink>
        to have the work read against your skills.
      </Notice>

      <template v-if="!result">
        <Loading
          v-if="reading?.status === 'running'"
          label="Reading it"
          note="This keeps going whether or not anybody is watching, and the link to it keeps working."
        />

        <template v-else>
          <!-- Older ones, because a review has a name and somebody may have
               been handed one, or run one this morning. -->
          <Empty v-if="!past.length" title="Nothing read here yet">
            <template #icon><ShieldCheck class="h-8 w-8" /></template>
            Everything comes off the checkout, so work you have not pushed, or not
            committed, still gets a review.
            <template v-if="!chosen" #actions>
              <UiButton
                v-for="repository in repositories"
                :key="repository.name"
                size="sm"
                variant="outline"
                @click="narrowTo(repository.name)"
              >
                <Boxes class="mr-2 h-4 w-4" />
                {{ repository.name }}
              </UiButton>
            </template>
          </Empty>

          <template v-if="past.length">
            <div class="mb-2 flex items-center justify-between gap-3">
              <p class="text-xs uppercase tracking-wider text-muted-foreground">Earlier</p>
              <div v-if="past.length > 5" class="relative">
                <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input v-model="term" size="sm" placeholder="Find a review" class="w-48 pl-8" aria-label="Find a review" />
              </div>
            </div>

            <Table :columns="columns" :rows="earlier" row-key="id" label="Earlier reviews">
              <template #status="{ row }">
                <UiBadge
                  :variant="row.status === 'done' ? 'success' : row.status === 'failed' ? 'destructive' : 'secondary'"
                >
                  {{ row.status }}
                </UiBadge>
              </template>
              <template #review="{ row }">
                <button
                  type="button"
                  class="block max-w-md truncate text-left font-medium hover:text-primary"
                  @click="router.push(`/reviews/${row.id}`)"
                >
                  {{ row.title || 'Unnamed' }}
                </button>
                <span v-if="row.error" class="block max-w-md truncate text-xs text-destructive">
                  {{ row.error }}
                </span>
              </template>
              <template #repository="{ row }">
                <span class="font-mono text-xs text-muted-foreground">{{ row.repository }}</span>
              </template>
              <template #started="{ row }">
                <span class="text-muted-foreground">{{ when(row.started) }}</span>
              </template>
              <template #took="{ row }">
                <span class="tabular-nums text-muted-foreground">{{ took(row) }}</span>
              </template>
            </Table>
          </template>
        </template>
      </template>

      <template v-else>
        <div class="mb-3 flex flex-wrap items-center gap-x-3 gap-y-2">
          <Tabs v-model="tab" :tabs="tabs" label="What to look at" class="w-fit" />

          <div class="ml-auto flex flex-wrap items-center gap-1.5">
            <span class="text-xs uppercase tracking-wider text-muted-foreground">
              Skills applied
            </span>
            <Chip
              v-for="id in picked"
              :key="id"
              :label="nameOf(id)"
              :tone="verdictFor(id) === undefined ? 'default' : verdictFor(id) ? 'success' : 'danger'"
              removable
              @remove="drop(id)"
            >
              {{ nameOf(id) }}
            </Chip>
            <Chip v-if="!picked.length" tone="muted">Whatever applies</Chip>
            <Select
              v-if="spare.length"
              :model-value="adding"
              size="sm"
              aria-label="Add a skill"
              @update:model-value="add"
            >
              <option value="">Add…</option>
              <option v-for="skill in spare" :key="skill.id" :value="skill.id">
                {{ skill.name }}
              </option>
            </Select>
          </div>
        </div>

        <Notice v-if="result.note && !read" tone="info" class="mb-3">{{ result.note }}</Notice>


        <!-- The review, in the order somebody reads one. -->
        <div v-if="tab === 'commits'" class="min-h-0 flex-1 overflow-y-auto">
          <Empty v-if="!commits.length" title="Nothing committed on this branch" compact>
            Everything here is uncommitted work, which is in the diff rather than
            in a commit.
          </Empty>

          <UiCard v-else class="divide-y px-4 py-1">
            <div v-for="commit in commits" :key="commit.sha" class="py-2.5">
              <component
                :is="commit.body ? 'details' : 'div'"
                :class="commit.body && 'group'"
              >
                <component
                  :is="commit.body ? 'summary' : 'div'"
                  class="flex items-baseline gap-2 text-sm"
                  :class="commit.body && 'cursor-pointer list-none'"
                >
                  <GitCommit class="h-3.5 w-3.5 shrink-0 self-center text-muted-foreground" />
                  <span class="min-w-0 flex-1 truncate font-medium">{{ commit.subject }}</span>
                  <span class="shrink-0 font-mono text-xs text-muted-foreground">
                    {{ commit.sha.slice(0, 8) }}
                  </span>
                  <span class="shrink-0 text-xs text-muted-foreground">{{ commit.author }}</span>
                  <span class="shrink-0 text-xs text-muted-foreground">{{ when(commit.at) }}</span>
                </component>
                <p
                  v-if="commit.body"
                  class="mt-2 whitespace-pre-wrap pl-5 text-sm text-muted-foreground"
                >
                  {{ commit.body }}
                </p>
              </component>
            </div>
          </UiCard>
        </div>

        <div v-else-if="tab === 'overview'" class="min-h-0 flex-1 overflow-y-auto">
          <Empty v-if="!files.length" title="No changes" compact>
            {{ where?.branch || 'This checkout' }} matches {{ where?.against || 'its base' }}.
            Nothing to read.
          </Empty>
          <Empty v-else-if="!anything" title="Read, not judged" compact>
            {{ result.note || 'No model was asked, so nothing was judged.' }}
          </Empty>

          <UiCard v-else class="divide-y px-5 py-1">
          <Section v-if="summary?.overview" title="What this change does" tone="info">
            <template #icon><FileText class="h-4 w-4" /></template>
            <Markdown :source="summary.overview" />
          </Section>

          <!-- One line each. A skill that has a page to say about a commit
               message says it to whoever opens it, not to everybody. -->
          <Section
            v-if="unmet.length || overall.length"
            title="Against what this team wrote down"
            tone="warning"
            :count="unmet.length + overall.length"
            collapsible
            closed
          >
            <template #icon><TriangleAlert class="h-4 w-4" /></template>
            <ul class="divide-y">
              <li v-for="verdict in unmet" :key="verdict.skill" class="py-2 first:pt-0">
                <details>
                  <summary class="flex cursor-pointer items-center gap-2 text-sm">
                    <DotIndicator tone="danger" :title="`${verdict.skill} is not met`" />
                    <span class="font-medium">{{ verdict.skill }}</span>
                    <span class="min-w-0 flex-1 truncate text-muted-foreground">
                      {{ (verdict.note || '').split('\n')[0] }}
                    </span>
                  </summary>
                  <Markdown
                    v-if="verdict.note"
                    :source="verdict.note"
                    class="mt-2 pl-5 text-sm text-muted-foreground"
                  />
                </details>
              </li>
              <li v-for="(one, index) in overall" :key="`o${index}`" class="py-2 first:pt-0">
                <details>
                  <summary class="flex cursor-pointer items-center gap-2 text-sm">
                    <DotIndicator
                      :tone="one.severity === 'blocking' ? 'danger' : 'warning'"
                      :title="one.skill || 'A skill'"
                    />
                    <span class="font-medium">{{ one.skill }}</span>
                    <span class="min-w-0 flex-1 truncate text-muted-foreground">
                      {{ (one.detail || '').split('\n')[0] }}
                    </span>
                  </summary>
                  <Markdown
                    :source="one.detail"
                    class="mt-2 pl-5 text-sm text-muted-foreground"
                  />
                </details>
              </li>
            </ul>
          </Section>

          <Section
            v-if="summary?.critical_issues?.length"
            title="Worth stopping for"
            tone="danger"
            :count="summary.critical_issues.length"
            collapsible
          >
            <template #icon><CircleAlert class="h-4 w-4" /></template>
            <ul class="space-y-2">
              <li v-for="(one, index) in summary.critical_issues" :key="index">
                <Markdown :source="one" />
              </li>
            </ul>
          </Section>

          <Section
            v-if="summary?.key_improvements?.length"
            title="Worth changing"
            tone="warning"
            :count="summary.key_improvements.length"
            collapsible
          >
            <template #icon><Lightbulb class="h-4 w-4" /></template>
            <ul class="space-y-2">
              <li v-for="(one, index) in summary.key_improvements" :key="index">
                <Markdown :source="one" />
              </li>
            </ul>
          </Section>

          <Section
            v-if="summary?.minor_suggestions?.length"
            title="Nice to have"
            :count="summary.minor_suggestions.length"
            collapsible
            closed
          >
            <template #icon><Sparkles class="h-4 w-4" /></template>
            <ul class="space-y-2">
              <li v-for="(one, index) in summary.minor_suggestions" :key="index">
                <Markdown :source="one" />
              </li>
            </ul>
          </Section>

          <Section
            v-if="suggestions.length"
            title="Suggestions"
            tone="info"
            :count="suggestions.length"
            collapsible
          >
            <template #icon><Bug class="h-4 w-4" /></template>
            <p class="mb-3 text-sm text-muted-foreground">
              Each one is drawn against the line it is about, under Files.
            </p>
            <ul class="space-y-1.5 text-sm">
              <li v-for="(one, index) in suggestions" :key="index" class="flex gap-2">
                <DotIndicator
                  v-if="one.category"
                  :tone="toneOf(one.category)"
                  :title="one.category"
                  :label="one.category.toLowerCase()"
                  class="mt-0.5 shrink-0"
                />
                <button
                  type="button"
                  class="min-w-0 text-left hover:underline"
                  @click="looking = one.path; tab = 'details'"
                >
                  <span class="font-mono text-xs">{{ one.path }}:{{ one.start_line }}</span>
                  <Markdown :source="one.comment" class="text-muted-foreground" />
                </button>
              </li>
            </ul>
          </Section>

          <Section
            v-for="(text, name) in read?.notes ?? {}"
            :key="name"
            :title="String(name).replace(/_/g, ' ')"
            collapsible
            closed
            class="capitalize"
          >
            <template #icon><FileText class="h-4 w-4" /></template>
            <Markdown :source="text" class="normal-case" />
          </Section>

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
