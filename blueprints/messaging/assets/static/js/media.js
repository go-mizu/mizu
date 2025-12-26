// Media Upload and Preview JavaScript

// ============================================
// MEDIA UPLOAD
// ============================================

// File type configurations
const MEDIA_TYPES = {
    image: {
        accept: 'image/jpeg,image/png,image/gif,image/webp',
        maxSize: 20 * 1024 * 1024, // 20MB
        icon: '🖼️'
    },
    video: {
        accept: 'video/mp4,video/webm,video/quicktime',
        maxSize: 100 * 1024 * 1024, // 100MB
        icon: '🎬'
    },
    audio: {
        accept: 'audio/mpeg,audio/mp4,audio/ogg,audio/wav',
        maxSize: 50 * 1024 * 1024, // 50MB
        icon: '🎵'
    },
    document: {
        accept: '.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.zip,.json,.csv',
        maxSize: 100 * 1024 * 1024, // 100MB
        icon: '📄'
    }
};

// Get media type from file
function getMediaType(file) {
    const type = file.type;
    if (type.startsWith('image/')) return 'image';
    if (type.startsWith('video/')) return 'video';
    if (type.startsWith('audio/')) return 'audio';
    return 'document';
}

// Format file size
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Upload media file
async function uploadMedia(file, onProgress = null) {
    const mediaType = getMediaType(file);
    const config = MEDIA_TYPES[mediaType];

    if (file.size > config.maxSize) {
        throw new Error(`File too large. Maximum size is ${formatFileSize(config.maxSize)}`);
    }

    const formData = new FormData();
    formData.append('file', file);
    formData.append('type', mediaType);

    return new Promise((resolve, reject) => {
        const xhr = new XMLHttpRequest();
        xhr.open('POST', '/api/v1/media/upload');

        xhr.upload.onprogress = (e) => {
            if (e.lengthComputable && onProgress) {
                const percent = Math.round((e.loaded / e.total) * 100);
                onProgress(percent);
            }
        };

        xhr.onload = () => {
            if (xhr.status >= 200 && xhr.status < 300) {
                try {
                    const data = JSON.parse(xhr.responseText);
                    resolve(data.data || data);
                } catch (e) {
                    reject(new Error('Invalid response'));
                }
            } else {
                try {
                    const data = JSON.parse(xhr.responseText);
                    reject(new Error(data.error || 'Upload failed'));
                } catch (e) {
                    reject(new Error('Upload failed'));
                }
            }
        };

        xhr.onerror = () => reject(new Error('Network error'));
        xhr.send(formData);
    });
}

// ============================================
// MEDIA PREVIEW MODAL
// ============================================

