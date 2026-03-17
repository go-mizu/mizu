/**
 * Deterministic SVG avatar generators.
 *
 * Human avatars: Notion-style minimalist faces with hair, eyes, mouth.
 * Bot avatars: Geometric robot heads with antennas and LED eyes.
 *
 * Both are deterministic — same name always produces the same avatar.
 */

function hashCode(s: string): number {
  let h = 0;
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) - h + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

function pick<T>(arr: T[], hash: number, offset: number = 0): T {
  return arr[((hash >>> offset) + offset) % arr.length];
}

// ===== Human Avatars =====

const SKIN_TONES = ["#FFDBB4", "#EDB98A", "#D08B5B", "#AE5D29", "#694D3D", "#F8D5C2"];
const HAIR_COLORS = ["#2C1810", "#4A3728", "#8B6914", "#C4723A", "#1A1A2E", "#6B4226", "#D4A574", "#8B8B8B"];
const BG_COLORS_HUMAN = ["#E8F4FD", "#FDE8E8", "#E8FDE8", "#FDF3E8", "#F0E8FD", "#E8FDFA", "#FDE8F5", "#EEEEF5"];

function humanEyes(hash: number): string {
  const variant = hash % 5;
  const y = 38;
  switch (variant) {
    case 0: // simple dots
      return `<circle cx="35" cy="${y}" r="2.5" fill="#1a1a2e"/><circle cx="53" cy="${y}" r="2.5" fill="#1a1a2e"/>`;
    case 1: // open circles
      return `<circle cx="35" cy="${y}" r="3.5" fill="none" stroke="#1a1a2e" stroke-width="1.5"/><circle cx="53" cy="${y}" r="3.5" fill="none" stroke="#1a1a2e" stroke-width="1.5"/><circle cx="35" cy="${y}" r="1" fill="#1a1a2e"/><circle cx="53" cy="${y}" r="1" fill="#1a1a2e"/>`;
    case 2: // half-closed
      return `<path d="M32 ${y} Q35 ${y - 3} 38 ${y}" fill="none" stroke="#1a1a2e" stroke-width="1.8" stroke-linecap="round"/><path d="M50 ${y} Q53 ${y - 3} 56 ${y}" fill="none" stroke="#1a1a2e" stroke-width="1.8" stroke-linecap="round"/>`;
    case 3: // wide
      return `<ellipse cx="35" cy="${y}" rx="3.5" ry="4" fill="#1a1a2e"/><ellipse cx="53" cy="${y}" rx="3.5" ry="4" fill="#1a1a2e"/><circle cx="36" cy="${y - 1}" r="1.2" fill="#fff"/>` +
             `<circle cx="54" cy="${y - 1}" r="1.2" fill="#fff"/>`;
    default: // wink
      return `<circle cx="35" cy="${y}" r="2.5" fill="#1a1a2e"/><path d="M50 ${y} Q53 ${y - 3} 56 ${y}" fill="none" stroke="#1a1a2e" stroke-width="1.8" stroke-linecap="round"/>`;
  }
}

function humanMouth(hash: number): string {
  const variant = (hash >>> 4) % 5;
  const y = 50;
  switch (variant) {
    case 0: // smile
      return `<path d="M38 ${y} Q44 ${y + 5} 50 ${y}" fill="none" stroke="#1a1a2e" stroke-width="1.5" stroke-linecap="round"/>`;
    case 1: // line
      return `<line x1="39" y1="${y}" x2="49" y2="${y}" stroke="#1a1a2e" stroke-width="1.5" stroke-linecap="round"/>`;
    case 2: // open smile
      return `<path d="M37 ${y} Q44 ${y + 7} 51 ${y}" fill="#c44" stroke="#1a1a2e" stroke-width="1.2"/>`;
    case 3: // smirk
      return `<path d="M39 ${y} Q46 ${y + 4} 50 ${y - 1}" fill="none" stroke="#1a1a2e" stroke-width="1.5" stroke-linecap="round"/>`;
    default: // tiny o
      return `<ellipse cx="44" cy="${y + 1}" rx="2.5" ry="3" fill="#c44" stroke="#1a1a2e" stroke-width="1"/>`;
  }
}

