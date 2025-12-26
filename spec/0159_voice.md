# Voice Messaging Feature Specification

## Overview

This specification defines the voice messaging feature for the Mizu messaging app blueprint. Voice messaging allows users to record, send, and playback voice messages with a rich, theme-aware UI that works across all supported themes (default dark/light, AIM 1.0, and Yahoo Messenger XP).

## Feature Requirements

### Functional Requirements

1. **Recording**
   - Press-and-hold to record (mobile-friendly)
   - Click to start, click to stop (desktop alternative)
   - Real-time recording duration display
   - Visual recording indicator (pulsing animation)
   - Cancel recording by sliding away or pressing cancel
   - Maximum recording duration: 5 minutes
   - Minimum recording duration: 1 second (to prevent accidental recordings)

2. **Playback**
   - Play/pause toggle button
   - Waveform visualization
   - Progress indicator showing current position
   - Duration display (elapsed/total)
   - Seek functionality (click on waveform to jump)
   - Auto-pause when another audio starts playing

3. **Message Display**
   - Voice messages show waveform preview
   - Duration badge
   - Play count for view-once messages
   - Sender avatar (in group chats)
   - Delivery status indicators

### Non-Functional Requirements

1. **Performance**
   - Lazy load audio files
   - Preload metadata only
   - Efficient waveform rendering using canvas/SVG

2. **Accessibility**
   - Keyboard navigation support
   - Screen reader announcements
   - High contrast visual indicators

3. **Browser Support**
   - Modern browsers with MediaRecorder API support
   - Fallback message for unsupported browsers

## Technical Design

### Data Model

Voice messages use the existing message and media infrastructure:

```go
// Message with type="voice"
type Message struct {
    ID             string    `json:"id"`
    ChatID         string    `json:"chat_id"`
    SenderID       string    `json:"sender_id"`
    Type           string    `json:"type"`           // "voice"
    Content        string    `json:"content"`        // Optional caption
    MediaID        string    `json:"media_id"`       // Reference to media
    MediaURL       string    `json:"media_url"`      // Direct URL
    MediaDuration  int       `json:"media_duration"` // Duration in ms
    MediaWaveform  []float64 `json:"media_waveform"` // Waveform data array
    CreatedAt      time.Time `json:"created_at"`
}

// Media table (existing)
type Media struct {
    ID              string    `json:"id"`
    UserID          string    `json:"user_id"`
    Type            string    `json:"type"`           // "voice"
    URL             string    `json:"url"`
    ContentType     string    `json:"content_type"`   // "audio/webm"
    Size            int64     `json:"size"`
    Duration        int       `json:"duration"`       // milliseconds
    Waveform        string    `json:"waveform"`       // JSON array of amplitudes
    IsVoiceNote     bool      `json:"is_voice_note"`
    IsViewOnce      bool      `json:"is_view_once"`
    ViewCount       int       `json:"view_count"`
    CreatedAt       time.Time `json:"created_at"`
}
```

### API Endpoints

Uses existing media upload endpoint:

```
POST /api/v1/media/upload
  - Form data: file (audio blob), type: "voice"
  - Response: { id, url, duration, waveform, content_type }

POST /api/v1/chats/{chatId}/messages
  - Body: { type: "voice", media_id: "...", content: "caption" }
  - Response: Message object

GET /api/v1/media/{id}/stream
  - Streams audio file
  - Supports Range headers for seeking
```

### Frontend Architecture

#### Voice Recorder Component

```javascript
class VoiceRecorder {
    constructor(options) {
        this.onStart = options.onStart || (() => {});
        this.onData = options.onData || (() => {});
        this.onStop = options.onStop || (() => {});
        this.onError = options.onError || (() => {});
        this.onCancel = options.onCancel || (() => {});
        this.onProgress = options.onProgress || (() => {});

        this.mediaRecorder = null;
        this.audioChunks = [];
        this.startTime = 0;
        this.isRecording = false;
        this.analyser = null;
        this.progressInterval = null;
    }

    async start() { ... }
    stop() { ... }
    cancel() { ... }
    getDuration() { ... }
    getAmplitude() { ... }  // For live waveform
}
```

#### Recording UI States

1. **Idle State**
   - Microphone button visible
   - Tooltip: "Hold to record voice message"

2. **Recording State**
   - Red pulsing indicator
   - Duration counter (00:00)
   - Cancel button (trash/X icon)
   - Stop/send button
   - Slide-to-cancel hint (mobile)

3. **Preview State** (optional)
   - Playback controls
   - Delete option
   - Send button

#### Playback UI States

1. **Loading State**
   - Shimmer/pulse animation on waveform
   - Spinner on play button

2. **Ready State**
   - Static waveform
   - Play button
   - Duration label

3. **Playing State**
   - Animated progress on waveform
   - Pause button
   - Elapsed time / total time

4. **Paused State**
   - Static waveform with progress position
   - Play button
   - Current position / total time

### Waveform Generation