// Show file preview before sending
function showMediaPreview(file, onSend, onCancel) {
    const mediaType = getMediaType(file);
    const overlay = document.createElement('div');
    overlay.className = 'media-preview-overlay';
    overlay.id = 'media-preview-overlay';

    let previewContent = '';

    if (mediaType === 'image') {
        previewContent = `<img id="media-preview-content" src="" alt="Preview">`;
    } else if (mediaType === 'video') {
        previewContent = `<video id="media-preview-content" controls></video>`;
    } else if (mediaType === 'audio') {
        previewContent = `
            <div class="audio-preview">
                <div class="audio-icon">🎵</div>
                <audio id="media-preview-content" controls></audio>
            </div>`;
    } else {
        previewContent = `
            <div class="document-preview">
                <div class="document-icon">${getDocumentIcon(file.name)}</div>
                <div class="document-name">${escapeHtml(file.name)}</div>
                <div class="document-size">${formatFileSize(file.size)}</div>
            </div>`;
    }

    overlay.innerHTML = `
        <div class="media-preview-modal">
            <div class="media-preview-header">
                <span class="media-preview-title">Send ${mediaType}</span>
                <button class="media-preview-close" onclick="closeMediaPreview()">&times;</button>
            </div>
            <div class="media-preview-body">
                ${previewContent}
            </div>
            <div class="media-preview-footer">
                <input type="text" id="media-caption" class="media-caption-input" placeholder="Add a caption...">
                <div class="media-preview-actions">
                    <button class="media-preview-btn cancel" onclick="closeMediaPreview()">Cancel</button>
                    <button class="media-preview-btn send" id="media-send-btn">
                        <span class="send-text">Send</span>
                        <span class="send-progress hidden">Uploading...</span>
                    </button>
                </div>
            </div>
            <div class="upload-progress hidden" id="upload-progress">
                <div class="upload-progress-bar" id="upload-progress-bar"></div>
            </div>
        </div>
    `;

    document.body.appendChild(overlay);

    // Load file preview
    if (mediaType === 'image' || mediaType === 'video' || mediaType === 'audio') {
        const reader = new FileReader();
        reader.onload = (e) => {
            const content = document.getElementById('media-preview-content');
            if (content) {
                content.src = e.target.result;
            }
        };
        reader.readAsDataURL(file);
    }

    // Send button handler
    const sendBtn = document.getElementById('media-send-btn');
    sendBtn.onclick = async () => {
        const caption = document.getElementById('media-caption').value;
        sendBtn.disabled = true;
        sendBtn.querySelector('.send-text').classList.add('hidden');
        sendBtn.querySelector('.send-progress').classList.remove('hidden');

        const progressContainer = document.getElementById('upload-progress');
        const progressBar = document.getElementById('upload-progress-bar');
        progressContainer.classList.remove('hidden');

        try {
            const media = await uploadMedia(file, (percent) => {
                progressBar.style.width = percent + '%';
            });
            closeMediaPreview();
            if (onSend) {
                onSend(media, caption);
            }
        } catch (err) {
            alert(err.message);
            sendBtn.disabled = false;
            sendBtn.querySelector('.send-text').classList.remove('hidden');
            sendBtn.querySelector('.send-progress').classList.add('hidden');
            progressContainer.classList.add('hidden');
        }
    };

    // Close on overlay click
    overlay.onclick = (e) => {
        if (e.target === overlay) {
            closeMediaPreview();
            if (onCancel) onCancel();
        }
    };

    // Close on Escape
    const escHandler = (e) => {
        if (e.key === 'Escape') {
            closeMediaPreview();
            if (onCancel) onCancel();
            document.removeEventListener('keydown', escHandler);
        }
    };
    document.addEventListener('keydown', escHandler);
}

// Close media preview
function closeMediaPreview() {
    const overlay = document.getElementById('media-preview-overlay');
    if (overlay) {
        overlay.remove();
    }
}

// ============================================
// LIGHTBOX VIEWER
// ============================================

let currentLightboxIndex = 0;
let lightboxMediaList = [];

// Show lightbox for image/video
function showLightbox(mediaList, startIndex = 0) {
    lightboxMediaList = mediaList;
    currentLightboxIndex = startIndex;

    const overlay = document.createElement('div');
    overlay.className = 'lightbox-overlay';
    overlay.id = 'lightbox-overlay';

    overlay.innerHTML = `
        <div class="lightbox-content" id="lightbox-content"></div>
        <button class="lightbox-nav lightbox-prev" onclick="lightboxPrev()">&#10094;</button>
        <button class="lightbox-nav lightbox-next" onclick="lightboxNext()">&#10095;</button>
        <div class="lightbox-toolbar">
            <button class="lightbox-btn" onclick="lightboxDownload()" title="Download">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="7 10 12 15 17 10"/>
                    <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
            </button>
            <span class="lightbox-counter" id="lightbox-counter"></span>
            <button class="lightbox-btn lightbox-close" onclick="closeLightbox()" title="Close">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"/>
                    <line x1="6" y1="6" x2="18" y2="18"/>
                </svg>
            </button>
        </div>
    `;

    document.body.appendChild(overlay);
    document.body.style.overflow = 'hidden';

    updateLightboxContent();

    // Close on overlay click
    overlay.onclick = (e) => {
        if (e.target === overlay) {
            closeLightbox();
        }
    };

    // Keyboard navigation
    document.addEventListener('keydown', lightboxKeyHandler);
}

function lightboxKeyHandler(e) {
    if (e.key === 'Escape') {
        closeLightbox();
    } else if (e.key === 'ArrowLeft') {
        lightboxPrev();
    } else if (e.key === 'ArrowRight') {
        lightboxNext();
    }
}

function updateLightboxContent() {
    const content = document.getElementById('lightbox-content');
    const counter = document.getElementById('lightbox-counter');
    const media = lightboxMediaList[currentLightboxIndex];

    if (!media) return;

    if (media.type === 'image') {
        content.innerHTML = `<img src="${media.url}" alt="${escapeHtml(media.original_filename || '')}">`;
    } else if (media.type === 'video') {
        content.innerHTML = `<video src="${media.url}" controls autoplay></video>`;
    }

    if (lightboxMediaList.length > 1) {
        counter.textContent = `${currentLightboxIndex + 1} / ${lightboxMediaList.length}`;
    } else {
        counter.textContent = '';
    }

    // Hide/show nav buttons
    document.querySelector('.lightbox-prev').style.display = currentLightboxIndex > 0 ? 'flex' : 'none';
    document.querySelector('.lightbox-next').style.display = currentLightboxIndex < lightboxMediaList.length - 1 ? 'flex' : 'none';
}

