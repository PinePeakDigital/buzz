# VHS tape for the buzz demo GIF. Render with `scripts/demo/record.sh`, which
# builds buzz, points it at the mock Beeminder server (scripts/demo/mockserver),
# and runs `vhs` against this file. See https://github.com/charmbracelet/vhs.

Output scripts/demo/demo.gif

Require buzz

Set Shell bash
Set FontSize 16
Set Width 1100
Set Height 720
Set Padding 20
Set Theme "Dracula"

# Quiet, clean prompt for the recording.
Hide
Type "export PS1='$ '"
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

# --- Scriptable output ---
Type@60ms "buzz list"
Enter
Sleep 2.5s

# Detailed view with the 7-day "amount due" forecast.
Type@60ms "buzz view meditate"
Enter
Sleep 4s
