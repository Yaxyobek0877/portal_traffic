#!/usr/bin/env python3
"""
Generate Portal launcher icons for all Android density buckets.

We mirror the desktop client's animated wormhole motif (Logo.tsx):
concentric rings (some dashed) around a glowing white centre.
The motif degenerates gracefully at 48x48; rings fewer than ~3 px
across collapse into a halo, which is fine for the smallest densities.

Outputs (all PNG):
  res/mipmap-{mdpi,hdpi,xhdpi,xxhdpi,xxxhdpi}/ic_launcher.png
  res/mipmap-{...}/ic_launcher_round.png
  res/mipmap-{...}/ic_launcher_foreground.png   (adaptive foreground;
      transparent bg, sized to fit the 66/108 safe zone)
  res/mipmap-{...}/ic_launcher_monochrome.png   (themed/monochrome icon)

Plus a one-off 512x512 master for the Play Store entry, written to
  build/playstore_icon.png (gitignored — generated artifact).

Run after cloning, or re-run when the desktop logo changes:
  python3 scripts/gen_icons.py
"""

from __future__ import annotations

import math
import os
import sys
from pathlib import Path

try:
    from PIL import Image, ImageDraw, ImageFilter
except ImportError:
    sys.exit("PIL/Pillow required: pip install Pillow")

ROOT = Path(__file__).resolve().parent.parent
RES = ROOT / "app" / "src" / "main" / "res"

# Adaptive icon canvas is 108dp; safe zone is the inner 66dp (radius 33/54).
# Legacy launcher canvas is 48dp at mdpi.
DENSITIES = {
    "mdpi": 1.0,
    "hdpi": 1.5,
    "xhdpi": 2.0,
    "xxhdpi": 3.0,
    "xxxhdpi": 4.0,
}

LEGACY_BASE = 48
ADAPTIVE_BASE = 108

# Brand palette — matches client/frontend/src/components/Logo.tsx and
# the desktop Tailwind theme (violet-500, indigo-500, cyan-400).
VIOLET = (139, 92, 246)
INDIGO = (99, 102, 241)
CYAN = (34, 211, 238)
NAVY_BG = (13, 19, 34)  # zinc-950-ish; same as the app shell


def hex_to_rgb(h: str) -> tuple[int, int, int]:
    h = h.lstrip("#")
    return tuple(int(h[i:i + 2], 16) for i in (0, 2, 4))


def lerp(a, b, t):
    return tuple(int(a[i] + (b[i] - a[i]) * t) for i in range(3))


def grad_color(t: float) -> tuple[int, int, int]:
    """Three-stop gradient: violet -> indigo -> cyan, t in [0,1]."""
    if t < 0.6:
        return lerp(VIOLET, INDIGO, t / 0.6)
    return lerp(INDIGO, CYAN, (t - 0.6) / 0.4)


def dashed_arc(draw: ImageDraw.ImageDraw, bbox, color, width, dash_deg, gap_deg, phase=0):
    angle = phase
    while angle < 360 + phase:
        end = min(angle + dash_deg, 360 + phase)
        draw.arc(bbox, angle, end, fill=color, width=width)
        angle += dash_deg + gap_deg


def make_foreground(size: int, fit_ratio: float = 0.61) -> Image.Image:
    """Wormhole motif on transparent background.

    fit_ratio is the fraction of the canvas radius the outer ring may
    occupy. 0.61 matches the adaptive-icon safe zone; pass 0.92 when
    rendering for legacy square icons that fill the canvas edge-to-edge.
    """
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    cx = cy = size / 2.0

    # Logo viewBox is 200 wide; outer ring sits at r=92 (i.e. 0.92 of
    # half-canvas in the source SVG). We rescale so that ring lands at
    # `fit_ratio` of our canvas radius.
    canvas_r = size / 2.0
    target_outer_r = canvas_r * fit_ratio
    s = target_outer_r / 92.0  # one viewBox unit in pixels

    # ----- glow layer (drawn first, multiplied via alpha-composite) -----
    glow = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    glow_draw = ImageDraw.Draw(glow)
    glow_max = max(2, int(28 * s))
    for i in range(glow_max, 0, -1):
        falloff = (1.0 - i / glow_max) ** 1.5
        a = int(170 * falloff)
        r, g, b = VIOLET
        glow_draw.ellipse((cx - i, cy - i, cx + i, cy + i), fill=(r, g, b, a))
    # soften
    glow = glow.filter(ImageFilter.GaussianBlur(radius=max(1, s * 0.6)))

    img = Image.alpha_composite(img, glow)
    draw = ImageDraw.Draw(img)

    # Rings: (radius_units, stroke_units, gradient_t, dashed, dash_deg, gap_deg, alpha)
    # Mirrors the radii and dash patterns from Logo.tsx but tuned so dashes
    # remain readable at small sizes.
    rings = [
        (92, 1.5, 0.10, True, 6, 14, 150),
        (78, 1.7, 0.30, True, 12, 8, 195),
        (62, 2.0, 0.50, True, 4, 8, 220),
        (46, 2.6, 0.75, True, 22, 10, 255),
        (30, 1.6, 0.92, False, 0, 0, 200),
    ]
    for r_u, w_u, t, dashed, dash_deg, gap_deg, alpha in rings:
        radius = r_u * s
        width = max(1, int(round(w_u * s)))
        bbox = (cx - radius, cy - radius, cx + radius, cy + radius)
        col = grad_color(t) + (alpha,)
        if dashed and width > 0 and 2 * math.pi * radius > 8:
            dashed_arc(draw, bbox, col, width, dash_deg, gap_deg)
        else:
            draw.ellipse(bbox, outline=col, width=width)

    # Centre dot (white)
    dot_r = max(2, int(7 * s))
    draw.ellipse((cx - dot_r, cy - dot_r, cx + dot_r, cy + dot_r), fill=(255, 255, 255, 250))

    return img


