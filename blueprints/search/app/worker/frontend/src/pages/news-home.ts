/**
 * News Home Page - Google News clone homepage.
 * Features: Top Stories, For You, Local News, Categories, Following.
 */

import { Router } from '../lib/router';
import { api } from '../api';
import type { NewsArticle, NewsCategory, NewsHomeResponse } from '../api';
import { renderSearchBox, initSearchBox } from '../components/search-box';

// Icons
const ICON_SETTINGS = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12.22 2h-.44a2 2 0 0 0-2 2v.18a2 2 0 0 1-1 1.73l-.43.25a2 2 0 0 1-2 0l-.15-.08a2 2 0 0 0-2.73.73l-.22.38a2 2 0 0 0 .73 2.73l.15.1a2 2 0 0 1 1 1.72v.51a2 2 0 0 1-1 1.74l-.15.09a2 2 0 0 0-.73 2.73l.22.38a2 2 0 0 0 2.73.73l.15-.08a2 2 0 0 1 2 0l.43.25a2 2 0 0 1 1 1.73V20a2 2 0 0 0 2 2h.44a2 2 0 0 0 2-2v-.18a2 2 0 0 1 1-1.73l.43-.25a2 2 0 0 1 2 0l.15.08a2 2 0 0 0 2.73-.73l.22-.39a2 2 0 0 0-.73-2.73l-.15-.08a2 2 0 0 1-1-1.74v-.5a2 2 0 0 1 1-1.74l.15-.09a2 2 0 0 0 .73-2.73l-.22-.38a2 2 0 0 0-2.73-.73l-.15.08a2 2 0 0 1-2 0l-.43-.25a2 2 0 0 1-1-1.73V4a2 2 0 0 0-2-2z"/><circle cx="12" cy="12" r="3"/></svg>`;
const ICON_HOME = `<svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path d="M10 20v-6h4v6h5v-8h3L12 3 2 12h3v8z"/></svg>`;
const ICON_STAR = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26 12 2"/></svg>`;
const ICON_BOOKMARK = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M19 21l-7-5-7 5V5a2 2 0 0 1 2-2h10a2 2 0 0 1 2 2z"/></svg>`;
const ICON_LOCATION = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 10c0 7-9 13-9 13s-9-6-9-13a9 9 0 0 1 18 0z"/><circle cx="12" cy="10" r="3"/></svg>`;
const ICON_GLOBE = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>`;
const ICON_BRIEFCASE = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="7" width="20" height="14" rx="2" ry="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/></svg>`;
const ICON_CPU = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="4" y="4" width="16" height="16" rx="2" ry="2"/><rect x="9" y="9" width="6" height="6"/><line x1="9" y1="1" x2="9" y2="4"/><line x1="15" y1="1" x2="15" y2="4"/><line x1="9" y1="20" x2="9" y2="23"/><line x1="15" y1="20" x2="15" y2="23"/><line x1="20" y1="9" x2="23" y2="9"/><line x1="20" y1="14" x2="23" y2="14"/><line x1="1" y1="9" x2="4" y2="9"/><line x1="1" y1="14" x2="4" y2="14"/></svg>`;
const ICON_FILM = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/><line x1="2" y1="7" x2="7" y2="7"/><line x1="2" y1="17" x2="7" y2="17"/><line x1="17" y1="17" x2="22" y2="17"/><line x1="17" y1="7" x2="22" y2="7"/></svg>`;
const ICON_ACTIVITY = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/></svg>`;
const ICON_BEAKER = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2v6.5a.5.5 0 0 0 .5.5h3a.5.5 0 0 1 .5.5v1a.5.5 0 0 1-.5.5H14.5a.5.5 0 0 0-.5.5V22H10V11.5a.5.5 0 0 0-.5-.5H6.5a.5.5 0 0 1-.5-.5v-1a.5.5 0 0 1 .5-.5h3a.5.5 0 0 0 .5-.5V2"/></svg>`;
const ICON_HEART = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"/></svg>`;
const ICON_ZAPP = `<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>`;

