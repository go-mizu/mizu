import { motion } from 'framer-motion'
import { useEffect, useState } from 'react'

interface OwlProps {
  size?: 'sm' | 'md' | 'lg' | 'xl'
  emotion?: 'happy' | 'excited' | 'waving'
  animate?: boolean
  className?: string
}

const sizes = {
  sm: 80,
  md: 120,
  lg: 200,
  xl: 280,
}

export function Owl({ size = 'lg', emotion = 'happy', animate = true, className }: OwlProps) {
  const [isBlinking, setIsBlinking] = useState(false)
  const pixelSize = sizes[size]

  // Random blink effect
  useEffect(() => {
    if (!animate) return

    const blinkInterval = setInterval(() => {
      // Random chance to blink every 2-4 seconds
      if (Math.random() > 0.3) {
        setIsBlinking(true)
        setTimeout(() => setIsBlinking(false), 150)
      }
    }, 2000 + Math.random() * 2000)

    return () => clearInterval(blinkInterval)
  }, [animate])

  return (
    <motion.div
      className={className}
      style={{ width: pixelSize, height: pixelSize }}
      animate={animate ? {
        y: [0, -8, 0],
        rotate: [0, -1, 1, 0],
      } : undefined}
      transition={{
        duration: 4,
        repeat: Infinity,
        ease: 'easeInOut',
      }}
    >
      <svg
        viewBox="0 0 200 220"
        width={pixelSize}
        height={pixelSize}
        style={{ overflow: 'visible' }}
      >
        {/* Drop shadow */}
        <defs>
          <filter id="owlShadow" x="-20%" y="-20%" width="140%" height="140%">
            <feDropShadow dx="0" dy="4" stdDeviation="4" floodOpacity="0.15" />
          </filter>
          <linearGradient id="bodyGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#6BD804" />
            <stop offset="100%" stopColor="#58CC02" />
          </linearGradient>
          <linearGradient id="bellyGradient" x1="0%" y1="0%" x2="0%" y2="100%">
            <stop offset="0%" stopColor="#8EE000" />
            <stop offset="100%" stopColor="#7AC70C" />
          </linearGradient>
        </defs>

        {/* Main body */}
        <g filter="url(#owlShadow)">
          {/* Body */}
          <ellipse cx="100" cy="130" rx="75" ry="80" fill="url(#bodyGradient)" />

          {/* Belly */}
          <ellipse cx="100" cy="150" rx="45" ry="50" fill="url(#bellyGradient)" />

          {/* Feet */}
          <ellipse cx="70" cy="205" rx="18" ry="10" fill="#FFC800" />
          <ellipse cx="130" cy="205" rx="18" ry="10" fill="#FFC800" />

          {/* Left wing */}
          <motion.ellipse
            cx="35"
            cy="130"
            rx="25"
            ry="45"
            fill="#4CAD02"
            animate={emotion === 'waving' ? {
              rotate: [-10, 30, -10],
              x: [0, 10, 0],
            } : undefined}
            transition={{
              duration: 0.6,
              repeat: Infinity,
              ease: 'easeInOut',
            }}
            style={{ transformOrigin: '35px 100px' }}
          />

          {/* Right wing */}
          <ellipse cx="165" cy="130" rx="25" ry="45" fill="#4CAD02" />

          {/* Eye whites */}
          <ellipse cx="70" cy="100" rx="30" ry="35" fill="white" />
          <ellipse cx="130" cy="100" rx="30" ry="35" fill="white" />

          {/* Pupils - animated for blink */}
          <motion.g
            animate={isBlinking ? { scaleY: 0.1 } : { scaleY: 1 }}
            transition={{ duration: 0.1 }}
            style={{ transformOrigin: '100px 100px' }}
          >
            <circle cx="75" cy="105" r="15" fill="#4B4B4B" />
            <circle cx="125" cy="105" r="15" fill="#4B4B4B" />

            {/* Eye highlights */}
            <circle cx="68" cy="98" r="6" fill="white" />
            <circle cx="118" cy="98" r="6" fill="white" />
            <circle cx="78" cy="108" r="3" fill="white" />
            <circle cx="128" cy="108" r="3" fill="white" />
          </motion.g>

          {/* Eyebrows based on emotion */}
          {emotion === 'excited' && (
            <>
              <path d="M45 70 Q70 60 90 75" stroke="#4CAD02" strokeWidth="4" fill="none" strokeLinecap="round" />
              <path d="M155 70 Q130 60 110 75" stroke="#4CAD02" strokeWidth="4" fill="none" strokeLinecap="round" />
            </>
          )}

          {/* Beak */}
          <path
            d="M100 130 L85 155 Q100 165 115 155 Z"
            fill="#FFC800"
          />
          <path
            d="M100 130 L92 145 Q100 150 108 145 Z"
            fill="#FFDD44"
          />

          {/* Cheek blush for happy emotion */}
          {(emotion === 'happy' || emotion === 'excited') && (
            <>
              <ellipse cx="40" cy="125" rx="12" ry="8" fill="#FF9999" opacity="0.4" />
              <ellipse cx="160" cy="125" rx="12" ry="8" fill="#FF9999" opacity="0.4" />
            </>
          )}
        </g>
      </svg>
    </motion.div>
  )
}

export default Owl
