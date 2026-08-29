<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Input,
  ItemCard,
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
const MACHINE = 'machine'
const OURS = [REPOSITORY, MACHINE]

const router = useRouter()
const { repositories, chosen, error, fetchRepositories } = useRepositories()
const skills = ref([])
const term = ref('')
const where = ref('all')

const wheres = [
  { id: 'all', label: 'All' },
  { id: REPOSITORY, label: 'This repository' },
  { id: MACHINE, label: 'Global' },
  { id: 'agents', label: 'Your coding agents' },
]

const shown = computed(() => {
  const wanted = term.value.trim().toLowerCase()
  return skills.value.filter((skill) => {
    if (where.value === REPOSITORY && skill.origin !== REPOSITORY) return false
    if (where.value === MACHINE && skill.origin !== MACHINE) return false
    if (where.value === 'agents' && OURS.includes(skill.origin)) return false
    if (!wanted) return true
    return (
      skill.name.toLowerCase().includes(wanted) ||
      skill.description.toLowerCase().includes(wanted)
    )
  })
})

const ours = (skill) => OURS.includes(skill.origin)
const home = (skill) =>
  skill.origin === MACHINE ? 'global' : skill.origin === REPOSITORY ? 'this repository' : skill.origin

async function load() {
  try {
    const page = await api.skills(chosen.value)
    skills.value = page.skills
    error.value = ''
  } catch (caught) {
    skills.value = []
    error.value = caught.message
  }
}

async function forget(skill) {
  if (!confirm(`Forget ${skill.name}?\n\nThe file is removed from ${home(skill)}.`)) return
  try {
    await api.forgetSkill(chosen.value, skill.origin, skill.id)
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
        <UiButton v-if="repositories.length" size="sm" variant="glow" @click="router.push('/skills/new')">
          <Plus class="mr-2 h-4 w-4" />
          Write one down
        </UiButton>
      </template>
    </PageHead>

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <EmptyMachine v-if="repositories.length === 0" />

    <template v-else>
      <Tabs v-model="where" :tabs="wheres" label="Where they are kept" class="mb-4" />

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
            <span v-if="skill.reviews === true" class="text-success">always in reviews</span>
            <span v-else-if="skill.reviews === false">not used in reviews</span>
            <span v-else-if="!skill.automatic">only when you invoke it</span>
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
          A skill says when it applies and what it says to do. This repository's live in its
          <code class="font-mono">.sourceant/skills</code>, so the team gets them by pulling;
          this machine's live in <code class="font-mono">~/.sourceant/skills</code>. Anything you
          already taught Claude or Codex is read from their own folders.
        </p>
        <UiButton class="mt-4" variant="glow" @click="router.push('/skills/new')">
          <Plus class="mr-2 h-4 w-4" />
          Write one down
        </UiButton>
      </UiCard>
    </template>
  </div>
</template>
