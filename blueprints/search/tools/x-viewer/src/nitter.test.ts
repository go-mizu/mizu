import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  decodeNitterCursor, encodeNitterCursor, fetchProfileFromNitter,
  fetchSearchTweetsFromNitter, fetchSearchUsersFromNitter, fetchTweetFromNitter,
  fetchUserTweetsFromNitter, parseNitterProfileHTML, parseNitterSearchUsersHTML,
  parseNitterTimelineHTML, parseNitterTweetHTML, isNitterCursor,
} from './nitter'

const TWEET_HTML = `
<!doctype html>
<html>
<head>
  <link rel="canonical" href="https://nitter.test/example/status/1234567890#m">
</head>
<body>
  <div class="main-tweet timeline-item">
    <a class="fullname" href="/example">Example Name</a>
    <a class="username" href="/example">@example</a>
    <img class="avatar" src="/pic/profile.jpg">
    <span class="tweet-date"><a href="/example/status/1234567890" title="Tue Feb 11 16:00:00 +0000 2025">x</a></span>
    <div class="tweet-content media-body">Hello from #golang @friend https://example.com</div>
    <div class="tweet-stats">5 Replies 3 Retweets 7 Likes</div>
    <a class="still-image" href="/pic/media%2Fabc.jpg"></a>
    <video class="gif" poster="/pic/thumb.jpg"><source src="/pic/video.mp4"></video>
  </div>
</body>
</html>
`

const PROFILE_HTML = `
<!doctype html>
<html>
<body>
  <img class="profile-banner" src="/pic/banner.jpg">
  <div class="profile-card">
    <img class="profile-card-avatar" src="/pic/avatar.jpg">
    <a class="profile-card-fullname" href="/example">Example Name</a>
    <a class="profile-card-username" href="/example">@example</a>
    <div class="profile-bio">Building tools</div>
    <div class="profile-location">Earth</div>
    <div class="profile-website"><a href="https://example.com">example.com</a></div>
    <div class="profile-joindate">Joined Jan 2020</div>
  </div>
  <ul class="profile-statlist">
    <li><a href="/example">123 Posts</a></li>
    <li><a href="/example/following">10 Following</a></li>
    <li><a href="/example/followers">20 Followers</a></li>
  </ul>
</body>
</html>
`

const TIMELINE_HTML = `
<!doctype html>
<html>
<body>
  <div class="timeline-item">
    <a class="fullname" href="/example">Example Name</a>
    <a class="username" href="/example">@example</a>
    <img class="avatar" src="/pic/avatar.jpg">
    <span class="tweet-date"><a href="/example/status/111" title="Tue Feb 11 16:00:00 +0000 2025">x</a></span>
    <div class="tweet-content media-body">Tweet one #one @friend</div>
    <div class="tweet-stats">1 Reply 2 Retweets 3 Likes</div>
  </div>
  <div class="timeline-item">
    <a class="fullname" href="/example">Example Name</a>
    <a class="username" href="/example">@example</a>
    <img class="avatar" src="/pic/avatar.jpg">
    <span class="tweet-date"><a href="/example/status/111" title="Tue Feb 11 16:00:00 +0000 2025">x</a></span>
    <div class="tweet-content media-body">Duplicate tweet one</div>
    <div class="tweet-stats">1 Reply 2 Retweets 3 Likes</div>
  </div>
  <div class="timeline-item">
    <a class="fullname" href="/example">Example Name</a>
    <a class="username" href="/example">@example</a>
    <img class="avatar" src="/pic/avatar.jpg">
    <span class="tweet-date"><a href="/example/status/222" title="Wed Feb 12 10:00:00 +0000 2025">x</a></span>
    <div class="tweet-content media-body">Tweet two</div>
    <div class="tweet-stats">2 Replies 4 Retweets 6 Likes</div>
  </div>
  <div class="show-more"><a href="/example?cursor=ABC123">Show more</a></div>
</body>
</html>
`

