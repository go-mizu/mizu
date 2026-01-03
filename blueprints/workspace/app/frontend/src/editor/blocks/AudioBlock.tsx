import { createReactBlockSpec } from '@blocknote/react'
import { useState, useRef, useEffect, useCallback } from 'react'
import {
  Music,
  Play,
  Pause,
  Volume2,
  VolumeX,
  Volume1,
  Upload,
  Link2,
  SkipBack,
  SkipForward,
  Loader2,
  AlertCircle,
  Download,
} from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

// Playback speed options
const PLAYBACK_SPEEDS = [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2]

export const AudioBlock = createReactBlockSpec(
  {
    type: 'audio',
    propSchema: {
      url: {
        default: '',
      },
      name: {
        default: '',
      },
      caption: {
        default: '',
      },
    },
    content: 'none',
  },
  {
    render: ({ block, editor }) => {
      const [isPlaying, setIsPlaying] = useState(false)
      const [isMuted, setIsMuted] = useState(false)
      const [volume, setVolume] = useState(1)
      const [previousVolume, setPreviousVolume] = useState(1)
      const [progress, setProgress] = useState(0)
      const [currentTime, setCurrentTime] = useState(0)
      const [duration, setDuration] = useState(0)
      const [isLoading, setIsLoading] = useState(true)
      const [error, setError] = useState<string | null>(null)
      const [playbackSpeed, setPlaybackSpeed] = useState(1)
      const [showSpeedMenu, setShowSpeedMenu] = useState(false)
      const [showVolumeSlider, setShowVolumeSlider] = useState(false)
      const [isDraggingProgress, setIsDraggingProgress] = useState(false)
      const [showUpload, setShowUpload] = useState(!block.props.url)
      const [isHovered, setIsHovered] = useState(false)

      const audioRef = useRef<HTMLAudioElement>(null)
      const progressRef = useRef<HTMLDivElement>(null)
      const fileInputRef = useRef<HTMLInputElement>(null)

      // Format time as mm:ss
      const formatTime = useCallback((seconds: number) => {
        if (!isFinite(seconds) || isNaN(seconds)) return '0:00'
        const mins = Math.floor(seconds / 60)
        const secs = Math.floor(seconds % 60)
        return `${mins}:${secs.toString().padStart(2, '0')}`
      }, [])

      // Handle file selection
      const handleFileSelect = async (e: React.ChangeEvent<HTMLInputElement>) => {
        const file = e.target.files?.[0]
        if (!file) return

        // Create object URL for local playback
        const url = URL.createObjectURL(file)
        editor.updateBlock(block, {
          props: {
            ...block.props,
            url,
            name: file.name,
          },
        })
        setShowUpload(false)
        setError(null)
      }

      // Handle URL input
      const handleUrlInput = () => {
        const url = prompt('Enter audio URL (direct link to audio file):')
        if (url) {
          editor.updateBlock(block, {
            props: {
              ...block.props,
              url,
              name: url.split('/').pop()?.split('?')[0] || 'Audio',
            },
          })
          setShowUpload(false)
          setError(null)
        }
      }

      // Toggle play/pause
      const togglePlay = useCallback(() => {
        if (!audioRef.current) return

        if (isPlaying) {
          audioRef.current.pause()
        } else {
          audioRef.current.play().catch((err) => {
            setError('Failed to play audio')
            console.error('Playback error:', err)
          })
        }
        setIsPlaying(!isPlaying)
      }, [isPlaying])

      // Toggle mute
      const toggleMute = useCallback(() => {
        if (!audioRef.current) return

        if (isMuted) {
          audioRef.current.volume = previousVolume
          setVolume(previousVolume)
        } else {
          setPreviousVolume(volume)
          audioRef.current.volume = 0
          setVolume(0)
        }
        setIsMuted(!isMuted)
      }, [isMuted, volume, previousVolume])

      // Handle volume change
      const handleVolumeChange = useCallback((newVolume: number) => {
        if (!audioRef.current) return

        const clampedVolume = Math.max(0, Math.min(1, newVolume))
        audioRef.current.volume = clampedVolume
        setVolume(clampedVolume)
        setIsMuted(clampedVolume === 0)
      }, [])

      // Handle playback speed change
      const handleSpeedChange = useCallback((speed: number) => {
        if (!audioRef.current) return

        audioRef.current.playbackRate = speed
        setPlaybackSpeed(speed)
        setShowSpeedMenu(false)
      }, [])

      // Handle progress bar seek
      const handleProgressSeek = useCallback((e: React.MouseEvent<HTMLDivElement>) => {
        if (!audioRef.current || !progressRef.current) return

        const rect = progressRef.current.getBoundingClientRect()
        const x = e.clientX - rect.left
        const percentage = Math.max(0, Math.min(1, x / rect.width))
        audioRef.current.currentTime = percentage * duration
      }, [duration])

      // Skip forward/backward
      const skip = useCallback((seconds: number) => {
        if (!audioRef.current) return
        audioRef.current.currentTime = Math.max(0, Math.min(duration, audioRef.current.currentTime + seconds))
      }, [duration])

      // Audio event handlers
      useEffect(() => {
        const audio = audioRef.current
        if (!audio) return

        const handleTimeUpdate = () => {
          if (!isDraggingProgress) {
            setCurrentTime(audio.currentTime)
            setProgress((audio.currentTime / audio.duration) * 100)
          }
        }

        const handleLoadedMetadata = () => {
          setDuration(audio.duration)
          setIsLoading(false)
          setError(null)
        }

        const handleLoadStart = () => {
          setIsLoading(true)
          setError(null)
        }

        const handleCanPlay = () => {
          setIsLoading(false)
        }

        const handleEnded = () => {
          setIsPlaying(false)
          setProgress(0)
          setCurrentTime(0)
        }

        const handleError = () => {
          setIsLoading(false)
          setError('Failed to load audio file')
          setIsPlaying(false)
        }

        const handlePlay = () => setIsPlaying(true)
        const handlePause = () => setIsPlaying(false)

        audio.addEventListener('timeupdate', handleTimeUpdate)
        audio.addEventListener('loadedmetadata', handleLoadedMetadata)
        audio.addEventListener('loadstart', handleLoadStart)
        audio.addEventListener('canplay', handleCanPlay)
        audio.addEventListener('ended', handleEnded)
        audio.addEventListener('error', handleError)
        audio.addEventListener('play', handlePlay)
        audio.addEventListener('pause', handlePause)

        return () => {
          audio.removeEventListener('timeupdate', handleTimeUpdate)
          audio.removeEventListener('loadedmetadata', handleLoadedMetadata)
          audio.removeEventListener('loadstart', handleLoadStart)
          audio.removeEventListener('canplay', handleCanPlay)
          audio.removeEventListener('ended', handleEnded)
          audio.removeEventListener('error', handleError)
          audio.removeEventListener('play', handlePlay)
          audio.removeEventListener('pause', handlePause)
        }
      }, [isDraggingProgress])

      // Get volume icon based on level
      const getVolumeIcon = () => {
        if (isMuted || volume === 0) return VolumeX
        if (volume < 0.5) return Volume1
        return Volume2
      }
      const VolumeIcon = getVolumeIcon()

      // Upload mode UI
      if (showUpload || !block.props.url) {
        return (
          <div
            className="audio-block upload-mode"
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              justifyContent: 'center',
              padding: '32px',
              background: 'var(--bg-secondary)',
              borderRadius: '8px',
              border: '1px dashed var(--border-color)',
              margin: '8px 0',
            }}
          >
            <input
              ref={fileInputRef}
              type="file"
              accept="audio/*"
              onChange={handleFileSelect}
              style={{ display: 'none' }}
            />
            <Music size={32} style={{ color: 'var(--text-tertiary)', marginBottom: '16px' }} />
            <p style={{ color: 'var(--text-secondary)', marginBottom: '16px', fontSize: '14px' }}>
              Add an audio file
            </p>
            <div style={{ display: 'flex', gap: '12px' }}>
              <button
                onClick={() => fileInputRef.current?.click()}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '8px 16px',
                  background: 'var(--accent-color)',
                  color: 'white',
                  border: 'none',
                  borderRadius: '6px',
                  fontSize: '14px',
                  fontWeight: 500,
                  cursor: 'pointer',
                  transition: 'opacity 0.15s',
                }}
                onMouseEnter={(e) => { e.currentTarget.style.opacity = '0.9' }}
                onMouseLeave={(e) => { e.currentTarget.style.opacity = '1' }}
              >
                <Upload size={16} />
                Upload
              </button>
              <button
                onClick={handleUrlInput}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: '8px',
                  padding: '8px 16px',
                  background: 'var(--bg-primary)',
                  color: 'var(--text-primary)',
                  border: '1px solid var(--border-color)',
                  borderRadius: '6px',
                  fontSize: '14px',
                  fontWeight: 500,
                  cursor: 'pointer',
                  transition: 'background 0.15s',
                }}
                onMouseEnter={(e) => { e.currentTarget.style.background = 'var(--bg-hover)' }}
                onMouseLeave={(e) => { e.currentTarget.style.background = 'var(--bg-primary)' }}
              >
                <Link2 size={16} />
                Embed URL
              </button>
            </div>
          </div>
        )
      }

      return (
        <div
          className="audio-block"
          onMouseEnter={() => setIsHovered(true)}
          onMouseLeave={() => {
            setIsHovered(false)
            setShowVolumeSlider(false)
            setShowSpeedMenu(false)
          }}
          style={{
            background: 'var(--bg-secondary)',
            borderRadius: '8px',
            padding: '16px',
            margin: '8px 0',
            position: 'relative',
          }}
        >
          <audio ref={audioRef} src={block.props.url} preload="metadata" />

          {/* Error state */}
          {error && (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                gap: '8px',
                color: 'var(--danger-color)',
                marginBottom: '12px',
                fontSize: '13px',
              }}
            >
              <AlertCircle size={16} />
              {error}
            </div>
          )}

          {/* Main player controls */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            {/* Play/Pause button */}
            <button
              onClick={togglePlay}
              disabled={isLoading || !!error}
              style={{
                width: '40px',
                height: '40px',
                borderRadius: '50%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                background: isLoading || error ? 'var(--bg-hover)' : 'var(--accent-color)',
                color: isLoading || error ? 'var(--text-tertiary)' : 'white',
                border: 'none',
                cursor: isLoading || error ? 'not-allowed' : 'pointer',
                flexShrink: 0,
                transition: 'transform 0.1s, opacity 0.15s',
              }}
              onMouseEnter={(e) => {
                if (!isLoading && !error) {
                  e.currentTarget.style.transform = 'scale(1.05)'
                }
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.transform = 'scale(1)'
              }}
            >
              {isLoading ? (
                <Loader2 size={20} className="animate-spin" style={{ animation: 'spin 1s linear infinite' }} />
              ) : isPlaying ? (
                <Pause size={20} />
              ) : (
                <Play size={20} style={{ marginLeft: '2px' }} />
              )}
            </button>

            {/* Track info and progress */}
            <div style={{ flex: 1, minWidth: 0 }}>
              {/* Track name */}
              <div
                style={{
                  fontSize: '14px',
                  fontWeight: 500,
                  color: 'var(--text-primary)',
                  marginBottom: '8px',
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {block.props.name || 'Audio'}
              </div>

              {/* Progress bar */}
              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                <span style={{ fontSize: '12px', color: 'var(--text-tertiary)', minWidth: '35px' }}>
                  {formatTime(currentTime)}
                </span>
                <div
                  ref={progressRef}
                  onClick={handleProgressSeek}
                  style={{
                    flex: 1,
                    height: '6px',
                    background: 'var(--bg-hover)',
                    borderRadius: '3px',
                    cursor: 'pointer',
                    position: 'relative',
                    overflow: 'hidden',
                  }}
                >
                  {/* Progress fill */}
                  <div
                    style={{
                      position: 'absolute',
                      left: 0,
                      top: 0,
                      height: '100%',
                      width: `${progress}%`,
                      background: 'var(--accent-color)',
                      borderRadius: '3px',
                      transition: isDraggingProgress ? 'none' : 'width 0.1s linear',
                    }}
                  />
                  {/* Hover handle */}
                  <div
                    style={{
                      position: 'absolute',
                      left: `${progress}%`,
                      top: '50%',
                      transform: 'translate(-50%, -50%)',
                      width: '12px',
                      height: '12px',
                      borderRadius: '50%',
                      background: 'var(--accent-color)',
                      opacity: isHovered ? 1 : 0,
                      transition: 'opacity 0.15s',
                      boxShadow: '0 1px 4px rgba(0,0,0,0.2)',
                    }}
                  />
                </div>
                <span style={{ fontSize: '12px', color: 'var(--text-tertiary)', minWidth: '35px', textAlign: 'right' }}>
                  {formatTime(duration)}
                </span>
              </div>
            </div>

            {/* Skip buttons (visible on hover) */}
            <AnimatePresence>
              {isHovered && (
                <motion.div
                  initial={{ opacity: 0, width: 0 }}
                  animate={{ opacity: 1, width: 'auto' }}
                  exit={{ opacity: 0, width: 0 }}
                  style={{ display: 'flex', gap: '4px', overflow: 'hidden' }}
                >
                  <button
                    onClick={() => skip(-10)}
                    title="Skip back 10 seconds"
                    style={{
                      width: '28px',
                      height: '28px',
                      borderRadius: '4px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: 'none',
                      color: 'var(--text-secondary)',
                      border: 'none',
                      cursor: 'pointer',
                      transition: 'background 0.15s, color 0.15s',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = 'var(--bg-hover)'
                      e.currentTarget.style.color = 'var(--text-primary)'
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = 'none'
                      e.currentTarget.style.color = 'var(--text-secondary)'
                    }}
                  >
                    <SkipBack size={16} />
                  </button>
                  <button
                    onClick={() => skip(10)}
                    title="Skip forward 10 seconds"
                    style={{
                      width: '28px',
                      height: '28px',
                      borderRadius: '4px',
                      display: 'flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: 'none',
                      color: 'var(--text-secondary)',
                      border: 'none',
                      cursor: 'pointer',
                      transition: 'background 0.15s, color 0.15s',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = 'var(--bg-hover)'
                      e.currentTarget.style.color = 'var(--text-primary)'
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = 'none'
                      e.currentTarget.style.color = 'var(--text-secondary)'
                    }}
                  >
                    <SkipForward size={16} />
                  </button>
                </motion.div>
              )}
            </AnimatePresence>

            {/* Playback speed control */}
            <div style={{ position: 'relative' }}>
              <button
                onClick={() => setShowSpeedMenu(!showSpeedMenu)}
                title="Playback speed"
                style={{
                  padding: '4px 8px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: playbackSpeed !== 1 ? 'var(--accent-bg)' : 'none',
                  color: playbackSpeed !== 1 ? 'var(--accent-color)' : 'var(--text-secondary)',
                  border: 'none',
                  cursor: 'pointer',
                  fontSize: '12px',
                  fontWeight: 600,
                  minWidth: '36px',
                  transition: 'background 0.15s, color 0.15s',
                }}
                onMouseEnter={(e) => {
                  if (playbackSpeed === 1) {
                    e.currentTarget.style.background = 'var(--bg-hover)'
                    e.currentTarget.style.color = 'var(--text-primary)'
                  }
                }}
                onMouseLeave={(e) => {
                  if (playbackSpeed === 1) {
                    e.currentTarget.style.background = 'none'
                    e.currentTarget.style.color = 'var(--text-secondary)'
                  }
                }}
              >
                {playbackSpeed}x
              </button>

              {/* Speed menu dropdown */}
              <AnimatePresence>
                {showSpeedMenu && (
                  <motion.div
                    initial={{ opacity: 0, y: -4, scale: 0.95 }}
                    animate={{ opacity: 1, y: 0, scale: 1 }}
                    exit={{ opacity: 0, y: -4, scale: 0.95 }}
                    transition={{ duration: 0.1 }}
                    style={{
                      position: 'absolute',
                      bottom: '100%',
                      right: 0,
                      marginBottom: '4px',
                      background: 'var(--bg-primary)',
                      borderRadius: '6px',
                      boxShadow: '0 2px 8px rgba(0,0,0,0.15), 0 0 0 1px rgba(0,0,0,0.05)',
                      padding: '4px',
                      zIndex: 10,
                    }}
                  >
                    {PLAYBACK_SPEEDS.map((speed) => (
                      <button
                        key={speed}
                        onClick={() => handleSpeedChange(speed)}
                        style={{
                          display: 'block',
                          width: '100%',
                          padding: '6px 12px',
                          background: speed === playbackSpeed ? 'var(--accent-bg)' : 'none',
                          color: speed === playbackSpeed ? 'var(--accent-color)' : 'var(--text-primary)',
                          border: 'none',
                          borderRadius: '4px',
                          cursor: 'pointer',
                          fontSize: '13px',
                          fontWeight: speed === playbackSpeed ? 600 : 400,
                          textAlign: 'left',
                          whiteSpace: 'nowrap',
                          transition: 'background 0.1s',
                        }}
                        onMouseEnter={(e) => {
                          if (speed !== playbackSpeed) {
                            e.currentTarget.style.background = 'var(--bg-hover)'
                          }
                        }}
                        onMouseLeave={(e) => {
                          if (speed !== playbackSpeed) {
                            e.currentTarget.style.background = 'none'
                          }
                        }}
                      >
                        {speed}x
                      </button>
                    ))}
                  </motion.div>
                )}
              </AnimatePresence>
            </div>

            {/* Volume control */}
            <div
              style={{ position: 'relative' }}
              onMouseEnter={() => setShowVolumeSlider(true)}
              onMouseLeave={() => setShowVolumeSlider(false)}
            >
              <button
                onClick={toggleMute}
                title={isMuted ? 'Unmute' : 'Mute'}
                style={{
                  width: '28px',
                  height: '28px',
                  borderRadius: '4px',
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'center',
                  background: 'none',
                  color: 'var(--text-secondary)',
                  border: 'none',
                  cursor: 'pointer',
                  transition: 'background 0.15s, color 0.15s',
                }}
                onMouseEnter={(e) => {
                  e.currentTarget.style.background = 'var(--bg-hover)'
                  e.currentTarget.style.color = 'var(--text-primary)'
                }}
                onMouseLeave={(e) => {
                  e.currentTarget.style.background = 'none'
                  e.currentTarget.style.color = 'var(--text-secondary)'
                }}
              >
                <VolumeIcon size={18} />
              </button>

              {/* Volume slider */}
              <AnimatePresence>
                {showVolumeSlider && (
                  <motion.div
                    initial={{ opacity: 0, y: 4 }}
                    animate={{ opacity: 1, y: 0 }}
                    exit={{ opacity: 0, y: 4 }}
                    transition={{ duration: 0.1 }}
                    style={{
                      position: 'absolute',
                      bottom: '100%',
                      left: '50%',
                      transform: 'translateX(-50%)',
                      marginBottom: '8px',
                      background: 'var(--bg-primary)',
                      borderRadius: '6px',
                      boxShadow: '0 2px 8px rgba(0,0,0,0.15), 0 0 0 1px rgba(0,0,0,0.05)',
                      padding: '12px 8px',
                      zIndex: 10,
                    }}
                  >
                    <input
                      type="range"
                      min="0"
                      max="1"
                      step="0.05"
                      value={volume}
                      onChange={(e) => handleVolumeChange(parseFloat(e.target.value))}
                      style={{
                        writingMode: 'vertical-lr',
                        direction: 'rtl',
                        height: '80px',
                        width: '4px',
                        appearance: 'none',
                        background: 'var(--bg-hover)',
                        borderRadius: '2px',
                        cursor: 'pointer',
                      }}
                    />
                  </motion.div>
                )}
              </AnimatePresence>
            </div>

            {/* Download button */}
            <AnimatePresence>
              {isHovered && (
                <motion.button
                  initial={{ opacity: 0, scale: 0.8 }}
                  animate={{ opacity: 1, scale: 1 }}
                  exit={{ opacity: 0, scale: 0.8 }}
                  onClick={() => {
                    const a = document.createElement('a')
                    a.href = block.props.url
                    a.download = block.props.name || 'audio'
                    a.click()
                  }}
                  title="Download"
                  style={{
                    width: '28px',
                    height: '28px',
                    borderRadius: '4px',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: 'none',
                    color: 'var(--text-secondary)',
                    border: 'none',
                    cursor: 'pointer',
                    transition: 'background 0.15s, color 0.15s',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = 'var(--bg-hover)'
                    e.currentTarget.style.color = 'var(--text-primary)'
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = 'none'
                    e.currentTarget.style.color = 'var(--text-secondary)'
                  }}
                >
                  <Download size={16} />
                </motion.button>
              )}
            </AnimatePresence>
          </div>

          {/* Caption */}
          {block.props.caption && (
            <div
              style={{
                marginTop: '8px',
                paddingTop: '8px',
                borderTop: '1px solid var(--border-color)',
                fontSize: '13px',
                color: 'var(--text-secondary)',
                fontStyle: 'italic',
              }}
            >
              {block.props.caption}
            </div>
          )}

          {/* CSS for spinner animation */}
          <style>{`
            @keyframes spin {
              from { transform: rotate(0deg); }
              to { transform: rotate(360deg); }
            }
            input[type="range"]::-webkit-slider-thumb {
              -webkit-appearance: none;
              width: 12px;
              height: 12px;
              background: var(--accent-color);
              border-radius: 50%;
              cursor: pointer;
            }
            input[type="range"]::-moz-range-thumb {
              width: 12px;
              height: 12px;
              background: var(--accent-color);
              border-radius: 50%;
              cursor: pointer;
              border: none;
            }
          `}</style>
        </div>
      )
    },
  }
)
