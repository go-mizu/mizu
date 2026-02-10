import { useState, useCallback, useRef } from 'react'
import type { Tweet } from '../api/types'
import { fetchTweets } from '../api/client'
import { cacheGet, cacheSet, cacheGetStale } from '../cache/store'
import { CACHE_TIMELINE } from '../api/config'
import { useNetwork } from './useNetwork'

export function useTimeline(username: string, tab: string) {
  const [tweets, setTweets] = useState<Tweet[]>([])
  const [loading, setLoading] = useState(false)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const cursorRef = useRef('')
  const hasMoreRef = useRef(true)
  const { isOnline } = useNetwork()

  const fetchPage = useCallback(async (cursor: string, isRefresh = false) => {
    if (!username) return
    if (!isRefresh && !hasMoreRef.current && cursor) return

    if (isRefresh) {
      setRefreshing(true)
    } else {
      setLoading(true)
    }
    setError(null)

    const cacheKey = `tweets:${username}:${tab}:${cursor}`

    try {
      // Try cache
      if (!isRefresh) {
        const cached = await cacheGet<{ tweets: Tweet[]; cursor: string }>(cacheKey)
        if (cached) {
          if (cursor) {
            setTweets(prev => [...prev, ...cached.tweets])
          } else {
            setTweets(cached.tweets)
          }
          cursorRef.current = cached.cursor
          hasMoreRef.current = !!cached.cursor
          setLoading(false)
          setRefreshing(false)
          return
        }
      }

      // When offline, serve stale
      if (!isOnline) {
        const stale = await cacheGetStale<{ tweets: Tweet[]; cursor: string }>(cacheKey)
        if (stale && !cursor) {
          setTweets(stale.tweets)
          cursorRef.current = stale.cursor
          hasMoreRef.current = false // don't paginate offline
        } else if (!cursor) {
          setError('Offline — no cached data available')
        }
        setLoading(false)
        setRefreshing(false)
        return
      }

      const result = await fetchTweets(username, tab, cursor || undefined)

      if (cursor && !isRefresh) {
        setTweets(prev => [...prev, ...result.tweets])
      } else {
        setTweets(result.tweets)
      }
      cursorRef.current = result.cursor
      hasMoreRef.current = !!result.cursor

      await cacheSet(cacheKey, result, CACHE_TIMELINE)
    } catch (e: any) {
      const stale = await cacheGetStale<{ tweets: Tweet[]; cursor: string }>(cacheKey)
      if (stale && !cursor) {
        setTweets(stale.tweets)
        cursorRef.current = stale.cursor
      } else {
        setError(e.message || 'Failed to load timeline')
      }
    }

    setLoading(false)
    setRefreshing(false)
  }, [username, tab, isOnline])

  const loadMore = useCallback(() => {
    if (!loading && hasMoreRef.current && cursorRef.current) {
      fetchPage(cursorRef.current)
    }
  }, [loading, fetchPage])

  const refresh = useCallback(() => {
    cursorRef.current = ''
    hasMoreRef.current = true
    fetchPage('', true)
  }, [fetchPage])

  return { tweets, loading, refreshing, error, loadMore, refresh, fetchInitial: () => fetchPage('') }
}
