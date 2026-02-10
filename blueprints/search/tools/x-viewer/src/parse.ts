import type { Profile, Tweet, XList, TimelineResult } from './types'

// JSON navigation helpers

function dig(m: Record<string, unknown>, ...keys: string[]): unknown {
  let cur: unknown = m
  for (const k of keys) {
    if (cur && typeof cur === 'object' && !Array.isArray(cur)) {
      cur = (cur as Record<string, unknown>)[k]
    } else {
      return undefined
    }
  }
  return cur
}

function asMap(v: unknown): Record<string, unknown> | undefined {
  if (v && typeof v === 'object' && !Array.isArray(v)) {
    return v as Record<string, unknown>
  }
  return undefined
}

function asSlice(v: unknown): unknown[] | undefined {
  if (Array.isArray(v)) return v
  return undefined
}

function asStr(v: unknown): string {
  if (typeof v === 'string') return v
  return ''
}

function asInt(v: unknown): number {
  if (typeof v === 'number') return Math.floor(v)
  if (typeof v === 'string') return parseInt(v, 10) || 0
  return 0
}

function asBool(v: unknown): boolean {
  if (typeof v === 'boolean') return v
  return false
}

// Twitter time: "Mon Jan 02 15:04:05 +0000 2006"
function parseTwitterTime(s: string): string {
  if (!s) return ''
  const d = new Date(s)
  return isNaN(d.getTime()) ? '' : d.toISOString()
}

function parseCreatedAt(legacy: Record<string, unknown>): string {
  const s = asStr(legacy['created_at'])
  if (s) return parseTwitterTime(s)
  const ms = legacy['created_at_ms']
  if (typeof ms === 'number') return new Date(ms).toISOString()
  if (typeof ms === 'string') {
    const n = parseInt(ms, 10)
    if (!isNaN(n)) return new Date(n).toISOString()
  }
  return ''
}

// User parsing

function parseGraphUserFromCore(core: Record<string, unknown> | undefined): { username: string; userID: string; name: string; avatar: string; isBlueVerified: boolean; verifiedType: string } {
  const empty = { username: '', userID: '', name: '', avatar: '', isBlueVerified: false, verifiedType: '' }
  if (!core) return empty

  let user = asMap(dig(core, 'user_results', 'result'))
  if (!user) user = asMap(dig(core, 'user_result', 'result'))
  if (!user) {
    return {
      username: asStr(core['screen_name']),
      userID: '',
      name: asStr(core['name']),
      avatar: '',
      isBlueVerified: false,
      verifiedType: '',
    }
  }

  const userID = asStr(user['rest_id'])
  const isBlueVerified = asBool(user['is_blue_verified'])
  const legacy = asMap(user['legacy'])
  const verifiedType = legacy ? asStr(legacy['verified_type']) : ''

  let username = ''
  let name = ''
  let avatar = ''
  if (legacy) {
    username = asStr(legacy['screen_name'])
    name = asStr(legacy['name'])
    const pic = asStr(legacy['profile_image_url_https'])
    avatar = pic.replace('_normal', '')
  }
  if (!username) username = asStr(dig(user, 'core', 'screen_name') as unknown as string) || asStr(user['screen_name'] as unknown as string)
  if (!name) name = asStr(dig(user, 'core', 'name') as unknown as string) || asStr(user['name'] as unknown as string)
  if (!avatar) {
    const pic = asStr(dig(user, 'avatar', 'image_url') as unknown as string)
    avatar = pic.replace('_normal', '')
  }

  return { username, userID, name, avatar, isBlueVerified, verifiedType }
}

export function parseUserResult(data: Record<string, unknown>): Profile | null {
  let node = asMap(dig(data, 'data', 'user', 'result'))
  if (!node) node = asMap(dig(data, 'data', 'user_result', 'result'))
  if (!node) return null
  return parseGraphUser(node)
}

