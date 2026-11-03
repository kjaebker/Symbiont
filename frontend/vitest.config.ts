import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config'

// vite.config.ts exports a function (it reads `mode` to load env vars), and
// mergeConfig only accepts plain objects — resolve it with the same env first.
export default defineConfig((env) =>
  mergeConfig(viteConfig(env), {
    test: {
      environment: 'jsdom',
      globals: true,
      setupFiles: [],
    },
  }),
)
