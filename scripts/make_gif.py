#!/usr/bin/env python3
"""Render an animated GIF version of contribution-graph.svg.

Each cell drifts through the Build color palette at its own frequency and
phase, giving the graph a gently sparkling shimmer when looped. The starting
frame matches the static SVG exactly so the spelled-out B-U-I-L-D is still
legible the moment the GIF loads.
"""
from __future__ import annotations

import argparse
import colorsys
import math
import random
import re
import shutil
import subprocess
from pathlib import Path

from PIL import Image, ImageDraw

REPO_ROOT = Path(__file__).resolve().parent.parent
SVG_PATH = REPO_ROOT / "contribution-graph.svg"
DEFAULT_OUT = REPO_ROOT / "contribution-graph.gif"

EMPTY_FILL = (0xEB, 0xED, 0xF0)  # GitHub light-theme empty contribution cell
CELL_PITCH = 13  # cell size + 3px gap, matches contribution-graph.svg layout
TRANSPARENT_INDEX = 255

RECT_RE = re.compile(
    r'<rect\s+x="(?P<x>-?\d+)"\s+y="(?P<y>-?\d+)"'
    r'\s+width="(?P<w>\d+)"\s+height="(?P<h>\d+)"'
    r'\s+rx="(?P<rx>\d+)"\s+ry="(?P<ry>\d+)"'
    r'\s+fill="#(?P<fill>[0-9a-fA-F]{6})"\s*/>'
)
VIEWBOX_RE = re.compile(r'viewBox="0 0 (\d+) (\d+)"')


def hex_to_rgb(h: str) -> tuple[int, int, int]:
    return int(h[0:2], 16), int(h[2:4], 16), int(h[4:6], 16)


def parse_svg(path: Path):
    text = path.read_text()
    vb = VIEWBOX_RE.search(text)
    if not vb:
        raise SystemExit("missing viewBox in svg")
    width, height = int(vb.group(1)), int(vb.group(2))
    cells = []
    empties = []
    for m in RECT_RE.finditer(text):
        rgb = hex_to_rgb(m.group("fill"))
        entry = {
            "x": int(m.group("x")),
            "y": int(m.group("y")),
            "w": int(m.group("w")),
            "h": int(m.group("h")),
            "rx": int(m.group("rx")),
            "fill": rgb,
        }
        # Treat the standard GitHub empty-cell grey as a static background tile;
        # everything else is a "contribution" that should sparkle.
        if rgb == EMPTY_FILL:
            empties.append(entry)
        else:
            cells.append(entry)
    if not cells:
        raise SystemExit("no contribution cells parsed from svg")
    return width, height, cells, empties


def fill_empty_grid(width: int, height: int, cells, existing_empties) -> list[dict]:
    """If the SVG omits empty cells, synthesize them on the standard grid."""
    if existing_empties:
        return existing_empties
    if not cells:
        return []
    cw = cells[0]["w"]
    ch = cells[0]["h"]
    rx = cells[0]["rx"]
    cols = (width + (CELL_PITCH - cw)) // CELL_PITCH
    rows = (height + (CELL_PITCH - ch)) // CELL_PITCH
    occupied = {(c["x"], c["y"]) for c in cells}
    out = []
    for r in range(rows):
        for c in range(cols):
            x = c * CELL_PITCH
            y = r * CELL_PITCH
            if (x, y) in occupied:
                continue
            out.append({"x": x, "y": y, "w": cw, "h": ch, "rx": rx, "fill": EMPTY_FILL})
    return out


def sorted_palette(cells) -> list[tuple[int, int, int]]:
    uniq = {c["fill"] for c in cells}

    def key(rgb):
        r, g, b = (v / 255 for v in rgb)
        h, s, v = colorsys.rgb_to_hsv(r, g, b)
        return (h, s, v)

    return sorted(uniq, key=key)


def lerp(a, b, t):
    return a + (b - a) * t


def lerp_rgb(c1, c2, t):
    return (
        int(round(lerp(c1[0], c2[0], t))),
        int(round(lerp(c1[1], c2[1], t))),
        int(round(lerp(c1[2], c2[2], t))),
    )


def empty_grid_cells(width: int, height: int, cells) -> list[dict]:
    """Backwards-compatible wrapper kept for clarity in callers/tests."""
    return fill_empty_grid(width, height, cells, [])


