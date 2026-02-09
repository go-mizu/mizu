import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { GraphQLClient } from '../graphql'
import { Cache } from '../cache'
import { parseConversation } from '../parse'
import { renderLayout, renderTweetDetail, renderError } from '../html'
import { gqlConversationTimeline, tweetDetailFieldToggles, CACHE_TWEET } from '../config'

const app = new Hono<HonoEnv>()

app.get('/:username/status/:id', async (c) => {
  const tweetID = c.req.param('id')
  const gql = new GraphQLClient(c.env.X_AUTH_TOKEN, c.env.X_CT0, c.env.X_BEARER_TOKEN)
  const cache = new Cache(c.env.KV)

  try {
    const cacheKey = `tweet:${tweetID}`
    let cached = await cache.get<{ mainTweet: unknown; replies: unknown[] }>(cacheKey)

    if (!cached) {
      const data = await gql.doGraphQL(gqlConversationTimeline, {
        focalTweetId: tweetID,
        referrer: 'tweet',
        with_rux_injections: false,
        rankingMode: 'Relevance',
        includePromotedContent: true,
        withCommunity: true,
        withQuickPromoteEligibilityTweetFields: true,
        withBirdwatchNotes: true,
        withVoice: true,
        withV2Timeline: true,
      }, tweetDetailFieldToggles)

      const result = parseConversation(data, tweetID)
      if (result.mainTweet) {
        cached = { mainTweet: result.mainTweet, replies: result.replies }
        await cache.set(cacheKey, cached, CACHE_TWEET)
      }
    }

    if (!cached || !cached.mainTweet) {
      return c.html(renderError('Tweet not found', 'This tweet may have been deleted.'), 404)
    }

    const tweet = cached.mainTweet as Parameters<typeof renderTweetDetail>[0]
    const replies = (cached.replies || []) as Parameters<typeof renderTweetDetail>[1]

    const content = `<div class="sh"><h2>Post</h2></div>` + renderTweetDetail(tweet, replies)
    return c.html(renderLayout(`${tweet.name} (@${tweet.username})`, content))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
