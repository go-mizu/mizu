/**
 * Perplexity Official API streaming — real token-by-token delivery.
 * Uses api.perplexity.ai/chat/completions with stream: true.
 * TTFB ~1.3s (real tokens, not simulated).
 */

import type { SearchResult, Citation, WebResult, ThinkingStep } from './types'

const API_URL = 'https://api.perplexity.ai/chat/completions'

/** Map our mode to API model name. */
function resolveModel(mode: string): string {
  switch (mode) {
    case 'pro': return 'sonar-pro'
    case 'reasoning': return 'sonar-reasoning-pro'
    case 'deep': return 'sonar-deep-research'
    default: return 'sonar'
  }
}

function extractDomain(url: string): string {
  try { return new URL(url).hostname.replace(/^www\./, '') } catch { return url }
}

function favicon(url: string): string {
  return `https://www.google.com/s2/favicons?domain=${extractDomain(url)}&sz=32`
}

interface APISearchResult {
  title: string
  url: string
  snippet: string
  date?: string
}

/** Stream search via official Perplexity API — real token streaming. */
export function streamSearchAPI(
  apiKey: string,
  query: string,
  mode: string = 'auto',
  conversationHistory?: Array<{ role: string; content: string }>,
): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()

  function sendEvent(controller: ReadableStreamDefaultController, event: string, data: unknown): void {
    controller.enqueue(encoder.encode(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`))
  }

  return new ReadableStream({
    async start(controller) {
      const t0 = Date.now()

      try {
        const model = resolveModel(mode)

        // Emit progress immediately
        sendEvent(controller, 'progress', { status: 'searching', message: 'Searching the web...' })

        // Build messages array
        const messages: Array<{ role: string; content: string }> = []
        if (conversationHistory?.length) {
          messages.push(...conversationHistory)
        }
        messages.push({ role: 'user', content: query })

        const resp = await fetch(API_URL, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${apiKey}`,
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            model,
            messages,
            stream: true,
            return_related_questions: true,
            return_images: true,
            web_search_options: { search_context_size: 'medium' },
          }),
        })

        const tFetch = Date.now()

        if (!resp.ok) {
          const text = await resp.text()
          sendEvent(controller, 'error', { message: `API ${resp.status}: ${text.slice(0, 300)}` })
          controller.close()
          return
        }

        if (!resp.body) {
          sendEvent(controller, 'error', { message: 'No response body from API' })
          controller.close()
          return
        }

        const reader = resp.body.getReader()
        const decoder = new TextDecoder()
        let buffer = ''
        let fullAnswer = ''
        let tFirstToken = 0
        let citations: string[] = []
        let searchResults: APISearchResult[] = []
        let relatedQueries: string[] = []
        let images: Array<{ type: string; url: string; title: string; sourceUrl: string }> = []
        let modelUsed = model
        let answering = false

        while (true) {
          const { done, value } = await reader.read()
          if (done) break

          buffer += decoder.decode(value, { stream: true })

          // SSE events separated by \n\n
          const events = buffer.split('\n\n')
          buffer = events.pop() || ''

          for (const event of events) {
            const trimmed = event.trim()
            if (!trimmed) continue

            for (const line of trimmed.split('\n')) {
              if (!line.startsWith('data: ')) continue
              const data = line.slice(6)
              if (data === '[DONE]') continue

              let chunk: Record<string, unknown>
              try { chunk = JSON.parse(data) } catch { continue }

              const choices = chunk.choices as Array<Record<string, unknown>> | undefined
              if (!choices?.length) continue

              const choice = choices[0]
              const delta = choice.delta as Record<string, unknown> | undefined
              const content = delta?.content as string | undefined

              if (content) {
                if (!tFirstToken) {
                  tFirstToken = Date.now()
                  sendEvent(controller, 'progress', { status: 'answering', message: 'Generating answer...' })
                  // Emit thinking step for search phase
                  sendEvent(controller, 'thinking', {
                    step: { stepType: 'SEARCH_WEB', content: `Searching: ${query}`, timestamp: tFetch - t0 },
                  })
                  answering = true
                }

                fullAnswer += content
                sendEvent(controller, 'chunk', { delta: content, full: fullAnswer })
              }

              // Citations + search_results arrive in final chunk
              if (choice.finish_reason) {
                if (Array.isArray(chunk.citations)) {
                  citations = chunk.citations as string[]
                }
                if (Array.isArray(chunk.search_results)) {
                  searchResults = chunk.search_results as APISearchResult[]
                }
                if (Array.isArray(chunk.related_questions)) {
                  relatedQueries = chunk.related_questions as string[]
                }
                if (Array.isArray(chunk.images)) {
                  images = (chunk.images as Array<Record<string, unknown>>).map(img => ({
                    type: 'image',
                    url: (img.url as string) || (img.image_url as string) || '',
                    title: (img.title as string) || '',
                    sourceUrl: (img.source_url as string) || (img.origin_url as string) || '',
                  }))
                }
                modelUsed = (chunk.model as string) || model
              }
            }
          }
        }

        // Build citations from API format
        const webResults: WebResult[] = searchResults.map(sr => ({
          name: sr.title || '',
          url: sr.url || '',
          snippet: sr.snippet || '',
          date: sr.date,
        }))

        // If no detailed search_results, build from citation URLs
        const builtCitations: Citation[] = searchResults.length > 0
          ? searchResults.map(sr => ({
            url: sr.url,
            title: sr.title || extractDomain(sr.url),
            snippet: sr.snippet || '',
            date: sr.date,
            domain: extractDomain(sr.url),
            favicon: favicon(sr.url),
          }))
          : citations.map(url => ({
            url,
            title: extractDomain(url),
            snippet: '',
            domain: extractDomain(url),
            favicon: favicon(url),
          }))

        // Emit sources
        if (builtCitations.length > 0) {
          sendEvent(controller, 'sources', { citations: builtCitations, webResults })
        }

        // Emit media
        if (images.length > 0) {
          sendEvent(controller, 'media', { images, videos: [] })
        }

        // Emit related
        if (relatedQueries.length > 0) {
          sendEvent(controller, 'related', { queries: relatedQueries })
        }

        // Build final result
        const result: SearchResult = {
          query,
          answer: fullAnswer,
          citations: builtCitations,
          webResults,
          relatedQueries,
          images: images as SearchResult['images'],
          videos: [],
          thinkingSteps: [{
            stepType: 'SEARCH_WEB',
            content: `Searched: ${query}`,
            timestamp: tFetch - t0,
          }],
          backendUUID: '',
          mode,
          model: modelUsed,
          durationMs: Date.now() - t0,
          createdAt: new Date().toISOString(),
        }

        const timing = {
          sessionMs: 0,
          fetchMs: tFetch - t0,
          firstByteMs: tFetch - t0,
          firstAnswerMs: tFirstToken ? tFirstToken - t0 : 0,
          totalMs: Date.now() - t0,
        }

        sendEvent(controller, 'done', { result, timing })
        controller.close()
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e)
        sendEvent(controller, 'error', { message: msg })
        controller.close()
      }
    },
  })
}
