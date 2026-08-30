<script setup>
import { Button as UiButton, Card as UiCard } from '@sourceant/design'
import { Plus } from 'lucide-vue-next'
</script>

<template>
  <UiCard class="text-center py-16 px-6">
    <h2 class="text-lg font-semibold mb-1">Nothing indexed yet</h2>
    <p class="text-muted-foreground text-sm mb-4 max-w-lg mx-auto">
      Point SourceAnt at a folder on this machine and it reads the code into a graph you can
      look at and record against.
    </p>
    <UiButton as="a" href="/repositories" variant="glow">
      <Plus class="mr-2 h-4 w-4" />
      Add a repository
    </UiButton>
  </UiCard>
</template>
