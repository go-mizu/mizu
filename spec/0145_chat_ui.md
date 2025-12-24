# Chat UI Redesign - Modern Theme System

**Spec ID**: 0145
**Status**: Draft
**Inspired By**: Discord, Slack

## Overview

Complete redesign of the chat blueprint UI with a proper theming system inspired by modern chat applications. This includes restructuring the template system to match the forum blueprint's theming architecture, server-side rendering focus, and minimal JavaScript.

## Goals

1. **Proper Theming System**: Theme inheritance with fallback to default
2. **Modern Chat UI**: Discord/Slack-inspired design with attention to detail
3. **Server-Side Rendering**: Minimize JavaScript by rendering on server
4. **Component Reusability**: Shared components across pages
5. **Clean Architecture**: Separation of layouts, pages, and components

---

## Directory Structure

### Current Structure (Before)
```
assets/
  views/
    layouts/
      base.html
    pages/
      landing.html
      login.html
      register.html
      app.html
      explore.html
      settings.html
  static/
    css/style.css
    js/app.js
```

### New Structure (After)
```
assets/
  embed.go                    # Theme-aware template loading
  views/
    default/                  # Default theme
      layouts/
        default.html          # Base layout with slots
        auth.html             # Auth-specific layout
        app.html              # Main app layout (three-column)
      pages/
        landing.html          # Public landing
        login.html            # Login form
        register.html         # Registration form
        explore.html          # Server discovery
        settings.html         # User settings
        chat.html             # Main chat view
        server_settings.html  # Server settings
        invite.html           # Invite page
      components/
        nav.html              # Top navigation
        server_list.html      # Left server icons
        channel_list.html     # Channel sidebar
        member_list.html      # Member sidebar
        message.html          # Single message
        message_group.html    # Grouped messages
        user_panel.html       # User panel at bottom
        modal.html            # Modal component
        toast.html            # Toast notifications
  static/
    css/
      app.css                 # Main styles (default theme)
    js/
      app.js                  # Minimal interactivity
```

---

## Theming System

### Theme Inheritance

```go
// Themes inherit from default - theme files override default files
func ViewsForTheme(theme string) fs.FS {
    if theme == "" || theme == "default" {
        return Views()
    }
    return &themeFS{
        theme:    theme,
        base:     FS,
        default_: "views/default",
        overlay:  "views/" + theme,
    }
}
```

### Template Loading

Each page gets an isolated template to avoid `{{define "content"}}` collisions:

```go
func TemplatesForTheme(theme string) (map[string]*template.Template, error) {
    // 1. Load layouts and components into base template
    // 2. For each page, clone base and add page template
    // 3. Return map[pageName]*template.Template
}
```

### Template Structure

**Layout Pattern:**
```html
{{define "default.html"}}
<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Chat</title>
    <link rel="stylesheet" href="/static/css/app.css">
</head>
<body>
    {{template "content" .}}
    <script src="/static/js/app.js" defer></script>
</body>
</html>
{{end}}
```

**Page Pattern:**
```html
{{define "chat.html"}}
{{template "default.html" .}}
{{end}}

{{define "content"}}
<div class="app">
    {{template "server_list.html" .}}
    {{template "channel_list.html" .}}
    <main class="chat-main">
        {{template "chat_header" .}}
        <div class="messages">
            {{range .Data.Messages}}
                {{template "message.html" .}}
            {{end}}
        </div>
        {{template "message_input" .}}
    </main>
    {{template "member_list.html" .}}
</div>
{{end}}
```

---

## Page Designs

### 1. Landing Page

Modern, clean landing with gradient accents:

- Hero section with animated background
- Feature cards highlighting key capabilities
- Clear CTAs: "Open App", "Explore Servers", "Login"
- Footer with links

**Design Elements:**
- Gradient text for headings
- Glassmorphism cards
- Subtle animations on scroll
- Dark theme by default

### 2. Authentication Pages (Login/Register)

Centered card design:

- Logo at top
- Form fields with floating labels
- Social login buttons (future)
- Link to alternate action (register/login)
- Error display

**Design Elements:**
- Card with subtle shadow
- Focus states on inputs
- Loading states on submit

### 3. Main Chat Interface

Discord-inspired three-column layout:

```
+---------------+------------------+------------------+---------------+
| Server List   | Channel Sidebar  |   Chat Area      | Member List   |
| (72px)        | (240px)          |   (flex)         | (240px)       |
+---------------+------------------+------------------+---------------+
| Home          | Server Name      | #channel topic   | Online        |
| ─────────     | ─────────────    | ──────────────   | ─────────     |
| Server1       | # general        | Messages...      | User1         |
| Server2       | # random         |                  | User2         |
| Server3       | # help           |                  |               |
| ─────────     | ─────────────    | ──────────────   | Offline       |
| + Add         | User Panel       | [Message Input]  | ─────────     |
+---------------+------------------+------------------+---------------+
```

**Server-Side Rendered:**
- Server list with active state
- Channel list with categories
- Messages (initial batch)
- Member list with presence

**Client-Side (Minimal):**
- WebSocket for real-time updates
- Message sending
- Typing indicators
- Presence updates

### 4. Settings Page

Tabbed interface:

- Account settings
- Profile settings
- Appearance (theme)
- Privacy
- Connections

**Design:**
- Left sidebar with tabs
- Content area with forms
- Save button at bottom

### 5. Explore Page

Server discovery grid:

