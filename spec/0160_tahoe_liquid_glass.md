# iMessage macOS Tahoe Liquid Glass Theme - Design Specification

## Overview

This document specifies the design language for the iMessage-inspired theme implementing Apple's macOS Tahoe "Liquid Glass" design system. The theme recreates the authentic iMessage experience with translucent frosted glass materials, subtle depth, and refined typography.

---

## 1. Design Philosophy

### 1.1 Core Principles
- **Translucency**: Layered frosted glass panels that reveal content beneath
- **Depth**: Subtle shadows and elevation to create spatial hierarchy
- **Clarity**: Clean, legible typography with generous whitespace
- **Fluidity**: Smooth transitions and micro-animations
- **Consistency**: Unified material language across all components

### 1.2 Material System
The Liquid Glass material consists of:
- Semi-transparent background with backdrop blur
- Subtle inner glow/highlight on edges
- Soft drop shadows
- Frosted glass effect using CSS `backdrop-filter`

---

## 2. Color Palette

### 2.1 Light Mode (Primary)

```css
/* Background Layers */
--bg-base: #f5f5f7;                          /* Desktop/Window background */
--bg-glass: rgba(255, 255, 255, 0.72);       /* Frosted glass panels */
--bg-glass-hover: rgba(255, 255, 255, 0.85); /* Hover state */
--bg-sidebar: rgba(245, 245, 247, 0.8);      /* Sidebar background */
--bg-content: rgba(255, 255, 255, 0.9);      /* Content areas */

/* Message Bubbles */
--bubble-sent: #007AFF;                      /* iMessage blue */
--bubble-sent-text: #FFFFFF;
--bubble-received: rgba(229, 229, 234, 0.9); /* Gray bubble */
--bubble-received-text: #1D1D1F;

/* Text Colors */
--text-primary: #1D1D1F;
--text-secondary: #6E6E73;
--text-tertiary: #AEAEB2;
--text-link: #007AFF;

/* Borders & Dividers */
--border-glass: rgba(0, 0, 0, 0.06);
--border-light: rgba(255, 255, 255, 0.5);
--divider: rgba(60, 60, 67, 0.12);

/* Accents */
--accent-blue: #007AFF;
--accent-green: #34C759;
--accent-red: #FF3B30;
--accent-orange: #FF9500;
--accent-purple: #AF52DE;

/* Shadows */
--shadow-sm: 0 1px 3px rgba(0, 0, 0, 0.08);
--shadow-md: 0 4px 12px rgba(0, 0, 0, 0.1);
--shadow-lg: 0 8px 32px rgba(0, 0, 0, 0.12);
--shadow-glass: 0 0 0 0.5px rgba(0, 0, 0, 0.05), 0 2px 8px rgba(0, 0, 0, 0.08);
```

### 2.2 Dark Mode

```css
/* Background Layers */
--bg-base: #1C1C1E;
--bg-glass: rgba(44, 44, 46, 0.72);
--bg-glass-hover: rgba(58, 58, 60, 0.85);
--bg-sidebar: rgba(28, 28, 30, 0.8);
--bg-content: rgba(44, 44, 46, 0.9);

/* Message Bubbles */
--bubble-sent: #0A84FF;
--bubble-sent-text: #FFFFFF;
--bubble-received: rgba(58, 58, 60, 0.9);
--bubble-received-text: #FFFFFF;

/* Text Colors */
--text-primary: #FFFFFF;
--text-secondary: #AEAEB2;
--text-tertiary: #636366;
--text-link: #0A84FF;

/* Borders & Dividers */
--border-glass: rgba(255, 255, 255, 0.08);
--border-light: rgba(255, 255, 255, 0.1);
--divider: rgba(84, 84, 88, 0.65);
```

---

## 3. Typography

### 3.1 Font Stack

```css
--font-system: -apple-system, BlinkMacSystemFont, 'SF Pro Display', 'SF Pro Text',
               'Helvetica Neue', 'Segoe UI', Arial, sans-serif;
--font-mono: 'SF Mono', SFMono-Regular, ui-monospace, Menlo, Monaco, monospace;
```

### 3.2 Type Scale

| Element | Size | Weight | Line Height | Letter Spacing |
|---------|------|--------|-------------|----------------|
| Window Title | 13px | 600 | 1.2 | -0.01em |
| Sidebar Header | 11px | 600 | 1.3 | 0.01em |
| Contact Name | 14px | 500 | 1.3 | -0.01em |
| Message Preview | 12px | 400 | 1.4 | 0 |
| Message Body | 15px | 400 | 1.4 | -0.01em |
| Timestamp | 11px | 400 | 1.2 | 0 |
| Button | 13px | 500 | 1 | -0.01em |
| Input | 14px | 400 | 1.4 | 0 |

---

## 4. Spacing System

