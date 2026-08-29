<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Input,
  ItemCard,
  Modal as UiModal,
  PageHead,
  Select,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { BookOpenCheck, Eye, ScrollText, Search } from 'lucide-vue-next'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* The rules a team already wrote down for whatever reads their code. People
 * have been teaching their coding agents how work is done here for a while, in
 * a folder the agent reads; this is that folder, and nothing else in the
 * product could see it. It is read only: these are files somebody owns and
 * edits with their editor. */

const { repositories, chosen, error, fetchRepositories } = useRepositories()
const skills = ref([])
const term = ref('')
const reading = ref(null)

const shown = computed(() => {
  const wanted = term.value.trim().toLowerCase()
  if (!wanted) return skills.value
  return skills.value.filter(
    (skill) =>
      skill.name.toLowerCase().includes(wanted) ||
      skill.description.toLowerCase().includes(wanted),
  )
})

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

async function open(skill) {
  try {
    reading.value = await api.skill(skill.id, chosen.value)
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
      sub="The rules this machine and its repositories already hold."
    >
      <template #icon><BookOpenCheck class="h-6 w-6" /></template>
      <template #actions>
        <Select v-if="repositories.length > 1" v-model="chosen" aria-label="Repository">
          <option v-for="repository in repositories" :key="repository.name" :value="repository.name">
            {{ repository.name }}
          </option>
        </Select>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
          <Input v-model="term" placeholder="Find a rule" class="w-56 pl-8" aria-label="Find a rule" />
        </div>
      </template>
    </PageHead>

    <p v-if="error" class="mb-4 rounded-md border border-destructive/40 bg-destructive/10 px-4 py-3 text-sm">
      {{ error }}
    </p>

    <div v-if="shown.length" class="space-y-3">
      <ItemCard
        v-for="skill in shown"
        :key="skill.id"
        :title="skill.name"
        :subtitle="skill.path"
        pillar="review"
      >
        <template #icon><ScrollText class="h-5 w-5" /></template>
        <template #badges><UiBadge variant="outline">{{ skill.origin }}</UiBadge></template>
        <p class="text-sm text-muted-foreground">{{ skill.description }}</p>
        <template #actions>
          <UiButton variant="ghost" size="icon" :aria-label="`Read ${skill.name}`" @click="open(skill)">
            <Eye class="h-4 w-4" />
          </UiButton>
        </template>
      </ItemCard>
    </div>

    <UiCard v-else class="p-10 text-center">
      <ScrollText class="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
      <p class="font-medium">Nothing written down yet.</p>
      <p class="mt-1 text-sm text-muted-foreground">
        A rule is a folder with a SKILL.md in it, under .claude/skills, .codex/skills or
        .sourceant/skills. Anything already taught to a coding agent is read from here.
      </p>
    </UiCard>

    <UiModal :open="!!reading" max-width="2xl" @close="reading = null">
      <div v-if="reading">
        <h2 class="pr-8 text-lg font-semibold">{{ reading.name }}</h2>
        <p class="mt-1 text-sm text-muted-foreground">{{ reading.description }}</p>
        <p class="mt-1 break-all font-mono text-xs text-muted-foreground">{{ reading.path }}</p>
        <pre class="mt-4 max-h-[55vh] overflow-auto whitespace-pre-wrap rounded-md bg-muted/50 p-4 text-sm">{{ reading.body }}</pre>
      </div>
    </UiModal>
  </div>
</template>