// Category configuration
const CATEGORIES: { id: NewsCategory; label: string; icon: string }[] = [
  { id: 'top', label: 'Top Stories', icon: ICON_ZAPP },
  { id: 'world', label: 'World', icon: ICON_GLOBE },
  { id: 'nation', label: 'U.S.', icon: ICON_HOME },
  { id: 'business', label: 'Business', icon: ICON_BRIEFCASE },
  { id: 'technology', label: 'Technology', icon: ICON_CPU },
  { id: 'entertainment', label: 'Entertainment', icon: ICON_FILM },
  { id: 'sports', label: 'Sports', icon: ICON_ACTIVITY },
  { id: 'science', label: 'Science', icon: ICON_BEAKER },
  { id: 'health', label: 'Health', icon: ICON_HEART },
];

export function renderNewsHomePage(): string {
  const today = new Date().toLocaleDateString('en-US', {
    weekday: 'long',
    month: 'long',
    day: 'numeric',
  });

  return `
    <div class="news-layout">
      <!-- Sidebar Navigation -->
      <nav class="news-sidebar">
        <div class="news-sidebar-header">
          <a href="/" data-link class="news-logo">
            <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
            <span class="news-logo-suffix">News</span>
          </a>
        </div>

        <div class="news-nav-section">
          <a href="/news-home" data-link class="news-nav-item active">
            ${ICON_HOME}
            <span>Home</span>
          </a>
          <a href="/news-home?section=for-you" data-link class="news-nav-item">
            ${ICON_STAR}
            <span>For you</span>
          </a>
          <a href="/news-home?section=following" data-link class="news-nav-item">
            ${ICON_BOOKMARK}
            <span>Following</span>
          </a>
        </div>

        <div class="news-nav-divider"></div>

        <div class="news-nav-section">
          ${CATEGORIES.map(
            (cat) => `
            <a href="/news-home?category=${cat.id}" data-link class="news-nav-item" data-category="${cat.id}">
              ${cat.icon}
              <span>${cat.label}</span>
            </a>
          `
          ).join('')}
        </div>
      </nav>

      <!-- Main Content -->
      <main class="news-main">
        <!-- Header -->
        <header class="news-header">
          <div class="news-header-left">
            <button class="news-menu-btn" id="menu-toggle">
              <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <line x1="3" y1="12" x2="21" y2="12"></line>
                <line x1="3" y1="6" x2="21" y2="6"></line>
                <line x1="3" y1="18" x2="21" y2="18"></line>
              </svg>
            </button>
            <a href="/" data-link class="news-logo-mobile">
              <span style="color: #4285F4">M</span><span style="color: #EA4335">i</span><span style="color: #FBBC05">z</span><span style="color: #34A853">u</span>
              <span class="news-logo-suffix">News</span>
            </a>
          </div>

          <div class="news-search-container">
            ${renderSearchBox({ size: 'sm', placeholder: 'Search for topics, locations & sources' })}
          </div>

          <div class="news-header-right">
            <button class="news-icon-btn" id="location-btn" title="Change location">
              ${ICON_LOCATION}
            </button>
            <a href="/settings" data-link class="news-icon-btn" title="Settings">
              ${ICON_SETTINGS}
            </a>
          </div>
        </header>

        <!-- Content Area -->
        <div class="news-content" id="news-content">
          <div class="news-briefing">
            <h1 class="news-briefing-title">Your briefing</h1>
            <p class="news-briefing-date">${today}</p>
          </div>

          <!-- Loading State -->
          <div class="news-loading" id="news-loading">
            <div class="spinner"></div>
            <p>Loading your news...</p>
          </div>

          <!-- Top Stories Section -->
          <section class="news-section" id="top-stories-section" style="display: none;">
            <h2 class="news-section-title">Top stories</h2>
            <div class="news-grid" id="top-stories-grid"></div>
          </section>

          <!-- For You Section -->
          <section class="news-section" id="for-you-section" style="display: none;">
            <h2 class="news-section-title">For you</h2>
            <div class="news-list" id="for-you-list"></div>
          </section>

          <!-- Local News Section -->
          <section class="news-section" id="local-section" style="display: none;">
            <div class="news-section-header">
              <h2 class="news-section-title">
                ${ICON_LOCATION}
                <span id="local-title">Local news</span>
              </h2>
              <button class="news-text-btn" id="change-location-btn">Change location</button>
            </div>
            <div class="news-horizontal-scroll" id="local-news-scroll"></div>
          </section>

          <!-- Category Sections -->
          <div id="category-sections"></div>
        </div>
      </main>
    </div>
  `;
}