def render_frame(
    size: tuple[int, int],
    cells,
    empties,
    palette: list[tuple[int, int, int]],
    phases: list[float],
    drift_k: list[int],
    drift_amp: list[float],
    drift_offset: list[float],
    sparkle_k: list[int],
    sparkle_offset: list[float],
    t: float,
    total_frames: int,
    scale: int,
) -> Image.Image:
    w, h = size
    img = Image.new("RGBA", (w * scale, h * scale), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    for cell in empties:
        x0 = cell["x"] * scale
        y0 = cell["y"] * scale
        x1 = (cell["x"] + cell["w"]) * scale - 1
        y1 = (cell["y"] + cell["h"]) * scale - 1
        draw.rounded_rectangle(
            (x0, y0, x1, y1),
            radius=cell["rx"] * scale,
            fill=cell["fill"] + (255,),
        )

    n = len(palette)
    omega = 2 * math.pi / total_frames
    for cell, phase, dk, damp, doff, sk, soff in zip(
        cells, phases, drift_k, drift_amp, drift_offset, sparkle_k, sparkle_offset
    ):
        # Sine-based palette oscillation with integer per-cell frequency: every
        # cell completes a whole number of cycles per loop, so frame N == frame 0.
        pos = phase + damp * math.sin(dk * omega * t + doff)
        i = int(math.floor(pos))
        frac = pos - i
        base = lerp_rgb(palette[i % n], palette[(i + 1) % n], frac)

        # White-twinkle sparkle, also at an integer per-cell frequency.
        sparkle = max(0.0, math.sin(sk * omega * t + soff))
        sparkle = sparkle**6  # narrow the peaks → twinkly, not pulsing
        r, g, b = base
        r = int(round(lerp(r, 255, 0.55 * sparkle)))
        g = int(round(lerp(g, 255, 0.55 * sparkle)))
        b = int(round(lerp(b, 255, 0.55 * sparkle)))

        x0 = cell["x"] * scale
        y0 = cell["y"] * scale
        x1 = (cell["x"] + cell["w"]) * scale - 1
        y1 = (cell["y"] + cell["h"]) * scale - 1
        draw.rounded_rectangle(
            (x0, y0, x1, y1), radius=cell["rx"] * scale, fill=(r, g, b, 255)
        )
    return img


def build_master_palette(frames: list[Image.Image]) -> Image.Image:
    """Quantize all frames together to derive one shared palette.

    Stacking frames and quantising the result once ensures every frame maps the
    same RGB to the same palette index — so static regions (like the empty grey
    cells) don't shimmer from per-frame quantisation drift.
    """
    if not frames:
        raise SystemExit("no frames to build palette from")
    w, h = frames[0].size
    stack = Image.new("RGB", (w, h * len(frames)))
    for i, f in enumerate(frames):
        stack.paste(f.convert("RGB"), (0, i * h))
    return stack.quantize(colors=TRANSPARENT_INDEX, method=Image.Quantize.FASTOCTREE)


def rgba_to_indexed(im: Image.Image, master: Image.Image) -> Image.Image:
    """Convert RGBA frame to a P-mode image using the shared master palette."""
    alpha = im.getchannel("A")
    rgb = im.convert("RGB")
    indexed = rgb.quantize(palette=master, dither=Image.Dither.NONE)
    transparency_mask = Image.eval(alpha, lambda a: 255 if a < 128 else 0)
    indexed.paste(TRANSPARENT_INDEX, mask=transparency_mask)
    return indexed


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--out", default=str(DEFAULT_OUT))
    ap.add_argument("--frames", type=int, default=60)
    ap.add_argument("--frame-ms", type=int, default=80)
    ap.add_argument("--scale", type=int, default=2)
    ap.add_argument("--seed", type=int, default=0xB011D)
    args = ap.parse_args()

    width, height, cells, parsed_empties = parse_svg(SVG_PATH)
    palette = sorted_palette(cells)
    n = len(palette)
    empties = fill_empty_grid(width, height, cells, parsed_empties)

    rng = random.Random(args.seed)

    # All per-cell oscillators use integer frequencies (cycles per loop) so the
    # last frame transitions cleanly back into the first. Drift offset starts at
    # 0 or π so sin() is exactly 0 at t=0 → frame 0 matches the static SVG.
    DRIFT_KS = [1, 1, 1, 2, 2, 3]
    SPARKLE_KS = [2, 3, 3, 4, 5, 5, 7]
    phases: list[float] = []
    drift_k: list[int] = []
    drift_amp: list[float] = []
    drift_offset: list[float] = []
    sparkle_k: list[int] = []
    sparkle_offset: list[float] = []
    for cell in cells:
        phases.append(float(palette.index(cell["fill"])))
        drift_k.append(rng.choice(DRIFT_KS))
        # Wide amplitude → cells sweep across multiple distinct hues (red →
        # orange → yellow → green → …) instead of just shading their own hue.
        drift_amp.append(rng.uniform(4.0, 8.0))
        drift_offset.append(rng.choice([0.0, math.pi]))
        sparkle_k.append(rng.choice(SPARKLE_KS))
        sparkle_offset.append(rng.choice([0.0, math.pi]))

    frames: list[Image.Image] = []
    for t in range(args.frames):
        frames.append(
            render_frame(
                (width, height),
                cells,
                empties,
                palette,
                phases,
                drift_k,
                drift_amp,
                drift_offset,
                sparkle_k,
                sparkle_offset,
                float(t),
                args.frames,
                args.scale,
            )
        )

    master = build_master_palette(frames)
    quantised = [rgba_to_indexed(f, master) for f in frames]

    out = Path(args.out)
    out.parent.mkdir(parents=True, exist_ok=True)
    quantised[0].save(
        out,
        save_all=True,
        append_images=quantised[1:],
        duration=args.frame_ms,
        loop=0,
        disposal=2,
        transparency=TRANSPARENT_INDEX,
        optimize=True,
    )
    print(
        f"wrote {out} ({len(frames)} frames @ {args.frame_ms}ms, "
        f"{width * args.scale}x{height * args.scale}, palette={n})"
    )

    gifsicle = shutil.which("gifsicle")
    if gifsicle:
        # No --lossy: lossy mode introduces per-pixel noise that would make the
        # static empty cells shimmer between frames.
        subprocess.run(
            [gifsicle, "-O3", str(out), "-o", str(out)],
            check=True,
        )
        print(f"optimised with gifsicle → {out.stat().st_size // 1024} KiB")


if __name__ == "__main__":
    main()
