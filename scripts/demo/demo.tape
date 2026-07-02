# VHS tape for the buzz demo GIF. Render with `scripts/demo/record.sh`, which
# builds buzz, points it at the mock Beeminder server (scripts/demo/mockserver),
# and runs `vhs` against this file. See https://github.com/charmbracelet/vhs.

Output scripts/demo/demo.gif

Require buzz

Set Shell bash
Set FontSize 16
Set Width 1100
# Tall enough that `buzz review`'s goal details + progress chart fit on screen
# without the chart (rendered at the bottom) being clipped in the alt-screen TUI.
Set Height 1000
Set Padding 20
Set Theme "Dracula"

# Quiet, clean prompt for the recording. Force a truecolor profile so buzz's
# lipgloss styling renders in color: headless CI terminals hand the program a
# TERM that termenv treats as colorless, falling back to monochrome. Setting
# COLORTERM=truecolor short-circuits that detection regardless of TERM.
Hide
Type "export PS1='$ '"
Enter
Type "export COLORTERM=truecolor TERM=xterm-256color CLICOLOR_FORCE=1"
Enter
Type "clear"
Enter
Show

# --- Interactive dashboard (TUI) ---
Type@80ms "buzz"
Enter
Sleep 3s
Right
Sleep 700ms
Down
Sleep 700ms
Enter
Sleep 2.5s
Escape
Sleep 800ms
Type "q"
Sleep 1s

# Clear the screen between commands so outputs don't stack up.
Type "clear"
Enter
Sleep 700ms

# --- Scriptable output ---
Type@60ms "buzz list"
Enter
Sleep 2.5s

Type "clear"
Enter
Sleep 700ms

# Goals due today.
Type@60ms "buzz today"
Enter
Sleep 2.5s

Type "clear"
Enter
Sleep 700ms

# --- Interactive review with progress chart (TUI) ---
# Walks goals one at a time, each with a progress chart of its datapoints
# (blue) against the bright red line (red). Goals are ordered alphabetically,
# so this opens on `inbox`; pressing Right advances through the cast.
#
# The two Right presses showcase the two chart densities:
#   - `meditate` — a short-history *sparse* chart: each datapoint is dotted (●)
#     on the blue staircase, mirroring Beeminder's graph, so it's clear the data
#     lands on the bright red line at each step.
#   - `pushups`  — a long-history *dense* chart: nodes fill nearly every column,
#     so the dots are suppressed and the line stays clean.
Type@60ms "buzz review"
Enter
Sleep 3.5s
Right
Sleep 3.5s
Right
Sleep 3.5s
Type "q"
Sleep 1s

Type "clear"
Enter
Sleep 700ms

# Detailed view with the 7-day "amount due" forecast.
Type@60ms "buzz view meditate"
Enter
Sleep 4s

Type "clear"
Enter
Sleep 700ms

# Log progress straight from the terminal. (Runs last; the demo's mock API
# acknowledges it without changing the earlier views.)
Type@60ms 'buzz add read 5 "read a chapter"'
Enter
Sleep 3s
