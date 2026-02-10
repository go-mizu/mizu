import { cssURL } from './asset'
import type { Profile, Tweet } from './types'

// Bold geometric Z logo (same style as X's logo)
const Z = 'M2 2H22V8L8 18H22V22H2V16L16 6H2Z'

// X logo (official path)
const X_LOGO = 'M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-5.214-6.817L4.99 21.75H1.68l7.73-8.835L1.254 2.25H8.08l4.713 6.231zm-1.161 17.52h1.833L7.084 4.126H5.117z'

const svg = {
  logo: `<svg width="32" height="32" viewBox="0 0 24 24"><path d="${Z}"/></svg>`,
  logoBig: `<svg width="56" height="56" viewBox="0 0 24 24"><path d="${Z}"/></svg>`,
  xLogo: `<svg width="24" height="24" viewBox="0 0 24 24" fill="currentColor"><path d="${X_LOGO}"/></svg>`,
  reply: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M1.751 10c0-4.42 3.584-8 8.005-8h4.366c4.49 0 8.129 3.64 8.129 8.13 0 2.96-1.607 5.68-4.196 7.11l-8.054 4.46v-3.69h-.067c-4.49.1-8.183-3.51-8.183-8.01zm8.005-6c-3.317 0-6.005 2.69-6.005 6 0 3.37 2.77 6.08 6.138 6.01l.351-.01h1.761v2.3l5.087-2.81c1.951-1.08 3.163-3.13 3.163-5.36 0-3.39-2.744-6.13-6.129-6.13H9.756z"/></svg>',
  rt: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M4.5 3.88l4.432 4.14-1.364 1.46L5.5 7.55V16c0 1.1.896 2 2 2H13v2H7.5c-2.209 0-4-1.79-4-4V7.55L1.432 9.48.068 8.02 4.5 3.88zM16.5 6H11V4h5.5c2.209 0 4 1.79 4 4v8.45l2.068-1.93 1.364 1.46-4.432 4.14-4.432-4.14 1.364-1.46 2.068 1.93V8c0-1.1-.896-2-2-2z"/></svg>',
  like: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M16.697 5.5c-1.222-.06-2.679.51-3.89 2.16l-.805 1.09-.806-1.09C9.984 6.01 8.526 5.44 7.304 5.5c-1.243.07-2.349.78-2.91 1.91-.552 1.12-.633 2.78.479 4.82 1.074 1.97 3.257 4.27 7.129 6.61 3.87-2.34 6.052-4.64 7.126-6.61 1.111-2.04 1.03-3.7.477-4.82-.561-1.13-1.666-1.84-2.908-1.91zm4.187 7.69c-1.351 2.48-4.001 5.12-8.379 7.67l-.503.3-.504-.3c-4.379-2.55-7.029-5.19-8.382-7.67-1.36-2.5-1.41-4.86-.514-6.67.887-1.79 2.647-2.91 4.601-3.01 1.651-.09 3.368.56 4.798 2.01 1.429-1.45 3.146-2.1 4.796-2.01 1.954.1 3.714 1.22 4.601 3.01.896 1.81.846 4.17-.514 6.67z"/></svg>',
  views: '<svg viewBox="0 0 24 24" fill="currentColor"><path d="M8.75 21V3h2v18h-2zM18.75 21V8.5h2V21h-2zM13.75 21v-9h2v9h-2zM3.75 21v-4h2v4h-2z"/></svg>',
  verified: '<svg width="18" height="18" viewBox="0 0 22 22" fill="#1d9bf0"><path d="M20.396 11c-.018-.646-.215-1.275-.57-1.816-.354-.54-.852-.972-1.438-1.246.223-.607.27-1.264.14-1.897-.131-.634-.437-1.218-.882-1.687-.47-.445-1.053-.75-1.687-.882-.633-.13-1.29-.083-1.897.14-.273-.587-.704-1.086-1.245-1.44S11.647 1.62 11 1.604c-.646.017-1.273.213-1.813.568s-.969.855-1.24 1.44c-.608-.223-1.267-.272-1.902-.14-.635.13-1.22.436-1.69.882-.445.47-.749 1.055-.878 1.69-.13.633-.08 1.29.144 1.896-.587.274-1.087.705-1.443 1.245-.356.54-.555 1.17-.574 1.817.02.647.218 1.276.574 1.817.356.54.856.972 1.443 1.245-.224.606-.274 1.263-.144 1.896.13.636.433 1.221.878 1.69.47.446 1.055.752 1.69.883.635.13 1.294.083 1.902-.144.271.586.702 1.084 1.24 1.438.54.354 1.167.551 1.813.568.647-.016 1.276-.213 1.817-.567s.972-.854 1.245-1.44c.604.224 1.26.272 1.894.141.636-.13 1.22-.435 1.69-.88.445-.47.75-1.054.88-1.69.132-.635.084-1.292-.139-1.899.584-.272 1.084-.705 1.439-1.246.354-.54.551-1.17.569-1.816zM9.662 14.85l-3.429-3.428 1.293-1.302 2.072 2.072 4.4-4.794 1.347 1.246z"/></svg>',
  pin: '<svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path d="M7 4.5C7 3.12 8.12 2 9.5 2h5C15.88 2 17 3.12 17 4.5v5.26L20.12 16H13v5l-1 2-1-2v-5H3.88L7 9.76V4.5z"/></svg>',
  calendar: '<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M7 4V3h2v1h6V3h2v1h1.5C19.89 4 21 5.12 21 6.5v12c0 1.38-1.11 2.5-2.5 2.5h-13C4.12 21 3 19.88 3 18.5v-12C3 5.12 4.12 4 5.5 4H7zm0 2H5.5c-.27 0-.5.22-.5.5v12c0 .28.23.5.5.5h13c.28 0 .5-.22.5-.5v-12c0-.28-.22-.5-.5-.5H17v1h-2V6H9v1H7V6zm-1 4h12v2H6v-2zm0 4h12v2H6v-2z"/></svg>',
  link: '<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M18.36 5.64c-1.95-1.96-5.11-1.96-7.07 0L9.88 7.05 8.46 5.64l1.42-1.42c2.73-2.73 7.16-2.73 9.9 0 2.73 2.74 2.73 7.17 0 9.9l-1.42 1.42-1.41-1.42 1.41-1.41c1.96-1.96 1.96-5.12 0-7.07zm-2.12 3.53l-7.07 7.07-1.41-1.41 7.07-7.07 1.41 1.41zm-5.66 8.49l-1.41 1.41c-2.73 2.74-7.17 2.74-9.9 0-2.73-2.73-2.73-7.16 0-9.9l1.42-1.41 1.41 1.41-1.41 1.42c-1.96 1.95-1.96 5.11 0 7.07 1.95 1.95 5.11 1.95 7.07 0l1.41-1.41 1.41 1.41z"/></svg>',
  location: '<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 7c-1.93 0-3.5 1.57-3.5 3.5S10.07 14 12 14s3.5-1.57 3.5-3.5S13.93 7 12 7zm0 5c-.827 0-1.5-.673-1.5-1.5S11.173 9 12 9s1.5.673 1.5 1.5S12.827 12 12 12zm0-10c-4.687 0-8.5 3.813-8.5 8.5 0 5.967 7.621 11.116 7.945 11.332l.555.37.555-.37c.324-.216 7.945-5.365 7.945-11.332C20.5 5.813 16.687 2 12 2z"/></svg>',
  back: '<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M7.414 13l5.043 5.04-1.414 1.42L3.586 12l7.457-7.46 1.414 1.42L7.414 11H21v2H7.414z"/></svg>',
  moon: '<svg class="icon-moon" width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M21.53 15.93c-1.18.49-2.47.76-3.81.76-5.57 0-10.09-4.55-10.09-10.18 0-1.34.26-2.63.73-3.81C4.48 4.4 1.88 8.09 1.88 12.4 1.88 17.85 6.34 22.25 11.73 22.25c4.26 0 7.91-2.61 9.8-6.32z"/></svg>',
  sun: '<svg class="icon-sun" width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M12 7c-2.76 0-5 2.24-5 5s2.24 5 5 5 5-2.24 5-5-2.24-5-5-5zm0 8c-1.65 0-3-1.35-3-3s1.35-3 3-3 3 1.35 3 3-1.35 3-3 3zm1-13h-2v3h2V2zm0 19h-2v3h2v-3zm9-10v2h-3v-2h3zM5 11v2H2v-2h3zm13.07-6.36l-1.42 1.42-2.12-2.12 1.41-1.42 2.13 2.12zM8.46 19.07l-1.41 1.41-2.12-2.12 1.41-1.42 2.12 2.13zm10.6 1.41l-1.41-1.41-2.12 2.12 1.41 1.42 2.12-2.13zM6.34 6.47L4.93 4.36l2.12-2.12 1.42 1.41L6.34 6.47z"/></svg>',
  lock: '<svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor"><path d="M17.5 7H17v-.25c0-2.76-2.24-5-5-5s-5 2.24-5 5V7h-.5C5.12 7 4 8.12 4 9.5v9C4 19.88 5.12 21 6.5 21h11c1.38 0 2.5-1.12 2.5-2.5v-9C20 8.12 18.88 7 17.5 7zM13 14.73V17h-2v-2.27c-.6-.34-1-.99-1-1.73 0-1.1.9-2 2-2s2 .9 2 2c0 .74-.4 1.39-1 1.73zM15 7H9v-.25c0-1.66 1.34-3 3-3s3 1.34 3 3V7z"/></svg>',
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
  s = s.replace(/#(\w+)/g, '<a href="/search/$1">#$1</a>')
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

function badge(isVerified: boolean, verifiedType?: string): string {
  if (!isVerified) return ''
  let fill = '#1d9bf0'
  if (verifiedType === 'Business') fill = '#e2b719'
  else if (verifiedType === 'Government') fill = '#829aab'
  const icon = `<svg width="18" height="18" viewBox="0 0 22 22" fill="${fill}"><path d="M20.396 11c-.018-.646-.215-1.275-.57-1.816-.354-.54-.852-.972-1.438-1.246.223-.607.27-1.264.14-1.897-.131-.634-.437-1.218-.882-1.687-.47-.445-1.053-.75-1.687-.882-.633-.13-1.29-.083-1.897.14-.273-.587-.704-1.086-1.245-1.44S11.647 1.62 11 1.604c-.646.017-1.273.213-1.813.568s-.969.855-1.24 1.44c-.608-.223-1.267-.272-1.902-.14-.635.13-1.22.436-1.69.882-.445.47-.749 1.055-.878 1.69-.13.633-.08 1.29.144 1.896-.587.274-1.087.705-1.443 1.245-.356.54-.555 1.17-.574 1.817.02.647.218 1.276.574 1.817.356.54.856.972 1.443 1.245-.224.606-.274 1.263-.144 1.896.13.636.433 1.221.878 1.69.47.446 1.055.752 1.69.883.635.13 1.294.083 1.902-.144.271.586.702 1.084 1.24 1.438.54.354 1.167.551 1.813.568.647-.016 1.276-.213 1.817-.567s.972-.854 1.245-1.44c.604.224 1.26.272 1.894.141.636-.13 1.22-.435 1.69-.88.445-.47.75-1.054.88-1.69.132-.635.084-1.292-.139-1.899.584-.272 1.084-.705 1.439-1.246.354-.54.551-1.17.569-1.816zM9.662 14.85l-3.429-3.428 1.293-1.302 2.072 2.072 4.4-4.794 1.347 1.246z"/></svg>`
  return `<span class="tweet-hd-badge">${icon}</span>`
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
  const thumbs = t.videoThumbnails || []
  for (let i = 0; i < t.videos.length; i++) {
    const poster = thumbs[i] ? ` poster="${esc(thumbs[i])}"` : ''
    h += `<div class="tweet-vid"><video src="${esc(t.videos[i])}"${poster} controls preload="none" playsinline></video></div>`
  }
  for (const g of t.gifs) h += `<div class="tweet-vid"><video src="${esc(g)}" autoplay loop muted playsinline></video></div>`
  return h
}

// ---- Quote tweet ----

function renderQt(qt: Tweet): string {
  return `<a href="/${esc(qt.username)}/status/${esc(qt.id)}" class="qt"><div class="qt-hd">${qt.avatar ? `<img class="qt-avi" src="${esc(qt.avatar)}" alt="">` : ''}<span class="qt-name">${esc(qt.name)}</span>${badge(qt.isBlueVerified, qt.verifiedType)}<span class="qt-user">@${esc(qt.username)}</span><span class="tweet-hd-dot">&middot;</span><span class="tweet-hd-time">${relTime(qt.postedAt)}</span></div><div class="qt-txt">${linkify(qt.text, qt.urls)}</div>${qt.photos.length > 0 ? `<div class="qt-media"><img src="${esc(qt.photos[0])}" alt="" loading="lazy"></div>` : ''}</a>`
}

// ---- Tweet card (timeline) ----
// Uses <div> + click overlay to avoid nested <a> tag issues

function renderOneTweet(t: Tweet): string {
  const avi = t.avatar || defaultAvi
  const url = `/${esc(t.username)}/status/${esc(t.id)}`
  const isLong = t.text.length > 280 || (t.text.match(/\n/g)?.length || 0) > 5
  const txtClass = isLong ? 'tweet-txt tweet-txt-long' : 'tweet-txt'
  const showMore = isLong ? `<a href="${url}" class="tweet-show-more">Show more</a>` : ''
  return `<div class="tweet"><a href="${url}" class="tweet-link" aria-label="View tweet"></a><a href="/${esc(t.username)}" class="tweet-avi-link"><img class="tweet-avi" src="${esc(avi)}" alt="" loading="lazy"></a><div class="tweet-body"><div class="tweet-hd"><a href="/${esc(t.username)}" class="tweet-hd-name">${esc(t.name)}</a>${badge(t.isBlueVerified, t.verifiedType)}<span class="tweet-hd-user">@${esc(t.username)}</span><span class="tweet-hd-dot">&middot;</span><span class="tweet-hd-time">${relTime(t.postedAt)}</span></div>${t.isReply && t.replyToUser ? `<div class="tweet-reply">Replying to <a href="/${esc(t.replyToUser)}">@${esc(t.replyToUser)}</a></div>` : ''}<div class="${txtClass}">${linkify(t.text, t.urls)}${showMore}</div>${renderMedia(t)}${t.isQuote && t.quotedTweet ? renderQt(t.quotedTweet) : ''}<div class="tweet-acts"><span class="tweet-act">${svg.reply} ${t.replies > 0 ? fmtNum(t.replies) : ''}</span><span class="tweet-act">${svg.rt} ${t.retweets > 0 ? fmtNum(t.retweets) : ''}</span><span class="tweet-act">${svg.like} ${t.likes > 0 ? fmtNum(t.likes) : ''}</span><span class="tweet-act">${svg.views} ${t.views > 0 ? fmtNum(t.views) : ''}</span></div></div></div>`
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

// ---- Media grid ----

export function renderMediaGrid(tweets: Tweet[]): string {
  let h = '<div class="media-grid">'
  for (const t of tweets) {
    const url = `/${esc(t.username)}/status/${esc(t.id)}`
    if (t.photos.length > 0) {
      h += `<a href="${url}" class="media-grid-item"><img src="${esc(t.photos[0])}" alt="" loading="lazy">${t.photos.length > 1 ? '<span class="mg-multi"></span>' : ''}</a>`
    } else if (t.videos.length > 0) {
      const thumb = (t.videoThumbnails || [])[0] || ''
      h += `<a href="${url}" class="media-grid-item">${thumb ? `<img src="${esc(thumb)}" alt="" loading="lazy">` : '<div class="mg-vid-placeholder"></div>'}<span class="mg-vid">&#9654;</span></a>`
    } else if (t.gifs.length > 0) {
      h += `<a href="${url}" class="media-grid-item"><video src="${esc(t.gifs[0])}" autoplay loop muted playsinline></video><span class="mg-gif">GIF</span></a>`
    }
  }
  h += '</div>'
  return h
}

// ---- Tweet detail (full page) ----

export function renderTweetDetail(tweet: Tweet, replies: Tweet[], cursor?: string, tweetPath?: string): string {
  const avi = tweet.avatar || defaultAvi
  const xURL = `https://x.com/${esc(tweet.username)}/status/${esc(tweet.id)}`
  let h = `<div class="td"><div class="td-top"><a href="/${esc(tweet.username)}"><img src="${esc(avi)}" alt="" loading="lazy"></a><div class="td-info"><a href="/${esc(tweet.username)}" class="td-name">${esc(tweet.name)} ${badge(tweet.isBlueVerified, tweet.verifiedType)}</a><span class="td-user">@${esc(tweet.username)}</span></div><a href="${xURL}" class="td-xlink" target="_blank" rel="noopener" title="View on X">${svg.xLogo}</a></div>${tweet.isReply && tweet.replyToUser ? `<div class="tweet-reply" style="margin-top:12px">Replying to <a href="/${esc(tweet.replyToUser)}">@${esc(tweet.replyToUser)}</a></div>` : ''}<div class="td-text">${linkify(tweet.text, tweet.urls)}</div>${renderMedia(tweet)}${tweet.isQuote && tweet.quotedTweet ? renderQt(tweet.quotedTweet) : ''}<div class="td-time">${fullDate(tweet.postedAt)}</div><div class="td-stats"><span><strong>${fmtNum(tweet.retweets)}</strong> Reposts</span><span><strong>${fmtNum(tweet.quotes)}</strong> Quotes</span><span><strong>${fmtNum(tweet.likes)}</strong> Likes</span><span><strong>${fmtNum(tweet.bookmarks)}</strong> Bookmarks</span>${tweet.views > 0 ? `<span><strong>${fmtNum(tweet.views)}</strong> Views</span>` : ''}</div></div>`
  for (const r of replies) h += renderTweetCard(r)
  if (cursor && tweetPath) h += renderPagination(cursor, tweetPath)
  return h
}

// ---- Profile header ----

export function renderProfileHeader(profile: Profile): string {
  const avi = profile.avatar || defaultAvi
  const v = badge(profile.isBlueVerified || profile.isVerified, profile.verifiedType)
  const joined = profile.joined ? (() => {
    const d = new Date(profile.joined)
    const mo = ['January','February','March','April','May','June','July','August','September','October','November','December']
    return mo[d.getMonth()] + ' ' + d.getFullYear()
  })() : ''
  return `<div class="p-banner">${profile.banner ? `<img src="${esc(profile.banner + '/1500x500')}" alt="">` : ''}</div><div class="p-info"><div class="p-avi"><img src="${esc(avi)}" alt=""></div><div class="p-name">${esc(profile.name)} ${v}${profile.isPrivate ? ` ${svg.lock}` : ''}</div><div class="p-user">@${esc(profile.username)}</div>${profile.biography ? `<div class="p-bio">${linkify(profile.biography, profile.website ? [profile.website] : [])}</div>` : ''}<div class="p-meta">${profile.location ? `<span>${svg.location}${esc(profile.location)}</span>` : ''}${profile.website ? `<span>${svg.link}<a href="${esc(profile.website)}" target="_blank" rel="noopener">${esc(profile.website.replace(/^https?:\/\//, '').slice(0, 30))}</a></span>` : ''}${joined ? `<span>${svg.calendar}Joined ${joined}</span>` : ''}</div><div class="p-stats"><span><strong>${fmtNum(profile.tweetsCount)}</strong> <span>Posts</span></span><a href="/${esc(profile.username)}/following"><strong>${fmtNum(profile.followingCount)}</strong> <span>Following</span></a><a href="/${esc(profile.username)}/followers"><strong>${fmtNum(profile.followersCount)}</strong> <span>Followers</span></a></div><div class="p-orig"><a href="https://x.com/${esc(profile.username)}" target="_blank" rel="noopener">View on x.com ${svg.xLogo}</a></div></div>`
}

// ---- User card (followers/following) ----

export function renderUserCard(u: Profile): string {
  const avi = u.avatar || defaultAvi
  return `<div class="user-card"><a href="/${esc(u.username)}" class="user-card-link" aria-label="View profile"></a><img class="user-card-avi" src="${esc(avi)}" alt="" loading="lazy"><div class="user-card-body"><div class="user-card-hd"><span class="user-card-name">${esc(u.name)}</span>${badge(u.isBlueVerified || u.isVerified, u.verifiedType)}</div><div class="user-card-user">@${esc(u.username)}</div>${u.biography ? `<div class="user-card-bio">${esc(u.biography)}</div>` : ''}</div></div>`
}

// ---- Follow page ----

export function renderFollowPage(profile: Profile, users: Profile[], tab: 'followers' | 'following', cursor: string): string {
  const headerBack = `<div class="sh"><div class="sh-back"><a href="/${esc(profile.username)}">${svg.back}</a><div class="sh-back-info"><strong>${esc(profile.name)}</strong><span>@${esc(profile.username)}</span></div></div></div>`
  const tabs = `<div class="tabs"><a href="/${esc(profile.username)}/followers" class="${tab === 'followers' ? 'active' : ''}">Followers</a><a href="/${esc(profile.username)}/following" class="${tab === 'following' ? 'active' : ''}">Following</a></div>`
  let content = headerBack + tabs
  if (users.length === 0) {
    content += `<div class="err"><p>No ${tab} found.</p></div>`
  } else {
    for (const u of users) content += renderUserCard(u)
  }
  content += renderPagination(cursor, `/${profile.username}/${tab}`)
  return content
}

// ---- Home page ----

export function renderHomePage(): string {
  return `<div class="home"><div class="home-logo">${svg.logoBig}</div><div class="home-sub">the X/Twitter Viewer</div><div class="home-box"><form action="/search" method="get"><input class="home-input" type="text" name="q" placeholder="Search posts or @username" autocomplete="off" autofocus></form><div class="home-hint">Type @username to view a profile</div><div class="home-links"><a href="/karpathy">@karpathy</a><a href="/elonmusk">@elonmusk</a><a href="/search/ai">#ai</a><a href="/search/golang">#golang</a><a href="/openai">@openai</a><a href="/search/typescript">#typescript</a></div></div><div class="home-theme"><button class="theme-toggle" onclick="T()" title="Toggle theme">${svg.moon}${svg.sun}</button></div></div>`
}

// ---- Pagination ----

export function renderPagination(cursor: string, currentPath: string): string {
  if (!cursor) return ''
  const sep = currentPath.includes('?') ? '&' : '?'
  return `<a href="${currentPath}${sep}cursor=${encodeURIComponent(cursor)}" class="more">Show more</a>`
}

// ---- Layout ----

const themeScript = `<script>(function(){var t=localStorage.getItem('t');if(!t)t=matchMedia('(prefers-color-scheme:dark)').matches?'d':'l';document.documentElement.dataset.t=t})();function T(){var h=document.documentElement,n=h.dataset.t==='d'?'l':'d';h.dataset.t=n;localStorage.setItem('t',n)}</script>`

function renderTopBar(query?: string): string {
  return `<div class="topbar"><a href="/" class="topbar-logo">${svg.logo}</a><form action="/search" method="get"><input class="topbar-input" type="text" name="q" placeholder="Search or @username" value="${query ? esc(query) : ''}" autocomplete="off"></form><button class="theme-toggle" onclick="T()" title="Toggle theme">${svg.moon}${svg.sun}</button></div>`
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
${themeScript}
</head>
<body>
<div class="wrap">${opts.isHome ? '' : renderTopBar(opts.query)}${content}</div>
</body>
</html>`
}

export function renderError(title: string, message: string): string {
  return renderLayout(title, `<div class="err"><h2>${esc(title)}</h2><p>${esc(message)}</p></div>`)
}
