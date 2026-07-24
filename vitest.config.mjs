import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  test: {
    environment: 'happy-dom',
    globals: true,
    include: ['src/**/*.test.ts'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html'],
      include: ['src/api/**/*.ts', 'src/stores/**/*.ts', 'src/features/schedule/**/*.ts'],
      exclude: ['**/*.test.ts'],
    },
  },
})
