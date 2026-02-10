export interface Env {
  KV: KVNamespace
  X_AUTH_TOKEN: string
  X_CT0: string
  X_BEARER_TOKEN: string
  ENVIRONMENT: string
}

export type HonoEnv = {
  Bindings: Env
}

export interface Profile {
  id: string
  username: string
  name: string
  biography: string
  avatar: string
  banner: string
  location: string
  website: string
  url: string
  joined: string
  birthday: string
  followersCount: number
  followingCount: number
  tweetsCount: number
  likesCount: number
  mediaCount: number
  listedCount: number
  isPrivate: boolean
  isVerified: boolean
  isBlueVerified: boolean
  verifiedType: string
  pinnedTweetIDs: string[]
  professionalType: string
  professionalCategory: string
}

export interface Tweet {
  id: string
  conversationID: string
  text: string
  username: string
  userID: string
  name: string
  avatar: string
  permanentURL: string
  isRetweet: boolean
  isReply: boolean
  isQuote: boolean
  isPin: boolean
  replyToID: string
  replyToUser: string
  quotedID: string
  retweetedID: string
  retweetedTweet?: Tweet
  quotedTweet?: Tweet
  likes: number
  retweets: number
  replies: number
  views: number
  bookmarks: number
  quotes: number
  photos: string[]
  videos: string[]
  videoThumbnails: string[]
  gifs: string[]
  hashtags: string[]
  mentions: string[]
  urls: string[]
  sensitive: boolean
  language: string
  source: string
  place: string
  isEdited: boolean
  isBlueVerified: boolean
  verifiedType: string
  postedAt: string
}

export interface XList {
  id: string
  name: string
  description: string
  banner: string
  memberCount: number
  ownerID: string
  ownerName: string
}

export interface TimelineResult {
  tweets: Tweet[]
  cursor: string
}
