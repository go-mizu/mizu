import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { SessionManager } from '../session'
import { Cache } from '../cache'
import { parsePostDetail, parseComments } from '../parse'
import { renderLayout, renderPostDetail, renderPagination, renderError } from '../html'
import { CACHE_POST, CACHE_COMMENTS, qhComments } from '../config'

const app = new Hono<HonoEnv>()

app.get('/:shortcode', async (c) => {
  const shortcode = c.req.param('shortcode')
  const client = await new SessionManager(c.env).getClient()
  const cache = new Cache(c.env.KV)

  try {
    // Fetch post
    const postKey = `post:${shortcode}`
    let post = await cache.get<any>(postKey)
    if (!post) {
      const data = await client.getPostDetail(shortcode)
      post = parsePostDetail(data)
      if (post) await cache.set(postKey, post, CACHE_POST)
    }

    if (!post) {
      return c.html(renderError('Post not found', 'This post may have been deleted or is not available.'), 404)
    }

    // Fetch comments
    const commentKey = `comments:${shortcode}`
    let commentsData = await cache.get<any>(commentKey)
    if (!commentsData) {
      try {
        const data = await client.graphqlQuery(qhComments, {
          shortcode,
          first: 24,
          after: '',
        })
        commentsData = parseComments(data)
        await cache.set(commentKey, commentsData, CACHE_COMMENTS)
      } catch {
        commentsData = { comments: [], cursor: '', hasMore: false }
      }
    }

    const content = renderPostDetail(post, commentsData.comments || [], commentsData.cursor || '')
    const hasCarousel = (post.children?.length || 0) > 1

    return c.html(renderLayout(
      `${post.ownerUsername} on Instagram`,
      content,
      { hasCarousel, hasComments: true }
    ))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) return c.html(renderError('Rate Limited', 'Too many requests.'), 429)
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