function humanHair(hash: number, hairColor: string): string {
  const variant = (hash >>> 8) % 7;
  switch (variant) {
    case 0: // short crop
      return `<path d="M25 35 Q25 18 44 16 Q63 18 63 35" fill="${hairColor}" stroke="none"/><path d="M25 35 Q25 30 27 28" fill="none" stroke="${hairColor}" stroke-width="2"/>`;
    case 1: // long sides
      return `<path d="M24 35 Q24 16 44 14 Q64 16 64 35" fill="${hairColor}"/><path d="M24 35 L24 55" stroke="${hairColor}" stroke-width="5" stroke-linecap="round"/><path d="M64 35 L64 55" stroke="${hairColor}" stroke-width="5" stroke-linecap="round"/>`;
    case 2: // curly top
      return `<path d="M26 34 Q22 14 44 12 Q66 14 62 34" fill="${hairColor}"/>` +
             `<circle cx="30" cy="20" r="4" fill="${hairColor}"/><circle cx="38" cy="16" r="4" fill="${hairColor}"/><circle cx="46" cy="15" r="4" fill="${hairColor}"/><circle cx="54" cy="17" r="4" fill="${hairColor}"/><circle cx="60" cy="22" r="4" fill="${hairColor}"/>`;
    case 3: // buzz cut
      return `<path d="M27 35 Q27 22 44 20 Q61 22 61 35" fill="${hairColor}" opacity="0.7"/>`;
    case 4: // bun
      return `<path d="M26 34 Q26 18 44 16 Q62 18 62 34" fill="${hairColor}"/><circle cx="44" cy="13" r="7" fill="${hairColor}"/>`;
    case 5: // side part
      return `<path d="M25 35 Q25 18 44 15 Q63 18 63 35" fill="${hairColor}"/><path d="M25 30 Q30 26 35 28" fill="${hairColor}" stroke="${hairColor}" stroke-width="1"/>`;
    default: // bald/minimal
      return `<path d="M30 32 Q30 26 44 24 Q58 26 58 32" fill="${hairColor}" opacity="0.4"/>`;
  }
}

function humanAccessory(hash: number): string {
  const variant = (hash >>> 12) % 5;
  switch (variant) {
    case 0: // glasses
      return `<circle cx="35" cy="38" r="6" fill="none" stroke="#333" stroke-width="1.5"/><circle cx="53" cy="38" r="6" fill="none" stroke="#333" stroke-width="1.5"/><line x1="41" y1="38" x2="47" y2="38" stroke="#333" stroke-width="1.5"/>`;
    case 1: // round glasses
      return `<rect x="29" y="34" width="12" height="8" rx="4" fill="none" stroke="#555" stroke-width="1.2"/><rect x="47" y="34" width="12" height="8" rx="4" fill="none" stroke="#555" stroke-width="1.2"/><line x1="41" y1="38" x2="47" y2="38" stroke="#555" stroke-width="1.2"/>`;
    default: // none
      return "";
  }
}

export function humanAvatar(name: string, size: number = 88): string {
  const h = hashCode(name);
  const skin = pick(SKIN_TONES, h, 0);
  const hair = pick(HAIR_COLORS, h, 3);
  const bg = pick(BG_COLORS_HUMAN, h, 7);

  return `<svg viewBox="0 0 88 88" width="${size}" height="${size}" xmlns="http://www.w3.org/2000/svg">
<rect width="88" height="88" rx="16" fill="${bg}"/>
<ellipse cx="44" cy="44" rx="20" ry="23" fill="${skin}"/>
${humanHair(h, hair)}
${humanEyes(h)}
${humanMouth(h)}
${humanAccessory(h)}
</svg>`;
}

// ===== Bot/Agent Avatars =====

const BOT_ACCENTS = ["#4FC3F7", "#81C784", "#FFB74D", "#E57373", "#BA68C8", "#4DB6AC", "#FF8A65", "#7986CB"];
const BG_COLORS_BOT = ["#1a1a2e", "#16213e", "#0f3460", "#1b1b2f", "#162447", "#1f1f3a", "#0d1b2a", "#1e1e30"];

function botEyes(hash: number, accent: string): string {
  const variant = hash % 5;
  const y = 40;
  switch (variant) {
    case 0: // LED rectangles
      return `<rect x="30" y="${y - 3}" width="8" height="5" rx="1" fill="${accent}"/><rect x="50" y="${y - 3}" width="8" height="5" rx="1" fill="${accent}"/>`;
    case 1: // glowing circles
      return `<circle cx="34" cy="${y}" r="4" fill="${accent}" opacity="0.3"/><circle cx="34" cy="${y}" r="2.5" fill="${accent}"/><circle cx="54" cy="${y}" r="4" fill="${accent}" opacity="0.3"/><circle cx="54" cy="${y}" r="2.5" fill="${accent}"/>`;
    case 2: // horizontal slits
      return `<rect x="29" y="${y - 1}" width="10" height="2.5" rx="1" fill="${accent}"/><rect x="49" y="${y - 1}" width="10" height="2.5" rx="1" fill="${accent}"/>`;
    case 3: // visor
      return `<rect x="28" y="${y - 3}" width="32" height="6" rx="3" fill="${accent}" opacity="0.8"/>`;
    default: // dot matrix
      return `<circle cx="32" cy="${y - 2}" r="1.5" fill="${accent}"/><circle cx="36" cy="${y - 2}" r="1.5" fill="${accent}"/><circle cx="32" cy="${y + 2}" r="1.5" fill="${accent}"/><circle cx="36" cy="${y + 2}" r="1.5" fill="${accent}"/>` +
             `<circle cx="52" cy="${y - 2}" r="1.5" fill="${accent}"/><circle cx="56" cy="${y - 2}" r="1.5" fill="${accent}"/><circle cx="52" cy="${y + 2}" r="1.5" fill="${accent}"/><circle cx="56" cy="${y + 2}" r="1.5" fill="${accent}"/>`;
  }
}

