# Spec 0158: Media and Files Feature

## Overview

This specification defines the implementation of media and file sharing functionality for the Mizu messaging blueprint, similar to popular chat messaging apps like WhatsApp, Telegram, and iMessage. The feature supports all three themes: default (modern), aim1.0 (Windows 98 AOL), and ymxp (Yahoo Messenger XP).

## Goals

1. **File Upload**: Allow users to attach and send files in messages
2. **Image Sharing**: Send images with inline preview and lightbox view
3. **Video Sharing**: Send videos with thumbnail preview and playback
4. **Audio/Voice**: Send audio files and voice messages with waveform visualization
5. **Document Sharing**: Share documents (PDF, Office files, etc.) with file info
6. **Media Gallery**: View all media shared in a chat
7. **Drag & Drop**: Support drag-and-drop file uploads
8. **Copy & Paste**: Support clipboard paste for images
9. **Theme Consistency**: Ensure all features work seamlessly across all themes

## Feature Details

### 1. File Types Supported

#### Images
- JPEG, PNG, GIF, WebP
- Max size: 20MB
- Thumbnails generated for preview
- Click to open lightbox viewer
- Support animated GIFs

#### Videos
- MP4, WebM, MOV
- Max size: 100MB
- Thumbnail extracted from first frame
- Inline video player
- Duration display

#### Audio
- MP3, M4A, OGG, WAV
- Max size: 50MB
- Duration and waveform display
- Inline audio player

#### Voice Messages
- WebM (recorded in browser)
- Max duration: 5 minutes
- Waveform visualization
- Play/pause controls

#### Documents
- PDF, DOC, DOCX, XLS, XLSX, PPT, PPTX, TXT, ZIP
- Max size: 100MB
- File icon based on type
- Size and filename display
- Click to download

### 2. Upload Experience

#### Default Theme
- Click paperclip button to open file picker
- Drag & drop zone appears on drag over chat
- Preview modal before sending with caption option
- Upload progress bar with percentage
- Cancel button during upload
- Paste images from clipboard (Ctrl/Cmd+V)

#### AIM 1.0 Theme
- Windows 98 "Send File" dialog with File > Send File menu option
- Classic file browser dialog styling
- Progress window with Windows 98 progress bar
- Transfer status in status bar
- "Direct Connection" style file transfer UI

#### YMXP Theme
- Windows XP file dialog with Luna styling
- Drag and drop with XP visual feedback
- XP-style progress dialog
- File transfer notification in system tray area

### 3. Media Preview/Display

#### Image Messages

**Default Theme:**
```
┌─────────────────────────────────┐
│  ┌─────────────────────────┐    │
│  │                         │    │
│  │        IMAGE            │    │
│  │       PREVIEW           │    │
│  │                         │    │
│  └─────────────────────────┘    │
│  Optional caption text here     │
│                        12:34 ✓✓ │
└─────────────────────────────────┘
```
- Rounded corners, max-width 300px
- Click opens lightbox with full-size image
- Lazy loading for performance

**AIM 1.0 Theme:**
- Image in classic bordered frame
- Status bar shows dimensions
- Double-click opens in new window

**YMXP Theme:**
- XP photo frame styling
- Hover shows "View Photo" tooltip
- Click opens XP-style image viewer

#### Video Messages

**Default Theme:**
- Thumbnail with play button overlay
- Duration badge in corner
- Click to play inline
- Fullscreen option

**AIM 1.0 Theme:**
- Windows Media Player style frame
- Classic play controls below video
- Duration in status bar

**YMXP Theme:**
- Windows Media Player 9 styling
- Luna-themed controls
- Playback slider with XP styling

#### Document Messages

**Default Theme:**
```
┌───────────────────────────────┐
│  📄  document.pdf             │
│      2.4 MB • PDF             │
│                     ⬇️ Download │
└───────────────────────────────┘
```
- File type icon (PDF, Word, Excel, etc.)
- Filename and file size
- Download button

**AIM 1.0 Theme:**
- Windows 98 file icon styling
- Classic "Save As" link
- File properties on hover

**YMXP Theme:**
- Windows XP file icons
- XP-styled download button
- File info tooltip

#### Voice Messages

**Default Theme:**
```
┌───────────────────────────────────────┐
│  ▶️  ▁▂▃▅▃▂▁▃▅▇▅▃▂▁▃▅▃▂  0:15 / 0:42  │
└───────────────────────────────────────┘
```
- Play/pause button
- Waveform visualization
- Duration and progress
- Playback speed option (1x, 1.5x, 2x)

**AIM 1.0 Theme:**
- Windows 98 audio player style
- Classic play/stop buttons
- Progress bar with classic styling

**YMXP Theme:**
- Windows Media Player 9 mini mode
- XP-styled slider
- Time display

### 4. Media Gallery

