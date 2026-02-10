// Instagram API base URLs
export const webBaseURL = 'https://www.instagram.com'
export const iPhoneBaseURL = 'https://i.instagram.com'

// Instagram App ID (public, required for all API calls)
export const appID = '936619743392459'

// User agent strings
export const webUserAgent = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/142.0.0.0 Safari/537.36'
export const iPhoneUserAgent = 'Instagram 317.0.0.0.64 (iPad13,8; iPadOS 18_4; en_US; en; scale=2.00; 2048x2732; 562243043) AppleWebKit/420+'

// GraphQL query hashes
export const qhComments = '97b41c52301f77ce508f55e66d17620e'
export const qhCommentReplies = '51fdd02b67508306ad4484ff574a0b62'
export const qhFollowers = '37479f2b8209594dde7facb0d904896a'
export const qhFollowing = '58712303d941c6855d4e888c5f0cd22f'
export const qhPostLikes = '1cb6ec562846122743b61e492c85999f'
export const qhHashtag = '9b498c08113f1e09617a1703c22b2f32'
export const qhTagged = 'e31a871f7301132ceaab56507a66bbb7'
export const qhHighlights = '7c16654f22c819fb63d1183034a5162f'
export const qhLocation = '1b84447a4d8b6d6d0426fefb34514485'

// GraphQL doc IDs (for POST requests)
export const docIdProfilePosts = '7898261790222653'
export const docIdProfilePostsAnon = '7950326061742207'
export const docIdReels = '7845543455542541'
export const docIdPostDetail = '8845758582119845'

// Cache TTLs in seconds — 0 = permanent (no expiration)
// Only stories expire (they disappear after 24h on Instagram)
export const CACHE_PROFILE = 0           // permanent
export const CACHE_POSTS = 0             // permanent
export const CACHE_POST = 0              // permanent (post content doesn't change)
export const CACHE_COMMENTS = 0          // permanent
export const CACHE_SEARCH = 0            // permanent
export const CACHE_HASHTAG = 0           // permanent
export const CACHE_LOCATION = 0          // permanent
export const CACHE_STORIES = 300         // 5 min (stories expire after 24h on Instagram)
export const CACHE_REELS = 0             // permanent
export const CACHE_FOLLOW = 0            // permanent
export const CACHE_HIGHLIGHTS = 0        // permanent
