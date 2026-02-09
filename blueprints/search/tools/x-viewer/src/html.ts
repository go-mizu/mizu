import { cssURL } from './asset'
import type { Profile, Tweet } from './types'

// Bold geometric Z logo (same style as X's logo)
const Z = 'M2 2H22V8L8 18H22V22H2V16L16 6H2Z'

const svg = {
  logo: `<svg width="32" height="32" viewBox="0 0 24 24" fill="#0f1419"><path d="${Z}"/></svg>`,
  logoBig: `<svg width="56" height="56" viewBox="0 0 24 24" fill="#0f1419"><path d="${Z}"/></svg>`,
  reply: '<svg viewBox="0 0 24 24" fill="#536471"><path d="M1.751 10c0-4.42 3.584-8 8.005-8h4.366c4.49 0 8.129 3.64 8.129 8.13 0 2.96-1.607 5.68-4.196 7.11l-8.054 4.46v-3.69h-.067c-4.49.1-8.183-3.51-8.183-8.01zm8.005-6c-3.317 0-6.005 2.69-6.005 6 0 3.37 2.77 6.08 6.138 6.01l.351-.01h1.761v2.3l5.087-2.81c1.951-1.08 3.163-3.13 3.163-5.36 0-3.39-2.744-6.13-6.129-6.13H9.756z"/></svg>',
  rt: '<svg viewBox="0 0 24 24" fill="#536471"><path d="M4.5 3.88l4.432 4.14-1.364 1.46L5.5 7.55V16c0 1.1.896 2 2 2H13v2H7.5c-2.209 0-4-1.79-4-4V7.55L1.432 9.48.068 8.02 4.5 3.88zM16.5 6H11V4h5.5c2.209 0 4 1.79 4 4v8.45l2.068-1.93 1.364 1.46-4.432 4.14-4.432-4.14 1.364-1.46 2.068 1.93V8c0-1.1-.896-2-2-2z"/></svg>',
  like: '<svg viewBox="0 0 24 24" fill="#536471"><path d="M16.697 5.5c-1.222-.06-2.679.51-3.89 2.16l-.805 1.09-.806-1.09C9.984 6.01 8.526 5.44 7.304 5.5c-1.243.07-2.349.78-2.91 1.91-.552 1.12-.633 2.78.479 4.82 1.074 1.97 3.257 4.27 7.129 6.61 3.87-2.34 6.052-4.64 7.126-6.61 1.111-2.04 1.03-3.7.477-4.82-.561-1.13-1.666-1.84-2.908-1.91zm4.187 7.69c-1.351 2.48-4.001 5.12-8.379 7.67l-.503.3-.504-.3c-4.379-2.55-7.029-5.19-8.382-7.67-1.36-2.5-1.41-4.86-.514-6.67.887-1.79 2.647-2.91 4.601-3.01 1.651-.09 3.368.56 4.798 2.01 1.429-1.45 3.146-2.1 4.796-2.01 1.954.1 3.714 1.22 4.601 3.01.896 1.81.846 4.17-.514 6.67z"/></svg>',
  views: '<svg viewBox="0 0 24 24" fill="#536471"><path d="M8.75 21V3h2v18h-2zM18.75 21V8.5h2V21h-2zM13.75 21v-9h2v9h-2zM3.75 21v-4h2v4h-2z"/></svg>',
  verified: '<svg width="18" height="18" viewBox="0 0 22 22" fill="#1d9bf0"><path d="M20.396 11c-.018-.646-.215-1.275-.57-1.816-.354-.54-.852-.972-1.438-1.246.223-.607.27-1.264.14-1.897-.131-.634-.437-1.218-.882-1.687-.47-.445-1.053-.75-1.687-.882-.633-.13-1.29-.083-1.897.14-.273-.587-.704-1.086-1.245-1.44S11.647 1.62 11 1.604c-.646.017-1.273.213-1.813.568s-.969.855-1.24 1.44c-.608-.223-1.267-.272-1.902-.14-.635.13-1.22.436-1.69.882-.445.47-.749 1.055-.878 1.69-.13.633-.08 1.29.144 1.896-.587.274-1.087.705-1.443 1.245-.356.54-.555 1.17-.574 1.817.02.647.218 1.276.574 1.817.356.54.856.972 1.443 1.245-.224.606-.274 1.263-.144 1.896.13.636.433 1.221.878 1.69.47.446 1.055.752 1.69.883.635.13 1.294.083 1.902-.144.271.586.702 1.084 1.24 1.438.54.354 1.167.551 1.813.568.647-.016 1.276-.213 1.817-.567s.972-.854 1.245-1.44c.604.224 1.26.272 1.894.141.636-.13 1.22-.435 1.69-.88.445-.47.75-1.054.88-1.69.132-.635.084-1.292-.139-1.899.584-.272 1.084-.705 1.439-1.246.354-.54.551-1.17.569-1.816zM9.662 14.85l-3.429-3.428 1.293-1.302 2.072 2.072 4.4-4.794 1.347 1.246z"/></svg>',
  pin: '<svg width="16" height="16" viewBox="0 0 24 24" fill="#536471"><path d="M7 4.5C7 3.12 8.12 2 9.5 2h5C15.88 2 17 3.12 17 4.5v5.26L20.12 16H13v5l-1 2-1-2v-5H3.88L7 9.76V4.5z"/></svg>',
  calendar: '<svg width="18" height="18" viewBox="0 0 24 24" fill="#536471"><path d="M7 4V3h2v1h6V3h2v1h1.5C19.89 4 21 5.12 21 6.5v12c0 1.38-1.11 2.5-2.5 2.5h-13C4.12 21 3 19.88 3 18.5v-12C3 5.12 4.12 4 5.5 4H7zm0 2H5.5c-.27 0-.5.22-.5.5v12c0 .28.23.5.5.5h13c.28 0 .5-.22.5-.5v-12c0-.28-.22-.5-.5-.5H17v1h-2V6H9v1H7V6zm-1 4h12v2H6v-2zm0 4h12v2H6v-2z"/></svg>',
  link: '<svg width="18" height="18" viewBox="0 0 24 24" fill="#536471"><path d="M18.36 5.64c-1.95-1.96-5.11-1.96-7.07 0L9.88 7.05 8.46 5.64l1.42-1.42c2.73-2.73 7.16-2.73 9.9 0 2.73 2.74 2.73 7.17 0 9.9l-1.42 1.42-1.41-1.42 1.41-1.41c1.96-1.96 1.96-5.12 0-7.07zm-2.12 3.53l-7.07 7.07-1.41-1.41 7.07-7.07 1.41 1.41zm-5.66 8.49l-1.41 1.41c-2.73 2.74-7.17 2.74-9.9 0-2.73-2.73-2.73-7.16 0-9.9l1.42-1.41 1.41 1.41-1.41 1.42c-1.96 1.95-1.96 5.11 0 7.07 1.95 1.95 5.11 1.95 7.07 0l1.41-1.41 1.41 1.41z"/></svg>',
  location: '<svg width="18" height="18" viewBox="0 0 24 24" fill="#536471"><path d="M12 7c-1.93 0-3.5 1.57-3.5 3.5S10.07 14 12 14s3.5-1.57 3.5-3.5S13.93 7 12 7zm0 5c-.827 0-1.5-.673-1.5-1.5S11.173 9 12 9s1.5.673 1.5 1.5S12.827 12 12 12zm0-10c-4.687 0-8.5 3.813-8.5 8.5 0 5.967 7.621 11.116 7.945 11.332l.555.37.555-.37c.324-.216 7.945-5.365 7.945-11.332C20.5 5.813 16.687 2 12 2z"/></svg>',
  ext: '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" style="vertical-align:-1px;margin-left:4px"><path d="M18 13v6a2 2 0 01-2 2H5a2 2 0 01-2-2V8a2 2 0 012-2h6M15 3h6v6M10 14L21 3"/><path d="M15 3h6v6" fill="none" stroke="currentColor" stroke-width="2"/></svg>',
}

