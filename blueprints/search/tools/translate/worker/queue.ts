/**
 * Queue consumer — translates text batches and writes results to KV.
 *
 * Message format: { texts: string[], tl: string }
 * On success: each translation written to KV as `t:{tl}:{hash}` → `{sl}\t{translation}`.
 * On failure: retries up to 3 times with exponential backoff.
 * Analytics: tracks every queue operation (received, translated, retried, dead-lettered).
 */

import type { Env, TranslateMessage } from './types'
import { batchTranslate, writeTranslations } from './translate'
import { track } from './analytics'

export async function handleQueue(
  batch: MessageBatch<TranslateMessage>,
  env: Env,
): Promise<void> {
  console.log(`[queue] BATCH received=${batch.messages.length} queue=${batch.queue}`)

  for (const msg of batch.messages) {
    const { texts, tl } = msg.body
    const attempt = msg.attempts
    const t0 = Date.now()

    console.log(`[queue] MSG id=${msg.id} texts=${texts.length} tl=${tl} attempt=${attempt}`)

    try {
      const { translations, detectedSl } = await batchTranslate(texts, 'auto', tl)
      const sl = detectedSl || 'en'

      console.log(`[queue] TRANSLATED ${translations.size}/${texts.length} sl=${sl}`)

      await writeTranslations(env.TRANSLATE_CACHE, translations, tl, sl)

      console.log(`[queue] KV_WRITTEN ${translations.size} entries`)

      track(env, {
        event: 'queue',
        sl,
        tl,
        latencyMs: Date.now() - t0,
        chars: texts.reduce((sum, t) => sum + t.length, 0),
        success: true,
        cacheHits: translations.size,
        total: texts.length,
      })

      msg.ack()
    } catch (e) {
      const err = e instanceof Error ? e.message : String(e)
      console.log(`[queue] ERROR id=${msg.id} err=${err} attempt=${attempt}`)

      if (attempt < 3) {
        track(env, {
          event: 'queue',
          tl,
          extra: `retry:${attempt + 1} ${err}`,
          latencyMs: Date.now() - t0,
          chars: texts.reduce((sum, t) => sum + t.length, 0),
          success: false,
          total: texts.length,
        })
        msg.retry({ delaySeconds: (attempt + 1) * 10 })
      } else {
        console.log(`[queue] DEAD_LETTER id=${msg.id} texts=${texts.length} tl=${tl} err=${err}`)
        track(env, {
          event: 'error',
          tl,
          provider: 'queue_dlq',
          extra: err,
          latencyMs: Date.now() - t0,
          chars: texts.reduce((sum, t) => sum + t.length, 0),
          success: false,
          total: texts.length,
        })
        msg.ack() // give up after 3 attempts
      }
    }
  }
}
