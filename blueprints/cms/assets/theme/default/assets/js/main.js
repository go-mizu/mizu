/**
 * Modern Theme - Main JavaScript
 * Version: 2.0.0
 * Enhanced interactions: glassmorphism effects, smooth animations, micro-interactions
 */

(function() {
    'use strict';

    // ==========================================================================
    // Theme Toggle (Dark Mode)
    // ==========================================================================

    const ThemeToggle = {
        storageKey: 'theme-preference',

        init() {
            const toggle = document.getElementById('theme-toggle');
            if (!toggle) return;

            // Set initial theme based on preference or system
            this.setTheme(this.getPreference());

            // Listen for toggle clicks with smooth icon transition
            toggle.addEventListener('click', () => {
                const current = document.documentElement.getAttribute('data-theme');
                const next = current === 'dark' ? 'light' : 'dark';

                // Add transition class for smooth theme switch
                document.documentElement.classList.add('theme-transitioning');
                this.setTheme(next);

                setTimeout(() => {
                    document.documentElement.classList.remove('theme-transitioning');
                }, 300);
            });

            // Listen for system preference changes
            window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
                if (!localStorage.getItem(this.storageKey)) {
                    this.setTheme(e.matches ? 'dark' : 'light');
                }
            });
        },

        getPreference() {
            const stored = localStorage.getItem(this.storageKey);
            if (stored) return stored;
            return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
        },

        setTheme(theme) {
            document.documentElement.setAttribute('data-theme', theme);
            localStorage.setItem(this.storageKey, theme);
        }
    };

    // ==========================================================================
    // Mobile Navigation
    // ==========================================================================

    const MobileNav = {
        init() {
            const toggle = document.getElementById('mobile-menu-toggle');
            const nav = document.getElementById('primary-nav');

            if (!toggle || !nav) return;

            toggle.addEventListener('click', () => {
                const expanded = toggle.getAttribute('aria-expanded') === 'true';
                toggle.setAttribute('aria-expanded', !expanded);
                nav.classList.toggle('is-open');
                document.body.classList.toggle('nav-open');
            });

            // Close on escape
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Escape' && nav.classList.contains('is-open')) {
                    toggle.setAttribute('aria-expanded', 'false');
                    nav.classList.remove('is-open');
                    document.body.classList.remove('nav-open');
                }
            });
        }
    };

    // ==========================================================================
    // Search Overlay
    // ==========================================================================

    const SearchOverlay = {
        init() {
            const toggle = document.getElementById('search-toggle');
            const close = document.getElementById('search-close');
            const overlay = document.getElementById('search-overlay');
            const input = document.getElementById('search-input');

            if (!toggle || !overlay) return;

            toggle.addEventListener('click', () => this.open());
            if (close) close.addEventListener('click', () => this.close());

            // Close on escape
            document.addEventListener('keydown', (e) => {
                if (e.key === 'Escape' && overlay.getAttribute('aria-hidden') === 'false') {
                    this.close();
                }
            });

            // Close on backdrop click
            overlay.addEventListener('click', (e) => {
                if (e.target === overlay) this.close();
            });

            // Keyboard shortcut: Cmd/Ctrl + K
            document.addEventListener('keydown', (e) => {
                if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                    e.preventDefault();
                    this.open();
                }
            });
        },

        open() {
            const overlay = document.getElementById('search-overlay');
            const input = document.getElementById('search-input');
            const toggle = document.getElementById('search-toggle');

            overlay.setAttribute('aria-hidden', 'false');
            toggle.setAttribute('aria-expanded', 'true');
            document.body.style.overflow = 'hidden';

            setTimeout(() => input && input.focus(), 100);
        },

        close() {
            const overlay = document.getElementById('search-overlay');
            const toggle = document.getElementById('search-toggle');

            overlay.setAttribute('aria-hidden', 'true');
            toggle.setAttribute('aria-expanded', 'false');
            document.body.style.overflow = '';
        }
    };

    // ==========================================================================
    // Reading Progress Bar
    // ==========================================================================

    const ReadingProgress = {
        init() {
            const progress = document.getElementById('reading-progress');
            if (!progress) return;

            const bar = progress.querySelector('.reading-progress-bar');
            if (!bar) return;

            const article = document.querySelector('.single-post-article, .single-page-article');
            if (!article) return;

            let ticking = false;

            const updateProgress = () => {
                const articleTop = article.offsetTop;
                const articleHeight = article.offsetHeight;
                const windowHeight = window.innerHeight;
                const scrollY = window.scrollY;

                const start = articleTop;
                const end = articleTop + articleHeight - windowHeight;
                const progressValue = Math.max(0, Math.min(1, (scrollY - start) / (end - start)));

                bar.style.width = `${progressValue * 100}%`;
                ticking = false;
            };

            window.addEventListener('scroll', () => {
                if (!ticking) {
                    requestAnimationFrame(updateProgress);
                    ticking = true;
                }
            }, { passive: true });
        }
    };

    // ==========================================================================
    // Back to Top Button
    // ==========================================================================

    const BackToTop = {
        init() {
            const button = document.getElementById('back-to-top');
            if (!button) return;

            let ticking = false;

            const checkVisibility = () => {
                if (window.scrollY > 500) {
                    button.classList.add('visible');
                } else {
                    button.classList.remove('visible');
                }
                ticking = false;
            };

            window.addEventListener('scroll', () => {
                if (!ticking) {
                    requestAnimationFrame(checkVisibility);
                    ticking = true;
                }
            }, { passive: true });

            button.addEventListener('click', () => {
                window.scrollTo({
                    top: 0,
                    behavior: 'smooth'
                });
            });
        }
    };

    // ==========================================================================
    // Enhanced Sticky Header with Glassmorphism
    // ==========================================================================

    const StickyHeader = {
        init() {
            const header = document.getElementById('site-header');
            if (!header) return;

            let lastScroll = 0;
            let ticking = false;

            const updateHeader = () => {
                const currentScroll = window.scrollY;

                // Add scrolled class for enhanced glassmorphism
                if (currentScroll > 50) {
                    header.classList.add('site-header--scrolled');
                } else {
                    header.classList.remove('site-header--scrolled');
                }

                // Optional: Hide/show header on scroll direction
                if (header.classList.contains('site-header--sticky')) {
                    if (currentScroll > lastScroll && currentScroll > 200) {
                        header.classList.add('header-hidden');
                    } else {
                        header.classList.remove('header-hidden');
                    }
                }

                lastScroll = currentScroll;
                ticking = false;
            };

            window.addEventListener('scroll', () => {
                if (!ticking) {
                    requestAnimationFrame(updateHeader);
                    ticking = true;
                }
            }, { passive: true });
        }
    };

    // ==========================================================================
    // Card Hover Effects (Micro-interactions)
    // ==========================================================================

    const CardEffects = {
        init() {
            // Add perspective effect on card hover
            const cards = document.querySelectorAll('.post-card, .hero-main, .hero-card');

            cards.forEach(card => {
                card.addEventListener('mousemove', (e) => {
                    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

                    const rect = card.getBoundingClientRect();
                    const x = e.clientX - rect.left;
                    const y = e.clientY - rect.top;

                    const centerX = rect.width / 2;
                    const centerY = rect.height / 2;

                    const rotateX = (y - centerY) / 30;
                    const rotateY = (centerX - x) / 30;

                    card.style.transform = `perspective(1000px) rotateX(${rotateX}deg) rotateY(${rotateY}deg) translateY(-6px) scale(1.01)`;
                });

                card.addEventListener('mouseleave', () => {
                    card.style.transform = '';
                });
            });
        }
    };

    // ==========================================================================
    // Scroll Animations (Fade in on scroll)
    // ==========================================================================

    const ScrollAnimations = {
        init() {
            if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

            const observerOptions = {
                threshold: 0.1,
                rootMargin: '0px 0px -50px 0px'
            };

            const observer = new IntersectionObserver((entries) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        entry.target.classList.add('is-visible');
                        observer.unobserve(entry.target);
                    }
                });
            }, observerOptions);

            // Animate cards on scroll
            document.querySelectorAll('.post-card, .hero-card, .widget, .related-post').forEach((el, index) => {
                el.style.opacity = '0';
                el.style.transform = 'translateY(20px)';
                el.style.transition = `opacity 0.5s ease ${index * 0.1}s, transform 0.5s ease ${index * 0.1}s`;
                observer.observe(el);
            });

            // Add visible styles
            const style = document.createElement('style');
            style.textContent = `
                .is-visible {
                    opacity: 1 !important;
                    transform: translateY(0) !important;
                }
            `;
            document.head.appendChild(style);
        }
    };

    // ==========================================================================
    // Comment Reply
    // ==========================================================================

    const CommentReply = {
        init() {
            const form = document.getElementById('comment-form');
            if (!form) return;

            // Reply button clicks
            document.querySelectorAll('.comment-reply-btn').forEach(btn => {
                btn.addEventListener('click', () => {
                    const commentId = btn.dataset.commentId;
                    const comment = document.getElementById(`comment-${commentId}`);
                    const authorName = comment.querySelector('.comment-author-name').textContent;

                    // Set parent ID
                    document.getElementById('comment-parent-id').value = commentId;

                    // Show reply info
                    const replyInfo = document.getElementById('comment-reply-info');
                    document.getElementById('reply-to-name').textContent = authorName;
                    replyInfo.style.display = 'flex';

                    // Scroll to form
                    form.scrollIntoView({ behavior: 'smooth', block: 'center' });
                    document.getElementById('comment-content').focus();
                });
            });

            // Cancel reply
            const cancelBtn = document.getElementById('cancel-reply');
            if (cancelBtn) {
                cancelBtn.addEventListener('click', () => {
                    document.getElementById('comment-parent-id').value = '';
                    document.getElementById('comment-reply-info').style.display = 'none';
                });
            }
        }
    };

    // ==========================================================================
    // Copy Link Button
    // ==========================================================================

    const CopyLink = {
        init() {
            document.querySelectorAll('.social-share-btn--copy').forEach(btn => {
                btn.addEventListener('click', async () => {
                    const url = btn.dataset.url || window.location.href;

                    try {
                        await navigator.clipboard.writeText(url);
                        btn.classList.add('copied');

                        // Show feedback
                        const originalTitle = btn.getAttribute('aria-label');
                        btn.setAttribute('aria-label', 'Copied!');

                        setTimeout(() => {
                            btn.classList.remove('copied');
                            btn.setAttribute('aria-label', originalTitle);
                        }, 2000);
                    } catch (err) {
                        console.error('Failed to copy:', err);
                    }
                });
            });
        }
    };

    // ==========================================================================
    // Lazy Loading Images with Fade
    // ==========================================================================

    const LazyLoad = {
        init() {
            const images = document.querySelectorAll('img[loading="lazy"]');

            images.forEach(img => {
                if (img.complete) {
                    img.classList.add('loaded');
                } else {
                    img.addEventListener('load', () => {
                        img.classList.add('loaded');
                    });
                }
            });

            // Add fade-in styles for lazy images
            const style = document.createElement('style');
            style.textContent = `
                img[loading="lazy"] {
                    opacity: 0;
                    transition: opacity 0.3s ease;
                }
                img[loading="lazy"].loaded {
                    opacity: 1;
                }
            `;
            document.head.appendChild(style);
        }
    };

    // ==========================================================================
    // External Links
    // ==========================================================================

    const ExternalLinks = {
        init() {
            document.querySelectorAll('.prose a').forEach(link => {
                if (link.hostname !== window.location.hostname) {
                    link.setAttribute('target', '_blank');
                    link.setAttribute('rel', 'noopener noreferrer');
                }
            });
        }
    };

    // ==========================================================================
    // Smooth Scroll for Anchor Links
    // ==========================================================================

    const SmoothScroll = {
        init() {
            document.querySelectorAll('a[href^="#"]').forEach(anchor => {
                anchor.addEventListener('click', (e) => {
                    const targetId = anchor.getAttribute('href');
                    if (targetId === '#') return;

                    const target = document.querySelector(targetId);
                    if (target) {
                        e.preventDefault();
                        const headerHeight = document.getElementById('site-header')?.offsetHeight || 0;
                        const targetPosition = target.offsetTop - headerHeight - 20;

                        window.scrollTo({
                            top: targetPosition,
                            behavior: 'smooth'
                        });
                    }
                });
            });
        }
    };

    // ==========================================================================
    // Table of Contents Generation
    // ==========================================================================

    const TableOfContents = {
        init() {
            const article = document.querySelector('.prose');
            const tocContainer = document.getElementById('table-of-contents');

            if (!article || !tocContainer) return;

            const headings = article.querySelectorAll('h2, h3');
            if (headings.length < 3) return;

            const toc = document.createElement('nav');
            toc.className = 'toc';
            toc.innerHTML = '<h4 class="toc-title">Table of Contents</h4>';

            const list = document.createElement('ul');
            list.className = 'toc-list';

            headings.forEach((heading, index) => {
                // Add ID to heading if not present
                if (!heading.id) {
                    heading.id = `heading-${index}`;
                }

                const item = document.createElement('li');
                item.className = heading.tagName === 'H3' ? 'toc-item toc-item--sub' : 'toc-item';

                const link = document.createElement('a');
                link.href = `#${heading.id}`;
                link.textContent = heading.textContent;
                link.className = 'toc-link';

                item.appendChild(link);
                list.appendChild(item);
            });

            toc.appendChild(list);
            tocContainer.appendChild(toc);
        }
    };

    // ==========================================================================
    // Button Ripple Effect
    // ==========================================================================

    const ButtonRipple = {
        init() {
            if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) return;

            document.querySelectorAll('.btn--primary, .newsletter-btn-large').forEach(btn => {
                btn.addEventListener('click', function(e) {
                    const rect = this.getBoundingClientRect();
                    const x = e.clientX - rect.left;
                    const y = e.clientY - rect.top;

                    const ripple = document.createElement('span');
                    ripple.style.cssText = `
                        position: absolute;
                        background: rgba(255, 255, 255, 0.3);
                        border-radius: 50%;
                        transform: scale(0);
                        animation: ripple 0.6s linear;
                        pointer-events: none;
                        left: ${x}px;
                        top: ${y}px;
                        width: 100px;
                        height: 100px;
                        margin-left: -50px;
                        margin-top: -50px;
                    `;

                    this.style.position = 'relative';
                    this.style.overflow = 'hidden';
                    this.appendChild(ripple);

                    setTimeout(() => ripple.remove(), 600);
                });
            });

            // Add ripple keyframes
            const style = document.createElement('style');
            style.textContent = `
                @keyframes ripple {
                    to {
                        transform: scale(4);
                        opacity: 0;
                    }
                }
            `;
            document.head.appendChild(style);
        }
    };

    // ==========================================================================
    // Newsletter Form Enhancement
    // ==========================================================================

    const NewsletterForm = {
        init() {
            const forms = document.querySelectorAll('.newsletter-form, .newsletter-form-large');

            forms.forEach(form => {
                form.addEventListener('submit', function(e) {
                    const btn = this.querySelector('button[type="submit"]');
                    const input = this.querySelector('input[type="email"]');

                    if (btn && input && input.value) {
                        btn.innerHTML = '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 6L9 17l-5-5"/></svg>';
                        btn.style.pointerEvents = 'none';
                    }
                });
            });
        }
    };

    // ==========================================================================
    // Focus Visible Polyfill (for older browsers)
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

            const style = document.createElement('style');
            style.textContent = `
                body.using-mouse *:focus {
                    outline: none !important;
                }
            `;
            document.head.appendChild(style);
        }
    };

    // ==========================================================================
    // Initialize All Modules
    // ==========================================================================

    document.addEventListener('DOMContentLoaded', () => {
        ThemeToggle.init();
        MobileNav.init();
        SearchOverlay.init();
        ReadingProgress.init();
        BackToTop.init();
        StickyHeader.init();
        CardEffects.init();
        ScrollAnimations.init();
        CommentReply.init();
        CopyLink.init();
        LazyLoad.init();
        ExternalLinks.init();
        SmoothScroll.init();
        TableOfContents.init();
        ButtonRipple.init();
        NewsletterForm.init();
        FocusVisible.init();
    });
})();
