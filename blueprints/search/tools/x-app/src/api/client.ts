// API client that calls x-viewer's JSON API (Cloudflare Worker proxy)
// This avoids TLS fingerprinting issues when calling X directly from React Native.

const BASE_URL = 'https://x-viewer.go-mizu.workers.dev/api'

async function apiGet<T>(path: string, params?: Record<string, string>): Promise<T> {
  let url = BASE_URL + path
  if (params) {
    const qs = new URLSearchParams(params).toString()
    if (qs) url += '?' + qs
  }

  const resp = await fetch(url)
  if (!resp.ok) {
    const body = await resp.text()
    throw new Error(`API ${resp.status}: ${body.slice(0, 200)}`)
  }
  return resp.json() as Promise<T>
}

import type { Profile, Tweet, XList } from './types'

export async function fetchProfile(username: string): Promise<Profile> {
  const data = await apiGet<{ profile: Profile }>(`/profile/${username}`)
  return data.profile
}

export async function fetchTweets(
  username: string,
  tab: string,
  cursor?: string
): Promise<{ tweets: Tweet[]; cursor: string }> {
  const params: Record<string, string> = { tab }
  if (cursor) params.cursor = cursor
  return apiGet(`/tweets/${username}`, params)
}

export async function fetchTweet(
  id: string,
  cursor?: string
): Promise<{ tweet: Tweet; replies: Tweet[]; cursor: string }> {
  const params: Record<string, string> = {}
  if (cursor) params.cursor = cursor
  return apiGet(`/tweet/${id}`, params)
}

export async function fetchSearch(
  query: string,
  mode: string,
  cursor?: string
): Promise<{ tweets?: Tweet[]; users?: Profile[]; cursor: string }> {
  const params: Record<string, string> = { q: query, mode }
  if (cursor) params.cursor = cursor
  return apiGet(`/search`, params)
}

export async function fetchFollowers(
  username: string,
  cursor?: string
): Promise<{ users: Profile[]; cursor: string }> {
  const params: Record<string, string> = {}
  if (cursor) params.cursor = cursor
  return apiGet(`/followers/${username}`, params)
}

export async function fetchFollowing(
  username: string,
  cursor?: string
): Promise<{ users: Profile[]; cursor: string }> {
  const params: Record<string, string> = {}
  if (cursor) params.cursor = cursor
  return apiGet(`/following/${username}`, params)
}

export async function fetchList(
  id: string,
  tab: string,
  cursor?: string
): Promise<{ list: XList; tweets?: Tweet[]; users?: Profile[]; cursor: string }> {
  const params: Record<string, string> = { tab }
  if (cursor) params.cursor = cursor
  return apiGet(`/list/${id}`, params)
}
