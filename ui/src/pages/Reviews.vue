<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Field,
  Input,
  ItemCard,
  PageHead,
  Select,
  Tabs,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import {
  BookOpen,
  Check,
  FileCode,
  Loader2,
  ScrollText,
  ShieldCheck,
  TriangleAlert,
  Wand2,
} from 'lucide-vue-next'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* Work read before anybody else has been asked to read it. Everything here
 * comes off the checkout itself, so a branch nobody has pushed still gets an
 * answer. */

const { repositories, chosen, error, fetchRepositories } = useRepositories()
const title = ref('')
const against = ref('')
const running = ref(false)
const judging = ref(false)
const result = ref(null)
const hasModel = ref(false)
const view = ref('verdicts')

const views = [
  { id: 'verdicts', label: 'What the rules say' },
  { id: 'changed', label: 'What changed' },
  { id: 'rules', label: 'Rules that apply' },
  { id: 'knowledge', label: 'What we know' },
]

const blocking = computed(() =>
  (result.value?.verdicts ?? []).flatMap((verdict) =>
    verdict.findings.filter((finding) => finding.severity === 'blocking'),
  ),
)

const advisory = computed(() =>
  (result.value?.verdicts ?? []).flatMap((verdict) =>
    verdict.findings.filter((finding) => finding.severity !== 'blocking'),
  ),
)

// Judging costs whatever the model costs, so it is only offered where there is
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

async function run(useModel) {
  const flag = useModel ? judging : running
  flag.value = true
  try {
    result.value = await api.review(chosen.value, {
      against: against.value,
      title: title.value,
      useModel,
    })
    view.value = useModel ? 'verdicts' : 'changed'
    error.value = ''
  } catch (caught) {
    result.value = null
    error.value = caught.message
  } finally {
    flag.value = false
  }
}

watch(chosen, () => {
  result.value = null
})

onMounted(async () => {
  await fetchRepositories()
  await checkModel()
})
</script>

