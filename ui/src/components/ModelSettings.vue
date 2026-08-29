<script setup>
import { computed, onMounted, ref } from 'vue'
import { Check, Loader2 } from 'lucide-vue-next'
import { Badge as UiBadge, Button as UiButton, Field, Input } from '@sourceant/design'
import { api } from '~/api'

/* Whose model, and whose bill.
 *
 * Reading a repository is deterministic and needs none of this. Anything that
 * proposes rather than reads does, and it stays off until somebody says which
 * model to ask.
 *
 * The key is written here and never read back: what comes the other way says
 * only whether one is set. */

const emit = defineEmits(['saved'])

const settings = ref([])
const draft = ref({})
const saving = ref(false)
const problem = ref('')
const saved = ref(false)

const model = computed(() => settings.value.find((s) => s.key === 'model.name'))
const key = computed(() => settings.value.find((s) => s.key === 'model.api_key'))
const endpoint = computed(() => settings.value.find((s) => s.key === 'model.base_url'))

const ready = computed(() => !!model.value?.value && !!key.value?.is_set)

async function load() {
  try {
    settings.value = (await api.settings()).filter((s) => s.group === 'Model')
    draft.value = {
      'model.name': model.value?.value ?? '',
      'model.api_key': '',
      'model.base_url': endpoint.value?.value ?? '',
    }
    problem.value = ''
  } catch (caught) {
    problem.value = caught.message
  }
}

async function save() {
  saving.value = true
  saved.value = false
  problem.value = ''
  try {
    for (const [setting, value] of Object.entries(draft.value)) {
      // An untouched key field means leave the key alone, not erase it.
      if (setting === 'model.api_key' && !value) continue
      await api.setSetting(setting, value)
    }
    await load()
    saved.value = true
    emit('saved')
  } catch (caught) {
    problem.value = caught.message
  } finally {
    saving.value = false
  }
}

defineExpose({ ready })
onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <Field label="Model" for="model-name" hint="Named the way the provider names it.">
      <Input id="model-name" v-model="draft['model.name']" placeholder="anthropic/claude-sonnet-4-5" />
    </Field>

    <Field
      label="API key"
      for="model-key"
      hint="Kept on this machine, sent to that provider and nowhere else, and never shown again."
    >
      <template #label>
        <UiBadge v-if="key?.is_set" variant="success">set</UiBadge>
      </template>
      <Input
        id="model-key"
        v-model="draft['model.api_key']"
        type="password"
        autocomplete="off"
        :placeholder="key?.is_set ? 'Leave empty to keep the key you have' : 'Your key for that provider'"
      />
    </Field>

    <Field label="Endpoint" for="model-endpoint">
      <Input
        id="model-endpoint"
        v-model="draft['model.base_url']"
        placeholder="Left empty unless the model runs somewhere of its own"
      />
    </Field>

    <p v-if="problem" class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm">
      {{ problem }}
    </p>

    <div class="flex items-center gap-3">
      <UiButton :disabled="saving" @click="save">
        <Loader2 v-if="saving" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
        <Check v-else class="mr-1.5 h-3.5 w-3.5" />
        Save
      </UiButton>
      <span v-if="saved" class="text-xs text-success">Saved.</span>
      <slot name="after" />
    </div>
  </div>
</template>
