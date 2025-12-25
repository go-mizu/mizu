# Spec 0154: Yahoo Messenger Windows XP Theme

**Status:** Implemented

## Overview

Implement an authentic Yahoo! Messenger 6.0-7.0 (circa 2004-2006) theme for the messaging blueprint that recreates the Windows XP Luna aesthetic with Yahoo Messenger's distinctive purple/maroon styling.

## Visual Reference

Yahoo Messenger on Windows XP had these distinctive characteristics:

### Windows XP Luna Theme Colors

The Windows XP Luna (Blue) theme provides the base OS chrome:

- **Title Bar Gradient**: #0054E3 → #3C8DED → #0054E3 (Royal blue gradient)
- **Title Bar Text**: #FFFFFF (White, bold)
- **Window Background**: #ECE9D8 (XP cream/tan)
- **Button Face**: #ECE9D8 (Cream)
- **Button Hover**: #B8D6FB (Light blue highlight)
- **Button Pressed**: #98B6E9 (Pressed blue)
- **Button Border**: #003C74 (Dark blue)
- **Selected Item**: #316AC5 (Selection blue)
- **Menu Background**: #FFFFFF
- **Menu Highlight**: #316AC5 (Blue selection)
- **Scrollbar Track**: #F1EFE2
- **Scrollbar Thumb**: #CDCAC3 with 3D borders
- **Desktop**: #004E98 (Classic XP blue)

### Yahoo Messenger Specific Colors

Yahoo Messenger's distinctive branding colors:

- **Yahoo Purple/Maroon**: #4C1130 (Dark burgundy header)
- **Yahoo Purple Light**: #72234B (Lighter purple)
- **Messenger Yellow**: #FFCC00 (Yahoo Yellow accent)
- **Online Status**: #008000 (Green)
- **Busy Status**: #D4361C (Red)
- **Invisible Status**: #808080 (Gray)
- **Away Status**: #FFA500 (Orange)
- **Stealth Status**: #666666 (Dark gray)
- **Chat Bubble Sent**: #D4E8FC (Light blue)
- **Chat Bubble Received**: #FFFFC2 (Pale yellow)
- **Buzz Color**: #FF0000 (Red flash)
- **Link Color**: #0066CC (Blue links)
- **IMVironment BG**: Various gradients

### Typography

- **Window Font**: "Tahoma", "Segoe UI", sans-serif (XP default)
- **Message Font**: "Arial", sans-serif
- **Base Size**: 11px (8pt Windows standard)
- **Title Bar**: 11px bold
- **Menu Items**: 11px regular
- **Status Bar**: 11px regular
- **Chat Messages**: 12px (user configurable in real Y!M)

### UI Elements

#### Windows XP Chrome
- Rounded window corners (2px radius on outer frame)
- Blue gradient title bars with fade effect
- Luna-style close/minimize/maximize buttons (rounded, colored)
- XP-style scrollbars with 3D appearance
- Soft drop shadows on windows
- XP button styling with rounded corners and hover states

#### Yahoo Messenger Specific
- Purple gradient header bar with Yahoo! logo
- Buddy list with categorized groups (Friends, Family, Co-Workers)
- Webcam icon indicators
- Smiley/emoticon picker
- Buzz button (!)
- Status dropdown with custom message
- Tab-based chat interface
- Avatar/profile picture displays
- IMVironment backgrounds in chat
- Audible emoticons (visual indicators)
- File transfer progress bars
- Voice/Video call buttons

### Iconic Visual Elements

1. **Smiley Face Icon**: Yellow circle face - Yahoo's signature
2. **Buzz Animation**: Red wavy line effect
3. **Online Indicator**: Green circle/ball
4. **Away Indicator**: Orange clock icon
5. **Busy Indicator**: Red circle with minus
6. **Invisible Indicator**: Gray outlined circle
7. **New Message**: Bouncing envelope
8. **Typing Indicator**: Animated pencil

## Implementation Plan

### 1. Directory Structure

