<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { Check, ChevronDown, ChevronRight, Loader2 } from 'lucide-vue-next'
import { Button as UiButton, Notice } from '@sourceant/design'
import SettingField from '~/components/SettingField.vue'
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
// Tuning is kept behind a line somebody has to open, because a screen of
// twenty fields asks twenty questions and most of them have an answer already.
const CATALOGUE = 'model.name'
const offered = ref([])

const plain = computed(() => mine.value.filter((s) => !s.advanced))
const tuning = computed(() => mine.value.filter((s) => s.advanced))
const opened = ref(false)

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

// Every model the core can name, grouped by provider.
const choices = computed(() => {
  const named = offered.value.flatMap((one) =>
    one.models.map((model) => ({ value: model, label: model, group: one.provider })),
  )
  const current = String(settings.value.find((one) => one.key === CATALOGUE)?.value ?? '')
  if (current && !named.some((one) => one.value === current)) {
    return [{ value: current, label: current, group: 'In force' }, ...named]
  }
  return named
})

const optionsFor = (setting) => (setting.key === CATALOGUE ? choices.value : [])

async function load() {
  loading.value = true
  try {
    settings.value = await api.settings()
    draft.value = Object.fromEntries(mine.value.map((s) => [s.key, shown(s)]))
    // Only where the setting that needs it is on this screen.
    if (mine.value.some((one) => one.key === CATALOGUE) && !offered.value.length) {
      offered.value = await api.models().catch(() => [])
    }
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
      <SettingField
        v-for="setting in plain"
        :key="setting.key"
        v-model="draft[setting.key]"
        :setting="setting"
        :options="optionsFor(setting)"
        @reset="reset"
      />

      <div v-if="tuning.length" class="space-y-4 border-t pt-4">
        <button
          type="button"
          class="flex items-center gap-1.5 text-sm text-muted-foreground hover:text-foreground"
          @click="opened = !opened"
        >
          <ChevronDown v-if="opened" class="h-3.5 w-3.5" />
          <ChevronRight v-else class="h-3.5 w-3.5" />
          Tuning ({{ tuning.length }})
        </button>
        <SettingField
          v-for="setting in tuning"
          v-show="opened"
          :key="setting.key"
          v-model="draft[setting.key]"
          :setting="setting"
          @reset="reset"
        />
      </div>

      <p v-if="!mine.length" class="py-6 text-sm text-muted-foreground">
        Nothing in this group is configurable here.
      </p>

      <Notice v-if="problem" tone="danger">
        {{ problem }}
      </Notice>

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
