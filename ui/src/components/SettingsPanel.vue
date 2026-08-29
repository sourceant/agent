<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { Check, Loader2, RotateCcw } from 'lucide-vue-next'
import {
  Badge as UiBadge,
  Button as UiButton,
  Field,
  Input,
  ListInput,
  Select,
} from '@sourceant/design'
import { api } from '~/api'

/* One group of settings, drawn from what the core says it has.
 *
 * The core declares each setting's label, type, choices and which group it
 * belongs to, so a setting it starts declaring appears here without a change to
 * this file. Anything hand-written per setting would be a second, staler copy
 * of that list.
 *
 * A credential answers whether it is set rather than what it is, so an
 * untouched field means leave the key alone rather than erase it.
 */

const props = defineProps({
  group: { type: String, required: true },
})

const emit = defineEmits(['saved'])

const settings = ref([])
const draft = ref({})
const saving = ref(false)
const loading = ref(true)
const problem = ref('')
const saved = ref(false)

const mine = computed(() => settings.value.filter((s) => s.group === props.group))

function shown(setting) {
  // A credential is never read back, so there is nothing to put in the box.
  if (setting.secret) return ''
  if (setting.listed) {
    return String(setting.value ?? '')
      .split('\n')
      .map((one) => one.trim())
      .filter(Boolean)
  }
  return setting.value ?? ''
}

// Stored one to a line, which is what a string setting can hold.
const written = (setting, value) => (setting.listed ? (value ?? []).join('\n') : value)

const same = (setting, value) =>
  String(written(setting, value) ?? '') === String(setting.value ?? '')

// A key already set stays set until somebody types a new one, so an empty box
// is not a change.
const changed = computed(() =>
  mine.value.some((setting) => {
    const value = draft.value[setting.key]
    if (setting.secret) return !!value
    return !same(setting, value)
  }),
)

async function load() {
  loading.value = true
  try {
    settings.value = await api.settings()
    draft.value = Object.fromEntries(mine.value.map((s) => [s.key, shown(s)]))
    problem.value = ''
  } catch (caught) {
    problem.value = caught.message
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  saved.value = false
  problem.value = ''
  try {
    for (const setting of mine.value) {
      const value = draft.value[setting.key]
      if (setting.secret && !value) continue
      if (!setting.secret && same(setting, value)) continue
      await api.setSetting(setting.key, written(setting, value))
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

async function reset(setting) {
  problem.value = ''
  try {
    await api.resetSetting(setting.key)
    await load()
  } catch (caught) {
    problem.value = caught.message
  }
}

function touched(setting) {
  return setting.secret ? setting.is_set : String(setting.value ?? '') !== String(setting.default ?? '')
}

watch(() => props.group, load)
onMounted(load)
</script>

<template>
  <div class="space-y-4">
    <p v-if="loading" class="flex items-center gap-2 py-6 text-sm text-muted-foreground">
      <Loader2 class="h-4 w-4 animate-spin" />
      Reading what this machine can be told.
    </p>

    <template v-else>
      <Field
        v-for="setting in mine"
        :key="setting.key"
        :label="setting.label"
        :for="setting.key"
        :hint="setting.description"
      >
        <template #label>
          <UiBadge v-if="setting.secret && setting.is_set" variant="success">set</UiBadge>
          <button
            v-if="touched(setting)"
            type="button"
            class="inline-flex items-center gap-1 text-[11px] font-normal normal-case tracking-normal text-muted-foreground hover:text-foreground"
            @click="reset(setting)"
          >
            <RotateCcw class="h-3 w-3" />
            put back
          </button>
        </template>

        <Select
          v-if="setting.choices?.length"
          :id="setting.key"
          v-model="draft[setting.key]"
          class="w-full"
        >
          <option v-for="choice in setting.choices" :key="choice" :value="choice">{{ choice }}</option>
        </Select>

        <label v-else-if="setting.type === 'bool'" class="flex items-center gap-2 text-sm">
          <input
            :id="setting.key"
            v-model="draft[setting.key]"
            type="checkbox"
            class="h-3.5 w-3.5 rounded border"
          >
          <span class="text-muted-foreground">{{ draft[setting.key] ? 'On' : 'Off' }}</span>
        </label>

        <ListInput
          v-else-if="setting.listed"
          v-model="draft[setting.key]"
          mono
          noun="a folder"
          placeholder="/home/you/work/knowledgebase/skills"
        />

        <Input
          v-else
          :id="setting.key"
          v-model="draft[setting.key]"
          :type="setting.secret ? 'password' : setting.type === 'int' || setting.type === 'float' ? 'number' : 'text'"
          :autocomplete="setting.secret ? 'off' : undefined"
          :placeholder="setting.secret && setting.is_set
            ? 'Leave empty to keep what is set'
            : String(setting.default ?? '')"
        />
      </Field>

      <p v-if="!mine.length" class="py-6 text-sm text-muted-foreground">
        Nothing in this group is configurable here.
      </p>

      <p v-if="problem" class="rounded-md border border-destructive/40 bg-destructive/10 px-3 py-2 text-sm">
        {{ problem }}
      </p>

      <div v-if="mine.length" class="flex items-center gap-3">
        <UiButton :disabled="saving || !changed" @click="save">
          <Loader2 v-if="saving" class="mr-1.5 h-3.5 w-3.5 animate-spin" />
          <Check v-else class="mr-1.5 h-3.5 w-3.5" />
          Save
        </UiButton>
        <span v-if="saved && !changed" class="text-xs text-success">Saved.</span>
        <slot name="after" />
      </div>
    </template>
  </div>
</template>