```
assets/
├── static/
│   ├── css/
│   │   ├── default.css      # Modern theme
│   │   ├── aim.css          # AIM 1.0 theme
│   │   └── ymxp.css         # Yahoo Messenger XP theme (NEW)
│   └── js/
│       └── app.js           # Theme switching (update)
└── views/
    ├── default/             # Modern theme views
    ├── aim1.0/              # AIM theme views
    └── ymxp/                # Yahoo Messenger XP views (NEW)
        ├── layouts/
        │   └── default.html
        ├── pages/
        │   ├── app.html     # Main messenger interface
        │   ├── home.html    # Welcome/landing page
        │   ├── login.html   # Sign-in dialog
        │   ├── register.html # Create account dialog
        │   └── settings.html # Preferences window
        └── components/      # Reusable components
```

### 2. CSS Implementation

#### 2.1 Windows XP Luna Variables
```css
:root {
    /* Windows XP Luna Blue Theme */
    --xp-desktop: #004E98;
    --xp-window-bg: #ECE9D8;
    --xp-window-border: #0054E3;
    --xp-titlebar-start: #0054E3;
    --xp-titlebar-mid: #3C8DED;
    --xp-titlebar-end: #0054E3;
    --xp-titlebar-inactive: #7996C8;
    --xp-titlebar-text: #FFFFFF;
    --xp-button-face: #ECE9D8;
    --xp-button-hover: #B8D6FB;
    --xp-button-pressed: #98B6E9;
    --xp-button-border: #003C74;
    --xp-button-shadow: #ACA899;
    --xp-selection: #316AC5;
    --xp-selection-text: #FFFFFF;
    --xp-menu-bg: #FFFFFF;
    --xp-menu-border: #ACA899;
    --xp-input-bg: #FFFFFF;
    --xp-input-border: #7F9DB9;
    --xp-scrollbar-track: #F1EFE2;
    --xp-scrollbar-thumb: #CDCAC3;
    --xp-text: #000000;
    --xp-text-disabled: #ACA899;
    --xp-link: #0066CC;

    /* Close/Min/Max buttons */
    --xp-close-start: #C32B0E;
    --xp-close-end: #D67462;
    --xp-close-hover: #E35644;
    --xp-minmax-start: #0058EE;
    --xp-minmax-end: #3C8DED;

    /* Yahoo Messenger Branding */
    --ym-purple: #4C1130;
    --ym-purple-light: #72234B;
    --ym-purple-dark: #2A0A1C;
    --ym-yellow: #FFCC00;
    --ym-yellow-light: #FFF0A0;
    --ym-header-gradient-start: #72234B;
    --ym-header-gradient-end: #4C1130;

    /* Status Colors */
    --ym-online: #008000;
    --ym-away: #FFA500;
    --ym-busy: #D4361C;
    --ym-invisible: #808080;
    --ym-offline: #B0B0B0;

    /* Chat Colors */
    --ym-sent-bubble: #D4E8FC;
    --ym-received-bubble: #FFFFC2;
    --ym-buzz: #FF0000;
    --ym-system-msg: #808080;
}
```

#### 2.2 XP Window Chrome
```css
.xp-window {
    background: var(--xp-window-bg);
    border: 1px solid var(--xp-window-border);
    border-radius: 8px 8px 0 0;
    box-shadow:
        0 0 0 1px rgba(0, 0, 0, 0.3),
        2px 2px 10px rgba(0, 0, 0, 0.3);
    overflow: hidden;
}

.xp-titlebar {
    background: linear-gradient(180deg,
        var(--xp-titlebar-start) 0%,
        var(--xp-titlebar-mid) 45%,
        var(--xp-titlebar-mid) 55%,
        var(--xp-titlebar-start) 100%);
    color: var(--xp-titlebar-text);
    font-family: "Tahoma", sans-serif;
    font-size: 11px;
    font-weight: bold;
    padding: 4px 6px;
    display: flex;
    align-items: center;
    gap: 4px;
    border-radius: 6px 6px 0 0;
    text-shadow: 1px 1px 1px rgba(0, 0, 0, 0.3);
}

.xp-titlebar-buttons {
    display: flex;
    gap: 2px;
    margin-left: auto;
}

.xp-btn-minimize,
.xp-btn-maximize {
    width: 21px;
    height: 21px;
    border: none;
    border-radius: 3px;
    background: linear-gradient(180deg,
        var(--xp-minmax-start) 0%,
        var(--xp-minmax-end) 50%,
        var(--xp-minmax-start) 100%);
    cursor: pointer;
}

.xp-btn-close {
    width: 21px;
    height: 21px;
    border: none;
    border-radius: 3px;
    background: linear-gradient(180deg,
        var(--xp-close-start) 0%,
        var(--xp-close-end) 50%,
        var(--xp-close-start) 100%);
    cursor: pointer;
}
```

