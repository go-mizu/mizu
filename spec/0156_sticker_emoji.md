# Spec 0156: Sticker and Emoji Feature

## Overview

This specification defines the implementation of emoji picker, sticker picker, and message reactions for the Mizu messaging blueprint. The feature supports all three themes: default (modern), aim1.0 (Windows 98 AOL), and ymxp (Yahoo Messenger XP).

## Goals

1. **Emoji Picker**: Allow users to select and send emoji in messages
2. **Sticker Picker**: Allow users to browse and send stickers from packs
3. **Message Reactions**: Allow users to react to messages with emoji
4. **Theme Consistency**: Ensure all features work seamlessly across all themes

## Feature Details

### 1. Emoji Picker

#### User Experience
- Trigger: Click emoji button in message input area
- Display: Popup/modal with categorized emoji grid
- Categories: Smileys, People, Animals, Food, Travel, Activities, Objects, Symbols, Flags
- Recent: Track recently used emoji per user (localStorage)
- Search: Filter emoji by name/keyword
- Skin tone: Support skin tone variants for applicable emoji

#### UI Specifications

**Default Theme:**
- Floating panel above message input
- Dark/light mode aware (follows theme)
- 8 emoji per row, scrollable grid
- Category tabs at top with icons
- Search bar at top
- Recent section pinned at top

**AIM 1.0 Theme:**
- Windows 98 style popup window with title bar
- Classic scrollbar styling
- 6 emoji per row (larger icons for retro feel)
- Category dropdown instead of tabs
- Yellow/buddy list style accents

**Yahoo Messenger XP Theme:**
- Windows XP style popup with gradient title bar
- Luna-style scrollbars
- 7 emoji per row
- Category buttons with XP styling

#### Technical Implementation
- Emoji data: Unicode emoji with shortcodes
- Storage: Recent emoji in localStorage (last 24)
- Insertion: Insert at cursor position in textarea
- Keyboard: Arrow keys to navigate, Enter to select

### 2. Sticker Picker

#### Sticker Pack Structure
```json
{
  "id": "pack_classic",
  "name": "Classic Stickers",
  "thumbnail": "/static/stickers/classic/thumb.png",
  "stickers": [
    {
      "id": "sticker_001",
      "name": "thumbs_up",
      "url": "/static/stickers/classic/thumbs_up.png",
      "tags": ["like", "approve", "yes"]
    }
  ]
}
```

#### Built-in Sticker Packs
1. **Classic** - Basic expression stickers (thumbs up, heart, laugh, cry, etc.)
2. **Animals** - Cute animal stickers
3. **Retro** - Pixel art / retro computing stickers (fits AIM/YMXP themes)
4. **Reactions** - Quick reaction stickers (OK, LOL, WOW, etc.)

#### User Experience
- Trigger: Click sticker button next to emoji button
- Display: Full panel with pack tabs at bottom
- Browse: Scroll through stickers in selected pack
- Recent: Track recently sent stickers
- Preview: Hover to see larger preview

#### UI Specifications

**Default Theme:**
- Larger panel than emoji picker (stickers need space)
- Pack icons as tabs at bottom
- 4 stickers per row
- Smooth hover animations
- Preview tooltip on hover

**AIM 1.0 Theme:**
- Classic window with "Stickers" title
- Pack selector as dropdown
- 3 stickers per row
- No animations (authentic to era)
- Status bar showing sticker name

**Yahoo Messenger XP Theme:**
- XP-style tabbed interface
- Gradient backgrounds
- 4 stickers per row
- Subtle hover effects

#### Technical Implementation
- Stickers stored as static assets
- Send as message type "sticker"
- Sticker messages contain: pack_id, sticker_id, sticker_url
- Rendered as images in chat (no bubble)

### 3. Message Reactions

#### User Experience
- Trigger: Hover message → Show reaction button OR double-tap on mobile
- Quick reactions: Row of 6 common emoji (customizable)
- Full picker: Click "+" to open full emoji picker
- Display: Reactions shown below message bubble
- Interaction: Click own reaction to remove, click others to add same

#### Quick Reaction Emoji (Default)
1. 👍 (thumbs up)
2. ❤️ (heart)
3. 😂 (laugh)
4. 😮 (wow)
5. 😢 (sad)
6. 🙏 (thank you)

#### UI Specifications

**Reaction Display (All Themes):**
- Compact pill badges below message
- Show emoji + count
- Highlight if current user reacted
- Click to toggle reaction
- Long press to see who reacted

**Default Theme:**
- Reactions in rounded pills with subtle shadow
- Hover reveals reaction tooltip with users
- Quick picker appears on message hover
- Smooth animations

**AIM 1.0 Theme:**
- Reactions in 3D bordered boxes
- Static display (no animations)
- Quick picker in small popup window
- Yellow highlight for own reactions

**Yahoo Messenger XP Theme:**
- Reactions in XP-style buttons
- Gradient hover effects
- Quick picker with XP styling

#### Technical Implementation
- API: POST /api/v1/chats/{id}/messages/{msg_id}/react
- One reaction per user (changes if different emoji sent)
- WebSocket broadcast for real-time updates
- Aggregate display: {emoji: "👍", count: 3, users: [...], me: true}

### 4. Sticker Messages

#### Rendering
- Stickers displayed without message bubble
- Larger than regular images (120x120 default, 100x100 AIM, 110x110 YMXP)
- Sender name shown above (in groups)
- Timestamp shown below
- Support for animated stickers (GIF format)

