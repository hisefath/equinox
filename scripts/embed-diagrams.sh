#!/usr/bin/env bash
# embed-diagrams.sh — keep the Mermaid blocks inside design docs identical to their .mmd sources.
#
# draw-diagrams writes docs/architecture/diagrams/<type>.mmd (the source of truth); docs carry a placeholder.
# This script copies each source under its placeholder, so GitLab/GitHub render it and the doc never drifts.
set -euo pipefail

usage() {
  cat <<'EOF'
usage: scripts/embed-diagrams.sh [--check] <doc.md>...

For every placeholder line `<!-- diagram: <type> -->` in each doc, inserts (or replaces) the fenced
```mermaid block directly below it with docs/architecture/diagrams/<type>.mmd, followed by an end marker:

    <!-- diagram: erd -->
    ```mermaid
    …contents of erd.mmd…
    ```
    <!-- /diagram -->

Idempotent: a second run changes nothing. Placeholders inside other fenced code blocks are ignored.
<type> may contain only letters, digits, '.', '_' and '-'.

  --check   change nothing; exit 1 if any doc's blocks differ from their .mmd (or are not embedded yet)
            or a referenced .mmd is missing — use it in `make check` / CI.

Env: DIAGRAMS_DIR  overrides <repo root of the doc>/docs/architecture/diagrams
Exit: 0 in sync / embedded · 1 out of sync (--check) or missing .mmd / bad placeholder · 2 usage
EOF
}

check=0
docs=()
for a in "$@"; do
  case $a in
    -h | --help) usage; exit 0 ;;
    --check) check=1 ;;
    -*) echo "embed-diagrams: unknown option: $a" >&2; exit 2 ;;
    *) docs+=("$a") ;;
  esac
done
[ "${#docs[@]}" -gt 0 ] || { usage >&2; exit 2; }

# render <doc> <diagrams-dir> — print the doc with every placeholder's block (re)generated.
# awk exit 3 = missing .mmd, 4 = bad placeholder, 5 = unterminated mermaid fence after a placeholder.
render() {
  awk -v dir="$2" '
    function emit(t,   f, l, n) {
      f = dir "/" t ".mmd"
      if ((getline l < f) <= 0) { print "embed-diagrams: missing or empty " f > "/dev/stderr"; err = 3; return }
      print "```mermaid"; print l
      while ((getline l < f) > 0) print l
      close(f)
      print "```"; print "<!-- /diagram -->"
    }
    # state: 0 normal · 1 just after a placeholder · 2 inside the old block (skipping) · 3 old block closed
    state == 2 { if ($0 ~ /^```[ \t]*$/) state = 3; next }
    state == 1 {
      state = 0
      if ($0 ~ /^```mermaid[ \t]*$/) { state = 2; next }
      if ($0 ~ /^[ \t]*<!--[ \t]*\/diagram[ \t]*-->[ \t]*$/) next
    }
    state == 3 { state = 0; if ($0 ~ /^[ \t]*<!--[ \t]*\/diagram[ \t]*-->[ \t]*$/) next }
    /^[ \t]*(```|~~~)/ { infence = !infence; print; next }
    !infence && /^[ \t]*<!--[ \t]*diagram:/ {
      t = $0
      sub(/^[ \t]*<!--[ \t]*diagram:[ \t]*/, "", t); sub(/[ \t]*-->[ \t]*$/, "", t)
      if (t !~ /^[A-Za-z0-9._-]+$/) { print "embed-diagrams: bad placeholder: " $0 > "/dev/stderr"; err = 4; print; next }
      print; emit(t); state = 1; next
    }
    { print }
    END { if (state == 2) { print "embed-diagrams: unterminated ```mermaid block" > "/dev/stderr"; exit 5 } exit err }
  ' "$1"
}

rc=0
for doc in "${docs[@]}"; do
  [ -f "$doc" ] || { echo "embed-diagrams: no such file: $doc" >&2; rc=1; continue; }
  if [ -n "${DIAGRAMS_DIR:-}" ]; then
    dir=$DIAGRAMS_DIR
  else
    root=$(git -C "$(dirname "$doc")" rev-parse --show-toplevel 2>/dev/null || pwd)
    dir="$root/docs/architecture/diagrams"
  fi
  n=$(grep -cE '^[[:space:]]*<!--[[:space:]]*diagram:' "$doc" || true)
  tmp=$(mktemp "${TMPDIR:-/tmp}/embed-diagrams.XXXXXX")
  r=0
  render "$doc" "$dir" >"$tmp" || r=$?
  if [ "$r" != 0 ]; then
    echo "embed-diagrams: $doc: not changed (fix the error above)" >&2
    rm -f "$tmp"; rc=1; continue
  fi
  if cmp -s "$doc" "$tmp"; then
    echo "embed-diagrams: $doc: $n diagram(s) in sync"
    rm -f "$tmp"
  elif [ "$check" = 1 ]; then
    echo "embed-diagrams: $doc: OUT OF SYNC with $dir — run scripts/embed-diagrams.sh $doc" >&2
    diff -u "$doc" "$tmp" | head -40 >&2 || true
    rm -f "$tmp"; rc=1
  else
    cat "$tmp" >"$doc" # keeps the doc's inode and permissions
    rm -f "$tmp"
    echo "embed-diagrams: $doc: embedded $n diagram(s)"
  fi
done
exit "$rc"
