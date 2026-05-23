#!/usr/bin/env python3
"""Render the embedded ASCII rickroll animation to a transparent animated GIF.

Produces two files (white-on-transparent and black-on-transparent) suitable for
GitHub README light/dark-mode picture tags.
"""
from __future__ import annotations

import argparse
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

REPO_ROOT = Path(__file__).resolve().parent.parent
FRAMES_PATH = REPO_ROOT / "internal" / "ui" / "rickroll_frames.txt"
FRAME_HEIGHT = 36  # must match internal/ui/rickroll.go


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
            line = line.ljust(width)[left:right]
            rows.append(line)
        out.append(rows)
    return out


def render_frame(frame, font, cell_w, cell_h, color):
    width = max((len(line) for line in frame), default=0) * cell_w
    height = len(frame) * cell_h
    img = Image.new("RGBA", (width, height), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    for r, line in enumerate(frame):
        y = r * cell_h
        for c, ch in enumerate(line):
            if ch == " ":
                continue
            draw.text((c * cell_w, y), ch, font=font, fill=color)
    return img


def rgba_to_transparent_p(img, fg):
    alpha = img.split()[-1]
    mask = alpha.point(lambda a: 255 if a > 64 else 0).convert("1")
    palette = [0, 0, 0] * 256
    palette[3:6] = list(fg)
    out = Image.new("P", img.size, 0)
    out.putpalette(palette)
    out.paste(1, mask=mask)
    out.info["transparency"] = 0
    return out


def build_gif(frames, out_path, color, font_path, font_size, frame_ms, stride):
    font = ImageFont.truetype(font_path, font_size)
    bbox = font.getbbox("M")
    cell_w = bbox[2] - bbox[0]
    cell_h = max(font_size, bbox[3] - bbox[1])

    sampled = frames[::stride] if stride > 1 else frames
    rendered = [
        rgba_to_transparent_p(render_frame(f, font, cell_w, cell_h, (*color, 255)), color)
        for f in sampled
    ]
    rendered[0].save(
        out_path,
        save_all=True,
        append_images=rendered[1:],
        duration=frame_ms,
        loop=0,
        disposal=2,
        optimize=False,
        transparency=0,
    )
    print(f"wrote {out_path} ({len(rendered)} frames, {rendered[0].size[0]}x{rendered[0].size[1]})")


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--font", default="/System/Library/Fonts/Menlo.ttc")
    ap.add_argument("--font-size", type=int, default=10)
    ap.add_argument("--frame-ms", type=int, default=120)
    ap.add_argument("--stride", type=int, default=1)
    ap.add_argument("--out-dir", default=str(REPO_ROOT / "docs"))
    args = ap.parse_args()

    frames = trim_frames(load_frames())
    if not frames:
        raise SystemExit("no frames found")

    out_dir = Path(args.out_dir)
    out_dir.mkdir(parents=True, exist_ok=True)

    build_gif(frames, out_dir / "buildlike-light.gif", (0, 0, 0),
              args.font, args.font_size, args.frame_ms, args.stride)
    build_gif(frames, out_dir / "buildlike-dark.gif", (255, 255, 255),
              args.font, args.font_size, args.frame_ms, args.stride)


if __name__ == "__main__":
    main()
