// Deterministic per-nickname color/initial avatar. We don't fetch
// from DiceBear or any other service so the app stays offline-capable.
// Hash the input → pick from a curated palette → initial letter.

const PALETTE = [
  "#8b5cf6", // violet
  "#6366f1", // indigo
  "#22d3ee", // cyan
  "#f472b6", // pink
  "#fb7185", // rose
  "#fbbf24", // amber
  "#34d399", // emerald
  "#60a5fa", // blue
];

function hashString(s: string): number {
  let h = 5381;
  for (let i = 0; i < s.length; i++) {
    h = ((h << 5) + h + s.charCodeAt(i)) | 0;
  }
  return Math.abs(h);
}

export function avatarColor(name: string): string {
  return PALETTE[hashString(name) % PALETTE.length];
}

export function avatarInitial(name: string): string {
  const trimmed = name.trim();
  if (!trimmed) return "?";
  return trimmed[0].toUpperCase();
}