```css
--space-1: 4px;
--space-2: 8px;
--space-3: 12px;
--space-4: 16px;
--space-5: 20px;
--space-6: 24px;
--space-8: 32px;
--space-10: 40px;
--space-12: 48px;
```

---

## 5. Component Specifications

### 5.1 Window Chrome

The macOS Tahoe window features:
- **Traffic Light Buttons**: Red (close), Yellow (minimize), Green (maximize)
  - Size: 12px diameter
  - Spacing: 8px between buttons
  - Position: 12px from left, vertically centered in titlebar
  - Hover: Show symbols (×, -, +)

- **Title Bar**
  - Height: 52px
  - Background: Frosted glass (`backdrop-filter: blur(20px) saturate(180%)`)
  - Border-bottom: 1px solid var(--divider)

- **Window Border Radius**: 10px

```css
.window {
  border-radius: 10px;
  box-shadow:
    0 0 0 0.5px rgba(0, 0, 0, 0.1),
    0 24px 68px rgba(0, 0, 0, 0.25),
    0 8px 20px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}
```

### 5.2 Sidebar (Conversations List)

- **Width**: 280px (resizable, min: 200px, max: 400px)
- **Background**: var(--bg-sidebar) with backdrop blur
- **Search Bar**
  - Height: 28px
  - Border-radius: 8px
  - Background: rgba(0, 0, 0, 0.06)
  - Placeholder: "Search" with magnifying glass icon

- **Conversation Item**
  - Height: 64px
  - Padding: 12px 16px
  - Avatar: 44px diameter, rounded
  - Hover: var(--bg-glass-hover)
  - Selected: var(--accent-blue) with opacity 0.15
  - Border-radius: 10px (for selection state)

```css
.conversation-item {
  display: flex;
  gap: 12px;
  padding: 10px 16px;
  border-radius: 10px;
  transition: background 0.15s ease;
}

.conversation-item:hover {
  background: rgba(0, 0, 0, 0.04);
}

.conversation-item.selected {
  background: rgba(0, 122, 255, 0.12);
}
```

### 5.3 Message Bubbles

- **Max Width**: 65% of container
- **Border Radius**: 18px (with tail: 18px 18px 4px 18px for sent)
- **Padding**: 8px 12px
- **Spacing Between Messages**: 2px (same sender), 8px (different sender)

**Sent Message (iMessage Blue)**
```css
.message-sent {
  background: linear-gradient(180deg, #1B8CFF 0%, #007AFF 100%);
  color: white;
  border-radius: 18px 18px 4px 18px;
  margin-left: auto;
}
```

**Received Message (Gray)**
```css
.message-received {
  background: var(--bubble-received);
  color: var(--text-primary);
  border-radius: 18px 18px 18px 4px;
  margin-right: auto;
}
```

**Message Tail**
- SVG tail element positioned at bottom corner
- Matches bubble background color
- Size: 10px × 10px

### 5.4 Message Input Area

- **Container Height**: 56px minimum (auto-expands)
- **Background**: Frosted glass
- **Input Field**
  - Background: rgba(0, 0, 0, 0.05)
  - Border-radius: 20px
  - Padding: 10px 16px
  - Min-height: 36px
  - Max-height: 120px

- **Send Button**
  - Size: 30px diameter
  - Background: var(--accent-blue)
  - Icon: Arrow up, white
  - Position: Right side of input

```css
.message-input-container {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  padding: 10px 16px;
  background: var(--bg-glass);
  backdrop-filter: blur(20px);
  border-top: 1px solid var(--divider);
}

.message-input {
  flex: 1;
  background: rgba(0, 0, 0, 0.05);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 20px;
  padding: 8px 16px;
  font-size: 15px;
  resize: none;
  outline: none;
}

.send-button {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--accent-blue);
  border: none;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: transform 0.1s, background 0.15s;
}

.send-button:hover {
  background: #0066D6;
}

.send-button:active {
  transform: scale(0.92);
}
```

### 5.5 Avatar Component

```css
.avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: linear-gradient(135deg, #5856D6 0%, #AF52DE 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  font-weight: 600;
  color: white;
  flex-shrink: 0;
}

.avatar img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  object-fit: cover;
}

/* Online indicator */
.avatar::after {
  content: '';
  position: absolute;
  bottom: 0;
  right: 0;
  width: 12px;
  height: 12px;
  background: var(--accent-green);
  border: 2px solid white;
  border-radius: 50%;
}
```

### 5.6 Buttons

**Primary Button**
```css
.button-primary {
  background: var(--accent-blue);
  color: white;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 13px;
  font-weight: 500;
  border: none;
  cursor: pointer;
  transition: background 0.15s, transform 0.1s;
}

.button-primary:hover {
  background: #0066D6;
}

.button-primary:active {
  transform: scale(0.97);
}
```

