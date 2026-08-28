<script setup lang="ts">
import type { HTMLAttributes } from 'vue'
import { cn } from '~/lib/utils'

interface Props {
  class?: HTMLAttributes['class']
  hover?: boolean
  glow?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  hover: false,
  glow: false,
})
</script>

<template>
  <div
    :class="cn(
      'rounded-lg border bg-card text-card-foreground shadow-sm',
      hover && 'transition-all duration-200 hover:border-primary/50 hover:-translate-y-1',
      glow && 'glow-sm',
      props.class
    )"
  >
    <slot />
  </div>
</template>
