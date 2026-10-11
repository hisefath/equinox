# Makefile — `make check` is this project's single green/red command (software-factory-kit WORKFLOW.md §2 rule 4).
# Every agent runs it before claiming done, the supervisor re-runs it to verify, and .githooks/pre-push runs it
# before anything leaves the machine. It must exit non-zero when ANY part fails.
#
# ADAPT PER STACK: replace the three recipes below with this project's real commands (and copy them into the
# Commands table in AGENTS.md). Until then they fail on purpose, so an unconfigured repo can never look green.
# Keep the target names: skills and hooks call `make check`, not the tools directly.

.PHONY: check lint types test decisions

## check: lint + types + tests + decision review in sync — the gate
check: lint types test decisions
	@echo "make check: OK"

## lint: formatting / style, in check-only mode (never rewrite files here)
lint:
	@echo "make lint: not configured — edit Makefile (see the Laravel example below)" >&2; exit 1

## types: static analysis / type checking
types:
	@echo "make types: not configured — edit Makefile (see the Laravel example below)" >&2; exit 1

## test: the whole test suite, including tests/acceptance/**
test:
	@echo "make test: not configured — edit Makefile (see the Laravel example below)" >&2; exit 1

## decisions: docs/decisions/REVIEW.md matches the decision journal (works as is; no adapting needed)
decisions:
	bash scripts/decision-views.sh --check

# --- Example: Laravel 13 / PHP 8.5 (Pint + Larastan/PHPStan + Pest or PHPUnit) -------------------------------
# Replace the three recipes above with these (recipe lines must start with a TAB):
#
# lint:
# 	vendor/bin/pint --test
#
# types:
# 	vendor/bin/phpstan analyse --no-progress --memory-limit=1G
#
# test:
# 	php artisan test --compact
#
# Optional extras once they exist: `composer validate --strict` in lint, `php artisan test --parallel` in test,
# `bash scripts/embed-diagrams.sh --check docs/architecture/*.md` as a docs target added to check.