<template>
  <div>
    <PageHead
      pillar="review"
      title="Reviews"
      sub="Your work read here, before anybody else is asked to read it."
    >
      <template #icon><ShieldCheck class="h-6 w-6" /></template>
      <template #actions>
        <Select v-if="repositories.length > 1" v-model="chosen" aria-label="Repository">
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>
        <UiButton v-if="repositories.length" variant="outline" :disabled="running" @click="run(false)">
          <Loader2 v-if="running" class="mr-2 h-4 w-4 animate-spin" />
          <FileCode v-else class="mr-2 h-4 w-4" />
          {{ running ? 'Reading…' : 'Read what changed' }}
        </UiButton>
        <UiButton
          v-if="repositories.length && hasModel"
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

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <EmptyMachine v-if="repositories.length === 0" />

    <template v-else>
      <UiCard class="mb-6 grid gap-4 p-5 sm:grid-cols-2">
        <Field
          label="What the work is"
          for="review-title"
          hint="What a pull request would be called. It decides which rules are picked."
        >
          <Input id="review-title" v-model="title" placeholder="Retry a failed charge three times" />
        </Field>
        <Field
          label="Going back to"
          for="review-against"
          hint="Left empty, whatever this checkout's remote calls its own head."
        >
          <Input id="review-against" v-model="against" placeholder="dev" />
        </Field>
      </UiCard>

      <p
        v-if="!hasModel"
        class="mb-6 rounded-md border border-primary/30 bg-primary/10 px-4 py-3 text-sm"
      >
        No model is configured, so nothing here can be judged. Reading what changed and which rules
        apply needs nothing. Choose a model in Settings to have the work reviewed against them.
      </p>

      <template v-if="result">
        <UiCard
          class="mb-6 flex flex-wrap items-center gap-3 p-5"
          :class="result.ready ? 'border-success/40' : 'border-destructive/40'"
        >
          <component
            :is="result.ready ? Check : TriangleAlert"
            class="h-5 w-5 shrink-0"
            :class="result.ready ? 'text-success' : 'text-destructive'"
          />
          <p class="font-medium">
            <template v-if="result.note">{{ result.note }}</template>
            <template v-else-if="result.ready">Nothing here says this is not ready.</template>
            <template v-else>
              {{ blocking.length }} thing{{ blocking.length === 1 ? '' : 's' }} to fix before this
              is proposed to anyone.
            </template>
          </p>
          <span class="ml-auto text-sm text-muted-foreground">
            {{ result.changed.length }} file{{ result.changed.length === 1 ? '' : 's' }} changed
            <template v-if="advisory.length">
              · {{ advisory.length }} suggestion{{ advisory.length === 1 ? '' : 's' }}
            </template>
          </span>
        </UiCard>

        <Tabs v-model="view" :tabs="views" label="What to look at" class="mb-4" />

        <div v-if="view === 'verdicts'" class="space-y-3">
          <ItemCard
            v-for="verdict in result.verdicts"
            :key="verdict.skill"
            :title="verdict.skill"
            pillar="review"
          >
            <template #icon><ScrollText class="h-5 w-5" /></template>
            <template #badges>
              <UiBadge :variant="verdict.passed ? 'success' : 'destructive'">
                {{ verdict.passed ? 'satisfied' : 'not satisfied' }}
              </UiBadge>
            </template>

            <p v-if="verdict.note" class="text-sm text-muted-foreground">{{ verdict.note }}</p>
            <ul v-if="verdict.findings.length" class="mt-3 space-y-2">
              <li
                v-for="(finding, index) in verdict.findings"
                :key="index"
                class="rounded-md border px-3 py-2 text-sm"
                :class="finding.severity === 'blocking' ? 'border-destructive/40 bg-destructive/5' : ''"
              >
                <div class="mb-1 flex flex-wrap items-center gap-2">
                  <UiBadge :variant="finding.severity === 'blocking' ? 'destructive' : 'secondary'">
                    {{ finding.severity }}
                  </UiBadge>
                  <span v-if="finding.path" class="font-mono text-xs text-muted-foreground">
                    {{ finding.path }}<template v-if="finding.line">:{{ finding.line }}</template>
                  </span>
                </div>
                {{ finding.detail }}
              </li>
            </ul>
          </ItemCard>

          <p v-if="!result.verdicts.length" class="py-8 text-center text-sm text-muted-foreground">
            Nothing has been judged yet.
          </p>
        </div>

        <div v-else-if="view === 'changed'" class="space-y-3">
          <ItemCard
            v-for="file in result.changed"
            :key="file.path"
            :title="file.path"
            pillar="graph"
          >
            <template #icon><FileCode class="h-5 w-5" /></template>
            <template #badges><UiBadge variant="secondary">{{ file.change }}</UiBadge></template>
          </ItemCard>

          <p v-if="!result.changed.length" class="py-8 text-center text-sm text-muted-foreground">
            Nothing has changed in this checkout.
          </p>
        </div>

        <div v-else-if="view === 'rules'" class="space-y-3">
          <ItemCard
            v-for="skill in result.skills"
            :key="skill.id"
            :title="skill.name"
            :subtitle="skill.path"
            pillar="review"
          >
            <template #icon><ScrollText class="h-5 w-5" /></template>
            <template #badges><UiBadge variant="outline">{{ skill.origin }}</UiBadge></template>
            <p class="text-sm text-muted-foreground">{{ skill.description }}</p>
          </ItemCard>

          <p v-if="!result.skills.length" class="py-8 text-center text-sm text-muted-foreground">
            No rule on this machine bears on what changed here.
          </p>
        </div>

        <div v-else class="space-y-3">
          <ItemCard
            v-for="item in result.knowledge"
            :key="item.id"
            :title="item.id"
            pillar="memory"
          >
            <template #icon><BookOpen class="h-5 w-5" /></template>
            <template #badges><UiBadge variant="secondary">{{ item.kind }}</UiBadge></template>
            <p class="text-sm text-muted-foreground">{{ item.summary }}</p>
          </ItemCard>

          <p v-if="!result.knowledge.length" class="py-8 text-center text-sm text-muted-foreground">
            Nothing has been recorded about this repository yet.
          </p>
        </div>
      </template>

      <p v-else class="py-12 text-center text-sm text-muted-foreground">
        Nothing read yet. Say what the work is, then read what changed.
      </p>
    </template>
  </div>
</template>