function lightboxPrev() {
    if (currentLightboxIndex > 0) {
        currentLightboxIndex--;
        updateLightboxContent();
    }
}

function lightboxNext() {
    if (currentLightboxIndex < lightboxMediaList.length - 1) {
        currentLightboxIndex++;
        updateLightboxContent();
    }
}

function lightboxDownload() {
    const media = lightboxMediaList[currentLightboxIndex];
    if (media) {
        const link = document.createElement('a');
        link.href = media.url;
        link.download = media.original_filename || 'download';
        link.click();
    }
}

function closeLightbox() {
    const overlay = document.getElementById('lightbox-overlay');
    if (overlay) {
        overlay.remove();
        document.body.style.overflow = '';
        document.removeEventListener('keydown', lightboxKeyHandler);
    }
}

// ============================================
// DRAG AND DROP
// ============================================

function initDragDrop(dropZone, onDrop) {
    let dragCounter = 0;

    dropZone.addEventListener('dragenter', (e) => {
        e.preventDefault();
        dragCounter++;
        showDropZone(dropZone);
    });

    dropZone.addEventListener('dragleave', (e) => {
        e.preventDefault();
        dragCounter--;
        if (dragCounter === 0) {
            hideDropZone(dropZone);
        }
    });

    dropZone.addEventListener('dragover', (e) => {
        e.preventDefault();
    });

    dropZone.addEventListener('drop', (e) => {
        e.preventDefault();
        dragCounter = 0;
        hideDropZone(dropZone);

        const files = Array.from(e.dataTransfer.files);
        if (files.length > 0 && onDrop) {
            onDrop(files);
        }
    });
}

function showDropZone(element) {
    let dropIndicator = element.querySelector('.drop-zone-indicator');
    if (!dropIndicator) {
        dropIndicator = document.createElement('div');
        dropIndicator.className = 'drop-zone-indicator';
        dropIndicator.innerHTML = `
            <div class="drop-zone-content">
                <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="17 8 12 3 7 8"/>
                    <line x1="12" y1="3" x2="12" y2="15"/>
                </svg>
                <span>Drop files here</span>
            </div>
        `;
        element.style.position = 'relative';
        element.appendChild(dropIndicator);
    }
    dropIndicator.classList.add('active');
}

function hideDropZone(element) {
    const dropIndicator = element.querySelector('.drop-zone-indicator');
    if (dropIndicator) {
        dropIndicator.classList.remove('active');
    }
}

// ============================================
// CLIPBOARD PASTE
// ============================================

function initClipboardPaste(inputElement, onPaste) {
    inputElement.addEventListener('paste', async (e) => {
        const items = e.clipboardData?.items;
        if (!items) return;

        for (const item of items) {
            if (item.type.startsWith('image/')) {
                e.preventDefault();
                const file = item.getAsFile();
                if (file && onPaste) {
                    onPaste(file);
                }
                break;
            }
        }
    });
}

// ============================================
// VOICE RECORDING
// ============================================

class VoiceRecorder {
    constructor(options = {}) {
        this.onStart = options.onStart || (() => {});
        this.onData = options.onData || (() => {});
        this.onStop = options.onStop || (() => {});
        this.onError = options.onError || (() => {});
        this.onCancel = options.onCancel || (() => {});
        this.onAmplitude = options.onAmplitude || (() => {});
        this.mediaRecorder = null;
        this.audioChunks = [];
        this.startTime = 0;
        this.isRecording = false;
        this.analyser = null;
        this.audioContext = null;
        this.stream = null;
        this.amplitudeInterval = null;
        this.waveformData = [];
        this.maxDuration = options.maxDuration || 300000; // 5 minutes default
        this.minDuration = options.minDuration || 1000; // 1 second minimum
        this.maxDurationTimeout = null;
    }

    async checkPermission() {
        try {
            const result = await navigator.permissions.query({ name: 'microphone' });
            return result.state;
        } catch {
            return 'prompt';
        }
    }

