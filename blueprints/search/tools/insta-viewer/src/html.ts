import { cssURL } from './asset'
import type { Profile, Post, Comment, StoryItem, Highlight, Reel, FollowUser, SearchResult } from './types'

// ── SVG Icons (Instagram-native) ──

const svg = {
  // Instagram wordmark-style camera
  camera: '<svg width="103" height="29" viewBox="0 0 103 29" fill="currentColor"><path d="M5.23 7.95c-2.72 0-5 2.24-5 5.01v6.53c0 2.76 2.28 5.01 5 5.01h6.52c2.72 0 5-2.25 5-5.01v-6.53c0-2.77-2.28-5.01-5-5.01H5.23zm3.26 13.4c-2.53 0-4.58-2.08-4.58-4.64 0-2.56 2.05-4.64 4.58-4.64 2.53 0 4.59 2.08 4.59 4.64 0 2.56-2.06 4.64-4.59 4.64zm4.78-8.38a1.08 1.08 0 110-2.16 1.08 1.08 0 010 2.16zm-4.78 1.4c-1.68 0-3.04 1.38-3.04 3.09 0 1.7 1.36 3.09 3.04 3.09 1.68 0 3.04-1.39 3.04-3.09 0-1.71-1.36-3.09-3.04-3.09z"/><text x="22" y="22" font-size="19" font-family="-apple-system,sans-serif" font-style="italic">Insta Viewer</text></svg>',
  cameraBig: '<svg width="175" height="50" viewBox="0 0 103 29" fill="currentColor"><path d="M5.23 7.95c-2.72 0-5 2.24-5 5.01v6.53c0 2.76 2.28 5.01 5 5.01h6.52c2.72 0 5-2.25 5-5.01v-6.53c0-2.77-2.28-5.01-5-5.01H5.23zm3.26 13.4c-2.53 0-4.58-2.08-4.58-4.64 0-2.56 2.05-4.64 4.58-4.64 2.53 0 4.59 2.08 4.59 4.64 0 2.56-2.06 4.64-4.59 4.64zm4.78-8.38a1.08 1.08 0 110-2.16 1.08 1.08 0 010 2.16zm-4.78 1.4c-1.68 0-3.04 1.38-3.04 3.09 0 1.7 1.36 3.09 3.04 3.09 1.68 0 3.04-1.39 3.04-3.09 0-1.71-1.36-3.09-3.04-3.09z"/><text x="22" y="22" font-size="19" font-family="-apple-system,sans-serif" font-style="italic">Insta Viewer</text></svg>',
  heart: '<svg viewBox="0 0 24 24"><path d="M16.792 3.904A4.989 4.989 0 0121.5 9.122c0 3.072-2.652 4.959-5.197 7.222-2.512 2.243-3.865 3.469-4.303 3.752-.477-.309-2.143-1.823-4.303-3.752C5.141 14.072 2.5 12.167 2.5 9.122a4.989 4.989 0 014.708-5.218 4.21 4.21 0 013.675 1.941c.84 1.175.98 1.763 1.12 1.763s.278-.588 1.11-1.766a4.17 4.17 0 013.679-1.938z"/></svg>',
  heartOutline: '<svg viewBox="0 0 24 24"><path d="M16.792 3.904A4.989 4.989 0 0121.5 9.122c0 3.072-2.652 4.959-5.197 7.222-2.512 2.243-3.865 3.469-4.303 3.752-.477-.309-2.143-1.823-4.303-3.752C5.141 14.072 2.5 12.167 2.5 9.122a4.989 4.989 0 014.708-5.218 4.21 4.21 0 013.675 1.941c.84 1.175.98 1.763 1.12 1.763s.278-.588 1.11-1.766a4.17 4.17 0 013.679-1.938z" fill="none" stroke="currentColor" stroke-width="2"/></svg>',
  comment: '<svg viewBox="0 0 24 24"><path d="M20.656 17.008a9.993 9.993 0 10-3.59 3.615L22 22z" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="2"/></svg>',
  share: '<svg viewBox="0 0 24 24"><line x1="22" y1="3" x2="9.218" y2="10.083" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="2"/><polygon points="22 3 15 22 11 13 2 9" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="2"/></svg>',
  bookmark: '<svg viewBox="0 0 24 24"><polygon points="20 21 12 13.44 4 21 4 3 20 3" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="2"/></svg>',
  grid: '<svg viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7" fill="none" stroke="currentColor" stroke-width="2"/><rect x="14" y="3" width="7" height="7" fill="none" stroke="currentColor" stroke-width="2"/><rect x="3" y="14" width="7" height="7" fill="none" stroke="currentColor" stroke-width="2"/><rect x="14" y="14" width="7" height="7" fill="none" stroke="currentColor" stroke-width="2"/></svg>',
  reels: '<svg viewBox="0 0 24 24"><path d="M12 2.982c2.937 0 3.285.011 4.445.064a6.087 6.087 0 012.042.379 3.408 3.408 0 011.265.823 3.408 3.408 0 01.823 1.265 6.087 6.087 0 01.379 2.042c.053 1.16.064 1.508.064 4.445s-.011 3.285-.064 4.445a6.087 6.087 0 01-.379 2.042 3.643 3.643 0 01-2.088 2.088 6.087 6.087 0 01-2.042.379c-1.16.053-1.508.064-4.445.064s-3.285-.011-4.445-.064a6.087 6.087 0 01-2.042-.379 3.408 3.408 0 01-1.265-.823 3.408 3.408 0 01-.823-1.265 6.087 6.087 0 01-.379-2.042c-.053-1.16-.064-1.508-.064-4.445s.011-3.285.064-4.445a6.087 6.087 0 01.379-2.042 3.408 3.408 0 01.823-1.265 3.408 3.408 0 011.265-.823 6.087 6.087 0 012.042-.379c1.16-.053 1.508-.064 4.445-.064M12 1c-2.987 0-3.362.013-4.535.066a8.074 8.074 0 00-2.67.511 5.392 5.392 0 00-1.949 1.27 5.392 5.392 0 00-1.269 1.948 8.074 8.074 0 00-.51 2.67C1.012 8.638 1 9.013 1 12s.013 3.362.066 4.535a8.074 8.074 0 00.511 2.67 5.392 5.392 0 001.27 1.949 5.392 5.392 0 001.948 1.269 8.074 8.074 0 002.67.51C8.638 22.988 9.013 23 12 23s3.362-.013 4.535-.066a8.074 8.074 0 002.67-.511 5.625 5.625 0 003.218-3.218 8.074 8.074 0 00.51-2.67C22.988 15.362 23 14.987 23 12s-.013-3.362-.066-4.535a8.074 8.074 0 00-.511-2.67 5.392 5.392 0 00-1.27-1.949 5.392 5.392 0 00-1.948-1.269 8.074 8.074 0 00-2.67-.51C15.362 1.012 14.987 1 12 1z" fill="currentColor"/><path d="M10 7.757l6 4.243-6 4.243z" fill="currentColor"/></svg>',
  tagged: '<svg viewBox="0 0 24 24"><path d="M10.201 3.797L12 1.997l1.799 1.8a1.59 1.59 0 001.124.465h2.55a1.59 1.59 0 011.59 1.59v2.55c0 .421.167.825.465 1.124l1.8 1.799-1.8 1.799a1.59 1.59 0 00-.465 1.124v2.55a1.59 1.59 0 01-1.59 1.59h-2.55a1.59 1.59 0 00-1.124.465l-1.799 1.8-1.799-1.8a1.59 1.59 0 00-1.124-.465h-2.55a1.59 1.59 0 01-1.59-1.59v-2.55a1.59 1.59 0 00-.465-1.124l-1.8-1.799 1.8-1.799a1.59 1.59 0 00.465-1.124v-2.55a1.59 1.59 0 011.59-1.59h2.55a1.59 1.59 0 001.124-.465z" fill="none" stroke="currentColor" stroke-linejoin="round" stroke-width="2"/><circle cx="12" cy="12" r="3" fill="none" stroke="currentColor" stroke-width="2"/></svg>',
  verified: '<svg width="18" height="18" viewBox="0 0 40 40"><circle cx="20" cy="20" r="20" fill="#0095f6"/><path d="M17.2 29.2l-6.6-6.6 2.4-2.4 4.2 4.2 10.2-10.2 2.4 2.4z" fill="#fff"/></svg>',
  lock: '<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M6 10V7c0-3.31 2.69-6 6-6s6 2.69 6 6v3h1c1.1 0 2 .9 2 2v9c0 1.1-.9 2-2 2H5c-1.1 0-2-.9-2-2v-9c0-1.1.9-2 2-2h1zm2 0h8V7c0-2.21-1.79-4-4-4S8 4.79 8 7v3zm4 9c1.1 0 2-.9 2-2s-.9-2-2-2-2 .9-2 2 .9 2 2 2z"/></svg>',
  carousel: '<svg viewBox="0 0 24 24" fill="#fff"><path d="M18 2H6c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h12c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm-2 12H8v-2h8v2zm2-4H6V4h12v6z"/></svg>',
  video: '<svg viewBox="0 0 24 24" fill="#fff"><path d="M8 5v14l11-7z"/></svg>',
  back: '<svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="M21 11H6.83l5.59-5.59L11 4l-8 8 8 8 1.41-1.41L6.83 13H21z"/></svg>',
  moon: '<svg class="icon-moon" width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M21.53 15.93c-1.18.49-2.47.76-3.81.76-5.57 0-10.09-4.55-10.09-10.18 0-1.34.26-2.63.73-3.81C4.48 4.4 1.88 8.09 1.88 12.4 1.88 17.85 6.34 22.25 11.73 22.25c4.26 0 7.91-2.61 9.8-6.32z"/></svg>',
  sun: '<svg class="icon-sun" width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 7c-2.76 0-5 2.24-5 5s2.24 5 5 5 5-2.24 5-5-2.24-5-5-5zm0 8c-1.65 0-3-1.35-3-3s1.35-3 3-3 3 1.35 3 3-1.35 3-3 3zm1-13h-2v3h2V2zm0 19h-2v3h2v-3zm9-10v2h-3v-2h3zM5 11v2H2v-2h3zm13.07-6.36l-1.42 1.42-2.12-2.12 1.41-1.42 2.13 2.12zM8.46 19.07l-1.41 1.41-2.12-2.12 1.41-1.42 2.12 2.13zm10.6 1.41l-1.41-1.41-2.12 2.12 1.41 1.42 2.12-2.13zM6.34 6.47L4.93 4.36l2.12-2.12 1.42 1.41L6.34 6.47z"/></svg>',
  igLogo: '<svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2.982c2.937 0 3.285.011 4.445.064a6.087 6.087 0 012.042.379 3.408 3.408 0 011.265.823 3.408 3.408 0 01.823 1.265 6.087 6.087 0 01.379 2.042c.053 1.16.064 1.508.064 4.445s-.011 3.285-.064 4.445a6.087 6.087 0 01-.379 2.042 3.643 3.643 0 01-2.088 2.088 6.087 6.087 0 01-2.042.379c-1.16.053-1.508.064-4.445.064s-3.285-.011-4.445-.064a6.087 6.087 0 01-2.042-.379 3.643 3.643 0 01-2.088-2.088 6.087 6.087 0 01-.379-2.042c-.053-1.16-.064-1.508-.064-4.445s.011-3.285.064-4.445a6.087 6.087 0 01.379-2.042 3.408 3.408 0 01.823-1.265 3.408 3.408 0 011.265-.823 6.087 6.087 0 012.042-.379c1.16-.053 1.508-.064 4.445-.064M12 1c-2.987 0-3.362.013-4.535.066a8.074 8.074 0 00-2.67.511 5.392 5.392 0 00-1.949 1.27 5.392 5.392 0 00-1.269 1.948 8.074 8.074 0 00-.51 2.67C1.012 8.638 1 9.013 1 12s.013 3.362.066 4.535a8.074 8.074 0 00.511 2.67 5.625 5.625 0 003.218 3.218 8.074 8.074 0 002.67.51C8.638 22.988 9.013 23 12 23s3.362-.013 4.535-.066a8.074 8.074 0 002.67-.511 5.625 5.625 0 003.218-3.218 8.074 8.074 0 00.51-2.67C22.988 15.362 23 14.987 23 12s-.013-3.362-.066-4.535a8.074 8.074 0 00-.511-2.67 5.392 5.392 0 00-1.27-1.949 5.392 5.392 0 00-1.948-1.269 8.074 8.074 0 00-2.67-.51C15.362 1.012 14.987 1 12 1z"/><circle cx="12" cy="12" r="3.2" fill="none" stroke="currentColor" stroke-width="1.8"/><circle cx="18.406" cy="5.594" r="1.44" fill="currentColor"/></svg>',
  chevronLeft: '<svg viewBox="0 0 24 24"><polyline points="15 18 9 12 15 6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>',
  chevronRight: '<svg viewBox="0 0 24 24"><polyline points="9 18 15 12 9 6" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/></svg>',
}