Access all shared media in a chat through info panel.

**Sections:**
- Images & Videos grid
- Documents list
- Voice Messages list
- Links (extracted from messages)

**Default Theme:**
- Modal or slide-in panel
- Tabbed interface
- Thumbnail grid for images/videos
- List view for documents

**AIM 1.0 Theme:**
- Windows 98 dialog with tabs
- ListView control styling
- File manager appearance

**YMXP Theme:**
- XP Explorer-style view
- Thumbnail view option
- Details view option

### 5. Lightbox Viewer

Full-screen image/video viewer.

**Features:**
- Swipe/arrow navigation between media
- Zoom in/out (pinch or scroll)
- Download button
- Share button
- Close button (X or Escape)
- Image info (filename, date, size)

**Default Theme:**
- Dark overlay with centered media
- Smooth animations
- Touch-friendly controls

**AIM 1.0 Theme:**
- Windows 98 picture viewer window
- Classic toolbar buttons
- Status bar with info

**YMXP Theme:**
- Windows Picture and Fax Viewer styling
- XP toolbar
- Slideshow controls

### 6. Voice Recording

Record and send voice messages.

**User Flow:**
1. Hold microphone button to record
2. Slide left to cancel, release to send
3. Or: click to start, click again to stop
4. Preview with playback before sending
5. Option to delete and re-record

**Default Theme:**
- Microphone button replaces send when input is empty
- Recording indicator with duration
- Waveform animation while recording
- Red dot pulsing indicator

**AIM 1.0 Theme:**
- Record button in toolbar
- Windows 98 Sound Recorder style
- VU meter display

**YMXP Theme:**
- XP Sound Recorder inspired
- Luna-styled record button
- Timer display

## Database Schema

### Media Table (extends existing message_media)

```sql
CREATE TABLE IF NOT EXISTS media (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id),
    message_id VARCHAR REFERENCES messages(id) ON DELETE CASCADE,
    type VARCHAR NOT NULL, -- 'image', 'video', 'audio', 'voice', 'document'
    filename VARCHAR NOT NULL,
    original_filename VARCHAR NOT NULL,
    content_type VARCHAR NOT NULL,
    size BIGINT NOT NULL,
    url VARCHAR NOT NULL,
    thumbnail_url VARCHAR,

    -- Dimensions (images/videos)
    width INTEGER,
    height INTEGER,

    -- Duration (audio/video in milliseconds)
    duration INTEGER,

    -- Voice message waveform (JSON array of amplitudes)
    waveform VARCHAR,

    -- Blurhash for progressive loading
    blurhash VARCHAR,

    -- View once support
    is_view_once BOOLEAN DEFAULT FALSE,
    view_count INTEGER DEFAULT 0,
    viewed_at TIMESTAMP,

    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP,
    deleted_at TIMESTAMP
);

CREATE INDEX idx_media_user_id ON media(user_id);
CREATE INDEX idx_media_message_id ON media(message_id);
CREATE INDEX idx_media_type ON media(type);
CREATE INDEX idx_media_created_at ON media(created_at);
```

### Upload Progress Table (for resumable uploads)

```sql
CREATE TABLE IF NOT EXISTS upload_progress (
    id VARCHAR PRIMARY KEY,
    user_id VARCHAR NOT NULL REFERENCES users(id),
    filename VARCHAR NOT NULL,
    content_type VARCHAR NOT NULL,
    total_size BIGINT NOT NULL,
    uploaded_size BIGINT DEFAULT 0,
    chunk_size INTEGER DEFAULT 1048576, -- 1MB chunks
    chunks_completed VARCHAR, -- JSON array of completed chunk numbers
    storage_path VARCHAR,
    status VARCHAR DEFAULT 'pending', -- 'pending', 'uploading', 'completed', 'failed', 'cancelled'
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP
);

CREATE INDEX idx_upload_progress_user_id ON upload_progress(user_id);
CREATE INDEX idx_upload_progress_status ON upload_progress(status);
```

## API Endpoints

### Upload Endpoints

```
POST   /api/v1/media/upload              # Upload file (multipart/form-data)
POST   /api/v1/media/upload/chunk        # Upload chunk (for large files)
POST   /api/v1/media/upload/init         # Initialize chunked upload
POST   /api/v1/media/upload/complete     # Complete chunked upload
DELETE /api/v1/media/upload/{id}         # Cancel upload
```

### Media Endpoints

```
GET    /api/v1/media/{id}                # Get media metadata
GET    /api/v1/media/{id}/download       # Download original file
GET    /api/v1/media/{id}/thumbnail      # Get thumbnail
GET    /api/v1/media/{id}/stream         # Stream video/audio
DELETE /api/v1/media/{id}                # Delete media
```

### Chat Media Endpoints

