import { Hono } from 'hono'
import type { Context } from 'hono'
import type { HonoEnv, Tweet } from '../types'
import { DB } from '../cache'
import { renderLayout, renderArticlePage, renderError } from '../html'
import { CACHE_TWEET } from '../config'
import { fetchTweetConversation } from '../tweet-fetch'
import { fetchArticleBody } from '../article-fetch'
import { isRateLimitedError } from '../rate-limit'

const app = new Hono<HonoEnv>()

// In-memory cache (survives across requests within same isolate)
const memCache = new Map<string, { tweet: Tweet; body: string; exp: number }>()

async function getArticle(c: Context<HonoEnv>, tweetID: string, username: string, nocache: boolean) {
  // 1. In-memory cache (only return if body is present)
  if (!nocache) {
    const mem = memCache.get(tweetID)
    if (mem && mem.exp > Date.now() && mem.body) {
      return { tweet: mem.tweet, body: mem.body }
    }
  }

  // 2. D1 cache (body stored in its own column — never lost in JSON)
  const db = new DB(c.env.DB)
  if (!nocache) {
    const row = await db.getArticle(tweetID)
    if (row && row.body) {
      const tweet = JSON.parse(row.tweetData) as Tweet
      memCache.set(tweetID, { tweet, body: row.body, exp: Date.now() + 600_000 })
      return { tweet, body: row.body }
    }
  }

  // 3. Fetch fresh from X API + Browser Rendering
  const result = await fetchTweetConversation(c.env, tweetID, '', true)
  if (!result.mainTweet) return null

  const tweet = result.mainTweet as Tweet
  let body = tweet.articleBody || ''

  if (!body) {
    const articleURL = `https://x.com/${username}/article/${tweetID}`
    body = await fetchArticleBody(c.env, articleURL)
  }

  // Only cache if we got a body
  if (body) {
    memCache.set(tweetID, { tweet, body, exp: Date.now() + 600_000 })
    await db.setArticle(tweetID, JSON.stringify(tweet), body, CACHE_TWEET)
  }

  return { tweet, body }
}

// Rendered article page
app.get('/:username/article/:id', async (c) => {
  const tweetID = c.req.param('id')
  const username = c.req.param('username')
  const debug = c.req.query('debug') === '1'
  const nocache = debug || c.req.query('nocache') === '1'

  try {
    const data = await getArticle(c, tweetID, username, nocache)
    if (!data) {
      return c.html(renderError('Article not found', 'This article may have been deleted.'), 404)
    }

    if (debug) {
      return c.json({
        hasBody: !!data.body,
        bodyLength: data.body?.length || 0,
        bodyPreview: data.body?.slice(0, 500) || '',
        tweet: {
          id: data.tweet.id,
          title: data.tweet.title,
          username: data.tweet.username,
          urls: data.tweet.urls,
        }
      })
    }

    const tweet: Tweet = { ...data.tweet, articleBody: data.body || data.tweet.articleBody || '' }
    const pageTitle = tweet.title || `Article by ${tweet.name}`
    return c.html(renderLayout(pageTitle, renderArticlePage(tweet)))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (debug) return c.json({ error: msg })
    if (isRateLimitedError(e)) return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

// Raw markdown (opens in browser as plain text)
app.get('/:username/article/:id/raw', async (c) => {
  const tweetID = c.req.param('id')
  const username = c.req.param('username')

  try {
    const data = await getArticle(c, tweetID, username, false)
    if (!data || !data.body) {
      return c.text('Article body not available', 404)
    }

    const title = data.tweet.title || `Article by ${data.tweet.name}`
    const md = `# ${title}\n\n**@${data.tweet.username}** · ${data.tweet.postedAt}\n\n---\n\n${data.body}\n`

    return new Response(md, {
      headers: { 'Content-Type': 'text/plain; charset=utf-8' }
    })
  } catch (e) {
    return c.text(e instanceof Error ? e.message : String(e), 500)
  }
})

export default app
