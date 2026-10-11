#!/usr/bin/env bash
# delegate.sh — dispatch ONE task file to the harness it names, in its own git worktree (WORKFLOW.md §6-7).
#
# The supervisor's only way to hand work to another harness. It records exactly what was sent (brief.md,
# cmd.txt) and what came back (result.md status, exit code, tree state) so nothing depends on a summary.
set -euo pipefail

usage() {
  cat <<'EOF'
usage: scripts/delegate.sh [--force] [--dry-run] [--why "<reason>"] <task-file> [milestone-branch]

Dispatches ONE task file (specs/M<N>-<slug>/tasks/T###.md, or a supervisor job file in the same format at
.handoff/runs/<id>/task.md, e.g. id M3-D01 for a draw-diagrams job) to the harness named in its frontmatter
(claude | codex | shell), in its own worktree ../<repo>-<id> on branch m<N>/t###-<slug> (a job: m<N>/d01-<slug>),
created from the milestone branch: argument > frontmatter `milestone_branch:` > the current milestone/* branch.
The task file must be committed on that branch: the worker gets the approved version, nothing newer.
The pre-push hook maps both branch shapes back to that file and runs scope-guard on them (WORKFLOW §2 rule 4).

Writes   .handoff/runs/<id>/brief.md      the prompt, derived mechanically from the task file
         .handoff/runs/<id>/cmd.txt       the exact command + harness version
         .handoff/runs/<id>/events.jsonl  harness event stream  (gitignored: may contain file contents)
         .handoff/runs/<id>/stderr.log    harness stderr        (gitignored)
Appends  "<date> DISPATCH <id> harness=… why=…" and "<date> RESULT <id> status=…" to TRACKER.md ## Log.
A previous attempt's files move to .handoff/runs/<id>.attempt-<timestamp>/ (evidence is never deleted),
except the task file and .handoff/runs/<id>/answers.md: the supervisor writes the human's verbatim answers there
before a relaunch, so it survives the dispatch (a copy also goes into the archive) for the worker to read.
Never commits for the worker, never pushes, never merges.

  --force     dispatch although the Log shows <id> in flight (last event DISPATCH without RESULT)
  --dry-run   print the brief and the command; create nothing, run nothing
  --why TEXT  reason recorded on the DISPATCH line (default: "task file names harness=<h>")

Harness commands (cwd = the worktree):
  claude  claude -p '<prompt>' --add-dir <run dir> <skill dir> --allowedTools <list>
            --permission-mode acceptEdits --output-format stream-json --verbose
  codex   ${CODEX_BIN:-codex} exec --sandbox workspace-write -C <worktree> --add-dir <run dir>
            -o <run dir>/last-message.md --json '<prompt>'
          (falls back to /Applications/ChatGPT.app/Contents/Resources/codex when codex is not on PATH)
  shell   bash -c '<frontmatter cmd:>'   token-free runs for testing this wrapper
Every harness gets HANDOFF_DIR, TASK_ID and RUN_DIR in its environment and stdin from /dev/null.

Env: FACTORY_KIT_DIR  skills root (default $HOME/Desktop/gauntlet_workbench/software-factory-kit)
     CODEX_BIN        codex binary
     DELEGATE_TIMEOUT_SEC  hard timeout (default: the "<n> min" in budget:, else 30 min)
     DELEGATE_CLAUDE_ALLOWED_TOOLS  --allowedTools rules for claude workers (default: make check/test/lint/types,
                      git status/diff/log/show/add/commit/rev-parse, date)
Exit: 0 result.md parsed (read its status!) · 1 usage/precondition · 3 refused: in flight
      · 4 result.md missing or unparseable · 5 timed out
EOF
}

die() { echo "delegate: $*" >&2; exit 1; }
now() { date '+%Y-%m-%dT%H:%M:%S%z'; }

# Frontmatter keys the parser knows; used to split WORKFLOW.md's compact "a: 1   b: 2" lines.
KEYS='id|milestone|skill|covers|after|harness|mode|hitl|parallel|budget|slug|milestone_branch|cmd|task|model|status|reason|base_sha|head_sha|changes|tests|not_verified'

# fm <key> <text> — value of <key> in the leading --- frontmatter --- block of <text>.
# Accepts one key per line (YAML style) and compact lines whose pairs are separated by 2+ spaces.
# `cmd:` always takes the rest of its line; other values lose trailing " # comments" and outer quotes.
fm() {
  printf '%s\n' "$2" | awk -v key="$1" -v keys="$KEYS" '
    NR == 1 { if ($0 !~ /^---[ \t]*$/) exit; next }
    /^---[ \t]*$/ { exit }
    {
      line = "  " $0
      if (!match(line, "[ \t][ \t]" key ":")) next
      v = substr(line, RSTART + RLENGTH)
      if (key != "cmd") {
        if (match(v, "[ \t][ \t]+(" keys "):")) v = substr(v, 1, RSTART - 1)
        sub(/[ \t]+#.*$/, "", v)
      }
      sub(/^[ \t]+/, "", v); sub(/[ \t\r]+$/, "", v)
      if (key != "cmd" && v ~ /^".*"$/) v = substr(v, 2, length(v) - 2)
      print v; exit
    }'
}

# q <args...> — shell-quote for cmd.txt (readable: only quotes what needs it).
q() {
  local a out=""
  for a in "$@"; do
    if [[ $a =~ ^[A-Za-z0-9_./:=@%+,-]+$ ]]; then out+="$a "; else out+="'${a//\'/\'\\\'\'}' "; fi
  done
  printf '%s' "${out% }"
}

slugify() { tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-+//; s/-+$//' | cut -c1-40 | sed -E 's/-+$//'; }

# ---- arguments -------------------------------------------------------------------------------------------
force=0 dry=0 why="" task_arg="" base=""
while [ $# -gt 0 ]; do
  case $1 in
    -h | --help) usage; exit 0 ;;
    --force) force=1 ;;
    --dry-run) dry=1 ;;
    --why) [ $# -ge 2 ] || die "--why needs a value"; why=$2; shift ;;
    -*) die "unknown option: $1 (see -h)" ;;
    *) if [ -z "$task_arg" ]; then task_arg=$1; elif [ -z "$base" ]; then base=$1; else die "too many arguments"; fi ;;
  esac
  shift
done
[ -n "$task_arg" ] || { usage >&2; exit 1; }
[ -f "$task_arg" ] || die "task file not found: $task_arg"

# ---- locate the main checkout (works when called from inside any worktree) -------------------------------
common=$(git rev-parse --path-format=absolute --git-common-dir 2>/dev/null) || die "not inside a git repository"
case $common in */.git) root=$(cd "${common%/.git}" && pwd -P) ;; *) die "bare repositories are not supported" ;; esac
hd="$root/.handoff"
tracker="$hd/TRACKER.md"
[ -f "$tracker" ] || die "no $tracker — run software-factory-kit scripts/init-repo.sh on this repo first"