#### 2.3 Yahoo Header Bar
```css
.ym-header {
    background: linear-gradient(180deg,
        var(--ym-header-gradient-start) 0%,
        var(--ym-header-gradient-end) 100%);
    color: white;
    padding: 8px 12px;
    display: flex;
    align-items: center;
    gap: 12px;
}

.ym-logo {
    font-family: "Arial Black", "Arial", sans-serif;
    font-size: 16px;
    font-weight: bold;
    color: #FFFFFF;
    text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.5);
}

.ym-logo::before {
    content: "YAHOO!";
    color: var(--ym-yellow);
}
```

#### 2.4 XP Buttons
```css
.xp-button {
    font-family: "Tahoma", sans-serif;
    font-size: 11px;
    padding: 4px 16px;
    background: linear-gradient(180deg,
        #FFFFFF 0%,
        var(--xp-button-face) 45%,
        var(--xp-button-face) 100%);
    border: 1px solid var(--xp-button-border);
    border-radius: 3px;
    box-shadow: 0 1px 0 var(--xp-button-shadow);
    cursor: pointer;
}

.xp-button:hover {
    background: linear-gradient(180deg,
        #FFFFFF 0%,
        var(--xp-button-hover) 45%,
        var(--xp-button-hover) 100%);
    border-color: var(--xp-selection);
}

.xp-button:active {
    background: var(--xp-button-pressed);
    box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.2);
}

.xp-button:focus {
    outline: 1px dotted #000;
    outline-offset: -4px;
}
```

#### 2.5 XP Input Fields
```css
.xp-input {
    font-family: "Tahoma", sans-serif;
    font-size: 11px;
    padding: 3px 4px;
    background: var(--xp-input-bg);
    border: 1px solid var(--xp-input-border);
    border-radius: 0;
}

.xp-input:focus {
    outline: none;
    border-color: var(--xp-selection);
    box-shadow: 0 0 0 1px var(--xp-selection);
}
```

#### 2.6 XP Scrollbars
```css
.xp-scrollbar::-webkit-scrollbar {
    width: 17px;
    height: 17px;
}

.xp-scrollbar::-webkit-scrollbar-track {
    background: var(--xp-scrollbar-track);
    border: 1px solid #D2CFC5;
}

.xp-scrollbar::-webkit-scrollbar-thumb {
    background: linear-gradient(90deg,
        #F0EDE5 0%,
        var(--xp-scrollbar-thumb) 20%,
        var(--xp-scrollbar-thumb) 80%,
        #A3A09A 100%);
    border: 1px solid #9D9A93;
    border-radius: 0;
}

.xp-scrollbar::-webkit-scrollbar-button {
    background: var(--xp-scrollbar-track);
    border: 1px solid #D2CFC5;
    width: 17px;
    height: 17px;
}
```

### 3. Component Specifications

#### 3.1 Buddy List Window
```
┌─────────────────────────────────────────┐
│ 🔵 Yahoo! Messenger         _ □ X │
├─────────────────────────────────────────┤
│ ┌─────────────────────────────────────┐ │
│ │ [Avatar]  Display Name              │ │
│ │           📝 Status Message         │ │
│ │           [Status ▼] [Change...]    │ │
│ └─────────────────────────────────────┘ │
├─────────────────────────────────────────┤
│ Messenger   Contacts   Actions   Help   │
├─────────────────────────────────────────┤
│ ▼ Friends (3/5)                         │
│   ● John Smith                          │
│   ● Jane Doe                            │
│   ◐ Mike Wilson (Away)                  │
│   ○ Sarah Connor (Offline)              │
│   ○ Bob Martin (Offline)                │
│                                         │
│ ▼ Family (1/2)                          │
│   ● Mom                                 │
│   ○ Dad (Offline)                       │
│                                         │
│ ▼ Co-Workers (0/3)                      │
│   ○ Boss (Offline)                      │
│   ○ Colleague 1 (Offline)               │
│   ○ Colleague 2 (Offline)               │
│                                         │
├─────────────────────────────────────────┤
│ [🙂] [📧] [📞] [🔍]    New Message...   │
└─────────────────────────────────────────┘
```

