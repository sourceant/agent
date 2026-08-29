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

/* The rules a team has for its own code.
 *
 * Some of them are already written down: people have been teaching their coding
 * agents how work is done here for a while, in a folder the agent reads, and
 * nothing else in this product could see it. Those are read and never written,
 * because they are somebody's own files and are frequently a link into a
 * checkout of their own.
 *
 * The rest are in somebody's head. Those get written here, into the repository
 * the rule is about, where the rest of the team gets them by pulling.
 */

const MINE = 'repository'

const router = useRouter()
const { repositories, chosen, error, fetchRepositories } = useRepositories()
const skills = ref([])
const term = ref('')
const where = ref('all')

const wheres = [
  { id: 'all', label: 'All' },
  { id: MINE, label: "This repository's" },
  { id: 'machine', label: 'Yours' },
]

const shown = computed(() => {
  const wanted = term.value.trim().toLowerCase()
  return skills.value.filter((skill) => {
    if (where.value === MINE && skill.origin !== MINE) return false
    if (where.value === 'machine' && skill.origin === MINE) return false
    if (!wanted) return true
    return (
      skill.name.toLowerCase().includes(wanted) ||
      skill.description.toLowerCase().includes(wanted)
    )
  })
})

const ours = (skill) => skill.origin === MINE

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
  if (!confirm(`Forget ${skill.name}?\n\nThe file is removed from this repository.`)) return
  try {
    await api.forgetSkill(chosen.value, skill.id)
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
      sub="The rules your work is read against, yours and this repository's."
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
          <Input v-model="term" size="sm" placeholder="Find a rule" class="w-48 pl-8" aria-label="Find a rule" />
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
      <Tabs v-model="where" :tabs="wheres" label="Whose rules" class="mb-4" />

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
            <UiBadge :variant="ours(skill) ? 'success' : 'outline'">
              {{ ours(skill) ? 'this repository' : skill.origin }}
            </UiBadge>
          </template>
          <p class="text-sm text-muted-foreground">{{ skill.description }}</p>
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
          A rule says when it applies and what it requires. This repository's live in
          <code class="font-mono">.sourceant/skills</code>, so the rest of the team gets them by
          pulling. Anything you already taught Claude or Codex is read from your own folders.
        </p>
        <UiButton class="mt-4" variant="glow" @click="router.push('/skills/new')">
          <Plus class="mr-2 h-4 w-4" />
          Write one down
        </UiButton>
      </UiCard>
    </template>
  </div>
</template>