// ---- Helpers ----

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function fmtNum(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1).replace(/\.0$/, '') + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1).replace(/\.0$/, '') + 'K'
  return n.toString()
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
  const mo = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec']
  if (d.getFullYear() === new Date().getFullYear()) return `${mo[d.getMonth()]} ${d.getDate()}`
  return `${mo[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`
}

function fullDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  const h = d.getHours(), m = d.getMinutes().toString().padStart(2, '0')
  const ampm = h >= 12 ? 'PM' : 'AM'
  const mo = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec']
  return `${h % 12 || 12}:${m} ${ampm} &middot; ${mo[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`
}

function linkify(text: string, urls?: string[]): string {
  let s = esc(text)
  s = s.replace(/@(\w+)/g, '<a href="/$1">@$1</a>')
  s = s.replace(/#(\w+)/g, '<a href="/search?q=%23$1">#$1</a>')
  if (urls && urls.length > 0) {
    let i = 0
    s = s.replace(/https?:\/\/t\.co\/\w+/g, () => {
      const u = urls[i] || urls[0]; i++
      return `<a href="${esc(u)}" target="_blank" rel="noopener">${esc(u.replace(/^https?:\/\//, '').slice(0, 40))}</a>`
    })
  } else {
    s = s.replace(/(https?:\/\/[^\s<]+)/g, '<a href="$1" target="_blank" rel="noopener">$1</a>')
  }
  return s
}

function badge(show: boolean): string {
  return show ? `<span class="tweet-hd-badge">${svg.verified}</span>` : ''
}

const defaultAvi = 'https://abs.twimg.com/sticky/default_profile_images/default_profile_normal.png'

// ---- Media ----

function renderMedia(t: Tweet): string {
  let h = ''
  if (t.photos.length === 1) {
    h += `<div class="tweet-media"><img src="${esc(t.photos[0])}" alt="" loading="lazy"></div>`
  } else if (t.photos.length > 1) {
    const n = Math.min(t.photos.length, 4)
    h += `<div class="tweet-grid g${n}">`
    for (let i = 0; i < n; i++) h += `<img src="${esc(t.photos[i])}" alt="" loading="lazy">`
    h += '</div>'
  }
  for (const v of t.videos) h += `<div class="tweet-vid"><video src="${esc(v)}" controls preload="none" playsinline></video></div>`
  for (const g of t.gifs) h += `<div class="tweet-vid"><video src="${esc(g)}" autoplay loop muted playsinline></video></div>`
  return h
}

// ---- Quote tweet ----

function renderQt(qt: Tweet): string {
  return `<a href="/${esc(qt.username)}/status/${esc(qt.id)}" class="qt"><div class="qt-hd">${qt.avatar ? `<img class="qt-avi" src="${esc(qt.avatar)}" alt="">` : ''}<span class="qt-name">${esc(qt.name)}</span>${badge(qt.isBlueVerified)}<span class="qt-user">@${esc(qt.username)}</span><span class="tweet-hd-dot">&middot;</span><span class="tweet-hd-time">${relTime(qt.postedAt)}</span></div><div class="qt-txt">${linkify(qt.text, qt.urls)}</div>${qt.photos.length > 0 ? `<div class="qt-media"><img src="${esc(qt.photos[0])}" alt="" loading="lazy"></div>` : ''}</a>`
}

// ---- Tweet card (timeline) ----
// Uses <div> + click overlay to avoid nested <a> tag issues

function renderOneTweet(t: Tweet): string {
  const avi = t.avatar || defaultAvi
  const url = `/${esc(t.username)}/status/${esc(t.id)}`
  return `<div class="tweet"><a href="${url}" class="tweet-link" aria-label="View tweet"></a><img class="tweet-avi" src="${esc(avi)}" alt="" loading="lazy"><div class="tweet-body"><div class="tweet-hd"><span class="tweet-hd-name">${esc(t.name)}</span>${badge(t.isBlueVerified)}<span class="tweet-hd-user">@${esc(t.username)}</span><span class="tweet-hd-dot">&middot;</span><span class="tweet-hd-time">${relTime(t.postedAt)}</span></div>${t.isReply && t.replyToUser ? `<div class="tweet-reply">Replying to <a href="/${esc(t.replyToUser)}">@${esc(t.replyToUser)}</a></div>` : ''}<div class="tweet-txt">${linkify(t.text, t.urls)}</div>${renderMedia(t)}${t.isQuote && t.quotedTweet ? renderQt(t.quotedTweet) : ''}<div class="tweet-acts"><span class="tweet-act">${svg.reply} ${t.replies > 0 ? fmtNum(t.replies) : ''}</span><span class="tweet-act">${svg.rt} ${t.retweets > 0 ? fmtNum(t.retweets) : ''}</span><span class="tweet-act">${svg.like} ${t.likes > 0 ? fmtNum(t.likes) : ''}</span><span class="tweet-act">${svg.views} ${t.views > 0 ? fmtNum(t.views) : ''}</span></div></div></div>`
}

export function renderTweetCard(tweet: Tweet): string {
  if (tweet.isRetweet && tweet.retweetedTweet) {
    return `<div class="rt-label">${svg.rt} <span>${esc(tweet.name)} reposted</span></div>${renderOneTweet(tweet.retweetedTweet)}`
  }
  if (tweet.isPin) {
    return `<div class="pin-label">${svg.pin} <span>Pinned</span></div>${renderOneTweet(tweet)}`
  }
  return renderOneTweet(tweet)
}

// ---- Tweet detail (full page) ----

export function renderTweetDetail(tweet: Tweet, replies: Tweet[]): string {
  const avi = tweet.avatar || defaultAvi
  let h = `<div class="td"><div class="td-top"><a href="/${esc(tweet.username)}"><img src="${esc(avi)}" alt="" loading="lazy"></a><div class="td-info"><a href="/${esc(tweet.username)}" class="td-name">${esc(tweet.name)} ${badge(tweet.isBlueVerified)}</a><span class="td-user">@${esc(tweet.username)}</span></div></div>${tweet.isReply && tweet.replyToUser ? `<div class="tweet-reply" style="margin-top:12px">Replying to <a href="/${esc(tweet.replyToUser)}">@${esc(tweet.replyToUser)}</a></div>` : ''}<div class="td-text">${linkify(tweet.text, tweet.urls)}</div>${renderMedia(tweet)}${tweet.isQuote && tweet.quotedTweet ? renderQt(tweet.quotedTweet) : ''}<div class="td-time">${fullDate(tweet.postedAt)}</div><div class="td-stats"><span><strong>${fmtNum(tweet.retweets)}</strong> Reposts</span><span><strong>${fmtNum(tweet.quotes)}</strong> Quotes</span><span><strong>${fmtNum(tweet.likes)}</strong> Likes</span><span><strong>${fmtNum(tweet.bookmarks)}</strong> Bookmarks</span>${tweet.views > 0 ? `<span><strong>${fmtNum(tweet.views)}</strong> Views</span>` : ''}</div><div class="td-orig"><a href="https://x.com/${esc(tweet.username)}/status/${esc(tweet.id)}" target="_blank" rel="noopener">View on x.com${svg.ext}</a></div></div>`
  for (const r of replies) h += renderTweetCard(r)
  return h
}

// ---- Profile header ----

export function renderProfileHeader(profile: Profile): string {
  const avi = profile.avatar || defaultAvi
  const v = (profile.isBlueVerified || profile.isVerified) ? svg.verified : ''
  const joined = profile.joined ? (() => {
    const d = new Date(profile.joined)
    const mo = ['January','February','March','April','May','June','July','August','September','October','November','December']
    return mo[d.getMonth()] + ' ' + d.getFullYear()
  })() : ''
  return `<div class="p-banner">${profile.banner ? `<img src="${esc(profile.banner + '/1500x500')}" alt="">` : ''}</div><div class="p-info"><div class="p-avi"><img src="${esc(avi)}" alt=""></div><div class="p-name">${esc(profile.name)} ${v}</div><div class="p-user">@${esc(profile.username)}</div>${profile.biography ? `<div class="p-bio">${linkify(profile.biography, profile.website ? [profile.website] : [])}</div>` : ''}<div class="p-meta">${profile.location ? `<span>${svg.location}${esc(profile.location)}</span>` : ''}${profile.website ? `<span>${svg.link}<a href="${esc(profile.website)}" target="_blank" rel="noopener">${esc(profile.website.replace(/^https?:\/\//, '').slice(0, 30))}</a></span>` : ''}${joined ? `<span>${svg.calendar}Joined ${joined}</span>` : ''}</div><div class="p-stats"><a href="/${esc(profile.username)}/following"><strong>${fmtNum(profile.followingCount)}</strong> <span>Following</span></a><a href="/${esc(profile.username)}/followers"><strong>${fmtNum(profile.followersCount)}</strong> <span>Followers</span></a></div><div class="p-orig"><a href="https://x.com/${esc(profile.username)}" target="_blank" rel="noopener">View on x.com${svg.ext}</a></div></div>`
}

// ---- Home page (Google-style) ----

export function renderHomePage(): string {
  return `<div class="home"><div class="home-logo">${svg.logoBig}</div><div class="home-sub">the X/Twitter Viewer</div><div class="home-box"><form action="/search" method="get"><input class="home-input" type="text" name="q" placeholder="Search posts or @username" autocomplete="off" autofocus></form><div class="home-hint">Type @username to view a profile</div></div></div>`
}

// ---- Pagination ----

export function renderPagination(cursor: string, currentPath: string): string {
  if (!cursor) return ''
  const sep = currentPath.includes('?') ? '&' : '?'
  return `<a href="${currentPath}${sep}cursor=${encodeURIComponent(cursor)}" class="more">Show more</a>`
}

// ---- Layout ----

function renderTopBar(query?: string): string {
  return `<div class="topbar"><a href="/" class="topbar-logo">${svg.logo}</a><form action="/search" method="get"><input class="topbar-input" type="text" name="q" placeholder="Search or @username" value="${query ? esc(query) : ''}" autocomplete="off"></form></div>`
}

export function renderLayout(title: string, content: string, opts: { isHome?: boolean; query?: string } = {}): string {
  const fav = `data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='%230f1419'><path d='${Z}'/></svg>`
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(title)}</title>
<meta name="description" content="Z - the X/Twitter Viewer">
<link rel="stylesheet" href="${cssURL}">
<link rel="icon" href="${fav}">
</head>
<body>
<div class="wrap">${opts.isHome ? '' : renderTopBar(opts.query)}${content}</div>
</body>
</html>`
}

export function renderError(title: string, message: string): string {
  return renderLayout(title, `<div class="err"><h2>${esc(title)}</h2><p>${esc(message)}</p></div>`)
}
