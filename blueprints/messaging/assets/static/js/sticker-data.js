// Sticker Data - Built-in sticker packs
// Stickers are embedded as SVG data URIs for portability

const STICKER_PACKS = {
    jarvis: {
        id: 'jarvis',
        name: 'J.A.R.V.I.S.',
        thumbnail: 'arc_reactor',
        stickers: [
            {
                id: 'arc_reactor',
                name: 'Arc Reactor',
                tags: ['arc', 'reactor', 'power', 'stark', 'ironman'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="arcG" cx="50%" cy="50%" r="50%"><stop offset="0%" style="stop-color:#FFFFFF"/><stop offset="30%" style="stop-color:#00D4FF"/><stop offset="70%" style="stop-color:#0066FF"/><stop offset="100%" style="stop-color:#001133"/></radialGradient><filter id="glow"><feGaussianBlur stdDeviation="3" result="coloredBlur"/><feMerge><feMergeNode in="coloredBlur"/><feMergeNode in="SourceGraphic"/></feMerge></filter></defs><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><circle cx="60" cy="60" r="48" fill="none" stroke="#00D4FF" stroke-width="2" opacity="0.3" filter="url(#glow)"/><circle cx="60" cy="60" r="40" fill="none" stroke="#00D4FF" stroke-width="2" opacity="0.5" filter="url(#glow)"/><circle cx="60" cy="60" r="32" fill="none" stroke="#00D4FF" stroke-width="3" opacity="0.7" filter="url(#glow)"/><circle cx="60" cy="60" r="20" fill="url(#arcG)" filter="url(#glow)"/><circle cx="60" cy="60" r="10" fill="#FFFFFF" filter="url(#glow)"/><path d="M60 15 L60 25 M60 95 L60 105 M15 60 L25 60 M95 60 L105 60" stroke="#00D4FF" stroke-width="2" opacity="0.5"/></svg>`
            },
            {
                id: 'ai_brain',
                name: 'AI Brain',
                tags: ['ai', 'brain', 'neural', 'thinking', 'jarvis'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><linearGradient id="brainG" x1="0%" y1="0%" x2="100%" y2="100%"><stop offset="0%" style="stop-color:#00D4FF"/><stop offset="100%" style="stop-color:#0066FF"/></linearGradient></defs><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><ellipse cx="60" cy="55" rx="35" ry="30" fill="none" stroke="url(#brainG)" stroke-width="2"/><path d="M35 45 Q45 35, 55 45 Q65 35, 75 45 Q85 55, 75 65 Q85 75, 75 80 Q65 90, 55 80 Q45 90, 35 80 Q25 75, 35 65 Q25 55, 35 45" fill="none" stroke="#00D4FF" stroke-width="2"/><circle cx="45" cy="50" r="3" fill="#00D4FF"/><circle cx="65" cy="50" r="3" fill="#00D4FF"/><circle cx="55" cy="65" r="3" fill="#00D4FF"/><line x1="45" y1="50" x2="55" y2="65" stroke="#00D4FF" stroke-width="1"/><line x1="65" y1="50" x2="55" y2="65" stroke="#00D4FF" stroke-width="1"/><line x1="45" y1="50" x2="65" y2="50" stroke="#00D4FF" stroke-width="1"/><text x="60" y="100" text-anchor="middle" fill="#00D4FF" font-size="10" font-family="monospace">NEURAL</text></svg>`
            },
            {
                id: 'hologram',
                name: 'Hologram',
                tags: ['holo', 'hologram', 'projection', 'future'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><linearGradient id="holoG" x1="0%" y1="100%" x2="0%" y2="0%"><stop offset="0%" style="stop-color:#00D4FF;stop-opacity:0.1"/><stop offset="100%" style="stop-color:#00D4FF;stop-opacity:0.8"/></linearGradient></defs><rect width="120" height="120" fill="#0a0a0f"/><ellipse cx="60" cy="100" rx="40" ry="10" fill="#00D4FF" opacity="0.3"/><path d="M30 100 L50 30 L70 30 L90 100 Z" fill="url(#holoG)"/><circle cx="60" cy="50" r="15" fill="none" stroke="#00D4FF" stroke-width="2"/><circle cx="60" cy="50" r="10" fill="none" stroke="#00D4FF" stroke-width="1" opacity="0.5"/><circle cx="60" cy="50" r="5" fill="#00D4FF"/><line x1="45" y1="50" x2="75" y2="50" stroke="#00D4FF" stroke-width="1" opacity="0.5"/><line x1="60" y1="35" x2="60" y2="65" stroke="#00D4FF" stroke-width="1" opacity="0.5"/></svg>`
            },
            {
                id: 'shield',
                name: 'Shield Active',
                tags: ['shield', 'protection', 'security', 'defense'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><linearGradient id="shieldG" x1="0%" y1="0%" x2="0%" y2="100%"><stop offset="0%" style="stop-color:#00D4FF"/><stop offset="100%" style="stop-color:#0044AA"/></linearGradient></defs><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><path d="M60 15 L95 30 L95 60 Q95 90 60 105 Q25 90 25 60 L25 30 Z" fill="none" stroke="url(#shieldG)" stroke-width="3"/><path d="M60 25 L85 38 L85 58 Q85 82 60 95 Q35 82 35 58 L35 38 Z" fill="url(#shieldG)" opacity="0.3"/><path d="M50 55 L55 65 L75 45" stroke="#FFFFFF" stroke-width="4" fill="none" stroke-linecap="round"/><text x="60" y="115" text-anchor="middle" fill="#00D4FF" font-size="8" font-family="monospace">SECURE</text></svg>`
            },
            {
                id: 'scan',
                name: 'Scanning',
                tags: ['scan', 'search', 'analyze', 'detect'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><circle cx="60" cy="60" r="40" fill="none" stroke="#00D4FF" stroke-width="1" opacity="0.3"/><circle cx="60" cy="60" r="30" fill="none" stroke="#00D4FF" stroke-width="1" opacity="0.5"/><circle cx="60" cy="60" r="20" fill="none" stroke="#00D4FF" stroke-width="1" opacity="0.7"/><circle cx="60" cy="60" r="10" fill="none" stroke="#00D4FF" stroke-width="2"/><line x1="60" y1="20" x2="60" y2="100" stroke="#00D4FF" stroke-width="1" opacity="0.3"/><line x1="20" y1="60" x2="100" y2="60" stroke="#00D4FF" stroke-width="1" opacity="0.3"/><path d="M60 20 L60 60" stroke="#00FF00" stroke-width="2"><animateTransform attributeName="transform" type="rotate" from="0 60 60" to="360 60 60" dur="2s" repeatCount="indefinite"/></path><circle cx="60" cy="60" r="5" fill="#00FF00"/></svg>`
            },
            {
                id: 'power_up',
                name: 'Power Up',
                tags: ['power', 'energy', 'boost', 'charge'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><linearGradient id="powerG" x1="0%" y1="100%" x2="0%" y2="0%"><stop offset="0%" style="stop-color:#FF6B00"/><stop offset="50%" style="stop-color:#FFD700"/><stop offset="100%" style="stop-color:#00D4FF"/></linearGradient></defs><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><polygon points="60,15 75,50 65,50 70,75 50,45 60,45 55,20" fill="url(#powerG)"/><circle cx="60" cy="85" r="15" fill="none" stroke="#00D4FF" stroke-width="2"/><text x="60" y="90" text-anchor="middle" fill="#00D4FF" font-size="12" font-weight="bold">100%</text></svg>`
            },
            {
                id: 'approved',
                name: 'Approved',
                tags: ['approved', 'yes', 'confirm', 'ok'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><circle cx="60" cy="60" r="45" fill="none" stroke="#00FF88" stroke-width="3"/><path d="M35 60 L52 77 L85 44" stroke="#00FF88" stroke-width="6" fill="none" stroke-linecap="round" stroke-linejoin="round"/><text x="60" y="105" text-anchor="middle" fill="#00FF88" font-size="10" font-family="monospace">APPROVED</text></svg>`
            },
            {
                id: 'denied',
                name: 'Denied',
                tags: ['denied', 'no', 'reject', 'error'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><circle cx="60" cy="60" r="45" fill="none" stroke="#FF4444" stroke-width="3"/><line x1="35" y1="35" x2="85" y2="85" stroke="#FF4444" stroke-width="6" stroke-linecap="round"/><line x1="85" y1="35" x2="35" y2="85" stroke="#FF4444" stroke-width="6" stroke-linecap="round"/><text x="60" y="105" text-anchor="middle" fill="#FF4444" font-size="10" font-family="monospace">DENIED</text></svg>`
            },
            {
                id: 'loading_ai',
                name: 'Processing',
                tags: ['loading', 'processing', 'wait', 'compute'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><circle cx="60" cy="60" r="35" fill="none" stroke="#00D4FF" stroke-width="4" stroke-dasharray="20 10" opacity="0.5"/><circle cx="60" cy="60" r="25" fill="none" stroke="#00D4FF" stroke-width="3" stroke-dasharray="15 8"/><circle cx="60" cy="60" r="15" fill="none" stroke="#00D4FF" stroke-width="2" stroke-dasharray="10 5" opacity="0.7"/><circle cx="60" cy="60" r="5" fill="#00D4FF"/><text x="60" y="105" text-anchor="middle" fill="#00D4FF" font-size="8" font-family="monospace">PROCESSING...</text></svg>`
            },
            {
                id: 'satellite',
                name: 'Satellite',
                tags: ['satellite', 'signal', 'communication', 'space'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><ellipse cx="60" cy="55" rx="25" ry="8" fill="none" stroke="#00D4FF" stroke-width="1" transform="rotate(-30 60 55)"/><ellipse cx="60" cy="55" rx="25" ry="8" fill="none" stroke="#00D4FF" stroke-width="1" transform="rotate(30 60 55)"/><ellipse cx="60" cy="55" rx="25" ry="8" fill="none" stroke="#00D4FF" stroke-width="1" transform="rotate(90 60 55)"/><circle cx="60" cy="55" r="8" fill="#00D4FF"/><rect x="40" y="70" width="40" height="6" rx="2" fill="#0066FF"/><rect x="55" y="62" width="10" height="8" fill="#0066FF"/><path d="M30 90 Q45 75 60 80 Q75 75 90 90" stroke="#00D4FF" stroke-width="1" fill="none" stroke-dasharray="3 2"/></svg>`
            },
            {
                id: 'voice_wave',
                name: 'Voice Active',
                tags: ['voice', 'sound', 'audio', 'speak', 'wave'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect width="120" height="120" fill="#0a0a0f"/><rect x="20" y="50" width="8" height="20" rx="2" fill="#00D4FF"/><rect x="35" y="35" width="8" height="50" rx="2" fill="#00D4FF"/><rect x="50" y="25" width="8" height="70" rx="2" fill="#00D4FF"/><rect x="65" y="40" width="8" height="40" rx="2" fill="#00D4FF"/><rect x="80" y="30" width="8" height="60" rx="2" fill="#00D4FF"/><rect x="95" y="45" width="8" height="30" rx="2" fill="#00D4FF"/><text x="60" y="110" text-anchor="middle" fill="#00D4FF" font-size="10" font-family="monospace">VOICE ACTIVE</text></svg>`
            },
            {
                id: 'quantum',
                name: 'Quantum',
                tags: ['quantum', 'atom', 'science', 'physics'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="55" fill="#0a0a0f"/><ellipse cx="60" cy="60" rx="40" ry="15" fill="none" stroke="#00D4FF" stroke-width="1.5"/><ellipse cx="60" cy="60" rx="40" ry="15" fill="none" stroke="#00D4FF" stroke-width="1.5" transform="rotate(60 60 60)"/><ellipse cx="60" cy="60" rx="40" ry="15" fill="none" stroke="#00D4FF" stroke-width="1.5" transform="rotate(-60 60 60)"/><circle cx="60" cy="60" r="8" fill="#00D4FF"/><circle cx="60" cy="20" r="4" fill="#FF6B00"/><circle cx="25" cy="80" r="4" fill="#00FF88"/><circle cx="95" cy="80" r="4" fill="#9966FF"/></svg>`
            }
        ]
    },
    classic: {
        id: 'classic',
        name: 'Classic',
        thumbnail: 'thumbs_up',
        stickers: [
            {
                id: 'thumbs_up',
                name: 'Thumbs Up',
                tags: ['like', 'approve', 'yes', 'ok', 'good'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><linearGradient id="skinG" x1="0%" y1="0%" x2="0%" y2="100%"><stop offset="0%" style="stop-color:#FFE0BD"/><stop offset="100%" style="stop-color:#FFCD94"/></linearGradient></defs><ellipse cx="60" cy="60" rx="55" ry="55" fill="#E8F5E9"/><path d="M45 85 L45 50 L55 50 L55 85 Z" fill="url(#skinG)" stroke="#D4A574" stroke-width="2"/><path d="M40 50 C40 35, 80 35, 80 50 L80 60 C80 65, 75 70, 70 70 L55 70 L55 50" fill="url(#skinG)" stroke="#D4A574" stroke-width="2"/><ellipse cx="67" cy="55" rx="8" ry="5" fill="url(#skinG)" stroke="#D4A574" stroke-width="1"/></svg>`
            },
            {
                id: 'heart',
                name: 'Heart',
                tags: ['love', 'like', 'heart', 'romance'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><linearGradient id="heartG" x1="0%" y1="0%" x2="0%" y2="100%"><stop offset="0%" style="stop-color:#FF6B6B"/><stop offset="100%" style="stop-color:#EE5A5A"/></linearGradient><filter id="heartShadow"><feDropShadow dx="0" dy="2" stdDeviation="3" flood-color="#000" flood-opacity="0.2"/></filter></defs><path d="M60 100 C20 70, 10 40, 35 25 C50 18, 60 30, 60 40 C60 30, 70 18, 85 25 C110 40, 100 70, 60 100 Z" fill="url(#heartG)" filter="url(#heartShadow)"/><ellipse cx="45" cy="40" rx="10" ry="8" fill="rgba(255,255,255,0.3)"/></svg>`
            },
            {
                id: 'laugh',
                name: 'Laugh',
                tags: ['lol', 'haha', 'funny', 'laugh', 'happy'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="faceG" cx="30%" cy="30%"><stop offset="0%" style="stop-color:#FFE066"/><stop offset="100%" style="stop-color:#FFCC00"/></radialGradient></defs><circle cx="60" cy="60" r="50" fill="url(#faceG)" stroke="#E6B800" stroke-width="2"/><ellipse cx="40" cy="45" rx="8" ry="10" fill="#5C3317"/><ellipse cx="80" cy="45" rx="8" ry="10" fill="#5C3317"/><path d="M35 70 Q60 95 85 70" stroke="#5C3317" stroke-width="4" fill="#FF6B6B" stroke-linecap="round"/><ellipse cx="25" cy="65" rx="8" ry="5" fill="#FFAA80" opacity="0.6"/><ellipse cx="95" cy="65" rx="8" ry="5" fill="#FFAA80" opacity="0.6"/></svg>`
            },
            {
                id: 'cry',
                name: 'Cry',
                tags: ['sad', 'cry', 'tears', 'upset'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="sadFaceG" cx="30%" cy="30%"><stop offset="0%" style="stop-color:#FFE066"/><stop offset="100%" style="stop-color:#FFCC00"/></radialGradient></defs><circle cx="60" cy="60" r="50" fill="url(#sadFaceG)" stroke="#E6B800" stroke-width="2"/><ellipse cx="40" cy="45" rx="6" ry="8" fill="#5C3317"/><ellipse cx="80" cy="45" rx="6" ry="8" fill="#5C3317"/><path d="M35 85 Q60 70 85 85" stroke="#5C3317" stroke-width="4" fill="none" stroke-linecap="round"/><ellipse cx="30" cy="60" rx="8" ry="12" fill="#6DB3F2" opacity="0.7"/><ellipse cx="90" cy="60" rx="8" ry="12" fill="#6DB3F2" opacity="0.7"/><path d="M30 55 L30 80" stroke="#6DB3F2" stroke-width="3" opacity="0.5"/><path d="M90 55 L90 80" stroke="#6DB3F2" stroke-width="3" opacity="0.5"/></svg>`
            },
            {
                id: 'wow',
                name: 'Wow',
                tags: ['surprised', 'wow', 'omg', 'shock'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="wowFaceG" cx="30%" cy="30%"><stop offset="0%" style="stop-color:#FFE066"/><stop offset="100%" style="stop-color:#FFCC00"/></radialGradient></defs><circle cx="60" cy="60" r="50" fill="url(#wowFaceG)" stroke="#E6B800" stroke-width="2"/><ellipse cx="40" cy="45" rx="10" ry="12" fill="#FFFFFF" stroke="#5C3317" stroke-width="2"/><ellipse cx="80" cy="45" rx="10" ry="12" fill="#FFFFFF" stroke="#5C3317" stroke-width="2"/><circle cx="40" cy="45" r="5" fill="#5C3317"/><circle cx="80" cy="45" r="5" fill="#5C3317"/><ellipse cx="60" cy="80" rx="12" ry="15" fill="#5C3317"/></svg>`
            },
            {
                id: 'angry',
                name: 'Angry',
                tags: ['angry', 'mad', 'upset', 'rage'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="angryFaceG" cx="30%" cy="30%"><stop offset="0%" style="stop-color:#FF8080"/><stop offset="100%" style="stop-color:#FF4444"/></radialGradient></defs><circle cx="60" cy="60" r="50" fill="url(#angryFaceG)" stroke="#CC0000" stroke-width="2"/><path d="M25 35 L50 45" stroke="#5C3317" stroke-width="4" stroke-linecap="round"/><path d="M95 35 L70 45" stroke="#5C3317" stroke-width="4" stroke-linecap="round"/><ellipse cx="40" cy="50" rx="6" ry="8" fill="#5C3317"/><ellipse cx="80" cy="50" rx="6" ry="8" fill="#5C3317"/><path d="M40 85 Q60 75 80 85" stroke="#5C3317" stroke-width="4" fill="none" stroke-linecap="round"/></svg>`
            },
            {
                id: 'cool',
                name: 'Cool',
                tags: ['cool', 'sunglasses', 'awesome'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="coolFaceG" cx="30%" cy="30%"><stop offset="0%" style="stop-color:#FFE066"/><stop offset="100%" style="stop-color:#FFCC00"/></radialGradient></defs><circle cx="60" cy="60" r="50" fill="url(#coolFaceG)" stroke="#E6B800" stroke-width="2"/><rect x="20" y="38" width="35" height="20" rx="3" fill="#333"/><rect x="65" y="38" width="35" height="20" rx="3" fill="#333"/><path d="M55 48 L65 48" stroke="#333" stroke-width="3"/><path d="M20 42 L10 35" stroke="#333" stroke-width="3"/><path d="M100 42 L110 35" stroke="#333" stroke-width="3"/><path d="M35 75 Q60 90 85 75" stroke="#5C3317" stroke-width="4" fill="none" stroke-linecap="round"/></svg>`
            },
            {
                id: 'wink',
                name: 'Wink',
                tags: ['wink', 'flirt', 'playful'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="winkFaceG" cx="30%" cy="30%"><stop offset="0%" style="stop-color:#FFE066"/><stop offset="100%" style="stop-color:#FFCC00"/></radialGradient></defs><circle cx="60" cy="60" r="50" fill="url(#winkFaceG)" stroke="#E6B800" stroke-width="2"/><ellipse cx="40" cy="45" rx="6" ry="8" fill="#5C3317"/><path d="M70 45 Q80 40 90 45" stroke="#5C3317" stroke-width="4" fill="none" stroke-linecap="round"/><path d="M35 75 Q60 90 85 75" stroke="#5C3317" stroke-width="4" fill="none" stroke-linecap="round"/><ellipse cx="25" cy="55" rx="8" ry="5" fill="#FFAA80" opacity="0.6"/></svg>`
            },
            {
                id: 'thinking',
                name: 'Thinking',
                tags: ['think', 'hmm', 'wondering', 'curious'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="thinkFaceG" cx="30%" cy="30%"><stop offset="0%" style="stop-color:#FFE066"/><stop offset="100%" style="stop-color:#FFCC00"/></radialGradient></defs><circle cx="60" cy="60" r="50" fill="url(#thinkFaceG)" stroke="#E6B800" stroke-width="2"/><ellipse cx="40" cy="45" rx="5" ry="6" fill="#5C3317"/><ellipse cx="80" cy="45" rx="5" ry="6" fill="#5C3317"/><path d="M30 35 L50 40" stroke="#5C3317" stroke-width="3" stroke-linecap="round"/><path d="M70 40 L90 35" stroke="#5C3317" stroke-width="3" stroke-linecap="round"/><path d="M50 78 Q65 78 70 78" stroke="#5C3317" stroke-width="4" fill="none" stroke-linecap="round"/><ellipse cx="90" cy="80" rx="12" ry="10" fill="url(#thinkFaceG)" stroke="#E6B800" stroke-width="2"/></svg>`
            },
            {
                id: 'clap',
                name: 'Clapping',
                tags: ['clap', 'applause', 'bravo', 'congrats'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><linearGradient id="clapSkinG" x1="0%" y1="0%" x2="0%" y2="100%"><stop offset="0%" style="stop-color:#FFE0BD"/><stop offset="100%" style="stop-color:#FFCD94"/></linearGradient></defs><ellipse cx="60" cy="60" rx="55" ry="55" fill="#FFF9E6"/><path d="M35 80 L35 40 C35 30 45 25 55 35 L55 50" fill="url(#clapSkinG)" stroke="#D4A574" stroke-width="2"/><path d="M85 80 L85 40 C85 30 75 25 65 35 L65 50" fill="url(#clapSkinG)" stroke="#D4A574" stroke-width="2"/><ellipse cx="45" cy="35" rx="8" ry="4" fill="url(#clapSkinG)" stroke="#D4A574" stroke-width="1"/><ellipse cx="75" cy="35" rx="8" ry="4" fill="url(#clapSkinG)" stroke="#D4A574" stroke-width="1"/><circle cx="60" cy="25" r="6" fill="#FFD700" opacity="0.8"/><circle cx="45" cy="20" r="4" fill="#FFD700" opacity="0.6"/><circle cx="75" cy="20" r="4" fill="#FFD700" opacity="0.6"/></svg>`
            },
            {
                id: 'fire',
                name: 'Fire',
                tags: ['fire', 'hot', 'lit', 'awesome'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><linearGradient id="fireG" x1="0%" y1="100%" x2="0%" y2="0%"><stop offset="0%" style="stop-color:#FF4500"/><stop offset="50%" style="stop-color:#FF8C00"/><stop offset="100%" style="stop-color:#FFD700"/></linearGradient></defs><ellipse cx="60" cy="60" rx="55" ry="55" fill="#FFF5E6"/><path d="M60 100 C30 80 25 50 45 35 C50 50 55 45 55 35 C55 25 65 15 75 25 C85 35 80 45 75 50 C85 45 95 55 90 75 C85 95 65 105 60 100 Z" fill="url(#fireG)"/><path d="M60 95 C45 80 42 60 55 50 C55 60 60 55 60 50 C65 55 70 50 68 60 C75 55 80 65 75 80 C70 95 62 98 60 95 Z" fill="#FFD700" opacity="0.8"/></svg>`
            },
            {
                id: 'party',
                name: 'Party',
                tags: ['party', 'celebrate', 'yay', 'confetti'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><defs><radialGradient id="partyFaceG" cx="30%" cy="30%"><stop offset="0%" style="stop-color:#FFE066"/><stop offset="100%" style="stop-color:#FFCC00"/></radialGradient></defs><circle cx="60" cy="60" r="50" fill="url(#partyFaceG)" stroke="#E6B800" stroke-width="2"/><ellipse cx="40" cy="45" rx="6" ry="8" fill="#5C3317"/><ellipse cx="80" cy="45" rx="6" ry="8" fill="#5C3317"/><path d="M35 70 Q60 95 85 70" stroke="#5C3317" stroke-width="4" fill="none" stroke-linecap="round"/><polygon points="20,10 25,35 5,25" fill="#FF6B6B"/><polygon points="100,10 95,35 115,25" fill="#4ECDC4"/><polygon points="60,5 55,25 65,25" fill="#FFE66D"/><circle cx="15" cy="50" r="5" fill="#9B59B6"/><circle cx="105" cy="50" r="5" fill="#3498DB"/><rect x="25" y="20" width="4" height="12" fill="#2ECC71" transform="rotate(15 27 26)"/><rect x="90" y="20" width="4" height="12" fill="#E74C3C" transform="rotate(-15 92 26)"/></svg>`
            }
        ]
    },
    animals: {
        id: 'animals',
        name: 'Animals',
        thumbnail: 'cat',
        stickers: [
            {
                id: 'cat',
                name: 'Happy Cat',
                tags: ['cat', 'cute', 'happy', 'kitty'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><ellipse cx="60" cy="60" rx="55" ry="55" fill="#FFF5E6"/><ellipse cx="60" cy="65" rx="35" ry="30" fill="#FFB347" stroke="#E59400" stroke-width="2"/><polygon points="35,35 25,15 45,30" fill="#FFB347" stroke="#E59400" stroke-width="2"/><polygon points="85,35 95,15 75,30" fill="#FFB347" stroke="#E59400" stroke-width="2"/><polygon points="35,35 30,20 40,32" fill="#FFB0B0"/><polygon points="85,35 90,20 80,32" fill="#FFB0B0"/><ellipse cx="45" cy="55" rx="8" ry="10" fill="#FFFFFF" stroke="#333" stroke-width="1"/><ellipse cx="75" cy="55" rx="8" ry="10" fill="#FFFFFF" stroke="#333" stroke-width="1"/><circle cx="45" cy="57" r="4" fill="#333"/><circle cx="75" cy="57" r="4" fill="#333"/><ellipse cx="60" cy="72" rx="5" ry="3" fill="#FFB0B0"/><path d="M55 78 Q60 83 65 78" stroke="#333" stroke-width="2" fill="none"/><path d="M35 65 L20 60" stroke="#333" stroke-width="1"/><path d="M35 70 L20 70" stroke="#333" stroke-width="1"/><path d="M35 75 L20 80" stroke="#333" stroke-width="1"/><path d="M85 65 L100 60" stroke="#333" stroke-width="1"/><path d="M85 70 L100 70" stroke="#333" stroke-width="1"/><path d="M85 75 L100 80" stroke="#333" stroke-width="1"/></svg>`
            },
            {
                id: 'dog',
                name: 'Happy Dog',
                tags: ['dog', 'cute', 'puppy', 'happy'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><ellipse cx="60" cy="60" rx="55" ry="55" fill="#E6F3FF"/><ellipse cx="60" cy="65" rx="35" ry="30" fill="#C4A77D" stroke="#8B7355" stroke-width="2"/><ellipse cx="30" cy="40" rx="15" ry="20" fill="#C4A77D" stroke="#8B7355" stroke-width="2"/><ellipse cx="90" cy="40" rx="15" ry="20" fill="#C4A77D" stroke="#8B7355" stroke-width="2"/><ellipse cx="45" cy="55" rx="8" ry="10" fill="#FFFFFF" stroke="#333" stroke-width="1"/><ellipse cx="75" cy="55" rx="8" ry="10" fill="#FFFFFF" stroke="#333" stroke-width="1"/><circle cx="45" cy="57" r="4" fill="#333"/><circle cx="75" cy="57" r="4" fill="#333"/><ellipse cx="60" cy="75" rx="10" ry="8" fill="#333"/><ellipse cx="60" cy="73" rx="4" ry="3" fill="#444" opacity="0.5"/><path d="M50 85 L60 95 L70 85" stroke="#FF6B6B" stroke-width="4" fill="#FF6B6B"/></svg>`
            },
            {
                id: 'bear',
                name: 'Cute Bear',
                tags: ['bear', 'cute', 'teddy'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><ellipse cx="60" cy="60" rx="55" ry="55" fill="#FFF0E6"/><circle cx="30" cy="30" r="15" fill="#8B4513" stroke="#5D2E0C" stroke-width="2"/><circle cx="90" cy="30" r="15" fill="#8B4513" stroke="#5D2E0C" stroke-width="2"/><circle cx="30" cy="30" r="8" fill="#C4A77D"/><circle cx="90" cy="30" r="8" fill="#C4A77D"/><ellipse cx="60" cy="60" rx="40" ry="35" fill="#8B4513" stroke="#5D2E0C" stroke-width="2"/><ellipse cx="45" cy="50" rx="6" ry="8" fill="#333"/><ellipse cx="75" cy="50" rx="6" ry="8" fill="#333"/><ellipse cx="60" cy="70" rx="15" ry="12" fill="#C4A77D"/><ellipse cx="60" cy="68" rx="8" ry="5" fill="#333"/><path d="M52 78 Q60 85 68 78" stroke="#333" stroke-width="2" fill="none"/></svg>`
            },
            {
                id: 'panda',
                name: 'Panda',
                tags: ['panda', 'cute', 'bear'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><ellipse cx="60" cy="60" rx="55" ry="55" fill="#E8F5E9"/><circle cx="30" cy="30" r="15" fill="#333"/><circle cx="90" cy="30" r="15" fill="#333"/><ellipse cx="60" cy="60" rx="40" ry="35" fill="#FFFFFF" stroke="#DDD" stroke-width="2"/><ellipse cx="40" cy="50" rx="12" ry="14" fill="#333"/><ellipse cx="80" cy="50" rx="12" ry="14" fill="#333"/><ellipse cx="42" cy="48" rx="5" ry="6" fill="#FFFFFF"/><ellipse cx="78" cy="48" rx="5" ry="6" fill="#FFFFFF"/><circle cx="42" cy="50" r="3" fill="#333"/><circle cx="78" cy="50" r="3" fill="#333"/><ellipse cx="60" cy="70" rx="10" ry="6" fill="#333"/><path d="M52 78 Q60 85 68 78" stroke="#333" stroke-width="2" fill="none"/></svg>`
            },
            {
                id: 'bunny',
                name: 'Bunny',
                tags: ['bunny', 'rabbit', 'cute', 'easter'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><ellipse cx="60" cy="60" rx="55" ry="55" fill="#FFF0F5"/><ellipse cx="40" cy="20" rx="10" ry="25" fill="#FFFFFF" stroke="#FFB0B0" stroke-width="2"/><ellipse cx="80" cy="20" rx="10" ry="25" fill="#FFFFFF" stroke="#FFB0B0" stroke-width="2"/><ellipse cx="40" cy="18" rx="5" ry="15" fill="#FFB0B0" opacity="0.5"/><ellipse cx="80" cy="18" rx="5" ry="15" fill="#FFB0B0" opacity="0.5"/><ellipse cx="60" cy="65" rx="35" ry="30" fill="#FFFFFF" stroke="#DDD" stroke-width="2"/><ellipse cx="45" cy="55" rx="5" ry="7" fill="#333"/><ellipse cx="75" cy="55" rx="5" ry="7" fill="#333"/><ellipse cx="60" cy="70" rx="6" ry="4" fill="#FFB0B0"/><path d="M54 75 L60 80 L66 75" stroke="#333" stroke-width="2" fill="none"/><ellipse cx="35" cy="70" rx="8" ry="5" fill="#FFB0B0" opacity="0.5"/><ellipse cx="85" cy="70" rx="8" ry="5" fill="#FFB0B0" opacity="0.5"/></svg>`
            },
            {
                id: 'fox',
                name: 'Fox',
                tags: ['fox', 'cute', 'clever'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><ellipse cx="60" cy="60" rx="55" ry="55" fill="#FFF5E6"/><polygon points="30,45 15,10 50,40" fill="#FF6B00" stroke="#CC5500" stroke-width="2"/><polygon points="90,45 105,10 70,40" fill="#FF6B00" stroke="#CC5500" stroke-width="2"/><polygon points="30,45 20,20 40,38" fill="#FFFFFF"/><polygon points="90,45 100,20 80,38" fill="#FFFFFF"/><ellipse cx="60" cy="60" rx="40" ry="35" fill="#FF6B00" stroke="#CC5500" stroke-width="2"/><ellipse cx="45" cy="50" rx="6" ry="8" fill="#333"/><ellipse cx="75" cy="50" rx="6" ry="8" fill="#333"/><ellipse cx="60" cy="75" rx="18" ry="12" fill="#FFFFFF"/><ellipse cx="60" cy="70" rx="5" ry="4" fill="#333"/><path d="M50 80 Q60 88 70 80" stroke="#333" stroke-width="2" fill="none"/></svg>`
            }
        ]
    },
    retro: {
        id: 'retro',
        name: 'Retro',
        thumbnail: 'computer',
        stickers: [
            {
                id: 'computer',
                name: 'Computer',
                tags: ['computer', 'pc', 'retro', 'tech'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect x="10" y="10" width="100" height="70" rx="5" fill="#C0C0C0" stroke="#808080" stroke-width="3"/><rect x="20" y="18" width="80" height="50" fill="#008080"/><rect x="25" y="23" width="70" height="40" fill="#000080"/><text x="30" y="48" fill="#00FF00" font-family="monospace" font-size="10">C:\\&gt;_</text><rect x="40" y="85" width="40" height="8" fill="#C0C0C0" stroke="#808080" stroke-width="2"/><rect x="30" y="93" width="60" height="15" fill="#C0C0C0" stroke="#808080" stroke-width="2"/></svg>`
            },
            {
                id: 'floppy',
                name: 'Floppy Disk',
                tags: ['floppy', 'disk', 'save', 'retro'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect x="15" y="15" width="90" height="90" rx="3" fill="#333" stroke="#222" stroke-width="2"/><rect x="35" y="15" width="50" height="30" fill="#C0C0C0" stroke="#808080" stroke-width="1"/><rect x="55" y="20" width="25" height="20" fill="#444"/><rect x="25" y="60" width="70" height="40" rx="2" fill="#F5F5F5" stroke="#DDD" stroke-width="1"/><rect x="30" y="65" width="60" height="5" fill="#DDD"/><rect x="30" y="75" width="40" height="3" fill="#DDD"/><rect x="30" y="82" width="50" height="3" fill="#DDD"/><circle cx="85" cy="25" r="5" fill="#FF0000" opacity="0.8"/></svg>`
            },
            {
                id: 'gameboy',
                name: 'Game Boy',
                tags: ['gameboy', 'game', 'retro', 'nintendo'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect x="25" y="5" width="70" height="110" rx="8" fill="#C0C0C0" stroke="#808080" stroke-width="2"/><rect x="35" y="15" width="50" height="45" rx="3" fill="#8B956D" stroke="#666" stroke-width="2"/><rect x="40" y="20" width="40" height="35" fill="#9DB87A"/><circle cx="45" cy="85" r="8" fill="#333" stroke="#222" stroke-width="2"/><circle cx="65" cy="85" r="5" fill="#990066" stroke="#660044" stroke-width="2"/><circle cx="80" cy="85" r="5" fill="#990066" stroke="#660044" stroke-width="2"/><rect x="50" y="75" width="10" height="3" rx="1" fill="#666"/><rect x="50" y="80" width="10" height="3" rx="1" fill="#666"/><text x="60" y="105" text-anchor="middle" fill="#666" font-size="6">GAME</text></svg>`
            },
            {
                id: 'cassette',
                name: 'Cassette',
                tags: ['cassette', 'tape', 'music', 'retro'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect x="10" y="25" width="100" height="70" rx="5" fill="#333" stroke="#222" stroke-width="2"/><rect x="15" y="30" width="90" height="50" rx="3" fill="#F5F5F5" stroke="#DDD" stroke-width="1"/><circle cx="40" cy="55" r="15" fill="#333" stroke="#222" stroke-width="2"/><circle cx="80" cy="55" r="15" fill="#333" stroke="#222" stroke-width="2"/><circle cx="40" cy="55" r="8" fill="#C0C0C0"/><circle cx="80" cy="55" r="8" fill="#C0C0C0"/><rect x="50" y="45" width="20" height="20" fill="#8B4513" opacity="0.6"/><rect x="25" y="85" width="15" height="8" rx="2" fill="#444"/><rect x="80" y="85" width="15" height="8" rx="2" fill="#444"/><text x="60" y="42" text-anchor="middle" fill="#333" font-size="7" font-family="Arial">SIDE A</text></svg>`
            },
            {
                id: 'pixel_heart',
                name: 'Pixel Heart',
                tags: ['pixel', 'heart', 'love', 'retro', '8bit'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect x="20" y="20" width="10" height="10" fill="#FF0000"/><rect x="30" y="20" width="10" height="10" fill="#FF0000"/><rect x="50" y="20" width="10" height="10" fill="#FF0000"/><rect x="60" y="20" width="10" height="10" fill="#FF0000"/><rect x="80" y="20" width="10" height="10" fill="#FF0000"/><rect x="90" y="20" width="10" height="10" fill="#FF0000"/><rect x="10" y="30" width="10" height="10" fill="#FF0000"/><rect x="20" y="30" width="10" height="10" fill="#FF6666"/><rect x="30" y="30" width="10" height="10" fill="#FF0000"/><rect x="40" y="30" width="10" height="10" fill="#FF0000"/><rect x="50" y="30" width="10" height="10" fill="#FF0000"/><rect x="60" y="30" width="10" height="10" fill="#FF0000"/><rect x="70" y="30" width="10" height="10" fill="#FF0000"/><rect x="80" y="30" width="10" height="10" fill="#FF0000"/><rect x="90" y="30" width="10" height="10" fill="#FF0000"/><rect x="100" y="30" width="10" height="10" fill="#FF0000"/><rect x="10" y="40" width="100" height="10" fill="#FF0000"/><rect x="10" y="40" width="10" height="10" fill="#FF6666"/><rect x="20" y="50" width="80" height="10" fill="#FF0000"/><rect x="30" y="60" width="60" height="10" fill="#FF0000"/><rect x="40" y="70" width="40" height="10" fill="#FF0000"/><rect x="50" y="80" width="20" height="10" fill="#FF0000"/><rect x="55" y="90" width="10" height="10" fill="#FF0000"/></svg>`
            },
            {
                id: 'loading',
                name: 'Loading',
                tags: ['loading', 'wait', 'hourglass', 'retro'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><rect x="35" y="15" width="50" height="10" fill="#C0C0C0" stroke="#808080" stroke-width="2"/><rect x="35" y="95" width="50" height="10" fill="#C0C0C0" stroke="#808080" stroke-width="2"/><path d="M40 25 L40 50 L60 70 L80 50 L80 25 Z" fill="#F5DEB3" stroke="#C0C0C0" stroke-width="2"/><path d="M40 95 L40 70 L60 50 L80 70 L80 95 Z" fill="#87CEEB" stroke="#C0C0C0" stroke-width="2"/><path d="M45 30 L55 50 L45 50 Z" fill="#DEB887"/><path d="M50 90 L60 60 L70 90 Z" fill="#87CEEB" opacity="0.5"/><circle cx="60" cy="58" r="3" fill="#DEB887"/><circle cx="60" cy="62" r="2" fill="#DEB887"/><circle cx="60" cy="65" r="2" fill="#DEB887"/></svg>`
            }
        ]
    },
    reactions: {
        id: 'reactions',
        name: 'Reactions',
        thumbnail: 'ok',
        stickers: [
            {
                id: 'ok',
                name: 'OK',
                tags: ['ok', 'okay', 'good', 'approve'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="50" fill="#4CAF50" stroke="#388E3C" stroke-width="3"/><text x="60" y="75" text-anchor="middle" fill="#FFFFFF" font-size="36" font-weight="bold" font-family="Arial">OK</text></svg>`
            },
            {
                id: 'lol',
                name: 'LOL',
                tags: ['lol', 'laugh', 'funny', 'haha'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="50" fill="#FF9800" stroke="#F57C00" stroke-width="3"/><text x="60" y="75" text-anchor="middle" fill="#FFFFFF" font-size="32" font-weight="bold" font-family="Arial">LOL</text></svg>`
            },
            {
                id: 'wtf',
                name: 'WTF',
                tags: ['wtf', 'confused', 'what'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="50" fill="#9C27B0" stroke="#7B1FA2" stroke-width="3"/><text x="60" y="75" text-anchor="middle" fill="#FFFFFF" font-size="28" font-weight="bold" font-family="Arial">WTF</text></svg>`
            },
            {
                id: 'omg',
                name: 'OMG',
                tags: ['omg', 'surprised', 'shock', 'wow'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="50" fill="#E91E63" stroke="#C2185B" stroke-width="3"/><text x="60" y="75" text-anchor="middle" fill="#FFFFFF" font-size="28" font-weight="bold" font-family="Arial">OMG</text></svg>`
            },
            {
                id: 'gg',
                name: 'GG',
                tags: ['gg', 'good game', 'game', 'win'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="50" fill="#2196F3" stroke="#1976D2" stroke-width="3"/><text x="60" y="75" text-anchor="middle" fill="#FFFFFF" font-size="36" font-weight="bold" font-family="Arial">GG</text></svg>`
            },
            {
                id: 'brb',
                name: 'BRB',
                tags: ['brb', 'be right back', 'away'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="50" fill="#607D8B" stroke="#455A64" stroke-width="3"/><text x="60" y="75" text-anchor="middle" fill="#FFFFFF" font-size="28" font-weight="bold" font-family="Arial">BRB</text></svg>`
            },
            {
                id: 'thx',
                name: 'Thanks',
                tags: ['thanks', 'thank you', 'thx', 'ty'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="50" fill="#00BCD4" stroke="#0097A7" stroke-width="3"/><text x="60" y="75" text-anchor="middle" fill="#FFFFFF" font-size="26" font-weight="bold" font-family="Arial">THX</text></svg>`
            },
            {
                id: 'np',
                name: 'No Problem',
                tags: ['np', 'no problem', 'welcome'],
                svg: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 120 120"><circle cx="60" cy="60" r="50" fill="#8BC34A" stroke="#689F38" stroke-width="3"/><text x="60" y="75" text-anchor="middle" fill="#FFFFFF" font-size="36" font-weight="bold" font-family="Arial">NP</text></svg>`
            }
        ]
    }
};

// Recent stickers management
const RECENT_STICKERS_KEY = 'mizu_recent_stickers';
const MAX_RECENT_STICKERS = 12;

function getRecentStickers() {
    try {
        const stored = localStorage.getItem(RECENT_STICKERS_KEY);
        return stored ? JSON.parse(stored) : [];
    } catch {
        return [];
    }
}

function addRecentSticker(packId, stickerId) {
    let recent = getRecentStickers();
    const key = `${packId}:${stickerId}`;
    // Remove if already exists
    recent = recent.filter(s => s !== key);
    // Add to front
    recent.unshift(key);
    // Limit size
    recent = recent.slice(0, MAX_RECENT_STICKERS);
    localStorage.setItem(RECENT_STICKERS_KEY, JSON.stringify(recent));
    return recent;
}

function getStickerByKey(key) {
    const [packId, stickerId] = key.split(':');
    const pack = STICKER_PACKS[packId];
    if (!pack) return null;
    return pack.stickers.find(s => s.id === stickerId);
}

// Export for use
window.STICKER_PACKS = STICKER_PACKS;
window.getRecentStickers = getRecentStickers;
window.addRecentSticker = addRecentSticker;
window.getStickerByKey = getStickerByKey;
