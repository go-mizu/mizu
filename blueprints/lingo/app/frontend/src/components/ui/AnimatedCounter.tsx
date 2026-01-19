import { useEffect, useRef, useState } from 'react'
import { motion, useInView } from 'framer-motion'

interface AnimatedCounterProps {
  value: number | string
  suffix?: string
  prefix?: string
  duration?: number
  className?: string
  style?: React.CSSProperties
}

export function AnimatedCounter({
  value,
  suffix = '',
  prefix = '',
  duration = 2000,
  className,
  style,
}: AnimatedCounterProps) {
  const ref = useRef<HTMLSpanElement>(null)
  const isInView = useInView(ref, { once: true, margin: '-50px' })
  const [displayValue, setDisplayValue] = useState('0')

  useEffect(() => {
    if (!isInView) return

    // Parse the target value
    const stringValue = String(value)
    const numericMatch = stringValue.match(/^(\d+(?:\.\d+)?)(.*)$/)

    if (!numericMatch) {
      setDisplayValue(stringValue)
      return
    }

    const targetNum = parseFloat(numericMatch[1])
    const valueSuffix = numericMatch[2] || ''
    const isDecimal = stringValue.includes('.')
    const decimalPlaces = isDecimal ? (stringValue.split('.')[1]?.match(/\d+/)?.[0]?.length || 0) : 0

    const startTime = Date.now()
    const animate = () => {
      const elapsed = Date.now() - startTime
      const progress = Math.min(elapsed / duration, 1)

      // Ease out cubic
      const easeOut = 1 - Math.pow(1 - progress, 3)
      const currentValue = targetNum * easeOut

      if (isDecimal) {
        setDisplayValue(currentValue.toFixed(decimalPlaces) + valueSuffix)
      } else {
        setDisplayValue(Math.round(currentValue) + valueSuffix)
      }

      if (progress < 1) {
        requestAnimationFrame(animate)
      } else {
        setDisplayValue(stringValue)
      }
    }

    animate()
  }, [isInView, value, duration])

  return (
    <motion.span
      ref={ref}
      className={className}
      style={style}
      initial={{ opacity: 0, y: 20 }}
      animate={isInView ? { opacity: 1, y: 0 } : {}}
      transition={{ duration: 0.5 }}
    >
      {prefix}{displayValue}{suffix}
    </motion.span>
  )
}

export default AnimatedCounter
