import { Hono } from 'hono'
import type {
  SearchOptions,
  ImageSearchFilters,
  ImageSearchOptions,
  ImageSize,
  ImageColor,
  ImageType,
  ImageAspect,
  ImageTime,
  ImageRights,
  ImageFileType,
  SafeSearchLevel,
  VideoSearchFilters,
  VideoSearchOptions,
  VideoDuration,
  VideoQuality,
  VideoSort,
  HonoEnv,
} from '../types'

function extractSearchOptions(c: { req: { query: (key: string) => string | undefined } }): SearchOptions {
  return {
    page: parseInt(c.req.query('page') ?? '1', 10),
    per_page: parseInt(c.req.query('per_page') ?? '10', 10),
    time_range: c.req.query('time') ?? '',
    region: c.req.query('region') ?? '',
    language: c.req.query('lang') ?? 'en',
    safe_search: c.req.query('safe') ?? 'moderate',
  }
}

const validImageSizes: ImageSize[] = ['any', 'large', 'medium', 'small', 'icon']
const validImageColors: ImageColor[] = ['any', 'color', 'gray', 'transparent', 'red', 'orange', 'yellow', 'green', 'teal', 'blue', 'purple', 'pink', 'white', 'black', 'brown']
const validImageTypes: ImageType[] = ['any', 'face', 'photo', 'clipart', 'lineart', 'animated']
const validImageAspects: ImageAspect[] = ['any', 'tall', 'square', 'wide', 'panoramic']
const validImageTimes: ImageTime[] = ['any', 'hour', 'day', 'week', 'month', 'year']
const validImageRights: ImageRights[] = ['any', 'creative_commons', 'commercial']
const validImageFileTypes: ImageFileType[] = ['any', 'jpg', 'png', 'gif', 'webp', 'svg', 'bmp', 'ico']
const validSafeSearch: SafeSearchLevel[] = ['off', 'moderate', 'strict']

function extractImageFilters(c: { req: { query: (key: string) => string | undefined } }): ImageSearchFilters {
  const size = c.req.query('size') as ImageSize | undefined
  const color = c.req.query('color') as ImageColor | undefined
  const type = c.req.query('type') as ImageType | undefined
  const aspect = c.req.query('aspect') as ImageAspect | undefined
  const time = c.req.query('time') as ImageTime | undefined
  const rights = c.req.query('rights') as ImageRights | undefined
  const filetype = c.req.query('filetype') as ImageFileType | undefined
  const safe = c.req.query('safe') as SafeSearchLevel | undefined

  const minWidth = c.req.query('min_width')
  const minHeight = c.req.query('min_height')
  const maxWidth = c.req.query('max_width')
  const maxHeight = c.req.query('max_height')

  const filters: ImageSearchFilters = {}

  if (size && validImageSizes.includes(size)) filters.size = size
  if (color && validImageColors.includes(color)) filters.color = color
  if (type && validImageTypes.includes(type)) filters.type = type
  if (aspect && validImageAspects.includes(aspect)) filters.aspect = aspect
  if (time && validImageTimes.includes(time)) filters.time = time
  if (rights && validImageRights.includes(rights)) filters.rights = rights
  if (filetype && validImageFileTypes.includes(filetype)) filters.filetype = filetype
  if (safe && validSafeSearch.includes(safe)) filters.safe = safe

  if (minWidth) {
    const val = parseInt(minWidth, 10)
    if (!isNaN(val) && val > 0) filters.min_width = val
  }
  if (minHeight) {
    const val = parseInt(minHeight, 10)
    if (!isNaN(val) && val > 0) filters.min_height = val
  }
  if (maxWidth) {
    const val = parseInt(maxWidth, 10)
    if (!isNaN(val) && val > 0) filters.max_width = val
  }
  if (maxHeight) {
    const val = parseInt(maxHeight, 10)
    if (!isNaN(val) && val > 0) filters.max_height = val
  }

  return filters
}

function extractImageSearchOptions(c: { req: { query: (key: string) => string | undefined } }): ImageSearchOptions {
  const base = extractSearchOptions(c)
  const filters = extractImageFilters(c)
  return { ...base, filters }
}

const validVideoDurations: VideoDuration[] = ['any', 'short', 'medium', 'long']
const validVideoQualities: VideoQuality[] = ['any', 'hd', '4k']
const validVideoSorts: VideoSort[] = ['relevance', 'date', 'views', 'duration']

function extractVideoFilters(c: { req: { query: (key: string) => string | undefined } }): VideoSearchFilters {
  const duration = c.req.query('duration')
  const quality = c.req.query('quality')
  const time = c.req.query('time')
  const source = c.req.query('source')
  const cc = c.req.query('cc')
  const safe = c.req.query('safe') as SafeSearchLevel | undefined

  const filters: VideoSearchFilters = {}

  if (duration && validVideoDurations.includes(duration as VideoDuration)) {
    filters.duration = duration as VideoDuration
  }
  if (quality && validVideoQualities.includes(quality as VideoQuality)) {
    filters.quality = quality as VideoQuality
  }
  if (time && validImageTimes.includes(time as ImageTime)) {
    filters.time = time as ImageTime
  }
  if (source) {
    filters.source = source
  }
  if (cc === 'true' || cc === '1') {
    filters.cc = true
  }
  if (safe && validSafeSearch.includes(safe)) {
    filters.safe = safe
  }

  return filters
}

function extractVideoSearchOptions(c: { req: { query: (key: string) => string | undefined } }): VideoSearchOptions {
  const base = extractSearchOptions(c)
  const filters = extractVideoFilters(c)
  const sort = c.req.query('sort')

  return {
    ...base,
    filters,
    sort: sort && validVideoSorts.includes(sort as VideoSort) ? sort as VideoSort : undefined,
  }
}

// Services are now injected via contextMiddleware in index.ts
// Access via c.get('services').search

const app = new Hono<HonoEnv>()

app.get('/', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const options = extractSearchOptions(c)
  const searchService = c.get('services')!.search
  const results = await searchService.search(q, options)
  return c.json(results)
})

app.get('/images', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const options = extractImageSearchOptions(c)
  const searchService = c.get('services')!.search
  const results = await searchService.searchImages(q, options)
  return c.json(results)
})

app.post('/images/reverse', async (c) => {
  let body: { url?: string; image_data?: string }
  try {
    body = await c.req.json()
  } catch {
    return c.json({ error: 'Invalid JSON body' }, 400)
  }

  if (!body.url && !body.image_data) {
    return c.json({ error: 'Either url or image_data is required' }, 400)
  }

  const searchService = c.get('services')!.search
  const results = await searchService.reverseImageSearch(body.url, body.image_data)
  return c.json(results)
})

app.get('/videos', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const options = extractVideoSearchOptions(c)
  const searchService = c.get('services')!.search
  const results = await searchService.searchVideos(q, options)
  return c.json(results)
})

app.get('/news', async (c) => {
  const q = c.req.query('q') ?? ''
  if (!q) {
    return c.json({ error: 'Missing required parameter: q' }, 400)
  }

  const options = extractSearchOptions(c)
  const searchService = c.get('services')!.search
  const results = await searchService.searchNews(q, options)
  return c.json(results)
})

export default app