```
GET    /api/v1/chats/{id}/media          # List all media in chat
GET    /api/v1/chats/{id}/media/images   # List images only
GET    /api/v1/chats/{id}/media/videos   # List videos only
GET    /api/v1/chats/{id}/media/docs     # List documents only
GET    /api/v1/chats/{id}/media/voice    # List voice messages only
```

### Request/Response Examples

**Upload Request:**
```
POST /api/v1/media/upload
Content-Type: multipart/form-data

file: (binary data)
type: image
caption: "Check out this photo!"
chat_id: 01HXYZ...
```

**Upload Response:**
```json
{
    "success": true,
    "data": {
        "id": "01HXYZ123...",
        "type": "image",
        "filename": "photo.jpg",
        "content_type": "image/jpeg",
        "size": 245678,
        "url": "/media/01HXYZ123.jpg",
        "thumbnail_url": "/media/01HXYZ123_thumb.jpg",
        "width": 1920,
        "height": 1080,
        "blurhash": "LKO2?U%2Tw=w]~RBVZRi...",
        "created_at": "2024-01-15T10:30:00Z"
    }
}
```

**Send Media Message:**
```
POST /api/v1/chats/{chat_id}/messages
Content-Type: application/json

{
    "type": "image",
    "content": "Check out this photo!",
    "media_ids": ["01HXYZ123..."]
}
```

## File Structure

```
assets/
├── static/
│   ├── css/
│   │   ├── default.css      # Updated with media styles
│   │   ├── aim.css          # Updated with media styles
│   │   └── ymxp.css         # Updated with media styles
│   └── js/
│       ├── app.js           # Updated with media logic
│       └── media.js         # Media upload/preview/playback logic
└── views/
    ├── default/pages/app.html
    ├── aim1.0/pages/app.html
    └── ymxp/pages/app.html

store/duckdb/
├── media_store.go          # Media CRUD operations

feature/media/
├── api.go                  # Already exists, extended
├── service.go              # Already exists, extended
└── filestore.go            # Local filesystem storage

app/web/
├── handler/
│   └── media.go            # Media upload/download handlers
└── server.go               # Updated with media routes
```

## Feature Implementation

### media/filestore.go

```go
// LocalFileStore implements FileStore for local filesystem storage
type LocalFileStore struct {
    basePath string
    baseURL  string
}

func NewLocalFileStore(basePath, baseURL string) *LocalFileStore

func (s *LocalFileStore) Save(ctx context.Context, filename, contentType string, reader io.Reader) (url string, err error)
func (s *LocalFileStore) Delete(ctx context.Context, url string) error
func (s *LocalFileStore) GetURL(ctx context.Context, path string) string
func (s *LocalFileStore) GetPath(ctx context.Context, url string) string

// Thumbnail generation
func (s *LocalFileStore) GenerateThumbnail(ctx context.Context, path string, mediaType string) (string, error)

// Metadata extraction
func (s *LocalFileStore) ExtractMetadata(ctx context.Context, path string, mediaType string) (*MediaMetadata, error)
```

### handler/media.go

```go
type MediaHandler struct {
    media media.API
    getUserID func(*mizu.Ctx) string
}

func (h *MediaHandler) Upload(c *mizu.Ctx) error
func (h *MediaHandler) Download(c *mizu.Ctx) error
func (h *MediaHandler) Thumbnail(c *mizu.Ctx) error
func (h *MediaHandler) Stream(c *mizu.Ctx) error
func (h *MediaHandler) Delete(c *mizu.Ctx) error
func (h *MediaHandler) GetChatMedia(c *mizu.Ctx) error
```

## JavaScript Implementation

### media.js

```javascript
// Media upload handler
class MediaUploader {
    constructor(options)
    upload(file, chatId, onProgress)
    cancel()

    // Chunk upload for large files
    uploadChunked(file, chatId, onProgress)
}

// Voice recorder
class VoiceRecorder {
    constructor(options)
    start()
    stop()
    cancel()
    getWaveform()
    getBlob()
}

// Media preview
function showMediaPreview(file, onSend, onCancel)
function showLightbox(mediaList, startIndex)
function closeAllMediaPreviews()

// Drag and drop
function initDragDrop(dropZone, onDrop)

// Clipboard paste
function initClipboardPaste(inputArea, onPaste)

// Audio player
class AudioPlayer {
    constructor(element, src, waveform)
    play()
    pause()
    seek(time)
    setPlaybackRate(rate)
}

// Video player
class VideoPlayer {
    constructor(element, src, poster)
    play()
    pause()
    toggleFullscreen()
}
```

## CSS Additions

### Default Theme Media Styles

