import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  define: {
    __CLAWFLEET_WEB_VERSION__: JSON.stringify(process.env.CLAWFLEET_WEB_VERSION || 'dev'),
    __CLAWFLEET_WEB_COMMIT__: JSON.stringify(process.env.CLAWFLEET_WEB_COMMIT || ''),
    __CLAWFLEET_WEB_BUILD_TIME__: JSON.stringify(process.env.CLAWFLEET_WEB_BUILD_TIME || ''),
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8090',
      '/dl': 'http://localhost:8090'
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
