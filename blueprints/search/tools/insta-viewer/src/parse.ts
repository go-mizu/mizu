import type {
  Profile, Post, Comment, StoryItem, Highlight, Reel, FollowUser,
  SearchResult, SearchUser, SearchHashtag, SearchPlace,
  ProfileWithPosts, PostsResult, CommentsResult, FollowResult, ReelsResult,
} from './types'

// ── Helpers ──

function dig(obj: unknown, ...keys: string[]): unknown {
  let cur = obj
  for (const k of keys) {
    if (cur == null || typeof cur !== 'object') return undefined
    cur = (cur as Record<string, unknown>)[k]
  }
  return cur
}

function asMap(v: unknown): Record<string, unknown> {
  if (v != null && typeof v === 'object' && !Array.isArray(v)) return v as Record<string, unknown>
  return {}
}

function asArr(v: unknown): unknown[] {
  return Array.isArray(v) ? v : []
}

function asStr(v: unknown): string {
  if (typeof v === 'string') return v
  if (typeof v === 'number') return v.toString()
  return ''
}

function asNum(v: unknown): number {
  if (typeof v === 'number') return v
  if (typeof v === 'string') return parseInt(v, 10) || 0
  return 0
}

function asBool(v: unknown): boolean {
  return v === true
}

function tsToISO(ts: unknown): string {
  const n = asNum(ts)
  if (n === 0) return ''
  return new Date(n * 1000).toISOString()
}

// ── Profile parsing ──

export function parseProfileResponse(data: Record<string, unknown>): ProfileWithPosts | null {
  const userData = dig(data, 'data', 'user') as Record<string, unknown> | undefined
  if (!userData) return null

  const profile = parseProfileUser(userData)
  if (!profile) return null

  const timeline = asMap(userData.edge_owner_to_timeline_media)
  const posts = parseMediaEdges(asArr(dig(timeline, 'edges')), profile.username, profile.profilePicUrl)
  const pageInfo = asMap(dig(timeline, 'page_info'))
  const cursor = asStr(pageInfo.end_cursor)
  const hasMore = asBool(pageInfo.has_next_page)

  return { profile, posts, cursor, hasMore }
}

function parseProfileUser(u: Record<string, unknown>): Profile | null {
  const id = asStr(u.id) || asStr(u.pk)
  const username = asStr(u.username)
  if (!username) return null

  return {
    id,
    username,
    fullName: asStr(u.full_name),
    biography: asStr(u.biography),
    profilePicUrl: asStr(u.profile_pic_url_hd) || asStr(u.profile_pic_url),
    externalUrl: asStr(u.external_url),
    isPrivate: asBool(u.is_private),
    isVerified: asBool(u.is_verified),
    isBusiness: asBool(u.is_business_account),
    categoryName: asStr(u.category_name),
    followerCount: asNum(dig(u, 'edge_followed_by', 'count')),
    followingCount: asNum(dig(u, 'edge_follow', 'count')),
    postCount: asNum(dig(u, 'edge_owner_to_timeline_media', 'count')),
  }
}

// ── Post parsing ──

