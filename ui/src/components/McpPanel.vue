<script setup>
import { Button as UiButton, Card as UiCard, Notice, Status } from '@sourceant/design'
import { computed, onMounted, ref } from 'vue'
import { Copy, Check } from 'lucide-vue-next'
import { api } from '~/api'

/* Connecting a coding agent to this machine's index. */

const endpoint = ref('')
const reachable = ref(null)
const copied = ref('')

const origin = computed(() => window.location.origin)

const stdio = computed(() =>
  JSON.stringify(
    {
      mcpServers: {
        sourceant: {
          command: 'sourceant',
          args: ['mcp'],
          env: { SOURCEANT_UI_URL: origin.value },
        },
      },
    },
    null,
    2,
  ),
)

const http = computed(() =>
  JSON.stringify(
    {
      mcpServers: {
        sourceant: { type: 'http', url: `${origin.value}/mcp` },
      },
    },
    null,
    2,
  ),
)

async function copy(which, text) {
  await navigator.clipboard.writeText(text)
  copied.value = which
  setTimeout(() => (copied.value = ''), 1500)
}

onMounted(async () => {
  endpoint.value = `${origin.value}/mcp`
  try {
    const answered = await fetch(endpoint.value, { method: 'GET' })
    // Anything that is not a 404 means something is mounted there.
    reachable.value = answered.status !== 404
  } catch {
    reachable.value = false
  }
})
</script>

<template>
  <div class="space-y-4">
    <UiCard class="p-5">
      <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-semibold">Endpoint</h2>
        <Status
          :label="reachable === null ? 'Checking' : reachable ? 'Serving' : 'Not mounted'"
          :tone="reachable === null ? 'neutral' : reachable ? 'success' : 'warning'"
          :busy="reachable === null"
        />
      </div>
      <p class="font-mono text-sm text-muted-foreground">{{ endpoint }}</p>
      <Notice v-if="reachable === false" tone="info" class="mt-3">
        Nothing is mounted there. The HTTP endpoint is served when this machine is
        running in local mode; over stdio it needs no endpoint at all.
      </Notice>
    </UiCard>

    <UiCard class="p-5">
      <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-semibold">Over stdio</h2>
        <UiButton size="sm" variant="outline" @click="copy('stdio', stdio)">
          <component :is="copied === 'stdio' ? Check : Copy" class="mr-2 h-3.5 w-3.5" />
          {{ copied === 'stdio' ? 'Copied' : 'Copy' }}
        </UiButton>
      </div>
      <p class="mb-3 text-sm text-muted-foreground">
        One process per client, started by the client. A review asked for this way is
        handed to this agent, so it finishes even after the client goes away.
      </p>
      <pre class="overflow-x-auto rounded-md bg-muted/50 p-3 text-xs"><code>{{ stdio }}</code></pre>
    </UiCard>

    <UiCard class="p-5">
      <div class="mb-2 flex flex-wrap items-center justify-between gap-2">
        <h2 class="font-semibold">Over HTTP</h2>
        <UiButton size="sm" variant="outline" @click="copy('http', http)">
          <component :is="copied === 'http' ? Check : Copy" class="mr-2 h-3.5 w-3.5" />
          {{ copied === 'http' ? 'Copied' : 'Copy' }}
        </UiButton>
      </div>
      <p class="mb-3 text-sm text-muted-foreground">
        One server, several clients. Reachable from this machine only.
      </p>
      <pre class="overflow-x-auto rounded-md bg-muted/50 p-3 text-xs"><code>{{ http }}</code></pre>
    </UiCard>
  </div>
</template>
