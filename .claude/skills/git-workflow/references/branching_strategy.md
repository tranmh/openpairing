# Git Branching Strategy Reference

## Branch Types and Naming Conventions

### Main Branches (Long-lived)

These branches have infinite lifetime and should never be deleted:

1. **main** - Production-ready code
   - Always deployable
   - Only receives merges from release or hotfix branches
   - Tagged with version numbers

2. **develop** - Integration branch for features
   - Latest delivered development changes for next release
   - Base branch for feature, bugfix, and refactor branches
   - When stable, merged to release branch

3. **release** - Release preparation branch
   - Created from develop when ready for release
   - Only bugfixes allowed (no new features)
   - When ready, merged to both main and develop
   - Tagged with version number

### Supporting Branches (Short-lived)

These branches have limited lifetime and should be deleted after merging:

1. **feature/** - New features
   - **Naming**: `feature/<ticket-id>-<short-description>`
   - **Examples**: `feature/USER-123-add-login`, `feature/FEAT-456-oauth-integration`
   - **Branch from**: `develop`
   - **Merge back to**: `develop`
   - **Deleted after**: Merge to develop

2. **bugfix/** - Bug fixes for non-production code
   - **Naming**: `bugfix/<ticket-id>-<short-description>`
   - **Examples**: `bugfix/BUG-789-fix-login-validation`, `bugfix/DEF-321-null-pointer`
   - **Branch from**: `develop`
   - **Merge back to**: `develop`
   - **Deleted after**: Merge to develop

3. **hotfix/** - Critical fixes for production
   - **Naming**: `hotfix/<ticket-id>-<short-description>`
   - **Examples**: `hotfix/CRIT-999-security-patch`, `hotfix/HOT-111-payment-failure`
   - **Branch from**: `main`
   - **Merge back to**: Both `main` and `develop`
   - **Deleted after**: Merge to both branches

4. **refactor/** - Code refactoring without feature changes
   - **Naming**: `refactor/<ticket-id>-<short-description>`
   - **Examples**: `refactor/TECH-555-extract-auth-service`, `refactor/DEBT-222-update-dependencies`
   - **Branch from**: `develop`
   - **Merge back to**: `develop`
   - **Deleted after**: Merge to develop

## Branch Naming Best Practices

1. **Use lowercase with hyphens**: `feature/user-123-add-login` not `Feature/USER_123_Add_Login`
2. **Include ticket/issue ID**: Makes tracking easier
3. **Be descriptive but concise**: `fix-auth` is too vague, `bugfix/BUG-789-fix-null-pointer-exception-in-authentication-service-when-user-login-fails` is too long
4. **Optimal length**: 3-5 words after the prefix
5. **Use present tense**: `add-feature` not `added-feature`

## Workflow Examples

### Creating a New Feature

```bash
# Start from latest develop
git checkout develop
git pull origin develop

# Create feature branch
git checkout -b feature/USER-123-add-login

# Work on feature with commits
git add .
git commit -m "feat(auth): add login form UI"
git commit -m "feat(auth): implement login API integration"

# Push to remote
git push -u origin feature/USER-123-add-login

# Create PR to merge back to develop
```

### Hotfix Emergency Fix

```bash
# Start from latest main
git checkout main
git pull origin main

# Create hotfix branch
git checkout -b hotfix/CRIT-999-security-patch

# Fix the critical issue
git add .
git commit -m "fix(security): patch XSS vulnerability in user input"

# Push to remote
git push -u origin hotfix/CRIT-999-security-patch

# Create PR to merge to main
# After merge to main, also merge to develop to keep them in sync
```

### Release Process

```bash
# Create release branch from develop
git checkout develop
git pull origin develop
git checkout -b release/v1.2.0

# Only bugfixes allowed on release branch
git commit -m "fix(docs): update changelog for v1.2.0"

# When ready, merge to main
git checkout main
git merge --no-ff release/v1.2.0
git tag -a v1.2.0 -m "Release version 1.2.0"
git push origin main --tags

# Also merge back to develop
git checkout develop
git merge --no-ff release/v1.2.0
git push origin develop

# Delete release branch
git branch -d release/v1.2.0
```

## Branch Protection Best Practices

1. **main**: Require PR reviews, no direct commits
2. **develop**: Require PR reviews, no direct commits
3. **release**: Require PR reviews, only bugfixes
4. **Supporting branches**: Can commit directly, but PR for merging
