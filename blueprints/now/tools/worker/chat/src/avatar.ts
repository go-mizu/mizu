/**
 * Deterministic SVG avatar generators — all monochrome.
 *
 * Human: Notion-style face parts (grayscale via CSS filter)
 * Bot: Minimal geometric droid outlines
 * Room: Letter badge
 */

import { FACES, EYES, MOUTHS, HAIRS, ACCESSORIES } from "./avatar-data.gen";

function hashCode(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) - h + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

// ===== Human Avatar =====

export function humanAvatar(name: string, size: number = 80): string {
  const h = hashCode(name);
  const face = FACES[h % FACES.length];
  const eyes = EYES[(h >>> 3) % EYES.length];
  const mouth = MOUTHS[(h >>> 6) % MOUTHS.length];
  const hair = HAIRS[(h >>> 9) % HAIRS.length];
  const hasAccessory = (h >>> 15) % 3 === 0;
  const accessory = hasAccessory ? ACCESSORIES[(h >>> 18) % ACCESSORIES.length] : "";

  return `<svg viewBox="0 0 306 306" width="${size}" height="${size}" xmlns="http://www.w3.org/2000/svg">
${face}${eyes}${mouth}${accessory}${hair}
</svg>`;
}

// ===== Bot/Agent Avatar =====
// Minimal geometric droid — thin strokes, monochrome.

const HEAD_SHAPES = [
  `<rect x="16" y="22" width="56" height="48" fill="none" stroke="#888" stroke-width="1.5"/>`,
  `<path d="M18 24h52v44H18z" fill="none" stroke="#888" stroke-width="1.5"/>`,
  `<path d="M44 18l26 15v26L44 74 18 59V33z" fill="none" stroke="#888" stroke-width="1.5"/>`,
  `<rect x="18" y="24" width="52" height="44" rx="22" fill="none" stroke="#888" stroke-width="1.5"/>`,
  `<path d="M22 26h44l6 6v30l-6 6H22l-6-6V32z" fill="none" stroke="#888" stroke-width="1.5"/>`,
];

const EYE_STYLES = [
  `<circle cx="32" cy="43" r="4" fill="#888"/><circle cx="56" cy="43" r="4" fill="#888"/>`,
  `<rect x="27" y="40" width="10" height="6" fill="#888"/><rect x="51" y="40" width="10" height="6" fill="#888"/>`,
  `<line x1="26" y1="43" x2="38" y2="43" stroke="#888" stroke-width="2"/><line x1="50" y1="43" x2="62" y2="43" stroke="#888" stroke-width="2"/>`,
  `<circle cx="30" cy="42" r="2" fill="#888"/><circle cx="38" cy="42" r="2" fill="#888"/><circle cx="50" cy="42" r="2" fill="#888"/><circle cx="58" cy="42" r="2" fill="#888"/>`,
  `<rect x="24" y="41" width="40" height="4" fill="#888" opacity="0.6"/>`,
  `<path d="M30 39l4 4-4 4" fill="none" stroke="#888" stroke-width="1.5"/><path d="M58 39l-4 4 4 4" fill="none" stroke="#888" stroke-width="1.5"/>`,
];

const MOUTH_STYLES = [
  `<line x1="34" y1="57" x2="54" y2="57" stroke="#888" stroke-width="1" opacity="0.4"/>`,
  `<circle cx="38" cy="57" r="1" fill="#888" opacity="0.4"/><circle cx="44" cy="57" r="1" fill="#888" opacity="0.4"/><circle cx="50" cy="57" r="1" fill="#888" opacity="0.4"/>`,
  `<rect x="37" y="55" width="2" height="2" fill="#888" opacity="0.3"/><rect x="43" y="55" width="2" height="2" fill="#888" opacity="0.3"/><rect x="49" y="55" width="2" height="2" fill="#888" opacity="0.3"/>`,
  `<path d="M36 55a10 10 0 0016 0" fill="none" stroke="#888" stroke-width="1" opacity="0.4"/>`,
  ``,
];

const ANTENNA_STYLES = [
  `<line x1="44" y1="22" x2="44" y2="10" stroke="#888" stroke-width="1"/><circle cx="44" cy="8" r="2" fill="#888"/>`,
  `<line x1="44" y1="22" x2="36" y2="10" stroke="#888" stroke-width="1"/><line x1="44" y1="22" x2="52" y2="10" stroke="#888" stroke-width="1"/><circle cx="36" cy="9" r="1.5" fill="#888"/><circle cx="52" cy="9" r="1.5" fill="#888"/>`,
  `<path d="M44 22v-8" fill="none" stroke="#888" stroke-width="1"/>`,
  ``,
];

export function botAvatar(name: string, size: number = 80): string {
  const h = hashCode(name);
  const head = HEAD_SHAPES[(h >>> 2) % HEAD_SHAPES.length];
  const eyes = EYE_STYLES[(h >>> 5) % EYE_STYLES.length];
  const mouth = MOUTH_STYLES[(h >>> 8) % MOUTH_STYLES.length];
  const antenna = ANTENNA_STYLES[(h >>> 11) % ANTENNA_STYLES.length];

  return `<svg viewBox="0 0 88 88" width="${size}" height="${size}" xmlns="http://www.w3.org/2000/svg">
${antenna}${head}${eyes}${mouth}
</svg>`;
}

// ===== Room Icon =====

export function roomIcon(title: string, size: number = 80): string {
  const letter = (title || "R")[0].toUpperCase();

  return `<svg viewBox="0 0 88 88" width="${size}" height="${size}" xmlns="http://www.w3.org/2000/svg">
<rect width="88" height="88" fill="#E5E5E5"/>
<text x="44" y="53" text-anchor="middle" font-family="'JetBrains Mono',monospace" font-size="34" font-weight="600" fill="#777">${letter}</text>
</svg>`;
}