function parseGraphUser(node: Record<string, unknown>): Profile | null {
  let user = asMap(dig(node, 'user_result', 'result'))
  if (!user) user = asMap(dig(node, 'user_results', 'result'))
  if (!user) {
    if (asStr(node['rest_id']) || (asMap(node['core']) && asMap(node['legacy']))) {
      user = node
    }
  }
  if (!user) return null

  const legacy = asMap(user['legacy'])
  if (!legacy) {
    // Newer format
    const p: Profile = {
      id: asStr(user['rest_id']),
      username: asStr(dig(user, 'core', 'screen_name') as unknown as string),
      name: asStr(dig(user, 'core', 'name') as unknown as string),
      biography: '',
      avatar: (asStr(dig(user, 'avatar', 'image_url') as unknown as string) || '').replace('_normal', ''),
      banner: '',
      location: '',
      website: '',
      url: '',
      joined: '',
      birthday: '',
      followersCount: 0,
      followingCount: 0,
      tweetsCount: 0,
      likesCount: 0,
      mediaCount: 0,
      listedCount: 0,
      isPrivate: false,
      isVerified: false,
      isBlueVerified: asBool(user['is_blue_verified']),
      verifiedType: '',
      pinnedTweetIDs: [],
      professionalType: '',
      professionalCategory: '',
    }
    return p
  }

  const restID = asStr(user['rest_id'])
  let username = asStr(legacy['screen_name'])
  if (!username) username = asStr(dig(user, 'core', 'screen_name') as unknown as string)
  let name = asStr(legacy['name'])
  if (!name) name = asStr(dig(user, 'core', 'name') as unknown as string)

  let joined = parseTwitterTime(asStr(legacy['created_at']))
  if (!joined) joined = parseTwitterTime(asStr(dig(user, 'core', 'created_at') as unknown as string))

  let biography = asStr(legacy['description'])
  if (!biography) biography = asStr(dig(user, 'profile_bio', 'description') as unknown as string)

  let location = asStr(legacy['location'])
  if (!location) location = asStr(dig(user, 'location', 'location') as unknown as string)

  let isPrivate = asBool(legacy['protected'])
  if (!isPrivate) isPrivate = asBool(dig(user, 'privacy', 'protected'))

  let pic = asStr(legacy['profile_image_url_https'])
  if (!pic) pic = asStr(dig(user, 'avatar', 'image_url') as unknown as string)
  const avatar = pic.replace('_normal', '')

  const banner = asStr(legacy['profile_banner_url'])

  let website = ''
  const urlEntities = asSlice(dig(legacy, 'entities', 'url', 'urls'))
  if (urlEntities && urlEntities.length > 0) {
    const first = asMap(urlEntities[0])
    if (first) website = asStr(first['expanded_url'])
  }

  let isBlueVerified = asBool(user['is_blue_verified'])
  let isVerified = asBool(dig(user, 'verification', 'verified'))
  if (asStr(legacy['verified_type']) || asBool(legacy['verified'])) isVerified = true
  const verifiedType = asStr(legacy['verified_type'])

  const pinnedTweetIDs: string[] = []
  const pins = asSlice(legacy['pinned_tweet_ids_str'])
  if (pins) {
    for (const pin of pins) {
      const id = asStr(pin)
      if (id) pinnedTweetIDs.push(id)
    }
  }

  let professionalType = ''
  let professionalCategory = ''
  const prof = asMap(user['professional'])
  if (prof) {
    professionalType = asStr(prof['professional_type'])
    const cats = asSlice(prof['category'])
    if (cats && cats.length > 0) {
      const cat = asMap(cats[0])
      if (cat) professionalCategory = asStr(cat['name'])
    }
  }

  return {
    id: restID,
    username,
    name,
    biography,
    avatar,
    banner,
    location,
    website,
    url: asStr(legacy['url']),
    joined,
    birthday: '',
    followersCount: asInt(legacy['followers_count']),
    followingCount: asInt(legacy['friends_count']),
    tweetsCount: asInt(legacy['statuses_count']),
    likesCount: asInt(legacy['favourites_count']),
    mediaCount: asInt(legacy['media_count']),
    listedCount: asInt(legacy['listed_count']),
    isPrivate,
    isVerified,
    isBlueVerified,
    verifiedType,
    pinnedTweetIDs,
    professionalType,
    professionalCategory,
  }
}

