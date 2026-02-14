import { GraphQLClient } from './graphql'
import { parseConversation } from './parse'
import { fetchTweetFromNitter, isNitterCursor, type TweetConversation } from './nitter'
import { gqlConversationTimeline, tweetDetailFieldToggles } from './config'
import type { Env } from './types'
import { isRateLimitedError } from './rate-limit'

export async function fetchTweetConversation(
  env: Env,
  tweetID: string,
  cursor = '',
  includePromotedContent = false
): Promise<TweetConversation> {
  const forceNitter = isNitterCursor(cursor)
  if (forceNitter) {
    try {
      return await fetchTweetFromNitter(tweetID, env.NITTER_INSTANCES, cursor)
    } catch {
      // If nitter cursor stops working, fall back to first page from GraphQL.
      cursor = ''
    }
  }

  const gql = new GraphQLClient(env.X_AUTH_TOKEN, env.X_CT0, env.X_BEARER_TOKEN)
  const vars: Record<string, unknown> = {
    focalTweetId: tweetID,
    referrer: 'tweet',
    with_rux_injections: false,
    rankingMode: 'Relevance',
    includePromotedContent,
    withCommunity: true,
    withQuickPromoteEligibilityTweetFields: true,
    withBirdwatchNotes: true,
    withVoice: true,
    withV2Timeline: true,
  }
  if (cursor) vars.cursor = cursor

  try {
    const data = await gql.doGraphQL(gqlConversationTimeline, vars, tweetDetailFieldToggles)
    return parseConversation(data, tweetID)
  } catch (e) {
    if (!isRateLimitedError(e)) throw e
    return fetchTweetFromNitter(tweetID, env.NITTER_INSTANCES, cursor)
  }
}
