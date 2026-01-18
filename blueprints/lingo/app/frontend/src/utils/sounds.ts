// Sound effects for Duolingo-style exercise interactions
// Using Web Audio API for reliable, embedded sounds

// Generate simple beep sounds using Web Audio API
function createBeep(frequency: number, duration: number, type: OscillatorType = 'sine'): string {
  // Return empty string - we'll use Web Audio API directly
  return `beep:${frequency}:${duration}:${type}`
}

// Sound configurations (frequency, duration, wave type)
const SOUNDS = {
  correct: createBeep(880, 150, 'sine'),      // High pleasant tone
  wrong: createBeep(220, 200, 'square'),      // Low buzzer
  complete: createBeep(523, 300, 'sine'),     // Celebratory
  click: createBeep(1000, 50, 'sine'),        // Quick click
  select: createBeep(600, 80, 'sine'),        // Selection
  deselect: createBeep(400, 60, 'sine'),      // Deselection
  levelUp: createBeep(784, 400, 'sine'),      // Achievement
  streak: createBeep(659, 350, 'sine'),       // Streak
  match: createBeep(698, 120, 'sine'),        // Match found
  keypress: createBeep(800, 30, 'sine'),      // Subtle keypress
  milestone: createBeep(740, 250, 'sine'),    // Milestone
  xpGain: createBeep(587, 150, 'sine'),       // XP gain
  heartLost: createBeep(196, 300, 'sawtooth'), // Heart lost
  notification: createBeep(523, 200, 'sine'), // Notification
} as const

export type SoundType = keyof typeof SOUNDS

// Web Audio API context (lazy initialized)
let audioContext: AudioContext | null = null

function getAudioContext(): AudioContext | null {
  if (typeof window === 'undefined') return null
  if (!audioContext) {
    try {
      audioContext = new (window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext)()
    } catch {
      return null
    }
  }
  return audioContext
}

// User preference for sounds
let soundsEnabled = true

// Check user preference from localStorage
if (typeof window !== 'undefined') {
  const stored = localStorage.getItem('soundsEnabled')
  soundsEnabled = stored === null ? true : stored === 'true'
}

// Enable/disable sounds
export function setSoundsEnabled(enabled: boolean): void {
  soundsEnabled = enabled
  if (typeof window !== 'undefined') {
    localStorage.setItem('soundsEnabled', String(enabled))
  }
}

export function getSoundsEnabled(): boolean {
  return soundsEnabled
}

// Preload sounds (no-op for Web Audio API)
export function preloadSounds(): void {
  // Web Audio API doesn't need preloading
}

// Play a sound effect using Web Audio API
export function playSound(sound: SoundType, volume = 0.5): void {
  if (!soundsEnabled) return

  try {
    const ctx = getAudioContext()
    if (!ctx) return

    // Resume context if suspended (browser autoplay policy)
    if (ctx.state === 'suspended') {
      ctx.resume()
    }

    // Parse the sound configuration
    const config = SOUNDS[sound]
    const [, freqStr, durStr, type] = config.split(':')
    const frequency = parseInt(freqStr, 10)
    const duration = parseInt(durStr, 10) / 1000 // Convert to seconds

    // Create oscillator
    const oscillator = ctx.createOscillator()
    const gainNode = ctx.createGain()

    oscillator.type = type as OscillatorType
    oscillator.frequency.setValueAtTime(frequency, ctx.currentTime)

    // Apply volume with envelope for smoother sound
    const adjustedVolume = Math.min(1, Math.max(0, volume)) * 0.3
    gainNode.gain.setValueAtTime(0, ctx.currentTime)
    gainNode.gain.linearRampToValueAtTime(adjustedVolume, ctx.currentTime + 0.01)
    gainNode.gain.linearRampToValueAtTime(0, ctx.currentTime + duration)

    oscillator.connect(gainNode)
    gainNode.connect(ctx.destination)

    oscillator.start(ctx.currentTime)
    oscillator.stop(ctx.currentTime + duration)
  } catch {
    // Silently handle any errors
  }
}

// Sound effect hooks for common actions
export const sounds = {
  // Exercise interactions
  correctAnswer: () => playSound('correct', 0.6),
  wrongAnswer: () => playSound('wrong', 0.5),
  lessonComplete: () => playSound('complete', 0.7),

  // UI interactions
  buttonClick: () => playSound('click', 0.3),
  wordSelect: () => playSound('select', 0.25),
  wordDeselect: () => playSound('deselect', 0.2),

  // Matching exercises
  matchFound: () => playSound('match', 0.5),

  // Progress & rewards
  levelUp: () => playSound('levelUp', 0.7),
  streakCelebration: () => playSound('streak', 0.7),
  milestone: () => playSound('milestone', 0.6),
  xpGained: () => playSound('xpGain', 0.5),
  heartLost: () => playSound('heartLost', 0.5),

  // Notifications
  notification: () => playSound('notification', 0.4),
}

