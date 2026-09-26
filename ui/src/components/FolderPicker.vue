<script setup>
import {
  Badge as UiBadge,
  Button as UiButton,
  Field,
  Input,
  Modal as UiModal,
  Notice,
} from '@sourceant/design'
import { ref, watch } from 'vue'
import { Folder, ChevronUp, Plus, Loader2, Search, X } from 'lucide-vue-next'
import { api } from '~/api'

/* A browser will not tell a page the absolute path of a folder somebody picked:
 * its file picker withholds it. The agent is already on the machine, so it
 * lists directories and this navigates what it lists. */

const props = defineProps({ open: Boolean })
const emit = defineEmits(['close', 'added'])

const listing = ref(null)
const name = ref('')
const busy = ref(false)
const problem = ref('')
const term = ref('')
const searching = ref(false)
/* A search answers with matches from anywhere under home, so the path is what
 * tells two folders of the same name apart. */
const found = ref(false)

async function show(path) {
  try {
    listing.value = await api.browse(path)
    found.value = false
    term.value = ''
    problem.value = ''
  } catch (error) {
    problem.value = error.message
  }
}

let pending
async function search() {
  const wanted = term.value.trim()
  if (!wanted) return show('')
  clearTimeout(pending)
  pending = setTimeout(async () => {
    searching.value = true
    try {
      listing.value = await api.browse('', wanted)
      found.value = true
      problem.value = ''
    } catch (error) {
      problem.value = error.message
    } finally {
      searching.value = false
    }
  }, 250)
}

watch(() => props.open, (open) => {
  if (!open) return
  name.value = ''
  term.value = ''
  found.value = false
  problem.value = ''
  busy.value = false
  show('')
})

async function add() {
  if (!listing.value) return
  busy.value = true
  problem.value = ''
  try {
    await api.addRepository(listing.value.path, name.value.trim())
    emit('added')
    emit('close')
  } catch (error) {
    problem.value = error.message
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <UiModal :open="open" max-width="lg" @close="emit('close')">
    <h2 class="text-lg font-semibold mb-3">Add folder or repository</h2>

    <div class="relative mb-3">
      <Search class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
      <Input
        v-model="term"
        class="pl-9 pr-9"
        placeholder="Search for a folder by name"
        @input="search"
        @keydown.enter.prevent="search"
      />
      <button
        v-if="term"
        type="button"
        class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
        aria-label="Clear the search"
        @click="show('')"
      >
        <X class="h-3.5 w-3.5" />
      </button>
    </div>

    <p class="mb-2 text-xs font-mono text-muted-foreground break-all">
      {{ found ? `Matches under ${listing?.path}` : listing?.path }}
    </p>

    <div class="h-64 overflow-y-auto rounded-md border bg-muted/30">
      <button
        v-if="listing?.parent && !found"
        class="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
        @click="show(listing.parent)"
      >
        <ChevronUp class="h-3.5 w-3.5" />
        <span class="text-muted-foreground">Up one</span>
      </button>
      <button
        v-for="entry in listing?.entries ?? []"
        :key="entry.path"
        class="flex w-full items-center gap-2 px-3 py-2 text-sm hover:bg-accent"
        @click="show(entry.path)"
      >
        <Folder class="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        <span class="min-w-0 flex-1 truncate text-left">
          {{ entry.name }}
          <span v-if="found" class="block truncate font-mono text-xs text-muted-foreground">{{ entry.path }}</span>
        </span>
        <UiBadge v-if="entry.repository" variant="glow" class="ml-auto shrink-0">git</UiBadge>
      </button>
      <p v-if="searching" class="px-3 py-6 text-center text-sm text-muted-foreground">
        Looking…
      </p>
      <p v-else-if="listing && listing.entries.length === 0" class="px-3 py-6 text-center text-sm text-muted-foreground">
        {{ found ? 'Nothing under your home folder matches that.' : 'Nothing inside.' }}
      </p>
    </div>

    <Field label="Name it (optional)" for="repo-name" class="mt-4">
      <Input id="repo-name" v-model="name" placeholder="Taken from the git remote, or the folder name" />
    </Field>

    <p class="mt-2 text-xs text-muted-foreground">
      Adding <code class="font-mono">{{ listing?.path }}</code>
    </p>
    <Notice v-if="problem" tone="danger" class="mt-3">
      {{ problem }}
    </Notice>
    <p v-if="busy" role="status" class="mt-3 text-sm text-muted-foreground">
      Reading this repository. Large repositories can take several minutes.
    </p>

    <div class="flex items-center gap-2 pt-5">
      <UiButton :disabled="busy || !listing" @click="add">
        <Loader2 v-if="busy" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
        <Plus v-else class="mr-1.5 h-3.5 w-3.5" />
        {{ busy ? 'Reading…' : 'Add and index' }}
      </UiButton>
      <UiButton variant="outline" @click="emit('close')">Cancel</UiButton>
    </div>
  </UiModal>
</template>
