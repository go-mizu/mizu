import { motion } from 'framer-motion'
import { Owl } from './Owl'

interface CharacterGroupProps {
  className?: string
}

// Supporting character - simple geometric style
function Character({
  color,
  delay,
  x,
  y,
  size = 60,
}: {
  color: string
  delay: number
  x: number
  y: number
  size?: number
}) {
  const shadowColor = color === '#1CB0F6' ? '#1899D6'
    : color === '#FF9600' ? '#E57D00'
    : color === '#CE82FF' ? '#A855E8'
    : color === '#FF4B4B' ? '#DD3333'
    : '#58A700'

  return (
    <motion.div
      style={{
        position: 'absolute',
        left: x,
        top: y,
        width: size,
        height: size,
      }}
      initial={{ opacity: 0, scale: 0, y: 20 }}
      animate={{
        opacity: 1,
        scale: 1,
        y: 0,
      }}
      transition={{
        delay: delay,
        duration: 0.5,
        type: 'spring',
        stiffness: 200,
      }}
    >
      <motion.div
        animate={{
          y: [0, -10 - Math.random() * 5, 0],
          rotate: [-2, 2, -2],
        }}
        transition={{
          duration: 3 + Math.random() * 2,
          repeat: Infinity,
          ease: 'easeInOut',
          delay: delay,
        }}
      >
        <svg viewBox="0 0 60 70" width={size} height={size * 1.17}>
          {/* Shadow */}
          <ellipse cx="30" cy="65" rx="20" ry="5" fill="rgba(0,0,0,0.1)" />

          {/* Body */}
          <circle cx="30" cy="35" r="25" fill={color} />

          {/* Depth shadow */}
          <path
            d="M10 45 Q30 70 50 45 Q50 55 30 60 Q10 55 10 45"
            fill={shadowColor}
          />

          {/* Eyes */}
          <ellipse cx="22" cy="32" rx="8" ry="10" fill="white" />
          <ellipse cx="38" cy="32" rx="8" ry="10" fill="white" />
          <circle cx="24" cy="34" r="5" fill="#4B4B4B" />
          <circle cx="40" cy="34" r="5" fill="#4B4B4B" />
          <circle cx="22" cy="32" r="2" fill="white" />
          <circle cx="38" cy="32" r="2" fill="white" />

          {/* Smile */}
          <path
            d="M22 45 Q30 52 38 45"
            stroke="#4B4B4B"
            strokeWidth="2"
            fill="none"
            strokeLinecap="round"
          />

          {/* Cheeks */}
          <circle cx="15" cy="40" r="4" fill="rgba(255,150,150,0.5)" />
          <circle cx="45" cy="40" r="4" fill="rgba(255,150,150,0.5)" />
        </svg>
      </motion.div>
    </motion.div>
  )
}

