<script setup>
defineProps({
  icon: { type: [Object, Function], required: true },
  title: { type: String, required: true },
  sub: { type: String, default: '' },
  pillar: { type: String, default: 'graph' },
})

const tints = {
  memory: 'bg-pillar-memory/15 text-pillar-memory',
  graph: 'bg-pillar-graph/15 text-pillar-graph',
  review: 'bg-pillar-review/15 text-pillar-review',
  tokens: 'bg-pillar-tokens/15 text-pillar-tokens',
}
</script>

<template>
  <div class="flex flex-wrap items-start justify-between gap-4 mb-6">
    <div class="flex items-center gap-3">
      <div :class="['w-11 h-11 rounded-lg flex items-center justify-center shrink-0', tints[pillar]]">
        <component :is="icon" class="w-6 h-6" />
      </div>
      <div>
        <h1 class="text-xl font-semibold">{{ title }}</h1>
        <p v-if="sub" class="text-sm text-muted-foreground">{{ sub }}</p>
      </div>
    </div>
    <div class="flex flex-wrap items-center gap-2">
      <slot name="actions" />
    </div>
  </div>
</template>
