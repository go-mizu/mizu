import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { InstagramClient } from '../instagram'
import { Cache } from '../cache'
import { parsePostDetail } from '../parse'
import { renderLayout, renderReelDetail, renderError } from '../html'
import { CACHE_POST } from '../config'
import type { Reel } from '../types'

const app = new Hono<HonoEnv>()

app.get('/:shortcode', async (c) => {
  const shortcode = c.req.param('shortcode')
  const client = new InstagramClient(c.env.INSTA_SESSION_ID, c.env.INSTA_CSRF_TOKEN, c.env.INSTA_DS_USER_ID, c.env.INSTA_MID, c.env.INSTA_IG_DID)
  const cache = new Cache(c.env.KV)

  try {
    const cacheKey = `reel:${shortcode}`
    let post = await cache.get<any>(cacheKey)
    if (!post) {
      const data = await client.getPostDetail(shortcode)
      post = parsePostDetail(data)
      if (post) await cache.set(cacheKey, post, CACHE_POST)
    }

    if (!post) return c.html(renderError('Reel not found', 'This reel may have been deleted.'), 404)

    const reel: Reel = {
      id: post.id,
      shortcode: post.shortcode,
      caption: post.caption,
      displayUrl: post.displayUrl,
      videoUrl: post.videoUrl,
      width: post.width,
      height: post.height,
      likeCount: post.likeCount,
      commentCount: post.commentCount,
      viewCount: post.viewCount,
      playCount: post.viewCount,
      takenAt: post.takenAt,
      ownerUsername: post.ownerUsername,
    }

    const content = renderReelDetail(reel)
    return c.html(renderLayout(`Reel by ${reel.ownerUsername}`, content))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
export { renderReelsGrid } from './profile'
