import { useRef, useState, useEffect } from 'react'
import { motion } from 'framer-motion'
import { IconChevronLeft, IconChevronRight } from '@tabler/icons-react'
import { colors } from '../../styles/tokens'

interface Language {
  code: string
  name: string
  flag: string
  learners: string
}

const languages: Language[] = [
  { code: 'en', name: 'ENGLISH', flag: '🇺🇸', learners: '45.2M' },
  { code: 'es', name: 'SPANISH', flag: '🇪🇸', learners: '34.2M' },
  { code: 'fr', name: 'FRENCH', flag: '🇫🇷', learners: '23.1M' },
  { code: 'de', name: 'GERMAN', flag: '🇩🇪', learners: '17.8M' },
  { code: 'it', name: 'ITALIAN', flag: '🇮🇹', learners: '12.5M' },
  { code: 'pt', name: 'PORTUGUESE', flag: '🇧🇷', learners: '11.3M' },
  { code: 'nl', name: 'DUTCH', flag: '🇳🇱', learners: '5.2M' },
  { code: 'ja', name: 'JAPANESE', flag: '🇯🇵', learners: '16.4M' },
  { code: 'ko', name: 'KOREAN', flag: '🇰🇷', learners: '13.2M' },
  { code: 'zh', name: 'CHINESE', flag: '🇨🇳', learners: '11.9M' },
  { code: 'ru', name: 'RUSSIAN', flag: '🇷🇺', learners: '8.7M' },
  { code: 'hi', name: 'HINDI', flag: '🇮🇳', learners: '6.1M' },
]

interface LanguageCarouselProps {
  onSelect?: (code: string) => void
}

export function LanguageCarousel({ onSelect }: LanguageCarouselProps) {
  const scrollRef = useRef<HTMLDivElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(true)
  const [selectedCode, setSelectedCode] = useState<string | null>(null)

  const checkScroll = () => {
    if (!scrollRef.current) return
    const { scrollLeft, scrollWidth, clientWidth } = scrollRef.current
    setCanScrollLeft(scrollLeft > 0)
    setCanScrollRight(scrollLeft < scrollWidth - clientWidth - 10)
  }

  useEffect(() => {
    checkScroll()
    const el = scrollRef.current
    if (el) {
      el.addEventListener('scroll', checkScroll)
      return () => el.removeEventListener('scroll', checkScroll)
    }
  }, [])

  const scroll = (direction: 'left' | 'right') => {
    if (!scrollRef.current) return
    const scrollAmount = 300
    scrollRef.current.scrollBy({
      left: direction === 'left' ? -scrollAmount : scrollAmount,
      behavior: 'smooth',
    })
  }

  const handleSelect = (code: string) => {
    setSelectedCode(code)
    onSelect?.(code)
  }

  return (
    <div
      style={{
        position: 'relative',
        width: '100%',
        borderTop: `1px solid ${colors.neutral.border}`,
        borderBottom: `1px solid ${colors.neutral.border}`,
        backgroundColor: 'white',
      }}
    >
      {/* Left scroll button */}
      <motion.button
        onClick={() => scroll('left')}
        style={{
          position: 'absolute',
          left: 0,
          top: '50%',
          transform: 'translateY(-50%)',
          zIndex: 10,
          width: 40,
          height: 40,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          border: 'none',
          background: 'linear-gradient(to right, white 60%, transparent)',
          cursor: canScrollLeft ? 'pointer' : 'default',
          opacity: canScrollLeft ? 1 : 0.3,
          paddingLeft: 8,
        }}
        whileHover={canScrollLeft ? { scale: 1.1 } : undefined}
        whileTap={canScrollLeft ? { scale: 0.95 } : undefined}
        disabled={!canScrollLeft}
      >
        <IconChevronLeft size={24} color={colors.text.secondary} />
      </motion.button>

      {/* Scrollable container */}
      <div
        ref={scrollRef}
        style={{
          display: 'flex',
          gap: 8,
          overflowX: 'auto',
          scrollSnapType: 'x mandatory',
          scrollbarWidth: 'none',
          msOverflowStyle: 'none',
          padding: '16px 48px',
          WebkitOverflowScrolling: 'touch',
        }}
      >
        {languages.map((lang, index) => (
          <motion.button
            key={lang.code}
            onClick={() => handleSelect(lang.code)}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: index * 0.03 }}
            style={{
              display: 'flex',
              alignItems: 'center',
              gap: 8,
              padding: '10px 16px',
              border: 'none',
              background: 'transparent',
              cursor: 'pointer',
              scrollSnapAlign: 'center',
              flexShrink: 0,
              borderRadius: 8,
              transition: 'all 0.15s ease',
            }}
            whileHover={{
              backgroundColor: colors.neutral.background,
            }}
          >
            <span style={{ fontSize: 20 }}>{lang.flag}</span>
            <span
              style={{
                fontSize: 14,
                fontWeight: 700,
                letterSpacing: '0.5px',
                color: selectedCode === lang.code ? colors.primary.green : colors.text.secondary,
                transition: 'color 0.15s ease',
              }}
            >
              {lang.name}
            </span>
          </motion.button>
        ))}
      </div>

      {/* Right scroll button */}
      <motion.button
        onClick={() => scroll('right')}
        style={{
          position: 'absolute',
          right: 0,
          top: '50%',
          transform: 'translateY(-50%)',
          zIndex: 10,
          width: 40,
          height: 40,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          border: 'none',
          background: 'linear-gradient(to left, white 60%, transparent)',
          cursor: canScrollRight ? 'pointer' : 'default',
          opacity: canScrollRight ? 1 : 0.3,
          paddingRight: 8,
        }}
        whileHover={canScrollRight ? { scale: 1.1 } : undefined}
        whileTap={canScrollRight ? { scale: 0.95 } : undefined}
        disabled={!canScrollRight}
      >
        <IconChevronRight size={24} color={colors.text.secondary} />
      </motion.button>

      {/* Hide scrollbar */}
      <style>{`
        div::-webkit-scrollbar {
          display: none;
        }
      `}</style>
    </div>
  )
}

export default LanguageCarousel