#### 3.2 Chat Window
```
┌─────────────────────────────────────────────────┐
│ 🔵 John Smith - Instant Message     _ □ X │
├─────────────────────────────────────────────────┤
│ Messenger  Edit  Insert  Actions  Help          │
├─────────────────────────────────────────────────┤
│ [📷 Avatar] John Smith      [🔔] [📹] [📞] [!] │
│             ● Available                          │
├─────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────┐ │
│ │                                             │ │
│ │ John (10:30 AM):                            │ │
│ │ ┌─────────────────────────────┐             │ │
│ │ │ Hey, how are you? :)       │             │ │
│ │ └─────────────────────────────┘             │ │
│ │                                             │ │
│ │                    Me (10:31 AM):           │ │
│ │             ┌─────────────────────────────┐ │ │
│ │             │ I'm doing great! Thanks     │ │ │
│ │             │ for asking :D               │ │ │
│ │             └─────────────────────────────┘ │ │
│ │                                             │ │
│ └─────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────┤
│ Font: [Arial ▼] Size: [12 ▼] [B][I][U] [🎨]    │
├─────────────────────────────────────────────────┤
│ ┌─────────────────────────────────────────────┐ │
│ │ Type your message here...                   │ │
│ │                                             │ │
│ └─────────────────────────────────────────────┘ │
├─────────────────────────────────────────────────┤
│ [😀 Emoticons] [🔊 Audibles] [📎]    [  Send  ]│
└─────────────────────────────────────────────────┘
```

#### 3.3 Status Icons (SVG)
```css
.ym-status-online::before {
    content: "";
    display: inline-block;
    width: 10px;
    height: 10px;
    background: radial-gradient(circle at 30% 30%,
        #00FF00 0%, #008000 100%);
    border-radius: 50%;
    border: 1px solid #006600;
}

.ym-status-away::before {
    /* Orange clock icon */
}

.ym-status-busy::before {
    /* Red minus circle */
}

.ym-status-invisible::before {
    /* Gray outlined circle */
}
```

#### 3.4 Buzz Effect
```css
@keyframes ym-buzz {
    0%, 100% { transform: translateX(0); }
    10%, 30%, 50%, 70%, 90% { transform: translateX(-5px); }
    20%, 40%, 60%, 80% { transform: translateX(5px); }
}

.ym-buzz-active {
    animation: ym-buzz 0.5s ease-in-out;
    background-color: rgba(255, 0, 0, 0.1);
}

.ym-buzz-button {
    color: var(--ym-buzz);
    font-weight: bold;
    font-size: 14px;
}
```

#### 3.5 Emoticon Picker
```css
.ym-emoticon-picker {
    background: var(--xp-window-bg);
    border: 1px solid var(--xp-menu-border);
    box-shadow: 2px 2px 5px rgba(0, 0, 0, 0.2);
    padding: 8px;
    display: grid;
    grid-template-columns: repeat(8, 24px);
    gap: 2px;
}

.ym-emoticon {
    width: 24px;
    height: 24px;
    cursor: pointer;
    border: 1px solid transparent;
    border-radius: 2px;
}

.ym-emoticon:hover {
    background: var(--xp-button-hover);
    border-color: var(--xp-selection);
}
```

### 4. Page Implementations

#### 4.1 home.html
- Yahoo! Messenger splash screen
- "Sign In" prominent button
- "Get Yahoo! ID" link
- Classic Y! branding with yellow exclamation mark
- Windows XP window chrome

#### 4.2 login.html
- "Sign in to Yahoo!" dialog box
- Yahoo! ID field
- Password field with "Remember my ID & password" checkbox
- "Sign In" button (XP styled)
- "Invisible Sign In" checkbox
- Links: "Forgot Password?", "Create New Account"
- XP dialog styling

#### 4.3 register.html
- "Create a Yahoo! ID" wizard-style dialog
- Step indicators (1 of 3, etc.)
- Form fields with XP styling
- Terms of Service checkbox
- "Continue" and "Cancel" buttons

#### 4.4 app.html
- Split-pane layout (buddy list + chat window)
- Resizable divider
- Tabbed chat interface
- Full Yahoo Messenger chrome
- Status bar at bottom

#### 4.5 settings.html
- "Preferences" dialog with XP tabs
- Tab categories: General, Messages, Alerts, Privacy
- XP checkbox and radio button styling
- Apply/OK/Cancel button row

### 5. JavaScript Enhancements

