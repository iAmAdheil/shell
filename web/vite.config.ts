import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      // ws lets the Session terminal socket through, not just plain requests.
      '/api': { target: 'http://localhost:8081', ws: true },
    },
  },
})