    async start() {
        try {
            this.stream = await navigator.mediaDevices.getUserMedia({
                audio: {
                    echoCancellation: true,
                    noiseSuppression: true,
                    autoGainControl: true
                }
            });

            // Set up audio analysis for waveform
            this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
            const source = this.audioContext.createMediaStreamSource(this.stream);
            this.analyser = this.audioContext.createAnalyser();
            this.analyser.fftSize = 256;
            source.connect(this.analyser);

            // Choose best available format
            let mimeType = 'audio/webm';
            if (MediaRecorder.isTypeSupported('audio/webm;codecs=opus')) {
                mimeType = 'audio/webm;codecs=opus';
            } else if (MediaRecorder.isTypeSupported('audio/mp4')) {
                mimeType = 'audio/mp4';
            } else if (MediaRecorder.isTypeSupported('audio/ogg')) {
                mimeType = 'audio/ogg';
            }

            this.mediaRecorder = new MediaRecorder(this.stream, { mimeType });
            this.audioChunks = [];
            this.waveformData = [];
            this.startTime = Date.now();
            this.isRecording = true;

            this.mediaRecorder.ondataavailable = (e) => {
                if (e.data.size > 0) {
                    this.audioChunks.push(e.data);
                    this.onData(e.data);
                }
            };

            this.mediaRecorder.onstop = () => {
                const duration = Date.now() - this.startTime;
                if (duration < this.minDuration) {
                    this.cleanup();
                    this.onCancel('Recording too short');
                    return;
                }
                const blob = new Blob(this.audioChunks, { type: mimeType });
                this.onStop(blob, duration, this.waveformData);
                this.cleanup();
            };

            this.mediaRecorder.onerror = (e) => {
                this.cleanup();
                this.onError(e.error || new Error('Recording failed'));
            };

            this.mediaRecorder.start(100);
            this.onStart();

            // Start amplitude monitoring
            this.startAmplitudeMonitor();

            // Set max duration timeout
            this.maxDurationTimeout = setTimeout(() => {
                if (this.isRecording) {
                    this.stop();
                }
            }, this.maxDuration);

        } catch (err) {
            console.error('Failed to start recording:', err);
            this.onError(err);
            throw err;
        }
    }

    startAmplitudeMonitor() {
        if (!this.analyser) return;

        const dataArray = new Uint8Array(this.analyser.frequencyBinCount);

        this.amplitudeInterval = setInterval(() => {
            if (!this.isRecording) return;

            this.analyser.getByteFrequencyData(dataArray);
            const sum = dataArray.reduce((a, b) => a + b, 0);
            const avg = sum / dataArray.length;
            const normalized = avg / 255;

            this.waveformData.push(normalized);
            this.onAmplitude(normalized);
        }, 100);
    }

    stop() {
        if (this.mediaRecorder && this.isRecording) {
            this.isRecording = false;
            if (this.maxDurationTimeout) {
                clearTimeout(this.maxDurationTimeout);
            }
            this.mediaRecorder.stop();
        }
    }

    cancel() {
        if (this.isRecording) {
            this.isRecording = false;
            if (this.maxDurationTimeout) {
                clearTimeout(this.maxDurationTimeout);
            }
            this.cleanup();
            this.onCancel('Cancelled by user');
        }
    }

    cleanup() {
        if (this.amplitudeInterval) {
            clearInterval(this.amplitudeInterval);
            this.amplitudeInterval = null;
        }
        if (this.audioContext) {
            this.audioContext.close();
            this.audioContext = null;
        }
        if (this.stream) {
            this.stream.getTracks().forEach(track => track.stop());
            this.stream = null;
        }
        this.analyser = null;
        this.audioChunks = [];
    }

    getDuration() {
        return this.isRecording ? Date.now() - this.startTime : 0;
    }

    getAmplitude() {
        if (!this.analyser) return 0;
        const dataArray = new Uint8Array(this.analyser.frequencyBinCount);
        this.analyser.getByteFrequencyData(dataArray);
        const sum = dataArray.reduce((a, b) => a + b, 0);
        return sum / dataArray.length / 255;
    }
}

// ============================================
// VOICE RECORDING UI
// ============================================

let activeVoiceRecorder = null;
let voiceRecordingUI = null;

// Check if browser supports voice recording
function supportsVoiceRecording() {
    return !!(navigator.mediaDevices && navigator.mediaDevices.getUserMedia && window.MediaRecorder);
}

