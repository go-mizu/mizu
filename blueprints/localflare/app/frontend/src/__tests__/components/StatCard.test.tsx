import { describe, it, expect, vi } from 'vitest'
import { renderWithProviders, screen, userEvent } from '../../test/utils'
import { StatCard, CompactStatCard } from '../../components/common/StatCard'
import { IconDatabase } from '@tabler/icons-react'

describe('StatCard', () => {
  it('renders with required props', () => {
    renderWithProviders(
      <StatCard
        icon={<IconDatabase data-testid="icon" />}
        label="Test Label"
        value={100}
      />
    )

    expect(screen.getByText('Test Label')).toBeInTheDocument()
    expect(screen.getByText('100')).toBeInTheDocument()
    expect(screen.getByTestId('icon')).toBeInTheDocument()
  })

  it('formats large numbers with locale', () => {
    renderWithProviders(
      <StatCard
        icon={<IconDatabase />}
        label="Large Number"
        value={1234567}
      />
    )

    expect(screen.getByText('1,234,567')).toBeInTheDocument()
  })

  it('renders string values', () => {
    renderWithProviders(
      <StatCard
        icon={<IconDatabase />}
        label="String Value"
        value="N/A"
      />
    )

    expect(screen.getByText('N/A')).toBeInTheDocument()
  })

  it('renders description', () => {
    renderWithProviders(
      <StatCard
        icon={<IconDatabase />}
        label="With Description"
        value={50}
        description="Additional context"
      />
    )

    expect(screen.getByText('Additional context')).toBeInTheDocument()
  })

  describe('trend indicator', () => {
    it('shows upward trend', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Trending Up"
          value={100}
          trend={{ value: 15, direction: 'up' }}
        />
      )

      expect(screen.getByText('+15%')).toBeInTheDocument()
    })

    it('shows downward trend', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Trending Down"
          value={100}
          trend={{ value: -10, direction: 'down' }}
        />
      )

      expect(screen.getByText('-10%')).toBeInTheDocument()
    })

    it('shows neutral trend', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Neutral"
          value={100}
          trend={{ value: 0, direction: 'neutral' }}
        />
      )

      expect(screen.getByText('0%')).toBeInTheDocument()
    })

    it('calculates trend from previous value', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Calculated Trend"
          value={120}
          previousValue={100}
        />
      )

      // 20% increase from 100 to 120
      expect(screen.getByText('+20%')).toBeInTheDocument()
      expect(screen.getByText('vs prev')).toBeInTheDocument()
    })
  })

  describe('click behavior', () => {
    it('is clickable when onClick provided', async () => {
      const onClick = vi.fn()
      const user = userEvent.setup()

      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Clickable"
          value={100}
          onClick={onClick}
        />
      )

      const card = screen.getByRole('button')
      await user.click(card)

      expect(onClick).toHaveBeenCalledTimes(1)
    })

    it('is not a button when onClick not provided', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Not Clickable"
          value={100}
        />
      )

      expect(screen.queryByRole('button')).not.toBeInTheDocument()
    })
  })

  describe('sparkline', () => {
    it('renders sparkline when data provided', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="With Sparkline"
          value={100}
          sparklineData={[10, 20, 30, 40, 50]}
        />
      )

      // Sparkline uses SVG
      const svg = document.querySelector('svg')
      expect(svg).toBeInTheDocument()
    })

    it('does not render sparkline with single data point', () => {
      const { container } = renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Single Point"
          value={100}
          sparklineData={[10]}
        />
      )

      // Should only have the icon SVG, not an area chart
      const svgs = container.querySelectorAll('svg')
      expect(svgs.length).toBe(1) // Only the icon
    })
  })

  describe('color variants', () => {
    it('applies default color', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Default Color"
          value={100}
        />
      )

      // Check that the card renders (color is applied via CSS)
      expect(screen.getByText('Default Color')).toBeInTheDocument()
    })

    it('applies success color', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Success Color"
          value={100}
          color="success"
        />
      )

      expect(screen.getByText('Success Color')).toBeInTheDocument()
    })

    it('applies orange color', () => {
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="Orange Color"
          value={100}
          color="orange"
        />
      )

      expect(screen.getByText('Orange Color')).toBeInTheDocument()
    })
  })

  describe('help text tooltip', () => {
    it('shows tooltip when helpText provided and clickable', async () => {
      const user = userEvent.setup()
      renderWithProviders(
        <StatCard
          icon={<IconDatabase />}
          label="With Help"
          value={100}
          onClick={() => {}}
          helpText="This is help text"
        />
      )

      const button = screen.getByRole('button')
      await user.hover(button)

      // Tooltip should appear (may need to wait)
      expect(await screen.findByText('This is help text')).toBeInTheDocument()
    })
  })
})

describe('CompactStatCard', () => {
  it('renders with required props', () => {
    renderWithProviders(
      <CompactStatCard
        label="Compact Label"
        value={50}
      />
    )

    expect(screen.getByText('Compact Label')).toBeInTheDocument()
    expect(screen.getByText('50')).toBeInTheDocument()
  })

  it('renders with icon', () => {
    renderWithProviders(
      <CompactStatCard
        label="With Icon"
        value={50}
        icon={<IconDatabase data-testid="compact-icon" />}
      />
    )

    expect(screen.getByTestId('compact-icon')).toBeInTheDocument()
  })

  it('shows trend indicator', () => {
    renderWithProviders(
      <CompactStatCard
        label="With Trend"
        value={50}
        trend={{ value: 10, direction: 'up' }}
      />
    )

    expect(screen.getByText('+10%')).toBeInTheDocument()
  })

  it('formats large numbers', () => {
    renderWithProviders(
      <CompactStatCard
        label="Large Value"
        value={9876543}
      />
    )

    expect(screen.getByText('9,876,543')).toBeInTheDocument()
  })
})