#### Message Structure
```json
{
  "type": "sticker",
  "content": "",
  "sticker": {
    "pack_id": "classic",
    "sticker_id": "thumbs_up",
    "url": "/static/stickers/classic/thumbs_up.png"
  }
}
```

## Database Schema

### Sticker Packs Table
```sql
CREATE TABLE IF NOT EXISTS sticker_packs (
  id VARCHAR PRIMARY KEY,
  name VARCHAR NOT NULL,
  description VARCHAR,
  thumbnail_url VARCHAR NOT NULL,
  author VARCHAR,
  is_animated BOOLEAN DEFAULT FALSE,
  is_official BOOLEAN DEFAULT TRUE,
  sort_order INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### Stickers Table
```sql
CREATE TABLE IF NOT EXISTS stickers (
  id VARCHAR PRIMARY KEY,
  pack_id VARCHAR NOT NULL REFERENCES sticker_packs(id),
  name VARCHAR NOT NULL,
  url VARCHAR NOT NULL,
  thumbnail_url VARCHAR,
  tags VARCHAR[], -- Array of searchable tags
  sort_order INTEGER DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### User Sticker Preferences
```sql
CREATE TABLE IF NOT EXISTS user_sticker_recent (
  user_id VARCHAR NOT NULL,
  sticker_id VARCHAR NOT NULL,
  used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, sticker_id)
);

CREATE TABLE IF NOT EXISTS user_emoji_recent (
  user_id VARCHAR NOT NULL,
  emoji VARCHAR NOT NULL,
  used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, emoji)
);
```

## API Endpoints

### Sticker Endpoints
```
GET    /api/v1/stickers/packs              # List all sticker packs
GET    /api/v1/stickers/packs/{id}         # Get pack with stickers
GET    /api/v1/stickers/recent             # Get user's recent stickers
POST   /api/v1/stickers/recent/{id}        # Add to recent (auto on send)
GET    /api/v1/stickers/search?q=keyword   # Search stickers by tag
```

### Updated Message Endpoint
```
POST   /api/v1/chats/{id}/messages
Body: {
  "type": "sticker",
  "sticker_pack_id": "classic",
  "sticker_id": "thumbs_up"
}
```

## File Structure

```
assets/
├── static/
│   ├── stickers/
│   │   ├── classic/
│   │   │   ├── thumb.png
│   │   │   ├── thumbs_up.png
│   │   │   ├── heart.png
│   │   │   └── ...
│   │   ├── animals/
│   │   ├── retro/
│   │   └── reactions/
│   ├── css/
│   │   ├── default.css      # Updated with emoji/sticker styles
│   │   ├── aim.css          # Updated with emoji/sticker styles
│   │   └── ymxp.css         # Updated with emoji/sticker styles
│   └── js/
│       ├── app.js           # Updated with emoji/sticker logic
│       └── emoji-data.js    # Emoji database with categories
└── views/
    ├── default/pages/app.html
    ├── aim1.0/pages/app.html
    └── ymxp/pages/app.html
```

## Feature Service

### stickers/api.go
```go
type Pack struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    Thumbnail   string    `json:"thumbnail"`
    IsAnimated  bool      `json:"is_animated"`
    Stickers    []Sticker `json:"stickers,omitempty"`
}

type Sticker struct {
    ID        string   `json:"id"`
    PackID    string   `json:"pack_id"`
    Name      string   `json:"name"`
    URL       string   `json:"url"`
    Thumbnail string   `json:"thumbnail,omitempty"`
    Tags      []string `json:"tags,omitempty"`
}

type Service interface {
    ListPacks(ctx context.Context) ([]Pack, error)
    GetPack(ctx context.Context, packID string) (*Pack, error)
    GetSticker(ctx context.Context, stickerID string) (*Sticker, error)
    SearchStickers(ctx context.Context, query string) ([]Sticker, error)
    GetRecentStickers(ctx context.Context, userID string, limit int) ([]Sticker, error)
    RecordStickerUse(ctx context.Context, userID, stickerID string) error
}
```

## Implementation Order

1. **Phase 1: Emoji Picker (UI)**
   - Create emoji-data.js with categorized emoji
   - Add emoji picker HTML/CSS for all themes
   - Add emoji picker JavaScript interactions
   - Integrate with message input

2. **Phase 2: Message Reactions (UI + Integration)**
   - Add reaction display below messages
   - Add reaction picker on message hover
   - Connect to existing reaction API
   - Handle WebSocket reaction updates

3. **Phase 3: Stickers (Backend + UI)**
   - Add sticker tables to schema
   - Create sticker feature service
   - Add sticker API endpoints
   - Create sticker assets
   - Add sticker picker UI for all themes
   - Add sticker message rendering

4. **Phase 4: Polish**
   - Recent emoji/stickers tracking
   - Search functionality
   - Keyboard navigation
   - Mobile touch support
   - Performance optimization

## Accessibility

- Keyboard navigation for emoji/sticker pickers
- ARIA labels for all interactive elements
- Screen reader announcements for reactions
- Focus management when opening/closing pickers
- High contrast support in default theme

## Performance Considerations

- Lazy load emoji data (split by category)
- Virtualized scrolling for large sticker packs
- Image lazy loading for sticker thumbnails
- Debounced search for emoji/sticker filtering
- WebSocket batching for rapid reaction updates

## Testing

- Unit tests for sticker service
- Integration tests for sticker API endpoints
- E2E tests for emoji picker interactions
- E2E tests for sticker sending flow
- E2E tests for reaction add/remove
- Cross-theme visual regression tests