// Show microphone permission modal
function showMicPermissionModal(onAllow) {
    const isRetroTheme = document.documentElement.getAttribute('data-theme')?.includes('aim') ||
                         document.documentElement.getAttribute('data-theme')?.includes('ymxp');

    const modal = document.createElement('div');
    modal.className = 'voice-permission-modal';
    modal.id = 'voice-permission-modal';

    modal.innerHTML = `
        <div class="voice-permission-content ${isRetroTheme ? 'retro' : ''}">
            <div class="voice-permission-icon">
                <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11a7 7 0 01-7 7m0 0a7 7 0 01-7-7m7 7v4m0 0H8m4 0h4m-4-8a3 3 0 01-3-3V5a3 3 0 116 0v6a3 3 0 01-3 3z"/>
                </svg>
            </div>
            <h3 class="voice-permission-title">Microphone Access</h3>
            <p class="voice-permission-text">
                To send voice messages, please allow microphone access when prompted by your browser.
            </p>
            <button class="voice-permission-btn" onclick="this.closest('.voice-permission-modal').remove(); (${onAllow.toString()})();">
                Allow Microphone
            </button>
        </div>
    `;

    document.body.appendChild(modal);

    modal.onclick = (e) => {
        if (e.target === modal) {
            modal.remove();
        }
    };
}

// Create voice recording UI
function createVoiceRecordingUI(container, onSend, onCancel) {
    const isRetroTheme = document.documentElement.getAttribute('data-theme')?.includes('aim') ||
                         document.documentElement.getAttribute('data-theme')?.includes('ymxp');

    const ui = document.createElement('div');
    ui.className = 'voice-recording-container';
    ui.id = 'voice-recording-ui';

    // Create waveform bars
    let barsHtml = '';
    for (let i = 0; i < 20; i++) {
        barsHtml += `<div class="voice-live-waveform-bar" style="height: 8px;"></div>`;
    }

    ui.innerHTML = `
        <button class="voice-cancel-btn" title="Cancel">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/>
            </svg>
        </button>
        <div class="voice-recording-indicator">
            <div class="voice-recording-dot"></div>
            <span class="voice-recording-time">0:00</span>
        </div>
        <div class="voice-live-waveform">${barsHtml}</div>
        <div class="voice-slide-hint">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 19l-7-7m0 0l7-7m-7 7h18"/>
            </svg>
            <span>Slide to cancel</span>
        </div>
        <button class="voice-send-btn" title="Send">
            <svg fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8"/>
            </svg>
        </button>
    `;

    const cancelBtn = ui.querySelector('.voice-cancel-btn');
    const sendBtn = ui.querySelector('.voice-send-btn');
    const timeDisplay = ui.querySelector('.voice-recording-time');
    const waveformBars = ui.querySelectorAll('.voice-live-waveform-bar');

    cancelBtn.onclick = () => {
        onCancel();
    };

    sendBtn.onclick = () => {
        onSend();
    };

    // Slide to cancel functionality
    let startX = 0;
    let isDragging = false;

    ui.addEventListener('touchstart', (e) => {
        startX = e.touches[0].clientX;
        isDragging = true;
    });

    ui.addEventListener('touchmove', (e) => {
        if (!isDragging) return;
        const currentX = e.touches[0].clientX;
        const diff = startX - currentX;
        if (diff > 100) {
            isDragging = false;
            onCancel();
        }
    });

    ui.addEventListener('touchend', () => {
        isDragging = false;
    });

    container.appendChild(ui);

    return {
        element: ui,
        updateTime: (ms) => {
            timeDisplay.textContent = formatDuration(ms);
        },
        updateWaveform: (amplitude) => {
            // Shift bars left and add new value
            for (let i = 0; i < waveformBars.length - 1; i++) {
                waveformBars[i].style.height = waveformBars[i + 1].style.height;
            }
            const height = Math.max(4, Math.min(28, amplitude * 28));
            waveformBars[waveformBars.length - 1].style.height = height + 'px';
        },
        remove: () => {
            ui.remove();
        }
    };
}

