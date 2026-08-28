import { ref } from 'vue'

const isDark = ref(true)

function apply(dark) {
  isDark.value = dark
  document.documentElement.classList.toggle('dark', dark)
  document.documentElement.classList.toggle('light', !dark)
  try {
    localStorage.setItem('sourceant-theme', dark ? 'dark' : 'light')
  } catch {
    // A browser that refuses storage still gets the theme, just not the memory.
  }
}

export function useTheme() {
  return {
    isDark,
    toggleTheme: () => apply(!isDark.value),
    restoreTheme: () => {
      let stored = null
      try {
        stored = localStorage.getItem('sourceant-theme')
      } catch {
        stored = null
      }
      apply(stored !== 'light')
    },
  }
}