#### 5.1 Buzz Feature
```javascript
function sendBuzz(chatId) {
    // Send buzz via WebSocket
    ws.send(JSON.stringify({
        type: 'buzz',
        chatId: chatId
    }));

    // Local visual feedback
    triggerBuzzAnimation();
}

function triggerBuzzAnimation() {
    const chatWindow = document.querySelector('.ym-chat-window');
    chatWindow.classList.add('ym-buzz-active');

    // Play buzz sound (if enabled)
    playSound('buzz');

    setTimeout(() => {
        chatWindow.classList.remove('ym-buzz-active');
    }, 500);
}
```

#### 5.2 IMVironment Support
```javascript
function setIMVironment(bgId) {
    const chatArea = document.querySelector('.ym-chat-area');
    chatArea.style.backgroundImage = `url('/static/img/imv/${bgId}.png')`;
    chatArea.classList.add('ym-imv-active');
}
```

#### 5.3 Emoticon Conversion
```javascript
const YM_EMOTICONS = {
    ':)': '😊',
    ':D': '😃',
    ':(': '😞',
    ';)': '😉',
    ':P': '😛',
    ':O': '😮',
    ':|': '😐',
    ':*': '😘',
    'X(': '😠',
    '(L)': '❤️',
    '(Y)': '👍',
    '(N)': '👎',
};
```

### 6. Files to Create/Modify

#### New Files:
1. `assets/static/css/ymxp.css` - Complete Windows XP + Yahoo Messenger styling
2. `assets/views/ymxp/layouts/default.html` - Base layout with XP/YM styling
3. `assets/views/ymxp/pages/home.html` - Landing page
4. `assets/views/ymxp/pages/login.html` - Sign in dialog
5. `assets/views/ymxp/pages/register.html` - Create account
6. `assets/views/ymxp/pages/app.html` - Main messenger UI
7. `assets/views/ymxp/pages/settings.html` - Preferences dialog

#### Modified Files:
1. `assets/embed.go` - Add "ymxp" to Themes slice
2. `assets/static/js/app.js` - Add "ymxp" to VIEW_THEMES array

### 7. Authentic Details

#### 7.1 Sound References (Visual Indicators)
- New Message: Ding sound → 📧 icon bounce
- Buzz: Buzzer sound → Red flash effect
- Online: Door open sound → Slide in animation
- Offline: Door close sound → Fade out animation

#### 7.2 Classic Yahoo Emoticons
Must include visual references to classic Yahoo emoticons:
- 😊 :) Standard smiley
- 😃 :D Big grin
- 😉 ;) Winking
- 😛 :P Tongue out
- 😎 B) Cool with sunglasses
- 😘 :* Kiss
- 😢 :'( Crying
- 😠 X( Angry

#### 7.3 Buddy List Categories
Default categories in Yahoo Messenger:
- Friends
- Family
- Co-Workers
- Recent Buddies

#### 7.4 Status Types
- Available (green circle)
- Busy (red circle with minus)
- Stepped Out (orange clock)
- Be Right Back
- Not at Home
- Not at My Desk
- On the Phone
- On Vacation
- Out to Lunch
- Invisible
- Custom...

### 8. Testing Checklist

- [ ] Windows XP Luna chrome renders correctly
- [ ] Yahoo purple header displays properly
- [ ] Buddy list categories expand/collapse
- [ ] Chat tabs work correctly
- [ ] Buzz animation triggers and resets
- [ ] Status icons display correctly for all states
- [ ] XP buttons have proper hover/active states
- [ ] XP scrollbars render in chat area
- [ ] Emoticons display in emoticon picker
- [ ] Theme persists across page reloads
- [ ] Theme selector in settings works
- [ ] Responsive behavior (where applicable)
- [ ] Cross-browser compatibility (Chrome, Firefox, Safari)

## Deliverables

1. This specification document
2. Complete ymxp.css with Windows XP Luna + Yahoo Messenger styling
3. All 5 view templates (layouts + pages)
4. Updated embed.go with ymxp theme
5. Updated app.js with ymxp theme support
6. Authentic Windows XP + Yahoo Messenger visual experience

## References

- Windows XP Luna Theme (Blue default)
- Yahoo! Messenger 6.0-7.5 (2004-2007)
- Windows XP UI Guidelines (MSDN archived)
- Yahoo! Messenger emoticon sets
