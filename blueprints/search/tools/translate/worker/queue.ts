/**
 * Queue consumer — translates text batches and writes results to KV.
 *
 * Message format: { texts: string[], tl: string }
 * On success: each translation written to KV as `t:{tl}:{hash}` → `{sl}\t{translation}`.
 * On failure: retries up to 3 times with exponential backoff.
 */

import type { Env, TranslateMessage } from './types'
import { batchTranslate, writeTranslations } from './translate'

export async function handleQueue(
  batch: MessageBatch<TranslateMessage>,
  env: Env,
): Promise<void> {
  console.log(`[queue] BATCH received=${batch.messages.length} queue=${batch.queue}`)

  for (const msg of batch.messages) {
    const { texts, tl } = msg.body
    const attempt = msg.attempts

    console.log(`[queue] MSG id=${msg.id} texts=${texts.length} tl=${tl} attempt=${attempt}`)

    try {
      const { translations, detectedSl } = await batchTranslate(texts, 'auto', tl)
      const sl = detectedSl || 'en'

      console.log(`[queue] TRANSLATED ${translations.size}/${texts.length} sl=${sl}`)

      await writeTranslations(env.TRANSLATE_CACHE, translations, tl, sl)

      console.log(`[queue] KV_WRITTEN ${translations.size} entries`)
      msg.ack()
    } catch (e) {
      const err = e instanceof Error ? e.message : String(e)
      console.log(`[queue] ERROR id=${msg.id} err=${err} attempt=${attempt}`)

      if (attempt < 3) {
        msg.retry({ delaySeconds: attempt * 10 })
      } else {
        console.log(`[queue] DEAD_LETTER id=${msg.id} texts=${texts.length} tl=${tl}`)
        msg.ack() // give up after 3 attempts
      }
    }
  }
}