- Search bar at top
- Filter buttons (Popular, New, Gaming, etc.)
- Server cards in grid
- Each card shows: Icon, Name, Description, Member count

---

## CSS Architecture

### Variables

```css
:root {
    /* Backgrounds */
    --bg-primary: #313338;
    --bg-secondary: #2b2d31;
    --bg-tertiary: #1e1f22;
    --bg-floating: #232428;

    /* Text */
    --text-primary: #f2f3f5;
    --text-secondary: #b5bac1;
    --text-muted: #949ba4;

    /* Accents */
    --accent-primary: #5865f2;
    --accent-success: #23a55a;
    --accent-danger: #f23f43;
    --accent-warning: #f0b232;

    /* Status */
    --status-online: #23a55a;
    --status-idle: #f0b232;
    --status-dnd: #f23f43;
    --status-offline: #80848e;

    /* Layout */
    --server-list-width: 72px;
    --channel-sidebar-width: 240px;
    --member-sidebar-width: 240px;

    /* Typography */
    --font-sans: 'Inter', system-ui, sans-serif;
    --font-mono: 'JetBrains Mono', monospace;
}

[data-theme="light"] {
    --bg-primary: #ffffff;
    --bg-secondary: #f2f3f5;
    --bg-tertiary: #e3e5e8;
    --text-primary: #060607;
    --text-secondary: #4e5058;
}
```

### Component Classes

```css
/* Message */
.message { }
.message-avatar { }
.message-content { }
.message-header { }
.message-body { }
.message-reactions { }

/* Channel */
.channel { }
.channel.active { }
.channel-icon { }
.channel-name { }
.channel-unread { }

/* Server Icon */
.server-icon { }
.server-icon.active { }
.server-icon:hover { }

/* Member */
.member { }
.member-avatar { }
.member-status { }
.member-name { }
```

---

## JavaScript (Minimal)

### Core Functionality

```javascript
// app.js - ~150 lines max

document.addEventListener('DOMContentLoaded', () => {
    initWebSocket();
    initMessageInput();
    initScrollBehavior();
});

// WebSocket for real-time
function initWebSocket() {
    const ws = new WebSocket(`ws://${location.host}/ws?token=${getToken()}`);

    ws.onmessage = (e) => {
        const msg = JSON.parse(e.data);
        handleEvent(msg.t, msg.d);
    };
}

function handleEvent(type, data) {
    switch (type) {
        case 'MESSAGE_CREATE':
            appendMessage(data);
            break;
        case 'MESSAGE_UPDATE':
            updateMessage(data);
            break;
        case 'MESSAGE_DELETE':
            removeMessage(data.id);
            break;
        case 'TYPING_START':
            showTyping(data);
            break;
        case 'PRESENCE_UPDATE':
            updatePresence(data);
            break;
    }
}

// Message sending
function initMessageInput() {
    const input = document.getElementById('message-input');
    input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            sendMessage();
        }
    });
}

async function sendMessage() {
    const input = document.getElementById('message-input');
    const content = input.value.trim();
    if (!content) return;

    input.value = '';

    await fetch(`/api/v1/channels/${CHANNEL_ID}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content })
    });
}
```

### No-JS Fallback

The app should work without JavaScript for:
- Navigation
- Reading messages
- Viewing members
- Settings changes (form submissions)

Only real-time features require JavaScript.

---

## Server-Side Rendering

### Page Handler

```go
type PageData struct {
    Title       string
    User        *accounts.User
    Data        map[string]any
    CurrentPath string
    Dev         bool
}

func (h *PageHandler) Chat(c *mizu.Ctx) error {
    serverID := c.Param("server_id")
    channelID := c.Param("channel_id")

    // Load all data server-side
    servers, _ := h.servers.ListForUser(ctx, userID)
    channels, _ := h.channels.ListForServer(ctx, serverID)
    messages, _ := h.messages.List(ctx, channelID, 50)
    members, _ := h.members.ListForServer(ctx, serverID)

    return h.render(c, "chat.html", PageData{
        Title: channel.Name,
        User:  user,
        Data: map[string]any{
            "servers":        servers,
            "channels":       channels,
            "messages":       messages,
            "members":        members,
            "currentServer":  server,
            "currentChannel": channel,
        },
    })
}
```

### Template Functions

```go
func templateFuncs() template.FuncMap {
    return template.FuncMap{
        "formatTime":         formatTime,
        "formatTimeRelative": formatTimeRelative,
        "formatNumber":       formatNumber,
        "truncate":           truncate,
        "safeHTML":           safeHTML,
        "statusClass":        statusClass,
        "userInitials":       userInitials,
        "dict":               dict,
        "list":               list,
    }
}
```

---

## Implementation Steps

### Phase 1: Structure
1. Create views/default/ directory
2. Move existing views to default/
3. Update embed.go with theming system

### Phase 2: Layout & Components
4. Create default.html layout
5. Create auth.html layout
6. Create app.html layout (3-column)
7. Create components (server_list, channel_list, etc.)

### Phase 3: Pages
8. Redesign landing.html
9. Redesign login.html & register.html
10. Redesign chat.html with SSR
11. Redesign settings.html
12. Redesign explore.html

### Phase 4: Polish
13. Refine CSS with proper variables
14. Minimize app.js
15. Add loading states
16. Test all flows

---

## Success Criteria

- Theme system works with inheritance
- All pages render correctly server-side
- JavaScript is minimal (~150 lines)
- No-JS fallback works for basic features
- UI matches Discord/Slack aesthetic
- Clean component separation
