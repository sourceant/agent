import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The build writes straight into what the agent embeds, so a built binary and
// the sources it was built from cannot disagree.
export default defineConfig({
  plugins: [vue()],
  base: './',
  resolve: {
    alias: { '~': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  build: {
    outDir: '../internal/ui/assets',
    emptyOutDir: true,
    // The agent serves this from a machine with no network, so nothing may be
    // left as a separate fetch that a CDN would have to answer.
    assetsInlineLimit: 0,
  },
})
