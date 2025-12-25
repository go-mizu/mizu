# Spec 0155: Enhance Messaging Themes for Authentic OS Recreation

**Status:** Implementing

## Overview

Enhance the existing messaging themes to achieve 100% authentic recreation of their respective operating system aesthetics:

1. **AIM 1.0 Theme**: Transform from Windows 3.1 style to **authentic Windows 98** style
2. **Yahoo Messenger (ymxp) Theme**: Refine to be **100% authentic Windows XP Luna Blue** style

## Research Sources

- [98.css](https://jdan.github.io/98.css/) - Windows 98 CSS library
- [XP.css](https://botoxparty.github.io/XP.css/) - Windows XP CSS library
- [Windows 98 Icon Viewer](https://win98icons.alexmeub.com/) - Windows 98 system icons
- [Windows XP Visual Styles - Wikipedia](https://en.wikipedia.org/wiki/Windows_XP_visual_styles)
- [Windows XP Color Scheme](https://www.schemecolor.com/windows-xp-color-scheme.php)

---

## Part 1: AIM 1.0 Theme - Windows 98 Authentic Style

### Current Issues

The current AIM 1.0 theme uses Windows 3.1 styling, but Windows 98 has distinct differences:
- Windows 98 has gradient title bars (blue gradient, not solid navy)
- Different 3D border styling
- MS Shell Dlg / Tahoma font instead of System font
- Different icon styling (256 color instead of 16 color)
- Slightly refined scrollbar appearance

### Windows 98 Authentic Color Palette

```css
:root {
    /* Windows 98 System Colors */
    --win98-bg: #c0c0c0;                    /* Button face / window background */
    --win98-text: #000000;                  /* Window text */
    --win98-desktop: #008080;               /* Desktop teal */

    /* 3D Border Colors (authentic 98.css) */
    --win98-button-highlight: #ffffff;       /* Top/left outer highlight */
    --win98-button-light: #dfdfdf;          /* Top/left inner highlight */
    --win98-button-shadow: #808080;         /* Bottom/right inner shadow */
    --win98-button-dark-shadow: #000000;    /* Bottom/right outer shadow */

    /* Title Bar - Windows 98 uses GRADIENT (key difference from 3.1) */
    --win98-titlebar-active-start: #000080; /* Navy blue start */
    --win98-titlebar-active-end: #1084d0;   /* Light blue end */
    --win98-titlebar-inactive: #808080;     /* Gray inactive */
    --win98-titlebar-text: #ffffff;         /* White text */

    /* Selection */
    --win98-highlight: #000080;             /* Navy blue selection */
    --win98-highlight-text: #ffffff;        /* White text on selection */

    /* AIM Specific Colors */
    --aim-yellow: #ffff00;                  /* AIM running man yellow */
    --aim-gold: #ffcc00;                    /* AIM logo gold */
    --aim-sent-msg: #ff0000;                /* Red for sent messages */
    --aim-received-msg: #0000ff;            /* Blue for received messages */
}
```

### Windows 98 Title Bar (Key Visual Difference)

Windows 98 title bars use a **horizontal gradient** (left-to-right), unlike Windows 3.1's solid color:

```css
.win-titlebar {
    background: linear-gradient(90deg, #000080 0%, #1084d0 100%);
    color: #ffffff;
    font-weight: bold;
    font-family: 'Tahoma', 'MS Sans Serif', sans-serif;
    font-size: 11px;
    padding: 3px 4px;
    height: 22px;
    display: flex;
    align-items: center;
}

.win-titlebar.inactive {
    background: linear-gradient(90deg, #808080 0%, #b5b5b5 100%);
}
```

### Windows 98 Authentic Button Style

```css
.win-button {
    background: #c0c0c0;
    border: none;
    padding: 4px 12px;
    min-width: 75px;
    min-height: 23px;
    font-family: 'Tahoma', sans-serif;
    font-size: 11px;
    cursor: pointer;
    box-shadow:
        inset -1px -1px 0 #0a0a0a,
        inset 1px 1px 0 #ffffff,
        inset -2px -2px 0 #808080,
        inset 2px 2px 0 #dfdfdf;
}

.win-button:active {
    box-shadow:
        inset -1px -1px 0 #ffffff,
        inset 1px 1px 0 #0a0a0a,
        inset -2px -2px 0 #dfdfdf,
        inset 2px 2px 0 #808080;
    padding: 5px 11px 3px 13px;
}

.win-button:focus::after {
    content: "";
    position: absolute;
    top: 4px;
    left: 4px;
    right: 4px;
    bottom: 4px;
    border: 1px dotted #000;
}
```

### Windows 98 Title Bar Buttons

Windows 98 title bar buttons are slightly different from Windows 3.1:
- 16x14 pixels
- Distinct minimize (underscore), maximize (square), close (X) symbols
- Same 3D border styling as other buttons

```css
.win-titlebar-btn {
    width: 16px;
    height: 14px;
    background: #c0c0c0;
    border: none;
    box-shadow:
        inset -1px -1px 0 #0a0a0a,
        inset 1px 1px 0 #ffffff,
        inset -2px -2px 0 #808080,
        inset 2px 2px 0 #dfdfdf;
    font-size: 9px;
    font-weight: bold;
    font-family: 'Marlett', 'Webdings', sans-serif;
    display: flex;
    align-items: center;
    justify-content: center;
}
```

### Windows 98 Authentic Icons (SVG Recreations)

#### Minimize Button Icon
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="14" viewBox="0 0 16 14">
    <rect x="4" y="10" width="6" height="2" fill="#000"/>
</svg>
```

#### Maximize Button Icon
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="14" viewBox="0 0 16 14">
    <rect x="3" y="3" width="9" height="7" fill="none" stroke="#000" stroke-width="1"/>
    <rect x="3" y="3" width="9" height="2" fill="#000"/>
</svg>
```

#### Close Button Icon
```svg
<svg xmlns="http://www.w3.org/2000/svg" width="16" height="14" viewBox="0 0 16 14">
    <path d="M4 3l8 8M12 3l-8 8" stroke="#000" stroke-width="2"/>
</svg>
```

#### AIM Running Man Icon (Online Status)
```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16">
    <!-- Head -->
    <circle cx="8" cy="3" r="2.5" fill="#FFCC00"/>
    <!-- Body in running pose -->
    <path d="M8 6 L8 9 L5 12 M8 9 L11 12 M6 7 L4 9 M10 7 L12 5"
          stroke="#FFCC00" stroke-width="2" stroke-linecap="round" fill="none"/>
</svg>
```

#### AIM Door Icon (Away Status)
```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 16 16">
    <!-- Door frame -->
    <rect x="4" y="2" width="8" height="12" fill="#8B4513"/>
    <!-- Door panel -->
    <rect x="5" y="3" width="6" height="10" fill="#D2691E"/>
    <!-- Door handle -->
    <circle cx="10" cy="8" r="1" fill="#FFD700"/>
</svg>
```

### Windows 98 Scrollbar (Dithered Pattern)

```css
::-webkit-scrollbar {
    width: 16px;
    height: 16px;
}

::-webkit-scrollbar-track {
    background-color: #c0c0c0;
    background-image: url("data:image/svg+xml,%3Csvg width='2' height='2' viewBox='0 0 2 2' xmlns='http://www.w3.org/2000/svg'%3E%3Crect x='0' y='0' width='1' height='1' fill='%23c0c0c0'/%3E%3Crect x='1' y='0' width='1' height='1' fill='%23ffffff'/%3E%3Crect x='0' y='1' width='1' height='1' fill='%23ffffff'/%3E%3Crect x='1' y='1' width='1' height='1' fill='%23c0c0c0'/%3E%3C/svg%3E");
    background-size: 2px 2px;
}

::-webkit-scrollbar-thumb {
    background: #c0c0c0;
    box-shadow:
        inset -1px -1px 0 #0a0a0a,
        inset 1px 1px 0 #ffffff,
        inset -2px -2px 0 #808080,
        inset 2px 2px 0 #dfdfdf;
}

::-webkit-scrollbar-button {
    background: #c0c0c0;
    box-shadow:
        inset -1px -1px 0 #0a0a0a,
        inset 1px 1px 0 #ffffff,
        inset -2px -2px 0 #808080,
        inset 2px 2px 0 #dfdfdf;
    height: 16px;
    width: 16px;
}
```

### Windows 98 Typography

```css
* {
    font-family: 'Tahoma', 'MS Sans Serif', 'Segoe UI', sans-serif !important;
    font-size: 11px;
    /* Disable anti-aliasing for authentic pixelated look */
    -webkit-font-smoothing: none;
    -moz-osx-font-smoothing: grayscale;
}
```

---

## Part 2: Yahoo Messenger Theme - Windows XP Luna Blue 100% Authentic

### Current Status

The ymxp theme already has a good Windows XP foundation, but needs refinement for 100% authenticity:

1. **Title bar gradient** needs exact XP values
2. **Window corners** need proper 8px radius (top only)
3. **Close/Minimize/Maximize buttons** need exact XP styling
4. **Start button style elements** where applicable
5. **Icons** need XP-style glossy appearance

### Windows XP Luna Blue Authentic Color Palette

```css
:root {
    /* Windows XP Desktop */
    --xp-desktop: #004E98;                  /* Famous XP blue desktop */

    /* Window Background */
    --xp-window-bg: #ECE9D8;                /* XP cream/tan background */
    --xp-window-bg-light: #F5F3E8;

    /* Title Bar Gradient - EXACT XP VALUES */
    --xp-titlebar-gradient: linear-gradient(180deg,
        #0a58ca 0%,                         /* Top edge */
        #3c8ced 8%,                         /* Upper gradient */
        #4698ed 15%,                        /* Peak brightness */
        #2878d8 40%,                        /* Middle */
        #0b5cc6 70%,                        /* Lower gradient */
        #0854b8 100%);                      /* Bottom edge */

    /* Inactive Title Bar */
    --xp-titlebar-inactive: linear-gradient(180deg,
        #7a8bb6 0%,
        #9ba8c6 8%,
        #a0adcc 15%,
        #8c99b8 40%,
        #7584a8 70%,
        #6b7a9c 100%);

    /* Close Button - RED gradient */
    --xp-close-gradient: linear-gradient(180deg,
        #c54f47 0%,                         /* Dark red top */
        #e89b8b 45%,                        /* Pink middle highlight */
        #c54f47 100%);                      /* Dark red bottom */

    --xp-close-hover: linear-gradient(180deg,
        #d55f57 0%,
        #f8ab9b 45%,
        #d55f57 100%);

    /* Min/Max Buttons - BLUE gradient */
    --xp-minmax-gradient: linear-gradient(180deg,
        #3c81c3 0%,                         /* Dark blue top */
        #73b2eb 45%,                        /* Light blue middle highlight */
        #3c81c3 100%);                      /* Dark blue bottom */

    --xp-minmax-hover: linear-gradient(180deg,
        #4c91d3 0%,
        #83c2fb 45%,
        #4c91d3 100%);

    /* Button Styling */
    --xp-button-gradient: linear-gradient(180deg,
        #ffffff 0%,
        #ece9d8 89%,
        #d4d0c8 100%);

    --xp-button-border: #003c74;
    --xp-button-shadow: #aca899;

    /* Primary Button (Blue) */
    --xp-button-primary: linear-gradient(180deg,
        #a4d3ff 0%,
        #6fb5f8 40%,
        #4a9aea 60%,
        #6fb5f8 100%);

    /* Selection & Focus */
    --xp-selection: #316AC5;
    --xp-selection-text: #ffffff;

    /* Input */
    --xp-input-border: #7f9db9;
    --xp-input-focus: #316AC5;

    /* Scrollbar */
    --xp-scrollbar-track: #f1efe2;
    --xp-scrollbar-arrow: #6e899a;

    /* Text */
    --xp-text: #000000;
    --xp-text-link: #0066CC;
    --xp-text-disabled: #aca899;

    /* Groupbox */
    --xp-groupbox-border: #d4d0c8;
}
```

### Windows XP Window Frame

XP windows have distinct characteristics:
- **8px border radius** on top corners only
- **Blue outer glow/border**
- **Drop shadow**

```css
.xp-window {
    background: var(--xp-window-bg);
    border-radius: 8px 8px 0 0;
    border: 1px solid #0054e3;
    box-shadow:
        0 0 0 1px #0054e3,
        inset 0 0 0 1px rgba(255, 255, 255, 0.4),
        3px 3px 12px rgba(0, 0, 0, 0.35);
    overflow: hidden;
}
```

### Windows XP Title Bar

```css
.xp-titlebar {
    background: linear-gradient(180deg,
        #0a58ca 0%,
        #3c8ced 8%,
        #4698ed 15%,
        #2878d8 40%,
        #0b5cc6 70%,
        #0854b8 100%);
    color: #ffffff;
    font-weight: bold;
    font-family: 'Trebuchet MS', 'Tahoma', sans-serif;
    font-size: 11px;
    padding: 3px 5px;
    min-height: 28px;
    border-radius: 8px 8px 0 0;
    text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.5);
    box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.3);
}
```

### Windows XP Title Bar Buttons (Authentic Recreation)

```css
.xp-btn-minimize,
.xp-btn-maximize,
.xp-btn-close {
    width: 21px;
    height: 21px;
    border: none;
    border-radius: 3px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: white;
    position: relative;
    box-shadow:
        inset -1px -1px 0 rgba(0, 0, 0, 0.3),
        inset 1px 1px 0 rgba(255, 255, 255, 0.3);
}

/* Blue buttons for min/max */
.xp-btn-minimize,
.xp-btn-maximize {
    background: linear-gradient(180deg,
        #3c81c3 0%,
        #73b2eb 45%,
        #3c81c3 100%);
    border: 1px solid #1d4f91;
}

/* Red button for close */
.xp-btn-close {
    background: linear-gradient(180deg,
        #c54f47 0%,
        #e89b8b 45%,
        #c54f47 100%);
    border: 1px solid #8c3a34;
}
```

### Windows XP Button Icons (Pure CSS)

```css
/* Minimize: Horizontal line at bottom */
.xp-btn-minimize::after {
    content: "";
    width: 8px;
    height: 2px;
    background: white;
    position: absolute;
    bottom: 5px;
    box-shadow: 0 1px 0 rgba(0,0,0,0.3);
}

/* Maximize: Square outline with thick top */
.xp-btn-maximize::after {
    content: "";
    width: 9px;
    height: 7px;
    border: 2px solid white;
    border-top-width: 3px;
    box-shadow: 1px 1px 0 rgba(0,0,0,0.3);
}

/* Close: X mark using two rotated bars */
.xp-btn-close::before,
.xp-btn-close::after {
    content: "";
    position: absolute;
    width: 10px;
    height: 2px;
    background: white;
    box-shadow: 0 1px 0 rgba(0,0,0,0.3);
}

.xp-btn-close::before {
    transform: rotate(45deg);
}

.xp-btn-close::after {
    transform: rotate(-45deg);
}
```

### Windows XP Scrollbar (Checkered Track)

```css
::-webkit-scrollbar {
    width: 17px;
    height: 17px;
}

::-webkit-scrollbar-track {
    background-color: #f1efe2;
    background-image: url("data:image/svg+xml,%3Csvg width='2' height='2' xmlns='http://www.w3.org/2000/svg'%3E%3Crect x='0' y='0' width='1' height='1' fill='%23f1efe2'/%3E%3Crect x='1' y='0' width='1' height='1' fill='%23aba899'/%3E%3Crect x='0' y='1' width='1' height='1' fill='%23aba899'/%3E%3Crect x='1' y='1' width='1' height='1' fill='%23f1efe2'/%3E%3C/svg%3E");
    background-size: 2px 2px;
}

::-webkit-scrollbar-thumb {
    background: linear-gradient(90deg,
        #ece9d8 0%,
        #ece9d8 30%,
        #d4d0c8 100%);
    border: 1px solid;
    border-color: #f1efe2 #848280 #848280 #f1efe2;
    box-shadow: inset 1px 1px 0 #fafafa;
}

::-webkit-scrollbar-button {
    background: linear-gradient(180deg,
        #ece9d8 0%,
        #ece9d8 80%,
        #d4d0c8 100%);
    border: 1px solid;
    border-color: #f1efe2 #848280 #848280 #f1efe2;
    height: 17px;
    width: 17px;
}
```

### Windows XP Icons (Glossy Style)

XP icons have distinctive characteristics:
- **Glossy/shiny appearance** with gradients
- **Soft shadows**
- **24x24 or 32x32 common sizes**
- **Rounded, friendly shapes**

#### Yahoo Messenger Smiley (XP Style)
```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 32 32">
    <defs>
        <radialGradient id="xpSmiley" cx="30%" cy="30%">
            <stop offset="0%" stop-color="#FFEE00"/>
            <stop offset="50%" stop-color="#FFCC00"/>
            <stop offset="100%" stop-color="#E5A800"/>
        </radialGradient>
        <filter id="xpShadow">
            <feDropShadow dx="1" dy="2" stdDeviation="1" flood-opacity="0.3"/>
        </filter>
    </defs>
    <!-- Face with XP glossy effect -->
    <circle cx="16" cy="16" r="14" fill="url(#xpSmiley)" filter="url(#xpShadow)"/>
    <!-- Highlight for glossy effect -->
    <ellipse cx="12" cy="10" rx="6" ry="4" fill="rgba(255,255,255,0.4)"/>
    <!-- Eyes -->
    <ellipse cx="11" cy="14" rx="2" ry="3" fill="#5C1A4A"/>
    <ellipse cx="21" cy="14" rx="2" ry="3" fill="#5C1A4A"/>
    <!-- Smile -->
    <path d="M9 20 Q16 26 23 20" stroke="#5C1A4A" stroke-width="2.5" fill="none" stroke-linecap="round"/>
</svg>
```

### XP Status Icons (Glossy Orbs)

```css
/* Online - Green glossy orb */
.ym-icon-online {
    background: radial-gradient(circle at 30% 30%,
        #90EE90 0%,      /* Light green highlight */
        #32CD32 40%,     /* Lime green */
        #228B22 80%,     /* Forest green */
        #006400 100%);   /* Dark green edge */
    border-radius: 50%;
    box-shadow:
        inset 0 2px 4px rgba(255,255,255,0.5),
        0 2px 4px rgba(0,0,0,0.3);
}

/* Away - Orange/Yellow glossy orb */
.ym-icon-away {
    background: radial-gradient(circle at 30% 30%,
        #FFE066 0%,
        #FFA500 40%,
        #FF8C00 80%,
        #CC7000 100%);
    border-radius: 50%;
    box-shadow:
        inset 0 2px 4px rgba(255,255,255,0.5),
        0 2px 4px rgba(0,0,0,0.3);
}

/* Busy - Red glossy orb */
.ym-icon-busy {
    background: radial-gradient(circle at 30% 30%,
        #FF6B6B 0%,
        #DC143C 40%,
        #B22222 80%,
        #8B0000 100%);
    border-radius: 50%;
    box-shadow:
        inset 0 2px 4px rgba(255,255,255,0.5),
        0 2px 4px rgba(0,0,0,0.3);
}

/* Offline - Gray glossy orb */
.ym-icon-offline {
    background: radial-gradient(circle at 30% 30%,
        #E0E0E0 0%,
        #B0B0B0 40%,
        #909090 80%,
        #606060 100%);
    border-radius: 50%;
    box-shadow:
        inset 0 2px 4px rgba(255,255,255,0.3),
        0 2px 4px rgba(0,0,0,0.2);
}
```

---

## Implementation Checklist

### AIM 1.0 (Windows 98) Changes

- [ ] Update title bar to use **horizontal gradient** (navy to light blue)
- [ ] Update 3D border shadows to exact 98.css values
- [ ] Add Windows 98 style minimize/maximize/close button icons (SVG)
- [ ] Update scrollbar styling with proper dithered pattern
- [ ] Ensure Tahoma font is primary
- [ ] Update AIM running man icon with proper pixel art style
- [ ] Add proper Windows 98 checkmark/radio button styling
- [ ] Update tabs to Windows 98 style

### Yahoo Messenger (Windows XP) Changes

- [ ] Verify title bar gradient matches exact XP Luna Blue values
- [ ] Ensure 8px top border radius on windows
- [ ] Update close button to red gradient with proper icon
- [ ] Update min/max buttons to blue gradient with proper icons
- [ ] Add glossy effect to status icons (online/away/busy)
- [ ] Ensure scrollbar matches XP checkered pattern
- [ ] Add XP-style drop shadows to windows
- [ ] Update button styling to exact XP gradient
- [ ] Add inner glow/shine effects to interactive elements

---

## Files to Modify

1. `/assets/static/css/aim.css` - Full Windows 98 overhaul
2. `/assets/static/css/ymxp.css` - Windows XP refinements
3. `/assets/views/aim1.0/pages/app.html` - Update icon SVGs
4. `/assets/views/ymxp/pages/app.html` - Update icon SVGs

## Testing

After implementation, verify:
1. Side-by-side comparison with real Windows 98/XP screenshots
2. All interactive states (hover, active, focus) work correctly
3. Icons are crisp and recognizable at all sizes
4. Fonts render correctly with/without anti-aliasing
5. Scrollbars function properly in all browsers

## References

- Microsoft Windows User Experience Guidelines (archived)
- 98.css source: https://github.com/jdan/98.css
- XP.css source: https://github.com/botoxparty/XP.css
- Windows 98 SE icon theme: https://github.com/nestoris/Win98SE
