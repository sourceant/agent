<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Input,
  Notice,
  PageHead,
  Select,
  Table,
  Tabs,
  ToggleList,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { BookOpenCheck, Plus, ScrollText, Search, Trash2 } from 'lucide-vue-next'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* What a team has written down about how work here is done.
 *
 * Some of it already exists: people have been teaching their coding agents for
 * a while, in a folder the agent reads, and nothing else in this product could
 * see it. Those are read and never written, because they are that agent's files
 * and are frequently a link into a checkout of their own.
 *
 * The rest is in somebody's head. That gets written here, into the repository
 * it is about so the team gets it by pulling, or onto this machine for what
 * somebody wants everywhere.
 *
 * A column rather than a card each: the question asked of sixty of them is
 * which ones a review reads, and that is one glance down a column.
 */

const REPOSITORY = 'repository'
const GLOBAL = 'global'
const OURS = [REPOSITORY, GLOBAL]

/* What a skill is for is one question with one answer: the purposes it applies
 * to. The file says it, and a skill that belongs to a coding agent cannot be
 * edited, so for the purpose this product reads the answer is kept here as well
 * and outranks the file. */
const REVIEW = 'review'
const NEVER = 'skills.never_in_reviews'
const ALWAYS = 'skills.always_in_reviews'

const KINDS = {
  guidance: 'guidance',
  'review-pass': 'review pass',
  'initialization-pass': 'initialization pass',
}

const columns = [
  { id: 'skill', label: 'Skill' },
  { id: 'applies', label: 'Applies to', width: '20rem' },
  { id: 'kind', label: 'Read as', narrow: true },
  { id: 'kept', label: 'Kept', narrow: true },
  { id: 'actions', label: '', align: 'right', width: '3rem' },
]

const router = useRouter()
const { repositories, chosen, error, fetchRepositories } = useRepositories()
const skills = ref([])
const term = ref('')
const where = ref('all')
const lists = ref({ [NEVER]: [], [ALWAYS]: [] })
const onlyRead = ref(false)

const wheres = [
  { id: 'all', label: 'All' },
  { id: REPOSITORY, label: 'This repository' },
  { id: GLOBAL, label: 'Everywhere' },
  { id: 'agents', label: 'Your coding agents' },
]

const shown = computed(() => {
  const wanted = term.value.trim().toLowerCase()
  return skills.value.filter((skill) => {
    if (where.value === REPOSITORY && skill.origin !== REPOSITORY) return false
    if (where.value === GLOBAL && skill.origin !== GLOBAL) return false
    if (where.value === 'agents' && OURS.includes(skill.origin)) return false
    if (onlyRead.value && !read(skill)) return false
    if (!wanted) return true
    return (
      skill.name.toLowerCase().includes(wanted) ||
      skill.description.toLowerCase().includes(wanted)
    )
  })
})

const lines = (value) =>
  String(value ?? '')
    .split('\n')
    .map((one) => one.trim())
    .filter(Boolean)

const ours = (skill) => OURS.includes(skill.origin)
const home = (skill) =>
  skill.origin === GLOBAL ? 'everywhere' : skill.origin === REPOSITORY ? chosen.value : skill.origin

// What the skill is for, with what this machine said about reviews on top of
// what its file says.
function applies(skill) {
  const said = { ...(skill.applications ?? {}) }
  if (lists.value[NEVER].includes(skill.id)) said[REVIEW] = false
  else if (lists.value[ALWAYS].includes(skill.id)) said[REVIEW] = true
  else if (!(REVIEW in said) && skill.reviews !== null) said[REVIEW] = skill.reviews
  return said
}

// Whether a review would read it at all, which is what the filter narrows to.
function read(skill) {
  const said = applies(skill)[REVIEW]
  if (said === false) return false
  if (said === true) return true
  return skill.automatic
}

async function decide(skill, said) {
  const was = applies(skill)
  const purposes = new Set([...Object.keys(was), ...Object.keys(said)])
  const changed = [...purposes].filter((purpose) => was[purpose] !== said[purpose])
  if (!changed.length) return

  if (!ours(skill) && changed.some((purpose) => purpose !== REVIEW)) {
    error.value = `${skill.name} belongs to ${skill.origin}. Save your own copy to change what it is for.`
    return
  }
  try {
    if (ours(skill)) await state(skill, said)
    else await override(skill, said[REVIEW])
    error.value = ''
  } catch (caught) {
    error.value = caught.message
  }
  await load()
}

// One of ours: what it is for belongs in its own file, so it travels with it.
async function state(skill, said) {
  const full = await api.skill(skill.id, skill.origin === REPOSITORY ? chosen.value : '')
  await api.recordSkill({
    scope: skill.origin,
    repository: skill.origin === REPOSITORY ? chosen.value : '',
    id: full.id,
    name: full.name,
    description: full.description,
    body: full.body ?? '',
    paths: full.paths ?? [],
    type: full.type ?? '',
    applications: said,
  })
  // A file that now says it for itself needs nothing said here.
  await override(skill, undefined)
}

