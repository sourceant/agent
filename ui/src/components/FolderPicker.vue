<script setup>
import { ref, watch } from 'vue'
import { Folder, ChevronUp, Plus, Loader2 } from 'lucide-vue-next'
import { Modal as UiModal } from '@sourceant/design'
import { Button as UiButton } from '@sourceant/design'
import { Badge as UiBadge } from '@sourceant/design'
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

async function show(path) {
  try {
    listing.value = await api.browse(path)
    problem.value = ''
  } catch (error) {
    problem.value = error.message
  }
}

watch(() => props.open, (open) => {
  if (!open) return
  name.value = ''
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
    await api.index('', { everything: true })
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
    <h2 class="text-lg font-semibold mb-3">Add a folder</h2>
    <p class="mb-2 text-xs font-mono text-muted-foreground break-all">{{ listing?.path }}</p>

    <div class="h-64 overflow-y-auto rounded-md border bg-muted/30">
      <button
        v-if="listing?.parent"
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
        <Folder class="h-3.5 w-3.5 text-muted-foreground" />
        <span class="truncate">{{ entry.name }}</span>
        <UiBadge v-if="entry.repository" variant="glow" class="ml-auto shrink-0">git</UiBadge>
      </button>
      <p v-if="listing && listing.entries.length === 0" class="px-3 py-6 text-center text-sm text-muted-foreground">
        Nothing inside.
      </p>
    </div>

    <div class="mt-4">
      <label class="text-xs font-medium text-muted-foreground uppercase tracking-wider mb-1.5 block" for="repo-name">
        Name it (optional)
      </label>
      <input
        id="repo-name"
        v-model="name"
        placeholder="Taken from the git remote, or the folder name"
        class="w-full bg-muted/50 border rounded-md px-3 py-2 text-sm outline-none focus:border-primary/50 text-foreground"
      >
    </div>

    <p class="mt-2 text-xs text-muted-foreground">
      Adding <code class="font-mono">{{ listing?.path }}</code>
    </p>
    <p v-if="problem" class="mt-3 rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm">
      {{ problem }}
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
