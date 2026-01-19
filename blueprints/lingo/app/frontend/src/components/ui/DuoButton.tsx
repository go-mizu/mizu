import { forwardRef } from 'react'
import { motion, HTMLMotionProps } from 'framer-motion'
import { colors } from '../../styles/tokens'

type ButtonVariant = 'primary' | 'secondary' | 'outline' | 'text'
type ButtonSize = 'sm' | 'md' | 'lg' | 'xl'

interface DuoButtonProps extends Omit<HTMLMotionProps<'button'>, 'ref'> {
  variant?: ButtonVariant
  size?: ButtonSize
  fullWidth?: boolean
  loading?: boolean
  glow?: boolean
}

const variantStyles: Record<ButtonVariant, {
  background: string
  color: string
  shadow: string
  hoverBackground: string
  border?: string
}> = {
  primary: {
    background: colors.primary.green,
    color: 'white',
    shadow: `0 4px 0 ${colors.primary.greenShadow}`,
    hoverBackground: colors.primary.greenHover,
  },
  secondary: {
    background: colors.secondary.blue,
    color: 'white',
    shadow: `0 4px 0 ${colors.secondary.blueShadow}`,
    hoverBackground: colors.secondary.blueHover,
  },
  outline: {
    background: 'white',
    color: colors.secondary.blue,
    shadow: `0 4px 0 ${colors.neutral.border}`,
    hoverBackground: colors.neutral.background,
    border: `2px solid ${colors.neutral.border}`,
  },
  text: {
    background: 'transparent',
    color: colors.text.secondary,
    shadow: 'none',
    hoverBackground: colors.neutral.background,
  },
}

const sizeStyles: Record<ButtonSize, {
  height: number
  fontSize: string
  padding: string
  borderRadius: string
}> = {
  sm: {
    height: 40,
    fontSize: '13px',
    padding: '0 16px',
    borderRadius: '12px',
  },
  md: {
    height: 48,
    fontSize: '15px',
    padding: '0 20px',
    borderRadius: '14px',
  },
  lg: {
    height: 56,
    fontSize: '15px',
    padding: '0 24px',
    borderRadius: '16px',
  },
  xl: {
    height: 64,
    fontSize: '16px',
    padding: '0 32px',
    borderRadius: '16px',
  },
}

export const DuoButton = forwardRef<HTMLButtonElement, DuoButtonProps>(
  ({
    variant = 'primary',
    size = 'md',
    fullWidth = false,
    loading = false,
    glow = false,
    disabled,
    children,
    style,
    ...props
  }, ref) => {
    const variantStyle = variantStyles[variant]
    const sizeStyle = sizeStyles[size]

    const isDisabled = disabled || loading

    return (
      <motion.button
        ref={ref}
        disabled={isDisabled}
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: '8px',
          height: sizeStyle.height,
          padding: sizeStyle.padding,
          fontSize: sizeStyle.fontSize,
          fontWeight: 700,
          fontFamily: 'inherit',
          textTransform: 'uppercase',
          letterSpacing: '0.8px',
          border: variantStyle.border || 'none',
          borderRadius: sizeStyle.borderRadius,
          cursor: isDisabled ? 'not-allowed' : 'pointer',
          backgroundColor: isDisabled ? colors.neutral.border : variantStyle.background,
          color: isDisabled ? colors.text.muted : variantStyle.color,
          boxShadow: isDisabled
            ? `0 4px 0 ${colors.text.disabled}`
            : variantStyle.shadow,
          width: fullWidth ? '100%' : 'auto',
          transition: 'background-color 0.1s ease',
          outline: 'none',
          ...style,
        }}
        whileHover={!isDisabled ? {
          backgroundColor: variantStyle.hoverBackground,
        } : undefined}
        whileTap={!isDisabled ? {
          y: 4,
          boxShadow: variant === 'text' ? 'none' : '0 0 0 transparent',
          transition: { duration: 0.1 },
        } : undefined}
        animate={glow && !isDisabled ? {
          boxShadow: [
            variantStyle.shadow,
            `${variantStyle.shadow}, 0 0 20px ${variantStyle.background}40`,
            variantStyle.shadow,
          ],
        } : undefined}
        transition={glow ? {
          duration: 2,
          repeat: Infinity,
          ease: 'easeInOut',
        } : undefined}
        {...props}
      >
        {loading ? (
          <motion.div
            style={{
              width: 20,
              height: 20,
              border: '2px solid currentColor',
              borderTopColor: 'transparent',
              borderRadius: '50%',
            }}
            animate={{ rotate: 360 }}
            transition={{ duration: 1, repeat: Infinity, ease: 'linear' }}
          />
        ) : children}
      </motion.button>
    )
  }
)

DuoButton.displayName = 'DuoButton'

export default DuoButton
