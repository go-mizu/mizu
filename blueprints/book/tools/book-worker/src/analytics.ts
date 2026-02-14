// Analytics Engine middleware for Hono
// Writes per-request data points to Cloudflare Analytics Engine

import { createMiddleware } from 'hono/factory'
import type { HonoEnv, Env } from './types'

// Normalize API paths: replace numeric IDs with :id, year-like segments with :year
// e.g. /books/42/reviews → /books/:id/reviews
//      /stats/2025 → /stats/:year
//      /shelves/3/books/5 → /shelves/:id/books/:id
function normalizePath(path: string): string {
  return path.replace(/\/\d+/g, (match, offset: number) => {
    const before = path.slice(0, offset)
    if (before.endsWith('/stats') || before.endsWith('/challenge')) return '/:year'
    return '/:id'
  })
}

export const analyticsMiddleware = createMiddleware<HonoEnv>(async (c, next) => {
  const start = Date.now()

  await next()

  const durationMs = Date.now() - start
  const url = new URL(c.req.url)
  // Strip /api prefix for route grouping since all routes are under /api
  const apiPath = url.pathname.replace(/^\/api/, '')
  const route = normalizePath(apiPath)
  const cf = (c.req.raw as any).cf as IncomingRequestCfProperties | undefined

  try {
    c.env.ANALYTICS.writeDataPoint({
      indexes: [route],
      blobs: [
        c.req.method,                                         // blob1: method
        route,                                                // blob2: normalized route
        String(c.res.status),                                 // blob3: status code
        cf?.colo ?? 'unknown',                                // blob4: CF edge colo
        (c.req.header('user-agent') ?? 'unknown').slice(0, 128), // blob5: user-agent
        '',                                                   // blob6: error (empty on success)
      ],
      doubles: [
        durationMs,  // double1: response time ms
        1,           // double2: request count (for SUM)
      ],
    })
  } catch {
    // Analytics should never break the request
  }
})

// Write a queue job data point
export function writeQueueMetric(
  analytics: AnalyticsEngineDataset,
  jobType: string,
  durationMs: number,
  success: boolean,
  error?: string,
): void {
  try {
    analytics.writeDataPoint({
      indexes: ['queue'],
      blobs: [
        jobType,                    // blob1: job type
        success ? 'success' : 'error', // blob2: status
        (error ?? '').slice(0, 256),   // blob3: error message
      ],
      doubles: [
        durationMs, // double1: duration ms
        1,          // double2: job count
      ],
    })
  } catch {
    // Never break queue processing for analytics
  }
}