function botAntenna(hash: number, accent: string): string {
  const variant = (hash >>> 4) % 5;
  switch (variant) {
    case 0: // single antenna
      return `<line x1="44" y1="22" x2="44" y2="12" stroke="#666" stroke-width="2"/><circle cx="44" cy="10" r="3" fill="${accent}"/>`;
    case 1: // dual antennas
      return `<line x1="36" y1="22" x2="32" y2="12" stroke="#666" stroke-width="1.5"/><circle cx="31" cy="11" r="2.5" fill="${accent}"/>` +
             `<line x1="52" y1="22" x2="56" y2="12" stroke="#666" stroke-width="1.5"/><circle cx="57" cy="11" r="2.5" fill="${accent}"/>`;
    case 2: // flat sensor bar
      return `<rect x="34" y="18" width="20" height="3" rx="1.5" fill="#666"/><rect x="38" y="16" width="12" height="2" rx="1" fill="${accent}" opacity="0.7"/>`;
    case 3: // zigzag
      return `<polyline points="44,22 44,16 40,13 44,10" fill="none" stroke="#666" stroke-width="1.5"/><circle cx="44" cy="9" r="2" fill="${accent}"/>`;
    default: // ears
      return `<rect x="22" y="32" width="4" height="10" rx="2" fill="#666"/><rect x="62" y="32" width="4" height="10" rx="2" fill="#666"/>` +
             `<rect x="23" y="34" width="2" height="4" rx="1" fill="${accent}" opacity="0.6"/><rect x="63" y="34" width="2" height="4" rx="1" fill="${accent}" opacity="0.6"/>`;
  }
}

function botMouth(hash: number, accent: string): string {
  const variant = (hash >>> 8) % 4;
  const y = 52;
  switch (variant) {
    case 0: // speaker grille
      return `<rect x="36" y="${y}" width="16" height="6" rx="1" fill="#333" stroke="#555" stroke-width="0.5"/>` +
             `<line x1="38" y1="${y + 1}" x2="38" y2="${y + 5}" stroke="#555" stroke-width="0.5"/>` +
             `<line x1="41" y1="${y + 1}" x2="41" y2="${y + 5}" stroke="#555" stroke-width="0.5"/>` +
             `<line x1="44" y1="${y + 1}" x2="44" y2="${y + 5}" stroke="#555" stroke-width="0.5"/>` +
             `<line x1="47" y1="${y + 1}" x2="47" y2="${y + 5}" stroke="#555" stroke-width="0.5"/>` +
             `<line x1="50" y1="${y + 1}" x2="50" y2="${y + 5}" stroke="#555" stroke-width="0.5"/>`;
    case 1: // simple line
      return `<line x1="37" y1="${y + 2}" x2="51" y2="${y + 2}" stroke="${accent}" stroke-width="1.5" stroke-linecap="round"/>`;
    case 2: // dots
      return `<circle cx="38" cy="${y + 3}" r="1.2" fill="${accent}"/><circle cx="44" cy="${y + 3}" r="1.2" fill="${accent}"/><circle cx="50" cy="${y + 3}" r="1.2" fill="${accent}"/>`;
    default: // zigzag mouth
      return `<polyline points="36,${y + 2} 40,${y + 5} 44,${y + 2} 48,${y + 5} 52,${y + 2}" fill="none" stroke="${accent}" stroke-width="1.2" stroke-linecap="round"/>`;
  }
}

export function botAvatar(name: string, size: number = 88): string {
  const h = hashCode(name);
  const accent = pick(BOT_ACCENTS, h, 0);
  const bg = pick(BG_COLORS_BOT, h, 5);

  return `<svg viewBox="0 0 88 88" width="${size}" height="${size}" xmlns="http://www.w3.org/2000/svg">
<rect width="88" height="88" rx="16" fill="${bg}"/>
${botAntenna(h, accent)}
<rect x="26" y="22" width="36" height="40" rx="6" fill="#4a4a5e" stroke="#5a5a6e" stroke-width="1"/>
<rect x="28" y="24" width="32" height="36" rx="4" fill="#3a3a4e"/>
${botEyes(h, accent)}
${botMouth(h, accent)}
</svg>`;
}

/**
 * Room icon — colored circle with first letter.
 */
export function roomIcon(title: string, size: number = 88): string {
  const h = hashCode(title || "room");
  const colors = ["#4FC3F7", "#81C784", "#FFB74D", "#E57373", "#BA68C8", "#4DB6AC", "#FF8A65", "#7986CB"];
  const color = pick(colors, h, 0);
  const letter = (title || "R")[0].toUpperCase();

  return `<svg viewBox="0 0 88 88" width="${size}" height="${size}" xmlns="http://www.w3.org/2000/svg">
<rect width="88" height="88" rx="16" fill="${color}" opacity="0.15"/>
<rect x="4" y="4" width="80" height="80" rx="12" fill="${color}" opacity="0.1"/>
<text x="44" y="52" text-anchor="middle" font-family="-apple-system,'Helvetica Neue',sans-serif" font-size="36" font-weight="700" fill="${color}">${letter}</text>
</svg>`;
}
