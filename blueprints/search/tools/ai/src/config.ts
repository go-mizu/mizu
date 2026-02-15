export const BASE_URL = 'https://www.perplexity.ai'
export const API_VERSION = '2.18'

export const ENDPOINTS = {
  session: `${BASE_URL}/api/auth/session`,
  csrf: `${BASE_URL}/api/auth/csrf`,
  signin: `${BASE_URL}/api/auth/signin/email`,
  sseAsk: `${BASE_URL}/rest/sse/perplexity_ask`,
} as const

export const CHROME_UA = 'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36'

export const CHROME_HEADERS: Record<string, string> = {
  'Accept': 'text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7',
  'Accept-Language': 'en-US,en;q=0.9',
  'Cache-Control': 'max-age=0',
  'Dnt': '1',
  'Priority': 'u=0, i',
  'Sec-Ch-Ua': '"Not;A=Brand";v="24", "Chromium";v="128"',
  'Sec-Ch-Ua-Arch': '"x86"',
  'Sec-Ch-Ua-Bitness': '"64"',
  'Sec-Ch-Ua-Full-Version': '"128.0.6613.120"',
  'Sec-Ch-Ua-Full-Version-List': '"Not;A=Brand";v="24.0.0.0", "Chromium";v="128.0.6613.120"',
  'Sec-Ch-Ua-Mobile': '?0',
  'Sec-Ch-Ua-Model': '""',
  'Sec-Ch-Ua-Platform': '"Windows"',
  'Sec-Ch-Ua-Platform-Version': '"19.0.0"',
  'Sec-Fetch-Dest': 'document',
  'Sec-Fetch-Mode': 'navigate',
  'Sec-Fetch-Site': 'same-origin',
  'Sec-Fetch-User': '?1',
  'Upgrade-Insecure-Requests': '1',
  'User-Agent': CHROME_UA,
}

// Mode → API mode field
export const MODE_PAYLOAD: Record<string, string> = {
  auto: 'concise',
  pro: 'copilot',
  reasoning: 'copilot',
  deep: 'copilot',
}

// (mode, model) → model_preference field
export const MODEL_PREFERENCE: Record<string, Record<string, string>> = {
  auto: { '': 'turbo' },
  pro: {
    '': 'pplx_pro',
    'sonar': 'experimental',
    'gpt-5.2': 'gpt52',
    'claude-4.5-sonnet': 'claude45sonnet',
    'grok-4.1': 'grok41nonreasoning',
  },
  reasoning: {
    '': 'pplx_reasoning',
    'gpt-5.2-thinking': 'gpt52_thinking',
    'claude-4.5-sonnet-thinking': 'claude45sonnetthinking',
    'gemini-3.0-pro': 'gemini30pro',
    'kimi-k2-thinking': 'kimik2thinking',
    'grok-4.1-reasoning': 'grok41reasoning',
  },
  deep: { '': 'pplx_alpha' },
}

export const MODELS = [
  { id: 'auto', name: 'Auto', desc: 'Quick answers', mode: 'auto', model: '' },
  { id: 'pro', name: 'Pro', desc: 'Advanced search', mode: 'pro', model: '' },
  { id: 'reasoning', name: 'Reasoning', desc: 'Step-by-step', mode: 'reasoning', model: '' },
  { id: 'deep', name: 'Deep Research', desc: 'Comprehensive', mode: 'deep', model: '' },
] as const

export const DEFAULT_MODE = 'auto'

export const CACHE_TTL = {
  session: 3600,      // 1h
  thread: 2592000,    // 30d
  threadIndex: 0,     // permanent
  account: 2592000,   // 30d
} as const

export const MAX_THREADS = 100
export const THREAD_ID_LEN = 8

export const MAGIC_LINK_REGEX = /"(https:\/\/www\.perplexity\.ai\/api\/auth\/callback\/email\?callbackUrl=.*?)"/
export const SIGNIN_SUBJECT = 'Sign in to Perplexity'
