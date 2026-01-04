---
name: git-workflow
description: This skill should be used when handling git operations, version control tasks, creating branches, making commits, or merging code. It provides comprehensive git workflow automation following industry best practices with conventional commits, proper branching strategies, and merge commit methodology for maintaining clean, traceable repository history.
---

# Git Workflow

## Overview

This skill provides comprehensive git workflow management following industry-standard best practices. It automates branch creation, enforces conventional commit standards, and manages merges using merge commit strategy to preserve full development history.

## When to Use This Skill

Use this skill whenever:
- Creating new branches for features, bugfixes, hotfixes, or refactoring
- Making commits that need to follow conventional commit format
- Merging branches while preserving complete history
- Managing releases and hotfixes across multiple branches
- Setting up git workflows for projects
- Validating commit messages
- Understanding git branching strategies

## Core Workflow

### Branching Strategy

The workflow uses these branch types:

**Main Branches (Long-lived):**
- `main` - Production-ready code, only receives merges from release/hotfix
- `develop` - Integration branch for next release, base for feature/bugfix/refactor
- `release` - Release preparation, created from develop, merged to main + develop

**Supporting Branches (Short-lived):**
- `feature/<ticket>-<description>` - New features, branch from develop
- `bugfix/<ticket>-<description>` - Bug fixes, branch from develop
- `hotfix/<ticket>-<description>` - Critical production fixes, branch from main
- `refactor/<ticket>-<description>` - Code refactoring, branch from develop

### Commit Convention

All commits follow Conventional Commits format:

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:** feat, fix, docs, style, refactor, perf, test, build, ci, chore

**Examples:**
- `feat(auth): add OAuth2 login support`
- `fix(api): handle null pointer in user service`
- `refactor(database): extract query builder to separate class`

### Merge Strategy

Uses **merge commit** strategy (`--no-ff`) to:
- Preserve complete commit history from feature branches
- Maintain branch context in git graph
- Enable easy feature reversion by reverting merge commits
- Show clear branch structure in visualization

## Quick Start Guide

### Creating a New Feature

```bash
# Option 1: Using the helper script
./scripts/create_branch.sh feature USER-123 add-login

# Option 2: Manual creation
git checkout develop
git pull origin develop
git checkout -b feature/USER-123-add-login
```

### Making Commits

```bash
# Make changes
git add <files>

# Commit with conventional format
git commit -m "feat(auth): add login form UI"
git commit -m "feat(auth): implement login API integration"

# Validate commit message (optional)
python scripts/validate_commit.py --message "feat(auth): add login form"
```

### Pushing and Creating PR

```bash
# Push to remote
git push -u origin feature/USER-123-add-login

# Create PR using GitHub CLI (semi-automatic)
gh pr create --base develop --title "feat: Add user authentication" --body "Implements OAuth2 login with Google and GitHub providers"
```

### Merging Feature to Develop

```bash
# Option 1: Using the helper script
./scripts/merge_branch.sh feature/USER-123-add-login develop

# Option 2: Manual merge
git checkout develop
git pull origin develop
git merge --no-ff feature/USER-123-add-login
git push origin develop

# Clean up
git branch -d feature/USER-123-add-login
git push origin --delete feature/USER-123-add-login
```

## Detailed Workflows

### Feature Development Workflow

1. **Start Feature**
   - Branch from: `develop`
   - Naming: `feature/<ticket>-<description>`
   - Example: `feature/USER-123-add-login`

2. **Develop**
   - Make changes and commit regularly
   - Use conventional commit messages
   - Keep commits atomic and focused

3. **Prepare for PR**
   - Ensure all tests pass
   - Push to remote: `git push -u origin feature/USER-123-add-login`
   - Review your own changes first

4. **Create PR**
   - Target branch: `develop`
   - Request reviews from team
   - Address feedback with additional commits

5. **Merge**
   - Use merge commit strategy (not squash or rebase)
   - Delete branch after successful merge

### Hotfix Workflow

Critical production fixes require merging to both `main` and `develop`:

```bash
# 1. Create hotfix from main
git checkout main
git pull origin main
git checkout -b hotfix/CRIT-999-security-patch

# 2. Fix the issue
git add <files>
git commit -m "fix(security): patch XSS vulnerability in user input"

# 3. Push and create PR to main
git push -u origin hotfix/CRIT-999-security-patch
gh pr create --base main --title "fix: Security patch for XSS vulnerability"

# 4. After merge to main, also merge to develop
git checkout develop
git pull origin develop
git merge --no-ff hotfix/CRIT-999-security-patch
git push origin develop

# 5. Clean up
git branch -d hotfix/CRIT-999-security-patch
git push origin --delete hotfix/CRIT-999-security-patch
```

### Release Workflow

```bash
# 1. Create release branch from develop
git checkout develop
git pull origin develop
git checkout -b release/v1.2.0

# 2. Prepare release (only bugfixes, no features)
git commit -m "fix(docs): update changelog for v1.2.0"
git commit -m "chore(version): bump version to 1.2.0"

# 3. Merge to main and tag
git checkout main
git merge --no-ff release/v1.2.0
git tag -a v1.2.0 -m "Release version 1.2.0"
git push origin main --tags

# 4. Merge back to develop
git checkout develop
git merge --no-ff release/v1.2.0
git push origin develop

# 5. Clean up
git branch -d release/v1.2.0
git push origin --delete release/v1.2.0
```

## Commit Message Guidelines

### Structure

```
<type>(<scope>): <subject>
<blank line>
<body>
<blank line>
<footer>
```

### Types and When to Use Them

- **feat**: New feature for the user
  - `feat(auth): add two-factor authentication`

- **fix**: Bug fix
  - `fix(api): resolve race condition in payment processing`

- **docs**: Documentation changes
  - `docs(readme): add installation instructions`

- **style**: Code style changes (formatting, no logic changes)
  - `style(lint): fix eslint warnings`

- **refactor**: Code refactoring (no bug fixes or features)
  - `refactor(auth): extract validation to separate service`

- **perf**: Performance improvements
  - `perf(database): add index on user_id column`

- **test**: Adding or updating tests
  - `test(auth): add integration tests for login flow`

- **build**: Build system or dependency changes
  - `build(deps): upgrade react to v18.2.0`

- **ci**: CI/CD configuration changes
  - `ci(github): add automated deployment workflow`

- **chore**: Other changes (no src/test modifications)
  - `chore(gitignore): add .env to gitignore`

### Best Practices

1. **Use imperative mood**: "add feature" not "added feature"
2. **Keep subject under 50 characters**: Be concise
3. **Don't capitalize first letter**: "add feature" not "Add feature"
4. **No period at end**: "add feature" not "add feature."
5. **Body explains why, not what**: Code shows what changed
6. **Reference issues in footer**: "Closes #123", "Fixes #456"

### Breaking Changes

When making breaking changes, add `BREAKING CHANGE:` to footer:

```
feat(api): change user endpoint response format

The /api/users endpoint now returns paginated results
instead of a simple array for better performance.

BREAKING CHANGE: API clients must update to handle the
new paginated response format.

Closes #234
```

## Using Helper Scripts

### create_branch.sh

Automatically creates properly named branches from correct base:

```bash
./scripts/create_branch.sh feature USER-123 add-login
# Creates: feature/user-123-add-login from develop

./scripts/create_branch.sh hotfix CRIT-999 security-patch
# Creates: hotfix/crit-999-security-patch from main
```

### validate_commit.py

Validates commit messages against conventional commits:

```bash
# Validate a commit message string
python scripts/validate_commit.py --message "feat(auth): add login"

# Use as git hook (add to .git/hooks/commit-msg)
python scripts/validate_commit.py "$1"
```

### merge_branch.sh

Safely merges branches with merge commit strategy:

```bash
./scripts/merge_branch.sh feature/USER-123-add-login develop
# - Checks out develop
# - Pulls latest changes
# - Merges with --no-ff
# - Prompts to push
# - Prompts to delete source branch
```

## Advanced Scenarios

### Handling Merge Conflicts

When conflicts occur during merge:

```bash
# Start merge
git merge --no-ff feature/USER-123-add-login

# Conflicts detected - resolve in your editor
# Look for markers: <<<<<<<, =======, >>>>>>>

# After resolving
git add <resolved-files>
git commit  # Complete the merge
```

### Syncing Fork with Upstream

```bash
# Add upstream remote (once)
git remote add upstream <original-repo-url>

# Sync develop branch
git checkout develop
git fetch upstream
git merge upstream/develop
git push origin develop
```

