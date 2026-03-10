import { Hono } from 'hono'
import type { HonoEnv, Tweet } from '../types'
import { DB } from '../cache'
import { renderLayout, renderTweetDetail, renderError } from '../html'
import { CACHE_TWEET } from '../config'
import { fetchTweetConversation } from '../tweet-fetch'
import { isRateLimitedError } from '../rate-limit'

const app = new Hono<HonoEnv>()

function tweetToMarkdown(t: Tweet): string {
  const lines: string[] = []
  lines.push(`**${t.name}** (@${t.username})`)
  lines.push('')
  lines.push(t.text)
  if (t.photos.length) { lines.push(''); for (const p of t.photos) lines.push(`![](${p})`) }
  lines.push('')
  lines.push(`---`)
  lines.push(`${t.likes} Likes · ${t.retweets} Reposts · ${t.views} Views · ${t.postedAt}`)
  lines.push(`https://x.com/${t.username}/status/${t.id}`)
  return lines.join('\n')
}

app.get('/:username/status/:id', async (c) => {
  const tweetID = c.req.param('id')
  const username = c.req.param('username')
  const cursor = c.req.query('cursor') || ''
  const db = new DB(c.env.DB)

  try {
    let cached = await db.getTweet<{ mainTweet: unknown; replies: unknown[]; cursor: string }>(tweetID, cursor)

    if (!cached) {
      const result = await fetchTweetConversation(c.env, tweetID, cursor, true)

      // If paginated and mainTweet missing, try first-page cache
      if (cursor && !result.mainTweet) {
        const firstPage = await db.getTweet<{ mainTweet: unknown }>(tweetID, '')
        if (firstPage?.mainTweet) {
          cached = { mainTweet: firstPage.mainTweet, replies: result.replies, cursor: result.cursor }
        }
      }

      if (!cached && result.mainTweet) {
        cached = { mainTweet: result.mainTweet, replies: result.replies, cursor: result.cursor }
      }

      if (cached) {
        await db.setTweet(tweetID, cursor, cached, CACHE_TWEET)
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

// Raw markdown for tweet
app.get('/:username/status/:id/raw', async (c) => {
  const tweetID = c.req.param('id')
  const db = new DB(c.env.DB)

  try {
    let cached = await db.getTweet<{ mainTweet: Tweet }>(tweetID, '')
    if (!cached) {
      const result = await fetchTweetConversation(c.env, tweetID, '', true)
      if (result.mainTweet) {
        cached = { mainTweet: result.mainTweet as Tweet }
        await db.setTweet(tweetID, '', { mainTweet: result.mainTweet, replies: result.replies, cursor: result.cursor }, CACHE_TWEET)
      }
    }
    if (!cached?.mainTweet) return c.text('Tweet not found', 404)

    return new Response(tweetToMarkdown(cached.mainTweet), {
      headers: { 'Content-Type': 'text/plain; charset=utf-8' }
    })
  } catch (e) {
    return c.text(e instanceof Error ? e.message : String(e), 500)
  }
})

export default app