// Tweet parsing

function extractSourceName(src: string): string {
  const i = src.indexOf('>')
  if (i >= 0) {
    const rest = src.slice(i + 1)
    const j = rest.indexOf('<')
    if (j >= 0) return rest.slice(0, j)
    return rest
  }
  return src
}

function bestVideoVariant(variants: unknown[]): string {
  let bestURL = ''
  let bestBitrate = 0
  for (const v of variants) {
    const vm = asMap(v)
    if (!vm) continue
    if (asStr(vm['content_type']) !== 'video/mp4') continue
    const bitrate = asInt(vm['bitrate'])
    if (bitrate > bestBitrate || !bestURL) {
      bestBitrate = bitrate
      bestURL = asStr(vm['url'])
    }
  }
  return bestURL
}

function parseMediaFromLegacy(legacy: Record<string, unknown>, t: Tweet): void {
  const media = asSlice(dig(legacy, 'extended_entities', 'media'))
  if (!media) return
  for (const m of media) {
    const mm = asMap(m)
    if (!mm) continue
    const mediaType = asStr(mm['type'])
    switch (mediaType) {
      case 'photo': {
        const url = asStr(mm['media_url_https'])
        if (url) t.photos.push(url)
        break
      }
      case 'video': {
        const variants = asSlice(dig(mm, 'video_info', 'variants'))
        if (variants) {
          const best = bestVideoVariant(variants)
          if (best) {
            t.videos.push(best)
            const thumb = asStr(mm['media_url_https'])
            t.videoThumbnails.push(thumb || '')
          }
        }
        break
      }
      case 'animated_gif': {
        const variants = asSlice(dig(mm, 'video_info', 'variants'))
        if (variants && variants.length > 0) {
          const first = asMap(variants[0])
          if (first) {
            const url = asStr(first['url'])
            if (url) t.gifs.push(url)
          }
        }
        break
      }
    }
  }
}

function parseMediaFromEntities(node: Record<string, unknown>, t: Tweet): void {
  const mediaEntities = asSlice(node['media_entities'])
  if (!mediaEntities) return
  for (const me of mediaEntities) {
    const mem = asMap(me)
    if (!mem) continue
    const mediaInfo = asMap(dig(mem, 'media_results', 'result', 'media_info'))
    if (!mediaInfo) continue
    const typeName = asStr(mediaInfo['__typename'])
    switch (typeName) {
      case 'ApiImage': {
        const url = asStr(mediaInfo['original_img_url'])
        if (url) t.photos.push(url)
        break
      }
      case 'ApiVideo': {
        const variants = asSlice(mediaInfo['variants'])
        if (variants) {
          const best = bestVideoVariant(variants)
          if (best) {
            t.videos.push(best)
            const thumb = asStr(mediaInfo['original_img_url']) || asStr(dig(mem, 'media_url_https') as unknown as string)
            t.videoThumbnails.push(thumb || '')
          }
        }
        break
      }
      case 'ApiGif': {
        const variants = asSlice(mediaInfo['variants'])
        if (variants && variants.length > 0) {
          const first = asMap(variants[0])
          if (first) {
            const url = asStr(first['url'])
            if (url) t.gifs.push(url)
          }
        }
        break
      }
    }
  }
}

