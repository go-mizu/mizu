export interface Env {
  KV: KVNamespace
  INSTA_SESSION_ID: string
  INSTA_CSRF_TOKEN: string
  INSTA_DS_USER_ID: string
  INSTA_MID: string
  INSTA_IG_DID: string
  INSTA_EMAIL: string
  INSTA_PWD: string
  ENVIRONMENT: string
}

export interface StoredSession {
  sessionId: string
  csrfToken: string
  dsUserId: string
  mid: string
  igDid: string
  loginAt: string
  source: 'login' | 'secrets'
}

export interface LoginError {
  error: string
  errorType: 'challenge_required' | '2fa_required' | 'wrong_password' | 'unknown'
  timestamp: number
  attempts: number
}

export type HonoEnv = { Bindings: Env }

export interface Profile {
  id: string
  username: string
  fullName: string
  biography: string
  profilePicUrl: string
  externalUrl: string
  isPrivate: boolean
  isVerified: boolean
  isBusiness: boolean
  categoryName: string
  followerCount: number
  followingCount: number
  postCount: number
}

export interface Post {
  id: string
  shortcode: string
  typeName: string
  caption: string
  displayUrl: string
  videoUrl: string
  isVideo: boolean
  width: number
  height: number
  likeCount: number
  commentCount: number
  viewCount: number
  takenAt: string
  locationId: string
  locationName: string
  ownerUsername: string
  ownerPic: string
  children: Post[]
}

export interface Comment {
  id: string
  text: string
  authorName: string
  authorPic: string
  likeCount: number
  createdAt: string
  replyCount: number
}

export interface StoryItem {
  id: string
  displayUrl: string
  videoUrl: string
  isVideo: boolean
  width: number
  height: number
  takenAt: string
  expiresAt: string
  ownerUsername: string
}

export interface Highlight {
  id: string
  title: string
  coverUrl: string
  itemCount: number
}

export interface Reel {
  id: string
  shortcode: string
  caption: string
  displayUrl: string
  videoUrl: string
  width: number
  height: number
  likeCount: number
  commentCount: number
  viewCount: number
  playCount: number
  takenAt: string
  ownerUsername: string
}

export interface FollowUser {
  id: string
  username: string
  fullName: string
  isPrivate: boolean
  isVerified: boolean
  picUrl: string
}

export interface SearchResult {
  users: SearchUser[]
  hashtags: SearchHashtag[]
  places: SearchPlace[]
}

export interface SearchUser {
  id: string
  username: string
  fullName: string
  isPrivate: boolean
  isVerified: boolean
  picUrl: string
  followers: number
}

export interface SearchHashtag {
  id: number
  name: string
  mediaCount: number
}

export interface SearchPlace {
  locationId: number
  title: string
  address: string
  city: string
  lat: number
  lng: number
}

export interface ProfileWithPosts {
  profile: Profile
  posts: Post[]
  cursor: string
  hasMore: boolean
}

export interface PostsResult {
  posts: Post[]
  cursor: string
  hasMore: boolean
}

export interface CommentsResult {
  comments: Comment[]
  cursor: string
  hasMore: boolean
}

export interface FollowResult {
  users: FollowUser[]
  cursor: string
  hasMore: boolean
}

export interface ReelsResult {
  reels: Reel[]
  cursor: string
  hasMore: boolean
}
