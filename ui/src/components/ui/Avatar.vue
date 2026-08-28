<script setup lang="ts">
import { computed, ref } from 'vue'
import { cn } from '~/lib/utils'

interface Props {
  src?: string | null
  alt?: string
  fallback?: string
  size?: 'sm' | 'md' | 'lg'
  class?: string
}

const props = withDefaults(defineProps<Props>(), {
  size: 'md',
  fallback: '?',
})

const sizeClasses = {
  sm: 'h-8 w-8 text-xs',
  md: 'h-10 w-10 text-sm',
  lg: 'h-12 w-12 text-base',
}

const imageError = ref(false)

const showFallback = computed(() => !props.src || imageError.value)
</script>

<template>
  <div
    :class="cn(
      'relative flex shrink-0 overflow-hidden rounded-full',
      sizeClasses[size],
      props.class
    )"
  >
    <img
      v-if="!showFallback"
      :src="src ?? undefined"
      :alt="alt"
      class="aspect-square h-full w-full object-cover"
      @error="imageError = true"
    />
    <div
      v-else
      class="flex h-full w-full items-center justify-center bg-muted font-medium text-muted-foreground"
    >
      {{ fallback }}
    </div>
  </div>
</template>
