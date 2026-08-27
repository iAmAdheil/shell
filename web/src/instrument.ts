import * as Sentry from '@sentry/react'

/**
 * Starts Sentry. An empty VITE_SENTRY_DSN turns Sentry off, which is what a
 * local run without a Sentry account wants.
 *
 * main.tsx imports this file first, so Sentry is already watching before any
 * other module runs.
 */
if (import.meta.env.VITE_SENTRY_DSN) {
  Sentry.init({
    dsn: import.meta.env.VITE_SENTRY_DSN,
    environment: import.meta.env.VITE_SENTRY_ENVIRONMENT || 'development',

    // A Session shows whatever Users type into a shared shell, so the SDK
    // sends as little as it can: no caller IP address, no request bodies.
    // The Account ID still goes out, because useMe sets it by hand.
    dataCollection: { userInfo: false, httpBodies: [] },

    integrations: [Sentry.browserTracingIntegration()],
    tracesSampleRate: tracesSampleRate(),

    // The API is same-origin, so trace headers reach this app's own server
    // and no third party.
    tracePropagationTargets: [/^\//],

    beforeBreadcrumb: hideSessionCode,
  })
}

/** Reads VITE_SENTRY_TRACES_SAMPLE_RATE, or falls back to one request in ten. */
function tracesSampleRate(): number {
  const wanted = Number(import.meta.env.VITE_SENTRY_TRACES_SAMPLE_RATE)
  return Number.isFinite(wanted) && wanted >= 0 && wanted <= 1 ? wanted : 0.1
}

/**
 * A Session Code is what lets anyone join a running Session, so it must not
 * travel to Sentry inside a request URL.
 */
function hideSessionCode(crumb: Sentry.Breadcrumb): Sentry.Breadcrumb {
  if (crumb.data && typeof crumb.data.url === 'string') {
    crumb.data.url = crumb.data.url.replace(/\/api\/sessions\/[^/?]+/, '/api/sessions/:code')
  }
  return crumb
}