export function parseGraphTweet(node: Record<string, unknown>): Tweet | null {
  if (!node) return null

  const typeName = asStr(node['__typename'])
  switch (typeName) {
    case 'TweetUnavailable':
    case 'TweetTombstone':
    case 'TweetPreviewDisplay':
      return null
    case 'TweetWithVisibilityResults': {
      const inner = asMap(node['tweet'])
      return inner ? parseGraphTweet(inner) : null
    }
  }

  const legacy = asMap(node['legacy'])
  if (!legacy) return null

  const restID = asStr(node['rest_id'])
  const { username, userID, name, avatar, isBlueVerified, verifiedType } = parseGraphUserFromCore(asMap(node['core']))

  const t: Tweet = {
    id: restID,
    conversationID: asStr(legacy['conversation_id_str']),
    text: asStr(legacy['full_text']),
    username,
    userID,
    name,
    avatar,
    permanentURL: username ? `https://x.com/${username}/status/${restID}` : '',
    isRetweet: false,
    isReply: asStr(legacy['in_reply_to_status_id_str']) !== '',
    isQuote: false,
    isPin: asBool(legacy['is_pinned']),
    replyToID: asStr(legacy['in_reply_to_status_id_str']),
    replyToUser: asStr(legacy['in_reply_to_screen_name']),
    quotedID: '',
    retweetedID: '',
    likes: asInt(legacy['favorite_count']),
    retweets: asInt(legacy['retweet_count']),
    replies: asInt(legacy['reply_count']),
    views: 0,
    bookmarks: asInt(legacy['bookmark_count']),
    quotes: asInt(legacy['quote_count']),
    photos: [],
    videos: [],
    videoThumbnails: [],
    gifs: [],
    hashtags: [],
    mentions: [],
    urls: [],
    sensitive: asBool(legacy['possibly_sensitive']),
    language: asStr(legacy['lang']),
    source: '',
    place: '',
    isEdited: false,
    isBlueVerified,
    verifiedType,
    postedAt: parseCreatedAt(legacy),
  }

  // Views
  const viewCount = asStr(dig(node, 'views', 'count') as unknown as string)
  if (viewCount) t.views = parseInt(viewCount, 10) || 0

  // Source
  const src = asStr(node['source']) || asStr(legacy['source'])
  if (src) t.source = extractSourceName(src)

  // Place
  const placeNode = asMap(legacy['place'])
  if (placeNode) t.place = asStr(placeNode['full_name'])

  // Edit info
  const editCtrl = asMap(node['edit_control'])
  if (editCtrl) {
    const edits = asSlice(editCtrl['edit_tweet_ids'])
    if (edits && edits.length > 1) t.isEdited = true
  }

  // Retweet
  let rt = asMap(dig(legacy, 'retweeted_status_result', 'result'))
  if (!rt) rt = asMap(dig(node, 'retweeted_status_result', 'result'))
  if (rt) {
    t.isRetweet = true
    t.retweetedID = asStr(rt['rest_id'])
    // Parse inner retweet
    const inner = parseGraphTweet(rt)
    if (inner) t.retweetedTweet = inner
  }

  // Quote
  if (asBool(legacy['is_quote_status'])) {
    t.isQuote = true
    t.quotedID = asStr(legacy['quoted_status_id_str'])
    // Parse quoted tweet
    let qt = asMap(dig(node, 'quoted_status_result', 'result'))
    if (!qt) qt = asMap(dig(legacy, 'quoted_status_result', 'result'))
    if (qt) {
      const inner = parseGraphTweet(qt)
      if (inner) t.quotedTweet = inner
    }
  }

  // Media
  parseMediaFromLegacy(legacy, t)
  parseMediaFromEntities(node, t)

  // Entities
  const entities = asMap(legacy['entities'])
  if (entities) {
    const hashtags = asSlice(entities['hashtags'])
    if (hashtags) {
      for (const h of hashtags) {
        const hm = asMap(h)
        if (hm) {
          const tag = asStr(hm['text'])
          if (tag) t.hashtags.push(tag)
        }
      }
    }
    const userMentions = asSlice(entities['user_mentions'])
    if (userMentions) {
      for (const m of userMentions) {
        const mm = asMap(m)
        if (mm) {
          const un = asStr(mm['screen_name'])
          if (un) t.mentions.push('@' + un)
        }
      }
    }
    const urlEntities = asSlice(entities['urls'])
    if (urlEntities) {
      for (const u of urlEntities) {
        const um = asMap(u)
        if (um) {
          let expanded = asStr(um['expanded_url'])
          if (!expanded) expanded = asStr(um['url'])
          if (expanded) t.urls.push(expanded)
        }
      }
    }
  }

  // Note tweet (long tweets)
  const noteTweet = asMap(dig(node, 'note_tweet', 'note_tweet_results', 'result'))
  if (noteTweet) {
    const noteText = asStr(noteTweet['text'])
    if (noteText) t.text = noteText
  }

  return t
}

