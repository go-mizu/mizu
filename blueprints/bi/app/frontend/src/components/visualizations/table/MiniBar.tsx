import { Box, Text } from '@mantine/core'
import classes from './MiniBar.module.css'

interface MiniBarProps {
  value: number
  min: number
  max: number
  color?: string
}

export default function MiniBar({ value, min, max, color = '#509ee3' }: MiniBarProps) {
  const range = max - min
  const percentage = range > 0 ? ((value - min) / range) * 100 : 0
  const clampedPercentage = Math.max(0, Math.min(100, percentage))

  const formattedValue = value.toLocaleString(undefined, {
    minimumFractionDigits: 0,
    maximumFractionDigits: 2,
  })

  return (
    <Box className={classes.container}>
      <Box className={classes.barWrapper}>
        <Box
          className={classes.barFill}
          style={{
            width: `${clampedPercentage}%`,
            backgroundColor: color,
          }}
        />
      </Box>
      <Text className={classes.value}>{formattedValue}</Text>
    </Box>
  )
}
