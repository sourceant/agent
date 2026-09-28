<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Card as UiCard,
  Markdown,
  Notice,
  PageHead,
} from '@sourceant/design'
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, Copy, Pencil, ScrollText, Trash2 } from 'lucide-vue-next'
import { useUp } from '~/composables/useUp'
import { useRepositories } from '~/composables/useRepositories'
import { api } from '~/api'

/* Reading a skill.
 *
 * Reading and changing are two screens. One that is both leaves somebody
 * looking at form controls when they came to read a document, and one keystroke
 * from changing it by accident.
 */

const REPOSITORY = 'repository'
const GLOBAL = 'global'
const OURS = [REPOSITORY, GLOBAL]

const route = useRoute()
const router = useRouter()
const up = useUp()
const { chosen, fetchRepositories } = useRepositories()

const id = computed(() => String(route.params.id ?? ''))
const skill = ref(null)
const uses = ref([])
const loading = ref(true)
const problem = ref('')

const theirs = computed(() => !!skill.value && !OURS.includes(skill.value.origin))
const where = computed(() => {
  if (!skill.value) return ''
  if (theirs.value) return skill.value.origin
  return skill.value.origin === GLOBAL ? 'everywhere' : chosen.value
})

const on = (use) => skill.value?.applications?.[use] !== false

async function load() {
  loading.value = true
  problem.value = ''
  if (!uses.value.length) uses.value = await api.uses().catch(() => [])
  try {
    skill.value = await api.skill(id.value, chosen.value)
  } catch (caught) {
    problem.value = caught.message
    skill.value = null
  } finally {
    loading.value = false
  }
}

async function forget() {
  if (!confirm(`Forget ${skill.value.name}?\n\nIt stops being read for ${where.value}.`)) return
  try {
    await api.forgetSkill(
      skill.value.origin === GLOBAL ? '' : chosen.value,
      skill.value.origin,
      id.value,
    )
    router.push('/skills')
  } catch (caught) {
    problem.value = caught.message
  }
}

watch(id, load)
onMounted(async () => {
  await fetchRepositories()
  await load()
})
</script>

<template>
  <div>
    <PageHead
      pillar="review"
      :title="skill?.name || id"
      :sub="skill?.path"
      :mono="!!skill?.path"
    >
      <template #back>
        <UiButton variant="ghost" size="icon" aria-label="Back to skills" @click="up('/skills')">
          <ArrowLeft class="h-4 w-4" />
        </UiButton>
      </template>
      <template #icon><ScrollText class="h-5 w-5" /></template>
      <template #badges>
        <UiBadge v-if="skill" :variant="theirs ? 'outline' : 'success'">{{ where }}</UiBadge>
        <UiBadge v-if="skill && !skill.automatic" variant="outline">manual</UiBadge>
      </template>
      <template v-if="skill" #actions>
        <UiButton
          v-if="!theirs"
          variant="ghost"
          size="icon"
          aria-label="Forget this skill"
          @click="forget"
        >
          <Trash2 class="h-4 w-4" />
        </UiButton>
        <UiButton size="sm" @click="router.push(`/skills/${id}/edit`)">
          <component :is="theirs ? Copy : Pencil" class="mr-1.5 h-3.5 w-3.5" />
          {{ theirs ? 'Copy' : 'Edit' }}
        </UiButton>
      </template>
    </PageHead>

    <Notice v-if="problem" tone="danger" class="mb-4">
      {{ problem }}
    </Notice>

    <p v-if="loading" class="py-10 text-center text-sm text-muted-foreground">Loading.</p>

    <template v-else-if="skill">
      <Notice v-if="theirs" tone="info" class="mb-4">
        Read-only file. Copy keeps your own, which is then the one used.
      </Notice>

      <UiCard class="mb-3 p-5">
        <p class="text-sm">{{ skill.description }}</p>

        <dl class="mt-4 grid gap-4 border-t pt-4 text-sm sm:grid-cols-3">
          <div>
            <dt class="mb-1.5 text-xs uppercase tracking-wide text-muted-foreground">Skill uses</dt>
            <dd class="flex flex-wrap gap-1">
              <span
                v-for="use in uses"
                :key="use.id"
                :class="[
                  'rounded-full border px-2 py-0.5 text-[11px]',
                  on(use.id)
                    ? 'border-primary/30 bg-primary/10 text-foreground'
                    : 'border-border text-muted-foreground line-through',
                ]"
              >
                {{ use.label }}
              </span>
            </dd>
          </div>
          <div>
            <dt class="mb-1.5 text-xs uppercase tracking-wide text-muted-foreground">Files</dt>
            <dd v-if="skill.paths?.length" class="flex flex-wrap gap-1 font-mono text-xs">
              <span v-for="path in skill.paths" :key="path" class="rounded bg-muted px-1.5 py-0.5">
                {{ path }}
              </span>
            </dd>
            <dd v-else class="text-muted-foreground">Any</dd>
          </div>
          <div>
            <dt class="mb-1.5 text-xs uppercase tracking-wide text-muted-foreground">Selection</dt>
            <dd>{{ skill.automatic ? 'Automatic' : 'Only when named' }}</dd>
          </div>
        </dl>
      </UiCard>

      <UiCard v-if="skill.body" class="p-5">
        <Markdown :source="skill.body" />
      </UiCard>
      <UiCard v-else class="p-10 text-center">
        <ScrollText class="mx-auto mb-3 h-8 w-8 text-muted-foreground" />
        <p class="font-medium">Nothing written down.</p>
        <p class="mx-auto mt-1 max-w-md text-sm text-muted-foreground">
          The body is what a person reads and what a model is given when your work is checked
          against this skill.
        </p>
        <UiButton v-if="!theirs" class="mt-4" size="sm" @click="router.push(`/skills/${id}/edit`)">
          <Pencil class="mr-1.5 h-3.5 w-3.5" />
          Write it
        </UiButton>
      </UiCard>
    </template>
  </div>
</template>
