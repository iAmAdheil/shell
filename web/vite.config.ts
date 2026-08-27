import { sentryVitePlugin } from '@sentry/vite-plugin'
import react from '@vitejs/plugin-react'
import { fileURLToPath } from 'node:url'
import { defineConfig, loadEnv } from 'vite'

// The repo root holds the one .env the server already reads, so the browser
// build reads the same file instead of a second copy.
const envDir = fileURLToPath(new URL('..', import.meta.url))

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  // The empty prefix reads every variable, not only the VITE_ ones. Only the
  // VITE_ ones ever reach the browser bundle, so SENTRY_AUTH_TOKEN and the
  // server's own secrets stay on this side.
  const env = loadEnv(mode, envDir, '')

  // Without source maps in Sentry, a production stack trace names minified
  // code and nothing else. Uploading them needs all three settings.
  const uploadSourceMaps = Boolean(env.SENTRY_AUTH_TOKEN && env.SENTRY_ORG && env.SENTRY_PROJECT)

  return {
    envDir,
    plugins: [
      react(),
      ...(uploadSourceMaps
        ? [
            sentryVitePlugin({
              authToken: env.SENTRY_AUTH_TOKEN,
              org: env.SENTRY_ORG,
              project: env.SENTRY_PROJECT,
              // Sentry keeps the maps. Deleting them stops the build from
              // publishing this app's source to everyone who opens it.
              sourcemaps: { filesToDeleteAfterUpload: ['./dist/**/*.map'] },
            }),
          ]
        : []),
    ],
    build: { sourcemap: uploadSourceMaps },
    server: {
      proxy: {
        // ws lets the Session terminal socket through, not just plain requests.
        '/api': { target: 'http://localhost:8081', ws: true },
      },
    },
  }
})