task_abs="$(cd "$(dirname "$task_arg")" && pwd -P)/$(basename "$task_arg")"
task_top=$(cd "$(git -C "$(dirname "$task_abs")" rev-parse --show-toplevel)" && pwd -P)
rel=${task_abs#"$task_top"/}

# ---- milestone branch: argument > frontmatter > current milestone/* branch -------------------------------
[ -n "$base" ] || base=$(fm milestone_branch "$(cat "$task_abs")")
if [ -z "$base" ]; then
  cur=$(git -C "$root" symbolic-ref --short -q HEAD || true)
  case $cur in milestone/*) base=$cur ;; *) die "no milestone branch: pass it as 2nd argument or set milestone_branch: in $rel" ;; esac
fi
base_sha=$(git -C "$root" rev-parse -q --verify "$base^{commit}") || die "milestone branch not found: $base"

# The worker gets the task file as committed on the milestone branch (the G4-approved version).
task_text=$(git -C "$root" show "$base:$rel" 2>/dev/null) ||
  die "$rel is not committed on $base — commit the approved task file (G4) before dispatching"
[ "$task_text" = "$(cat "$task_abs")" ] || echo "delegate: warning: working copy of $rel differs from $base; dispatching the committed version" >&2

id=$(fm id "$task_text")
harness=$(fm harness "$task_text")
skill=$(fm skill "$task_text")
budget=$(fm budget "$task_text")
# Task ids are M<N>-T### (branch m<N>/t###-<slug>); supervisor job files may use another letter (M3-D01).
[[ $id =~ ^M([0-9]+)-([A-Z]+[0-9]+)$ ]] || die "frontmatter id must look like M1-T003 (or M3-D01 for a job file), got '${id}' in $rel"
mnum=${BASH_REMATCH[1]}
jnum=$(printf '%s' "${BASH_REMATCH[2]}" | tr '[:upper:]' '[:lower:]')
case $harness in claude | codex | shell) ;; *) die "frontmatter harness must be claude | codex | shell, got '${harness}'" ;; esac
printf '%s\n' "$task_text" | grep -q '^Owns:[[:space:]]*[^[:space:]]' || die "$rel has no Owns: line — scope-guard would fail closed; fix the task file"

slug=$(fm slug "$task_text" | slugify)
if [ -z "$slug" ]; then # first non-empty line under "## Ask"
  slug=$(printf '%s\n' "$task_text" | awk '/^## /{a=($0 ~ /^## Ask/); next} a && NF {print; exit}' | slugify)
fi
branch="m${mnum}/${jnum}-${slug:-task}"
git check-ref-format --branch "$branch" >/dev/null 2>&1 || die "bad branch name derived: $branch (set slug: in $rel)"
wt="$(dirname "$root")/$(basename "$root")-$id"
run_dir="$hd/runs/$id"

# ---- refuse if already in flight -------------------------------------------------------------------------
last=$(awk -v id="$id" '
  /^## / { inlog = ($0 ~ /^## Log[ \t]*$/); next }
  inlog && ($2 == "DISPATCH" || $2 == "RESULT") && $3 == id { ev = $2 }
  END { print ev }' "$tracker")
if [ "$last" = DISPATCH ] && [ "$force" = 0 ] && [ "$dry" = 0 ]; then
  echo "delegate: refused: $id is in flight (last Log event is DISPATCH with no RESULT)." >&2
  echo "          Check the worker / $run_dir first; re-run with --force only if it is really dead." >&2
  exit 3
fi

# ---- harness binary, skill, prompt -----------------------------------------------------------------------
kit=${FACTORY_KIT_DIR:-$HOME/Desktop/gauntlet_workbench/software-factory-kit}
skill_md="$kit/skills/${skill:-none}/SKILL.md"
case $harness in
  claude) bin=$(command -v claude || true); [ -n "$bin" ] || die "claude is not on PATH" ;;
  codex)
    bin=${CODEX_BIN:-codex}
    if ! command -v "$bin" >/dev/null 2>&1 && [ -z "${CODEX_BIN:-}" ]; then bin=/Applications/ChatGPT.app/Contents/Resources/codex; fi
    command -v "$bin" >/dev/null 2>&1 || die "codex not found (set CODEX_BIN)"
    ;;
  shell) bin=bash ;;
esac
if [ "$harness" != shell ]; then
  [ -n "$skill" ] || die "frontmatter skill: is required for harness $harness"
  [ -f "$skill_md" ] || die "skill not found: $skill_md (set FACTORY_KIT_DIR or run software-factory-kit scripts/install.sh)"
fi
version=$("$bin" --version 2>&1 | head -1 || true)

prompt="Run the ${skill:-none} skill: read $skill_md and follow it for task file $rel (relative to your working directory $wt); HANDOFF_DIR=$hd; write $run_dir/result.md per the result contract (specs/_templates/result.md); do not ask questions — if blocked, write status: human_review with the question under ## Questions."

claude_tools=${DELEGATE_CLAUDE_ALLOWED_TOOLS:-"Bash(make check) Bash(make check:*) Bash(make test:*) Bash(make lint:*) Bash(make types:*) Bash(git status:*) Bash(git diff:*) Bash(git log:*) Bash(git show:*) Bash(git add:*) Bash(git commit:*) Bash(git rev-parse:*) Bash(date) Bash(date:*)"}
case $harness in
  claude)
    # --add-dir / --allowedTools are variadic: the prompt must come before them and a flag must follow them.
    # Without the allow-list, -p mode silently denies every Bash call: no make check, no commit.
    # The list is ONE argument; claude splits it on spaces/commas outside parentheses.
    argv=("$bin" -p "$prompt" --add-dir "$run_dir" "$(dirname "$skill_md")" --allowedTools "$claude_tools"
      --permission-mode acceptEdits --output-format stream-json --verbose)
    ;;
  codex)
    # --add-dir: the run dir lives in the main checkout, outside the worktree sandbox.
    # UNVERIFIED: workspace-write may also block `git commit` in a linked worktree (its git metadata lives
    # in $root/.git/worktrees/); if workers report that, route it via route-failure before widening the sandbox.
    argv=("$bin" exec --sandbox workspace-write -C "$wt" --add-dir "$run_dir" -o "$run_dir/last-message.md" --json "$prompt")
    ;;
  shell)
    cmd=$(fm cmd "$task_text")
    [ -n "$cmd" ] || die "harness shell needs a cmd: line in the frontmatter of $rel"
    argv=(bash -c "$cmd")
    ;;
esac

if [ -n "${DELEGATE_TIMEOUT_SEC:-}" ]; then
  secs=$DELEGATE_TIMEOUT_SEC
else
  mins=$(printf ' %s' "$budget" | sed -nE 's/.*[^0-9]([0-9]+)[[:space:]]*min.*/\1/p')
  secs=$((10#${mins:-30} * 60))
fi
[[ $secs =~ ^[0-9]+$ ]] && [ "$secs" -gt 0 ] || die "bad timeout: $secs"
[ -n "$why" ] || why="task file names harness=$harness"

brief() {
  cat <<EOF
# Dispatch brief — $id

Derived mechanically from the task file by scripts/delegate.sh; never hand-edited (WORKFLOW.md §7).

- Task file: \`$rel\` as committed on \`$base\` (blob $(git -C "$root" rev-parse "$base:$rel"))
- Skill: ${skill:-none} → \`$skill_md\`
- Harness: $harness — $version
- Worktree: \`$wt\` on branch \`$branch\` from \`$base\` @ $base_sha
- HANDOFF_DIR: \`$hd\` · result: \`$run_dir/result.md\`
- Timeout: ${secs}s
- Dispatched: $1

## Prompt

$prompt
EOF
}
command_line="$(q "${argv[@]}") < /dev/null > $(q "$run_dir/events.jsonl") 2> $(q "$run_dir/stderr.log")"

if [ "$dry" = 1 ]; then
  brief "(dry run — nothing dispatched)"
  printf '\n## Command (cwd: %s)\n\n%s\n' "$wt" "$command_line"
  exit 0
fi

# ---- TRACKER ## Log append (mkdir lock: parallel fan-out dispatches write the same file) -----------------
log_line() {
  local lock="$hd/.tracker.lock" i=0
  until mkdir "$lock" 2>/dev/null; do
    i=$((i + 1))
    [ "$i" -lt 300 ] || die "TRACKER lock held for 30s: $lock (rmdir it if no delegate.sh is running)"
    sleep 0.1
  done
  awk -v add="$1" '
    function put() { if (content) { print add; flush() } else { flush(); print add } done = 1 }
    function flush() { while (nb > 0) { print ""; nb-- } }
    /^## / {
      if (inlog && !done) put()
      flush()
      inlog = ($0 ~ /^## Log[ \t]*$/); content = 0
      print; next
    }
    inlog && /^[ \t]*$/ { nb++; next }
    { flush(); if (inlog) content = 1; print }
    END {
      if (inlog && !done) put()
      if (!done) { flush(); print ""; print "## Log"; print ""; print add }
      flush()
    }' "$tracker" >"$tracker.tmp.$$" && mv "$tracker.tmp.$$" "$tracker"
  rmdir "$lock"
}

# ---- worktree --------------------------------------------------------------------------------------------
if [ -e "$wt" ]; then
  wt_common=$(git -C "$wt" rev-parse --path-format=absolute --git-common-dir 2>/dev/null || true)
  [ "$wt_common" = "$common" ] || die "$wt exists but is not a worktree of $root"
  cur=$(git -C "$wt" symbolic-ref --short -q HEAD || true)
  [ "$cur" = "$branch" ] || die "$wt exists but is on '${cur:-detached HEAD}', not $branch"
  echo "delegate: reusing worktree $wt ($branch)"
elif git -C "$root" show-ref --verify -q "refs/heads/$branch"; then
  git -C "$root" worktree add -q "$wt" "$branch"
else
  git -C "$root" worktree add -q -b "$branch" "$wt" "$base"
fi

# ---- run dir: archive the previous attempt so a stale result.md can never count ----------------------------
# A supervisor job file may live in the run dir itself (.handoff/runs/<id>/task.md); it stays put.
# answers.md is supervisor-written input for THIS dispatch (the human's answers to the last attempt's questions):
# it is copied into the archive as evidence and stays in place so the resumed worker can read it.
prev="$run_dir.attempt-$(date '+%Y%m%dT%H%M%S')"
for e in "$run_dir"/* "$run_dir"/.[!.]*; do
  [ -e "$e" ] && [ "$e" != "$task_abs" ] || continue
  mkdir -p "$prev"
  if [ "$e" = "$run_dir/answers.md" ]; then cp -p "$e" "$prev/"; else mv "$e" "$prev/"; fi
done
mkdir -p "$run_dir"
started=$(now)
brief "$started" >"$run_dir/brief.md"
{
  echo "# scripts/delegate.sh $started"
  echo "# harness: $harness — $version ($bin)"
  echo "# cwd: $wt"
  echo "# env: HANDOFF_DIR=$hd TASK_ID=$id RUN_DIR=$run_dir"
  echo "$command_line"
} >"$run_dir/cmd.txt"
log_line "$started DISPATCH $id harness=$harness why=$why evidence=.handoff/runs/$id/cmd.txt"
echo "delegate: $id → $harness in $wt ($branch), timeout ${secs}s"

# ---- run with a watchdog ---------------------------------------------------------------------------------
pid="" wd=""
stop_children() {
  [ -z "$wd" ] || { pkill -P "$wd" 2>/dev/null || true; kill "$wd" 2>/dev/null || true; }
  [ -z "$pid" ] || kill "$pid" 2>/dev/null || true
}
trap 'stop_children; log_line "$(now) RESULT $id status=missing harness=$harness interrupted=yes"; exit 130' INT TERM

cd "$wt"
env HANDOFF_DIR="$hd" TASK_ID="$id" RUN_DIR="$run_dir" "${argv[@]}" </dev/null >"$run_dir/events.jsonl" 2>"$run_dir/stderr.log" &
pid=$!
(
  sleep "$secs"
  if kill -0 "$pid" 2>/dev/null; then
    : >"$run_dir/.timed-out"
    kill -TERM "$pid" 2>/dev/null || true
    sleep 10
    kill -KILL "$pid" 2>/dev/null || true
  fi
) &
wd=$!
rc=0
wait "$pid" || rc=$?
pid=""
stop_children
wd=""
trap - INT TERM
cd "$root"

timed_out=no
if [ -f "$run_dir/.timed-out" ]; then timed_out=yes; rm -f "$run_dir/.timed-out"; fi

# ---- read the result (exit codes don't count; the contract does) -----------------------------------------
status=missing
if [ -f "$run_dir/result.md" ]; then
  rtext=$(cat "$run_dir/result.md")
  rstatus=$(fm status "$rtext")
  rtask=$(fm task "$rtext")
  case $rstatus in done | human_review | relaunch) status=$rstatus ;; *) status=invalid ;; esac
  if [ -n "$rtask" ] && [ "$rtask" != "$id" ]; then status=invalid; fi
fi
tree=clean
[ -z "$(git -C "$wt" status --porcelain)" ] || tree=dirty
head=$(git -C "$wt" rev-parse --short HEAD)
commits=$(git -C "$wt" rev-list --count "$base_sha..HEAD")

log_line "$(now) RESULT $id status=$status harness=$harness exit=$rc timeout=$timed_out tree=$tree head=$head commits=$commits evidence=.handoff/runs/$id/result.md"
echo "delegate: RESULT $id status=$status exit=$rc timeout=$timed_out tree=$tree head=$head commits=$commits"
if [ "$status" = done ] && [ "$tree" = dirty ]; then
  echo "delegate: NOTE: status=done but the worktree is dirty ⇒ treat as human_review (WORKFLOW.md §7)" >&2
fi

if [ "$timed_out" = yes ]; then
  echo "delegate: timed out after ${secs}s ⇒ human_review" >&2
  exit 5
fi
case $status in
  missing) echo "delegate: no $run_dir/result.md ⇒ human_review (see stderr.log / events.jsonl)" >&2; exit 4 ;;
  invalid) echo "delegate: $run_dir/result.md does not parse (status/task) ⇒ human_review" >&2; exit 4 ;;
esac
echo "delegate: next (supervisor): cd $wt && scripts/scope-guard.sh $rel $base && make check; then read git diff $base...HEAD"
