---
name: ship
description: Full ship workflow — test, CSS build, commit, push, open PR to dev. Use when a feature is ready to ship.
---

# Ship

## Current branch state
- Branch: !`git branch --show-current`
- Status: !`git status --short`
- Commits ahead of dev: !`git log origin/dev..HEAD --oneline`

## Workflow

### 1. Tests

```sh
make test
```

Fix any failures before continuing. Do not skip.

### 2. CSS (only if Go templates changed)

This project is React-only on the current branch — Go templates and their CSS are removed.
Skip this step unless you're on a branch that still has `static/input.css`.

```sh
# Only if static/input.css exists and was modified:
make css
```

### 3. Commit

Ask the user to review the diff (`git diff --staged` or `git status`) before committing.

Use the `/commit` skill or:

```sh
git add <specific files>   # never git add -A blindly
git commit -m "type: description

Co-Authored-By: Claude Sonnet 4.6 <noreply@anthropic.com>"
```

Commit types: `feat` | `fix` | `refactor` | `docs` | `chore` | `test`

### 4. Push

```sh
git push origin <branch>
```

Never push to `main` or `staging` directly. Feature branches push to `feature/*` only.

### 5. PR to dev

```sh
gh pr create --base dev --title "type: description" --body "$(cat <<'EOF'
## Summary
- <what changed>
- <why>

## Test plan
- [ ] `make test` passes
- [ ] Manually verified <key flow>

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
```

## Branching rules

```
feature/*  →  dev  →  (PR to) staging  →  (PR to) main
```

- `feature/*` → `dev`: direct push OK
- `dev` → `staging`: PR required
- `staging` → `main`: PR required
- Never admin-bypass GitHub rulesets

## Checklist

- [ ] `make test` passes
- [ ] CSS rebuilt if Go templates touched
- [ ] User reviewed diff before commit
- [ ] PR targets `dev` (not `main` or `staging`)
- [ ] No secrets or sensitive data in commit
