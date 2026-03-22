/**
 * One-time script to fetch all avatar SVG parts from the notion-avatar-generator repo.
 * Run: node scripts/fetch-avatars.mjs
 *
 * Saves inner SVG content (paths only, no wrapper <svg> tag) to src/avatars/{category}/
 *
 * Note: The repo has typos in filenames ("Accesories" not "Accessories",
 * "Mousthace" not "Moustache") and spaces in names. We map to clean local names.
 */

import { writeFileSync, mkdirSync } from "fs";
import { join, dirname } from "path";
import { fileURLToPath } from "url";

const __dirname = dirname(fileURLToPath(import.meta.url));
const BASE = join(__dirname, "..", "src", "avatars");

const REPO_RAW = "https://raw.githubusercontent.com/HMarzban/notion-avatar-generator/main/src/assets/avatar-elements";

// Each entry: [repoFilename (without .svg), localFilename (without .svg)]
const PARTS = {
  face: {
    repo: "face",
    files: [
      ["Head=Head01", "Head01"], ["Head=Head02", "Head02"],
      ["Head=Head03", "Head03"], ["Head=Head04", "Head04"],
      ["Head=Head05", "Head05"], ["Head=Head06", "Head06"],
      ["Head=Head07", "Head07"], ["Head=Head08", "Head08"],
    ],
  },
  eyes: {
    repo: "eyes",
    files: [
      ["Eyes=Normal", "Normal"], ["Eyes=Closed", "Closed"],
      ["Eyes=Angry", "Angry"], ["Eyes=Cynic", "Cynic"],
      ["Eyes=Sad", "Sad"], ["Eyes=Thin", "Thin"],
    ],
  },
  mouth: {
    repo: "mouth",
    files: [
      ["Mouth=Normal Smile 1", "NormalSmile1"],
      ["Mouth=Normal Smile 2", "NormalSmile2"],
      ["Mouth=Open Mouth", "OpenMouth"],
      ["Mouth=Sad", "Sad"],
      ["Mouth=Nervous", "Nervous"],
      ["Mouth=Open Tooth", "OpenTooth"],
      ["Mouth=Angry", "Angry"],
      ["Mouth=Eat", "Eat"],
      ["Mouth=Hate", "Hate"],
      ["Mouth=Mouth11", "Mouth11"],
      ["Mouth=Whistle", "Whistle"],
    ],
  },
  hair: {
    repo: "hair",
    files: Array.from({ length: 28 }, (_, i) => {
      const n = String(i + 1).padStart(2, "0");
      return [`Hair=Style${n}`, `Style${n}`];
    }),
  },
  accessories: {
    repo: "accessories",
    // Note: repo has typo "Accesories" and "Mousthace"
    files: [
      ["Accesories=Glasses", "Glasses"],
      ["Accesories=Rounded Glasses", "RoundedGlasses"],
      ["Accesories=Blush", "Blush"],
      ["Accesories=Beard 1", "Beard1"],
      ["Accesories=Beard 2", "Beard2"],
      ["Accesories=Beard 3", "Beard3"],
      ["Accesories=Beard 4", "Beard4"],
      ["Accesories=Mousthace 1", "Moustache1"],
      ["Accesories=Mousthace 2", "Moustache2"],
      ["Accesories=Mousthace 3", "Moustache3"],
      ["Accesories=Mousthace 4", "Moustache4"],
      ["Accesories=Stylish Glasses", "StylishGlasses"],
      ["Accesories=Cap", "Cap"],
      ["Accesories=Earphone", "Earphone"],
      ["Accesories=Futuristic Glasses", "FuturisticGlasses"],
      ["Accesories=Mask", "Mask"],
      ["Accesories=Mask Google", "MaskGoogle"],
      ["Accesories=Waitress Tie", "WaitressTie"],
    ],
  },
};

function stripSvgWrapper(svg) {
  return svg
    .replace(/^\s*<svg[^>]*>\s*/i, "")
    .replace(/\s*<\/svg>\s*$/i, "")
    .trim();
}

async function fetchPart(category, repoFolder, repoName, localName) {
  // GitHub raw URLs need URL-encoding for spaces
  const url = `${REPO_RAW}/${repoFolder}/${encodeURIComponent(repoName + ".svg")}`;
  const res = await fetch(url);
  if (!res.ok) {
    console.error(`  FAILED: ${localName} (${res.status}) ${url}`);
    return false;
  }
  const svg = await res.text();
  const inner = stripSvgWrapper(svg);
  const outPath = join(BASE, category, `${localName}.svg`);
  writeFileSync(outPath, inner + "\n");
  console.log(`  OK: ${localName}`);
  return true;
}

async function main() {
  let total = 0, ok = 0, fail = 0;
  for (const [category, { repo, files }] of Object.entries(PARTS)) {
    const dir = join(BASE, category);
    mkdirSync(dir, { recursive: true });
    console.log(`\n[${category}] ${files.length} parts`);
    total += files.length;

    const results = await Promise.all(
      files.map(([repoName, localName]) => fetchPart(category, repo, repoName, localName))
    );
    ok += results.filter(Boolean).length;
    fail += results.filter((r) => !r).length;
  }
  console.log(`\nDone: ${ok}/${total} saved, ${fail} failed`);
}

main().catch(console.error);