// Timeline parsing

function getEntryID(entry: Record<string, unknown>): string {
  return asStr(entry['entryId']) || asStr(entry['entry_id'])
}

function getTweetResult(entry: Record<string, unknown>): Record<string, unknown> | undefined {
  let r = asMap(dig(entry, 'content', 'itemContent', 'tweet_results', 'result'))
  if (r) return r
  r = asMap(dig(entry, 'content', 'content', 'tweetResult', 'result'))
  if (r) return r
  r = asMap(dig(entry, 'content', 'content', 'tweet_results', 'result'))
  if (r) return r
  return undefined
}

function extractTweetFromItem(item: Record<string, unknown> | undefined, prefix: string): Tweet | null {
  if (!item) return null
  let r = asMap(dig(item, prefix, 'content', 'tweet_results', 'result'))
  if (r) return parseGraphTweet(r)
  r = asMap(dig(item, prefix, 'itemContent', 'tweet_results', 'result'))
  if (r) return parseGraphTweet(r)
  return null
}

function extractTweetsFromEntry(entry: Record<string, unknown>): Tweet[] {
  const tweets: Tweet[] = []
  const r = getTweetResult(entry)
  if (r) {
    const t = parseGraphTweet(r)
    if (t) tweets.push(t)
    return tweets
  }
  const items = asSlice(dig(entry, 'content', 'items'))
  if (items) {
    for (const item of items) {
      const t = extractTweetFromItem(asMap(item), 'item')
      if (t) tweets.push(t)
    }
  }
  return tweets
}

function findInstructions(data: Record<string, unknown>): unknown[] | undefined {
  const paths = [
    ['data', 'user', 'result', 'timeline', 'timeline', 'instructions'],
    ['data', 'user_result', 'result', 'timeline_response', 'timeline', 'instructions'],
    ['data', 'list', 'timeline_response', 'timeline', 'instructions'],
    ['data', 'timeline_response', 'timeline', 'instructions'],
    ['data', 'home', 'home_timeline_urt', 'instructions'],
    ['data', 'bookmark_timeline_v2', 'timeline', 'instructions'],
    ['data', 'bookmark_search_timeline', 'timeline', 'instructions'],
  ]
  for (const path of paths) {
    const v = asSlice(dig(data, ...path))
    if (v) return v
  }
  return undefined
}

function findSearchInstructions(data: Record<string, unknown>): unknown[] | undefined {
  const paths = [
    ['data', 'search_by_raw_query', 'search_timeline', 'timeline', 'instructions'],
    ['data', 'search', 'timeline_response', 'timeline', 'instructions'],
  ]
  for (const path of paths) {
    const v = asSlice(dig(data, ...path))
    if (v) return v
  }
  return undefined
}

function findConversationInstructions(data: Record<string, unknown>): unknown[] | undefined {
  const paths = [
    ['data', 'threaded_conversation_with_injections_v2', 'instructions'],
    ['data', 'timelineResponse', 'instructions'],
    ['data', 'timeline_response', 'instructions'],
  ]
  for (const path of paths) {
    const v = asSlice(dig(data, ...path))
    if (v) return v
  }
  return undefined
}

function findListInstructions(data: Record<string, unknown>): unknown[] | undefined {
  const paths = [
    ['data', 'list', 'tweets_timeline', 'timeline', 'instructions'],
    ['data', 'list', 'timeline_response', 'timeline', 'instructions'],
  ]
  for (const path of paths) {
    const v = asSlice(dig(data, ...path))
    if (v) return v
  }
  return undefined
}

