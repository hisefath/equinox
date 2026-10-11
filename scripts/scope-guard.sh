#!/usr/bin/env bash
# scope-guard.sh — fail any change that falls outside a task's `Owns:` globs (WORKFLOW.md §2 rule 4, §7).
#
# Why this exists: a prompt saying "only edit your files" is a request; this script is a wall. The supervisor,
# the pre-push hook and CI all run it, so a worker in any harness gets the same answer.
set -euo pipefail
set -f # Owns globs must never expand against the working tree

usage() {
  cat <<'EOF'
usage: scripts/scope-guard.sh <task-file> [BASE_REF [HEAD_REF]]

  no BASE_REF  staged mode (pre-commit): checks `git diff --cached`; reads Owns: from the working copy.
  BASE_REF     branch mode (supervisor, pre-push, CI): checks BASE_REF...HEAD_REF (default HEAD) and reads
               Owns: from the task file AS COMMITTED ON BASE_REF, so a branch cannot widen its own scope
               by editing its task file.

Owns: line(s) in the task file hold comma/space separated globs. `*` matches across `/` (shell case
pattern), so `app/Services/**` and `app/Services/*` both cover every file below app/Services/.
`[` is literal, so route folders like `server/api/orders/[orderId]/**` work (no character classes).
A rename counts as delete + add, so both paths must be owned.

Exit: 0 every changed path is owned · 1 a path is out of scope · 2 usage error, unknown ref, or no Owns:
line (fails closed).
EOF
}

case "${1:-}" in -h | --help) usage; exit 0 ;; "") usage >&2; exit 2 ;; esac
[ $# -le 3 ] || { usage >&2; exit 2; }
task=$1

if [ $# -ge 2 ]; then
  # Resolve the task file to a repo-relative path so `git show <ref>:<path>` works from any cwd.
  rel=$(git ls-files --full-name --error-unmatch -- "$task" 2>/dev/null || true)
  if [ -z "$rel" ]; then
    top=$(git rev-parse --show-toplevel)
    case $task in "$top"/*) rel=${task#"$top"/} ;; *) rel=${task#./} ;; esac
  fi
  spec=$(git show "$2:$rel" 2>/dev/null) || {
    echo "scope-guard: $rel is not committed on $2 (commit the approved task file to the base first)" >&2
    exit 2
  }
  src="$2:$rel"
  range=("$2...${3:-HEAD}")
else
  [ -f "$task" ] || { echo "scope-guard: no such task file: $task" >&2; exit 2; }
  spec=$(cat "$task")
  src=$task
  range=(--cached)
fi

# Backticks and CRs are stripped so "Owns: `app/x.php`" and CRLF files still work.
owns=$(printf '%s\n' "$spec" | sed -n 's/^Owns:[[:space:]]*//p' | tr ',`\r' '   ')
[ -n "${owns//[[:space:]]/}" ] || { echo "scope-guard: no Owns: line in $src (fails closed)" >&2; exit 2; }

files=$(git -c core.quotePath=false diff --name-only --no-renames "${range[@]}") || {
  echo "scope-guard: git diff ${range[*]} failed" >&2
  exit 2
}

bad=0 n=0
while IFS= read -r f; do
  [ -n "$f" ] || continue
  n=$((n + 1)) ok=0
  for p in $owns; do
    q=${p//\[/\\[} # a literal [ (Nuxt/Next route folders), never a character class
    # shellcheck disable=SC2254 # $q is deliberately a pattern
    case $f in $q) ok=1; break ;; esac
  done
  [ "$ok" = 1 ] || { echo "scope-guard: out of scope: $f" >&2; bad=1; }
done <<EOF
$files
EOF

if [ "$bad" = 1 ]; then
  echo "scope-guard: FAIL — allowed by Owns ($src): $(printf '%s ' $owns)" >&2
  exit 1
fi
echo "scope-guard: OK — $n changed path(s) within Owns ($src)"