```css
/* Media Message */
.message-media {
    max-width: 300px;
    border-radius: 12px;
    overflow: hidden;
}

.message-media img,
.message-media video {
    width: 100%;
    display: block;
}

.message-media-overlay {
    position: absolute;
    bottom: 0;
    left: 0;
    right: 0;
    padding: 8px;
    background: linear-gradient(transparent, rgba(0,0,0,0.6));
}

/* Document Message */
.message-document {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: 12px;
}

.document-icon { /* File type specific */ }
.document-info { /* Filename, size */ }
.document-download { /* Download button */ }

/* Voice Message */
.message-voice {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    min-width: 200px;
}

.voice-play-btn { /* Play/pause button */ }
.voice-waveform { /* SVG waveform */ }
.voice-duration { /* Time display */ }

/* Upload Progress */
.upload-progress {
    position: relative;
    background: var(--bg-tertiary);
    border-radius: 8px;
    overflow: hidden;
}

.upload-progress-bar {
    height: 4px;
    background: var(--accent);
    transition: width 0.2s;
}

/* Drag and Drop Zone */
.drop-zone {
    position: absolute;
    inset: 0;
    background: rgba(37, 211, 102, 0.1);
    border: 2px dashed var(--accent);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 100;
}

/* Lightbox */
.lightbox {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.95);
    z-index: 200;
    display: flex;
    align-items: center;
    justify-content: center;
}

.lightbox-content { /* Centered media */ }
.lightbox-nav { /* Navigation arrows */ }
.lightbox-toolbar { /* Download, share, close */ }
```

### AIM 1.0 Theme Media Styles

```css
/* Windows 98 styled media */
.message-media {
    border: 2px solid;
    border-color: var(--win98-button-shadow) var(--win98-button-highlight)
                  var(--win98-button-highlight) var(--win98-button-shadow);
}

.message-document {
    background: var(--win98-bg);
    border: 2px solid;
    /* Inset 3D border */
}

/* Classic file icons */
.document-icon-pdf { background-image: url('...'); }
.document-icon-doc { background-image: url('...'); }

/* Windows Sound Recorder style voice message */
.message-voice {
    background: var(--win98-bg);
    border: 2px inset;
}
```

### YMXP Theme Media Styles

```css
/* Windows XP styled media */
.message-media {
    border: 1px solid #8e8f8f;
    border-radius: 3px;
}

.message-document {
    background: linear-gradient(180deg, #fefefe 0%, #ece9d8 100%);
    border: 1px solid #8e8f8f;
    border-radius: 3px;
}

/* XP file icons */
.document-icon { /* XP icon styling */ }

/* Windows Media Player style audio */
.message-voice {
    background: linear-gradient(180deg, #3e6aaa 0%, #1d3a5c 100%);
    border-radius: 5px;
}
```

## WebSocket Events

### New Events

```javascript
// Media upload progress (for multi-device sync)
{
    type: 'MEDIA_UPLOAD_PROGRESS',
    data: {
        upload_id: '...',
        progress: 75,
        status: 'uploading'
    }
}

// Media processing complete (thumbnail ready, etc.)
{
    type: 'MEDIA_PROCESSED',
    data: {
        media_id: '...',
        thumbnail_url: '...',
        width: 1920,
        height: 1080
    }
}
```

## Security Considerations

1. **File Validation**: Validate file magic bytes, not just extension
2. **Size Limits**: Enforce per-type size limits
3. **Content Type**: Validate Content-Type matches actual file
4. **Malware Scan**: Optional integration with antivirus scanning
5. **Private URLs**: Use signed URLs with expiration for private media
6. **CORS**: Proper CORS headers for media endpoints
7. **Rate Limiting**: Limit upload frequency per user

## Performance Considerations

1. **Lazy Loading**: Load media thumbnails only when in viewport
2. **Progressive Loading**: Use blurhash placeholders
3. **Chunked Upload**: Support resumable uploads for large files
4. **Thumbnail Caching**: Cache generated thumbnails
5. **CDN Ready**: URL structure supports CDN integration
6. **Compression**: Compress images before storage (optional)
7. **Stream Video**: Use HTTP range requests for video streaming

## Implementation Order

1. **Phase 1: Core Upload**
   - Implement LocalFileStore
   - Add media store (DuckDB)
   - Add upload endpoint
   - Basic file serving

2. **Phase 2: Media Messages**
   - Update message creation to support media
   - Render media messages in chat
   - Default theme styling

3. **Phase 3: Advanced Features**
   - Thumbnail generation
   - Voice recording
   - Video playback
   - Document previews

4. **Phase 4: Theme Support**
   - AIM 1.0 theme styling
   - YMXP theme styling
   - Theme-specific icons

5. **Phase 5: Polish**
   - Drag and drop
   - Clipboard paste
   - Media gallery
   - Lightbox viewer
   - Voice message waveforms

## Testing

- Unit tests for media service
- Integration tests for upload/download endpoints
- E2E tests for file sharing flow
- Cross-theme visual regression tests
- Large file upload tests
- Concurrent upload tests
- Mobile touch interaction tests
