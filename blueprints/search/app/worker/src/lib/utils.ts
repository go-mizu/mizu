/**
 * Utility functions for the search worker.
 */

/**
 * Generate a random 16-character hex string.
 */
export function generateId(): string {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * Hash a query string and params into a SHA-256 hex digest for cache keys.
 */
export async function hashQuery(
  query: string,
  params: Record<string, string | number | boolean>
): Promise<string> {
  const raw = query + JSON.stringify(params);
  const encoded = new TextEncoder().encode(raw);
  const digest = await crypto.subtle.digest('SHA-256', encoded);
  const bytes = new Uint8Array(digest);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, '0'))
    .join('');
}

/**
 * Strip dangerous HTML tags (script, style, iframe, object, embed, form)
 * while keeping basic formatting tags (b, i, em, strong, p, br, ul, ol, li).
 */
export function sanitizeHtml(text: string): string {
  const dangerousTags = [
    'script',
    'style',
    'iframe',
    'object',
    'embed',
    'form',
    'input',
    'textarea',
    'select',
    'button',
    'link',
    'meta',
    'base',
    'applet',
  ];

  let result = text;

  for (const tag of dangerousTags) {
    // Remove opening and closing tags and their content for script/style
    if (tag === 'script' || tag === 'style') {
      const re = new RegExp(
        `<${tag}[^>]*>[\\s\\S]*?<\\/${tag}>`,
        'gi'
      );
      result = result.replace(re, '');
    }
    // Remove self-closing and opening/closing tags for the rest
    const openClose = new RegExp(
      `<\\/?${tag}[^>]*\\/?>`,
      'gi'
    );
    result = result.replace(openClose, '');
  }

  return result;
}

/**
 * Extract the hostname from a URL string.
 * Returns empty string if parsing fails.
 */
export function extractDomain(url: string): string {
  try {
    const parsed = new URL(url);
    return parsed.hostname;
  } catch {
    // Try a regex fallback for partial URLs
    const match = url.match(/^(?:https?:\/\/)?([^/?#]+)/);
    return match ? match[1] : '';
  }
}

/**
 * Truncate text to maxLen characters, adding ellipsis if truncated.
 */
export function truncate(text: string, maxLen: number): string {
  if (text.length <= maxLen) {
    return text;
  }
  return text.slice(0, maxLen) + '...';
}
