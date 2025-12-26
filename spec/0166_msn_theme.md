# Spec 0166: MSN Messenger Windows Vista Theme

## Overview

Implement a pixel-perfect MSN Messenger theme for the messaging blueprint that authentically recreates the Windows Vista/Windows Live Messenger 8.x experience. This theme captures the distinctive Windows Aero glass aesthetic with translucent panels, gradient title bars, and the iconic green/blue MSN color scheme.

## Visual Design Reference

### Windows Vista Aero Glass Styling
- **Glass Effect**: Translucent panels with blur backdrop (CSS `backdrop-filter: blur()`)
- **Title Bar**: Dark gradient with glass effect, white text with subtle glow
- **Window Frame**: Subtle rounded corners (6-8px), soft drop shadows
- **Colors**: Vista blue (#0078D4), Vista green (#7FBA00 for online status)

### MSN Messenger Specific Elements
- **Logo**: Windows Live Messenger butterfly icon (green/orange gradient)
- **Contact List**: Hierarchical tree with expand/collapse groups
- **Status Icons**: Colored orb indicators (green=online, yellow=away, red=busy, gray=offline)
- **Chat Window**: Split view with contact info header, message area, and input toolbar
- **Emoticons**: Classic MSN emoticon style
- **Nudge**: Shake animation for nudge feature

## Color Palette

```css
:root {
  /* Windows Vista Aero */
  --vista-glass-bg: rgba(45, 45, 50, 0.75);
  --vista-glass-border: rgba(255, 255, 255, 0.3);
  --vista-titlebar-bg: linear-gradient(180deg, #3a3a3c 0%, #2a2a2c 100%);
  --vista-window-bg: #1e1e20;
  --vista-sidebar-bg: rgba(30, 30, 32, 0.95);

  /* MSN Brand Colors */
  --msn-green: #7FBA00;
  --msn-green-light: #9AD200;
  --msn-orange: #FF8C00;
  --msn-blue: #0078D4;
  --msn-blue-light: #00A4EF;

  /* Status Colors */
  --status-online: #7FBA00;
  --status-away: #FFB900;
  --status-busy: #E81123;
  --status-invisible: #767676;
  --status-offline: #505050;

  /* Text */
  --text-primary: #FFFFFF;
  --text-secondary: #B0B0B0;
  --text-muted: #808080;

  /* Message Bubbles */
  --bubble-sent: rgba(0, 120, 212, 0.85);
  --bubble-received: rgba(60, 60, 65, 0.9);
}
```

## Component Specifications

### 1. Main Window Layout
```
+--------------------------------------------------+
|  [MSN Icon] Windows Live Messenger    [_ □ X]    |  <- Glass titlebar
+--------------------------------------------------+
|  [Status Orb] Display Name v                     |  <- User panel
|  [Personal Message field]                        |
+--------------------------------------------------+
|  [Search contacts...]                            |  <- Search bar
+--------------------------------------------------+
|  - Favorites                                     |  <- Contact groups
|    ├─ [●] Contact 1                              |
|    └─ [●] Contact 2                              |
|  + Online (3/5)                                  |
|  + Offline                                       |
+--------------------------------------------------+
|  [Email] [Games] [Directory]                     |  <- Action bar
+--------------------------------------------------+
```

### 2. Chat Window Layout
```
+--------------------------------------------------+
|  [Avatar] Contact Name          [Call] [Video]   |  <- Contact header
|  Status: Available                               |
+--------------------------------------------------+
|                                                  |
|          [Received message bubble]               |  <- Message area
|                      [Sent message bubble]       |
|                                                  |
+--------------------------------------------------+
|  [Font] [Emoticons] [Nudge] [Files] [Games]     |  <- Toolbar
+--------------------------------------------------+
|  [Message input field]               [Send]      |  <- Input area
+--------------------------------------------------+
```

### 3. Window Chrome
- **Title Bar Height**: 30px
- **Window Border Radius**: 6px (top corners), 0px (bottom)
- **Control Buttons**: Minimize (−), Maximize (□), Close (×)
- **Glass Blur**: 12-20px blur radius
- **Shadow**: 0 8px 32px rgba(0, 0, 0, 0.4)

### 4. Status Orb Icons
Pure CSS gradient orbs with glow effects:
- Online: Green gradient with outer glow
- Away: Yellow/orange gradient
- Busy: Red gradient
- Invisible: Gray with subtle outline
- Offline: Dark gray, no glow

### 5. Typography
- **Primary Font**: Segoe UI (Vista system font)
- **Fallback**: Tahoma, Arial, sans-serif
- **Title Bar**: 12px, bold
- **Contact Names**: 12px, regular
- **Messages**: 11px, regular
- **Timestamps**: 10px, muted color

## File Structure

```
assets/
├── views/
│   └── msn/
│       ├── layouts/
│       │   └── default.html
│       └── pages/
│           ├── home.html
│           ├── login.html
│           ├── register.html
│           ├── app.html
│           └── settings.html
└── static/
    └── css/
        └── msn.css
```

## Integration Points

### 1. Theme Registration (embed.go)
Add `"msn"` to the Themes slice.

### 2. JavaScript Updates (app.js)
- Add `'msn'` to `THEMES` array
- Add `'msn'` to `VIEW_THEMES` array
- Add `'msn': 'MSN Messenger'` to `THEME_NAMES`

### 3. Theme Cycling Order
Update cycling order for better chronological flow:
```javascript
const THEMES = ['dark', 'light', 'aim1.0', 'ymxp', 'msn', 'im26', 'imos9', 'imosx', 'team11'];
```

This places MSN (2006-2010 era) between Yahoo XP (early 2000s) and iMessage (2010s+).

## Key Features

### Required Functionality
1. **Contact List**: Expandable groups with buddy counts
2. **Chat Selection**: Double-click to open chat
3. **Message Display**: Styled bubbles with timestamps
4. **Typing Indicator**: "Contact is typing..." status
5. **Emoji Picker**: Retro-styled with MSN emoticons
6. **Sticker Support**: Full sticker pack integration
7. **File Sharing**: Drag-drop and file picker
8. **Voice Messages**: Recording and playback
9. **Reactions**: Quick emoji reactions
10. **Theme Switcher**: FAB for cycling themes

### Vista-Specific Effects
1. **Glass Effect**: CSS `backdrop-filter: blur(12px)`
2. **Window Glow**: Subtle colored border glow
3. **Hover States**: Smooth transitions with highlight
4. **Nudge Animation**: CSS keyframe shake effect
5. **Status Animations**: Pulse effect on status change

## CSS Class Naming Convention

Prefix all MSN-specific classes with `msn-`:
- `.msn-window` - Main window container
- `.msn-titlebar` - Window title bar
- `.msn-contact-list` - Buddy list container
- `.msn-chat-window` - Chat panel
- `.msn-message` - Individual message
- `.msn-status-orb` - Status indicator
- `.msn-glass` - Glass effect mixin

## Modal Styling

All modals should match Vista dialog styling:
- Semi-transparent dark background
- Centered glass-effect dialog box
- Standard Vista button styling
- Close button in title bar

## Responsive Considerations

- Minimum width: 320px (mobile)
- Contact list collapses on narrow screens
- Chat window takes full width when contact list hidden
- Touch-friendly tap targets (44x44px minimum)

## Testing Checklist

- [ ] Theme loads correctly on first visit
- [ ] Theme persists across page reloads
- [ ] Smooth transition when cycling themes
- [ ] All modals display correctly
- [ ] Emoji picker works with retro styling
- [ ] Sticker picker works correctly
- [ ] Voice recording UI matches theme
- [ ] File upload works correctly
- [ ] WebSocket messages render properly
- [ ] Settings page displays all theme options
- [ ] Back navigation works correctly
- [ ] Logout functionality works
- [ ] Mobile responsive layout works

## Implementation Notes

1. Use CSS custom properties for all colors to enable easy theming
2. Implement glass effect with fallback for browsers without backdrop-filter
3. Include @supports queries for progressive enhancement
4. Test in all major browsers (Chrome, Firefox, Safari, Edge)
5. Ensure accessibility with proper contrast ratios
6. Add ARIA labels for screen reader support
