Output equinox-demo.mp4

Set FontSize 13
Set Width 1280
Set Height 720
Set Padding 18
Set TypingSpeed 30ms
Set Theme "Catppuccin Mocha"
Set Shell "bash"
Set WaitTimeout 240s

Hide
Type "export PATH=$PWD:$PATH PS1='> ' && clear" Enter
Wait
Show

Type "# Project Equinox: Kalshi x Polymarket equivalence + deterministic routing (offline replay of a live 2026-10-06 recording)" Enter
Type "equinox scan -replay testdata/snapshot -near-misses 0 2>/dev/null | head -30 | cut -c1-175" Enter
Wait
Sleep 9s
Type "clear" Enter
Type "# Near misses: they read alike but a structural veto rejects them (one or two per veto kind)" Enter
Type `equinox scan -replay testdata/snapshot -near-misses 200 2>/dev/null | grep 'veto:' | grep -v 'veto: timing' | awk -F'veto: ' '{split($2,a,":"); if (c[a[1]]++ < 2) print}' | head -14 | cut -c1-200` Enter
Wait
Sleep 10s
Type "clear" Enter
Type "# Fees flip the decision: Kalshi has the cheaper ask on the Mississippi Senate race, Polymarket wins all-in" Enter
Type "equinox route -replay testdata/snapshot -pair 23 -side yes -qty 100 2>/dev/null" Enter
Wait
Sleep 11s
Type "clear" Enter
Type "# Price beats fees: 2,000 YES on the Iowa Senate race, splitting considered" Enter
Type "equinox route -replay testdata/snapshot -pair 26 -side yes -qty 2000 -split 2>/dev/null | sed -n '/^Decision/,$p'" Enter
Wait
Sleep 9s
Type "clear" Enter
Type "# Deterministic: same inputs, same decision id, every run" Enter
Type "for i in 1 2 3; do equinox route -replay testdata/snapshot -pair 26 -qty 2000 -split 2>/dev/null | grep '^Decision'; done" Enter
Wait
Sleep 4s
Type "# Each decision is appended to the audit log together with the match evidence it relied on" Enter
Type "tail -1 data/decisions.jsonl | cut -c1-700" Enter
Wait
Sleep 8s