// Start voice recording
async function startVoiceRecording(container, onComplete, onError) {
    if (!supportsVoiceRecording()) {
        onError(new Error('Voice recording not supported in this browser'));
        return;
    }

    if (activeVoiceRecorder) {
        activeVoiceRecorder.cancel();
    }

    let timeUpdateInterval = null;

    activeVoiceRecorder = new VoiceRecorder({
        onStart: () => {
            voiceRecordingUI = createVoiceRecordingUI(
                container,
                () => activeVoiceRecorder.stop(),
                () => activeVoiceRecorder.cancel()
            );

            timeUpdateInterval = setInterval(() => {
                if (activeVoiceRecorder) {
                    voiceRecordingUI.updateTime(activeVoiceRecorder.getDuration());
                }
            }, 100);
        },
        onAmplitude: (amp) => {
            if (voiceRecordingUI) {
                voiceRecordingUI.updateWaveform(amp);
            }
        },
        onStop: async (blob, duration, waveform) => {
            if (timeUpdateInterval) clearInterval(timeUpdateInterval);
            if (voiceRecordingUI) voiceRecordingUI.remove();
            voiceRecordingUI = null;
            activeVoiceRecorder = null;

            // Normalize waveform to 50 samples
            const normalizedWaveform = normalizeWaveform(waveform, 50);

            onComplete(blob, duration, normalizedWaveform);
        },
        onCancel: (reason) => {
            if (timeUpdateInterval) clearInterval(timeUpdateInterval);
            if (voiceRecordingUI) voiceRecordingUI.remove();
            voiceRecordingUI = null;
            activeVoiceRecorder = null;
        },
        onError: (err) => {
            if (timeUpdateInterval) clearInterval(timeUpdateInterval);
            if (voiceRecordingUI) voiceRecordingUI.remove();
            voiceRecordingUI = null;
            activeVoiceRecorder = null;
            onError(err);
        }
    });

    try {
        await activeVoiceRecorder.start();
    } catch (err) {
        if (err.name === 'NotAllowedError' || err.name === 'PermissionDeniedError') {
            showMicPermissionModal(() => startVoiceRecording(container, onComplete, onError));
        } else {
            onError(err);
        }
    }
}

// Normalize waveform to specified number of samples
function normalizeWaveform(data, samples) {
    if (!data || data.length === 0) return [];

    const result = [];
    const step = data.length / samples;

    for (let i = 0; i < samples; i++) {
        const start = Math.floor(i * step);
        const end = Math.floor((i + 1) * step);
        let sum = 0;
        for (let j = start; j < end && j < data.length; j++) {
            sum += data[j];
        }
        result.push(sum / (end - start) || 0);
    }

    // Normalize to 0-1 range
    const max = Math.max(...result, 0.1);
    return result.map(v => v / max);
}

// Upload voice message
async function uploadVoiceMessage(blob, duration, waveform) {
    const formData = new FormData();
    const filename = `voice_${Date.now()}.webm`;
    formData.append('file', blob, filename);
    formData.append('type', 'voice');
    formData.append('duration', duration.toString());
    if (waveform && waveform.length > 0) {
        formData.append('waveform', JSON.stringify(waveform));
    }

    const response = await fetch('/api/v1/media/upload', {
        method: 'POST',
        body: formData
    });

    if (!response.ok) {
        throw new Error('Failed to upload voice message');
    }

    const data = await response.json();
    return data.data || data;
}

// Render voice message in chat
function renderVoiceMessage(mediaId, url, duration, waveform, isPlayed = false) {
    const waveformData = typeof waveform === 'string' ? JSON.parse(waveform) : waveform;
    const durationStr = formatDuration(duration);

    // Generate waveform bars
    let barsHtml = '';
    const barCount = waveformData?.length || 30;
    for (let i = 0; i < barCount; i++) {
        const height = waveformData ? Math.max(4, waveformData[i] * 24) : 8;
        barsHtml += `<div class="voice-waveform-bar" style="height: ${height}px;" data-index="${i}"></div>`;
    }

    return `
        <div class="voice-message ${isPlayed ? '' : 'unplayed'}" data-media-id="${mediaId}" data-duration="${duration}">
            <button class="voice-play-btn" onclick="toggleVoiceMessage(this)">
                <svg class="icon-play" fill="currentColor" viewBox="0 0 24 24">
                    <polygon points="5 3 19 12 5 21 5 3"/>
                </svg>
                <svg class="icon-pause" fill="currentColor" viewBox="0 0 24 24">
                    <rect x="6" y="4" width="4" height="16"/>
                    <rect x="14" y="4" width="4" height="16"/>
                </svg>
                <svg class="icon-loading" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <circle cx="12" cy="12" r="10" stroke-width="2" stroke-dasharray="32" stroke-dashoffset="32"/>
                </svg>
            </button>
            <div class="voice-waveform-container">
                <div class="voice-waveform" onclick="seekVoiceMessage(event, this)">
                    <div class="voice-waveform-bars">${barsHtml}</div>
                </div>
                <div class="voice-duration">
                    <span class="voice-time-current">0:00</span>
                    <span class="voice-time-total">${durationStr}</span>
                </div>
            </div>
            <div class="voice-unplayed-indicator"></div>
            <audio src="${url}" preload="metadata"></audio>
        </div>
    `;
}