// Somebody else's file: the answer for reviews is kept on this machine instead.
async function override(skill, said) {
  const next = {
    [NEVER]: lists.value[NEVER].filter((id) => id !== skill.id),
    [ALWAYS]: lists.value[ALWAYS].filter((id) => id !== skill.id),
  }
  if (said === false) next[NEVER].push(skill.id)
  if (said === true) next[ALWAYS].push(skill.id)
  for (const key of [NEVER, ALWAYS]) {
    if (next[key].join('\n') !== lists.value[key].join('\n')) {
      await api.setSetting(key, next[key].join('\n'))
    }
  }
  lists.value = next
}

async function load() {
  try {
    const [page, settings] = await Promise.all([api.skills(chosen.value), api.settings()])
    skills.value = page.skills
    lists.value = {
      [NEVER]: lines(settings.find((one) => one.key === NEVER)?.value),
      [ALWAYS]: lines(settings.find((one) => one.key === ALWAYS)?.value),
    }
    error.value = ''
  } catch (caught) {
    skills.value = []
    error.value = caught.message
  }
}

async function forget(skill) {
  if (!confirm(`Forget ${skill.name}?\n\nThe file is removed from ${home(skill)}.`)) return
  try {
    await api.forgetSkill(
      skill.origin === GLOBAL ? '' : chosen.value,
      skill.origin,
      skill.id,
    )
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
      sub="What your work is read against: this repository's, this machine's, and your own."
    >
      <template #icon><BookOpenCheck class="h-6 w-6" /></template>
      <template #actions>
        <Select v-if="repositories.length > 1" v-model="chosen" size="sm" aria-label="Repository">
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="term" size="sm" placeholder="Find a skill" class="w-48 pl-8" aria-label="Find a skill" />
        </div>
        <UiButton v-if="repositories.length" size="sm" variant="glow" @click="router.push({ path: '/skills/new', query: { for: chosen } })">
          <Plus class="mr-2 h-4 w-4" />
          Write one down
        </UiButton>
      </template>
    </PageHead>

    <Notice v-if="error" tone="danger" class="mb-4">
      {{ error }}
    </Notice>

    <EmptyMachine v-if="repositories.length === 0" />

    <template v-else>
      <div class="mb-2 flex flex-wrap items-center justify-between gap-3">
        <Tabs v-model="where" :tabs="wheres" label="Where they are kept" />
        <label class="flex items-center gap-2 text-sm text-muted-foreground">
          <input v-model="onlyRead" type="checkbox" class="h-3.5 w-3.5 rounded border">
          Only what a review reads
        </label>
      </div>
      <p class="mb-4 text-xs text-muted-foreground">
        A purpose nobody said is decided by the change: a skill is picked when its wording or its
        files match what changed, and at most five are read against one change.
      </p>

      <Table v-if="shown.length" :columns="columns" :rows="shown" row-key="id" label="Skills">
        <template #skill="{ row }">
          <button
            type="button"
            class="block max-w-xl text-left"
            @click="router.push(`/skills/${row.id}`)"
          >
            <span class="block font-medium">{{ row.name }}</span>
            <span class="block truncate text-xs text-muted-foreground">{{ row.description }}</span>
          </button>
        </template>

        <template #applies="{ row }">
          <ToggleList
            :model-value="applies(row)"
            :suggestions="['review']"
            :addable="false"
            :refusal="ours(row) ? '' : `Only reviews can be answered for ${row.origin}'s own file.`"
            @update:model-value="decide(row, $event)"
          />
        </template>

        <template #kind="{ row }">
          <span class="text-xs text-muted-foreground">{{ KINDS[row.type] ?? row.type }}</span>
        </template>

        <template #kept="{ row }">
          <UiBadge :variant="ours(row) ? 'success' : 'outline'">{{ home(row) }}</UiBadge>
        </template>

        <template #actions="{ row }">
          <UiButton
            v-if="ours(row)"
            variant="ghost"
            size="icon"
            :aria-label="`Forget ${row.name}`"
            @click="forget(row)"
          >
            <Trash2 class="h-4 w-4" />
          </UiButton>
        </template>
      </Table>

      <UiCard v-else class="p-10 text-center">
        <ScrollText class="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
        <p class="font-medium">Nothing written down here yet.</p>
        <p class="mx-auto mt-1 max-w-lg text-sm text-muted-foreground">
          A skill says when it applies and what it says to do. Anything you have already taught
          Claude or Codex is read from their own folders, and anything your team committed is read
          from the repository. What you write here is kept beside the index rather than in
          anybody's checkout.
        </p>
        <UiButton class="mt-4" variant="glow" @click="router.push({ path: '/skills/new', query: { for: chosen } })">
          <Plus class="mr-2 h-4 w-4" />
          Write one down
        </UiButton>
      </UiCard>
    </template>
  </div>
</template>
