<script setup>
import { RotateCcw } from 'lucide-vue-next'
import {
  Badge as UiBadge,
  Field,
  Input,
  ListInput,
  SearchableSelect,
  Select,
} from '@sourceant/design'

/* One setting, drawn from what the core says about it.
 *
 * Which control appears is decided by the setting's own type, choices and
 * whether it holds several of something, so a setting the core starts
 * declaring appears without a change here.
 */

defineProps({
  setting: { type: Object, required: true },
  // One of a known set, where somebody else keeps the set. Long enough that a
  // native select is useless, which is what SearchableSelect is for.
  options: { type: Array, default: () => [] },
})

defineEmits(['reset'])
const modelValue = defineModel({ required: true })

// Whether this is still what it shipped as, which is what makes putting it
// back worth offering.
function touched(setting) {
  return setting.secret
    ? setting.is_set
    : String(setting.value ?? '') !== String(setting.default ?? '')
}
</script>

<template>
  <Field :label="setting.label" :for="setting.key" :hint="setting.description">
    <template #label>
      <UiBadge v-if="setting.secret && setting.is_set" variant="success">set</UiBadge>
      <button
        v-if="touched(setting)"
        type="button"
        class="inline-flex items-center gap-1 text-[11px] font-normal normal-case tracking-normal text-muted-foreground hover:text-foreground"
        @click="$emit('reset', setting)"
      >
        <RotateCcw class="h-3 w-3" />
        put back
      </button>
    </template>

    <SearchableSelect
      v-if="options.length"
      v-model="modelValue"
      :options="options"
      empty-label="Nothing chosen"
      placeholder="Choose one"
      search-placeholder="Search"
    />

    <Select v-else-if="setting.choices?.length" :id="setting.key" v-model="modelValue" class="w-full">
      <option v-for="choice in setting.choices" :key="choice" :value="choice">{{ choice }}</option>
    </Select>

    <label v-else-if="setting.type === 'bool'" class="flex items-center gap-2 text-sm">
      <input :id="setting.key" v-model="modelValue" type="checkbox" class="h-3.5 w-3.5 rounded border">
      <span class="text-muted-foreground">{{ modelValue ? 'On' : 'Off' }}</span>
    </label>

    <ListInput
      v-else-if="setting.listed || setting.type === 'json'"
      v-model="modelValue"
      mono
      :noun="setting.type === 'json' ? 'a path' : 'a folder'"
      :placeholder="setting.type === 'json' ? 'node_modules' : '/home/you/work/knowledgebase/skills'"
    />

    <Input
      v-else
      :id="setting.key"
      v-model="modelValue"
      :type="setting.secret ? 'password' : setting.type === 'int' || setting.type === 'float' ? 'number' : 'text'"
      :autocomplete="setting.secret ? 'off' : undefined"
      :placeholder="setting.secret && setting.is_set
        ? 'Leave empty to keep what is set'
        : String(setting.default ?? '')"
    />
  </Field>
</template>
