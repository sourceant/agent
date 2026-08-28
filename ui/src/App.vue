<script setup>
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import {
  LayoutDashboard,
  Network,
  BookOpen,
  Boxes,
  Settings,
  Sun,
  Moon,
  Menu,
  X,
} from 'lucide-vue-next'
import { createAvatar } from '@dicebear/core'
import { notionists } from '@dicebear/collection'
import UiLogo from '~/components/ui/Logo.vue'
import UiAvatar from '~/components/ui/Avatar.vue'
import { useTheme } from '~/composables/useTheme'

const { isDark, toggleTheme, restoreTheme } = useTheme()
const route = useRoute()
const menuOpen = ref(false)
const userOpen = ref(false)

/* What a machine has. Anything needing a pull request or another person is
 * not here, because nothing on a machine produces one. */
const navigation = [
  { name: 'Overview', href: '/', icon: LayoutDashboard },
  { name: 'Knowledge', href: '/knowledge', icon: BookOpen },
  { name: 'Graphs', href: '/graph', icon: Network },
  { name: 'Repositories', href: '/repositories', icon: Boxes },
  { name: 'Settings', href: '/settings', icon: Settings },
]

/* Nobody signs in to their own machine, so there is no account to show. The
 * avatar is generated here, from the machine's name, purely so the shell has
 * the shape a person already knows. */
const avatar = computed(() =>
  createAvatar(notionists, { seed: location.hostname || 'sourceant', radius: 50 }).toDataUri())

function closeDropdowns(event) {
  if (!event.target.closest('[data-dropdown="user"]')) userOpen.value = false
}

onMounted(() => {
  restoreTheme()
  document.addEventListener('click', closeDropdowns)
})
onUnmounted(() => document.removeEventListener('click', closeDropdowns))
</script>

<template>
  <div class="flex h-screen flex-col bg-background">
    <header class="z-40 h-12 shrink-0 border-b bg-card/80 backdrop-blur-sm">
      <div class="flex items-center h-full min-w-0 px-3 gap-1 sm:px-4">
        <RouterLink to="/" class="shrink-0 mr-1 sm:mr-3">
          <UiLogo size="sm" :show-text="false" />
        </RouterLink>

        <span class="hidden lg:block h-4 w-px bg-border mx-1 shrink-0" />

        <nav class="hidden lg:flex items-center gap-0.5 min-w-0">
          <RouterLink
            v-for="item in navigation"
            :key="item.href"
            :to="item.href"
            :title="item.name"
            :class="[
              'flex shrink-0 items-center gap-1.5 px-2 py-1 rounded-md text-sm transition-colors 2xl:px-2.5',
              route.path === item.href
                ? 'bg-primary/10 text-primary font-medium'
                : 'text-muted-foreground hover:bg-muted hover:text-foreground',
            ]"
          >
            <component :is="item.icon" class="h-3.5 w-3.5 shrink-0" />
            <span class="hidden xl:inline">{{ item.name }}</span>
          </RouterLink>
        </nav>

        <div class="flex-1 min-w-0" />

        <div class="relative shrink-0" data-dropdown="user">
          <button
            class="flex items-center gap-1.5 p-1 rounded-md hover:bg-muted transition-colors"
            aria-label="Account"
            @click.stop="userOpen = !userOpen"
          >
            <UiAvatar :src="avatar" size="sm" class="h-6 w-6" />
          </button>

          <Transition
            enter-active-class="transition duration-100 ease-out"
            enter-from-class="opacity-0 scale-95"
            enter-to-class="opacity-100 scale-100"
            leave-active-class="transition duration-75 ease-in"
            leave-from-class="opacity-100 scale-100"
            leave-to-class="opacity-0 scale-95"
          >
            <div
              v-if="userOpen"
              class="absolute top-full right-0 mt-1 w-52 bg-card border rounded-lg shadow-lg py-1 z-50"
            >
              <div class="px-3 py-2 border-b">
                <p class="text-sm font-medium truncate">This machine</p>
                <p class="text-xs text-muted-foreground truncate">Nothing here has been shared.</p>
              </div>
              <RouterLink
                to="/settings"
                class="flex items-center gap-2 px-3 py-1.5 text-sm hover:bg-muted transition-colors"
                @click="userOpen = false"
              >
                <Settings class="h-3.5 w-3.5 text-muted-foreground" />
                Settings
              </RouterLink>
              <button
                class="flex items-center gap-2 w-full px-3 py-1.5 text-sm hover:bg-muted transition-colors"
                @click="toggleTheme"
              >
                <component :is="isDark ? Sun : Moon" class="h-3.5 w-3.5 text-muted-foreground" />
                <span>{{ isDark ? 'Light mode' : 'Dark mode' }}</span>
              </button>
            </div>
          </Transition>
        </div>

        <button
          class="lg:hidden shrink-0 p-1 rounded-md hover:bg-muted transition-colors"
          :aria-label="menuOpen ? 'Close menu' : 'Open menu'"
          @click="menuOpen = !menuOpen"
        >
          <Menu v-if="!menuOpen" class="h-5 w-5" />
          <X v-else class="h-5 w-5" />
        </button>
      </div>
    </header>

    <div
      v-if="menuOpen"
      class="lg:hidden shrink-0 border-b bg-card px-4 py-2 space-y-0.5"
    >
      <RouterLink
        v-for="item in navigation"
        :key="item.href"
        :to="item.href"
        :class="[
          'flex items-center gap-2 px-3 py-2 rounded-md text-sm transition-colors',
          route.path === item.href
            ? 'bg-primary/10 text-primary font-medium'
            : 'text-muted-foreground hover:bg-muted hover:text-foreground',
        ]"
        @click="menuOpen = false"
      >
        <component :is="item.icon" class="h-4 w-4" />
        {{ item.name }}
      </RouterLink>
    </div>

    <main class="min-h-0 flex-1 overflow-y-auto">
      <!-- A definite height, not a minimum: a page that fills the space it is
           given has nothing to resolve flex-1 against otherwise, and grows to
           its content instead. Pages taller than this still scroll here. -->
      <div class="container mx-auto flex h-full flex-col px-4 pb-4 pt-3 lg:px-6 lg:pb-6 lg:pt-4">
        <RouterView />
      </div>
    </main>
  </div>
</template>