export function parseTimeline(data: Record<string, unknown>): TimelineResult {
  const result: TimelineResult = { tweets: [], cursor: '' }
  const instructions = findInstructions(data)
  if (!instructions) return result

  for (const inst of instructions) {
    const im = asMap(inst)
    if (!im) continue

    const moduleItems = asSlice(im['moduleItems'])
    if (moduleItems) {
      for (const item of moduleItems) {
        const t = extractTweetFromItem(asMap(item), 'item')
        if (t) result.tweets.push(t)
      }
      continue
    }

    const entries = asSlice(im['entries'])
    if (!entries) continue

    for (const e of entries) {
      const em = asMap(e)
      if (!em) continue
      const entryID = getEntryID(em)

      if (entryID.startsWith('tweet') || entryID.startsWith('profile-grid')) {
        result.tweets.push(...extractTweetsFromEntry(em))
      } else if (entryID.includes('-conversation-') || entryID.startsWith('homeConversation')) {
        const items = asSlice(dig(em, 'content', 'items'))
        if (items) {
          for (const item of items) {
            const t = extractTweetFromItem(asMap(item), 'item')
            if (t) result.tweets.push(t)
          }
        }
      } else if (entryID.startsWith('cursor-bottom')) {
        result.cursor = asStr(dig(em, 'content', 'value') as unknown as string)
      }
    }
  }

  return result
}

export function parseSearchTweets(data: Record<string, unknown>): TimelineResult {
  const result: TimelineResult = { tweets: [], cursor: '' }
  const instructions = findSearchInstructions(data)
  if (!instructions) return result

  for (const inst of instructions) {
    const im = asMap(inst)
    if (!im) continue
    let typeName = asStr(im['type'])
    if (!typeName) typeName = asStr(im['__typename'])

    if (typeName === 'TimelineAddEntries') {
      const entries = asSlice(im['entries'])
      if (!entries) continue
      for (const e of entries) {
        const em = asMap(e)
        if (!em) continue
        const entryID = getEntryID(em)
        if (entryID.startsWith('tweet')) {
          const tweet = getTweetResult(em)
          if (tweet) {
            const t = parseGraphTweet(tweet)
            if (t) result.tweets.push(t)
          }
        } else if (entryID.startsWith('cursor-bottom')) {
          result.cursor = asStr(dig(em, 'content', 'value') as unknown as string)
        }
      }
    } else if (typeName === 'TimelineReplaceEntry') {
      const entryToReplace = asStr(im['entry_id_to_replace'])
      if (entryToReplace.startsWith('cursor-bottom')) {
        result.cursor = asStr(dig(im, 'entry', 'content', 'value') as unknown as string)
      }
    }
  }

  return result
}