// ── Helpers ──

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

// Proxy Instagram CDN URLs through /img/ to bypass hotlinking protection.
// Returns HTML-escaped URL ready for use in src attributes.
function img(url: string): string {
  if (!url) return ''
  if (url.includes('.cdninstagram.com/') || url.includes('.fbcdn.net/')) {
    return esc('/img/' + encodeURIComponent(url))
  }
  return esc(url)
}

function fmtNum(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1).replace(/\.0$/, '') + 'M'
  if (n >= 10_000) return (n / 1_000).toFixed(0) + 'K'
  if (n >= 1_000) return (n / 1_000).toFixed(1).replace(/\.0$/, '') + 'K'
  return n.toLocaleString()
}

function relTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const sec = Math.floor((Date.now() - d.getTime()) / 1000)
  if (sec < 60) return `${sec}s`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h`
  const days = Math.floor(hr / 24)
  if (days < 7) return `${days}d`
  const weeks = Math.floor(days / 7)
  if (weeks < 52) return `${weeks}w`
  const mo = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']
  return `${mo[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`
}

function fullDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const mo = ['January', 'February', 'March', 'April', 'May', 'June', 'July', 'August', 'September', 'October', 'November', 'December']
  return `${mo[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`
}

function linkify(text: string): string {
  let s = esc(text)
  s = s.replace(/@([\w.]+)/g, '<a href="/$1">@$1</a>')
  s = s.replace(/#(\w+)/g, '<a href="/explore/tags/$1">#$1</a>')
  s = s.replace(/(https?:\/\/[^\s<]+)/g, '<a href="$1" target="_blank" rel="noopener">$1</a>')
  return s
}

function verifiedBadge(isVerified: boolean): string {
  if (!isVerified) return ''
  return ` <span class="profile-verified">${svg.verified}</span>`
}

const defaultAvi = 'https://dummyimage.com/150x150/dbdbdb/8e8e8e&text=+'

// ── Post Grid ──

export function renderPostGrid(posts: Post[]): string {
  let h = '<div class="post-grid">'
  for (const p of posts) {
    const url = `/p/${esc(p.shortcode)}`
    const thumb = p.displayUrl || (p.children.length > 0 ? p.children[0].displayUrl : '')
    const isCarousel = p.children.length > 0 || p.typeName === 'GraphSidecar'
    const isVideo = p.isVideo || p.typeName === 'GraphVideo'
    h += `<a href="${url}" class="post-grid-item">`
    h += `<img src="${img(thumb)}" alt="" loading="lazy">`
    h += `<div class="post-grid-overlay"><span class="post-grid-stat">${svg.heart} ${fmtNum(p.likeCount)}</span><span class="post-grid-stat">${svg.comment} ${fmtNum(p.commentCount)}</span></div>`
    if (isCarousel) h += `<span class="post-grid-badge">${svg.carousel}</span>`
    else if (isVideo) h += `<span class="post-grid-badge">${svg.video}</span>`
    h += '</a>'
  }
  h += '</div>'
  return h
}

// ── Profile Header ──

export function renderProfileHeader(profile: Profile): string {
  const avi = profile.profilePicUrl || defaultAvi
  return `<div class="profile"><div class="profile-header"><div class="profile-avatar-wrap"><img class="profile-avatar" src="${img(avi)}" alt="${esc(profile.username)}"></div><div class="profile-info"><div class="profile-top"><span class="profile-username">${esc(profile.username)}${verifiedBadge(profile.isVerified)}${profile.isPrivate ? ` ${svg.lock}` : ''}</span></div><div class="profile-stats"><span><strong>${fmtNum(profile.postCount)}</strong> posts</span><a href="/${esc(profile.username)}/followers"><strong>${fmtNum(profile.followerCount)}</strong> followers</a><a href="/${esc(profile.username)}/following"><strong>${fmtNum(profile.followingCount)}</strong> following</a></div><div class="profile-bio">${profile.fullName ? `<div class="profile-bio-name">${esc(profile.fullName)}</div>` : ''}${profile.categoryName ? `<div class="profile-bio-category">${esc(profile.categoryName)}</div>` : ''}${profile.biography ? `<div class="profile-bio-text">${linkify(profile.biography)}</div>` : ''}${profile.externalUrl ? `<a class="profile-bio-link" href="${esc(profile.externalUrl)}" target="_blank" rel="noopener">${esc(profile.externalUrl.replace(/^https?:\/\//, '').replace(/\/$/, ''))}</a>` : ''}</div><div class="profile-orig"><a href="https://www.instagram.com/${esc(profile.username)}/" target="_blank" rel="noopener">View on Instagram ${svg.igLogo}</a></div></div></div></div>`
}

// ── Highlights Row ──

export function renderHighlights(highlights: Highlight[]): string {
  if (highlights.length === 0) return ''
  let h = '<div class="highlights">'
  for (const hl of highlights) {
    h += `<a href="/stories/highlights/${esc(hl.id)}" class="highlight-item"><div class="highlight-circle"><img src="${img(hl.coverUrl || defaultAvi)}" alt="" loading="lazy"></div><span class="highlight-title">${esc(hl.title)}</span></a>`
  }
  h += '</div>'
  return h
}

// ── Post Detail ──

export function renderPostDetail(post: Post, comments: Comment[], profile?: Profile): string {
  const avi = post.ownerPic || profile?.profilePicUrl || defaultAvi
  const hasMultiple = post.children.length > 0

  // Media section
  let mediaHtml = ''
  if (hasMultiple) {
    const items = post.children
    mediaHtml += `<div class="post-media" id="carousel">`
    for (let i = 0; i < items.length; i++) {
      const child = items[i]
      const display = i === 0 ? 'block' : 'none'
      if (child.isVideo && child.videoUrl) {
        mediaHtml += `<video class="carousel-slide" style="display:${display}" src="${img(child.videoUrl)}" poster="${img(child.displayUrl)}" controls playsinline preload="none"></video>`
      } else {
        mediaHtml += `<img class="carousel-slide" style="display:${display}" src="${img(child.displayUrl)}" alt="" loading="lazy">`
      }
    }
    if (items.length > 1) {
      mediaHtml += `<button class="carousel-nav carousel-prev" onclick="slide(-1)">${svg.chevronLeft}</button>`
      mediaHtml += `<button class="carousel-nav carousel-next" onclick="slide(1)">${svg.chevronRight}</button>`
      mediaHtml += `<div class="carousel-dots">`
      for (let i = 0; i < items.length; i++) {
        mediaHtml += `<span class="carousel-dot${i === 0 ? ' active' : ''}"></span>`
      }
      mediaHtml += '</div>'
    }
    mediaHtml += '</div>'
  } else if (post.isVideo && post.videoUrl) {
    mediaHtml = `<div class="post-media"><video src="${img(post.videoUrl)}" poster="${img(post.displayUrl)}" controls playsinline preload="metadata" style="width:100%"></video></div>`
  } else {
    mediaHtml = `<div class="post-media"><img src="${img(post.displayUrl)}" alt=""></div>`
  }

  // Sidebar
  let sidebar = `<div class="post-sidebar"><div class="post-sidebar-header"><a href="/${esc(post.ownerUsername)}"><img class="post-sidebar-avi" src="${img(avi)}" alt=""></a><div><a href="/${esc(post.ownerUsername)}" class="post-sidebar-user">${esc(post.ownerUsername)}</a>${post.locationName ? `<div class="post-sidebar-loc">${esc(post.locationName)}</div>` : ''}</div></div><div class="post-sidebar-body">`

  // Caption
  if (post.caption) {
    sidebar += `<div class="post-caption"><img class="post-caption-avi" src="${img(avi)}" alt="" loading="lazy"><div class="post-caption-body"><a href="/${esc(post.ownerUsername)}" class="post-caption-user">${esc(post.ownerUsername)}</a><span class="post-caption-text">${linkify(post.caption)}</span><span class="post-caption-time">${relTime(post.takenAt)}</span></div></div>`
  }

  // Comments
  for (const c of comments) {
    sidebar += renderComment(c)
  }

  sidebar += '</div>'

  // Actions + stats
  sidebar += `<div class="post-actions"><div class="post-action-row"><span class="post-action-btn">${svg.heartOutline}</span><span class="post-action-btn">${svg.comment}</span><span class="post-action-btn">${svg.share}</span><span class="post-action-spacer"></span><span class="post-action-btn">${svg.bookmark}</span></div></div>`
  sidebar += `<div class="post-likes">${fmtNum(post.likeCount)} likes</div>`
  sidebar += `<div class="post-time">${fullDate(post.takenAt)}</div>`
  sidebar += '</div>'

  return `<div class="post-detail">${mediaHtml}${sidebar}</div>`
}

// ── Comment ──

function renderComment(c: Comment): string {
  const avi = c.authorPic || defaultAvi
  return `<div class="comment"><img class="comment-avi" src="${img(avi)}" alt="" loading="lazy"><div class="comment-body"><a href="/${esc(c.authorName)}" class="comment-user">${esc(c.authorName)}</a> <span class="comment-text">${linkify(c.text)}</span><div class="comment-meta"><span>${relTime(c.createdAt)}</span>${c.likeCount > 0 ? `<span class="comment-likes">${fmtNum(c.likeCount)} likes</span>` : ''}${c.replyCount > 0 ? `<span>${c.replyCount} replies</span>` : ''}</div></div></div>`
}

// ── Stories Viewer ──

export function renderStoriesViewer(username: string, items: StoryItem[], profilePic: string): string {
  if (items.length === 0) return '<div class="err"><h2>No stories</h2><p>This account has no active stories.</p></div>'

  const avi = profilePic || defaultAvi
  let h = `<div class="stories"><div class="story-container">`

  // Progress bars
  h += '<div class="story-progress">'
  for (let i = 0; i < items.length; i++) {
    const cls = i === 0 ? 'active' : ''
    h += `<div class="story-bar ${cls}"><div class="story-bar-fill"></div></div>`
  }
  h += '</div>'

  // Header
  h += `<div class="story-header"><img class="story-avi" src="${img(avi)}" alt=""><a href="/${esc(username)}" class="story-username">${esc(username)}</a><span class="story-time">${relTime(items[0].takenAt)}</span></div>`

  // Items
  for (let i = 0; i < items.length; i++) {
    const item = items[i]
    const cls = i === 0 ? 'active' : ''
    if (item.isVideo && item.videoUrl) {
      h += `<div class="story-item ${cls}"><video class="story-media" src="${img(item.videoUrl)}" poster="${img(item.displayUrl)}" playsinline muted></video></div>`
    } else {
      h += `<div class="story-item ${cls}"><img class="story-media" src="${img(item.displayUrl)}" alt=""></div>`
    }
  }

  h += '</div></div>'
  return h
}

// ── Reel Detail ──

export function renderReelDetail(reel: Reel): string {
  return `<div class="reel-detail"><div class="reel-container"><video class="reel-video" src="${img(reel.videoUrl)}" poster="${img(reel.displayUrl)}" controls playsinline preload="metadata"></video><div class="reel-overlay"><div class="reel-user"><a href="/${esc(reel.ownerUsername)}">${esc(reel.ownerUsername)}</a></div>${reel.caption ? `<div class="reel-caption">${linkify(reel.caption)}</div>` : ''}</div><div class="reel-stats"><div class="reel-stat">${svg.heart} ${fmtNum(reel.likeCount)}</div><div class="reel-stat">${svg.comment} ${fmtNum(reel.commentCount)}</div></div></div></div>`
}

// ── User Card ──

export function renderUserCard(u: FollowUser): string {
  const avi = u.picUrl || defaultAvi
  return `<div class="user-card"><a href="/${esc(u.username)}" class="user-card-link"></a><img class="user-card-avi" src="${img(avi)}" alt="" loading="lazy"><div class="user-card-body"><div class="user-card-name">${esc(u.username)}${verifiedBadge(u.isVerified)}</div><div class="user-card-full">${esc(u.fullName)}</div></div></div>`
}

// ── Search User Card ──

export function renderSearchUserCard(u: { id: string; username: string; fullName: string; isVerified: boolean; picUrl: string; followers?: number; isPrivate?: boolean }): string {
  const avi = u.picUrl || defaultAvi
  return `<div class="user-card"><a href="/${esc(u.username)}" class="user-card-link"></a><img class="user-card-avi" src="${img(avi)}" alt="" loading="lazy"><div class="user-card-body"><div class="user-card-name">${esc(u.username)}${verifiedBadge(u.isVerified)}</div><div class="user-card-full">${esc(u.fullName)}</div>${u.followers ? `<div class="user-card-extra">${fmtNum(u.followers)} followers</div>` : ''}</div></div>`
}

// ── Search Results ──

export function renderSearchResults(result: SearchResult): string {
  let h = '<div class="search-page">'

  if (result.users.length > 0) {
    h += '<div class="search-section"><div class="search-section-title">Accounts</div>'
    for (const u of result.users) h += renderSearchUserCard(u)
    h += '</div>'
  }

  if (result.hashtags.length > 0) {
    h += '<div class="search-section"><div class="search-section-title">Hashtags</div>'
    for (const tag of result.hashtags) {
      h += `<div class="hashtag-card"><a href="/explore/tags/${esc(tag.name)}" class="hashtag-card-link"></a><div class="hashtag-icon">#</div><div class="hashtag-body"><div class="hashtag-name">#${esc(tag.name)}</div><div class="hashtag-count">${fmtNum(tag.mediaCount)} posts</div></div></div>`
    }
    h += '</div>'
  }

  if (result.places.length > 0) {
    h += '<div class="search-section"><div class="search-section-title">Places</div>'
    for (const place of result.places) {
      h += `<div class="hashtag-card"><a href="/explore/locations/${place.locationId}" class="hashtag-card-link"></a><div class="hashtag-icon">${svg.lock.replace('lock', 'pin')}</div><div class="hashtag-body"><div class="hashtag-name">${esc(place.title)}</div><div class="hashtag-count">${esc(place.address || place.city)}</div></div></div>`
    }
    h += '</div>'
  }

  if (result.users.length === 0 && result.hashtags.length === 0 && result.places.length === 0) {
    h += '<div class="err"><h2>No results</h2><p>No results found for this search.</p></div>'
  }

  h += '</div>'
  return h
}

// ── Follow Page ──

export function renderFollowPage(username: string, users: FollowUser[], tab: 'followers' | 'following'): string {
  let h = `<div class="follow-page"><div class="sh"><a href="/${esc(username)}">${svg.back}</a><span class="sh-title">${esc(username)}</span></div><div class="follow-tabs"><a href="/${esc(username)}/followers" class="${tab === 'followers' ? 'active' : ''}">Followers</a><a href="/${esc(username)}/following" class="${tab === 'following' ? 'active' : ''}">Following</a></div>`
  if (users.length === 0) {
    h += `<div class="err"><p>No ${tab} found.</p></div>`
  } else {
    for (const u of users) h += renderUserCard(u)
  }
  h += '</div>'
  return h
}

// ── Hashtag / Location Page Header ──

export function renderPageHeader(icon: string, title: string, stat: string): string {
  return `<div class="page-header"><div class="page-header-icon">${icon}</div><div class="page-header-info"><div class="page-header-title">${esc(title)}</div><div class="page-header-stat">${stat}</div></div></div>`
}

// ── Private Account Message ──

export function renderPrivateMessage(): string {
  return `<div class="private-msg"><div class="private-icon">${svg.lock}</div><div class="private-title">This Account is Private</div><div class="private-text">Follow this account to see their photos and videos.</div></div>`
}

// ── Pagination ──

export function renderPagination(cursor: string, currentPath: string): string {
  if (!cursor) return ''
  const sep = currentPath.includes('?') ? '&' : '?'
  const href = `${currentPath}${sep}cursor=${encodeURIComponent(cursor)}`
  return `<div class="more" data-href="${esc(href)}"><span class="more-spinner"></span></div>`
}

// ── Home Page ──

export function renderHomePage(): string {
  return `<div class="home"><div class="home-logo">${svg.cameraBig}</div><div class="home-sub">the Instagram Viewer</div><div class="home-box"><form action="/search" method="get"><input class="home-input" type="text" name="q" placeholder="Search @username, #hashtag, or keyword" autocomplete="off" autofocus></form><div class="home-hint">Type @username to view a profile</div><div class="home-links"><a href="/nasa">@nasa</a><a href="/natgeo">@natgeo</a><a href="/explore/tags/photography">#photography</a><a href="/cristiano">@cristiano</a><a href="/explore/tags/travel">#travel</a></div></div><div class="home-theme"><button class="theme-toggle" onclick="T()" title="Toggle theme">${svg.moon}${svg.sun}</button></div></div>`
}

// ── Layout ──

const themeScript = `<script>(function(){var t=localStorage.getItem('it');if(!t)t=matchMedia('(prefers-color-scheme:dark)').matches?'d':'l';document.documentElement.dataset.t=t})();function T(){var h=document.documentElement,n=h.dataset.t==='d'?'l':'d';h.dataset.t=n;localStorage.setItem('it',n)}</script>`

const scrollScript = `<script>(function(){var loading=false;function observe(){var el=document.querySelector('.more[data-href]');if(!el)return;var io=new IntersectionObserver(function(entries){if(!entries[0].isIntersecting||loading)return;loading=true;var href=el.getAttribute('data-href');fetch(href).then(function(r){return r.text()}).then(function(html){var doc=new DOMParser().parseFromString(html,'text/html');var wrap=doc.querySelector('.wrap');if(!wrap)return;var items=wrap.querySelectorAll('.post-grid-item,.user-card,.hashtag-card,.comment');var parent=el.parentNode;items.forEach(function(n){parent.insertBefore(n.cloneNode(true),el)});var next=wrap.querySelector('.more[data-href]');if(next){el.setAttribute('data-href',next.getAttribute('data-href'))}else{el.remove()}loading=false}).catch(function(){el.innerHTML='<a href="'+href+'" class="more-fallback">Load more</a>';el.removeAttribute('data-href');loading=false})},{rootMargin:'600px'});io.observe(el)}observe();new MutationObserver(observe).observe(document.body,{childList:true,subtree:true})})()</script>`

const carouselScript = `<script>var ci=0;function slide(d){var s=document.querySelectorAll('.carousel-slide');var dots=document.querySelectorAll('.carousel-dot');if(!s.length)return;s[ci].style.display='none';if(dots[ci])dots[ci].classList.remove('active');ci=(ci+d+s.length)%s.length;s[ci].style.display='block';if(dots[ci])dots[ci].classList.add('active')}</script>`

function renderNav(query?: string): string {
  return `<div class="nav"><div class="nav-inner"><a href="/" class="nav-logo">${svg.camera}</a><form action="/search" method="get"><input class="nav-search" type="text" name="q" placeholder="Search" value="${query ? esc(query) : ''}" autocomplete="off"></form><div class="nav-right"><button class="theme-toggle" onclick="T()" title="Toggle theme">${svg.moon}${svg.sun}</button></div></div></div>`
}

export function renderLayout(title: string, content: string, opts: { isHome?: boolean; query?: string; hasCarousel?: boolean; isStory?: boolean } = {}): string {
  const fav = `data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%23e4405f'><rect width='22' height='22' x='1' y='1' rx='6' fill='none' stroke='%23e4405f' stroke-width='2'/><circle cx='12' cy='12' r='5' fill='none' stroke='%23e4405f' stroke-width='2'/><circle cx='18.5' cy='5.5' r='1.5' fill='%23e4405f'/></svg>`

  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(title)}</title>
<meta name="description" content="Insta Viewer - Instagram Viewer">
<link rel="stylesheet" href="${cssURL}">
<link rel="icon" href="${fav}">
${themeScript}
</head>
<body>
${opts.isHome || opts.isStory ? '' : renderNav(opts.query)}
<div class="wrap">${content}</div>
${opts.isHome ? '' : scrollScript}
${opts.hasCarousel ? carouselScript : ''}
</body>
</html>`
}

export function renderError(title: string, message: string): string {
  return renderLayout(title, `<div class="err"><h2>${esc(title)}</h2><p>${esc(message)}</p></div>`)
}
