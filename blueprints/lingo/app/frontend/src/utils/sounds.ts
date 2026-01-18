// Sound effects for Duolingo-style exercise interactions
// Uses free sound effect URLs from reliable CDNs

// Sound URLs (using royalty-free sounds)
const SOUNDS = {
  // Correct answer - pleasant ding
  correct: 'https://assets.mixkit.co/active_storage/sfx/2000/2000-preview.mp3',
  // Wrong answer - soft error tone
  wrong: 'https://assets.mixkit.co/active_storage/sfx/2001/2001-preview.mp3',
  // Lesson complete - celebratory fanfare
  complete: 'https://assets.mixkit.co/active_storage/sfx/1997/1997-preview.mp3',
  // Button click - subtle tap
  click: 'https://assets.mixkit.co/active_storage/sfx/2568/2568-preview.mp3',
  // Level up / achievement
  levelUp: 'https://assets.mixkit.co/active_storage/sfx/2020/2020-preview.mp3',
  // Streak celebration
  streak: 'https://assets.mixkit.co/active_storage/sfx/2018/2018-preview.mp3',
} as const

type SoundType = keyof typeof SOUNDS

// Audio cache to prevent re-loading sounds
const audioCache: Map<SoundType, HTMLAudioElement> = new Map()

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
  correctAnswer: () => playSound('correct', 0.6),
  wrongAnswer: () => playSound('wrong', 0.5),
  lessonComplete: () => playSound('complete', 0.7),
  buttonClick: () => playSound('click', 0.3),
  levelUp: () => playSound('levelUp', 0.7),
  streakCelebration: () => playSound('streak', 0.7),
}

// React hook for sound effects
export function useSounds() {
  return sounds
}

// Initialize sounds on module load (optional preload)
if (typeof window !== 'undefined') {
  // Delay preloading to not block initial page load
  setTimeout(() => {
    preloadSounds()
  }, 2000)
}
