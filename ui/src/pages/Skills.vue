<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Input,
  ItemCard,
  Notice,
  PageHead,
  Select,
  Tabs,
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
 */

const REPOSITORY = 'repository'
const GLOBAL = 'global'
const OURS = [REPOSITORY, GLOBAL]

/* Whether a skill is read against a change is two questions. Its author
 * answered one of them in the file; the other belongs to whoever runs this
 * machine, and is the only answer available for the folders a coding agent
 * syncs, which nothing here may edit. */
const NEVER = 'skills.never_in_reviews'
const ALWAYS = 'skills.always_in_reviews'

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

// What this machine said, which outranks the author.
function decided(skill) {
  if (lists.value[NEVER].includes(skill.id)) return 'never'
  if (lists.value[ALWAYS].includes(skill.id)) return 'always'
  return 'relevant'
}

// What the author said, worth showing only where nobody here has answered.
function author(skill) {
  if (decided(skill) !== 'relevant') return ''
  if (skill.reviews === true) return 'its author says always'
  if (skill.reviews === false) return 'its author says never'
  if (!skill.automatic) return 'only when you invoke it'
  return ''
}

// Whether a review would read it at all, which is what the filter narrows to.
function read(skill) {
  const mine = decided(skill)
  if (mine !== 'relevant') return mine === 'always'
  if (skill.reviews === false) return false
  return skill.automatic || skill.reviews === true
}

async function decide(skill, choice) {
  const next = {
    [NEVER]: lists.value[NEVER].filter((id) => id !== skill.id),
    [ALWAYS]: lists.value[ALWAYS].filter((id) => id !== skill.id),
  }
  if (choice === 'never') next[NEVER].push(skill.id)
  if (choice === 'always') next[ALWAYS].push(skill.id)
  try {
    for (const key of [NEVER, ALWAYS]) {
      if (next[key].join('\n') !== lists.value[key].join('\n')) {
        await api.setSetting(key, next[key].join('\n'))
      }
    }
    lists.value = next
    error.value = ''
  } catch (caught) {
    error.value = caught.message
  }
}

const ours = (skill) => OURS.includes(skill.origin)
const home = (skill) =>
  skill.origin === GLOBAL ? 'everywhere' : skill.origin === REPOSITORY ? chosen.value : skill.origin

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
        At most five skills are read against one change. Anything set to always counts towards
        those five.
      </p>

      <div v-if="shown.length" class="space-y-3">
        <ItemCard
          v-for="skill in shown"
          :key="skill.id"
          :title="skill.name"
          :subtitle="skill.path"
          pillar="review"
          hover
          class="cursor-pointer"
          @click="router.push(`/skills/${skill.id}`)"
        >
          <template #icon><ScrollText class="h-5 w-5" /></template>
          <template #badges>
            <UiBadge :variant="ours(skill) ? 'success' : 'outline'">{{ home(skill) }}</UiBadge>
          </template>
          <p class="text-sm text-muted-foreground">{{ skill.description }}</p>
          <template #meta>
            <Select
              :model-value="decided(skill)"
              size="sm"
              :aria-label="`When ${skill.name} is read against a change`"
              @click.stop
              @change="decide(skill, $event.target.value)"
            >
              <option value="relevant">When it looks relevant</option>
              <option value="always">Always in reviews</option>
              <option value="never">Never in reviews</option>
            </Select>
            <span v-if="author(skill)">{{ author(skill) }}</span>
            <span v-if="skill.paths?.length" class="font-mono">{{ skill.paths.join(' ') }}</span>
          </template>
          <template #actions>
            <UiButton
              v-if="ours(skill)"
              variant="ghost"
              size="icon"
              :aria-label="`Forget ${skill.name}`"
              @click.stop="forget(skill)"
            >
              <Trash2 class="h-4 w-4" />
            </UiButton>
          </template>
        </ItemCard>
      </div>

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