def make_legacy_square(size: int) -> Image.Image:
    """Pre-API-26 square launcher: dark navy bg + wormhole, slightly rounded corners."""
    bg = Image.new("RGBA", (size, size), NAVY_BG + (255,))
    fg = make_foreground(size, fit_ratio=0.78)
    bg = Image.alpha_composite(bg, fg)
    # Round corners to ~22% — matches Pixel/A14 default mask better than
    # a perfect square, and on launchers that ignore the adaptive icon
    # it's still visually consistent with the round variant.
    radius = int(size * 0.22)
    mask = Image.new("L", (size, size), 0)
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, size, size), radius=radius, fill=255)
    out = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    out.paste(bg, mask=mask)
    return out


def make_legacy_round(size: int) -> Image.Image:
    bg = Image.new("RGBA", (size, size), NAVY_BG + (255,))
    fg = make_foreground(size, fit_ratio=0.78)
    bg = Image.alpha_composite(bg, fg)
    mask = Image.new("L", (size, size), 0)
    ImageDraw.Draw(mask).ellipse((0, 0, size, size), fill=255)
    out = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    out.paste(bg, mask=mask)
    return out


def make_monochrome(size: int) -> Image.Image:
    """Themed/monochrome icon: white-ish wormhole on transparent bg.

    Android 13+ themed icons are tinted by the launcher; the source
    must be a white-on-transparent silhouette using only the alpha
    channel. We render the foreground in white and discard the colour
    information, leaving alpha to carry the shape.
    """
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    cx = cy = size / 2.0
    canvas_r = size / 2.0
    target_outer_r = canvas_r * 0.61
    s = target_outer_r / 92.0

    draw = ImageDraw.Draw(img)
    rings = [
        (92, 1.5, True, 6, 14, 200),
        (78, 1.7, True, 12, 8, 230),
        (62, 2.0, True, 4, 8, 240),
        (46, 2.6, True, 22, 10, 255),
        (30, 1.6, False, 0, 0, 230),
    ]
    for r_u, w_u, dashed, dash_deg, gap_deg, alpha in rings:
        radius = r_u * s
        width = max(1, int(round(w_u * s)))
        bbox = (cx - radius, cy - radius, cx + radius, cy + radius)
        col = (255, 255, 255, alpha)
        if dashed and 2 * math.pi * radius > 8:
            dashed_arc(draw, bbox, col, width, dash_deg, gap_deg)
        else:
            draw.ellipse(bbox, outline=col, width=width)

    dot_r = max(2, int(7 * s))
    draw.ellipse((cx - dot_r, cy - dot_r, cx + dot_r, cy + dot_r), fill=(255, 255, 255, 255))
    return img


def main() -> None:
    # Drop the old WebPs first — gradle merges every variant in mipmap-* into
    # the APK, so leftover webps would shadow the PNGs. (mipmap-anydpi/*.xml
    # stays; we update its drawable references separately.)
    for d in DENSITIES:
        folder = RES / f"mipmap-{d}"
        folder.mkdir(parents=True, exist_ok=True)
        for stale in folder.glob("ic_launcher*.webp"):
            print(f"  removing stale {stale.relative_to(RES)}")
            stale.unlink()

    for density, mult in DENSITIES.items():
        folder = RES / f"mipmap-{density}"
        legacy = int(LEGACY_BASE * mult)
        adaptive = int(ADAPTIVE_BASE * mult)

        make_legacy_square(legacy).save(folder / "ic_launcher.png", optimize=True)
        make_legacy_round(legacy).save(folder / "ic_launcher_round.png", optimize=True)
        make_foreground(adaptive).save(folder / "ic_launcher_foreground.png", optimize=True)
        make_monochrome(adaptive).save(folder / "ic_launcher_monochrome.png", optimize=True)
        print(f"  {density}: legacy={legacy}px, adaptive={adaptive}px")

    # Play Store icon — keep alongside repo so we don't have to regenerate
    # it manually whenever the brand evolves.
    out_dir = ROOT / "build"
    out_dir.mkdir(exist_ok=True)
    make_legacy_square(512).save(out_dir / "playstore_icon.png", optimize=True)
    print(f"  playstore: 512px -> {out_dir / 'playstore_icon.png'}")


if __name__ == "__main__":
    main()