**Glass Button**
```css
.button-glass {
  background: rgba(0, 0, 0, 0.05);
  color: var(--text-primary);
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid var(--border-glass);
  backdrop-filter: blur(10px);
  transition: background 0.15s;
}

.button-glass:hover {
  background: rgba(0, 0, 0, 0.08);
}
```

### 5.7 Form Inputs

```css
.input {
  width: 100%;
  padding: 10px 14px;
  background: rgba(0, 0, 0, 0.04);
  border: 1px solid var(--border-glass);
  border-radius: 8px;
  font-size: 14px;
  color: var(--text-primary);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.input:focus {
  outline: none;
  border-color: var(--accent-blue);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.15);
}

.input::placeholder {
  color: var(--text-tertiary);
}
```

---

## 6. Icons

### 6.1 SF Symbols Style
All icons follow Apple SF Symbols guidelines:
- Stroke width: 1.5px
- Rounded caps and joins
- Optical sizing

### 6.2 Key Icons
- **Compose**: Plus sign in circle
- **Search**: Magnifying glass
- **Send**: Arrow pointing up
- **Attach**: Paperclip or Plus
- **Emoji**: Smiley face
- **Voice**: Microphone
- **Settings**: Gear
- **Back**: Chevron left

---

## 7. Animations & Transitions

### 7.1 Timing Functions
```css
--ease-default: cubic-bezier(0.25, 0.1, 0.25, 1);
--ease-out: cubic-bezier(0, 0, 0.2, 1);
--ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);
--spring: cubic-bezier(0.34, 1.56, 0.64, 1);
```

### 7.2 Duration Scale
- **Micro**: 100ms (button press)
- **Fast**: 150ms (hover states)
- **Normal**: 200ms (panel transitions)
- **Slow**: 300ms (modal/overlay)
- **Deliberate**: 400ms (page transitions)

### 7.3 Message Animation
```css
@keyframes message-appear {
  from {
    opacity: 0;
    transform: translateY(8px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

.message-new {
  animation: message-appear 0.25s var(--spring);
}
```

---

## 8. Layout Structure

### 8.1 Main App Layout
```
+------------------------------------------------------------------+
|  [Traffic Lights]          Messages                    [+ New]   |
+------------------------------------------------------------------+
|                    |                                              |
|  [Search]          |           Contact Name                      |
|                    |                                              |
|  +------------+    |   +---------------------------------+        |
|  | Avatar  N  |    |   | Received message bubble        |        |
|  | Preview... |    |   +---------------------------------+        |
|  +------------+    |                                              |
|                    |        +---------------------------------+   |
|  +------------+    |        |         Sent message bubble    |   |
|  | Avatar  N  |    |        +---------------------------------+   |
|  | Preview... |    |                                              |
|  +------------+    |                                              |
|                    |------------------------------------------------
|                    |  [+] [Message input...               ] [->]  |
+--------------------+----------------------------------------------+
     280px                            Flexible
```

### 8.2 Responsive Breakpoints
- **Desktop**: > 900px (sidebar + content)
- **Tablet**: 600-900px (overlay sidebar)
- **Mobile**: < 600px (full-screen views)

---

## 9. Accessibility

### 9.1 Focus States
```css
:focus-visible {
  outline: 2px solid var(--accent-blue);
  outline-offset: 2px;
}
```

### 9.2 Color Contrast
- All text meets WCAG AA (4.5:1 for body, 3:1 for large text)
- Interactive elements have visible focus indicators
- Status indicators use color + icon/shape

### 9.3 Motion
```css
@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    animation-duration: 0.01ms !important;
    transition-duration: 0.01ms !important;
  }
}
```

---

## 10. Implementation Notes

### 10.1 Backdrop Filter Support
Provide fallback for browsers without `backdrop-filter`:
```css
.glass-panel {
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(20px) saturate(180%);
}

@supports not (backdrop-filter: blur(20px)) {
  .glass-panel {
    background: rgba(255, 255, 255, 0.97);
  }
}
```

### 10.2 Theme Identifier
```css
[data-theme="im26"] { /* Light mode */ }
[data-theme="im26"][data-color-scheme="dark"] { /* Dark mode */ }
```

### 10.3 File Structure
```
assets/views/im26/
  layouts/
    default.html
  pages/
    home.html
    login.html
    register.html
    settings.html
    app.html

assets/static/css/
  imessage.css
```

---

## 11. Reference Screenshots

The design replicates these key iMessage elements:
1. Frosted glass sidebar with conversation list
2. Blue gradient sent messages with tail
3. Gray received messages
4. Traffic light window controls
5. Pill-shaped message input with plus and send buttons
6. Circular avatar with gradient fallback
7. Subtle shadows and material depth
8. SF Pro typography throughout

---

**Version**: 1.0
**Theme ID**: im26
**Compatibility**: macOS Tahoe style (2024+)
