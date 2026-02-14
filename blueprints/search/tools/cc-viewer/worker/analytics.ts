export class Analytics {
  private engine: AnalyticsEngineDataset | undefined

  constructor(engine?: AnalyticsEngineDataset) {
    this.engine = engine
  }

  track(event: string, data: Record<string, string | number>): void {
    if (!this.engine) return
    const blobs: string[] = [event]
    const doubles: number[] = []
    for (const [key, val] of Object.entries(data)) {
      if (typeof val === 'string') {
        blobs.push(val)
      } else {
        doubles.push(val)
      }
    }
    try {
      this.engine.writeDataPoint({ blobs, doubles })
    } catch {
      // Best-effort — analytics should never break request handling
    }
  }

  pageView(route: string, params?: Record<string, string>): void {
    const data: Record<string, string | number> = { route }
    if (params) {
      for (const [key, val] of Object.entries(params)) {
        data[key] = val
      }
    }
    this.track('page_view', data)
  }

  apiCall(endpoint: string, latencyMs: number, cacheHit: boolean): void {
    this.track('api_call', {
      endpoint,
      latencyMs,
      cacheHit: cacheHit ? 1 : 0,
    })
  }

  search(query: string, type: string, resultsCount: number): void {
    this.track('search', {
      query,
      type,
      resultsCount,
    })
  }

  error(endpoint: string, statusCode: number, message: string): void {
    this.track('error', {
      endpoint,
      statusCode,
      message,
    })
  }
}