export function initNewsHomePage(router: Router, query: Record<string, string>): void {
  initSearchBox((q) => {
    router.navigate(`/news?q=${encodeURIComponent(q)}`);
  });

  // Menu toggle for mobile
  const menuToggle = document.getElementById('menu-toggle');
  const sidebar = document.querySelector('.news-sidebar');
  if (menuToggle && sidebar) {
    menuToggle.addEventListener('click', () => {
      sidebar.classList.toggle('open');
    });
  }

  // Load news data
  loadNewsHome(query);
}

async function loadNewsHome(query: Record<string, string>): Promise<void> {
  const loading = document.getElementById('news-loading');
  const topStoriesSection = document.getElementById('top-stories-section');
  const forYouSection = document.getElementById('for-you-section');
  const localSection = document.getElementById('local-section');

  try {
    const response = await api.newsHome();

    if (loading) loading.style.display = 'none';

    // Render Top Stories
    if (topStoriesSection && response.topStories.length > 0) {
      topStoriesSection.style.display = 'block';
      const grid = document.getElementById('top-stories-grid');
      if (grid) {
        grid.innerHTML = renderTopStoriesGrid(response.topStories);
      }
    }

    // Render For You
    if (forYouSection && response.forYou.length > 0) {
      forYouSection.style.display = 'block';
      const list = document.getElementById('for-you-list');
      if (list) {
        list.innerHTML = response.forYou.slice(0, 10).map((a) => renderNewsListItem(a)).join('');
      }
    }

    // Render Local News
    if (localSection && response.localNews.length > 0) {
      localSection.style.display = 'block';
      const scroll = document.getElementById('local-news-scroll');
      if (scroll) {
        scroll.innerHTML = response.localNews.map((a) => renderCompactCard(a)).join('');
      }
    }

    // Render Category Sections
    const categorySections = document.getElementById('category-sections');
    if (categorySections && response.categories) {
      const html = Object.entries(response.categories)
        .filter(([_, articles]) => articles && articles.length > 0)
        .map(([category, articles]) => renderCategorySection(category as NewsCategory, articles!))
        .join('');
      categorySections.innerHTML = html;
    }

    // Track article clicks for personalization
    initArticleTracking();
  } catch (err) {
    if (loading) {
      loading.innerHTML = `
        <div class="news-error">
          <p>Failed to load news. Please try again.</p>
          <button class="news-btn" onclick="location.reload()">Retry</button>
        </div>
      `;
    }
    console.error('Failed to load news:', err);
  }
}

function renderTopStoriesGrid(articles: NewsArticle[]): string {
  if (articles.length === 0) return '';

  const featured = articles[0];
  const secondary = articles.slice(1, 3);
  const rest = articles.slice(3, 9);

  return `
    <div class="news-featured-row">
      ${renderFeaturedCard(featured)}
      <div class="news-secondary-col">
        ${secondary.map((a) => renderMediumCard(a)).join('')}
      </div>
    </div>
    <div class="news-grid-row">
      ${rest.map((a) => renderSmallCard(a)).join('')}
    </div>
  `;
}

