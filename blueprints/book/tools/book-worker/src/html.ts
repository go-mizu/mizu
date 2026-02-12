import { cssURL } from './asset'
import { icons, renderStars } from './icons'
import type { Book, Author, Shelf, ShelfBook, Review, Quote, FeedItem, ReadingProgress, ReadingChallenge, BookList, BookNote, OLSearchResult, Genre, ReadingStats } from './types'

// ---- Helpers ----

function esc(s: string | null | undefined): string {
  if (!s) return ''
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

function fmtNum(n: number): string {
  if (n >= 1_000_000) return (n / 1_000_000).toFixed(1).replace(/\.0$/, '') + 'M'
  if (n >= 1_000) return (n / 1_000).toFixed(1).replace(/\.0$/, '') + 'K'
  return n.toString()
}

function relTime(iso: string | null | undefined): string {
  if (!iso) return ''
  const d = new Date(iso)
  const sec = Math.floor((Date.now() - d.getTime()) / 1000)
  if (sec < 60) return `${sec}s ago`
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m ago`
  const hr = Math.floor(min / 60)
  if (hr < 24) return `${hr}h ago`
  const days = Math.floor(hr / 24)
  if (days < 30) return `${days}d ago`
  const mo = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
  if (d.getFullYear() === new Date().getFullYear()) return `${mo[d.getMonth()]} ${d.getDate()}`
  return `${mo[d.getMonth()]} ${d.getDate()}, ${d.getFullYear()}`
}

function coverImg(book: Partial<Book>, cls: string = ''): string {
  const clsAttr = cls ? ` class="${cls}"` : ''
  if (book.cover_id && book.cover_id > 0) {
    return `<img src="https://covers.openlibrary.org/b/id/${book.cover_id}-M.jpg" alt=""${clsAttr} loading="lazy">`
  }
  if (book.cover_url) {
    return `<img src="${esc(book.cover_url)}" alt=""${clsAttr} loading="lazy">`
  }
  return `<div class="book-cover-placeholder">${esc(book.title || 'No Cover')}</div>`
}

function pagination(base: string, page: number, total: number, limit: number): string {
  const totalPages = Math.ceil(total / limit)
  if (totalPages <= 1) return ''
  const sep = base.includes('?') ? '&' : '?'
  let h = '<div class="pagination">'
  if (page > 1) h += `<a href="${base}${sep}page=${page - 1}">Prev</a>`
  for (let i = 1; i <= totalPages && i <= 10; i++) {
    if (i === page) h += `<span class="active">${i}</span>`
    else h += `<a href="${base}${sep}page=${i}">${i}</a>`
  }
  if (totalPages > 10 && page < totalPages) h += `<a href="${base}${sep}page=${page + 1}">Next</a>`
  else if (page < totalPages) h += `<a href="${base}${sep}page=${page + 1}">Next</a>`
  h += '</div>'
  return h
}

function authorLinks(authorNames: string): string {
  if (!authorNames) return ''
  return authorNames.split(',').map(a => {
    const name = a.trim()
    return `<a href="/search?q=${encodeURIComponent(name)}">${esc(name)}</a>`
  }).join(', ')
}

// ---- Theme Script ----

const themeScript = `<script>(function(){var t=localStorage.getItem('t');if(!t)t=matchMedia('(prefers-color-scheme:dark)').matches?'d':'l';document.documentElement.dataset.t=t})();function T(){var h=document.documentElement,n=h.dataset.t==='d'?'l':'d';h.dataset.t=n;localStorage.setItem('t',n)}</script>`

// ---- Navigation ----

function renderNav(opts: { currentPath?: string; query?: string } = {}): string {
  const path = opts.currentPath || ''
  const links = [
    { href: '/', label: 'Home' },
    { href: '/shelf', label: 'My Books' },
    { href: '/browse', label: 'Browse' },
    { href: '/lists', label: 'Lists' },
    { href: '/stats', label: 'Stats' },
  ]
  let nav = '<nav class="nav"><div class="nav-inner">'
  nav += `<a href="/" class="nav-logo">${icons.book} Books</a>`
  nav += '<div class="nav-links">'
  for (const l of links) {
    const active = path === l.href || (l.href !== '/' && path.startsWith(l.href)) ? ' active' : ''
    nav += `<a href="${l.href}" class="${active}">${l.label}</a>`
  }
  nav += '</div>'
  nav += `<form class="nav-search" action="/search" method="get"><input type="text" name="q" placeholder="Search books..." value="${esc(opts.query)}" autocomplete="off"></form>`
  nav += `<div class="nav-right"><button class="theme-toggle" onclick="T()" title="Toggle theme">${icons.moon}${icons.sun}</button></div>`
  nav += '</div></nav>'
  return nav
}

// ---- Layout ----

export function renderLayout(title: string, content: string, opts: { isHome?: boolean; currentPath?: string; flash?: string; query?: string } = {}): string {
  const fav = `data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 24 24' fill='none' stroke='%2300635d' stroke-width='2' stroke-linecap='round' stroke-linejoin='round'><path d='M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H20v20H6.5a2.5 2.5 0 0 1 0-5H20'/></svg>`
  let flash = ''
  if (opts.flash) {
    flash = `<div class="flash flash-success">${esc(opts.flash)}</div>`
  }
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>${esc(title)}</title>
<meta name="description" content="Books - track your reading">
<link rel="stylesheet" href="${cssURL}">
<link rel="icon" href="${fav}">
${themeScript}
</head>
<body>
${opts.isHome ? '' : renderNav({ currentPath: opts.currentPath, query: opts.query })}
<div class="wrap"><div class="main${opts.isHome ? '' : ''}">${flash}${content}</div></div>
</body>
</html>`
}

// ---- Book Card ----

export function renderBookCard(book: Book): string {
  return `<a href="/book/${book.id}" class="book-card"><div class="book-cover-wrap">${coverImg(book)}</div><div class="book-card-title">${esc(book.title)}</div><div class="book-card-author">${esc(book.author_names)}</div>${book.average_rating > 0 ? `<div class="book-card-rating">${renderStars(book.average_rating, 12)} <span>${book.average_rating.toFixed(2)}</span></div>` : ''}</a>`
}

// ---- Book Grid ----

export function renderBookGrid(books: Book[]): string {
  if (books.length === 0) return '<div class="empty"><h3>No books found</h3></div>'
  let h = '<div class="book-grid">'
  for (const b of books) h += renderBookCard(b)
  h += '</div>'
  return h
}

// ---- Home Page ----

export function renderHomePage(trending: Book[], currentlyReading: ShelfBook[], feed: FeedItem[], challenge: ReadingChallenge | null): string {
  let h = ''

  // Hero
  h += '<div class="home-hero">'
  h += `<h1 class="home-hero-title">What are you reading?</h1>`
  h += `<p class="home-hero-sub">Track books you've read, want to read, and are currently reading.</p>`
  h += `<div class="home-search"><form action="/search" method="get"><input type="text" name="q" placeholder="Search by title, author, or ISBN..." autocomplete="off" autofocus></form></div>`
  h += '</div>'

  // Currently Reading
  if (currentlyReading.length > 0) {
    h += '<div class="mb-24">'
    h += `<div class="section-header"><h2 class="section-title">Currently Reading</h2><a href="/shelf/currently-reading" class="section-link">View all</a></div>`
    for (const sb of currentlyReading) {
      const b = sb.book
      if (!b) continue
      h += `<div class="currently-reading"><a href="/book/${b.id}">${coverImg(b)}</a><div class="currently-reading-info"><div class="currently-reading-title"><a href="/book/${b.id}">${esc(b.title)}</a></div><div class="currently-reading-author">${esc(b.author_names)}</div><div class="progress-bar"><div class="progress-fill" style="width:0%"></div></div><span style="font-size:12px;color:var(--text-secondary)">Just started</span></div></div>`
    }
    h += '</div>'
  }

  // Challenge widget
  if (challenge) {
    const pct = challenge.goal > 0 ? Math.min(100, Math.round(((challenge.progress || 0) / challenge.goal) * 100)) : 0
    h += '<div class="card mb-24">'
    h += `<div class="section-header"><h2 class="section-title">${challenge.year} Reading Challenge</h2><a href="/challenge" class="section-link">View</a></div>`
    h += `<div class="flex items-center gap-16"><div class="flex-col" style="flex:1"><div class="progress-bar"><div class="progress-fill" style="width:${pct}%"></div></div><div style="font-size:13px;color:var(--text-secondary);margin-top:4px">${challenge.progress || 0} of ${challenge.goal} books (${pct}%)</div></div><div style="font-size:32px;font-weight:700;color:var(--primary)">${challenge.progress || 0}/${challenge.goal}</div></div>`
    h += '</div>'
  }

  // Trending
  if (trending.length > 0) {
    h += '<div class="mb-24">'
    h += `<div class="section-header"><h2 class="section-title">Trending Books</h2><a href="/browse" class="section-link">Browse all</a></div>`
    h += renderBookGrid(trending)
    h += '</div>'
  }

  // Recent Activity
  if (feed.length > 0) {
    h += '<div class="mb-24">'
    h += `<div class="section-header"><h2 class="section-title">Recent Activity</h2></div>`
    for (const item of feed) {
      h += `<div class="feed-item"><div class="feed-icon">${icons.book}</div><div class="feed-text"><div>${renderFeedText(item)}</div><div class="feed-time">${relTime(item.created_at)}</div></div></div>`
    }
    h += '</div>'
  }

  return h
}

function renderFeedText(item: FeedItem): string {
  const bookLink = item.book_id ? `<a href="/book/${item.book_id}">${esc(item.book_title)}</a>` : esc(item.book_title)
  switch (item.type) {
    case 'shelved': return `Added ${bookLink} to a shelf`
    case 'review': return `Reviewed ${bookLink}`
    case 'rating': return `Rated ${bookLink}`
    case 'progress': return `Updated progress on ${bookLink}`
    case 'finished': return `Finished reading ${bookLink}`
    case 'started': return `Started reading ${bookLink}`
    default: return `Activity on ${bookLink}`
  }
}

// ---- Book Detail ----

export function renderBookDetail(
  book: Book,
  reviews: Review[],
  quotes: Quote[],
  similar: Book[],
  progress: ReadingProgress[],
  note: BookNote | null,
  shelves: Shelf[],
  userShelf: string,
  userRating: number
): string {
  let h = '<div class="book-detail">'

  // Left: Cover
  h += `<div><div class="book-cover-large">${coverImg(book)}</div></div>`

  // Right: Info
  h += '<div class="book-info">'
  h += `<h1>${esc(book.title)}</h1>`
  if (book.subtitle) h += `<div class="book-subtitle">${esc(book.subtitle)}</div>`
  if (book.series) h += `<div style="font-size:14px;color:var(--text-secondary);margin-bottom:8px">${esc(book.series)}</div>`
  if (book.author_names) h += `<div class="book-authors">by ${authorLinks(book.author_names)}</div>`

  // Rating
  if (book.average_rating > 0) {
    h += `<div class="book-rating-block">${renderStars(book.average_rating, 20)} <span class="book-rating-big">${book.average_rating.toFixed(2)}</span><span class="book-rating-count">${fmtNum(book.ratings_count)} ratings &middot; ${fmtNum(book.reviews_count || 0)} reviews</span></div>`
  }

  // Shelf actions (form-based PRG)
  h += '<div class="shelf-actions">'
  h += `<form method="POST" action="/book/${book.id}/shelf" style="display:inline-flex;gap:8px;align-items:center">`
  h += `<select name="shelf_id" class="shelf-select">`
  for (const s of shelves) {
    const sel = s.slug === userShelf ? ' selected' : ''
    h += `<option value="${s.id}"${sel}>${esc(s.name)}</option>`
  }
  h += '</select>'
  h += `<button type="submit" class="shelf-btn${userShelf ? ' on-shelf' : ''}">${userShelf ? icons.check + ' On Shelf' : icons.bookmark + ' Want to Read'}</button>`
  h += '</form>'
  if (userShelf) {
    h += `<form method="POST" action="/book/${book.id}/shelf/remove" style="display:inline"><button type="submit" class="btn btn-secondary btn-sm">Remove</button></form>`
  }
  h += '</div>'

  // Star rating input
  h += `<form method="POST" action="/book/${book.id}/rate" class="mb-16">`
  h += `<input type="hidden" name="rating" id="rating-val" value="${userRating}">`
  h += '<div class="star-input" id="star-input">'
  for (let i = 5; i >= 1; i--) {
    const checked = i === userRating ? ' checked' : ''
    h += `<input type="radio" name="star" id="star${i}" value="${i}"${checked}>`
    h += `<label for="star${i}">${icons.starFill}</label>`
  }
  h += '</div>'
  h += `<noscript><button type="submit" class="btn btn-sm btn-secondary mt-8">Rate</button></noscript>`
  h += '</form>'
  h += `<script>document.getElementById('star-input').addEventListener('change',function(e){document.getElementById('rating-val').value=e.target.value;e.target.closest('form').submit()})</script>`

  // Description
  if (book.description) {
    const isLong = book.description.length > 500
    h += `<div class="book-desc${isLong ? ' truncated' : ''}" id="book-desc">${esc(book.description)}</div>`
    if (isLong) {
      h += `<button onclick="document.getElementById('book-desc').classList.remove('truncated');this.remove()" style="font-size:13px;color:var(--primary);margin-bottom:16px;cursor:pointer;background:none;border:none;padding:0">Show more</button>`
    }
  }

  // Metadata
  h += '<dl class="book-meta">'
  if (book.page_count) h += `<div><dt>Pages</dt><dd>${book.page_count}</dd></div>`
  if (book.publisher) h += `<div><dt>Publisher</dt><dd>${esc(book.publisher)}</dd></div>`
  if (book.publish_date) h += `<div><dt>Published</dt><dd>${esc(book.publish_date)}</dd></div>`
  else if (book.publish_year) h += `<div><dt>Published</dt><dd>${book.publish_year}</dd></div>`
  if (book.isbn13) h += `<div><dt>ISBN</dt><dd>${esc(book.isbn13)}</dd></div>`
  else if (book.isbn10) h += `<div><dt>ISBN</dt><dd>${esc(book.isbn10)}</dd></div>`
  if (book.language) h += `<div><dt>Language</dt><dd>${esc(book.language)}</dd></div>`
  if (book.format) h += `<div><dt>Format</dt><dd>${esc(book.format)}</dd></div>`
  if (book.editions_count) h += `<div><dt>Editions</dt><dd>${book.editions_count}</dd></div>`
  h += '</dl>'

  // Tags
  const tags: { label: string; items: string[] }[] = []
  if (book.subjects && book.subjects.length > 0) tags.push({ label: 'Subjects', items: book.subjects })
  if (book.characters && book.characters.length > 0) tags.push({ label: 'Characters', items: book.characters })
  if (book.settings && book.settings.length > 0) tags.push({ label: 'Settings', items: book.settings })
  if (book.literary_awards && book.literary_awards.length > 0) tags.push({ label: 'Awards', items: book.literary_awards })
  for (const group of tags) {
    h += `<div class="mb-8" style="font-size:13px;color:var(--text-secondary);font-weight:600">${group.label}</div>`
    h += '<div class="tags">'
    for (const item of group.items.slice(0, 15)) {
      if (group.label === 'Subjects') {
        h += `<a href="/genre/${encodeURIComponent(item)}" class="tag">${esc(item)}</a>`
      } else {
        h += `<span class="tag">${esc(item)}</span>`
      }
    }
    h += '</div>'
  }

  // External links
  if (book.goodreads_url || book.asin) {
    h += '<div class="flex gap-12 mb-16" style="font-size:13px">'
    if (book.goodreads_url) h += `<a href="${esc(book.goodreads_url)}" target="_blank" rel="noopener">Goodreads ${icons.external}</a>`
    if (book.asin) h += `<a href="https://www.amazon.com/dp/${esc(book.asin)}" target="_blank" rel="noopener">Amazon ${icons.external}</a>`
    h += '</div>'
  }

  h += '</div></div>' // close book-info, book-detail

  // Rating Distribution
  if (book.rating_distribution && book.rating_distribution.length === 5) {
    const maxCount = Math.max(...book.rating_distribution, 1)
    h += '<div class="card mb-24">'
    h += `<div class="card-title">Rating Distribution</div>`
    h += '<div class="rating-dist">'
    for (let i = 4; i >= 0; i--) {
      const count = book.rating_distribution[i]
      const pct = Math.round((count / maxCount) * 100)
      h += `<div class="rating-row"><span class="rating-row-label">${i + 1}</span><div class="rating-row-bar"><div class="rating-row-fill" style="width:${pct}%"></div></div><span class="rating-row-count">${fmtNum(count)}</span></div>`
    }
    h += '</div></div>'
  }

  // Reading Progress
  h += '<div class="card mb-24">'
  h += `<div class="card-title">Reading Progress</div>`
  h += `<form method="POST" action="/book/${book.id}/progress" class="flex gap-8 items-center flex-wrap">`
  h += `<div class="form-group" style="margin:0"><input type="number" name="page" class="form-input" placeholder="Page #" style="width:100px" min="0" max="${book.page_count || 9999}"></div>`
  h += `<div class="form-group" style="margin:0"><input type="number" name="percent" class="form-input" placeholder="%" style="width:80px" min="0" max="100"></div>`
  h += `<div class="form-group" style="margin:0;flex:1"><input type="text" name="note" class="form-input" placeholder="Note (optional)"></div>`
  h += `<button type="submit" class="btn btn-primary">Update</button>`
  h += '</form>'
  if (progress.length > 0) {
    h += '<div class="mt-16">'
    for (const p of progress.slice(0, 5)) {
      const pct = p.percent || (book.page_count ? Math.round((p.page / book.page_count) * 100) : 0)
      h += `<div style="font-size:13px;padding:6px 0;border-bottom:1px solid var(--border-light)">`
      h += `<div class="flex items-center justify-between"><span>${p.page ? `Page ${p.page}` : ''}${p.page && pct ? ' - ' : ''}${pct ? `${pct}%` : ''}</span><span style="color:var(--text-secondary)">${relTime(p.created_at)}</span></div>`
      if (p.note) h += `<div style="color:var(--text-secondary);margin-top:2px">${esc(p.note)}</div>`
      h += '</div>'
    }
    h += '</div>'
  }
  h += '</div>'

  // Notes
  h += '<div class="card mb-24">'
  h += `<div class="card-title">My Notes</div>`
  h += `<form method="POST" action="/book/${book.id}/notes">`
  h += `<div class="form-group"><textarea name="text" class="form-input" placeholder="Write your notes about this book...">${esc(note?.text)}</textarea></div>`
  h += `<div class="flex gap-8">`
  h += `<button type="submit" class="btn btn-primary">Save Note</button>`
  if (note) h += `<button type="submit" formaction="/book/${book.id}/notes/delete" class="btn btn-secondary">Delete</button>`
  h += '</div></form></div>'

  // Reviews
  h += '<div class="card mb-24">'
  h += `<div class="section-header"><h2 class="card-title" style="margin:0">Reviews</h2></div>`
  if (reviews.length === 0) {
    h += '<div class="empty" style="padding:24px 0"><p>No reviews yet.</p></div>'
  }
  for (const r of reviews) {
    h += renderReviewCard(r)
  }
  h += `<div class="mt-16"><a href="/book/${book.id}/review" class="btn btn-secondary">Write a Review</a></div>`
  h += '</div>'

  // Quotes
  if (quotes.length > 0) {
    h += '<div class="card mb-24">'
    h += `<div class="card-title">Quotes</div>`
    for (const q of quotes) {
      h += `<div class="quote-card"><div class="quote-text">"${esc(q.text)}"</div><div class="quote-attr">&mdash; ${esc(q.author_name)}${q.likes_count ? ` &middot; ${icons.heart} ${q.likes_count}` : ''}</div></div>`
    }
    h += '</div>'
  }

  // Similar Books
  if (similar.length > 0) {
    h += '<div class="mb-24">'
    h += `<div class="section-header"><h2 class="section-title">Readers Also Enjoyed</h2></div>`
    h += renderBookGrid(similar)
    h += '</div>'
  }

  return h
}

function renderReviewCard(r: Review): string {
  let h = '<div class="review-card">'
  h += '<div class="review-header">'
  if (r.reviewer_name) h += `<span class="review-name">${esc(r.reviewer_name)}</span>`
  if (r.rating > 0) h += renderStars(r.rating, 14)
  h += `<span class="review-date">${relTime(r.created_at)}</span>`
  h += '</div>'
  if (r.is_spoiler) {
    h += `<div class="review-spoiler" onclick="this.nextElementSibling.style.display='block';this.remove()">This review contains spoilers. Click to reveal.</div><div style="display:none">`
  }
  if (r.text) h += `<div class="review-text">${esc(r.text)}</div>`
  if (r.is_spoiler) h += '</div>'
  h += `<div class="review-actions"><form method="POST" action="/api/reviews/${r.id}/like" style="display:inline"><button type="submit">${icons.heart} ${r.likes_count || 0}</button></form><span>${icons.message} ${r.comments_count || 0}</span></div>`
  h += '</div>'
  return h
}

// ---- Search Results ----

export function renderSearchResults(query: string, localBooks: Book[], olResults: OLSearchResult[], page: number, total: number): string {
  let h = ''
  h += `<h1 class="page-title">Results for "${esc(query)}"</h1>`

  // Local results
  if (localBooks.length > 0) {
    h += `<div class="section-header"><h2 class="section-title">In Your Library</h2></div>`
    h += renderBookGrid(localBooks)
    h += '<div class="mb-24"></div>'
  }

  // Open Library results
  if (olResults.length > 0) {
    h += `<div class="section-header"><h2 class="section-title">Open Library</h2></div>`
    for (const r of olResults) {
      h += '<div class="book-list-row">'
      h += `<div class="book-list-cover">`
      if (r.cover_i) {
        h += `<img src="https://covers.openlibrary.org/b/id/${r.cover_i}-M.jpg" alt="" loading="lazy">`
      } else {
        h += `<div class="book-cover-placeholder" style="aspect-ratio:2/3;width:50px;font-size:10px">${esc(r.title)}</div>`
      }
      h += '</div>'
      h += '<div class="book-list-info">'
      h += `<div class="book-list-title">${esc(r.title)}</div>`
      if (r.author_name && r.author_name.length > 0) {
        h += `<div class="book-list-author">${esc(r.author_name.join(', '))}</div>`
      }
      h += '<div class="book-list-meta">'
      if (r.first_publish_year) h += `${r.first_publish_year} &middot; `
      if (r.edition_count) h += `${r.edition_count} editions &middot; `
      if (r.ratings_average) h += `${r.ratings_average.toFixed(1)} avg rating`
      h += '</div></div>'
      h += `<form method="POST" action="/import/ol-import"><input type="hidden" name="ol_key" value="${esc(r.key)}"><input type="hidden" name="q" value="${esc(query)}"><button type="submit" class="btn btn-primary btn-sm">${icons.plus} Import</button></form>`
      h += '</div>'
    }
  }

  if (localBooks.length === 0 && olResults.length === 0) {
    h += `<div class="empty"><h3>No results found</h3><p>Try a different search term or browse by genre.</p></div>`
  }

  h += pagination(`/search?q=${encodeURIComponent(query)}`, page, total, 20)
  return h
}

// ---- Author Page ----

export function renderAuthorPage(author: Author, books: Book[], page: number, total: number): string {
  let h = '<div class="author-header">'

  // Photo
  h += '<div class="author-photo">'
  if (author.photo_url) {
    h += `<img src="${esc(author.photo_url)}" alt="${esc(author.name)}">`
  } else {
    h += `<div style="width:100%;height:100%;display:flex;align-items:center;justify-content:center;color:var(--text-secondary)">${icons.user}</div>`
  }
  h += '</div>'

  // Info
  h += '<div class="author-info">'
  h += `<h1>${esc(author.name)}</h1>`
  h += '<div class="author-meta">'
  if (author.birth_date) {
    h += `<span>Born: ${esc(author.birth_date)}</span>`
  }
  if (author.death_date) {
    h += `<span>Died: ${esc(author.death_date)}</span>`
  }
  if (author.works_count) {
    h += `<span>${author.works_count} works</span>`
  }
  if (author.followers) {
    h += `<span>${fmtNum(author.followers)} followers</span>`
  }
  h += '</div>'

  if (author.genres) {
    h += '<div class="tags mt-8">'
    for (const g of author.genres.split(',').slice(0, 10)) {
      h += `<a href="/genre/${encodeURIComponent(g.trim())}" class="tag">${esc(g.trim())}</a>`
    }
    h += '</div>'
  }

  if (author.influences) {
    h += `<div style="font-size:13px;color:var(--text-secondary);margin-top:8px">Influences: ${esc(author.influences)}</div>`
  }

  if (author.website) {
    h += `<div style="margin-top:8px;font-size:14px"><a href="${esc(author.website)}" target="_blank" rel="noopener">${esc(author.website)} ${icons.external}</a></div>`
  }

  if (author.bio) {
    h += `<div class="author-bio">${esc(author.bio)}</div>`
  }
  h += '</div></div>' // close author-info, author-header

  // Books
  h += `<div class="section-header"><h2 class="section-title">Books by ${esc(author.name)}</h2></div>`
  h += renderBookGrid(books)
  h += pagination(`/author/${author.id}`, page, total, 20)

  return h
}

// ---- Shelf Page (My Books) ----

export function renderShelfPage(shelves: Shelf[], currentShelf: Shelf | null, books: ShelfBook[], page: number, total: number): string {
  const currentSlug = currentShelf?.slug || 'all'
  let h = `<h1 class="page-title">My Books</h1>`
  h += '<div class="shelf-layout">'

  // Sidebar
  h += '<div class="shelf-sidebar">'
  const allCount = shelves.reduce((sum, s) => sum + (s.book_count || 0), 0)
  h += `<a href="/shelf" class="shelf-sidebar-item${currentSlug === 'all' ? ' active' : ''}"><span>All</span><span class="shelf-sidebar-count">${allCount}</span></a>`
  for (const s of shelves) {
    h += `<a href="/shelf/${esc(s.slug)}" class="shelf-sidebar-item${s.slug === currentSlug ? ' active' : ''}"><span>${esc(s.name)}</span><span class="shelf-sidebar-count">${s.book_count || 0}</span></a>`
  }
  h += `<div style="margin-top:12px;padding-top:12px;border-top:1px solid var(--border-light)">`
  h += `<form method="POST" action="/shelf/create" class="flex gap-8"><input type="text" name="name" class="form-input" placeholder="New shelf..." style="font-size:12px;padding:4px 8px" required><button type="submit" class="btn btn-sm btn-secondary">${icons.plus}</button></form>`
  h += '</div></div>'

  // Book list
  h += '<div>'

  // Sort form
  h += `<form method="GET" action="/shelf/${esc(currentSlug)}" class="flex items-center justify-between mb-16">`
  h += `<span style="font-size:14px;color:var(--text-secondary)">${total} book${total !== 1 ? 's' : ''}</span>`
  h += '<select name="sort" class="shelf-select" onchange="this.form.submit()">'
  h += '<option value="date_added">Date Added</option><option value="title">Title</option><option value="rating">Rating</option><option value="date_read">Date Read</option>'
  h += '</select></form>'

  if (books.length === 0) {
    h += `<div class="empty"><h3>No books on this shelf</h3><p>Search for books to add them to your shelves.</p></div>`
  }
  for (const sb of books) {
    const b = sb.book
    if (!b) continue
    h += '<div class="book-list-row">'
    h += `<div class="book-list-cover"><a href="/book/${b.id}">${coverImg(b)}</a></div>`
    h += '<div class="book-list-info">'
    h += `<div class="book-list-title"><a href="/book/${b.id}">${esc(b.title)}</a></div>`
    h += `<div class="book-list-author">${esc(b.author_names)}</div>`
    h += `<div class="book-list-meta">`
    if (b.average_rating > 0) h += `${renderStars(b.average_rating, 12)} ${b.average_rating.toFixed(2)} &middot; `
    h += `Added ${relTime(sb.date_added)}`
    if (sb.date_read) h += ` &middot; Read ${relTime(sb.date_read)}`
    h += '</div></div></div>'
  }
  h += pagination(`/shelf/${esc(currentSlug)}`, page, total, 20)
  h += '</div></div>' // close book list, shelf-layout

  return h
}

// ---- Browse Page ----

export function renderBrowsePage(genres: { name: string; book_count: number }[], popular: Book[], newReleases: Book[]): string {
  let h = `<h1 class="page-title">Browse</h1>`

  // Genres
  h += `<div class="section-header"><h2 class="section-title">Genres</h2></div>`
  h += '<div class="genre-grid mb-24">'
  for (const g of genres) {
    h += `<a href="/genre/${encodeURIComponent(g.name)}" class="genre-tile"><span class="genre-tile-name">${esc(g.name)}</span><span class="genre-tile-count">${fmtNum(g.book_count)}</span></a>`
  }
  h += '</div>'

  // Popular
  if (popular.length > 0) {
    h += `<div class="section-header"><h2 class="section-title">Popular</h2></div>`
    h += renderBookGrid(popular)
    h += '<div class="mb-24"></div>'
  }

  // New Releases
  if (newReleases.length > 0) {
    h += `<div class="section-header"><h2 class="section-title">New Releases</h2></div>`
    h += renderBookGrid(newReleases)
  }

  return h
}

// ---- Genre Page ----

export function renderGenrePage(genre: string, books: Book[], page: number, total: number): string {
  let h = `<h1 class="page-title">${esc(genre)}</h1>`
  h += `<p class="page-sub">${fmtNum(total)} books</p>`
  h += renderBookGrid(books)
  h += pagination(`/genre/${encodeURIComponent(genre)}`, page, total, 20)
  return h
}

// ---- Lists Page ----

export function renderListsPage(lists: BookList[], page: number, total: number): string {
  let h = `<h1 class="page-title">Book Lists</h1>`

  // Create list form
  h += '<div class="card mb-24">'
  h += `<div class="card-title">Create a New List</div>`
  h += `<form method="POST" action="/lists">`
  h += `<div class="form-group"><label class="form-label">Title</label><input type="text" name="title" class="form-input" placeholder="My Best Books of 2024..." required></div>`
  h += `<div class="form-group"><label class="form-label">Description</label><textarea name="description" class="form-input" placeholder="What's this list about?"></textarea></div>`
  h += `<button type="submit" class="btn btn-primary">Create List</button>`
  h += '</form></div>'

  if (lists.length === 0) {
    h += `<div class="empty"><h3>No lists yet</h3><p>Create your first book list above.</p></div>`
  }
  for (const list of lists) {
    h += `<a href="/list/${list.id}" class="list-card"><div class="list-card-info"><div class="list-card-title">${esc(list.title)}</div><div class="list-card-meta">${list.item_count || 0} books &middot; ${list.voter_count || 0} voters &middot; ${relTime(list.created_at)}</div>${list.description ? `<div style="font-size:13px;color:var(--text-secondary);margin-top:4px">${esc(list.description.slice(0, 120))}</div>` : ''}</div>${icons.list}</a>`
  }
  h += pagination('/lists', page, total, 20)
  return h
}

// ---- List Detail Page ----

export function renderListDetailPage(list: BookList): string {
  let h = `<h1 class="page-title">${esc(list.title)}</h1>`
  if (list.description) h += `<p class="page-sub" style="margin-top:-12px;margin-bottom:20px">${esc(list.description)}</p>`
  h += `<div style="font-size:14px;color:var(--text-secondary);margin-bottom:20px">${list.item_count || 0} books &middot; ${list.voter_count || 0} voters</div>`

  if (list.goodreads_url) {
    h += `<div class="mb-16"><a href="${esc(list.goodreads_url)}" target="_blank" rel="noopener" style="font-size:13px">View on Goodreads ${icons.external}</a></div>`
  }

  if (!list.items || list.items.length === 0) {
    h += `<div class="empty"><h3>This list is empty</h3></div>`
    return h
  }

  for (let i = 0; i < list.items.length; i++) {
    const item = list.items[i]
    const b = item.book
    if (!b) continue
    h += '<div class="book-list-row">'
    h += `<span style="font-size:16px;font-weight:700;color:var(--text-secondary);width:28px;text-align:right;flex-shrink:0">${i + 1}</span>`
    h += `<div class="book-list-cover"><a href="/book/${b.id}">${coverImg(b)}</a></div>`
    h += '<div class="book-list-info">'
    h += `<div class="book-list-title"><a href="/book/${b.id}">${esc(b.title)}</a></div>`
    if (b.author_names) h += `<div class="book-list-author">${esc(b.author_names)}</div>`
    if (b.average_rating > 0) h += `<div class="book-list-meta">${renderStars(b.average_rating, 12)} ${b.average_rating.toFixed(2)} &middot; ${fmtNum(b.ratings_count)} ratings</div>`
    h += '</div>'
    h += `<form method="POST" action="/list/${list.id}/vote/${b.id}"><button type="submit" class="list-vote">${icons.heart} ${item.votes || 0}</button></form>`
    h += '</div>'
  }

  return h
}

// ---- Quotes Page ----

export function renderQuotesPage(quotes: Quote[], page: number, total: number): string {
  let h = `<h1 class="page-title">Quotes</h1>`

  if (quotes.length === 0) {
    h += `<div class="empty"><h3>No quotes yet</h3><p>Quotes added from book pages will appear here.</p></div>`
    return h
  }

  for (const q of quotes) {
    h += '<div class="quote-card">'
    h += `<div class="quote-text">"${esc(q.text)}"</div>`
    h += `<div class="quote-attr">&mdash; ${esc(q.author_name)}`
    if (q.book) h += `, <a href="/book/${q.book.id}">${esc((q.book as any).title)}</a>`
    h += `${q.likes_count ? ` &middot; ${icons.heart} ${q.likes_count}` : ''}</div>`
    h += '</div>'
  }

  h += pagination('/quotes', page, total, 20)
  return h
}

// ---- Stats Page ----

export function renderStatsPage(stats: ReadingStats, year?: number): string {
  const thisYear = new Date().getFullYear()
  let h = `<h1 class="page-title">Reading Statistics</h1>`

  // Year selector
  h += '<div class="flex items-center gap-8 mb-24">'
  h += `<form method="GET" action="/stats" class="flex items-center gap-8">`
  h += '<label class="form-label" style="margin:0">Year:</label>'
  h += '<select name="year" class="shelf-select" onchange="this.form.submit()">'
  h += `<option value="">All Time</option>`
  for (let y = thisYear; y >= thisYear - 5; y--) {
    h += `<option value="${y}"${year === y ? ' selected' : ''}>${y}</option>`
  }
  h += '</select></form></div>'

  // Stats cards
  h += '<div class="stats-grid">'
  h += `<div class="stat-card"><div class="stat-value">${stats.total_books}</div><div class="stat-label">Books Read</div></div>`
  h += `<div class="stat-card"><div class="stat-value">${fmtNum(stats.total_pages)}</div><div class="stat-label">Pages Read</div></div>`
  h += `<div class="stat-card"><div class="stat-value">${stats.average_rating > 0 ? stats.average_rating.toFixed(1) : '-'}</div><div class="stat-label">Average Rating</div></div>`
  h += '</div>'

  if (stats.total_books === 0) {
    h += `<div class="empty"><h3>No reading data</h3><p>Mark books as read to see your statistics.</p></div>`
    return h
  }

  // Rating distribution
  if (stats.rating_distribution) {
    const maxVal = Math.max(...Object.values(stats.rating_distribution), 1)
    h += '<div class="card mb-24">'
    h += '<div class="card-title">Your Ratings</div>'
    h += '<div class="bar-chart">'
    for (let i = 5; i >= 1; i--) {
      const count = stats.rating_distribution[i] || 0
      const pct = Math.round((count / maxVal) * 100)
      h += `<div class="bar-row"><span class="bar-label">${i} star${i > 1 ? 's' : ''}</span><div class="bar-track"><div class="bar-fill" style="width:${pct}%"></div></div><span class="bar-value">${count}</span></div>`
    }
    h += '</div></div>'
  }

  // Genre breakdown
  if (stats.genre_breakdown && Object.keys(stats.genre_breakdown).length > 0) {
    const sorted = Object.entries(stats.genre_breakdown).sort((a, b) => b[1] - a[1]).slice(0, 10)
    const maxGenre = sorted[0]?.[1] || 1
    h += '<div class="card mb-24">'
    h += '<div class="card-title">Genre Breakdown</div>'
    h += '<div class="bar-chart">'
    for (const [genre, count] of sorted) {
      const pct = Math.round((count / maxGenre) * 100)
      h += `<div class="bar-row"><span class="bar-label" style="width:100px" title="${esc(genre)}">${esc(genre.length > 12 ? genre.slice(0, 11) + '...' : genre)}</span><div class="bar-track"><div class="bar-fill" style="width:${pct}%"></div></div><span class="bar-value">${count}</span></div>`
    }
    h += '</div></div>'
  }

  // Notable books
  const notable: { label: string; book: Book | null | undefined }[] = [
    { label: 'Shortest Book', book: stats.shortest_book },
    { label: 'Longest Book', book: stats.longest_book },
    { label: 'Highest Rated', book: stats.highest_rated },
    { label: 'Most Popular', book: stats.most_popular },
  ]
  const hasNotable = notable.some(n => n.book)
  if (hasNotable) {
    h += '<div class="card mb-24">'
    h += '<div class="card-title">Notable Books</div>'
    h += '<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px">'
    for (const n of notable) {
      if (!n.book) continue
      h += `<div class="flex gap-12"><div style="width:40px;flex-shrink:0;border-radius:3px;overflow:hidden">${coverImg(n.book)}</div><div><div style="font-size:12px;color:var(--text-secondary);font-weight:600">${n.label}</div><div style="font-size:14px;font-weight:600"><a href="/book/${n.book.id}">${esc(n.book.title)}</a></div><div style="font-size:12px;color:var(--text-secondary)">${n.book.page_count ? n.book.page_count + ' pages' : ''}</div></div></div>`
    }
    h += '</div></div>'
  }

  return h
}

// ---- Challenge Page ----

export function renderChallengePage(challenge: ReadingChallenge | null): string {
  const thisYear = new Date().getFullYear()
  let h = `<div class="main-narrow">`
  h += `<h1 class="page-title">${thisYear} Reading Challenge</h1>`

  if (challenge) {
    const pct = challenge.goal > 0 ? Math.min(100, Math.round(((challenge.progress || 0) / challenge.goal) * 100)) : 0
    h += '<div class="challenge-box">'
    h += `${icons.target}`
    h += `<div class="challenge-year">${challenge.year} Reading Challenge</div>`
    h += `<div class="challenge-progress">${challenge.progress || 0} / ${challenge.goal}</div>`
    h += `<div class="challenge-goal">books read</div>`
    h += `<div class="progress-bar" style="max-width:300px;margin:16px auto"><div class="progress-fill" style="width:${pct}%"></div></div>`
    h += `<div style="font-size:14px;color:var(--text-secondary)">${pct}% complete</div>`
    h += '</div>'

    // Update form
    h += '<div class="card mt-24">'
    h += '<div class="card-title">Update Goal</div>'
    h += `<form method="POST" action="/challenge" class="flex gap-8 items-center">`
    h += `<input type="hidden" name="year" value="${thisYear}">`
    h += `<div class="form-group" style="margin:0"><input type="number" name="goal" class="form-input" value="${challenge.goal}" min="1" style="width:100px"></div>`
    h += `<button type="submit" class="btn btn-primary">Update</button>`
    h += '</form></div>'
  } else {
    // Create form
    h += '<div class="challenge-box">'
    h += `${icons.target}`
    h += `<div class="challenge-year" style="margin-top:8px">Set your ${thisYear} reading goal</div>`
    h += `<form method="POST" action="/challenge" class="flex gap-8 items-center justify-center mt-16">`
    h += `<input type="hidden" name="year" value="${thisYear}">`
    h += `<span style="font-size:16px">I want to read</span>`
    h += `<input type="number" name="goal" class="form-input" value="12" min="1" style="width:80px;text-align:center;font-size:18px;font-weight:700">`
    h += `<span style="font-size:16px">books in ${thisYear}</span>`
    h += `</form>`
    h += `<button type="submit" form="" onclick="this.previousElementSibling.submit()" class="btn btn-primary mt-16">Start Challenge</button>`
    h += '</div>'
  }

  h += '</div>'
  return h
}

// ---- Import Page ----

export function renderImportPage(): string {
  let h = `<div class="main-narrow">`
  h += `<h1 class="page-title">Import & Export</h1>`

  // Import from Open Library
  h += '<div class="import-section">'
  h += `<h2 class="card-title">${icons.download} Import from Open Library</h2>`
  h += `<p style="font-size:14px;color:var(--text-secondary);margin-bottom:16px">Import a book by its Open Library work key (e.g. /works/OL45883W)</p>`
  h += `<form method="POST" action="/import/ol-import" class="flex gap-8">`
  h += `<input type="text" name="ol_key" class="form-input" placeholder="/works/OL45883W" required>`
  h += `<button type="submit" class="btn btn-primary">Import</button>`
  h += '</form></div>'

  // Import from ISBN
  h += '<div class="import-section">'
  h += `<h2 class="card-title">${icons.download} Import by ISBN</h2>`
  h += `<p style="font-size:14px;color:var(--text-secondary);margin-bottom:16px">Look up a book by its ISBN-10 or ISBN-13.</p>`
  h += `<form method="POST" action="/import/ol-import" class="flex gap-8">`
  h += `<input type="text" name="isbn" class="form-input" placeholder="978-0-14-028329-7" required>`
  h += `<button type="submit" class="btn btn-primary">Import</button>`
  h += '</form></div>'

  // Export
  h += '<div class="import-section">'
  h += `<h2 class="card-title">${icons.upload} Export Data</h2>`
  h += `<p style="font-size:14px;color:var(--text-secondary);margin-bottom:16px">Download your library data as JSON.</p>`
  h += `<a href="/api/export/csv" class="btn btn-secondary">${icons.download} Export Library</a>`
  h += '</div>'

  h += '</div>'
  return h
}

// ---- Error Page ----

export function renderError(title: string, message: string): string {
  return `<div class="error"><h2>${esc(title)}</h2><p>${esc(message)}</p><div class="mt-16"><a href="/" class="btn btn-primary">Go Home</a></div></div>`
}