function parseMediaNode(node: unknown, ownerUsername?: string, ownerPic?: string): Post | null {
  const n = asMap(node)
  const id = asStr(n.id) || asStr(n.pk)
  const shortcode = asStr(n.shortcode) || asStr(n.code)
  if (!shortcode && !id) return null

  const caption = asStr(dig(n, 'edge_media_to_caption', 'edges', '0', 'node', 'text'))
    || (n.caption && typeof n.caption === 'object' ? asStr((n.caption as Record<string, unknown>).text) : asStr(n.caption))

  const likeCount = asNum(dig(n, 'edge_media_preview_like', 'count'))
    || asNum(dig(n, 'edge_liked_by', 'count'))
    || asNum(n.like_count)

  const commentCount = asNum(dig(n, 'edge_media_to_comment', 'count')) || asNum(n.comment_count)

  let displayUrl = asStr(n.display_url)
  let videoUrl = asStr(n.video_url)

  // XDT format: image_versions2.candidates
  if (!displayUrl) {
    const candidates = asArr(dig(n, 'image_versions2', 'candidates'))
    if (candidates.length > 0) displayUrl = asStr(asMap(candidates[0]).url)
  }
  if (!videoUrl) {
    const versions = asArr(n.video_versions)
    if (versions.length > 0) videoUrl = asStr(asMap(versions[0]).url)
  }

  const dims = asMap(n.dimensions)
  const width = asNum(dims.width) || asNum(n.original_width) || asNum(n.width)
  const height = asNum(dims.height) || asNum(n.original_height) || asNum(n.height)

  const mediaType = asNum(n.media_type)
  const isVideo = asBool(n.is_video) || mediaType === 2

  // Children (carousel)
  let children: Post[] = []
  const sidecar = dig(n, 'edge_sidecar_to_children', 'edges')
  if (sidecar) {
    children = asArr(sidecar).map(e => parseMediaNode(asMap(e).node, ownerUsername, ownerPic)).filter(Boolean) as Post[]
  }
  const carouselMedia = asArr(n.carousel_media)
  if (carouselMedia.length > 0) {
    children = carouselMedia.map(m => parseMediaNode(m, ownerUsername, ownerPic)).filter(Boolean) as Post[]
  }

  const typeName = asStr(n.__typename)
    || (children.length > 0 ? 'GraphSidecar' : isVideo ? 'GraphVideo' : 'GraphImage')

  const owner = asMap(n.owner)
  const un = ownerUsername || asStr(owner.username) || asStr(dig(n, 'user', 'username'))
  const pic = ownerPic || asStr(owner.profile_pic_url) || ''

  const loc = asMap(n.location)

  return {
    id,
    shortcode,
    typeName,
    caption,
    displayUrl,
    videoUrl,
    isVideo,
    width,
    height,
    likeCount,
    commentCount,
    viewCount: asNum(n.video_view_count) || asNum(n.view_count),
    takenAt: tsToISO(n.taken_at_timestamp) || tsToISO(n.taken_at),
    locationId: asStr(loc.id) || asStr(loc.pk),
    locationName: asStr(loc.name),
    ownerUsername: un,
    ownerPic: pic,
    children,
  }
}

function parseMediaEdges(edges: unknown[], ownerUsername?: string, ownerPic?: string): Post[] {
  return edges.map(e => parseMediaNode(asMap(e).node || e, ownerUsername, ownerPic)).filter(Boolean) as Post[]
}

// ── Posts result (paginated) ──

export function parsePostsResult(data: Record<string, unknown>, ownerUsername?: string, ownerPic?: string): PostsResult {
  // XDT format (doc_id response)
  const xdt = dig(data, 'data', 'xdt_api__v1__feed__user_timeline_graphql_connection')
  if (xdt) {
    const conn = asMap(xdt)
    const edges = asArr(conn.edges)
    const posts = edges.map(e => parseMediaNode(asMap(e).node, ownerUsername, ownerPic)).filter(Boolean) as Post[]
    const pageInfo = asMap(conn.page_info)
    return {
      posts,
      cursor: asStr(pageInfo.end_cursor),
      hasMore: asBool(pageInfo.has_next_page),
    }
  }

  // Classic GraphQL format
  const timeline = dig(data, 'data', 'user', 'edge_owner_to_timeline_media')
  if (timeline) {
    const conn = asMap(timeline)
    const posts = parseMediaEdges(asArr(conn.edges), ownerUsername, ownerPic)
    const pageInfo = asMap(conn.page_info)
    return {
      posts,
      cursor: asStr(pageInfo.end_cursor),
      hasMore: asBool(pageInfo.has_next_page),
    }
  }

  return { posts: [], cursor: '', hasMore: false }
}

// ── Single post detail ──

export function parsePostDetail(data: Record<string, unknown>): Post | null {
  // GraphQL format (?__a=1)
  const media = dig(data, 'graphql', 'shortcode_media')
  if (media) return parseMediaNode(media)

  // Items format (newer API)
  const items = asArr(data.items)
  if (items.length > 0) return parseMediaNode(items[0])

  // XDT format via doc_id
  const xdt = dig(data, 'data', 'xdt_shortcode_media')
  if (xdt) return parseMediaNode(xdt)

  return null
}

// ── Comments ──

export function parseComments(data: Record<string, unknown>): CommentsResult {
  const conn = dig(data, 'data', 'shortcode_media', 'edge_media_to_parent_comment')
    || dig(data, 'data', 'shortcode_media', 'edge_media_to_comment')
  if (!conn) return { comments: [], cursor: '', hasMore: false }

  const c = asMap(conn)
  const edges = asArr(c.edges)
  const comments: Comment[] = edges.map(e => {
    const n = asMap(asMap(e).node)
    return {
      id: asStr(n.id),
      text: asStr(n.text),
      authorName: asStr(dig(n, 'owner', 'username')),
      authorPic: asStr(dig(n, 'owner', 'profile_pic_url')),
      likeCount: asNum(dig(n, 'edge_liked_by', 'count')),
      createdAt: tsToISO(n.created_at),
      replyCount: asNum(dig(n, 'edge_threaded_comments', 'count')),
    }
  })

  const pageInfo = asMap(c.page_info)
  return {
    comments,
    cursor: asStr(pageInfo.end_cursor),
    hasMore: asBool(pageInfo.has_next_page),
  }
}

