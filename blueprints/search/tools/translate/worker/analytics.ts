/**
 * Analytics Engine helper — writes structured events.
 *
 * Single dataset: translate_events
 * Schema:
 *   blob1  = event type (translate|page|tts|detect|queue|error)
 *   blob2  = source lang or URL
 *   blob3  = target lang
 *   blob4  = provider (google|mymemory|libre)
 *   blob5  = extra context (error msg, render mode)
 *   blob6  = cache status (HIT|MISS)
 *   double1 = latency ms
 *   double2 = char count or text count
 *   double3 = success (1) or failure (0)
 *   double4 = cache hits
 *   double5 = total items
 */

import type { Env } from './types'

export interface AnalyticsEvent {
  event: string
  sl?: string
  tl?: string
  provider?: string
  extra?: string
  cache?: string
  latencyMs?: number
  chars?: number
  success?: boolean
  cacheHits?: number
  total?: number
}

export function track(env: Env, data: AnalyticsEvent): void {
  try {
    env.ANALYTICS?.writeDataPoint({
      blobs: [
        data.event,
        data.sl ?? '',
        data.tl ?? '',
        data.provider ?? '',
        data.extra ?? '',
        data.cache ?? '',
      ],
      doubles: [
        data.latencyMs ?? 0,
        data.chars ?? 0,
        data.success === false ? 0 : 1,
        data.cacheHits ?? 0,
        data.total ?? 0,
      ],
    })
  } catch {
    // Analytics should never break the request
  }
}
