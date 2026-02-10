import { Hono } from 'hono'
import type { HonoEnv } from '../types'
import { GraphQLClient } from '../graphql'
import { Cache } from '../cache'
import { parseGraphList, parseListTimeline, parseListMembers } from '../parse'
import { renderLayout, renderTweetCard, renderUserCard, renderPagination, renderError } from '../html'
import { gqlListById, gqlListTweets, gqlListMembers, CACHE_LIST } from '../config'

const app = new Hono<HonoEnv>()

app.get('/:id', async (c) => {
  const listID = c.req.param('id')
  const tab = c.req.query('tab') || 'tweets'
  const cursor = c.req.query('cursor') || ''
  const gql = new GraphQLClient(c.env.X_AUTH_TOKEN, c.env.X_CT0, c.env.X_BEARER_TOKEN)
  const cache = new Cache(c.env.KV)

  try {
    // Fetch list metadata
    const listKey = `list:${listID}`
    let list = await cache.get<ReturnType<typeof parseGraphList>>(listKey)
    if (!list) {
      const data = await gql.doGraphQL(gqlListById, { listId: listID }, '')
      list = parseGraphList(data)
      if (list) await cache.set(listKey, list, CACHE_LIST)
    }

    if (!list) {
      return c.html(renderError('List not found', 'This list may have been deleted or is private.'), 404)
    }

    const basePath = `/i/lists/${listID}`
    let content = `<div class="sh"><h2>${list.name}</h2>${list.description ? `<div class="sh-sub">${list.description}</div>` : ''}<div class="sh-sub">${list.memberCount} members · by @${list.ownerName}</div></div>`
    content += `<div class="tabs"><a href="${basePath}" class="${tab === 'tweets' ? 'active' : ''}">Tweets</a><a href="${basePath}?tab=members" class="${tab === 'members' ? 'active' : ''}">Members</a></div>`

    if (tab === 'members') {
      const membersKey = `list-members:${listID}:${cursor}`
      let membersData = await cache.get<{ users: unknown[]; cursor: string }>(membersKey)
      if (!membersData) {
        const vars: Record<string, unknown> = { listId: listID, count: 200 }
        if (cursor) vars.cursor = cursor
        const data = await gql.doGraphQL(gqlListMembers, vars, '')
        const result = parseListMembers(data)
        membersData = { users: result.users, cursor: result.cursor }
        await cache.set(membersKey, membersData, CACHE_LIST)
      }

      const users = (membersData.users || []) as Parameters<typeof renderUserCard>[0][]
      const nextCursor = membersData.cursor as string

      if (users.length === 0) {
        content += `<div class="err"><p>No members found.</p></div>`
      } else {
        for (const u of users) content += renderUserCard(u)
      }
      content += renderPagination(nextCursor, `${basePath}?tab=members`)
    } else {
      const tweetsKey = `list-tweets:${listID}:${cursor}`
      let tweetsData = await cache.get<{ tweets: unknown[]; cursor: string }>(tweetsKey)
      if (!tweetsData) {
        const vars: Record<string, unknown> = { rest_id: listID, count: 40 }
        if (cursor) vars.cursor = cursor
        const data = await gql.doGraphQL(gqlListTweets, vars, '')
        const result = parseListTimeline(data)
        tweetsData = { tweets: result.tweets, cursor: result.cursor }
        await cache.set(tweetsKey, tweetsData, CACHE_LIST)
      }

      const tweets = (tweetsData.tweets || []) as Parameters<typeof renderTweetCard>[0][]
      const nextCursor = tweetsData.cursor as string

      for (const tweet of tweets) {
        content += renderTweetCard(tweet)
      }
      content += renderPagination(nextCursor, basePath)
    }

    return c.html(renderLayout(`${list.name} - List`, content))
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e)
    if (msg.includes('rate limited')) {
      return c.html(renderError('Rate Limited', 'Too many requests. Please try again later.'), 429)
    }
    return c.html(renderError('Error', msg), 500)
  }
})

export default app
