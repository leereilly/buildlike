#!/usr/bin/env python3
"""Render the embedded ASCII rickroll animation to animated SVGs.

Produces two files (black-on-transparent and white-on-transparent) suitable
for GitHub README light/dark-mode picture tags. Each SVG stacks all frames
vertically inside a clip-path and uses a CSS `steps()` keyframe animation
to scroll through them — yielding a small file that animates natively in
browsers without external assets.
"""
from __future__ import annotations

import argparse
from pathlib import Path
from xml.sax.saxutils import escape

REPO_ROOT = Path(__file__).resolve().parent.parent
FRAMES_PATH = REPO_ROOT / "internal" / "ui" / "rickroll_frames.txt"
FRAME_HEIGHT = 36  # must match internal/ui/rickroll.go

FONT_SIZE = 10
CELL_W = 6  # approx monospace character width at font-size 10
CELL_H = 12  # line box height at font-size 10


def load_frames() -> list[list[str]]:
    raw = FRAMES_PATH.read_text().replace("\r\n", "\n").rstrip("\n")
    lines = raw.split("\n")
    frames: list[list[str]] = []
    for i in range(0, len(lines), FRAME_HEIGHT):
        chunk = lines[i : i + FRAME_HEIGHT]
        if not chunk:
            continue
        frames.append(chunk)
    return frames


def trim_frames(frames: list[list[str]]) -> list[list[str]]:
    if not frames:
        return frames
    width = max(max((len(line) for line in f), default=0) for f in frames)
    height = FRAME_HEIGHT

    def nonempty(line: str) -> bool:
        return line.strip() != ""

    top = 0
    while top < height and all(not nonempty(f[top]) for f in frames if top < len(f)):
        top += 1
    bottom = height
    while bottom > top and all(
        not nonempty(f[bottom - 1]) for f in frames if bottom - 1 < len(f)
    ):
        bottom -= 1

    def col_empty(c: int) -> bool:
        for f in frames:
            for line in f:
                if c < len(line) and not line[c].isspace():
                    return False
        return True

    left = 0
    while left < width and col_empty(left):
        left += 1
    right = width
    while right > left and col_empty(right - 1):
        right -= 1

    out = []
    for f in frames:
        rows = []
        for r in range(top, bottom):
            line = f[r] if r < len(f) else ""
            line = line.ljust(width)[left:right].rstrip()
            rows.append(line)
        out.append(rows)
    return out


def frame_to_text(frame: list[str], y_offset: int) -> str:
    """Emit one <text> element containing all rows of a frame as <tspan>s."""
    parts = [f'<text x="0" y="{y_offset + FONT_SIZE}" xml:space="preserve">']
    for i, line in enumerate(frame):
        if not line:
            parts.append("<tspan/>")
            continue
        dy = 0 if i == 0 else CELL_H
        parts.append(
            f'<tspan x="0" dy="{dy}">{escape(line)}</tspan>'
        )
    parts.append("</text>")
    return "".join(parts)


def build_svg(frames: list[list[str]], fill: str, frame_ms: int) -> str:
    cols = max(len(line) for f in frames for line in f)
    rows = len(frames[0])
    frame_w = cols * CELL_W
    frame_h = rows * CELL_H
    n = len(frames)
    total_h = n * frame_h
    duration_s = (n * frame_ms) / 1000.0

    body = []
    for i, f in enumerate(frames):
        body.append(frame_to_text(f, i * frame_h))

    return f'''<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {frame_w} {frame_h}" width="{frame_w}" height="{frame_h}" role="img" aria-label="buildlike ASCII animation">
<style>
text {{ font: {FONT_SIZE}px ui-monospace, SFMono-Regular, Menlo, Consolas, "Liberation Mono", monospace; fill: {fill}; white-space: pre; }}
.strip {{ animation: buildlike-roll {duration_s}s steps({n}) infinite; }}
@keyframes buildlike-roll {{ from {{ transform: translateY(0); }} to {{ transform: translateY(-{total_h}px); }} }}
</style>
<clipPath id="buildlike-clip"><rect width="{frame_w}" height="{frame_h}"/></clipPath>
<g clip-path="url(#buildlike-clip)"><g class="strip">{''.join(body)}</g></g>
</svg>
'''


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--frame-ms", type=int, default=120)
    ap.add_argument("--out-dir", default=str(REPO_ROOT / "docs"))
    args = ap.parse_args()

    frames = trim_frames(load_frames())
    if not frames:
        raise SystemExit("no frames found")

    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    for name, fill in (("buildlike-light.svg", "#000"), ("buildlike-dark.svg", "#fff")):
        svg = build_svg(frames, fill, args.frame_ms)
        path = out_dir / name
        path.write_text(svg)
        print(f"wrote {path} ({len(frames)} frames, {len(svg)} bytes)")


if __name__ == "__main__":
    main()
