// Goodreads HTML scraper — ported from Go pkg/goodreads/

// ---- Types ----

export interface GoodreadsBook {
  goodreads_id: string
  work_id: string
  url: string
  title: string
  original_title: string
  author_name: string
  author_url: string
  description: string
  isbn: string
  isbn13: string
  asin: string
  page_count: number
  format: string
  publisher: string
  publish_date: string
  first_published: string
  language: string
  edition_language: string
  cover_url: string
  series: string
  edition_count: number
  characters: string[]
  settings: string[]
  literary_awards: string[]
  average_rating: number
  ratings_count: number
  reviews_count: number
  currently_reading: number
  want_to_read: number
  rating_dist: number[] // [5star, 4star, 3star, 2star, 1star]
  genres: string[]
  reviews: GoodreadsReview[]
  quotes: GoodreadsQuote[]
}

export interface GoodreadsReview {
  reviewer_name: string
  rating: number
  date: string
  text: string
  likes_count: number
  shelves: string
  is_spoiler: boolean
}

export interface GoodreadsQuote {
  text: string
  author_name: string
  likes_count: number
}

export interface GoodreadsAuthor {
  goodreads_id: string
  url: string
  name: string
  bio: string
  photo_url: string
  born_date: string
  died_date: string
  works_count: number
  followers: number
  genres: string
  influences: string
  website: string
}

export interface GoodreadsList {
  title: string
  description: string
  voter_count: number
  books: GoodreadsListItem[]
}

export interface GoodreadsListItem {
  goodreads_id: string
  url: string
  position: number
  title: string
  author_name: string
  cover_url: string
  average_rating: number
  ratings_count: number
  score: number
  voters: number
}

export interface GoodreadsListSummary {
  goodreads_id: string
  title: string
  url: string
  book_count: number
  voter_count: number
  tag: string
}

// ---- JSON-LD type ----

interface JsonLD {
  '@type'?: string
  name?: string
  image?: string
  bookFormat?: string
  numberOfPages?: number
  inLanguage?: string
  isbn?: string
  author?: { name: string; url: string }[]
  aggregateRating?: { ratingValue: number; ratingCount: number; reviewCount: number }
}

// ---- Regex Patterns ----

