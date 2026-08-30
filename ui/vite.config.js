import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// The build writes straight into what the agent embeds, so a built binary and
// the sources it was built from cannot disagree.
export default defineConfig({
  plugins: [vue()],
  // Absolute, not relative: routes are real paths now, so a page served at
  // /reviews/abc123 would otherwise resolve ./assets against /reviews/.
  base: '/',
  resolve: {
    alias: { '~': fileURLToPath(new URL('./src', import.meta.url)) },
    // The design package is linked from a sibling checkout, so its own imports
    // have to resolve against this app's dependencies rather than against the
    // checkout it physically lives in, which installs nothing.
    preserveSymlinks: true,
  },
  // The design package ships source rather than a build, so it is compiled with
  // the app instead of pre-bundled as a dependency.
  optimizeDeps: { exclude: ['@sourceant/design'] },
  build: {
    outDir: '../internal/ui/assets',
    emptyOutDir: true,
    // The agent serves this from a machine with no network, so nothing may be
    // left as a separate fetch that a CDN would have to answer.
    assetsInlineLimit: 0,
  },
})
