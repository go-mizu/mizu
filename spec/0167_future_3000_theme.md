# Future 3000 Theme - Jarvis-Inspired UI

## Overview

The Year 3000 theme is a futuristic, holographic interface inspired by the JARVIS (Just A Rather Very Intelligent System) UI from the Iron Man films. This theme represents human-computer interaction 1000 years into the future—clean, intuitive, and powered by advanced AI visualization.

## Design Philosophy

### Core Principles

1. **Holographic Transparency** - All UI elements appear to float in space with translucent, glass-morphic surfaces
2. **Ambient Intelligence** - The interface feels alive with subtle animations, pulses, and reactive elements
3. **Minimal Cognitive Load** - Despite being futuristic, every interaction is intuitive and effortless
4. **Spatial Hierarchy** - Z-depth and glow effects create clear visual hierarchy without traditional borders

### Visual Language

The Jarvis aesthetic combines:
- **Dark void backgrounds** - Deep space black (#0a0a0f) as the canvas
- **Cyan/electric blue accents** - Primary interaction color (#00d4ff)
- **Warm orange highlights** - Secondary accent for alerts and AI responses (#ff6b35)
- **Holographic gradients** - Subtle rainbow refractions on glass surfaces
- **Geometric precision** - Hexagonal patterns, arc segments, and radial designs

## Color Palette

### Primary Colors
```css
--void-black: #0a0a0f;        /* Deep space background */
--void-surface: #0d1117;      /* Elevated surface */
--holo-cyan: #00d4ff;         /* Primary accent - interaction */
--holo-blue: #0066ff;         /* Secondary blue */
--holo-teal: #00ffc8;         /* Success/online states */
--holo-orange: #ff6b35;       /* AI/system accent */
--holo-magenta: #ff00aa;      /* Notification accent */
```

### Glass Surfaces
```css
--glass-bg: rgba(13, 17, 23, 0.7);
--glass-border: rgba(0, 212, 255, 0.2);
--glass-highlight: rgba(255, 255, 255, 0.05);
--glass-glow: 0 0 30px rgba(0, 212, 255, 0.3);
```

### Text Hierarchy
```css
--text-primary: #ffffff;
--text-secondary: rgba(255, 255, 255, 0.7);
--text-tertiary: rgba(255, 255, 255, 0.4);
--text-glow: 0 0 10px rgba(0, 212, 255, 0.5);
```

## Typography

### Font Stack
- **Primary**: "Rajdhani", "Orbitron", sans-serif - Geometric, futuristic
- **Monospace**: "JetBrains Mono", "Fira Code", monospace - Data readouts
- **Display**: "Audiowide" - Headers and system titles

### Type Scales
- **H1 (System Title)**: 2.5rem, 100 weight, letter-spacing: 0.3em, uppercase
- **H2 (Section Header)**: 1.5rem, 500 weight, letter-spacing: 0.15em
- **Body**: 0.95rem, 400 weight, letter-spacing: 0.02em
- **Caption/Meta**: 0.75rem, 300 weight, letter-spacing: 0.05em

## UI Components

### 1. Holographic Panels

Floating panels with:
- Translucent glass background with backdrop blur (20px)
- Subtle cyan border glow
- Corner accent marks (geometric L-shapes)
- Subtle scan-line animation overlay

```
┌─────────────────────────────────────┐
│ ┌─                               ─┐ │
│                                     │
│         PANEL CONTENT               │
│                                     │
│ └─                               ─┘ │
└─────────────────────────────────────┘
```

### 2. Arc Status Indicators

Circular/arc-based status indicators inspired by Jarvis HUD:
- User avatar surrounded by rotating status ring
- Online status as animated arc segment
- Message counts in circular badges

```
      ╭────╮
    ╱        ╲
   │    ◯    │  ← Avatar with rotating ring
    ╲        ╱
      ╰────╯
```

### 3. Message Bubbles

**Sent Messages (User)**:
- Right-aligned
- Subtle cyan gradient background
- Geometric corner accent
- Timestamp with data-style formatting

**Received Messages (Contact/AI)**:
- Left-aligned
- Glass surface with orange accent line
- AI responses have pulsing indicator

```
┌──────────────────────────────┐
│░▒ INCOMING TRANSMISSION      │
│   "Hello from the future"    │
│                    ⌐ 14:32:07│
└──────────────────────────────┘
```

### 4. Contact List (Neural Network View)

Instead of a traditional list:
- Contacts displayed in a hexagonal grid pattern
- Active conversations glow brighter
- Hover reveals full name and status
- Connection lines between related contacts

### 5. Input Field (Command Interface)

- Spans full width with minimal chrome
- Pulsing cursor animation
- Voice input indicator (sound wave visualization)
- Predictive text appears as ghost text

```
┌─ NEURAL LINK ACTIVE ──────────────────────────────┐
│ > Type your message...                         ⎔  │
│   ▃▅▇▅▃ VOICE                            [SEND]   │
└───────────────────────────────────────────────────┘
```

### 6. Buttons

**Primary Button**:
- Cyan gradient background
- Subtle glow on hover
- Geometric shape (slightly angled corners)
- Text with slight glow

**Secondary Button**:
- Transparent with cyan border
- Fill animation on hover
- Ripple effect on click

**Icon Button**:
- Circular with glass background
- Icon glows on hover
- Rotation animation for loading states

### 7. Scrollbars

- Ultra-thin (4px) with glass appearance
- Glow effect when scrolling
- Fades to near-invisible when inactive

### 8. Emoji/Sticker Picker

- Holographic modal overlay
- Categories displayed as arc segments
- Items float in 3D grid
- Selection creates ripple effect

## Animations

### Ambient Animations
1. **Scan Line**: Subtle horizontal line that sweeps down the screen every 10s
2. **Grid Pulse**: Background grid pattern pulses softly
3. **Particle Field**: Floating particles in background (optional, low density)

### Interaction Animations
1. **Hover Glow**: Elements brighten and gain glow on hover (200ms ease)
2. **Click Ripple**: Cyan ripple from click point (400ms ease-out)
3. **Focus Ring**: Animated dashed border that rotates
4. **Send Animation**: Message shoots up with trail effect

### State Transitions
1. **Panel Enter**: Fade in + slight scale up (300ms cubic-bezier)
2. **Modal Open**: Zoom from center with blur background
3. **Toast/Notification**: Slide in from top-right with glow pulse

## Layout Structure

### Main App Layout

```
┌──────────────────────────────────────────────────────────────┐
│  ◈ MIZU NEURAL NETWORK          ▲ USER ◯ ⚙ SETTINGS         │
├─────────────────────┬────────────────────────────────────────┤
│                     │                                        │
│   ⬡ ⬡ ⬡            │  ┌─ ACTIVE CHANNEL ─────────────────┐  │
│    ⬡ ⬡             │  │                                   │  │
│   ⬡ ⬡ ⬡            │  │  MESSAGE STREAM                   │  │
│  CONTACTS           │  │                                   │  │
│  HEXGRID            │  │  ░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░  │  │
│                     │  │                                   │  │
│   ──────            │  │                                   │  │
│   SYSTEM STATUS     │  └───────────────────────────────────┘  │
│   ◐ AI: ONLINE      │                                        │
│   ◉ SYNC: 100%      │  ┌─ NEURAL INPUT ────────────────────┐  │
│                     │  │ > _                          [▶]  │  │
│                     │  └───────────────────────────────────┘  │
└─────────────────────┴────────────────────────────────────────┘
```

### Login/Register Layout

```
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│                                                              │
│              ╭──────────────────────╮                        │
│              │                      │                        │
│              │   ◈ MIZU             │                        │
│              │   NEURAL ACCESS      │                        │
│              │                      │                        │
│              │   ┌──────────────┐   │                        │
│              │   │ IDENTITY     │   │                        │
│              │   └──────────────┘   │                        │
│              │   ┌──────────────┐   │                        │
│              │   │ PASSKEY      │   │                        │
│              │   └──────────────┘   │                        │
│              │                      │                        │
│              │   [ AUTHENTICATE ]   │                        │
│              │                      │                        │
│              ╰──────────────────────╯                        │
│                                                              │
│   ▃▅▇▅▃▅▇▅▃   BIOMETRIC READY   ▃▅▇▅▃▅▇▅▃                    │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

### Settings Layout

```
┌──────────────────────────────────────────────────────────────┐
│  ◈ SYSTEM CONFIGURATION                              [CLOSE] │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─ NEURAL PROFILE ─────────────────────────────────────┐    │
│  │  ◯ AVATAR        USERNAME: future_user               │    │
│  │  ╭───╮           STATUS: ● ONLINE                    │    │
│  │  │   │           BIO: Living in the year 3000        │    │
│  │  ╰───╯                                               │    │
│  └───────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─ INTERFACE SETTINGS ─────────────────────────────────┐    │
│  │                                                       │    │
│  │  THEME ENGINE     [ ◉ JARVIS  ○ LEGACY  ○ MINIMAL ]  │    │
│  │  HAPTIC FEEDBACK  [ ═══════◉══ ]                     │    │
│  │  TRANSPARENCY     [ ════◉═════ ]                     │    │
│  │  PARTICLE DENSITY [ ══◉═══════ ]                     │    │
│  │                                                       │    │
│  └───────────────────────────────────────────────────────┘    │
│                                                              │
│  ┌─ PRIVACY & SECURITY ─────────────────────────────────┐    │
│  │  ENCRYPTION: QUANTUM-256     ◉ ACTIVE                │    │
│  │  NEURAL LINK: AUTHORIZED     ◉ CONNECTED             │    │
│  └───────────────────────────────────────────────────────┘    │
│                                                              │
│                           [ SAVE CONFIGURATION ]             │
│                           [ DISCONNECT SESSION ]             │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

## Accessibility Considerations

Despite the futuristic aesthetic, accessibility remains paramount:

1. **Color Contrast**: All text meets WCAG AA standards against backgrounds
2. **Focus Indicators**: Clear, visible focus states with animated rings
3. **Motion Reduction**: Respects `prefers-reduced-motion` - disables ambient animations
4. **Screen Reader Support**: Proper ARIA labels and semantic HTML
5. **Keyboard Navigation**: Full keyboard support with visible focus states

## Technical Implementation

### CSS Architecture
- CSS custom properties for all colors and values
- `@keyframes` for ambient animations
- `backdrop-filter: blur()` for glass effect
- CSS Grid for hexagonal layouts
- `clip-path` for geometric shapes

### Required Assets
- Rajdhani font (Google Fonts)
- Custom SVG icons for futuristic UI elements
- Optional: Particle effect canvas overlay

### Performance Considerations
- Animations use `transform` and `opacity` only (GPU accelerated)
- Backdrop blur limited to key elements (performance intensive)
- Particle effects are optional and can be disabled
- Prefers-reduced-motion query disables all non-essential animations

## Implementation Checklist

- [ ] CSS file with all variables and component styles
- [ ] Layout template with proper head/script setup
- [ ] Home page with futuristic landing design
- [ ] Login page with neural access theme
- [ ] Register page with identity creation flow
- [ ] App page with full messaging interface
- [ ] Settings page with configuration panels
- [ ] Theme registration in embed.go
- [ ] E2E test updates for new theme

## Inspiration Sources

1. **Iron Man JARVIS UI** - Primary inspiration for holographic aesthetic
2. **Minority Report** - Gesture-based interaction visualization
3. **Tron Legacy** - Geometric patterns and light trails
4. **Blade Runner 2049** - Color palette and atmospheric depth
5. **Westworld** - Clean, minimal futuristic UI
6. **Interstellar** - Data visualization and HUD elements

---

*"The future is not something we enter. The future is something we create."* — Year 3000