export function parseConversation(data: Record<string, unknown>, tweetID: string): { mainTweet: Tweet | null; replies: Tweet[]; cursor: string } {
  let mainTweet: Tweet | null = null
  const replies: Tweet[] = []
  let cursor = ''

  const instructions = findConversationInstructions(data)
  if (!instructions) return { mainTweet: null, replies: [], cursor: '' }

  for (const inst of instructions) {
    const im = asMap(inst)
    if (!im) continue
    let typeName = asStr(im['type'])
    if (!typeName) typeName = asStr(im['__typename'])
    if (typeName === 'TimelineAddToModule') {
      const moduleItems = asSlice(im['moduleItems'])
      if (moduleItems) {
        for (const item of moduleItems) {
          const itemMap = asMap(item)
          if (!itemMap) continue
          const itemEntryID = getEntryID(itemMap)
          if (itemEntryID.includes('cursor')) {
            let val = asStr(dig(itemMap, 'item', 'itemContent', 'value') as unknown as string)
            if (!val) val = asStr(dig(itemMap, 'item', 'content', 'value') as unknown as string)
            if (val) cursor = val
          } else {
            const tr = extractTweetFromItem(itemMap, 'item')
            if (tr) replies.push(tr)
          }
        }
      }
      continue
    }

    if (typeName !== 'TimelineAddEntries') continue

    const entries = asSlice(im['entries'])
    if (!entries) continue
    for (const e of entries) {
      const em = asMap(e)
      if (!em) continue
      const entryID = getEntryID(em)

      if (entryID.startsWith('tweet')) {
        const tweetResult = getTweetResult(em)
        if (tweetResult) {
          const t = parseGraphTweet(tweetResult)
          if (t) {
            if (t.id === tweetID) {
              mainTweet = t
            } else {
              replies.push(t)
            }
          }
        }
      } else if (entryID.startsWith('conversationthread')) {
        const items = asSlice(dig(em, 'content', 'items'))
        if (items) {
          for (const item of items) {
            const itemMap = asMap(item)
            if (!itemMap) continue
            const itemEntryID = getEntryID(itemMap)
            if (itemEntryID.includes('cursor-showmore')) {
              let val = asStr(dig(itemMap, 'item', 'content', 'value') as unknown as string)
              if (!val) val = asStr(dig(itemMap, 'item', 'itemContent', 'value') as unknown as string)
              if (!cursor) cursor = val
            } else if (itemEntryID.includes('tweet')) {
              const tr = extractTweetFromItem(itemMap, 'item')
              if (tr) replies.push(tr)
            }
          }
        }
      } else if (entryID.startsWith('cursor-bottom')) {
        let val = asStr(dig(em, 'content', 'value') as unknown as string)
        if (!val) val = asStr(dig(em, 'content', 'content', 'value') as unknown as string)
        if (!cursor) cursor = val
      }
    }
  }

  return { mainTweet, replies, cursor }
}

export function parseListTimeline(data: Record<string, unknown>): TimelineResult {
  const result: TimelineResult = { tweets: [], cursor: '' }

  let instructions = findListInstructions(data)
  if (!instructions) instructions = findInstructions(data)
  if (!instructions) return result

  for (const inst of instructions) {
    const im = asMap(inst)
    if (!im) continue
    const entries = asSlice(im['entries'])
    if (!entries) continue

    for (const e of entries) {
      const em = asMap(e)
      if (!em) continue
      const entryID = getEntryID(em)

      if (entryID.startsWith('tweet') || entryID.startsWith('list-')) {
        result.tweets.push(...extractTweetsFromEntry(em))
      } else if (entryID.includes('-conversation-')) {
        const items = asSlice(dig(em, 'content', 'items'))
        if (items) {
          for (const item of items) {
            const t = extractTweetFromItem(asMap(item), 'item')
            if (t) result.tweets.push(t)
          }
        }
      } else if (entryID.startsWith('cursor-bottom')) {
        result.cursor = asStr(dig(em, 'content', 'value') as unknown as string)
      }
    }
  }

  return result
}

export function parseGraphList(data: Record<string, unknown>): XList | null {
  let node = asMap(dig(data, 'data', 'list'))
  if (!node) node = asMap(dig(data, 'data', 'list_by_rest_id', 'result'))
  if (!node) node = asMap(dig(data, 'data', 'list_by_slug', 'result'))
  if (!node) return null

  const l: XList = {
    id: asStr(node['id_str']) || asStr(node['rest_id']),
    name: asStr(node['name']),
    description: asStr(node['description']),
    banner: '',
    memberCount: asInt(node['member_count']),
    ownerID: '',
    ownerName: '',
  }

  const banner = asMap(node['custom_banner_media'])
  if (banner) {
    const mediaInfo = asMap(banner['media_info'])
    if (mediaInfo) l.banner = asStr(mediaInfo['original_img_url'])
  }
  if (!l.banner) l.banner = asStr(node['default_banner_media_url'])

  const userResults = asMap(dig(node, 'user_results', 'result'))
  if (userResults) {
    l.ownerID = asStr(userResults['rest_id'])
    const legacy = asMap(userResults['legacy'])
    if (legacy) l.ownerName = asStr(legacy['screen_name'])
  }

  return l
}

