import { useState, useCallback, useRef } from 'react'
import type { Profile } from '../api/types'
import { fetchFollowers, fetchFollowing } from '../api/client'
import { cacheGet, cacheSet, cacheGetStale } from '../cache/store'
import { CACHE_FOLLOW } from '../api/config'
import { useNetwork } from './useNetwork'

export function useFollows(username: string, type: 'followers' | 'following') {
  const [users, setUsers] = useState<Profile[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const cursorRef = useRef('')
  const hasMoreRef = useRef(true)
  const { isOnline } = useNetwork()

  const fetcher = type === 'followers' ? fetchFollowers : fetchFollowing

  const fetchPage = useCallback(async (cursor: string, isRefresh = false) => {
    if (!username) return
    setLoading(true)
    setError(null)

    const cacheKey = `${type}:${username}:${cursor}`

    try {
      if (!isRefresh) {
        const cached = await cacheGet<{ users: Profile[]; cursor: string }>(cacheKey)
        if (cached) {
          setUsers(cursor ? prev => [...prev, ...cached.users] : cached.users)
          cursorRef.current = cached.cursor
          hasMoreRef.current = !!cached.cursor
          setLoading(false)
          return
        }
      }

      // When offline, try stale
      if (!isOnline) {
        const stale = await cacheGetStale<{ users: Profile[]; cursor: string }>(cacheKey)
        if (stale && !cursor) {
          setUsers(stale.users)
          hasMoreRef.current = false
        } else if (!cursor) {
          setError('Offline — no cached data available')
        }
        setLoading(false)
        return
      }

      const result = await fetcher(username, cursor || undefined)

      if (cursor && !isRefresh) {
        setUsers(prev => [...prev, ...result.users])
      } else {
        setUsers(result.users)
      }
      cursorRef.current = result.cursor
      hasMoreRef.current = !!result.cursor

      await cacheSet(cacheKey, result, CACHE_FOLLOW)
    } catch (e: any) {
      const stale = await cacheGetStale<{ users: Profile[]; cursor: string }>(cacheKey)
      if (stale && !cursor) {
        setUsers(stale.users)
      } else {
        setError(e.message || `Failed to load ${type}`)
      }
    }
    setLoading(false)
  }, [username, type, fetcher, isOnline])

  const loadMore = useCallback(() => {
    if (!loading && hasMoreRef.current && cursorRef.current) {
      fetchPage(cursorRef.current)
    }
  }, [loading, fetchPage])

  const refresh = useCallback(() => {
    cursorRef.current = ''
    hasMoreRef.current = true
    setUsers([])
    fetchPage('', true)
  }, [fetchPage])

  return { users, loading, error, loadMore, refresh, fetchInitial: () => fetchPage('') }
}