const USERS_HTML = `
<!doctype html>
<html>
<body>
  <div class="timeline-item">
    <a class="fullname" href="/alice">Alice</a>
    <a class="username" href="/alice">@alice</a>
    <img class="avatar" src="/pic/alice.jpg">
    <div class="profile-bio">Alice bio</div>
  </div>
  <div class="timeline-item">
    <a class="fullname" href="/alice">Alice Duplicate</a>
    <a class="username" href="/alice">@ALICE</a>
    <img class="avatar" src="/pic/alice2.jpg">
    <div class="profile-bio">Alice bio duplicate</div>
  </div>
  <div class="timeline-item">
    <a class="fullname" href="/bob">Bob</a>
    <a class="username" href="/bob">@bob</a>
    <img class="avatar" src="/pic/bob.jpg">
    <div class="profile-bio">Bob bio</div>
  </div>
  <div class="show-more"><a href="/search?f=users&q=golang&cursor=USERS2">Show more</a></div>
</body>
</html>
`

afterEach(() => {
  vi.restoreAllMocks()
})

describe('nitter parsers', () => {
  it('parses tweet detail html', () => {
    const parsed = parseNitterTweetHTML(TWEET_HTML, '1234567890', 'https://nitter.test/i/status/1234567890')
    expect(parsed.mainTweet).not.toBeNull()
    expect(parsed.mainTweet?.id).toBe('1234567890')
    expect(parsed.mainTweet?.username).toBe('example')
    expect(parsed.mainTweet?.name).toBe('Example Name')
    expect(parsed.mainTweet?.text).toContain('#golang')
    expect(parsed.mainTweet?.likes).toBe(7)
    expect(parsed.mainTweet?.retweets).toBe(3)
    expect(parsed.mainTweet?.replies).toBe(5)
    expect(parsed.mainTweet?.urls).toContain('https://example.com')
    expect(parsed.mainTweet?.hashtags).toContain('golang')
    expect(parsed.mainTweet?.mentions).toContain('@friend')
    expect(parsed.mainTweet?.photos[0]).toBe('https://nitter.test/pic/media%2Fabc.jpg')
    expect(parsed.mainTweet?.gifs[0]).toBe('https://nitter.test/pic/video.mp4')
    expect(parsed.mainTweet?.videoThumbnails[0]).toBe('https://nitter.test/pic/thumb.jpg')
  })

  it('returns null main tweet for non-tweet html', () => {
    const parsed = parseNitterTweetHTML('<html><body>none</body></html>', '123', 'https://nitter.test/i/status/123')
    expect(parsed.mainTweet).toBeNull()
    expect(parsed.replies).toEqual([])
    expect(parsed.cursor).toBe('')
  })

  it('parses profile html', () => {
    const parsed = parseNitterProfileHTML(PROFILE_HTML, 'https://nitter.test/example', 'example')
    expect(parsed).not.toBeNull()
    expect(parsed?.username).toBe('example')
    expect(parsed?.name).toBe('Example Name')
    expect(parsed?.followersCount).toBe(20)
    expect(parsed?.followingCount).toBe(10)
    expect(parsed?.tweetsCount).toBe(123)
    expect(parsed?.website).toBe('https://example.com/')
    expect(parsed?.avatar).toBe('https://nitter.test/pic/avatar.jpg')
    expect(parsed?.banner).toBe('https://nitter.test/pic/banner.jpg')
    expect(parsed?.biography).toBe('Building tools')
  })

  it('returns null profile when no data is present', () => {
    const parsed = parseNitterProfileHTML('<html><body></body></html>', 'https://nitter.test/empty')
    expect(parsed).toBeNull()
  })

  it('parses timeline html with cursor', () => {
    const parsed = parseNitterTimelineHTML(TIMELINE_HTML, 'https://nitter.test/example')
    expect(parsed.tweets.length).toBe(2)
    expect(parsed.tweets[0].id).toBe('111')
    expect(parsed.tweets[1].id).toBe('222')
    expect(parsed.cursor.startsWith('nitter:')).toBe(true)
    expect(decodeNitterCursor(parsed.cursor)).toBe('/example?cursor=ABC123')
    expect(parsed.tweets[0].likes).toBe(3)
  })

  it('parses user search html', () => {
    const parsed = parseNitterSearchUsersHTML(USERS_HTML, 'https://nitter.test/search?f=users&q=golang')
    expect(parsed.users.length).toBe(2)
    expect(parsed.users[0].username).toBe('alice')
    expect(parsed.users[1].username).toBe('bob')
    expect(decodeNitterCursor(parsed.cursor)).toBe('/search?f=users&q=golang&cursor=USERS2')
  })

  it('handles nitter cursor helpers', () => {
    const encoded = encodeNitterCursor('/search?f=tweets&q=go&cursor=XYZ')
    expect(isNitterCursor(encoded)).toBe(true)
    expect(decodeNitterCursor(encoded)).toBe('/search?f=tweets&q=go&cursor=XYZ')
    expect(isNitterCursor('cursor123')).toBe(false)
    expect(decodeNitterCursor('cursor123')).toBe('')
  })
})