// Follow list parsing

export function parseFollowList(data: Record<string, unknown>): { users: Profile[]; cursor: string } {
  const result: { users: Profile[]; cursor: string } = { users: [], cursor: '' }
  const instructions = findInstructions(data)
  if (!instructions) return result

  for (const inst of instructions) {
    const im = asMap(inst)
    if (!im) continue
    const entries = asSlice(im['entries'])
    if (!entries) continue

    for (const e of entries) {
      const em = asMap(e)
      if (!em) continue
      const entryID = getEntryID(em)

      if (entryID.startsWith('user')) {
        const itemContent = asMap(dig(em, 'content', 'itemContent'))
        if (itemContent) {
          const user = parseGraphUser(itemContent)
          if (user) result.users.push(user)
        }
      } else if (entryID.startsWith('cursor-bottom')) {
        result.cursor = asStr(dig(em, 'content', 'value') as unknown as string)
      }
    }
  }

  return result
}

// Search users parsing

export function parseSearchUsers(data: Record<string, unknown>): { users: Profile[]; cursor: string } {
  const result: { users: Profile[]; cursor: string } = { users: [], cursor: '' }
  const instructions = findSearchInstructions(data)
  if (!instructions) return result

  for (const inst of instructions) {
    const im = asMap(inst)
    if (!im) continue
    const entries = asSlice(im['entries'])
    if (!entries) continue

    for (const e of entries) {
      const em = asMap(e)
      if (!em) continue
      const entryID = getEntryID(em)

      if (entryID.startsWith('user')) {
        const itemContent = asMap(dig(em, 'content', 'itemContent'))
        if (itemContent) {
          const user = parseGraphUser(itemContent)
          if (user) result.users.push(user)
        }
      } else if (entryID.startsWith('cursor-bottom')) {
        result.cursor = asStr(dig(em, 'content', 'value') as unknown as string)
      }
    }
  }

  return result
}

// List members parsing

export function parseListMembers(data: Record<string, unknown>): { users: Profile[]; cursor: string } {
  const result: { users: Profile[]; cursor: string } = { users: [], cursor: '' }
  let instructions = asSlice(dig(data, 'data', 'list', 'members_timeline', 'timeline', 'instructions'))
  if (!instructions) instructions = findInstructions(data)
  if (!instructions) return result

  for (const inst of instructions) {
    const im = asMap(inst)
    if (!im) continue
    const entries = asSlice(im['entries'])
    if (!entries) continue

    for (const e of entries) {
      const em = asMap(e)
      if (!em) continue
      const entryID = getEntryID(em)

      if (entryID.startsWith('user') || entryID.startsWith('list-user')) {
        const itemContent = asMap(dig(em, 'content', 'itemContent'))
        if (itemContent) {
          const user = parseGraphUser(itemContent)
          if (user) result.users.push(user)
        }
      } else if (entryID.startsWith('cursor-bottom')) {
        result.cursor = asStr(dig(em, 'content', 'value') as unknown as string)
      }
    }
  }

  return result
}

// Trends parsing

export function parseTrends(data: Record<string, unknown>): string[] {
  const trends: string[] = []
  findTrends(data, trends)
  return trends
}

function findTrends(node: unknown, trends: string[]): void {
  if (node && typeof node === 'object' && !Array.isArray(node)) {
    const m = node as Record<string, unknown>
    const trend = asMap(m['trend'])
    if (trend) {
      const name = asStr(trend['name'])
      if (name) {
        trends.push(name)
        return
      }
    }
    if (asStr(m['__typename']) === 'TimelineTrend') {
      const name = asStr(m['name'])
      if (name) {
        trends.push(name)
        return
      }
    }
    for (const val of Object.values(m)) {
      findTrends(val, trends)
    }
  } else if (Array.isArray(node)) {
    for (const item of node) {
      findTrends(item, trends)
    }
  }
}
