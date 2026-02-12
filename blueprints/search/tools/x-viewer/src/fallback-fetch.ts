import { GraphQLClient } from './graphql'
import {
  parseUserResult, parseTimeline, parseSearchTweets, parseSearchUsers,
} from './parse'
import {
  gqlUserByScreenName, gqlUserTweetsV2, gqlUserTweetsAndRepliesV2, gqlUserMedia,
  gqlSearchTimeline, userFieldToggles, userTweetsFieldToggles,
  SearchMedia,
} from './config'
import {
  fetchProfileFromNitter, fetchUserTweetsFromNitter,
  fetchSearchTweetsFromNitter, fetchSearchUsersFromNitter, isNitterCursor,
} from './nitter'
import { isRateLimitedError } from './rate-limit'
import type { Env, Profile, TimelineResult } from './types'

function gqlClient(env: Env): GraphQLClient {
  return new GraphQLClient(env.X_AUTH_TOKEN, env.X_CT0, env.X_BEARER_TOKEN)
}

export async function fetchProfileWithFallback(env: Env, username: string): Promise<Profile | null> {
  try {
    const data = await gqlClient(env).doGraphQL(gqlUserByScreenName, {
      screen_name: username,
      withSafetyModeUserFields: true,
      withSuperFollowsUserFields: true,
    }, userFieldToggles)
    return parseUserResult(data)
  } catch (e) {
    if (!isRateLimitedError(e)) throw e
    return fetchProfileFromNitter(username, env.NITTER_INSTANCES)
  }
}

export async function fetchUserTimelineWithFallback(
  env: Env,
  username: string,
  tab: string,
  cursor: string,
  userID: string
): Promise<TimelineResult> {
  if (!userID) {
    return fetchUserTweetsFromNitter(username, tab, cursor, env.NITTER_INSTANCES)
  }
  if (isNitterCursor(cursor)) {
    try {
      return await fetchUserTweetsFromNitter(username, tab, cursor, env.NITTER_INSTANCES)
    } catch {
      // Nitter cursor became stale/unavailable; continue with first-page GraphQL.
      cursor = ''
    }
  }

  let endpoint = gqlUserTweetsV2
  let toggles = userTweetsFieldToggles
  if (tab === 'media') {
    endpoint = gqlUserMedia
    toggles = ''
  } else if (tab === 'replies') {
    endpoint = gqlUserTweetsAndRepliesV2
  }

  const vars: Record<string, unknown> = {
    userId: userID,
    count: 40,
    includePromotedContent: false,
    withQuickPromoteEligibilityTweetFields: true,
    withVoice: true,
  }
  if (cursor) vars.cursor = cursor

  try {
    const data = await gqlClient(env).doGraphQL(endpoint, vars, toggles)
    return parseTimeline(data)
  } catch (e) {
    if (!isRateLimitedError(e)) throw e
    return fetchUserTweetsFromNitter(username, tab, cursor, env.NITTER_INSTANCES)
  }
}

export async function fetchSearchTweetsWithFallback(
  env: Env,
  query: string,
  mode: string,
  cursor: string
): Promise<TimelineResult> {
  if (isNitterCursor(cursor)) {
    try {
      return await fetchSearchTweetsFromNitter(query, mode, cursor, env.NITTER_INSTANCES)
    } catch {
      cursor = ''
    }
  }

  const apiProduct = mode === SearchMedia ? 'Photos' : mode
  const vars: Record<string, unknown> = {
    rawQuery: query,
    count: 40,
    querySource: 'typed_query',
    product: apiProduct,
  }
  if (cursor) vars.cursor = cursor

  try {
    const data = await gqlClient(env).doGraphQL(gqlSearchTimeline, vars, '')
    return parseSearchTweets(data)
  } catch (e) {
    if (!isRateLimitedError(e)) throw e
    return fetchSearchTweetsFromNitter(query, mode, cursor, env.NITTER_INSTANCES)
  }
}

export async function fetchSearchUsersWithFallback(
  env: Env,
  query: string,
  cursor: string
): Promise<{ users: Profile[]; cursor: string }> {
  if (isNitterCursor(cursor)) {
    try {
      return await fetchSearchUsersFromNitter(query, cursor, env.NITTER_INSTANCES)
    } catch {
      cursor = ''
    }
  }

  const vars: Record<string, unknown> = {
    rawQuery: query,
    count: 40,
    querySource: 'typed_query',
    product: 'People',
  }
  if (cursor) vars.cursor = cursor

  try {
    const data = await gqlClient(env).doGraphQL(gqlSearchTimeline, vars, '')
    return parseSearchUsers(data)
  } catch (e) {
    if (!isRateLimitedError(e)) throw e
    return fetchSearchUsersFromNitter(query, cursor, env.NITTER_INSTANCES)
  }
}
