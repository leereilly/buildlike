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
    for m in RECT_RE.finditer(text):
        cells.append(
            {
                "x": int(m.group("x")),
                "y": int(m.group("y")),
                "w": int(m.group("w")),
                "h": int(m.group("h")),
                "rx": int(m.group("rx")),
                "fill": hex_to_rgb(m.group("fill")),
            }
        )
    if not cells:
        raise SystemExit("no <rect> cells parsed from svg")
    return width, height, cells


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
    """Return cells for every grid slot not covered by an SVG rect."""
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


def render_frame(
    size: tuple[int, int],
    cells,
    empties,
    palette: list[tuple[int, int, int]],
    phases: list[float],
    speeds: list[float],
    sparkle_phase: list[float],
    sparkle_rate: list[float],
    t: float,
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
    for cell, phase, speed, s_phase, s_rate in zip(
        cells, phases, speeds, sparkle_phase, sparkle_rate
    ):
        pos = (phase + speed * t) % n
        i = int(math.floor(pos))
        frac = pos - i
        base = lerp_rgb(palette[i], palette[(i + 1) % n], frac)

        # Occasional brightness sparkle: a soft sine that briefly lifts toward white.
        sparkle = max(0.0, math.sin(2 * math.pi * s_rate * t + s_phase))
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


def rgba_to_indexed(im: Image.Image) -> Image.Image:
    """Convert RGBA frame to a P-mode image with a reserved transparency index."""
    alpha = im.getchannel("A")
    rgb = im.convert("RGB")
    indexed = rgb.quantize(
        colors=TRANSPARENT_INDEX, method=Image.Quantize.FASTOCTREE
    )
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

    width, height, cells = parse_svg(SVG_PATH)
    palette = sorted_palette(cells)
    n = len(palette)
    empties = empty_grid_cells(width, height, cells)

    rng = random.Random(args.seed)

    phases: list[float] = []
    speeds: list[float] = []
    sparkle_phase: list[float] = []
    sparkle_rate: list[float] = []
    for cell in cells:
        phases.append(float(palette.index(cell["fill"])))
        # Per-cell drift speed in palette-steps per frame: low enough to feel
        # gentle, varied enough that the field shimmers instead of marching.
        speeds.append(rng.uniform(0.04, 0.18))
        sparkle_phase.append(rng.uniform(0, 2 * math.pi))
        # Sparkle frequency in cycles per frame: keeps twinkles desynchronised.
        sparkle_rate.append(rng.uniform(0.01, 0.05))

    frames: list[Image.Image] = []
    for t in range(args.frames):
        frames.append(
            render_frame(
                (width, height),
                cells,
                empties,
                palette,
                phases,
                speeds,
                sparkle_phase,
                sparkle_rate,
                float(t),
                args.scale,
            )
        )

    quantised = [rgba_to_indexed(f) for f in frames]

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
        subprocess.run(
            [gifsicle, "-O3", "--lossy=60", "--colors", "128", str(out), "-o", str(out)],
            check=True,
        )
        print(f"optimised with gifsicle → {out.stat().st_size // 1024} KiB")


if __name__ == "__main__":
    main()