// ── Comment Replies ──

export function parseCommentReplies(data: Record<string, unknown>): CommentsResult {
  const conn = dig(data, 'data', 'comment', 'edge_threaded_comments')
  if (!conn) return { comments: [], cursor: '', hasMore: false }

  const c = asMap(conn)
  const edges = asArr(c.edges)
  const comments: Comment[] = edges.map(e => {
    const n = asMap(asMap(e).node)
    return {
      id: asStr(n.id),
      text: asStr(n.text),
      authorName: asStr(dig(n, 'owner', 'username')),
      authorPic: asStr(dig(n, 'owner', 'profile_pic_url')),
      likeCount: asNum(dig(n, 'edge_liked_by', 'count')),
      createdAt: tsToISO(n.created_at),
      replyCount: 0,
    }
  })

  const pageInfo = asMap(c.page_info)
  return {
    comments,
    cursor: asStr(pageInfo.end_cursor),
    hasMore: asBool(pageInfo.has_next_page),
  }
}

// ── Stories ──

export function parseStories(data: Record<string, unknown>): StoryItem[] {
  const reelsMedia = asArr(data.reels_media)
  if (reelsMedia.length === 0) {
    // Try reels map format
    const reels = asMap(data.reels)
    for (const key of Object.keys(reels)) {
      const reel = asMap(reels[key])
      return parseStoryItems(asArr(reel.items), asStr(dig(reel, 'user', 'username')))
    }
    return []
  }

  const reel = asMap(reelsMedia[0])
  const items = asArr(reel.items)
  const username = asStr(dig(reel, 'user', 'username'))
  return parseStoryItems(items, username)
}

function parseStoryItems(items: unknown[], username: string): StoryItem[] {
  return items.map(item => {
    const m = asMap(item)
    let displayUrl = ''
    const imgVersions = asArr(dig(m, 'image_versions2', 'candidates'))
    if (imgVersions.length > 0) displayUrl = asStr(asMap(imgVersions[0]).url)

    let videoUrl = ''
    const vidVersions = asArr(m.video_versions)
    if (vidVersions.length > 0) videoUrl = asStr(asMap(vidVersions[0]).url)

    return {
      id: asStr(m.id) || asStr(m.pk),
      displayUrl,
      videoUrl,
      isVideo: asNum(m.media_type) === 2,
      width: asNum(m.original_width),
      height: asNum(m.original_height),
      takenAt: tsToISO(m.taken_at),
      expiresAt: tsToISO(m.expiring_at),
      ownerUsername: username || asStr(dig(m, 'user', 'username')),
    }
  })
}

// ── Highlights ──

export function parseHighlights(data: Record<string, unknown>): Highlight[] {
  const edges = asArr(dig(data, 'data', 'user', 'edge_highlight_reels', 'edges'))
  return edges.map(e => {
    const n = asMap(asMap(e).node)
    return {
      id: asStr(n.id),
      title: asStr(n.title),
      coverUrl: asStr(dig(n, 'cover_media_cropped_thumbnail', 'cropped_image_version'))
        || asStr(dig(n, 'cover_media', 'thumbnail_src')),
      itemCount: asNum(dig(n, 'edge_highlight_items', 'count')),
    }
  })
}

// ── Reels ──

export function parseReels(data: Record<string, unknown>): ReelsResult {
  const conn = dig(data, 'data', 'xdt_api__v1__clips__user__connection_v2')
  if (!conn) return { reels: [], cursor: '', hasMore: false }

  const c = asMap(conn)
  const edges = asArr(c.edges)
  const reels: Reel[] = edges.map(e => {
    const n = asMap(asMap(e).node)
    const media = asMap(n.media)

    let displayUrl = ''
    const candidates = asArr(dig(media, 'image_versions2', 'candidates'))
    if (candidates.length > 0) displayUrl = asStr(asMap(candidates[0]).url)

    let videoUrl = ''
    const versions = asArr(media.video_versions)
    if (versions.length > 0) videoUrl = asStr(asMap(versions[0]).url)

    return {
      id: asStr(media.id) || asStr(media.pk),
      shortcode: asStr(media.code),
      caption: asStr(dig(media, 'caption', 'text')),
      displayUrl,
      videoUrl,
      width: asNum(media.original_width),
      height: asNum(media.original_height),
      likeCount: asNum(media.like_count),
      commentCount: asNum(media.comment_count),
      viewCount: asNum(media.view_count) || asNum(media.play_count),
      playCount: asNum(media.play_count),
      takenAt: tsToISO(media.taken_at),
      ownerUsername: asStr(dig(media, 'user', 'username')),
    }
  })

  const pageInfo = asMap(c.page_info)
  return {
    reels,
    cursor: asStr(pageInfo.end_cursor),
    hasMore: asBool(pageInfo.has_next_page),
  }
}