// React hook for sound effects
export function useSounds() {
  return {
    ...sounds,
    enabled: soundsEnabled,
    setEnabled: setSoundsEnabled,
  }
}

// Play sound on interaction (for touch feedback)
export function playInteractionSound(): void {
  playSound('click', 0.2)
}

// Text-to-Speech using Web Speech API
// This is a fallback for when external TTS URLs (like Google Translate) don't work

// Language code mapping for Web Speech API
const TTS_LANG_MAP: Record<string, string> = {
  es: 'es-ES',
  fr: 'fr-FR',
  de: 'de-DE',
  it: 'it-IT',
  pt: 'pt-BR',
  ja: 'ja-JP',
  ko: 'ko-KR',
  zh: 'zh-CN',
  zs: 'zh-CN',
  ru: 'ru-RU',
  ar: 'ar-SA',
  en: 'en-US',
  nl: 'nl-NL',
  pl: 'pl-PL',
  tr: 'tr-TR',
  sv: 'sv-SE',
  no: 'nb-NO',
  da: 'da-DK',
  fi: 'fi-FI',
  el: 'el-GR',
  he: 'he-IL',
  hi: 'hi-IN',
  id: 'id-ID',
  th: 'th-TH',
  vi: 'vi-VN',
  uk: 'uk-UA',
  cs: 'cs-CZ',
  ro: 'ro-RO',
  hu: 'hu-HU',
}

// Check if Web Speech API is available
export function isTTSAvailable(): boolean {
  return typeof window !== 'undefined' && 'speechSynthesis' in window
}

// Get available voices for a language
export function getVoicesForLanguage(langCode: string): SpeechSynthesisVoice[] {
  if (!isTTSAvailable()) return []

  const mappedLang = TTS_LANG_MAP[langCode] || langCode
  const voices = window.speechSynthesis.getVoices()

  return voices.filter(voice =>
    voice.lang.startsWith(mappedLang.split('-')[0]) ||
    voice.lang.startsWith(langCode)
  )
}

// Speak text using Web Speech API
export function speakText(
  text: string,
  langCode: string = 'en',
  rate: number = 1.0,
  onEnd?: () => void,
  onError?: (error: Error) => void
): SpeechSynthesisUtterance | null {
  if (!isTTSAvailable()) {
    onError?.(new Error('Web Speech API not available'))
    return null
  }

  // Cancel any ongoing speech
  window.speechSynthesis.cancel()

  const utterance = new SpeechSynthesisUtterance(text)
  const mappedLang = TTS_LANG_MAP[langCode] || langCode

  utterance.lang = mappedLang
  utterance.rate = rate
  utterance.pitch = 1.0
  utterance.volume = 1.0

  // Try to find a good voice for this language
  const voices = getVoicesForLanguage(langCode)
  if (voices.length > 0) {
    // Prefer non-local voices (usually higher quality)
    const preferredVoice = voices.find(v => !v.localService) || voices[0]
    utterance.voice = preferredVoice
  }

  utterance.onend = () => onEnd?.()
  utterance.onerror = (event) => onError?.(new Error(event.error))

  window.speechSynthesis.speak(utterance)
  return utterance
}

// Stop any ongoing speech
export function stopSpeaking(): void {
  if (isTTSAvailable()) {
    window.speechSynthesis.cancel()
  }
}

// Extract language code from audio URL (for Google TTS URLs)
export function extractLangFromAudioUrl(url: string): string | null {
  if (!url) return null

  // Match tl=XX parameter in Google Translate URLs
  const match = url.match(/[?&]tl=([a-z]{2}(?:-[A-Z]{2})?)/i)
  return match ? match[1].toLowerCase() : null
}

// Extract text from audio URL (for Google TTS URLs)
export function extractTextFromAudioUrl(url: string): string | null {
  if (!url) return null

  // Match q=XXX parameter in Google Translate URLs
  const match = url.match(/[?&]q=([^&]+)/)
  if (match) {
    try {
      return decodeURIComponent(match[1])
    } catch {
      return match[1]
    }
  }
  return null
}

// Play audio URL or fallback to Web Speech API
export function playTTS(
  url: string | undefined,
  text?: string,
  langCode?: string,
  slow: boolean = false,
  onEnd?: () => void,
  onError?: () => void
): void {
  if (!url && !text) {
    onError?.()
    return
  }

  // If we have a URL, try to extract text and language from it
  if (url) {
    const extractedLang = extractLangFromAudioUrl(url)
    const extractedText = extractTextFromAudioUrl(url)

    // Use extracted values as fallbacks
    text = text || extractedText || ''
    langCode = langCode || extractedLang || 'en'
  }

  if (!text) {
    onError?.()
    return
  }

  // Use Web Speech API directly (Google TTS URLs often fail)
  const rate = slow ? 0.6 : 1.0
  speakText(text, langCode || 'en', rate, onEnd, () => onError?.())
}