### Undoing Commits

```bash
# Undo last commit, keep changes staged
git reset --soft HEAD~1

# Undo last commit, keep changes unstaged
git reset HEAD~1

# Undo last commit, discard changes (dangerous!)
git reset --hard HEAD~1
```

### Amending Last Commit

```bash
# Fix last commit message
git commit --amend -m "feat(auth): add login form"

# Add more changes to last commit
git add <forgotten-file>
git commit --amend --no-edit
```

## Setting Up Git Hooks

Install commit message validator as pre-commit hook:

```bash
# Copy validator to git hooks
cp scripts/validate_commit.py .git/hooks/commit-msg
chmod +x .git/hooks/commit-msg

# Now all commits are automatically validated
```

## Pull Request Best Practices

### Creating PRs

1. **Title**: Use conventional commit format
   - `feat: Add user authentication`
   - `fix: Resolve payment timeout`

2. **Description**: Include
   - Summary of changes
   - Why the changes were made
   - Test plan
   - Screenshots (if UI changes)
   - Related issues

3. **Size**: Keep PRs small and focused
   - Easier to review
   - Faster to merge
   - Less risk of conflicts

### Reviewing PRs

1. Check code quality and style
2. Verify tests are included
3. Ensure conventional commits are used
4. Test locally if possible
5. Provide constructive feedback

### Merging PRs

1. Use "Create a merge commit" option (not squash or rebase)
2. Ensure CI/CD passes
3. Get required approvals
4. Delete branch after merge

## Repository Configuration

### GitHub Settings

Configure repository to enforce merge commits:

1. Settings → General → Pull Requests
2. ✅ Allow merge commits
3. ❌ Allow squash merging
4. ❌ Allow rebase merging
5. ✅ Automatically delete head branches

### Branch Protection

Configure protection for main branches:

**main branch:**
- Require pull request reviews (2 approvals)
- Require status checks to pass
- No direct commits

**develop branch:**
- Require pull request reviews (1 approval)
- Require status checks to pass
- No direct commits

## Resources

### scripts/

**create_branch.sh** - Automatically create properly named branches from correct base branch

**validate_commit.py** - Validate commit messages against Conventional Commits specification (can be used as git hook)

**merge_branch.sh** - Safely merge branches with merge commit strategy, with prompts for pushing and cleanup

### references/

**branching_strategy.md** - Comprehensive guide to branch types, naming conventions, and workflow examples

**commit_conventions.md** - Complete Conventional Commits reference with examples and best practices

**merge_strategies.md** - Detailed explanation of merge commit strategy vs alternatives, with commands and visualizations

## Troubleshooting

### Commit Message Rejected

If commit message validation fails:
1. Read the error message carefully
2. Check commit message format: `type(scope): subject`
3. Ensure type is valid: feat, fix, docs, etc.
4. Use imperative mood: "add" not "added"
5. Keep subject under 50 characters

### Merge Conflicts

If merge has conflicts:
1. Don't panic - conflicts are normal
2. Open conflicted files in your editor
3. Look for conflict markers: `<<<<<<<`, `=======`, `>>>>>>>`
4. Decide which changes to keep
5. Remove conflict markers
6. Stage resolved files: `git add <file>`
7. Complete merge: `git commit`

### Wrong Branch

If you committed to wrong branch:
1. Create correct branch: `git checkout -b correct-branch`
2. Reset wrong branch: `git checkout wrong-branch && git reset --hard HEAD~1`

### Pushed Wrong Commit

If you pushed a commit you want to remove:
1. **Never use force push on shared branches**
2. Instead, create a revert commit: `git revert <commit-hash>`
3. Push the revert: `git push origin <branch>`

## Best Practices Summary

1. **Always branch from develop** (except hotfixes from main)
2. **Use conventional commit messages** for all commits
3. **Keep commits atomic** - one logical change per commit
4. **Pull before creating branches** to start from latest code
5. **Use merge commits** (--no-ff) to preserve history
6. **Delete branches after merge** to keep repository clean
7. **Create small, focused PRs** for easier review
8. **Request reviews** before merging to main/develop
9. **Run tests** before pushing and creating PRs
10. **Semi-automatic workflow** - prepare locally, ask before pushing/creating PRs