// ── Followers / Following ──

export function parseFollowList(data: Record<string, unknown>): FollowResult {
  const conn = dig(data, 'data', 'user', 'edge_followed_by')
    || dig(data, 'data', 'user', 'edge_follow')
  if (!conn) return { users: [], cursor: '', hasMore: false }

  const c = asMap(conn)
  const edges = asArr(c.edges)
  const users: FollowUser[] = edges.map(e => {
    const n = asMap(asMap(e).node)
    return {
      id: asStr(n.id),
      username: asStr(n.username),
      fullName: asStr(n.full_name),
      isPrivate: asBool(n.is_private),
      isVerified: asBool(n.is_verified),
      picUrl: asStr(n.profile_pic_url),
    }
  })

  const pageInfo = asMap(c.page_info)
  return {
    users,
    cursor: asStr(pageInfo.end_cursor),
    hasMore: asBool(pageInfo.has_next_page),
  }
}

// ── Search ──

export function parseSearchResults(data: Record<string, unknown>): SearchResult {
  const users: SearchUser[] = asArr(data.users).map(wrapper => {
    const u = asMap(asMap(wrapper).user)
    return {
      id: asStr(u.pk),
      username: asStr(u.username),
      fullName: asStr(u.full_name),
      isPrivate: asBool(u.is_private),
      isVerified: asBool(u.is_verified),
      picUrl: asStr(u.profile_pic_url),
      followers: asNum(u.follower_count),
    }
  })

  const hashtags: SearchHashtag[] = asArr(data.hashtags).map(wrapper => {
    const h = asMap(asMap(wrapper).hashtag)
    return {
      id: asNum(h.id),
      name: asStr(h.name),
      mediaCount: asNum(h.media_count),
    }
  })

  const places: SearchPlace[] = asArr(data.places).map(wrapper => {
    const p = asMap(wrapper)
    const loc = asMap(asMap(p.place).location)
    return {
      locationId: asNum(loc.pk),
      title: asStr(asMap(p.place).title),
      address: asStr(loc.address),
      city: asStr(loc.city),
      lat: asNum(loc.lat) as number,
      lng: asNum(loc.lng) as number,
    }
  })

  return { users, hashtags, places }
}

// ── Hashtag posts ──

export function parseHashtagPosts(data: Record<string, unknown>): PostsResult {
  const conn = dig(data, 'data', 'hashtag', 'edge_hashtag_to_media')
  if (!conn) return { posts: [], cursor: '', hasMore: false }

  const c = asMap(conn)
  const posts = parseMediaEdges(asArr(c.edges))
  const pageInfo = asMap(c.page_info)
  return {
    posts,
    cursor: asStr(pageInfo.end_cursor),
    hasMore: asBool(pageInfo.has_next_page),
  }
}

// ── Location posts ──

export function parseLocationPosts(data: Record<string, unknown>): PostsResult & { locationName: string } {
  const loc = dig(data, 'data', 'location')
  if (!loc) return { posts: [], cursor: '', hasMore: false, locationName: '' }

  const l = asMap(loc)
  const conn = asMap(l.edge_location_to_media)
  const posts = parseMediaEdges(asArr(conn.edges))
  const pageInfo = asMap(conn.page_info)
  return {
    posts,
    cursor: asStr(pageInfo.end_cursor),
    hasMore: asBool(pageInfo.has_next_page),
    locationName: asStr(l.name),
  }
}

// ── Tagged posts ──

export function parseTaggedPosts(data: Record<string, unknown>): PostsResult {
  const conn = dig(data, 'data', 'user', 'edge_user_to_photos_of_you')
  if (!conn) return { posts: [], cursor: '', hasMore: false }

  const c = asMap(conn)
  const posts = parseMediaEdges(asArr(c.edges))
  const pageInfo = asMap(c.page_info)
  return {
    posts,
    cursor: asStr(pageInfo.end_cursor),
    hasMore: asBool(pageInfo.has_next_page),
  }
}