// Toggle voice message playback
function toggleVoiceMessage(button) {
    const container = button.closest('.voice-message');
    const audio = container.querySelector('audio');
    const waveformBars = container.querySelectorAll('.voice-waveform-bar');
    const currentTime = container.querySelector('.voice-time-current');
    const duration = parseInt(container.dataset.duration, 10);

    // Pause all other voice messages
    document.querySelectorAll('.voice-message audio').forEach(a => {
        if (a !== audio && !a.paused) {
            a.pause();
            const otherContainer = a.closest('.voice-message');
            otherContainer.querySelector('.voice-play-btn').classList.remove('playing');
        }
    });

    if (audio.paused) {
        button.classList.add('playing');
        container.classList.remove('unplayed');
        audio.play();

        // Update progress
        audio.ontimeupdate = () => {
            const progress = audio.currentTime / audio.duration;
            currentTime.textContent = formatDuration(audio.currentTime * 1000);

            // Update waveform bars
            const playedCount = Math.floor(progress * waveformBars.length);
            waveformBars.forEach((bar, i) => {
                bar.classList.toggle('played', i < playedCount);
            });
        };

        audio.onended = () => {
            button.classList.remove('playing');
            currentTime.textContent = '0:00';
            waveformBars.forEach(bar => bar.classList.remove('played'));
        };
    } else {
        button.classList.remove('playing');
        audio.pause();
    }
}

// Seek voice message
function seekVoiceMessage(event, waveformContainer) {
    const container = waveformContainer.closest('.voice-message');
    const audio = container.querySelector('audio');
    const rect = waveformContainer.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const progress = x / rect.width;

    audio.currentTime = progress * audio.duration;

    if (audio.paused) {
        const button = container.querySelector('.voice-play-btn');
        toggleVoiceMessage(button);
    }
}

// ============================================
// MESSAGE MEDIA RENDERING
// ============================================

// Render media message content
function renderMediaMessage(media) {
    if (!media) return '';

    const type = media.type;
    const url = media.url;
    const thumbUrl = media.thumbnail_url || url;
    const filename = media.original_filename || media.filename || 'File';

    if (type === 'image') {
        return `
            <div class="message-media message-image" onclick="showLightbox([${JSON.stringify(media).replace(/"/g, '&quot;')}])">
                <img src="${escapeHtml(thumbUrl)}" alt="${escapeHtml(filename)}" loading="lazy">
            </div>
        `;
    }

    if (type === 'video') {
        return `
            <div class="message-media message-video">
                <video src="${escapeHtml(url)}" poster="${escapeHtml(thumbUrl)}" controls></video>
            </div>
        `;
    }

    if (type === 'audio' || type === 'voice') {
        const waveformData = media.waveform ? JSON.parse(media.waveform) : null;
        const duration = media.duration ? formatDuration(media.duration) : '';

        return `
            <div class="message-media message-audio">
                <button class="audio-play-btn" onclick="toggleAudio(this)">
                    <svg class="play-icon" width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                        <polygon points="5 3 19 12 5 21 5 3"/>
                    </svg>
                    <svg class="pause-icon hidden" width="24" height="24" viewBox="0 0 24 24" fill="currentColor">
                        <rect x="6" y="4" width="4" height="16"/>
                        <rect x="14" y="4" width="4" height="16"/>
                    </svg>
                </button>
                <div class="audio-waveform">
                    ${waveformData ? renderWaveform(waveformData) : '<div class="audio-progress-track"></div>'}
                </div>
                <span class="audio-duration">${duration}</span>
                <audio src="${escapeHtml(url)}" preload="metadata"></audio>
            </div>
        `;
    }

    // Document
    return `
        <div class="message-media message-document">
            <div class="document-icon">${getDocumentIcon(filename)}</div>
            <div class="document-info">
                <div class="document-name">${escapeHtml(filename)}</div>
                <div class="document-size">${formatFileSize(media.size)}</div>
            </div>
            <a class="document-download" href="${escapeHtml(url)}" download="${escapeHtml(filename)}" title="Download">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                    <polyline points="7 10 12 15 17 10"/>
                    <line x1="12" y1="15" x2="12" y2="3"/>
                </svg>
            </a>
        </div>
    `;
}

