# Commit Crawl gameplay demo (vhs tape)
#
# Regenerate the README's gameplay GIF:
#   go build -o commit-crawl ./cmd/commit-crawl
#   vhs scripts/demo.tape
#   gifsicle -O3 --lossy=30 scripts/_out/gameplay.gif -o gameplay.gif
#
# Requires: vhs (charm.sh), gifsicle. Runs the game with --seed 89 so the
# bug-bump → commits → stairs → "ascend to U" path is deterministic.

Output "scripts/_out/gameplay.gif"
Output "scripts/_out/gameplay.webm"

Set Shell "bash"
# JetBrains Mono (plain) lacks single-cell glyphs for ◆ ✦ ★ ⠋ ─ ╮ etc., so the
# terminal falls back to a proportional font for those runes and the grid
# breaks. The Nerd Font Mono variant ships those at one cell wide.
Set FontFamily "JetBrainsMono Nerd Font Mono"
Set FontSize 16
Set Width 1280
Set Height 760
Set Padding 24
Set Theme "GitHub Dark"
Set TypingSpeed 70ms
Set PlaybackSpeed 1.0
Set LoopOffset 7%

Hide
Type "clear"
Enter
Show

# Launch with a fixed seed so the demo is deterministic.
Type "./commit-crawl --seed 89"
Sleep 400ms
Enter

# --- Title screen: rainbow wordmark shimmer.
Sleep 2.2s

# --- Press any key to advance to username entry.
Space
Sleep 600ms

# --- Type a GitHub handle (the @ is already shown on screen).
Type "octocat"
Sleep 800ms
Enter

# --- Intro bloom: screen clears to a single @, then the dungeon blooms in.
Sleep 2.6s

# --- Walk floor 1 (B): cross the corridor, bump-attack a bug, grab commits, find stairs.
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Type "n" Sleep 230ms
Type "n" Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Type "u" Sleep 230ms
Type "u" Sleep 230ms
Up Sleep 230ms
Up Sleep 230ms
Type "y" Sleep 230ms
Type "y" Sleep 230ms
Right Sleep 230ms
Type "n" Sleep 230ms
Type "n" Sleep 230ms
Type "n" Sleep 230ms
Type "n" Sleep 230ms
Type "n" Sleep 230ms
Type "n" Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Right Sleep 230ms
Type "u" Sleep 230ms
Up Sleep 230ms
Type "u" Sleep 230ms

# --- Brief hold on the stairs cell so the "Stairs up. Press > to ascend." log line is readable.
Sleep 1.0s

# --- Ascend to floor 2 (U) — the dungeon silhouette becomes a U.
Type ">"
Sleep 3.0s
