<script setup>
import { computed, onMounted, ref } from 'vue'
import { Plus, ArrowRight } from 'lucide-vue-next'
import { Modal as UiModal, Button as UiButton } from '@sourceant/design'
import ModelSettings from '~/components/ModelSettings.vue'
import FolderPicker from '~/components/FolderPicker.vue'
import { useRepositories } from '~/composables/useRepositories'

/* What a machine needs before any of this is useful: somewhere to read, and,
 * for the parts that propose rather than read, a model to ask.
 *
 * Asked once. A person who skips it is not asked again, because a box that
 * keeps reappearing is a box people learn to dismiss without reading. Both
 * answers live in Settings afterwards. */

const REMEMBERED = 'sourceant-onboarded'

const { repositories, fetchRepositories } = useRepositories()
const open = ref(false)
const step = ref('folder')
const picking = ref(false)

const done = computed(() => repositories.value.length > 0)

function remember() {
  try {
    localStorage.setItem(REMEMBERED, 'yes')
  } catch {
    // A browser that refuses storage asks again. Better than not opening.
  }
  open.value = false
}

onMounted(async () => {
  let asked = null
  try {
    asked = localStorage.getItem(REMEMBERED)
  } catch {
    asked = null
  }
  if (asked) return
  await fetchRepositories()
  open.value = repositories.value.length === 0
})
</script>

<template>
  <UiModal :open="open" max-width="xl" @close="remember">
    <template v-if="step === 'folder'">
      <h2 class="text-lg font-semibold mb-1">Point it at some code</h2>
      <p class="text-sm text-muted-foreground mb-5">
        SourceAnt reads a folder on this machine into a graph, and keeps what you record
        about it beside the code it belongs to. Nothing leaves this machine.
      </p>
      <div class="flex items-center gap-2">
        <UiButton @click="picking = true">
          <Plus class="mr-1.5 h-3.5 w-3.5" />
          Add a folder
        </UiButton>
        <UiButton variant="outline" @click="step = 'model'">
          <template v-if="done">Next</template>
          <template v-else>Skip for now</template>
          <ArrowRight class="ml-1.5 h-3.5 w-3.5" />
        </UiButton>
      </div>
      <FolderPicker :open="picking" @close="picking = false" @added="fetchRepositories" />
    </template>

    <template v-else>
      <h2 class="text-lg font-semibold mb-1">Bring a model, or don't</h2>
      <p class="text-sm text-muted-foreground mb-5">
        Reading your code and reading what it already states about itself need no model at
        all. Proposing what nobody wrote down does. Your key stays on this machine and goes
        to that provider and nowhere else.
      </p>
      <ModelSettings @saved="remember">
        <template #after>
          <UiButton variant="ghost" @click="remember">Not now</UiButton>
        </template>
      </ModelSettings>
    </template>
  </UiModal>
</template>
