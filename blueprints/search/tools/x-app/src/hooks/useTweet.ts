import { useState, useEffect, useCallback } from 'react'
import type { Tweet } from '../api/types'
import { fetchTweet } from '../api/client'
import { cacheGet, cacheSet, cacheGetStale } from '../cache/store'
import { CACHE_TWEET } from '../api/config'

export function useTweet(tweetID: string) {
  const [tweet, setTweet] = useState<Tweet | null>(null)
  const [replies, setReplies] = useState<Tweet[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [cursor, setCursor] = useState('')

  const load = useCallback(async (loadCursor?: string) => {
    if (!tweetID) return
    if (!loadCursor) setLoading(true)
    setError(null)

    const cacheKey = `tweet:${tweetID}:${loadCursor || ''}`

    try {
      const cached = await cacheGet<{ tweet: Tweet; replies: Tweet[]; cursor: string }>(cacheKey)
      if (cached && !loadCursor) {
        setTweet(cached.tweet)
        setReplies(cached.replies)
        setCursor(cached.cursor)
        setLoading(false)
      }

      const result = await fetchTweet(tweetID, loadCursor)

      if (loadCursor) {
        setReplies(prev => [...prev, ...result.replies])
      } else {
        if (result.tweet) setTweet(result.tweet)
        setReplies(result.replies)
      }
      setCursor(result.cursor)

      await cacheSet(cacheKey, result, CACHE_TWEET)
    } catch (e: any) {
      if (!tweet) {
        const stale = await cacheGetStale<{ tweet: Tweet; replies: Tweet[]; cursor: string }>(cacheKey)
        if (stale) {
          setTweet(stale.tweet)
          setReplies(stale.replies)
          setCursor(stale.cursor)
        } else {
          setError(e.message || 'Failed to load tweet')
        }
      }
    }
    setLoading(false)
  }, [tweetID])

  useEffect(() => { load() }, [load])

  const loadMore = useCallback(() => {
    if (cursor) load(cursor)
  }, [cursor, load])

  return { tweet, replies, loading, error, loadMore, refresh: () => load() }
}
