// Sound effects for Duolingo-style exercise interactions
// Uses free sound effect URLs from reliable CDNs

// Sound URLs (using royalty-free sounds that match Duolingo's style)
const SOUNDS = {
  // Correct answer - pleasant ascending ding (like Duolingo's success sound)
  correct: 'https://assets.mixkit.co/active_storage/sfx/2000/2000-preview.mp3',
  // Wrong answer - soft descending tone
  wrong: 'https://assets.mixkit.co/active_storage/sfx/2001/2001-preview.mp3',
  // Lesson complete - celebratory fanfare
  complete: 'https://assets.mixkit.co/active_storage/sfx/1997/1997-preview.mp3',
  // Button click/tap - subtle pop
  click: 'https://assets.mixkit.co/active_storage/sfx/2568/2568-preview.mp3',
  // Word selection - soft tap
  select: 'https://assets.mixkit.co/active_storage/sfx/2571/2571-preview.mp3',
  // Word deselection - subtle release
  deselect: 'https://assets.mixkit.co/active_storage/sfx/2570/2570-preview.mp3',
  // Level up / achievement
  levelUp: 'https://assets.mixkit.co/active_storage/sfx/2020/2020-preview.mp3',
  // Streak celebration
  streak: 'https://assets.mixkit.co/active_storage/sfx/2018/2018-preview.mp3',
  // Match found (for matching exercises)
  match: 'https://assets.mixkit.co/active_storage/sfx/2003/2003-preview.mp3',
  // Typing keypress (subtle)
  keypress: 'https://assets.mixkit.co/active_storage/sfx/2567/2567-preview.mp3',
  // Progress milestone
  milestone: 'https://assets.mixkit.co/active_storage/sfx/2019/2019-preview.mp3',
  // XP gain
  xpGain: 'https://assets.mixkit.co/active_storage/sfx/2017/2017-preview.mp3',
  // Heart lost
  heartLost: 'https://assets.mixkit.co/active_storage/sfx/2002/2002-preview.mp3',
  // Notification/alert
  notification: 'https://assets.mixkit.co/active_storage/sfx/2869/2869-preview.mp3',
} as const

export type SoundType = keyof typeof SOUNDS

// Audio cache to prevent re-loading sounds
const audioCache: Map<SoundType, HTMLAudioElement> = new Map()

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

// Preload sounds for faster playback
export function preloadSounds(): void {
  Object.entries(SOUNDS).forEach(([key, url]) => {
    const audio = new Audio(url)
    audio.preload = 'auto'
    audioCache.set(key as SoundType, audio)
  })
}

// Play a sound effect
export function playSound(sound: SoundType, volume = 0.5): void {
  if (!soundsEnabled) return

  try {
    // Try to get cached audio, or create new one
    let audio = audioCache.get(sound)

    if (!audio) {
      audio = new Audio(SOUNDS[sound])
      audioCache.set(sound, audio)
    }

    // Clone the audio to allow overlapping sounds
    const audioClone = audio.cloneNode() as HTMLAudioElement
    audioClone.volume = Math.min(1, Math.max(0, volume))

    // Play with error handling for autoplay restrictions
    audioClone.play().catch((err) => {
      // Silently fail if autoplay is blocked
      console.debug('Sound playback blocked:', err.message)
    })
  } catch (err) {
    // Silently handle any errors
    console.debug('Sound error:', err)
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

// Initialize sounds on module load (optional preload)
if (typeof window !== 'undefined') {
  // Delay preloading to not block initial page load
  setTimeout(() => {
    preloadSounds()
  }, 2000)
}
