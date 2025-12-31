/**
 * 8-Bit Retro Theme - JavaScript Functionality
 * NES-inspired interactions and effects
 * Version: 1.0.0
 */

(function() {
    'use strict';

    // ==========================================================================
    // Theme Toggle (Day/Night Mode)
    // ==========================================================================

    const ThemeToggle = {
        init() {
            this.toggle = document.getElementById('theme-toggle');
            this.html = document.documentElement;

            if (!this.toggle) return;

            // Load saved theme
            const savedTheme = localStorage.getItem('theme') || 'light';
            this.setTheme(savedTheme);

            // Toggle event
            this.toggle.addEventListener('click', () => {
                const currentTheme = this.html.getAttribute('data-theme');
                const newTheme = currentTheme === 'dark' ? 'light' : 'dark';
                this.setTheme(newTheme);
                this.playSound('toggle');
            });
        },

        setTheme(theme) {
            this.html.setAttribute('data-theme', theme);
            localStorage.setItem('theme', theme);
        },

        playSound(type) {
            // Optional: Play 8-bit sound effects
            // This is a placeholder for future sound implementation
        }
    };

    // ==========================================================================
    // Mobile Navigation
    // ==========================================================================

    const MobileNav = {
        init() {
            this.toggle = document.getElementById('mobile-menu-toggle');
            this.nav = document.getElementById('mobile-nav');

            if (!this.toggle || !this.nav) return;

            this.toggle.addEventListener('click', () => {
                const isOpen = this.nav.classList.toggle('is-open');
                this.toggle.setAttribute('aria-expanded', isOpen);
            });

            // Close on escape
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Escape' && this.nav.classList.contains('is-open')) {
                    this.nav.classList.remove('is-open');
                    this.toggle.setAttribute('aria-expanded', 'false');
                    this.toggle.focus();
                }
            });

            // Close on link click
            this.nav.querySelectorAll('a').forEach(link => {
                link.addEventListener('click', () => {
                    this.nav.classList.remove('is-open');
                    this.toggle.setAttribute('aria-expanded', 'false');
                });
            });
        }
    };

    // ==========================================================================
    // Search Overlay
    // ==========================================================================

    const SearchOverlay = {
        init() {
            this.toggle = document.getElementById('search-toggle');
            this.overlay = document.getElementById('search-overlay');
            this.close = document.getElementById('search-close');
            this.input = document.getElementById('search-input');

            if (!this.overlay) return;

            // Open search
            if (this.toggle) {
                this.toggle.addEventListener('click', () => this.open());
            }

            // Close search
            if (this.close) {
                this.close.addEventListener('click', () => this.closeOverlay());
            }

            // Close on escape
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Escape' && this.overlay.classList.contains('is-open')) {
                    this.closeOverlay();
                }
            });

            // Close on backdrop click
            this.overlay.addEventListener('click', (e) => {
                if (e.target === this.overlay) {
                    this.closeOverlay();
                }
            });

            // Keyboard shortcut: Cmd/Ctrl + K
            document.addEventListener('keydown', (e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                    e.preventDefault();
                    if (this.overlay.classList.contains('is-open')) {
                        this.closeOverlay();
                    } else {
                        this.open();
                    }
                }
            });
        },

        open() {
            this.overlay.classList.add('is-open');
            if (this.input) {
                setTimeout(() => this.input.focus(), 100);
            }
        },

        closeOverlay() {
            this.overlay.classList.remove('is-open');
            if (this.toggle) {
                this.toggle.focus();
            }
        }
    };

    // ==========================================================================
    // Reading Progress Bar
    // ==========================================================================

    const ReadingProgress = {
        init() {
            this.progress = document.getElementById('reading-progress');
            this.bar = document.getElementById('reading-progress-bar');

            if (!this.progress || !this.bar) return;

            // Calculate on scroll
            window.addEventListener('scroll', () => this.update(), { passive: true });
            this.update();
        },

        update() {
            const windowHeight = window.innerHeight;
            const documentHeight = document.documentElement.scrollHeight - windowHeight;
            const scrollTop = window.scrollY;
            const progress = Math.min((scrollTop / documentHeight) * 100, 100);

            this.bar.style.width = progress + '%';

            // Update state classes for color changes
            this.progress.classList.remove('reading-progress--warning', 'reading-progress--danger');

            if (progress > 90) {
                this.progress.classList.add('reading-progress--danger');
            } else if (progress > 75) {
                this.progress.classList.add('reading-progress--warning');
            }
        }
    };

    // ==========================================================================
    // Back to Top
    // ==========================================================================

    const BackToTop = {
        init() {
            this.button = document.getElementById('back-to-top');

            if (!this.button) return;

            // Show/hide based on scroll
            window.addEventListener('scroll', () => {
                if (window.scrollY > 300) {
                    this.button.classList.add('is-visible');
                } else {
                    this.button.classList.remove('is-visible');
                }
            }, { passive: true });

            // Scroll to top on click
            this.button.addEventListener('click', () => {
                // Use stepped animation for 8-bit feel
                this.scrollToTop();
            });
        },

        scrollToTop() {
            const scrollStep = window.scrollY / 20;
            const scroll = () => {
                if (window.scrollY > 0) {
                    window.scrollBy(0, -scrollStep);
                    requestAnimationFrame(scroll);
                }
            };
            requestAnimationFrame(scroll);
        }
    };

    // ==========================================================================
    // Sticky Header
    // ==========================================================================

    const StickyHeader = {
        init() {
            this.header = document.getElementById('site-header');

            if (!this.header) return;

            let lastScrollY = 0;

            window.addEventListener('scroll', () => {
                const scrollY = window.scrollY;

                // Add scrolled class for styling
                if (scrollY > 10) {
                    this.header.classList.add('is-scrolled');
                } else {
                    this.header.classList.remove('is-scrolled');
                }

                lastScrollY = scrollY;
            }, { passive: true });
        }
    };

    // ==========================================================================
    // Copy Link Button
    // ==========================================================================

    const CopyLink = {
        init() {
            this.button = document.getElementById('copy-link-btn');

            if (!this.button) return;

            this.button.addEventListener('click', async () => {
                const url = this.button.getAttribute('data-url') || window.location.href;

                try {
                    await navigator.clipboard.writeText(url);
                    this.showFeedback('COPIED!');
                } catch (err) {
                    // Fallback for older browsers
                    this.fallbackCopy(url);
                }
            });
        },

        fallbackCopy(text) {
            const textarea = document.createElement('textarea');
            textarea.value = text;
            textarea.style.position = 'fixed';
            textarea.style.opacity = '0';
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);
            this.showFeedback('COPIED!');
        },

        showFeedback(message) {
            const originalContent = this.button.innerHTML;
            this.button.innerHTML = '<span>' + message + '</span>';
            this.button.style.background = 'var(--8bit-green)';

            setTimeout(() => {
                this.button.innerHTML = originalContent;
                this.button.style.background = '';
            }, 1500);
        }
    };

    // ==========================================================================
    // Card Hover Effects (Pixel-style)
    // ==========================================================================

    const CardEffects = {
        init() {
            const cards = document.querySelectorAll('.post-card');

            cards.forEach(card => {
                card.addEventListener('mouseenter', () => {
                    // Optional: Play hover sound
                });
            });
        }
    };

    // ==========================================================================
    // Scroll Animations (Stepped for 8-bit feel)
    // ==========================================================================

    const ScrollAnimations = {
        init() {
            if ('IntersectionObserver' in window) {
                this.setupObserver();
            } else {
                // Fallback: show all elements
                document.querySelectorAll('.animate-on-scroll').forEach(el => {
                    el.style.opacity = '1';
                });
            }
        },

        setupObserver() {
            const options = {
                root: null,
                rootMargin: '0px',
                threshold: 0.1
            };

            const observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        entry.target.classList.add('is-visible');
                        observer.unobserve(entry.target);
                    }
                });
            }, options);

            document.querySelectorAll('.animate-on-scroll').forEach(el => {
                observer.observe(el);
            });
        }
    };

    // ==========================================================================
    // Comment Reply
    // ==========================================================================

    const CommentReply = {
        init() {
            const replyButtons = document.querySelectorAll('[data-reply-to]');
            const parentInput = document.getElementById('comment-parent-id');
            const form = document.getElementById('comment-form');

            if (!form || !parentInput) return;

            replyButtons.forEach(button => {
                button.addEventListener('click', () => {
                    const parentId = button.getAttribute('data-reply-to');
                    parentInput.value = parentId;

                    // Move form after the comment
                    const comment = document.getElementById('comment-' + parentId);
                    if (comment) {
                        comment.after(form);
                    }

                    // Focus on textarea
                    const textarea = form.querySelector('textarea');
                    if (textarea) textarea.focus();

                    // Update button text
                    button.textContent = 'REPLYING...';
                });
            });
        }
    };

    // ==========================================================================
    // Lazy Load Images
    // ==========================================================================

    const LazyLoad = {
        init() {
            if ('loading' in HTMLImageElement.prototype) {
                // Native lazy loading supported
                document.querySelectorAll('img[loading="lazy"]').forEach(img => {
                    img.addEventListener('load', () => {
                        img.classList.add('is-loaded');
                    });
                });
            } else {
                // Fallback with IntersectionObserver
                this.observeImages();
            }
        },

        observeImages() {
            if (!('IntersectionObserver' in window)) return;

            const observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        const img = entry.target;
                        if (img.dataset.src) {
                            img.src = img.dataset.src;
                        }
                        img.classList.add('is-loaded');
                        observer.unobserve(img);
                    }
                });
            });

            document.querySelectorAll('img[data-src]').forEach(img => {
                observer.observe(img);
            });
        }
    };

    // ==========================================================================
    // External Links
    // ==========================================================================

    const ExternalLinks = {
        init() {
            document.querySelectorAll('a[href^="http"]').forEach(link => {
                if (!link.href.includes(window.location.hostname)) {
                    link.setAttribute('target', '_blank');
                    link.setAttribute('rel', 'noopener noreferrer');
                }
            });
        }
    };

    // ==========================================================================
    // Keyboard Navigation Indicator
    // ==========================================================================

    const FocusVisible = {
        init() {
            document.body.addEventListener('mousedown', () => {
                document.body.classList.add('using-mouse');
            });

            document.body.addEventListener('keydown', (e) => {
                if (e.key === 'Tab') {
                    document.body.classList.remove('using-mouse');
                }
            });
        }
    };

    // ==========================================================================
    // Newsletter Form Feedback
    // ==========================================================================

    const NewsletterForm = {
        init() {
            const forms = document.querySelectorAll('.newsletter__form, form[action="/subscribe"]');

            forms.forEach(form => {
                form.addEventListener('submit', (e) => {
                    // Show loading state
                    const button = form.querySelector('button[type="submit"]');
                    if (button) {
                        const originalText = button.textContent;
                        button.textContent = 'LOADING...';
                        button.disabled = true;

                        // Reset after timeout (actual submission handled by server)
                        setTimeout(() => {
                            button.textContent = originalText;
                            button.disabled = false;
                        }, 2000);
                    }
                });
            });
        }
    };

    // ==========================================================================
    // 8-Bit Sound Effects (Optional)
    // ==========================================================================

    const SoundEffects = {
        enabled: false,
        audioContext: null,

        init() {
            // Check if sound effects are enabled
            this.enabled = document.body.classList.contains('sound-enabled');
            if (!this.enabled) return;

            // Create audio context on first user interaction
            document.addEventListener('click', () => {
                if (!this.audioContext) {
                    this.audioContext = new (window.AudioContext || window.webkitAudioContext)();
                }
            }, { once: true });
        },

        play(type) {
            if (!this.enabled || !this.audioContext) return;

            const oscillator = this.audioContext.createOscillator();
            const gainNode = this.audioContext.createGain();

            oscillator.connect(gainNode);
            gainNode.connect(this.audioContext.destination);

            // Different sounds for different actions
            switch (type) {
                case 'click':
                    oscillator.frequency.value = 440;
                    gainNode.gain.value = 0.1;
                    break;
                case 'success':
                    oscillator.frequency.value = 880;
                    gainNode.gain.value = 0.1;
                    break;
                case 'error':
                    oscillator.frequency.value = 220;
                    gainNode.gain.value = 0.1;
                    break;
            }

            oscillator.type = 'square'; // 8-bit sound
            oscillator.start();
            oscillator.stop(this.audioContext.currentTime + 0.1);
        }
    };

    // ==========================================================================
    // Konami Code Easter Egg
    // ==========================================================================

    const KonamiCode = {
        code: ['ArrowUp', 'ArrowUp', 'ArrowDown', 'ArrowDown', 'ArrowLeft', 'ArrowRight', 'ArrowLeft', 'ArrowRight', 'b', 'a'],
        position: 0,

        init() {
            document.addEventListener('keydown', (e) => {
                if (e.key === this.code[this.position]) {
                    this.position++;

                    if (this.position === this.code.length) {
                        this.activate();
                        this.position = 0;
                    }
                } else {
                    this.position = 0;
                }
            });
        },

        activate() {
            // Easter egg: Add special effects
            document.body.style.animation = 'shake-8bit 0.5s steps(4)';

            const message = document.createElement('div');
            message.innerHTML = '<span style="font-family: var(--font-pixel); font-size: 24px; color: var(--8bit-gold);">+30 LIVES!</span>';
            message.style.cssText = `
                position: fixed;
                top: 50%;
                left: 50%;
                transform: translate(-50%, -50%);
                z-index: 9999;
                animation: bounce-8bit 1s steps(4);
            `;
            document.body.appendChild(message);

            setTimeout(() => {
                message.remove();
                document.body.style.animation = '';
            }, 2000);
        }
    };

    // ==========================================================================
    // Initialize All Modules
    // ==========================================================================

    const init = () => {
        ThemeToggle.init();
        MobileNav.init();
        SearchOverlay.init();
        ReadingProgress.init();
        BackToTop.init();
        StickyHeader.init();
        CopyLink.init();
        CardEffects.init();
        ScrollAnimations.init();
        CommentReply.init();
        LazyLoad.init();
        ExternalLinks.init();
        FocusVisible.init();
        NewsletterForm.init();
        SoundEffects.init();
        KonamiCode.init();
    };

    // Run on DOM ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

})();
