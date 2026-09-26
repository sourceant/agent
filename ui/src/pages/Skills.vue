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
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { BookOpenCheck, Plus, ScrollText, Search, Trash2 } from 'lucide-vue-next'
import EmptyMachine from '~/components/EmptyMachine.vue'
import { EVERY, useRepositories } from '~/composables/useRepositories'
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

/* A skill is used for everything this product does until a use is turned off,
 * which is stored on the skill itself. A skill belonging to a coding agent
 * cannot be edited, so for code review the answer is kept on this machine,
 * where it outranks the file. */
const REVIEW = 'review'
const NEVER = 'skills.never_in_reviews'

const columns = [
  { id: 'skill', label: 'Skill' },
  { id: 'uses', label: 'Skill uses', width: '22rem' },
  { id: 'source', label: 'Source', narrow: true },
  { id: 'actions', label: '', align: 'right', width: '3rem' },
]

const router = useRouter()
const { repositories, chosen, error, fetchRepositories } = useRepositories({ all: true })
const skills = ref([])
const term = ref('')
const where = ref('all')
const off = ref([])
const uses = ref([])
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

// Whether one use has it. Off is said; anything else is on.
function on(skill, use) {
  if (use === REVIEW && off.value.includes(skill.id)) return false
  return skill.applications?.[use] !== false
}

// Whether a review would read it at all, which is what the filter narrows to.
const read = (skill) => on(skill, REVIEW)

// A use somebody may answer here: a file that belongs to a coding agent is not
// ours to change, and code review is the one this machine keeps its own answer
// for.
const answerable = (skill, use) => ours(skill) || use === REVIEW

async function toggle(skill, use) {
  const wanted = !on(skill, use)
  try {
    if (ours(skill)) await state(skill, use, wanted)
    else await keep(skill, wanted)
    error.value = ''
  } catch (caught) {
    error.value = caught.message
  }
  await load()
}

// One of ours: what it is used for belongs in its own file, so it travels with
// the skill rather than staying on this machine.
async function state(skill, use, wanted) {
  const full = await api.skill(skill.id, skill.origin === REPOSITORY ? chosen.value : '')
  const applications = { ...(full.applications ?? {}) }
  if (wanted) delete applications[use]
  else applications[use] = false
  await api.recordSkill({
    scope: skill.origin,
    repository: skill.origin === REPOSITORY ? chosen.value : '',
    id: full.id,
    name: full.name,
    description: full.description,
    body: full.body ?? '',
    paths: full.paths ?? [],
    type: full.type ?? '',
    automatic: full.automatic ?? true,
    applications,
  })
  if (use === REVIEW) await keep(skill, true)
}

// Somebody else's file: kept on this machine instead.
async function keep(skill, wanted) {
  const next = wanted
    ? off.value.filter((id) => id !== skill.id)
    : [...off.value, skill.id]
  await api.setSetting(NEVER, next.join('\n'))
  off.value = next
}

async function load() {
  try {
    const [page, settings, offered] = await Promise.all([
      api.skills(chosen.value),
      api.settings(),
      uses.value.length ? uses.value : api.uses(),
    ])
    skills.value = page.skills
    off.value = lines(settings.find((one) => one.key === NEVER)?.value)
    uses.value = offered
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
          <option :value="EVERY">All repositories</option>
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
          Add skill
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
          Used by code review
        </label>
      </div>
      <p class="mb-4 text-xs text-muted-foreground">
        Used for everything unless turned off. A review reads five at most.
      </p>

      <Table v-if="shown.length" :columns="columns" :rows="shown" row-key="id" label="Skills">
        <template #skill="{ row }">
          <button
            type="button"
            class="block max-w-xl text-left"
            @click="router.push(`/skills/${row.id}`)"
          >
            <span class="block font-medium">
              {{ row.name }}
              <UiBadge v-if="!row.automatic" variant="outline" class="ml-1.5">manual</UiBadge>
            </span>
            <span class="block truncate text-xs text-muted-foreground">{{ row.description }}</span>
          </button>
        </template>

        <template #uses="{ row }">
          <div class="flex flex-wrap gap-1">
            <button
              v-for="use in uses"
              :key="use.id"
              type="button"
              :disabled="!answerable(row, use.id)"
              :aria-pressed="on(row, use.id)"
              :title="`${use.label}: ${on(row, use.id) ? 'on' : 'off'}`"
              :class="[
                'rounded-full border px-2 py-0.5 text-[11px] transition-colors',
                on(row, use.id)
                  ? 'border-primary/30 bg-primary/10 text-foreground'
                  : 'border-border text-muted-foreground line-through',
                answerable(row, use.id) ? 'hover:border-primary/50' : 'cursor-not-allowed opacity-60',
              ]"
              @click="toggle(row, use.id)"
            >
              {{ use.label }}
            </button>
          </div>
        </template>

        <template #source="{ row }">
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
        <p class="font-medium">No skills here.</p>
        <p class="mx-auto mt-1 max-w-lg text-sm text-muted-foreground">
          Claude's and Codex's own folders are read, as is anything your team committed. What you
          write here is kept beside the index, not in a checkout.
        </p>
        <UiButton class="mt-4" variant="glow" @click="router.push({ path: '/skills/new', query: { for: chosen } })">
          <Plus class="mr-2 h-4 w-4" />
          Add skill
        </UiButton>
      </UiCard>
    </template>
  </div>
</template>