Server-side waveform generation for consistent display:

1. Extract audio samples using FFmpeg
2. Normalize to 0-1 range
3. Downsample to ~50-100 data points
4. Store as JSON array in database

Client-side fallback for preview:
1. Use Web Audio API analyser
2. Sample amplitude during recording
3. Normalize and smooth data

### Theme Integration

#### Default Theme (Dark/Light)

```css
.voice-recorder {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 16px;
    background: var(--bg-tertiary);
    border-radius: 24px;
}

.voice-record-btn {
    width: 40px;
    height: 40px;
    border-radius: 50%;
    background: var(--accent);
    color: white;
}

.voice-recording-indicator {
    animation: pulse 1.5s infinite;
}

.voice-waveform {
    fill: var(--accent);
    opacity: 0.7;
}

.voice-waveform-progress {
    fill: var(--accent);
    opacity: 1;
}
```

#### AIM 1.0 Theme (Windows 98)

```css
.voice-recorder {
    background: var(--win98-bg);
    box-shadow: inset -1px -1px 0 var(--win98-button-dark-shadow),
                inset 1px 1px 0 var(--win98-button-highlight);
}

.voice-record-btn {
    background: var(--win98-bg);
    box-shadow: /* Windows 98 button style */;
}

.voice-recording-indicator {
    background: #ff0000;
}
```

#### Yahoo XP Theme

```css
.voice-recorder {
    background: linear-gradient(180deg, #ffffff 0%, #ece9d8 100%);
    border: 1px solid var(--xp-input-border);
    border-radius: 4px;
}

.voice-record-btn {
    background: linear-gradient(/* XP button gradient */);
    border-radius: 3px;
}
```

## User Interface Mockups

### Recording Flow

```
Default Theme:
┌────────────────────────────────────────┐
│ [📎] [Message input...        ] [😀] [🎨] │
│      [🎙️ Press and hold to record]     │
└────────────────────────────────────────┘

Recording:
┌────────────────────────────────────────┐
│ [🗑️]  ●  0:05  ████████░░░░░░░  [⬆️]  │
│       Recording... Slide to cancel      │
└────────────────────────────────────────┘

AIM Theme:
┌────────────────────────────────────────┐
│ ┌──────────────────────────────────┐   │
│ │ 🎙️ │ 0:05 │ ▓▓▓▓░░░░ │ Stop │ X  │   │
│ └──────────────────────────────────┘   │
└────────────────────────────────────────┘
```

### Voice Message Display

```
Default Theme:
┌─────────────────────────────────────┐
│  ┌───────────────────────────────┐  │
│  │ [▶️] ▁▃▅▇▅▃▁▃▅▇▅▃▁  0:15     │  │
│  └───────────────────────────────┘  │
│                           12:34 ✓✓   │
└─────────────────────────────────────┘

Playing:
┌─────────────────────────────────────┐
│  ┌───────────────────────────────┐  │
│  │ [⏸] ▁▃▅▇▅|▃▁▃▅▇▅▃▁  0:07/0:15 │  │
│  └───────────────────────────────┘  │
│                           12:34 ✓✓   │
└─────────────────────────────────────┘
```

## Implementation Checklist

### Phase 1: Core Recording
- [x] VoiceRecorder class in media.js (existing)
- [ ] Recording UI component
- [ ] Recording state management
- [ ] Audio upload integration
- [ ] Waveform capture during recording

### Phase 2: Playback
- [ ] Voice message renderer
- [ ] Waveform SVG/Canvas component
- [ ] Audio player controls
- [ ] Progress tracking
- [ ] Seek functionality

### Phase 3: Theme Support
- [ ] Default theme styles
- [ ] AIM 1.0 theme styles
- [ ] Yahoo XP theme styles
- [ ] Responsive design adjustments

### Phase 4: Polish
- [ ] Accessibility improvements
- [ ] Error handling
- [ ] Browser compatibility
- [ ] Performance optimization

## Error Handling

1. **Microphone Permission Denied**
   - Show modal explaining need for permission
   - Link to browser settings

2. **Recording Failed**
   - Toast notification
   - Option to retry

3. **Upload Failed**
   - Retry button
   - Option to save locally

4. **Playback Failed**
   - Error icon on waveform
   - Retry/download option

## Security Considerations

1. Audio files stored in user-specific directories
2. Access controlled by authentication
3. File size limits enforced (50MB max)
4. Content-type validation
5. No execution of uploaded audio files

## Testing Plan

1. **Unit Tests**
   - VoiceRecorder class methods
   - Waveform generation
   - Duration formatting

2. **Integration Tests**
   - Upload flow
   - Message creation
   - WebSocket broadcast

3. **E2E Tests**
   - Record and send voice message
   - Playback voice message
   - Cross-browser testing

## Future Enhancements

1. Voice-to-text transcription
2. Speed control (0.5x, 1x, 1.5x, 2x)
3. Voice message forwarding
4. Background upload for large files
5. Opus codec support for smaller files