const reJSONLD = /<script\s+type="application\/ld\+json"[^>]*>([\s\S]*?)<\/script>/g
const reCanonicalURL = /<link[^>]*rel="canonical"[^>]*href="([^"]+)"/
const reDescription = /<div[^>]*data-testid="description"[^>]*>([\s\S]*?)<\/div>/
const reDescSpan = /<span[^>]*>([\s\S]*?)<\/span>/g
const reGenre = /<a[^>]*href="\/genres\/[^"]*"[^>]*>([^<]+)<\/a>/g
const reCurrentReading = /([\d,]+)\s*people?\s*(?:are\s+)?currently\s+reading/i
const reWantToRead = /([\d,]+)\s*people?\s*want\s+to\s+read/i
const reRatingBar = /(\d)\s*(?:star|Stars)[^<]*?([\d,]+)/gi
const reSeries = /<a[^>]*href="\/series\/[^"]*"[^>]*>([^<]+)<\/a>/
const reAuthorURL = /<a[^>]*href="(\/author\/show\/[^"]+)"[^>]*>/
const reDetailRow = /<dt[^>]*>\s*([^<]+?)\s*<\/dt>\s*<dd[^>]*>([\s\S]*?)<\/dd>/gis
const reDetailSplit = /\s*(?:[·;,|])\s*/
const rePublisher = /(?:published|publisher)[^<]*?(?:by\s+)?([A-Z][^<\n]{2,80})/i
const reFirstPub = /first\s+published?\s+([^<\n]+?)(?:\)|<)/i
const reASIN = /ASIN[:\s]+([A-Z0-9]{10})/i
const reEditionCount = /([\d,]+)\s+editions?/i
const reCoverImg = /<img[^>]*class="[^"]*ResponsiveImage[^"]*"[^>]*src="([^"]+)"/
const reReviewBlock = /<article[^>]*class="[^"]*ReviewCard[^"]*"[^>]*>([\s\S]*?)<\/article>/gs
const reReviewerName = /class="ReviewerProfile__name"[^>]*><a[^>]*>([^<]+)<\/a>/
const reReviewDate = /<span[^>]*class="[^"]*Text__body3[^"]*"[^>]*>([A-Z][a-z]+\s+\d{1,2},?\s+\d{4})<\/span>/
const reReviewText = /<span[^>]*class="[^"]*Formatted[^"]*"[^>]*>([\s\S]*?)<\/span>/
const reReviewLikes = /(\d+)\s*like/
const reReviewStars = /Rating\s+(\d)\s+out\s+of\s+5/
const reReviewShelf = /shelves?\s*[:\-]\s*([^<]+)/i
const reReviewSpoiler = /contains spoilers|hidden because of spoilers/i
const reRatingDist5 = /5\s*(?:star|Stars)\s*[^0-9]*([\d,]+)/i
const reRatingDist4 = /4\s*(?:star|Stars)\s*[^0-9]*([\d,]+)/i
const reRatingDist3 = /3\s*(?:star|Stars)\s*[^0-9]*([\d,]+)/i
const reRatingDist2 = /2\s*(?:star|Stars)\s*[^0-9]*([\d,]+)/i
const reRatingDist1 = /1\s*(?:star|Stars)\s*[^0-9]*([\d,]+)/i
const reStripTags = /<[^>]*>/g
const reWorkID = /\/work\/(?:quotes\/)?(\d+)/

const reQuoteBlock = /<div[^>]*class="quoteText"[^>]*>([\s\S]*?)<\/div>/g
const reQuoteText = /\u201c([\s\S]*?)\u201d/
const reQuoteTextAlt = /&ldquo;([\s\S]*?)&rdquo;/
const reQuoteAuthor = /class="authorOrTitle"[^>]*>\s*([^<,]+)/
const reQuoteLikes = /(\d+)\s*likes?/
const reSearchBookID = /\/book\/show\/(\d+)/

// Author patterns
const reAuthorName = /<h1[^>]*class="[^"]*authorName[^"]*"[^>]*>\s*<span[^>]*>([^<]+)<\/span>/
const reAuthorNameAlt = /<title>([^(<]+?)(?:\s*\(Author)/
const reAuthorBio = /<div[^>]*class="[^"]*aboutAuthorInfo[^"]*"[^>]*>([\s\S]*?)<\/div>/s
const reAuthorBioSpan = /<span[^>]*>([\s\S]*?)<\/span>/g
const reAuthorPhoto = /<img[^>]*(?:itemprop="image"|class="[^"]*authorPhoto[^"]*")[^>]*src="([^"]+)"/
const reAuthorBorn = /(?:born|Born)\s*(?:in\s+)?([A-Z][a-z]+\s+\d{1,2},\s+\d{4})/i
const reAuthorBornData = /Born\s*<\/dt>\s*<dd[^>]*>([^<]+)/i
const reAuthorDied = /(?:died|Died)\s*(?:in\s+)?([A-Z][a-z]+\s+\d{1,2},\s+\d{4})/i
const reAuthorDiedData = /Died\s*<\/dt>\s*<dd[^>]*>([^<]+)/i
const reAuthorWorks = /(\d[\d,]*)\s*(?:distinct\s+)?works?/
const reAuthorFollowers = /(\d[\d,]*)\s*followers?/
const reAuthorGenre = /<a[^>]*href="\/genres\/[^"]*"[^>]*>([^<]+)<\/a>/g
const reAuthorWebsite = /Website\s*<\/dt>\s*<dd[^>]*>[\s\S]*?href="([^"]+)"/is
const reAuthorInfluence = /Influences?\s*<\/dt>\s*<dd[^>]*>([\s\S]*?)<\/dd>/is
const reAuthorLinkName = />([^<]+)<\/a>/g

// List patterns
const reListTitle = /<h1[^>]*>([^<]+)<\/h1>/
const reListHTMLTitle = /<title>\s*([^<]+)\s*<\/title>/is
const reListDesc = /(?:description|about)[^>]*>\s*(?:<[^>]*>)*\s*"?([^"<]{10,200})"?/i
const reListVoters = /([\d,]+)\s*voters?/g
const reListBook = /<tr[^>]*class="[^"]*bookalike[^"]*"[^>]*>([\s\S]*?)<\/tr>/gs
const reListBookTable = /<tr[^>]*itemscope[^>]*itemtype="http:\/\/schema\.org\/Book"[^>]*>([\s\S]*?)<\/tr>/gs
const reListBookRank = /<td[^>]*class="[^"]*number[^"]*"[^>]*>\s*([0-9]+)\s*<\/td>/s
const reListBookURL = /<a[^>]*class="[^"]*bookTitle[^"]*"[^>]*href="([^"]+)"/
const reListBookTitle = /<a[^>]*class="[^"]*bookTitle[^"]*"[^>]*>[\s\S]*?<span[^>]*>([^<]+)<\/span>/
const reListBookAuthor = /<a[^>]*class="[^"]*authorName[^"]*"[^>]*>[\s\S]*?<span[^>]*>([^<]+)<\/span>/
const reListBookCover = /<img[^>]*src="([^"]+(?:books|compressed)[^"]+)"/
const reListBookRating = /([\d.]+)\s*avg\s*rating/
const reListBookRatings = /([\d,]+)\s*ratings?/
const reListBookScore = /score:\s*([\d,]+)\s*,\s*and\s*([\d,]+)\s*people\s*voted/i
const reBrowseList = /<a[^>]*href="(\/list\/show\/(\d+)[^"]*)"[^>]*>([^<]+)<\/a>/gs
const reBrowseListTitle = /<a[^>]*class="[^"]*listTitle[^"]*"[^>]*href="(\/list\/show\/(\d+)[^"]*)"[^>]*>([^<]+)<\/a>/gs
const reBrowseListInfo = /([\d,]+)\s*books.*?([\d,]+)\s*voters?/s