function renderFeaturedCard(article: NewsArticle): string {
  const timeAgo = formatTimeAgo(article.publishedAt);

  return `
    <article class="news-card news-card-featured">
      ${
        article.imageUrl
          ? `<img class="news-card-image" src="${escapeAttr(article.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`
          : ''
      }
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${escapeAttr(article.sourceIcon || '')}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${escapeHtml(article.source)}</span>
          <span class="news-time">${timeAgo}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${escapeAttr(article.url)}" target="_blank" rel="noopener" onclick="trackArticleClick('${article.id}')">${escapeHtml(article.title)}</a>
        </h3>
        <p class="news-card-snippet">${escapeHtml(article.snippet)}</p>
        ${
          article.clusterId
            ? `<a href="/news-home?story=${article.clusterId}" data-link class="news-full-coverage">Full coverage</a>`
            : ''
        }
      </div>
    </article>
  `;
}

function renderMediumCard(article: NewsArticle): string {
  const timeAgo = formatTimeAgo(article.publishedAt);

  return `
    <article class="news-card news-card-medium">
      <div class="news-card-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${escapeAttr(article.sourceIcon || '')}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${escapeHtml(article.source)}</span>
          <span class="news-time">${timeAgo}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${escapeAttr(article.url)}" target="_blank" rel="noopener">${escapeHtml(article.title)}</a>
        </h3>
      </div>
      ${
        article.imageUrl
          ? `<img class="news-card-thumb" src="${escapeAttr(article.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`
          : ''
      }
    </article>
  `;
}

function renderSmallCard(article: NewsArticle): string {
  const timeAgo = formatTimeAgo(article.publishedAt);

  return `
    <article class="news-card news-card-small">
      <div class="news-card-content">
        <div class="news-card-meta">
          <span class="news-source-name">${escapeHtml(article.source)}</span>
          <span class="news-time">${timeAgo}</span>
        </div>
        <h3 class="news-card-title">
          <a href="${escapeAttr(article.url)}" target="_blank" rel="noopener">${escapeHtml(article.title)}</a>
        </h3>
      </div>
    </article>
  `;
}

function renderCompactCard(article: NewsArticle): string {
  const timeAgo = formatTimeAgo(article.publishedAt);

  return `
    <article class="news-card news-card-compact">
      ${
        article.imageUrl
          ? `<img class="news-card-thumb-sm" src="${escapeAttr(article.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`
          : '<div class="news-card-thumb-placeholder"></div>'
      }
      <div class="news-card-content">
        <span class="news-source-name">${escapeHtml(article.source)}</span>
        <h4 class="news-card-title-sm">
          <a href="${escapeAttr(article.url)}" target="_blank" rel="noopener">${escapeHtml(article.title)}</a>
        </h4>
        <span class="news-time">${timeAgo}</span>
      </div>
    </article>
  `;
}

function renderNewsListItem(article: NewsArticle): string {
  const timeAgo = formatTimeAgo(article.publishedAt);

  return `
    <article class="news-list-item">
      <div class="news-list-content">
        <div class="news-card-meta">
          <img class="news-source-icon" src="${escapeAttr(article.sourceIcon || '')}" alt="" onerror="this.style.display='none'" />
          <span class="news-source-name">${escapeHtml(article.source)}</span>
          <span class="news-time">${timeAgo}</span>
        </div>
        <h3 class="news-list-title">
          <a href="${escapeAttr(article.url)}" target="_blank" rel="noopener">${escapeHtml(article.title)}</a>
        </h3>
        <p class="news-list-snippet">${escapeHtml(article.snippet)}</p>
      </div>
      ${
        article.imageUrl
          ? `<img class="news-list-thumb" src="${escapeAttr(article.imageUrl)}" alt="" loading="lazy" onerror="this.style.display='none'" />`
          : ''
      }
    </article>
  `;
}

function renderCategorySection(category: NewsCategory, articles: NewsArticle[]): string {
  const categoryInfo = CATEGORIES.find((c) => c.id === category);
  if (!categoryInfo) return '';

  return `
    <section class="news-section">
      <div class="news-section-header">
        <h2 class="news-section-title">
          ${categoryInfo.icon}
          <span>${categoryInfo.label}</span>
        </h2>
        <a href="/news-home?category=${category}" data-link class="news-text-btn">More ${categoryInfo.label.toLowerCase()}</a>
      </div>
      <div class="news-horizontal-scroll">
        ${articles.slice(0, 5).map((a) => renderCompactCard(a)).join('')}
      </div>
    </section>
  `;
}

function formatTimeAgo(dateStr: string): string {
  try {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - date.getTime();
    const hours = Math.floor(diff / (1000 * 60 * 60));
    const days = Math.floor(diff / (1000 * 60 * 60 * 24));

    if (hours < 1) return 'Just now';
    if (hours < 24) return `${hours}h ago`;
    if (days === 1) return '1 day ago';
    if (days < 7) return `${days} days ago`;

    return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
  } catch {
    return '';
  }
}

function initArticleTracking(): void {
  // Track clicks on article links for personalization
  document.querySelectorAll('.news-card a, .news-list-item a').forEach((link) => {
    link.addEventListener('click', function (this: HTMLAnchorElement) {
      const card = this.closest('.news-card, .news-list-item');
      if (card) {
        // Fire and forget - don't wait for response
        const articleId = card.getAttribute('data-article-id');
        if (articleId) {
          api.recordNewsRead({
            id: articleId,
            url: this.href,
            title: this.textContent || '',
            snippet: '',
            source: '',
            sourceUrl: '',
            publishedAt: '',
            category: 'top',
            engines: [],
            score: 1,
          }).catch(() => {});
        }
      }
    });
  });
}

function escapeHtml(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function escapeAttr(str: string): string {
  return str.replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;').replace(/>/g, '&gt;');
}
