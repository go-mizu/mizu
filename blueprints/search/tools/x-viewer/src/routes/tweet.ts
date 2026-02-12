import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { Cache } from '../cache'
import { renderLayout, renderTweetDetail, renderError } from '../html'
import { CACHE_TWEET } from '../config'
import { fetchTweetConversation } from '../tweet-fetch'
import { isRateLimitedError } from '../rate-limit'

const app = new Hono<HonoEnv>()

app.get('/:username/status/:id', async (c) => {
  const tweetID = c.req.param('id')
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const cache = new Cache(c.env.KV)

  try {
    const cacheKey = cursor ? `tweet:${tweetID}:${cursor}` : `tweet:${tweetID}`
    let cached = await cache.get<{ mainTweet: unknown; replies: unknown[]; cursor: string }>(cacheKey)

    if (!cached) {
      const result = await fetchTweetConversation(c.env, tweetID, cursor, true)

      // If paginated and mainTweet missing, try first-page cache
      if (cursor && !result.mainTweet) {
        const firstPage = await cache.get<{ mainTweet: unknown }>(
          `tweet:${tweetID}`
        )
        if (firstPage?.mainTweet) {
          cached = { mainTweet: firstPage.mainTweet, replies: result.replies, cursor: result.cursor }
        }
      }

      if (!cached && result.mainTweet) {
        cached = { mainTweet: result.mainTweet, replies: result.replies, cursor: result.cursor }
      }

      if (cached) {
        await cache.set(cacheKey, cached, CACHE_TWEET)
      }
    }

    if (!cached || !cached.mainTweet) {
      return c.html(renderError('Tweet not found', 'This tweet may have been deleted.'), 404)
    }

    const tweet = cached.mainTweet as Parameters<typeof renderTweetDetail>[0]
    const replies = (cached.replies || []) as Parameters<typeof renderTweetDetail>[1]
    const nextCursor = (cached.cursor || '') as string
    const tweetPath = `/${username}/status/${tweetID}`

    const content = `<div class="sh"><h2>Post</h2></div>` + renderTweetDetail(tweet, replies, nextCursor, tweetPath)
    return c.html(renderLayout(`${tweet.name} (@${tweet.username})`, content))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (isRateLimitedError(e)) return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