// ---- Helpers ----

function stripTags(s: string): string {
  return s.replace(reStripTags, '').trim()
}

function parseCommaInt(s: string): number {
  return parseInt(s.replace(/,/g, '').trim(), 10) || 0
}

function normalizeSpace(s: string): string {
  return s.trim().split(/\s+/).join(' ')
}

function unescapeHTML(s: string): string {
  return s
    .replace(/&amp;/g, '&')
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&quot;/g, '"')
    .replace(/&#39;/g, "'")
    .replace(/&apos;/g, "'")
    .replace(/&ldquo;/g, '\u201c')
    .replace(/&rdquo;/g, '\u201d')
    .replace(/&lsquo;/g, '\u2018')
    .replace(/&rsquo;/g, '\u2019')
    .replace(/&mdash;/g, '\u2014')
    .replace(/&ndash;/g, '\u2013')
    .replace(/&middot;/g, '\u00B7')
    .replace(/&#(\d+);/g, (_, n) => String.fromCharCode(parseInt(n, 10)))
    .replace(/&#x([0-9a-fA-F]+);/g, (_, n) => String.fromCharCode(parseInt(n, 16)))
}

function splitDetailList(raw: string): string[] {
  raw = raw.trim()
  if (!raw) return []
  const parts = raw.split(reDetailSplit)
  const seen = new Set<string>()
  const out: string[] = []
  for (const p of parts) {
    const v = p.trim()
    if (!v || seen.has(v)) continue
    seen.add(v)
    out.push(v)
  }
  return out
}

function parseISBNValues(raw: string): [string, string] {
  let digits = ''
  for (const ch of raw.toUpperCase()) {
    if ((ch >= '0' && ch <= '9') || ch === 'X') digits += ch
  }
  if (digits.length >= 23) return [digits.slice(0, 10), digits.slice(-13)]
  if (digits.length >= 13) return ['', digits.slice(0, 13)]
  if (digits.length >= 10) return [digits.slice(0, 10), '']
  return ['', '']
}

function parseASINValue(raw: string): string {
  let out = ''
  for (const ch of raw.toUpperCase()) {
    if ((ch >= '0' && ch <= '9') || (ch >= 'A' && ch <= 'Z')) out += ch
  }
  return out.length >= 10 ? out.slice(0, 10) : out
}

// ---- Book Page Parser ----

function parseBookPage(body: string): GoodreadsBook {
  const book: GoodreadsBook = {
    goodreads_id: '', work_id: '', url: '',
    title: '', original_title: '', author_name: '', author_url: '',
    description: '', isbn: '', isbn13: '', asin: '',
    page_count: 0, format: '', publisher: '', publish_date: '',
    first_published: '', language: '', edition_language: '',
    cover_url: '', series: '', edition_count: 0,
    characters: [], settings: [], literary_awards: [],
    average_rating: 0, ratings_count: 0, reviews_count: 0,
    currently_reading: 0, want_to_read: 0,
    rating_dist: [0, 0, 0, 0, 0],
    genres: [], reviews: [], quotes: [],
  }

  // Canonical URL
  const canonical = reCanonicalURL.exec(body)
  if (canonical) book.url = canonical[1].trim()

  // JSON-LD
  reJSONLD.lastIndex = 0
  let m: RegExpExecArray | null
  while ((m = reJSONLD.exec(body))) {
    try {
      const ld: JsonLD = JSON.parse(m[1])
      if (ld['@type'] !== 'Book') continue
      book.title = ld.name || ''
      book.cover_url = ld.image || ''
      book.format = ld.bookFormat || ''
      book.page_count = ld.numberOfPages || 0
      book.language = ld.inLanguage || ''
      book.isbn13 = ld.isbn || ''
      if (ld.isbn && ld.isbn.length === 10) {
        book.isbn = ld.isbn
        book.isbn13 = ''
      }
      if (ld.aggregateRating) {
        book.average_rating = ld.aggregateRating.ratingValue || 0
        book.ratings_count = ld.aggregateRating.ratingCount || 0
        book.reviews_count = ld.aggregateRating.reviewCount || 0
      }
      if (ld.author && ld.author.length > 0) {
        book.author_name = ld.author.map(a => a.name).join(', ')
        const firstURL = ld.author.find(a => a.url?.trim())?.url?.trim()
        if (firstURL) book.author_url = firstURL
      }
      break
    } catch { /* ignore bad JSON */ }
  }

  // Description
  const descM = reDescription.exec(body)
  if (descM) {
    const content = descM[1]
    reDescSpan.lastIndex = 0
    let longest = ''
    let spanM: RegExpExecArray | null
    while ((spanM = reDescSpan.exec(content))) {
      const text = stripTags(spanM[1])
      if (text.length > longest.length) longest = text
    }
    book.description = (longest || stripTags(content)).trim()
    book.description = unescapeHTML(book.description)
  }

  // Genres
  reGenre.lastIndex = 0
  const seenGenres = new Set<string>()
  while ((m = reGenre.exec(body))) {
    const g = m[1].trim()
    if (g && !seenGenres.has(g)) {
      seenGenres.add(g)
      book.genres.push(g)
    }
  }

  // Stats
  const crM = reCurrentReading.exec(body)
  if (crM) book.currently_reading = parseCommaInt(crM[1])
  const wtrM = reWantToRead.exec(body)
  if (wtrM) book.want_to_read = parseCommaInt(wtrM[1])

  // Rating distribution
  const rd5 = reRatingDist5.exec(body)
  if (rd5) book.rating_dist[0] = parseCommaInt(rd5[1])
  const rd4 = reRatingDist4.exec(body)
  if (rd4) book.rating_dist[1] = parseCommaInt(rd4[1])
  const rd3 = reRatingDist3.exec(body)
  if (rd3) book.rating_dist[2] = parseCommaInt(rd3[1])
  const rd2 = reRatingDist2.exec(body)
  if (rd2) book.rating_dist[3] = parseCommaInt(rd2[1])
  const rd1 = reRatingDist1.exec(body)
  if (rd1) book.rating_dist[4] = parseCommaInt(rd1[1])
  // Fallback: generic pattern
  if (book.rating_dist.reduce((a, b) => a + b, 0) === 0) {
    reRatingBar.lastIndex = 0
    while ((m = reRatingBar.exec(body))) {
      const star = parseInt(m[1], 10)
      const count = parseCommaInt(m[2])
      if (star >= 1 && star <= 5) book.rating_dist[5 - star] = count
    }
  }

  // Series
  const seriesM = reSeries.exec(body)
  if (seriesM) book.series = unescapeHTML(seriesM[1].trim())

  // Metadata from <dt>/<dd> rows
  const details = parseDetailRows(body)
  if (details['original title']) book.original_title = details['original title']
  if (details['edition language']) book.edition_language = details['edition language']
  if (details['published'] && !book.publish_date) book.publish_date = details['published']
  if (details['first published'] && !book.first_published) book.first_published = details['first published']
  if (details['publisher'] && !book.publisher) book.publisher = details['publisher']
  if (details['isbn']) {
    const [isbn10, isbn13] = parseISBNValues(details['isbn'])
    if (!book.isbn && isbn10) book.isbn = isbn10
    if (!book.isbn13 && isbn13) book.isbn13 = isbn13
  }
  if (details['asin'] && !book.asin) book.asin = parseASINValue(details['asin'])
  if (details['setting']) book.settings = splitDetailList(details['setting'])
  if (details['characters']) book.characters = splitDetailList(details['characters'])
  if (details['literary awards']) book.literary_awards = splitDetailList(details['literary awards'])

  // Fallback metadata from body text
  const fpM = reFirstPub.exec(body)
  if (fpM && !book.first_published) book.first_published = fpM[1].trim()
  const asinM = reASIN.exec(body)
  if (asinM) book.asin = asinM[1]
  const pubM = rePublisher.exec(body)
  if (pubM && !book.publisher) book.publisher = pubM[1].trim()

  // Cover image fallback
  if (!book.cover_url) {
    const coverM = reCoverImg.exec(body)
    if (coverM) book.cover_url = coverM[1]
  }

  // Author URL
  if (!book.author_url) {
    const auM = reAuthorURL.exec(body)
    if (auM) {
      book.author_url = auM[1].startsWith('http') ? auM[1].trim() : BASE_URL + auM[1].trim()
    }
  }

  // Edition count
  const edM = reEditionCount.exec(body)
  if (edM) book.edition_count = parseCommaInt(edM[1])

  // Reviews
  reReviewBlock.lastIndex = 0
  let count = 0
  while ((m = reReviewBlock.exec(body)) && count < 30) {
    const content = m[1]
    const review: GoodreadsReview = {
      reviewer_name: '', rating: 0, date: '', text: '',
      likes_count: 0, shelves: '', is_spoiler: false,
    }
    const rnM = reReviewerName.exec(content)
    if (rnM) review.reviewer_name = rnM[1].trim()
    const rsM = reReviewStars.exec(content)
    if (rsM) review.rating = parseInt(rsM[1], 10)
    const rdM = reReviewDate.exec(content)
    if (rdM) review.date = rdM[1].trim()
    const rtM = reReviewText.exec(content)
    if (rtM) review.text = unescapeHTML(stripTags(rtM[1])).trim()
    const rlM = reReviewLikes.exec(content)
    if (rlM) review.likes_count = parseInt(rlM[1], 10)
    const rshM = reReviewShelf.exec(content)
    if (rshM) review.shelves = unescapeHTML(stripTags(rshM[1])).trim()
    review.is_spoiler = reReviewSpoiler.test(content)
    if (review.reviewer_name || review.text || review.rating > 0) {
      book.reviews.push(review)
      count++
    }
  }

  // Work ID
  const widM = reWorkID.exec(body)
  if (widM) book.work_id = widM[1]

  return book
}

function parseDetailRows(body: string): Record<string, string> {
  const out: Record<string, string> = {}
  reDetailRow.lastIndex = 0
  let m: RegExpExecArray | null
  while ((m = reDetailRow.exec(body))) {
    const label = normalizeSpace(stripTags(m[1])).toLowerCase()
    const value = normalizeSpace(unescapeHTML(stripTags(m[2])))
    if (label && value) out[label] = value
  }
  return out
}

// ---- Author Page Parser ----

function parseAuthorPage(body: string): GoodreadsAuthor {
  const a: GoodreadsAuthor = {
    goodreads_id: '', url: '', name: '', bio: '', photo_url: '',
    born_date: '', died_date: '', works_count: 0, followers: 0,
    genres: '', influences: '', website: '',
  }

  // Name
  const nameM = reAuthorName.exec(body)
  if (nameM) a.name = unescapeHTML(nameM[1].trim())
  else {
    const nameAlt = reAuthorNameAlt.exec(body)
    if (nameAlt) a.name = unescapeHTML(nameAlt[1].trim())
  }

  // Bio
  const bioM = reAuthorBio.exec(body)
  if (bioM) {
    const content = bioM[1]
    reAuthorBioSpan.lastIndex = 0
    let longest = ''
    let spanM: RegExpExecArray | null
    while ((spanM = reAuthorBioSpan.exec(content))) {
      const text = stripTags(spanM[1])
      if (text.length > longest.length) longest = text
    }
    a.bio = unescapeHTML((longest || stripTags(content)).trim())
  }

  // Photo
  const photoM = reAuthorPhoto.exec(body)
  if (photoM) a.photo_url = photoM[1]

  // Born date
  const bornD = reAuthorBornData.exec(body)
  if (bornD) a.born_date = bornD[1].trim()
  else {
    const bornM = reAuthorBorn.exec(body)
    if (bornM) a.born_date = bornM[1].trim()
  }

  // Died date
  const diedD = reAuthorDiedData.exec(body)
  if (diedD) a.died_date = diedD[1].trim()
  else {
    const diedM = reAuthorDied.exec(body)
    if (diedM) a.died_date = diedM[1].trim()
  }

  // Works count
  const worksM = reAuthorWorks.exec(body)
  if (worksM) a.works_count = parseCommaInt(worksM[1])

  // Followers
  const followM = reAuthorFollowers.exec(body)
  if (followM) a.followers = parseCommaInt(followM[1])

  // Genres
  reAuthorGenre.lastIndex = 0
  const seenGenres = new Set<string>()
  const genres: string[] = []
  let gm: RegExpExecArray | null
  while ((gm = reAuthorGenre.exec(body))) {
    const g = gm[1].trim()
    if (g && !seenGenres.has(g)) { seenGenres.add(g); genres.push(g) }
  }
  a.genres = genres.join(', ')

  // Website
  const webM = reAuthorWebsite.exec(body)
  if (webM) a.website = webM[1].trim()

  // Influences
  const infM = reAuthorInfluence.exec(body)
  if (infM) {
    const seenInf = new Set<string>()
    const infs: string[] = []
    reAuthorLinkName.lastIndex = 0
    let lnM: RegExpExecArray | null
    while ((lnM = reAuthorLinkName.exec(infM[1]))) {
      const name = unescapeHTML(lnM[1].trim())
      if (name && !seenInf.has(name)) { seenInf.add(name); infs.push(name) }
    }
    a.influences = infs.join(', ')
  }

  return a
}

// ---- Quotes Page Parser ----

function parseQuotesPage(body: string): GoodreadsQuote[] {
  const quotes: GoodreadsQuote[] = []
  reQuoteBlock.lastIndex = 0
  let m: RegExpExecArray | null
  let count = 0
  while ((m = reQuoteBlock.exec(body)) && count < 30) {
    const content = m[1]
    let qt = ''
    const qtM = reQuoteText.exec(content) || reQuoteTextAlt.exec(content)
    if (qtM) qt = unescapeHTML(stripTags(qtM[1])).trim()
    if (!qt) continue

    let author = ''
    const qaM = reQuoteAuthor.exec(content)
    if (qaM) author = qaM[1].trim()

    let likes = 0
    const end = Math.min(m.index + m[0].length + 500, body.length)
    const after = body.slice(m.index + m[0].length, end)
    const qlM = reQuoteLikes.exec(after)
    if (qlM) likes = parseCommaInt(qlM[1])

    quotes.push({ text: qt, author_name: author, likes_count: likes })
    count++
  }
  return quotes
}

// ---- List Page Parser ----

function parseListPage(body: string): GoodreadsList {
  const list: GoodreadsList = { title: '', description: '', voter_count: 0, books: [] }

  // Title
  const titleM = reListTitle.exec(body)
  if (titleM) list.title = unescapeHTML(titleM[1].trim())
  if (!list.title || list.title.toLowerCase() === 'score') {
    const htM = reListHTMLTitle.exec(body)
    if (htM) {
      let t = unescapeHTML(htM[1].trim())
      const pIdx = t.indexOf('(')
      if (pIdx > 0) t = t.slice(0, pIdx).trim()
      if (t) list.title = t
    }
  }

  // Description
  const descM = reListDesc.exec(body)
  if (descM) list.description = unescapeHTML(descM[1].trim())

  // Voter count (take max)
  reListVoters.lastIndex = 0
  let vm: RegExpExecArray | null
  let maxVotes = 0
  while ((vm = reListVoters.exec(body))) {
    const v = parseCommaInt(vm[1])
    if (v > maxVotes) maxVotes = v
  }
  list.voter_count = maxVotes

  // Book entries
  reListBook.lastIndex = 0
  reListBookTable.lastIndex = 0
  let blocks: RegExpExecArray[] = []
  let bm: RegExpExecArray | null
  while ((bm = reListBook.exec(body))) blocks.push(bm)
  if (blocks.length === 0) {
    while ((bm = reListBookTable.exec(body))) blocks.push(bm)
  }

  for (let i = 0; i < blocks.length; i++) {
    const content = blocks[i][1]
    const item: GoodreadsListItem = {
      goodreads_id: '', url: '', position: i + 1,
      title: '', author_name: '', cover_url: '',
      average_rating: 0, ratings_count: 0, score: 0, voters: 0,
    }

    const rankM = reListBookRank.exec(content)
    if (rankM) item.position = parseCommaInt(rankM[1])

    const urlM = reListBookURL.exec(content)
    if (urlM) {
      const url = urlM[1].trim()
      item.url = url.startsWith('/') ? 'https://www.goodreads.com' + url : url
      const idM = reSearchBookID.exec(url)
      if (idM) item.goodreads_id = idM[1]
    }

    const tM = reListBookTitle.exec(content)
    if (tM) item.title = unescapeHTML(tM[1].trim())

    const aM = reListBookAuthor.exec(content)
    if (aM) item.author_name = unescapeHTML(aM[1].trim())

    const cM = reListBookCover.exec(content)
    if (cM) item.cover_url = cM[1]

    const rM = reListBookRating.exec(content)
    if (rM) item.average_rating = parseFloat(rM[1]) || 0

    const rcM = reListBookRatings.exec(content)
    if (rcM) item.ratings_count = parseCommaInt(rcM[1])

    const sM = reListBookScore.exec(content)
    if (sM) { item.score = parseCommaInt(sM[1]); item.voters = parseCommaInt(sM[2]) }

    if (item.title) list.books.push(item)
  }

  return list
}

// ---- List Browse Parser ----

function parseListsBrowse(body: string): GoodreadsListSummary[] {
  const lists: GoodreadsListSummary[] = []

  reBrowseListTitle.lastIndex = 0
  reBrowseList.lastIndex = 0

  let matches: RegExpExecArray[] = []
  let m: RegExpExecArray | null
  while ((m = reBrowseListTitle.exec(body))) matches.push(m)
  if (matches.length === 0) {
    while ((m = reBrowseList.exec(body))) matches.push(m)
  }

  const seen = new Set<string>()
  for (const match of matches) {
    const url = match[1]
    const title = unescapeHTML(match[3].trim())
    if (seen.has(url) || !title) continue
    seen.add(url)

    const entry: GoodreadsListSummary = {
      goodreads_id: match[2],
      title,
      url: 'https://www.goodreads.com' + url,
      book_count: 0,
      voter_count: 0,
      tag: '',
    }

    // Search near anchor for metadata
    const idx = body.indexOf(match[0])
    if (idx >= 0) {
      const end = Math.min(idx + 1200, body.length)
      const snippet = body.slice(idx, end)
      const infoM = reBrowseListInfo.exec(snippet)
      if (infoM) {
        entry.book_count = parseCommaInt(infoM[1])
        entry.voter_count = parseCommaInt(infoM[2])
      }
    }

    lists.push(entry)
  }

  return lists
}

// ---- HTTP Client ----

const BASE_URL = 'https://www.goodreads.com'
const USER_AGENT = 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36'

async function fetchPage(url: string): Promise<string> {
  const resp = await fetch(url, {
    headers: {
      'User-Agent': USER_AGENT,
      'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8',
      'Accept-Language': 'en-US,en;q=0.9',
    },
  })
  if (!resp.ok) throw new Error(`Goodreads returned HTTP ${resp.status}`)
  return await resp.text()
}

// ---- Public API with KV caching ----

async function kvGet<T>(kv: KVNamespace, key: string): Promise<T | null> {
  const val = await kv.get(key)
  if (!val) return null
  return JSON.parse(val) as T
}

async function kvPut(kv: KVNamespace, key: string, data: unknown): Promise<void> {
  await kv.put(key, JSON.stringify(data))
}

export async function getBook(kv: KVNamespace, goodreadsID: string): Promise<GoodreadsBook | null> {
  goodreadsID = goodreadsID.trim()
  if (!goodreadsID) return null

  const key = `gr:book:${goodreadsID}`
  const cached = await kvGet<GoodreadsBook>(kv, key)
  if (cached) return cached

  const url = `${BASE_URL}/book/show/${goodreadsID}`
  const body = await fetchPage(url)
  const book = parseBookPage(body)
  book.goodreads_id = goodreadsID
  if (!book.url) book.url = url

  // Fetch quotes
  if (book.work_id) {
    try {
      const quotes = await getQuotes(kv, book.work_id)
      if (quotes.length > 0) book.quotes = quotes
    } catch { /* ignore */ }
  }

  await kvPut(kv, key, book)
  return book
}

export async function getAuthor(kv: KVNamespace, goodreadsID: string): Promise<GoodreadsAuthor | null> {
  goodreadsID = goodreadsID.trim()
  if (!goodreadsID) return null

  const key = `gr:author:${goodreadsID}`
  const cached = await kvGet<GoodreadsAuthor>(kv, key)
  if (cached) return cached

  const url = `${BASE_URL}/author/show/${goodreadsID}`
  const body = await fetchPage(url)
  const author = parseAuthorPage(body)
  author.goodreads_id = goodreadsID
  author.url = url

  await kvPut(kv, key, author)
  return author
}

export async function getList(kv: KVNamespace, urlOrID: string): Promise<GoodreadsList | null> {
  const id = urlOrID.includes('/') ? (reSearchBookID.exec(urlOrID.replace('/list/show/', '/book/show/'))?.[1] || urlOrID) : urlOrID
  const key = `gr:list:${id}`
  const cached = await kvGet<GoodreadsList>(kv, key)
  if (cached) return cached

  const url = urlOrID.includes('/') ? urlOrID : `${BASE_URL}/list/show/${urlOrID}`
  const body = await fetchPage(url)
  const list = parseListPage(body)

  await kvPut(kv, key, list)
  return list
}

export async function getQuotes(kv: KVNamespace, workID: string): Promise<GoodreadsQuote[]> {
  workID = workID.trim()
  if (!workID) return []

  const key = `gr:quotes:${workID}`
  const cached = await kvGet<GoodreadsQuote[]>(kv, key)
  if (cached) return cached

  const url = `${BASE_URL}/work/quotes/${workID}`
  const body = await fetchPage(url)
  const quotes = parseQuotesPage(body)

  await kvPut(kv, key, quotes)
  return quotes
}

export async function searchBook(kv: KVNamespace, title: string): Promise<string> {
  title = title.trim()
  if (!title) return ''

  const key = `gr:search-id:${title}`
  const cached = await kvGet<string>(kv, key)
  if (cached) return cached

  const url = `${BASE_URL}/search?q=${encodeURIComponent(title)}`
  const body = await fetchPage(url)
  const m = reSearchBookID.exec(body)
  const bookID = m ? m[1] : ''

  if (bookID) await kvPut(kv, key, bookID)
  return bookID
}

export async function searchLists(kv: KVNamespace, query: string): Promise<GoodreadsListSummary[]> {
  query = query.trim().toLowerCase()
  if (!query) return []

  const key = `gr:search-lists:${query}`
  const cached = await kvGet<GoodreadsListSummary[]>(kv, key)
  if (cached) return cached

  // Try Goodreads list tag page (works well for genre/topic searches)
  const tagURL = `${BASE_URL}/list/tag/${query.replace(/\s+/g, '-')}`
  try {
    const body = await fetchPage(tagURL)
    const lists = parseListsBrowse(body)
    if (lists.length > 0) {
      for (const l of lists) l.tag = query
      await kvPut(kv, key, lists)
      return lists
    }
  } catch { /* try next */ }

  // Fallback: Goodreads search with search_type=lists
  const searchURL = `${BASE_URL}/search?q=${encodeURIComponent(query)}&search_type=lists`
  try {
    const body = await fetchPage(searchURL)
    const lists = parseListsBrowse(body)
    if (lists.length > 0) {
      await kvPut(kv, key, lists)
      return lists
    }
  } catch { /* fall through */ }

  // Cache empty result to avoid re-searching
  await kvPut(kv, key, [])
  return []
}

export async function getPopularLists(kv: KVNamespace, tag: string = ''): Promise<GoodreadsListSummary[]> {
  tag = tag.trim().toLowerCase()
  const key = `gr:popular-lists:${tag || 'all'}`
  const cached = await kvGet<GoodreadsListSummary[]>(kv, key)
  if (cached) return cached

  const urls: string[] = []
  if (tag) urls.push(`${BASE_URL}/list/tag/${tag.replace(/\s+/g, '-')}`)
  urls.push(`${BASE_URL}/list/popular_lists`, `${BASE_URL}/list?ref=nav_brws_lists`)

  for (const url of urls) {
    try {
      const body = await fetchPage(url)
      const lists = parseListsBrowse(body)
      if (lists.length > 0) {
        if (tag) for (const l of lists) l.tag = tag
        await kvPut(kv, key, lists)
        return lists
      }
    } catch { continue }
  }

  return []
}

// ---- URL Parsers ----

export function parseGoodreadsURL(input: string): string {
  input = input.trim()
  if (!input.includes('/')) return input.split('.')[0]
  const idx = input.indexOf('/book/show/')
  if (idx >= 0) {
    let path = input.slice(idx + '/book/show/'.length)
    path = path.split('?')[0].split('#')[0].split('.')[0].split('-')[0]
    return path
  }
  return input
}

export function parseGoodreadsAuthorURL(input: string): string {
  input = input.trim()
  if (!input.includes('/')) return input.split('.')[0]
  const idx = input.indexOf('/author/show/')
  if (idx >= 0) {
    let path = input.slice(idx + '/author/show/'.length)
    path = path.split('?')[0].split('#')[0].split('.')[0]
    return path
  }
  return input
}

export function parseGoodreadsListURL(input: string): string {
  input = input.trim()
  if (!input.includes('/')) return input
  const idx = input.indexOf('/list/show/')
  if (idx >= 0) {
    let path = input.slice(idx + '/list/show/'.length)
    path = path.split('?')[0].split('#')[0].split('.')[0]
    return path
  }
  return input
}