describe('nitter fetchers', () => {
  it('fetchTweetFromNitter retries instances and succeeds on second', async () => {
    const calls: string[] = []
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      calls.push(url)
      if (url.startsWith('https://n1.test/')) return new Response('rate limited', { status: 429 })
      if (url.startsWith('https://n2.test/') && url.includes('/i/status/1234567890')) return new Response(TWEET_HTML, { status: 200 })
      return new Response('not found', { status: 404 })
    })

    const tweet = await fetchTweetFromNitter('1234567890', 'https://n1.test,https://n2.test')
    expect(tweet.mainTweet?.id).toBe('1234567890')
    expect(calls.length).toBe(2)
    expect(calls[0]).toContain('https://n1.test/i/status/1234567890')
    expect(calls[1]).toContain('https://n2.test/i/status/1234567890')
  })

  it('fetchProfileFromNitter returns parsed profile', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/example')) return new Response(PROFILE_HTML, { status: 200 })
      return new Response('not found', { status: 404 })
    })
    const profile = await fetchProfileFromNitter('example', 'https://nitter.test')
    expect(profile?.username).toBe('example')
    expect(profile?.name).toBe('Example Name')
  })

  it('fetchUserTweetsFromNitter supports tab paths and encoded cursor', async () => {
    const calls: string[] = []
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      calls.push(url)
      if (url.includes('/example?cursor=ABC123')) return new Response(TIMELINE_HTML, { status: 200 })
      if (url.includes('/example/with_replies')) return new Response(TIMELINE_HTML, { status: 200 })
      if (url.includes('/example/media')) return new Response(TIMELINE_HTML, { status: 200 })
      return new Response('not found', { status: 404 })
    })

    const withCursor = await fetchUserTweetsFromNitter(
      'example',
      'tweets',
      encodeNitterCursor('/example?cursor=ABC123'),
      'https://nitter.test'
    )
    const replies = await fetchUserTweetsFromNitter('example', 'replies', '', 'https://nitter.test')
    const media = await fetchUserTweetsFromNitter('example', 'media', '', 'https://nitter.test')

    expect(withCursor.tweets.length).toBe(2)
    expect(replies.tweets.length).toBe(2)
    expect(media.tweets.length).toBe(2)
    expect(calls.some((u) => u.includes('/example/with_replies'))).toBe(true)
    expect(calls.some((u) => u.includes('/example/media'))).toBe(true)
    expect(calls.some((u) => u.includes('/example?cursor=ABC123'))).toBe(true)
  })

  it('fetchSearchTweetsFromNitter supports tweets/media modes', async () => {
    const calls: string[] = []
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      calls.push(url)
      if (url.includes('/search?f=tweets&q=golang')) return new Response(TIMELINE_HTML, { status: 200 })
      if (url.includes('/search?f=media&q=golang')) return new Response(TIMELINE_HTML, { status: 200 })
      return new Response('not found', { status: 404 })
    })

    const top = await fetchSearchTweetsFromNitter('golang', 'Top', '', 'https://nitter.test')
    const media = await fetchSearchTweetsFromNitter('golang', 'Media', '', 'https://nitter.test')
    expect(top.tweets.length).toBe(2)
    expect(media.tweets.length).toBe(2)
    expect(calls.some((u) => u.includes('/search?f=tweets&q=golang'))).toBe(true)
    expect(calls.some((u) => u.includes('/search?f=media&q=golang'))).toBe(true)
  })

  it('fetchSearchUsersFromNitter parses users list', async () => {
    vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
      const url = String(input)
      if (url.includes('/search?f=users&q=golang')) return new Response(USERS_HTML, { status: 200 })
      return new Response('not found', { status: 404 })
    })

    const users = await fetchSearchUsersFromNitter('golang', '', 'https://nitter.test')
    expect(users.users.length).toBe(2)
    expect(users.users[0].username).toBe('alice')
    expect(users.users[1].username).toBe('bob')
  })

  it('throws when no nitter instance succeeds', async () => {
    vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('not found', { status: 404 }))
    await expect(fetchTweetFromNitter('1234567890', 'https://nitter.test')).rejects.toThrow()
  })
})
