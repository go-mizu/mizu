import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  fetchProfileWithFallback, fetchSearchTweetsWithFallback,
  fetchSearchUsersWithFallback, fetchUserTimelineWithFallback,
} from './fallback-fetch'
import type { Env } from './types'

const env: Env = {
  KV: {} as KVNamespace,
  X_AUTH_TOKEN: 'x_auth',
  X_CT0: 'x_ct0',
  X_BEARER_TOKEN: 'Bearer token',
  NITTER_INSTANCES: 'https://nitter.test',
  ENVIRONMENT: 'test',
}

const PROFILE_HTML = `
<html><body>
  <div class="profile-card">
    <a class="profile-card-fullname" href="/example">Example</a>
    <a class="profile-card-username" href="/example">@example</a>
  </div>
</body></html>
`

const TIMELINE_HTML = `
<html><body>
  <div class="timeline-item">
    <a class="fullname" href="/example">Example</a>
    <a class="username" href="/example">@example</a>
    <span class="tweet-date"><a href="/example/status/99" title="Tue Feb 11 16:00:00 +0000 2025">x</a></span>
    <div class="tweet-content media-body">from fallback</div>
  </div>
</body></html>
`

const USERS_HTML = `
<html><body>
  <div class="timeline-item">
    <a class="fullname" href="/alice">Alice</a>
    <a class="username" href="/alice">@alice</a>
  </div>
</body></html>
`

afterEach(() => {
  vi.restoreAllMocks()
})

describe('fallback fetch', () => {
  it('falls back to nitter profile on graphql 429', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('x.com/i/api/graphql')) return new Response('', { status: 429 })
      if (url.includes('nitter.test/example')) return new Response(PROFILE_HTML, { status: 200 })
      return new Response('not found', { status: 404 })
    })

    const profile = await fetchProfileWithFallback(env, 'example')
    expect(profile).not.toBeNull()
    expect(profile?.username).toBe('example')
  })

  it('falls back to nitter user tweets on graphql 429', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('x.com/i/api/graphql')) return new Response('', { status: 429 })
      if (url.includes('nitter.test/example')) return new Response(TIMELINE_HTML, { status: 200 })
      return new Response('not found', { status: 404 })
    })

    const timeline = await fetchUserTimelineWithFallback(env, 'example', 'tweets', '', '123')
    expect(timeline.tweets.length).toBeGreaterThan(0)
    expect(timeline.tweets[0].id).toBe('99')
  })

  it('falls back to nitter search tweets and users on graphql 429', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('x.com/i/api/graphql')) return new Response('', { status: 429 })
      if (url.includes('/search?f=users')) return new Response(USERS_HTML, { status: 200 })
      if (url.includes('/search?f=tweets')) return new Response(TIMELINE_HTML, { status: 200 })
      return new Response('not found', { status: 404 })
    })

    const tweets = await fetchSearchTweetsWithFallback(env, 'golang', 'Top', '')
    expect(tweets.tweets.length).toBeGreaterThan(0)

    const users = await fetchSearchUsersWithFallback(env, 'golang', '')
    expect(users.users.length).toBeGreaterThan(0)
    expect(users.users[0].username).toBe('alice')
  })
})