// Character with arms raised
function ExcitedCharacter({
  color,
  delay,
  x,
  y,
  size = 70
}: {
  color: string
  delay: number
  x: number
  y: number
  size?: number
}) {
  return (
    <motion.div
      style={{
        position: 'absolute',
        left: x,
        top: y,
        width: size,
        height: size,
      }}
      initial={{ opacity: 0, scale: 0, y: 20 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      transition={{
        delay: delay,
        duration: 0.5,
        type: 'spring',
        stiffness: 200,
      }}
    >
      <motion.div
        animate={{
          y: [0, -12, 0],
          rotate: [-3, 3, -3],
        }}
        transition={{
          duration: 2.5,
          repeat: Infinity,
          ease: 'easeInOut',
          delay: delay,
        }}
      >
        <svg viewBox="0 0 80 90" width={size} height={size * 1.125}>
          {/* Shadow */}
          <ellipse cx="40" cy="85" rx="25" ry="5" fill="rgba(0,0,0,0.1)" />

          {/* Left arm raised */}
          <motion.ellipse
            cx="12"
            cy="30"
            rx="10"
            ry="18"
            fill={color}
            animate={{ rotate: [-10, 10, -10] }}
            transition={{ duration: 0.5, repeat: Infinity }}
            style={{ transformOrigin: '20px 45px' }}
          />

          {/* Right arm raised */}
          <motion.ellipse
            cx="68"
            cy="30"
            rx="10"
            ry="18"
            fill={color}
            animate={{ rotate: [10, -10, 10] }}
            transition={{ duration: 0.5, repeat: Infinity }}
            style={{ transformOrigin: '60px 45px' }}
          />

          {/* Body */}
          <ellipse cx="40" cy="50" rx="28" ry="32" fill={color} />

          {/* Eyes */}
          <ellipse cx="30" cy="45" rx="9" ry="11" fill="white" />
          <ellipse cx="50" cy="45" rx="9" ry="11" fill="white" />
          <circle cx="32" cy="47" r="5" fill="#4B4B4B" />
          <circle cx="52" cy="47" r="5" fill="#4B4B4B" />
          <circle cx="30" cy="44" r="2" fill="white" />
          <circle cx="50" cy="44" r="2" fill="white" />

          {/* Big smile */}
          <path
            d="M28 60 Q40 72 52 60"
            stroke="#4B4B4B"
            strokeWidth="2.5"
            fill="none"
            strokeLinecap="round"
          />
        </svg>
      </motion.div>
    </motion.div>
  )
}

// Character looking at phone/tablet
function StudyingCharacter({
  color,
  delay,
  x,
  y,
  size = 65
}: {
  color: string
  delay: number
  x: number
  y: number
  size?: number
}) {
  return (
    <motion.div
      style={{
        position: 'absolute',
        left: x,
        top: y,
        width: size,
        height: size,
      }}
      initial={{ opacity: 0, scale: 0, y: 20 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      transition={{
        delay: delay,
        duration: 0.5,
        type: 'spring',
        stiffness: 200,
      }}
    >
      <motion.div
        animate={{
          y: [0, -6, 0],
          rotate: [0, -2, 0],
        }}
        transition={{
          duration: 3.5,
          repeat: Infinity,
          ease: 'easeInOut',
          delay: delay,
        }}
      >
        <svg viewBox="0 0 80 90" width={size} height={size * 1.125}>
          {/* Shadow */}
          <ellipse cx="40" cy="85" rx="22" ry="5" fill="rgba(0,0,0,0.1)" />

          {/* Body */}
          <ellipse cx="40" cy="50" rx="26" ry="30" fill={color} />

          {/* Tablet/phone */}
          <rect x="50" y="55" width="22" height="30" rx="3" fill="#4B4B4B" />
          <rect x="52" y="58" width="18" height="22" rx="1" fill="#1CB0F6" />

          {/* Arm holding device */}
          <ellipse cx="55" cy="60" rx="12" ry="8" fill={color} />

          {/* Eyes looking down at device */}
          <ellipse cx="32" cy="42" rx="8" ry="10" fill="white" />
          <ellipse cx="50" cy="44" rx="7" ry="9" fill="white" />
          <circle cx="34" cy="46" r="4" fill="#4B4B4B" />
          <circle cx="51" cy="48" r="4" fill="#4B4B4B" />
          <circle cx="32" cy="44" r="1.5" fill="white" />
          <circle cx="49" cy="46" r="1.5" fill="white" />

          {/* Concentrated expression */}
          <path
            d="M30 58 Q38 62 46 58"
            stroke="#4B4B4B"
            strokeWidth="2"
            fill="none"
            strokeLinecap="round"
          />
        </svg>
      </motion.div>
    </motion.div>
  )
}

export function CharacterGroup({ className }: CharacterGroupProps) {
  return (
    <motion.div
      className={className}
      style={{
        position: 'relative',
        width: 350,
        height: 350,
      }}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.5 }}
    >
      {/* Background floating characters */}
      <Character color="#1CB0F6" delay={0.3} x={0} y={40} size={55} />
      <ExcitedCharacter color="#FF9600" delay={0.4} x={260} y={20} size={65} />
      <StudyingCharacter color="#CE82FF" delay={0.5} x={20} y={180} size={60} />
      <Character color="#FF4B4B" delay={0.6} x={270} y={200} size={50} />

      {/* Main owl mascot in center */}
      <motion.div
        style={{
          position: 'absolute',
          left: '50%',
          top: '50%',
          transform: 'translate(-50%, -50%)',
        }}
        initial={{ opacity: 0, scale: 0 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{
          delay: 0.2,
          duration: 0.6,
          type: 'spring',
          stiffness: 150,
        }}
      >
        <Owl size="lg" emotion="happy" animate />
      </motion.div>
    </motion.div>
  )
}

export default CharacterGroup