// Get document icon based on file extension
function getDocumentIcon(filename) {
    const ext = filename.split('.').pop().toLowerCase();
    const icons = {
        pdf: '📕',
        doc: '📘',
        docx: '📘',
        xls: '📗',
        xlsx: '📗',
        ppt: '📙',
        pptx: '📙',
        txt: '📄',
        zip: '🗜️',
        json: '📋',
        csv: '📊'
    };
    return icons[ext] || '📄';
}

// Format duration in milliseconds to mm:ss
function formatDuration(ms) {
    const totalSeconds = Math.floor(ms / 1000);
    const minutes = Math.floor(totalSeconds / 60);
    const seconds = totalSeconds % 60;
    return `${minutes}:${seconds.toString().padStart(2, '0')}`;
}

// Render waveform SVG
function renderWaveform(data) {
    if (!data || data.length === 0) return '';

    const width = 150;
    const height = 32;
    const barWidth = width / data.length;
    const maxValue = Math.max(...data);

    let bars = '';
    data.forEach((value, i) => {
        const barHeight = (value / maxValue) * height * 0.8;
        const y = (height - barHeight) / 2;
        bars += `<rect x="${i * barWidth}" y="${y}" width="${barWidth - 1}" height="${barHeight}" rx="1"/>`;
    });

    return `<svg class="waveform-svg" width="${width}" height="${height}" viewBox="0 0 ${width} ${height}">${bars}</svg>`;
}

// Toggle audio playback
function toggleAudio(button) {
    const container = button.closest('.message-audio');
    const audio = container.querySelector('audio');
    const playIcon = button.querySelector('.play-icon');
    const pauseIcon = button.querySelector('.pause-icon');

    if (audio.paused) {
        // Pause all other audio
        document.querySelectorAll('.message-audio audio').forEach(a => {
            if (a !== audio) {
                a.pause();
                const btn = a.closest('.message-audio').querySelector('.audio-play-btn');
                btn.querySelector('.play-icon').classList.remove('hidden');
                btn.querySelector('.pause-icon').classList.add('hidden');
            }
        });

        audio.play();
        playIcon.classList.add('hidden');
        pauseIcon.classList.remove('hidden');
    } else {
        audio.pause();
        playIcon.classList.remove('hidden');
        pauseIcon.classList.add('hidden');
    }

    audio.onended = () => {
        playIcon.classList.remove('hidden');
        pauseIcon.classList.add('hidden');
    };
}

// ============================================
// HELPER FUNCTIONS
// ============================================

function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

// ============================================
// SEND MESSAGE WITH MEDIA
// ============================================

async function sendMessageWithMedia(chatId, mediaId, caption = '') {
    try {
        const response = await fetch(`/api/v1/chats/${chatId}/messages`, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({
                type: 'media',
                content: caption,
                media_ids: [mediaId],
            }),
        });
        return response.ok;
    } catch (err) {
        console.error('Failed to send message with media:', err);
        return false;
    }
}

// ============================================
// ATTACH BUTTON HANDLER
// ============================================

function createAttachButton(chatId, onMediaSent) {
    const button = document.createElement('button');
    button.className = 'input-attach-btn';
    button.innerHTML = `
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <path d="M21.44 11.05l-9.19 9.19a6 6 0 0 1-8.49-8.49l9.19-9.19a4 4 0 0 1 5.66 5.66l-9.2 9.19a2 2 0 0 1-2.83-2.83l8.49-8.48"/>
        </svg>
    `;
    button.title = 'Attach file';

    button.onclick = () => {
        const input = document.createElement('input');
        input.type = 'file';
        input.multiple = false;
        input.accept = 'image/*,video/*,audio/*,.pdf,.doc,.docx,.xls,.xlsx,.ppt,.pptx,.txt,.zip';

        input.onchange = () => {
            const file = input.files[0];
            if (file) {
                showMediaPreview(file, async (media, caption) => {
                    await sendMessageWithMedia(chatId, media.id, caption);
                    if (onMediaSent) onMediaSent(media);
                });
            }
        };

        input.click();
    };

    return button;
}

// Export for global use
window.MediaUpload = {
    upload: uploadMedia,
    uploadMedia,
    showPreview: showMediaPreview,
    showMediaPreview,
    showLightbox,
    initDragDrop,
    initClipboardPaste,
    VoiceRecorder,
    renderMediaMessage,
    createAttachButton,
    sendMessageWithMedia,
    formatFileSize,
    formatDuration,
    getMediaType,
    // Voice recording
    supportsVoiceRecording,
    startVoiceRecording,
    uploadVoiceMessage,
    renderVoiceMessage,
    toggleVoiceMessage,
    seekVoiceMessage
};
