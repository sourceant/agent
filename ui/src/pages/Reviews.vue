<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Diff,
  Notice,
  PageHead,
  Select,
} from '@sourceant/design'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import {
  Check,
  FileCode,
  GitBranch,
  Loader2,
  ScrollText,
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

const { repositories, chosen, error, fetchRepositories } = useRepositories()
const running = ref(false)
const judging = ref(false)
const result = ref(null)
const hasModel = ref(false)
const looking = ref('')
const only = ref('')
const skills = ref([])

const files = computed(() => result.value?.changed ?? [])
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
const notesFor = (path) => findings.value.filter((one) => one.path === path)

// How loudly a file is asking to be looked at, so the worst sorts first.
const weight = computed(() => {
  const by = {}
  for (const one of findings.value) {
    if (!one.path) continue
    by[one.path] = (by[one.path] ?? 0) + (one.severity === 'blocking' ? 10 : 1)
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
    const { id } = await api.startReview(chosen.value, {
      useModel,
      skills: only.value ? [only.value] : [],
    })
    await collect(id, useModel)
  } catch (caught) {
    error.value = caught.message
    flag.value = false
  }
}

async function collect(id, useModel) {
  const flag = useModel ? judging : running
  let answered
  try {
    answered = await api.reviewed(id)
  } catch (caught) {
    error.value = caught.message
    flag.value = false
    return
  }

  if (answered.status === 'running') {
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
}

watch(chosen, () => {
  stopAsking()
  result.value = null
  only.value = ''
  loadSkills()
})

onUnmounted(stopAsking)

onMounted(async () => {
  await fetchRepositories()
  await Promise.all([checkModel(), loadSkills()])
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

        <!-- Where the choosing gets it wrong, somebody names the one they
             wanted rather than arguing with the matcher. -->
        <Select v-if="skills.length" v-model="only" size="sm" aria-label="Which skills">
          <option value="">Whatever applies</option>
          <option v-for="skill in skills" :key="skill.id" :value="skill.id">
            Only {{ skill.name }}
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

    <EmptyMachine v-if="repositories.length === 0" />

    <template v-else>
      <Notice v-if="!hasModel" tone="info" class="mb-4">
        No model is configured, so nothing here can be judged. Reading what changed needs nothing.
        Choose a model in Settings to have the work read against your skills.
      </Notice>

      <UiCard
        v-if="!result"
        class="flex flex-1 flex-col items-center justify-center p-12 text-center"
      >
        <ShieldCheck class="mb-3 h-8 w-8 text-muted-foreground" />
        <p class="font-medium">Nothing read yet.</p>
        <p class="mx-auto mt-1 max-w-md text-sm text-muted-foreground">
          Everything comes off this checkout, so work you have not pushed, or not committed, still
          gets an answer.
        </p>
      </UiCard>

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

        <Notice
          v-for="(one, index) in overall"
          :key="index"
          :tone="one.severity === 'blocking' ? 'danger' : 'warning'"
          class="mb-3"
        >
          {{ one.detail }}
          <span class="text-muted-foreground">— {{ one.skill }}</span>
        </Notice>

        <div class="grid min-h-0 flex-1 gap-3 lg:grid-cols-[18rem_1fr]">
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
                    v-if="notesFor(file.path).length"
                    :variant="weight[file.path] >= 10 ? 'destructive' : 'secondary'"
                  >
                    {{ notesFor(file.path).length }}
                  </UiBadge>
                  <span class="shrink-0 text-[10px] uppercase text-muted-foreground">
                    {{ file.change.slice(0, 3) }}
                  </span>
                </button>
              </li>
            </ul>

            <div v-if="verdicts.length" class="border-t">
              <div class="px-3 py-2 text-xs uppercase tracking-wider text-muted-foreground">
                Skills it was read against
              </div>
              <ul class="max-h-48 overflow-y-auto pb-2">
                <li
                  v-for="verdict in verdicts"
                  :key="verdict.skill"
                  class="flex items-start gap-2 px-3 py-1.5 text-sm"
                >
                  <ScrollText
                    class="mt-0.5 h-3.5 w-3.5 shrink-0"
                    :class="verdict.passed ? 'text-success' : 'text-destructive'"
                  />
                  <span class="min-w-0 flex-1">
                    <span class="block truncate" :title="verdict.note">{{ verdict.skill }}</span>
                    <span class="text-xs text-muted-foreground">
                      {{ verdict.passed ? 'satisfied' : `${verdict.findings.length} to look at` }}
                    </span>
                  </span>
                </li>
              </ul>
            </div>
          </UiCard>

          <!-- What changed in it, with anything said about a line against it. -->
          <div class="min-h-0 overflow-y-auto">
            <template v-if="showing">
              <div class="mb-2 flex flex-wrap items-center gap-2">
                <span class="break-all font-mono text-sm">{{ showing.path }}</span>
                <UiBadge variant="secondary">{{ showing.change }}</UiBadge>
              </div>
              <Diff :patch="showing.patch" :notes="notesFor(showing.path)" />
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
