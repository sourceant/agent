<script setup lang="ts">
interface Props {
  size?: 'sm' | 'md' | 'lg'
  showText?: boolean
}

withDefaults(defineProps<Props>(), {
  size: 'md',
  showText: true,
})

const sizeClasses = {
  sm: 'h-7 w-7 min-h-7 min-w-7',
  md: 'h-8 w-8 min-h-8 min-w-8',
  lg: 'h-10 w-10 min-h-10 min-w-10',
}

const textSizes = {
  sm: 'text-base',
  md: 'text-lg',
  lg: 'text-xl',
}
</script>

<template>
  <div class="flex items-center gap-2.5">
    <div :class="[sizeClasses[size], 'relative flex-shrink-0']">
      <svg viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" class="w-full h-full">
        <rect width="32" height="32" rx="6" fill="#E20C18"/>
        <!-- Ant head -->
        <ellipse cx="16" cy="12" rx="5.5" ry="5" fill="white"/>
        <!-- Ant thorax -->
        <ellipse cx="16" cy="19" rx="4" ry="3" fill="white"/>
        <!-- Ant abdomen -->
        <ellipse cx="16" cy="25" rx="5" ry="4" fill="white"/>
        <!-- Antennae -->
        <path d="M12 8L8 4" stroke="white" stroke-width="1.5" stroke-linecap="round"/>
        <path d="M20 8L24 4" stroke="white" stroke-width="1.5" stroke-linecap="round"/>
        <!-- Eyes -->
        <circle cx="14" cy="11" r="1.2" fill="#E20C18"/>
        <circle cx="18" cy="11" r="1.2" fill="#E20C18"/>
      </svg>
    </div>
    <span v-if="showText" :class="[textSizes[size], 'font-semibold tracking-tight']">
      <span class="text-brand">Source</span><span class="text-brand">Ant</span>
      <span class="text-muted-foreground font-normal ml-1">Memory</span>
    </span>
  </div>
</template>
